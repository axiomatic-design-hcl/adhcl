package assessment

import (
	"fmt"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
)

// classifyTestMatrix builds a Matrix from a compact table. Row i holds the cells
// of frs[i] across dps. Each byte is one cell: X material, 0 none, ? unknown,
// and C a material effect that conflicts with a none record.
func classifyTestMatrix(t *testing.T, frs, dps []model.ID, table []string) Matrix {
	t.Helper()
	if len(table) != len(frs) {
		t.Fatalf("table has %d rows for %d FRs", len(table), len(frs))
	}
	cells := make(map[model.ID]map[model.ID]Cell, len(frs))
	for i, fr := range frs {
		if len(table[i]) != len(dps) {
			t.Fatalf("row %d has %d cells for %d DPs", i, len(table[i]), len(dps))
		}
		cells[fr] = make(map[model.ID]Cell, len(dps))
		for j, dp := range dps {
			cells[fr][dp] = classifyTestCell(t, fr, dp, table[i][j])
		}
	}
	return Matrix{Scope: "design.D", FRs: frs, DPs: dps, Cells: cells}
}

func classifyTestCell(t *testing.T, fr, dp model.ID, mark byte) Cell {
	t.Helper()
	cell := Cell{FR: fr, DP: dp}
	switch mark {
	case 'X':
		cell.Effect = "material"
	case '0':
		cell.Effect = "none"
	case '?':
		cell.Effect = "unknown"
	case 'C':
		cell.Effect = "material"
		cell.Conflict = true
	default:
		t.Fatalf("unknown cell mark %q", mark)
	}
	return cell
}

func classifyTestCodes(findings []result.Finding) []string {
	codes := make([]string, 0, len(findings))
	for _, f := range findings {
		codes = append(codes, f.Code)
	}
	return codes
}

func classifyTestHasCode(findings []result.Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func classifyTestRequireCode(t *testing.T, findings []result.Finding, code string) {
	t.Helper()
	if !classifyTestHasCode(findings, code) {
		t.Errorf("findings %v lack code %q", classifyTestCodes(findings), code)
	}
}

func classifyTestForbidCode(t *testing.T, findings []result.Finding, code string) {
	t.Helper()
	if classifyTestHasCode(findings, code) {
		t.Errorf("findings %v include code %q, want it absent", classifyTestCodes(findings), code)
	}
}

func classifyTestRequireNoErrors(t *testing.T, findings []result.Finding) {
	t.Helper()
	for _, f := range findings {
		if f.Severity == result.Error {
			t.Errorf("unexpected error finding %q", f.Code)
		}
	}
}

func classifyTestRequireClassification(t *testing.T, got ScopeResult, want string) {
	t.Helper()
	if string(got.Classification) != want {
		t.Errorf("classification = %q, want %q", got.Classification, want)
	}
}

// classifyTestRequireNoConcrete accepts any outcome except the three
// classifications that a complete, material mapping can earn.
func classifyTestRequireNoConcrete(t *testing.T, got ScopeResult) {
	t.Helper()
	for _, concrete := range []string{"uncoupled", "decoupled", "coupled"} {
		if string(got.Classification) == concrete {
			t.Errorf("classification = %q, want no concrete classification", got.Classification)
		}
	}
}

func classifyTestSamePairing(a, b map[model.ID]model.ID) bool {
	if len(a) != len(b) {
		return false
	}
	for fr, dp := range b {
		if a[fr] != dp {
			return false
		}
	}
	return true
}

func classifyTestRequirePairing(t *testing.T, got, want map[model.ID]model.ID) {
	t.Helper()
	if !classifyTestSamePairing(got, want) {
		t.Errorf("pairing = %v, want %v", got, want)
	}
}

func classifyTestRequirePairings(t *testing.T, got []map[model.ID]model.ID, want ...map[model.ID]model.ID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d pairings %v, want %d", len(got), got, len(want))
	}
	for _, expected := range want {
		found := false
		for _, candidate := range got {
			if classifyTestSamePairing(candidate, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pairings %v lack %v", got, expected)
		}
	}
}

func classifyTestSet(ids []model.ID) map[model.ID]bool {
	set := make(map[model.ID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

func classifyTestSameMembers(a, b []model.ID) bool {
	setA := classifyTestSet(a)
	setB := classifyTestSet(b)
	if len(setA) != len(setB) {
		return false
	}
	for id := range setA {
		if !setB[id] {
			return false
		}
	}
	return true
}

func classifyTestRequireCycles(t *testing.T, got [][]model.ID, members ...model.ID) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("cycles = %v, want exactly one cycle over %v", got, members)
	}
	if !classifyTestSameMembers(got[0], members) {
		t.Errorf("cycle = %v, want members %v", got[0], members)
	}
}

func classifyTestRequireAxes(t *testing.T, got Matrix, frs, dps []model.ID) {
	t.Helper()
	if !classifyTestSameOrder(got.FRs, frs) {
		t.Errorf("FR axis = %v, want %v", got.FRs, frs)
	}
	if !classifyTestSameOrder(got.DPs, dps) {
		t.Errorf("DP axis = %v, want %v", got.DPs, dps)
	}
}

func classifyTestSameOrder(a, b []model.ID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// classifyTestRequirePrecedence checks that order is a permutation of dps and
// that every edge in before runs forward. It does not require a particular order.
func classifyTestRequirePrecedence(t *testing.T, order, dps []model.ID, before [][2]model.ID) {
	t.Helper()
	if len(order) != len(dps) {
		t.Fatalf("order %v has %d DPs, want %d", order, len(order), len(dps))
	}
	position := make(map[model.ID]int, len(order))
	for i, dp := range order {
		if _, repeated := position[dp]; repeated {
			t.Fatalf("order %v repeats %q", order, dp)
		}
		position[dp] = i
	}
	for _, dp := range dps {
		if _, ok := position[dp]; !ok {
			t.Fatalf("order %v lacks %q", order, dp)
		}
	}
	for _, edge := range before {
		if position[edge[0]] >= position[edge[1]] {
			t.Errorf("order %v places %q at or after %q, want the reverse", order, edge[0], edge[1])
		}
	}
}

func TestClassifyDiagonalWithoutOffDiagonalEffectIsUncoupled(t *testing.T) {
	// Given two material diagonal cells and justified zeros elsewhere.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"X0", "0X"})
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}}

	// When the sibling set is classified.
	got, findings := Classify(matrix, controls)

	// Then the mapping is uncoupled, uses the diagonal pairing, and reports nothing.
	classifyTestRequireClassification(t, got, "uncoupled")
	classifyTestRequirePairing(t, got.Pairing, map[model.ID]model.ID{"fr.A": "dp.A", "fr.B": "dp.B"})
	if len(findings) != 0 {
		t.Errorf("findings = %v, want none", classifyTestCodes(findings))
	}
}

func TestClassifyLowerTriangleIsDecoupledInPrecedenceOrder(t *testing.T) {
	// Given a material effect from DP A to the FR paired with DP B.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"X0", "XX"})
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}}

	// When the sibling set is classified.
	got, findings := Classify(matrix, controls)

	// Then the mapping is decoupled, DP A precedes DP B, and no cycle exists.
	classifyTestRequireClassification(t, got, "decoupled")
	classifyTestRequirePairing(t, got.Pairing, map[model.ID]model.ID{"fr.A": "dp.A", "fr.B": "dp.B"})
	classifyTestRequirePrecedence(t, got.Order, dps, [][2]model.ID{{"dp.A", "dp.B"}})
	if len(got.Cycles) != 0 {
		t.Errorf("cycles = %v, want none", got.Cycles)
	}
	classifyTestRequireNoErrors(t, findings)
}

func TestClassifyPermutedTriangleKeepsLabelsAndFindsValidOrder(t *testing.T) {
	// Given a triangular mapping whose rows and columns are authored in reverse,
	// with the material effect from DP A to the FR paired with DP B.
	frs := []model.ID{"fr.B", "fr.A"}
	dps := []model.ID{"dp.B", "dp.A"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"XX", "0X"})
	controls := map[model.ID][]model.ID{"dp.B": {"fr.B"}, "dp.A": {"fr.A"}}

	// When the sibling set is classified.
	got, _ := Classify(matrix, controls)

	// Then the authored labels remain, the labelled cells keep their effects,
	// the pairing follows the labels, and the valid order places DP A first.
	classifyTestRequireClassification(t, got, "decoupled")
	classifyTestRequireAxes(t, got.Matrix, frs, dps)
	if effect := got.Matrix.Cells["fr.A"]["dp.B"].Effect; effect != "none" {
		t.Errorf("cell fr.A/dp.B effect = %q, want %q", effect, "none")
	}
	classifyTestRequirePairing(t, got.Pairing, map[model.ID]model.ID{"fr.A": "dp.A", "fr.B": "dp.B"})
	classifyTestRequirePrecedence(t, got.Order, dps, [][2]model.ID{{"dp.A", "dp.B"}})
}

func TestClassifyAcceptsAnotherValidAuthoredOrder(t *testing.T) {
	// Given three paired DPs whose only material off-diagonal effect runs from
	// DP A to DP B, and an authored DP order that already respects that edge.
	frs := []model.ID{"fr.C", "fr.A", "fr.B"}
	dps := []model.ID{"dp.C", "dp.A", "dp.B"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"X00", "0X0", "0XX"})
	controls := map[model.ID][]model.ID{
		"dp.C": {"fr.C"},
		"dp.A": {"fr.A"},
		"dp.B": {"fr.B"},
	}

	// When the sibling set is classified.
	got, _ := Classify(matrix, controls)

	// Then the mapping is decoupled, and the returned order satisfies the
	// precedence edge without matching any particular display order.
	classifyTestRequireClassification(t, got, "decoupled")
	classifyTestRequirePrecedence(t, got.Order, dps, [][2]model.ID{{"dp.A", "dp.B"}})
	if len(got.Cycles) != 0 {
		t.Errorf("cycles = %v, want none", got.Cycles)
	}
}

func TestClassifyMaterialCycleWithKnownCellsIsCoupled(t *testing.T) {
	tests := []struct {
		name     string
		frs      []model.ID
		dps      []model.ID
		table    []string
		controls map[model.ID][]model.ID
		cycle    []model.ID
	}{
		{
			name:     "two DPs",
			frs:      []model.ID{"fr.A", "fr.B"},
			dps:      []model.ID{"dp.A", "dp.B"},
			table:    []string{"XX", "XX"},
			controls: map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}},
			cycle:    []model.ID{"dp.A", "dp.B"},
		},
		{
			name:     "three DPs",
			frs:      []model.ID{"fr.A", "fr.B", "fr.C"},
			dps:      []model.ID{"dp.A", "dp.B", "dp.C"},
			table:    []string{"X0X", "XX0", "0XX"},
			controls: map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}, "dp.C": {"fr.C"}},
			cycle:    []model.ID{"dp.A", "dp.B", "dp.C"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given a material cycle among the paired DPs and no unknown cell.
			matrix := classifyTestMatrix(t, tc.frs, tc.dps, tc.table)

			// When the sibling set is classified.
			got, findings := Classify(matrix, tc.controls)

			// Then the mapping is coupled, reports its cycle, and is not
			// treated as malformed input.
			classifyTestRequireClassification(t, got, "coupled")
			classifyTestRequireCycles(t, got.Cycles, tc.cycle...)
			classifyTestRequireNoErrors(t, findings)
		})
	}
}

func TestClassifyCycleBesideUnknownCellIsUnresolvedAndReportsCycle(t *testing.T) {
	// Given a material cycle between DP A and DP B, and an unknown effect from
	// DP A on the FR paired with DP C.
	frs := []model.ID{"fr.A", "fr.B", "fr.C"}
	dps := []model.ID{"dp.A", "dp.B", "dp.C"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"XX0", "XX0", "?0X"})
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}, "dp.C": {"fr.C"}}

	// When the sibling set is classified.
	got, findings := Classify(matrix, controls)

	// Then the classification is unresolved, the known cycle is still reported,
	// and the unknown influence is named.
	classifyTestRequireClassification(t, got, "unresolved")
	classifyTestRequireCycles(t, got.Cycles, "dp.A", "dp.B")
	classifyTestRequireCode(t, findings, "unresolved_influence")
}

func TestClassifyUnknownDiagonalIsUnresolved(t *testing.T) {
	// Given an unknown diagonal cell for the first pair and a material diagonal
	// for the second pair.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"?0", "0X"})
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}}

	// When the sibling set is classified.
	got, findings := Classify(matrix, controls)

	// Then the classification is unresolved, the unknown influence is named, and
	// the unknown is not reported as a known zero.
	classifyTestRequireClassification(t, got, "unresolved")
	classifyTestRequirePairing(t, got.Pairing, map[model.ID]model.ID{"fr.A": "dp.A", "fr.B": "dp.B"})
	classifyTestRequireCode(t, findings, "unresolved_influence")
	classifyTestForbidCode(t, findings, "ineffective_control")
}

func TestClassifyKnownZeroDiagonalIsIneffectiveControl(t *testing.T) {
	// Given a known zero on the diagonal of the second pair.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"X0", "X0"})
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}}

	// When the sibling set is classified.
	got, findings := Classify(matrix, controls)

	// Then the zero diagonal is an ineffective control and the mapping is not
	// classified as uncoupled or decoupled.
	classifyTestRequireCode(t, findings, "ineffective_control")
	classifyTestRequireNoConcrete(t, got)
}

func TestClassifyTwoCompletePairingsAreAmbiguous(t *testing.T) {
	// Given two DPs that each control both FRs, with all four effects material.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"XX", "XX"})
	controls := map[model.ID][]model.ID{
		"dp.A": {"fr.A", "fr.B"},
		"dp.B": {"fr.A", "fr.B"},
	}

	// When the sibling set is classified.
	got, findings := Classify(matrix, controls)

	// Then both complete pairings are reported, no pairing is chosen, no order
	// is given, and the classification is unresolved.
	classifyTestRequirePairings(t, got.PairingWitnesses,
		map[model.ID]model.ID{"fr.A": "dp.A", "fr.B": "dp.B"},
		map[model.ID]model.ID{"fr.A": "dp.B", "fr.B": "dp.A"},
	)
	classifyTestRequirePairing(t, got.Pairing, nil)
	if len(got.Order) != 0 {
		t.Errorf("order = %v, want none", got.Order)
	}
	classifyTestRequireClassification(t, got, "unresolved")
	classifyTestRequireCode(t, findings, "ambiguous_pairing")
}

func TestClassifyFRWithoutDistinctControlIsUncontrolled(t *testing.T) {
	// Given one DP that is the only control for two FRs.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"X", "X"})
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A", "fr.B"}}

	// When the sibling set is classified.
	_, findings := Classify(matrix, controls)

	// Then one FR lacks a distinct control, so the requirement is uncontrolled.
	classifyTestRequireCode(t, findings, "uncontrolled_requirement")
}

func TestClassifyDPOutsideCompletePairingIsRedundant(t *testing.T) {
	// Given two FRs with distinct controls and a third DP that controls nothing.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B", "dp.C"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"X00", "0X0"})
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}}

	// When the sibling set is classified.
	got, findings := Classify(matrix, controls)

	// Then the spare DP is redundant, every FR keeps its distinct control, and the
	// unique complete pairing is used.
	classifyTestRequireCode(t, findings, "redundant_control")
	classifyTestForbidCode(t, findings, "uncontrolled_requirement")
	classifyTestRequirePairing(t, got.Pairing, map[model.ID]model.ID{"fr.A": "dp.A", "fr.B": "dp.B"})
}

func TestClassifyRectangularMappingKeepsAllAxes(t *testing.T) {
	tests := []struct {
		name        string
		frs         []model.ID
		dps         []model.ID
		table       []string
		controls    map[model.ID][]model.ID
		wantPairing map[model.ID]model.ID
		wantCode    string
		forbidCode  string
	}{
		{
			name:     "more FRs than DPs",
			frs:      []model.ID{"fr.A", "fr.B", "fr.C"},
			dps:      []model.ID{"dp.A", "dp.B"},
			table:    []string{"X0", "0X", "00"},
			controls: map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}},
			wantCode: "uncontrolled_requirement",
			// Every DP is matched, so none is redundant.
			forbidCode: "redundant_control",
		},
		{
			name:        "more DPs than FRs",
			frs:         []model.ID{"fr.A", "fr.B"},
			dps:         []model.ID{"dp.A", "dp.B", "dp.C"},
			table:       []string{"X00", "0X0"},
			controls:    map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}},
			wantPairing: map[model.ID]model.ID{"fr.A": "dp.A", "fr.B": "dp.B"},
			wantCode:    "redundant_control",
			// Every FR has a distinct control, so none is uncontrolled.
			forbidCode: "uncontrolled_requirement",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given a rectangular sibling set with the stated controls.
			matrix := classifyTestMatrix(t, tc.frs, tc.dps, tc.table)

			// When the sibling set is classified.
			got, findings := Classify(matrix, tc.controls)

			// Then all authored axes remain, the coverage finding appears, the
			// extra axis is not removed, and no square classification is claimed.
			classifyTestRequireAxes(t, got.Matrix, tc.frs, tc.dps)
			classifyTestRequirePairing(t, got.Pairing, tc.wantPairing)
			classifyTestRequireCode(t, findings, tc.wantCode)
			classifyTestForbidCode(t, findings, tc.forbidCode)
			classifyTestRequireNoConcrete(t, got)
		})
	}
}

func TestClassifyConflictCellCannotSatisfyClassification(t *testing.T) {
	tests := []struct {
		name  string
		table []string
	}{
		{name: "conflict on a diagonal cell", table: []string{"C0", "0X"}},
		// Without the conflict, this off-diagonal material effect would make the
		// mapping decoupled.
		{name: "conflict on an off-diagonal cell", table: []string{"X0", "CX"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given a conflicting cell in an otherwise complete diagonal mapping.
			frs := []model.ID{"fr.A", "fr.B"}
			dps := []model.ID{"dp.A", "dp.B"}
			matrix := classifyTestMatrix(t, frs, dps, tc.table)
			controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}}

			// When the sibling set is classified.
			got, _ := Classify(matrix, controls)

			// Then the conflict prevents any uncoupled, decoupled, or coupled result.
			classifyTestRequireNoConcrete(t, got)
		})
	}
}

func TestClassifyNoneCellWithUnknownRecordIsUnresolved(t *testing.T) {
	// Given a diagonal 2x2 mapping whose off-diagonal cell has effect none but
	// still carries an unknown influence record.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"X0", "0X"})
	cell := matrix.Cells["fr.A"]["dp.B"]
	cell.Unknowns = []selection.AppliedInfluence{{}}
	matrix.Cells["fr.A"]["dp.B"] = cell
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}}

	// When the sibling set is classified.
	got, findings := Classify(matrix, controls)

	// Then the classification is unresolved, the unknown influence is named, and
	// the none effect remains on the cell.
	classifyTestRequireClassification(t, got, "unresolved")
	classifyTestRequireCode(t, findings, "unresolved_influence")
	if effect := got.Matrix.Cells["fr.A"]["dp.B"].Effect; effect != EffectNone {
		t.Fatalf("effect = %q, want %q", effect, EffectNone)
	}
}

func TestClassifyUnresolvedNoneCellIsUnresolved(t *testing.T) {
	// Given a diagonal 2x2 mapping whose off-diagonal cell has effect none and is
	// marked unresolved.
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B"}
	matrix := classifyTestMatrix(t, frs, dps, []string{"X0", "0X"})
	cell := matrix.Cells["fr.A"]["dp.B"]
	cell.Unresolved = true
	matrix.Cells["fr.A"]["dp.B"] = cell
	controls := map[model.ID][]model.ID{"dp.A": {"fr.A"}, "dp.B": {"fr.B"}}

	// When the sibling set is classified.
	got, _ := Classify(matrix, controls)

	// Then the classification is unresolved because the cell stays open.
	classifyTestRequireClassification(t, got, "unresolved")
}

func TestClassifyCompleteTenByTenRetainsOnlyTwoPairingWitnesses(t *testing.T) {
	frs, dps := make([]model.ID, 10), make([]model.ID, 10)
	for i := range frs {
		frs[i] = model.ID(fmt.Sprintf("fr.F%d", i))
		dps[i] = model.ID(fmt.Sprintf("dp.D%d", i))
	}
	controls := make(map[model.ID][]model.ID, len(dps))
	for _, dp := range dps {
		controls[dp] = frs
	}

	got, findings := Classify(Matrix{Scope: "design.D", FRs: frs, DPs: dps}, controls)

	classifyTestRequireAxes(t, got.Matrix, frs, dps)
	classifyTestRequireClassification(t, got, "unresolved")
	classifyTestRequireCode(t, findings, "ambiguous_pairing")
	classifyTestRequireWitnesses(t, got, controls)
}

func TestClassifyRectangularAmbiguityPreservesEveryAxis(t *testing.T) {
	frs := []model.ID{"fr.A", "fr.B"}
	dps := []model.ID{"dp.A", "dp.B", "dp.C"}
	controls := map[model.ID][]model.ID{
		"dp.A": {"fr.A", "fr.B"},
		"dp.B": {"fr.A", "fr.B"},
		"dp.C": {"fr.A", "fr.B"},
	}

	got, findings := Classify(Matrix{Scope: "design.D", FRs: frs, DPs: dps}, controls)

	classifyTestRequireAxes(t, got.Matrix, frs, dps)
	classifyTestRequireClassification(t, got, "unresolved")
	classifyTestRequireCode(t, findings, "ambiguous_pairing")
	classifyTestRequireWitnesses(t, got, controls)
}

func classifyTestRequireWitnesses(t *testing.T, got ScopeResult, controls map[model.ID][]model.ID) {
	t.Helper()
	if len(got.PairingWitnesses) != 2 {
		t.Fatalf("got %d pairing witnesses, want 2", len(got.PairingWitnesses))
	}
	if len(got.Pairing) != 0 || len(got.Order) != 0 {
		t.Fatalf("ambiguous result chooses a pairing or order: %v, %v", got.Pairing, got.Order)
	}
	if classifyTestSamePairing(got.PairingWitnesses[0], got.PairingWitnesses[1]) {
		t.Fatal("pairing witnesses are identical")
	}
	for _, pairing := range got.PairingWitnesses {
		if len(pairing) != len(got.Matrix.FRs) {
			t.Fatalf("witness %v does not cover all FRs", pairing)
		}
		used := make(map[model.ID]bool, len(pairing))
		for _, fr := range got.Matrix.FRs {
			dp := pairing[fr]
			if used[dp] || !classifyTestSet(controls[dp])[fr] {
				t.Fatalf("witness %v is not a distinct-control pairing", pairing)
			}
			used[dp] = true
		}
	}
}
