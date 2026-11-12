// Package selection resolves a request against a resolved graph into one snapshot of
// active membership. It reads the graph and never edits its records.
package selection

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// Request chooses at most one candidate and one context. Empty fields mean none.
type Request struct {
	Candidate model.ID // empty, or decision.D.candidate.C
	Context   model.ID // empty, or context.C
}

// Snapshot is the active membership for one request. The graph keeps every record unchanged.
type Snapshot struct {
	Graph      *model.Graph
	Candidate  model.ID // empty for the default snapshot
	Context    model.ID // effective context, empty when none
	Workload   model.ID // effective workload, empty when none
	Horizon    string   // effective horizon, empty when none
	ActiveDPs  map[model.ID]bool
	ActiveFRs  map[model.ID]bool
	Modules    map[model.ID][]model.ID // active module -> its active realized DPs
	Influences []AppliedInfluence      // one per active source DP
}

// Applicability states whether a condition reference holds for a snapshot.
type Applicability string

// PremiseState summarizes the assumptions that an influence depends on.
type PremiseState string

// AppliedInfluence is one influence expanded to one active source DP.
type AppliedInfluence struct {
	Record        *model.Influence // the authored record, unchanged
	From          model.ID         // one expanded source DP
	To            model.ID         // target FR
	Applicability Applicability
	Premises      PremiseState
}

const (
	invalidSelection = "invalid_selection"

	applicabilityApplies       Applicability = "applies"
	applicabilityNotApplicable Applicability = "not_applicable"
	applicabilityUnresolved    Applicability = "unresolved"

	premisesNone        PremiseState = "none"
	premisesConditional PremiseState = "conditional"
	premisesInvalidated PremiseState = "invalidated"
)

// Select resolves the request. It returns a nil snapshot and invalid_selection findings when
// the request or the graph conflicts with itself. Findings keep the authored ranges.
func Select(graph *model.Graph, request Request) (*Snapshot, []result.Finding) {
	ix := newIndex(graph)
	if len(ix.membershipFindings) > 0 {
		return nil, ix.membershipFindings
	}
	choice, findings := ix.chooseCandidate(request.Candidate)
	if len(findings) > 0 {
		return nil, findings
	}
	conds, condFindings := ix.resolveConditions(choice, request.Context)
	activeDPs, dpFindings := ix.activeDPs(choice)
	activeFRs := ix.activeFRs(activeDPs)
	findings = append(findings, condFindings...)
	findings = append(findings, dpFindings...)
	findings = append(findings, ix.checkClosure(activeDPs, activeFRs)...)
	if len(findings) > 0 {
		return nil, findings
	}
	snapshot := &Snapshot{
		Graph:     graph,
		Candidate: request.Candidate,
		Context:   conds.context,
		Workload:  conds.workload,
		Horizon:   conds.horizon,
		ActiveDPs: activeDPs,
		ActiveFRs: activeFRs,
		Modules:   ix.activeModules(activeDPs),
	}
	snapshot.Influences = ix.appliedInfluences(snapshot)
	return snapshot, nil
}

// index holds lookups over one graph so that selection can find records by ID.
type index struct {
	core               *model.Core
	dps                map[model.ID]*model.DP
	contexts           map[model.ID]*model.Context
	assumptions        map[model.ID]*model.Assumption
	candidates         map[model.ID]*candidateChoice
	dpDecisions        map[model.ID]model.ID
	membershipFindings []result.Finding
}

// candidateChoice is a candidate together with the decision that declares it.
type candidateChoice struct {
	decision  *model.Decision
	candidate *model.Candidate
}

func newIndex(graph *model.Graph) *index {
	core := &graph.Core
	ix := &index{
		core:        core,
		dps:         make(map[model.ID]*model.DP, len(core.DPs)),
		contexts:    make(map[model.ID]*model.Context, len(core.Contexts)),
		assumptions: make(map[model.ID]*model.Assumption, len(core.Assumptions)),
		candidates:  make(map[model.ID]*candidateChoice),
		dpDecisions: make(map[model.ID]model.ID),
	}
	for i := range core.DPs {
		ix.dps[core.DPs[i].ID] = &core.DPs[i]
	}
	for i := range core.Contexts {
		ix.contexts[core.Contexts[i].ID] = &core.Contexts[i]
	}
	for i := range core.Assumptions {
		ix.assumptions[core.Assumptions[i].ID] = &core.Assumptions[i]
	}
	for i := range core.Decisions {
		decision := &core.Decisions[i]
		for j := range decision.Candidates {
			ix.candidates[decision.Candidates[j].ID] = &candidateChoice{
				decision:  decision,
				candidate: &decision.Candidates[j],
			}
			for k := range decision.Candidates[j].DPs {
				ref := &decision.Candidates[j].DPs[k]
				owner, listed := ix.dpDecisions[ref.Target]
				if listed && owner != decision.ID {
					ix.membershipFindings = append(ix.membershipFindings, invalidFinding(
						string(ref.Target),
						&ref.Range,
						fmt.Sprintf("DP %s is listed by distinct decisions %s and %s.", ref.Target, owner, decision.ID),
					))
				} else {
					ix.dpDecisions[ref.Target] = decision.ID
				}
			}
		}
	}
	return ix
}

// chooseCandidate looks up the requested candidate. An empty request selects the default snapshot.
func (ix *index) chooseCandidate(id model.ID) (*candidateChoice, []result.Finding) {
	if id == "" {
		return nil, nil
	}
	choice, ok := ix.candidates[id]
	if !ok {
		return nil, []result.Finding{
			invalidFinding(string(id), nil, fmt.Sprintf("The requested candidate %s is not declared.", id)),
		}
	}
	return choice, nil
}

// invalidFinding builds an invalid_selection finding. A nil span leaves the finding unplaced.
func invalidFinding(subject string, span *hcl.Range, message string) result.Finding {
	finding := result.Finding{
		Code:     invalidSelection,
		Severity: result.Error,
		Subject:  subject,
		Message:  message,
	}
	if span != nil {
		copied := *span
		finding.Range = &copied
	}
	return finding
}

// refTarget returns the target of a reference, or the empty ID when the reference is absent.
func refTarget(ref *model.Ref) model.ID {
	if ref == nil {
		return ""
	}
	return ref.Target
}

// firstID returns the first non-empty ID.
func firstID(ids ...model.ID) model.ID {
	for _, id := range ids {
		if id != "" {
			return id
		}
	}
	return ""
}

// firstString returns the first present string value.
func firstString(values ...*string) string {
	for _, value := range values {
		if value != nil {
			return *value
		}
	}
	return ""
}
