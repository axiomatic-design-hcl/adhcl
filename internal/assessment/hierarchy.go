package assessment

import (
	"fmt"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

// masterScope names the master matrix. No cells default targets it; each leaf cell takes the default of the sibling set that holds both its FR and its DP.
const masterScope model.ID = "master"

// MasterMatrix builds the leaf matrix. Its rows are the active leaf FRs and its
// columns are the distinct active leaf DPs, both in authored order. Each cell
// resolves from its exact influences first. When none applies, it takes the
// cells default of the sibling set that holds both its FR and its DP. A pair
// that spans two branches has no such set, so it stays unknown. Parent effects
// are not expanded into leaf cells.
func MasterMatrix(snapshot *selection.Snapshot) (Matrix, []result.Finding) {
	set := model.SiblingSet{
		Scope: masterScope,
		FRs:   leafFRs(snapshot),
		DPs:   leafDPs(snapshot),
	}
	matrix := newMatrix(snapshot, set)
	defaults := siblingDefaults(snapshot.Graph)
	governing := indexSiblingMembership(&matrix, snapshot.Graph)
	for scope := range defaults {
		if !governing[scope] {
			delete(defaults, scope)
		}
	}
	matrix.defaults = defaults
	findings := resolveExactCells(snapshot, &matrix)
	return matrix, append(findings, defaultConflicts(snapshot.Graph, defaults, governing)...)
}

// siblingDefaults resolves the cells default of every sibling set, keyed by scope.
func siblingDefaults(graph *model.Graph) map[model.ID]scopeDefault {
	defaults := make(map[model.ID]scopeDefault)
	for _, record := range graph.Core.Cells {
		found := defaults[record.Scope.Target]
		found.add(record)
		defaults[record.Scope.Target] = found
	}
	return defaults
}

// indexSiblingMembership indexes each leaf endpoint's owning sibling scope.
// Only scopes with both a leaf FR and a leaf DP can govern a master cell.
func indexSiblingMembership(matrix *Matrix, graph *model.Graph) map[model.ID]bool {
	governing := make(map[model.ID]bool)
	for _, set := range graph.SiblingSets {
		hasFR, hasDP := false, false
		for _, fr := range set.FRs {
			if _, ok := matrix.frScopes[fr]; ok {
				matrix.frScopes[fr] = set.Scope
				hasFR = true
			}
		}
		for _, dp := range set.DPs {
			if _, ok := matrix.dpScopes[dp]; ok {
				matrix.dpScopes[dp] = set.Scope
				hasDP = true
			}
		}
		if hasFR && hasDP {
			governing[set.Scope] = true
		}
	}
	return governing
}

// defaultConflicts reports each governing sibling set whose cells defaults disagree.
func defaultConflicts(graph *model.Graph, defaults map[model.ID]scopeDefault, governing map[model.ID]bool) []result.Finding {
	var findings []result.Finding
	for _, set := range graph.SiblingSets {
		if governing[set.Scope] && defaults[set.Scope].conflict {
			findings = append(findings, defaultConflictFinding(set.Scope, defaults[set.Scope]))
		}
	}
	return findings
}

// Refinement lists the hierarchy obligations that the leaf matrix cannot
// discharge. It reports each applicable material influence that reaches a
// parent FR or leaves from a parent DP, and each active criterion that is
// allocated from a parent criterion. Version 1 has no review input, so every
// obligation stays unresolved.
func Refinement(snapshot *selection.Snapshot) []result.Finding {
	frLeaves := idSet(leafFRs(snapshot))
	dpLeaves := idSet(leafDPs(snapshot))

	var findings []result.Finding
	for _, influence := range snapshot.Influences {
		if !claimsMaterialEffect(influence) {
			continue
		}
		if frLeaves[influence.To] && dpLeaves[influence.From] {
			continue
		}
		findings = append(findings, parentEffectFinding(influence))
	}
	return append(findings, allocationFindings(snapshot)...)
}

// leafFRs returns the active leaf FRs in authored order.
func leafFRs(snapshot *selection.Snapshot) []model.ID {
	var leaves []model.ID
	for _, fr := range snapshot.Graph.Core.FRs {
		if isLeafFR(snapshot, fr.ID) {
			leaves = append(leaves, fr.ID)
		}
	}
	return leaves
}

// leafDPs returns the active leaf DPs in authored order. Each DP appears once,
// even when it controls several FRs.
func leafDPs(snapshot *selection.Snapshot) []model.ID {
	var leaves []model.ID
	for _, dp := range snapshot.Graph.Core.DPs {
		if isLeafDP(snapshot, dp.ID) {
			leaves = append(leaves, dp.ID)
		}
	}
	return leaves
}

// isLeafFR reports whether an active FR has no active child FR under any of
// its active controlling DPs.
func isLeafFR(snapshot *selection.Snapshot, fr model.ID) bool {
	if !snapshot.ActiveFRs[fr] {
		return false
	}
	for _, dp := range snapshot.Graph.Controllers[fr] {
		if snapshot.ActiveDPs[dp] && hasActiveChildFRs(snapshot, dp) {
			return false
		}
	}
	return true
}

// isLeafDP reports whether an active DP has no active child FR.
func isLeafDP(snapshot *selection.Snapshot, dp model.ID) bool {
	return snapshot.ActiveDPs[dp] && !hasActiveChildFRs(snapshot, dp)
}

// hasActiveChildFRs reports whether a DP parents at least one active FR.
func hasActiveChildFRs(snapshot *selection.Snapshot, dp model.ID) bool {
	for _, child := range snapshot.Graph.Children[dp] {
		if snapshot.ActiveFRs[child] {
			return true
		}
	}
	return false
}

// idSet turns an ID list into a membership set.
func idSet(ids []model.ID) map[model.ID]bool {
	set := make(map[model.ID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// claimsMaterialEffect reports whether an influence applies in the snapshot and
// claims a material effect. An unresolved applicability or an invalidated
// premise disables the claim.
func claimsMaterialEffect(influence selection.AppliedInfluence) bool {
	return influence.Applicability == selection.Applicability("applies") &&
		!isOpenRecord(influence) &&
		recordEffect(influence) == EffectMaterial
}

// parentEffectFinding reports a material influence that sits above leaf level.
// Its subject is the target FR and its range is the authored influence.
func parentEffectFinding(influence selection.AppliedInfluence) result.Finding {
	message := fmt.Sprintf("Material influence from %s to %s sits above leaf level, so the leaf matrix cannot state it. Its refinement stays unresolved.", influence.From, influence.To)
	rng := influence.Record.Range
	return result.Finding{
		Code:    result.CodeUnresolvedRefinement,
		Subject: string(influence.To),
		Message: message,
		Range:   &rng,
	}
}

// allocationFindings reports each active criterion that is allocated from a
// parent criterion. Its subject is the criterion ID.
func allocationFindings(snapshot *selection.Snapshot) []result.Finding {
	var findings []result.Finding
	for _, fr := range snapshot.Graph.Core.FRs {
		if !snapshot.ActiveFRs[fr.ID] {
			continue
		}
		for _, criterion := range fr.Criteria {
			if criterion.AllocatedFrom == nil {
				continue
			}
			message := fmt.Sprintf("Criterion %s is allocated from a parent criterion, and no outcome argument supports that allocation. Its refinement stays unresolved.", criterion.ID)
			findings = append(findings, result.Finding{
				Code:    result.CodeUnresolvedRefinement,
				Subject: string(criterion.ID),
				Message: message,
				Range:   &criterion.Range,
			})
		}
	}
	return findings
}
