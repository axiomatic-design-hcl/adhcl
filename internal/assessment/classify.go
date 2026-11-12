package assessment

import (
	"fmt"
	"slices"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

// Classify pairs each FR with its controlling DP and classifies the scope.
// Controls map a DP to the FRs it controls. Findings name the scope as their subject.
func Classify(matrix Matrix, controls map[model.ID][]model.ID) (ScopeResult, []result.Finding) {
	graph := newControlGraph(matrix, controls)
	match := maximumMatching(graph)
	findings := coverageFindings(matrix, graph, match)
	if !match.coversAllFRs() {
		return ScopeResult{Matrix: matrix, Classification: Unresolved}, findings
	}
	if alternative, ok := alternativePairing(graph, match); ok {
		findings = append(findings, newFinding(result.CodeAmbiguousPairing, matrix.Scope,
			fmt.Sprintf("Scope %s has more than one complete pairing of requirements to controls.", matrix.Scope)))
		witnesses := []map[model.ID]model.ID{pairingOf(graph, match.frToDP), pairingOf(graph, alternative.frToDP)}
		return ScopeResult{Matrix: matrix, PairingWitnesses: witnesses, Classification: Unresolved}, findings
	}
	scope := ScopeResult{Matrix: matrix, Pairing: pairingOf(graph, match.frToDP), Classification: Unresolved}
	if len(graph.frs) != len(graph.dps) {
		return scope, findings
	}
	square := analyzeSquare(matrix, graph, match)
	scope.Classification = square.classification
	scope.Order = square.order
	scope.Cycles = square.cycles
	return scope, append(findings, square.findings...)
}

// controlGraph is the bipartite control graph restricted to the matrix axes.
// Vertices are indices into the authored FR and DP axes.
type controlGraph struct {
	frs        []model.ID
	dps        []model.ID
	frControls [][]int // FR index -> controlling DP indices, in authored DP order
	dpControls [][]int // DP index -> controlled FR indices, in authored FR order
}

func newControlGraph(matrix Matrix, controls map[model.ID][]model.ID) controlGraph {
	frIndex := indexByID(matrix.FRs)
	graph := controlGraph{
		frs:        matrix.FRs,
		dps:        matrix.DPs,
		frControls: make([][]int, len(matrix.FRs)),
		dpControls: make([][]int, len(matrix.DPs)),
	}
	for j, dp := range matrix.DPs {
		for _, fr := range controls[dp] {
			i, ok := frIndex[fr]
			if !ok || slices.Contains(graph.dpControls[j], i) {
				continue
			}
			graph.dpControls[j] = append(graph.dpControls[j], i)
			graph.frControls[i] = append(graph.frControls[i], j)
		}
	}
	return graph
}

func indexByID(ids []model.ID) map[model.ID]int {
	index := make(map[model.ID]int, len(ids))
	for i, id := range ids {
		index[id] = i
	}
	return index
}

// matching holds a maximum bipartite matching. An entry of -1 marks an unmatched vertex.
type matching struct {
	frToDP []int
	dpToFR []int
}

func maximumMatching(graph controlGraph) matching {
	match := matching{frToDP: unsetIndices(len(graph.frs)), dpToFR: unsetIndices(len(graph.dps))}
	for fr := range graph.frs {
		match.augment(graph, fr, make([]bool, len(graph.dps)), -1, -1)
	}
	return match
}

// augment matches fr, rerouting earlier matches along an augmenting path when needed.
// The omitted edge is excluded while finding an alternative pairing.
func (m matching) augment(graph controlGraph, fr int, visited []bool, omittedFR, omittedDP int) bool {
	for _, dp := range graph.frControls[fr] {
		if visited[dp] || (fr == omittedFR && dp == omittedDP) {
			continue
		}
		visited[dp] = true
		if m.dpToFR[dp] == -1 || m.augment(graph, m.dpToFR[dp], visited, omittedFR, omittedDP) {
			m.frToDP[fr] = dp
			m.dpToFR[dp] = fr
			return true
		}
	}
	return false
}

func (m matching) coversAllFRs() bool {
	return !slices.Contains(m.frToDP, -1)
}

func unsetIndices(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = -1
	}
	return out
}

// coverageFindings reports FRs without a distinct control and DPs outside every
// complete pairing. It runs before any classification decision.
func coverageFindings(matrix Matrix, graph controlGraph, match matching) []result.Finding {
	var findings []result.Finding
	for i, fr := range graph.frs {
		if match.frToDP[i] == -1 {
			findings = append(findings, newFinding(result.CodeUncontrolledRequirement, matrix.Scope,
				fmt.Sprintf("Requirement %s has no distinct control in scope %s.", fr, matrix.Scope)))
		}
	}
	for j, dp := range graph.dps {
		if isSpareDP(graph, match, j) {
			findings = append(findings, newFinding(result.CodeRedundantControl, matrix.Scope,
				fmt.Sprintf("Design parameter %s lies outside every complete pairing in scope %s.", dp, matrix.Scope)))
		}
	}
	return findings
}

// isSpareDP reports whether an unmatched DP is outside every complete pairing.
// Without a complete pairing, every unmatched DP is outside. With one, an
// unmatched DP that controls an FR can replace that FR's DP, so it is not spare.
func isSpareDP(graph controlGraph, match matching, dp int) bool {
	if match.dpToFR[dp] != -1 {
		return false
	}
	return !match.coversAllFRs() || len(graph.dpControls[dp]) == 0
}

// alternativePairing finds a second complete matching without enumerating
// assignments. Any alternative omits at least one edge of the first matching.
// Remove each such edge in turn and seek one augmenting path to restore coverage.
func alternativePairing(graph controlGraph, match matching) (matching, bool) {
	alternative := matching{frToDP: make([]int, len(match.frToDP)), dpToFR: make([]int, len(match.dpToFR))}
	visited := make([]bool, len(graph.dps))
	for fr, dp := range match.frToDP {
		copy(alternative.frToDP, match.frToDP)
		copy(alternative.dpToFR, match.dpToFR)
		clear(visited)
		alternative.frToDP[fr] = -1
		alternative.dpToFR[dp] = -1
		if alternative.augment(graph, fr, visited, fr, dp) {
			return alternative, true
		}
	}
	return matching{}, false
}

func pairingOf(graph controlGraph, frToDP []int) map[model.ID]model.ID {
	pairing := make(map[model.ID]model.ID, len(frToDP))
	for i, dp := range frToDP {
		pairing[graph.frs[i]] = graph.dps[dp]
	}
	return pairing
}

// squareAnalysis is the classification of a square mapping with a unique pairing.
type squareAnalysis struct {
	classification Classification
	order          []model.ID
	cycles         [][]model.ID
	findings       []result.Finding
}

// squareFacts are the observations that decide a square classification.
type squareFacts struct {
	conflict    bool
	open        bool
	ineffective bool
	cyclic      bool
	material    bool // at least one known material off-diagonal effect
}

func (f squareFacts) classification() Classification {
	switch {
	case f.conflict || f.open || f.ineffective:
		return Unresolved
	case f.cyclic:
		return Coupled
	case f.material:
		return Decoupled
	default:
		return Uncoupled
	}
}

func analyzeSquare(matrix Matrix, graph controlGraph, match matching) squareAnalysis {
	precedence := precedenceEdges(matrix, graph, match)
	cyclic := cyclicComponents(precedence)
	ineffective := ineffectiveFindings(matrix, graph, match)
	facts := squareFacts{
		conflict:    anyCell(matrix, func(c Cell) bool { return c.Conflict }),
		open:        anyCell(matrix, isOpen),
		ineffective: len(ineffective) > 0,
		cyclic:      len(cyclic) > 0,
		material:    hasEdges(precedence),
	}
	analysis := squareAnalysis{
		classification: facts.classification(),
		cycles:         idGroups(graph.dps, cyclic),
		findings:       ineffective,
	}
	if facts.open {
		analysis.findings = append(analysis.findings, newFinding(result.CodeUnresolvedInfluence, matrix.Scope,
			fmt.Sprintf("Scope %s has influence cells that are not fully known, so its classification is unresolved.", matrix.Scope)))
	}
	if analysis.classification == Uncoupled || analysis.classification == Decoupled {
		analysis.order = idsAt(graph.dps, topologicalOrder(precedence))
	}
	return analysis
}

// precedenceEdges returns, for each DP index, the DP indices that must follow it.
// DP a precedes DP b when a materially affects the FR paired with b. Conflicting
// cells are not known effects, so they add no edge.
func precedenceEdges(matrix Matrix, graph controlGraph, match matching) [][]int {
	edges := make([][]int, len(graph.dps))
	for b := range graph.dps {
		fr := graph.frs[match.dpToFR[b]]
		for a, source := range graph.dps {
			if a != b && isKnownMaterial(matrix.Cell(fr, source)) {
				edges[a] = append(edges[a], b)
			}
		}
	}
	return edges
}

func ineffectiveFindings(matrix Matrix, graph controlGraph, match matching) []result.Finding {
	var findings []result.Finding
	for j, dp := range graph.dps {
		fr := graph.frs[match.dpToFR[j]]
		if isKnownNone(matrix.Cell(fr, dp)) {
			findings = append(findings, newFinding(result.CodeIneffectiveControl, matrix.Scope,
				fmt.Sprintf("Design parameter %s has no material effect on its paired requirement %s in scope %s.", dp, fr, matrix.Scope)))
		}
	}
	return findings
}

func anyCell(matrix Matrix, predicate func(Cell) bool) bool {
	for _, fr := range matrix.FRs {
		for _, dp := range matrix.DPs {
			if predicate(matrix.Cell(fr, dp)) {
				return true
			}
		}
	}
	return false
}

// isOpen reports whether a cell leaves its influence unresolved. An unknown effect,
// an exact unknown record, or an unresolved premise each keep the cell open. An open
// cell still keeps its known effect for precedence edges and cycles.
func isOpen(cell Cell) bool {
	effectUnknown := cell.Effect != EffectMaterial && cell.Effect != EffectNone
	return effectUnknown || len(cell.Unknowns) > 0 || cell.Unresolved
}

func isKnownMaterial(cell Cell) bool {
	return cell.Effect == EffectMaterial && !cell.Conflict
}

func isKnownNone(cell Cell) bool {
	return cell.Effect == EffectNone && !cell.Conflict
}

func hasEdges(edges [][]int) bool {
	for _, targets := range edges {
		if len(targets) > 0 {
			return true
		}
	}
	return false
}

// cyclicComponents returns the strongly connected components that hold a cycle.
// Each component lists its vertices in ascending order. Components are sorted by
// their first vertex.
func cyclicComponents(edges [][]int) [][]int {
	var cyclic [][]int
	for _, component := range stronglyConnectedComponents(edges) {
		if len(component) > 1 {
			slices.Sort(component)
			cyclic = append(cyclic, component)
		}
	}
	slices.SortFunc(cyclic, func(a, b []int) int { return a[0] - b[0] })
	return cyclic
}

// topologicalOrder returns vertex indices so that every edge runs forward. Ties go
// to the lowest index. It returns nil when the edges contain a cycle.
func topologicalOrder(edges [][]int) []int {
	inDegree := make([]int, len(edges))
	for _, targets := range edges {
		for _, w := range targets {
			inDegree[w]++
		}
	}
	placed := make([]bool, len(edges))
	order := make([]int, 0, len(edges))
	for len(order) < len(edges) {
		next := firstReady(inDegree, placed)
		if next == -1 {
			return nil
		}
		placed[next] = true
		order = append(order, next)
		for _, w := range edges[next] {
			inDegree[w]--
		}
	}
	return order
}

func firstReady(inDegree []int, placed []bool) int {
	for v := range inDegree {
		if !placed[v] && inDegree[v] == 0 {
			return v
		}
	}
	return -1
}

// tarjan holds the state of Tarjan's strongly connected components algorithm.
type tarjan struct {
	edges      [][]int
	index      []int
	low        []int
	onStack    []bool
	stack      []int
	nextIndex  int
	components [][]int
}

func stronglyConnectedComponents(edges [][]int) [][]int {
	t := tarjan{
		edges:   edges,
		index:   unsetIndices(len(edges)),
		low:     make([]int, len(edges)),
		onStack: make([]bool, len(edges)),
	}
	for v := range edges {
		if t.index[v] == -1 {
			t.visit(v)
		}
	}
	return t.components
}

func (t *tarjan) visit(v int) {
	t.index[v] = t.nextIndex
	t.low[v] = t.nextIndex
	t.nextIndex++
	t.stack = append(t.stack, v)
	t.onStack[v] = true
	for _, w := range t.edges[v] {
		switch {
		case t.index[w] == -1:
			t.visit(w)
			t.low[v] = min(t.low[v], t.low[w])
		case t.onStack[w]:
			t.low[v] = min(t.low[v], t.index[w])
		}
	}
	if t.low[v] == t.index[v] {
		t.popComponent(v)
	}
}

func (t *tarjan) popComponent(root int) {
	var component []int
	for {
		last := len(t.stack) - 1
		w := t.stack[last]
		t.stack = t.stack[:last]
		t.onStack[w] = false
		component = append(component, w)
		if w == root {
			break
		}
	}
	t.components = append(t.components, component)
}

func idsAt(ids []model.ID, indices []int) []model.ID {
	out := make([]model.ID, len(indices))
	for i, index := range indices {
		out[i] = ids[index]
	}
	return out
}

func idGroups(ids []model.ID, groups [][]int) [][]model.ID {
	out := make([][]model.ID, len(groups))
	for i, group := range groups {
		out[i] = idsAt(ids, group)
	}
	return out
}

// newFinding leaves Severity empty. Composition applies the severity for the profile.
func newFinding(code string, scope model.ID, message string) result.Finding {
	return result.Finding{Code: code, Subject: string(scope), Message: message}
}
