package assessment

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// profileTestBase is a complete model with zero completeness and zero setup findings.
const profileTestBase = `design "two_outputs" {
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

dp "left_output" {
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

// profileTestCycleModel has two sibling DPs that materially affect each other's paired FR.
const profileTestCycleModel = `design "checkout" {
  boundary = "Record accepted payments and report their total."
}

need "record_payments" {
  statement = "Record every accepted payment."
}

fr "capture" {
  need      = [need.record_payments]
  statement = "Capture each accepted payment."
}

fr "report" {
  need      = [need.record_payments]
  statement = "Report the total of accepted payments."
}

dp "writer" {
  controls       = [fr.capture]
  responsibility = "Write the captured payment."
  decision       = "selected"
}

dp "reader" {
  controls       = [fr.report]
  responsibility = "Read the reported total."
  decision       = "selected"
}

influence {
  from      = [dp.writer]
  to        = fr.report
  effect    = "material"
  mechanism = "Writes change the reported total."
}

influence {
  from      = [dp.reader]
  to        = fr.capture
  effect    = "material"
  mechanism = "Reads change which payments are captured."
}
`

const profileTestCouplingComplete = `coupling "capture_report" {
  scope = design.checkout
  affects = [fr.capture, fr.report, dp.writer, dp.reader]
  reason = "Each total depends on the other writer."
  iteration_bound = 3
  convergence = "The totals settle after one exchange."
  stopping = "Stop after the bound or when both totals match."
  failure_behavior = "Report the cycle as unresolved when iteration cannot finish."
}
`

// profileTestPremiseModel has one material influence that depends on an assumption.
const profileTestPremiseModel = `design "checkout" {
  boundary = "Record accepted payments and report their total."
}

need "record_payments" {
  statement = "Record every accepted payment."
}

fr "capture" {
  need      = [need.record_payments]
  statement = "Capture each accepted payment."
}

fr "report" {
  need      = [need.record_payments]
  statement = "Report the total of accepted payments."
}

dp "writer" {
  controls       = [fr.capture]
  responsibility = "Write the captured payment."
  decision       = "selected"
}

dp "reader" {
  controls       = [fr.report]
  responsibility = "Read the reported total."
  decision       = "selected"
}

assumption "semantic_contracts" {
  statement = "Both writers keep the shared payment record stable."
}

influence {
  from      = [dp.writer]
  to        = fr.report
  effect    = "material"
  mechanism = "Writes change the reported total."
  holds_if  = [assumption.semantic_contracts]
}
`

// profileTestNoInfluenceModel has one leaf FR and one leaf DP and no influences.
const profileTestNoInfluenceModel = `design "checkout" {
  boundary = "Record accepted payments."
}

need "record_payments" {
  statement = "Record every accepted payment."
}

fr "capture" {
  need      = [need.record_payments]
  statement = "Capture each accepted payment."
}

dp "writer" {
  controls       = [fr.capture]
  responsibility = "Write the captured payment."
  decision       = "selected"
}
`

// profileTestSnapshot builds the default selection from inline sources. It fails the test when any setup step reports a finding.
func profileTestSnapshot(t *testing.T, sources ...string) *selection.Snapshot {
	t.Helper()

	units := make([]syntax.Unit, 0, len(sources))
	for index, source := range sources {
		unit := syntax.Parse(model.Source{Path: fmt.Sprintf("profile-%d.ad.hcl", index+1), Bytes: []byte(source)})
		if len(unit.Findings) != 0 {
			t.Fatalf("source %d has %d native parse findings, want none", index+1, len(unit.Findings))
		}
		units = append(units, unit)
	}

	core, decodeFindings := schema.DecodeCore(units)
	if len(decodeFindings) != 0 {
		t.Fatalf("decode produced %d findings, want none", len(decodeFindings))
	}

	graph, resolveFindings := schema.Resolve(core)
	if len(resolveFindings) != 0 {
		t.Fatalf("resolve produced %d findings, want none", len(resolveFindings))
	}
	if graph == nil {
		t.Fatal("resolve returned no graph, want a graph")
	}

	snapshot, selectFindings := selection.Select(graph, selection.Request{})
	if len(selectFindings) != 0 {
		t.Fatalf("selection produced %d findings, want none", len(selectFindings))
	}
	if snapshot == nil {
		t.Fatal("selection returned no snapshot, want a snapshot")
	}
	return snapshot
}

func profileTestSubjects(findings []result.Finding, code string) []string {
	var subjects []string
	for _, finding := range findings {
		if finding.Code == code {
			subjects = append(subjects, finding.Subject)
		}
	}
	return subjects
}

func profileTestRequireSubjects(t *testing.T, label string, got []string, want ...string) {
	t.Helper()

	gotSorted := slices.Clone(got)
	slices.Sort(gotSorted)
	wantSorted := slices.Clone(want)
	slices.Sort(wantSorted)
	if !slices.Equal(gotSorted, wantSorted) {
		t.Fatalf("%s subjects = %v, want %v", label, gotSorted, wantSorted)
	}
}

func profileTestRequireOnlyCode(t *testing.T, findings []result.Finding, code string) {
	t.Helper()

	if len(findings) == 0 {
		t.Fatalf("findings = 0, want at least one %s finding", code)
	}
	for _, finding := range findings {
		if finding.Code != code {
			t.Fatalf("finding code = %q, want only %q", finding.Code, code)
		}
	}
}

func profileTestRequireNoCode(t *testing.T, findings []result.Finding, code string) {
	t.Helper()

	for _, finding := range findings {
		if finding.Code == code {
			t.Fatalf("findings include %q, want none", code)
		}
	}
}

// profileTestReplace swaps one fragment for another. The fragment must occur exactly once.
func profileTestReplace(t *testing.T, source, old, replacement string) string {
	t.Helper()

	if count := strings.Count(source, old); count != 1 {
		t.Fatalf("fragment occurs %d times, want exactly once", count)
	}
	return strings.Replace(source, old, replacement, 1)
}

// profileTestDropLine removes the whole line that contains the fragment. The fragment must occur exactly once.
func profileTestDropLine(t *testing.T, source, fragment string) string {
	t.Helper()

	if count := strings.Count(source, fragment); count != 1 {
		t.Fatalf("fragment occurs %d times, want exactly once", count)
	}
	at := strings.Index(source, fragment)
	lineStart := strings.LastIndex(source[:at], "\n") + 1
	lineEnd := at + strings.Index(source[at:], "\n") + 1
	return source[:lineStart] + source[lineEnd:]
}

// profileTestRemoveBlock removes a top-level block that starts with the header and ends at the next closing brace line.
func profileTestRemoveBlock(t *testing.T, source, header string) string {
	t.Helper()

	if count := strings.Count(source, header); count != 1 {
		t.Fatalf("block header occurs %d times, want exactly once", count)
	}
	start := strings.Index(source, header)
	closing := strings.Index(source[start:], "\n}\n")
	if closing < 0 {
		t.Fatalf("block %q has no closing brace line", header)
	}
	end := start + closing + len("\n}\n")
	return source[:start] + source[end:]
}

// profileTestCycleScopes describes the sibling set of the cycle model with its residual two-DP cycle.
func profileTestCycleScopes() []ScopeResult {
	return []ScopeResult{{
		Matrix: Matrix{Scope: model.ID("design.checkout")},
		Pairing: map[model.ID]model.ID{
			model.ID("fr.capture"): model.ID("dp.writer"),
			model.ID("fr.report"):  model.ID("dp.reader"),
		},
		Cycles: [][]model.ID{{model.ID("dp.writer"), model.ID("dp.reader")}},
	}}
}

func TestProfileCompletenessAcceptsCompleteBaseModel(t *testing.T) {
	// Given the complete base model with every realization, scenario, and criterion field present
	snapshot := profileTestSnapshot(t, profileTestBase)

	// When completeness is checked with no residual cycles
	findings := Completeness(snapshot, nil)

	// Then no completeness finding is reported
	if len(findings) != 0 {
		t.Fatalf("findings = %d, want none", len(findings))
	}
}

func TestProfileCompletenessFlagsQualitativeFRAsIncompleteCriterion(t *testing.T) {
	// Given the base model with the left criterion rewritten as a qualitative statement only
	source := profileTestReplace(t, profileTestBase,
		"    metric = \"For every supplied left string, the returned left string is byte-for-byte equal to it.\"\n    op = \"==\"\n    value = true\n",
		"    statement = \"The returned left string is byte-for-byte equal to the input.\"\n")
	snapshot := profileTestSnapshot(t, source)

	// When completeness is checked
	findings := Completeness(snapshot, nil)

	// Both the qualitative predicate and the missing operational coverage remain visible.
	profileTestRequireSubjects(t, "incomplete_criterion", profileTestSubjects(findings, "incomplete_criterion"), "fr.left", "fr.left.criterion.exact")
}

func TestProfileCompletenessFlagsLeafDPWithoutRealizationModule(t *testing.T) {
	// Given the base model with the left realization module removed
	source := profileTestRemoveBlock(t, profileTestBase, `module "left" {`)
	snapshot := profileTestSnapshot(t, source)

	// When completeness is checked
	findings := Completeness(snapshot, nil)

	// Then the module contract finding names the leaf DP that has no realization module
	profileTestRequireSubjects(t, "incomplete_module_contract", profileTestSubjects(findings, "incomplete_module_contract"), "dp.left_output")
}

func TestProfileCompletenessFlagsModuleWithoutFailureBehavior(t *testing.T) {
	// Given the base model with failure behavior removed from the left realization module only
	source := profileTestReplace(t, profileTestBase,
		"  invariants = [\"The right input and output are not read or written.\"]\n  failure_behavior = \"Do not publish a different string as a successful copy. Process availability is outside this contract.\"\n",
		"  invariants = [\"The right input and output are not read or written.\"]\n")
	snapshot := profileTestSnapshot(t, source)

	// When completeness is checked
	findings := Completeness(snapshot, nil)

	// Then the module contract finding names only the DP whose module lacks failure behavior
	profileTestRequireSubjects(t, "incomplete_module_contract", profileTestSubjects(findings, "incomplete_module_contract"), "dp.left_output")
}

func TestProfileCompletenessFlagsLeafFRsWithoutScenarioSteps(t *testing.T) {
	// Given the base model with the scenario steps removed
	source := profileTestDropLine(t, profileTestBase, `steps = ["Supply empty strings`)
	snapshot := profileTestSnapshot(t, source)

	// When completeness is checked
	findings := Completeness(snapshot, nil)

	// Then both leaf FRs are reported because their verifying scenario has no steps
	profileTestRequireSubjects(t, "missing_leaf_scenario", profileTestSubjects(findings, "missing_leaf_scenario"), "fr.left", "fr.right")
}

func TestProfileCompletenessFlagsLeafFRWhoseScenarioMissesCriterion(t *testing.T) {
	// Given the base model where the scenario verifies only the left criterion
	source := profileTestReplace(t, profileTestBase,
		"verifies = [fr.left.criterion.exact, fr.right.criterion.exact]",
		"verifies = [fr.left.criterion.exact]")
	snapshot := profileTestSnapshot(t, source)

	// When completeness is checked
	findings := Completeness(snapshot, nil)

	// Then only the right FR is reported because no scenario verifies its criterion
	profileTestRequireSubjects(t, "missing_leaf_scenario", profileTestSubjects(findings, "missing_leaf_scenario"), "fr.right")
}

func TestProfileCompletenessFlagsCoupledScopeWithoutCouplingRecord(t *testing.T) {
	// Given a sibling set with a residual material cycle and no coupling record
	snapshot := profileTestSnapshot(t, profileTestCycleModel)

	// When completeness is checked with the residual cycle
	findings := Completeness(snapshot, profileTestCycleScopes())

	// Then the feedback contract finding names the scope that carries the cycle
	profileTestRequireSubjects(t, "incomplete_feedback_contract", profileTestSubjects(findings, "incomplete_feedback_contract"), "design.checkout")
}

func TestProfileCompletenessAcceptsCompleteCouplingRecord(t *testing.T) {
	// Given a sibling set with a residual material cycle and a coupling record that covers every cycle member and field
	snapshot := profileTestSnapshot(t, profileTestCycleModel, profileTestCouplingComplete)

	// When completeness is checked with the residual cycle
	findings := Completeness(snapshot, profileTestCycleScopes())

	// Then no feedback contract finding is reported
	profileTestRequireNoCode(t, findings, "incomplete_feedback_contract")
}

func TestProfileCompletenessFlagsCouplingThatOmitsCycleDP(t *testing.T) {
	// Given a coupling record whose affects list leaves out one DP on the cycle
	coupling := profileTestReplace(t, profileTestCouplingComplete,
		"affects = [fr.capture, fr.report, dp.writer, dp.reader]",
		"affects = [fr.capture, fr.report, dp.writer]")
	snapshot := profileTestSnapshot(t, profileTestCycleModel, coupling)

	// When completeness is checked with the residual cycle
	findings := Completeness(snapshot, profileTestCycleScopes())

	// Then the feedback contract finding names the scope that carries the cycle
	profileTestRequireSubjects(t, "incomplete_feedback_contract", profileTestSubjects(findings, "incomplete_feedback_contract"), "design.checkout")
}

func TestProfileCompletenessFlagsCouplingWithoutFailureBehavior(t *testing.T) {
	// Given a coupling record that covers the cycle but has no failure behavior
	coupling := profileTestDropLine(t, profileTestCouplingComplete, `failure_behavior = "Report the cycle`)
	snapshot := profileTestSnapshot(t, profileTestCycleModel, coupling)

	// When completeness is checked with the residual cycle
	findings := Completeness(snapshot, profileTestCycleScopes())

	// Then the feedback contract finding names the scope that carries the cycle
	profileTestRequireSubjects(t, "incomplete_feedback_contract", profileTestSubjects(findings, "incomplete_feedback_contract"), "design.checkout")
}

func TestProfileSupportGapsReportsAppliedInfluenceWithoutPremises(t *testing.T) {
	// Given the base model whose influences have no premises
	snapshot := profileTestSnapshot(t, profileTestBase)

	// When support gaps are checked
	findings := SupportGaps(snapshot)

	// Then support gap findings are reported and no other code appears
	profileTestRequireOnlyCode(t, findings, "support_gap")
}

func TestProfileSupportGapsNamesPremiseOfAppliedInfluence(t *testing.T) {
	// Given an applied influence whose holds_if premise is an assumption
	snapshot := profileTestSnapshot(t, profileTestPremiseModel)

	// When support gaps are checked
	findings := SupportGaps(snapshot)

	// Then the support gap finding names the assumption that supports the influence
	profileTestRequireSubjects(t, "support_gap", profileTestSubjects(findings, "support_gap"), "assumption.semantic_contracts")
}

func TestProfileSupportGapsEmptyWithoutInfluence(t *testing.T) {
	// Given a model with one leaf FR and no influences
	snapshot := profileTestSnapshot(t, profileTestNoInfluenceModel)

	// When support gaps are checked
	findings := SupportGaps(snapshot)

	// Then no support gap finding is reported
	if len(findings) != 0 {
		t.Fatalf("findings = %d, want none", len(findings))
	}
}
