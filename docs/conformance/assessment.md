# Core version 1 assessment and conformance expectations

This document records how the pinned contracts assess decoded facts and what the current conformance fixtures expect. The normative authorities are the assessment contract for assessment rules, schema for fields, and diagnostic contract for codes, severity, and operation status. The `expected.json` fixtures record acceptance expectations, not measured validator success. No validator result is claimed here.

Use the vendored, pinned copies at `../../testdata/spec/docs/schema.md`, `../../testdata/spec/docs/assessment-contract.md`, `../../testdata/spec/docs/diagnostics.md`, and `../../testdata/spec/conformance/core-1/expected.json`. The inspected source example is `../../testdata/spec/examples/requirements_hcl.ad.hcl`. [Pin provenance](pin.md) records the base revision and states that the closure is pinned by the copied working-tree bytes, not by the base commit alone.

## Assessment rules used by the decoder

The assessment contract is authoritative for snapshot selection, applicability, cell resolution, pairing, hierarchy, support, expectation evaluation, profile completeness, feasibility, and evidence. This summary names the high-impact rules; implementations must use the complete normative contract.

- Default snapshots activate selected DPs. Explicit candidates activate exactly their members plus selected common DPs. Active child FRs require active parents; active DPs require all controlled FRs. Never insert missing ancestors or delete inactive history. Version 1 accepts at most one candidate and context per assessment.
- Resolve conditions before effects. Context-specific facts do not transfer to another context. Missing context leaves conditional facts unresolved. Invalidated premises disable the claim; assumed or unevaluated premises make it conditional.
- Merge compatible exact facts without losing provenance. Contradictory exact effects and conflicting defaults are conflicts. An exact unknown remains visible and prevents a default from replacing it. A missing exact fact and default resolves unknown, not zero.
- Pair only when there is exactly one complete distinct control assignment. More than one is ambiguous; no complete assignment reports uncontrolled or redundant controls as applicable. A square classification requires every material diagonal. Preserve rectangular axes. Known cycles remain reportable while any unknown still prevents complete classification.
- Do not expand parent effects into leaf facts. An ancestor disagreement remains unresolved refinement until an applicable outcome argument proves contradiction. Core version 1 has no trusted review import, so qualitative supported independence, allocation, and documentary feasibility cannot be asserted by citation, authored prose, or premise status.
- Expectations are local to their named sibling scope. `basis` defaults to `source`; conditional basis retains named premises; supported basis requires accepted support. All bases retain unknowns and conflicts. Conflict cannot pass classification. Unknown can pass only explicit `unresolved` without conflict. A requested order must include each active DP exactly once and honor every known precedence edge. An unknown cell prevents a complete design sequence.
- The architecture profile requires operational criteria for active FRs, complete realization contracts for active leaf DPs, applicable scenarios for active leaves, and a safety contract for residual material feedback. These declarations establish completeness, not success or support.
- Feasibility is `infeasible` only with applicable evidence of failure; otherwise missing required success evidence leaves it unresolved. Evidence uses exact candidate version and exposure identity, positive joint trial counts, failures within range, confidence strictly between zero and one, and the two-sided Wilson interval without continuity correction. Do not pool blocks automatically or claim certainty from zero failures.

## Profile severity and stable diagnostic catalog

Severity changes with profile only where shown. It never changes claim truth. `error` fails the operation; `warning` retains a gap without failing it. Both profiles use the same rule codes.

| Code | Condition | Source | Architecture |
|---|---|---:|---:|
| `input_failure` | Input read, selection, or containment failure | error | error |
| `invalid_syntax` | Native parse or expression-profile failure | error | error |
| `invalid_structure` | Schema defect, unknown field, null, enum, or cardinality | error | error |
| `duplicate_identity` | Label, attribute, object key, physical input, or result identity collision | error | error |
| `invalid_reference` | Missing or wrong-kind typed target | error | error |
| `invalid_hierarchy` | Cycle, unreachable entity, or invalid parent | error | error |
| `invalid_selection` | Candidate lifecycle, membership, ancestry, or condition conflict | error | error |
| `conflicting_influence` | Applicable exact effects or defaults conflict | error | error |
| `ambiguous_pairing` | Multiple complete control pairings | error | error |
| `uncontrolled_requirement` | No distinct control can be assigned to each sibling FR | error | error |
| `redundant_control` | A control remains outside a complete one-to-one pairing | error | error |
| `ineffective_control` | Unique paired diagonal is known zero | error | error |
| `conflicting_ownership` | Duplicate logical state owner or realization claim | error | error |
| `incomplete_criterion` | Active FR lacks criteria or operational predicate | warning | error |
| `incomplete_module_contract` | Active leaf lacks realization module or required contract | warning | error |
| `missing_leaf_scenario` | Active leaf lacks applicable criterion-verifying scenario | warning | error |
| `incomplete_feedback_contract` | Residual iteration lacks bound, convergence, stopping, or failure contract | warning | error |
| `unresolved_refinement` | Parent effect/allocation lacks outcome argument | warning | error |
| `unresolved_applicability` | Conditions or invalidated premise leave required relation unresolved | warning | error |
| `unresolved_influence` | Unknown cells prevent complete active sibling/master classification | warning | error |
| `shared_resource_review_gap` | Shared state/resource/PV lacks applicable influence assessment | warning | error |
| `support_gap` | Claim lacks applicable reviewed support, including absent review import | warning | warning |
| `unresolved_feasibility` | Candidate lacks applicable success evidence | warning | warning |
| `infeasible_candidate` | Evidence establishes failure of required criterion/constraint | error | error |
| `invalid_evidence` | Trial or evidence-specific metadata violates contract | error | error |
| `incomparable_evidence` | Comparison lacks common outcomes or compatible exposure | warning | warning |
| `failed_expectation` | Expectation is false or cannot establish its requested basis | error | error |
| `invalid_attachment` | Envelope or owner payload violates contract | error | error |
| `unavailable_owner` | Owner absent, wrong role, or wrong exact contract | error | error |
| `owner_failure` | Required owner cannot complete | error | error |
| `duplicate_binding` | PV core address exactly equals attachment address under binding rule | error | error |
| `binding_mismatch` | Available observation contradicts required binding | error | error |
| `behavior_target_missing` | Exact available behavior revision lacks target | error | error |
| `unavailable_external_input` | Observation data or exact behavior revision unavailable | warning | warning |
| `render_failure` | Requested report cannot render or deliver | error | error |

The complete diagnostic contract also governs location retention, specific-code precedence, blocked work, partial results, stage statuses, and the distinction between processing completion and conformance. Do not emit downstream findings from a guessed graph. Do retain independent findings after another prerequisite fails. Binding duplication compares admitted strings exactly and reports both declarations; it is not inferred from similar prose.

Owner outcomes are separate from validator processing: binding match and behavior resolved can complete without error, while mismatch/missing target fail. Missing external inputs yield completed unknown with warning, not fabricated success. Invalid required attachment, unavailable required owner, owner failure, or requested render failure fails the project operation. Validation does not request HTML rendering.

## Expected conformance cases

The fixture set identifies operation `validate`, profiles `source` and `architecture`, fixed registry labels `terraform_binding`, `event_model_link`, and `html_view`, unavailable observations/revisions/review import, and available text delivery. Its case table follows [`expected.json`](../../testdata/spec/conformance/core-1/expected.json); these are expected outcomes, not observed runs.

| Case / input | Expected core and operation behavior |
|---|---|
| `baseline` / `base.ad.hcl` | Core and composition complete; owners not requested. Both profiles succeed. `design.two_outputs` is conditionally uncoupled with the stated two-control diagonal and off-diagonal none. Supported independence remains unresolved. `support_gap` warning is required; `unresolved_feasibility` is permitted. |
| `invalid_schema` / `invalid-schema.ad.project.hcl` | Core blocked; both profiles fail. Required `invalid_structure` errors cover null `source.title`, unknown `need.ignored_typo`, empty required `dp.controls`, and string instead of traversal in `scenario.verifies`. Classification is not assessed. |
| `wrong_reference_kind` / `wrong-kind.ad.project.hcl` | Core blocked; both profiles fail with `invalid_reference` for `dp.wrong_kind.controls` targeting a need where FR is required. No classification is assessed. |
| `cross_file_duplicate` / `duplicate.ad.project.hcl` | Core blocked; both profiles fail with `duplicate_identity` for `need.separate_outputs`, retaining both source files. |
| `exact_unknown_over_default` / `exact-unknown.ad.project.hcl` | Core complete. Source succeeds and architecture fails. Classification is unresolved; exact unknown for `dp.right_output` → `fr.left` prevents the default from applying without creating a conflict. `unresolved_influence` is warning/error by profile; `support_gap` and possible unresolved feasibility remain warnings. |
| `ambiguous_pairing` / `ambiguous.ad.hcl` | Core completes but both operations fail. `ambiguous_pairing` error; no selected pairing despite two complete pairings. Classification unresolved; support gap remains warning. |
| `invalid_required_attachment` / `invalid-attachment.ad.project.hcl` | Core remains complete and uncoupled; HTML owner stage completes with invalid payload. Both operations fail due to `invalid_attachment` for `data.show_source_locations`. Core matrix remains unchanged; support gap remains warning. |
| `starter_core` / `examples/starter/design.ad.hcl` | Core and both operations complete. `design.receipt` is conditionally decoupled in amount-admission then exact-addition order. Feasibility, information, implementation conformance, and supported independence remain unresolved. `support_gap` and `unresolved_feasibility` are warnings. |
| `starter_project_external_unknown` / `examples/starter/model.ad.project.hcl` | Core and each of three owners complete; both operations succeed. Conditional classification remains decoupled in the same order. Binding and behavior results are unknown because external inputs are unavailable; each has `unavailable_external_input` warning. Core matrix remains unchanged. Feasibility, information, implementation conformance, and supported independence remain unresolved. |

Expected findings may include only the case's required and explicitly permitted additional codes. The fixture's `other_codes: forbidden` constraints are acceptance assertions, not proof that an implementation has been run.

## Self-design field coverage and expected gaps

The input `../../testdata/spec/examples/requirements_hcl.ad.hcl` is an integration example.
It is not schema authority and does not prove that its assertions pass.
The schema tables cover every field used by the example.
The following inventory names those fields:

- Common metadata and provenance: `title`, `url`, `cites`, `origin`, `alias`.
- Design and operating premises: `boundary`, `level`, `workload`, `description`, `excludes`, `statement`, `why`, `kind`, `status`.
- FR criteria and hierarchy: `need`, `parent`, `metric`, `op`, `value`, `under`, `allocated_from`.
- DP controls and state: `controls`, `responsibility`, `decision`, `bounded_by`, `owns_state`.
- Influence claims: `from`, `to`, `effect`, `mechanism`, `mechanism_source`, `because`, `holds_if`.
- State and module contracts: `realizes`, `inputs`, `outputs`, `dependencies`, `invariants`, `failure_behavior`, `acceptance_checks`.
- Flow: `owner`, `input`, `output`, `steps`.
- Decision/candidate/information: `addresses`, `context`, `horizon`, `dps`, `version`, `basis`, `selected`, `provisional`.
- Scenarios: `kind`, `verifies`, `vary`, `expect_affected`, `expect_unaffected`, `description`, `steps`.

Each identifier appears in a normative inventory row.
The [`schema.md` block reference](schema.md#field-inventory-and-precise-authority-pointers) identifies its owning block.
The [vendored schema section](../../testdata/spec/docs/schema.md) gives its exact type, cardinality, required and default rules, null behavior, and enum or target contract.
Values such as `FR0`, `DP1`, HCL references, and description text are not additional fields.
Aliases do not create identities.
Influence `source` differs from `cites`.
Flow `steps` differs from scenario steps.
The `under` field is a typed context or workload reference.

The example declares a selected proposed Go/HCL target.
It has no binding adapter, observations, probability observations, or completed implementation.
Its premises are `to_validate`, and the local contracts have no trusted review importer.
Qualitative support therefore remains unresolved.
Unavailable binding evidence cannot become verified.
Comparative feasibility and information remain unresolved.
The example's leaf controls and checks are not test execution.
Derive profile completeness and support findings under the pinned contracts.
Do not infer them from the example's prose or labels.

## Self-design inspection expectations

The inspection fixture is `../../testdata/spec/conformance/core-1/self-design-inspection.json`.
Its `input` value is `../../examples/requirements_hcl.ad.hcl`, relative to the fixture directory.
The vendored input path is `../../testdata/spec/examples/requirements_hcl.ad.hcl`.
The fixture selects candidate `decision.assessment_contracts.candidate.go_hcl_target`, context `context.semantic_changes`, and workload `workload.semantic_changes`.
These are focused expectations, not a complete diagnostic list or measured results.

| Expected code | Critical subjects | Source severity | Architecture severity | Expected gap |
|---|---|---|---|---|
| `incomplete_module_contract` | `dp.binding_owner`, `dp.behavior_owner`, `dp.view_owner` | warning | error | These active leaf DPs have no realization modules. The six pipeline leaf modules do not realize them. |
| `missing_leaf_scenario` | `fr.binding_links`, `fr.behavior_links`, `fr.view_layout` | warning | error | The owner-change scenario verifies the parent `extension_results` criterion, not these three leaf criteria. |
| `unresolved_refinement` | Includes `fr.capture.criterion.lossless`, `fr.admit_syntax.criterion.admitted_expressions`, `fr.resolve_identity.criterion.typed_graph` | warning | error | Authored criterion allocations and aggregate parent effects lack reviewed outcome arguments. Citations alone cannot supply those arguments. |
| `unresolved_influence` | `design.requirements_hcl`, `master` | warning | error | Unknown cross-root and leaf cells prevent full classification. The `dp.syntax_profile` to `fr.trace_selection` leaf relation is explicitly unknown. |
| `support_gap` | Includes `assumption.semantic_contracts`, `assumption.fixed_expression_profile`, `assumption.fixed_owner_contracts` | warning | warning | These premises remain `to_validate`. Mechanisms and citations are not trusted substantive review results. |
| `unresolved_feasibility` | `decision.assessment_contracts.candidate.go_hcl_target` | warning | warning | No applicable success evidence establishes every required criterion or constraint. The selected candidate remains provisional. |

The fixture expects these conditional classifications: `design.requirements_hcl` and `master` are unresolved.
`dp.assessment_pipeline` and `dp.checked_graph` are decoupled.
`dp.owner_envelopes` is uncoupled.
Supported independence and feasibility remain unresolved.
Information has status `unknown` and value `null`.
The expected source operation is `success_with_warnings`.
The expected architecture operation is `failure`.
These findings do not change authored criteria, mechanisms, or conclusions.
Assess the `context.process_failure` context separately.
Do not carry semantic-change zeros into that context.

## Step 1 gate disposition

The schema, assessment, diagnostics, fixture expectations, self-design fields, and owner identities are documented in `schema.md` and this file. The vendored contract bytes and their provenance are recorded in [pin.md](pin.md). No concrete normative blocker remains. No validator, test, build, lint, or formatting result is claimed.
