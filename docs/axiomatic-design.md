# adhcl v0.1.0 axiomatic design

This document helps maintainers design the Go tool before implementation. It defines the responsibilities and contracts for `version` and `validate`. The companion [implementation plan](plans/implementation-plan.md) defines the work and acceptance runs.

## Scope and evidence

The user selected design and planning only for this task. No executable or Go implementation is claimed. The release target is v0.1.0.

The tool serves design authors, reviewers, and continuous integration callers. It reads RequirementsHCL models and reports invalid facts, incomplete arguments, and failed assertions. It does not prove that a described system meets its requirements.

The language authority is the sibling [spec repository](../../spec/README.md), inspected at revision `3eb0baf79490aab8a1129adc4d58c330ecd0cbd8`. These documents define the baseline:

- [Language reference](../../spec/docs/language.md) defines syntax, inputs, blocks, and operation results.
- [Assessment contract](../../spec/docs/assessment-contract.md) defines assessment rules and takes precedence when assessment rules differ.
- [November review](../../spec/docs/review-2026-11-11.md) identifies gaps that remain relevant to implementation.
- [Self-design example](../../spec/examples/requirements_hcl.ad.hcl) supplies an integration case, not a proof or a complete schema.

The schema closure after that baseline adds the [field schema](../../spec/docs/schema.md), [diagnostic contract](../../spec/docs/diagnostics.md), and [conformance cases](../../spec/conformance/core-1/README.md).
Use those contracts with the updated assessment contract before pinning an implementation revision.
The fixtures state expected behavior, not completed Go conformance runs.

Some links inside the spec point to absent files. In particular, the extension RFC and several historical examples are absent. The plan must not depend on their availability.

The current `cmd`, `internal`, `examples`, `rfcs`, and `testdata` directories are empty. There is no existing runtime mapping to reconstruct. The Makefile and release configuration are inherited build scaffolding, not observed tool behavior.

The scaffolding still names `fmt`, profiles named `valid` and `strict`, and WebAssembly commands. The repository remote, Go module, and release destination identify `github.com/axiomatic-design-hcl/adhcl`. The remaining command conflicts require a clean cutover during implementation.

## Needs, constraints, and assumptions

A functional requirement (FR) states a required outcome. A design parameter (DP) specifies a mechanism that controls it. This design uses architectural mechanisms as DPs, rather than limiting DPs to input data in the 1991 paper.

The needs retain stable identifiers:

| ID | Stakeholder outcome | Reason |
|---|---|---|
| N1 | A caller identifies the executable release. | Reports and automation need an exact tool identity. |
| N2 | An author detects invalid RequirementsHCL facts. | Native HCL syntax alone does not establish language validity. |
| N3 | A reviewer receives qualified AD findings. | Unknown effects and missing support must not become independence claims. |
| N4 | A caller receives usable locations and reliable status. | A failed operation must fail automation and identify its cause. |

The fixed constraints are Go, native HCL, `.ad.hcl` core files, release v0.1.0, and exactly two subcommands. Help flags are interface support, not additional subcommands. Validation is read-only and offline. It does not evaluate expressions, execute modeled flows, install owners, or inspect deployed systems.

The following targets are design proposals, not user-supplied measurements. The operating range includes single files, immediate directory members, manifests, malformed input, incomplete designs, candidates, contexts, and unavailable external evidence. No file-size, latency, memory, or availability guarantee is claimed. Process exhaustion remains an operational risk.

Assumption A1 fixes the language and stage contracts during a change analysis. Assumption A2 prevents consumers from changing upstream facts. Assumption A3 gives each invocation its own state, with no persistent assessment cache. All three require implementation evidence.

## Command contract

The interface below is proposed for implementation. It is not a description of an installed command. `PATH` is one core file, core directory, or explicit manifest.

| Invocation | Behavior |
|---|---|
| `adhcl version` | Write `adhcl VERSION` and one newline to standard output. Read no model files. |
| `adhcl validate [--profile source\|architecture] [--candidate REF] [--context REF] PATH` | Assess one input and write a qualified text report. |
| `adhcl --help` or `adhcl -h` | Describe the two commands and exit successfully. |
| `adhcl validate --help` or `adhcl version --help` | Describe that command and exit successfully without model access. |

`VERSION` is `v0.1.0` in the release build. An untagged build reports `dev`. Preserve the existing linker injection point `main.buildVersion`. Do not derive the version from the working directory or a network service.

The default profile is `source`. `--candidate` accepts a complete `decision.NAME.candidate.NAME` identifier. `--context` accepts `context.NAME`. These argument strings select typed model references. They do not change the rule that authored reference fields require HCL traversals.

Flags precede `PATH`. `--` ends flag parsing and permits paths that start with `-`. Reject unknown flags, repeated flags, extra arguments, missing input, unknown subcommands, and invalid profile names. There is no implicit directory input, standard-input mode, recursive scan, JSON mode, formatting command, or compatibility alias.

Use these exit codes:

| Code | Meaning |
|---|---|
| 0 | The requested operation completes without error findings. Warnings and qualified unknowns can remain. |
| 1 | Input access, parsing, assessment, required attachment processing, or output fails. |
| 2 | Command usage is invalid. |

Write validation reports and findings to standard output. Write usage failures and failures to deliver a report to standard error. Do not print a success summary after an output failure. Exit 0 means that validation completed under its profile, not that the modeled design is independent or implemented.

Each validation report identifies the input, tool version, core contract, selected candidate, context, workload, and profile. It lists findings in stable path, range, rule-code, and subject order. Each finding carries severity, rule code, subject, message, and available source ranges. Related records retain their own locations.

The summary separates core assessment, each required owner, composition, and text rendering. It states error and warning counts and unresolved assessment status. An unsupported conclusion stays unresolved even when the source profile exits 0.

## Top-level traceability and matrix

FR0 is the parent outcome: provide a traceable command result. DP0 selects explicit command routing with separate version and validation paths. The following sibling requirements arise within DP0.

| FR | Parent | Need | Proposed acceptance criterion | Paired DP and responsibility |
|---|---|---|---|---|
| FR1 | DP0 | N1 | `version` reports the injected release or `dev`, without accessing models. | DP1: A version value and dedicated version handler. |
| FR2 | DP0 | N2,N3,N4 | `validate` reports every applicable finding, qualification, and operation status under the selected profile. | DP2: A staged validation operation with a read-only result contract. |

The matrix models material effects on acceptance criteria. Rows are FRs and columns are DPs. `X` denotes material influence, `0` denotes justified absence, and `?` denotes unknown influence.

| FR / DP | DP1 | DP2 |
|---|---|---|
| FR1 | X | 0 |
| FR2 | X | X |

This proposed mapping is decoupled, not diagonal. DP1 affects FR2 because every report must identify its tool version. A malformed or absent version can violate report traceability. DP2 cannot change FR1 because the version path does not load or assess a model.

Establish DP1 before DP2. After a version change, regenerate report expectations that include the version. Hold the command grammar and version contract fixed when changing validation policy. A common dispatcher, linker, process, or output failure remains outside the zero's semantic-change range.

## Validation decomposition

Selecting DP2 creates the following sibling outcomes. Each child contributes to FR2 without claiming the complete parent outcome alone.

| FR | Parent | Needs | Proposed acceptance criterion | Paired DP |
|---|---|---|---|---|
| FR2.1 | DP2 | N2 | Read exactly the selected files and retain accepted values, identities, and ranges. Reject malformed structure before dependent analysis. | DP2.1: Input selection, syntax admission, schema decoding, and typed graph resolution. |
| FR2.2 | DP2 | N3 | Select the exact candidate and conditions without deleting alternatives or inserting missing ancestors. | DP2.2: An invocation-local snapshot selector. |
| FR2.3 | DP2 | N3 | Derive findings without losing exact facts, unknowns, conflicts, support limits, or refinement obligations. | DP2.3: One core assessment engine. |
| FR2.4 | DP2 | N2,N3 | Assess required attachment envelopes and owner payloads without altering core conclusions. | DP2.4: Fixed local owners with separate payload and result authority. |
| FR2.5 | DP2 | N4 | Deliver all independent findings and correct profile status, including partial failure. | DP2.5: A result composer and read-only text presenter. |

The proposed semantic-change matrix uses fixed contracts and one invocation:

| FR / DP | DP2.1 | DP2.2 | DP2.3 | DP2.4 | DP2.5 |
|---|---|---|---|---|---|
| FR2.1 | X | 0 | 0 | 0 | 0 |
| FR2.2 | X | X | 0 | 0 | 0 |
| FR2.3 | X | X | X | 0 | 0 |
| FR2.4 | X | X | 0 | X | 0 |
| FR2.5 | X | X | X | X | X |

The causal effects are not merely calls between stages. Admission determines which identities reach every later outcome. Selection changes applicable influences and attachments. Core policy changes required core findings. Owner policy changes required attachment findings. All of these can affect faithful delivery.

The zeros depend on authority limits. Downstream stages cannot change syntax, schema, or graph facts. Core assessment cannot change snapshot membership. Owners cannot supply core matrix cells or alter core policy. The presenter cannot alter any completed result.

DP2.3 has no material effect on FR2.4 within this range because owners receive the graph and snapshot, not a rewritten core interpretation. DP2.4 has no material effect on FR2.3 because core conclusions exclude binding and external behavior outcomes. Composition reports their separate consequences.

The design order is DP2.1, DP2.2, then DP2.3 and DP2.4, then DP2.5. The two middle controls have no required order between them. An upstream contract change requires renewed downstream assessment. A snapshot change requires new core, attachment, and composed results.

The mapping is conditionally decoupled. A1 and A2 remain untested, so supported independence is unresolved. Shared memory exhaustion, file changes during a read, and process termination can affect several outcomes. No separate-process isolation is claimed.

Within DP2.1, admission and meaning form one further sibling set:

| FR / DP | DP2.1.1: File and syntax admission | DP2.1.2: Schema and graph resolution |
|---|---|---|
| FR2.1.1: Preserve selected source and reject forbidden syntax | X | 0 |
| FR2.1.2: Preserve valid entities and typed links | X | X |

Both children have parent DP2.1 and originate in N2. The first criterion requires exact input membership and positioned syntax findings. The second requires correct fields, namespaces, references, and hierarchy without silent replacement. Establish DP2.1.1 before DP2.1.2. A schema change that expands admitted expressions invalidates the zero and requires a new admission decision.

## Logical module contracts

The modules are responsibilities, not separate services. State belongs to one invocation and becomes read-only at each accepted stage. Each module returns findings instead of terminating the process. Only the executable entry point owns process exit.

| Module / leaf FR | Inputs and outputs | Owned state and dependencies | Invariants and failure contract |
|---|---|---|---|
| Version / FR1 | Build version to one output line. | Build identity only. No model dependency. | No file discovery or assessment. Output failure produces code 1. |
| Admission / FR2.1.1 | Path to selected source units and native syntax trees. | Source bytes, file identities, and ranges. Depends on read-only file access and native HCL parsing. | No recursive or implicit input discovery. Retain independent parse diagnostics. Never execute input. |
| Resolution / FR2.1.2 | Syntax trees to typed core graph, manifest declarations, and attachment envelopes. | Namespaces, reference paths, and graph links. Depends on the pinned schema. | Reject duplicates and wrong-kind references. Invalid core structure blocks dependent assessment, not diagnostic collection. |
| Selection / FR2.2 | Graph plus requested candidate and context to a snapshot. | Active membership and applicability metadata. Depends on valid hierarchy. | Preserve inactive alternatives. Invalid selection stops dependent assessment. |
| Core assessment / FR2.3 | Graph and snapshot to qualified core findings and derived results. | Resolved cells, pairings, orders, coverage, assertions, and evidence estimates. | One cell model feeds every finding. No claim of argument truth from citations alone. |
| Attachment assessment / FR2.4 | Envelopes, graph, snapshot, and local owner registry to separate owner results. | Owner-local decoded payload and result identities. | No payload sharing or input mutation. One owner failure cannot erase independent core or owner results. |
| Delivery / FR2.5 | Completed stage results and profile to composed report and exit status. | Finding order, severity projection, and output state. | Preserve unknowns and causes. Output failure cannot return success. |

Modules return blocked status for work whose prerequisites fail. They must not run analysis on guessed references or incomplete typed graphs. They can continue independent work when its prerequisites remain valid. A source finding uses its actual file and range. A path-access failure names the path without inventing a line number.

### Input and schema contract

A file input selects exactly that `.ad.hcl` file. A directory selects immediate `*.ad.hcl` members, sorted for display, then merges them. A directory with no core members fails. Neither directory selection nor a core file loads attachments or a manifest implicitly.

A manifest selects only its listed core and attachment files. Resolve symbolic links before identity and containment tests. Reject absolute paths, escapes, duplicate physical files, missing members, wrong suffixes, and directories in file lists. The merged core contains exactly one design root.

Admission accepts only the expression forms in the language reference. Inspect native syntax nodes before evaluating literal values. Reject computed keys, duplicate object keys, interpolation, directives, functions, indexing, splats, and operators other than a minus on a numeric literal. Preserve full static reference paths and source ranges.

The [field schema](../../spec/docs/schema.md) now defines types, null rules, reference cardinalities, metadata, and nested namespaces.
The [diagnostic contract](../../spec/docs/diagnostics.md) defines profile severity and extension outcome status.
Pin the revision that contains these contracts before implementing the decoder.
Do not infer a field type merely because one example uses it.

### Core assessment contract

Implement the assessment contract as the authority for snapshots, applicability, conflicts, pairing, classification, hierarchy, feasibility, and evidence. Preserve exact unknowns over defaults. Never expand parent effects into invented leaf facts. Retain cross-branch effects and state-ownership findings.

A proposed pairing is not a material diagonal. Use a complete pairing only when it is unique. Preserve rectangular axes and report missing or redundant controls. Reorder paired rows and columns together. Report a known cycle even when unrelated unknowns prevent complete classification.

The assessment contract takes precedence over shorter language summaries. In particular, unknown cells prevent complete classification despite a known cycle. An apparent ancestor disagreement remains a refinement issue unless an applicable argument establishes contradiction.

Evaluate expectations against the requested basis and complete valid order. Do not replace the author's order with a single preferred order. The source profile retains support gaps as warnings. The architecture profile requires operational criteria, complete module contracts, leaf scenarios, and safe residual feedback contracts.

Evidence estimates require compatible joint trials and the specified Wilson interval. Keep feasibility unresolved without applicable success evidence. Do not pool blocks automatically or infer probability from architectural simplicity. Validation can assess these rules without exposing matrix or evidence subcommands.

Core version 1 defines no trusted machine-readable review importer.
Qualitative supported conclusions therefore remain unresolved for this offline interface.
Report `support_gap` rather than accepting a citation or premise status as substantive approval.
This does not prevent conditional structural findings or qualified empirical estimates.

### Attachment contract

Support explicit manifests and all three initial payload contracts for validation. Use a fixed registry of built-in owners for `requirements_hcl/terraform_binding/1`, `requirements_hcl/event_model_link/1`, and `requirements_hcl/html_view/1`. Do not introduce dynamic loading or owner installation in v0.1.0.

The binding owner accepts exactly `address` and `observation_source` strings for a PV subject. The behavior owner accepts exactly `model`, `revision`, `target_kind`, and `target_label` strings for a module subject. `target_kind` must be `slice`. The view owner accepts the specified layout and `show_source_locations = true` for a design subject.

Require the declared role, exact contract, and available registry label. Validate every declared owner even when it has no attachment. Retain duplicate core binding declarations as errors. Reject duplicate view attachments and duplicate result identities.

No external observation input or behavior resolver exists in the proposed v0.1.0 interface. Binding and behavior owners therefore report `unknown` with an unavailable-input cause. They must still fully assess payloads and subjects. Do not return fabricated matches or resolved targets.

The view owner assesses its payload but does not render HTML during `validate`. HTML delivery is a separate operation outside this release. This boundary preserves validation of the initial view contract without adding another command.

Unknown external evidence alone is not an operation failure. Invalid required attachments and failed owners are failures. Report composition separately from implementation conformance. A successful validation result does not establish a binding match or behavioral correspondence.

## Composition and change control

An S junction combines independently achieved outcomes. A C junction enforces a required design sequence. An F junction exposes coupled iteration. These meanings adapt the paper's integration model without requiring a centralized runtime service.

DP0 uses C for identity before qualified validation delivery. DP2 uses C for admission, selection, and delivery. It uses S to combine core and attachment findings after their shared prerequisites. DP2.1 uses C for source admission before graph resolution.

Runtime dependencies require source admission before resolution, resolution before selection, and selection before active assessment. Delivery follows available results. Core and owner work can run sequentially in one process. No runtime concurrency or feedback loop is required.

After an upstream change, discard dependent invocation results and recompute them. Do not add a persistent cache in this release. A source read failure cannot erase independent readable-file findings. A failed attachment cannot suppress completed core findings.

The semantic matrices do not establish operational isolation. Shared output or memory failure affects report delivery, and schema changes can affect every branch. Retain these limits rather than relabeling the whole tool uncoupled.

## Alternatives and Information Axiom

Compare mechanisms against the same command contracts, input population, core semantics, and v0.1.0 horizon:

| Choice | Alternative | Decision and risk |
|---|---|---|
| Input parser | Handwritten HCL subset | Use the existing native HCL dependency. A subset parser risks rejecting native numbers and heredocs. |
| Command handling | A command framework | Use standard Go argument parsing with explicit dispatch. Two commands do not require a new dependency. |
| Validation structure | One mutable model shared by all phases | Use staged ownership. Shared mutation can let rendering or owner rules change core facts. |
| Owner support | External plugins and observation discovery | Use fixed built-in payload owners. Preserve external outcomes as unknown without adding execution or discovery. |
| Independence claim | Force every stage into a diagonal matrix | Retain the causal triangular mapping. Downstream acceptance depends on upstream meaning. |

These are provisional risk-based choices. No benchmark or probability model establishes their joint success rate. Information content is `I = -log2(p)`, where `p` measures success of all required outcomes together. No numerical score is justified.

Compare alternatives with the same conformance fixtures, CLI failure cases, and build targets. Record failures, workload, and revisions. Functional independence does not prove statistical independence. A finite passing run does not establish `p = 1`.

## Validation scenarios and evidence limits

These scenarios test the claimed influence boundaries and user-visible outcomes:

| ID | Change or operating case | Required observation |
|---|---|---|
| V1 | Change the injected version while a model path is unreadable. | Version output changes and still succeeds without model access. Validation reports retain the new identity. |
| V2 | Switch file order and split a valid model across immediate directory members. | Semantic findings remain equal. Every finding retains its actual source. |
| V3 | Supply a wrong-kind reference or forbidden expression. | A positioned error appears. Dependent analysis is blocked without panic or guessed data. |
| V4 | Select another candidate or invalidate a premise. | Active findings change. Inactive alternatives remain stored and dependent support is withdrawn. |
| V5 | Replace an absent effect with an exact unknown beside a default zero. | The cell stays unknown. No supported diagonal result appears. |
| V6 | Supply rectangular controls, ambiguous pairing, a permuted triangle, or a cycle with unknowns. | Axes and labels remain intact. Pairing, order, cycle, and unresolved status follow the contract. |
| V7 | Add an invalid required attachment to a valid core model. | Core findings remain available. Owner and composition fail, and the command exits 1. |
| V8 | Change a valid owner payload or view layout under fixed contracts. | Core facts and conclusions remain unchanged. Only the relevant owner result and faithful report change. |
| V9 | Run both profiles on incomplete module contracts. | Qualifications retain the same truth. Severity and operation success follow the selected profile. |
| V10 | Supply incompatible evidence or all-success trials. | Comparison stays unresolved when incompatible. Wilson uncertainty remains with all-success samples. |
| V11 | Fail report output or provide invalid command usage. | Output failure exits 1. Usage failure exits 2 without model access. |
| V12 | Make a manifest member escape through a symbolic link. | Input fails containment without reading the escaped model. |

Each in-scope need maps to FR1 or FR2 and its leaves. The matrices, contracts, and scenarios describe proposed controls. They do not supply implementation evidence. The schema gaps, untested ownership limits, and shared-process risks remain explicit release concerns.
