package schema

import (
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

const (
	premisesPath             = "premises-case.ad.hcl"
	premisesInvalidStructure = "invalid_structure"
)

type premisesWant struct {
	code string
	line int
}

type premisesRejectCase struct {
	name   string
	source string
	marker string
}

func premisesDecode(t *testing.T, path, source string) (model.Core, []result.Finding) {
	t.Helper()
	unit := syntax.Parse(model.Source{Path: path, Bytes: []byte(source)})
	if len(unit.Findings) != 0 {
		t.Fatalf("test source has %d native parse findings, want none", len(unit.Findings))
	}
	return DecodeCore([]syntax.Unit{unit})
}

func premisesLine(t *testing.T, source, marker string) int {
	t.Helper()
	for index, line := range strings.Split(source, "\n") {
		if strings.Contains(line, marker) {
			return index + 1
		}
	}
	t.Fatalf("marker %q is not in the test source", marker)
	return 0
}

func premisesRequireNoFindings(t *testing.T, findings []result.Finding) {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("got %d findings, want none", len(findings))
	}
}

// premisesRequireFindings checks the exact finding count, then matches each finding to one
// expected code and start line in the given file. Order does not matter.
func premisesRequireFindings(t *testing.T, path string, findings []result.Finding, wants ...premisesWant) {
	t.Helper()
	if len(findings) != len(wants) {
		t.Fatalf("got %d findings, want %d", len(findings), len(wants))
	}
	unmatched := append([]premisesWant(nil), wants...)
	for _, finding := range findings {
		if finding.Range == nil {
			t.Fatalf("finding %q has no source range", finding.Code)
		}
		if finding.Range.Filename != path {
			t.Fatalf("finding file = %q, want %q", finding.Range.Filename, path)
		}
		matched := false
		for index, want := range unmatched {
			if want.code == finding.Code && want.line == finding.Range.Start.Line {
				unmatched = append(unmatched[:index], unmatched[index+1:]...)
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("unexpected finding %q at line %d", finding.Code, finding.Range.Start.Line)
		}
	}
}

func premisesRequireNumber(t *testing.T, got *big.Float, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("number is unspecified, want %s", want)
	}
	expected, _, err := big.ParseFloat(want, 10, 256, big.ToNearestEven)
	if err != nil {
		t.Fatalf("bad expected number %q: %v", want, err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatalf("number = %s, want %s", got.Text('g', 20), want)
	}
}

func premisesRequireStrings(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("values = %v, want %v", got, want)
	}
}

func premisesTargets(refs []model.Ref) []string {
	targets := make([]string, len(refs))
	for index, ref := range refs {
		targets[index] = string(ref.Target)
	}
	return targets
}

// premisesRejectEach decodes each source, which holds one schema defect, and expects exactly one
// invalid_structure finding on the marker line.
func premisesRejectEach(t *testing.T, cases []premisesRejectCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Given a source with one schema defect.
			// When the source is decoded.
			_, findings := premisesDecode(t, premisesPath, tc.source)
			// Then exactly one invalid_structure finding points at the defect line.
			premisesRequireFindings(t, premisesPath, findings,
				premisesWant{premisesInvalidStructure, premisesLine(t, tc.source, tc.marker)})
		})
	}
}

func premisesCriterionSource(origin string) string {
	return `need "n" {
  statement = "Need."
}
fr "f" {
  statement = "Outcome."
  need = [need.n]
  criterion "c" {
    statement = "Measured outcome."
    origin = "` + origin + `"
  }
}
`
}

func premisesWorkloadWith(rangesLine string) string {
	return `workload "w" {
  description = "Envelope."
  ` + rangesLine + `
}
`
}

func TestDecodePremisesAcceptsLabelGrammar(t *testing.T) {
	// Given source blocks whose labels match the snake_case grammar.
	labels := []string{"catalog", "build_db", "a1", "v2_rev3"}
	for _, label := range labels {
		label := label
		t.Run(label, func(t *testing.T) {
			// Given a source block with a grammar-valid label.
			source := "source \"" + label + "\" {\n  title = \"Catalog study.\"\n}\n"
			// When the block is decoded.
			core, findings := premisesDecode(t, premisesPath, source)
			// Then the block decodes with no findings and keeps its identity.
			premisesRequireNoFindings(t, findings)
			if len(core.Sources) != 1 {
				t.Fatalf("got %d sources, want 1", len(core.Sources))
			}
			if string(core.Sources[0].ID) != "source."+label {
				t.Fatalf("source ID = %q, want %q", core.Sources[0].ID, "source."+label)
			}
		})
	}
}

func TestDecodePremisesRejectsInvalidLabelGrammar(t *testing.T) {
	// Given source blocks whose labels break the snake_case grammar.
	labels := []string{"Catalog", "1catalog", "build__db", "build_", "_build", "build-db", "build db", ""}
	cases := make([]premisesRejectCase, 0, len(labels))
	for _, label := range labels {
		cases = append(cases, premisesRejectCase{
			name:   "label " + label,
			source: "source \"" + label + "\" {\n  title = \"Catalog study.\"\n}\n",
			marker: "source \"",
		})
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the header line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesKeepsSameLabelAcrossBlockTypes(t *testing.T) {
	// Given a source, a functional requirement, and a design parameter that all use the label catalog.
	source := `need "n" {
  statement = "Need."
}
fr "catalog" {
  statement = "Catalog outcome."
  need = [need.n]
}
dp "catalog" {
  controls = [fr.catalog]
  responsibility = "Owns the catalog."
  decision = "proposed"
}
source "catalog" {
  title = "Catalog study."
}
`
	// When the core is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then each block keeps its own namespace identity with no findings.
	premisesRequireNoFindings(t, findings)
	if len(core.Sources) != 1 || string(core.Sources[0].ID) != "source.catalog" {
		t.Fatalf("sources = %+v, want one source.catalog", core.Sources)
	}
	if len(core.FRs) != 1 || string(core.FRs[0].ID) != "fr.catalog" {
		t.Fatalf("FRs = %+v, want one fr.catalog", core.FRs)
	}
	if len(core.DPs) != 1 || string(core.DPs[0].ID) != "dp.catalog" {
		t.Fatalf("DPs = %+v, want one dp.catalog", core.DPs)
	}
}

func TestDecodePremisesRejectsWrongLabelCount(t *testing.T) {
	// Given a named source block with no label and one with two labels.
	cases := []premisesRejectCase{
		{name: "no label", source: "source {\n  title = \"Catalog study.\"\n}\n", marker: "source {"},
		{name: "two labels", source: "source \"a\" \"b\" {\n  title = \"Catalog study.\"\n}\n", marker: "source \"a\" \"b\""},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the header line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesRejectsUnknownBlock(t *testing.T) {
	// Given a block type that is not part of the core schema.
	source := "widget \"w\" {\n  title = \"Catalog study.\"\n}\n"
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the unknown block.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "widget")})
}

func TestDecodePremisesRejectsUnknownAttribute(t *testing.T) {
	// Given a need block with a field that the schema does not list.
	source := `need "n" {
  statement = "Retain an unknown field as an error."
  ignored_typo = true
}
`
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the unknown field.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "ignored_typo")})
}

func TestDecodePremisesRejectsTopLevelAttribute(t *testing.T) {
	// Given a file with an attribute outside every block.
	source := "version = 1\nsource \"a\" {\n  title = \"Catalog study.\"\n}\n"
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the top-level attribute.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "version = 1")})
}

func TestDecodePremisesRejectsChildBlockInLeafBlock(t *testing.T) {
	// Given a source block that contains a child need block.
	source := `source "a" {
  title = "Catalog study."
  need "n" {
    statement = "Nested need."
  }
}
`
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the child block.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "need \"n\"")})
}

func TestDecodePremisesRejectsRemovedImportBlock(t *testing.T) {
	// Given a removed import block.
	source := "import \"shared\" {\n  path = \"shared.ad.hcl\"\n}\n"
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the import block.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "import \"shared\"")})
}

func TestDecodePremisesRejectsRemovedModuleHandles(t *testing.T) {
	// Given a valid realization module that still declares the removed handles field.
	source := `need "n" {
  statement = "Need."
}
fr "f" {
  statement = "Outcome."
  need = [need.n]
}
dp "d" {
  controls = [fr.f]
  responsibility = "Owns the outcome."
  decision = "proposed"
}
module "m" {
  realizes = [dp.d]
  handles = ["request"]
}
`
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the handles field.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "handles = ")})
}

func TestDecodePremisesRejectsAuthoredNull(t *testing.T) {
	// Given null in a required string, an optional string, a reference list, a cites list, and a scalar reference.
	cases := []premisesRejectCase{
		{name: "required string", source: "source \"a\" {\n  title = null\n}\n", marker: "title = null"},
		{name: "optional string", source: "source \"a\" {\n  title = \"Catalog study.\"\n  doi = null\n}\n", marker: "doi = null"},
		{name: "reference list element", source: "context \"c\" {\n  description = \"d\"\n  assumes = [assumption.a, null]\n}\n", marker: "assumes = ["},
		{name: "cites list element", source: "need \"n\" {\n  statement = \"s\"\n  cites = [null]\n}\n", marker: "cites = [null]"},
		{name: "scalar reference", source: "need \"n\" {\n  statement = \"s\"\n  stakeholder = null\n}\n", marker: "stakeholder = null"},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the null line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesRejectsBlankStrings(t *testing.T) {
	// Given required and optional strings that are empty or only whitespace.
	cases := []premisesRejectCase{
		{name: "empty required", source: "source \"a\" {\n  title = \"\"\n}\n", marker: "title = \"\""},
		{name: "blank required", source: "source \"a\" {\n  title = \"   \"\n}\n", marker: "title = \"   \""},
		{name: "blank optional", source: "source \"a\" {\n  title = \"Catalog study.\"\n  doi = \" \"\n}\n", marker: "doi = \" \""},
		{name: "empty constraint kind", source: "constraint \"c\" {\n  statement = \"s\"\n  kind = \"\"\n}\n", marker: "kind = \"\""},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the blank field.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesKeepsStringsExact(t *testing.T) {
	// Given a quoted title with surrounding spaces and a heredoc title.
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{name: "quoted with spaces", source: "source \"a\" {\n  title = \"  Spaced title  \"\n}\n", want: "  Spaced title  "},
		{name: "heredoc", source: "source \"a\" {\n  title = <<TEXT\nHeredoc title\nTEXT\n}\n", want: "Heredoc title\n"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the source is decoded.
			core, findings := premisesDecode(t, premisesPath, tc.source)
			// Then the title keeps the authored text byte for byte with no findings.
			premisesRequireNoFindings(t, findings)
			if len(core.Sources) != 1 || core.Sources[0].Title != tc.want {
				t.Fatalf("title = %q, want %q", core.Sources[0].Title, tc.want)
			}
		})
	}
}

func TestDecodePremisesRejectsWrongScalarTypes(t *testing.T) {
	// Given string fields that receive numbers, booleans, or lists.
	cases := []premisesRejectCase{
		{name: "number as string", source: "source \"a\" {\n  title = 1\n}\n", marker: "title = 1"},
		{name: "boolean as string", source: "source \"a\" {\n  title = true\n}\n", marker: "title = true"},
		{name: "list as string", source: "source \"a\" {\n  title = [\"a\"]\n}\n", marker: "title = [\"a\"]"},
		{name: "boolean as need statement", source: "need \"n\" {\n  statement = true\n}\n", marker: "statement = true"},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the field line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesRejectsStringWhereReferenceRequired(t *testing.T) {
	// Given reference fields that receive a quoted string that looks like a reference.
	cases := []premisesRejectCase{
		{name: "need stakeholder", source: "need \"n\" {\n  statement = \"s\"\n  stakeholder = \"stakeholder.ops\"\n}\n", marker: "stakeholder = \"stakeholder.ops\""},
		{name: "design workload", source: "design \"d\" {\n  boundary = \"b\"\n  workload = \"workload.w\"\n}\n", marker: "workload = \"workload.w\""},
		{name: "context assumes", source: "context \"c\" {\n  description = \"d\"\n  assumes = [\"assumption.a\"]\n}\n", marker: "assumes = [\"assumption.a\"]"},
		{name: "cites", source: "need \"n\" {\n  statement = \"s\"\n  cites = \"source.a\"\n}\n", marker: "cites = \"source.a\""},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the field line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesRejectsReferenceWhereStringRequired(t *testing.T) {
	// Given string fields that receive a static reference traversal.
	cases := []premisesRejectCase{
		{name: "source title", source: "source \"a\" {\n  title = source.b\n}\n", marker: "title = source.b"},
		{name: "design boundary", source: "design \"d\" {\n  boundary = workload.w\n}\n", marker: "boundary = workload.w"},
		{name: "stakeholder role", source: "stakeholder \"s\" {\n  role = stakeholder.t\n}\n", marker: "role = stakeholder.t"},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the field line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesRejectsScalarWhereListRequired(t *testing.T) {
	// Given reference list fields that receive a single reference.
	cases := []premisesRejectCase{
		{name: "assumes scalar", source: "context \"c\" {\n  description = \"d\"\n  assumes = assumption.a\n}\n", marker: "assumes = assumption.a"},
		{name: "excludes scalar", source: "context \"c\" {\n  description = \"d\"\n  excludes = context.b\n}\n", marker: "excludes = context.b"},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the field line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesAcceptsCitesAsScalarOrSingleItemList(t *testing.T) {
	// Given the same citation written as a scalar and as a one-item list.
	cases := []struct {
		name   string
		source string
	}{
		{name: "scalar", source: "need \"n\" {\n  statement = \"s\"\n  cites = source.a\n}\n"},
		{name: "one item list", source: "need \"n\" {\n  statement = \"s\"\n  cites = [source.a]\n}\n"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the source is decoded.
			core, findings := premisesDecode(t, premisesPath, tc.source)
			// Then both forms decode to the same single citation with its own range.
			premisesRequireNoFindings(t, findings)
			if len(core.Needs) != 1 || len(core.Needs[0].Cites) != 1 {
				t.Fatalf("needs = %+v, want one need with one citation", core.Needs)
			}
			cite := core.Needs[0].Cites[0]
			if string(cite.Target) != "source.a" {
				t.Fatalf("citation target = %q, want source.a", cite.Target)
			}
			if cite.Range.Start.Line != premisesLine(t, tc.source, "cites = ") {
				t.Fatalf("citation line = %d, want %d", cite.Range.Start.Line, premisesLine(t, tc.source, "cites = "))
			}
		})
	}
}

func TestDecodePremisesKeepsCitesListOrder(t *testing.T) {
	// Given a need that cites two sources in a deliberate order.
	source := "need \"n\" {\n  statement = \"s\"\n  cites = [source.b, source.a]\n}\n"
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then the citations keep the authored order.
	premisesRequireNoFindings(t, findings)
	if len(core.Needs) != 1 {
		t.Fatalf("got %d needs, want 1", len(core.Needs))
	}
	premisesRequireStrings(t, premisesTargets(core.Needs[0].Cites), []string{"source.b", "source.a"})
}

func TestDecodePremisesRejectsEmptyCites(t *testing.T) {
	// Given a need with an explicit empty citation list.
	source := "need \"n\" {\n  statement = \"s\"\n  cites = []\n}\n"
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the cites field.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "cites = []")})
}

func TestDecodePremisesRejectsRepeatedReferences(t *testing.T) {
	// Given reference lists that repeat one identity.
	cases := []premisesRejectCase{
		{name: "cites", source: "need \"n\" {\n  statement = \"s\"\n  cites = [source.a, source.a]\n}\n", marker: "cites = [source.a, source.a]"},
		{name: "assumes", source: "context \"c\" {\n  description = \"d\"\n  assumes = [assumption.a, assumption.a]\n}\n", marker: "assumes = [assumption.a, assumption.a]"},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the list line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesRejectsCitesOnSource(t *testing.T) {
	// Given a source block, which does not admit citations.
	source := "source \"a\" {\n  title = \"Catalog study.\"\n  cites = [source.b]\n}\n"
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the cites field.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "cites = [source.b]")})
}

func TestDecodePremisesAcceptsEachOriginOnNeed(t *testing.T) {
	// Given a need with each origin value that the schema admits on a need.
	cases := []struct {
		name string
		want model.Origin
	}{
		{name: "user", want: model.OriginUser},
		{name: "source", want: model.OriginSource},
		{name: "proposed", want: model.OriginProposed},
		{name: "inferred", want: model.OriginInferred},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			source := "need \"n\" {\n  statement = \"s\"\n  origin = \"" + string(tc.want) + "\"\n}\n"
			// When the source is decoded.
			core, findings := premisesDecode(t, premisesPath, source)
			// Then the origin decodes to the authored value with no findings.
			premisesRequireNoFindings(t, findings)
			if len(core.Needs) != 1 || core.Needs[0].Origin == nil || *core.Needs[0].Origin != tc.want {
				t.Fatalf("origin = %+v, want %q", core.Needs, tc.want)
			}
		})
	}
}

func TestDecodePremisesLeavesOmittedOriginUnspecified(t *testing.T) {
	// Given a need with no origin field.
	source := "need \"n\" {\n  statement = \"s\"\n}\n"
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then the origin stays unspecified rather than receiving a value.
	premisesRequireNoFindings(t, findings)
	if len(core.Needs) != 1 || core.Needs[0].Origin != nil {
		t.Fatalf("origin = %+v, want unspecified", core.Needs)
	}
}

func TestDecodePremisesRejectsUnlistedOrigin(t *testing.T) {
	// Given origin values that are not in the enum, including wrong case and an empty string.
	values := []string{"validated", "User", "", "unknown"}
	cases := make([]premisesRejectCase, 0, len(values))
	for _, value := range values {
		cases = append(cases, premisesRejectCase{
			name:   "origin " + value,
			source: "need \"n\" {\n  statement = \"s\"\n  origin = \"" + value + "\"\n}\n",
			marker: "origin = ",
		})
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the origin line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesAcceptsCriterionOriginSubset(t *testing.T) {
	// Given a criterion with each origin that the criterion block admits.
	cases := []struct {
		name string
		want model.Origin
	}{
		{name: "user", want: model.OriginUser},
		{name: "proposed", want: model.OriginProposed},
		{name: "source", want: model.OriginSource},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the source is decoded.
			core, findings := premisesDecode(t, premisesPath, premisesCriterionSource(string(tc.want)))
			// Then the criterion keeps the authored origin with no findings.
			premisesRequireNoFindings(t, findings)
			if len(core.FRs) != 1 || len(core.FRs[0].Criteria) != 1 {
				t.Fatalf("FRs = %+v, want one FR with one criterion", core.FRs)
			}
			got := core.FRs[0].Criteria[0].Origin
			if got == nil || *got != tc.want {
				t.Fatalf("criterion origin = %+v, want %q", got, tc.want)
			}
		})
	}
}

func TestDecodePremisesRejectsCriterionInferredOrigin(t *testing.T) {
	// Given a criterion whose origin is inferred, which the criterion block does not admit.
	source := premisesCriterionSource("inferred")
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the origin line.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "origin = ")})
}

func TestDecodePremisesRejectsOriginOnSource(t *testing.T) {
	// Given a source block with an origin field, which source blocks do not admit.
	source := "source \"a\" {\n  title = \"Catalog study.\"\n  origin = \"user\"\n}\n"
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the origin field.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "origin = ")})
}

func TestDecodePremisesAcceptsAliasOnFRDPPVModule(t *testing.T) {
	// Given a valid functional requirement, design parameter, process variable, and module, each with an alias.
	cases := []struct {
		name   string
		source string
		alias  func(core model.Core) *string
		want   string
	}{
		{
			name: "fr",
			source: `need "n" {
  statement = "Need."
}
fr "f" {
  statement = "Outcome."
  need = [need.n]
  alias = "FR11"
}
`,
			alias: func(core model.Core) *string { return core.FRs[0].Alias },
			want:  "FR11",
		},
		{
			name: "dp",
			source: `dp "d" {
  controls = [fr.f]
  responsibility = "Owns the outcome."
  decision = "proposed"
  alias = "DP1"
}
`,
			alias: func(core model.Core) *string { return core.DPs[0].Alias },
			want:  "DP1",
		},
		{
			name: "pv",
			source: `pv "p" {
  realizes = [dp.d]
  alias = "PV1"
}
`,
			alias: func(core model.Core) *string { return core.PVs[0].Alias },
			want:  "PV1",
		},
		{
			name: "module",
			source: `module "m" {
  realizes = [dp.d]
  alias = "M1"
}
`,
			alias: func(core model.Core) *string { return core.Modules[0].Alias },
			want:  "M1",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the source is decoded.
			core, findings := premisesDecode(t, premisesPath, tc.source)
			// Then the alias decodes to the authored text with no findings.
			premisesRequireNoFindings(t, findings)
			got := tc.alias(core)
			if got == nil || *got != tc.want {
				t.Fatalf("alias = %+v, want %q", got, tc.want)
			}
		})
	}
}

func TestDecodePremisesKeepsAliasTextAndAllowsRepeats(t *testing.T) {
	// Given two functional requirements that share one alias with spaces and punctuation.
	source := `need "n" {
  statement = "Need."
}
fr "a" {
  statement = "First."
  need = [need.n]
  alias = "FR 11 / legacy!"
}
fr "b" {
  statement = "Second."
  need = [need.n]
  alias = "FR 11 / legacy!"
}
`
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then both aliases keep the exact text with no findings.
	premisesRequireNoFindings(t, findings)
	if len(core.FRs) != 2 {
		t.Fatalf("got %d FRs, want 2", len(core.FRs))
	}
	for index, fr := range core.FRs {
		if fr.Alias == nil || *fr.Alias != "FR 11 / legacy!" {
			t.Fatalf("FR %d alias = %+v, want the exact authored text", index, fr.Alias)
		}
	}
}

func TestDecodePremisesRejectsAliasOnOtherBlocks(t *testing.T) {
	// Given each premise block type with an alias field, which only fr, dp, pv, and module admit.
	cases := []premisesRejectCase{
		{name: "source", source: "source \"a\" {\n  title = \"t\"\n  alias = \"x\"\n}\n", marker: "alias = "},
		{name: "design", source: "design \"root\" {\n  boundary = \"b\"\n  alias = \"x\"\n}\n", marker: "alias = "},
		{name: "stakeholder", source: "stakeholder \"ops\" {\n  role = \"r\"\n  alias = \"x\"\n}\n", marker: "alias = "},
		{name: "need", source: "need \"n\" {\n  statement = \"s\"\n  alias = \"x\"\n}\n", marker: "alias = "},
		{name: "constraint", source: "constraint \"c\" {\n  statement = \"s\"\n  alias = \"x\"\n}\n", marker: "alias = "},
		{name: "assumption", source: "assumption \"a\" {\n  statement = \"s\"\n  alias = \"x\"\n}\n", marker: "alias = "},
		{name: "context", source: "context \"c\" {\n  description = \"d\"\n  alias = \"x\"\n}\n", marker: "alias = "},
		{name: "workload", source: "workload \"w\" {\n  description = \"d\"\n  alias = \"x\"\n}\n", marker: "alias = "},
		{name: "state", source: "state \"s\" {\n  description = \"d\"\n  alias = \"x\"\n}\n", marker: "alias = "},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the alias line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesDefaultsAssumptionStatusToValidate(t *testing.T) {
	// Given an assumption with no status field.
	source := "assumption \"a\" {\n  statement = \"Rate stays flat.\"\n}\n"
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then the status takes the default to_validate value.
	premisesRequireNoFindings(t, findings)
	if len(core.Assumptions) != 1 || core.Assumptions[0].Status == nil || *core.Assumptions[0].Status != model.AssumptionToValidate {
		t.Fatalf("assumptions = %+v, want status to_validate", core.Assumptions)
	}
}

func TestDecodePremisesAcceptsAssumptionStatusValues(t *testing.T) {
	// Given an assumption with each status value that the schema admits.
	cases := []struct {
		name string
		want model.AssumptionStatus
	}{
		{name: "assumed", want: model.AssumptionAssumed},
		{name: "to_validate", want: model.AssumptionToValidate},
		{name: "validated", want: model.AssumptionValidated},
		{name: "invalidated", want: model.AssumptionInvalidated},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			source := "assumption \"a\" {\n  statement = \"Rate stays flat.\"\n  status = \"" + tc.name + "\"\n}\n"
			// When the source is decoded.
			core, findings := premisesDecode(t, premisesPath, source)
			// Then the status decodes to the authored value with no findings.
			premisesRequireNoFindings(t, findings)
			if len(core.Assumptions) != 1 || core.Assumptions[0].Status == nil || *core.Assumptions[0].Status != tc.want {
				t.Fatalf("assumptions = %+v, want status %q", core.Assumptions, tc.want)
			}
		})
	}
}

func TestDecodePremisesRejectsUnlistedAssumptionStatus(t *testing.T) {
	// Given assumption status values outside the enum, including wrong case and an empty string.
	values := []string{"approved", "Validated", ""}
	cases := make([]premisesRejectCase, 0, len(values))
	for _, value := range values {
		cases = append(cases, premisesRejectCase{
			name:   "status " + value,
			source: "assumption \"a\" {\n  statement = \"s\"\n  status = \"" + value + "\"\n}\n",
			marker: "status = ",
		})
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the status line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesDefaultsContextListsToEmpty(t *testing.T) {
	// Given a context with no assumes, excludes, or workload fields.
	source := "context \"c\" {\n  description = \"d\"\n}\n"
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then both reference lists are empty and the workload stays unspecified.
	premisesRequireNoFindings(t, findings)
	if len(core.Contexts) != 1 {
		t.Fatalf("got %d contexts, want 1", len(core.Contexts))
	}
	context := core.Contexts[0]
	if len(context.Assumes) != 0 || len(context.Excludes) != 0 || context.Workload != nil {
		t.Fatalf("context = %+v, want empty lists and no workload", context)
	}
}

func TestDecodePremisesKeepsContextReferenceLists(t *testing.T) {
	// Given a context with ordered assumes, excludes, and a workload reference.
	source := `assumption "a" {
  statement = "First."
}
assumption "b" {
  statement = "Second."
}
workload "w" {
  description = "Envelope."
}
context "c" {
  description = "Operating context."
  assumes = [assumption.b, assumption.a]
  excludes = [context.d]
  workload = workload.w
}
context "d" {
  description = "Excluded context."
}
`
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then the lists keep their authored order and the workload keeps its target.
	premisesRequireNoFindings(t, findings)
	if len(core.Contexts) != 2 || string(core.Contexts[0].ID) != "context.c" {
		t.Fatalf("contexts = %+v, want context.c first", core.Contexts)
	}
	context := core.Contexts[0]
	premisesRequireStrings(t, premisesTargets(context.Assumes), []string{"assumption.b", "assumption.a"})
	premisesRequireStrings(t, premisesTargets(context.Excludes), []string{"context.d"})
	if context.Workload == nil || string(context.Workload.Target) != "workload.w" {
		t.Fatalf("workload = %+v, want workload.w", context.Workload)
	}
}

func TestDecodePremisesRejectsSelfExcludingContext(t *testing.T) {
	// Given a context that excludes itself.
	source := "context \"a\" {\n  description = \"d\"\n  excludes = [context.a]\n}\n"
	// When the source is decoded.
	_, findings := premisesDecode(t, premisesPath, source)
	// Then one invalid_structure finding points at the excludes field.
	premisesRequireFindings(t, premisesPath, findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "excludes = [context.a]")})
}

func TestDecodePremisesDecodesWorkloadRangeMap(t *testing.T) {
	// Given a workload with two range dimensions, one with a unit and one without.
	source := `workload "traffic" {
  description = "Traffic envelope."
  ranges = {
    request_rate = { min = 0, max = 100, unit = "requests/s" }
    batch_size = { min = 1, max = 50 }
  }
}
`
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then the dimensions keep their authored order, endpoints, and units.
	premisesRequireNoFindings(t, findings)
	if len(core.Workloads) != 1 {
		t.Fatalf("got %d workloads, want 1", len(core.Workloads))
	}
	ranges := core.Workloads[0].Ranges
	if !ranges.Present || len(ranges.Items) != 2 {
		t.Fatalf("ranges = %+v, want two dimensions", ranges)
	}
	first := ranges.Items[0]
	if first.Name != "request_rate" {
		t.Fatalf("first dimension = %q, want request_rate", first.Name)
	}
	premisesRequireNumber(t, first.Min, "0")
	premisesRequireNumber(t, first.Max, "100")
	if first.Unit == nil || *first.Unit != "requests/s" {
		t.Fatalf("first unit = %+v, want requests/s", first.Unit)
	}
	second := ranges.Items[1]
	if second.Name != "batch_size" {
		t.Fatalf("second dimension = %q, want batch_size", second.Name)
	}
	premisesRequireNumber(t, second.Min, "1")
	premisesRequireNumber(t, second.Max, "50")
	if second.Unit != nil {
		t.Fatalf("second unit = %+v, want unspecified", second.Unit)
	}
}

func TestDecodePremisesKeepsAbsentAndEmptyWorkloadRangesApart(t *testing.T) {
	// Given a workload with no ranges field and a workload with an empty ranges object.
	cases := []struct {
		name    string
		source  string
		present bool
	}{
		{name: "absent", source: "workload \"w\" {\n  description = \"d\"\n}\n", present: false},
		{name: "empty object", source: "workload \"w\" {\n  description = \"d\"\n  ranges = {}\n}\n", present: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the source is decoded.
			core, findings := premisesDecode(t, premisesPath, tc.source)
			// Then presence matches the authored form with no dimensions.
			premisesRequireNoFindings(t, findings)
			if len(core.Workloads) != 1 {
				t.Fatalf("got %d workloads, want 1", len(core.Workloads))
			}
			ranges := core.Workloads[0].Ranges
			if ranges.Present != tc.present || len(ranges.Items) != 0 {
				t.Fatalf("ranges = %+v, want present %v with no dimensions", ranges, tc.present)
			}
		})
	}
}

func TestDecodePremisesAcceptsEqualAndNegativeWorkloadEndpoints(t *testing.T) {
	// Given a range whose endpoints are equal and negative.
	source := premisesWorkloadWith("ranges = { offset = { min = -5, max = -5 } }")
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then the inclusive endpoints are accepted and keep their exact values.
	premisesRequireNoFindings(t, findings)
	if len(core.Workloads) != 1 || len(core.Workloads[0].Ranges.Items) != 1 {
		t.Fatalf("workloads = %+v, want one dimension", core.Workloads)
	}
	dimension := core.Workloads[0].Ranges.Items[0]
	premisesRequireNumber(t, dimension.Min, "-5")
	premisesRequireNumber(t, dimension.Max, "-5")
}

func TestDecodePremisesRejectsInvalidWorkloadRanges(t *testing.T) {
	// Given range maps that break the endpoint, key, or unit rules.
	lines := []struct {
		name string
		line string
	}{
		{name: "min above max", line: "ranges = { rate = { min = 10, max = 1 } }"},
		{name: "extra key", line: "ranges = { rate = { min = 0, max = 1, step = 1 } }"},
		{name: "null min", line: "ranges = { rate = { min = null, max = 1 } }"},
		{name: "null max", line: "ranges = { rate = { min = 0, max = null } }"},
		{name: "missing max", line: "ranges = { rate = { min = 0 } }"},
		{name: "string endpoint", line: "ranges = { rate = { min = \"0\", max = 1 } }"},
		{name: "blank unit", line: "ranges = { rate = { min = 0, max = 1, unit = \"\" } }"},
		{name: "blank key", line: "ranges = { \"\" = { min = 0, max = 1 } }"},
		{name: "list instead of map", line: "ranges = [{ min = 0, max = 1 }]"},
	}
	cases := make([]premisesRejectCase, 0, len(lines))
	for _, item := range lines {
		cases = append(cases, premisesRejectCase{
			name:   item.name,
			source: premisesWorkloadWith(item.line),
			marker: "ranges = ",
		})
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the ranges line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesDecodesSourceFields(t *testing.T) {
	// Given a source with every admitted field.
	source := `source "catalog" {
  title = "Catalog study"
  doi = "10.1000/xyz"
  url = "https://example.test/catalog"
  pages = "15-19"
}
`
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then every field keeps its authored value.
	premisesRequireNoFindings(t, findings)
	if len(core.Sources) != 1 {
		t.Fatalf("got %d sources, want 1", len(core.Sources))
	}
	got := core.Sources[0]
	if string(got.ID) != "source.catalog" || got.Title != "Catalog study" {
		t.Fatalf("source = %q %q, want source.catalog and Catalog study", got.ID, got.Title)
	}
	if got.DOI == nil || *got.DOI != "10.1000/xyz" {
		t.Fatalf("doi = %+v, want 10.1000/xyz", got.DOI)
	}
	if got.URL == nil || *got.URL != "https://example.test/catalog" {
		t.Fatalf("url = %+v, want the authored URL", got.URL)
	}
	if got.Pages == nil || *got.Pages != "15-19" {
		t.Fatalf("pages = %+v, want 15-19", got.Pages)
	}
}

func TestDecodePremisesDecodesDesignRecord(t *testing.T) {
	// Given a design with a level and a workload reference.
	source := `workload "traffic" {
  description = "Traffic envelope."
}
design "root" {
  boundary = "Order service."
  level = "system"
  workload = workload.traffic
}
`
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then the design keeps its boundary, level, and workload target.
	premisesRequireNoFindings(t, findings)
	if len(core.Designs) != 1 {
		t.Fatalf("got %d designs, want 1", len(core.Designs))
	}
	got := core.Designs[0]
	if string(got.ID) != "design.root" || got.Boundary != "Order service." {
		t.Fatalf("design = %q %q, want design.root and Order service.", got.ID, got.Boundary)
	}
	if got.Level == nil || *got.Level != "system" {
		t.Fatalf("level = %+v, want system", got.Level)
	}
	if got.Workload == nil || string(got.Workload.Target) != "workload.traffic" {
		t.Fatalf("workload = %+v, want workload.traffic", got.Workload)
	}
}

func TestDecodePremisesDecodesNeedRecord(t *testing.T) {
	// Given a need with a rationale and a stakeholder reference.
	source := `stakeholder "ops" {
  role = "Operator"
}
need "uptime" {
  statement = "Keep orders flowing."
  why = "Revenue depends on it."
  stakeholder = stakeholder.ops
}
`
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then the need keeps its statement, rationale, and stakeholder target.
	premisesRequireNoFindings(t, findings)
	if len(core.Needs) != 1 {
		t.Fatalf("got %d needs, want 1", len(core.Needs))
	}
	got := core.Needs[0]
	if string(got.ID) != "need.uptime" || got.Statement != "Keep orders flowing." {
		t.Fatalf("need = %q %q, want need.uptime and the authored statement", got.ID, got.Statement)
	}
	if got.Why == nil || *got.Why != "Revenue depends on it." {
		t.Fatalf("why = %+v, want the authored rationale", got.Why)
	}
	if got.Stakeholder == nil || string(got.Stakeholder.Target) != "stakeholder.ops" {
		t.Fatalf("stakeholder = %+v, want stakeholder.ops", got.Stakeholder)
	}
}

func TestDecodePremisesDecodesStakeholderConstraintAndState(t *testing.T) {
	// Given a stakeholder, a constraint with a kind, and a state.
	source := `stakeholder "ops" {
  role = "Operator"
}
constraint "budget" {
  statement = "Fixed budget."
  kind = "cost"
}
state "queue" {
  description = "Pending orders."
}
`
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then each record keeps its authored fields.
	premisesRequireNoFindings(t, findings)
	if len(core.Stakeholders) != 1 || string(core.Stakeholders[0].ID) != "stakeholder.ops" || core.Stakeholders[0].Role != "Operator" {
		t.Fatalf("stakeholders = %+v, want stakeholder.ops with role Operator", core.Stakeholders)
	}
	if len(core.Constraints) != 1 || string(core.Constraints[0].ID) != "constraint.budget" || core.Constraints[0].Statement != "Fixed budget." {
		t.Fatalf("constraints = %+v, want constraint.budget with the authored statement", core.Constraints)
	}
	if core.Constraints[0].Kind == nil || *core.Constraints[0].Kind != "cost" {
		t.Fatalf("constraint kind = %+v, want cost", core.Constraints[0].Kind)
	}
	if len(core.States) != 1 || string(core.States[0].ID) != "state.queue" || core.States[0].Description != "Pending orders." {
		t.Fatalf("states = %+v, want state.queue with the authored description", core.States)
	}
}

func TestDecodePremisesKeepsOmittedOptionalFieldsAbsent(t *testing.T) {
	// Given minimal source, design, need, and constraint blocks with no optional fields.
	source := `source "a" {
  title = "t"
}
design "root" {
  boundary = "b"
}
need "n" {
  statement = "s"
}
constraint "c" {
  statement = "s"
}
`
	// When the source is decoded.
	core, findings := premisesDecode(t, premisesPath, source)
	// Then every omitted optional field stays absent rather than receiving a value.
	premisesRequireNoFindings(t, findings)
	if len(core.Sources) != 1 || core.Sources[0].DOI != nil || core.Sources[0].URL != nil || core.Sources[0].Pages != nil {
		t.Fatalf("source = %+v, want doi, url, and pages absent", core.Sources)
	}
	if len(core.Designs) != 1 || core.Designs[0].Level != nil || core.Designs[0].Workload != nil {
		t.Fatalf("design = %+v, want level and workload absent", core.Designs)
	}
	if len(core.Needs) != 1 || core.Needs[0].Why != nil || core.Needs[0].Stakeholder != nil {
		t.Fatalf("need = %+v, want why and stakeholder absent", core.Needs)
	}
	if len(core.Constraints) != 1 || core.Constraints[0].Kind != nil {
		t.Fatalf("constraint = %+v, want kind absent", core.Constraints)
	}
}

func TestDecodePremisesRejectsMissingRequiredFields(t *testing.T) {
	// Given one block per premise type, each missing its required field.
	cases := []premisesRejectCase{
		{name: "source without title", source: "source \"a\" {\n  doi = \"10.1000/xyz\"\n}\n", marker: "source \"a\""},
		{name: "design without boundary", source: "design \"root\" {\n  level = \"system\"\n}\n", marker: "design \"root\""},
		{name: "stakeholder without role", source: "stakeholder \"ops\" {\n}\n", marker: "stakeholder \"ops\""},
		{name: "need without statement", source: "need \"n\" {\n  why = \"w\"\n}\n", marker: "need \"n\""},
		{name: "constraint without statement", source: "constraint \"c\" {\n  kind = \"k\"\n}\n", marker: "constraint \"c\""},
		{name: "assumption without statement", source: "assumption \"a\" {\n}\n", marker: "assumption \"a\""},
		{name: "context without description", source: "context \"c\" {\n}\n", marker: "context \"c\""},
		{name: "workload without description", source: "workload \"w\" {\n}\n", marker: "workload \"w\""},
		{name: "state without description", source: "state \"s\" {\n}\n", marker: "state \"s\""},
	}
	// When each source is decoded, and // Then each yields one invalid_structure at the block header line.
	premisesRejectEach(t, cases)
}

func TestDecodePremisesFixtureReportsFourSchemaDefects(t *testing.T) {
	// Given the pinned core-1 invalid-fields fixture with four schema defects.
	bytes, err := os.ReadFile("../../testdata/spec/conformance/core-1/invalid-fields.ad.hcl")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	source := string(bytes)
	// When the fixture is decoded under its conformance name.
	_, findings := premisesDecode(t, "invalid-fields.ad.hcl", source)
	// Then exactly four invalid_structure findings point at the defective fields in their blocks.
	premisesRequireFindings(t, "invalid-fields.ad.hcl", findings,
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "title = null")},
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "ignored_typo = true")},
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "controls = []")},
		premisesWant{premisesInvalidStructure, premisesLine(t, source, "verifies = [")},
	)
}
