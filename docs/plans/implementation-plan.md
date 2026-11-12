# adhcl v0.1.0 implementation plan

This plan gives a human or an LLM the work needed to implement the Go tool. Read [the axiomatic design](../axiomatic-design.md) for the command contract, ownership rules, and acceptance criteria. This task produces documents only. None of the steps below claims completed implementation.

## Baseline and release boundary

Implement only the `version` and `validate` subcommands. Support core files, immediate core directories, and explicit project manifests. Include syntax, structure, method, profile, assertion, evidence, and required attachment findings.

Do not add formatting, report subcommands, HTML output, a playground, dynamic plugins, observation discovery, or a persistent cache. Do not retain the old `valid` and `strict` profiles. Preserve qualified unknown external results rather than inventing successful observations.

The language baseline is spec revision `3eb0baf79490aab8a1129adc4d58c330ecd0cbd8`. Its `docs/language.md` and `docs/assessment-contract.md` are authoritative. The assessment contract takes precedence for assessment differences. The review and example provide evidence and cases, not replacement field rules.

The spec's later contract closure supplies [field rules](../../../spec/docs/schema.md), [diagnostic rules](../../../spec/docs/diagnostics.md), and [versioned acceptance cases](../../../spec/conformance/core-1/README.md).
Step 1 must pin a revision that includes those files and the updated assessment contract.
The old baseline alone does not include these decisions.

The implementation currently lacks source files, examples, fixtures, and `go.sum`. The Go module and release configuration identify this repository. The Makefile and CI still contain targets outside the release scope. Keep working release-tag scripts unless a changed contract requires an update.

## Work order and completion gates

Complete the work in dependency order. Steps 7 and 8 can proceed independently after step 6. Integration requires both paths before step 9.

| Step | Responsibility | Design coverage | Completion gate |
|---|---|---|---|
| 1 | Close and pin the language contract | All validation leaves | A complete schema and expected conformance cases exist. |
| 2 | Align the repository and build | DP0, DP1 | A minimal executable builds under the supported Go matrix. |
| 3 | Implement command routing and version | FR1 and command boundary | Exact version, help, usage, and output-failure behavior runs. |
| 4 | Select files and admit syntax | FR2.1.1 | Input and expression cases preserve locations and reject unsafe input. |
| 5 | Decode schema and resolve the graph | FR2.1.2 | Every field and reference follows the pinned schema. |
| 6 | Select snapshots and conditions | FR2.2 | Activation and applicability cases match the assessment contract. |
| 7 | Derive core findings | FR2.3 | Method, profile, assertion, and evidence cases pass. |
| 8 | Assess attachments | FR2.4 | Payload and isolation cases produce qualified owner results. |
| 9 | Compose and deliver results | FR2.5 | Reports and exit codes preserve complete and partial outcomes. |
| 10 | Prove the complete CLI path | FR0 and V1–V12 | Built binaries exercise success, failure, and change boundaries. |
| 11 | Prepare release v0.1.0 | N1–N4 | Documentation, CI, archives, and injected version agree. |

Do not skip a blocked gate by reducing validation to HCL parsing. Record unresolved language decisions explicitly. This plan includes resolving those decisions before release, not claiming that the current specification is complete.

## 1. Close and pin the language contract

The spec now supplies the normative field and diagnostic contracts that review R1 requested.
Step 1 adopts and pins those contracts rather than inferring a decoder from examples.
Do not change the sibling repository silently as part of tool implementation.

1. Record the spec revision and the precedence rule in a local conformance reference.
2. Enumerate every top-level and nested block from the language reference.
3. Create a field table with type, cardinality, required status, default, null behavior, enum values, and reference targets.
4. Define the exact label grammar and namespace of every named nested block.
5. Define which blocks accept `cites`, `origin`, `alias`, and other common metadata.
6. Define absent, empty, and null behavior separately.
7. Resolve the remaining ambiguities listed below through normative decisions.
8. Create expected cases for each decision before decoder implementation.
9. Pin the accepted revision and copy permitted conformance assets into this repository.

Make sure that the adopted schema covers these areas:

| Area | Required decision |
|---|---|
| References | Define scalar versus list forms, allowed targets, criterion paths, candidate paths, and hierarchy reachability from the single root. |
| Core fields | Define `ranges`, `criterion.value`, `under`, origin fields, enum fields, `flow.steps`, and nested loop types. |
| Metadata | Define whether `cites` accepts one source, several sources, or both. Distinguish influence `source` from `cites`. |
| Example-only fields | Define `scenario.kind`, `expect.under`, and decision-level context, workload, and horizon. |
| Review support | Define machine-recognizable evidence of an externally reviewed argument. Never infer support from citation presence. |
| Profile severity | Assign rule codes and severity to incomplete coverage, unresolved refinement, pairing ambiguity, and support gaps in each profile. |
| Expectations | Define results for source, conditional, and supported assertions when cells conflict or remain unknown. |
| Extension authority | Define whether a known mismatch changes validation status. The proposed CLI supplies no observations, so unavailable-input results remain unknown. |
| Owner identity | Fix the built-in labels that select the three exact owner contracts. Use labels from adopted manifest examples where supported. |

The minimum block inventory includes these groups:

- Sources, design roots, stakeholders, needs, constraints, assumptions, contexts, workloads, and state.
- FRs with criteria, DPs, influences, cell defaults, and expectations.
- Decisions with candidates and information blocks, coupling records, and empirical evidence.
- PVs, modules, flows with loops, and scenarios.
- Manifest projects, extension declarations, and attachment envelopes.

Use the spec's diagnostic catalog with stable codes and both profile severities.
Separate invalid structure, failed assertions, incomplete support, and qualified external unknowns.
Core version 1 defines no trusted review-import input, so this offline interface must retain qualitative support as unresolved.
Do not treat citation presence or a premise's authored status as an accepted review.

Make sure that all fields in the supplied self-design example appear in the schema. Do not assume that the example must pass every assertion. Record its expected gaps and findings independently. Missing historical examples and linked RFCs are not release fixtures.

The gate ends only when another implementer can decode each field without guessing. Any unresolved normative question blocks the affected conformance claim. It does not justify ignoring unknown fields or accepting arbitrary HCL values.

## 2. Align the repository and build

Use the repository remote as the identity source. Keep Go 1.25 as the current declared minimum unless dependency compatibility requires an explicit change. Do not change the toolchain pin merely to bypass a build failure.

1. Keep the module path `github.com/axiomatic-design-hcl/adhcl`.
2. Establish the package boundaries listed below.
3. Retain native HCL and cty dependencies that the implementation uses.
4. Run dependency resolution and commit the resulting `go.sum`.
5. Remove unused dependencies and tool directives through normal module maintenance.
6. Replace inherited Makefile targets with targets for the two-command release.
7. Update CI to run those targets under each declared Go version.
8. Keep the GoReleaser owner and repository as `axiomatic-design-hcl/adhcl`.
9. Replace the release-version expectation `v0.18.0` with a v0.1.0 injection case.

Use this proposed source layout:

| Path | Responsibility |
|---|---|
| `cmd/adhcl/main.go` | Own `buildVersion`, construct dependencies, and exit once. |
| `internal/cli/` | Parse arguments, route commands, select profile, and deliver text. |
| `internal/input/` | Select sources, enforce manifest path rules, and preserve bytes and ranges. |
| `internal/syntax/` | Parse native HCL and enforce the expression profile. |
| `internal/schema/` | Define the pinned field rules and decode admitted syntax. |
| `internal/model/` | Own typed facts, reference identities, and resolved graph contracts. |
| `internal/selection/` | Own active snapshots and applicability. |
| `internal/assessment/` | Derive core cells, coverage, classification, assertions, and evidence findings. |
| `internal/owners/` | Own the fixed registry and separate payload contracts. |
| `internal/result/` | Define diagnostics, stage statuses, and composition without text formatting. |
| `testdata/` | Hold isolated input cases and expected behavior. |
| `examples/` | Hold complete user examples for the supported release. |

Use concrete data and functions unless a boundary needs replacement during a failure test. Do not create a plugin framework, generic repository layer, or one package per rule. Define contracts once and keep imports directed toward facts and results.

Remove `fmt`, strict-profile, WebAssembly, and playground targets from the active build contract. Add actual `.ad.hcl` examples and explicit manifest cases. Keep `build`, `test`, `test-race`, `vet`, module hygiene, formatting, relevant analysis tools, and release-tag gates when they apply.

Inspect the Go toolchain with `go version` and `go env GOTOOLCHAIN`. Record any unavailable toolchain or dependency as a build prerequisite, not an implementation success. Make sure that CI and the documented local commands use the same supported versions.

## 3. Implement command routing and version

Implement the exact command contract in the design document. Standard Go argument parsing is sufficient for two commands. Preserve dependency injection for output streams and file access where failure scenarios need it.

1. Keep `main.buildVersion` writable through linker injection.
2. Use `dev` when the linker supplies no release version.
3. Implement version output without constructing model-processing dependencies.
4. Parse flags before the single validation path.
5. Support `--` for a path that starts with a dash.
6. Reject repeated flags and unexpected arguments explicitly.
7. Return operation results to the entry point instead of calling process exit inside modules.
8. Return code 1 when output cannot be delivered.

Cover the release and development outputs, unavailable model access, help, unknown commands, duplicate flags, and missing input. Make sure that invalid usage returns code 2 before model access. Do not add `--version` or an implicit `validate` alias.

## 4. Select files and admit syntax

Decode the manifest envelope only enough to select inputs. Apply the same restricted literal rules to its fields. Never evaluate input as general HCL configuration.

1. Classify the explicit input by path kind and required suffix.
2. Select immediate core members for a directory without recursion.
3. Sort selected paths for display without assigning semantic precedence.
4. Resolve manifest paths relative to their manifest directory.
5. Resolve symbolic links before containment and physical-identity tests.
6. Reject aliases to the same physical file, including hard links where the platform exposes identity.
7. Reject missing files, directory members, escapes, globs, nested manifests, and wrong suffixes.
8. Read each selected file once and retain its bytes for parsing and diagnostics.
9. Parse native HCL with `hclsyntax` and retain source ranges.
10. Inspect expression nodes before evaluating accepted literals.
11. Detect duplicate object keys before converting entries to a map.
12. Retain all independent source findings when another file fails.

Allow native strings, escaped template markers, heredocs, literal collections, Boolean values, and permitted numbers. Admit null only under the pinned field contract. Permit a leading minus only on a numeric literal. Keep reference traversals distinct from strings.

Reject interpolation, directives, calls, arithmetic, comparisons, conditionals, comprehensions, indexing, splats, and computed object keys. Exercise nested collections and heredocs so forbidden expressions cannot hide inside accepted containers. Do not impose unsupported numeric or heredoc restrictions.

Keep the sources fixed after reading. Files can change during initial collection, and the release makes no atomic filesystem snapshot guarantee. Do not reread a file later and combine different revisions inside one result.

## 5. Decode schema and resolve the graph

Use the accepted schema from step 1 as the decoder's only field authority. Decode all records before resolving cross-file references. Preserve multiple influence records instead of using endpoint pairs as replacement keys.

1. Enforce block label counts, names, required fields, types, and enum values.
2. Reject unknown blocks, attributes, duplicate labels, and invalid nulls.
3. Create typed identities with namespace, full path, and source range.
4. Resolve core references across all selected core files.
5. Resolve nested criteria and candidate references without shortening their paths.
6. Require exactly one root and an acyclic, reachable hierarchy.
7. Resolve attachment owners and subjects only against valid declarations and graph facts.
8. Resolve module containment, realizations, state ownership, and flow references.
9. Reject conflicting containment and duplicate ownership or realization claims.
10. Reject removed `import` blocks and `module.handles` fields without compatibility shims.

An invalid core graph blocks dependent snapshots and assessments. Return its diagnostics and blocked statuses. Continue independent envelope and source findings when they do not require guessed core structure.

Cover wrong-kind references, cross-file duplicates, missing nested targets, cycles, multiple roots, duplicate state owners, module relationships, and unknown fields. Include a string that resembles a reference and require its rejection in a traversal field.

## 6. Select snapshots and conditions

Follow the assessment contract's snapshot algorithm. Keep the full graph and store active membership separately. Do not delete rejected, proposed, superseded, or inactive records.

1. Activate selected DPs for the default snapshot.
2. Resolve an explicit candidate and activate its listed DPs.
3. Add only the selected DPs that qualify as common across candidates.
4. Reject rejected or superseded candidate members.
5. Require active ancestors instead of inserting them silently.
6. Require active controlled FRs and retain required root FRs.
7. Require default candidate and lifecycle agreement.
8. Restrict modules and PVs to their active realization sets.
9. Resolve the requested context and its workload.
10. Reject incompatible context selections and retain unresolved applicability.
11. Expand multi-source influences without losing their source identity.
12. Bind applicable evidence to its exact candidate version and operating range.

Cover common-DP selection, competing candidates, inactive branch coverage, missing ancestors, and explicit proposed alternatives. A missing context cannot silently turn conditional records into unconditional facts.

## 7. Derive core findings

Use one resolved influence model for all validation findings. Keep structural classification separate from support qualification and feasibility. The CLI does not need extra report commands to compute the results needed by validation.

### Cells, hierarchy, and control

1. Resolve applicability before comparing effects.
2. Disable effects whose premises are invalidated and retain their unresolved obligations.
3. Keep assumed and unevaluated premises conditional.
4. Merge compatible exact records without losing sources.
5. Retain contradictory known effects as conflicts.
6. Keep an exact unknown visible beside known assertions.
7. Apply defaults only when no applicable exact record exists.
8. Keep unlisted cells unknown when no default applies.
9. Build sibling control graphs and require a unique complete pairing.
10. Preserve all rectangular axes and missing-control findings.
11. Require material diagonals before square classification.
12. Derive precedence from off-diagonal material effects.
13. Find cycles and valid triangular orders while preserving labels.
14. Keep unknowns visible even when a known cycle exists.
15. Build active leaf axes without duplicating multi-control DPs.
16. Retain exact cross-branch effects and parent refinement obligations.

Do not implement exhaustive enumeration of every possible pairing or order. Use graph matching to determine completeness and uniqueness. Use graph traversal to derive precedence, cycles, and a stable display order. Test another valid authored order instead of requiring equality with that display order.

Do not fill a dense master matrix merely to report sparse validation findings. Preserve every axis and compute missing cells from exact facts and defaults. No performance guarantee follows from this storage choice.

### Profiles, assertions, and evidence

1. Apply the pinned rule catalog for both review profiles.
2. Require complete operational criteria and module contracts in the architecture profile.
3. Require explicit empty state and dependency lists where the contract requires them.
4. Require each leaf scenario and each applicable residual-feedback safety contract.
5. Evaluate expectations under their declared basis and order.
6. Keep failed assertions as errors under both profiles.
7. Assess candidate feasibility before any supported information comparison.
8. Require common FRs, criteria, contexts, workloads, and horizons for comparison.
9. Require qualified joint trials and compatible evidence metadata.
10. Reject invalid trial counts, failure counts, and confidence values.
11. Compute the specified two-sided Wilson interval without continuity correction.
12. Transform probability bounds into information bounds without losing infinite or unknown status.
13. Preserve separate evidence blocks unless an accepted joint argument authorizes pooling.
14. Retain support gaps when arguments lack substantive review.

The source profile must not invent metrics or contracts to close historical gaps. The architecture profile must not promote complete prose into evidence of success. Keep unsupported feasibility unresolved.

Use independently calculated numerical cases for Wilson estimates, boundary samples, and information bounds. Include all-success and all-failure samples. A printed probability estimate must retain its sampling assumptions and confidence range.

## 8. Assess attachments

Implement fixed local owners for the three initial role contracts. Adopt their registry labels during step 1. Unknown owners, incompatible versions, and wrong roles are errors even when unused.

1. Require exactly `owner`, `subject`, and literal `data` in each attachment.
2. Require unique attachment labels and valid typed subjects.
3. Dispatch a payload only to its declared owner.
4. Enforce each owner's exact fields, types, enums, and null prohibition.
5. Detect tool-specific binding data duplicated in core PV facts.
6. Require at most one view attachment for the relevant design and contract.
7. Attach owner, contract, subject, source, and invocation revisions to each result.
8. Detect duplicate result identities instead of keeping the last result.
9. Return unavailable-observation and unavailable-model outcomes as unknown.
10. Preserve completed core and independent owner findings after owner failure.

Do not read `observation_source`, `model`, or `revision` strings as file paths. Do not execute Terraform or an external behavior tool. The proposed CLI supplies neither observation sets nor behavior models.

Assess the HTML view payload without creating HTML during validation. Preserve separate validation and rendering status. Do not count a valid view payload as evidence that HTML rendering succeeds.

Cover unknown owner, wrong role, wrong subject, missing field, extra field, bad enum, null, duplicate attachment, and duplicate binding declarations. Change each valid owner payload while holding core facts fixed and compare core conclusions. Include a partial failure case with a valid core and an invalid required attachment.

## 9. Compose and deliver results

Compose results without modifying them. Keep core assessment, each owner, composition, and text delivery statuses separate. Do not hide a failed prerequisite behind an empty result list.

1. Include the input and selected snapshot identity in the report.
2. Include the tool version, core contract, profile, and workload.
3. Render each finding with severity, code, subject, message, and source ranges.
4. Include related locations for conflicting or duplicate records.
5. Order findings by stable path, range, code, and subject.
6. List unresolved outcomes and their causes in the summary.
7. Apply profile severity without changing a claim's truth.
8. Return code 1 for any error or incomplete required processing.
9. Return code 0 when the operation completes with warnings or qualified unknowns alone.
10. Return code 1 when report output fails.

Use standard output for validation reports. Use standard error for usage failures and inability to deliver the report. Do not invent source ranges for path or output errors.

Do not define a hash algorithm or interchange format in this release. Track source paths and bytes separately from semantic values and references. Invocation-local identities are enough because the tool stores no reusable assessment cache.

## 10. Prove the complete CLI path

Create permanent tests for plausible user-visible failures. Use the design scenarios V1–V12 as acceptance coverage. Do not test package names, source text, forwarding calls, or incidental wording as behavior.

Build an isolated fixture set with these groups:

| Group | Required cases |
|---|---|
| Commands | Release and development version, help, malformed usage, flag repetition, dash-prefixed path, and output failure. |
| Input | Single file, split directory, empty directory, explicit manifest, excluded neighbors, missing member, duplicate identity, and symbolic-link escape. |
| Syntax | Native numbers, negative literals, Unicode, escaped markers, heredocs, duplicate keys, forbidden nested expressions, and templates. |
| Graph | Cross-file references, wrong-kind references, duplicates, missing targets, cycles, module conflicts, and removed syntax. |
| Selection | Default lifecycle, explicit alternatives, common DPs, missing ancestors, inactive leaves, contexts, and premises. |
| Method | Diagonal, triangle, permuted triangle, cycle, unknown diagonal, unknown beside a cycle, rectangular axes, ambiguous pairing, and ineffective controls. |
| Hierarchy | Exact cross-branch effects, parent refinement, unsupported allocation, state ownership, and shared-resource review gaps. |
| Assertions and profiles | Valid alternative orders, failed expectations, incomplete criteria, missing contracts, missing scenarios, and residual-feedback gaps. |
| Evidence | Metadata mismatch, incompatible comparison, invalid counts, Wilson boundaries, unknown feasibility, and unpooled evidence. |
| Attachments | Each valid payload, unavailable external evidence, bad subjects, unavailable owners, invalid required payloads, and retained core findings. |

For each fixture, record expected status, rule codes, affected references, and critical range behavior. Avoid whole-report snapshots that make incidental prose a contract. Use temporary directories and independent files so tests run safely in parallel.

Add one small complete source example, one complete architecture example, and one manifest with required attachments. Copy or construct them only after the schema gate. Keep expected incomplete-design examples separate from examples that must exit 0 under the architecture profile.

Run the built tool, not only package tests. After the implementation creates the named examples, run this acceptance sequence:

```sh
go test ./...
go test -race ./...
go vet ./...
go build -o bin/adhcl ./cmd/adhcl
./bin/adhcl version
./bin/adhcl validate examples/minimal.ad.hcl
./bin/adhcl validate --profile architecture examples/architecture.ad.hcl
./bin/adhcl validate examples/project/model.ad.project.hcl
go build -ldflags '-X main.buildVersion=v0.1.0' -o bin/adhcl-release ./cmd/adhcl
./bin/adhcl-release version
```

The development binary must print `adhcl dev`. The injected binary must print `adhcl v0.1.0`. The manifest case must retain unknown external outcomes without claiming implementation conformance.

Exercise failure cases through a subprocess harness. Assert exact exit codes for usage, malformed source, failed expectations, invalid attachments, and report delivery failure. Make sure that each failure returns normally without panic. Run file-order and candidate-change scenarios against the actual binary and compare semantic findings.

Run the self-design example from the pinned conformance assets. Compare its result with the expected findings from step 1. Do not call an unexpected pass proof of conformance, and do not edit assertions merely to obtain success.

## 11. Prepare release v0.1.0

Publish commands only after the executable exercises their behavior. Update documentation after the smoke runs so examples describe observed output. Do not create or push a release tag without a separate release action.

1. Write the README with build prerequisites, the two commands, input selection, profiles, output streams, and exit codes.
2. Explain that successful validation does not prove independence or implementation conformance.
3. Document the three built-in owner contracts and unavailable external observations.
4. Record the accepted language revision and any remaining evidence limits.
5. Add v0.1.0 changes to the existing changelog.
6. Update contributing instructions with the supported local quality gates.
7. Make every active Makefile and CI target refer to an implemented command.
8. Run the local quality gate and release-tag script tests.
9. Cross-build the configured Linux, macOS, and Windows targets for amd64 and arm64.
10. Produce a local GoReleaser snapshot and inspect archive paths, license, README, and checksums.
11. Run the native release binary from an archive and observe its version and validation results.
12. Remove temporary smoke assets and retain only purposeful examples and regression fixtures.

The existing release workflow invokes `make verify` before GoReleaser. Keep this gate aligned with the supported command set. Make sure that the release destination and attestation paths refer to this repository's artifacts.

## Definition of complete

The release is complete only when all gates pass. Both commands must run from a built binary, not merely compile. Both profiles, all input forms, and all required payload contracts must follow the accepted schema.

Every in-scope need must retain its mapped acceptance coverage. Unknowns, conflicts, refinement obligations, and missing support must remain visible. A failed required attachment must fail the operation without erasing independent core findings.

Record the actual commands, observed outcomes, supported toolchains, and unexercised platform limits in the implementation handoff. A release remains blocked while normative schema decisions lack accepted rules. The current document deliverable does not claim that those future gates pass.
