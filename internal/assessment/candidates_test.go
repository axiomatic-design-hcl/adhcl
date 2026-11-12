package assessment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// candidatesTestAlpha and candidatesTestBeta are the two candidates of the receipt decision.
const (
	candidatesTestAlpha model.ID = "decision.receipt_mechanism.candidate.integer_cents"
	candidatesTestBeta  model.ID = "decision.receipt_mechanism.candidate.decimal_cents"
)

// candidatesTestRoot declares the design, the exposure identities, the FR with its criterion,
// the two proposed DPs, and the scenario that verifies the criterion.
const candidatesTestRoot = `design "receipt" {
  boundary = "Serve one receipt request."
  workload = workload.load
}

workload "load" {
  description = "One request at a time."
}

workload "other" {
  description = "A bulk batch of requests."
}

context "lab" {
  description = "Controlled test bench."
}

context "field" {
  description = "Live customer traffic."
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
`

// candidatesTestDecisionTemplate has the information block of the first candidate and optional
// extra candidates after it.
const candidatesTestDecisionTemplate = `decision "receipt_mechanism" {
  addresses = [fr.serve]
  context   = context.lab
  workload  = workload.load
  horizon   = "One request under fixed input."
  candidate "integer_cents" {
    dps     = [dp.serve_path]
    version = "receipt-proposal-1"
%s  }
%s}
`

// candidatesTestDecimalCandidate is a second candidate with its own horizon.
const candidatesTestDecimalCandidate = `  candidate "decimal_cents" {
    dps     = [dp.alt_path]
    version = "receipt-proposal-2"
    horizon = "Ten requests under fixed input."
    information {
      basis    = "measured"
      evidence = [evidence.run_b]
    }
  }
`

// candidatesTestUnknownInformation declares a candidate that has no empirical evidence.
const candidatesTestUnknownInformation = `    information {
      basis = "unknown"
    }
`

// candidatesTestEvidenceTemplate declares one joint binomial evidence block.
const candidatesTestEvidenceTemplate = `evidence %q {
  basis             = "measured"
  event             = "A request completes within its bound."
  scenario          = scenario.load_check
  candidate         = %s
  version           = %q
  context           = context.%s
  workload          = workload.%s
  horizon           = %q
  dependence        = "joint"
  sampling          = "Independent requests from one fixed input."
  trials            = %d
  failures          = %d
  confidence_method = "wilson"
  confidence        = %g
}
`

// candidatesTestEvidenceSpec holds the fields of one evidence block.
type candidatesTestEvidenceSpec struct {
	label      string
	candidate  model.ID
	version    string
	context    string
	workload   string
	horizon    string
	trials     int
	failures   int
	confidence float64
}

// candidatesTestMatching returns an evidence spec that exactly matches the alpha candidate.
func candidatesTestMatching(label string, trials, failures int, confidence float64) candidatesTestEvidenceSpec {
	return candidatesTestEvidenceSpec{
		label:      label,
		candidate:  candidatesTestAlpha,
		version:    "receipt-proposal-1",
		context:    "lab",
		workload:   "load",
		horizon:    "One request under fixed input.",
		trials:     trials,
		failures:   failures,
		confidence: confidence,
	}
}

func candidatesTestEvidence(spec candidatesTestEvidenceSpec) string {
	return fmt.Sprintf(candidatesTestEvidenceTemplate, spec.label, spec.candidate, spec.version,
		spec.context, spec.workload, spec.horizon, spec.trials, spec.failures, spec.confidence)
}

// candidatesTestMeasured declares the alpha candidate's information with measured evidence.
func candidatesTestMeasured(refs ...string) string {
	return "    information {\n      basis    = \"measured\"\n      evidence = [" + strings.Join(refs, ", ") + "]\n    }\n"
}

func candidatesTestDecision(information, extra string) string {
	return fmt.Sprintf(candidatesTestDecisionTemplate, information, extra)
}

// candidatesTestPipeline parses, decodes, resolves, and selects the candidate. It fails the
// test when any setup step reports a finding.
func candidatesTestPipeline(t *testing.T, candidate model.ID, sources ...string) *selection.Snapshot {
	t.Helper()
	units := make([]syntax.Unit, 0, len(sources))
	for index, source := range sources {
		unit := syntax.Parse(model.Source{Path: fmt.Sprintf("candidates-%d.ad.hcl", index+1), Bytes: []byte(source)})
		if len(unit.Findings) != 0 {
			t.Fatalf("source %d has %d parse findings, want none", index+1, len(unit.Findings))
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
	snapshot, selectFindings := selection.Select(graph, selection.Request{Candidate: candidate})
	if len(selectFindings) != 0 {
		t.Fatalf("select produced %d findings, want none", len(selectFindings))
	}
	if snapshot == nil {
		t.Fatal("select returned no snapshot")
	}
	return snapshot
}

// candidatesTestRun selects the candidate, then assesses every candidate of the snapshot.
func candidatesTestRun(t *testing.T, candidate model.ID, sources ...string) ([]CandidateResult, []result.Finding) {
	t.Helper()
	snapshot := candidatesTestPipeline(t, candidate, sources...)
	return AssessCandidates(snapshot)
}

// candidatesTestResult returns the result for the candidate, failing the test when it is absent.
func candidatesTestResult(t *testing.T, results []CandidateResult, candidate model.ID) CandidateResult {
	t.Helper()
	for _, r := range results {
		if r.Candidate == candidate {
			return r
		}
	}
	t.Fatalf("no result for candidate %s", candidate)
	return CandidateResult{}
}

// candidatesTestEstimate returns the estimate for the evidence block, failing the test when it is absent.
func candidatesTestEstimate(t *testing.T, result CandidateResult, evidence model.ID) EvidenceEstimate {
	t.Helper()
	for _, e := range result.Evidence {
		if e.Evidence == evidence {
			return e
		}
	}
	t.Fatalf("no estimate for evidence %s", evidence)
	return EvidenceEstimate{}
}

// candidatesTestRequireInformation compares an information bound with an expected bound.
func candidatesTestRequireInformation(t *testing.T, name string, got, want Information) {
	t.Helper()
	if got.Status != want.Status {
		t.Fatalf("%s status is %q, want %q", name, got.Status, want.Status)
	}
	if want.Status == StatusFinite {
		requireNear(t, name, got.Value, want.Value)
	}
}

// candidatesTestRequireFinding fails unless a finding has the given code and subject.
func candidatesTestRequireFinding(t *testing.T, findings []result.Finding, code string, subject model.ID) {
	t.Helper()
	for _, f := range findings {
		if f.Code == code && f.Subject == string(subject) {
			return
		}
	}
	t.Fatalf("got %d findings, want %s on %s", len(findings), code, subject)
}

// candidatesTestFindingKey identifies a finding by its code and subject.
type candidatesTestFindingKey struct {
	code    string
	subject string
}

// candidatesTestRequireExactFindings fails unless the findings are exactly the wanted findings, each reported once. The wanted keys must be unique.
func candidatesTestRequireExactFindings(t *testing.T, findings []result.Finding, want ...candidatesTestFindingKey) {
	t.Helper()
	counts := make(map[candidatesTestFindingKey]int, len(findings))
	for _, f := range findings {
		counts[candidatesTestFindingKey{code: f.Code, subject: f.Subject}]++
	}
	if len(findings) != len(want) {
		t.Fatalf("got %d findings, want %d", len(findings), len(want))
	}
	for _, key := range want {
		if counts[key] != 1 {
			t.Fatalf("got %d findings %s on %s, want 1", counts[key], key.code, key.subject)
		}
	}
}

func TestAssessCandidatesWilsonBoundsForMatchingEvidence(t *testing.T) {
	// Given a candidate with one measured evidence block of 100 trials, 10 failures, and 95% confidence.
	sources := []string{
		candidatesTestRoot,
		candidatesTestDecision(candidatesTestMeasured("evidence.run_a"), ""),
		candidatesTestEvidence(candidatesTestMatching("run_a", 100, 10, 0.95)),
	}

	// When the candidate is assessed.
	results, findings := candidatesTestRun(t, candidatesTestAlpha, sources...)

	// Then the block keeps its Wilson bounds and the observed violation makes the candidate infeasible.
	result := candidatesTestResult(t, results, candidatesTestAlpha)
	if len(result.Evidence) != 1 {
		t.Fatalf("got %d estimates, want 1", len(result.Evidence))
	}
	estimate := candidatesTestEstimate(t, result, "evidence.run_a")
	if estimate.Trials != 100 || estimate.Failures != 10 || estimate.Confidence != 0.95 ||
		estimate.Sampling != "Independent requests from one fixed input." || estimate.Dependence != "joint" ||
		estimate.Horizon != "One request under fixed input." || estimate.Version != "receipt-proposal-1" ||
		estimate.Context != "context.lab" || estimate.Workload != "workload.load" || estimate.Scenario != "scenario.load_check" {
		t.Fatalf("estimate lost exact evidence provenance: %+v", estimate)
	}
	requireNear(t, "low", estimate.Low, 0.8256343384950865)
	requireNear(t, "high", estimate.High, 0.9447708629393249)
	wantBest, wantWorst := InformationBounds(0.8256343384950865, 0.9447708629393249)
	candidatesTestRequireInformation(t, "best", estimate.Best, wantBest)
	candidatesTestRequireInformation(t, "worst", estimate.Worst, wantWorst)
	if result.Feasibility != "infeasible" {
		t.Fatalf("feasibility is %q, want infeasible", result.Feasibility)
	}
	candidatesTestRequireExactFindings(t, findings, candidatesTestFindingKey{code: "infeasible_candidate", subject: string(candidatesTestAlpha)})
}

func TestAssessCandidatesAllSuccessKeepsLowerUncertainty(t *testing.T) {
	// Given a candidate with one measured evidence block of 10 trials, no failures, and 95% confidence.
	sources := []string{
		candidatesTestRoot,
		candidatesTestDecision(candidatesTestMeasured("evidence.run_a"), ""),
		candidatesTestEvidence(candidatesTestMatching("run_a", 10, 0, 0.95)),
	}

	// When the candidate is assessed.
	results, _ := candidatesTestRun(t, candidatesTestAlpha, sources...)

	// Then the upper bound reaches one, but the lower bound keeps its Wilson uncertainty.
	estimate := candidatesTestEstimate(t, candidatesTestResult(t, results, candidatesTestAlpha), "evidence.run_a")
	requireNear(t, "low", estimate.Low, 0.7224672001371109)
	requireNear(t, "high", estimate.High, 1.0)
	if !(estimate.Low < 1) {
		t.Fatalf("low is %.16g, want below 1", estimate.Low)
	}
}

func TestAssessCandidatesMismatchedExposureHasNoEstimate(t *testing.T) {
	tests := []struct {
		name string
		edit func(*candidatesTestEvidenceSpec)
	}{
		{name: "version", edit: func(s *candidatesTestEvidenceSpec) { s.version = "receipt-proposal-0" }},
		{name: "horizon", edit: func(s *candidatesTestEvidenceSpec) { s.horizon = "Two requests under fixed input." }},
		{name: "context", edit: func(s *candidatesTestEvidenceSpec) { s.context = "field" }},
		{name: "workload", edit: func(s *candidatesTestEvidenceSpec) { s.workload = "other" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given a candidate whose only evidence block differs from the candidate in one exposure field.
			spec := candidatesTestMatching("run_a", 100, 10, 0.95)
			tc.edit(&spec)
			sources := []string{
				candidatesTestRoot,
				candidatesTestDecision(candidatesTestMeasured("evidence.run_a"), ""),
				candidatesTestEvidence(spec),
			}

			// When the candidate is assessed.
			results, findings := candidatesTestRun(t, candidatesTestAlpha, sources...)

			// Then no estimate is reported, feasibility is unresolved, and the candidate lacks evidence.
			result := candidatesTestResult(t, results, candidatesTestAlpha)
			if len(result.Evidence) != 0 {
				t.Fatalf("got %d estimates, want none", len(result.Evidence))
			}
			if result.Feasibility != "unresolved" {
				t.Fatalf("feasibility is %q, want unresolved", result.Feasibility)
			}
			candidatesTestRequireFinding(t, findings, "unresolved_feasibility", candidatesTestAlpha)
		})
	}
}

func TestAssessCandidatesConfidenceAboveOneHasNoEstimate(t *testing.T) {
	// Given a candidate whose evidence block declares confidence 1.5, above the valid range.
	sources := []string{
		candidatesTestRoot,
		candidatesTestDecision(candidatesTestMeasured("evidence.run_a"), ""),
		candidatesTestEvidence(candidatesTestMatching("run_a", 10, 1, 1.5)),
	}

	// When the candidate is assessed.
	results, findings := candidatesTestRun(t, candidatesTestAlpha, sources...)

	// Then the invalid evidence is reported against its block and produces no estimate.
	candidatesTestRequireFinding(t, findings, "invalid_evidence", "evidence.run_a")
	if got := len(candidatesTestResult(t, results, candidatesTestAlpha).Evidence); got != 0 {
		t.Fatalf("got %d estimates, want none", got)
	}
}

func TestAssessCandidatesReportsEachEvidenceSeparately(t *testing.T) {
	// Given a candidate with two matching evidence blocks, one of 100 trials and one of 50 trials.
	sources := []string{
		candidatesTestRoot,
		candidatesTestDecision(candidatesTestMeasured("evidence.run_a", "evidence.run_b"), ""),
		candidatesTestEvidence(candidatesTestMatching("run_a", 100, 10, 0.95)),
		candidatesTestEvidence(candidatesTestMatching("run_b", 50, 5, 0.9)),
	}

	// When the candidate is assessed.
	results, findings := candidatesTestRun(t, candidatesTestAlpha, sources...)

	// Then each block keeps its own Wilson bounds without pooling, and observed failures make the candidate infeasible.
	result := candidatesTestResult(t, results, candidatesTestAlpha)
	if len(result.Evidence) != 2 {
		t.Fatalf("got %d estimates, want 2", len(result.Evidence))
	}
	first := candidatesTestEstimate(t, result, "evidence.run_a")
	requireNear(t, "run_a low", first.Low, 0.8256343384950865)
	requireNear(t, "run_a high", first.High, 0.9447708629393249)
	second := candidatesTestEstimate(t, result, "evidence.run_b")
	requireNear(t, "run_b low", second.Low, 0.8084624858484292)
	requireNear(t, "run_b high", second.High, 0.9504709655225548)
	candidatesTestRequireExactFindings(t, findings, candidatesTestFindingKey{code: "infeasible_candidate", subject: string(candidatesTestAlpha)})
}

func TestAssessCandidatesIncomparableHorizonsReported(t *testing.T) {
	// Given two candidates whose effective horizons differ, each with its own matching evidence.
	sources := []string{
		candidatesTestRoot,
		candidatesTestDecision(candidatesTestMeasured("evidence.run_a"), candidatesTestDecimalCandidate),
		candidatesTestEvidence(candidatesTestMatching("run_a", 100, 10, 0.95)),
		candidatesTestEvidence(candidatesTestBetaEvidence()),
	}

	// When the integer candidate is selected and every candidate is assessed.
	results, findings := candidatesTestRun(t, candidatesTestAlpha, sources...)

	// Then both candidates are infeasible, and the horizon difference is the only other finding.
	for _, candidate := range []model.ID{candidatesTestAlpha, candidatesTestBeta} {
		if got := candidatesTestResult(t, results, candidate).Feasibility; got != "infeasible" {
			t.Fatalf("feasibility of %s is %q, want infeasible", candidate, got)
		}
	}
	candidatesTestRequireExactFindings(t, findings,
		candidatesTestFindingKey{code: "infeasible_candidate", subject: string(candidatesTestAlpha)},
		candidatesTestFindingKey{code: "infeasible_candidate", subject: string(candidatesTestBeta)},
		candidatesTestFindingKey{code: "incomparable_evidence", subject: "decision.receipt_mechanism"},
	)
}

func TestAssessCandidatesMissingEvidenceIsUnresolved(t *testing.T) {
	// Given a candidate whose information is unknown and that has no evidence block.
	sources := []string{
		candidatesTestRoot,
		candidatesTestDecision(candidatesTestUnknownInformation, ""),
	}

	// When the candidate is assessed.
	results, findings := candidatesTestRun(t, candidatesTestAlpha, sources...)

	// Then feasibility is unresolved, no estimate is reported, and the missing evidence is named.
	result := candidatesTestResult(t, results, candidatesTestAlpha)
	if result.Feasibility != "unresolved" {
		t.Fatalf("feasibility is %q, want unresolved", result.Feasibility)
	}
	if len(result.Evidence) != 0 {
		t.Fatalf("got %d estimates, want none", len(result.Evidence))
	}
	candidatesTestRequireFinding(t, findings, "unresolved_feasibility", candidatesTestAlpha)
}

// candidatesTestBetaEvidence is the evidence block of the decimal candidate, with the horizon it declares.
func candidatesTestBetaEvidence() candidatesTestEvidenceSpec {
	return candidatesTestEvidenceSpec{
		label:      "run_b",
		candidate:  candidatesTestBeta,
		version:    "receipt-proposal-2",
		context:    "lab",
		workload:   "load",
		horizon:    "Ten requests under fixed input.",
		trials:     50,
		failures:   5,
		confidence: 0.9,
	}
}

func TestAssessCandidatesInformationRequestsOnlyReferencedMeasuredEvidence(t *testing.T) {
	for _, information := range []string{"", candidatesTestUnknownInformation,
		"    information {\n      basis = \"unknown\"\n      evidence = [evidence.run_a]\n    }\n",
		candidatesTestMeasured("evidence.run_b")} {
		t.Run(information, func(t *testing.T) {
			results, _ := candidatesTestRun(t, candidatesTestAlpha, candidatesTestRoot,
				candidatesTestDecision(information, ""),
				candidatesTestEvidence(candidatesTestMatching("run_a", 20, 0, 0.95)),
				candidatesTestEvidence(candidatesTestMatching("run_b", 10, 0, 0.95)))
			got := candidatesTestResult(t, results, candidatesTestAlpha)
			want := 0
			if information == candidatesTestMeasured("evidence.run_b") {
				want = 1
				candidatesTestEstimate(t, got, "evidence.run_b")
			}
			if len(got.Evidence) != want || got.Feasibility != "unresolved" {
				t.Fatalf("got %+v, want %d requested estimates and unresolved feasibility", got, want)
			}
		})
	}
}

func TestAssessCandidatesRequiresApplicableCompleteOperationalCoverage(t *testing.T) {
	tests := []struct {
		name string
		edit func(*model.Core)
	}{
		{"missing criterion", func(core *model.Core) {
			criterion := core.FRs[0].Criteria[0]
			criterion.ID = "fr.serve.criterion.accuracy"
			core.FRs[0].Criteria = append(core.FRs[0].Criteria, criterion)
		}},
		{"nonoperational criterion", func(core *model.Core) { core.FRs[0].Criteria[0].Metric = nil }},
		{"no criteria", func(core *model.Core) { core.FRs[0].Criteria = nil }},
		{"inapplicable scenario", func(core *model.Core) {
			core.Scenarios[0].Under = &model.Ref{Target: "context.field"}
		}},
		{"unresolved scenario", func(core *model.Core) {
			core.Scenarios[0].Under = &model.Ref{Target: "workload.missing"}
		}},
		{"missing scenario", func(core *model.Core) { core.Scenarios = nil }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := candidatesTestPipeline(t, candidatesTestAlpha, candidatesTestRoot,
				candidatesTestDecision(candidatesTestMeasured("evidence.run_a"), ""),
				candidatesTestEvidence(candidatesTestMatching("run_a", 20, 20, 0.95)))
			tc.edit(&snapshot.Graph.Core)
			results, findings := AssessCandidates(snapshot)
			got := candidatesTestResult(t, results, candidatesTestAlpha)
			if len(got.Evidence) != 0 || got.Feasibility != "unresolved" {
				t.Fatalf("unqualified evidence produced %+v", got)
			}
			candidatesTestRequireFinding(t, findings, "unresolved_feasibility", candidatesTestAlpha)
		})
	}
}

func TestAssessCandidatesObservedRequiredFailureIdentifiesCriterion(t *testing.T) {
	for _, information := range []string{candidatesTestMeasured("evidence.run_a"), candidatesTestUnknownInformation} {
		results, findings := candidatesTestRun(t, candidatesTestAlpha, candidatesTestRoot,
			candidatesTestDecision(information, ""),
			candidatesTestEvidence(candidatesTestMatching("run_a", 20, 20, 0.95)))
		got := candidatesTestResult(t, results, candidatesTestAlpha)
		if got.Feasibility != "infeasible" {
			t.Fatalf("feasibility is %q, want infeasible", got.Feasibility)
		}
		candidatesTestRequireFinding(t, findings, "infeasible_candidate", candidatesTestAlpha)
		for _, finding := range findings {
			if finding.Code == "infeasible_candidate" && !strings.Contains(finding.Message, "fr.serve.criterion.latency") {
				t.Fatalf("failure omits required criterion: %s", finding.Message)
			}
			if finding.Code == "unresolved_feasibility" {
				t.Fatalf("infeasible candidate also reported unresolved: %+v", finding)
			}
		}
	}
}

func TestAssessCandidatesJointCoverageDoesNotCombineEvidenceBlocks(t *testing.T) {
	snapshot := candidatesTestPipeline(t, candidatesTestAlpha, candidatesTestRoot,
		candidatesTestDecision(candidatesTestMeasured("evidence.run_a", "evidence.run_b"), ""),
		candidatesTestEvidence(candidatesTestMatching("run_a", 20, 0, 0.95)),
		candidatesTestEvidence(candidatesTestMatching("run_b", 30, 0, 0.95)))
	core := &snapshot.Graph.Core
	criterion := core.FRs[0].Criteria[0]
	criterion.ID = "fr.serve.criterion.accuracy"
	core.FRs[0].Criteria = append(core.FRs[0].Criteria, criterion)
	scenario := core.Scenarios[0]
	scenario.ID = "scenario.accuracy_check"
	scenario.Verifies.Items = []model.Ref{{Target: criterion.ID}}
	core.Scenarios = append(core.Scenarios, scenario)
	core.Evidence[1].Scenario.Target = scenario.ID
	results, _ := AssessCandidates(snapshot)
	if got := candidatesTestResult(t, results, candidatesTestAlpha); len(got.Evidence) != 0 {
		t.Fatalf("separate partial scenarios produced a joint estimate: %+v", got)
	}
	core.Scenarios[0].Verifies.Items = append(core.Scenarios[0].Verifies.Items, model.Ref{Target: criterion.ID})
	results, _ = AssessCandidates(snapshot)
	got := candidatesTestResult(t, results, candidatesTestAlpha)
	if len(got.Evidence) != 1 || got.Evidence[0].Evidence != "evidence.run_a" || got.Feasibility != "unresolved" {
		t.Fatalf("complete all-success joint block produced %+v", got)
	}
}

func TestAssessCandidatesUsesEachCandidatesOwnPredicateConditions(t *testing.T) {
	snapshot := candidatesTestPipeline(t, candidatesTestAlpha, candidatesTestRoot,
		candidatesTestDecision(candidatesTestMeasured("evidence.run_a"), candidatesTestDecimalCandidate),
		candidatesTestEvidence(candidatesTestMatching("run_a", 20, 0, 0.95)),
		candidatesTestEvidence(candidatesTestBetaEvidence()))
	core := &snapshot.Graph.Core
	core.Decisions[0].Candidates[1].Context = &model.Ref{Target: "context.field"}
	core.Evidence[1].Context.Target = "context.field"
	core.FRs[0].Criteria[0].Under = &model.Ref{Target: "context.field"}
	core.Scenarios[0].Under = &model.Ref{Target: "context.field"}
	results, _ := AssessCandidates(snapshot)
	alpha := candidatesTestResult(t, results, candidatesTestAlpha)
	beta := candidatesTestResult(t, results, candidatesTestBeta)
	if len(alpha.Evidence) != 0 || alpha.Feasibility != "unresolved" {
		t.Fatalf("inapplicable selected-candidate predicates produced %+v", alpha)
	}
	if len(beta.Evidence) != 1 || beta.Feasibility != "infeasible" {
		t.Fatalf("alternative candidate did not use its own conditions: %+v", beta)
	}
}

func TestAssessCandidatesRejectsCrossCandidateRequestedEvidence(t *testing.T) {
	snapshot := candidatesTestPipeline(t, candidatesTestAlpha, candidatesTestRoot,
		candidatesTestDecision(candidatesTestMeasured("evidence.run_a"), ""),
		candidatesTestEvidence(candidatesTestMatching("run_a", 20, 0, 0.95)))
	snapshot.Graph.Core.Evidence[0].Candidate.Target = candidatesTestBeta
	results, findings := AssessCandidates(snapshot)
	if got := candidatesTestResult(t, results, candidatesTestAlpha); len(got.Evidence) != 0 {
		t.Fatalf("cross-candidate evidence produced %+v", got)
	}
	candidatesTestRequireFinding(t, findings, "invalid_evidence", "evidence.run_a")
}
