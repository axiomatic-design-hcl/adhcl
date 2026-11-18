package diagram

import (
	"fmt"
	"slices"
	"strings"

	"github.com/axiomatic-design-hcl/adhcl/internal/assessment"
	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

// Matrix presents an assessed scope in authored axis order.
type Matrix struct {
	Anchor, Scope, Label, Classification, Meaning string
	ScopeRef                                      Ref
	Master                                        bool
	DPs                                           []Ref
	Rows                                          []MatrixRow
	Order                                         []Ref
	Expects                                       []ExpectRow
	Couplings                                     []Ref
	Reasons                                       []Reason
	Unknown, Total                                int
}

// MatrixRow holds one functional requirement and its assessed cells.
type MatrixRow struct {
	FR    Ref
	Cells []MatrixCell
}

// MatrixCell describes an assessed effect, including source explanations.
type MatrixCell struct {
	DP, Symbol, Explanation string
	Conflict, Diagonal      bool
}

// Reason is one authored explanation of a matrix cell.
type Reason struct {
	FR, DP                                 Ref
	Symbol, Text, Locator, Label, Location string
}

func matrixAnchor(scope model.ID) string { return "matrix." + string(scope) }

// orderedScopes returns the assessed scopes in hierarchy order, with the master last.
func (b *builder) orderedScopes() []*assessment.ScopeResult {
	var ordered []*assessment.ScopeResult
	seen := map[model.ID]bool{}
	var walk func(model.ID)
	walk = func(scope model.ID) {
		if seen[scope] {
			return
		}
		seen[scope] = true
		if result := b.scopes[scope]; result != nil {
			ordered = append(ordered, result)
		}
		set, ok := b.graph.SiblingSet(scope)
		if !ok {
			return
		}
		for _, fr := range set.FRs {
			for _, dp := range set.DPs {
				if b.controls(dp, fr) {
					walk(dp)
				}
			}
		}
	}
	walk(b.graph.Design)
	var master *assessment.ScopeResult
	for i := range b.res.Scopes {
		scope := &b.res.Scopes[i]
		if seen[scope.Matrix.Scope] {
			continue
		}
		if scope.Matrix.Scope == "master" {
			master = scope
			continue
		}
		ordered = append(ordered, scope)
	}
	if master != nil {
		ordered = append(ordered, master)
	}
	return ordered
}

func (b *builder) matrices() []Matrix {
	scopes := b.orderedScopes()
	matrices := make([]Matrix, 0, len(scopes))
	for _, scope := range scopes {
		id := scope.Matrix.Scope
		matrix := Matrix{Anchor: matrixAnchor(id), Scope: string(id), ScopeRef: b.ref(id), Classification: string(scope.Classification), Meaning: classificationMeaning[scope.Classification], Master: id == "master"}
		if scope.Classification == assessment.Decoupled {
			matrix.Order = b.refs(scope.Order)
		}
		switch {
		case matrix.Master:
			matrix.Label = "All leaf FRs and leaf DPs"
		case id == b.graph.Design:
			matrix.Label = "Level 1"
		default:
			parent := string(id)
			if alias := b.aliases[id]; alias != "" {
				parent = alias
			}
			matrix.Label = fmt.Sprintf("Level %d, under %s", b.depths[id], parent)
		}
		matrix.DPs = b.refs(scope.Matrix.DPs)
		for _, fr := range scope.Matrix.FRs {
			row := MatrixRow{FR: b.ref(fr)}
			for _, dp := range scope.Matrix.DPs {
				cell := scope.Matrix.Cell(fr, dp)
				item := MatrixCell{DP: string(dp), Symbol: "?", Conflict: cell.Conflict, Diagonal: scope.Pairing[fr] == dp}
				switch cell.Effect {
				case assessment.EffectMaterial:
					item.Symbol = "X"
				case assessment.EffectNone:
					item.Symbol = "0"
				default:
					matrix.Unknown++
				}
				matrix.Total++
				reasons := b.cellReasons(fr, dp, item.Symbol, cell)
				explanations := make([]string, 0, len(reasons))
				for _, reason := range reasons {
					explanations = append(explanations, fmt.Sprintf("%s (%s)", reason.Text, reason.Location))
				}
				item.Explanation = strings.Join(explanations, "; ")
				matrix.Reasons = append(matrix.Reasons, reasons...)
				row.Cells = append(row.Cells, item)
			}
			matrix.Rows = append(matrix.Rows, row)
		}
		for _, coupling := range b.core.Couplings {
			if coupling.Scope.Target == id {
				matrix.Couplings = append(matrix.Couplings, b.ref(coupling.ID))
			}
		}
		matrices = append(matrices, matrix)
	}
	return matrices
}

// cellReasons collects the authored explanations of one cell: the influence records
// that apply, or the default of the sibling set when the cell comes from it.
func (b *builder) cellReasons(fr, dp model.ID, symbol string, cell assessment.Cell) []Reason {
	var reasons []Reason
	for _, group := range [][]selection.AppliedInfluence{cell.Exact, cell.Unknowns} {
		for _, applied := range group {
			record := applied.Record
			if record == nil {
				continue
			}
			var parts []string
			if record.Mechanism != nil {
				parts = append(parts, *record.Mechanism)
			}
			if record.Because != nil {
				parts = append(parts, *record.Because)
			}
			if len(parts) == 0 {
				continue
			}
			reason := Reason{FR: b.ref(fr), DP: b.ref(dp), Symbol: symbol, Text: strings.Join(parts, " "), Locator: text(record.SourceLocator), Label: labelSource, Location: location(record.Range)}
			if record.MechanismSource != nil && *record.MechanismSource == model.MechanismInferred || record.MechanismSource == nil && record.Origin != nil && *record.Origin == model.OriginInferred {
				reason.Label = labelInferred
			}
			reasons = append(reasons, reason)
		}
	}
	if cell.FromDefault {
		for _, set := range b.graph.SiblingSets {
			if !slices.Contains(set.FRs, fr) || !slices.Contains(set.DPs, dp) {
				continue
			}
			for _, declared := range b.core.Cells {
				if declared.Scope.Target == set.Scope && declared.Because != nil {
					reasons = append(reasons, Reason{FR: b.ref(fr), DP: b.ref(dp), Symbol: symbol, Text: *declared.Because, Label: originLabel(declared.Origin), Location: location(declared.Range)})
				}
			}
		}
	}
	return reasons
}
