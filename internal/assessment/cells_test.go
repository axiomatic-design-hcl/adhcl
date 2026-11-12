package assessment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// cellsCore mirrors the design, FRs, DPs, and material diagonal of testdata/spec/conformance/core-1/base.ad.hcl.
const cellsCore = `design "two_outputs" {
  boundary = "Two independent outputs."
}

need "separate_outputs" {
  statement = "Each output is produced independently."
}

fr "left" {
  statement = "Produce the left output."
  need = need.separate_outputs
}

fr "right" {
  statement = "Produce the right output."
  need = need.separate_outputs
}

dp "left_output" {
  controls = [fr.left]
  responsibility = "Choose the left output mechanism."
  decision = "selected"
}

dp "right_output" {
  controls = [fr.right]
  responsibility = "Choose the right output mechanism."
  decision = "selected"
}

influence {
  from = [dp.left_output]
  to = fr.left
  effect = "material"
  mechanism = "The left mechanism drives the left output."
}

influence {
  from = [dp.right_output]
  to = fr.right
  effect = "material"
  mechanism = "The right mechanism drives the right output."
}

`

// cellsDefaultNone mirrors the unlisted-none cells default of the base fixture.
const cellsDefaultNone = `cells "separate_inputs" {
  scope = design.two_outputs
  unlisted = "none"
  because = "Outputs share no state."
}

`

// cellsContexts declares two operating contexts for the context-specific tests.
const cellsContexts = `context "alpha" {
  description = "Alpha operating condition."
}

context "beta" {
  description = "Beta operating condition."
}

`

// cellsTwoOutputs is the base design with its default: the input of most tests.
const cellsTwoOutputs = cellsCore + cellsDefaultNone

// cellsBuild parses, decodes, resolves, and selects the inline sources with zero findings from setup.
func cellsBuild(t *testing.T, sources string, request selection.Request) (*model.Graph, *selection.Snapshot) {
	t.Helper()
	unit := syntax.Parse(model.Source{Path: "cells.ad.hcl", Bytes: []byte(sources)})
	cellsRequireNoSetupFindings(t, "parse", unit.Findings)
	core, findings := schema.DecodeCore([]syntax.Unit{unit})
	cellsRequireNoSetupFindings(t, "decode", findings)
	graph, findings := schema.Resolve(core)
	cellsRequireNoSetupFindings(t, "resolve", findings)
	snapshot, findings := selection.Select(graph, request)
	cellsRequireNoSetupFindings(t, "select", findings)
	return graph, snapshot
}

// cellsRequireNoSetupFindings fails the test when setup reports any finding.
func cellsRequireNoSetupFindings(t *testing.T, stage string, findings []result.Finding) {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("%s findings = %d, want 0; first = %s %q", stage, len(findings), findings[0].Code, findings[0].Subject)
	}
}

// cellsDesignSet returns the sibling set of the design root, which owns the two-output axes.
func cellsDesignSet(t *testing.T, graph *model.Graph) model.SiblingSet {
	t.Helper()
	set, ok := graph.SiblingSet(model.ID("design.two_outputs"))
	if !ok {
		t.Fatalf("graph has no sibling set design.two_outputs")
	}
	return set
}

// cellsLookup returns the resolved cell at fr and dp.
func cellsLookup(t *testing.T, matrix Matrix, fr, dp model.ID) Cell {
	t.Helper()
	if !classifyTestSet(matrix.FRs)[fr] || !classifyTestSet(matrix.DPs)[dp] {
		t.Fatalf("matrix has no coordinate %q x %q", fr, dp)
	}
	return matrix.Cell(fr, dp)
}

// cellsRequireFinding fails unless findings contain the code with the subject.
func cellsRequireFinding(t *testing.T, findings []result.Finding, code, subject string) {
	t.Helper()
	for _, f := range findings {
		if f.Code == code && f.Subject == subject {
			return
		}
	}
	t.Fatalf("findings lack code %q with subject %q; got %d findings", code, subject, len(findings))
}

// cellsRequireNoFinding fails when any finding carries the code.
func cellsRequireNoFinding(t *testing.T, findings []result.Finding, code string) {
	t.Helper()
	for _, f := range findings {
		if f.Code == code {
			t.Fatalf("unexpected finding code %q with subject %q", f.Code, f.Subject)
		}
	}
}

// cellsWith appends extra source blocks to the given base design.
func cellsWith(base string, blocks string) string {
	return base + blocks + "\n"
}

func TestCellsExactUnknownKeepsCellUnknownBesideDefaultNone(t *testing.T) {
	// Given the base design with a cells default of none, plus an exact unknown from dp.right_output to fr.left.
	sources := cellsWith(cellsTwoOutputs, `influence {
  from = [dp.right_output]
  to = fr.left
  effect = "unknown"
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then fr.left x dp.right_output stays unknown, the default does not apply, and no conflict is raised.
	cell := cellsLookup(t, matrix, model.ID("fr.left"), model.ID("dp.right_output"))
	if cell.Effect != Effect("unknown") {
		t.Fatalf("effect = %q, want unknown", cell.Effect)
	}
	if cell.FromDefault {
		t.Fatalf("FromDefault = true, want false")
	}
	if len(cell.Unknowns) != 1 {
		t.Fatalf("unknowns = %d, want 1", len(cell.Unknowns))
	}
	if cell.Conflict {
		t.Fatalf("Conflict = true, want false")
	}

	// And the other three cells match the expected matrix in the conformance fixture.
	left := cellsLookup(t, matrix, model.ID("fr.left"), model.ID("dp.left_output"))
	if left.Effect != Effect("material") || left.FromDefault {
		t.Fatalf("fr.left x dp.left_output = %q fromDefault %v, want material without default", left.Effect, left.FromDefault)
	}
	other := cellsLookup(t, matrix, model.ID("fr.right"), model.ID("dp.left_output"))
	if other.Effect != Effect("none") || !other.FromDefault {
		t.Fatalf("fr.right x dp.left_output = %q fromDefault %v, want none from default", other.Effect, other.FromDefault)
	}
	right := cellsLookup(t, matrix, model.ID("fr.right"), model.ID("dp.right_output"))
	if right.Effect != Effect("material") || right.FromDefault {
		t.Fatalf("fr.right x dp.right_output = %q fromDefault %v, want material without default", right.Effect, right.FromDefault)
	}
	cellsRequireNoFinding(t, findings, "conflicting_influence")
}

func TestCellsCompatibleMaterialExactRecordsMergeAndKeepBothSources(t *testing.T) {
	// Given two material exact records from dp.left_output to fr.left with different mechanisms.
	sources := cellsWith(cellsTwoOutputs, `influence {
  from = [dp.left_output]
  to = fr.left
  effect = "material"
  mechanism = "A second mechanism also drives the left output."
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then fr.left x dp.left_output is material with both exact records kept and no conflict.
	cell := cellsLookup(t, matrix, model.ID("fr.left"), model.ID("dp.left_output"))
	if cell.Effect != Effect("material") {
		t.Fatalf("effect = %q, want material", cell.Effect)
	}
	if cell.Conflict {
		t.Fatalf("Conflict = true, want false")
	}
	if len(cell.Exact) != 2 {
		t.Fatalf("exact records = %d, want 2", len(cell.Exact))
	}
	cellsRequireNoFinding(t, findings, "conflicting_influence")
}

func TestCellsMaterialAndNoneOnOneCellConflict(t *testing.T) {
	// Given a material exact record and a none exact record from dp.left_output to fr.left.
	sources := cellsWith(cellsTwoOutputs, `influence {
  from = [dp.left_output]
  to = fr.left
  effect = "none"
  because = "The left mechanism leaves the left output unchanged."
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then fr.left x dp.left_output is flagged as a conflict with a conflicting_influence finding.
	cell := cellsLookup(t, matrix, model.ID("fr.left"), model.ID("dp.left_output"))
	if !cell.Conflict {
		t.Fatalf("Conflict = false, want true")
	}
	cellsRequireFinding(t, findings, "conflicting_influence", "fr.left")
}

func TestCellsNoRecordAndNoDefaultLeavesCellUnknown(t *testing.T) {
	// Given the base design without a cells default, so fr.left x dp.right_output has no record and no default.
	sources := cellsCore
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, _ := ResolveCells(snapshot, set)

	// Then the unlisted cell is unknown and no default applies.
	cell := cellsLookup(t, matrix, model.ID("fr.left"), model.ID("dp.right_output"))
	if cell.Effect != Effect("unknown") {
		t.Fatalf("effect = %q, want unknown", cell.Effect)
	}
	if cell.FromDefault {
		t.Fatalf("FromDefault = true, want false")
	}
}

func TestCellsDefaultNoneFillsUnlistedCells(t *testing.T) {
	// Given the base design with a cells default of none and no exact record for fr.right x dp.left_output.
	graph, snapshot := cellsBuild(t, cellsTwoOutputs, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then the unlisted cell is none, marked as coming from the default, with no exact records and no conflict.
	cell := cellsLookup(t, matrix, model.ID("fr.right"), model.ID("dp.left_output"))
	if cell.Effect != Effect("none") {
		t.Fatalf("effect = %q, want none", cell.Effect)
	}
	if !cell.FromDefault {
		t.Fatalf("FromDefault = false, want true")
	}
	if cell.Conflict {
		t.Fatalf("Conflict = true, want false")
	}
	if len(cell.Exact) != 0 {
		t.Fatalf("exact records = %d, want 0", len(cell.Exact))
	}
	cellsRequireNoFinding(t, findings, "conflicting_influence")
}

func TestCellsDisagreeingApplicableDefaultsConflict(t *testing.T) {
	// Given two applicable cells defaults for the same scope that disagree: one none and one unknown.
	sources := cellsWith(cellsTwoOutputs, `cells "second_default" {
  scope = design.two_outputs
  unlisted = "unknown"
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then a cell that depends on the defaults is flagged as a conflict with a conflicting_influence finding.
	cell := cellsLookup(t, matrix, model.ID("fr.right"), model.ID("dp.left_output"))
	if !cell.Conflict {
		t.Fatalf("Conflict = false, want true")
	}
	cellsRequireFinding(t, findings, "conflicting_influence", "design.two_outputs")
}

func TestCellsContextSpecificInfluenceIgnoredInAnotherContext(t *testing.T) {
	// Given a material influence under context.alpha, a cells default of none, and context.beta selected.
	sources := cellsWith(cellsContexts+cellsTwoOutputs, `influence {
  from = [dp.left_output]
  to = fr.right
  effect = "material"
  mechanism = "The left mechanism drives the right output under alpha."
  under = context.alpha
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{Context: model.ID("context.beta")})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then the alpha influence does not count, so the cell falls back to the default none.
	cell := cellsLookup(t, matrix, model.ID("fr.right"), model.ID("dp.left_output"))
	if cell.Effect != Effect("none") {
		t.Fatalf("effect = %q, want none", cell.Effect)
	}
	if !cell.FromDefault {
		t.Fatalf("FromDefault = false, want true")
	}
	if len(cell.Exact) != 0 {
		t.Fatalf("exact records = %d, want 0", len(cell.Exact))
	}
	cellsRequireNoFinding(t, findings, "unresolved_applicability")
}

func TestCellsContextSpecificInfluenceUnresolvedWithoutContext(t *testing.T) {
	// Given a material influence under context.alpha, no cells default, and no context selected.
	sources := cellsWith(cellsContexts+cellsCore, `influence {
  from = [dp.left_output]
  to = fr.right
  effect = "material"
  mechanism = "The left mechanism drives the right output under alpha."
  under = context.alpha
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then the cell is unresolved, unknown, and an applicability finding names the design scope.
	cell := cellsLookup(t, matrix, model.ID("fr.right"), model.ID("dp.left_output"))
	if !cell.Unresolved {
		t.Fatalf("Unresolved = false, want true")
	}
	if cell.Effect != Effect("unknown") {
		t.Fatalf("effect = %q, want unknown", cell.Effect)
	}
	cellsRequireFinding(t, findings, "unresolved_applicability", "design.two_outputs")
}

func TestCellsInvalidatedPremiseDisablesEffect(t *testing.T) {
	// Given a material influence whose premise assumption.gate is invalidated.
	sources := cellsWith(cellsTwoOutputs, `assumption "gate" {
  statement = "The gate holds."
  status = "invalidated"
}

influence {
  from = [dp.left_output]
  to = fr.right
  effect = "material"
  mechanism = "The left mechanism drives the right output while the gate holds."
  holds_if = [assumption.gate]
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then the material effect is disabled and the cell keeps an unresolved obligation.
	cell := cellsLookup(t, matrix, model.ID("fr.right"), model.ID("dp.left_output"))
	if cell.Effect == Effect("material") {
		t.Fatalf("effect = %q, want the disabled effect not to be material", cell.Effect)
	}
	if !cell.Unresolved {
		t.Fatalf("Unresolved = false, want true")
	}
	cellsRequireFinding(t, findings, "unresolved_applicability", "design.two_outputs")
}

func TestCellsAssumedPremiseMakesCellConditional(t *testing.T) {
	// Given a material influence whose premise assumption.gate is assumed.
	sources := cellsWith(cellsTwoOutputs, `assumption "gate" {
  statement = "The gate holds."
  status = "assumed"
}

influence {
  from = [dp.left_output]
  to = fr.right
  effect = "material"
  mechanism = "The left mechanism drives the right output while the gate holds."
  holds_if = [assumption.gate]
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then the material cell is conditional and no applicability finding is raised.
	cell := cellsLookup(t, matrix, model.ID("fr.right"), model.ID("dp.left_output"))
	if cell.Effect != Effect("material") {
		t.Fatalf("effect = %q, want material", cell.Effect)
	}
	if !cell.Conditional {
		t.Fatalf("Conditional = false, want true")
	}
	cellsRequireNoFinding(t, findings, "unresolved_applicability")
}

func TestCellsCrossBranchInfluenceStaysOutOfSiblingMatrix(t *testing.T) {
	// Given a child FR under dp.left_output with a child DP that controls it, and a material influence from that child DP to the root fr.right.
	sources := cellsWith(cellsTwoOutputs, `fr "left_leaf" {
  statement = "Produce one left leaf output."
  parent = dp.left_output
}

dp "left_leaf_mechanism" {
  controls = [fr.left_leaf]
  responsibility = "Choose the left leaf mechanism."
  decision = "selected"
}

influence {
  from = [dp.left_leaf_mechanism]
  to = fr.right
  effect = "material"
  mechanism = "The child mechanism reaches the right output."
}
`)
	graph, snapshot := cellsBuild(t, sources, selection.Request{})
	set := cellsDesignSet(t, graph)

	// When the cells of the design scope are resolved.
	matrix, findings := ResolveCells(snapshot, set)

	// Then the cross-branch record does not enter the design matrix: no child column and no exact record from the child DP.
	row, ok := matrix.Cells[model.ID("fr.right")]
	if !ok {
		t.Fatalf("matrix has no row fr.right")
	}
	if _, ok := row[model.ID("dp.left_leaf_mechanism")]; ok {
		t.Fatalf("matrix has a cross-branch column dp.left_leaf_mechanism")
	}
	for dp, cell := range row {
		for _, rec := range cell.Exact {
			if rec.Record != nil && rec.Record.From[0].Target == model.ID("dp.left_leaf_mechanism") {
				t.Fatalf("cell fr.right x %q carries the cross-branch record", dp)
			}
		}
	}
	cellsRequireNoFinding(t, findings, "conflicting_influence")
}

func TestCellsSparseStorageRetainsRectangularAxesAndScopedDefault(t *testing.T) {
	for _, count := range []int{8, 32, 128} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var source strings.Builder
			source.WriteString(hierarchyTestDesign)
			for i := range count {
				fmt.Fprintf(&source, `
fr "f%d" {
  statement = "Produce one output."
  need = [need.record_payments]
}
dp "d%d" {
  controls = [fr.f%d]
  responsibility = "Choose one mechanism."
  decision = "selected"
}
influence {
  from = [dp.d%d]
  to = fr.f%d
  effect = "material"
  mechanism = "The mechanism produces the output."
}
`, i, i, i, i, i)
			}
			source.WriteString(`
dp "extra" {
  controls = [fr.f0]
  responsibility = "Choose another mechanism."
  decision = "selected"
}
cells "defaults" {
  scope = design.checkout
  unlisted = "none"
  because = "Mechanisms have independent outputs."
}
`)
			snapshot := hierarchyTestSnapshot(t, source.String())
			set, ok := snapshot.Graph.SiblingSet("design.checkout")
			if !ok {
				t.Fatal("root sibling set is missing")
			}

			matrix, findings := ResolveCells(snapshot, set)

			hierarchyTestRequireNoFindings(t, findings)
			if len(matrix.FRs) != count || len(matrix.DPs) != count+1 {
				t.Fatalf("axes = %d x %d, want %d x %d", len(matrix.FRs), len(matrix.DPs), count, count+1)
			}
			if got := cellsStoredCount(matrix); got != count {
				t.Fatalf("stored cells = %d, want %d authored cells", got, count)
			}
			if len(matrix.defaults) != 1 {
				t.Fatalf("stored defaults = %d, want 1", len(matrix.defaults))
			}
			for _, fr := range matrix.FRs {
				cell := matrix.Cell(fr, "dp.extra")
				if cell.Effect != EffectNone || !cell.FromDefault {
					t.Fatalf("unlisted cell %s x dp.extra = %+v, want default none", fr, cell)
				}
			}
			if got := cellsStoredCount(matrix); got != count {
				t.Fatalf("default lookup expanded storage to %d cells", got)
			}
			if cell := matrix.Cell("fr.outside", "dp.extra"); cell.Effect != EffectUnknown || cell.FromDefault {
				t.Fatalf("default escaped matrix axes: %+v", cell)
			}
		})
	}
}

func cellsStoredCount(matrix Matrix) int {
	count := 0
	for _, row := range matrix.Cells {
		count += len(row)
	}
	return count
}
