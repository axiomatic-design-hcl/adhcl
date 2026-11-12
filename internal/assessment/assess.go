package assessment

import (
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

// Result is the complete structural assessment of one selected snapshot.
type Result struct {
	Scopes     []ScopeResult
	Master     ScopeResult
	Candidates []CandidateResult
	Findings   []result.Finding
}

// Assess runs every assessment stage over the snapshot in pinned order and collects
// the findings of each stage in that order. It never edits the snapshot.
func Assess(snapshot *selection.Snapshot) Result {
	var out Result
	scopes, scopeFindings := assessSiblingSets(snapshot)
	master, masterFindings := assessMaster(snapshot)
	out.Scopes = scopes
	out.Master = master
	out.Findings = append(out.Findings, scopeFindings...)
	out.Findings = append(out.Findings, masterFindings...)
	out.Findings = append(out.Findings, Refinement(snapshot)...)
	out.Findings = append(out.Findings, Completeness(snapshot, scopes)...)
	out.Findings = append(out.Findings, SupportGaps(snapshot)...)
	out.Findings = append(out.Findings, EvaluateExpectations(snapshot, scopes)...)
	candidates, candidateFindings := AssessCandidates(snapshot)
	out.Candidates = candidates
	out.Findings = append(out.Findings, candidateFindings...)
	return out
}

// assessSiblingSets resolves and classifies every sibling set that has at least one active FR.
func assessSiblingSets(snapshot *selection.Snapshot) ([]ScopeResult, []result.Finding) {
	var scopes []ScopeResult
	var findings []result.Finding
	for _, set := range snapshot.Graph.SiblingSets {
		if !hasActiveFR(snapshot, set) {
			continue
		}
		matrix, cellFindings := ResolveCells(snapshot, set)
		scope, classifyFindings := Classify(matrix, controlsOf(snapshot.Graph, matrix))
		scopes = append(scopes, scope)
		findings = append(findings, cellFindings...)
		findings = append(findings, classifyFindings...)
	}
	return scopes, findings
}

// assessMaster expands the leaf axes into one master matrix and classifies it.
func assessMaster(snapshot *selection.Snapshot) (ScopeResult, []result.Finding) {
	matrix, findings := MasterMatrix(snapshot)
	scope, classifyFindings := Classify(matrix, controlsOf(snapshot.Graph, matrix))
	return scope, append(findings, classifyFindings...)
}

// hasActiveFR reports whether the sibling set has at least one active FR.
func hasActiveFR(snapshot *selection.Snapshot, set model.SiblingSet) bool {
	for _, fr := range set.FRs {
		if snapshot.ActiveFRs[fr] {
			return true
		}
	}
	return false
}

// controlsOf maps each DP of the matrix to the matrix FRs it controls, in authored FR order.
// The graph stores controllers by FR, so this inverts that map and keeps only matrix members.
func controlsOf(graph *model.Graph, matrix Matrix) map[model.ID][]model.ID {
	activeDP := make(map[model.ID]bool, len(matrix.DPs))
	for _, dp := range matrix.DPs {
		activeDP[dp] = true
	}
	controls := make(map[model.ID][]model.ID)
	for _, fr := range matrix.FRs {
		for _, dp := range graph.Controllers[fr] {
			if activeDP[dp] {
				controls[dp] = append(controls[dp], fr)
			}
		}
	}
	return controls
}
