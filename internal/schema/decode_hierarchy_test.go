package schema

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

const hierarchyTestPath = "hierarchy-case.ad.hcl"

const (
	hierarchyStatement          = `statement = "Build the database."`
	hierarchyCriterionStatement = `statement = "Retention is kept."`
	hierarchyMetric             = `metric = "retention_days"`
)

func decodeHierarchyCore(t *testing.T, source string) (model.Core, []result.Finding) {
	t.Helper()
	unit := syntax.Parse(model.Source{Path: hierarchyTestPath, Bytes: []byte(source)})
	return DecodeCore([]syntax.Unit{unit})
}

// hierarchyBlock renders a block header, then each attribute or child block indented
// one level under it.
func hierarchyBlock(header string, lines ...string) string {
	var source strings.Builder
	source.WriteString(header + " {\n")
	for _, line := range lines {
		for _, part := range strings.Split(strings.TrimSuffix(line, "\n"), "\n") {
			source.WriteString("  " + part + "\n")
		}
	}
	source.WriteString("}\n")
	return source.String()
}

// hierarchyCriterionCore places one criterion with the given attributes inside an FR.
func hierarchyCriterionCore(attributes ...string) string {
	criterion := hierarchyBlock(`criterion "retention"`, attributes...)
	return hierarchyBlock(`fr "build_db"`, hierarchyStatement, "need = need.build", criterion)
}

func hierarchyOnly[T any](t *testing.T, name string, items []T) T {
	t.Helper()
	if len(items) != 1 {
		t.Fatalf("decoded %d %s records, want 1", len(items), name)
	}
	return items[0]
}

func hierarchyFirstCriterion(t *testing.T, core model.Core) model.Criterion {
	t.Helper()
	fr := hierarchyOnly(t, "FR", core.FRs)
	return hierarchyOnly(t, "criterion", fr.Criteria)
}

func hierarchyFindingCodes(findings []result.Finding) []string {
	codes := make([]string, len(findings))
	for index, finding := range findings {
		codes[index] = finding.Code
	}
	return codes
}

func requireHierarchyNoFindings(t *testing.T, findings []result.Finding) {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("finding codes = %v, want none", hierarchyFindingCodes(findings))
	}
}

// requireHierarchyFinding checks that exactly one finding exists, with the given code,
// starting on the given line of the test file.
func requireHierarchyFinding(t *testing.T, findings []result.Finding, code string, line int) {
	t.Helper()
	if len(findings) != 1 {
		t.Fatalf("finding codes = %v, want exactly [%s]", hierarchyFindingCodes(findings), code)
	}
	finding := findings[0]
	if finding.Code != code {
		t.Fatalf("finding code = %q, want %q", finding.Code, code)
	}
	if finding.Range == nil {
		t.Fatal("finding has no source range")
	}
	if finding.Range.Filename != hierarchyTestPath || finding.Range.Start.Line != line {
		t.Fatalf("finding range = %s:%d, want %s:%d", finding.Range.Filename, finding.Range.Start.Line, hierarchyTestPath, line)
	}
}

// hierarchyLines returns every 1-based line of source that contains marker.
func hierarchyLines(t *testing.T, source, marker string) []int {
	t.Helper()
	var lines []int
	for index, text := range strings.Split(source, "\n") {
		if strings.Contains(text, marker) {
			lines = append(lines, index+1)
		}
	}
	if len(lines) == 0 {
		t.Fatalf("marker %q is not in the source", marker)
	}
	return lines
}

func hierarchyLine(t *testing.T, source, marker string) int {
	t.Helper()
	lines := hierarchyLines(t, source, marker)
	if len(lines) != 1 {
		t.Fatalf("marker %q is on lines %v, want exactly one line", marker, lines)
	}
	return lines[0]
}

func requireHierarchyTargets(t *testing.T, refs []model.Ref, want ...string) {
	t.Helper()
	got := make([]string, len(refs))
	for index, ref := range refs {
		got[index] = string(ref.Target)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("targets = %v, want %v", got, want)
	}
}

func requireHierarchyString(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is absent, want %q", field, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", field, *got, want)
	}
}

// requireHierarchyNumber compares exact values. The expected text is parsed at the
// same 512-bit precision that the schema number uses, so a rounded float64 fails.
func requireHierarchyNumber(t *testing.T, got *big.Float, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("number is absent, want %s", want)
	}
	exact, _, err := big.ParseFloat(want, 10, 512, big.ToNearestEven)
	if err != nil {
		t.Fatalf("parse expected number %s: %v", want, err)
	}
	if got.Cmp(exact) != 0 {
		t.Fatalf("number = %s, want %s", got.Text('g', -1), want)
	}
}

func hierarchyScalarText(scalar model.Scalar) string {
	switch scalar.Kind {
	case model.ValueNumber:
		if scalar.Number == nil {
			return "<absent number>"
		}
		return scalar.Number.Text('g', -1)
	case model.ValueBool:
		return strconv.FormatBool(scalar.Flag)
	default:
		return scalar.Text
	}
}

// TestDecodeHierarchyFRNeedFormsKeepAuthoredTargets covers the scalar and list forms of fr.need.
func TestDecodeHierarchyFRNeedFormsKeepAuthoredTargets(t *testing.T) {
	// Given an FR whose need is one reference or a list of references.
	cases := []struct {
		name string
		need string
		want []string
	}{
		{name: "scalar reference", need: "need = need.build", want: []string{"need.build"}},
		{name: "list in authored order", need: "need = [need.build, need.audit]", want: []string{"need.build", "need.audit"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the FR is decoded.
			source := hierarchyBlock(`fr "build_db"`, hierarchyStatement, tc.need)
			core, findings := decodeHierarchyCore(t, source)
			fr := hierarchyOnly(t, "FR", core.FRs)

			// Then the need is present with the authored targets and no finding is reported.
			requireHierarchyNoFindings(t, findings)
			if !fr.Need.Present {
				t.Fatal("need is absent, want present")
			}
			requireHierarchyTargets(t, fr.Need.Items, tc.want...)
		})
	}
}

func TestDecodeHierarchyFROmittedNeedIsAbsent(t *testing.T) {
	// Given a child FR that omits its need.
	source := hierarchyBlock(`fr "build_db"`, hierarchyStatement, "parent = dp.split")

	// When the FR is decoded.
	core, findings := decodeHierarchyCore(t, source)
	fr := hierarchyOnly(t, "FR", core.FRs)

	// Then the need is recorded as absent, not as an empty list, and no finding is reported.
	requireHierarchyNoFindings(t, findings)
	if fr.Need.Present {
		t.Fatalf("need is present with %d items, want absent", len(fr.Need.Items))
	}
}

// TestDecodeHierarchyRootFRWithoutNeedIsInvalid covers the rule that a root FR must name its need.
func TestDecodeHierarchyRootFRWithoutNeedIsInvalid(t *testing.T) {
	// Given a root FR with no parent and no need.
	source := hierarchyBlock(`fr "build_db"`, hierarchyStatement)

	// When the FR is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the fr block header.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, `fr "build_db" {`))
}

// TestDecodeHierarchyChildFRWithoutNeedIsValid covers the rule that a child FR may omit its need.
func TestDecodeHierarchyChildFRWithoutNeedIsValid(t *testing.T) {
	// Given a child FR with a parent DP and no need.
	source := hierarchyBlock(`fr "build_db"`, hierarchyStatement, "parent = dp.split")

	// When the FR is decoded.
	core, findings := decodeHierarchyCore(t, source)
	fr := hierarchyOnly(t, "FR", core.FRs)

	// Then no finding is reported and the need is absent.
	requireHierarchyNoFindings(t, findings)
	if fr.Need.Present {
		t.Fatalf("need is present with %d items, want absent", len(fr.Need.Items))
	}
}

func TestDecodeHierarchyFRRejectsExplicitEmptyNeed(t *testing.T) {
	// Given an FR that writes an explicit empty need list.
	source := hierarchyBlock(`fr "build_db"`, hierarchyStatement, "need = []")

	// When the FR is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the need attribute.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, "need = []"))
}

func TestDecodeHierarchyFRRecordsParentOnlyWhenAuthored(t *testing.T) {
	// Given an FR written either as a root with no parent or as a child of dp.split.
	cases := []struct {
		name   string
		parent string
		want   string
	}{
		{name: "root without parent", parent: "", want: ""},
		{name: "child of a DP", parent: "parent = dp.split", want: "dp.split"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := []string{hierarchyStatement, "need = need.build"}
			if tc.parent != "" {
				lines = append(lines, tc.parent)
			}
			source := hierarchyBlock(`fr "writes"`, lines...)

			// When the FR is decoded.
			core, findings := decodeHierarchyCore(t, source)
			fr := hierarchyOnly(t, "FR", core.FRs)

			// Then the parent is absent for a root and keeps its DP target for a child.
			requireHierarchyNoFindings(t, findings)
			if tc.want == "" {
				if fr.Parent != nil {
					t.Fatalf("parent = %s, want absent", fr.Parent.Target)
				}
				return
			}
			if fr.Parent == nil {
				t.Fatal("parent is absent, want present")
			}
			requireHierarchyTargets(t, []model.Ref{*fr.Parent}, tc.want)
		})
	}
}

// TestDecodeHierarchyCriterionAcceptsStatementOnly covers the qualitative criterion form.
func TestDecodeHierarchyCriterionAcceptsStatementOnly(t *testing.T) {
	// Given a criterion with only a qualitative statement.
	source := hierarchyCriterionCore(hierarchyCriterionStatement)

	// When the source is decoded.
	core, findings := decodeHierarchyCore(t, source)
	criterion := hierarchyFirstCriterion(t, core)

	// Then the ID and statement are kept, the operational fields stay absent, and no finding is reported.
	requireHierarchyNoFindings(t, findings)
	if criterion.ID != "fr.build_db.criterion.retention" {
		t.Fatalf("ID = %s, want fr.build_db.criterion.retention", criterion.ID)
	}
	requireHierarchyString(t, "statement", criterion.Statement, "Retention is kept.")
	if criterion.Metric != nil || criterion.Op != nil || criterion.Value != nil {
		t.Fatal("operational fields are present, want absent for a statement-only criterion")
	}
}

// TestDecodeHierarchyCriterionRejectsPartialOperationalTriple covers every incomplete metric, op, value combination.
func TestDecodeHierarchyCriterionRejectsPartialOperationalTriple(t *testing.T) {
	// Given criteria that carry a statement and only part of the metric, op, value triple.
	cases := []struct {
		name       string
		attributes []string
	}{
		{name: "metric only", attributes: []string{hierarchyMetric}},
		{name: "op only", attributes: []string{`op = "<"`}},
		{name: "value only", attributes: []string{"value = 30"}},
		{name: "metric and op", attributes: []string{hierarchyMetric, `op = "<"`}},
		{name: "metric and value", attributes: []string{hierarchyMetric, "value = 30"}},
		{name: "op and value", attributes: []string{`op = "<"`, "value = 30"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the criterion is decoded.
			attributes := append([]string{hierarchyCriterionStatement}, tc.attributes...)
			source := hierarchyCriterionCore(attributes...)
			_, findings := decodeHierarchyCore(t, source)

			// Then one invalid_structure finding points at the criterion block.
			requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, `criterion "retention" {`))
		})
	}
}

// TestDecodeHierarchyCriterionNumericOperatorKeepsExactNumber covers number values for ordering operators.
func TestDecodeHierarchyCriterionNumericOperatorKeepsExactNumber(t *testing.T) {
	// Given numeric criteria with an exponent form and a negative fraction.
	cases := []struct {
		name  string
		op    model.CriterionOp
		value string
		want  string
	}{
		{name: "exponent form", op: model.CriterionOpLess, value: "1e3", want: "1e3"},
		{name: "negative fraction", op: model.CriterionOpGreaterEqual, value: "-0.5", want: "-0.5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the criterion is decoded.
			source := hierarchyCriterionCore(
				hierarchyCriterionStatement,
				hierarchyMetric,
				fmt.Sprintf("op = %q", string(tc.op)),
				"value = "+tc.value,
			)
			core, findings := decodeHierarchyCore(t, source)
			criterion := hierarchyFirstCriterion(t, core)

			// Then the operator, the number kind, and the exact value are kept with no finding.
			requireHierarchyNoFindings(t, findings)
			if criterion.Op == nil || *criterion.Op != tc.op {
				t.Fatalf("op = %v, want %s", criterion.Op, tc.op)
			}
			if criterion.Value == nil || criterion.Value.Kind != model.ValueNumber {
				t.Fatal("value is not a number")
			}
			if criterion.Value.Scalar.Kind != model.ValueNumber {
				t.Fatalf("scalar kind = %s, want number", criterion.Value.Scalar.Kind)
			}
			requireHierarchyNumber(t, criterion.Value.Scalar.Number, tc.want)
		})
	}
}

// TestDecodeHierarchyCriterionEqualityKeepsScalarType covers equality values and their types.
func TestDecodeHierarchyCriterionEqualityKeepsScalarType(t *testing.T) {
	// Given equality criteria with a quoted number, a word, a Boolean, and a bare number.
	cases := []struct {
		name      string
		value     string
		wantKind  model.ValueKind
		wantValue string
	}{
		{name: "quoted digits stay a string", value: `"1"`, wantKind: model.ValueString, wantValue: "1"},
		{name: "word", value: `"ready"`, wantKind: model.ValueString, wantValue: "ready"},
		{name: "boolean", value: "true", wantKind: model.ValueBool, wantValue: "true"},
		{name: "bare number", value: "2", wantKind: model.ValueNumber, wantValue: "2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the criterion is decoded.
			source := hierarchyCriterionCore(
				hierarchyCriterionStatement,
				hierarchyMetric,
				`op = "=="`,
				"value = "+tc.value,
			)
			core, findings := decodeHierarchyCore(t, source)
			criterion := hierarchyFirstCriterion(t, core)

			// Then the value keeps its kind and text, with no finding.
			requireHierarchyNoFindings(t, findings)
			if criterion.Value == nil {
				t.Fatal("value is absent")
			}
			if criterion.Value.Kind != tc.wantKind || criterion.Value.Scalar.Kind != tc.wantKind {
				t.Fatalf("value kind = %s, want %s", criterion.Value.Kind, tc.wantKind)
			}
			if got := hierarchyScalarText(criterion.Value.Scalar); got != tc.wantValue {
				t.Fatalf("value = %q, want %q", got, tc.wantValue)
			}
		})
	}
}

func TestDecodeHierarchyCriterionMembershipListKeepsAuthoredOrder(t *testing.T) {
	// Given a membership criterion with a list of two numbers in descending order.
	source := hierarchyCriterionCore(
		hierarchyCriterionStatement,
		hierarchyMetric,
		`op = "in"`,
		"value = [10, 1]",
	)

	// When the criterion is decoded.
	core, findings := decodeHierarchyCore(t, source)
	criterion := hierarchyFirstCriterion(t, core)

	// Then the value is a list of exactly those two numbers in the authored order.
	requireHierarchyNoFindings(t, findings)
	if criterion.Value == nil || criterion.Value.Kind != model.ValueList {
		t.Fatal("value is not a list")
	}
	texts := make([]string, len(criterion.Value.List))
	for index, scalar := range criterion.Value.List {
		if scalar.Kind != model.ValueNumber {
			t.Fatalf("list element %d kind = %s, want number", index, scalar.Kind)
		}
		texts[index] = hierarchyScalarText(scalar)
	}
	if strings.Join(texts, ",") != "10,1" {
		t.Fatalf("list = %v, want [10 1]", texts)
	}
}

// TestDecodeHierarchyCriterionMembershipIntervalKeepsEndpoints covers inclusive interval values.
func TestDecodeHierarchyCriterionMembershipIntervalKeepsEndpoints(t *testing.T) {
	// Given interval criteria with distinct endpoints and with equal endpoints.
	cases := []struct {
		name    string
		value   string
		wantMin string
		wantMax string
	}{
		{name: "distinct endpoints", value: "{ min = 1, max = 10 }", wantMin: "1", wantMax: "10"},
		{name: "equal endpoints", value: "{ min = 2, max = 2 }", wantMin: "2", wantMax: "2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the criterion is decoded.
			source := hierarchyCriterionCore(
				hierarchyCriterionStatement,
				hierarchyMetric,
				`op = "in"`,
				"value = "+tc.value,
			)
			core, findings := decodeHierarchyCore(t, source)
			criterion := hierarchyFirstCriterion(t, core)

			// Then the value is an interval whose inclusive endpoints are kept exactly, with no finding.
			requireHierarchyNoFindings(t, findings)
			if criterion.Value == nil || criterion.Value.Kind != model.ValueInterval {
				t.Fatal("value is not an interval")
			}
			requireHierarchyNumber(t, criterion.Value.Interval.Min, tc.wantMin)
			requireHierarchyNumber(t, criterion.Value.Interval.Max, tc.wantMax)
		})
	}
}

// TestDecodeHierarchyCriterionRejectsValueShapeForOperator covers each value shape that its operator forbids.
func TestDecodeHierarchyCriterionRejectsValueShapeForOperator(t *testing.T) {
	// Given criteria whose value shape does not match the operator.
	cases := []struct {
		name  string
		op    string
		value string
	}{
		{name: "string for less than", op: "<", value: `"fast"`},
		{name: "list for equality", op: "==", value: "[1]"},
		{name: "scalar for membership", op: "in", value: "5"},
		{name: "mixed list", op: "in", value: `[1, "a"]`},
		{name: "duplicate list value", op: "in", value: "[1, 1]"},
		{name: "empty list", op: "in", value: "[]"},
		{name: "nested list", op: "in", value: "[[1], [2]]"},
		{name: "interval with minimum above maximum", op: "in", value: "{ min = 10, max = 1 }"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the criterion is decoded.
			source := hierarchyCriterionCore(
				hierarchyCriterionStatement,
				hierarchyMetric,
				fmt.Sprintf("op = %q", tc.op),
				"value = "+tc.value,
			)
			_, findings := decodeHierarchyCore(t, source)

			// Then one invalid_structure finding points at the value attribute.
			requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, "value = "+tc.value))
		})
	}
}

func TestDecodeHierarchyCriterionRejectsOperatorOutsideEnum(t *testing.T) {
	// Given a criterion whose operator is not in the admitted set.
	source := hierarchyCriterionCore(
		hierarchyCriterionStatement,
		hierarchyMetric,
		`op = "!="`,
		"value = 1",
	)

	// When the criterion is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the op attribute.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, `op = "!="`))
}

// TestDecodeHierarchyCriterionOriginAdmitsOnlyCriterionOrigins covers the criterion-specific origin set.
func TestDecodeHierarchyCriterionOriginAdmitsOnlyCriterionOrigins(t *testing.T) {
	// Given criteria with each origin value, including one that only other blocks admit.
	cases := []struct {
		name    string
		origin  string
		want    model.Origin
		rejects bool
	}{
		{name: "user", origin: "user", want: model.OriginUser},
		{name: "proposed", origin: "proposed", want: model.OriginProposed},
		{name: "source", origin: "source", want: model.OriginSource},
		{name: "inferred", origin: "inferred", rejects: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the criterion is decoded.
			originLine := fmt.Sprintf("origin = %q", tc.origin)
			source := hierarchyCriterionCore(hierarchyCriterionStatement, originLine)
			core, findings := decodeHierarchyCore(t, source)

			// Then admitted origins are kept and the inferred origin gets one invalid_structure finding.
			if tc.rejects {
				requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, originLine))
				return
			}
			criterion := hierarchyFirstCriterion(t, core)
			requireHierarchyNoFindings(t, findings)
			if criterion.Origin == nil || *criterion.Origin != tc.want {
				t.Fatalf("origin = %v, want %s", criterion.Origin, tc.want)
			}
		})
	}
}

func TestDecodeHierarchyDPKeepsControlsAndDecision(t *testing.T) {
	// Given a DP that controls two sibling FRs and is marked selected.
	source := hierarchyBlock(
		`dp "split"`,
		"controls = [fr.reads, fr.writes]",
		`responsibility = "Choose the storage path."`,
		`decision = "selected"`,
	)

	// When the DP is decoded.
	core, findings := decodeHierarchyCore(t, source)
	dp := hierarchyOnly(t, "DP", core.DPs)

	// Then the ID, control targets in authored order, responsibility, and decision are kept with no finding.
	requireHierarchyNoFindings(t, findings)
	if dp.ID != "dp.split" {
		t.Fatalf("ID = %s, want dp.split", dp.ID)
	}
	requireHierarchyTargets(t, dp.Controls, "fr.reads", "fr.writes")
	if dp.Responsibility != "Choose the storage path." {
		t.Fatalf("responsibility = %q, want the authored text", dp.Responsibility)
	}
	if dp.Decision != model.LifecycleSelected {
		t.Fatalf("decision = %s, want selected", dp.Decision)
	}
}

// TestDecodeHierarchyDPDecisionAdmitsLifecycleValues covers the four lifecycle values.
func TestDecodeHierarchyDPDecisionAdmitsLifecycleValues(t *testing.T) {
	// Given a DP with each admitted lifecycle value.
	cases := []struct {
		name  string
		value string
		want  model.LifecycleState
	}{
		{name: "proposed", value: "proposed", want: model.LifecycleProposed},
		{name: "selected", value: "selected", want: model.LifecycleSelected},
		{name: "rejected", value: "rejected", want: model.LifecycleRejected},
		{name: "superseded", value: "superseded", want: model.LifecycleSuperseded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the DP is decoded.
			source := hierarchyBlock(
				`dp "split"`,
				"controls = [fr.writes]",
				`responsibility = "Choose the storage path."`,
				fmt.Sprintf("decision = %q", tc.value),
			)
			core, findings := decodeHierarchyCore(t, source)
			dp := hierarchyOnly(t, "DP", core.DPs)

			// Then the lifecycle value is kept with no finding.
			requireHierarchyNoFindings(t, findings)
			if dp.Decision != tc.want {
				t.Fatalf("decision = %s, want %s", dp.Decision, tc.want)
			}
		})
	}
}

func TestDecodeHierarchyDPRejectsUnlistedDecision(t *testing.T) {
	// Given a DP whose decision is a value outside the lifecycle enum.
	source := hierarchyBlock(
		`dp "split"`,
		"controls = [fr.writes]",
		`responsibility = "Choose the storage path."`,
		`decision = "active"`,
	)

	// When the DP is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the decision attribute.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, `decision = "active"`))
}

func TestDecodeHierarchyDPRejectsEmptyControls(t *testing.T) {
	// Given a DP with an explicit empty controls list.
	source := hierarchyBlock(
		`dp "split"`,
		"controls = []",
		`responsibility = "Choose the storage path."`,
		`decision = "proposed"`,
	)

	// When the DP is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the controls attribute.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, "controls = []"))
}

// TestDecodeHierarchyDPOwnsStateKeepsAbsentSeparateFromEmpty covers the optional owns_state list.
func TestDecodeHierarchyDPOwnsStateKeepsAbsentSeparateFromEmpty(t *testing.T) {
	// Given a DP that omits owns_state, writes it as an empty list, or lists one state.
	cases := []struct {
		name        string
		attribute   string
		wantPresent bool
		wantTargets []string
	}{
		{name: "omitted", attribute: "", wantPresent: false},
		{name: "explicit empty list", attribute: "owns_state = []", wantPresent: true},
		{name: "one state", attribute: "owns_state = [state.cache]", wantPresent: true, wantTargets: []string{"state.cache"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the DP is decoded.
			lines := []string{
				"controls = [fr.writes]",
				`responsibility = "Choose the storage path."`,
				`decision = "selected"`,
			}
			if tc.attribute != "" {
				lines = append(lines, tc.attribute)
			}
			core, findings := decodeHierarchyCore(t, hierarchyBlock(`dp "split"`, lines...))
			dp := hierarchyOnly(t, "DP", core.DPs)

			// Then absence and an explicit empty list stay distinct, with no finding.
			requireHierarchyNoFindings(t, findings)
			if dp.OwnsState.Present != tc.wantPresent {
				t.Fatalf("owns_state present = %v, want %v", dp.OwnsState.Present, tc.wantPresent)
			}
			requireHierarchyTargets(t, dp.OwnsState.Items, tc.wantTargets...)
		})
	}
}

func TestDecodeHierarchyInfluenceKeepsMaterialClaimAndProvenance(t *testing.T) {
	// Given a material influence with a source-stated mechanism, a locator, and a context.
	source := hierarchyBlock(
		"influence",
		"from = [dp.split]",
		"to = fr.writes",
		`effect = "material"`,
		`mechanism = "The split adds a commit step."`,
		`mechanism_source = "source"`,
		`source = "Figure 5, FR2/DP1 cell"`,
		"under = context.peak",
	)

	// When the influence is decoded.
	core, findings := decodeHierarchyCore(t, source)
	influence := hierarchyOnly(t, "influence", core.Influences)

	// Then every authored field is kept with its exact value and the block range, with no finding.
	requireHierarchyNoFindings(t, findings)
	requireHierarchyTargets(t, influence.From, "dp.split")
	requireHierarchyTargets(t, []model.Ref{influence.To}, "fr.writes")
	if influence.Effect != model.EffectMaterial {
		t.Fatalf("effect = %s, want material", influence.Effect)
	}
	requireHierarchyString(t, "mechanism", influence.Mechanism, "The split adds a commit step.")
	if influence.MechanismSource == nil || *influence.MechanismSource != model.MechanismFromSource {
		t.Fatalf("mechanism_source = %v, want source", influence.MechanismSource)
	}
	requireHierarchyString(t, "source", influence.SourceLocator, "Figure 5, FR2/DP1 cell")
	if influence.Under == nil {
		t.Fatal("under is absent, want context.peak")
	}
	requireHierarchyTargets(t, []model.Ref{*influence.Under}, "context.peak")
	if influence.Because != nil {
		t.Fatal("because is present, want absent")
	}
	if influence.Range.Start.Line != hierarchyLine(t, source, "influence {") {
		t.Fatalf("range line = %d, want the influence block line", influence.Range.Start.Line)
	}
}

func TestDecodeHierarchyInfluenceUnknownKeepsMechanismAndBecauseTogether(t *testing.T) {
	// Given an unknown influence that carries both a mechanism and a because note.
	source := hierarchyBlock(
		"influence",
		"from = [dp.split]",
		"to = fr.writes",
		`effect = "unknown"`,
		`mechanism = "A cache may change the write path."`,
		`because = "No measurement exists yet."`,
	)

	// When the influence is decoded.
	core, findings := decodeHierarchyCore(t, source)
	influence := hierarchyOnly(t, "influence", core.Influences)

	// Then both fields are kept with their exact text and the effect stays unknown, with no finding.
	requireHierarchyNoFindings(t, findings)
	if influence.Effect != model.EffectUnknown {
		t.Fatalf("effect = %s, want unknown", influence.Effect)
	}
	requireHierarchyString(t, "mechanism", influence.Mechanism, "A cache may change the write path.")
	requireHierarchyString(t, "because", influence.Because, "No measurement exists yet.")
}

func TestDecodeHierarchyInfluenceMaterialRequiresMechanism(t *testing.T) {
	// Given a material influence with no mechanism.
	source := hierarchyBlock(
		"influence",
		"from = [dp.split]",
		"to = fr.writes",
		`effect = "material"`,
	)

	// When the influence is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the influence block.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, "influence {"))
}

func TestDecodeHierarchyInfluenceNoneRequiresBecause(t *testing.T) {
	// Given a none influence with no because note.
	source := hierarchyBlock(
		"influence",
		"from = [dp.split]",
		"to = fr.writes",
		`effect = "none"`,
	)

	// When the influence is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the influence block.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, "influence {"))
}

func TestDecodeHierarchyInfluenceMechanismSourceRequiresMechanism(t *testing.T) {
	// Given an unknown influence that gives a mechanism_source without a mechanism.
	source := hierarchyBlock(
		"influence",
		"from = [dp.split]",
		"to = fr.writes",
		`effect = "unknown"`,
		`mechanism_source = "source"`,
	)

	// When the influence is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the mechanism_source attribute.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, `mechanism_source = "source"`))
}

func TestDecodeHierarchyInfluenceKeepsMultipleSourcesAsOneRecord(t *testing.T) {
	// Given a none influence whose from list names two source DPs.
	source := hierarchyBlock(
		"influence",
		"from = [dp.split, dp.cache]",
		"to = fr.writes",
		`effect = "none"`,
		`because = "Both sources leave the write path unchanged."`,
	)

	// When the influence is decoded.
	core, findings := decodeHierarchyCore(t, source)

	// Then the block decodes to one record whose from list keeps both sources in authored order.
	requireHierarchyNoFindings(t, findings)
	influence := hierarchyOnly(t, "influence", core.Influences)
	requireHierarchyTargets(t, influence.From, "dp.split", "dp.cache")
}

func TestDecodeHierarchyInfluenceKeepsRepeatedEndpointsInAuthoredOrder(t *testing.T) {
	// Given two influence blocks with the same source and target, one material and one none.
	source := hierarchyBlock(
		"influence",
		"from = [dp.split]",
		"to = fr.writes",
		`effect = "material"`,
		`mechanism = "The split adds a commit step."`,
	) + hierarchyBlock(
		"influence",
		"from = [dp.split]",
		"to = fr.writes",
		`effect = "none"`,
		`because = "The read path is unchanged."`,
	)

	// When the influences are decoded.
	core, findings := decodeHierarchyCore(t, source)
	blockLines := hierarchyLines(t, source, "influence {")

	// Then both records are kept in authored order, each with its own block line, and no finding is reported.
	requireHierarchyNoFindings(t, findings)
	if len(core.Influences) != 2 {
		t.Fatalf("decoded %d influences, want 2", len(core.Influences))
	}
	wantEffects := []model.Effect{model.EffectMaterial, model.EffectNone}
	for index, influence := range core.Influences {
		if influence.Effect != wantEffects[index] {
			t.Fatalf("influence %d effect = %s, want %s", index, influence.Effect, wantEffects[index])
		}
		if influence.Range.Start.Line != blockLines[index] {
			t.Fatalf("influence %d line = %d, want %d", index, influence.Range.Start.Line, blockLines[index])
		}
	}
}

func TestDecodeHierarchyCellsNoneRequiresBecause(t *testing.T) {
	// Given a cells default that leaves unlisted cells as none and gives no because note.
	source := hierarchyBlock(
		`cells "defaults"`,
		"scope = dp.split",
		`unlisted = "none"`,
	)

	// When the cells block is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the cells block.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, `cells "defaults" {`))
}

// TestDecodeHierarchyCellsKeepsScopeAndUnlistedDefault covers the default effect of unlisted cells.
func TestDecodeHierarchyCellsKeepsScopeAndUnlistedDefault(t *testing.T) {
	// Given cells defaults with unlisted none and a because note, and with unlisted unknown and no note.
	cases := []struct {
		name      string
		unlisted  string
		because   string
		wantNote  string
		wantNoted bool
	}{
		{name: "none with because", unlisted: "none", because: `because = "No other pair is material."`, wantNote: "No other pair is material.", wantNoted: true},
		{name: "unknown without because", unlisted: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the cells block is decoded.
			lines := []string{
				"scope = dp.split",
				fmt.Sprintf("unlisted = %q", tc.unlisted),
			}
			if tc.because != "" {
				lines = append(lines, tc.because)
			}
			core, findings := decodeHierarchyCore(t, hierarchyBlock(`cells "defaults"`, lines...))
			cells := hierarchyOnly(t, "cells", core.Cells)

			// Then the ID, scope, unlisted default, and because note are kept with no finding.
			requireHierarchyNoFindings(t, findings)
			if cells.ID != "cells.defaults" {
				t.Fatalf("ID = %s, want cells.defaults", cells.ID)
			}
			requireHierarchyTargets(t, []model.Ref{cells.Scope}, "dp.split")
			if string(cells.Unlisted) != tc.unlisted {
				t.Fatalf("unlisted = %s, want %s", cells.Unlisted, tc.unlisted)
			}
			if !tc.wantNoted {
				if cells.Because != nil {
					t.Fatal("because is present, want absent")
				}
				return
			}
			requireHierarchyString(t, "because", cells.Because, tc.wantNote)
		})
	}
}

func TestDecodeHierarchyExpectBasisDefaultsToSource(t *testing.T) {
	// Given an expect block that omits basis.
	source := hierarchyBlock(
		`expect "split_order"`,
		"scope = dp.split",
		`classification = "uncoupled"`,
	)

	// When the expect block is decoded.
	core, findings := decodeHierarchyCore(t, source)
	expect := hierarchyOnly(t, "expect", core.Expects)

	// Then the basis takes its default value of source, with no finding.
	requireHierarchyNoFindings(t, findings)
	if expect.Basis == nil || *expect.Basis != model.ExpectBasisSource {
		t.Fatalf("basis = %v, want source", expect.Basis)
	}
}

func TestDecodeHierarchyExpectKeepsAuthoredOrderAndExplicitBasis(t *testing.T) {
	// Given an expect block with an authored DP order, an explicit basis, and a context.
	source := hierarchyBlock(
		`expect "split_order"`,
		"scope = dp.split",
		`classification = "decoupled"`,
		"order = [dp.cache, dp.split]",
		`basis = "conditional"`,
		"under = context.peak",
	)

	// When the expect block is decoded.
	core, findings := decodeHierarchyCore(t, source)
	expect := hierarchyOnly(t, "expect", core.Expects)

	// Then the ID, scope, classification, order, basis, and context keep their authored values, with no finding.
	requireHierarchyNoFindings(t, findings)
	if expect.ID != "expect.split_order" {
		t.Fatalf("ID = %s, want expect.split_order", expect.ID)
	}
	requireHierarchyTargets(t, []model.Ref{expect.Scope}, "dp.split")
	if expect.Classification != model.ClassificationDecoupled {
		t.Fatalf("classification = %s, want decoupled", expect.Classification)
	}
	if !expect.Order.Present {
		t.Fatal("order is absent, want present")
	}
	requireHierarchyTargets(t, expect.Order.Items, "dp.cache", "dp.split")
	if expect.Basis == nil || *expect.Basis != model.ExpectBasisConditional {
		t.Fatalf("basis = %v, want conditional", expect.Basis)
	}
	if expect.Under == nil {
		t.Fatal("under is absent, want context.peak")
	}
	requireHierarchyTargets(t, []model.Ref{*expect.Under}, "context.peak")
}

func TestDecodeHierarchyExpectOmittedOrderIsAbsent(t *testing.T) {
	// Given an expect block that omits order.
	source := hierarchyBlock(
		`expect "split_order"`,
		"scope = dp.split",
		`classification = "uncoupled"`,
	)

	// When the expect block is decoded.
	core, findings := decodeHierarchyCore(t, source)
	expect := hierarchyOnly(t, "expect", core.Expects)

	// Then the order is recorded as absent, not as an empty list, with no finding.
	requireHierarchyNoFindings(t, findings)
	if expect.Order.Present {
		t.Fatalf("order is present with %d items, want absent", len(expect.Order.Items))
	}
}

func TestDecodeHierarchyExpectRejectsEmptyOrder(t *testing.T) {
	// Given an expect block with an explicit empty order list.
	source := hierarchyBlock(
		`expect "split_order"`,
		"scope = dp.split",
		`classification = "uncoupled"`,
		"order = []",
	)

	// When the expect block is decoded.
	_, findings := decodeHierarchyCore(t, source)

	// Then one invalid_structure finding points at the order attribute.
	requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, "order = []"))
}

// TestDecodeHierarchyCouplingAcceptsPositiveIntegerBound covers bounds that are positive integers.
func TestDecodeHierarchyCouplingAcceptsPositiveIntegerBound(t *testing.T) {
	// Given coupling blocks whose iteration bound is written as an integer, an integral decimal, or an integral exponent form.
	cases := []struct {
		name  string
		bound string
		want  string
	}{
		{name: "integer", bound: "1", want: "1"},
		{name: "integral decimal", bound: "1.0", want: "1"},
		{name: "integral exponent form", bound: "1e0", want: "1"},
		{name: "larger integer", bound: "7", want: "7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the coupling block is decoded.
			source := hierarchyBlock(
				`coupling "split_feedback"`,
				"scope = dp.split",
				"affects = [fr.writes]",
				`reason = "The writer and the reader share a cache."`,
				"iteration_bound = "+tc.bound,
			)
			core, findings := decodeHierarchyCore(t, source)
			coupling := hierarchyOnly(t, "coupling", core.Couplings)

			// Then the bound keeps its exact value, with no finding.
			requireHierarchyNoFindings(t, findings)
			requireHierarchyNumber(t, coupling.IterationBound, tc.want)
		})
	}
}

// TestDecodeHierarchyCouplingRejectsNonIntegralOrNonPositiveBound covers bounds that are zero, negative, or fractional.
func TestDecodeHierarchyCouplingRejectsNonIntegralOrNonPositiveBound(t *testing.T) {
	// Given coupling blocks whose iteration bound is zero, negative, or fractional.
	cases := []struct {
		name  string
		bound string
	}{
		{name: "zero", bound: "0"},
		{name: "negative", bound: "-1"},
		{name: "fractional", bound: "1.5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// When the coupling block is decoded.
			source := hierarchyBlock(
				`coupling "split_feedback"`,
				"scope = dp.split",
				"affects = [fr.writes]",
				`reason = "The writer and the reader share a cache."`,
				"iteration_bound = "+tc.bound,
			)
			_, findings := decodeHierarchyCore(t, source)

			// Then one invalid_structure finding points at the iteration_bound attribute.
			requireHierarchyFinding(t, findings, "invalid_structure", hierarchyLine(t, source, "iteration_bound = "+tc.bound))
		})
	}
}
