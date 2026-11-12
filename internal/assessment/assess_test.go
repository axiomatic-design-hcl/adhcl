package assessment

import (
	"os"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// assessTestAllowedCodes lists the only finding codes the starter_core case permits.
var assessTestAllowedCodes = map[string]bool{
	"support_gap":            true,
	"unresolved_feasibility": true,
}

func TestAssessStarterCoreMatchesConformance(t *testing.T) {
	// Given the pinned starter design from the conformance suite
	snapshot := assessTestStarterSnapshot(t)

	// When the assessment runs on the default snapshot
	got := Assess(snapshot)

	// Then the receipt scope is decoupled with the expected pairing, matrix, and order
	scope := assessTestScope(t, got, "design.receipt")
	if scope.Classification != Decoupled {
		t.Fatalf("classification = %q, want %q", scope.Classification, Decoupled)
	}
	assessTestRequirePairing(t, scope, "fr.admit_amount", "dp.amount_admission")
	assessTestRequirePairing(t, scope, "fr.calculate_total", "dp.exact_addition")
	assessTestRequireEffect(t, scope, "fr.admit_amount", "dp.amount_admission", EffectMaterial)
	assessTestRequireEffect(t, scope, "fr.admit_amount", "dp.exact_addition", EffectNone)
	assessTestRequireEffect(t, scope, "fr.calculate_total", "dp.amount_admission", EffectMaterial)
	assessTestRequireEffect(t, scope, "fr.calculate_total", "dp.exact_addition", EffectMaterial)
	assessTestRequireOrder(t, scope, "dp.amount_admission", "dp.exact_addition")

	// Then the findings are exactly the permitted support gap and unresolved feasibility
	assessTestRequireFindingCodes(t, got.Findings, "support_gap", "unresolved_feasibility")
}

// assessTestStarterSnapshot reads the starter design and selects it with zero setup findings.
func assessTestStarterSnapshot(t *testing.T) *selection.Snapshot {
	t.Helper()
	bytes, err := os.ReadFile("../../testdata/spec/examples/starter/design.ad.hcl")
	if err != nil {
		t.Fatalf("read starter design: %v", err)
	}
	unit := syntax.Parse(model.Source{Path: "design.ad.hcl", Bytes: bytes})
	assessTestRequireNoSetupFindings(t, "parse", unit.Findings)
	core, findings := schema.DecodeCore([]syntax.Unit{unit})
	assessTestRequireNoSetupFindings(t, "decode", findings)
	graph, findings := schema.Resolve(core)
	assessTestRequireNoSetupFindings(t, "resolve", findings)
	snapshot, findings := selection.Select(graph, selection.Request{})
	assessTestRequireNoSetupFindings(t, "select", findings)
	if snapshot == nil {
		t.Fatal("select returned no snapshot")
	}
	return snapshot
}

// assessTestRequireNoSetupFindings fails the test when a setup stage reports any finding.
func assessTestRequireNoSetupFindings(t *testing.T, stage string, findings []result.Finding) {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("%s findings = %d, want 0; first = %s %q", stage, len(findings), findings[0].Code, findings[0].Subject)
	}
}

// assessTestScope returns the scope result with the given scope ID.
func assessTestScope(t *testing.T, got Result, scope model.ID) ScopeResult {
	t.Helper()
	for _, candidate := range got.Scopes {
		if candidate.Matrix.Scope == scope {
			return candidate
		}
	}
	t.Fatalf("no scope result for %q", scope)
	return ScopeResult{}
}

// assessTestRequirePairing checks that the scope pairs the FR with the DP.
func assessTestRequirePairing(t *testing.T, scope ScopeResult, fr, dp model.ID) {
	t.Helper()
	if scope.Pairing[fr] != dp {
		t.Fatalf("pairing of %s = %q, want %q", fr, scope.Pairing[fr], dp)
	}
}

// assessTestRequireEffect checks one cell of the scope matrix.
func assessTestRequireEffect(t *testing.T, scope ScopeResult, fr, dp model.ID, want Effect) {
	t.Helper()
	cell := scope.Matrix.Cell(fr, dp)
	if cell.Effect != want {
		t.Fatalf("cell %s x %s = %q, want %q", fr, dp, cell.Effect, want)
	}
}

// assessTestRequireOrder checks the stable display order of the scope DPs.
func assessTestRequireOrder(t *testing.T, scope ScopeResult, want ...model.ID) {
	t.Helper()
	if len(scope.Order) != len(want) {
		t.Fatalf("order = %v, want %v", scope.Order, want)
	}
	for index := range want {
		if scope.Order[index] != want[index] {
			t.Fatalf("order = %v, want %v", scope.Order, want)
		}
	}
}

// assessTestRequireFindingCodes checks that every finding uses a permitted code and each required code appears.
func assessTestRequireFindingCodes(t *testing.T, findings []result.Finding, required ...string) {
	t.Helper()
	seen := make(map[string]bool, len(findings))
	for _, finding := range findings {
		if !assessTestAllowedCodes[finding.Code] {
			t.Errorf("finding code %q is not permitted; subject %q", finding.Code, finding.Subject)
		}
		seen[finding.Code] = true
	}
	for _, code := range required {
		if !seen[code] {
			t.Errorf("no finding with code %q", code)
		}
	}
}
