package assessment

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

const hierarchyTestDesign = `design "checkout" {
  boundary = "Record each accepted payment."
}

need "record_payments" {
  statement = "Record every accepted payment."
}
`

const hierarchyTestStoreTree = `fr "store" {
  need      = [need.record_payments]
  statement = "Store each accepted payment record."
}

dp "engine" {
  controls       = [fr.store]
  responsibility = "Choose the storage engine."
  decision       = "selected"
}

fr "cache_hit" {
  parent    = dp.engine
  statement = "Serve repeated reads from memory."
}

dp "cache" {
  controls       = [fr.cache_hit]
  responsibility = "Hold recent records in memory."
  decision       = "selected"
}
`

const hierarchyTestAllocationTree = `fr "store" {
  need      = [need.record_payments]
  statement = "Store each accepted payment record."

  criterion "latency" {
    statement = "Respond within the storage latency budget."
  }
}

dp "engine" {
  controls       = [fr.store]
  responsibility = "Choose the storage engine."
  decision       = "selected"
}

fr "cache_hit" {
  parent    = dp.engine
  statement = "Serve repeated reads from memory."

  criterion "latency" {
    statement      = "Serve cached reads within the cache latency budget."
    allocated_from = fr.store.criterion.latency
  }
}

dp "cache" {
  controls       = [fr.cache_hit]
  responsibility = "Hold recent records in memory."
  decision       = "selected"
}
`

const hierarchyTestReportBranch = `fr "report" {
  need      = [need.record_payments]
  statement = "Report the total of accepted payments."
}

dp "reporter" {
  controls       = [fr.report]
  responsibility = "Compute the report total."
  decision       = "selected"
}
`

const hierarchyTestAlternateReporter = `dp "alt_reporter" {
  controls       = [fr.report]
  responsibility = "Compute an alternate report total."
  decision       = "proposed"
}
`

const hierarchyTestSharedWriter = `fr "capture" {
  need      = [need.record_payments]
  statement = "Capture each accepted payment."
}

fr "admit" {
  need      = [need.record_payments]
  statement = "Admit each accepted payment."
}

dp "writer" {
  controls       = [fr.capture, fr.admit]
  responsibility = "Write each accepted payment."
  decision       = "selected"
}
`

const hierarchyTestCrossInfluence = `influence {
  from      = [dp.cache]
  to        = fr.report
  effect    = "material"
  mechanism = "Changing the cache changes the reported total."
}
`

const hierarchyTestParentInfluence = `influence {
  from      = [dp.cache]
  to        = fr.store
  effect    = "material"
  mechanism = "Cache writes change the stored record."
}
`

const hierarchyTestRootDefault = `cells "root_unlisted" {
  scope    = design.checkout
  unlisted = "none"
  because  = "The report does not change the store or its cache."
}
`

// hierarchyTestSnapshot builds the default selection from inline sources. It fails the test when any setup step reports a finding.
func hierarchyTestSnapshot(t *testing.T, sources ...string) *selection.Snapshot {
	t.Helper()

	units := make([]syntax.Unit, 0, len(sources))
	for index, source := range sources {
		unit := syntax.Parse(model.Source{Path: fmt.Sprintf("hierarchy-%d.ad.hcl", index+1), Bytes: []byte(source)})
		if len(unit.Findings) != 0 {
			t.Fatalf("source %d has %d native parse findings, want none", index+1, len(unit.Findings))
		}
		units = append(units, unit)
	}

	core, decodeFindings := schema.DecodeCore(units)
	if len(decodeFindings) != 0 {
		t.Fatalf("decode produced %d findings, want none", len(decodeFindings))
	}

	graph, resolveFindings := schema.Resolve(core)
	if len(resolveFindings) != 0 {
		t.Fatalf("resolve produced %d findings, want none", len(resolveFindings))
	}
	if graph == nil {
		t.Fatal("resolve returned no graph, want a graph")
	}

	snapshot, selectFindings := selection.Select(graph, selection.Request{})
	if len(selectFindings) != 0 {
		t.Fatalf("selection produced %d findings, want none", len(selectFindings))
	}
	if snapshot == nil {
		t.Fatal("selection returned no snapshot, want a snapshot")
	}
	return snapshot
}

func hierarchyTestIDs(ids []model.ID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	slices.Sort(out)
	return out
}

func hierarchyTestRequireIDs(t *testing.T, label string, got []string, want ...string) {
	t.Helper()

	gotSorted := slices.Clone(got)
	slices.Sort(gotSorted)
	wantSorted := slices.Clone(want)
	slices.Sort(wantSorted)
	if !slices.Equal(gotSorted, wantSorted) {
		t.Fatalf("%s = %v, want %v", label, gotSorted, wantSorted)
	}
}

func hierarchyTestSubjects(findings []result.Finding, code string) []string {
	var subjects []string
	for _, finding := range findings {
		if finding.Code == code {
			subjects = append(subjects, finding.Subject)
		}
	}
	return subjects
}

func hierarchyTestHasCode(findings []result.Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func hierarchyTestRequireNoFindings(t *testing.T, findings []result.Finding) {
	t.Helper()

	if len(findings) != 0 {
		t.Fatalf("findings = %d, want none", len(findings))
	}
}

func TestHierarchyMasterMatrixHoldsActiveLeavesAndCrossBranchEffect(t *testing.T) {
	// Given a parent DP with a leaf child branch and a separate leaf report branch, linked by a cross-branch material influence
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestStoreTree, hierarchyTestReportBranch, hierarchyTestCrossInfluence)

	// When the master matrix is built
	matrix, findings := MasterMatrix(snapshot)

	// Then the axes hold only the active leaves and the report row records the material cross-branch effect
	hierarchyTestRequireNoFindings(t, findings)
	hierarchyTestRequireIDs(t, "master rows", hierarchyTestIDs(matrix.FRs), "fr.cache_hit", "fr.report")
	hierarchyTestRequireIDs(t, "master columns", hierarchyTestIDs(matrix.DPs), "dp.cache", "dp.reporter")
	cell := matrix.Cell("fr.report", "dp.cache")
	if string(cell.Effect) != "material" {
		t.Errorf("cell (fr.report, dp.cache) effect = %q, want %q", cell.Effect, "material")
	}
	if len(cell.Exact) != 1 {
		t.Errorf("cell (fr.report, dp.cache) exact records = %d, want 1", len(cell.Exact))
	}
}

func TestHierarchyMasterMatrixLeavesUnrecordedLeafCellUnknown(t *testing.T) {
	// Given the same leaf branches where only the cache DP has a recorded effect on the report FR
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestStoreTree, hierarchyTestReportBranch, hierarchyTestCrossInfluence)

	// When the master matrix is built
	matrix, _ := MasterMatrix(snapshot)

	// Then the unrecorded cell between the report FR and the reporter DP is unknown
	cell := matrix.Cell("fr.report", "dp.reporter")
	if string(cell.Effect) != "unknown" {
		t.Errorf("cell (fr.report, dp.reporter) effect = %q, want %q", cell.Effect, "unknown")
	}
	if len(cell.Exact) != 0 {
		t.Errorf("cell (fr.report, dp.reporter) exact records = %d, want 0", len(cell.Exact))
	}
}

func TestHierarchyMasterMatrixKeepsSharedDPAsOneColumn(t *testing.T) {
	// Given two root FRs that are both controlled by one selected DP
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestSharedWriter)

	// When the master matrix is built
	matrix, _ := MasterMatrix(snapshot)

	// Then the shared DP occupies one column and both FRs are rows
	hierarchyTestRequireIDs(t, "master rows", hierarchyTestIDs(matrix.FRs), "fr.capture", "fr.admit")
	hierarchyTestRequireIDs(t, "master columns", hierarchyTestIDs(matrix.DPs), "dp.writer")
}

func TestHierarchyMasterMatrixExcludesInactiveDP(t *testing.T) {
	// Given a root FR controlled by a selected DP and by a proposed alternate DP
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestReportBranch, hierarchyTestAlternateReporter)

	// When the master matrix is built
	matrix, _ := MasterMatrix(snapshot)

	// Then only the selected DP appears as a column
	hierarchyTestRequireIDs(t, "master rows", hierarchyTestIDs(matrix.FRs), "fr.report")
	hierarchyTestRequireIDs(t, "master columns", hierarchyTestIDs(matrix.DPs), "dp.reporter")
}

func TestHierarchyMasterMatrixParentEffectDoesNotFillLeafCell(t *testing.T) {
	// Given a material influence from the cache DP onto the parent store FR
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestStoreTree, hierarchyTestReportBranch, hierarchyTestParentInfluence)

	// When the master matrix is built
	matrix, _ := MasterMatrix(snapshot)

	// Then the parent effect does not become a leaf fact for the child FR and the cell stays unknown
	hierarchyTestRequireIDs(t, "master rows", hierarchyTestIDs(matrix.FRs), "fr.cache_hit", "fr.report")
	cell := matrix.Cell("fr.cache_hit", "dp.cache")
	if string(cell.Effect) != "unknown" {
		t.Errorf("cell (fr.cache_hit, dp.cache) effect = %q, want %q", cell.Effect, "unknown")
	}
	if len(cell.Exact) != 0 {
		t.Errorf("cell (fr.cache_hit, dp.cache) exact records = %d, want 0", len(cell.Exact))
	}
}

func TestHierarchyMasterMatrixAppliesOnlyContainingSetDefault(t *testing.T) {
	// Given a root sibling set whose cells default is none, with a store branch under dp.engine and a separate report branch.
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestStoreTree, hierarchyTestReportBranch, hierarchyTestRootDefault)

	// When the master matrix is built.
	matrix, findings := MasterMatrix(snapshot)

	// Then no findings are reported, the sibling pair in the root set takes the none default, and the pairs without a containing set default stay unknown.
	hierarchyTestRequireNoFindings(t, findings)
	reporter := matrix.Cell("fr.report", "dp.reporter")
	if reporter.Effect != EffectNone || !reporter.FromDefault {
		t.Fatalf("cell (fr.report, dp.reporter) effect = %v FromDefault = %t, want none from default", reporter.Effect, reporter.FromDefault)
	}
	for _, pair := range []struct{ fr, dp string }{
		{"fr.report", "dp.cache"},
		{"fr.cache_hit", "dp.reporter"},
		{"fr.cache_hit", "dp.cache"},
	} {
		cell := matrix.Cell(model.ID(pair.fr), model.ID(pair.dp))
		if cell.Effect != EffectUnknown || cell.FromDefault {
			t.Fatalf("cell (%s, %s) effect = %v FromDefault = %t, want unknown without default", pair.fr, pair.dp, cell.Effect, cell.FromDefault)
		}
	}
}

func TestHierarchyRefinementFlagsParentMaterialEffect(t *testing.T) {
	// Given a material influence from a child leaf DP onto its parent FR
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestStoreTree, hierarchyTestReportBranch, hierarchyTestParentInfluence)

	// When refinement is checked
	findings := Refinement(snapshot)

	// Then an unresolved refinement finding is reported
	if !hierarchyTestHasCode(findings, "unresolved_refinement") {
		t.Errorf("findings = %v, want an unresolved_refinement finding", findings)
	}
}

func TestHierarchyRefinementFlagsAllocatedCriterionBySubject(t *testing.T) {
	// Given a child criterion allocated from a criterion on its parent FR
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestAllocationTree)

	// When refinement is checked
	findings := Refinement(snapshot)

	// Then the unresolved refinement finding names the child criterion that carries the allocation
	hierarchyTestRequireIDs(t, "unresolved_refinement subjects", hierarchyTestSubjects(findings, "unresolved_refinement"), "fr.cache_hit.criterion.latency")
}

func TestHierarchyRefinementIgnoresLeafOnlyEffects(t *testing.T) {
	// Given a clean cross-branch leaf-to-leaf material influence with no parent effects or allocations
	snapshot := hierarchyTestSnapshot(t, hierarchyTestDesign, hierarchyTestStoreTree, hierarchyTestReportBranch, hierarchyTestCrossInfluence)

	// When refinement is checked
	findings := Refinement(snapshot)

	// Then no unresolved refinement finding is reported
	if hierarchyTestHasCode(findings, "unresolved_refinement") {
		t.Errorf("findings = %v, want no unresolved_refinement finding", findings)
	}
}

func TestHierarchyMasterSparseStorageFollowsAuthoredFactsAndDefaults(t *testing.T) {
	for _, branches := range []int{4, 16, 64} {
		t.Run(fmt.Sprint(branches), func(t *testing.T) {
			var source strings.Builder
			source.WriteString(hierarchyTestDesign)
			for branch := range branches {
				fmt.Fprintf(&source, `
fr "parent%d" {
  statement = "Produce a branch."
  need = [need.record_payments]
}
dp "parent%d" {
  controls = [fr.parent%d]
  responsibility = "Choose a branch mechanism."
  decision = "selected"
}
cells "defaults%d" {
  scope = dp.parent%d
  unlisted = "none"
  because = "The branch outputs have independent controls."
}
`, branch, branch, branch, branch, branch)
				for leaf := range 2 {
					fmt.Fprintf(&source, `
fr "f%d_%d" {
  parent = dp.parent%d
  statement = "Produce a leaf output."
}
dp "d%d_%d" {
  controls = [fr.f%d_%d]
  responsibility = "Choose a leaf mechanism."
  decision = "selected"
}
influence {
  from = [dp.d%d_%d]
  to = fr.f%d_%d
  effect = "material"
  mechanism = "The mechanism produces the leaf output."
}
`, branch, leaf, branch, branch, leaf, branch, leaf, branch, leaf, branch, leaf)
				}
			}
			source.WriteString(`
influence {
  from = [dp.d1_0]
  to = fr.f0_0
  effect = "unknown"
}
`)
			snapshot := hierarchyTestSnapshot(t, source.String())

			matrix, findings := MasterMatrix(snapshot)

			hierarchyTestRequireNoFindings(t, findings)
			leaves := 2 * branches
			if len(matrix.FRs) != leaves || len(matrix.DPs) != leaves {
				t.Fatalf("axes = %d x %d, want %d x %d", len(matrix.FRs), len(matrix.DPs), leaves, leaves)
			}
			if got := cellsStoredCount(matrix); got != leaves+1 {
				t.Fatalf("stored cells = %d, want %d authored leaf facts", got, leaves+1)
			}
			if len(matrix.defaults) != branches || len(matrix.frScopes) != leaves || len(matrix.dpScopes) != leaves {
				t.Fatalf("defaults/memberships = %d/%d/%d, want %d/%d/%d",
					len(matrix.defaults), len(matrix.frScopes), len(matrix.dpScopes), branches, leaves, leaves)
			}
			if cell := matrix.Cell("fr.f0_0", "dp.d0_1"); cell.Effect != EffectNone || !cell.FromDefault {
				t.Fatalf("same-branch unlisted cell = %+v, want default none", cell)
			}
			if cell := matrix.Cell("fr.f0_0", "dp.d1_0"); cell.Effect != EffectUnknown || len(cell.Unknowns) != 1 || cell.FromDefault {
				t.Fatalf("cross-branch exact unknown = %+v, want retained exact unknown", cell)
			}
			for _, fr := range matrix.FRs {
				for _, dp := range matrix.DPs {
					matrix.Cell(fr, dp)
				}
			}
			if got := cellsStoredCount(matrix); got != leaves+1 {
				t.Fatalf("lookup expanded storage to %d cells", got)
			}
			scope, classifyFindings := Classify(matrix, controlsOf(snapshot.Graph, matrix))
			classifyTestRequireClassification(t, scope, "unresolved")
			classifyTestRequireCode(t, classifyFindings, "unresolved_influence")
			if cell := matrix.Cell("fr.f1_0", "dp.d0_0"); cell.Effect != EffectUnknown || cell.FromDefault {
				t.Fatalf("unlisted cross-branch cell = %+v, want unknown without default", cell)
			}
		})
	}
}
