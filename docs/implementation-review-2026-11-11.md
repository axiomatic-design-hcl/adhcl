# adhcl implementation review

Date: 2026-11-11.

This review helps maintainers correct the implementation before relying on its assessment results. It covers the implementation plan, current Go source, and RequirementsHCL specification. It does not change implementation files or certify a release.

Do not treat this implementation as complete against the plan. The review found false-success cases in ownership, evidence, selection, and profile assessment. Keep the staged architecture, but correct its contract enforcement and result boundaries.

## Scope and evidence

The primary authorities are [the implementation plan](plans/implementation-plan.md), [the language reference](../../spec/docs/language.md), [the schema](../../spec/docs/schema.md), [the assessment contract](../../spec/docs/assessment-contract.md), and [the diagnostic contract](../../spec/docs/diagnostics.md). The assessment contract takes precedence for assessment rules.

The implementation pins specification revision `01c50bf3d141624c8f541a4797f0224b7658d003`. SHA-256 comparisons show that its four vendored normative documents match the sibling specification files. These findings therefore do not result from a contract revision mismatch.

The review applies Axiomatic Design, Clean Architecture, and Clean Code. A functional requirement (FR) states a required outcome. A design parameter (DP) supplies its controlling mechanism. This review uses architectural mechanisms as DPs, following the skill's adaptation of the 1991 paper.

P1 means correct before trusting results or processing untrusted models. P2 means a contract or design defect that needs correction. Observed findings name an executed scenario. Static findings name source evidence and do not claim runtime proof.

## Findings

### R1. P1: Ambiguous controls trigger factorial enumeration

Locations: `internal/assessment/classify.go:20-23,180-212`, `internal/assessment/types.go:54-55`, and `internal/report/render_sections.go:34-42`.

`hasAlternativePairing` already establishes ambiguity. `Classify` then calls `enumeratePairings`, which stores every complete pairing. The master assessment repeats this work for a flat design.

Observed: a seven-FR, seven-DP model with every DP controlling every FR reports `5040 complete pairings` in both scopes. The required answer is already known after a second pairing exists. For a complete square control graph, the stored pairing count grows as `n!`.

This directly violates implementation-plan line 222, which prohibits exhaustive pairing enumeration. A small authored model can consume disproportionate CPU and memory before the tool reports its ambiguity. No large-input timing or exhaustion test was run.

Retain the matching-based ambiguity decision. Return at most two witness pairings when explanations need examples. Do not compute an exact total merely for display. Preserve the two-pairing conformance example without requiring all pairings for every model.

Acceptance: exercise a complete ten-by-ten control graph. It must report `ambiguous_pairing` without creating all 3,628,800 assignments. A rectangular model must retain all its axes.

### R2. P2: Composition discards calculated evidence results

Locations: `internal/assessment/assess.go:9-15,31-33`, `internal/validate/pipeline.go:28-34,68-71`, and `internal/report/report.go:30-43`.

The assessment returns candidate feasibility and evidence estimates through `Result.Candidates`. The pipeline copies only scopes and findings. Its report type has no place for candidate estimates. Language-server references confirm that no downstream consumer reads `Result.Candidates`.

Observed: an explicit candidate with a matching measured evidence block produces no probability interval or information bounds in CLI output. The fixture contains 20 joint trials and 20 failures. The report shows only structural scopes and an unresolved-feasibility warning.

The result cannot travel from the assessment engine to the user. Independent numerical tests cannot expose this integration defect. The report also cannot retain the sampling assumptions and confidence metadata required beside a printed estimate.

The validation text contract does not explicitly require a numerical evidence section. This finding concerns discarded application results, not a missing evidence subcommand.

Carry candidate results through the composed result. When validation presents estimates, include their evidence identity, counts, confidence, sampling assumptions, exposure, and information bounds. Do not add a new command or infer a supported ranking.

Acceptance: exercise matching evidence through composition, not only the numerical helper. Make sure that the composed result preserves the estimate and its qualifications. Keep two evidence blocks separate.

### R3. P2: The unresolved summary omits missing support

Locations: `internal/validate/compose.go:88-109` and `internal/report/render_sections.go:25-31,90-99`.

`unresolvedCauses` includes unresolved structural classification, external unknowns, and unresolved feasibility. It excludes `support_gap`, unresolved refinement, and other unresolved obligations. Scope output also prints a classification without an explicit support status.

Observed: `validate --profile architecture examples/architecture.ad.hcl` exits 0 with four `support_gap` warnings. Both scopes say `decoupled`. Its summary then says `Unresolved causes: - none.`

The warnings remain visible, so this is not complete suppression. The summary still contradicts the qualification that supported independence remains unresolved. The command contract requires the summary to retain unresolved outcomes and causes.

Represent structural classification and support status separately. Include missing support and other unresolved obligations in the summary. Keep profile severity separate from claim truth.

Acceptance: the existing architecture example must retain its structural result and warnings. Its summary must not state that no unresolved cause exists.

### R4. P2: Owner results omit required input provenance

Locations: `internal/owners/owners.go:34-49,97-103,177-184`, `internal/validate/compose.go:16-27`, and `internal/report/render_sections.go:73-87`.

An attachment result contains its label, subject, outcome, and cause. The enclosing owner result adds its owner and contract. Neither result retains source locations, attachment revision, core revision, or an invocation identity. The renderer does not print the owner's contract.

Observed: the project example prints a valid view result with no source location or input revision. Binding and behavior warnings retain diagnostic locations, but those locations are not part of their result identities. The report cannot distinguish two runs that change payload bytes while keeping attachment labels unchanged.

The language requires exact input provenance for owner results at `language.md:387-402`. Implementation-plan line 257 also requires source and invocation revisions. The absence of a persistent cache removes stale-cache risk, but it does not satisfy result traceability.

Keep source bytes and paths distinct from semantic identity. Attach invocation-local source identities and snapshot identity to each result. Preserve the owner contract and source range through presentation. No hash algorithm or interchange format is required for this release.

Acceptance: change a valid payload while keeping its label and subject fixed. Core conclusions must remain unchanged, and the result must identify its changed input.

### R5. P2: Invalid envelopes hide duplicate attachment labels

Locations: `internal/schema/decode_manifest.go:129-154,160-195`, `internal/schema/resolve_attachments.go:16-39,71-91`, and `internal/validate/pipeline.go:94-97`.

`DecodeAttachments` drops an invalid envelope before label uniqueness runs. `labelFindings` therefore sees only valid envelopes. The pipeline also skips all attachment reference and label checks when the core graph is unavailable.

Observed: two view attachments named `same`, with `data` absent from the second, produce only `invalid_attachment`. The report emits no `duplicate_identity` and retains a valid result for the first declaration.

Label uniqueness does not depend on a valid payload or core graph. The specification requires independent findings after a prerequisite fails. The current order hides an independent identity defect and loses its related location.

Collect attachment labels and ranges before envelope filtering. Run project-wide label checks without requiring a resolved core graph. Continue to block payload assessment for invalid envelopes.

Acceptance: duplicate labels must retain both ranges when either envelope is invalid. The same identity finding must remain visible when an unrelated core reference fails.

### R6. P2: The validation boundary depends on the CLI adapter

Locations: `internal/cli/cli.go:13-21`, `internal/validate/validate.go:7,19-31`, `internal/validate/pipeline.go:7,40`, and `internal/validate/compose.go:8,16`.

The CLI package owns the validation request. The validation operation imports that package and accepts `cli.Request`. Validation also renders text and returns a process exit code. Process-success policy lives in `report.ExitCode`.

This reverses the Clean Architecture dependency rule at the application boundary. A non-CLI caller must depend on CLI types and text delivery. A new presenter cannot reuse an exported assessment operation without repeating pipeline work or using the CLI-shaped interface.

This is a static dependency finding, not an observed command failure. The core assessment and selection packages remain separate from the CLI. No dependency cycle is claimed.

Put the application request and composed result at the validation boundary. Let the CLI translate arguments into that request. Let the CLI or an outer presenter render the result and translate operation status into process exit codes. Keep `main` as the composition root. A generic framework or repository layer is unnecessary.

Acceptance: a non-CLI caller can invoke validation and inspect a composed result without importing `internal/cli` or providing an output writer.

### R7. P2: Validation allocates a dense master matrix

Locations: `internal/assessment/hierarchy.go:22-55` and `internal/assessment/cells.go:23-40,53-60`.

Both matrix builders allocate and store a `Cell` for every FR-DP combination. The master matrix includes missing cross-branch relations as allocated unknown cells. Each master cell also searches sibling sets through `containingSet`.

This contradicts implementation-plan line 224, which requires sparse facts and defaults for validation. The map representation looks sparse, but both nested loops populate every coordinate. Storage therefore grows with the product of the axis sizes, even when authored influences stay sparse.

This is a static allocation finding. The review does not claim a measured memory limit or latency failure. Shared-process memory exhaustion remains an operational risk rather than proof of semantic coupling.

Retain all axes, exact facts, and scoped defaults. Resolve absent cells on demand for classification and findings. Index sibling membership once rather than scanning every sibling set for every coordinate. Preserve unknown cells without storing each one separately.

Acceptance: compare sparse models with increasing leaf counts. Retained cell storage must follow authored facts and defaults rather than the full Cartesian product. Classification must still retain rectangular axes and cross-branch unknowns.

### R8. P1: State ownership rules stop at duplicate DP claims

Location: `internal/schema/resolve_modules.go:89-121`.

`checkStateOwners` detects two DPs that claim the same state. It does not require the owning DP to be a leaf. It does not compare module ownership with DP ownership.

Observed: a parent DP owns `state.cache`, a leaf module realizes its child DP, and both that module and a composite claim the cache. The built executable exits 0 with no errors. Core assessment completes.

All three claims violate `schema.md:459`. State ownership belongs to leaf controls. A module cannot claim state owned by an unrelated DP. A composite cannot introduce independent ownership.

The resolved graph therefore does not establish the ownership invariant that downstream assessment assumes. This undermines cross-branch influence review. It does not prove that a particular runtime data race exists.

Make ownership resolution establish one logical leaf owner for each claimed state. Require module claims to match DPs that the module realizes. Reject composite ownership without counting a matching leaf/module declaration as two logical owners.

Acceptance: reject each invalid claim independently. Retain both ranges for competing logical owners. Accept a leaf DP and its realizing module that describe the same owner.

### R9. P2: Cell defaults can target a nonexistent sibling set

Locations: `internal/schema/resolve_refs.go:91-94` and `internal/assessment/cells.go:94-109`.

The resolver checks that `cells.scope` names a design or DP. It does not require that identity to own a nonempty sibling set. Assessment then ignores defaults that match no assessed scope.

Observed: add a `cells` block scoped to a leaf DP with no child FRs. The built executable exits 0 with no errors. The default never applies.

The schema explicitly requires an existing nonempty sibling set at `schema.md:310`. A valid reference kind is not sufficient. Silent acceptance leaves authors unable to distinguish an unused declaration from a valid default.

Resolve cell scopes against the derived sibling sets. Reject scopes without children before assessment.

Acceptance: a default scoped to a leaf DP must produce a structural error. A default scoped to its parent sibling set must still apply only to unlisted cells in that set.

### R10. P1: Measured information can reference another candidate's evidence

Locations: `internal/schema/resolve_refs.go:149-162` and `internal/assessment/candidates.go:105-121`.

The resolver checks that `information.evidence` names an evidence block. It does not require that block to name the enclosing candidate. Assessment scans all evidence by its own `candidate` field instead of respecting the information block's reference list.

Observed: candidate `current` requests measured information from `evidence.trial`, but that block names candidate `alternative`. Both candidates share the same exposure and version. Validation exits 0 without `invalid_evidence` or another structural error.

This violates `schema.md:389`. An evidence reference must belong to its exact enclosing candidate. The current graph accepts a false link, then assessment silently attributes the evidence elsewhere.

A temporary program also exercised the assessment API with `information.basis = "unknown"` and an unreferenced matching evidence block. The API still returned one estimate with bounds `[0.8388748419471808, 1]`. The schema forbids turning retained unknown-basis evidence into an estimate.

During resolution, compare each referenced evidence block's candidate with the enclosing candidate. During assessment, honor the requested information basis and evidence references. Keep unrelated observations separate from requested estimates.

Acceptance: reject a cross-candidate evidence reference. Adding an unrelated evidence block must not change another candidate's requested information result.

### R11. P1: Observed required failures remain unresolved instead of infeasible

Location: `internal/assessment/candidates.go:13-15,55-75,198-207`.

Every candidate receives the hard-coded feasibility value `unresolved`. The code always emits `unresolved_feasibility`, regardless of applicable observed failures. Its comment incorrectly states that version 1 cannot establish failure.

Observed: one FR requires `value = true` with `op = "=="`. A matching joint scenario verifies that sole criterion. Evidence records 20 failures in 20 trials under the candidate's exact version and exposure. Validation exits 0 and emits no `infeasible_candidate`.

The assessment contract explicitly permits observed criterion failures to establish infeasibility at lines 170 and 237-241. The absence of trusted qualitative review does not authorize ignoring measured failure. Uncertainty about a future success probability is distinct from an observed violation of a required predicate.

Assess applicable failures before retaining unresolved feasibility. Emit `infeasible_candidate` when qualified evidence demonstrates a required failure. Do not automatically convert an all-success finite sample into feasibility or certainty.

Acceptance: the reproduced all-failure case must exit 1 with the affected candidate and criterion identified. Missing evidence must remain unresolved. All-success samples must retain sampling uncertainty.

### R12. P1: Numerical estimates bypass predicate coverage

Locations: `internal/assessment/candidates.go:105-121,151-159`.

`estimatesFor` checks numeric validity and exposure equality. It does not inspect the evidence scenario's applicability or its coverage of required operational criteria. It does not establish that all addressed outcomes form the observed joint event.

Observed through the assessment API: a decision addresses an FR with two operational criteria. Its evidence scenario verifies only one. The API returns one probability estimate with bounds `[0.8388748419471808, 1]` and no `invalid_evidence` finding.

The schema requires the scenario to identify all addressed operational predicates at lines 399 and 412. `dependence = "joint"` is an authored declaration, not a substitute for predicate coverage.

Resolve the candidate's required applicable predicates before calculating estimates. Require its evidence scenario to apply and cover that complete set. Keep missing coverage unresolved or diagnose the violated evidence contract. Do not present a partial-outcome sample as joint success.

Acceptance: removing one required criterion from `scenario.verifies` must prevent a qualified estimate. Evidence for an inapplicable scenario must also remain unqualified.

### R13. P2: Selected decisions can overlap on the same DP

Locations: `internal/selection/activation.go:20-32,37-53,118-128`.

Default selection checks each decision's lifecycle agreement separately. Candidate membership indexing stores only whether a DP appears anywhere. It does not preserve the declaring decision or detect overlapping declarations.

Observed: two decisions each select a candidate containing the same selected DP. Validation exits 0 without `invalid_selection`. Each decision passes its local lifecycle check.

The assessment contract rejects overlapping decision declarations at lines 38-43. Local checks do not establish consistency across decisions.

Index candidate membership with its declaring decision. Reject a DP listed by distinct decisions before default or explicit snapshot assessment. Do not confuse multiple candidates within one decision with overlap across decisions.

Acceptance: the reproduced two-decision model must fail selection. Separate decisions with disjoint DP membership must remain admissible.

### R14. P2: One operational criterion hides a qualitative criterion

Location: `internal/assessment/profile.go:123-147`.

`criteria` skips an FR as soon as `hasOperationalCriterion` finds one complete applicable predicate. It does not report other qualitative criteria on that FR.

Observed: add a qualitative `criterion` beside a complete operational criterion. Validation emits no `incomplete_criterion`. This omits a source warning and the corresponding architecture-profile error.

The assessment contract requires qualitative criteria to remain visible at lines 204-208. Finding one complete predicate does not complete another acceptance condition.

Assess each applicable criterion for completeness. Separately require that every active FR has at least one applicable operational predicate. Preserve qualitative source statements instead of inventing thresholds.

Acceptance: a mixed-criterion FR must retain its qualitative gap under both profiles. An inapplicable criterion must not satisfy active criterion coverage.

### R15. P2: Shared PVs bypass shared-resource review

Location: `internal/assessment/profile.go:354-367`.

A process variable (PV) describes a DP realization. `sharedResourceGaps` inspects only state declarations. It never examines a PV that realizes several active leaf DPs.

Observed: one PV realizes two active leaf DPs, and only their diagonal influences are declared. Validation reports unknown cells but no `shared_resource_review_gap`. The source profile exits 0.

The diagnostic contract explicitly includes shared PVs at line 47. Unknown matrix cells do not explain that shared realization creates a specific review obligation.

Include active PV realization sets in shared-resource assessment. Preserve the PV and affected controls in the finding. Sharing alone does not prove a material influence or justify a zero.

Acceptance: the reproduced shared PV must produce a profile-dependent review-gap finding. Inactive realizations must not create an active sharing claim.

## Architecture assessment

The staged pipeline is worth keeping. `main` supplies the version and validation handler. Admission retains source bytes. Resolution precedes selection. Core assessment and fixed owner assessment produce separate findings. Composition applies severity and orders findings before text delivery.

N1 maps separately to FR1 and DP1, the dedicated version value and handler. The freshly built executable reports `adhcl dev`. Existing acceptance tests also exercise linker injection. This review found no new version-path defect.

The following mapping reconstructs the implementation using the design document's stable identifiers. The acceptance criteria come from the existing plan, not new numerical targets.

| Need | FR under DP2 | Acceptance criterion | Actual DP responsibility | Main findings |
|---|---|---|---|---|
| N2: Detect invalid facts | FR2.1 | Reject invalid fields, references, hierarchy, and ownership before dependent assessment | DP2.1: Input, syntax, schema, and graph resolution | R5, R8, R9, R10 |
| N3: Preserve selected meaning | FR2.2 | Select one valid candidate and its applicable conditions without deleting alternatives | DP2.2: Invocation-local selection | R13 |
| N3: Derive qualified findings | FR2.3 | Preserve effects, unknowns, support limits, and evidence qualification | DP2.3: Core assessment | R1, R7, R10, R11, R12, R14, R15 |
| N2, N3: Assess attachments | FR2.4 | Apply exact owner contracts without changing core conclusions | DP2.4: Fixed owner dispatch and payload assessment | R4, R5 |
| N4: Deliver traceable results | FR2.5 | Preserve independent results, provenance, unresolved causes, and correct operation status | DP2.5: Composition and text delivery | R2, R3, R4, R6 |

The influence model applies to semantic changes within fixed stage contracts and one invocation. `X` means a material influence. `0` means no material influence under those conditions. This is not a call graph or a performance model.

| FR / DP | DP2.1 | DP2.2 | DP2.3 | DP2.4 | DP2.5 |
|---|---|---|---|---|---|
| FR2.1 | X | 0 | 0 | 0 | 0 |
| FR2.2 | X | X | 0 | 0 | 0 |
| FR2.3 | X | X | X | 0 | 0 |
| FR2.4 | X | X | 0 | X | 0 |
| FR2.5 | X | X | X | X | X |

Admission changes which facts and subjects downstream stages can assess. Selection changes active influences and attachments. Core and owner policies change the results that delivery must preserve. These mechanisms explain the off-diagonal entries.

The important zeros depend on retained authority boundaries. Owners consume graph facts and active membership, not core classification. Core assessment does not interpret owner payloads. Presentation does not rewrite admitted facts or active membership. The review found no contrary write in those paths, but did not prove immutability for every possible change.

The matrix has a triangular influence shape. Establish admission first, then selection, then core and owner policy, then composition. Core and owner policy have no required order between them. After an upstream contract changes, recompute dependent results.

This shape does not establish a compliant or supported design. The findings show missing controls over parts of the acceptance criteria, including ownership and result traceability. Shared memory exhaustion can block multiple stages, especially through R1 and R7. No process-level isolation is claimed.

The control junction (C) orders stages with prerequisites. The summation junction (S) combines independent core and owner results. No feedback junction (F) is required by this validator pipeline. Cycles inside an authored design do not require a runtime feedback loop in the validator.

### Target boundaries

Keep the FR hierarchy and staged mechanisms. Correct their contracts rather than introducing services or a generic plugin system. The revised ownership allocation is:

| Responsibility | Current owner | Target owner |
|---|---|---|
| Request data for validation | CLI adapter | Application validation boundary |
| Composed assessment and operation status | Split across validation and text-report code | Application result contract |
| Text layout and process exit translation | Mixed with application execution | CLI and text presenter |
| State ownership and evidence-reference validity | Incomplete graph checks | Schema resolver before assessment |
| Owner result provenance | Missing fields | Owner result linked to invocation sources and snapshot |

Each boundary returns facts and findings rather than exiting the process. Invalid graph facts block dependent assessment. Independent source and envelope findings remain available. The presenter receives a complete result and cannot change its truth or support status.

Clean Architecture score: 8/10 for the reviewed boundary. Five of seven diagnostic rows pass, while source-dependency direction and delivery-independent use cases fail through R6. Database replacement is not exercised because this program has no database. Main wiring, core testability, and the acyclic package graph are strengths.

Clean Code score: 6/10 for the reviewed paths. Names and package responsibilities are mostly clear, and the existing suite passes. Exhaustive search, discarded results, incomplete relational checks, and split ownership rules prevent a higher score. Correct those behaviors before shortening functions or renaming local variables. No finding relies solely on function length or argument count.

## Alternative selection and evidence limits

Bounded ambiguity witnesses and sparse cell resolution fit the existing requirements. Exhaustive enumeration and dense storage add work without improving the required validation decision. Their rejection rests on the explicit plan and algorithmic growth, not a measured success probability.

A small application-owned request and result contract separates policy from delivery. A callback that still accepts CLI-owned data leaves the dependency direction unchanged. A full framework adds no required capability for the two-command release.

No observed joint success rate supports an Information Axiom score for these implementation alternatives. Information content is `I = -log2(p)`, where `p` means joint success probability. Code size, package count, and these review scores are not information content. Compare corrected alternatives with the same fixtures, malformed inputs, operating conditions, and release horizon.

## Verification performed

The review built a fresh executable instead of assuming that `bin/adhcl` matched source. The existing `go test ./...` suite passed. Most package results came from Go's test cache.

These commands completed successfully:

```sh
go build -o /tmp/adhcl-review ./cmd/adhcl
go test ./...
/tmp/adhcl-review version
/tmp/adhcl-review validate examples/minimal.ad.hcl
/tmp/adhcl-review validate --profile architecture examples/architecture.ad.hcl
/tmp/adhcl-review validate examples/project/model.ad.project.hcl
```

Focused temporary inputs exercised ambiguity, observed failures, ownership, orphan defaults, crossed evidence references, duplicate attachment labels, overlapping decisions, mixed criteria, and shared PVs. A temporary program exercised the real parser, decoder, resolver, selector, and candidate assessment API. It exposed the unknown-basis and incomplete-coverage estimates described in R10 and R12.

The review added no permanent tests and changed no implementation source. Temporary inputs and the assessment probe were removed after execution. This document retains the source locations, input conditions, observed outcomes, and proposed acceptance cases.

The review did not run race checks, vet, cross-platform builds, release archives, or resource-exhaustion tests. It did not establish latency, memory, availability, or success-probability guarantees. The existing suite's success does not cover the reproduced counterexamples.

## Correction order

First, correct R8 and R10 so that invalid ownership and evidence links cannot enter a resolved graph. Correct R13 alongside snapshot membership checks. Keep independent diagnostics available when a prerequisite fails.

Next, correct R11 and R12 before exposing or trusting numerical evidence results. Correct R14 and R15 so that profile completeness does not hide required obligations. Bound ambiguity work in R1 before accepting untrusted model sizes.

Then, complete the application result contract for R2, R3, R4, and R6. Correct duplicate-label collection in R5 and sparse storage in R7. R9 belongs in schema scope checks.

For each correction, retain a regression case that reproduces the reported consumer-visible defect. Exercise the corrected behavior through the built executable or the actual public assessment API. Do not change authored expectations merely to recover a passing suite.
