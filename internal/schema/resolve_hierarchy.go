package schema

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// checkHierarchy links FRs to the DPs that control them and to the child FRs they
// parent, then checks sibling scopes, allocation, cycles, and reachability.
func (r *resolver) checkHierarchy() {
	r.linkHierarchy()
	r.checkCellScopes()
	r.checkControlScopes()
	r.checkAllocations()
	r.checkCyclesAndReachability()
}

// linkHierarchy fills controllers and children from the references that resolve.
// An invalid reference marks the hierarchy as broken.
func (r *resolver) linkHierarchy() {
	r.controllers = make(map[model.ID][]model.ID)
	r.children = make(map[model.ID][]model.ID)
	for _, dp := range r.core.DPs {
		for _, ref := range dp.Controls {
			if !r.resolves(ref, model.KindFR) {
				r.brokenHierarchy = true
				continue
			}
			r.controllers[ref.Target] = appendUnique(r.controllers[ref.Target], dp.ID)
		}
	}
	for _, fr := range r.core.FRs {
		if fr.Parent == nil {
			continue
		}
		if !r.resolves(*fr.Parent, model.KindDP) {
			r.brokenHierarchy = true
			continue
		}
		r.children[fr.Parent.Target] = append(r.children[fr.Parent.Target], fr.ID)
	}
}

// checkCellScopes rejects defaults that cannot apply to a nonempty sibling set.
// The linked child FRs and root FRs define the same scopes retained in the graph.
func (r *resolver) checkCellScopes() {
	rootNonempty := len(r.rootFRs()) > 0
	for _, cells := range r.core.Cells {
		if !r.resolves(cells.Scope, model.KindDesign, model.KindDP) {
			continue
		}
		scope := cells.Scope.Target
		nonempty := len(r.children[scope]) > 0
		if scope == r.designScope() {
			nonempty = rootNonempty
		}
		if !nonempty {
			rng := cells.Scope.Range
			r.report(codeInvalidStructure, string(scope), &rng, nil,
				fmt.Sprintf("%s has no child sibling set. A cells scope must identify an existing nonempty sibling set.", scope))
		}
	}
}

// frScope is the sibling scope of an FR: the design for a root FR, or the DP that
// parents it. It reports false when the parent reference is invalid.
func (r *resolver) frScope(fr *model.FR) (model.ID, bool) {
	if fr.Parent == nil {
		return r.designScope(), true
	}
	if !r.resolves(*fr.Parent, model.KindDP) {
		return "", false
	}
	return fr.Parent.Target, true
}

// checkControlScopes requires every FR that one DP controls to share one sibling scope.
// It reports the first control whose scope differs from the first control's scope.
func (r *resolver) checkControlScopes() {
	for _, dp := range r.core.DPs {
		var scope model.ID
		found := false
		for _, ref := range dp.Controls {
			if !r.resolves(ref, model.KindFR) {
				continue
			}
			frScope, ok := r.frScope(r.frs[ref.Target])
			if !ok {
				continue
			}
			if !found {
				scope, found = frScope, true
				continue
			}
			if frScope != scope {
				rng := ref.Range
				r.report(codeInvalidHierarchy, string(dp.ID), &rng, nil,
					fmt.Sprintf("%s controls FRs in different sibling sets. Every FR one DP controls must share one parent.", dp.ID))
				break
			}
		}
	}
}

// checkAllocations checks each criterion that names an allocation source.
func (r *resolver) checkAllocations() {
	for _, fr := range r.core.FRs {
		for _, cr := range fr.Criteria {
			if cr.AllocatedFrom == nil || !r.resolves(*cr.AllocatedFrom, model.KindCriterion) {
				continue
			}
			r.checkAllocation(fr, *cr.AllocatedFrom)
		}
	}
}

// checkAllocation requires a child FR to allocate only from criteria of an FR its
// parent DP controls. A root FR may not allocate at all.
func (r *resolver) checkAllocation(fr model.FR, from model.Ref) {
	rng := from.Range
	if fr.Parent == nil {
		r.report(codeInvalidHierarchy, string(from.Target), &rng, nil,
			fmt.Sprintf("%s is a root FR, so it cannot allocate from %s. Only a child FR can.", fr.ID, from.Target))
		return
	}
	if !r.resolves(*fr.Parent, model.KindDP) {
		return
	}
	owner := criterionOwner(from.Target)
	if containsID(r.controllers[owner], fr.Parent.Target) {
		return
	}
	r.report(codeInvalidHierarchy, string(from.Target), &rng, nil,
		fmt.Sprintf("%s is not controlled by %s, the parent of %s. A child FR may allocate only from FRs its parent controls.", owner, fr.Parent.Target, fr.ID))
}

// criterionOwner returns the FR that owns a criterion, from fr.F.criterion.C.
func criterionOwner(id model.ID) model.ID {
	segments := strings.Split(string(id), ".")
	if len(segments) < 2 {
		return id
	}
	return model.ID(segments[0] + "." + segments[1])
}

// checkCyclesAndReachability reports cycles through the FR and DP links, then any
// FR or DP that the design cannot reach. A cycle counts as an explanation for the
// nodes that it traps, so it is not reported again as unreachable.
func (r *resolver) checkCyclesAndReachability() {
	g, ranges := r.hierarchyGraph()
	for _, rng := range g.backEdgeRanges() {
		rng := rng
		r.report(codeInvalidHierarchy, "hierarchy", &rng, nil,
			"The FR and DP hierarchy contains a cycle. Each FR and DP must trace back to the design without looping.")
	}
	if r.brokenHierarchy {
		return
	}
	live := g.reach(r.rootFRs()...)
	explained := g.reach(cyclicNodes(g)...)
	for _, n := range g.nodes {
		if live[n] || explained[n] {
			continue
		}
		rng := ranges[n]
		r.report(codeInvalidHierarchy, string(n), &rng, nil,
			fmt.Sprintf("%s is not reachable from the design. Link it through controls and parents.", n))
	}
}

// hierarchyGraph has an edge from each DP to each child FR, and from each controlled
// FR to the DP that controls it. It also returns the declaration range of each node.
func (r *resolver) hierarchyGraph() (*digraph, map[model.ID]hcl.Range) {
	var nodes []model.ID
	ranges := make(map[model.ID]hcl.Range)
	for _, fr := range r.core.FRs {
		nodes = append(nodes, fr.ID)
		ranges[fr.ID] = fr.Range
	}
	for _, dp := range r.core.DPs {
		nodes = append(nodes, dp.ID)
		ranges[dp.ID] = dp.Range
	}
	g := newDigraph(nodes)
	for _, fr := range r.core.FRs {
		if fr.Parent != nil && r.resolves(*fr.Parent, model.KindDP) {
			g.addEdge(fr.Parent.Target, fr.ID, fr.Parent.Range)
		}
	}
	for _, dp := range r.core.DPs {
		for _, ref := range dp.Controls {
			if r.resolves(ref, model.KindFR) {
				g.addEdge(ref.Target, dp.ID, ref.Range)
			}
		}
	}
	return g, ranges
}

// rootFRs lists the FRs that have no parent, in authored order.
func (r *resolver) rootFRs() []model.ID {
	var roots []model.ID
	for _, fr := range r.core.FRs {
		if fr.Parent == nil {
			roots = append(roots, fr.ID)
		}
	}
	return roots
}

// cyclicNodes lists the nodes that lie on a cycle, in graph order.
func cyclicNodes(g *digraph) []model.ID {
	onCycle := g.cyclic()
	var out []model.ID
	for _, n := range g.nodes {
		if onCycle[n] {
			out = append(out, n)
		}
	}
	return out
}

func appendUnique(list []model.ID, id model.ID) []model.ID {
	if containsID(list, id) {
		return list
	}
	return append(list, id)
}

func containsID(list []model.ID, id model.ID) bool {
	for _, item := range list {
		if item == id {
			return true
		}
	}
	return false
}
