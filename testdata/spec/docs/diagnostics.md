# Diagnostic contract

This contract tells implementers which findings affect a RequirementsHCL operation.
It defines stable rule codes for `requirements_hcl/core/1` and the initial attachment contracts.
The [schema](schema.md) defines admitted facts.
The [assessment contract](assessment-contract.md) defines the conclusions drawn from those facts.
These are acceptance rules, not results from an implemented validator.

## Finding records and severity

Each finding contains a code, severity, subject, message, and available source ranges.
Related claims retain their own ranges.
A missing file or output failure has a path or operation subject, not an invented line number.
Unlabeled influences use their source file and block range as their displayed subject.
This source locator is not a persistent semantic identity.

The severities are `error` and `warning`.
An error makes the requested operation fail.
A warning retains a gap without failing the operation.
Changing a profile changes severity, not the underlying claim or its support.
The default profile is `source`.

The following catalog covers both profiles:

| Code | Condition | Source | Architecture |
|---|---|---|---|
| `input_failure` | A selected input cannot be read or violates file selection or containment rules. | error | error |
| `invalid_syntax` | Native parsing fails or an expression violates the syntax profile. | error | error |
| `invalid_structure` | A block or field violates its schema, including unknown fields, nulls, enums, and cardinality. | error | error |
| `duplicate_identity` | Labels, attributes, object keys, selected physical files, or result identities collide. | error | error |
| `invalid_reference` | A traversal has the wrong target kind or no target. | error | error |
| `invalid_hierarchy` | The hierarchy has a cycle, an unreachable entity, or an invalid parent relationship. | error | error |
| `invalid_selection` | Candidate lifecycle, membership, ancestry, or selected conditions conflict. | error | error |
| `conflicting_influence` | Applicable exact records assert both material and none, or applicable defaults conflict. | error | error |
| `ambiguous_pairing` | More than one complete control pairing exists. | error | error |
| `uncontrolled_requirement` | No distinct control can be assigned to every required sibling FR. | error | error |
| `redundant_control` | A sibling set has controls left outside any complete one-to-one pairing. | error | error |
| `ineffective_control` | A uniquely paired diagonal has a known zero effect. | error | error |
| `conflicting_ownership` | Leaf controls claim the same state or modules claim the same realization. | error | error |
| `incomplete_criterion` | An active FR has no criterion, or a criterion lacks a complete operational predicate. | warning | error |
| `incomplete_module_contract` | An active leaf lacks a realization module or required contract fields. | warning | error |
| `missing_leaf_scenario` | No applicable scenario covers an active leaf's acceptance criteria. | warning | error |
| `incomplete_feedback_contract` | Residual coupled iteration lacks its bound, convergence, stopping, or failure contract. | warning | error |
| `unresolved_refinement` | A parent effect or criterion allocation lacks the required outcome argument. | warning | error |
| `unresolved_applicability` | Conditions or invalidated premises leave a required relation unresolved. | warning | error |
| `unresolved_influence` | Unknown cells prevent a complete active sibling or master classification. | warning | error |
| `shared_resource_review_gap` | Shared state, resources, or PVs lack an applicable influence assessment. | warning | error |
| `support_gap` | A claim lacks applicable reviewed support, including unavailable review import. | warning | warning |
| `unresolved_feasibility` | A candidate lacks applicable success evidence for a required criterion or constraint. | warning | warning |
| `infeasible_candidate` | Applicable evidence establishes failure of a required criterion or constraint. | error | error |
| `invalid_evidence` | Trials or evidence metadata violate the empirical evidence contract. | error | error |
| `incomparable_evidence` | A requested information comparison lacks common outcomes or compatible exposure. | warning | warning |
| `failed_expectation` | An authored expectation is false or cannot establish its requested basis. | error | error |
| `invalid_attachment` | An envelope or owner payload violates its contract. | error | error |
| `unavailable_owner` | A required owner is absent, has the wrong role, or lacks the exact contract. | error | error |
| `owner_failure` | A required owner cannot complete its processing. | error | error |
| `duplicate_binding` | A binding repeats the corresponding core PV address under the rule below. | error | error |
| `binding_mismatch` | Available observations contradict a required binding. | error | error |
| `behavior_target_missing` | An available exact behavior revision lacks the required target. | error | error |
| `unavailable_external_input` | Required observation data or an exact behavior revision is unavailable. | warning | warning |
| `render_failure` | The requested report cannot be rendered or delivered. | error | error |

A valid coupled mapping is not itself malformed input.
Report its cycle and residual coupling without inventing an error for the classification alone.
A false expectation or incomplete required feedback contract can still fail that operation.

Emit independent findings after a prerequisite fails, but do not analyze a guessed graph.
For example, an invalid reference blocks dependent pairing rather than producing an invented uncontrolled FR.
One cause can affect several subjects.
Report the cause and its affected subjects without creating a separate error for each blocked downstream calculation.

Use the most specific code when several catalog descriptions cover the same defect.
Use `duplicate_identity` for duplicate attributes or object keys, even when the native parser reports them.
Use `invalid_syntax` for other parsing and expression-profile failures.
Use `invalid_reference` for a missing or wrong-kind attachment owner or subject.
Use `invalid_attachment` for other envelope and payload schema defects.
Use `invalid_evidence` for invalid trial counts and evidence-specific metadata rules.
Do not also emit `invalid_structure` for the same defect.

## Partial processing and operation status

Each stage reports one of these statuses:

| Status | Meaning |
|---|---|
| `complete` | The stage finished all applicable processing, including findings and qualified unknowns. |
| `failed` | The stage attempted required work but could not complete it. |
| `blocked` | A failed prerequisite prevents required work. |
| `not_requested` | The requested operation does not require this stage. |

A completed assessment can contain error findings.
Its processing status is not a claim that the model passes.
An operation succeeds only when every required stage completes and no applicable error finding remains.
An invalid model therefore fails even when diagnostic collection completes.
A stage with no applicable active subjects completes with no results.
A required unavailable owner does not count as a stage with no subjects.

Publish separate statuses for core assessment, each owner, composition, and requested rendering.
Retain completed core results after an attachment failure.
Composition completes when it faithfully combines available results and their failures.
Composition cannot turn a failed required stage into operation success.
A validation-only operation does not require HTML rendering, even when it assesses an HTML view payload.

The following outcomes keep processing and conformance separate:

| Owner outcome | Processing | Finding | Requested project operation | Conformance conclusion |
|---|---|---|---|---|
| Binding `match` | complete | None for this comparison | Can succeed | Only this binding matches. |
| Binding `mismatch` | complete | `binding_mismatch` | Fails | This binding does not conform. |
| Behavior `resolved` | complete | None for target lookup | Can succeed | Target identity only, not behavior satisfaction. |
| Behavior `missing` | complete | `behavior_target_missing` | Fails | Required target is absent. |
| External `unknown` | complete | `unavailable_external_input` | Can succeed with warning | Unresolved. |
| Invalid envelope or payload | complete if diagnostics finish | `invalid_attachment` | Fails | Not established. |
| Unavailable or failed owner | blocked or failed | `unavailable_owner` or `owner_failure` | Fails | Not established. |
| Requested rendering fails | failed | `render_failure` | Fails | Completed assessment remains unchanged. |

A core-only operation has no required attachment stages.
A manifest operation requires every declared owner, including unused declarations.
Exact numeric exit codes and output stream choices belong to the tool's command contract.
The language requires success versus failure and a distinct usage-failure category.

## Binding duplication authority

The Terraform binding owner checks the PV that is its attachment subject.
For this contract, duplication means that the PV has `address` equal to the attachment's `data.address` string.
Compare admitted string values exactly, without normalization or address inference.
The owner reports `duplicate_binding` at both declarations, even when the strings agree.
A different descriptive core address does not establish duplication by this rule.
The owner cannot reinterpret `note`, `kind`, or another owner's data to invent a duplicate.

Run this check during attachment assessment and retain it during composition.
The core-only assessment does not run this owner rule.
Adding the attachment can change project findings and success, but cannot change core cells or classification.
This explicit correspondence replaces an unrestricted search for semantically similar tool-specific prose.
