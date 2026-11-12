package schema

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

const (
	codeInvalidReference     = "invalid_reference"
	codeInvalidHierarchy     = "invalid_hierarchy"
	codeConflictingOwnership = "conflicting_ownership"
)

// stepKinds are the identities that a flow step or loop endpoint may name.
var stepKinds = []model.Kind{model.KindModule, model.KindFR, model.KindDP}

// resolver checks the relations of one decoded core and records every finding.
// It reads only facts that the decoders already checked for shape.
type resolver struct {
	core     model.Core
	findings []result.Finding

	declared map[model.ID]bool
	frs      map[model.ID]*model.FR
	modules  map[model.ID]*model.Module

	// controllers maps an FR to the DPs that control it, and children maps a DP to
	// the child FRs it parents. Both follow authored order.
	controllers map[model.ID][]model.ID
	children    map[model.ID][]model.ID

	// parents maps a module to its parent composite, with the reference that declares it.
	parents map[model.ID]parentDecl

	// brokenHierarchy is set when a hierarchy reference is invalid. The reachability
	// check then stays silent, because an unknown link can hide the real cause.
	brokenHierarchy bool
}

type parentDecl struct {
	parent model.ID
	rng    hcl.Range
}

// Resolve checks identities, references, hierarchy, module containment, ownership,
// and flow order of a decoded core. It returns a graph only when no finding blocks it.
func Resolve(core model.Core) (*model.Graph, []result.Finding) {
	r := newResolver(core)
	r.checkIdentities()
	r.checkDesignCount()
	r.checkReferences()
	r.checkHierarchy()
	r.checkModules()
	r.checkOwnership()
	r.checkFlows()
	if len(r.findings) > 0 {
		return nil, r.findings
	}
	return r.buildGraph(), nil
}

func newResolver(core model.Core) *resolver {
	r := &resolver{
		core:     core,
		declared: make(map[model.ID]bool),
		frs:      make(map[model.ID]*model.FR),
		modules:  make(map[model.ID]*model.Module),
	}
	for _, d := range core.Declarations() {
		r.declared[d.ID] = true
	}
	for i := range core.FRs {
		r.frs[core.FRs[i].ID] = &core.FRs[i]
	}
	for i := range core.Modules {
		r.modules[core.Modules[i].ID] = &core.Modules[i]
	}
	return r
}

// newFinding builds an error finding. The range is copied so that callers may pass
// the address of a local value.
func newFinding(code, subject string, rng *hcl.Range, related []hcl.Range, message string) result.Finding {
	return result.Finding{
		Code:     code,
		Severity: result.Error,
		Subject:  subject,
		Message:  message,
		Range:    rng,
		Related:  related,
	}
}

func (r *resolver) report(code, subject string, rng *hcl.Range, related []hcl.Range, message string) {
	r.findings = append(r.findings, newFinding(code, subject, rng, related, message))
}

// checkIdentities reports each label that more than one block declares. The
// first declaration is the primary range, and the later ones are related.
func (r *resolver) checkIdentities() {
	byID := make(map[model.ID][]model.Declaration)
	var order []model.ID
	for _, d := range r.core.Declarations() {
		if _, seen := byID[d.ID]; !seen {
			order = append(order, d.ID)
		}
		byID[d.ID] = append(byID[d.ID], d)
	}
	for _, id := range order {
		group := byID[id]
		if len(group) < 2 {
			continue
		}
		first := group[0].Range
		related := make([]hcl.Range, 0, len(group)-1)
		for _, d := range group[1:] {
			related = append(related, d.Range)
		}
		r.report(codeDuplicateIdentity, string(id), &first, related,
			fmt.Sprintf("The label %s is declared more than once. Each label must name one block.", id))
	}
}

// checkDesignCount requires exactly one design. A second design with a different
// label is reported here; a repeated label is already covered by checkIdentities.
func (r *resolver) checkDesignCount() {
	designs := r.core.Designs
	if len(designs) == 0 {
		r.report(codeInvalidStructure, "design", nil, nil,
			"The core has no design block. Exactly one design is required.")
		return
	}
	first := designs[0]
	for _, extra := range designs[1:] {
		if extra.ID == first.ID {
			continue
		}
		rng := extra.Range
		r.report(codeDuplicateIdentity, string(extra.ID), &rng, []hcl.Range{first.Range},
			"The core has more than one design. Exactly one design is required.")
	}
}

// checkReferences reports each authored reference that names no declared identity,
// or names one of the wrong kind for its field.
func (r *resolver) checkReferences() {
	for _, t := range coreTraversals(r.core) {
		if r.resolves(t.ref, t.kinds...) {
			continue
		}
		rng := t.ref.Range
		target := string(t.ref.Target)
		if !r.declared[t.ref.Target] {
			r.report(codeInvalidReference, target, &rng, nil,
				fmt.Sprintf("%s does not name a declared block. This field needs a %s.", target, kindNames(t.kinds)))
			continue
		}
		r.report(codeInvalidReference, target, &rng, nil,
			fmt.Sprintf("%s is a %s, but this field needs a %s.", target, t.ref.Target.Kind(), kindNames(t.kinds)))
	}
	r.checkInformationEvidence()
}

// resolves reports whether the reference names a declared identity of an admitted kind.
func (r *resolver) resolves(ref model.Ref, kinds ...model.Kind) bool {
	if !r.declared[ref.Target] {
		return false
	}
	return hasKind(kinds, ref.Target.Kind())
}

func hasKind(kinds []model.Kind, kind model.Kind) bool {
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// kindNames joins kinds for a message, such as "fr or dp".
func kindNames(kinds []model.Kind) string {
	names := make([]string, 0, len(kinds))
	for _, k := range kinds {
		names = append(names, string(k))
	}
	return strings.Join(names, " or ")
}

// designScope is the scope of root FRs. It is empty only when no design is declared,
// and then a finding already blocks the graph.
func (r *resolver) designScope() model.ID {
	if len(r.core.Designs) == 0 {
		return ""
	}
	return r.core.Designs[0].ID
}

// buildGraph assembles the resolved graph. It runs only when no finding exists.
func (r *resolver) buildGraph() *model.Graph {
	modules := make(map[model.ID]model.ID, len(r.parents))
	for child, decl := range r.parents {
		modules[child] = decl.parent
	}
	return model.NewGraph(r.core, r.designScope(), r.siblingSets(), r.controllers, r.children, modules)
}

// siblingSets lists the design's root set first, then one set per DP that has child FRs, in authored order.
func (r *resolver) siblingSets() []model.SiblingSet {
	var rootFRs []model.ID
	for _, fr := range r.core.FRs {
		if fr.Parent == nil {
			rootFRs = append(rootFRs, fr.ID)
		}
	}
	sets := []model.SiblingSet{r.siblingSetOf(r.designScope(), rootFRs)}
	for _, dp := range r.core.DPs {
		if children := r.children[dp.ID]; len(children) > 0 {
			sets = append(sets, r.siblingSetOf(dp.ID, children))
		}
	}
	return sets
}

// siblingSetOf lists the DPs, in authored order, that control any FR of the set.
func (r *resolver) siblingSetOf(scope model.ID, frs []model.ID) model.SiblingSet {
	inSet := make(map[model.ID]bool, len(frs))
	for _, id := range frs {
		inSet[id] = true
	}
	var dps []model.ID
	for _, dp := range r.core.DPs {
		for _, ref := range dp.Controls {
			if inSet[ref.Target] {
				dps = append(dps, dp.ID)
				break
			}
		}
	}
	return model.SiblingSet{Scope: scope, FRs: frs, DPs: dps}
}
