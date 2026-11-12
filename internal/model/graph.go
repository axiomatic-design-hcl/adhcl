package model

import "github.com/hashicorp/hcl/v2"

// Declaration is one labeled identity as authored, with the block that declares it.
type Declaration struct {
	ID    ID
	File  string
	Range hcl.Range
}

// Declarations lists every labeled identity in authored order. Nested criteria
// and candidates are included, because their IDs are global.
func (c Core) Declarations() []Declaration {
	var out []Declaration
	add := func(h Header) {
		out = append(out, Declaration{ID: h.ID, File: h.File, Range: h.Range})
	}
	for _, d := range c.Designs {
		add(d.Header)
	}
	for _, d := range c.Sources {
		add(d.Header)
	}
	for _, d := range c.Stakeholders {
		add(d.Header)
	}
	for _, d := range c.Needs {
		add(d.Header)
	}
	for _, d := range c.Constraints {
		add(d.Header)
	}
	for _, d := range c.Assumptions {
		add(d.Header)
	}
	for _, d := range c.Contexts {
		add(d.Header)
	}
	for _, d := range c.Workloads {
		add(d.Header)
	}
	for _, d := range c.States {
		add(d.Header)
	}
	for _, fr := range c.FRs {
		add(fr.Header)
		for _, cr := range fr.Criteria {
			add(cr.Header)
		}
	}
	for _, d := range c.DPs {
		add(d.Header)
	}
	for _, d := range c.Cells {
		add(d.Header)
	}
	for _, d := range c.Expects {
		add(d.Header)
	}
	for _, d := range c.Couplings {
		add(d.Header)
	}
	for _, d := range c.Decisions {
		add(d.Header)
		for _, cand := range d.Candidates {
			add(cand.Header)
		}
	}
	for _, d := range c.Evidence {
		add(d.Header)
	}
	for _, d := range c.PVs {
		add(d.Header)
	}
	for _, d := range c.Modules {
		add(d.Header)
	}
	for _, d := range c.Flows {
		add(d.Header)
	}
	for _, d := range c.Scenarios {
		add(d.Header)
	}
	return out
}

// SiblingSet is the FRs that share one parent scope and the DPs that control them.
// Scope is the design for root FRs, or the DP that parents the FRs.
type SiblingSet struct {
	Scope ID
	FRs   []ID
	DPs   []ID
}

// Graph is the resolved view of a core. Resolution returns one only when it found
// no blocking defect, so every reference in it names a declared target.
type Graph struct {
	Core        Core
	Design      ID
	SiblingSets []SiblingSet
	Controllers map[ID][]ID
	Children    map[ID][]ID
	Modules     map[ID]ID
	declared    map[ID]struct{}
}

// NewGraph builds a graph from resolved parts. The identity set is taken from
// the core, so Has answers for every labeled declaration.
func NewGraph(core Core, design ID, sets []SiblingSet, controllers, children map[ID][]ID, modules map[ID]ID) *Graph {
	declared := make(map[ID]struct{})
	for _, d := range core.Declarations() {
		declared[d.ID] = struct{}{}
	}
	return &Graph{
		Core:        core,
		Design:      design,
		SiblingSets: sets,
		Controllers: controllers,
		Children:    children,
		Modules:     modules,
		declared:    declared,
	}
}

// SiblingSet returns the sibling set with the given scope.
func (g *Graph) SiblingSet(scope ID) (SiblingSet, bool) {
	for _, set := range g.SiblingSets {
		if set.Scope == scope {
			return set, true
		}
	}
	return SiblingSet{}, false
}

// Has reports whether the core declares the identity.
func (g *Graph) Has(id ID) bool {
	_, ok := g.declared[id]
	return ok
}
