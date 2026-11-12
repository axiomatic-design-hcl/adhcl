# RequirementsHCL language reference

This reference is for authors and tool integrators who describe software with Axiomatic Design.
It defines the language contracts, authored facts, derived results, and required diagnostics.
It does not define the runtime behavior of the software that a design describes.

This specification defines the attachment contracts in this reference.
The planned implementation uses Go and `github.com/hashicorp/hcl/v2`.
The earlier Python analyzer is available in Git history at commit `9fef41f`.
It is not a reference implementation or a compatibility constraint.
Specification acceptance does not mean that the Go tool or extension owners exist.

## Method and scope

A functional requirement (FR) states an outcome.
A design parameter (DP) controls an outcome.
A process variable (PV) realizes a design parameter.
Needs explain why the outcomes matter.
Constraints limit the acceptable solutions.

The Independence Axiom concerns changes to outcomes, not calls between routines.
A material influence changes whether an FR meets its acceptance range.
A call or dependency alone does not establish that influence.
Authors must state the relevant operating range and causal mechanism.

A matrix has FR rows and DP columns.
`X` means a stated material effect, `0` means a justified absence of that effect, and `?` means unknown.
A proposed control pairing does not prove that the control works.
Square classification requires a meaningful pairing and a material diagonal.
Rectangular mappings retain every distinct FR and DP and require a separate control assessment.

The reports use `P` for a proposed pairing without a supported material effect.
An exclamation mark identifies a conflict.
The `?r` symbol identifies an unknown cell with a parent refinement obligation.
An `X!` cell retains its exact material effect and exposes a conflicting ancestor claim.

An uncoupled matrix is diagonal.
A decoupled matrix can become triangular through a meaningful order.
A coupled matrix contains a material cycle between paired controls.
A known cycle remains visible even if other cells remain unknown.
Unknown cells prevent complete classification, including when a known cycle is present.

Zigzag decomposition defines child FRs inside a chosen parent DP.
Sibling results do not establish independence across the whole design.
Shared state, failure paths, and resource limits can create effects across branches.
The tool must retain these effects and unknowns.

The Information Axiom compares feasible designs under the same acceptance ranges, workload, and time horizon.
Information content is `I = -log2(p)`.
Here, `p` is the probability that all required outcomes succeed together.
A diagonal influence matrix does not prove that success events are statistically independent.

## Authored facts and derived results

Authors write each fact once.
The tool derives matrices and reports from the selected facts.
Assertions test results but do not supply missing evidence.

| Authored facts | Derived results |
|---|---|
| Needs, constraints, assumptions, and contexts | Applicable premises and unresolved conditions |
| FRs, criteria, parent links, and proposed controls | Sibling sets and control assessments |
| Influences with reasons and sources | Matrices, conflicts, and refinement obligations |
| Decision history and candidate membership | The selected design snapshot |
| Modules, PVs, flows, and scenarios | Coverage gaps and change obligations |
| `expect` blocks | Assertion results |
| Qualified evidence | Probability estimates and information bounds, when supported |

A snapshot is the selected design and its operating context.
Each report must identify its snapshot and distinguish conditional structure from a supported conclusion.
A report must not erase an exact relation because an ancestor relation differs.
An invalidated premise withdraws the conclusion that depends on it.

## Model files and layout

A core model is one `.ad.hcl` file or one directory of core files.
An attachment is tool data linked to a core subject.
A project combines one core model with explicitly listed attachments and extension owners.
A manifest is the file that declares those project inputs.

A tool accepts a core file, a core directory, or an explicit `.ad.project.hcl` manifest.
For a directory input, it merges the immediate `*.ad.hcl` files before structural analysis.
It does not scan subdirectories or load a manifest implicitly.
A core-only input does not discover attachments or load extension owners.

For a manifest input, the tool reads only the listed core and attachment files.
The suffix `.ad.ext.hcl` identifies attachment files and does not match the core directory scan.
The suffix `.ad.project.hcl` identifies manifests and also does not match that scan.
One project contains exactly one design root across its merged core files.

Each block label is unique within its namespace across all files of the project.
A core reference can point to a core block in any listed core file.
Core blocks cannot refer to attachments or manifest extension declarations.
An attachment refers to its owner and its core subject, never to another attachment.

File names and file order do not change analysis results.
An implementation can use them to set display order.
Each diagnostic and cited location identifies the file and source range.

The language does not require a layout.
The layout follows the three mappings of the method: needs to FRs, FRs to DPs, and DPs to PVs.
Each link is written on the downstream side.
An FR refers to its need.
A DP refers to the FR that it controls.
A PV or module refers to the DP that it realizes.
A needs file therefore depends on no other file, and a process file depends only on DPs.

Do not put FRs and DPs in separate files.
A child FR refers to its parent DP, and that DP refers to the FRs that it controls.
One zigzag step chooses a DP, defines its child FRs, and states their influences.
Split the design by branch instead, so that one file holds one zigzag unit.

The analyzer derives every matrix.
Do not write a matrix file.
Keep each `influence` block with the FRs and DPs that it explains.
Put an influence that crosses branches in the file of the nearest common parent.

The recommended layout uses these files:

| File | Contents |
|---|---|
| `needs.ad.hcl` | Stakeholders, needs, and constraints |
| `context.ad.hcl` | Assumptions, contexts, and workloads that all domains use |
| `design.ad.hcl` | Sources, the design root, and the top-level FRs, DPs, influences, and decisions |
| `design.BRANCH.ad.hcl` | The FRs, DPs, influences, decisions, and assertions of one branch |
| `process.ad.hcl` | PVs, modules, flows, and scenarios |
| `model.ad.project.hcl` | The core file list, attachment file list, and extension owners |
| `bindings.ad.ext.hcl` | Tool-specific realization bindings |
| `behavior.ad.ext.hcl` | References to external behavior models |
| `views.ad.ext.hcl` | Presentation configuration |

Replace `BRANCH` with the label of the branch, for example `design.search.ad.hcl`.
The starter in `examples/starter/` shows an explicit manifest and attachments.
A small model can stay in one file.

## Syntax profile and references

RequirementsHCL uses native HCL syntax with a restricted expression profile.
A profile defines the expressions that the language accepts.
The planned Go implementation uses `github.com/hashicorp/hcl/v2/hclsyntax` for native parsing.
Parsing a valid HCL expression does not authorize its use in a RequirementsHCL field.

The accepted forms are:

- Literal strings, numbers, Boolean values, and `null` where the field permits it.
- Literal lists and objects whose values satisfy the field contract.
- Static references such as `dp.book_content` and `fr.build_db.criterion.retention` in reference fields.
- Labeled blocks, unlabeled `influence` blocks, and the nested blocks defined by this specification.
- Native HCL comments, quoted strings, and heredoc strings without template evaluation.

Numeric literals follow native HCL syntax, including exponent notation.
A negative numeric literal can use a leading minus sign.
String escapes and heredoc indentation follow the [native HCL specification](https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md).
The language does not inherit the Python experiment's numeric, string, or heredoc limits.

Computed expressions are not part of this profile.
The tool rejects function calls, arithmetic, comparisons, conditionals, comprehensions, indexing, and splats.
The leading minus sign on a numeric literal is the only permitted operator.
Object keys must be literal identifiers or quoted strings, not computed expressions.
String interpolation and template directives are errors, including those inside heredoc strings.
Native HCL escapes can represent literal `${` and `%{` text without interpolation.

Block labels use quoted strings and follow each block's label count.
Named blocks have one label.
The `influence`, `information`, and nested `loop` blocks have no labels.
Duplicate attributes, object keys, and labels within one namespace are errors.
Unknown blocks, fields, or enum values are errors rather than ignored input.

A reference is an absolute static traversal from a declared namespace.
For example, `fr.build_db.criterion.retention` names a criterion inside `fr.build_db`.
Reference fields require traversals, not strings that resemble references.
Reference resolution preserves the namespace, complete path, and source range.
The tool diagnoses a missing target or incorrect target type before dependent analysis.

Literal data cannot read environment variables, execute commands, access files, or download an owner.
An attachment's `data` object contains only literal values, including literal collections.
External identifiers inside that object are strings interpreted by the declared owner, not core references.

Structural processing follows this order:

1. Parse the selected input files and retain source ranges.
2. Decode the manifest, core blocks, and attachment envelopes.
3. Resolve typed core references and the design hierarchy.
4. Resolve attachment owners and subjects against the valid core graph.
5. Select the candidate and operating context.
6. Derive core conclusions and obtain applicable extension results.
7. Compose qualified results and render the requested report.

Invalid core structure stops analyses that depend on it.
An invalid attachment does not erase independent core findings or another owner's completed results.
The tool retains all diagnostics and does not present partial results as complete success.
Every diagnostic identifies its source location when source input exists.

## Project manifests and extension owners

A manifest contains one `project` block and zero or more `extension` blocks.
No other top-level blocks or attributes belong in a manifest.
The project label identifies the project and does not replace the core design label.
The following fields define the manifest:

| Block | Required fields | Contract |
|---|---|---|
| `project "label"` | `language`, `core_files`, `attachment_files` | Selects one core contract and explicit input files. |
| `extension "label"` | `role`, `contract` | Selects an installed owner and its exact contract. |

`language` must be the string `requirements_hcl/core/1`.
`core_files` is a nonempty list of paths to `.ad.hcl` files.
`attachment_files` is a list of paths to `.ad.ext.hcl` files and can be empty.
The tool merges all listed core files into one core model.
Attachment files contain only `attachment` blocks.

Paths are literal strings relative to the manifest directory.
Absolute paths, duplicate files, and paths that leave that directory are errors.
The tool resolves symbolic links before it determines file identity and containment.
A missing file, a directory in either list, or an incorrect suffix is an error.
The tool does not expand globs, load nested manifests, or infer additional files.

An extension owner is an installed implementation of one versioned contract.
The extension label selects that owner from a trusted local registry.
The registry is a list of owners available to the implementation.
Registry installation and packaging are outside the authored language.

`role` is one of `binding`, `behavior_link`, or `view`.
`contract` is a literal string that selects the exact contract version supported by that owner.
An unavailable owner, an incompatible version, or an incorrect role produces an extension diagnostic.
Every declared owner is required, even if no attachment refers to it.

The manifest cannot install an owner, execute a command, or download software.
An owner declaration does not add core blocks, reference namespaces, or expression functions.
Core-only inputs select the same `requirements_hcl/core/1` semantics without a manifest.
This manifest declares a project with three extension roles:

```hcl
project "catalog" {
  language = "requirements_hcl/core/1"
  core_files = ["design.ad.hcl"]
  attachment_files = [
    "bindings.ad.ext.hcl",
    "behavior.ad.ext.hcl",
    "views.ad.ext.hcl",
  ]
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
```

The contract names in this specification identify required behavior, not available packages.
The manifest requires all three named owners to run as a complete project.
Its core file remains independently usable as a core-only input.

## Attachments and role contracts

An `attachment "label"` block requires exactly `owner`, `subject`, and `data`.
`owner` is a static reference to a manifest declaration, such as `extension.terraform_binding`.
`subject` is a static reference to a core block of the type required by the owner's role.
`data` is a literal object interpreted only under that owner's contract.

Attachment labels are unique across the project's attachment files.
An undeclared owner, missing subject, or incorrect subject type is an error before owner invocation.
A subject must belong to the core model of that project.
The loader must not convert an incorrect reference into a string.

The owner rejects unknown payload fields, missing required values, incorrect types, and unsupported enum values.
No automatic string conversion or silent field removal is permitted.
The field contracts below do not permit `null`.
Each diagnostic identifies the attachment, field, file, and source range where applicable.

The initial role contracts have these subject types:

| Role | Subject | Responsibility |
|---|---|---|
| `binding` | `pv` | Compare a declared realization binding with caller-supplied observations. |
| `behavior_link` | `module` | Resolve an external behavior target at its exact revision. |
| `view` | `design` | Present a qualified assessment without changing its findings. |

Core facts remain authoritative for requirements, controls, influences, candidates, criteria, premises, and evidence qualification.
An attachment cannot override those facts or supply missing matrix cells.
An owner cannot read another owner's payloads or mutate its inputs, rules, or results.
These restrictions do not prohibit a view from reading the composed assessment results.

### Realization bindings

The contract `requirements_hcl/terraform_binding/1` requires exactly two string fields in `data`.
`address` identifies the Terraform resource to compare.
`observation_source` identifies a caller-supplied observation set.
Neither string is a file path to read, a command to execute, or proof of realization.

The following attachment assumes that the core declares `pv.catalog_database`.
It moves the tool-specific address into the binding owner's contract.
The core PV retains `realizes` and any generic descriptive fields:

```hcl
attachment "catalog_storage" {
  owner = extension.terraform_binding
  subject = pv.catalog_database
  data = {
    address = "aws_db_instance.catalog",
    observation_source = "terraform_state",
  }
}
```

The owner reports `match`, `mismatch`, or `unknown`.
An available observation set with no resource at the declared address produces `mismatch`.
Unavailable observations produce `unknown` with their cause.
A `match` establishes only this binding comparison, not verified behavior or satisfaction of the FRs.

The generic core PV fields `kind`, `address`, and `note` remain valid descriptive facts.
They do not select an owner or start an observation.
If a tool-specific value moves into an attachment, it must not remain duplicated in the core PV.
A duplicate declaration of that value is an error, even when both values agree.

### External behavior links

The contract `requirements_hcl/event_model_link/1` requires exactly four string fields in `data`.
`model` identifies a caller-supplied behavior model, and `revision` identifies its exact revision.
`target_kind` identifies the external block type, and `target_label` identifies its label.
For this contract version, `target_kind` must be `slice`.

The following attachment assumes that the core declares `module.catalog_search`.
The owner resolves the external strings without adding its schema to the core.
The attachment does not duplicate the module's realization or hierarchy:

```hcl
attachment "catalog_search_behavior" {
  owner = extension.event_model_link
  subject = module.catalog_search
  data = {
    model = "library_behavior",
    revision = "library-behavior-r17",
    target_kind = "slice",
    target_label = "search_catalog",
  }
}
```

The owner reports `resolved`, `missing`, or `unknown`.
An available model at the requested revision with no matching target produces `missing`.
An unavailable model or unavailable exact revision produces `unknown`.
The owner must not substitute a different revision or infer success from the identifier alone.

A resolved link establishes target identity, not correspondence between behavior and FR acceptance criteria.
That correspondence requires separate evidence under the core evidence rules.
Behavior remains in external models, and the core does not embed their schemas.
Existing external `import` and `module.handles` declarations must move to behavior attachments without compatibility aliases.

### Presentation

The contract `requirements_hcl/html_view/1` requires exactly `layout` and `show_source_locations` in `data`.
`layout` is the string `requirements_first` or `matrices_first`.
`show_source_locations` is the Boolean value `true` in this contract version.
A project can declare at most one view attachment for a design under this contract.

The following attachment assumes that the core declares `design.catalog`.
It changes presentation rather than the selected facts or assessment rules.
Both permitted layouts retain every finding and qualification:

```hcl
attachment "review_layout" {
  owner = extension.html_view
  subject = design.catalog
  data = {
    layout = "requirements_first",
    show_source_locations = true,
  }
}
```

The view receives the composed assessment as read-only input.
It cannot change cells, evidence status, conditions, candidate selection, or process status.
A rendering failure produces a diagnostic and nonzero command status without changing the completed assessment.
Without a view attachment, a renderer can use its default layout but must preserve the same findings.

## Extension results and composition

An extension result is derived output, not another authored core fact.
The tool retains its subject, owner, contract, input revisions, status, cause, and source locations.
The binding and behavior contracts define their status values independently.
Invalid attachments and failed owners produce diagnostics, not fabricated `match` or `resolved` results.

Each result identifies the exact inputs that produced it.
These include the core revision, selected candidate and context, attachment revision, owner implementation and contract, and relevant observations or external revisions.
A changed input invalidates dependent results.
The tool must not reuse a stale result as support for the current assessment.

The tool derives result identities rather than asking authors to copy them.
An identity distinguishes the attachment, subject, owner, and selected input revisions.
Multiple attachments to one subject retain separate results.
If two results claim the same identity, the composer reports a conflict rather than keeping the last result.

A composer combines core findings and extension results into one qualified assessment.
It cannot promote unknown results to success or suppress a failed prerequisite.
An owner failure does not erase another owner's completed results.
Every declared extension and listed attachment is required for complete project processing.

Core independence results and implementation conformance remain separate conclusions.
A core-only analysis does not require a successful binding observation.
An unknown binding result cannot establish implementation conformance.
A behavior link cannot establish an FR's acceptance predicate without qualified evidence.

This design permits independent extension controls within fixed contracts and one core snapshot.
It does not establish that the complete language or tool is uncoupled.
A core identity change can invalidate attachments, and changed findings must change report content.
Shared resources and process failures can affect several owners and require separate influence analysis.

The tool must assess those effects rather than infer zeros from separate files or owner declarations.
A missing relation remains unknown under the core influence rules.
Assertions about uncoupled mappings still require effective controls, justified zeros, and applicable premises.

## Core blocks

Each block label is unique within its namespace.
Labels use snake_case.
Use `alias` to retain IDs from a source, such as `FR11` or `DP121`.
The `cites` field accepts one source reference or a nonempty list of source references.

The [field schema](schema.md) is authoritative for every field's type, cardinality, metadata, namespace, and absence rules.
All authored fields reject `null`.
The tables in this reference summarize the schema rather than defining a second decoder contract.

| Block | Fields and meaning |
|---|---|
| `source "label"` | Required `title`. Optional `doi`, `url`, and `pages`. |
| `design "label"` | Required `boundary`. Optional `level` and `workload`. Exactly one design root is required. |
| `stakeholder "label"` | `role` identifies the affected person or group. |
| `need "label"` | Required `statement`. Optional `why`, `stakeholder`, and `origin`. |
| `constraint "label"` | Required `statement`. `kind` identifies the limit. `origin` identifies its basis. |
| `assumption "label"` | Required `statement`. `status` is `assumed`, `to_validate`, `validated`, or `invalidated`. |
| `context "label"` | `description` names the operating condition. `assumes` lists premises. `excludes` names incompatible contexts. `workload` identifies the workload. |
| `workload "label"` | `description` and `ranges` describe the input population and permitted ranges. |
| `state "label"` | `description` identifies information that a leaf DP owns or reads. |

The `origin` field distinguishes stated facts from inferred facts.
The examples cite source facts and mark added mechanisms with `mechanism_source = "inferred"`.
Do not invent missing thresholds, modules, or historical mechanisms.

An FR uses `statement`, `need`, `parent`, `alias`, `bounded_by`, and nested `criterion` blocks.
A top-level FR refers to its need.
A child FR uses `parent = dp.x` and inherits the need context through that link.
The hierarchy must be acyclic and reachable from the design root.

A criterion defines an acceptance predicate, which is a condition for success.
Its operational form uses `metric`, `op`, `value`, and optional `unit` and `under`.
The operators are `<=`, `<`, `>=`, `>`, `==`, and `in`.
Its `origin` is `user`, `proposed`, or `source`.

A source reconstruction can retain a qualitative `statement` without a complete operational predicate.
Kim gives qualitative criteria for search completeness, database compactness, warpage tolerance, and reinforcement.
The examples retain these criteria without invented numeric limits.
Missing metrics and thresholds remain documented gaps.

The `allocated_from` field links a child criterion to a parent criterion.
A link records an allocation but does not prove that the child outcomes satisfy the parent range.
Children inherit context, not an unchanged copy of every parent outcome criterion.
The allocation needs its own outcome argument.

A DP requires `controls`, `responsibility`, and `decision`.
`controls` is a nonempty list of FR references and records proposed control pairings.
Several controls or several FRs remain distinct in analysis.
Optional fields include `why`, `alias`, `role`, `owns_state`, `reads_state`, `bounded_by`, and `holds_if`.

Only `decision = "selected"` activates a DP in the default snapshot.
`proposed`, `rejected`, and `superseded` retain alternatives without activating them.
An explicit candidate selects its listed DPs for alternative analysis.
The selected candidate and selected lifecycle states must agree for the default design.

## Influence resolution

An unlabeled `influence` requires `from`, `to`, and `effect`.
`from` is a nonempty list of DP references.
`to` is an FR reference.
`effect` is `material`, `none`, or `unknown`.

A material effect requires `mechanism`.
A zero requires `because`.
Optional fields include `mechanism_source`, `source`, `under`, `holds_if`, and `cites`.
The tool retains every relation and its source instead of replacing an earlier relation with the same endpoints.

The `under` field selects a named context or workload.
Legacy prose conditions remain unresolved until the author supplies defined applicability.
Relations in overlapping contexts cannot assert contradictory effects.
Explicitly excluded contexts can contain different models without conflict.
A missing applicable relation remains unknown.

The `holds_if` field lists assumptions that qualify a relation or control.
Assumed and unevaluated premises permit a conditional structural account, not an unconditional independence claim.
Invalidated premises cannot support the dependent relation.
The library synonym result depends on both compactness and preserved search completeness.

A `cells "label"` block supplies a default for unlisted sibling cells.
It uses `scope`, `unlisted`, optional `because`, and `cites`.
A default of `none` requires a reason.
Both the source DP and target FR must belong to that sibling set.
Conflicting defaults are errors.

An exact leaf relation remains visible in the master matrix.
A parent effect describes an aggregate outcome, not every descendant pair.
A parent material effect therefore creates a refinement obligation rather than automatic leaf effects.
A parent zero does not establish descendant zeros without a compositional argument.
An apparent disagreement produces an unresolved refinement obligation.
A contradiction requires an applicable outcome argument that establishes incompatible claims.

The master matrix, dependency matrix, and impact report use the same resolved influence facts.
The dependency matrix distinguishes known zero from unknown.
The impact report follows definite effects and exposes possible parent obligations separately.
A child change can require reconsideration of its parent contract and downstream branches.

## Assertions, decisions, and evidence

An `expect "label"` block uses `scope`, `classification`, and optional `order`.
`scope` names the design root or a parent DP.
An expected order must contain every relevant DP once and respect every required precedence.
Any valid order passes, even if it differs from the stable display order.

The `basis` field selects `source`, `conditional`, or `supported` assertions.
An omitted basis selects `source`.
Every basis requires applicable effects rather than pairings alone.
The [assessment contract](assessment-contract.md#expectation-results) defines assertion results for unknowns, conflicts, and support gaps.

A `coupling "label"` block records residual coupling.
Its required fields are `scope`, `affects`, and `reason`.
Optional fields include `condition`, `decision`, `normal_practice`, `residual_risk`, and `iteration_bound`.
A coupling record does not remove coupling.
The architecture profile also requires `convergence`, `stopping`, and `failure_behavior` for residual feedback.

A `decision "label"` requires `addresses` and contains named `candidate` blocks.
Each candidate uses `dps` and can identify its `version`, `context`, `workload`, and `horizon`.
The decision uses `selected`, `provisional`, and `why`.
A candidate can contain an `information` block with `basis` and evidence references.

Evidence must identify the joint success event, exact candidate version, context, workload, and exposure horizon.
It must also state its sampling assumptions, dependence model, and confidence method.
Counts alone do not identify comparable probabilities.
Probability estimates require qualified empirical evidence, not unsupported typed-in probabilities.

An `evidence "label"` block uses these fields for empirical estimation:

- `basis = "measured"` identifies observed trials.
- `event` names the joint success predicate.
- `scenario` refers to a scenario that covers every operational criterion of the addressed FRs.
- `candidate` refers to `decision.name.candidate.name`.
- `version` identifies the exact candidate revision.
- `context` identifies the operating context.
- `workload` and `horizon` identify the common exposure.
- `dependence = "joint"` states that each trial measures all required outcomes together.
- `sampling` states the assumptions behind the trial sample.
- `trials` and `failures` give the observed counts.
- `confidence_method = "wilson"` selects the Wilson interval for a binomial success rate.
- `confidence` gives the confidence level as a number between zero and one.

The Wilson interval bounds an estimated success rate.
It requires a trial model that supports a binomial interpretation.
The estimate is `(trials - failures) / trials`.
The analyzer must retain the stated sampling assumptions beside the estimate.
The scenario uses `verifies` references to identify the acceptance predicates.
Candidate and evidence metadata must identify the same context, workload, version, and horizon.

Success means that all specified FR acceptance predicates meet their ranges in the same trial.
Do not multiply marginal rates merely because the influence matrix is diagonal.
Zero observed failures do not establish `p = 1`.
Missing or incompatible evidence leaves information content unknown and the selection provisional.

If probability bounds are `[L,U]`, information bounds are `[-log2(U), -log2(L)]`.
The upper information bound is infinite when `L = 0`.
Estimates and bounds must remain distinct from measured outcomes.
The tool must not infer probability from code size, edit counts, or module counts.

## Modules, realization, and runtime flow

A `pv "label"` block requires `realizes` and can use `kind`, `address`, and `note`.
PVs describe code, routines, packages, or infrastructure that realize DPs.
A declaration alone does not prove that an implementation exists or matches the design.
Unavailable observations leave implementation status unknown.

A `module "label"` block represents a logical implementation boundary.
A realization module uses `realizes`.
A composite module uses `contains` to group modules without claiming another realization.
If both `within` and `contains` describe a relationship, they must agree.

One module can realize several FR responsibilities through their DPs.
Two modules cannot silently claim the same realization.
Module contracts use `inputs`, `outputs`, `owns_state`, `dependencies`, `invariants`, `failure_behavior`, and `acceptance_checks`.
Behavior links use attachments owned by a `behavior_link` extension.
The former `module.handles` field and `import` block are not part of this language contract.
The tool rejects them rather than creating an implicit owner or external namespace.

Architecture contracts require explicit `owns_state` and `dependencies`, even when these lists are empty.
Their `acceptance_checks` refer to criteria or scenarios, not prose that resembles a reference.
Each leaf needs a scenario that covers its outcome.

State ownership belongs to leaf controls.
Multiple owners are an error.
Shared reads and writes require influence analysis rather than an automatic zero.
Shared PVs and shared resources require review of possible cross-branch effects.

A `flow "label"` block describes runtime order through `steps`.
A nested list combines independent outcomes without imposing an order.
Other fields include `owner`, `input`, `output`, `after`, and nested `loop` blocks.
A loop uses `after`, `back_to`, `when`, and `bound`.

S junctions combine outcomes without requiring arithmetic addition or concurrent execution.
C junctions record a required design sequence.
They imply runtime order only when module contracts require that order.
F junctions expose residual iteration but do not supply a safe feedback algorithm.
Runtime iteration needs convergence, stopping, and failure behavior.

A `scenario "label"` describes a change, operational, or failure case.
It can use `verifies`, `vary`, `expect_affected`, `expect_unaffected`, `steps`, `description`, and `under`.
Scenarios provide acceptance checks for proposed architectural contracts.
They do not replace observations from a running implementation.
Scenario `steps` can contain authored instructions.
Flow `steps` contain typed module, FR, or DP references instead of instructions.

## Review profiles and command results

The `source` profile preserves gaps in historical reconstructions as warnings.
It does not turn missing criteria or unnamed modules into invented facts.
The `architecture` profile requires operational criteria and complete module contracts.
Both profiles reject malformed structure and contradictory design facts.
The [diagnostic contract](diagnostics.md) assigns stable rule codes and severity for each profile.
Support and feasibility can remain unresolved even when a complete architecture model passes validation.

The language defines report content, input selection, and failure semantics independently of command spelling.
The planned Go tool's executable name, flags, and subcommands are not specified here.
The following names identify report operations, not an implemented Go command line:

| Operation | Result |
|---|---|
| `validate` | Structural, method, profile, assertion, and evidence findings |
| `matrix` | A matrix for each sibling set |
| `master` | A matrix over leaf FRs and distinct leaf DPs |
| `junctions` | Conditional module composition |
| `dsm` | Dependencies between leaf controls |
| `impact` | Definite effects and possible change obligations |
| `evidence` | Joint probability bounds, information bounds, and candidate feasibility limits |
| `html` | A single HTML page that shows the model to customers, product owners, and engineers |
| `all` | Every report except a DP-specific impact report and the HTML page |

Each report identifies the selected input, candidate, context, and review profile.
Reports for one assessment use the same resolved cells, extension results, and qualifications.
An attachment cannot select a different candidate, context, or profile.
The HTML operation produces a self-contained report with no required external assets.
Its layout cannot suppress findings, change cell meanings, or override process status.

A command-line implementation returns zero only when its requested operation succeeds.
Invalid input, invalid required attachments, failed assertions, and rendering failures produce nonzero status.
Missing observations or external revisions produce qualified unknown results, not fabricated success.
Unknown results alone do not establish supported independence or implementation conformance.
Usage errors are distinct from assessment failures.

The earlier Python experiment and its generated HTML are in Git history at commit `9fef41f`.
They do not define the Go interface or demonstrate conformance with this specification.
The [design rationale](design-rationale.md) records the conditional architecture and its evidence limits.
The [November review](review-2026-11-11.md) records the specification gaps that motivated the detailed contracts.

## Normative assessment rules

The [assessment contract](assessment-contract.md) defines version 1 snapshot selection, cell resolution, pairing, hierarchy, support, feasibility, statistics, and identity.
Use these rules for every report and extension composition.
They replace underspecified assessment behavior in earlier drafts.

The [field schema](schema.md) and [diagnostic contract](diagnostics.md) complete the decoding and failure rules.
The [versioned conformance cases](../conformance/core-1/README.md) record expected behavior for future implementations.
Their expected outcomes are not measured validator results.
