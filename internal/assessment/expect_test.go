package assessment

import (
	"fmt"
	"os"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

// expectTestRoot declares the design and the need that every expectation fixture shares.
const expectTestRoot = `design "order" {
  boundary = "Price one order."
}

need "priced_order" {
  statement = "Price each order correctly."
}
`

// expectTestUncoupledBody has two root FRs, each controlled by its own selected DP, with
// material diagonal effects and no off-diagonal effect. The %s placeholder holds extra
// lines for the price DP, such as a premise.
const expectTestUncoupledBody = `fr "price" {
  need      = [need.priced_order]
  statement = "Compute the price of each order."
}

fr "tax" {
  need      = [need.priced_order]
  statement = "Compute the tax of each order."
}

dp "price_rule" {
  controls       = [fr.price]
  responsibility = "Set the price rule."
  decision       = "selected"
%s}

dp "tax_rule" {
  controls       = [fr.tax]
  responsibility = "Set the tax rule."
  decision       = "selected"
}

influence {
  from      = [dp.price_rule]
  to        = fr.price
  effect    = "material"
  mechanism = "The price rule sets the price."
}

influence {
  from      = [dp.tax_rule]
  to        = fr.tax
  effect    = "material"
  mechanism = "The tax rule sets the tax."
}

cells "isolated" {
  scope    = design.order
  unlisted = "none"
  because  = "The price and tax rules do not affect each other's FR."
}
`

// expectTestConflictInfluence adds a none record that contradicts the material price diagonal.
const expectTestConflictInfluence = `influence {
  from    = [dp.price_rule]
  to      = fr.price
  effect  = "none"
  because = "The price rule does not set the price in this case."
}
`

// expectTestPremise declares an assumed premise that the price DP depends on.
const expectTestPremise = `assumption "rate_stable" {
  statement = "The price rate stays fixed for one order."
  status    = "assumed"
}
`

// expectTestDecoupledBody has three root FRs. The price and tax rules each materially affect
// the total rule, so both must precede it. Price and tax can come in either order.
const expectTestDecoupledBody = `fr "price" {
  need      = [need.priced_order]
  statement = "Compute the price of each order."
}

fr "tax" {
  need      = [need.priced_order]
  statement = "Compute the tax of each order."
}

fr "total" {
  need      = [need.priced_order]
  statement = "Compute the total of each order."
}

dp "price_rule" {
  controls       = [fr.price]
  responsibility = "Set the price rule."
  decision       = "selected"
}

dp "tax_rule" {
  controls       = [fr.tax]
  responsibility = "Set the tax rule."
  decision       = "selected"
}

dp "total_rule" {
  controls       = [fr.total]
  responsibility = "Set the total rule."
  decision       = "selected"
}

influence {
  from      = [dp.price_rule]
  to        = fr.price
  effect    = "material"
  mechanism = "The price rule sets the price."
}

influence {
  from      = [dp.tax_rule]
  to        = fr.tax
  effect    = "material"
  mechanism = "The tax rule sets the tax."
}

influence {
  from      = [dp.total_rule]
  to        = fr.total
  effect    = "material"
  mechanism = "The total rule sets the total."
}

influence {
  from      = [dp.price_rule]
  to        = fr.total
  effect    = "material"
  mechanism = "The total reads the price."
}

influence {
  from      = [dp.tax_rule]
  to        = fr.total
  effect    = "material"
  mechanism = "The total reads the tax."
}
`

// expectTestDecoupledCells states that every other cell is known to be zero.
const expectTestDecoupledCells = `cells "isolated" {
  scope    = design.order
  unlisted = "none"
  because  = "Only the authored price and tax inputs reach the total."
}
`

// expectTestInactiveDP is a proposed DP that no candidate selects, so it is not active.
const expectTestInactiveDP = `dp "discount_rule" {
  controls       = [fr.total]
  responsibility = "Apply a discount to the total."
  decision       = "proposed"
}
`

// expectTestExpectBlock formats an expect block for the order design. The extra lines are
// appended before the closing brace.
func expectTestExpectBlock(label, scope, classification, extra string) string {
	return fmt.Sprintf("expect %q {\n  scope          = %s\n  classification = %q\n%s}\n", label, scope, classification, extra)
}

// expectTestPipeline parses, decodes, resolves, and selects the sources. It fails the test
// when any setup step reports a finding.
func expectTestPipeline(t *testing.T, sources ...string) (*model.Graph, *selection.Snapshot) {
	t.Helper()
	units := make([]syntax.Unit, 0, len(sources))
	for index, source := range sources {
		unit := syntax.Parse(model.Source{Path: fmt.Sprintf("expect-%d.ad.hcl", index+1), Bytes: []byte(source)})
		if len(unit.Findings) != 0 {
			t.Fatalf("source %d has %d parse findings, want none", index+1, len(unit.Findings))
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
	snapshot, selectFindings := selection.Select(graph, selection.Request{})
	if len(selectFindings) != 0 {
		t.Fatalf("select produced %d findings, want none", len(selectFindings))
	}
	if snapshot == nil {
		t.Fatal("select returned no snapshot")
	}
	return graph, snapshot
}

// expectTestControls maps each active DP of the matrix to the active FRs it controls.
func expectTestControls(graph *model.Graph, matrix Matrix) map[model.ID][]model.ID {
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

// expectTestScopes resolves and classifies every sibling set of the graph.
func expectTestScopes(graph *model.Graph, snapshot *selection.Snapshot) []ScopeResult {
	var scopes []ScopeResult
	for _, set := range graph.SiblingSets {
		matrix, _ := ResolveCells(snapshot, set)
		scope, _ := Classify(matrix, expectTestControls(graph, matrix))
		scopes = append(scopes, scope)
	}
	return scopes
}

// expectTestRun builds the sources and evaluates their expectations.
func expectTestRun(t *testing.T, sources ...string) []result.Finding {
	t.Helper()
	graph, snapshot := expectTestPipeline(t, sources...)
	return EvaluateExpectations(snapshot, expectTestScopes(graph, snapshot))
}

// expectTestRequireNone fails unless the findings are empty.
func expectTestRequireNone(t *testing.T, findings []result.Finding) {
	t.Helper()
	if len(findings) != 0 {
		for _, f := range findings {
			t.Errorf("unexpected finding %s on %s", f.Code, f.Subject)
		}
		t.Fatalf("got %d findings, want none", len(findings))
	}
}

// expectTestRequireOnly fails unless the findings are exactly one finding with the given code
// and subject.
func expectTestRequireOnly(t *testing.T, findings []result.Finding, code, subject string) {
	t.Helper()
	if len(findings) != 1 {
		for _, f := range findings {
			t.Errorf("got finding %s on %s", f.Code, f.Subject)
		}
		t.Fatalf("got %d findings, want exactly one %s on %s", len(findings), code, subject)
	}
	if findings[0].Code != code || findings[0].Subject != subject {
		t.Fatalf("got %s on %s, want %s on %s", findings[0].Code, findings[0].Subject, code, subject)
	}
}

// expectTestDecoupledSources returns the decoupled fixture with the given order, the optional
// extra body, and the cells that make every other cell known to be zero.
func expectTestDecoupledSources(order, extra string) []string {
	return []string{
		expectTestRoot,
		expectTestDecoupledBody + expectTestDecoupledCells + extra,
		expectTestExpectBlock("order_check", "design.order", "decoupled", fmt.Sprintf("  order          = [%s]\n", order)),
	}
}

func TestEvaluateExpectationsUncoupledMatchingPasses(t *testing.T) {
	// Given a design whose two FRs have only their own material diagonal effects, and an expectation of uncoupled.
	sources := []string{
		expectTestRoot,
		fmt.Sprintf(expectTestUncoupledBody, ""),
		expectTestExpectBlock("isolated_order", "design.order", "uncoupled", ""),
	}

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then no finding is reported.
	expectTestRequireNone(t, findings)
}

func TestEvaluateExpectationsWrongClassificationFails(t *testing.T) {
	// Given a design that classifies as uncoupled, and an expectation that claims decoupled.
	sources := []string{
		expectTestRoot,
		fmt.Sprintf(expectTestUncoupledBody, ""),
		expectTestExpectBlock("isolated_order", "design.order", "decoupled", ""),
	}

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then one failed expectation names the expectation.
	expectTestRequireOnly(t, findings, "failed_expectation", "expect.isolated_order")
}

func TestEvaluateExpectationsDecoupledAcceptsAlternativeValidOrder(t *testing.T) {
	// Given a decoupled design whose price and tax rules both precede the total rule,
	// and a decoupled expectation that lists the tax rule before the price rule.
	sources := expectTestDecoupledSources("dp.tax_rule, dp.price_rule, dp.total_rule", "")

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then the alternative valid order passes with no finding.
	expectTestRequireNone(t, findings)
}

func TestEvaluateExpectationsDecoupledRejectsInvalidOrders(t *testing.T) {
	tests := []struct {
		name  string
		order string
		extra string
	}{
		{
			name:  "missing DP",
			order: "dp.price_rule, dp.total_rule",
		},
		{
			name:  "inactive DP",
			order: "dp.price_rule, dp.tax_rule, dp.total_rule, dp.discount_rule",
			extra: expectTestInactiveDP,
		},
		{
			name:  "precedence violated",
			order: "dp.total_rule, dp.price_rule, dp.tax_rule",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given a decoupled design and an order that breaks the DP set or precedence.
			sources := expectTestDecoupledSources(tc.order, tc.extra)

			// When the expectations are evaluated.
			findings := expectTestRun(t, sources...)

			// Then one failed expectation names the expectation.
			expectTestRequireOnly(t, findings, "failed_expectation", "expect.order_check")
		})
	}
}

func TestEvaluateExpectationsUnknownCellsFailDecoupledOrder(t *testing.T) {
	// Given a decoupled design with no cells record, so its off-diagonal cells are unknown,
	// and a decoupled expectation with a valid order.
	sources := []string{
		expectTestRoot,
		expectTestDecoupledBody,
		expectTestExpectBlock("order_check", "design.order", "decoupled", "  order          = [dp.tax_rule, dp.price_rule, dp.total_rule]\n"),
	}

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then unknown cells leave precedence unresolved, so the order fails.
	expectTestRequireOnly(t, findings, "failed_expectation", "expect.order_check")
}

func TestEvaluateExpectationsUnresolvedAcceptsUnknownWithoutConflict(t *testing.T) {
	// Given a decoupled design with no cells record, so its classification is unresolved,
	// and an expectation that claims unresolved with no order.
	sources := []string{
		expectTestRoot,
		expectTestDecoupledBody,
		expectTestExpectBlock("unknown_order", "design.order", "unresolved", ""),
	}

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then the explicit unresolved claim passes with no finding.
	expectTestRequireNone(t, findings)
}

func TestEvaluateExpectationsConflictSatisfiesNoClassification(t *testing.T) {
	tests := []struct {
		name           string
		classification string
	}{
		{name: "uncoupled", classification: "uncoupled"},
		{name: "unresolved", classification: "unresolved"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Given a design whose price diagonal has both a material and a none record,
			// and an expectation of the table's classification.
			sources := []string{
				expectTestRoot,
				fmt.Sprintf(expectTestUncoupledBody, "") + expectTestConflictInfluence,
				expectTestExpectBlock("conflict_check", "design.order", tc.classification, ""),
			}

			// When the expectations are evaluated.
			findings := expectTestRun(t, sources...)

			// Then the conflict cannot satisfy the classification, so one failed expectation is reported.
			expectTestRequireOnly(t, findings, "failed_expectation", "expect.conflict_check")
		})
	}
}

func TestEvaluateExpectationsSupportedBasisFails(t *testing.T) {
	// Given a design that classifies as uncoupled, and an expectation with basis supported.
	// Version 1 has no accepted review input, so supported support cannot be established.
	sources := []string{
		expectTestRoot,
		fmt.Sprintf(expectTestUncoupledBody, ""),
		expectTestExpectBlock("isolated_order", "design.order", "uncoupled", "  basis          = \"supported\"\n"),
	}

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then the requested basis cannot be established, so one failed expectation is reported.
	if !expectTestHasFinding(findings, "failed_expectation", "expect.isolated_order") {
		t.Fatalf("got %d findings, want failed_expectation on expect.isolated_order", len(findings))
	}
}

func TestEvaluateExpectationsConditionalBasisWithAssumedPremisePasses(t *testing.T) {
	// Given a design whose price DP depends on an assumed premise, and an expectation with basis conditional.
	sources := []string{
		expectTestRoot,
		expectTestPremise,
		fmt.Sprintf(expectTestUncoupledBody, "  holds_if       = [assumption.rate_stable]\n"),
		expectTestExpectBlock("isolated_order", "design.order", "uncoupled", "  basis          = \"conditional\"\n"),
	}

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then the assumed premise supports a conditional basis, so no finding is reported.
	expectTestRequireNone(t, findings)
}

func TestEvaluateExpectationsStarterOrderPasses(t *testing.T) {
	// Given the starter design and an added expectation with the author's valid order.
	starter := expectTestStarterSource(t)
	sources := []string{
		starter,
		expectTestExpectBlock("receipt_order", "design.receipt", "decoupled", "  order          = [dp.amount_admission, dp.exact_addition]\n"),
	}

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then the starter order passes with no finding.
	expectTestRequireNone(t, findings)
}

func TestEvaluateExpectationsStarterWrongOrderFails(t *testing.T) {
	// Given the starter design and an added expectation with the reversed order, which breaks precedence.
	starter := expectTestStarterSource(t)
	sources := []string{
		starter,
		expectTestExpectBlock("receipt_order", "design.receipt", "decoupled", "  order          = [dp.exact_addition, dp.amount_admission]\n"),
	}

	// When the expectations are evaluated.
	findings := expectTestRun(t, sources...)

	// Then one failed expectation names the starter expectation.
	expectTestRequireOnly(t, findings, "failed_expectation", "expect.receipt_order")
}

// expectTestStarterSource reads the starter design from the repository's testdata folder.
func expectTestStarterSource(t *testing.T) string {
	t.Helper()
	bytes, err := os.ReadFile("../../testdata/spec/examples/starter/design.ad.hcl")
	if err != nil {
		t.Fatalf("read starter design: %v", err)
	}
	return string(bytes)
}

// expectTestHasFinding reports whether a finding has the given code and subject.
func expectTestHasFinding(findings []result.Finding, code, subject string) bool {
	for _, f := range findings {
		if f.Code == code && f.Subject == subject {
			return true
		}
	}
	return false
}
