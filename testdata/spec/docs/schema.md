# RequirementsHCL core version 1 schema

This reference defines every admitted block, field, type, label, and reference in `requirements_hcl/core/1`.
It is for implementers and model authors.
It does not claim that a decoder or validator exists.

This document is authoritative for schema details.
The [language reference](language.md) explains their use.
The [assessment contract](assessment-contract.md) defines selection, influence analysis, support, feasibility, and evidence rules.
The [diagnostic contract](diagnostics.md) defines profile severities and operation status.
A structurally valid field does not establish its claim or satisfy a review profile.

## Decoding rules

Decode native HCL under the language reference's restricted expression profile.
Admit literals and static traversals only where this schema permits them.
Do not evaluate computed expressions or templates.
The only permitted operator is a minus on a numeric literal.
Valid native HCL can still violate a field's contract.

The following rules apply to every table:

- `R` means required. `O` means optional. Conditional requirements appear in the contract column.
- An omitted optional field has no value unless its table states a default or inheritance rule. Absence does not assert zero, false, an empty string, success, or source provenance.
- No authored field admits `null`, including values inside collections. Derived output nulls are a separate assessment contract.
- Unknown fields, blocks, enum values, duplicate attributes, duplicate object keys, and duplicate labels in one namespace are errors. Block bodies admit only their listed fields, common metadata where permitted, and listed child blocks. Top-level attributes are forbidden.
- There are no implicit type conversions. In particular, a string is not a reference, a number is not a string, `1` is not `true`, and a scalar is not a list unless that field explicitly admits both.
- Required strings and optional strings, when present, must contain at least one non-whitespace character. Do not trim or rewrite admitted strings. Heredocs and quoted strings have the same field type.
- Lists preserve authored order. A reference list must not repeat an identity, even when order has no assessment meaning. Distinct influence blocks with the same endpoints are not duplicate declarations and must be retained.
- An explicitly empty list or object is valid only where stated. An empty collection records an explicit empty declaration; it is not automatically equivalent to an omitted field for profile completeness or source identity.

The schema uses these type names:

| Type | Exact admitted value |
|---|---|
| `string` | A nonblank literal HCL string. |
| `number` | A native HCL number, including negative and exponent forms. No binary floating-point rounding is required during admission. |
| `integer` | A `number` whose exact value is integral; `1.0` and `1e0` are integral. |
| `bool` | Literal `true` or `false`. |
| `scalar` | A `string`, `number`, or `bool`, never a reference or null. |
| `ref(T)` | One absolute static traversal resolving to an identity of target type `T`. |
| `list(T)` | A literal list containing only values of `T`. Tables state its permitted length. |
| `refs(T)` | `list(ref(T))`. Tables state its permitted length. |
| `one_or_more_refs(T)` | One `ref(T)` or a nonempty `refs(T)` list. Only fields using this type admit the scalar/list union. |
| `scope_ref` | `ref(design)` or `ref(dp)`, selecting the corresponding sibling set. |
| `under_ref` | `ref(context)` or `ref(workload)`, selecting applicability, not a prose condition. |

An enum is a string with exactly one of the listed, case-sensitive values. A descriptive string such as `constraint.kind` is not an enum and is never interpreted as a new block type or namespace.

## Files and block inventory

A core input contains exactly one merged design root. A core directory merges immediate `*.ad.hcl` files only. It does not discover attachments or a manifest. A directory with no core members is invalid. Exclude `.ad.ext.hcl` and `.ad.project.hcl` files from the core suffix match.

The following table is the complete block inventory. `0..n` means any number, including none. Every named block has exactly one quoted label. Unnamed blocks have zero labels.

| File or parent body | Permitted blocks and cardinality |
|---|---|
| Merged core files | `design`: exactly 1; each of `source`, `stakeholder`, `need`, `constraint`, `assumption`, `context`, `workload`, `state`, `fr`, `dp`, `influence`, `cells`, `expect`, `coupling`, `decision`, `evidence`, `pv`, `module`, `flow`, `scenario`: 0..n |
| `fr` | Named `criterion`: 0..n |
| `decision` | Named `candidate`: 1..n |
| `candidate` | Unnamed `information`: 0..1 |
| `flow` | Unnamed `loop`: 0..n |
| Manifest | Named `project`: exactly 1; named `extension`: 0..n |
| Attachment file | Named `attachment`: 0..n |
| Every other block | No child blocks |

`influence`, `information`, and `loop` are the only unnamed blocks. Core `import`, `module.handles`, matrix blocks, review blocks, and argument blocks do not exist. No attachment payload adds core syntax.

## Labels, namespaces, and references

Every block label must match the ASCII regular expression `^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`. Labels cannot start with a digit, contain uppercase letters, or contain adjacent or trailing underscores. A single word such as `catalog` is valid snake_case.

Top-level named core blocks have separate namespaces by block type.
For example, `fr.catalog`, `dp.catalog`, and `source.catalog` are different identities and can coexist.
Labels are unique within their type across the merged core.
The design label is not a prefix for other core identities.

Nested namespaces are exact paths:

| Declaration | Reference identity | Uniqueness scope |
|---|---|---|
| Top-level named core block `TYPE "LABEL"` | `TYPE.LABEL` | The merged core's `TYPE` namespace |
| `criterion "C"` inside `fr "F"` | `fr.F.criterion.C` | Criteria of that FR only |
| `candidate "C"` inside `decision "D"` | `decision.D.candidate.C` | Candidates of that decision only |
| Manifest `extension "E"` | `extension.E` | All extension declarations in that manifest |
| Manifest `project "P"` | No authored reference target | Exactly one project block |
| `attachment "A"` | No authored reference target | All listed attachment files |
| Unnamed blocks | No authored reference target | Retain each occurrence and source range |

A criterion label can recur in another FR.
A candidate label can recur in another decision.
Shortened paths such as `criterion.fast`, `candidate.fast`, or `fr.F.fast` are invalid.
A traversal cannot access an attribute, list element, or object key.
Each reference must use exactly the path shape in this table.

The following target aliases complete the reference contracts used in field tables:

| Target | Admitted identities |
|---|---|
| `criterion` | `fr.F.criterion.C` |
| `candidate` | `decision.D.candidate.C` |
| `acceptance` | A `criterion` or `scenario.S` |
| `step` | `module.M`, `fr.F`, or `dp.D` |
| `affected` | `fr.F` or `dp.D` |
| `scope` | `design.D` or `dp.D` |

Resolve core references only against core blocks in the selected model. Core facts cannot reference `extension`, `project`, or attachments. Attachment envelopes alone use `extension` references. Forward and cross-file references are permitted. Missing targets and wrong-kind targets are errors, not unresolved effect evidence.

Labels, aliases, titles, metric strings, source locators, candidate versions, external target strings, and file names are distinct concepts. An `alias = "FR11"` is display provenance, not another reference identity. `selected = "fast"` is a local candidate label, not a global lookup for every candidate named `fast`. An external `target_label` is an owner-interpreted string, not a core label or traversal.

## Common metadata

The following is the complete applicability table for common metadata. No metadata field is implicitly admitted elsewhere.

| Field | Blocks accepting it | Type and presence | Meaning |
|---|---|---|---|
| `cites` | Every core block except `source`, including `criterion`, `candidate`, `information`, and `loop` | O `one_or_more_refs(source)`; no default; empty list forbidden | Provenance sources for this record. A scalar and a one-element list denote the same citation set. |
| `origin` | Every core block except `source` and `criterion` | O enum `user`, `source`, `proposed`, `inferred`; no default | A user-supplied fact, a source-stated fact, an authored proposal, or an author inference. |
| `origin` | `criterion` only | O enum `user`, `proposed`, `source`; no default | The basis of the acceptance target, not evidence that it was achieved. |
| `alias` | `fr`, `dp`, `pv`, `module` only | O `string`; no default | An external or historical display identifier. Aliases need not satisfy the block label grammar or be unique. |

An omitted origin remains unspecified.
Do not infer `source` from a citation.
A citation does not establish an argument, a validated premise, or supported independence.
A review document can be a cited source without becoming a trusted support record.
No field in this schema imports accepted substantive reviews.

Manifests, extension declarations, attachments, and owner payload objects accept no common metadata. Owner payloads accept only their contract's listed keys.

## Sources and operating premises

The following tables list fields in addition to permitted common metadata.

### `source`

| Field | Presence and type | Contract |
|---|---|---|
| `title` | R `string` | Source title. |
| `doi` | O `string` | Documentary identifier; no automatic lookup. |
| `url` | O `string` | Documentary locator, including a relative path; no automatic fetching or file inclusion. |
| `pages` | O `string` | Source page locator, for example `"15–19"`; not a number or list. |

### `design`

| Field | Presence and type | Contract |
|---|---|---|
| `boundary` | R `string` | What the design includes and excludes. |
| `level` | O `string` | Descriptive abstraction level; not a profile selector or enum. |
| `workload` | O `ref(workload)` | Default design workload when more specific selection does not supply one. Absence leaves workload unspecified. |

### `stakeholder`, `need`, and `constraint`

| Block | Field | Presence and type | Contract |
|---|---|---|---|
| `stakeholder` | `role` | R `string` | Affected person or group; not an enum. |
| `need` | `statement` | R `string` | Desired stakeholder outcome. |
| `need` | `why` | O `string` | Reason for the need. |
| `need` | `stakeholder` | O `ref(stakeholder)` | One originating stakeholder. |
| `constraint` | `statement` | R `string` | Limit on acceptable solutions. |
| `constraint` | `kind` | O `string` | Descriptive category such as `technology` or `compatibility`; not an enum. |

### `assumption`, `context`, `workload`, and `state`

| Block | Field | Presence and type | Contract |
|---|---|---|---|
| `assumption` | `statement` | R `string` | Premise being asserted. |
| `assumption` | `status` | O enum `assumed`, `to_validate`, `validated`, `invalidated` | Default `to_validate`. `validated` is an authored status, not imported proof. |
| `context` | `description` | R `string` | Named operating condition. |
| `context` | `assumes` | O `refs(assumption)` | Empty allowed; default `[]`. |
| `context` | `excludes` | O `refs(context)` | Empty allowed; default `[]`; self-reference forbidden. Context incompatibility is assessed under the assessment contract. |
| `context` | `workload` | O `ref(workload)` | Workload associated with this condition. |
| `workload` | `description` | R `string` | Input population and exposure description. |
| `workload` | `ranges` | O range map | Empty object allowed; absence leaves ranges unspecified. Exact shape follows this table. |
| `state` | `description` | R `string` | Information owned or read by leaf controls. |

A range map is a literal object.
Its nonblank string keys name measured dimensions, not block labels or references.
Each value requires numeric `min` and `max` fields and permits an optional string `unit`.
Require `min <= max`, with inclusive endpoints.
Reject null endpoints, implicit infinities, and extra keys.
Describe nonnumeric dimensions in `description`. Do not invent numeric values.

Range keys need not match criterion metric strings.
They create no criterion references.

For example, the following is a range map:

```hcl
ranges = {
  request_rate = { min = 0, max = 100, unit = "requests/s" }
  batch_size   = { min = 1, max = 50 }
}
```

## Functional hierarchy and controls

### `fr`

| Field | Presence and type | Contract |
|---|---|---|
| `statement` | R `string` | Required outcome. |
| `need` | O `one_or_more_refs(need)` | Required on a root FR; optional on a child FR. An explicit empty list is invalid. |
| `parent` | O `ref(dp)` | Absence makes this a root FR attached implicitly to the one design root. Presence makes it a child FR inside that DP. |
| `bounded_by` | O `refs(constraint)` | Empty allowed; default `[]`. |

An FR contains zero or more named `criterion` blocks.
Missing criteria are profile gaps. The decoder does not invent them.
A child without `need` inherits the union of needs from its parent DP's controlled FRs.
An explicit child `need` supplies its own links without deleting ancestor context.
The tool does not copy need links or criteria into new authored blocks.

### `criterion`

| Field | Presence and type | Contract |
|---|---|---|
| `statement` | O `string` | Qualitative acceptance condition, including a source condition lacking measurable limits. |
| `metric` | O `string` | Name or description of the observed quantity or predicate. |
| `op` | O enum `<=`, `<`, `>=`, `>`, `==`, `in` | Comparison operation. |
| `value` | O operator-dependent value | Exact shapes follow this table. |
| `unit` | O `string` | Measurement unit; meaningful only with the operational triple. No implicit unit conversion. |
| `under` | O `under_ref` | Applies the criterion to this context or workload. Absence adds no condition beyond inherited operating context. |
| `allocated_from` | O `ref(criterion)` | Criterion on an FR controlled by this FR's parent DP. Invalid on a root FR. |

A criterion requires `statement`, the complete operational triple `metric`, `op`, `value`, or both.
Use a qualitative statement to retain a source gap.
A partial operational triple is invalid schema.
The architecture profile requires the complete triple.
The metadata table defines the criterion-specific `origin` values.

The operator fixes `value`'s type and meaning:

| `op` | Exact `value` type | Meaning |
|---|---|---|
| `<=`, `<`, `>=`, `>` | `number` | Numeric comparison. Strings are not lexically ordered by these operators. |
| `==` | `scalar` | Exact type-preserving equality. Numeric equality compares numeric values; strings and Booleans retain their types. |
| `in` | Nonempty homogeneous `list(scalar)`, or exactly `{ min = number, max = number }` | Membership in a finite set, or in an inclusive numeric interval with `min <= max`. |

A membership list must use one scalar type throughout and contain no duplicate values. `[1, 10]` means either of those two numbers, not all numbers between them. `{ min = 1, max = 10 }` means the inclusive interval. Nested lists, reference values, arbitrary objects, empty sets, and null values are invalid. Units belong in `unit`, not the interval object. A Boolean `value = true` with `op = "=="` is an operational predicate declaration, not evidence of success.

`allocated_from` records allocation only. It does not supply an outcome argument or copy the parent's acceptance range. A root criterion's `under` condition remains part of its own outcome; child operational predicates must not silently become unchanged copies.

### `dp`

| Field | Presence and type | Contract |
|---|---|---|
| `controls` | R nonempty `refs(fr)` | Proposed control pairings. All targets must belong to one sibling set. |
| `responsibility` | R `string` | Mechanism or parameter responsibility. |
| `decision` | R enum `proposed`, `selected`, `rejected`, `superseded` | Authored lifecycle state; not implementation status. |
| `why` | O `string` | Selection rationale. |
| `role` | O `string` | Descriptive role; not an enum or module identity. |
| `owns_state` | O `refs(state)` | Empty allowed; absence leaves ownership unspecified. |
| `reads_state` | O `refs(state)` | Empty allowed; default `[]`. |
| `bounded_by` | O `refs(constraint)` | Empty allowed; default `[]`. |
| `holds_if` | O `refs(assumption)` | Empty allowed; default `[]`. |

### Hierarchy constraints

Root FRs belong to the single design root without an explicit `design` field.
Hierarchy edges run from design to root FR, controlled FR to controlling DP, and parent DP to child FR.
Every declared FR and DP must be reachable from that root.
This includes inactive alternatives.
The graph must be acyclic.
Need, state, module, and citation links are not hierarchy edges.

Sibling FRs share one parent DP, or all have no parent.
A DP's `controls` must stay within one sibling set.
That set determines the DP's hierarchy position. A DP has no separate `parent` field.
A multi-FR control retains every FR and one DP identity.
Do not force the graph into a one-control-per-FR tree.

A DP is a hierarchy leaf when it has no child FRs.
An FR is a hierarchy leaf when none of its controlling DPs has child FRs.
Active-leaf assessment uses the selected snapshot, not deletion of inactive declarations.

Other core records belong to the merged design by project membership; they do not require an artificial hierarchy path. Module containment is a separate graph.

## Influences, defaults, and assertions

### `influence`

| Field | Presence and type | Contract |
|---|---|---|
| `from` | R nonempty `refs(dp)` | One independently asserted source control per list member. |
| `to` | R `ref(fr)` | Target outcome. |
| `effect` | R enum `material`, `none`, `unknown` | Claimed effect, never an implementation observation. |
| `mechanism` | O `string` | Required when `effect = "material"`. |
| `because` | O `string` | Required when `effect = "none"`. |
| `mechanism_source` | O enum `source`, `inferred` | Whether the mechanism is source-stated or inferred. Requires `mechanism`. Omission leaves its provenance unspecified. |
| `source` | O `string` | Human-readable claim locator, for example `"Figure 5, FR2/DP1 cell"`; not a source reference, external namespace, or support classification. |
| `under` | O `under_ref` | Exact context/workload applicability. Absence is unconditional applicability. |
| `holds_if` | O `refs(assumption)` | Empty allowed; default `[]`. |

`source` locates a claim within its provenance.
`cites` resolves the declared source identities.
Neither replaces the other.
`mechanism` and `because` can coexist, including on an unknown relation.
They do not change `effect` or import an accepted review.
Mark an author-added mechanism with `mechanism_source = "inferred"`.

Expand a multi-source influence into independently asserted records retaining the original record and citation provenance. This list does not represent an interaction that occurs only when the sources change together. Core version 1 has no interaction-only influence form. Source DP and target FR need not be siblings; cross-branch effects are valid.

### `cells`

| Field | Presence and type | Contract |
|---|---|---|
| `scope` | R `scope_ref` | The design's root sibling set or the child sibling set inside a DP. |
| `unlisted` | R enum `none`, `unknown` | Default effect for otherwise unlisted cells in that set. No implicit zero. |
| `because` | O `string` | Required when `unlisted = "none"`. |

A cell default cannot introduce a material effect without an exact influence and its mechanism. The scope must identify an existing nonempty sibling set. Both cell endpoints must belong to that sibling set. Defaults do not apply to arbitrary cross-branch cells. Exact facts, conflicting defaults, and support are assessed under the assessment contract.

### `expect`

| Field | Presence and type | Contract |
|---|---|---|
| `scope` | R `scope_ref` | Sibling set being asserted. |
| `classification` | R enum `uncoupled`, `decoupled`, `coupled`, `unresolved` | Expected classification; never an authored matrix result. |
| `order` | O nonempty `refs(dp)` | A proposed complete design order for the scoped active DPs, each exactly once. Not a runtime order. |
| `basis` | O enum `source`, `conditional`, `supported` | Default `source`. |
| `under` | O `under_ref` | Assertion's operating context/workload. Absence uses the assessment snapshot's conditions. |

`expect.under` does not alter global facts or select a different candidate. Assertion applicability, failure, blocked evaluation, and support qualification are defined in the assessment contract. An assertion does not supply a missing relation, premise, criterion, or trusted review result.

### `coupling`

| Field | Presence and type | Contract |
|---|---|---|
| `scope` | R `scope_ref` | Sibling set containing the residual coupling. |
| `affects` | R nonempty `refs(affected)` | Affected FRs and/or DPs in the scoped branch. |
| `reason` | R `string` | Why the residual coupling exists. |
| `condition` | O `string` | Descriptive condition; not resolved applicability. Use defined model contexts for conditional influence facts. |
| `decision` | O `string` | Authored disposition or rationale, not a decision reference or lifecycle enum. |
| `normal_practice` | O `bool` | Whether iteration is stated to be normal practice; absence leaves this unspecified. |
| `residual_risk` | O `string` | Remaining risk. |
| `iteration_bound` | O positive `integer` | Maximum stated iteration count. |
| `convergence` | O `string` | Convergence argument or condition. |
| `stopping` | O `string` | Stopping rule. |
| `failure_behavior` | O `string` | Behavior when iteration cannot complete safely. |

The architecture profile requires the feedback safety fields where applicable. A complete coupling record does not remove a material cycle or prove convergence.

## Decisions and empirical evidence

### `decision`

| Field | Presence and type | Contract |
|---|---|---|
| `addresses` | R nonempty `refs(fr)` | Required outcomes for this comparison. |
| `context` | O `ref(context)` | Default operating context for candidates in this decision. |
| `workload` | O `ref(workload)` | Default workload for candidates in this decision. |
| `horizon` | O `string` | Default exposure horizon for candidates in this decision. |
| `selected` | O `string` matching the label grammar | Label of a candidate in this decision only. No selected candidate is inferred when absent. |
| `provisional` | O `bool` | Default `true`; declaring `false` does not override missing feasibility, independence, or evidence. |
| `why` | O `string` | Authored reason for the decision. |

A decision contains one or more named `candidate` blocks.
Candidate combination and selection are assessment concerns.
The decoder never chooses between decisions by source order.

### `candidate`

| Field | Presence and type | Contract |
|---|---|---|
| `dps` | R nonempty `refs(dp)` | Explicit members of this alternative. Common DPs and ancestor closure follow the assessment contract. |
| `version` | O `string` | Exact candidate revision. No implicit version and no inheritance from another field. |
| `context` | O `ref(context)` | Overrides this decision's context when present. |
| `workload` | O `ref(workload)` | Overrides this decision's workload when present. |
| `horizon` | O `string` | Overrides this decision's horizon when present. |

A candidate contains at most one unnamed `information` block.
Resolve context, workload, and horizon separately.
Use the candidate's explicit value first, then the enclosing decision's value.
If workload remains absent, use the effective context's workload, then the design's workload.
Missing context or horizon remains unspecified.
Never infer a context from a matching workload label.

If the effective context and workload disagree, the combination is invalid.
Keep authored values separate from inherited values.

An override is structurally allowed but can make candidates incomparable. Equality of effective values, not merely the presence of inheritance, governs empirical evidence applicability. The enclosing decision's metadata does not change the meaning of unrelated context or workload labels.

### `information`

| Field | Presence and type | Contract |
|---|---|---|
| `basis` | R enum `unknown`, `measured` | Unknown information or an empirical estimate request. |
| `evidence` | O `refs(evidence)` | Empty allowed for `unknown`; `measured` requires a nonempty list. Default `[]` when omitted. |

Each evidence reference must name this exact enclosing candidate through the evidence block's `candidate` field. `unknown` can retain evidence that is incomplete or inapplicable; it does not turn it into an estimate. Core version 1 has no authored probability, information score, marginal product, or analytical probability-model field. The trial contract is intentionally limited to joint empirical observations.

### `evidence`

Every field in this table is required. No default or metadata inheritance applies to evidence; it must repeat its exact exposure identity for comparison with the candidate's effective values.

| Field | Type | Contract |
|---|---|---|
| `basis` | Enum `measured` | Observed joint trials. |
| `event` | `string` | Joint success predicate description. |
| `scenario` | `ref(scenario)` | Scenario identifying all addressed operational predicates through `verifies`. |
| `candidate` | `ref(candidate)` | Exact `decision.D.candidate.C` identity. |
| `version` | `string` | Must match the candidate's explicit version. A candidate without a version cannot qualify this evidence. |
| `context` | `ref(context)` | Must match effective candidate context. |
| `workload` | `ref(workload)` | Must match effective candidate workload. |
| `horizon` | `string` | Must match effective candidate horizon exactly. No implicit duration parsing or conversion. |
| `dependence` | Enum `joint` | Every trial evaluates all required outcomes together. |
| `sampling` | `string` | Sampling assumptions supporting the trial interpretation. |
| `trials` | Positive `integer` | Number of observed joint trials. |
| `failures` | `integer` | From zero through `trials`, inclusive. |
| `confidence_method` | Enum `wilson` | Two-sided Wilson convention in the assessment contract. |
| `confidence` | `number` | Strictly greater than zero and less than one. |

A decoded evidence block is not automatically qualified evidence. Predicate coverage, compatible exposure, feasibility, sampling validity, uncertainty, and substantive support remain assessment checks. A nonmatching version or exposure must not be silently rewritten. Zero failures do not establish certainty.

## Realization, modules, and runtime flow

### `pv`

| Field | Presence and type | Contract |
|---|---|---|
| `realizes` | R nonempty `refs(dp)` | Controls realized by this process variable. |
| `kind` | O `string` | Descriptive realization category; not an enum or owner selector. |
| `address` | O `string` | Generic descriptive locator; not a command, observation, or Terraform binding request. |
| `note` | O `string` | Additional descriptive information. |

A tool-specific binding belongs in an attachment.
The [diagnostic contract](diagnostics.md#binding-duplication-authority) defines the exact duplicate-address rule and its effect on project processing.
A generic core address alone does not load an owner.

### `module`

| Field | Presence and type | Contract |
|---|---|---|
| `realizes` | O nonempty `refs(dp)` | Required for a realization module; mutually exclusive with `contains`. |
| `contains` | O nonempty `refs(module)` | Required for a composite module; mutually exclusive with `realizes`. |
| `within` | O `ref(module)` | Parent composite module. |
| `inputs` | O `list(string)` | Empty allowed; absence leaves the input contract unspecified. |
| `outputs` | O `list(string)` | Empty allowed; absence leaves the output contract unspecified. |
| `owns_state` | O `refs(state)` | Empty allowed; absence leaves ownership unspecified. |
| `dependencies` | O `refs(module)` | Empty allowed; absence leaves dependencies unspecified. |
| `invariants` | O `list(string)` | Empty allowed; absence leaves invariants unspecified. |
| `failure_behavior` | O `string` | Failure contract. |
| `acceptance_checks` | O `refs(acceptance)` | Empty allowed; absence leaves checks unspecified. Only criterion or scenario references, never prose. |

Every module is either a realization module or a composite, never both.
A composite can contain realization modules or other composites.
Containment must be acyclic, without self-containment or multiple parents.
Either `within` or `contains` can declare the relation.
When both declare it, they must agree.
`within` must target a composite module.
Missing fields do not create implicit authored declarations.

A dependency list cannot contain the module itself.
Retain dependency cycles as architecture facts.
They are not containment cycles and do not alone establish FR coupling.
Multiple realization modules cannot claim the same DP.
PV and module realization are distinct mappings.
Both can refer to the same DP.

DP state ownership belongs to leaf DPs. A state's logical owner cannot be claimed by multiple DPs. A realizing module's matching ownership declaration describes that same owner, not a second logical owner. A module cannot claim state owned by a DP it does not realize; composite modules cannot introduce independent state ownership. State reads and shared resources do not imply zero influence. Explicit `owns_state = []` and `dependencies = []` satisfy declaration presence only, not all architecture obligations.

### `flow`

| Field | Presence and type | Contract |
|---|---|---|
| `steps` | R nonempty step list | Exact shape follows this table. |
| `owner` | O `ref(module)` | Module responsible for the flow. |
| `input` | O `string` | Flow input description. |
| `output` | O `string` | Flow output description. |
| `after` | O `refs(flow)` | Empty allowed; default `[]`. Prerequisite flows, not design-order DPs. |

A step list is an ordered literal list.
Each item is one `ref(step)` or a nonempty flat list of those references.
The inner list combines independent outcomes without ordering its members.
Reject deeper lists, empty groups, instruction strings, arbitrary objects, and nulls.
Mixed module, FR, and DP identities are permitted.
Each traversal retains its type.

Each identity can occur at most once per flow, including inside groups.
Use a loop for repeated execution, not duplicate steps.
Flow prerequisites cannot contain self-references or cycles.

A flow can contain zero or more unnamed `loop` blocks. A flow's order is runtime order, not a replacement for the derived design sequence.

### `loop`

| Field | Presence and type | Contract |
|---|---|---|
| `after` | R `ref(step)` | A step in the enclosing flow after which feedback is considered. |
| `back_to` | R `ref(step)` | An earlier step in the enclosing flow. |
| `when` | R `string` | Condition for repeating; authored description, not executable code. |
| `bound` | R positive `integer` | Maximum number of feedback traversals. |

Both loop endpoints must occur in the enclosing flow's `steps`.
`back_to` must precede `after` in the outer sequence.
Inner group members share one outer position.
A loop cannot assert an order between those members.
A finite count does not supply convergence, stopping, or failure behavior.
Residual feedback needs the separate coupling and architecture contracts.
Common metadata is permitted, but no other fields or child blocks are admitted.

### `scenario`

| Field | Presence and type | Contract |
|---|---|---|
| `kind` | R enum `change`, `operational`, `failure` | Scenario category, not an inferred value. |
| `verifies` | O `refs(criterion)` | Empty allowed; absence leaves predicate coverage unspecified. An FR reference is not a criterion reference. |
| `vary` | O `refs(dp)` | Empty allowed; default `[]`. Controls varied by the scenario. |
| `expect_affected` | O `refs(fr)` | Empty allowed; default `[]`. Outcomes expected to change. |
| `expect_unaffected` | O `refs(fr)` | Empty allowed; default `[]`. Outcomes expected to remain within their criteria. |
| `steps` | O `list(string)` | Empty allowed; absence leaves scenario procedure unspecified. These are instructions, unlike flow steps. |
| `description` | O `string` | Scenario description and limits. |
| `under` | O `under_ref` | Scenario applicability. Absence adds no condition beyond the assessment snapshot. |

An FR cannot appear in both expected-effect lists of one scenario. An empty or missing `verifies` cannot qualify empirical evidence or outcome coverage. A scenario describes an acceptance check; its declaration does not assert that a run occurred.

## Manifests, owners, and attachments

### `project`

| Field | Presence and type | Contract |
|---|---|---|
| `language` | R enum `requirements_hcl/core/1` | Exact core contract. Core-only input selects this contract without a manifest. |
| `core_files` | R nonempty `list(string)` | Explicit core `.ad.hcl` paths. |
| `attachment_files` | R `list(string)` | Explicit `.ad.ext.hcl` paths; empty allowed. |

Every path is relative to the manifest directory. Reject absolute paths, glob patterns, missing files, directory entries, suffix mismatches, paths escaping the manifest directory, and repeated physical files, including aliases across lists. Resolve symbolic links before containment and file-identity checks. A `.ad.ext.hcl` or `.ad.project.hcl` file cannot be a core member. Do not recurse, expand globs, load nested manifests, or infer inputs from a source URL. All core files merge into exactly one design.

The project label identifies the input collection; it does not replace or qualify the design identity.

### `extension`

| Field | Presence and type | Contract |
|---|---|---|
| `role` | R enum `binding`, `behavior_link`, `view` | Declared owner responsibility. |
| `contract` | R `string` | Exact installed contract version; no version-range syntax or fallback. |

The following labels select the initial standard owners. Their label, role, and contract must agree:

| Label and reference | Role | Contract | Subject type |
|---|---|---|---|
| `terraform_binding`, `extension.terraform_binding` | `binding` | `requirements_hcl/terraform_binding/1` | `pv` |
| `event_model_link`, `extension.event_model_link` | `behavior_link` | `requirements_hcl/event_model_link/1` | `module` |
| `html_view`, `extension.html_view` | `view` | `requirements_hcl/html_view/1` | `design` |

Registry installation, packaging, and availability are tool policy. A manifest selects an already trusted local owner; it cannot install, download, or execute one. Other labels or contracts require an available registry entry with its own exact closed payload schema and one declared role; they are not aliases for these owners. An implementation supporting only the three standard entries rejects unavailable entries rather than accepting arbitrary payloads. Every declared owner is required even when unused.

### `attachment`

| Field | Presence and type | Contract |
|---|---|---|
| `owner` | R `ref(extension)` | Declaration in this manifest. Strings and undeclared owners are invalid. |
| `subject` | R `ref(pv)`, `ref(module)`, or `ref(design)` | Exactly the subject type required by the owner's role; must belong to this project's core model. |
| `data` | R literal object | Closed, exact owner payload schema. No references, extra keys, nulls, or expression evaluation. |

An attachment requires exactly these three fields and no child blocks.
Its label must be unique across listed attachment files.
Multiple attachments can name one subject and retain separate results.
The view contract has the stricter limit defined next.
Attachments cannot refer to each other.
An invalid envelope does not authorize owner invocation with guessed inputs.

### Standard owner payloads

These tables exhaust the admitted keys. All fields are required; none has a default or accepts an empty string, null, additional object keys, or implicit conversion. External identifiers are strings, never core traversals.

For `requirements_hcl/terraform_binding/1`, `data` has exactly these fields:

| Field | Type | Contract |
|---|---|---|
| `address` | `string` | Terraform resource address to compare. |
| `observation_source` | `string` | Identifier of caller-supplied observations. |

For `requirements_hcl/event_model_link/1`, `data` has exactly these fields:

| Field | Type | Contract |
|---|---|---|
| `model` | `string` | Identifier of a caller-supplied external behavior model. |
| `revision` | `string` | Exact external revision; no latest-revision substitution. |
| `target_kind` | Enum `slice` | External target block type. |
| `target_label` | `string` | External label interpreted by this owner, not the core label grammar. |

For `requirements_hcl/html_view/1`, `data` has exactly these fields:

| Field | Type | Contract |
|---|---|---|
| `layout` | Enum `requirements_first`, `matrices_first` | Presentation order without changing findings. |
| `show_source_locations` | `bool`, exactly `true` | Source locations cannot be suppressed in this contract. |

At most one attachment using `requirements_hcl/html_view/1` may target a design. Binding and behavior payload strings are not paths to read or commands to execute. They do not establish observations, deployed realization, target existence, or FR satisfaction. Missing caller inputs remain qualified unknown results under the owner and assessment contracts.

A future owner contract must publish its own finite field inventory, types, cardinalities, and value constraints. `data` is not an arbitrary-HCL escape hatch, and unknown owner schemas must never be treated as valid opaque success.
