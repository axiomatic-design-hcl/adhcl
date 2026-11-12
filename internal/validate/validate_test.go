package validate_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
)

// behaviorDesignPart holds the design, the need, and both functional requirements.
const behaviorDesignPart = `design "two_outputs" {
  boundary = "Return two independent configured strings under a fixed immutable input contract. Process failures are outside the change range."
  level = "architecture"
}

need "separate_outputs" {
  statement = "Receive the exact configured left and right strings."
  origin = "inferred"
}

fr "left" {
  need = [need.separate_outputs]
  statement = "Return the configured left string unchanged."
  criterion "exact" {
    metric = "For every supplied left string, the returned left string is byte-for-byte equal to it."
    op = "=="
    value = true
    origin = "proposed"
  }
}

fr "right" {
  need = [need.separate_outputs]
  statement = "Return the configured right string unchanged."
  criterion "exact" {
    metric = "For every supplied right string, the returned right string is byte-for-byte equal to it."
    op = "=="
    value = true
    origin = "proposed"
  }
}
`

// behaviorRestPart holds the decisions, influences, cells, modules, and scenario.
const behaviorRestPart = `dp "left_output" {
  controls = [fr.left]
  responsibility = "Copy the immutable left input to the left output without reading or changing the right input."
  decision = "selected"
}

dp "right_output" {
  controls = [fr.right]
  responsibility = "Copy the immutable right input to the right output without reading or changing the left input."
  decision = "selected"
}

influence {
  from = [dp.left_output]
  to = fr.left
  effect = "material"
  mechanism = "Changing the left-copy rule changes whether the exact-left predicate holds."
  mechanism_source = "inferred"
}

influence {
  from = [dp.right_output]
  to = fr.right
  effect = "material"
  mechanism = "Changing the right-copy rule changes whether the exact-right predicate holds."
  mechanism_source = "inferred"
}

cells "separate_inputs" {
  scope = design.two_outputs
  unlisted = "none"
  because = "Within the fixed immutable-input contract, each copy rule reads only its own input and cannot change the other rule or input. This is a proposed semantic boundary, not measured isolation."
}

module "left" {
  realizes = [dp.left_output]
  inputs = ["An immutable left string."]
  outputs = ["The same left string."]
  owns_state = []
  dependencies = []
  invariants = ["The right input and output are not read or written."]
  failure_behavior = "Do not publish a different string as a successful copy. Process availability is outside this contract."
  acceptance_checks = [fr.left.criterion.exact, scenario.copy_strings]
}

module "right" {
  realizes = [dp.right_output]
  inputs = ["An immutable right string."]
  outputs = ["The same right string."]
  owns_state = []
  dependencies = []
  invariants = ["The left input and output are not read or written."]
  failure_behavior = "Do not publish a different string as a successful copy. Process availability is outside this contract."
  acceptance_checks = [fr.right.criterion.exact, scenario.copy_strings]
}

scenario "copy_strings" {
  kind = "operational"
  verifies = [fr.left.criterion.exact, fr.right.criterion.exact]
  description = "Proposed checks only; no observations are supplied."
  steps = ["Supply empty strings, different strings, and strings containing non-ASCII characters.", "Require each output to equal its own input byte for byte.", "Vary each input independently and require the other output to remain unchanged."]
}
`

// behaviorBaselineModel is the single-file baseline. It is the conformance core-1
// model with the same declarations in the same order.
const behaviorBaselineModel = behaviorDesignPart + "\n" + behaviorRestPart

// behaviorAlternativeModel adds two alternatives for the left requirement.
// Candidate alpha lists left_output; candidate beta lists left_alt. right_output is common and listed by no candidate.
const behaviorAlternativeModel = behaviorDesignPart + `
dp "left_output" {
  controls = [fr.left]
  responsibility = "Copy the immutable left input to the left output without reading or changing the right input."
  decision = "selected"
}

dp "left_alt" {
  controls = [fr.left]
  responsibility = "Copy the immutable left input through the alternative path without reading or changing the right input."
  decision = "proposed"
}

dp "right_output" {
  controls = [fr.right]
  responsibility = "Copy the immutable right input to the right output without reading or changing the left input."
  decision = "selected"
}

decision "left_choice" {
  addresses = [fr.left]
  selected  = "alpha"

  candidate "alpha" {
    dps = [dp.left_output]
  }

  candidate "beta" {
    dps = [dp.left_alt]
  }
}

influence {
  from = [dp.left_output]
  to = fr.left
  effect = "material"
  mechanism = "Changing the left-copy rule changes whether the exact-left predicate holds."
  mechanism_source = "inferred"
}

influence {
  from = [dp.left_alt]
  to = fr.left
  effect = "material"
  mechanism = "Changing the alternative left-copy rule changes whether the exact-left predicate holds."
  mechanism_source = "inferred"
}

influence {
  from = [dp.right_output]
  to = fr.right
  effect = "material"
  mechanism = "Changing the right-copy rule changes whether the exact-right predicate holds."
  mechanism_source = "inferred"
}

cells "separate_inputs" {
  scope = design.two_outputs
  unlisted = "none"
  because = "Within the fixed immutable-input contract, each copy rule reads only its own input and cannot change the other rule or input. This is a proposed semantic boundary, not measured isolation."
}
`

// behaviorWriteFiles writes each name and content pair under dir.
func behaviorWriteFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

// behaviorRequest builds a source-profile request for path.
func behaviorRequest(path string) validate.Request {
	return validate.Request{Path: path, Profile: "source", Version: "dev"}
}

// behaviorFindingKeys returns sorted "code|subject|severity" keys for every finding.
// Keys ignore file order and message wording.
func behaviorFindingKeys(findings []result.Finding) []string {
	keys := make([]string, 0, len(findings))
	for _, finding := range findings {
		keys = append(keys, finding.Code+"|"+finding.Subject+"|"+string(finding.Severity))
	}
	sort.Strings(keys)
	return keys
}

// behaviorLabeledSubject matches a block identity such as fr.left. Its character classes
// exclude "/" and ":", so file paths and source locators never match.
var behaviorLabeledSubject = regexp.MustCompile(`^[a-z_]+\.[a-z0-9_.]+$`)

// behaviorSemanticKeys returns sorted "code|subject|severity" keys. A subject enters a key
// only when it is a labeled identity, so layout-dependent source locators do not change the key.
func behaviorSemanticKeys(findings []result.Finding) []string {
	keys := make([]string, 0, len(findings))
	for _, finding := range findings {
		subject := ""
		if behaviorLabeledSubject.MatchString(finding.Subject) {
			subject = finding.Subject
		}
		keys = append(keys, finding.Code+"|"+subject+"|"+string(finding.Severity))
	}
	sort.Strings(keys)
	return keys
}

// behaviorHasCode reports whether any finding carries the code.
func behaviorHasCode(findings []result.Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

// behaviorStageStatus returns the status of the named stage, or the empty string.
func behaviorStageStatus(rep validate.Result, name string) string {
	for _, stage := range rep.Stages {
		if stage.Name == name {
			return string(stage.Status)
		}
	}
	return ""
}

// behaviorScope returns the scope result whose matrix scope equals scope.
func behaviorScope(t *testing.T, rep validate.Result, scope string) assessment.ScopeResult {
	t.Helper()
	for _, scopeResult := range rep.Scopes {
		if string(scopeResult.Matrix.Scope) == scope {
			return scopeResult
		}
	}
	t.Fatalf("scope %q not found in report", scope)
	return assessment.ScopeResult{}
}

// behaviorMatrixKey renders a matrix as sorted strings that name the axes and each cell effect.
func behaviorMatrixKey(matrix assessment.Matrix) []string {
	var keys []string
	for _, fr := range matrix.FRs {
		for _, dp := range matrix.DPs {
			effect := ""
			if row, ok := matrix.Cells[fr]; ok {
				if cell, ok := row[dp]; ok {
					effect = string(cell.Effect)
				}
			}
			keys = append(keys, string(fr)+"/"+string(dp)+"="+effect)
		}
	}
	sort.Strings(keys)
	return keys
}

// behaviorPairing returns the FR-to-DP pairing of a scope as plain strings.
func behaviorPairing(scope assessment.ScopeResult) map[string]string {
	pairing := make(map[string]string, len(scope.Pairing))
	for fr, dp := range scope.Pairing {
		pairing[string(fr)] = string(dp)
	}
	return pairing
}

// failingWriter rejects every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("sink closed")
}

func TestSplitDirectoryMatchesSingleFileFindings(t *testing.T) {
	// Given a baseline model in one file and the same declarations split across two
	// immediate directory members, in both file-name orders.
	singleDir := t.TempDir()
	behaviorWriteFiles(t, singleDir, map[string]string{"model.ad.hcl": behaviorBaselineModel})

	frsFirstDir := t.TempDir()
	behaviorWriteFiles(t, frsFirstDir, map[string]string{
		"a_frs.ad.hcl":  behaviorDesignPart,
		"z_rest.ad.hcl": behaviorRestPart,
	})

	frsLastDir := t.TempDir()
	behaviorWriteFiles(t, frsLastDir, map[string]string{
		"z_frs.ad.hcl":  behaviorDesignPart,
		"a_rest.ad.hcl": behaviorRestPart,
	})

	// declaredIn maps each block identity in the split layouts to the file that declares it.
	declaredIn := map[string]string{
		"fr.left":               "a_frs.ad.hcl",
		"fr.right":              "a_frs.ad.hcl",
		"need.separate_outputs": "a_frs.ad.hcl",
		"dp.left_output":        "z_rest.ad.hcl",
		"dp.right_output":       "z_rest.ad.hcl",
	}
	declaredInReversed := map[string]string{
		"fr.left":               "z_frs.ad.hcl",
		"fr.right":              "z_frs.ad.hcl",
		"need.separate_outputs": "z_frs.ad.hcl",
		"dp.left_output":        "a_rest.ad.hcl",
		"dp.right_output":       "a_rest.ad.hcl",
	}

	// When the pipeline assembles each layout.
	singleReport := validate.Run(behaviorRequest(filepath.Join(singleDir, "model.ad.hcl")))
	frsFirstReport := validate.Run(behaviorRequest(frsFirstDir))
	frsLastReport := validate.Run(behaviorRequest(frsLastDir))

	// Then every layout yields the same finding codes, labeled subjects, and severities.
	singleKeys := behaviorSemanticKeys(singleReport.Findings)
	if got := behaviorSemanticKeys(frsFirstReport.Findings); strings.Join(got, "\n") != strings.Join(singleKeys, "\n") {
		t.Errorf("file order a_frs first: findings %v, want %v", got, singleKeys)
	}
	if got := behaviorSemanticKeys(frsLastReport.Findings); strings.Join(got, "\n") != strings.Join(singleKeys, "\n") {
		t.Errorf("file order z_frs first: findings %v, want %v", got, singleKeys)
	}

	// And every layout yields the same stage statuses and the same matrices.
	for name, rep := range map[string]validate.Result{
		"a_frs first": frsFirstReport,
		"z_frs first": frsLastReport,
	} {
		if got, want := behaviorStageStatus(rep, "core_assessment"), behaviorStageStatus(singleReport, "core_assessment"); got != want {
			t.Errorf("%s: core_assessment status %q, want %q", name, got, want)
		}
		singleScope := behaviorScope(t, singleReport, "design.two_outputs")
		scope := behaviorScope(t, rep, "design.two_outputs")
		if got, want := strings.Join(behaviorMatrixKey(scope.Matrix), "\n"), strings.Join(behaviorMatrixKey(singleScope.Matrix), "\n"); got != want {
			t.Errorf("%s: matrix %q, want %q", name, got, want)
		}
		if got, want := scope.Classification, singleScope.Classification; got != want {
			t.Errorf("%s: classification %q, want %q", name, got, want)
		}
	}

	// And each finding that carries a range keeps the file that declares its subject.
	for name, pair := range map[string]struct {
		rep        validate.Result
		declaredIn map[string]string
	}{
		"a_frs first": {frsFirstReport, declaredIn},
		"z_frs first": {frsLastReport, declaredInReversed},
	} {
		for _, finding := range pair.rep.Findings {
			if finding.Range == nil {
				continue
			}
			want, known := pair.declaredIn[finding.Subject]
			if !known {
				continue
			}
			if got := filepath.Base(finding.Range.Filename); got != want {
				t.Errorf("%s: finding %q range file %q, want %q", name, finding.Code, got, want)
			}
		}
	}
}

func TestForbiddenExpressionBlocksCoreWithPositionedFinding(t *testing.T) {
	// Given a valid baseline file and a second file whose need uses a function call,
	// which the syntax profile forbids.
	dir := t.TempDir()
	behaviorWriteFiles(t, dir, map[string]string{
		"base.ad.hcl": behaviorBaselineModel,
		"extra.ad.hcl": `need "extra_need" {
  statement = upper("forbidden")
}
`,
	})
	var output bytes.Buffer

	// When the pipeline runs on the directory.
	rep := validate.Run(behaviorRequest(dir))
	code, err := renderValidation(behaviorRequest(dir), &output)

	// Then the syntax finding is located in the file that contains the expression.
	var syntaxFinding *result.Finding
	for i := range rep.Findings {
		if rep.Findings[i].Code == "invalid_syntax" {
			syntaxFinding = &rep.Findings[i]
		}
	}
	if syntaxFinding == nil {
		t.Fatalf("expected an invalid_syntax finding, got %v", behaviorFindingKeys(rep.Findings))
	}
	if syntaxFinding.Range == nil {
		t.Fatalf("invalid_syntax finding has no range")
	}
	if got := filepath.Base(syntaxFinding.Range.Filename); got != "extra.ad.hcl" {
		t.Errorf("invalid_syntax range file %q, want %q", got, "extra.ad.hcl")
	}
	if syntaxFinding.Range.Start.Line < 1 {
		t.Errorf("invalid_syntax range start line %d, want at least 1", syntaxFinding.Range.Start.Line)
	}

	// And the core assessment is blocked, with no downstream resolution findings.
	if got := behaviorStageStatus(rep, "core_assessment"); got != "blocked" {
		t.Errorf("core_assessment status %q, want %q", got, "blocked")
	}
	if behaviorHasCode(rep.Findings, "invalid_hierarchy") {
		t.Errorf("unexpected invalid_hierarchy after a syntax error")
	}
	if behaviorHasCode(rep.Findings, "invalid_reference") {
		t.Errorf("unexpected invalid_reference after a syntax error")
	}

	// And the run exits with code 1 without a panic.
	if err != nil {
		t.Errorf("Run returned error %v, want nil for a rendered report", err)
	}
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestDecodeErrorDoesNotRunDependentResolution(t *testing.T) {
	// Given a baseline whose left decision has an empty controls list, which the schema
	// forbids, and an influence that names a missing requirement, which only resolution checks.
	dir := t.TempDir()
	broken := strings.Replace(behaviorBaselineModel,
		"controls = [fr.left]", "controls = []", 1)
	broken += `
influence {
  from = [dp.right_output]
  to = fr.missing_requirement
  effect = "none"
  because = "The right copy never changes a missing requirement."
}
`
	behaviorWriteFiles(t, dir, map[string]string{"model.ad.hcl": broken})

	// When the pipeline assembles the report.
	rep := validate.Run(behaviorRequest(filepath.Join(dir, "model.ad.hcl")))

	// Then the decode error is reported and core assessment is blocked.
	if !behaviorHasCode(rep.Findings, "invalid_structure") {
		t.Errorf("expected an invalid_structure finding, got %v", behaviorFindingKeys(rep.Findings))
	}
	if got := behaviorStageStatus(rep, "core_assessment"); got != "blocked" {
		t.Errorf("core_assessment status %q, want %q", got, "blocked")
	}

	// And resolution never runs, so no hierarchy or reference finding appears.
	if behaviorHasCode(rep.Findings, "invalid_hierarchy") {
		t.Errorf("unexpected invalid_hierarchy after a decode error")
	}
	if behaviorHasCode(rep.Findings, "invalid_reference") {
		t.Errorf("unexpected invalid_reference after a decode error")
	}
}

func TestCandidateSelectionChangesActiveScope(t *testing.T) {
	// Given a model with two candidates for the left requirement: alpha lists left_output
	// and beta lists the proposed alternative left_alt. right_output is common and listed by no candidate.
	dir := t.TempDir()
	behaviorWriteFiles(t, dir, map[string]string{"model.ad.hcl": behaviorAlternativeModel})
	path := filepath.Join(dir, "model.ad.hcl")

	defaultRequest := behaviorRequest(path)
	betaRequest := behaviorRequest(path)
	betaRequest.Candidate = "decision.left_choice.candidate.beta"

	// When the pipeline assembles the default snapshot and the beta snapshot.
	defaultReport := validate.Run(defaultRequest)
	betaReport := validate.Run(betaRequest)

	// Then the default snapshot pairs the left requirement with left_output and omits left_alt.
	defaultScope := behaviorScope(t, defaultReport, "design.two_outputs")
	if got := behaviorPairing(defaultScope)["fr.left"]; got != "dp.left_output" {
		t.Errorf("default pairing for fr.left = %q, want %q", got, "dp.left_output")
	}
	if containsDP(defaultScope.Matrix, "dp.left_alt") {
		t.Errorf("default matrix includes dp.left_alt, want it inactive")
	}

	// And the beta snapshot pairs the left requirement with left_alt and omits left_output.
	betaScope := behaviorScope(t, betaReport, "design.two_outputs")
	if got := behaviorPairing(betaScope)["fr.left"]; got != "dp.left_alt" {
		t.Errorf("beta pairing for fr.left = %q, want %q", got, "dp.left_alt")
	}
	if containsDP(betaScope.Matrix, "dp.left_output") {
		t.Errorf("beta matrix includes dp.left_output, want it inactive")
	}

	// And the unselected alternative is never reported as uncovered in either snapshot.
	for name, rep := range map[string]validate.Result{"default": defaultReport, "beta": betaReport} {
		if behaviorHasCode(rep.Findings, "uncontrolled_requirement") {
			t.Errorf("%s snapshot reports uncontrolled_requirement, want none", name)
		}
	}
	for _, finding := range defaultReport.Findings {
		if finding.Subject == "dp.left_alt" {
			t.Errorf("default snapshot reports dp.left_alt (code %q), want no finding", finding.Code)
		}
	}
}

// containsDP reports whether the matrix has a DP column named by id.
func containsDP(matrix assessment.Matrix, id string) bool {
	for _, dp := range matrix.DPs {
		if string(dp) == id {
			return true
		}
	}
	return false
}

func TestRunReturnsExitOneWhenOutputFails(t *testing.T) {
	// Given a valid baseline file and a writer that rejects every write.
	dir := t.TempDir()
	behaviorWriteFiles(t, dir, map[string]string{"model.ad.hcl": behaviorBaselineModel})
	request := behaviorRequest(filepath.Join(dir, "model.ad.hcl"))

	// When the pipeline runs with the failing writer.
	code, err := renderValidation(request, failingWriter{})

	// Then the run returns an error and exit code 1.
	if err == nil {
		t.Errorf("Run error = nil, want a write failure")
	}
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
}

func TestManifestSymlinkEscapeIsNeverRead(t *testing.T) {
	// Given a manifest whose second core member is a symbolic link to a file outside the
	// manifest directory. The outside file holds a forbidden expression that would be
	// reported if it were read.
	root := t.TempDir()
	modelDir := filepath.Join(root, "model")
	outsideDir := filepath.Join(root, "outside")
	if err := os.Mkdir(modelDir, 0o755); err != nil {
		t.Fatalf("mkdir model: %v", err)
	}
	if err := os.Mkdir(outsideDir, 0o755); err != nil {
		t.Fatalf("mkdir outside: %v", err)
	}
	outsidePath := filepath.Join(outsideDir, "secret.ad.hcl")
	if err := os.WriteFile(outsidePath, []byte(`need "escaped" {
  statement = upper("escaped")
}
`), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	behaviorWriteFiles(t, modelDir, map[string]string{
		"base.ad.hcl": behaviorBaselineModel,
		"model.ad.project.hcl": `project "escape" {
  language = "requirements_hcl/core/1"
  core_files = ["base.ad.hcl", "linked.ad.hcl"]
  attachment_files = []
}
`,
	})
	if err := os.Symlink(filepath.Join("..", "outside", "secret.ad.hcl"), filepath.Join(modelDir, "linked.ad.hcl")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// When the pipeline assembles the manifest.
	rep := validate.Run(behaviorRequest(filepath.Join(modelDir, "model.ad.project.hcl")))

	// Then an input failure blocks core assessment.
	if !behaviorHasCode(rep.Findings, "input_failure") {
		t.Errorf("expected an input_failure finding, got %v", behaviorFindingKeys(rep.Findings))
	}
	if got := behaviorStageStatus(rep, "core_assessment"); got != "blocked" {
		t.Errorf("core_assessment status %q, want %q", got, "blocked")
	}

	// And the escaped file is never read: no syntax finding and no range in that file.
	if behaviorHasCode(rep.Findings, "invalid_syntax") {
		t.Errorf("unexpected invalid_syntax, which means the escaped file was read")
	}
	for _, finding := range rep.Findings {
		if finding.Range != nil && filepath.Base(finding.Range.Filename) == "secret.ad.hcl" {
			t.Errorf("finding %q is located in the escaped file", finding.Code)
		}
	}
}

func TestReportCarriesRequestVersion(t *testing.T) {
	// Given a valid baseline file and a request that carries a release version.
	dir := t.TempDir()
	behaviorWriteFiles(t, dir, map[string]string{"model.ad.hcl": behaviorBaselineModel})
	const version = "v0.1.0-behavior"
	request := validate.Request{
		Path:    filepath.Join(dir, "model.ad.hcl"),
		Profile: "source",
		Version: version,
	}
	var output bytes.Buffer

	// When the pipeline assembles the report and renders it.
	rep := validate.Run(request)
	if _, err := renderValidation(request, &output); err != nil {
		t.Fatalf("Run returned error %v, want nil", err)
	}

	// Then the report header carries the requested tool version, and the rendered output shows it.
	if rep.ToolVersion != version {
		t.Errorf("report ToolVersion %q, want %q", rep.ToolVersion, version)
	}
	if !strings.Contains(output.String(), version) {
		t.Errorf("rendered output does not contain version %q", version)
	}
}
