package owners

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
	"github.com/axiomatic-design-hcl/adhcl/internal/result"
	"github.com/axiomatic-design-hcl/adhcl/internal/schema"
	"github.com/axiomatic-design-hcl/adhcl/internal/selection"
	"github.com/axiomatic-design-hcl/adhcl/internal/syntax"
	"github.com/zclconf/go-cty/cty"
)

const (
	coreFile     = "design.ad.hcl"
	manifestFile = "model.ad.project.hcl"
	bindingFile  = "bindings.ad.ext.hcl"
	behaviorFile = "behavior.ad.ext.hcl"
	viewFile     = "views.ad.ext.hcl"
)

const (
	bindingOwnerID  = model.ID("extension.terraform_binding")
	behaviorOwnerID = model.ID("extension.event_model_link")
	viewOwnerID     = model.ID("extension.html_view")

	expectedBindingCause  = "No caller-supplied observations are available."
	expectedBehaviorCause = "No external behavior model at the exact requested revision is available."

	invalidLabel = "payload_case"
)

// receiptCoreSource is the starter design with one selected candidate.
const receiptCoreSource = `# Proposed architecture, not implemented code or measured evidence.
# This semantic-change range excludes process failure and resource exhaustion.
design "receipt" {
  boundary = "Admit one amount in integer cents and calculate its total with a configured fee. Payment, storage, deployment, and process availability are outside the boundary."
  level = "architecture"
  workload = workload.receipt_inputs
}

workload "receipt_inputs" {
  description = "One amount and one fee per request. Inputs outside the admitted amount range must be rejected. A fee is fixed for a request. No latency or availability target is claimed."
  ranges = {
    amount_cents = { min = 0, max = 100000, unit = "cents" }
    fee_cents = { min = 0, max = 1000, unit = "cents" }
  }
}

assumption "fixed_contracts" {
  statement = "During a semantic change, the integer-cent input contract and immutable admission result remain fixed. The total module cannot change admission policy or rewrite its result. Shared process failures are outside this change range."
  status = "to_validate"
}

need "correct_receipt" {
  statement = "Receive a correct total without treating invalid amounts as accepted amounts."
  why = "A caller must distinguish invalid input from a valid receipt."
  origin = "inferred"
}

fr "admit_amount" {
  need = [need.correct_receipt]
  statement = "Accept exactly the permitted integer-cent amounts."
  criterion "admission" {
    metric = "An amount is admitted if and only if it is an integer from 0 through 100000 inclusive; every other input returns invalid_amount and no admitted amount."
    op = "=="
    value = true
    origin = "proposed"
  }
}

fr "calculate_total" {
  need = [need.correct_receipt]
  statement = "Return the exact total for an admitted amount, or preserve the invalid-input result."
  criterion "total" {
    metric = "For every admitted amount and integer fee from 0 through 1000 inclusive, total_cents equals amount_cents plus fee_cents exactly. An invalid admission returns invalid_amount and no total."
    op = "=="
    value = true
    origin = "proposed"
  }
}

state "admitted_amount" {
  description = "The request-local immutable admitted integer amount or invalid_amount result."
}

state "total_result" {
  description = "The request-local total or invalid_amount result."
}

dp "amount_admission" {
  controls = [fr.admit_amount]
  responsibility = "Check integer type and the inclusive amount range before publishing an immutable admission result."
  decision = "selected"
  owns_state = [state.admitted_amount]
  holds_if = [assumption.fixed_contracts]
}

dp "exact_addition" {
  controls = [fr.calculate_total]
  responsibility = "Read the immutable admission result, propagate invalid_amount, and otherwise add the request's integer fee without rounding or overflow in the stated range."
  decision = "selected"
  owns_state = [state.total_result]
  reads_state = [state.admitted_amount]
  holds_if = [assumption.fixed_contracts]
}

influence {
  from = [dp.amount_admission]
  to = fr.admit_amount
  effect = "material"
  mechanism = "Admission policy determines which inputs are accepted or rejected."
  mechanism_source = "inferred"
}

influence {
  from = [dp.exact_addition]
  to = fr.calculate_total
  effect = "material"
  mechanism = "The addition and invalid-result propagation rules determine whether the returned total satisfies its exact predicate."
  mechanism_source = "inferred"
}

influence {
  from = [dp.amount_admission]
  to = fr.calculate_total
  effect = "material"
  mechanism = "Accepting a noninteger or rejecting a permitted amount changes whether a correct receipt can be returned. A changed admission contract requires renewed total acceptance checks."
  mechanism_source = "inferred"
}

cells "fixed_admission" {
  scope = design.receipt
  unlisted = "none"
  because = "Within the fixed immutable-result contract, changing the downstream addition policy cannot alter amount admission. This default states a proposed semantic boundary, not measured isolation."
}

pv "admission_policy" {
  realizes = [dp.amount_admission]
  kind = "configuration"
  note = "A proposed realization subject for the binding attachment. No implementation or Terraform observation is supplied."
}

pv "total_routine" {
  realizes = [dp.exact_addition]
  kind = "routine"
  note = "A proposed exact integer addition routine. No code or execution evidence is supplied."
}

module "admission" {
  realizes = [dp.amount_admission]
  inputs = ["One untrusted amount value."]
  outputs = ["An immutable admitted integer amount or invalid_amount without an amount."]
  owns_state = [state.admitted_amount]
  dependencies = []
  invariants = ["No invalid value becomes an admitted amount.", "A published admission result cannot be rewritten by consumers."]
  failure_behavior = "Reject values outside the integer-cent contract with invalid_amount. Do not supply an amount after rejection. Process availability is outside this contract."
  acceptance_checks = [fr.admit_amount.criterion.admission, scenario.amount_boundaries]
}

module "total" {
  realizes = [dp.exact_addition]
  inputs = ["The immutable admission result and an integer fee from 0 through 1000 cents."]
  outputs = ["The exact integer total or invalid_amount without a total."]
  owns_state = [state.total_result]
  dependencies = [module.admission]
  invariants = ["Admission results remain unchanged.", "The stated integer range requires no rounding and fits the selected arithmetic representation."]
  failure_behavior = "Propagate invalid_amount without calculating a total. Never replace invalid input with zero. An out-of-contract fee must be rejected rather than used to claim the total predicate."
  acceptance_checks = [fr.calculate_total.criterion.total, scenario.total_boundaries]
}

flow "receipt_request" {
  owner = module.total
  input = "An untrusted amount and an in-range integer fee."
  output = "An exact total or invalid_amount."
  steps = [module.admission, module.total]
}

scenario "amount_boundaries" {
  kind = "operational"
  verifies = [fr.admit_amount.criterion.admission]
  description = "Proposed checks for the admission leaf. These are instructions, not recorded observations."
  steps = ["Admit 0 and 100000 integer cents.", "Reject -1, 100001, 1.5, a string, and null without publishing an amount.", "For each remaining integer in the permitted range, require admission."]
}

scenario "total_boundaries" {
  kind = "operational"
  verifies = [fr.calculate_total.criterion.total]
  description = "Proposed checks for the total leaf. No completed trial is claimed."
  steps = ["For each admitted amount and in-range integer fee, require their exact integer sum.", "Include 0 plus 0 and 100000 plus 1000.", "Pass an invalid admission result and require invalid_amount with no total.", "Require the admission result to remain unchanged after calculation."]
}

scenario "change_addition_policy" {
  kind = "change"
  verifies = [fr.admit_amount.criterion.admission, fr.calculate_total.criterion.total]
  vary = [dp.exact_addition]
  expect_affected = [fr.calculate_total]
  expect_unaffected = [fr.admit_amount]
  description = "Proposed change check under the fixed immutable admission contract. No change or isolation result has been measured."
  steps = ["Compare two addition-policy variants while holding the admission policy, input range, and immutable admission results fixed.", "Require identical admission decisions and unchanged published admission results.", "Re-evaluate the exact-total predicate for each addition variant rather than assuming the changed policy still satisfies it."]
}

scenario "propagate_invalid_admission" {
  kind = "failure"
  verifies = [fr.admit_amount.criterion.admission, fr.calculate_total.criterion.total]
  description = "Proposed failure check for a rejected amount. No execution result is supplied."
  steps = ["Reject -1, 1.5, and a string as invalid_amount with no admitted amount.", "Pass each immutable invalid_amount result to the total module with an in-range fee.", "Require invalid_amount with no total; do not fabricate a zero amount, a zero total, or a fee-only total.", "Require the original admission result to remain unchanged."]
}

decision "receipt_mechanism" {
  addresses = [fr.admit_amount, fr.calculate_total]
  workload = workload.receipt_inputs
  horizon = "One request under fixed input and result contracts."
  candidate "integer_cents" {
    dps = [dp.amount_admission, dp.exact_addition]
    version = "receipt-proposal-1"
    information {
      basis = "unknown"
    }
  }
  selected = "integer_cents"
  provisional = true
  why = "Select explicit admission followed by exact integer addition provisionally. No reviewed support or joint empirical success evidence is supplied."
}
`

// alternativeCoreBlocks adds a second candidate with its own proposed DP and module.
// That candidate is not selected, so its DP and module are inactive in the default snapshot.
const alternativeCoreBlocks = `
dp "alt_admission" {
  controls = [fr.admit_amount]
  responsibility = "Admit amounts under a rule that is not selected for this change."
  decision = "proposed"
  holds_if = [assumption.fixed_contracts]
}

module "alternative_admission" {
  realizes = [dp.alt_admission]
  inputs = ["One untrusted amount value."]
  outputs = ["An alternative admission result."]
  dependencies = []
  invariants = ["No invalid value becomes an admitted amount."]
  failure_behavior = "Reject invalid amounts without publishing an amount."
  acceptance_checks = [fr.admit_amount.criterion.admission]
}
`

// receiptManifestSource declares the three registry extensions.
const receiptManifestSource = `project "receipt" {
  language = "requirements_hcl/core/1"
  core_files = ["design.ad.hcl"]
  attachment_files = ["bindings.ad.ext.hcl", "behavior.ad.ext.hcl", "views.ad.ext.hcl"]
}

extension "terraform_binding" {
  role = "binding"
  contract = "requirements_hcl/terraform_binding/1"
}

extension "event_model_link" {
  role = "behavior_link"
  contract = "requirements_hcl/event_model_link/1"
}

extension "html_view" {
  role = "view"
  contract = "requirements_hcl/html_view/1"
}
`

const bindingSource = `attachment "admission_configuration" {
  owner = extension.terraform_binding
  subject = pv.admission_policy
  data = {
    address = "terraform_data.admission_policy"
    observation_source = "receipt_configuration"
  }
}
`

const behaviorSource = `attachment "total_behavior" {
  owner = extension.event_model_link
  subject = module.total
  data = {
    model = "receipt_behavior"
    revision = "receipt-behavior-proposal-1"
    target_kind = "slice"
    target_label = "calculate_total"
  }
}
`

const viewSource = `attachment "receipt_layout" {
  owner = extension.html_view
  subject = design.receipt
  data = {
    layout = "requirements_first"
    show_source_locations = true
  }
}
`

// behaviorPairSource has one link on an inactive module and one on an active module.
const behaviorPairSource = `attachment "alternative_behavior" {
  owner = extension.event_model_link
  subject = module.alternative_admission
  data = {
    model = "receipt_behavior"
    revision = "receipt-behavior-proposal-1"
    target_kind = "slice"
    target_label = "admit_amount"
  }
}

attachment "total_behavior" {
  owner = extension.event_model_link
  subject = module.total
  data = {
    model = "receipt_behavior"
    revision = "receipt-behavior-proposal-1"
    target_kind = "slice"
    target_label = "calculate_total"
  }
}
`

// source is one authored file: its path and its text.
type source struct {
	path string
	text string
}

// ownersFixture is one decoded and selected project, ready for Assess.
type ownersFixture struct {
	snapshot    *selection.Snapshot
	manifest    model.Manifest
	attachments model.Attachments
}

// loadOwnersFixture decodes the core, manifest, and attachments into an Assess input.
// Setup stages must produce no findings.
func loadOwnersFixture(t *testing.T, request selection.Request, core, manifest string, attachments ...source) ownersFixture {
	t.Helper()
	decodedCore, findings := schema.DecodeCore([]syntax.Unit{parseOwnersSource(t, source{coreFile, core})})
	requireNoSetupFindings(t, "decode core", findings)
	graph, findings := schema.Resolve(decodedCore)
	requireNoSetupFindings(t, "resolve core", findings)
	snapshot, findings := selection.Select(graph, request)
	requireNoSetupFindings(t, "select", findings)
	decodedManifest, findings := schema.DecodeManifest(parseOwnersSource(t, source{manifestFile, manifest}))
	requireNoSetupFindings(t, "decode manifest", findings)
	units := make([]syntax.Unit, 0, len(attachments))
	for _, att := range attachments {
		units = append(units, parseOwnersSource(t, att))
	}
	decodedAttachments, findings := schema.DecodeAttachments(units)
	requireNoSetupFindings(t, "decode attachments", findings)
	return ownersFixture{snapshot: snapshot, manifest: decodedManifest, attachments: decodedAttachments}
}

func parseOwnersSource(t *testing.T, src source) syntax.Unit {
	t.Helper()
	unit := syntax.Parse(model.Source{Path: src.path, Bytes: []byte(src.text)})
	requireNoSetupFindings(t, "parse "+src.path, unit.Findings)
	return unit
}

func requireNoSetupFindings(t *testing.T, stage string, findings []result.Finding) {
	t.Helper()
	if len(findings) != 0 {
		t.Fatalf("%s: %d findings, want 0; first code %q", stage, len(findings), string(findings[0].Code))
	}
}

func runOwners(f ownersFixture) []OwnerResult {
	return Assess(f.snapshot, f.manifest, f.attachments)
}

func ownerResultFor(t *testing.T, results []OwnerResult, extension model.ID) OwnerResult {
	t.Helper()
	for _, r := range results {
		if r.Extension == extension {
			return r
		}
	}
	t.Fatalf("no owner result for %s", extension)
	return OwnerResult{}
}

func attachmentResultFor(testOwner OwnerResult, label string) (AttachmentResult, bool) {
	for _, r := range testOwner.Results {
		if r.Attachment == label {
			return r, true
		}
	}
	return AttachmentResult{}, false
}

func findingWithCode(findings []result.Finding, code string) (result.Finding, bool) {
	for _, f := range findings {
		if string(f.Code) == code {
			return f, true
		}
	}
	return result.Finding{}, false
}

// attachmentFile renders one attachment block whose data lines are already indented.
func attachmentFile(label, testOwner, subject, data string) string {
	return fmt.Sprintf("attachment %q {\n  owner = %s\n  subject = %s\n  data = {\n%s  }\n}\n", label, testOwner, subject, data)
}

// coreWithPVAddress sets the address of the admission policy PV in the starter core.
func coreWithPVAddress(address string) string {
	return strings.Replace(receiptCoreSource,
		"  kind = \"configuration\"\n",
		"  kind = \"configuration\"\n  address = \""+address+"\"\n", 1)
}

// coreWithAlternative adds an unselected candidate and its DP and module to the starter core.
func coreWithAlternative() string {
	core := strings.Replace(receiptCoreSource,
		"  selected = \"integer_cents\"\n",
		"  candidate \"alternative\" {\n    dps = [dp.alt_admission]\n  }\n  selected = \"integer_cents\"\n", 1)
	return core + alternativeCoreBlocks
}

func TestAssessBlocksOwnerWithUnknownLabelRoleOrContract(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		attached string
		owner    model.ID
	}{
		{
			name:     "unknown label",
			manifest: strings.Replace(receiptManifestSource, `extension "terraform_binding"`, `extension "mystery"`, 1),
			attached: strings.Replace(bindingSource, "extension.terraform_binding", "extension.mystery", 1),
			owner:    model.ID("extension.mystery"),
		},
		{
			name:     "wrong role",
			manifest: strings.Replace(receiptManifestSource, `role = "binding"`, `role = "view"`, 1),
			attached: bindingSource,
			owner:    bindingOwnerID,
		},
		{
			name:     "wrong contract",
			manifest: strings.Replace(receiptManifestSource, "requirements_hcl/terraform_binding/1", "requirements_hcl/terraform_binding/2", 1),
			attached: bindingSource,
			owner:    bindingOwnerID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a declared binding extension that is unknown, has the wrong role, or has the wrong contract, and one binding attachment for it.
			f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, tc.manifest, source{bindingFile, tc.attached})

			// When Assess runs.
			got := ownerResultFor(t, runOwners(f), tc.owner)

			// Then the owner is blocked, reports unavailable_owner, and processes no payloads.
			if got.Status != Status("blocked") {
				t.Fatalf("status = %q, want blocked", got.Status)
			}
			if _, ok := findingWithCode(got.Findings, "unavailable_owner"); !ok {
				t.Fatalf("findings lack unavailable_owner")
			}
			if len(got.Results) != 0 {
				t.Fatalf("results = %d, want 0", len(got.Results))
			}
		})
	}
}

func TestAssessKeepsUnusedValidOwnersComplete(t *testing.T) {
	// Given the three valid registry extensions declared and no attachments.
	f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource)

	// When Assess runs.
	results := runOwners(f)

	// Then every declared owner is present, complete, and has no results.
	for _, id := range []model.ID{bindingOwnerID, behaviorOwnerID, viewOwnerID} {
		got := ownerResultFor(t, results, id)
		if got.Status != Status("complete") {
			t.Fatalf("%s status = %q, want complete", id, got.Status)
		}
		if len(got.Results) != 0 {
			t.Fatalf("%s results = %d, want 0", id, len(got.Results))
		}
	}
}

func TestAssessBlocksUnusedUnknownOwner(t *testing.T) {
	// Given a declared extension with an unknown label and no attachments.
	manifest := strings.Replace(receiptManifestSource, `extension "terraform_binding"`, `extension "mystery"`, 1)
	f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, manifest)

	// When Assess runs.
	got := ownerResultFor(t, runOwners(f), model.ID("extension.mystery"))

	// Then the unused unknown owner is still reported as blocked with unavailable_owner.
	if got.Status != Status("blocked") {
		t.Fatalf("status = %q, want blocked", got.Status)
	}
	if _, ok := findingWithCode(got.Findings, "unavailable_owner"); !ok {
		t.Fatalf("findings lack unavailable_owner")
	}
}

func TestAssessSkipsAttachmentWithWrongSubjectKind(t *testing.T) {
	// Given a binding attachment on module.total and a valid binding attachment on pv.admission_policy.
	wrongKind := attachmentFile("wrong_subject", "extension.terraform_binding", "module.total",
		"    address = \"terraform_data.total\"\n    observation_source = \"receipt_configuration\"\n")
	f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource,
		source{bindingFile, wrongKind + "\n" + bindingSource})

	// When Assess runs.
	binding := ownerResultFor(t, runOwners(f), bindingOwnerID)

	// Then the wrong-kind attachment has no result, and the valid sibling is processed.
	if _, ok := attachmentResultFor(binding, "wrong_subject"); ok {
		t.Fatalf("wrong-kind attachment produced a result")
	}
	sibling, ok := attachmentResultFor(binding, "admission_configuration")
	if !ok {
		t.Fatalf("valid sibling attachment has no result")
	}
	if sibling.Subject != model.ID("pv.admission_policy") {
		t.Fatalf("sibling subject = %q, want pv.admission_policy", sibling.Subject)
	}
}

func TestAssessRejectsInvalidPayloadWithInvalidAttachment(t *testing.T) {
	cases := []struct {
		name  string
		owner model.ID
		path  string
		text  string
	}{
		{
			name:  "binding missing observation_source",
			owner: bindingOwnerID,
			path:  bindingFile,
			text: attachmentFile(invalidLabel, "extension.terraform_binding", "pv.admission_policy",
				"    address = \"terraform_data.admission_policy\"\n"),
		},
		{
			name:  "binding extra key",
			owner: bindingOwnerID,
			path:  bindingFile,
			text: attachmentFile(invalidLabel, "extension.terraform_binding", "pv.admission_policy",
				"    address = \"terraform_data.admission_policy\"\n    observation_source = \"receipt_configuration\"\n    region = \"eu\"\n"),
		},
		{
			name:  "binding blank address",
			owner: bindingOwnerID,
			path:  bindingFile,
			text: attachmentFile(invalidLabel, "extension.terraform_binding", "pv.admission_policy",
				"    address = \"\"\n    observation_source = \"receipt_configuration\"\n"),
		},
		{
			name:  "behavior missing revision",
			owner: behaviorOwnerID,
			path:  behaviorFile,
			text: attachmentFile(invalidLabel, "extension.event_model_link", "module.total",
				"    model = \"receipt_behavior\"\n    target_kind = \"slice\"\n    target_label = \"calculate_total\"\n"),
		},
		{
			name:  "behavior unknown target kind",
			owner: behaviorOwnerID,
			path:  behaviorFile,
			text: attachmentFile(invalidLabel, "extension.event_model_link", "module.total",
				"    model = \"receipt_behavior\"\n    revision = \"receipt-behavior-proposal-1\"\n    target_kind = \"method\"\n    target_label = \"calculate_total\"\n"),
		},
		{
			name:  "behavior extra key",
			owner: behaviorOwnerID,
			path:  behaviorFile,
			text: attachmentFile(invalidLabel, "extension.event_model_link", "module.total",
				"    model = \"receipt_behavior\"\n    revision = \"receipt-behavior-proposal-1\"\n    target_kind = \"slice\"\n    target_label = \"calculate_total\"\n    owner_hint = \"total\"\n"),
		},
		{
			name:  "view unknown layout",
			owner: viewOwnerID,
			path:  viewFile,
			text: attachmentFile(invalidLabel, "extension.html_view", "design.receipt",
				"    layout = \"other\"\n    show_source_locations = true\n"),
		},
		{
			name:  "view missing show_source_locations",
			owner: viewOwnerID,
			path:  viewFile,
			text: attachmentFile(invalidLabel, "extension.html_view", "design.receipt",
				"    layout = \"requirements_first\"\n"),
		},
		{
			name:  "view extra key",
			owner: viewOwnerID,
			path:  viewFile,
			text: attachmentFile(invalidLabel, "extension.html_view", "design.receipt",
				"    layout = \"requirements_first\"\n    show_source_locations = true\n    theme = \"dark\"\n"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a valid registry owner and an attachment whose payload is invalid.
			f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource, source{tc.path, tc.text})

			// When Assess runs.
			got := ownerResultFor(t, runOwners(f), tc.owner)

			// Then the owner completes, reports invalid_attachment, and has no result for the invalid attachment.
			if got.Status != Status("complete") {
				t.Fatalf("status = %q, want complete", got.Status)
			}
			if _, ok := findingWithCode(got.Findings, "invalid_attachment"); !ok {
				t.Fatalf("findings lack invalid_attachment")
			}
			if _, ok := attachmentResultFor(got, invalidLabel); ok {
				t.Fatalf("invalid attachment produced a result")
			}
		})
	}
}

func TestAssessRejectsNullPayloadValue(t *testing.T) {
	// Given a valid binding attachment whose address value is replaced with null after decoding.
	f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource, source{bindingFile, bindingSource})
	if len(f.attachments) == 0 {
		t.Fatalf("fixture has no attachments")
	}
	replaced := false
	for i := range f.attachments[0].Data {
		if f.attachments[0].Data[i].Key == "address" {
			f.attachments[0].Data[i].Value = cty.NullVal(cty.String)
			replaced = true
		}
	}
	if !replaced {
		t.Fatalf("fixture has no address field")
	}

	// When Assess runs.
	got := ownerResultFor(t, runOwners(f), bindingOwnerID)

	// Then the owner reports invalid_attachment and produces no result for the attachment.
	if _, ok := findingWithCode(got.Findings, "invalid_attachment"); !ok {
		t.Fatalf("findings lack invalid_attachment")
	}
	if _, ok := attachmentResultFor(got, "admission_configuration"); ok {
		t.Fatalf("null-valued attachment produced a result")
	}
}

func TestAssessRejectsFalseShowSourceLocations(t *testing.T) {
	// Given a view attachment on design.receipt with show_source_locations set to false.
	view := attachmentFile("invalid_view", "extension.html_view", "design.receipt",
		"    layout = \"requirements_first\"\n    show_source_locations = false\n")
	f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource, source{viewFile, view})

	// When Assess runs.
	got := ownerResultFor(t, runOwners(f), viewOwnerID)

	// Then the owner completes, reports invalid_attachment, and produces no view result.
	if got.Status != Status("complete") {
		t.Fatalf("status = %q, want complete", got.Status)
	}
	if _, ok := findingWithCode(got.Findings, "invalid_attachment"); !ok {
		t.Fatalf("findings lack invalid_attachment")
	}
	if _, ok := attachmentResultFor(got, "invalid_view"); ok {
		t.Fatalf("invalid view produced a result")
	}
}

func TestAssessReportsSecondViewAsDuplicateIdentity(t *testing.T) {
	// Given two valid view attachments on the same design.
	second := attachmentFile("second_layout", "extension.html_view", "design.receipt",
		"    layout = \"matrices_first\"\n    show_source_locations = true\n")
	f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource,
		source{viewFile, viewSource + "\n" + second})

	// When Assess runs.
	got := ownerResultFor(t, runOwners(f), viewOwnerID)

	// Then the owner completes and reports duplicate_identity.
	if got.Status != Status("complete") {
		t.Fatalf("status = %q, want complete", got.Status)
	}
	if _, ok := findingWithCode(got.Findings, "duplicate_identity"); !ok {
		t.Fatalf("findings lack duplicate_identity")
	}
}

func TestAssessReportsDuplicateBindingAddress(t *testing.T) {
	// Given a PV whose core address equals the binding attachment address.
	f := loadOwnersFixture(t, selection.Request{}, coreWithPVAddress("terraform_data.admission_policy"),
		receiptManifestSource, source{bindingFile, bindingSource})

	// When Assess runs.
	got := ownerResultFor(t, runOwners(f), bindingOwnerID)

	// Then the binding owner reports duplicate_binding at both the core and attachment declarations.
	finding, ok := findingWithCode(got.Findings, "duplicate_binding")
	if !ok {
		t.Fatalf("findings lack duplicate_binding")
	}
	if finding.Range == nil || len(finding.Related) != 1 {
		t.Fatalf("duplicate_binding range = %v with %d related, want one primary and one related range", finding.Range, len(finding.Related))
	}
	files := map[string]bool{finding.Range.Filename: true, finding.Related[0].Filename: true}
	if !files[coreFile] || !files[bindingFile] {
		t.Fatalf("duplicate_binding files = %v, want both %s and %s", files, coreFile, bindingFile)
	}
}

func TestAssessGivesValidExternalPayloadsUnknownOutcome(t *testing.T) {
	cases := []struct {
		name    string
		owner   model.ID
		path    string
		text    string
		label   string
		subject model.ID
		cause   string
	}{
		{
			name:    "binding",
			owner:   bindingOwnerID,
			path:    bindingFile,
			text:    bindingSource,
			label:   "admission_configuration",
			subject: model.ID("pv.admission_policy"),
			cause:   expectedBindingCause,
		},
		{
			name:    "behavior link",
			owner:   behaviorOwnerID,
			path:    behaviorFile,
			text:    behaviorSource,
			label:   "total_behavior",
			subject: model.ID("module.total"),
			cause:   expectedBehaviorCause,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Given a valid payload for the owner under test.
			f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource, source{tc.path, tc.text})

			// When Assess runs.
			got := ownerResultFor(t, runOwners(f), tc.owner)

			// Then the owner completes with one unknown result carrying the pinned cause and an unavailable_external_input finding whose subject is attachment.<label>.
			if got.Status != Status("complete") {
				t.Fatalf("status = %q, want complete", got.Status)
			}
			r, ok := attachmentResultFor(got, tc.label)
			if !ok {
				t.Fatalf("no result for %s", tc.label)
			}
			if r.Subject != tc.subject {
				t.Fatalf("subject = %q, want %q", r.Subject, tc.subject)
			}
			if r.Outcome != Outcome("unknown") {
				t.Fatalf("outcome = %q, want unknown", r.Outcome)
			}
			if r.Cause != tc.cause {
				t.Fatalf("cause = %q, want %q", r.Cause, tc.cause)
			}
			finding, ok := findingWithCode(got.Findings, "unavailable_external_input")
			if !ok {
				t.Fatalf("findings lack unavailable_external_input")
			}
			if finding.Subject != "attachment."+tc.label {
				t.Fatalf("finding subject = %q, want %q", finding.Subject, "attachment."+tc.label)
			}
		})
	}
}

func TestAssessGivesValidViewOutcomeValid(t *testing.T) {
	// Given a valid view attachment on design.receipt.
	f := loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource, source{viewFile, viewSource})

	// When Assess runs.
	got := ownerResultFor(t, runOwners(f), viewOwnerID)

	// Then the owner completes and the view result is valid.
	if got.Status != Status("complete") {
		t.Fatalf("status = %q, want complete", got.Status)
	}
	r, ok := attachmentResultFor(got, "receipt_layout")
	if !ok {
		t.Fatalf("no result for receipt_layout")
	}
	if r.Outcome != Outcome("valid") {
		t.Fatalf("outcome = %q, want valid", r.Outcome)
	}
}

func TestAssessSkipsAttachmentForInactiveSubjectInDefaultSnapshot(t *testing.T) {
	// Given a link on module.alternative_admission, which is inactive in the default snapshot, and a link on module.total, which is active.
	f := loadOwnersFixture(t, selection.Request{}, coreWithAlternative(), receiptManifestSource, source{behaviorFile, behaviorPairSource})

	// When Assess runs.
	got := ownerResultFor(t, runOwners(f), behaviorOwnerID)

	// Then the inactive link has no result and the active link is processed.
	if _, ok := attachmentResultFor(got, "alternative_behavior"); ok {
		t.Fatalf("inactive-subject attachment produced a result")
	}
	if _, ok := attachmentResultFor(got, "total_behavior"); !ok {
		t.Fatalf("active-subject attachment has no result")
	}
}

func TestAssessSkipsAttachmentForInactiveSubjectUnderAlternativeCandidate(t *testing.T) {
	// Given the alternative candidate selected, so module.total is inactive and module.alternative_admission is active.
	request := selection.Request{Candidate: model.ID("decision.receipt_mechanism.candidate.alternative")}
	f := loadOwnersFixture(t, request, coreWithAlternative(), receiptManifestSource, source{behaviorFile, behaviorPairSource})

	// When Assess runs.
	got := ownerResultFor(t, runOwners(f), behaviorOwnerID)

	// Then the link on the now-inactive module.total has no result and the link on the active module is processed.
	if _, ok := attachmentResultFor(got, "total_behavior"); ok {
		t.Fatalf("inactive-subject attachment produced a result")
	}
	if _, ok := attachmentResultFor(got, "alternative_behavior"); !ok {
		t.Fatalf("active-subject attachment has no result")
	}
}

func TestAssessChangingBindingPayloadChangesOnlyBindingOwner(t *testing.T) {
	// Given a baseline whose binding address duplicates the PV address, and a changed binding address that does not.
	core := coreWithPVAddress("terraform_data.admission_policy")
	changedBinding := strings.Replace(bindingSource, `address = "terraform_data.admission_policy"`, `address = "terraform_data.other_policy"`, 1)
	baseline := runOwners(loadOwnersFixture(t, selection.Request{}, core, receiptManifestSource,
		source{bindingFile, bindingSource}, source{behaviorFile, behaviorSource}, source{viewFile, viewSource}))
	changed := runOwners(loadOwnersFixture(t, selection.Request{}, core, receiptManifestSource,
		source{bindingFile, changedBinding}, source{behaviorFile, behaviorSource}, source{viewFile, viewSource}))

	// When the two runs are compared owner by owner.
	baseBinding := ownerResultFor(t, baseline, bindingOwnerID)
	changedBindingResult := ownerResultFor(t, changed, bindingOwnerID)

	// Then the binding owner result changes, and the behavior and view owner results are unchanged.
	if reflect.DeepEqual(baseBinding, changedBindingResult) {
		t.Fatalf("binding owner result did not change")
	}
	if _, ok := findingWithCode(baseBinding.Findings, "duplicate_binding"); !ok {
		t.Fatalf("baseline binding owner lacks duplicate_binding")
	}
	if _, ok := findingWithCode(changedBindingResult.Findings, "duplicate_binding"); ok {
		t.Fatalf("changed binding owner still reports duplicate_binding")
	}
	for _, id := range []model.ID{behaviorOwnerID, viewOwnerID} {
		if !reflect.DeepEqual(ownerResultFor(t, baseline, id), ownerResultFor(t, changed, id)) {
			t.Fatalf("%s owner result changed when only the binding payload changed", id)
		}
	}
}

func TestAssessChangingViewLayoutLeavesOtherOwnersUnchanged(t *testing.T) {
	// Given a baseline view with requirements_first layout and a changed view with matrices_first layout.
	changedView := strings.Replace(viewSource, `layout = "requirements_first"`, `layout = "matrices_first"`, 1)
	baseline := runOwners(loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource,
		source{bindingFile, bindingSource}, source{behaviorFile, behaviorSource}, source{viewFile, viewSource}))
	changed := runOwners(loadOwnersFixture(t, selection.Request{}, receiptCoreSource, receiptManifestSource,
		source{bindingFile, bindingSource}, source{behaviorFile, behaviorSource}, source{viewFile, changedView}))

	// When the two runs are compared owner by owner.
	changedViewOwner := ownerResultFor(t, changed, viewOwnerID)

	// Then the view stays valid, and the binding and behavior owner results are unchanged.
	r, ok := attachmentResultFor(changedViewOwner, "receipt_layout")
	if !ok {
		t.Fatalf("changed view has no result")
	}
	if r.Outcome != Outcome("valid") {
		t.Fatalf("changed view outcome = %q, want valid", r.Outcome)
	}
	for _, id := range []model.ID{bindingOwnerID, behaviorOwnerID} {
		if !reflect.DeepEqual(ownerResultFor(t, baseline, id), ownerResultFor(t, changed, id)) {
			t.Fatalf("%s owner result changed when only the view layout changed", id)
		}
	}
}
