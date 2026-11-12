package schema

import (
	"fmt"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// checkFlows checks flow prerequisites for cycles, then checks each loop's endpoints.
func (r *resolver) checkFlows() {
	r.checkFlowOrder()
	for _, f := range r.core.Flows {
		for _, loop := range f.Loops {
			r.checkLoop(f, loop)
		}
	}
}

// checkFlowOrder reports each prerequisite edge that closes a cycle among flows.
func (r *resolver) checkFlowOrder() {
	nodes := make([]model.ID, 0, len(r.core.Flows))
	for _, f := range r.core.Flows {
		nodes = append(nodes, f.ID)
	}
	g := newDigraph(nodes)
	for _, f := range r.core.Flows {
		for _, ref := range f.After {
			if r.resolves(ref, model.KindFlow) {
				g.addEdge(f.ID, ref.Target, ref.Range)
			}
		}
	}
	for _, rng := range g.backEdgeRanges() {
		rng := rng
		r.report(codeInvalidHierarchy, "flow", &rng, nil,
			"Flow prerequisites form a cycle. A flow may not wait on itself.")
	}
}

// checkLoop requires a loop's after and back_to steps to be steps of the enclosing
// flow, with back_to earlier than after. An invalid endpoint is skipped, because
// the reference check has already reported it.
func (r *resolver) checkLoop(f model.Flow, loop model.Loop) {
	if !r.resolves(loop.After, stepKinds...) || !r.resolves(loop.BackTo, stepKinds...) {
		return
	}
	rng := loop.Range
	positions := stepPositions(f)
	after, hasAfter := positions[loop.After.Target]
	back, hasBack := positions[loop.BackTo.Target]
	if !hasAfter || !hasBack {
		r.report(codeInvalidStructure, string(f.ID), &rng, nil,
			fmt.Sprintf("The loop in %s must name steps of that flow for after and back_to.", f.ID))
		return
	}
	if back >= after {
		r.report(codeInvalidStructure, string(f.ID), &rng, nil,
			fmt.Sprintf("The loop in %s must return to a step that comes before its after step.", f.ID))
	}
}

// stepPositions maps each step to the index of its first group in the flow.
func stepPositions(f model.Flow) map[model.ID]int {
	positions := make(map[model.ID]int)
	for i, group := range f.Steps {
		for _, ref := range group {
			if _, seen := positions[ref.Target]; !seen {
				positions[ref.Target] = i
			}
		}
	}
	return positions
}
