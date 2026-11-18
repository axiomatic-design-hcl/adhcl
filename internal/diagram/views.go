package diagram

import (
	"fmt"
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// DecisionView is one decision with its candidates.
type DecisionView struct {
	Ref
	Why, Selected, Horizon, Label, Location string
	Provisional                             bool
	Addresses, Context, Workload            []Ref
	Candidates                              []CandidateRow
	Rejected                                []RejectedDP
}

// CandidateRow is one alternative of a decision.
type CandidateRow struct {
	Ref
	Version, Basis, Feasibility string
	Selected                    bool
	DPs, Evidence               []Ref
	Estimates                   []string
}

// RejectedDP is an inactive DP that a candidate of the decision names.
type RejectedDP struct {
	Ref
	Decision, Why string
}

// CouplingView is one coupling record.
type CouplingView struct {
	Ref
	Reason, Condition, Decision, NormalPractice, ResidualRisk string
	IterationBound, Convergence, Stopping, Failure            string
	Label, Location                                           string
	Scope                                                     Ref
	Affects                                                   []Ref
}

// ExpectRow compares an expected classification with the assessed one.
type ExpectRow struct {
	Ref
	Scope                              Ref
	ScopeID, Expected, Assessed, Basis string
	Result                             string // holds, fails, or not applicable
	Order                              []Ref
}

// FlowView is one runtime flow. A step with more than one ref runs in any order.
type FlowView struct {
	Ref
	Input, Output, Label, Location string
	Steps                          [][]Ref
	Owner, After                   []Ref
	Loops                          []LoopView
}

// LoopView is one feedback loop of a flow.
type LoopView struct {
	After, BackTo Ref
	When, Bound   string
}

// JunctionRow derives the module junction of one sibling set from its classification.
type JunctionRow struct {
	Scope                             Ref
	ScopeID, Classification, Junction string
	Meaning, Bound                    string
	Order, Couplings                  []Ref
}

func (b *builder) decisions() []DecisionView {
	var views []DecisionView
	for _, d := range b.core.Decisions {
		view := DecisionView{Ref: b.ref(d.ID), Why: text(d.Why), Selected: text(d.Selected), Horizon: text(d.Horizon), Label: originLabel(d.Origin), Location: location(d.Range), Addresses: b.modelRefs(d.Addresses), Provisional: d.Provisional != nil && *d.Provisional}
		if d.Context != nil {
			view.Context = []Ref{b.ref(d.Context.Target)}
		}
		if d.Workload != nil {
			view.Workload = []Ref{b.ref(d.Workload.Target)}
		}
		for _, c := range d.Candidates {
			row := CandidateRow{Ref: b.ref(c.ID), Version: text(c.Version), DPs: b.modelRefs(c.DPs), Selected: view.Selected != "" && strings.HasSuffix(string(c.ID), ".candidate."+view.Selected)}
			if c.Information != nil {
				row.Basis = string(c.Information.Basis)
				row.Evidence = b.modelRefs(c.Information.Evidence)
			}
			for _, a := range b.res.Candidates {
				if a.Candidate != c.ID {
					continue
				}
				row.Feasibility = a.Feasibility
				for _, e := range a.Evidence {
					row.Estimates = append(row.Estimates, fmt.Sprintf("%s: probability %.6g–%.6g, information %s–%s bits, confidence %.6g, %d trials, %d failures", e.Evidence, e.Low, e.High, information(e.Best), information(e.Worst), e.Confidence, e.Trials, e.Failures))
				}
			}
			for _, r := range c.DPs {
				dp := b.dps[r.Target]
				if dp == nil || b.res.Snapshot.ActiveDPs[dp.ID] || containsRejected(view.Rejected, dp.ID) {
					continue
				}
				view.Rejected = append(view.Rejected, RejectedDP{Ref: b.ref(dp.ID), Decision: string(dp.Decision), Why: text(dp.Why)})
			}
			view.Candidates = append(view.Candidates, row)
		}
		views = append(views, view)
	}
	return views
}

func containsRejected(rejected []RejectedDP, id model.ID) bool {
	for _, r := range rejected {
		if r.ID == string(id) {
			return true
		}
	}
	return false
}

func (b *builder) couplings() []CouplingView {
	var views []CouplingView
	for _, c := range b.core.Couplings {
		view := CouplingView{Ref: b.ref(c.ID), Scope: b.ref(c.Scope.Target), Affects: b.modelRefs(c.Affects), Reason: c.Reason, Condition: text(c.Condition), Decision: text(c.Decision), ResidualRisk: text(c.ResidualRisk), Convergence: text(c.Convergence), Stopping: text(c.Stopping), Failure: text(c.FailureBehavior), Label: originLabel(c.Origin), Location: location(c.Range)}
		if c.NormalPractice != nil {
			view.NormalPractice = fmt.Sprint(*c.NormalPractice)
		}
		if c.IterationBound != nil {
			view.IterationBound = c.IterationBound.Text('g', -1)
		}
		views = append(views, view)
	}
	return views
}

func (b *builder) expects() []ExpectRow {
	var rows []ExpectRow
	for _, e := range b.core.Expects {
		row := ExpectRow{Ref: b.ref(e.ID), Scope: b.ref(e.Scope.Target), ScopeID: string(e.Scope.Target), Expected: string(e.Classification), Order: b.modelRefs(e.Order.Items), Result: "holds"}
		if e.Basis != nil {
			row.Basis = string(*e.Basis)
		}
		if scope := b.scopes[e.Scope.Target]; scope != nil {
			row.Assessed = string(scope.Classification)
		}
		if b.res.Snapshot.Applies(e.Under) != "applies" {
			row.Result = "not applicable"
		} else {
			for _, f := range b.res.Findings {
				if f.Code == result.CodeFailedExpectation && f.Subject == string(e.ID) {
					row.Result = "fails"
					break
				}
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func (b *builder) flows() []FlowView {
	var views []FlowView
	for _, f := range b.core.Flows {
		view := FlowView{Ref: b.ref(f.ID), Input: text(f.Input), Output: text(f.Output), Label: originLabel(f.Origin), Location: location(f.Range), After: b.modelRefs(f.After)}
		if f.Owner != nil {
			view.Owner = []Ref{b.ref(f.Owner.Target)}
		}
		for _, step := range f.Steps {
			view.Steps = append(view.Steps, b.modelRefs(step))
		}
		for _, loop := range f.Loops {
			lv := LoopView{After: b.ref(loop.After.Target), BackTo: b.ref(loop.BackTo.Target), When: loop.When}
			if loop.Bound != nil {
				lv.Bound = loop.Bound.Text('g', -1)
			}
			view.Loops = append(view.Loops, lv)
		}
		views = append(views, view)
	}
	return views
}

// junctions gives each classified sibling set its junction. The master scope and
// unresolved scopes have none.
func (b *builder) junctions() []JunctionRow {
	var rows []JunctionRow
	for _, scope := range b.orderedScopes() {
		junction := assessment.Junction(scope.Classification)
		if scope.Matrix.Scope == "master" || junction == "" {
			continue
		}
		row := JunctionRow{Scope: b.ref(scope.Matrix.Scope), ScopeID: string(scope.Matrix.Scope), Classification: string(scope.Classification), Junction: junction, Meaning: junctionMeaning[junction]}
		if junction == "C" {
			row.Order = b.refs(scope.Order)
		}
		for _, c := range b.core.Couplings {
			if c.Scope.Target == scope.Matrix.Scope {
				row.Couplings = append(row.Couplings, b.ref(c.ID))
				if c.IterationBound != nil {
					row.Bound = c.IterationBound.Text('g', -1)
				}
			}
		}
		rows = append(rows, row)
	}
	return rows
}
