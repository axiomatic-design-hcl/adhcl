package syntax

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
)

const syntaxTestPath = "syntax-case.ad.hcl"

func parseSyntaxTest(t *testing.T, source string) Unit {
	t.Helper()
	return Parse(model.Source{Path: syntaxTestPath, Bytes: []byte(source)})
}

func findingCodes(findings []result.Finding) []string {
	codes := make([]string, len(findings))
	for index, finding := range findings {
		codes[index] = finding.Code
	}
	return codes
}

func requireFindingCode(t *testing.T, unit Unit, code string) result.Finding {
	t.Helper()
	for _, finding := range unit.Findings {
		if finding.Code == code {
			return finding
		}
	}
	t.Fatalf("no %q finding in %#v", code, findingCodes(unit.Findings))
	return result.Finding{}
}

func requireLocatedFinding(t *testing.T, finding result.Finding, line, startColumn, endColumn int) {
	t.Helper()
	if finding.Range == nil {
		t.Fatal("finding has no source range")
	}
	got := finding.Range
	if got.Filename != syntaxTestPath || got.Start.Line != line || got.Start.Column != startColumn || got.End.Line != line || got.End.Column != endColumn {
		t.Fatalf("finding range = %#v, want %s:%d:%d-%d:%d", got, syntaxTestPath, line, startColumn, line, endColumn)
	}
}

func TestParseAdmitsNativeLiteralValues(t *testing.T) {
	// Given a source with native literal forms, including exact numbers and strings.
	source := `integer = 17
negative = -2.5e3
large = 1e100
unicode = "café 雪"
escaped = "literal $${name} and %%{if text}"
heredoc = <<-TEXT
  first line
    second line
TEXT
list = [true, false, 1e0]
object = { answer = 42, "nested" = ["value"] }
null_value = null
`

	// When the source is parsed.
	unit := parseSyntaxTest(t, source)

	// Then each attribute has its exact native value and parsing has no findings.
	if len(unit.Findings) != 0 {
		t.Fatalf("findings = %#v, want none", findingCodes(unit.Findings))
	}
	want := map[string]cty.Value{
		"integer":    cty.NumberIntVal(17),
		"negative":   cty.MustParseNumberVal("-2500"),
		"large":      cty.MustParseNumberVal("1e100"),
		"unicode":    cty.StringVal("café 雪"),
		"escaped":    cty.StringVal("literal ${name} and %{if text}"),
		"heredoc":    cty.StringVal("first line\n  second line\n"),
		"list":       cty.TupleVal([]cty.Value{cty.True, cty.False, cty.NumberIntVal(1)}),
		"object":     cty.ObjectVal(map[string]cty.Value{"answer": cty.NumberIntVal(42), "nested": cty.TupleVal([]cty.Value{cty.StringVal("value")})}),
		"null_value": cty.NullVal(cty.DynamicPseudoType),
	}
	for name, expected := range want {
		attribute := unit.Body.Attributes[name]
		if attribute == nil {
			t.Fatalf("attribute %q is missing", name)
		}
		actual, diagnostics := attribute.Expr.Value(nil)
		if diagnostics.HasErrors() {
			t.Fatalf("attribute %q did not evaluate as a native literal: %s", name, diagnostics.Error())
		}
		if !actual.RawEquals(expected) {
			t.Errorf("attribute %q = %#v, want exact value %#v", name, actual, expected)
		}
	}
}

func TestParsePreservesCompleteStaticTraversalsAndRanges(t *testing.T) {
	// Given reference expressions with namespace roots and nested identity paths.
	source := "first = dp.book_content\nsecond = fr.build_db.criterion.retention\n"

	// When the source is parsed.
	unit := parseSyntaxTest(t, source)

	// Then the complete static traversal paths and their exact source ranges remain in the AST.
	if len(unit.Findings) != 0 {
		t.Fatalf("findings = %#v, want none", findingCodes(unit.Findings))
	}
	want := []struct {
		name       string
		steps      []string
		line       int
		start, end int
	}{
		{name: "first", steps: []string{"dp", "book_content"}, line: 1, start: 9, end: 24},
		{name: "second", steps: []string{"fr", "build_db", "criterion", "retention"}, line: 2, start: 10, end: 41},
	}
	for _, expected := range want {
		attribute := unit.Body.Attributes[expected.name]
		if attribute == nil {
			t.Fatalf("attribute %q is missing", expected.name)
		}
		traversal, ok := attribute.Expr.(*hclsyntax.ScopeTraversalExpr)
		if !ok {
			t.Fatalf("attribute %q expression has type %T, want static traversal", expected.name, attribute.Expr)
		}
		if len(traversal.Traversal) != len(expected.steps) {
			t.Fatalf("attribute %q traversal has %d steps, want %d", expected.name, len(traversal.Traversal), len(expected.steps))
		}
		for index, step := range traversal.Traversal {
			var actual string
			switch value := step.(type) {
			case hcl.TraverseRoot:
				actual = value.Name
			case hcl.TraverseAttr:
				actual = value.Name
			default:
				t.Fatalf("attribute %q traversal step %d has type %T", expected.name, index, step)
			}
			if actual != expected.steps[index] {
				t.Errorf("attribute %q traversal step %d = %q, want %q", expected.name, index, actual, expected.steps[index])
			}
		}
		rangeValue := traversal.Range()
		if rangeValue.Filename != syntaxTestPath || rangeValue.Start.Line != expected.line || rangeValue.Start.Column != expected.start || rangeValue.End.Line != expected.line || rangeValue.End.Column != expected.end {
			t.Errorf("attribute %q traversal range = %#v, want %s:%d:%d-%d", expected.name, rangeValue, syntaxTestPath, expected.line, expected.start, expected.end)
		}
	}
}

func TestParseRejectsForbiddenExpressions(t *testing.T) {
	// Given expressions that are outside the restricted literal and traversal profile.
	cases := []struct {
		name string
		expr string
	}{
		{name: "call", expr: `value = upper("text")`},
		{name: "interpolation", expr: `value = "${name}"`},
		{name: "directive", expr: `value = "%{if true}yes%{endif}"`},
		{name: "addition", expr: `value = 1 + 2`},
		{name: "comparison", expr: `value = 1 < 2`},
		{name: "conditional", expr: `value = true ? 1 : 2`},
		{name: "list comprehension", expr: `value = [for item in [1] : item]`},
		{name: "object comprehension", expr: `value = {for item in [1] : item => item}`},
		{name: "indexing", expr: `value = [1][0]`},
		{name: "splat", expr: `value = list[*].name`},
		{name: "computed object key", expr: `value = {("key") = 1}`},
		{name: "nested call", expr: `value = [1, upper("x")]`},
		{name: "nested interpolation", expr: `value = {nested = "${name}"}`},
		{name: "nested arithmetic", expr: `value = {nested = [1 + 2]}`},
		{name: "nested conditional", expr: `value = {nested = true ? 1 : 2}`},
		{name: "nested comprehension", expr: `value = {nested = [for item in [1] : item]}`},
		{name: "nested indexing", expr: `value = [ { nested = [1][0] } ]`},
		{name: "nested splat", expr: `value = {nested = list[*].name}`},
		{name: "nested computed object key", expr: `value = {nested = {("key") = 1}}`},
		{name: "nested heredoc interpolation", expr: "value = <<-TEXT\n  %{ if true }yes%{ endif }\nTEXT"},
		{name: "unary minus on traversal", expr: `value = -dp.rate`},
		{name: "unary plus", expr: `value = +1`},
		{name: "multiplication", expr: `value = 2 * 3`},
		{name: "logical operator", expr: `value = true && false`},
		{name: "unary not", expr: `value = !true`},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			// Given one forbidden expression in an otherwise native HCL attribute.
			unit := parseSyntaxTest(t, test.expr+"\n")

			// When syntax admission inspects the expression tree.

			// Then it reports invalid_syntax at an actual source range.
			finding := requireFindingCode(t, unit, "invalid_syntax")
			if finding.Range == nil || finding.Range.Filename != syntaxTestPath || finding.Range.Start.Line < 1 || finding.Range.Start.Column < 1 || finding.Range.End.Line < finding.Range.Start.Line || finding.Range.End.Column < 1 {
				t.Fatalf("finding has no actual source range: %#v", finding.Range)
			}
		})
	}
}

func TestParseRejectsIndexedTraversalsAtTheirSourceRange(t *testing.T) {
	cases := []struct {
		name      string
		source    string
		endColumn int
	}{
		{name: "list index", source: "value = dp.rates[0]\n", endColumn: 20},
		{name: "object key", source: "value = dp.rates[\"key\"]\n", endColumn: 24},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			// Given one reference that indexes a value with a literal key.

			// When the source is parsed.
			unit := parseSyntaxTest(t, test.source)

			// Then exactly one invalid_syntax finding covers the whole reference.
			if codes := findingCodes(unit.Findings); len(codes) != 1 || codes[0] != "invalid_syntax" {
				t.Fatalf("findings = %#v, want one invalid_syntax finding", codes)
			}
			requireLocatedFinding(t, unit.Findings[0], 1, 9, test.endColumn)
		})
	}
}

func TestParseAllowsMinusOnlyDirectlyOnNumericLiterals(t *testing.T) {
	// Given a negative number and expressions that apply minus to a computed value.
	source := "literal = -1.25e2\ncomputed = -(1 + 2)\n"

	// When the source is parsed.
	unit := parseSyntaxTest(t, source)
	// Then the literal is exact and computed negation is rejected at its source range.
	if len(unit.Findings) != 1 || unit.Findings[0].Code != "invalid_syntax" {
		t.Fatalf("findings = %#v, want one invalid_syntax finding", findingCodes(unit.Findings))
	}
	actual, diagnostics := unit.Body.Attributes["literal"].Expr.Value(nil)
	if diagnostics.HasErrors() || !actual.RawEquals(cty.MustParseNumberVal("-125")) {
		t.Fatalf("negative literal = %#v, diagnostics %v, want exact -125", actual, diagnostics)
	}
	requireLocatedFinding(t, unit.Findings[0], 2, 12, 20)
}

func TestParseReportsDuplicateObjectKeysBeforeMapConversion(t *testing.T) {
	// Given an object with the same literal key twice.
	unit := parseSyntaxTest(t, "value = { answer = 1, answer = 2 }\n")

	// When the source is parsed before any object map conversion.

	// Then the duplicate is identified as duplicate_identity at the repeated key's real range.
	finding := requireFindingCode(t, unit, "duplicate_identity")
	requireLocatedFinding(t, finding, 1, 23, 29)
}

func TestParseReportsNativeDuplicateAttributes(t *testing.T) {
	// Given two native attributes with the same name.
	unit := parseSyntaxTest(t, "value = 1\nvalue = 2\n")

	// When the native body is parsed.

	// Then the collision is duplicate_identity and points to the repeated declaration.
	finding := requireFindingCode(t, unit, "duplicate_identity")
	requireLocatedFinding(t, finding, 2, 1, 6)
}

func TestParseKeepsIndependentFindingsFromOneSource(t *testing.T) {
	// Given independent forbidden expressions in separate attributes of one source.
	source := "first = upper(\"x\")\nsecond = 1 + 2\nthird = [for item in [1] : item]\n"

	// When the source is parsed.
	unit := parseSyntaxTest(t, source)

	// Then all three independently located syntax findings remain available.
	if len(unit.Findings) != 3 {
		t.Fatalf("findings = %#v, want three independent findings", findingCodes(unit.Findings))
	}
	for _, finding := range unit.Findings {
		if finding.Code != "invalid_syntax" {
			t.Errorf("finding code = %q, want invalid_syntax", finding.Code)
		}
		if finding.Range == nil || finding.Range.Filename != syntaxTestPath {
			t.Errorf("finding has no actual source location: %#v", finding.Range)
		}
	}
	lines := map[int]bool{}
	for _, finding := range unit.Findings {
		if finding.Range != nil {
			lines[finding.Range.Start.Line] = true
		}
	}
	for _, line := range []int{1, 2, 3} {
		if !lines[line] {
			t.Errorf("no independent finding retained for line %d", line)
		}
	}
}

func TestParseReportsNativeParseErrorsAsInvalidSyntax(t *testing.T) {
	// Given malformed native HCL with a real filename and source position.
	unit := parseSyntaxTest(t, "value = [1,\n")

	// When native parsing fails.

	// Then it reports invalid_syntax at the parser's actual location.
	finding := requireFindingCode(t, unit, "invalid_syntax")
	if finding.Range == nil || finding.Range.Filename != syntaxTestPath || finding.Range.Start.Line < 1 || finding.Range.Start.Column < 1 {
		t.Fatalf("parse finding has no actual file and source position: %#v", finding.Range)
	}
}
