package validate_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/owners"
	"github.com/axiomatic-design-hcl/adhcl/internal/report"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/validate"
	"github.com/hashicorp/hcl/v2"
)

const (
	conformanceCoreDir        = "../../testdata/spec/conformance/core-1"
	conformanceExpectedPath   = conformanceCoreDir + "/expected.json"
	conformanceSelfDesignPath = conformanceCoreDir + "/self-design-inspection.json"
	conformanceProfileSource  = "source"
	conformanceProfileArch    = "architecture"
	conformanceStageCore      = "core_assessment"
	conformanceStageCompose   = "composition"
	conformanceStageOwnerPre  = "owner:"
	conformanceExtensionPre   = "extension."
	conformanceNotRequested   = "not_requested"
	conformanceOpSuccess      = "success"
	conformanceOpWarnings     = "success_with_warnings"
	conformanceOpFailure      = "failure"
	conformanceCodeInvalidAtt = "invalid_attachment"
)

// conformanceFile mirrors the top level of expected.json.
type conformanceFile struct {
	Cases []conformanceCase `json:"cases"`
}

// conformanceCase is one entry in expected.json.
type conformanceCase struct {
	ID     string                 `json:"id"`
	Input  string                 `json:"input"`
	Expect conformanceExpectation `json:"expected"`
}

// conformanceExpectation holds the fixture expectations for one case.
type conformanceExpectation struct {
	Stages struct {
		CoreAssessment string          `json:"core_assessment"`
		Owners         json.RawMessage `json:"owners"`
		Composition    string          `json:"composition"`
	} `json:"stages"`
	Operation                 map[string]string            `json:"operation"`
	ConditionalClassification string                       `json:"conditional_classification"`
	RequiredFindings          []conformanceFinding         `json:"required_findings"`
	PermittedAdditionalCodes  []string                     `json:"permitted_additional_codes"`
	Scope                     string                       `json:"scope"`
	Pairing                   map[string]string            `json:"pairing"`
	CompletePairings          []map[string]string          `json:"complete_pairings"`
	Matrix                    map[string]map[string]string `json:"matrix"`
	ValidOrder                []string                     `json:"valid_order"`
	CoreMatrixUnchangedFrom   string                       `json:"core_matrix_unchanged_from"`
	OwnerResults              map[string]conformanceOwner  `json:"owner_results"`
	ExactUnknown              *conformanceExactUnknown     `json:"exact_unknown"`
}

// conformanceFinding is one required finding. Its subject comes from subject, at.block, or target.
type conformanceFinding struct {
	Code     string            `json:"code"`
	Severity map[string]string `json:"severity"`
	Subject  string            `json:"subject"`
	At       *conformanceAt    `json:"at"`
	Target   string            `json:"target"`
	Related  []string          `json:"related_files"`
}

// conformanceAt is a source locator. Lines are not part of the fixture.
type conformanceAt struct {
	File  string `json:"file"`
	Block string `json:"block"`
	Field string `json:"field"`
}

// conformanceOwner is one owner result expectation.
type conformanceOwner struct {
	Stage        string            `json:"stage"`
	Subject      string            `json:"subject"`
	Outcome      string            `json:"outcome"`
	PayloadValid *bool             `json:"payload_valid"`
	Outcomes     []json.RawMessage `json:"outcomes"`
}

// conformanceExactUnknown describes the exact unknown cell in the exact-unknown case.
type conformanceExactUnknown struct {
	File           string `json:"file"`
	From           string `json:"from"`
	To             string `json:"to"`
	DefaultApplied bool   `json:"default_applied"`
	Conflict       bool   `json:"conflict"`
}

// conformanceSelfDesign mirrors self-design-inspection.json.
type conformanceSelfDesign struct {
	Input     string `json:"input"`
	Selection struct {
		Candidate model.ID `json:"candidate"`
		Context   model.ID `json:"context"`
	} `json:"selection"`
	ConditionalClassifications map[string]string `json:"conditional_classifications"`
	ExpectedGaps               []conformanceGap  `json:"expected_gaps"`
	ExpectedOperation          map[string]string `json:"expected_operation"`
}

// conformanceGap is one expected gap in the self-design inspection.
type conformanceGap struct {
	Code            string            `json:"code"`
	Severity        map[string]string `json:"severity"`
	Subjects        []string          `json:"subjects"`
	SubjectsInclude []string          `json:"subjects_include"`
}

// conformanceLoadExpected reads expected.json at test time.
func conformanceLoadExpected(t *testing.T) []conformanceCase {
	t.Helper()
	raw, err := os.ReadFile(conformanceExpectedPath)
	if err != nil {
		t.Fatalf("read %s: %v", conformanceExpectedPath, err)
	}
	var file conformanceFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode %s: %v", conformanceExpectedPath, err)
	}
	if len(file.Cases) == 0 {
		t.Fatalf("%s has no cases", conformanceExpectedPath)
	}
	return file.Cases
}

// conformanceFindCase returns the fixture case with the given ID.
func conformanceFindCase(t *testing.T, id string) conformanceCase {
	t.Helper()
	for _, c := range conformanceLoadExpected(t) {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("fixture has no case %q", id)
	return conformanceCase{}
}

// conformanceLoadSelfDesign reads self-design-inspection.json at test time.
func conformanceLoadSelfDesign(t *testing.T) conformanceSelfDesign {
	t.Helper()
	raw, err := os.ReadFile(conformanceSelfDesignPath)
	if err != nil {
		t.Fatalf("read %s: %v", conformanceSelfDesignPath, err)
	}
	var spec conformanceSelfDesign
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("decode %s: %v", conformanceSelfDesignPath, err)
	}
	return spec
}

// conformanceRequest builds the request for one profile, path, and selection.
func conformanceRequest(path, profile string, candidate, context model.ID) validate.Request {
	return validate.Request{
		Path:      path,
		Profile:   profile,
		Candidate: string(candidate),
		Context:   string(context),
		Version:   "conformance-test",
	}
}

// conformanceCasePath resolves a fixture input relative to the core-1 directory.
func conformanceCasePath(input string) string {
	return filepath.Join(conformanceCoreDir, input)
}

// conformanceRunCase assembles one fixture case for one profile.
func conformanceRunCase(c conformanceCase, profile string) validate.Result {
	return validate.Run(conformanceRequest(conformanceCasePath(c.Input), profile, "", ""))
}

// conformanceExitFor maps a fixture operation word to its exit code.
func conformanceExitFor(word string) (int, bool) {
	switch word {
	case conformanceOpSuccess, conformanceOpWarnings:
		return 0, true
	case conformanceOpFailure:
		return 1, true
	}
	return 0, false
}

// conformanceSeverity reads the expected severity for one profile.
func conformanceSeverity(t *testing.T, severities map[string]string, profile string) result.Severity {
	t.Helper()
	switch severities[profile] {
	case "error":
		return result.Error
	case "warning":
		return result.Warning
	}
	t.Fatalf("fixture has no severity for profile %q", profile)
	return ""
}

// conformanceEffectWord names an effect the way the fixture does.
func conformanceEffectWord(effect assessment.Effect) string {
	switch effect {
	case assessment.EffectMaterial:
		return "material"
	case assessment.EffectNone:
		return "none"
	case assessment.EffectUnknown:
		return "unknown"
	}
	return string(effect)
}

// conformanceFileBase returns the file name of a range, or empty when none is set.
func conformanceFileBase(rng *hcl.Range) string {
	if rng == nil {
		return ""
	}
	return filepath.Base(rng.Filename)
}

// conformanceHasFinding reports whether a finding exists with the code and severity.
// An empty subject or file matches any subject or file.
func conformanceHasFinding(findings []result.Finding, code, subject, file string, severity result.Severity) bool {
	for _, f := range findings {
		if f.Code != code || f.Severity != severity {
			continue
		}
		if subject != "" && f.Subject != subject {
			continue
		}
		if file != "" && conformanceFileBase(f.Range) != file {
			continue
		}
		return true
	}
	return false
}

// conformanceCodes returns the set of codes present in the findings.
func conformanceCodes(findings []result.Finding) map[string]bool {
	out := map[string]bool{}
	for _, f := range findings {
		out[f.Code] = true
	}
	return out
}

// conformanceScope returns the scope result for one scope ID.
func conformanceScope(t *testing.T, rep validate.Result, scope string) assessment.ScopeResult {
	t.Helper()
	for _, s := range rep.Scopes {
		if string(s.Matrix.Scope) == scope {
			return s
		}
	}
	t.Fatalf("report has no scope %q", scope)
	return assessment.ScopeResult{}
}

// conformanceStage returns the stage with the given name.
func conformanceStage(rep validate.Result, name string) (validate.Stage, bool) {
	for _, s := range rep.Stages {
		if s.Name == name {
			return s, true
		}
	}
	return validate.Stage{}, false
}

// conformanceMatrixMatches checks every expected effect in a fixture matrix.
func conformanceMatrixMatches(t *testing.T, got assessment.Matrix, want map[string]map[string]string) {
	t.Helper()
	if len(got.FRs) != len(want) {
		t.Fatalf("matrix has %d functional requirements, want %d", len(got.FRs), len(want))
	}
	for fr, row := range want {
		for dp, word := range row {
			cell := got.Cell(model.ID(fr), model.ID(dp))
			if conformanceEffectWord(cell.Effect) != word {
				t.Fatalf("cell %s and %s effect = %s, want %s", fr, dp, conformanceEffectWord(cell.Effect), word)
			}
		}
	}
}

// conformanceSameMatrix compares two matrices by axes and cell outcomes, not by source ranges.
func conformanceSameMatrix(a, b assessment.Matrix) bool {
	if len(a.FRs) != len(b.FRs) || len(a.DPs) != len(b.DPs) {
		return false
	}
	for i := range a.FRs {
		if a.FRs[i] != b.FRs[i] {
			return false
		}
	}
	for i := range a.DPs {
		if a.DPs[i] != b.DPs[i] {
			return false
		}
	}
	for _, fr := range a.FRs {
		for _, dp := range a.DPs {
			ca, cb := a.Cell(fr, dp), b.Cell(fr, dp)
			if ca.Effect != cb.Effect || ca.Conflict != cb.Conflict || ca.FromDefault != cb.FromDefault ||
				ca.Conditional != cb.Conditional || ca.Unresolved != cb.Unresolved ||
				len(ca.Exact) != len(cb.Exact) || len(ca.Unknowns) != len(cb.Unknowns) {
				return false
			}
		}
	}
	return true
}

// conformancePairingEquals compares a pairing with a fixture pairing.
func conformancePairingEquals(got map[model.ID]model.ID, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for fr, dp := range want {
		if got[model.ID(fr)] != model.ID(dp) {
			return false
		}
	}
	return true
}

// conformanceOrderIsValid reports whether an order respects the matrix. A material effect from a
// requirement onto another design parameter must come earlier in the order than the requirement's own.
func conformanceOrderIsValid(order []model.ID, pairing map[model.ID]model.ID, matrix assessment.Matrix) bool {
	if len(order) != len(pairing) {
		return false
	}
	position := map[model.ID]int{}
	for i, dp := range order {
		position[dp] = i
	}
	for fr, own := range pairing {
		ownPos, ok := position[own]
		if !ok {
			return false
		}
		for _, dp := range matrix.DPs {
			if dp == own || matrix.Cell(fr, dp).Effect != assessment.EffectMaterial {
				continue
			}
			pos, present := position[dp]
			if !present || pos >= ownPos {
				return false
			}
		}
	}
	return true
}

// conformanceCheckFindings checks the required findings, their severities, and the allowed codes.
func conformanceCheckFindings(t *testing.T, want conformanceExpectation, findings []result.Finding, profile string) {
	t.Helper()
	allowed := map[string]bool{}
	for _, code := range want.PermittedAdditionalCodes {
		allowed[code] = true
	}
	for _, req := range want.RequiredFindings {
		allowed[req.Code] = true
	}
	for code := range conformanceCodes(findings) {
		if !allowed[code] {
			t.Fatalf("finding code %q is neither required nor permitted", code)
		}
	}
	for _, req := range want.RequiredFindings {
		severity := conformanceSeverity(t, req.Severity, profile)
		subject, file := req.Subject, ""
		if req.At != nil {
			subject, file = req.At.Block, req.At.File
		}
		if req.Target != "" {
			subject = req.Target
		}
		if !conformanceHasFinding(findings, req.Code, subject, file, severity) {
			t.Fatalf("missing required finding %s on %q in %q with the expected severity", req.Code, subject, file)
		}
		if len(req.Related) > 0 {
			conformanceCheckRelatedFiles(t, findings, req)
		}
	}
}

// conformanceCheckRelatedFiles checks that a finding names every related file.
func conformanceCheckRelatedFiles(t *testing.T, findings []result.Finding, req conformanceFinding) {
	t.Helper()
	for _, f := range findings {
		if f.Code != req.Code || f.Subject != req.Subject {
			continue
		}
		files := map[string]bool{conformanceFileBase(f.Range): true}
		for i := range f.Related {
			files[filepath.Base(f.Related[i].Filename)] = true
		}
		covered := true
		for _, want := range req.Related {
			covered = covered && files[want]
		}
		if covered {
			return
		}
	}
	t.Fatalf("no %s finding on %q names every file in %v", req.Code, req.Subject, req.Related)
}

// conformanceCheckOwnerStages checks owner stage statuses. A missing owner list means owners were not requested.
func conformanceCheckOwnerStages(t *testing.T, rep validate.Result, want conformanceExpectation) {
	t.Helper()
	if len(want.Stages.Owners) == 0 || string(want.Stages.Owners) == `"`+conformanceNotRequested+`"` {
		for _, s := range rep.Stages {
			if strings.HasPrefix(s.Name, conformanceStageOwnerPre) && string(s.Status) != conformanceNotRequested {
				t.Fatalf("owner stage %q has status %s, want not requested", s.Name, s.Status)
			}
		}
		return
	}
	var stages map[string]string
	if err := json.Unmarshal(want.Stages.Owners, &stages); err != nil {
		t.Fatalf("decode owner stages: %v", err)
	}
	for label, status := range stages {
		stage, ok := conformanceStage(rep, conformanceStageOwnerPre+label)
		if !ok {
			t.Fatalf("report has no stage for owner %q", label)
		}
		if string(stage.Status) != status {
			t.Fatalf("owner %q stage status = %s, want %s", label, stage.Status, status)
		}
	}
}

// conformanceOwnerResult returns the owner result for a registry label.
func conformanceOwnerResult(t *testing.T, rep validate.Result, label string) owners.OwnerResult {
	t.Helper()
	for _, o := range rep.Owners {
		if strings.TrimPrefix(string(o.Extension), conformanceExtensionPre) == label {
			return o
		}
	}
	t.Fatalf("report has no owner result for %q", label)
	return owners.OwnerResult{}
}

// conformanceCheckOwnerResult checks one owner result against the fixture.
func conformanceCheckOwnerResult(t *testing.T, rep validate.Result, label string, want conformanceOwner) {
	t.Helper()
	got := conformanceOwnerResult(t, rep, label)
	if string(got.Status) != want.Stage {
		t.Fatalf("owner %q status = %s, want %s", label, got.Status, want.Stage)
	}
	if want.PayloadValid != nil && !*want.PayloadValid && !conformanceHasInvalidAttachment(got.Findings) {
		t.Fatalf("owner %q payload is invalid but reports no invalid attachment", label)
	}
	if want.PayloadValid != nil && *want.PayloadValid {
		conformanceCheckValidPayload(t, label, got, want)
		return
	}
	if want.Subject == "" && want.Outcome == "" {
		if len(got.Results) != 0 {
			t.Fatalf("owner %q has %d outcomes, want none", label, len(got.Results))
		}
		return
	}
	if len(got.Results) != 1 {
		t.Fatalf("owner %q has %d outcomes, want 1", label, len(got.Results))
	}
	outcome := got.Results[0]
	if string(outcome.Subject) != want.Subject {
		t.Fatalf("owner %q outcome subject = %s, want %s", label, outcome.Subject, want.Subject)
	}
	if string(outcome.Outcome) != want.Outcome {
		t.Fatalf("owner %q outcome = %s, want %s", label, outcome.Outcome, want.Outcome)
	}
	if strings.TrimSpace(outcome.Cause) == "" {
		t.Fatalf("owner %q outcome has no cause", label)
	}
}

// conformanceCheckValidPayload checks an owner whose payload is valid. It needs at least one valid
// attachment result and no invalid attachment finding. Results that are not valid must match the fixture outcomes.
func conformanceCheckValidPayload(t *testing.T, label string, got owners.OwnerResult, want conformanceOwner) {
	t.Helper()
	validCount := 0
	otherCount := 0
	for _, r := range got.Results {
		if r.Outcome == owners.OutcomeValid {
			validCount++
		} else {
			otherCount++
		}
	}
	if validCount == 0 {
		t.Fatalf("owner %q payload is valid but has no valid result", label)
	}
	if conformanceHasInvalidAttachment(got.Findings) {
		t.Fatalf("owner %q payload is valid but reports an invalid attachment", label)
	}
	if otherCount != len(want.Outcomes) {
		t.Fatalf("owner %q has %d outcomes other than valid, want %d", label, otherCount, len(want.Outcomes))
	}
}

// conformanceHasInvalidAttachment reports whether an owner finding list has an invalid attachment.
func conformanceHasInvalidAttachment(findings []result.Finding) bool {
	for _, f := range findings {
		if f.Code == conformanceCodeInvalidAtt {
			return true
		}
	}
	return false
}

// conformanceCheckScope checks the pairing, matrix, order, and classification of the named scope.
func conformanceCheckScope(t *testing.T, rep validate.Result, want conformanceExpectation) {
	t.Helper()
	scope := conformanceScope(t, rep, want.Scope)
	if want.CompletePairings != nil {
		conformanceCheckAmbiguousScope(t, scope, want)
		return
	}
	if want.ConditionalClassification != "" && want.ConditionalClassification != "not_assessed" &&
		string(scope.Classification) != want.ConditionalClassification {
		t.Fatalf("scope %s classification = %s, want %s", want.Scope, scope.Classification, want.ConditionalClassification)
	}
	if want.Pairing != nil && !conformancePairingEquals(scope.Pairing, want.Pairing) {
		t.Fatalf("scope %s pairing = %v, want %v", want.Scope, scope.Pairing, want.Pairing)
	}
	if want.Matrix != nil {
		conformanceMatrixMatches(t, scope.Matrix, want.Matrix)
	}
	if len(want.ValidOrder) > 0 && !conformanceOrderIsValid(scope.Order, scope.Pairing, scope.Matrix) {
		t.Fatalf("scope %s order %v is not valid for its matrix", want.Scope, scope.Order)
	}
}

// conformanceCheckAmbiguousScope checks an ambiguous scope: no selected pairing, both complete pairings, and unresolved classification.
func conformanceCheckAmbiguousScope(t *testing.T, scope assessment.ScopeResult, want conformanceExpectation) {
	t.Helper()
	if len(scope.Pairing) != 0 {
		t.Fatalf("scope %s selected a pairing %v, want none", want.Scope, scope.Pairing)
	}
	if string(scope.Classification) != want.ConditionalClassification {
		t.Fatalf("scope %s classification = %s, want %s", want.Scope, scope.Classification, want.ConditionalClassification)
	}
	if len(scope.PairingWitnesses) != len(want.CompletePairings) {
		t.Fatalf("scope %s has %d complete pairings, want %d", want.Scope, len(scope.PairingWitnesses), len(want.CompletePairings))
	}
	for _, wantPairing := range want.CompletePairings {
		matched := false
		for _, got := range scope.PairingWitnesses {
			matched = matched || conformancePairingEquals(got, wantPairing)
		}
		if !matched {
			t.Fatalf("scope %s is missing complete pairing %v", want.Scope, wantPairing)
		}
	}
}

// conformanceCheckExactUnknown checks the exact unknown cell is not from a default and has no conflict.
func conformanceCheckExactUnknown(t *testing.T, rep validate.Result, want conformanceExpectation, exact conformanceExactUnknown) {
	t.Helper()
	cell := conformanceScope(t, rep, want.Scope).Matrix.Cell(model.ID(exact.To), model.ID(exact.From))
	if cell.Effect != assessment.EffectUnknown {
		t.Fatalf("cell %s and %s effect is not unknown", exact.To, exact.From)
	}
	if cell.FromDefault != exact.DefaultApplied {
		t.Fatalf("cell %s and %s default applied = %v, want %v", exact.To, exact.From, cell.FromDefault, exact.DefaultApplied)
	}
	if cell.Conflict != exact.Conflict {
		t.Fatalf("cell %s and %s conflict = %v, want %v", exact.To, exact.From, cell.Conflict, exact.Conflict)
	}
}

// TestConformanceCasesMatchFixture runs every expected.json case under both profiles and checks the fixture expectations.
func TestConformanceCasesMatchFixture(t *testing.T) {
	for _, c := range conformanceLoadExpected(t) {
		for _, profile := range []string{conformanceProfileSource, conformanceProfileArch} {
			t.Run(c.ID+"/"+profile, func(t *testing.T) {
				// Given a fixture case and one profile.
				want := c.Expect
				// When the case is assembled into a report.
				rep := conformanceRunCase(c, profile)
				// Then the core and owner stages, findings, scope, and matrix expectations hold.
				if stage, ok := conformanceStage(rep, conformanceStageCore); !ok || string(stage.Status) != want.Stages.CoreAssessment {
					t.Fatalf("core_assessment stage does not match %s", want.Stages.CoreAssessment)
				}
				if stage, ok := conformanceStage(rep, conformanceStageCompose); want.Stages.Composition != "" && (!ok || string(stage.Status) != want.Stages.Composition) {
					t.Fatalf("composition stage does not match %s", want.Stages.Composition)
				}
				conformanceCheckOwnerStages(t, rep, want)
				for label, ownerWant := range want.OwnerResults {
					conformanceCheckOwnerResult(t, rep, label, ownerWant)
				}
				conformanceCheckFindings(t, want, rep.Findings, profile)
				if want.Scope != "" {
					conformanceCheckScope(t, rep, want)
				}
				if want.CoreMatrixUnchangedFrom != "" {
					base := conformanceRunCase(conformanceFindCase(t, want.CoreMatrixUnchangedFrom), profile)
					if !conformanceSameMatrix(conformanceScope(t, rep, want.Scope).Matrix, conformanceScope(t, base, want.Scope).Matrix) {
						t.Fatalf("scope %s matrix differs from the %s matrix", want.Scope, want.CoreMatrixUnchangedFrom)
					}
				}
				if want.ExactUnknown != nil {
					conformanceCheckExactUnknown(t, rep, want, *want.ExactUnknown)
				}
			})
		}
	}
}

// TestConformanceOperationOutcomeIsExitCode checks Run's exit code for each fixture case and profile.
func TestConformanceOperationOutcomeIsExitCode(t *testing.T) {
	for _, c := range conformanceLoadExpected(t) {
		for _, profile := range []string{conformanceProfileSource, conformanceProfileArch} {
			t.Run(c.ID+"/"+profile, func(t *testing.T) {
				// Given a fixture case with an expected operation for the profile.
				word, ok := c.Expect.Operation[profile]
				if !ok {
					t.Fatalf("fixture has no operation for profile %q", profile)
				}
				want, known := conformanceExitFor(word)
				if !known {
					t.Fatalf("unknown operation word %q", word)
				}
				var output bytes.Buffer
				// When the case is run through the pipeline.
				code, err := renderValidation(conformanceRequest(conformanceCasePath(c.Input), profile, "", ""), &output)
				// Then Run returns no error and the expected exit code.
				if err != nil {
					t.Fatalf("renderValidation() error = %v, want nil", err)
				}
				if code != want {
					t.Fatalf("renderValidation() exit code = %d, want %d", code, want)
				}
			})
		}
	}
}

// TestConformanceExitCodeMatchesReport checks that Run's exit code equals ExitCode of the assembled report.
func TestConformanceExitCodeMatchesReport(t *testing.T) {
	for _, c := range conformanceLoadExpected(t) {
		for _, profile := range []string{conformanceProfileSource, conformanceProfileArch} {
			t.Run(c.ID+"/"+profile, func(t *testing.T) {
				// Given a fixture case assembled into a report for one profile.
				rep := conformanceRunCase(c, profile)
				// When the same case is run through Run with an output buffer.
				var output bytes.Buffer
				code, err := renderValidation(conformanceRequest(conformanceCasePath(c.Input), profile, "", ""), &output)
				// Then Run returns no error and the exit code equals the report's exit code.
				if err != nil {
					t.Fatalf("renderValidation() error = %v, want nil", err)
				}
				if want := report.ExitCode(rep); code != want {
					t.Fatalf("renderValidation() exit code = %d, report exit code = %d", code, want)
				}
			})
		}
	}
}

// TestConformanceStarterExternalMatrixMatchesCore checks that external unknowns leave the core design.receipt matrix unchanged.
func TestConformanceStarterExternalMatrixMatchesCore(t *testing.T) {
	// Given the starter core case and the starter project with external unknowns, both under the source profile.
	core := conformanceRunCase(conformanceFindCase(t, "starter_core"), conformanceProfileSource)
	external := conformanceRunCase(conformanceFindCase(t, "starter_project_external_unknown"), conformanceProfileSource)
	// When the design.receipt matrices are compared.
	got := conformanceScope(t, external, "design.receipt").Matrix
	want := conformanceScope(t, core, "design.receipt").Matrix
	// Then the matrices match.
	if !conformanceSameMatrix(got, want) {
		t.Fatal("external unknown matrix differs from the starter core matrix")
	}
}

// TestConformanceSelfDesignInspection checks the self-design inspection fixture under both profiles.
func TestConformanceSelfDesignInspection(t *testing.T) {
	spec := conformanceLoadSelfDesign(t)
	for _, profile := range []string{conformanceProfileSource, conformanceProfileArch} {
		t.Run(profile, func(t *testing.T) {
			// Given the self-design inspection input with its candidate and context selection.
			path := filepath.Join(conformanceCoreDir, spec.Input)
			request := conformanceRequest(path, profile, spec.Selection.Candidate, spec.Selection.Context)
			// When the inspection is assembled.
			rep := validate.Run(request)
			// Then each expected gap appears with its subjects at the profile severity.
			for _, gap := range spec.ExpectedGaps {
				severity := conformanceSeverity(t, gap.Severity, profile)
				for _, subject := range append(append([]string{}, gap.Subjects...), gap.SubjectsInclude...) {
					if !conformanceHasFinding(rep.Findings, gap.Code, subject, "", severity) {
						t.Fatalf("missing %s on %q with the expected severity", gap.Code, subject)
					}
				}
			}
			// And each named scope has the expected classification.
			for scope, classification := range spec.ConditionalClassifications {
				if got := string(conformanceScope(t, rep, scope).Classification); got != classification {
					t.Fatalf("scope %s classification = %s, want %s", scope, got, classification)
				}
			}
			// And the exit code matches the expected operation for the profile.
			word, ok := spec.ExpectedOperation[profile]
			if !ok {
				t.Fatalf("fixture has no operation for profile %q", profile)
			}
			want, known := conformanceExitFor(word)
			if !known {
				t.Fatalf("unknown operation word %q", word)
			}
			if got := report.ExitCode(rep); got != want {
				t.Fatalf("exit code = %d, want %d", got, want)
			}
		})
	}
}
