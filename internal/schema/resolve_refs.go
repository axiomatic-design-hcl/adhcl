package schema

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// traversal is one authored reference with the kinds its field admits.
type traversal struct {
	ref   model.Ref
	kinds []model.Kind
}

// collector gathers the traversals of a core in authored order.
type collector struct {
	out []traversal
}

func (c *collector) refs(refs []model.Ref, kinds ...model.Kind) {
	for _, ref := range refs {
		c.out = append(c.out, traversal{ref: ref, kinds: kinds})
	}
}

func (c *collector) one(ref *model.Ref, kinds ...model.Kind) {
	if ref != nil {
		c.refs([]model.Ref{*ref}, kinds...)
	}
}

func (c *collector) optional(list model.Optional[model.Ref], kinds ...model.Kind) {
	c.refs(list.Items, kinds...)
}

func (c *collector) header(h model.Header) {
	c.refs(h.Cites, model.KindSource)
}

func (c *collector) unlabeled(u model.Unlabeled) {
	c.refs(u.Cites, model.KindSource)
}

// coreTraversals lists every authored reference in the core with the kinds its
// field admits. The kinds follow the field table in the schema.
func coreTraversals(core model.Core) []traversal {
	c := &collector{}
	for _, d := range core.Designs {
		c.header(d.Header)
		c.one(d.Workload, model.KindWorkload)
	}
	for _, d := range core.Stakeholders {
		c.header(d.Header)
	}
	for _, d := range core.Needs {
		c.header(d.Header)
		c.one(d.Stakeholder, model.KindStakeholder)
	}
	for _, d := range core.Constraints {
		c.header(d.Header)
	}
	for _, d := range core.Assumptions {
		c.header(d.Header)
	}
	for _, d := range core.Contexts {
		c.header(d.Header)
		c.refs(d.Assumes, model.KindAssumption)
		c.refs(d.Excludes, model.KindContext)
		c.one(d.Workload, model.KindWorkload)
	}
	for _, d := range core.Workloads {
		c.header(d.Header)
	}
	for _, d := range core.States {
		c.header(d.Header)
	}
	for _, fr := range core.FRs {
		c.frTraversals(fr)
	}
	for _, d := range core.DPs {
		c.header(d.Header)
		c.refs(d.Controls, model.KindFR)
		c.optional(d.OwnsState, model.KindState)
		c.refs(d.ReadsState, model.KindState)
		c.refs(d.BoundedBy, model.KindConstraint)
		c.refs(d.HoldsIf, model.KindAssumption)
	}
	for _, d := range core.Influences {
		c.unlabeled(d.Unlabeled)
		c.refs(d.From, model.KindDP)
		c.refs([]model.Ref{d.To}, model.KindFR)
		c.one(d.Under, model.KindContext, model.KindWorkload)
		c.refs(d.HoldsIf, model.KindAssumption)
	}
	for _, d := range core.Cells {
		c.header(d.Header)
		c.refs([]model.Ref{d.Scope}, model.KindDesign, model.KindDP)
	}
	for _, d := range core.Expects {
		c.header(d.Header)
		c.refs([]model.Ref{d.Scope}, model.KindDesign, model.KindDP)
		c.optional(d.Order, model.KindDP)
		c.one(d.Under, model.KindContext, model.KindWorkload)
	}
	for _, d := range core.Couplings {
		c.header(d.Header)
		c.refs([]model.Ref{d.Scope}, model.KindDesign, model.KindDP)
		c.refs(d.Affects, model.KindFR, model.KindDP)
	}
	for _, d := range core.Decisions {
		c.decisionTraversals(d)
	}
	for _, d := range core.Evidence {
		c.header(d.Header)
		c.refs([]model.Ref{d.Scenario}, model.KindScenario)
		c.refs([]model.Ref{d.Candidate}, model.KindCandidate)
		c.refs([]model.Ref{d.Context}, model.KindContext)
		c.refs([]model.Ref{d.Workload}, model.KindWorkload)
	}
	for _, d := range core.PVs {
		c.header(d.Header)
		c.refs(d.Realizes, model.KindDP)
	}
	for _, d := range core.Modules {
		c.moduleTraversals(d)
	}
	for _, d := range core.Flows {
		c.flowTraversals(d)
	}
	for _, d := range core.Scenarios {
		c.header(d.Header)
		c.optional(d.Verifies, model.KindCriterion)
		c.refs(d.Vary, model.KindDP)
		c.refs(d.ExpectAffected, model.KindFR)
		c.refs(d.ExpectUnaffected, model.KindFR)
		c.one(d.Under, model.KindContext, model.KindWorkload)
	}
	return c.out
}

func (c *collector) frTraversals(fr model.FR) {
	c.header(fr.Header)
	c.optional(fr.Need, model.KindNeed)
	c.one(fr.Parent, model.KindDP)
	c.refs(fr.BoundedBy, model.KindConstraint)
	for _, cr := range fr.Criteria {
		c.header(cr.Header)
		c.one(cr.Under, model.KindContext, model.KindWorkload)
		c.one(cr.AllocatedFrom, model.KindCriterion)
	}
}

func (c *collector) decisionTraversals(d model.Decision) {
	c.header(d.Header)
	c.refs(d.Addresses, model.KindFR)
	c.one(d.Context, model.KindContext)
	c.one(d.Workload, model.KindWorkload)
	for _, cand := range d.Candidates {
		c.header(cand.Header)
		c.refs(cand.DPs, model.KindDP)
		c.one(cand.Context, model.KindContext)
		c.one(cand.Workload, model.KindWorkload)
		if cand.Information != nil {
			c.unlabeled(cand.Information.Unlabeled)
			c.refs(cand.Information.Evidence, model.KindEvidence)
		}
	}
}

func (c *collector) moduleTraversals(m model.Module) {
	c.header(m.Header)
	c.optional(m.Realizes, model.KindDP)
	c.optional(m.Contains, model.KindModule)
	c.one(m.Within, model.KindModule)
	c.optional(m.OwnsState, model.KindState)
	c.optional(m.Dependencies, model.KindModule)
	c.optional(m.AcceptanceChecks, model.KindCriterion, model.KindScenario)
}

func (c *collector) flowTraversals(f model.Flow) {
	c.header(f.Header)
	for _, group := range f.Steps {
		c.refs(group, model.KindModule, model.KindFR, model.KindDP)
	}
	c.one(f.Owner, model.KindModule)
	c.refs(f.After, model.KindFlow)
	for _, loop := range f.Loops {
		c.unlabeled(loop.Unlabeled)
		c.refs([]model.Ref{loop.After, loop.BackTo}, model.KindModule, model.KindFR, model.KindDP)
	}
}

// checkInformationEvidence requires every requested observation to belong to
// its exact enclosing candidate, regardless of the requested information basis.
func (r *resolver) checkInformationEvidence() {
	evidence := make(map[model.ID]*model.Evidence, len(r.core.Evidence))
	for i := range r.core.Evidence {
		ev := &r.core.Evidence[i]
		evidence[ev.ID] = ev
	}
	for _, decision := range r.core.Decisions {
		for _, candidate := range decision.Candidates {
			if candidate.Information == nil {
				continue
			}
			for _, ref := range candidate.Information.Evidence {
				ev := evidence[ref.Target]
				if ev == nil || !r.resolves(ev.Candidate, model.KindCandidate) || ev.Candidate.Target == candidate.ID {
					continue
				}
				rng := ref.Range
				r.report(codeInvalidEvidence, string(ref.Target), &rng, []hcl.Range{ev.Candidate.Range},
					fmt.Sprintf("%s names %s, not its requesting candidate %s. Information evidence must belong to its exact enclosing candidate.", ref.Target, ev.Candidate.Target, candidate.ID))
			}
		}
	}
}
