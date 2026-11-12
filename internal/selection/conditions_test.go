package selection

import (
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// conditionFRs declares a need and the two root FRs that the condition fixtures use.
const conditionFRs = `need "serve" {
  statement = "Customers need one order accepted."
}

fr "root" {
  statement = "Accept one order."
  need = need.serve
}

fr "other" {
  statement = "Record one order."
  need = need.serve
}

`

// conditionWorkloads declares two input populations.
const conditionWorkloads = `workload "w1" {
  description = "First input population."
}

workload "w2" {
  description = "Second input population."
}

`

// conditionRouteProposed declares a proposed DP that controls the root FR.
const conditionRouteProposed = `dp "route" {
  controls = [fr.root]
  responsibility = "Choose the route for one order."
  decision = "proposed"
}

`

// conditionRouteSelected declares a selected DP that controls the root FR.
const conditionRouteSelected = `dp "route" {
  controls = [fr.root]
  responsibility = "Choose the route for one order."
  decision = "selected"
}

`

// conditionExcludingContexts declares alpha, which excludes beta, and beta.
const conditionExcludingContexts = `context "alpha" {
  description = "Alpha operating condition."
  excludes = [context.beta]
}

context "beta" {
  description = "Beta operating condition."
}

`

// conditionDesign declares the design root. An empty workload leaves the design without one.
func conditionDesign(workload string) string {
	line := ""
	if workload != "" {
		line = "  workload = " + workload + "\n"
	}
	return "design \"checkout\" {\n  boundary = \"Checkout order handling.\"\n" + line + "}\n\n"
}

// conditionContext declares one named context. An empty workload leaves the context without one.
func conditionContext(label, workload string) string {
	line := ""
	if workload != "" {
		line = "  workload = " + workload + "\n"
	}
	return "context \"" + label + "\" {\n  description = \"" + label + " operating condition.\"\n" + line + "}\n\n"
}

// conditionSelect runs Select on a graph the test already built and requires a snapshot.
func conditionSelect(t *testing.T, graph *model.Graph, request Request) *Snapshot {
	t.Helper()
	snapshot, findings := Select(graph, request)
	return conditionRequireSnapshot(t, snapshot, findings)
}

// conditionRequireSnapshot fails the test when selection reports a finding or returns no snapshot.
func conditionRequireSnapshot(t *testing.T, snapshot *Snapshot, findings []result.Finding) *Snapshot {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("finding codes = %v, want none", conditionCodes(findings))
	}
	if snapshot == nil {
		t.Fatal("snapshot is nil, want a snapshot")
	}
	return snapshot
}

// conditionRequireInvalid fails the test unless selection returns no snapshot and reports only
// invalid_selection findings.
func conditionRequireInvalid(t *testing.T, snapshot *Snapshot, findings []result.Finding) {
	t.Helper()
	if snapshot != nil {
		t.Fatal("snapshot is non-nil, want nil for an invalid selection")
	}
	if len(findings) == 0 {
		t.Fatal("findings are empty, want invalid_selection")
	}
	for _, finding := range findings {
		if finding.Code != "invalid_selection" {
			t.Fatalf("finding code = %q, want invalid_selection", finding.Code)
		}
	}
}

// conditionCodes lists the finding codes in order.
func conditionCodes(findings []result.Finding) []string {
	codes := make([]string, 0, len(findings))
	for _, finding := range findings {
		codes = append(codes, finding.Code)
	}
	return codes
}

// conditionRef returns a reference to the target identity.
func conditionRef(target string) *model.Ref {
	return &model.Ref{Target: model.ID(target)}
}

// conditionInfluenceFrom returns the one applied influence whose source DP is from.
func conditionInfluenceFrom(t *testing.T, snapshot *Snapshot, from model.ID) AppliedInfluence {
	t.Helper()
	var matches []AppliedInfluence
	for _, applied := range snapshot.Influences {
		if applied.From == from {
			matches = append(matches, applied)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("applied influences from %s = %d, want 1", from, len(matches))
	}
	return matches[0]
}

func TestSelectRequestContextFillsEmptyCandidateContext(t *testing.T) {
	// Given a candidate that names no context and a declared context alpha.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected with a request for context alpha.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one", Context: "context.alpha"})

	// Then alpha becomes the effective context.
	if snapshot.Context != "context.alpha" {
		t.Fatalf("context = %q, want context.alpha", snapshot.Context)
	}
}

func TestSelectCandidateContextKeptWhenRequestOmitted(t *testing.T) {
	// Given a candidate whose context is alpha.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
    context = context.alpha
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected without a context request.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the candidate context alpha is the effective context.
	if snapshot.Context != "context.alpha" {
		t.Fatalf("context = %q, want context.alpha", snapshot.Context)
	}
}

func TestSelectRequestContextMatchingCandidateContextIsAccepted(t *testing.T) {
	// Given a candidate whose context is alpha.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
    context = context.alpha
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected with a request for the same context alpha.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one", Context: "context.alpha"})

	// Then the effective context is alpha.
	if snapshot.Context != "context.alpha" {
		t.Fatalf("context = %q, want context.alpha", snapshot.Context)
	}
}

func TestSelectRejectsRequestContextDifferentFromCandidateContext(t *testing.T) {
	// Given a candidate whose context is alpha, and a second declared context beta.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionContext("beta", "") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
    context = context.alpha
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected with a request for context beta.
	snapshot, findings := Select(graph, Request{Candidate: "decision.options.candidate.one", Context: "context.beta"})

	// Then the selection is invalid.
	conditionRequireInvalid(t, snapshot, findings)
}

func TestSelectUsesDecisionContextWhenCandidateOmitsContext(t *testing.T) {
	// Given a decision whose context is alpha and a candidate that names no context.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]
  context = context.alpha

  candidate "one" {
    dps = [dp.route]
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected without a context request.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the decision context alpha is the effective context.
	if snapshot.Context != "context.alpha" {
		t.Fatalf("context = %q, want context.alpha", snapshot.Context)
	}
}

func TestSelectCandidateContextOverridesDecisionContext(t *testing.T) {
	// Given a decision whose context is alpha and a candidate whose context is beta.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionContext("beta", "") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]
  context = context.alpha

  candidate "one" {
    dps = [dp.route]
    context = context.beta
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected without a context request.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the candidate context beta is the effective context.
	if snapshot.Context != "context.beta" {
		t.Fatalf("context = %q, want context.beta", snapshot.Context)
	}
}

func TestSelectRejectsUnknownRequestContext(t *testing.T) {
	// Given a design with a selected DP and no declared context.
	source := conditionDesign("") + conditionFRs + conditionRouteSelected
	graph := buildGraph(t, source)

	// When selection requests a context that is not declared.
	snapshot, findings := Select(graph, Request{Context: "context.missing"})

	// Then the selection is invalid.
	conditionRequireInvalid(t, snapshot, findings)
}

func TestSelectRejectsRequestContextExcludedByCandidateContext(t *testing.T) {
	// Given alpha, which excludes beta, as the candidate context.
	source := conditionDesign("") + conditionFRs + conditionExcludingContexts + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
    context = context.alpha
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected with a request for beta.
	snapshot, findings := Select(graph, Request{Candidate: "decision.options.candidate.one", Context: "context.beta"})

	// Then the selection is invalid.
	conditionRequireInvalid(t, snapshot, findings)
}

func TestSelectRejectsCandidateContextExcludedByRequestContext(t *testing.T) {
	// Given beta as the candidate context and alpha, which excludes beta, as the request.
	source := conditionDesign("") + conditionFRs + conditionExcludingContexts + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
    context = context.beta
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected with a request for alpha.
	snapshot, findings := Select(graph, Request{Candidate: "decision.options.candidate.one", Context: "context.alpha"})

	// Then the selection is invalid, because exclusion is symmetric.
	conditionRequireInvalid(t, snapshot, findings)
}

func TestSelectHorizonPrefersCandidateOverDecision(t *testing.T) {
	// Given a decision horizon of 30 days and a candidate horizon of 90 days.
	source := conditionDesign("") + conditionFRs + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]
  horizon = "30 days"

  candidate "one" {
    dps = [dp.route]
    horizon = "90 days"
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the candidate horizon is the effective horizon.
	if snapshot.Horizon != "90 days" {
		t.Fatalf("horizon = %q, want 90 days", snapshot.Horizon)
	}
}

func TestSelectHorizonFallsBackToDecision(t *testing.T) {
	// Given a decision horizon of 30 days and a candidate that names no horizon.
	source := conditionDesign("") + conditionFRs + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]
  horizon = "30 days"

  candidate "one" {
    dps = [dp.route]
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the decision horizon is the effective horizon.
	if snapshot.Horizon != "30 days" {
		t.Fatalf("horizon = %q, want 30 days", snapshot.Horizon)
	}
}

func TestSelectHorizonIsEmptyWhenNoSourceAuthorsIt(t *testing.T) {
	// Given a candidate with no horizon and a decision with no horizon.
	source := conditionDesign("") + conditionFRs + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the effective horizon stays unspecified.
	if snapshot.Horizon != "" {
		t.Fatalf("horizon = %q, want empty", snapshot.Horizon)
	}
}

func TestSelectWorkloadPrefersCandidateOverDecision(t *testing.T) {
	// Given a decision workload of w2 and a candidate workload of w1, with no context.
	source := conditionDesign("") + conditionFRs + conditionWorkloads + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]
  workload = workload.w2

  candidate "one" {
    dps = [dp.route]
    workload = workload.w1
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the candidate workload w1 is the effective workload.
	if snapshot.Workload != "workload.w1" {
		t.Fatalf("workload = %q, want workload.w1", snapshot.Workload)
	}
}

func TestSelectWorkloadFallsBackToDecision(t *testing.T) {
	// Given a decision workload of w2 and a candidate that names no workload, with no context.
	source := conditionDesign("") + conditionFRs + conditionWorkloads + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]
  workload = workload.w2

  candidate "one" {
    dps = [dp.route]
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the decision workload w2 is the effective workload.
	if snapshot.Workload != "workload.w2" {
		t.Fatalf("workload = %q, want workload.w2", snapshot.Workload)
	}
}

func TestSelectWorkloadFromContextBeforeDesignWorkload(t *testing.T) {
	// Given a design workload of w2 and a candidate context alpha whose workload is w1.
	source := conditionDesign("workload.w2") + conditionFRs + conditionWorkloads + conditionContext("alpha", "workload.w1") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
    context = context.alpha
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the context workload w1 is the effective workload, because the design root is the last fallback.
	if snapshot.Workload != "workload.w1" {
		t.Fatalf("workload = %q, want workload.w1", snapshot.Workload)
	}
}

func TestSelectWorkloadFallsBackToDesignWorkload(t *testing.T) {
	// Given a design workload of w2 and a candidate with no workload and no context.
	source := conditionDesign("workload.w2") + conditionFRs + conditionWorkloads + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot := conditionSelect(t, graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the design workload w2 is the effective workload.
	if snapshot.Workload != "workload.w2" {
		t.Fatalf("workload = %q, want workload.w2", snapshot.Workload)
	}
}

func TestSelectRejectsCandidateWorkloadDisagreeingWithContextWorkload(t *testing.T) {
	// Given a candidate workload of w1 and a candidate context alpha whose workload is w2.
	source := conditionDesign("") + conditionFRs + conditionWorkloads + conditionContext("alpha", "workload.w2") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]

  candidate "one" {
    dps = [dp.route]
    context = context.alpha
    workload = workload.w1
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot, findings := Select(graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the selection is invalid.
	conditionRequireInvalid(t, snapshot, findings)
}

func TestSelectRejectsDecisionWorkloadDisagreeingWithContextWorkload(t *testing.T) {
	// Given a decision workload of w1 and a candidate context alpha whose workload is w2.
	source := conditionDesign("") + conditionFRs + conditionWorkloads + conditionContext("alpha", "workload.w2") + conditionRouteProposed + `decision "options" {
  addresses = [fr.root]
  workload = workload.w1

  candidate "one" {
    dps = [dp.route]
    context = context.alpha
  }
}
`
	graph := buildGraph(t, source)

	// When the candidate is selected.
	snapshot, findings := Select(graph, Request{Candidate: "decision.options.candidate.one"})

	// Then the selection is invalid.
	conditionRequireInvalid(t, snapshot, findings)
}

func TestSelectWithoutCandidateRejectsContextWorkloadDisagreeingWithDesign(t *testing.T) {
	// Given a design workload of w1 and a context alpha whose workload is w2.
	source := conditionDesign("workload.w1") + conditionFRs + conditionWorkloads + conditionContext("alpha", "workload.w2") + conditionRouteSelected
	graph := buildGraph(t, source)

	// When selection requests context alpha without a candidate.
	snapshot, findings := Select(graph, Request{Context: "context.alpha"})

	// Then the selection is invalid.
	conditionRequireInvalid(t, snapshot, findings)
}

func TestSelectWithoutCandidateAcceptsContextWorkloadAgreeingWithDesign(t *testing.T) {
	// Given a design workload of w1 and a context alpha whose workload is also w1.
	source := conditionDesign("workload.w1") + conditionFRs + conditionWorkloads + conditionContext("alpha", "workload.w1") + conditionRouteSelected
	graph := buildGraph(t, source)

	// When selection requests context alpha without a candidate.
	snapshot := conditionSelect(t, graph, Request{Context: "context.alpha"})

	// Then alpha and workload w1 are the effective values.
	if snapshot.Context != "context.alpha" {
		t.Fatalf("context = %q, want context.alpha", snapshot.Context)
	}
	if snapshot.Workload != "workload.w1" {
		t.Fatalf("workload = %q, want workload.w1", snapshot.Workload)
	}
}

func TestSelectWithoutCandidateTakesWorkloadFromRequestedContext(t *testing.T) {
	// Given a design with no workload and a context alpha whose workload is w2.
	source := conditionDesign("") + conditionFRs + conditionWorkloads + conditionContext("alpha", "workload.w2") + conditionRouteSelected
	graph := buildGraph(t, source)

	// When selection requests context alpha without a candidate.
	snapshot := conditionSelect(t, graph, Request{Context: "context.alpha"})

	// Then the context workload w2 is the effective workload.
	if snapshot.Workload != "workload.w2" {
		t.Fatalf("workload = %q, want workload.w2", snapshot.Workload)
	}
}

func TestSelectWithoutCandidateUsesDesignWorkloadWithoutRequest(t *testing.T) {
	// Given a design workload of w1 and a declared context alpha with no workload, and no context request.
	source := conditionDesign("workload.w1") + conditionFRs + conditionWorkloads + conditionContext("alpha", "") + conditionRouteSelected
	graph := buildGraph(t, source)

	// When selection runs without a candidate or context request.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the design workload w1 is the effective workload and no context is selected.
	if snapshot.Workload != "workload.w1" {
		t.Fatalf("workload = %q, want workload.w1", snapshot.Workload)
	}
	if snapshot.Context != "" {
		t.Fatalf("context = %q, want empty", snapshot.Context)
	}
}

func TestSelectDesignWorkloadDoesNotSelectContextWithMatchingLabel(t *testing.T) {
	// Given a design workload of w1 and a declared context whose label is also w1.
	source := conditionDesign("workload.w1") + conditionFRs + conditionWorkloads + conditionContext("w1", "") + conditionRouteSelected
	graph := buildGraph(t, source)

	// When selection runs without a candidate or context request.
	snapshot := conditionSelect(t, graph, Request{})

	// Then no context is selected merely because its label matches the design workload.
	if snapshot.Context != "" {
		t.Fatalf("context = %q, want empty", snapshot.Context)
	}
}

func TestSelectWithoutCandidateLeavesUnauthoredConditionsEmpty(t *testing.T) {
	// Given a design with no workload, no context, and no horizon.
	source := conditionDesign("") + conditionFRs + conditionRouteSelected
	graph := buildGraph(t, source)

	// When selection runs without a candidate or context request.
	snapshot := conditionSelect(t, graph, Request{})

	// Then context, workload, and horizon all stay unspecified.
	if snapshot.Context != "" || snapshot.Workload != "" || snapshot.Horizon != "" {
		t.Fatalf("context, workload, horizon = %q, %q, %q, want all empty",
			snapshot.Context, snapshot.Workload, snapshot.Horizon)
	}
}

func TestAppliesNilReferenceAppliesWithoutSelectedContext(t *testing.T) {
	// Given a snapshot with no selected context.
	graph := buildGraph(t, conditionDesign("")+conditionFRs+conditionRouteSelected)
	snapshot := conditionSelect(t, graph, Request{})

	// When applicability is asked for an absent reference.
	got := snapshot.Applies(nil)

	// Then the condition applies.
	if got != Applicability("applies") {
		t.Fatalf("Applies(nil) = %q, want applies", got)
	}
}

func TestAppliesMatchingContextApplies(t *testing.T) {
	// Given a snapshot whose selected context is alpha.
	graph := buildGraph(t, conditionDesign("")+conditionFRs+conditionContext("alpha", "")+conditionContext("beta", "")+conditionRouteSelected)
	snapshot := conditionSelect(t, graph, Request{Context: "context.alpha"})

	// When applicability is asked for context alpha.
	got := snapshot.Applies(conditionRef("context.alpha"))

	// Then the condition applies.
	if got != Applicability("applies") {
		t.Fatalf("Applies(context.alpha) = %q, want applies", got)
	}
}

func TestAppliesDifferentSelectedContextIsNotApplicable(t *testing.T) {
	// Given a snapshot whose selected context is alpha.
	graph := buildGraph(t, conditionDesign("")+conditionFRs+conditionContext("alpha", "")+conditionContext("beta", "")+conditionRouteSelected)
	snapshot := conditionSelect(t, graph, Request{Context: "context.alpha"})

	// When applicability is asked for context beta.
	got := snapshot.Applies(conditionRef("context.beta"))

	// Then the condition is not applicable.
	if got != Applicability("not_applicable") {
		t.Fatalf("Applies(context.beta) = %q, want not_applicable", got)
	}
}

func TestAppliesContextWithoutSelectedContextIsUnresolved(t *testing.T) {
	// Given a snapshot with no selected context and a declared context alpha.
	graph := buildGraph(t, conditionDesign("")+conditionFRs+conditionContext("alpha", "")+conditionRouteSelected)
	snapshot := conditionSelect(t, graph, Request{})

	// When applicability is asked for context alpha.
	got := snapshot.Applies(conditionRef("context.alpha"))

	// Then the condition is unresolved.
	if got != Applicability("unresolved") {
		t.Fatalf("Applies(context.alpha) = %q, want unresolved", got)
	}
}

func TestAppliesMatchingWorkloadApplies(t *testing.T) {
	// Given a snapshot whose effective workload is w1 from the design.
	graph := buildGraph(t, conditionDesign("workload.w1")+conditionFRs+conditionWorkloads+conditionRouteSelected)
	snapshot := conditionSelect(t, graph, Request{})

	// When applicability is asked for workload w1.
	got := snapshot.Applies(conditionRef("workload.w1"))

	// Then the condition applies.
	if got != Applicability("applies") {
		t.Fatalf("Applies(workload.w1) = %q, want applies", got)
	}
}

func TestAppliesDifferentKnownWorkloadIsNotApplicable(t *testing.T) {
	// Given a snapshot whose effective workload is w1 from the design.
	graph := buildGraph(t, conditionDesign("workload.w1")+conditionFRs+conditionWorkloads+conditionRouteSelected)
	snapshot := conditionSelect(t, graph, Request{})

	// When applicability is asked for workload w2.
	got := snapshot.Applies(conditionRef("workload.w2"))

	// Then the condition is not applicable.
	if got != Applicability("not_applicable") {
		t.Fatalf("Applies(workload.w2) = %q, want not_applicable", got)
	}
}

func TestAppliesWorkloadWithoutKnownWorkloadIsUnresolved(t *testing.T) {
	// Given a snapshot with no effective workload.
	graph := buildGraph(t, conditionDesign("")+conditionFRs+conditionWorkloads+conditionRouteSelected)
	snapshot := conditionSelect(t, graph, Request{})

	// When applicability is asked for workload w2.
	got := snapshot.Applies(conditionRef("workload.w2"))

	// Then the condition is unresolved.
	if got != Applicability("unresolved") {
		t.Fatalf("Applies(workload.w2) = %q, want unresolved", got)
	}
}

func TestInfluenceWithoutUnderAppliesWithoutSelectedContext(t *testing.T) {
	// Given an active influence with no under reference and no selected context.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionRouteSelected + `influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
}
`
	graph := buildGraph(t, source)

	// When selection runs without a context request.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the influence applies.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Applicability != Applicability("applies") {
		t.Fatalf("applicability = %q, want applies", applied.Applicability)
	}
}

func TestInfluenceUnderContextIsUnresolvedWithoutSelectedContext(t *testing.T) {
	// Given an active influence whose under reference is context alpha.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionRouteSelected + `influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
  under = context.alpha
}
`
	graph := buildGraph(t, source)

	// When selection runs without a context request.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the influence is unresolved because no context is selected.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Applicability != Applicability("unresolved") {
		t.Fatalf("applicability = %q, want unresolved", applied.Applicability)
	}
}

func TestInfluenceUnderContextAppliesInSelectedContext(t *testing.T) {
	// Given an active influence whose under reference is context alpha.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionRouteSelected + `influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
  under = context.alpha
}
`
	graph := buildGraph(t, source)

	// When selection requests context alpha.
	snapshot := conditionSelect(t, graph, Request{Context: "context.alpha"})

	// Then the influence applies.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Applicability != Applicability("applies") {
		t.Fatalf("applicability = %q, want applies", applied.Applicability)
	}
}

func TestInfluenceUnderOtherContextIsNotApplicable(t *testing.T) {
	// Given an active influence whose under reference is context alpha.
	source := conditionDesign("") + conditionFRs + conditionContext("alpha", "") + conditionContext("beta", "") + conditionRouteSelected + `influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
  under = context.alpha
}
`
	graph := buildGraph(t, source)

	// When selection requests context beta.
	snapshot := conditionSelect(t, graph, Request{Context: "context.beta"})

	// Then the influence is not applicable.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Applicability != Applicability("not_applicable") {
		t.Fatalf("applicability = %q, want not_applicable", applied.Applicability)
	}
}

func TestInfluenceUnderWorkloadIsNotApplicableForOtherWorkload(t *testing.T) {
	// Given a design workload of w1 and an active influence whose under reference is workload w2.
	source := conditionDesign("workload.w1") + conditionFRs + conditionWorkloads + conditionRouteSelected + `influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
  under = workload.w2
}
`
	graph := buildGraph(t, source)

	// When selection runs without a context request.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the influence is not applicable to the known workload w1.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Applicability != Applicability("not_applicable") {
		t.Fatalf("applicability = %q, want not_applicable", applied.Applicability)
	}
}

func TestPremiseIsNoneWithoutAnyHoldsIf(t *testing.T) {
	// Given an active influence and an active source DP, neither with holds_if.
	source := conditionDesign("") + conditionFRs + conditionRouteSelected + `influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
}
`
	graph := buildGraph(t, source)

	// When selection runs.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the premise state is none.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Premises != PremiseState("none") {
		t.Fatalf("premises = %q, want none", applied.Premises)
	}
}

func TestPremiseIsInvalidatedWhenInfluencePremiseIsInvalidated(t *testing.T) {
	// Given an influence whose holds_if names an invalidated assumption.
	source := conditionDesign("") + conditionFRs + conditionRouteSelected + `assumption "stable" {
  statement = "The order rate stays in range."
  status = "invalidated"
}

influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
  holds_if = [assumption.stable]
}
`
	graph := buildGraph(t, source)

	// When selection runs.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the premise state is invalidated.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Premises != PremiseState("invalidated") {
		t.Fatalf("premises = %q, want invalidated", applied.Premises)
	}
}

func TestPremiseIsInvalidatedWhenSourceDPPremiseIsInvalidated(t *testing.T) {
	// Given a source DP whose holds_if names an invalidated assumption, and an influence with no holds_if.
	source := conditionDesign("") + conditionFRs + `assumption "stable" {
  statement = "The order rate stays in range."
  status = "invalidated"
}

dp "route" {
  controls = [fr.root]
  responsibility = "Choose the route for one order."
  decision = "selected"
  holds_if = [assumption.stable]
}

influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
}
`
	graph := buildGraph(t, source)

	// When selection runs.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the premise state is invalidated because the source DP premise counts.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Premises != PremiseState("invalidated") {
		t.Fatalf("premises = %q, want invalidated", applied.Premises)
	}
}

func TestPremiseIsConditionalWhenInfluencePremiseIsAssumed(t *testing.T) {
	// Given an influence whose holds_if names an assumed assumption.
	source := conditionDesign("") + conditionFRs + conditionRouteSelected + `assumption "rate_known" {
  statement = "The order rate is known."
  status = "assumed"
}

influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
  holds_if = [assumption.rate_known]
}
`
	graph := buildGraph(t, source)

	// When selection runs.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the premise state is conditional.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Premises != PremiseState("conditional") {
		t.Fatalf("premises = %q, want conditional", applied.Premises)
	}
}

func TestPremiseIsConditionalWhenSourceDPPremiseIsToValidate(t *testing.T) {
	// Given a source DP whose holds_if names an assumption with the default status to_validate.
	source := conditionDesign("") + conditionFRs + `assumption "rate_known" {
  statement = "The order rate is known."
}

dp "route" {
  controls = [fr.root]
  responsibility = "Choose the route for one order."
  decision = "selected"
  holds_if = [assumption.rate_known]
}

influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
}
`
	graph := buildGraph(t, source)

	// When selection runs.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the premise state is conditional.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Premises != PremiseState("conditional") {
		t.Fatalf("premises = %q, want conditional", applied.Premises)
	}
}

func TestPremiseStaysConditionalWhenPremiseIsValidated(t *testing.T) {
	// Given an influence whose holds_if names a validated assumption.
	source := conditionDesign("") + conditionFRs + conditionRouteSelected + `assumption "rate_known" {
  statement = "The order rate is known."
  status = "validated"
}

influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
  holds_if = [assumption.rate_known]
}
`
	graph := buildGraph(t, source)

	// When selection runs.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the premise state is conditional, because an authored validated status is not proof.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Premises != PremiseState("conditional") {
		t.Fatalf("premises = %q, want conditional", applied.Premises)
	}
}

func TestPremiseIsInvalidatedWhenAnyPremiseIsInvalidated(t *testing.T) {
	// Given an influence whose holds_if names one assumed assumption and one invalidated assumption.
	source := conditionDesign("") + conditionFRs + conditionRouteSelected + `assumption "rate_known" {
  statement = "The order rate is known."
  status = "assumed"
}

assumption "rate_stable" {
  statement = "The order rate stays in range."
  status = "invalidated"
}

influence {
  from = [dp.route]
  to = fr.root
  effect = "material"
  mechanism = "Route choice changes the accepted order."
  holds_if = [assumption.rate_known, assumption.rate_stable]
}
`
	graph := buildGraph(t, source)

	// When selection runs.
	snapshot := conditionSelect(t, graph, Request{})

	// Then the premise state is invalidated.
	applied := conditionInfluenceFrom(t, snapshot, "dp.route")
	if applied.Premises != PremiseState("invalidated") {
		t.Fatalf("premises = %q, want invalidated", applied.Premises)
	}
}

func TestPremiseIsKeptPerSourceDP(t *testing.T) {
	// Given a multi-source influence where dp.route has an invalidated premise and dp.log has none.
	source := conditionDesign("") + conditionFRs + `assumption "stable" {
  statement = "The order rate stays in range."
  status = "invalidated"
}

dp "route" {
  controls = [fr.root]
  responsibility = "Choose the route for one order."
  decision = "selected"
  holds_if = [assumption.stable]
}

dp "log" {
  controls = [fr.other]
  responsibility = "Write the order record."
  decision = "selected"
}

influence {
  from = [dp.route, dp.log]
  to = fr.root
  effect = "material"
  mechanism = "Route and log choices both reach the accepted order."
}
`
	graph := buildGraph(t, source)

	// When selection runs.
	snapshot := conditionSelect(t, graph, Request{})

	// Then each source DP keeps its own premise state, and the authored record keeps both sources.
	if len(snapshot.Influences) != 2 {
		t.Fatalf("applied influences = %d, want 2", len(snapshot.Influences))
	}
	route := conditionInfluenceFrom(t, snapshot, "dp.route")
	if route.Premises != PremiseState("invalidated") {
		t.Fatalf("route premises = %q, want invalidated", route.Premises)
	}
	logged := conditionInfluenceFrom(t, snapshot, "dp.log")
	if logged.Premises != PremiseState("none") {
		t.Fatalf("log premises = %q, want none", logged.Premises)
	}
	if len(snapshot.Influences[0].Record.From) != 2 {
		t.Fatalf("authored record has %d sources, want 2", len(snapshot.Influences[0].Record.From))
	}
}
