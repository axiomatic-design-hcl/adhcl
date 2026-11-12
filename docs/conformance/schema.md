# Core version 1 schema gate

This document records the step 1 schema decisions for the implementation plan. The pinned RequirementsHCL core version 1 contracts define the normative rules.
The schema defines fields and types. The assessment contract takes precedence for assessment behavior. The diagnostic contract defines codes, severities, and operation status.
Conformance fixtures record expected outcomes. They do not show measured validator success.

The inspected upstream sources are `../spec/docs/schema.md`, `../spec/docs/language.md`, `../spec/docs/assessment-contract.md`, and `../spec/docs/diagnostics.md`.
They also include `../spec/conformance/core-1/expected.json` and `../spec/examples/requirements_hcl.ad.hcl`.
The implementation uses exact vendored copies under `../../testdata/spec/`. Their provenance is recorded in [pin.md](pin.md).
The pinned revision is `01c50bf3d141624c8f541a4797f0224b7658d003`.
It contains the normative contracts and all copied fixtures.

## Decoding rules

Use the schema tables as the complete inventory. Examples do not extend the schema. All authored fields forbid null.
Unknown fields, blocks, enum values, duplicate attributes, object keys, and namespace labels are errors. Do not coerce values.
Required or optional strings must be nonblank when supplied. Lists preserve order. Reference lists cannot repeat an identity.
An empty collection is accepted only where stated. It remains an explicit declaration. An omitted optional value is not an empty value, zero, false, provenance, or success.
No defaults exist unless listed below. The schema document defines type grammar, exact cardinalities, conditional requirements, null rules, reference paths, and field inventories.

| Type | Meaning |
|---|---|
| `string` | Nonblank literal HCL string. |
| `number` | Exact native HCL numeric value, including negative and exponent forms. |
| `integer` | Number with exact integral value, including `1.0` and `1e0`. |
| `bool` | Literal Boolean. |
| `scalar` | String, number, or Boolean, never null or reference. |
| `ref(T)` | One absolute static traversal to target `T`. |
| `list(T)`, `refs(T)` | Literal list; `refs(T)` is a list of references. Cardinality is field-specific. |
| `one_or_more_refs(T)` | One reference or a nonempty reference list. This is the only scalar/list reference union. |
| `scope_ref` | `design.D` or `dp.D`. |
| `under_ref` | `context.C` or `workload.W`. |
| enums | Exact, case-sensitive strings from each field table. |

For native syntax, literal collections, strings, heredocs, escaping, minus-number support, and rejected expressions, use [`language.md#syntax-profile-and-references`](../../testdata/spec/docs/language.md#syntax-profile-and-references).
Strings that look like traversals are not references. References cannot access attributes, indexes, or object keys.
Preserve full target identity and source range.

## Blocks, labels, and namespaces

The complete block/cardinality matrix is [`schema.md#files-and-block-inventory`](../../testdata/spec/docs/schema.md#files-and-block-inventory).
It allows one merged `design` and any number of listed top-level core blocks.
It also allows named FR `criterion` blocks and one or more named candidates in each decision.
Each candidate can contain at most one unnamed `information` block.
Flows can contain unnamed `loop` blocks. A manifest can contain one named project and any number of extensions.
Attachment files can contain named attachments. Only `influence`, `information`, and `loop` are unlabeled.
No other child blocks are allowed.
The cited upstream table lists every block, including zero cardinalities.

Every label matches `^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`.
Core named top-level namespaces are separate by block type and unique across merged core files.
Nested identities are `fr.F.criterion.C` and `decision.D.candidate.C`.
Their labels are unique only within the containing FR or decision.
Manifest extensions use `extension.E` in one manifest-wide namespace.
Project and attachment labels have no authored reference target. Unnamed block occurrences retain distinct ranges.
Never shorten paths or treat aliases, candidate-local labels, or external labels as references.
The full namespace and uniqueness tables are [`schema.md#labels-namespaces-and-references`](../../testdata/spec/docs/schema.md#labels-namespaces-and-references).

Core target aliases are: `criterion` → `fr.F.criterion.C`; `candidate` → `decision.D.candidate.C`; `acceptance` → criterion or scenario; `step` → module, FR, or DP; `affected` → FR or DP; `scope` → design or DP.
All other field targets appear in their block inventory below.
Core references resolve across selected core files, including forward references.
They cannot target manifests, extensions, or attachments. Only attachment envelopes refer to extensions.


## Field inventory and precise authority pointers

The tables below name every block family. Each linked upstream section gives every field and its required or optional status, type, list length, default, enum, null rule, conditional requirement, and reference.
Follow those rows exactly. This index adds no rules.

| Block | Normative field table / rules |
|---|---|
| `source` | [`schema.md#source`](../../testdata/spec/docs/schema.md#source): required `title`; optional `doi`, `url`, `pages`. |
| `design` | [`schema.md#design`](../../testdata/spec/docs/schema.md#design): required `boundary`; optional descriptive `level`; optional workload reference. |
| `stakeholder`, `need`, `constraint` | [`schema.md#stakeholder-need-and-constraint`](../../testdata/spec/docs/schema.md#stakeholder-need-and-constraint): required role/statement fields; optional `why`, stakeholder link, and descriptive kind as specified. |
| `assumption`, `context`, `workload`, `state` | [`schema.md#assumption-context-workload-and-state`](../../testdata/spec/docs/schema.md#assumption-context-workload-and-state): status enum/default; assumption/context references; range-map shape and constraints. |
| `fr`, `criterion` | [`schema.md#fr`](../../testdata/spec/docs/schema.md#fr) and [`schema.md#criterion`](../../testdata/spec/docs/schema.md#criterion): criteria cardinality; hierarchy and need links; operational triple, operator-dependent value types, `under`, allocation target. |
| `dp` | [`schema.md#dp`](../../testdata/spec/docs/schema.md#dp): required nonempty controls, responsibility, lifecycle enum; state, constraints, and premise reference lists. |
| `influence` | [`schema.md#influence`](../../testdata/spec/docs/schema.md#influence): required source DP list, target FR, effect enum; conditional mechanism/because; provenance and applicability. `source` is a locator string; `cites` is source identity metadata. |
| `cells` | [`schema.md#cells`](../../testdata/spec/docs/schema.md#cells): scoped default, `none`/`unknown`, conditional `because`; no default material effect. |
| `expect` | [`schema.md#expect`](../../testdata/spec/docs/schema.md#expect): classification enum, optional full order, default `basis = "source"`, assertion-only `under`. |
| `coupling` | [`schema.md#coupling`](../../testdata/spec/docs/schema.md#coupling): scope and affected FR/DP list, reason, optional descriptive fields and positive iteration bound. |
| `decision`, `candidate`, `information` | [`schema.md#decision`](../../testdata/spec/docs/schema.md#decision), [`schema.md#candidate`](../../testdata/spec/docs/schema.md#candidate), [`schema.md#information`](../../testdata/spec/docs/schema.md#information): required comparison and member refs, context/workload/horizon inheritance, candidate revision, information basis/evidence rules. |
| `evidence` | [`schema.md#evidence`](../../testdata/spec/docs/schema.md#evidence): every field required; exact candidate/exposure identity; joint Wilson trial metadata and numeric bounds. |
| `pv` | [`schema.md#pv`](../../testdata/spec/docs/schema.md#pv): required realized DP references and optional descriptive fields. Tool-specific binding is attachment data. |
| `module` | [`schema.md#module`](../../testdata/spec/docs/schema.md#module): exclusive realization/composite forms, containment, contracts, state, dependency, and acceptance references. |
| `flow`, `loop` | [`schema.md#flow`](../../testdata/spec/docs/schema.md#flow), [`schema.md#loop`](../../testdata/spec/docs/schema.md#loop): ordered nested step shapes and typed references; flow prerequisite rules; loop endpoints and finite bound. |
| `scenario` | [`schema.md#scenario`](../../testdata/spec/docs/schema.md#scenario): required `kind` enum; `verifies`, `vary`, expected effect lists, steps, description, and applicability. |
| `project`, `extension`, `attachment` | [`schema.md#project`](../../testdata/spec/docs/schema.md#project), [`schema.md#extension`](../../testdata/spec/docs/schema.md#extension), [`schema.md#attachment`](../../testdata/spec/docs/schema.md#attachment): exact manifest fields, paths, owner references, subject types, and closed payload. |
### Metadata

`schema.md#common-metadata` is exhaustive. Optional `cites` accepts one source reference or a nonempty list.
It applies to every core block except `source`, including criterion, candidate, information, and loop. It records provenance only.
`origin` applies to core blocks except source and criterion. Its enum is `user`, `source`, `proposed`, or `inferred`.
Criterion has its own `user`, `proposed`, or `source` enum. `alias` applies only to FR, DP, PV, and module.
An alias is a display string without uniqueness or label grammar. No other block implicitly accepts metadata.
Influence `source` is a free-text locator. It does not replace `cites`.
Citations, origin, and authored `validated` premise status do not establish review support.

### Special value and absence rules

The exact range map is a literal object. It has nonblank dimension keys, required numeric `min` and `max`, and optional string `unit`.
Endpoints are inclusive, and `min <= max`. Extra keys, null, and implicit infinities are not allowed.
`criterion.value` depends on `op`. Numeric comparison uses a number. `==` uses one scalar.
`in` uses a nonempty homogeneous unique scalar list or an exact `{min,max}` numeric interval.
`flow.steps` is a nonempty ordered list of typed references or nonempty flat reference groups.
It does not allow deeper nesting, strings, objects, or duplicates.
`under` references context or workload. It is not prose. `expect.under` qualifies only that assertion.
It does not select a global context.
Decision-level and candidate context, workload, and horizon resolve separately under the schema and assessment contract.
Do not infer them from labels. See the linked field tables for conditional presence and defaults.

## Fixed owner identities and payloads

The fixed built-in registry identities are exactly:

| Label / reference | Role | Contract | Required subject |
|---|---|---|---|
| `terraform_binding` / `extension.terraform_binding` | `binding` | `requirements_hcl/terraform_binding/1` | PV |
| `event_model_link` / `extension.event_model_link` | `behavior_link` | `requirements_hcl/event_model_link/1` | module |
| `html_view` / `extension.html_view` | `view` | `requirements_hcl/html_view/1` | design |

These are exact identities, not inferred aliases. Every declared owner is required, even when unused.
An unknown label, wrong role, unavailable implementation, or contract mismatch is an error.
Owner inventories are closed. Binding requires strings `address` and `observation_source`.
Behavior link requires strings `model`, `revision`, and `target_label`, plus `target_kind = "slice"`.
HTML view requires `layout` (`requirements_first` or `matrices_first`) and `show_source_locations = true`.
All payload fields forbid null, blank strings, extra keys, and implicit conversion.
The schema's owner payload section is authoritative.

## Explicit answers to implementation-plan step 1

| Old gate question | Normative answer |
|---|---|
| Pin revision and precedence | Revision: `01c50bf3d141624c8f541a4797f0224b7658d003`. The files listed in [pin.md](pin.md) match that commit. The schema defines fields. The assessment contract governs assessment and takes precedence over language summaries. Diagnostics defines codes and status. |
| Complete block inventory | `schema.md#files-and-block-inventory` gives complete core, nested, manifest, and attachment blocks, labels, and cardinalities. |
| Type/cardinality/required/default/null/enums/references | Every block field table linked above states these. No authored null is admitted. No implicit coercion. |
| Reference forms, target paths, hierarchy reachability | Schema reference table and hierarchy rules define scalar/list forms, exact identities, typed targets, cross-file resolution, exactly one root, acyclic reachability including inactive alternatives. |
| `ranges`, criterion value, `under`, origins, enums, flow and loop | Exact shapes and domains are defined by the workload, criterion, metadata, influence, flow, loop, scenario, and expectation field tables. |
| `cites` and influence `source` | `cites` accepts scalar or nonempty source-ref list. `influence.source` is a locator string, distinct from provenance references. |
| Example-only scenario and decision fields | `scenario.kind` enum and all scenario fields are specified; decision context/workload/horizon and candidate overrides/inheritance are specified. |
| Review support | No trusted review importer or accepted-review field exists. Qualitative support requiring review remains unresolved; citation or premise status cannot establish it. |
| Profile severity | Diagnostic contract assigns each rule its source and architecture severity. The complete stable catalog is in `assessment.md`; incomplete coverage, refinement, applicability, influence and support gaps are explicitly separated. |
| Expectations with conflict/unknown | Assessment contract defines source, conditional, supported basis, conflicts, unknowns, and order. Conflict cannot satisfy classification; unknown can satisfy only explicit unresolved without conflict; supported still requires accepted support. |
| Known mismatch and unavailable input | Mismatch findings such as `binding_mismatch` and `behavior_target_missing` are errors. Missing external observations or exact behavior revision is `unavailable_external_input`, warning in both profiles, qualified unknown and can succeed. |
| Owner identities | Three exact labels, roles, contracts, and subject types are listed above and in the schema. |
| Self-design field coverage and expected gaps | The example is an integration case, not schema authority. `assessment.md` records its field coverage and expected unresolved support/feasibility and profile findings. |
| Expected conformance cases | `assessment.md` indexes all fixture cases and expected outcomes. Fixture status expressly says expectations are not measured validator results. |

The vendored bytes listed in [pin.md](pin.md) pin the local conformance snapshot. The baseline SHA identifies the older upstream revision, not the updated source contents.
No new upstream commit is required. The fixtures record expected outcomes, not measured validator conformance.
No missing normative field rule remains in the cited schema.
