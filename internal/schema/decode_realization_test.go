package schema

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
)

const (
	realizationCorePath     = "realization.ad.hcl"
	realizationManifestPath = "realization.ad.project.hcl"
	realizationExtPath      = "realization.ad.ext.hcl"
)

type realizationAttribute struct {
	name  string
	value string
}

var realizationEvidenceAttributes = []realizationAttribute{
	{"basis", `"measured"`},
	{"event", `"fare_paid"`},
	{"scenario", `scenario.checkout`},
	{"candidate", `decision.payment.candidate.card`},
	{"version", `"v1"`},
	{"context", `context.eu`},
	{"workload", `workload.peak`},
	{"horizon", `"30d"`},
	{"dependence", `"joint"`},
	{"sampling", `"random"`},
	{"trials", `100`},
	{"failures", `3`},
	{"confidence_method", `"wilson"`},
	{"confidence", `0.95`},
}

var realizationLoopAttributes = []realizationAttribute{
	{"after", "fr.retry"},
	{"back_to", "fr.checkout"},
	{"when", `"gateway times out"`},
	{"bound", "3"},
}

func realizationUnit(t *testing.T, path, source string) syntax.Unit {
	t.Helper()
	unit := syntax.Parse(model.Source{Path: path, Bytes: []byte(source)})
	if len(unit.Findings) != 0 {
		t.Fatalf("test input has %d syntax findings, want none", len(unit.Findings))
	}
	return unit
}

func realizationDecodeCore(t *testing.T, source string) (model.Core, []result.Finding) {
	t.Helper()
	return DecodeCore([]syntax.Unit{realizationUnit(t, realizationCorePath, source)})
}

func realizationDecodeManifest(t *testing.T, source string) (model.Manifest, []result.Finding) {
	t.Helper()
	return DecodeManifest(realizationUnit(t, realizationManifestPath, source))
}

func realizationDecodeAttachments(t *testing.T, source string) (model.Attachments, []result.Finding) {
	t.Helper()
	return DecodeAttachments([]syntax.Unit{realizationUnit(t, realizationExtPath, source)})
}

// realizationLine returns the 1-based line of the nth line that contains needle.
// Runs of whitespace count as one space on both sides, so an aligned attribute
// such as `core_files       = []` matches the needle `core_files =`.
// An occurrence of zero means the first occurrence.
func realizationLine(t *testing.T, source, needle string, occurrence int) int {
	t.Helper()
	if occurrence == 0 {
		occurrence = 1
	}
	wantNeedle := realizationCollapseSpaces(needle)
	seen := 0
	for index, line := range strings.Split(source, "\n") {
		if strings.Contains(realizationCollapseSpaces(line), wantNeedle) {
			seen++
			if seen == occurrence {
				return index + 1
			}
		}
	}
	t.Fatalf("test source has no occurrence %d of %q", occurrence, needle)
	return 0
}

// realizationCollapseSpaces replaces each run of whitespace with one space.
func realizationCollapseSpaces(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func realizationCodes(findings []result.Finding) []string {
	codes := make([]string, len(findings))
	for index, finding := range findings {
		codes[index] = finding.Code
	}
	return codes
}

func realizationRequireNoFindings(t *testing.T, findings []result.Finding) {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("got findings %v, want none", realizationCodes(findings))
	}
}

func realizationRequireFinding(t *testing.T, findings []result.Finding, code, file string, line int) {
	t.Helper()
	if len(findings) != 1 {
		t.Fatalf("got findings %v, want exactly one %q", realizationCodes(findings), code)
	}
	finding := findings[0]
	if finding.Code != code {
		t.Fatalf("finding code = %q, want %q", finding.Code, code)
	}
	if finding.Severity != result.Error {
		t.Fatalf("finding severity = %v, want error", finding.Severity)
	}
	if finding.Range == nil {
		t.Fatal("finding has no source range")
	}
	if finding.Range.Filename != file || finding.Range.Start.Line != line {
		t.Fatalf("finding at %s:%d, want %s:%d", finding.Range.Filename, finding.Range.Start.Line, file, line)
	}
}

// realizationOnly returns the single record of a kind and fails when the count differs.
func realizationOnly[T any](t *testing.T, kind string, records []T) T {
	t.Helper()
	if len(records) != 1 {
		t.Fatalf("%s records = %d, want 1", kind, len(records))
	}
	return records[0]
}

func realizationRequireString(t *testing.T, name string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is absent, want %q", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", name, *got, want)
	}
}

func realizationRequireAbsent[T any](t *testing.T, name string, got *T) {
	t.Helper()
	if got != nil {
		t.Fatalf("%s is present, want absent", name)
	}
}

func realizationRequireRef(t *testing.T, name string, got *model.Ref, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is absent, want %q", name, want)
	}
	if string(got.Target) != want {
		t.Fatalf("%s = %q, want %q", name, got.Target, want)
	}
}

func realizationRequireRefs(t *testing.T, name string, got []model.Ref, want ...string) {
	t.Helper()
	targets := make([]string, len(got))
	for index, ref := range got {
		targets[index] = string(ref.Target)
	}
	if fmt.Sprint(targets) != fmt.Sprint(want) {
		t.Fatalf("%s = %v, want %v", name, targets, want)
	}
}

func realizationTargetsOf(refs []model.Ref) []string {
	targets := make([]string, len(refs))
	for index, ref := range refs {
		targets[index] = string(ref.Target)
	}
	return targets
}

func realizationRequireList(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func realizationRequireOptional[T any](t *testing.T, name string, got model.Optional[T], wantPresent bool, wantCount int) {
	t.Helper()
	if got.Present != wantPresent {
		t.Fatalf("%s present = %t, want %t", name, got.Present, wantPresent)
	}
	if len(got.Items) != wantCount {
		t.Fatalf("%s items = %d, want %d", name, len(got.Items), wantCount)
	}
}

func realizationDecimal(t *testing.T, text string) *big.Float {
	t.Helper()
	value, _, err := big.ParseFloat(text, 10, 512, big.ToNearestEven)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", text, err)
	}
	return value
}

func realizationExactInt(t *testing.T, name string, value *big.Float) int64 {
	t.Helper()
	if value == nil {
		t.Fatalf("%s is absent, want a value", name)
	}
	integer, accuracy := value.Int64()
	if accuracy != big.Exact {
		t.Fatalf("%s = %s, want an exact integer", name, value.Text('g', -1))
	}
	return integer
}

// realizationBlockSource renders one block from attributes. An override replaces
// the value of the same name, and omit removes that attribute.
func realizationBlockSource(header string, attributes []realizationAttribute, overrides map[string]string, omit string) string {
	var builder strings.Builder
	builder.WriteString(header + "\n")
	for _, attribute := range attributes {
		if attribute.name == omit {
			continue
		}
		value := attribute.value
		if override, ok := overrides[attribute.name]; ok {
			value = override
		}
		builder.WriteString("  " + attribute.name + " = " + value + "\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

func realizationEvidence(overrides map[string]string, omit string) string {
	return realizationBlockSource(`evidence "e" {`, realizationEvidenceAttributes, overrides, omit)
}

func realizationLoopSource(overrides map[string]string, omit string) string {
	return "flow \"checkout\" {\n  steps = [fr.checkout]\n" +
		realizationBlockSource("  loop {", realizationLoopAttributes, overrides, omit) + "}\n"
}

func realizationDecisionSelecting(selected string) string {
	return fmt.Sprintf(`decision "payment" {
  addresses = [fr.checkout]
  selected  = %q

  candidate "card" {
    dps = [dp.gateway]
  }
}
`, selected)
}

func realizationMeasuredInformation(evidenceLine string) string {
	return fmt.Sprintf(`decision "payment" {
  addresses = [fr.checkout]

  candidate "card" {
    dps = [dp.gateway]
    information {
      basis = "measured"
%s    }
  }
}
`, evidenceLine)
}

func realizationFlowWithSteps(steps string) string {
	return fmt.Sprintf("flow \"checkout\" {\n  steps = %s\n}\n", steps)
}

func realizationScenarioWithKind(kind string) string {
	return fmt.Sprintf("scenario \"retry\" {\n  kind = %s\n}\n", kind)
}

func realizationProject(lines ...string) string {
	return "project \"checkout\" {\n" + strings.Join(lines, "\n") + "\n}\n"
}

func realizationExtension(lines ...string) string {
	return "extension \"terraform_binding\" {\n" + strings.Join(lines, "\n") + "\n}\n"
}

const (
	realizationValidProjectLanguage = `  language         = "requirements_hcl/core/1"`
	realizationValidCoreFiles       = `  core_files       = ["design.ad.hcl", "checkout/model.ad.hcl"]`
	realizationValidAttachmentFiles = `  attachment_files = []`
	realizationValidRole            = `  role     = "binding"`
	realizationValidContract        = `  contract = "requirements_hcl/terraform_binding/1"`
	realizationOwnerLine            = `  owner   = extension.terraform_binding`
	realizationSubjectLine          = `  subject = pv.gateway_timeout`
	realizationDataLine             = `  data    = { address = "payments.gateway.timeout" }`
)

func realizationAttachmentSource(lines ...string) string {
	return "attachment \"pv_binding\" {\n" + strings.Join(lines, "\n") + "\n}\n"
}

func TestDecodeRealizationDecisionDecodesAuthoredFields(t *testing.T) {
	// Given a decision with every optional field, cites, origin, and two candidates.
	source := `decision "payment" {
  addresses   = [fr.checkout]
  context     = context.eu
  workload    = workload.peak
  horizon     = "30d"
  selected    = "card"
  provisional = false
  why         = "Card fits peak load."
  cites       = [source.spec]
  origin      = "proposed"

  candidate "card" {
    dps = [dp.gateway]
  }

  candidate "wallet" {
    dps = [dp.wallet]
  }
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then the decision keeps every authored value and its candidates in order.
	realizationRequireNoFindings(t, findings)
	decision := realizationOnly(t, "decision", core.Decisions)
	if decision.ID != "decision.payment" {
		t.Fatalf("decision ID = %q, want decision.payment", decision.ID)
	}
	realizationRequireRefs(t, "addresses", decision.Addresses, "fr.checkout")
	realizationRequireRef(t, "context", decision.Context, "context.eu")
	realizationRequireRef(t, "workload", decision.Workload, "workload.peak")
	realizationRequireString(t, "horizon", decision.Horizon, "30d")
	realizationRequireString(t, "selected", decision.Selected, "card")
	if decision.Provisional == nil || *decision.Provisional {
		t.Fatalf("provisional = %v, want explicit false", decision.Provisional)
	}
	realizationRequireString(t, "why", decision.Why, "Card fits peak load.")
	realizationRequireRefs(t, "cites", decision.Cites, "source.spec")
	if decision.Origin == nil || *decision.Origin != model.OriginProposed {
		t.Fatalf("origin = %v, want proposed", decision.Origin)
	}
	if decision.Alias != nil {
		t.Fatal("alias is present on a decision, want absent")
	}
	if len(decision.Candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(decision.Candidates))
	}
	if decision.Candidates[0].ID != "decision.payment.candidate.card" {
		t.Fatalf("first candidate ID = %q", decision.Candidates[0].ID)
	}
	if decision.Candidates[1].ID != "decision.payment.candidate.wallet" {
		t.Fatalf("second candidate ID = %q", decision.Candidates[1].ID)
	}
}

func TestDecodeRealizationCandidateKeepsAbsentOptionalFields(t *testing.T) {
	// Given a candidate with only dps and an unknown information block, and a second candidate with only dps.
	source := `decision "payment" {
  addresses = [fr.checkout]

  candidate "card" {
    dps = [dp.gateway]
    information {
      basis = "unknown"
    }
  }

  candidate "wallet" {
    dps = [dp.wallet]
  }
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then omitted version, context, workload, and horizon stay absent and unknown information keeps no evidence.
	realizationRequireNoFindings(t, findings)
	decision := realizationOnly(t, "decision", core.Decisions)
	if len(decision.Candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(decision.Candidates))
	}
	card := decision.Candidates[0]
	realizationRequireRefs(t, "card dps", card.DPs, "dp.gateway")
	realizationRequireAbsent(t, "card version", card.Version)
	realizationRequireAbsent(t, "card context", card.Context)
	realizationRequireAbsent(t, "card workload", card.Workload)
	realizationRequireAbsent(t, "card horizon", card.Horizon)
	if card.Information == nil {
		t.Fatal("card information is absent, want present")
	}
	if string(card.Information.Basis) != "unknown" {
		t.Fatalf("information basis = %q, want unknown", card.Information.Basis)
	}
	if len(card.Information.Evidence) != 0 {
		t.Fatalf("information evidence = %d, want none", len(card.Information.Evidence))
	}
	wallet := decision.Candidates[1]
	realizationRequireAbsent(t, "wallet information", wallet.Information)
	realizationRequireAbsent(t, "wallet version", wallet.Version)
}

func TestDecodeRealizationDecisionRequiresCandidate(t *testing.T) {
	// Given a decision that has addresses but no candidate block.
	source := `decision "payment" {
  addresses = [fr.checkout]
}
`

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the decision block.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, `decision "payment"`, 1))
}

func TestDecodeRealizationRequiredReferenceListsStayRequired(t *testing.T) {
	cases := []struct {
		name   string
		source string
		needle string
	}{
		{
			name:   "decision without addresses",
			source: "decision \"payment\" {\n  candidate \"card\" {\n    dps = [dp.gateway]\n  }\n}\n",
			needle: `decision "payment"`,
		},
		{
			name:   "candidate without dps",
			source: "decision \"payment\" {\n  addresses = [fr.checkout]\n  candidate \"card\" {\n  }\n}\n",
			needle: `candidate "card"`,
		},
		{
			name:   "pv without realizes",
			source: "pv \"gateway_timeout\" {\n  address = \"payments.gateway.timeout\"\n}\n",
			needle: `pv "gateway_timeout"`,
		},
		{
			name:   "pv with empty realizes",
			source: "pv \"gateway_timeout\" {\n  realizes = []\n}\n",
			needle: "realizes =",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a block whose required nonempty reference list is missing or empty.
			source := tc.source

			// When the core is decoded.
			_, findings := realizationDecodeCore(t, source)

			// Then one structural finding points at the offending line.
			realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, tc.needle, 1))
		})
	}
}

func TestDecodeRealizationSelectedFollowsLabelGrammar(t *testing.T) {
	cases := []struct {
		name     string
		selected string
		valid    bool
	}{
		{name: "lowercase word", selected: "card", valid: true},
		{name: "word with digit segment", selected: "card_2", valid: true},
		{name: "uppercase letter", selected: "Card"},
		{name: "leading digit", selected: "2card"},
		{name: "double underscore", selected: "card__x"},
		{name: "trailing underscore", selected: "card_"},
		{name: "hyphen", selected: "card-x"},
		{name: "empty", selected: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a decision that selects the candidate label under test.
			source := realizationDecisionSelecting(tc.selected)

			// When the core is decoded.
			core, findings := realizationDecodeCore(t, source)

			// Then a valid label is kept exactly and an invalid label is rejected at the selected line.
			if !tc.valid {
				realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "selected", 1))
				return
			}
			realizationRequireNoFindings(t, findings)
			decision := realizationOnly(t, "decision", core.Decisions)
			realizationRequireString(t, "selected", decision.Selected, tc.selected)
		})
	}
}

func TestDecodeRealizationProvisionalDefaultsToTrue(t *testing.T) {
	// Given a decision with a candidate and no provisional attribute.
	source := `decision "payment" {
  addresses = [fr.checkout]

  candidate "card" {
    dps = [dp.gateway]
  }
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then the decision is provisional.
	realizationRequireNoFindings(t, findings)
	decision := realizationOnly(t, "decision", core.Decisions)
	if decision.Provisional == nil || !*decision.Provisional {
		t.Fatalf("provisional = %v, want true by default", decision.Provisional)
	}
}

func TestDecodeRealizationProvisionalRejectsNonBoolean(t *testing.T) {
	// Given a decision whose provisional value is the number 1 instead of a Boolean.
	source := `decision "payment" {
  addresses   = [fr.checkout]
  provisional = 1

  candidate "card" {
    dps = [dp.gateway]
  }
}
`

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the provisional line.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "provisional =", 1))
}

func TestDecodeRealizationDecisionRejectsAlias(t *testing.T) {
	// Given a decision that declares an alias, which only fr, dp, pv, and module admit.
	source := `decision "payment" {
  addresses = [fr.checkout]
  alias     = "D-1"

  candidate "card" {
    dps = [dp.gateway]
  }
}
`

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the alias line.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "alias", 1))
}

func TestDecodeRealizationCandidateAllowsAtMostOneInformation(t *testing.T) {
	// Given a candidate with two unlabeled information blocks.
	source := `decision "payment" {
  addresses = [fr.checkout]

  candidate "card" {
    dps = [dp.gateway]
    information {
      basis = "unknown"
    }
    information {
      basis = "unknown"
    }
  }
}
`

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the second information block.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "information {", 2))
}

func TestDecodeRealizationMeasuredInformationRequiresEvidence(t *testing.T) {
	cases := []struct {
		name   string
		source string
		needle string
	}{
		{
			name:   "omitted evidence",
			source: realizationMeasuredInformation(""),
			needle: "information {",
		},
		{
			name:   "empty evidence list",
			source: realizationMeasuredInformation("      evidence = []\n"),
			needle: "evidence = []",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a measured information block without a nonempty evidence list.
			source := tc.source

			// When the core is decoded.
			_, findings := realizationDecodeCore(t, source)

			// Then one structural finding points at the information evidence.
			realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, tc.needle, 1))
		})
	}
}

func TestDecodeRealizationInformationKeepsEvidenceReferences(t *testing.T) {
	// Given a measured information block that names one evidence block.
	source := `decision "payment" {
  addresses = [fr.checkout]

  candidate "card" {
    dps = [dp.gateway]
    information {
      basis    = "measured"
      evidence = [evidence.e]
    }
  }
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then the information keeps the measured basis and the evidence reference.
	realizationRequireNoFindings(t, findings)
	decision := realizationOnly(t, "decision", core.Decisions)
	information := decision.Candidates[0].Information
	if information == nil {
		t.Fatal("information is absent, want present")
	}
	if string(information.Basis) != "measured" {
		t.Fatalf("basis = %q, want measured", information.Basis)
	}
	realizationRequireRefs(t, "evidence", information.Evidence, "evidence.e")
}

func TestDecodeRealizationInformationBasisIsEnumerated(t *testing.T) {
	// Given an information block whose basis is an estimate, which the enum does not admit.
	source := `decision "payment" {
  addresses = [fr.checkout]

  candidate "card" {
    dps = [dp.gateway]
    information {
      basis = "estimated"
    }
  }
}
`

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the basis line.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "basis =", 1))
}

func TestDecodeRealizationEvidenceDecodesEveryField(t *testing.T) {
	// Given a valid evidence block with every required field.
	source := realizationEvidence(nil, "")

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then each field keeps its exact decoded value.
	realizationRequireNoFindings(t, findings)
	evidence := realizationOnly(t, "evidence", core.Evidence)
	if evidence.ID != "evidence.e" {
		t.Fatalf("evidence ID = %q, want evidence.e", evidence.ID)
	}
	if string(evidence.Basis) != "measured" {
		t.Fatalf("basis = %q, want measured", evidence.Basis)
	}
	if evidence.Event != "fare_paid" {
		t.Fatalf("event = %q, want fare_paid", evidence.Event)
	}
	realizationRequireRef(t, "scenario", &evidence.Scenario, "scenario.checkout")
	realizationRequireRef(t, "candidate", &evidence.Candidate, "decision.payment.candidate.card")
	if evidence.Version != "v1" {
		t.Fatalf("version = %q, want v1", evidence.Version)
	}
	realizationRequireRef(t, "context", &evidence.Context, "context.eu")
	realizationRequireRef(t, "workload", &evidence.Workload, "workload.peak")
	if evidence.Horizon != "30d" {
		t.Fatalf("horizon = %q, want 30d", evidence.Horizon)
	}
	if string(evidence.Dependence) != "joint" {
		t.Fatalf("dependence = %q, want joint", evidence.Dependence)
	}
	if evidence.Sampling != "random" {
		t.Fatalf("sampling = %q, want random", evidence.Sampling)
	}
	if got := realizationExactInt(t, "trials", evidence.Trials); got != 100 {
		t.Fatalf("trials = %d, want 100", got)
	}
	if got := realizationExactInt(t, "failures", evidence.Failures); got != 3 {
		t.Fatalf("failures = %d, want 3", got)
	}
	if string(evidence.ConfidenceMethod) != "wilson" {
		t.Fatalf("confidence method = %q, want wilson", evidence.ConfidenceMethod)
	}
	if evidence.Confidence == nil || evidence.Confidence.Cmp(realizationDecimal(t, "0.95")) != 0 {
		t.Fatalf("confidence = %v, want exact 0.95", evidence.Confidence)
	}
}

func TestDecodeRealizationEvidenceRequiresEveryField(t *testing.T) {
	for _, attribute := range realizationEvidenceAttributes {
		t.Run(attribute.name, func(t *testing.T) {
			// Given an evidence block that omits one required field.
			source := realizationEvidence(nil, attribute.name)

			// When the core is decoded.
			_, findings := realizationDecodeCore(t, source)

			// Then one structural finding points at the evidence block.
			realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, `evidence "e"`, 1))
		})
	}
}

func TestDecodeRealizationEvidenceEnumsAreExact(t *testing.T) {
	cases := []struct {
		name  string
		field string
		value string
	}{
		{name: "basis unknown", field: "basis", value: `"unknown"`},
		{name: "dependence independent", field: "dependence", value: `"independent"`},
		{name: "confidence method normal", field: "confidence_method", value: `"normal"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given an evidence block whose enumerated field holds a value outside its enum.
			source := realizationEvidence(map[string]string{tc.field: tc.value}, "")

			// When the core is decoded.
			_, findings := realizationDecodeCore(t, source)

			// Then one structural finding points at the field.
			realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, tc.field+" =", 1))
		})
	}
}

func TestDecodeRealizationEvidenceTrialsArePositiveIntegers(t *testing.T) {
	cases := []struct {
		name  string
		value string
		valid bool
		want  int64
	}{
		{name: "one", value: "1", valid: true, want: 1},
		{name: "integral decimal", value: "1.0", valid: true, want: 1},
		{name: "integral exponent", value: "1e2", valid: true, want: 100},
		{name: "zero", value: "0"},
		{name: "negative", value: "-3"},
		{name: "fraction", value: "2.5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given an evidence block whose trials value is the case value and whose failures value is zero.
			source := realizationEvidence(map[string]string{"trials": tc.value, "failures": "0"}, "")

			// When the core is decoded.
			core, findings := realizationDecodeCore(t, source)

			// Then a positive integer keeps its exact value and any other value is an evidence finding.
			if !tc.valid {
				realizationRequireFinding(t, findings, "invalid_evidence", realizationCorePath, realizationLine(t, source, "trials =", 1))
				return
			}
			realizationRequireNoFindings(t, findings)
			evidence := realizationOnly(t, "evidence", core.Evidence)
			if got := realizationExactInt(t, "trials", evidence.Trials); got != tc.want {
				t.Fatalf("trials = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDecodeRealizationEvidenceFailuresAreIntegers(t *testing.T) {
	cases := []struct {
		name  string
		value string
		valid bool
		want  int64
	}{
		{name: "zero", value: "0", valid: true, want: 0},
		{name: "two", value: "2", valid: true, want: 2},
		{name: "fraction", value: "1.5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given an evidence block whose failures value is the case value.
			source := realizationEvidence(map[string]string{"failures": tc.value}, "")

			// When the core is decoded.
			core, findings := realizationDecodeCore(t, source)

			// Then an integer keeps its exact value and a fraction is an evidence finding.
			if !tc.valid {
				realizationRequireFinding(t, findings, "invalid_evidence", realizationCorePath, realizationLine(t, source, "failures =", 1))
				return
			}
			realizationRequireNoFindings(t, findings)
			evidence := realizationOnly(t, "evidence", core.Evidence)
			if got := realizationExactInt(t, "failures", evidence.Failures); got != tc.want {
				t.Fatalf("failures = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDecodeRealizationEvidenceCountsRejectStrings(t *testing.T) {
	cases := []struct {
		name  string
		field string
		value string
	}{
		{name: "trials as string", field: "trials", value: `"10"`},
		{name: "confidence as string", field: "confidence", value: `"0.9"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given an evidence block whose numeric field is a string.
			source := realizationEvidence(map[string]string{tc.field: tc.value}, "")

			// When the core is decoded.
			_, findings := realizationDecodeCore(t, source)

			// Then one structural finding points at the field.
			realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, tc.field+" =", 1))
		})
	}
}

func TestDecodeRealizationEvidenceConfidenceIsKeptExact(t *testing.T) {
	// Given an evidence block whose confidence has more digits than a float64 can keep.
	const literal = "0.1234567890123456789012345678901234567890"
	source := realizationEvidence(map[string]string{"confidence": literal}, "")

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then the decoded confidence equals the decimal literal exactly.
	realizationRequireNoFindings(t, findings)
	evidence := realizationOnly(t, "evidence", core.Evidence)
	if evidence.Confidence == nil || evidence.Confidence.Cmp(realizationDecimal(t, literal)) != 0 {
		t.Fatalf("confidence = %v, want exact literal %s", evidence.Confidence, literal)
	}
}

func TestDecodeRealizationEvidenceConfidenceChecksOnlyType(t *testing.T) {
	// Given an evidence block whose confidence is the number 1.5, outside the unit interval.
	source := realizationEvidence(map[string]string{"confidence": "1.5"}, "")

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then decoding accepts the number because the range rule belongs to assessment.
	realizationRequireNoFindings(t, findings)
	evidence := realizationOnly(t, "evidence", core.Evidence)
	if evidence.Confidence == nil || evidence.Confidence.Cmp(realizationDecimal(t, "1.5")) != 0 {
		t.Fatalf("confidence = %v, want 1.5", evidence.Confidence)
	}
}

func TestDecodeRealizationPVDecodesAuthoredFields(t *testing.T) {
	// Given a pv with realizes, kind, address, note, alias, and cites.
	source := `pv "gateway_timeout" {
  realizes = [dp.gateway]
  kind     = "timeout"
  address  = "payments.gateway.timeout"
  note     = "Seconds"
  alias    = "PV-7"
  cites    = [source.spec]
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then each authored value is kept.
	realizationRequireNoFindings(t, findings)
	pv := realizationOnly(t, "pv", core.PVs)
	if pv.ID != "pv.gateway_timeout" {
		t.Fatalf("pv ID = %q, want pv.gateway_timeout", pv.ID)
	}
	realizationRequireList(t, "realizes", realizationTargetsOf(pv.Realizes), "dp.gateway")
	realizationRequireString(t, "kind", pv.Kind, "timeout")
	realizationRequireString(t, "address", pv.Address, "payments.gateway.timeout")
	realizationRequireString(t, "note", pv.Note, "Seconds")
	realizationRequireString(t, "alias", pv.Alias, "PV-7")
	realizationRequireList(t, "cites", realizationTargetsOf(pv.Cites), "source.spec")
}

func TestDecodeRealizationPVKeepsAbsentOptionalFields(t *testing.T) {
	// Given a pv that declares only its realizes list.
	source := "pv \"gateway_timeout\" {\n  realizes = [dp.gateway]\n}\n"

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then the omitted kind, address, note, and alias stay absent.
	realizationRequireNoFindings(t, findings)
	pv := realizationOnly(t, "pv", core.PVs)
	realizationRequireAbsent(t, "kind", pv.Kind)
	realizationRequireAbsent(t, "address", pv.Address)
	realizationRequireAbsent(t, "note", pv.Note)
	realizationRequireAbsent(t, "alias", pv.Alias)
}

func TestDecodeRealizationModuleDecodesAuthoredFields(t *testing.T) {
	// Given a realization module that uses every optional list and scalar.
	source := `module "gateway" {
  realizes          = [dp.gateway]
  within            = module.checkout
  inputs            = ["order"]
  outputs           = ["receipt"]
  owns_state        = [state.session]
  dependencies      = [module.ledger]
  invariants        = ["One charge per order."]
  failure_behavior  = "Retry once."
  acceptance_checks = [scenario.retry, fr.checkout.criterion.timely]
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then each authored list and scalar keeps its exact value.
	realizationRequireNoFindings(t, findings)
	module := realizationOnly(t, "module", core.Modules)
	if module.ID != "module.gateway" {
		t.Fatalf("module ID = %q, want module.gateway", module.ID)
	}
	realizationRequireOptional(t, "realizes", module.Realizes, true, 1)
	realizationRequireRefs(t, "realizes", module.Realizes.Items, "dp.gateway")
	realizationRequireOptional(t, "contains", module.Contains, false, 0)
	realizationRequireRef(t, "within", module.Within, "module.checkout")
	realizationRequireOptional(t, "inputs", module.Inputs, true, 1)
	realizationRequireList(t, "inputs", module.Inputs.Items, "order")
	realizationRequireOptional(t, "outputs", module.Outputs, true, 1)
	realizationRequireList(t, "outputs", module.Outputs.Items, "receipt")
	realizationRequireOptional(t, "owns_state", module.OwnsState, true, 1)
	realizationRequireRefs(t, "owns_state", module.OwnsState.Items, "state.session")
	realizationRequireOptional(t, "dependencies", module.Dependencies, true, 1)
	realizationRequireRefs(t, "dependencies", module.Dependencies.Items, "module.ledger")
	realizationRequireOptional(t, "invariants", module.Invariants, true, 1)
	realizationRequireList(t, "invariants", module.Invariants.Items, "One charge per order.")
	realizationRequireString(t, "failure behavior", module.FailureBehavior, "Retry once.")
	realizationRequireOptional(t, "acceptance checks", module.AcceptanceChecks, true, 2)
	realizationRequireRefs(t, "acceptance checks", module.AcceptanceChecks.Items, "scenario.retry", "fr.checkout.criterion.timely")
}

func TestDecodeRealizationModuleKeepsExplicitEmptyListsPresent(t *testing.T) {
	// Given a module whose optional lists are explicitly empty.
	source := `module "gateway" {
  realizes          = [dp.gateway]
  inputs            = []
  outputs           = []
  owns_state        = []
  dependencies      = []
  invariants        = []
  acceptance_checks = []
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then each explicit empty list is present with no items.
	realizationRequireNoFindings(t, findings)
	module := realizationOnly(t, "module", core.Modules)
	realizationRequireOptional(t, "inputs", module.Inputs, true, 0)
	realizationRequireOptional(t, "outputs", module.Outputs, true, 0)
	realizationRequireOptional(t, "owns_state", module.OwnsState, true, 0)
	realizationRequireOptional(t, "dependencies", module.Dependencies, true, 0)
	realizationRequireOptional(t, "invariants", module.Invariants, true, 0)
	realizationRequireOptional(t, "acceptance checks", module.AcceptanceChecks, true, 0)
}

func TestDecodeRealizationModuleKeepsOmittedListsAbsent(t *testing.T) {
	// Given a module that omits every optional list.
	source := "module \"gateway\" {\n  realizes = [dp.gateway]\n}\n"

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then each omitted list stays absent.
	realizationRequireNoFindings(t, findings)
	module := realizationOnly(t, "module", core.Modules)
	realizationRequireOptional(t, "contains", module.Contains, false, 0)
	realizationRequireOptional(t, "inputs", module.Inputs, false, 0)
	realizationRequireOptional(t, "outputs", module.Outputs, false, 0)
	realizationRequireOptional(t, "owns_state", module.OwnsState, false, 0)
	realizationRequireOptional(t, "dependencies", module.Dependencies, false, 0)
	realizationRequireOptional(t, "invariants", module.Invariants, false, 0)
	realizationRequireOptional(t, "acceptance checks", module.AcceptanceChecks, false, 0)
}

func TestDecodeRealizationModuleRejectsRealizesAndContains(t *testing.T) {
	// Given a module that declares both realizes and contains.
	source := `module "gateway" {
  realizes = [dp.gateway]
  contains = [module.inner]
}
`

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the module block.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, `module "gateway"`, 1))
}

func TestDecodeRealizationModuleRejectsEmptyRealizationLists(t *testing.T) {
	cases := []struct {
		name   string
		source string
		needle string
	}{
		{
			name:   "empty realizes",
			source: "module \"gateway\" {\n  realizes = []\n}\n",
			needle: "realizes =",
		},
		{
			name:   "empty contains",
			source: "module \"gateway\" {\n  contains = []\n}\n",
			needle: "contains =",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a module whose realization list is empty where a nonempty list is required.
			source := tc.source

			// When the core is decoded.
			_, findings := realizationDecodeCore(t, source)

			// Then one structural finding points at the empty list.
			realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, tc.needle, 1))
		})
	}
}

func TestDecodeRealizationFlowAcceptsSingleRefAndFlatGroupSteps(t *testing.T) {
	// Given a flow with a single-ref step, a flat group step, and a module step, plus owner, text, after, and a loop.
	source := `flow "checkout" {
  steps  = [fr.checkout, [fr.retry, dp.gateway], module.gateway]
  owner  = module.gateway
  input  = "order"
  output = "receipt"
  after  = [flow.intake]

  loop {
    after   = fr.retry
    back_to = fr.checkout
    when    = "gateway times out"
    bound   = 3
  }
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then each step keeps its position and refs, and the other fields keep their values.
	realizationRequireNoFindings(t, findings)
	flow := realizationOnly(t, "flow", core.Flows)
	if len(flow.Steps) != 3 {
		t.Fatalf("steps = %d, want 3", len(flow.Steps))
	}
	realizationRequireRefs(t, "step 1", flow.Steps[0], "fr.checkout")
	realizationRequireRefs(t, "step 2", flow.Steps[1], "fr.retry", "dp.gateway")
	realizationRequireRefs(t, "step 3", flow.Steps[2], "module.gateway")
	realizationRequireRef(t, "owner", flow.Owner, "module.gateway")
	realizationRequireString(t, "input", flow.Input, "order")
	realizationRequireString(t, "output", flow.Output, "receipt")
	realizationRequireRefs(t, "after", flow.After, "flow.intake")
	if len(flow.Loops) != 1 {
		t.Fatalf("loops = %d, want 1", len(flow.Loops))
	}
	loop := flow.Loops[0]
	realizationRequireRefs(t, "loop after", []model.Ref{loop.After}, "fr.retry")
	realizationRequireRefs(t, "loop back_to", []model.Ref{loop.BackTo}, "fr.checkout")
	if loop.When != "gateway times out" {
		t.Fatalf("loop when = %q, want gateway times out", loop.When)
	}
	if got := realizationExactInt(t, "loop bound", loop.Bound); got != 3 {
		t.Fatalf("loop bound = %d, want 3", got)
	}
}

func TestDecodeRealizationFlowRejectsMalformedSteps(t *testing.T) {
	cases := []struct {
		name  string
		steps string
	}{
		{name: "empty steps", steps: "[]"},
		{name: "scalar instead of list", steps: "fr.checkout"},
		{name: "string step", steps: `[fr.checkout, "fr.retry"]`},
		{name: "empty group", steps: "[fr.checkout, []]"},
		{name: "deeper nesting", steps: "[fr.checkout, [[fr.retry]]]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a flow whose steps value is malformed.
			source := realizationFlowWithSteps(tc.steps)

			// When the core is decoded.
			_, findings := realizationDecodeCore(t, source)

			// Then one structural finding points at the steps line.
			realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "steps =", 1))
		})
	}
}

func TestDecodeRealizationFlowRejectsDuplicateStepIdentity(t *testing.T) {
	// Given a flow that names fr.checkout twice, once inside a group.
	source := realizationFlowWithSteps("[fr.checkout, [fr.retry, fr.checkout]]")

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the flow steps.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "steps =", 1))
}

func TestDecodeRealizationLoopBoundIsPositiveInteger(t *testing.T) {
	cases := []struct {
		name  string
		bound string
		valid bool
		want  int64
	}{
		{name: "one", bound: "1", valid: true, want: 1},
		{name: "three", bound: "3", valid: true, want: 3},
		{name: "zero", bound: "0"},
		{name: "negative", bound: "-1"},
		{name: "fraction", bound: "2.5"},
		{name: "string", bound: `"3"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a loop whose bound is the case value.
			source := realizationLoopSource(map[string]string{"bound": tc.bound}, "")

			// When the core is decoded.
			core, findings := realizationDecodeCore(t, source)

			// Then a positive integer keeps its exact value and any other value is a structural finding.
			if !tc.valid {
				realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "bound =", 1))
				return
			}
			realizationRequireNoFindings(t, findings)
			flow := realizationOnly(t, "flow", core.Flows)
			if got := realizationExactInt(t, "loop bound", flow.Loops[0].Bound); got != tc.want {
				t.Fatalf("loop bound = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDecodeRealizationLoopRequiresEveryField(t *testing.T) {
	for _, attribute := range realizationLoopAttributes {
		t.Run(attribute.name, func(t *testing.T) {
			// Given a loop that omits one required field.
			source := realizationLoopSource(nil, attribute.name)

			// When the core is decoded.
			_, findings := realizationDecodeCore(t, source)

			// Then one structural finding points at the loop block.
			realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "loop {", 1))
		})
	}
}

func TestDecodeRealizationScenarioKindIsEnumerated(t *testing.T) {
	cases := []struct {
		name  string
		kind  string
		valid bool
		want  string
	}{
		{name: "change", kind: `"change"`, valid: true, want: "change"},
		{name: "operational", kind: `"operational"`, valid: true, want: "operational"},
		{name: "failure", kind: `"failure"`, valid: true, want: "failure"},
		{name: "capitalized", kind: `"Change"`},
		{name: "unlisted word", kind: `"manual"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a scenario whose kind is the case value.
			source := realizationScenarioWithKind(tc.kind)

			// When the core is decoded.
			core, findings := realizationDecodeCore(t, source)

			// Then a listed kind is kept and any other kind is a structural finding at the kind line.
			if !tc.valid {
				realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "kind =", 1))
				return
			}
			realizationRequireNoFindings(t, findings)
			scenario := realizationOnly(t, "scenario", core.Scenarios)
			if string(scenario.Kind) != tc.want {
				t.Fatalf("kind = %q, want %q", scenario.Kind, tc.want)
			}
		})
	}
}

func TestDecodeRealizationScenarioRequiresKind(t *testing.T) {
	// Given a scenario with no kind.
	source := "scenario \"retry\" {\n  description = \"Retry after timeout.\"\n}\n"

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the scenario block.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, `scenario "retry"`, 1))
}

func TestDecodeRealizationScenarioDecodesAuthoredFields(t *testing.T) {
	// Given a failure scenario that uses every optional field.
	source := `scenario "retry" {
  kind              = "failure"
  verifies          = [fr.checkout.criterion.timely]
  vary              = [dp.gateway]
  expect_affected   = [fr.checkout]
  expect_unaffected = [fr.billing]
  steps             = ["Make the gateway time out.", "Submit an order."]
  description       = "Retry after timeout."
  under             = context.eu
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then each authored field keeps its exact value.
	realizationRequireNoFindings(t, findings)
	scenario := realizationOnly(t, "scenario", core.Scenarios)
	if scenario.ID != "scenario.retry" {
		t.Fatalf("scenario ID = %q, want scenario.retry", scenario.ID)
	}
	if string(scenario.Kind) != "failure" {
		t.Fatalf("kind = %q, want failure", scenario.Kind)
	}
	realizationRequireOptional(t, "verifies", scenario.Verifies, true, 1)
	realizationRequireRefs(t, "verifies", scenario.Verifies.Items, "fr.checkout.criterion.timely")
	realizationRequireRefs(t, "vary", scenario.Vary, "dp.gateway")
	realizationRequireRefs(t, "expect affected", scenario.ExpectAffected, "fr.checkout")
	realizationRequireRefs(t, "expect unaffected", scenario.ExpectUnaffected, "fr.billing")
	realizationRequireOptional(t, "steps", scenario.Steps, true, 2)
	realizationRequireList(t, "steps", scenario.Steps.Items, "Make the gateway time out.", "Submit an order.")
	realizationRequireString(t, "description", scenario.Description, "Retry after timeout.")
	realizationRequireRef(t, "under", scenario.Under, "context.eu")
}

func TestDecodeRealizationScenarioKeepsExplicitEmptyListsPresent(t *testing.T) {
	// Given a scenario whose verifies and steps lists are explicitly empty.
	source := `scenario "retry" {
  kind     = "failure"
  verifies = []
  steps    = []
}
`

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then both lists are present with no items.
	realizationRequireNoFindings(t, findings)
	scenario := realizationOnly(t, "scenario", core.Scenarios)
	realizationRequireOptional(t, "verifies", scenario.Verifies, true, 0)
	realizationRequireOptional(t, "steps", scenario.Steps, true, 0)
}

func TestDecodeRealizationScenarioKeepsOmittedListsAbsent(t *testing.T) {
	// Given a scenario that omits verifies, steps, description, and under.
	source := "scenario \"retry\" {\n  kind = \"failure\"\n}\n"

	// When the core is decoded.
	core, findings := realizationDecodeCore(t, source)

	// Then the omitted fields stay absent.
	realizationRequireNoFindings(t, findings)
	scenario := realizationOnly(t, "scenario", core.Scenarios)
	realizationRequireOptional(t, "verifies", scenario.Verifies, false, 0)
	realizationRequireOptional(t, "steps", scenario.Steps, false, 0)
	realizationRequireAbsent(t, "description", scenario.Description)
	realizationRequireAbsent(t, "under", scenario.Under)
}

func TestDecodeRealizationScenarioRejectsSameFRInBothExpectLists(t *testing.T) {
	// Given a scenario that names fr.checkout as both affected and unaffected.
	source := `scenario "retry" {
  kind              = "failure"
  expect_affected   = [fr.checkout]
  expect_unaffected = [fr.checkout]
}
`

	// When the core is decoded.
	_, findings := realizationDecodeCore(t, source)

	// Then one structural finding points at the unaffected list.
	realizationRequireFinding(t, findings, "invalid_structure", realizationCorePath, realizationLine(t, source, "expect_unaffected =", 1))
}

func TestDecodeRealizationManifestDecodesProjectAndExtension(t *testing.T) {
	// Given a manifest with a core project that has one core file and no attachment files, and one binding extension.
	source := realizationProject(realizationValidProjectLanguage, realizationValidCoreFiles, realizationValidAttachmentFiles) +
		"\n" +
		realizationExtension(realizationValidRole, realizationValidContract)

	// When the manifest is decoded.
	manifest, findings := realizationDecodeManifest(t, source)

	// Then the project keeps its language and listed paths, and the extension keeps its role and contract.
	realizationRequireNoFindings(t, findings)
	if manifest.Project == nil {
		t.Fatal("project is absent, want present")
	}
	project := manifest.Project
	if project.Label != "checkout" {
		t.Fatalf("project label = %q, want checkout", project.Label)
	}
	if project.Language != model.LanguageCore1 {
		t.Fatalf("language = %q, want requirements_hcl/core/1", project.Language)
	}
	if len(project.CoreFiles) != 2 {
		t.Fatalf("core files = %d, want 2", len(project.CoreFiles))
	}
	if project.CoreFiles[0].Path != "design.ad.hcl" || project.CoreFiles[1].Path != "checkout/model.ad.hcl" {
		t.Fatalf("core file paths = %q, %q", project.CoreFiles[0].Path, project.CoreFiles[1].Path)
	}
	wantLine := realizationLine(t, source, "core_files =", 1)
	if project.CoreFiles[0].Range.Start.Line != wantLine {
		t.Fatalf("core file range line = %d, want %d", project.CoreFiles[0].Range.Start.Line, wantLine)
	}
	if len(project.AttachmentFiles) != 0 {
		t.Fatalf("attachment files = %d, want 0", len(project.AttachmentFiles))
	}
	if len(manifest.Extensions) != 1 {
		t.Fatalf("extensions = %d, want 1", len(manifest.Extensions))
	}
	extension := manifest.Extensions[0]
	if string(extension.ID) != "extension.terraform_binding" {
		t.Fatalf("extension ID = %q, want extension.terraform_binding", extension.ID)
	}
	if extension.Role != model.ExtensionRoleBinding {
		t.Fatalf("role = %q, want binding", extension.Role)
	}
	if extension.Contract != "requirements_hcl/terraform_binding/1" {
		t.Fatalf("contract = %q", extension.Contract)
	}
}

func TestDecodeRealizationManifestLanguageIsEnumerated(t *testing.T) {
	// Given a project whose language names a core contract version that does not exist.
	source := realizationProject(
		`  language         = "requirements_hcl/core/2"`,
		realizationValidCoreFiles,
		realizationValidAttachmentFiles,
	)

	// When the manifest is decoded.
	_, findings := realizationDecodeManifest(t, source)

	// Then one structural finding points at the language line.
	realizationRequireFinding(t, findings, "invalid_structure", realizationManifestPath, realizationLine(t, source, "language =", 1))
}

func TestDecodeRealizationManifestCoreFilesMustBeNonempty(t *testing.T) {
	cases := []struct {
		name   string
		source string
		needle string
	}{
		{
			name: "empty core files",
			source: realizationProject(
				realizationValidProjectLanguage,
				"  core_files       = []",
				realizationValidAttachmentFiles,
			),
			needle: "core_files =",
		},
		{
			name: "omitted core files",
			source: realizationProject(
				realizationValidProjectLanguage,
				realizationValidAttachmentFiles,
			),
			needle: `project "checkout"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a project whose core file list is empty or missing.
			source := tc.source

			// When the manifest is decoded.
			_, findings := realizationDecodeManifest(t, source)

			// Then one structural finding points at the project element.
			realizationRequireFinding(t, findings, "invalid_structure", realizationManifestPath, realizationLine(t, source, tc.needle, 1))
		})
	}
}

func TestDecodeRealizationManifestAllowsEmptyAttachmentFiles(t *testing.T) {
	// Given a project with an explicitly empty attachment file list.
	source := realizationProject(realizationValidProjectLanguage, realizationValidCoreFiles, realizationValidAttachmentFiles)

	// When the manifest is decoded.
	manifest, findings := realizationDecodeManifest(t, source)

	// Then no finding is reported and the attachment list is empty.
	realizationRequireNoFindings(t, findings)
	if manifest.Project == nil {
		t.Fatal("project is absent, want present")
	}
	if len(manifest.Project.AttachmentFiles) != 0 {
		t.Fatalf("attachment files = %d, want 0", len(manifest.Project.AttachmentFiles))
	}
}

func TestDecodeRealizationManifestRequiresAttachmentFilesList(t *testing.T) {
	// Given a project that omits attachment_files entirely.
	source := realizationProject(realizationValidProjectLanguage, realizationValidCoreFiles)

	// When the manifest is decoded.
	_, findings := realizationDecodeManifest(t, source)

	// Then one structural finding points at the project block.
	realizationRequireFinding(t, findings, "invalid_structure", realizationManifestPath, realizationLine(t, source, `project "checkout"`, 1))
}

func TestDecodeRealizationManifestRejectsSecondProject(t *testing.T) {
	// Given a manifest with two project blocks.
	source := realizationProject(realizationValidProjectLanguage, realizationValidCoreFiles, realizationValidAttachmentFiles) +
		"project \"other\" {\n" +
		realizationValidProjectLanguage + "\n" +
		realizationValidCoreFiles + "\n" +
		realizationValidAttachmentFiles + "\n" +
		"}\n"

	// When the manifest is decoded.
	_, findings := realizationDecodeManifest(t, source)

	// Then one structural finding points at the second project block.
	realizationRequireFinding(t, findings, "invalid_structure", realizationManifestPath, realizationLine(t, source, `project "other"`, 1))
}

func TestDecodeRealizationManifestRequiresProject(t *testing.T) {
	// Given a manifest that declares an extension but no project.
	source := realizationExtension(realizationValidRole, realizationValidContract)

	// When the manifest is decoded.
	manifest, findings := realizationDecodeManifest(t, source)

	// Then no project is decoded and one structural finding is reported.
	if manifest.Project != nil {
		t.Fatal("project is present, want absent")
	}
	if len(findings) != 1 || findings[0].Code != "invalid_structure" {
		t.Fatalf("findings = %v, want one invalid_structure", realizationCodes(findings))
	}
}

func TestDecodeRealizationManifestExtensionRoleIsEnumerated(t *testing.T) {
	// Given an extension whose role is outside the owner role enum.
	source := realizationProject(realizationValidProjectLanguage, realizationValidCoreFiles, realizationValidAttachmentFiles) +
		"\n" +
		realizationExtension(`  role     = "other"`, realizationValidContract)

	// When the manifest is decoded.
	_, findings := realizationDecodeManifest(t, source)

	// Then one structural finding points at the role line.
	realizationRequireFinding(t, findings, "invalid_structure", realizationManifestPath, realizationLine(t, source, "role =", 1))
}

func TestDecodeRealizationManifestExtensionRequiresContract(t *testing.T) {
	// Given an extension that declares a role but no contract.
	source := realizationProject(realizationValidProjectLanguage, realizationValidCoreFiles, realizationValidAttachmentFiles) +
		"\n" +
		realizationExtension(realizationValidRole)

	// When the manifest is decoded.
	_, findings := realizationDecodeManifest(t, source)

	// Then one structural finding points at the extension block.
	realizationRequireFinding(t, findings, "invalid_structure", realizationManifestPath, realizationLine(t, source, `extension "terraform_binding"`, 1))
}

func TestDecodeRealizationManifestRejectsExtraTopLevelBlock(t *testing.T) {
	// Given a manifest with a design block, which a manifest does not admit.
	source := realizationProject(realizationValidProjectLanguage, realizationValidCoreFiles, realizationValidAttachmentFiles) +
		"\n" +
		"design \"d\" {\n}\n"

	// When the manifest is decoded.
	_, findings := realizationDecodeManifest(t, source)

	// Then one structural finding points at the extra block.
	realizationRequireFinding(t, findings, "invalid_structure", realizationManifestPath, realizationLine(t, source, `design "d"`, 1))
}

func TestDecodeRealizationAttachmentDecodesEnvelope(t *testing.T) {
	// Given an attachment with owner, subject, and a two-field literal data object.
	source := `attachment "pv_binding" {
  owner   = extension.terraform_binding
  subject = pv.gateway_timeout
  data = {
    address            = "payments.gateway.timeout"
    observation_source = "obs.csv"
  }
}
`

	// When the attachments are decoded.
	attachments, findings := realizationDecodeAttachments(t, source)

	// Then the envelope keeps its owner, subject, and data in authored order.
	realizationRequireNoFindings(t, findings)
	if len(attachments) != 1 {
		t.Fatalf("attachments = %d, want 1", len(attachments))
	}
	attachment := attachments[0]
	if attachment.Label != "pv_binding" {
		t.Fatalf("label = %q, want pv_binding", attachment.Label)
	}
	realizationRequireRefs(t, "owner", []model.Ref{attachment.Owner}, "extension.terraform_binding")
	realizationRequireRefs(t, "subject", []model.Ref{attachment.Subject}, "pv.gateway_timeout")
	if len(attachment.Data) != 2 {
		t.Fatalf("data fields = %d, want 2", len(attachment.Data))
	}
	first := attachment.Data[0]
	if first.Key != "address" || !first.Value.RawEquals(cty.StringVal("payments.gateway.timeout")) {
		t.Fatalf("first field = %q %#v, want address string", first.Key, first.Value)
	}
	if first.KeyRange.Start.Line != realizationLine(t, source, "address", 1) {
		t.Fatalf("first key line = %d, want the address line", first.KeyRange.Start.Line)
	}
	second := attachment.Data[1]
	if second.Key != "observation_source" || !second.Value.RawEquals(cty.StringVal("obs.csv")) {
		t.Fatalf("second field = %q %#v, want observation_source string", second.Key, second.Value)
	}
}

func TestDecodeRealizationAttachmentRequiresExactlyThreeAttributes(t *testing.T) {
	cases := []struct {
		name   string
		source string
		needle string
	}{
		{
			name:   "owner omitted",
			source: realizationAttachmentSource(realizationSubjectLine, realizationDataLine),
			needle: `attachment "pv_binding"`,
		},
		{
			name:   "subject omitted",
			source: realizationAttachmentSource(realizationOwnerLine, realizationDataLine),
			needle: `attachment "pv_binding"`,
		},
		{
			name:   "data omitted",
			source: realizationAttachmentSource(realizationOwnerLine, realizationSubjectLine),
			needle: `attachment "pv_binding"`,
		},
		{
			name:   "extra attribute",
			source: realizationAttachmentSource(realizationOwnerLine, realizationSubjectLine, realizationDataLine, `  note    = "extra"`),
			needle: "note =",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given an attachment envelope that is missing one attribute or has an extra attribute.
			source := tc.source

			// When the attachments are decoded.
			_, findings := realizationDecodeAttachments(t, source)

			// Then one attachment finding points at the envelope defect.
			realizationRequireFinding(t, findings, "invalid_attachment", realizationExtPath, realizationLine(t, source, tc.needle, 1))
		})
	}
}

func TestDecodeRealizationAttachmentOwnerMustBeReference(t *testing.T) {
	// Given an attachment whose owner is a string instead of an extension reference.
	source := realizationAttachmentSource(
		`  owner   = "extension.terraform_binding"`,
		realizationSubjectLine,
		realizationDataLine,
	)

	// When the attachments are decoded.
	_, findings := realizationDecodeAttachments(t, source)

	// Then one attachment finding points at the owner line.
	realizationRequireFinding(t, findings, "invalid_attachment", realizationExtPath, realizationLine(t, source, "owner =", 1))
}

func TestDecodeRealizationAttachmentDataRejectsTraversalAndNull(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{name: "traversal value", data: `  data    = { address = pv.gateway_timeout }`},
		{name: "null value", data: `  data    = { address = null }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given an attachment whose data value is not a literal.
			source := realizationAttachmentSource(realizationOwnerLine, realizationSubjectLine, tc.data)

			// When the attachments are decoded.
			_, findings := realizationDecodeAttachments(t, source)

			// Then one attachment finding points at the data line.
			realizationRequireFinding(t, findings, "invalid_attachment", realizationExtPath, realizationLine(t, source, "data =", 1))
		})
	}
}

func TestDecodeRealizationAttachmentRejectsChildBlock(t *testing.T) {
	// Given an attachment that contains a child block.
	source := realizationAttachmentSource(
		realizationOwnerLine,
		realizationSubjectLine,
		realizationDataLine,
		`  extra "x" {}`,
	)

	// When the attachments are decoded.
	_, findings := realizationDecodeAttachments(t, source)

	// Then one attachment finding points at the child block.
	realizationRequireFinding(t, findings, "invalid_attachment", realizationExtPath, realizationLine(t, source, `extra "x"`, 1))
}

func TestDecodeRealizationAttachmentDataPayloadIsNotJudged(t *testing.T) {
	// Given an attachment whose data object holds a key that no owner payload defines.
	source := realizationAttachmentSource(
		realizationOwnerLine,
		realizationSubjectLine,
		`  data    = { anything = "accepted here" }`,
	)

	// When the attachments are decoded.
	attachments, findings := realizationDecodeAttachments(t, source)

	// Then the envelope decodes because owner payload rules belong to a later stage.
	realizationRequireNoFindings(t, findings)
	attachment := realizationOnly(t, "attachment", []model.Attachment(attachments))
	if len(attachment.Data) != 1 || attachment.Data[0].Key != "anything" {
		t.Fatalf("data = %#v, want one anything field", attachment.Data)
	}
}

func TestDecodeRealizationStarterFixturesDecodeWithoutFindings(t *testing.T) {
	fixtures := []string{
		"../../testdata/spec/examples/starter/design.ad.hcl",
		"../../testdata/spec/examples/requirements_hcl.ad.hcl",
	}
	for _, path := range fixtures {
		t.Run(path, func(t *testing.T) {
			// Given a pinned spec fixture read from disk.
			bytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			// When the fixture is parsed and decoded as a core file.
			unit := syntax.Parse(model.Source{Path: path, Bytes: bytes})
			_, findings := DecodeCore([]syntax.Unit{unit})

			// Then parsing and decoding report no findings.
			if len(unit.Findings) != 0 {
				t.Fatalf("syntax findings = %v, want none", realizationCodes(unit.Findings))
			}
			realizationRequireNoFindings(t, findings)
		})
	}
}
