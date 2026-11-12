package schema

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// checkModules checks each module's shape, its parent links, and containment cycles.
func (r *resolver) checkModules() {
	r.parents = make(map[model.ID]parentDecl)
	for _, m := range r.core.Modules {
		if m.Realizes.Present == m.Contains.Present {
			rng := m.Range
			r.report(codeInvalidHierarchy, string(m.ID), &rng, nil,
				fmt.Sprintf("%s must declare exactly one of realizes or contains.", m.ID))
		}
	}
	r.linkWithin()
	r.linkContains()
	r.checkContainmentCycles()
}

// linkWithin records the parent named by each valid within reference. The target
// must be a composite, which is a module that declares contains.
func (r *resolver) linkWithin() {
	for _, m := range r.core.Modules {
		if m.Within == nil || !r.resolves(*m.Within, model.KindModule) {
			continue
		}
		rng := m.Within.Range
		if !r.modules[m.Within.Target].Contains.Present {
			r.report(codeInvalidHierarchy, string(m.Within.Target), &rng, nil,
				fmt.Sprintf("%s is not a composite module. Within must name a module that declares contains.", m.Within.Target))
			continue
		}
		r.declareParent(m.ID, m.Within.Target, rng)
	}
}

// linkContains records the composite that contains each valid module reference.
func (r *resolver) linkContains() {
	for _, c := range r.core.Modules {
		if !c.Contains.Present {
			continue
		}
		for _, ref := range c.Contains.Items {
			if r.resolves(ref, model.KindModule) {
				r.declareParent(ref.Target, c.ID, ref.Range)
			}
		}
	}
}

// declareParent records one parent for a module. A second, different parent is a finding.
func (r *resolver) declareParent(child, parent model.ID, rng hcl.Range) {
	existing, ok := r.parents[child]
	if !ok {
		r.parents[child] = parentDecl{parent: parent, rng: rng}
		return
	}
	if existing.parent == parent {
		return
	}
	r.report(codeInvalidHierarchy, string(child), &rng, []hcl.Range{existing.rng},
		fmt.Sprintf("%s has more than one parent composite. Each module may sit in one composite.", child))
}

// checkContainmentCycles reports a module that contains itself through its parents.
// Each module has at most one parent, so each node has at most one outgoing edge.
func (r *resolver) checkContainmentCycles() {
	nodes := make([]model.ID, 0, len(r.core.Modules))
	for _, m := range r.core.Modules {
		nodes = append(nodes, m.ID)
	}
	g := newDigraph(nodes)
	for child, decl := range r.parents {
		g.addEdge(child, decl.parent, decl.rng)
	}
	for _, rng := range g.backEdgeRanges() {
		rng := rng
		r.report(codeInvalidHierarchy, "module", &rng, nil,
			"Modules contain each other in a cycle. A composite may not sit inside itself.")
	}
}

// checkOwnership checks realization uniqueness and one logical leaf owner per state.
func (r *resolver) checkOwnership() {
	r.checkRealizationOwners()
	r.checkStateOwners()
}

// claim is the first authored claim on an identity.
type claim struct {
	owner model.ID
	rng   hcl.Range
}

func (r *resolver) checkRealizationOwners() {
	claims := make(map[model.ID]claim)
	for _, m := range r.core.Modules {
		for _, ref := range m.Realizes.Items {
			if r.resolves(ref, model.KindDP) {
				r.claimOnce(claims, ref, m.ID, "module")
			}
		}
	}
}

func (r *resolver) checkStateOwners() {
	claims := make(map[model.ID]claim)
	dpClaims := make(map[model.ID][]claim)
	for _, dp := range r.core.DPs {
		for _, ref := range dp.OwnsState.Items {
			if !r.resolves(ref, model.KindState) {
				continue
			}
			if len(r.children[dp.ID]) > 0 {
				rng := ref.Range
				r.report(codeInvalidStructure, string(dp.ID), &rng, nil,
					fmt.Sprintf("%s is not a leaf DP. State ownership belongs to leaf controls.", dp.ID))
			}
			dpClaims[ref.Target] = append(dpClaims[ref.Target], claim{owner: dp.ID, rng: ref.Range})
			r.claimOnce(claims, ref, dp.ID, "logical leaf owner")
		}
	}
	for _, module := range r.core.Modules {
		r.checkModuleStateOwners(module, dpClaims, claims)
	}
}

// checkModuleStateOwners maps matching module claims to the same logical DP
// owner. Without an authored DP claim, exactly one realized leaf must identify it.
func (r *resolver) checkModuleStateOwners(module model.Module, dpClaims map[model.ID][]claim, claims map[model.ID]claim) {
	if len(module.OwnsState.Items) == 0 {
		return
	}
	var leaves []model.ID
	for _, ref := range module.Realizes.Items {
		if r.resolves(ref, model.KindDP) && len(r.children[ref.Target]) == 0 {
			leaves = appendUnique(leaves, ref.Target)
		}
	}
	for _, ref := range module.OwnsState.Items {
		if !r.resolves(ref, model.KindState) {
			continue
		}
		rng := ref.Range
		if module.Contains.Present {
			r.report(codeInvalidStructure, string(module.ID), &rng, nil,
				fmt.Sprintf("%s is a composite module. Composites cannot introduce state ownership.", module.ID))
			continue
		}
		if len(leaves) == 0 {
			r.report(codeInvalidStructure, string(module.ID), &rng, nil,
				fmt.Sprintf("%s realizes no leaf DP. Module state ownership must describe a realized leaf owner.", module.ID))
			continue
		}
		owners := dpClaims[ref.Target]
		var owner model.ID
		for _, declared := range owners {
			if containsID(leaves, declared.owner) {
				owner = declared.owner
				break
			}
		}
		if len(owners) > 0 && owner == "" {
			related := make([]hcl.Range, 0, len(owners))
			for _, declared := range owners {
				related = append(related, declared.rng)
			}
			r.report(codeConflictingOwnership, string(ref.Target), &rng, related,
				fmt.Sprintf("%s claims %s but does not realize its owning DP. A module's declaration must describe the same logical leaf owner.", module.ID, ref.Target))
			continue
		}
		if owner == "" {
			if len(leaves) != 1 {
				r.report(codeInvalidStructure, string(module.ID), &rng, nil,
					fmt.Sprintf("%s does not identify one logical leaf owner for %s. Declare the owning DP's state ownership.", module.ID, ref.Target))
				continue
			}
			owner = leaves[0]
		}
		r.claimOnce(claims, ref, owner, "logical leaf owner")
	}
}

// claimOnce records the first claim on an identity. A later claim by a different
// owner is a finding, pointed at the later claim and related to the first.
func (r *resolver) claimOnce(claims map[model.ID]claim, ref model.Ref, owner model.ID, claimantKind string) {
	prev, ok := claims[ref.Target]
	if !ok {
		claims[ref.Target] = claim{owner: owner, rng: ref.Range}
		return
	}
	if prev.owner == owner {
		return
	}
	rng := ref.Range
	r.report(codeConflictingOwnership, string(ref.Target), &rng, []hcl.Range{prev.rng},
		fmt.Sprintf("%s is claimed by more than one %s. Only one may claim it.", ref.Target, claimantKind))
}
