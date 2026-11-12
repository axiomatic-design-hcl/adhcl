package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

const resolveFixtureDir = "../../testdata/spec/conformance/core-1"

const resolveManifestSource = `project "checkout" {
  language         = "requirements_hcl/core/1"
  core_files       = ["base.ad.hcl"]
  attachment_files = ["attachments.ad.hcl"]
}

extension "terraform_binding" {
  role     = "binding"
  contract = "requirements_hcl/terraform_binding/1"
}

extension "event_model_link" {
  role     = "behavior_link"
  contract = "requirements_hcl/event_model_link/1"
}

extension "html_view" {
  role     = "view"
  contract = "requirements_hcl/html_view/1"
}
`

const resolvePVBlock = "pv \"gateway_timeout\" {\n  realizes = [dp.left_output]\n}\n"

// resolveFile is one named source file.
type resolveFile struct {
	path   string
	source string
}

// resolveLocation is a 1-based line in one file.
type resolveLocation struct {
	file string
	line int
}

func (l resolveLocation) String() string {
	return fmt.Sprintf("%s:%d", l.file, l.line)
}

// resolveExpect is one finding a test requires: its code and the start of its range.
type resolveExpect struct {
	code string
	at   resolveLocation
}

func (e resolveExpect) String() string {
	return e.code + " " + e.at.String()
}

// resolveFixture reads one pinned conformance fixture.
func resolveFixture(t *testing.T, name string) resolveFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(resolveFixtureDir, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return resolveFile{path: name, source: string(data)}
}

// resolveUnits parses each inline file. It fails the test when native parsing reports a finding.
func resolveUnits(t *testing.T, files ...resolveFile) []syntax.Unit {
	t.Helper()
	units := make([]syntax.Unit, 0, len(files))
	for _, file := range files {
		unit := syntax.Parse(model.Source{Path: file.path, Bytes: []byte(file.source)})
		if len(unit.Findings) != 0 {
			t.Fatalf("test source %s has %d native parse findings, want none", file.path, len(unit.Findings))
		}
		units = append(units, unit)
	}
	return units
}

// resolveRun decodes the core files and resolves them. It fails the test when decoding reports
// a finding, so each resolve test starts from a valid decode.
func resolveRun(t *testing.T, files ...resolveFile) (*model.Graph, []result.Finding) {
	t.Helper()
	core, decodeFindings := DecodeCore(resolveUnits(t, files...))
	if len(decodeFindings) != 0 {
		t.Fatalf("decode produced %d findings, want none", len(decodeFindings))
	}
	return Resolve(core)
}

// resolveRequireGraph fails the test when the graph is nil.
func resolveRequireGraph(t *testing.T, graph *model.Graph) *model.Graph {
	t.Helper()
	if graph == nil {
		t.Fatal("graph is nil, want a graph")
	}
	return graph
}

// resolveLines returns every 1-based line that contains marker.
func resolveLines(t *testing.T, source, marker string) []int {
	t.Helper()
	var lines []int
	for index, text := range strings.Split(source, "\n") {
		if strings.Contains(text, marker) {
			lines = append(lines, index+1)
		}
	}
	if len(lines) == 0 {
		t.Fatalf("marker %q not found in source", marker)
	}
	return lines
}

// resolveLine returns the single 1-based line that contains marker.
func resolveLine(t *testing.T, source, marker string) int {
	t.Helper()
	lines := resolveLines(t, source, marker)
	if len(lines) != 1 {
		t.Fatalf("marker %q found on %d lines, want 1", marker, len(lines))
	}
	return lines[0]
}

// resolveEdit replaces one fragment that must occur exactly once in source.
func resolveEdit(t *testing.T, source, old, replacement string) string {
	t.Helper()
	if count := strings.Count(source, old); count != 1 {
		t.Fatalf("edit target %q occurs %d times, want 1", old, count)
	}
	return strings.Replace(source, old, replacement, 1)
}

// resolveAppend adds one block after the existing source.
func resolveAppend(source, block string) string {
	return source + "\n" + block
}

// resolveSplitBlock cuts the top-level block whose header starts the line. It returns the block
// text, including its closing brace line, and the remaining source.
func resolveSplitBlock(t *testing.T, source, header string) (string, string) {
	t.Helper()
	lines := strings.Split(source, "\n")
	start, end := -1, -1
	for index, text := range lines {
		if start < 0 && strings.HasPrefix(text, header+" {") {
			start = index
			continue
		}
		if start >= 0 && text == "}" {
			end = index
			break
		}
	}
	if start < 0 || end < 0 {
		t.Fatalf("block %q not found in source", header)
	}
	block := strings.Join(lines[start:end+1], "\n") + "\n"
	rest := make([]string, 0, len(lines))
	rest = append(rest, lines[:start]...)
	rest = append(rest, lines[end+1:]...)
	return block, strings.Join(rest, "\n")
}

// resolveModule returns a realization module with the fields the base fixture uses.
func resolveModule(name, dp string) string {
	return "module \"" + name + "\" {\n" +
		"  realizes = [" + dp + "]\n" +
		"  inputs = [\"An input.\"]\n" +
		"  outputs = [\"An output.\"]\n" +
		"  owns_state = []\n" +
		"  dependencies = []\n" +
		"  invariants = [\"An invariant.\"]\n" +
		"  failure_behavior = \"Fail closed.\"\n" +
		"  acceptance_checks = [scenario.copy_strings]\n" +
		"}\n"
}

// resolveAllocatedChild returns a child FR of dp.left_output whose criterion allocates from target.
func resolveAllocatedChild(target string) string {
	return "fr \"child\" {\n" +
		"  parent = dp.left_output\n" +
		"  statement = \"Child of the left output.\"\n" +
		"  criterion \"allocated\" {\n" +
		"    statement = \"Allocated criterion.\"\n" +
		"    origin = \"proposed\"\n" +
		"    allocated_from = " + target + "\n" +
		"  }\n" +
		"}\n"
}

// resolveFlowWithLoop returns a flow whose loop uses the given after and back_to targets.
func resolveFlowWithLoop(after, backTo string) string {
	return "flow \"checkout\" {\n" +
		"  steps = [fr.left, fr.right]\n" +
		"  loop {\n" +
		"    after   = " + after + "\n" +
		"    back_to = " + backTo + "\n" +
		"    when    = \"retry\"\n" +
		"    bound   = 3\n" +
		"  }\n" +
		"}\n"
}

// resolveAttachmentFile returns one attachment file with one attachment block.
func resolveAttachmentFile(path, label, owner, subject string) resolveFile {
	source := "attachment \"" + label + "\" {\n" +
		"  owner   = " + owner + "\n" +
		"  subject = " + subject + "\n" +
		"  data    = { note = \"Link.\" }\n" +
		"}\n"
	return resolveFile{path: path, source: source}
}

// resolveAttachmentFindings resolves the base fixture plus one PV, decodes the manifest and the
// given attachment files, and returns the attachment findings.
func resolveAttachmentFindings(t *testing.T, attachments ...resolveFile) []result.Finding {
	t.Helper()
	base := resolveFixture(t, "base.ad.hcl")
	core := resolveFile{path: base.path, source: resolveAppend(base.source, resolvePVBlock)}
	graph, coreFindings := resolveRun(t, core)
	resolveRequireNoFindings(t, coreFindings)
	graph = resolveRequireGraph(t, graph)

	manifestUnit := resolveUnits(t, resolveFile{path: "checkout.ad.project.hcl", source: resolveManifestSource})[0]
	manifest, manifestFindings := DecodeManifest(manifestUnit)
	if len(manifestFindings) != 0 {
		t.Fatalf("manifest decode produced %d findings, want none", len(manifestFindings))
	}

	units := resolveUnits(t, attachments...)
	attachmentsDecoded, decodeFindings := DecodeAttachments(units)
	if len(decodeFindings) != 0 {
		t.Fatalf("attachment decode produced %d findings, want none", len(decodeFindings))
	}
	return append(CheckAttachmentLabels(units), ResolveAttachments(graph, manifest, attachmentsDecoded)...)
}

// resolveRequireNoFindings fails the test when any finding is present.
func resolveRequireNoFindings(t *testing.T, findings []result.Finding) {
	t.Helper()
	if len(findings) != 0 {
		codes := make([]string, 0, len(findings))
		for _, finding := range findings {
			codes = append(codes, finding.Code)
		}
		t.Fatalf("findings = %v, want none", codes)
	}
}

// resolveFindingText names one finding by its code and the start of its range.
func resolveFindingText(finding result.Finding) string {
	if finding.Range == nil {
		return finding.Code + " <no range>"
	}
	return fmt.Sprintf("%s %s:%d", finding.Code, finding.Range.Filename, finding.Range.Start.Line)
}

// resolveRequireExact fails the test unless the findings match the wanted codes and locations, in
// any order, with no extra findings.
func resolveRequireExact(t *testing.T, findings []result.Finding, wants ...resolveExpect) {
	t.Helper()
	got := make([]string, 0, len(findings))
	for _, finding := range findings {
		got = append(got, resolveFindingText(finding))
	}
	want := make([]string, 0, len(wants))
	for _, expect := range wants {
		want = append(want, expect.String())
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "; ") != strings.Join(want, "; ") {
		t.Fatalf("findings = [%s], want [%s]", strings.Join(got, "; "), strings.Join(want, "; "))
	}
}

// resolveRequireOneOf fails the test unless exactly one finding has the code and its range starts
// at one of the wanted locations. Use it when the spec allows either declaration to carry the defect.
func resolveRequireOneOf(t *testing.T, findings []result.Finding, code string, wants ...resolveLocation) {
	t.Helper()
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	finding := findings[0]
	if finding.Code != code {
		t.Fatalf("finding code = %q, want %q", finding.Code, code)
	}
	if finding.Range == nil {
		t.Fatal("finding has no range, want a range")
	}
	got := resolveLocation{file: finding.Range.Filename, line: finding.Range.Start.Line}
	for _, want := range wants {
		if got == want {
			return
		}
	}
	t.Fatalf("finding at %s, want one of %v", got, wants)
}

// resolveRequireDuplicate fails the test unless one duplicate_identity finding names every wanted
// declaration through its range or its related ranges.
func resolveRequireDuplicate(t *testing.T, findings []result.Finding, wants ...resolveLocation) {
	t.Helper()
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	finding := findings[0]
	if finding.Code != "duplicate_identity" {
		t.Fatalf("finding code = %q, want duplicate_identity", finding.Code)
	}
	if finding.Range == nil {
		t.Fatal("finding has no range, want a range")
	}
	locations := []string{resolveLocation{file: finding.Range.Filename, line: finding.Range.Start.Line}.String()}
	for _, related := range finding.Related {
		locations = append(locations, resolveLocation{file: related.Filename, line: related.Start.Line}.String())
	}
	got := make([]string, 0, len(wants))
	for _, want := range wants {
		got = append(got, want.String())
	}
	sort.Strings(locations)
	sort.Strings(got)
	if strings.Join(locations, "; ") != strings.Join(got, "; ") {
		t.Fatalf("duplicate locations = [%s], want [%s]", strings.Join(locations, "; "), strings.Join(got, "; "))
	}
}

// resolveRequireOnlyCode fails the test unless there is at least one finding and every finding has
// the code. Use it when the spec pins the code but not the declaration that carries it.
func resolveRequireOnlyCode(t *testing.T, findings []result.Finding, code string) {
	t.Helper()
	if len(findings) == 0 {
		t.Fatalf("findings are empty, want code %q", code)
	}
	for _, finding := range findings {
		if finding.Code != code {
			t.Fatalf("finding code = %q, want %q", finding.Code, code)
		}
	}
}

// resolveRequireIDs fails the test unless the IDs equal the wanted values in order.
func resolveRequireIDs(t *testing.T, name string, got []model.ID, want ...string) {
	t.Helper()
	values := make([]string, 0, len(got))
	for _, id := range got {
		values = append(values, string(id))
	}
	if strings.Join(values, ", ") != strings.Join(want, ", ") {
		t.Fatalf("%s = [%s], want [%s]", name, strings.Join(values, ", "), strings.Join(want, ", "))
	}
}

func TestResolveBaseFixtureResolvesWithoutFindings(t *testing.T) {
	// Given the pinned base fixture, which is a valid two-output core.
	base := resolveFixture(t, "base.ad.hcl")

	// When the core is resolved.
	graph, findings := resolveRun(t, base)

	// Then the graph is built, has no findings, and records the design.
	resolveRequireNoFindings(t, findings)
	graph = resolveRequireGraph(t, graph)
	if graph.Design != model.ID("design.two_outputs") {
		t.Fatalf("design = %q, want design.two_outputs", graph.Design)
	}
}

func TestResolveRootSiblingSetKeepsAuthoredOrder(t *testing.T) {
	// Given the base fixture, whose root FRs and DPs are authored left before right.
	base := resolveFixture(t, "base.ad.hcl")

	// When the core is resolved.
	graph, findings := resolveRun(t, base)

	// Then the design sibling set lists the FRs and DPs in authored order.
	resolveRequireNoFindings(t, findings)
	graph = resolveRequireGraph(t, graph)
	set, ok := graph.SiblingSet(model.ID("design.two_outputs"))
	if !ok {
		t.Fatal("design sibling set is missing")
	}
	resolveRequireIDs(t, "design FRs", set.FRs, "fr.left", "fr.right")
	resolveRequireIDs(t, "design DPs", set.DPs, "dp.left_output", "dp.right_output")
}

func TestResolveDesignSiblingSetComesFirst(t *testing.T) {
	// Given the base fixture, which has one design scope and no child scopes.
	base := resolveFixture(t, "base.ad.hcl")

	// When the core is resolved.
	graph, findings := resolveRun(t, base)

	// Then the design scope is the only sibling set, and dp.left_output has none because it has
	// no child FRs.
	resolveRequireNoFindings(t, findings)
	graph = resolveRequireGraph(t, graph)
	if len(graph.SiblingSets) != 1 {
		t.Fatalf("sibling set count = %d, want 1 (the design scope only)", len(graph.SiblingSets))
	}
	if graph.SiblingSets[0].Scope != model.ID("design.two_outputs") {
		t.Fatalf("first sibling scope = %q, want design.two_outputs", graph.SiblingSets[0].Scope)
	}
	if _, ok := graph.SiblingSet(model.ID("dp.left_output")); ok {
		t.Fatal("sibling set of dp.left_output exists, want none because it has no child FRs")
	}
}

func TestResolveControllersMapEachFRToItsDPs(t *testing.T) {
	// Given the base fixture, where each root FR has one controlling DP.
	base := resolveFixture(t, "base.ad.hcl")

	// When the core is resolved.
	graph, findings := resolveRun(t, base)

	// Then each FR maps to the DP that controls it.
	resolveRequireNoFindings(t, findings)
	graph = resolveRequireGraph(t, graph)
	resolveRequireIDs(t, "controllers of fr.left", graph.Controllers[model.ID("fr.left")], "dp.left_output")
	resolveRequireIDs(t, "controllers of fr.right", graph.Controllers[model.ID("fr.right")], "dp.right_output")
}

func TestResolveMultiFRControlListsDPOnceInSiblingSet(t *testing.T) {
	// Given the base fixture where dp.left_output controls both root FRs.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.left, fr.right]\n")

	// When the core is resolved.
	graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then the DP appears once in the sibling set and both FRs list it as a controller.
	resolveRequireNoFindings(t, findings)
	graph = resolveRequireGraph(t, graph)
	set, ok := graph.SiblingSet(model.ID("design.two_outputs"))
	if !ok {
		t.Fatal("design sibling set is missing")
	}
	resolveRequireIDs(t, "design DPs", set.DPs, "dp.left_output", "dp.right_output")
	resolveRequireIDs(t, "controllers of fr.right", graph.Controllers[model.ID("fr.right")], "dp.left_output", "dp.right_output")
}

func TestResolveChildFRsFormSiblingSetOfParentDP(t *testing.T) {
	// Given the base fixture plus a child FR whose parent is dp.left_output.
	base := resolveFixture(t, "base.ad.hcl")
	child := "fr \"child\" {\n  parent = dp.left_output\n  statement = \"Child of the left output.\"\n}\n"
	source := resolveAppend(base.source, child)

	// When the core is resolved.
	graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then the child is listed under its parent and forms the parent's sibling set, the leaf DP
	// dp.right_output has no sibling set, and the design sibling set keeps only the root FRs.
	resolveRequireNoFindings(t, findings)
	graph = resolveRequireGraph(t, graph)
	resolveRequireIDs(t, "children of dp.left_output", graph.Children[model.ID("dp.left_output")], "fr.child")
	childSet, ok := graph.SiblingSet(model.ID("dp.left_output"))
	if !ok {
		t.Fatal("sibling set of dp.left_output is missing")
	}
	resolveRequireIDs(t, "child FRs", childSet.FRs, "fr.child")
	if _, ok := graph.SiblingSet(model.ID("dp.right_output")); ok {
		t.Fatal("sibling set of dp.right_output exists, want none because it has no child FRs")
	}
	designSet, ok := graph.SiblingSet(model.ID("design.two_outputs"))
	if !ok {
		t.Fatal("design sibling set is missing")
	}
	resolveRequireIDs(t, "design FRs", designSet.FRs, "fr.left", "fr.right")
}

func TestResolveHasReportsDeclaredIdentities(t *testing.T) {
	// Given the base fixture, which declares module.left and not module.absent.
	base := resolveFixture(t, "base.ad.hcl")

	// When the core is resolved.
	graph, findings := resolveRun(t, base)

	// Then Has reports the declared module as present and the missing one as absent.
	resolveRequireNoFindings(t, findings)
	graph = resolveRequireGraph(t, graph)
	if !graph.Has(model.ID("module.left")) {
		t.Fatal("Has(module.left) = false, want true")
	}
	if graph.Has(model.ID("module.absent")) {
		t.Fatal("Has(module.absent) = true, want false")
	}
}

func TestResolveForwardReferenceAcrossFiles(t *testing.T) {
	// Given the base fixture with its need block moved into a second file listed after it.
	base := resolveFixture(t, "base.ad.hcl")
	need, rest := resolveSplitBlock(t, base.source, `need "separate_outputs"`)
	files := []resolveFile{
		{path: base.path, source: rest},
		{path: "need.ad.hcl", source: need},
	}

	// When the core is resolved.
	graph, findings := resolveRun(t, files...)

	// Then the forward reference to the need in the later file resolves with no findings.
	resolveRequireNoFindings(t, findings)
	resolveRequireGraph(t, graph)
}

func TestResolveDuplicateFixtureKeepsBothDeclarations(t *testing.T) {
	// Given the base fixture and the duplicate fixture, listed in each order.
	base := resolveFixture(t, "base.ad.hcl")
	duplicate := resolveFixture(t, "duplicate.ad.hcl")
	baseLine := resolveLine(t, base.source, `need "separate_outputs"`)
	duplicateLine := resolveLine(t, duplicate.source, `need "separate_outputs"`)
	cases := []struct {
		name  string
		files []resolveFile
	}{
		{name: "base first", files: []resolveFile{base, duplicate}},
		{name: "duplicate first", files: []resolveFile{duplicate, base}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the core is resolved.
			_, findings := resolveRun(t, tc.files...)

			// Then one duplicate_identity finding names both declarations, whatever the file order.
			resolveRequireDuplicate(t, findings,
				resolveLocation{file: base.path, line: baseLine},
				resolveLocation{file: duplicate.path, line: duplicateLine},
			)
		})
	}
}

func TestResolveWrongKindFixtureReportsControlsTraversal(t *testing.T) {
	// Given the base fixture and the wrong-kind fixture, whose dp controls a need.
	base := resolveFixture(t, "base.ad.hcl")
	wrongKind := resolveFixture(t, "wrong-kind.ad.hcl")

	// When the core is resolved.
	_, findings := resolveRun(t, base, wrongKind)

	// Then one invalid_reference points at the controls line of the wrong-kind file.
	line := resolveLine(t, wrongKind.source, "controls = [need.separate_outputs]")
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_reference", at: resolveLocation{file: wrongKind.path, line: line}})
}

func TestResolveDuplicateDesignAcrossFilesReportsDuplicateIdentity(t *testing.T) {
	// Given the base fixture and a second file that declares design.two_outputs again.
	base := resolveFixture(t, "base.ad.hcl")
	other := resolveFile{
		path:   "other.ad.hcl",
		source: "design \"two_outputs\" {\n  boundary = \"A second design with the same label.\"\n}\n",
	}

	// When the core is resolved.
	_, findings := resolveRun(t, base, other)

	// Then one duplicate_identity finding names both design declarations.
	resolveRequireDuplicate(t, findings,
		resolveLocation{file: base.path, line: resolveLine(t, base.source, `design "two_outputs"`)},
		resolveLocation{file: other.path, line: 1},
	)
}

func TestResolveMissingDesignReportsInvalidStructure(t *testing.T) {
	// Given the base fixture with its design block and its cells block removed.
	base := resolveFixture(t, "base.ad.hcl")
	_, withoutDesign := resolveSplitBlock(t, base.source, `design "two_outputs"`)
	_, withoutCells := resolveSplitBlock(t, withoutDesign, `cells "separate_inputs"`)

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: withoutCells})

	// Then the missing design is reported as invalid_structure.
	resolveRequireOnlyCode(t, findings, "invalid_structure")
}

func TestResolveDuplicateLabelWithinOneFileReportsDuplicateIdentity(t *testing.T) {
	// Given the base fixture with a second need or fr block that reuses an existing label.
	base := resolveFixture(t, "base.ad.hcl")
	cases := []struct {
		name   string
		block  string
		marker string
	}{
		{
			name:   "need",
			block:  "need \"separate_outputs\" {\n  statement = \"Again.\"\n}\n",
			marker: `need "separate_outputs"`,
		},
		{
			name:   "fr",
			block:  "fr \"left\" {\n  statement = \"Again.\"\n  need = need.separate_outputs\n}\n",
			marker: `fr "left"`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			source := resolveAppend(base.source, tc.block)
			lines := resolveLines(t, source, tc.marker)
			if len(lines) != 2 {
				t.Fatalf("marker %q found on %d lines, want 2", tc.marker, len(lines))
			}

			// When the core is resolved.
			_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

			// Then one duplicate_identity finding names both declarations.
			resolveRequireDuplicate(t, findings,
				resolveLocation{file: base.path, line: lines[0]},
				resolveLocation{file: base.path, line: lines[1]},
			)
		})
	}
}

func TestResolveMissingTargetReportsInvalidReference(t *testing.T) {
	// Given the base fixture with one reference changed to a target that does not exist.
	base := resolveFixture(t, "base.ad.hcl")
	cases := []struct {
		name   string
		source string
		marker string
	}{
		{
			name:   "dp controls a missing FR",
			source: resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.missing]\n"),
			marker: "controls = [fr.missing]",
		},
		{
			name:   "module acceptance names a missing criterion",
			source: resolveEdit(t, base.source, "fr.left.criterion.exact, scenario.copy_strings", "fr.left.criterion.gone, scenario.copy_strings"),
			marker: "fr.left.criterion.gone",
		},
		{
			name:   "module acceptance uses the short criterion form",
			source: resolveEdit(t, base.source, "fr.left.criterion.exact, scenario.copy_strings", "fr.left.exact, scenario.copy_strings"),
			marker: "fr.left.exact",
		},
		{
			name:   "child FR parent is missing",
			source: resolveAppend(base.source, "fr \"child\" {\n  parent = dp.absent\n  statement = \"Child.\"\n}\n"),
			marker: "parent = dp.absent",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the core is resolved.
			_, findings := resolveRun(t, resolveFile{path: base.path, source: tc.source})

			// Then one invalid_reference points at the traversal line.
			line := resolveLine(t, tc.source, tc.marker)
			resolveRequireExact(t, findings, resolveExpect{code: "invalid_reference", at: resolveLocation{file: base.path, line: line}})
		})
	}
}

func TestResolveWrongKindReferenceReportsInvalidReference(t *testing.T) {
	// Given the base fixture with one reference that names a declaration of the wrong kind.
	base := resolveFixture(t, "base.ad.hcl")
	cases := []struct {
		name   string
		source string
		marker string
	}{
		{
			name:   "cells scope names an FR",
			source: resolveEdit(t, base.source, "scope = design.two_outputs", "scope = fr.left"),
			marker: "scope = fr.left",
		},
		{
			name:   "scenario verifies names an FR",
			source: resolveEdit(t, base.source, "verifies = [fr.left.criterion.exact,", "verifies = [fr.left,"),
			marker: "verifies = [fr.left,",
		},
		{
			name:   "module realizes names an FR",
			source: resolveEdit(t, base.source, "realizes = [dp.left_output]\n", "realizes = [fr.left]\n"),
			marker: "realizes = [fr.left]",
		},
		{
			name:   "flow step names a state",
			source: resolveAppend(base.source, "state \"s\" {\n  description = \"State.\"\n}\n\nflow \"f\" {\n  steps = [state.s]\n}\n"),
			marker: "steps = [state.s]",
		},
		{
			name:   "dp owns_state names an FR",
			source: resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.left]\n  owns_state = [fr.right]\n"),
			marker: "owns_state = [fr.right]",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the core is resolved.
			_, findings := resolveRun(t, resolveFile{path: base.path, source: tc.source})

			// Then one invalid_reference points at the traversal line.
			line := resolveLine(t, tc.source, tc.marker)
			resolveRequireExact(t, findings, resolveExpect{code: "invalid_reference", at: resolveLocation{file: base.path, line: line}})
		})
	}
}

func TestResolveIndependentDefectsEachReported(t *testing.T) {
	// Given the base fixture with two unrelated broken references in different blocks.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.missing]\n")
	source = resolveEdit(t, source, "fr.left.criterion.exact, scenario.copy_strings", "fr.left.criterion.gone, scenario.copy_strings")

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then each broken reference is reported on its own line.
	resolveRequireExact(t, findings,
		resolveExpect{code: "invalid_reference", at: resolveLocation{file: base.path, line: resolveLine(t, source, "controls = [fr.missing]")}},
		resolveExpect{code: "invalid_reference", at: resolveLocation{file: base.path, line: resolveLine(t, source, "fr.left.criterion.gone")}},
	)
}

func TestResolveDPParentCycleReportsInvalidHierarchy(t *testing.T) {
	// Given the base fixture plus FRs whose parent DPs form a cycle.
	base := resolveFixture(t, "base.ad.hcl")
	cases := []struct {
		name  string
		block string
	}{
		{
			name: "self parent",
			block: "fr \"loop_fr\" {\n  parent = dp.loop_dp\n  statement = \"Loop.\"\n}\n\n" +
				"dp \"loop_dp\" {\n  controls = [fr.loop_fr]\n  responsibility = \"Loop.\"\n  decision = \"proposed\"\n}\n",
		},
		{
			name: "two-DP cycle",
			block: "fr \"a\" {\n  parent = dp.y\n  statement = \"A.\"\n}\n\n" +
				"fr \"b\" {\n  parent = dp.x\n  statement = \"B.\"\n}\n\n" +
				"dp \"x\" {\n  controls = [fr.a]\n  responsibility = \"X.\"\n  decision = \"proposed\"\n}\n\n" +
				"dp \"y\" {\n  controls = [fr.b]\n  responsibility = \"Y.\"\n  decision = \"proposed\"\n}\n",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the core is resolved.
			_, findings := resolveRun(t, resolveFile{path: base.path, source: resolveAppend(base.source, tc.block)})

			// Then the cycle is reported as invalid_hierarchy.
			resolveRequireOnlyCode(t, findings, "invalid_hierarchy")
		})
	}
}

func TestResolveDPControlsSpanningSiblingSetsReportsInvalidHierarchy(t *testing.T) {
	// Given a child FR of dp.left_output and a DP that controls it together with a root FR.
	base := resolveFixture(t, "base.ad.hcl")
	block := "fr \"child\" {\n  parent = dp.left_output\n  statement = \"Child.\"\n}\n\n" +
		"dp \"split\" {\n  controls = [fr.child, fr.right]\n  responsibility = \"Split.\"\n  decision = \"proposed\"\n}\n"
	source := resolveAppend(base.source, block)

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then the controls line is reported as invalid_hierarchy.
	line := resolveLine(t, source, "controls = [fr.child, fr.right]")
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_hierarchy", at: resolveLocation{file: base.path, line: line}})
}

func TestResolveAllocatedFromRootCriterionReportsInvalidHierarchy(t *testing.T) {
	// Given the base fixture where a root FR criterion allocates from another root FR.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source,
		"    origin = \"proposed\"\n  }\n}\n\nfr \"right\" {",
		"    origin = \"proposed\"\n    allocated_from = fr.right.criterion.exact\n  }\n}\n\nfr \"right\" {",
	)

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then the allocation line is reported as invalid_hierarchy.
	line := resolveLine(t, source, "allocated_from = fr.right.criterion.exact")
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_hierarchy", at: resolveLocation{file: base.path, line: line}})
}

func TestResolveAllocatedFromCriterionOfParentControlledFR(t *testing.T) {
	// Given a child FR of dp.left_output whose criterion allocates from a criterion of an FR that
	// dp.left_output controls.
	base := resolveFixture(t, "base.ad.hcl")
	cases := []struct {
		name   string
		target string
		want   []resolveExpect
	}{
		{
			name:   "criterion of a controlled FR is valid",
			target: "fr.left.criterion.exact",
			want:   nil,
		},
		{
			name:   "criterion of an FR outside the parent is invalid",
			target: "fr.right.criterion.exact",
			want:   []resolveExpect{{code: "invalid_hierarchy"}},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			source := resolveAppend(base.source, resolveAllocatedChild(tc.target))

			// When the core is resolved.
			_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

			// Then the valid allocation yields no findings and the invalid one is invalid_hierarchy
			// on the allocation line.
			if tc.want == nil {
				resolveRequireNoFindings(t, findings)
				return
			}
			line := resolveLine(t, source, "allocated_from = "+tc.target)
			resolveRequireExact(t, findings, resolveExpect{code: "invalid_hierarchy", at: resolveLocation{file: base.path, line: line}})
		})
	}
}

func TestResolveModuleWithinCompositeIsValid(t *testing.T) {
	// Given the base fixture where module.left sits within the composite module.outer.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "  realizes = [dp.left_output]\n", "  realizes = [dp.left_output]\n  within = module.outer\n")
	source = resolveAppend(source, "module \"outer\" {\n  contains = [module.left]\n}\n")

	// When the core is resolved.
	graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then no findings are reported and the parent of module.left is module.outer.
	resolveRequireNoFindings(t, findings)
	graph = resolveRequireGraph(t, graph)
	if graph.Modules[model.ID("module.left")] != model.ID("module.outer") {
		t.Fatalf("parent of module.left = %q, want module.outer", graph.Modules[model.ID("module.left")])
	}
}

func TestResolveModuleWithinRealizationModuleReportsInvalidHierarchy(t *testing.T) {
	// Given the base fixture where module.left is within module.right, which realizes a DP.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "  realizes = [dp.left_output]\n", "  realizes = [dp.left_output]\n  within = module.right\n")

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then the within line is reported as invalid_hierarchy.
	line := resolveLine(t, source, "within = module.right")
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_hierarchy", at: resolveLocation{file: base.path, line: line}})
}

func TestResolveModuleWithinDisagreeingWithContainsReportsInvalidHierarchy(t *testing.T) {
	// Given module.left is within module.other, but module.outer also contains module.left.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "  realizes = [dp.left_output]\n", "  realizes = [dp.left_output]\n  within = module.other\n")
	source = resolveAppend(source, "module \"outer\" {\n  contains = [module.left]\n}\n\nmodule \"other\" {\n  contains = [module.right]\n}\n")

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then the disagreement is reported as invalid_hierarchy.
	resolveRequireOnlyCode(t, findings, "invalid_hierarchy")
}

func TestResolveModuleContainmentCycleReportsInvalidHierarchy(t *testing.T) {
	// Given two composite modules that contain each other.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveAppend(base.source, "module \"ring_a\" {\n  contains = [module.ring_b]\n}\n\nmodule \"ring_b\" {\n  contains = [module.ring_a]\n}\n")

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then the cycle is reported as invalid_hierarchy.
	resolveRequireOnlyCode(t, findings, "invalid_hierarchy")
}

func TestResolveTwoRealizationModulesOnOneDPReportsConflictingOwnership(t *testing.T) {
	// Given the base fixture plus a second module that realizes dp.left_output.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveAppend(base.source, resolveModule("left_copy", "dp.left_output"))
	lines := resolveLines(t, source, "realizes = [dp.left_output]")

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then one conflicting_ownership finding points at one of the two realizes lines.
	resolveRequireOneOf(t, findings, "conflicting_ownership",
		resolveLocation{file: base.path, line: lines[0]},
		resolveLocation{file: base.path, line: lines[1]},
	)
}

func TestResolveTwoDPsOwningOneStateReportsConflictingOwnership(t *testing.T) {
	// Given the base fixture where both leaf DPs claim state.shared.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.left]\n  owns_state = [state.shared]\n")
	source = resolveEdit(t, source, "controls = [fr.right]\n", "controls = [fr.right]\n  owns_state = [state.shared]\n")
	source = resolveAppend(source, "state \"shared\" {\n  description = \"Shared state.\"\n}\n")
	lines := resolveLines(t, source, "owns_state = [state.shared]")

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then one conflicting_ownership finding points at one of the two owns_state lines.
	resolveRequireOneOf(t, findings, "conflicting_ownership",
		resolveLocation{file: base.path, line: lines[0]},
		resolveLocation{file: base.path, line: lines[1]},
	)
}

func TestResolveFlowStepsAndAfterAcceptValidOrder(t *testing.T) {
	// Given the base fixture plus two flows, where checkout runs after intake and intake uses a
	// flat group of steps.
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveAppend(base.source,
		"flow \"intake\" {\n  steps = [module.left, [fr.right, dp.right_output]]\n}\n\n"+
			"flow \"checkout\" {\n  steps = [fr.left]\n  after = [flow.intake]\n}\n",
	)

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then no findings are reported.
	resolveRequireNoFindings(t, findings)
}

func TestResolveFlowAfterCycleReportsInvalidHierarchy(t *testing.T) {
	// Given the base fixture plus flows whose after prerequisites form a cycle.
	base := resolveFixture(t, "base.ad.hcl")
	cases := []struct {
		name  string
		block string
	}{
		{
			name:  "self prerequisite",
			block: "flow \"a\" {\n  steps = [fr.left]\n  after = [flow.a]\n}\n",
		},
		{
			name: "two-flow cycle",
			block: "flow \"a\" {\n  steps = [fr.left]\n  after = [flow.b]\n}\n\n" +
				"flow \"b\" {\n  steps = [fr.right]\n  after = [flow.a]\n}\n",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the core is resolved.
			_, findings := resolveRun(t, resolveFile{path: base.path, source: resolveAppend(base.source, tc.block)})

			// Then the cycle is reported as invalid_hierarchy.
			resolveRequireOnlyCode(t, findings, "invalid_hierarchy")
		})
	}
}

func TestResolveLoopEndpointOutsideStepsReportsInvalidStructure(t *testing.T) {
	// Given the base fixture plus a flow whose loop returns to fr.right, which is not one of its steps.
	base := resolveFixture(t, "base.ad.hcl")
	block := "flow \"checkout\" {\n  steps = [fr.left]\n  loop {\n    after   = fr.left\n    back_to = fr.right\n    when    = \"retry\"\n    bound   = 3\n  }\n}\n"
	source := resolveAppend(base.source, block)

	// When the core is resolved.
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

	// Then the loop block is reported as invalid_structure.
	line := resolveLine(t, source, "loop {")
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_structure", at: resolveLocation{file: base.path, line: line}})
}

func TestResolveLoopBackToMustPrecedeAfter(t *testing.T) {
	// Given a flow with steps fr.left then fr.right and a loop whose back_to target is either earlier
	// or later than its after target.
	base := resolveFixture(t, "base.ad.hcl")
	cases := []struct {
		name    string
		after   string
		backTo  string
		invalid bool
	}{
		{name: "back_to precedes after", after: "fr.right", backTo: "fr.left", invalid: false},
		{name: "back_to follows after", after: "fr.left", backTo: "fr.right", invalid: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			source := resolveAppend(base.source, resolveFlowWithLoop(tc.after, tc.backTo))

			// When the core is resolved.
			_, findings := resolveRun(t, resolveFile{path: base.path, source: source})

			// Then the earlier back_to is valid, and the later one is invalid_structure on the loop line.
			if !tc.invalid {
				resolveRequireNoFindings(t, findings)
				return
			}
			line := resolveLine(t, source, "loop {")
			resolveRequireExact(t, findings, resolveExpect{code: "invalid_structure", at: resolveLocation{file: base.path, line: line}})
		})
	}
}

func TestResolveBlockingDefectReturnsNilGraph(t *testing.T) {
	// Given three cores, each with one blocking core-structure defect.
	base := resolveFixture(t, "base.ad.hcl")
	_, withoutDesign := resolveSplitBlock(t, base.source, `design "two_outputs"`)
	_, withoutCells := resolveSplitBlock(t, withoutDesign, `cells "separate_inputs"`)
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "missing target",
			source: resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.missing]\n"),
		},
		{
			name:   "duplicate label",
			source: resolveAppend(base.source, "need \"separate_outputs\" {\n  statement = \"Again.\"\n}\n"),
		},
		{
			name:   "missing design",
			source: withoutCells,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// When the core is resolved.
			graph, findings := resolveRun(t, resolveFile{path: base.path, source: tc.source})

			// Then the graph is nil and at least one finding is reported.
			if graph != nil {
				t.Fatal("graph is non-nil, want nil when core structure is invalid")
			}
			if len(findings) == 0 {
				t.Fatal("findings are empty, want at least one")
			}
		})
	}
}

func TestResolveAttachmentsAcceptSubjectForEachOwnerRole(t *testing.T) {
	// Given attachments whose subject kind matches the owner role.
	cases := []struct {
		name    string
		owner   string
		subject string
	}{
		{name: "binding to a PV", owner: "extension.terraform_binding", subject: "pv.gateway_timeout"},
		{name: "behavior link to a module", owner: "extension.event_model_link", subject: "module.left"},
		{name: "view to the design", owner: "extension.html_view", subject: "design.two_outputs"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			file := resolveAttachmentFile("attachments.ad.hcl", "link", tc.owner, tc.subject)

			// When the attachments are resolved.
			findings := resolveAttachmentFindings(t, file)

			// Then no findings are reported.
			resolveRequireNoFindings(t, findings)
		})
	}
}

func TestResolveAttachmentsUndeclaredOwnerReportsInvalidReference(t *testing.T) {
	// Given an attachment whose owner is not declared in the manifest.
	file := resolveAttachmentFile("attachments.ad.hcl", "link", "extension.absent", "pv.gateway_timeout")

	// When the attachments are resolved.
	findings := resolveAttachmentFindings(t, file)

	// Then the owner line is reported as invalid_reference.
	line := resolveLine(t, file.source, "extension.absent")
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_reference", at: resolveLocation{file: file.path, line: line}})
}

func TestResolveAttachmentsWrongKindSubjectReportsInvalidReference(t *testing.T) {
	// Given attachments whose subject kind does not match the owner role.
	cases := []struct {
		name    string
		owner   string
		subject string
	}{
		{name: "binding to a module", owner: "extension.terraform_binding", subject: "module.left"},
		{name: "behavior link to a PV", owner: "extension.event_model_link", subject: "pv.gateway_timeout"},
		{name: "view to a module", owner: "extension.html_view", subject: "module.left"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			file := resolveAttachmentFile("attachments.ad.hcl", "link", tc.owner, tc.subject)

			// When the attachments are resolved.
			findings := resolveAttachmentFindings(t, file)

			// Then the subject line is reported as invalid_reference.
			line := resolveLine(t, file.source, "subject = ")
			resolveRequireExact(t, findings, resolveExpect{code: "invalid_reference", at: resolveLocation{file: file.path, line: line}})
		})
	}
}

func TestResolveAttachmentsMissingSubjectReportsInvalidReference(t *testing.T) {
	// Given an attachment whose subject names a PV that the core does not declare.
	file := resolveAttachmentFile("attachments.ad.hcl", "link", "extension.terraform_binding", "pv.absent")

	// When the attachments are resolved.
	findings := resolveAttachmentFindings(t, file)

	// Then the subject line is reported as invalid_reference.
	line := resolveLine(t, file.source, "pv.absent")
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_reference", at: resolveLocation{file: file.path, line: line}})
}

func TestResolveAttachmentLabelDuplicateAcrossFiles(t *testing.T) {
	// Given two attachment files that each declare an attachment with the same label.
	first := resolveAttachmentFile("attachments-a.ad.hcl", "pv_binding", "extension.terraform_binding", "pv.gateway_timeout")
	second := resolveAttachmentFile("attachments-b.ad.hcl", "pv_binding", "extension.terraform_binding", "pv.gateway_timeout")

	// When the attachments are resolved.
	findings := resolveAttachmentFindings(t, first, second)

	// Then one duplicate_identity finding names both declarations.
	resolveRequireDuplicate(t, findings,
		resolveLocation{file: first.path, line: resolveLine(t, first.source, `attachment "pv_binding"`)},
		resolveLocation{file: second.path, line: resolveLine(t, second.source, `attachment "pv_binding"`)},
	)
}

func TestCheckAttachmentLabelsKeepsInvalidEnvelopeRanges(t *testing.T) {
	for _, invalid := range []int{0, 1} {
		t.Run(fmt.Sprint(invalid), func(t *testing.T) {
			files := []resolveFile{
				resolveAttachmentFile("first.ad.ext.hcl", "same", "extension.html_view", "design.two_outputs"),
				resolveAttachmentFile("second.ad.ext.hcl", "same", "extension.html_view", "design.two_outputs"),
			}
			files[invalid].source = resolveEdit(t, files[invalid].source, "  data    = { note = \"Link.\" }\n", "")
			units := resolveUnits(t, files...)
			attachments, findings := DecodeAttachments(units)
			resolveRequireOnlyCode(t, findings, "invalid_attachment")
			if len(attachments) != 1 {
				t.Fatalf("decoded %d attachments, want only the valid envelope", len(attachments))
			}
			resolveRequireDuplicate(t, CheckAttachmentLabels(units),
				resolveLocation{file: files[0].path, line: 1},
				resolveLocation{file: files[1].path, line: 1})
		})
	}
}

func TestCheckAttachmentLabelsDoesNotRequireCoreGraph(t *testing.T) {
	base := resolveFixture(t, "base.ad.hcl")
	base.source = resolveEdit(t, base.source, "controls = [fr.left]", "controls = [fr.missing]")
	graph, findings := resolveRun(t, base)
	if graph != nil || len(findings) == 0 {
		t.Fatal("invalid core must have findings and no graph")
	}
	first := resolveAttachmentFile("first.ad.ext.hcl", "same", "extension.html_view", "design.two_outputs")
	second := resolveAttachmentFile("second.ad.ext.hcl", "same", "extension.html_view", "design.two_outputs")
	resolveRequireDuplicate(t, CheckAttachmentLabels(resolveUnits(t, first, second)),
		resolveLocation{file: first.path, line: 1},
		resolveLocation{file: second.path, line: 1})
}

func TestCheckAttachmentLabelsIgnoresNonAttachmentAndUnlabeledBlocks(t *testing.T) {
	files := []resolveFile{{path: "attachments.ad.ext.hcl", source: `attachment {
  data = {}
}
attachment {
  data = {}
}
other "same" {}
attachment "same" {
  owner = extension.html_view
  subject = design.two_outputs
  data = {}
}
`}}
	resolveRequireNoFindings(t, CheckAttachmentLabels(resolveUnits(t, files...)))
}

func TestResolveStateOwnershipRequiresLeafDP(t *testing.T) {
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.left]\n  owns_state = [state.cache]\n")
	source = resolveAppend(source, `state "cache" {
  description = "Cache."
}
fr "child" {
  parent = dp.left_output
  statement = "Child."
}
dp "child" {
  controls = [fr.child]
  responsibility = "Child control."
  decision = "proposed"
}
`)
	graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
	if graph != nil {
		t.Fatal("nonleaf state owner must block the graph")
	}
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_structure",
		at: resolveLocation{file: base.path, line: resolveLine(t, source, "owns_state = [state.cache]")}})
}

func TestResolveStateOwnershipRejectsCompositeClaims(t *testing.T) {
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveAppend(base.source, `state "cache" {
  description = "Cache."
}
module "outer" {
  contains = [module.left, module.right]
  owns_state = [state.cache]
}
`)
	graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
	if graph != nil {
		t.Fatal("composite state claim must block the graph")
	}
	resolveRequireExact(t, findings, resolveExpect{code: "invalid_structure",
		at: resolveLocation{file: base.path, line: resolveLine(t, source, "owns_state = [state.cache]")}})
}

func TestResolveStateOwnershipModuleMustRealizeOwner(t *testing.T) {
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.left]\n  owns_state = [state.cache]\n")
	source = resolveEdit(t, source, "outputs = [\"The same right string.\"]\n  owns_state = []", "outputs = [\"The same right string.\"]\n  owns_state = [state.cache]")
	source = resolveAppend(source, "state \"cache\" {\n  description = \"Cache.\"\n}\n")
	graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
	if graph != nil {
		t.Fatal("unrelated module state claim must block the graph")
	}
	lines := resolveLines(t, source, "owns_state = [state.cache]")
	resolveRequireExact(t, findings, resolveExpect{code: "conflicting_ownership",
		at: resolveLocation{file: base.path, line: lines[1]}})
	if len(findings[0].Related) != 1 || findings[0].Related[0].Start.Line != lines[0] {
		t.Fatalf("related ranges = %v, want the DP state claim on line %d", findings[0].Related, lines[0])
	}
}

func TestResolveStateOwnershipMatchingModuleIsSameLogicalOwner(t *testing.T) {
	for _, dpClaim := range []bool{false, true} {
		t.Run(fmt.Sprint(dpClaim), func(t *testing.T) {
			base := resolveFixture(t, "base.ad.hcl")
			source := resolveEdit(t, base.source, "outputs = [\"The same left string.\"]\n  owns_state = []", "outputs = [\"The same left string.\"]\n  owns_state = [state.cache]")
			if dpClaim {
				source = resolveEdit(t, source, "controls = [fr.left]\n", "controls = [fr.left]\n  owns_state = [state.cache]\n")
			}
			source = resolveAppend(source, "state \"cache\" {\n  description = \"Cache.\"\n}\n")
			graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
			resolveRequireNoFindings(t, findings)
			resolveRequireGraph(t, graph)
		})
	}
}

func TestResolveStateOwnershipCompetingModulesRetainBothClaims(t *testing.T) {
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "outputs = [\"The same left string.\"]\n  owns_state = []", "outputs = [\"The same left string.\"]\n  owns_state = [state.cache]")
	source = resolveEdit(t, source, "outputs = [\"The same right string.\"]\n  owns_state = []", "outputs = [\"The same right string.\"]\n  owns_state = [state.cache]")
	source = resolveAppend(source, "state \"cache\" {\n  description = \"Cache.\"\n}\n")
	_, findings := resolveRun(t, resolveFile{path: base.path, source: source})
	lines := resolveLines(t, source, "owns_state = [state.cache]")
	resolveRequireExact(t, findings, resolveExpect{code: "conflicting_ownership",
		at: resolveLocation{file: base.path, line: lines[1]}})
	if len(findings[0].Related) != 1 || findings[0].Related[0].Start.Line != lines[0] {
		t.Fatalf("related ranges = %v, want the first module claim on line %d", findings[0].Related, lines[0])
	}
}

func TestResolveCellsRequireNonemptySiblingSet(t *testing.T) {
	base := resolveFixture(t, "base.ad.hcl")
	t.Run("leaf DP", func(t *testing.T) {
		source := resolveEdit(t, base.source, "scope = design.two_outputs", "scope = dp.left_output")
		graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
		if graph != nil {
			t.Fatal("orphan cell scope must block the graph")
		}
		resolveRequireExact(t, findings, resolveExpect{code: "invalid_structure",
			at: resolveLocation{file: base.path, line: resolveLine(t, source, "scope = dp.left_output")}})
	})
	t.Run("empty design", func(t *testing.T) {
		source := "design \"empty\" {\n  boundary = \"Empty.\"\n}\ncells \"orphan\" {\n  scope = design.empty\n  unlisted = \"unknown\"\n}\n"
		graph, findings := resolveRun(t, resolveFile{path: "empty.ad.hcl", source: source})
		if graph != nil {
			t.Fatal("empty root cell scope must block the graph")
		}
		resolveRequireExact(t, findings, resolveExpect{code: "invalid_structure",
			at: resolveLocation{file: "empty.ad.hcl", line: resolveLine(t, source, "scope = design.empty")}})
	})
	t.Run("parent DP", func(t *testing.T) {
		source := resolveEdit(t, base.source, "scope = design.two_outputs", "scope = dp.left_output")
		source = resolveAppend(source, `fr "child" {
  parent = dp.left_output
  statement = "Child."
}
dp "child" {
  controls = [fr.child]
  responsibility = "Child control."
  decision = "proposed"
}
`)
		graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
		resolveRequireNoFindings(t, findings)
		set, ok := resolveRequireGraph(t, graph).SiblingSet("dp.left_output")
		if !ok || len(set.FRs) != 1 || set.FRs[0] != "fr.child" {
			t.Fatalf("child sibling set = %v, %t", set, ok)
		}
	})
}

func TestResolveInformationEvidenceBelongsToExactCandidate(t *testing.T) {
	for _, basis := range []string{"unknown", "measured"} {
		for _, evidenceCandidate := range []string{"decision.payment.candidate.current", "decision.payment.candidate.alternative", "decision.other.candidate.current"} {
			t.Run(basis+"/"+evidenceCandidate, func(t *testing.T) {
				base := resolveFixture(t, "base.ad.hcl")
				source := resolveAppend(base.source, fmt.Sprintf(`decision "payment" {
  addresses = [fr.left]
  candidate "current" {
    dps = [dp.left_output]
    information {
      basis = %q
      evidence = [evidence.e]
    }
  }
  candidate "alternative" {
    dps = [dp.left_output]
  }
}
decision "other" {
  addresses = [fr.left]
  candidate "current" {
    dps = [dp.left_output]
  }
}
context "eu" {
  description = "EU."
}
workload "peak" {
  description = "Peak."
}
`, basis))
				source = resolveAppend(source, realizationEvidence(map[string]string{
					"candidate": evidenceCandidate,
					"scenario":  "scenario.copy_strings",
				}, ""))
				graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
				if evidenceCandidate == "decision.payment.candidate.current" {
					resolveRequireNoFindings(t, findings)
					resolveRequireGraph(t, graph)
					return
				}
				if graph != nil {
					t.Fatal("cross-candidate evidence reference must block the graph")
				}
				resolveRequireExact(t, findings, resolveExpect{code: "invalid_evidence",
					at: resolveLocation{file: base.path, line: resolveLine(t, source, "evidence = [evidence.e]")}})
				if len(findings[0].Related) != 1 || findings[0].Related[0].Start.Line != resolveLine(t, source, "candidate = "+evidenceCandidate) {
					t.Fatalf("related ranges = %v, want evidence candidate reference", findings[0].Related)
				}
			})
		}
	}
}

func TestResolveStateOwnershipModuleRequiresOneLogicalLeaf(t *testing.T) {
	for _, kind := range []string{"parent", "ambiguous"} {
		t.Run(kind, func(t *testing.T) {
			base := resolveFixture(t, "base.ad.hcl")
			source := resolveEdit(t, base.source, "outputs = [\"The same left string.\"]\n  owns_state = []", "outputs = [\"The same left string.\"]\n  owns_state = [state.cache]")
			source = resolveAppend(source, "state \"cache\" {\n  description = \"Cache.\"\n}\n")
			if kind == "parent" {
				source = resolveAppend(source, `fr "child" {
  parent = dp.left_output
  statement = "Child."
}
dp "child" {
  controls = [fr.child]
  responsibility = "Child control."
  decision = "proposed"
}
`)
			} else {
				_, source = resolveSplitBlock(t, source, `module "right"`)
				source = resolveEdit(t, source, "realizes = [dp.left_output]", "realizes = [dp.left_output, dp.right_output]")
			}
			graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
			if graph != nil {
				t.Fatal("module state ownership without one logical leaf must block the graph")
			}
			resolveRequireExact(t, findings, resolveExpect{code: "invalid_structure",
				at: resolveLocation{file: base.path, line: resolveLine(t, source, "owns_state = [state.cache]")}})
		})
	}
}

func TestResolveStateOwnershipReportsIndependentInvalidClaims(t *testing.T) {
	base := resolveFixture(t, "base.ad.hcl")
	source := resolveEdit(t, base.source, "controls = [fr.left]\n", "controls = [fr.left]\n  owns_state = [state.cache]\n")
	source = resolveAppend(source, `state "cache" {
  description = "Cache."
}
fr "child" {
  parent = dp.left_output
  statement = "Child."
}
dp "child" {
  controls = [fr.child]
  responsibility = "Child control."
  decision = "proposed"
}
module "child" {
  realizes = [dp.child]
  owns_state = [state.cache]
}
module "outer" {
  contains = [module.child]
  owns_state = [state.cache]
}
`)
	graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
	if graph != nil {
		t.Fatal("invalid independent ownership claims must block the graph")
	}
	lines := resolveLines(t, source, "owns_state = [state.cache]")
	resolveRequireExact(t, findings,
		resolveExpect{code: "invalid_structure", at: resolveLocation{file: base.path, line: lines[0]}},
		resolveExpect{code: "conflicting_ownership", at: resolveLocation{file: base.path, line: lines[1]}},
		resolveExpect{code: "invalid_structure", at: resolveLocation{file: base.path, line: lines[2]}})
	for _, finding := range findings {
		if finding.Code == "conflicting_ownership" && (len(finding.Related) != 1 || finding.Related[0].Start.Line != lines[0]) {
			t.Fatalf("related ranges = %v, want nonleaf DP's independent claim", finding.Related)
		}
	}
}

func TestResolveStateOwnershipMultiLeafModuleMatchesExplicitOwner(t *testing.T) {
	base := resolveFixture(t, "base.ad.hcl")
	_, source := resolveSplitBlock(t, base.source, `module "right"`)
	source = resolveEdit(t, source, "controls = [fr.left]\n", "controls = [fr.left]\n  owns_state = [state.cache]\n")
	source = resolveEdit(t, source, "realizes = [dp.left_output]", "realizes = [dp.left_output, dp.right_output]")
	source = resolveEdit(t, source, "  owns_state = []", "  owns_state = [state.cache]")
	source = resolveAppend(source, "state \"cache\" {\n  description = \"Cache.\"\n}\n")
	graph, findings := resolveRun(t, resolveFile{path: base.path, source: source})
	resolveRequireNoFindings(t, findings)
	resolveRequireGraph(t, graph)
}
