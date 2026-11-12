package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// acceptanceRelease is the version injected into the binary under test.
const acceptanceRelease = "v0.1.0"

// acceptanceHorizonModel has one decision with two candidates. Each candidate has measured
// evidence, and the two candidates declare different horizons.
const acceptanceHorizonModel = `design "receipt" {
  boundary = "Serve one receipt request."
  workload = workload.load
}

workload "load" {
  description = "One request at a time."
}

context "lab" {
  description = "Controlled test bench."
}

need "served" {
  statement = "Serve each receipt request."
}

fr "serve" {
  need      = [need.served]
  statement = "Serve one receipt request."
  criterion "latency" {
    metric = "Each request completes within its bound."
    op     = "=="
    value  = true
    origin = "proposed"
  }
}

dp "serve_path" {
  controls       = [fr.serve]
  responsibility = "Serve the request on the integer-cent path."
  decision       = "proposed"
}

dp "alt_path" {
  controls       = [fr.serve]
  responsibility = "Serve the request on a decimal path."
  decision       = "proposed"
}

scenario "load_check" {
  kind        = "operational"
  verifies    = [fr.serve.criterion.latency]
  description = "Proposed load check for one request."
  steps       = ["Send one request and record whether it completes within its bound."]
}

decision "receipt_mechanism" {
  addresses = [fr.serve]
  context   = context.lab
  workload  = workload.load
  horizon   = "One request under fixed input."
  candidate "integer_cents" {
    dps     = [dp.serve_path]
    version = "receipt-proposal-1"
    information {
      basis    = "measured"
      evidence = [evidence.run_a]
    }
  }
  candidate "decimal_cents" {
    dps     = [dp.alt_path]
    version = "receipt-proposal-2"
    horizon = "Ten requests under fixed input."
    information {
      basis    = "measured"
      evidence = [evidence.run_b]
    }
  }
}

evidence "run_a" {
  basis             = "measured"
  event             = "A request completes within its bound."
  scenario          = scenario.load_check
  candidate         = decision.receipt_mechanism.candidate.integer_cents
  version           = "receipt-proposal-1"
  context           = context.lab
  workload          = workload.load
  horizon           = "One request under fixed input."
  dependence        = "joint"
  sampling          = "Independent requests from one fixed input."
  trials            = 50
  failures          = 5
  confidence_method = "wilson"
  confidence        = 0.9
}

evidence "run_b" {
  basis             = "measured"
  event             = "A request completes within its bound."
  scenario          = scenario.load_check
  candidate         = decision.receipt_mechanism.candidate.decimal_cents
  version           = "receipt-proposal-2"
  context           = context.lab
  workload          = workload.load
  horizon           = "Ten requests under fixed input."
  dependence        = "joint"
  sampling          = "Independent requests from one fixed input."
  trials            = 50
  failures          = 5
  confidence_method = "wilson"
  confidence        = 0.9
}
`

// acceptanceResult is what one CLI run wrote and returned.
type acceptanceResult struct {
	stdout string
	stderr string
	code   int
}

// TestAcceptance builds the binary once with the injected release version and runs every
// acceptance case against it. The binary lives in this test's temporary directory, so every
// case can use it.
func TestAcceptance(t *testing.T) {
	binary := buildExecutable(t, "-X main.buildVersion="+acceptanceRelease)
	cases := []struct {
		name string
		run  func(t *testing.T, binary string)
	}{
		{"help goes to stdout", acceptanceHelp},
		{"usage errors exit two", acceptanceUsageErrors},
		{"version ignores unreadable models", acceptanceVersionIgnoresModels},
		{"validation header carries release", acceptanceValidationHeader},
		{"output failure exits one", acceptanceOutputFailure},
		{"dash path after double dash", acceptanceDashPath},
		{"forbidden expression is positioned", acceptanceForbiddenExpression},
		{"rejected models exit one", acceptanceRejectedModels},
		{"exact unknown stays unknown", acceptanceExactUnknown},
		{"split directory keeps findings", acceptanceSplitDirectory},
		{"candidate selection is reported", acceptanceCandidateSelection},
		{"layout change keeps findings", acceptanceLayoutChange},
		{"module contract follows profile", acceptanceModuleContract},
		{"incompatible horizons stay unresolved", acceptanceIncompatibleHorizons},
		{"invalid attachment keeps core scope", acceptanceInvalidAttachment},
		{"manifest symlink escape fails input", acceptanceManifestSymlink},
		{"failed expectation exits one", acceptanceFailedExpectation},
		{"release examples exit zero", acceptanceReleaseExamples},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, binary)
		})
	}
}

func acceptanceHelp(t *testing.T, binary string) {
	// Given the built binary with no model access needed for help
	helpArgs := [][]string{{"--help"}, {"-h"}, {"validate", "--help"}}

	// When help is requested with each accepted spelling
	for _, args := range helpArgs {
		result := runCLI(t, binary, "", args...)

		// Then each request exits 0, writes help to stdout, and leaves stderr empty
		requireExit(t, result, 0)
		if result.stdout == "" {
			t.Fatalf("%v wrote no help to stdout", args)
		}
		if result.stderr != "" {
			t.Fatalf("%v wrote to stderr: %s", args, result.stderr)
		}
	}
}

func acceptanceUsageErrors(t *testing.T, binary string) {
	// Given a model path that does not exist
	missing := filepath.Join(t.TempDir(), "missing.ad.hcl")
	usage := []struct {
		name string
		args []string
	}{
		{"no command", nil},
		{"unknown command", []string{"frobnicate"}},
		{"missing path", []string{"validate"}},
		{"invalid profile", []string{"validate", "--profile", "bogus", missing}},
		{"repeated flag", []string{"validate", "--profile", "source", "--profile", "source", missing}},
		{"version with argument", []string{"version", "extra"}},
	}

	for _, tc := range usage {
		t.Run(tc.name, func(t *testing.T) {
			// When the invalid command runs
			result := runCLI(t, binary, "", tc.args...)

			// Then it exits 2, writes the problem to stderr, and writes nothing to stdout
			requireExit(t, result, 2)
			if result.stdout != "" {
				t.Fatalf("usage error wrote to stdout: %s", result.stdout)
			}
			if result.stderr == "" {
				t.Fatal("usage error wrote nothing to stderr")
			}
		})
	}
}

func acceptanceVersionIgnoresModels(t *testing.T, binary string) {
	// Given a directory whose only model file cannot be read
	dir := t.TempDir()
	locked := writeFile(t, dir, "locked.ad.hcl", "design \"locked\" {}\n")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}

	// When the version command runs from that directory
	result := runCLI(t, binary, dir, "version")

	// Then it exits 0 and prints the injected release version
	requireExit(t, result, 0)
	if got := strings.TrimSpace(result.stdout); got != "adhcl "+acceptanceRelease {
		t.Fatalf("version output = %q, want %q", got, "adhcl "+acceptanceRelease)
	}
}

func acceptanceValidationHeader(t *testing.T, binary string) {
	// Given the starter design, which has no setup problems under the source profile
	design := starterPath(t, "design.ad.hcl")

	// When it is validated with the default selections
	result := runCLI(t, binary, "", "validate", "--profile", "source", design)

	// Then the run succeeds and the header names the release, profile, candidate, and context
	requireExit(t, result, 0)
	requireField(t, result.stdout, "Tool version", acceptanceRelease)
	requireField(t, result.stdout, "Profile", "source")
	requireField(t, result.stdout, "Candidate", "default")
	requireField(t, result.stdout, "Context", "none")
}

func acceptanceOutputFailure(t *testing.T, binary string) {
	// Given a standard output device that fails every write
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full is not available")
	}
	full, err := os.OpenFile("/dev/full", os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer full.Close()
	design := starterPath(t, "design.ad.hcl")

	// When each successful command writes its output to that device
	for _, args := range [][]string{{"version"}, {"validate", "--profile", "source", design}} {
		result := execCLI(t, binary, "", full, args...)

		// Then the command exits 1 and reports the delivery failure on stderr
		requireExit(t, result, 1)
		if result.stderr == "" {
			t.Fatalf("%v wrote nothing to stderr after a delivery failure", args)
		}
	}
}

func acceptanceDashPath(t *testing.T, binary string) {
	// Given a model file whose name starts with a dash in the working directory
	dir := t.TempDir()
	writeFile(t, dir, "-model.ad.hcl", readFile(t, starterPath(t, "design.ad.hcl")))

	// When it is validated after the double dash
	result := runCLI(t, binary, dir, "validate", "--", "-model.ad.hcl")

	// Then the file is read as the model and the run succeeds
	requireExit(t, result, 0)
	if !strings.HasSuffix(reportField(t, result.stdout, "Input"), "-model.ad.hcl") {
		t.Fatalf("Input = %q, want the dash-named model", reportField(t, result.stdout, "Input"))
	}
}

func acceptanceForbiddenExpression(t *testing.T, binary string) {
	// Given a model whose design boundary is an arithmetic expression
	path := writeFile(t, t.TempDir(), "expression.ad.hcl", "design \"receipt\" {\n  boundary = 1 + 2\n}\n")

	// When it is validated
	result := runCLI(t, binary, "", "validate", path)

	// Then it exits 1 with an invalid syntax error that names the file and position
	requireExit(t, result, 1)
	positioned := regexp.MustCompile(`(?m)^- error invalid_syntax on .* at \S*expression\.ad\.hcl:\d+:\d+: `)
	if !positioned.MatchString(result.stdout) {
		t.Fatalf("report has no positioned invalid_syntax error; stdout:\n%s", result.stdout)
	}
}

func acceptanceRejectedModels(t *testing.T, binary string) {
	// Given conformance inputs that the core pipeline must reject
	cases := []struct {
		name string
		file string
		code string
	}{
		{"wrong reference kind", "wrong-kind.ad.project.hcl", "invalid_reference"},
		{"ambiguous pairing", "ambiguous.ad.hcl", "ambiguous_pairing"},
		{"invalid schema", "invalid-schema.ad.project.hcl", "invalid_structure"},
		{"cross file duplicate", "duplicate.ad.project.hcl", "duplicate_identity"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the input is validated under the default profile
			result := runCLI(t, binary, "", "validate", conformancePath(t, tc.file))

			// Then it exits 1 and reports the expected error code
			requireExit(t, result, 1)
			requireFinding(t, result.stdout, "error", tc.code)
		})
	}
}

func acceptanceExactUnknown(t *testing.T, binary string) {
	// Given a model whose absent influence is an exact unknown beside a default zero
	project := conformancePath(t, "exact-unknown.ad.project.hcl")
	cases := []struct {
		name     string
		profile  string
		code     int
		severity string
	}{
		{"source profile", "source", 0, "warning"},
		{"architecture profile", "architecture", 1, "error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the model is validated under the profile
			result := runCLI(t, binary, "", "validate", "--profile", tc.profile, project)

			// Then the exit code follows the profile and the unknown influence keeps its severity
			requireExit(t, result, tc.code)
			requireFinding(t, result.stdout, tc.severity, "unresolved_influence")
		})
	}
}

func acceptanceSplitDirectory(t *testing.T, binary string) {
	// Given the starter design and the same design split into two files at the decision boundary
	source := readFile(t, starterPath(t, "design.ad.hcl"))
	marker := `dp "amount_admission" {`
	cut := strings.Index(source, marker)
	if cut < 0 {
		t.Fatalf("starter design has no %q block", marker)
	}
	first, second := source[:cut], source[cut:]
	whole := runCLI(t, binary, "", "validate", "--profile", "source", starterPath(t, "design.ad.hcl"))
	requireExit(t, whole, 0)
	wantCodes := findingCodes(whole.stdout)

	// When the two parts are stored under two names, once in each name order
	layouts := []map[string]string{
		{"a.ad.hcl": first, "z.ad.hcl": second},
		{"a.ad.hcl": second, "z.ad.hcl": first},
	}
	for _, files := range layouts {
		dir := t.TempDir()
		for name, content := range files {
			writeFile(t, dir, name, content)
		}
		result := runCLI(t, binary, "", "validate", "--profile", "source", dir)

		// Then each order exits like the single file and keeps the same finding codes
		requireExit(t, result, 0)
		if got := findingCodes(result.stdout); !slices.Equal(got, wantCodes) {
			t.Fatalf("finding codes = %v, want %v", got, wantCodes)
		}
	}
}

func acceptanceCandidateSelection(t *testing.T, binary string) {
	// Given the self-design example with a named candidate and a named context
	const candidateRef = "decision.assessment_contracts.candidate.go_hcl_target"
	const contextRef = "context.semantic_changes"
	model := repoPath(t, "testdata", "spec", "examples", "requirements_hcl.ad.hcl")

	// When the candidate and context are selected under the source profile
	result := runCLI(t, binary, "", "validate", "--profile", "source",
		"--candidate", candidateRef, "--context", contextRef, model)

	// Then the run succeeds and the header names the selected candidate and context
	requireExit(t, result, 0)
	requireField(t, result.stdout, "Candidate", candidateRef)
	requireField(t, result.stdout, "Context", contextRef)
}

func acceptanceLayoutChange(t *testing.T, binary string) {
	// Given the starter project with its permitted layout
	baselineViews := readFile(t, starterPath(t, "views.ad.ext.hcl"))
	baselineDir := starterProjectDir(t, baselineViews)
	baseline := runCLI(t, binary, "", "validate", "--profile", "source",
		filepath.Join(baselineDir, "model.ad.project.hcl"))
	requireExit(t, baseline, 0)

	// When only the view layout changes to the other permitted layout
	changedViews := strings.Replace(baselineViews, `layout = "requirements_first"`, `layout = "matrices_first"`, 1)
	if changedViews == baselineViews {
		t.Fatal("starter views have no requirements_first layout to change")
	}
	changedDir := starterProjectDir(t, changedViews)
	result := runCLI(t, binary, "", "validate", "--profile", "source",
		filepath.Join(changedDir, "model.ad.project.hcl"))

	// Then the run still exits 0 with the same finding codes
	requireExit(t, result, 0)
	if got, want := findingCodes(result.stdout), findingCodes(baseline.stdout); !slices.Equal(got, want) {
		t.Fatalf("finding codes = %v, want %v", got, want)
	}
}

func acceptanceModuleContract(t *testing.T, binary string) {
	// Given the conformance baseline with its left realization module removed
	base := readFile(t, conformancePath(t, "base.ad.hcl"))
	model := writeFile(t, t.TempDir(), "no-module.ad.hcl", withoutBlock(t, base, `module "left" {`))
	cases := []struct {
		name     string
		profile  string
		code     int
		severity string
	}{
		{"source profile", "source", 0, "warning"},
		{"architecture profile", "architecture", 1, "error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the model is validated under the profile
			result := runCLI(t, binary, "", "validate", "--profile", tc.profile, model)

			// Then the exit code follows the profile and the missing contract keeps its severity
			requireExit(t, result, tc.code)
			requireFinding(t, result.stdout, tc.severity, "incomplete_module_contract")
		})
	}
}

func acceptanceIncompatibleHorizons(t *testing.T, binary string) {
	// Given a decision whose two candidates declare different horizons with their own evidence
	path := writeFile(t, t.TempDir(), "horizons.ad.hcl", acceptanceHorizonModel)

	// When the integer candidate is selected under the source profile
	result := runCLI(t, binary, "", "validate", "--profile", "source",
		"--candidate", "decision.receipt_mechanism.candidate.integer_cents", path)

	// Observed required failures refute feasibility; different horizons still prevent comparison.
	requireExit(t, result, 1)
	requireFinding(t, result.stdout, "error", "infeasible_candidate")
	requireFinding(t, result.stdout, "warning", "incomparable_evidence")
}

func acceptanceInvalidAttachment(t *testing.T, binary string) {
	// Given a valid core model with an invalid required view attachment
	manifest := conformancePath(t, "invalid-attachment.ad.project.hcl")

	// When it is validated under the source profile
	result := runCLI(t, binary, "", "validate", "--profile", "source", manifest)

	// Then the command exits 1, reports the attachment error, and still prints the core scope
	requireExit(t, result, 1)
	requireFinding(t, result.stdout, "error", "invalid_attachment")
	if !strings.Contains(result.stdout, "- Scope design.two_outputs:") {
		t.Fatalf("report has no core scope line; stdout:\n%s", result.stdout)
	}
}

func acceptanceManifestSymlink(t *testing.T, binary string) {
	// Given a manifest whose core file is a symbolic link to a model outside its directory
	dir := t.TempDir()
	if err := os.Symlink(starterPath(t, "design.ad.hcl"), filepath.Join(dir, "core.ad.hcl")); err != nil {
		t.Fatal(err)
	}
	manifest := writeFile(t, dir, "model.ad.project.hcl",
		"project \"escape\" {\n  language = \"requirements_hcl/core/1\"\n  core_files = [\"core.ad.hcl\"]\n  attachment_files = []\n}\n")

	// When the manifest is validated
	result := runCLI(t, binary, "", "validate", manifest)

	// Then the input fails containment and the command exits 1
	requireExit(t, result, 1)
	requireCode(t, result.stdout, "input_failure")
}

func acceptanceFailedExpectation(t *testing.T, binary string) {
	// Given the starter design with an added expectation whose order breaks precedence
	expectation := "\nexpect \"receipt_order\" {\n" +
		"  scope          = design.receipt\n" +
		"  classification = \"decoupled\"\n" +
		"  order          = [dp.exact_addition, dp.amount_admission]\n" +
		"}\n"
	path := writeFile(t, t.TempDir(), "expectation.ad.hcl", readFile(t, starterPath(t, "design.ad.hcl"))+expectation)

	// When it is validated under the source profile
	result := runCLI(t, binary, "", "validate", "--profile", "source", path)

	// Then the command exits 1 with a failed expectation error
	requireExit(t, result, 1)
	requireFinding(t, result.stdout, "error", "failed_expectation")
}

func acceptanceReleaseExamples(t *testing.T, binary string) {
	// Given the three release examples and the profile each one is documented with
	cases := []struct {
		name string
		args []string
	}{
		{"minimal", []string{repoPath(t, "examples", "minimal.ad.hcl")}},
		{"architecture", []string{"--profile", "architecture", repoPath(t, "examples", "architecture.ad.hcl")}},
		{"project", []string{repoPath(t, "examples", "project", "model.ad.project.hcl")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the example is validated
			result := runCLI(t, binary, "", append([]string{"validate"}, tc.args...)...)

			// Then it exits 0 and reports no error finding
			requireExit(t, result, 0)
			if findingPattern("error", `[a-z_]+`).MatchString(result.stdout) {
				t.Fatalf("example reported an error finding; stdout:\n%s", result.stdout)
			}
		})
	}
}

// runCLI runs the binary in dir and captures its standard output.
func runCLI(t *testing.T, binary, dir string, args ...string) acceptanceResult {
	t.Helper()
	var stdout bytes.Buffer
	result := execCLI(t, binary, dir, &stdout, args...)
	result.stdout = stdout.String()
	return result
}

// execCLI runs the binary in dir, sends standard output to stdout, and captures standard error.
func execCLI(t *testing.T, binary, dir string, stdout io.Writer, args ...string) acceptanceResult {
	t.Helper()
	var stderr bytes.Buffer
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run %s %v: %v", binary, args, err)
		}
		code = exitErr.ExitCode()
	}
	return acceptanceResult{stderr: stderr.String(), code: code}
}

func requireExit(t *testing.T, result acceptanceResult, want int) {
	t.Helper()
	if result.code != want {
		t.Fatalf("exit code = %d, want %d; stderr:\n%s", result.code, want, result.stderr)
	}
}

// reportField returns the value of the first report line with the given name.
func reportField(t *testing.T, stdout, name string) string {
	t.Helper()
	for _, line := range strings.Split(stdout, "\n") {
		if value, ok := strings.CutPrefix(line, name+": "); ok {
			return value
		}
	}
	t.Fatalf("report has no %s line; stdout:\n%s", name, stdout)
	return ""
}

func requireField(t *testing.T, stdout, name, want string) {
	t.Helper()
	if got := reportField(t, stdout, name); got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}

var findingCodePattern = regexp.MustCompile(`(?m)^- (?:error|warning) (\S+) on `)

// findingCodes returns the sorted, unique finding codes in a report.
func findingCodes(stdout string) []string {
	seen := map[string]bool{}
	var codes []string
	for _, match := range findingCodePattern.FindAllStringSubmatch(stdout, -1) {
		if !seen[match[1]] {
			seen[match[1]] = true
			codes = append(codes, match[1])
		}
	}
	slices.Sort(codes)
	return codes
}

// findingPattern matches a report finding line with the given severity and code.
func findingPattern(severity, code string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^- ` + regexp.QuoteMeta(severity) + ` ` + code + ` on `)
}

func requireFinding(t *testing.T, stdout, severity, code string) {
	t.Helper()
	if !findingPattern(severity, regexp.QuoteMeta(code)).MatchString(stdout) {
		t.Fatalf("report has no %s %s finding; stdout:\n%s", severity, code, stdout)
	}
}

func requireCode(t *testing.T, stdout, code string) {
	t.Helper()
	if !slices.Contains(findingCodes(stdout), code) {
		t.Fatalf("report has no %s finding; stdout:\n%s", code, stdout)
	}
}

// repoPath returns an absolute path under the repository root.
func repoPath(t *testing.T, parts ...string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(append([]string{"..", ".."}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func starterPath(t *testing.T, name string) string {
	t.Helper()
	return repoPath(t, "testdata", "spec", "examples", "starter", name)
}

func conformancePath(t *testing.T, name string) string {
	t.Helper()
	return repoPath(t, "testdata", "spec", "conformance", "core-1", name)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// writeFile writes content to name inside dir and returns the new path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// starterProjectDir copies the starter project into a new directory, using views as its view file.
func starterProjectDir(t *testing.T, views string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"model.ad.project.hcl", "design.ad.hcl", "bindings.ad.ext.hcl", "behavior.ad.ext.hcl"} {
		writeFile(t, dir, name, readFile(t, starterPath(t, name)))
	}
	writeFile(t, dir, "views.ad.ext.hcl", views)
	return dir
}

// withoutBlock returns source with the top-level block that starts with header removed.
func withoutBlock(t *testing.T, source, header string) string {
	t.Helper()
	const closer = "\n}\n"
	start := strings.Index(source, header)
	if start < 0 {
		t.Fatalf("source has no block starting with %q", header)
	}
	end := strings.Index(source[start:], closer)
	if end < 0 {
		t.Fatalf("block %q has no closing brace", header)
	}
	return source[:start] + source[start+end+len(closer):]
}
