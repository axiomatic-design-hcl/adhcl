# Assessment contract

This document defines core version 1 assessment semantics.
It supplements the field definitions in the language reference.
If an assessment rule differs, this document takes precedence.
The rules describe required behavior, not completed implementation tests.

The [field schema](schema.md) defines types, cardinalities, and namespaces.
The [diagnostic contract](diagnostics.md) defines rule codes, profile severities, and operation status.
All three contracts apply together.

## Snapshot selection

A snapshot is the active model for one candidate and context.
Without a candidate request, activate all DPs whose decision is `selected`.
An explicit candidate activates exactly its `dps` list.
DPs outside any candidate are common only when their decision is `selected`.
Add these common DPs to each explicit candidate.
A DP listed in another candidate is never common.

Every active child FR requires its parent DP to be active.
Every active DP requires all its controlled FRs to be active.
Root FRs remain required across candidates of the same decision.
Reject an explicit selection that lacks an ancestor. Do not insert ancestors silently.
A candidate cannot select rejected or superseded DPs.
A proposed DP can participate in an explicit alternative snapshot.
The default selected candidate must agree with selected DP lifecycle states.

Retain inactive records and their provenance for comparison.
Do not report inactive branch leaves as uncovered default requirements.
Apply an influence only when its target FR and source DP are active.
Expand a multi-source influence into one record per source DP.
Activate a module when at least one realized DP is active.
Restrict its assessed realization set to active DPs.
Apply a PV or attachment only when its subject is active.
Retain evidence only for its exact candidate version and operating range.

Version 1 accepts at most one explicit candidate and one context per assessment.
A candidate describes a complete alternative with common selected DPs, not an implicit replacement of a branch.
Multiple decisions remain recorded, but the tool does not construct a combination of their candidates.
In the default snapshot, each decision's selected candidate must agree with the active DPs that belong to that decision.
Conflicting lifecycle or overlapping decision declarations produce `invalid_selection`.
There is no file-order winner.

Resolve candidate context, workload, and horizon by the inheritance rules in the schema.
An explicit context request can supply an omitted candidate context.
It cannot override a different candidate context or an incompatible inherited workload.
Without a context request, use the candidate's effective context when present.
Resolve workload from the effective candidate, then its context, then the design root.
If two supplied declarations disagree, report `invalid_selection` rather than applying that fallback.
Without a candidate, a context request and design workload must also agree when both specify a workload.
A design workload does not select a context merely because its label matches.
The absence of a context keeps context-specific records unresolved.

## Cell resolution

Resolve applicability before effects.
An unconditional record applies in every context.
A context-specific record applies only in that exact context.
A workload-specific record applies when the snapshot names that workload.
Missing context information leaves conditional applicability unresolved.
Never select mutually excluded contexts together.

Treat `excludes` as symmetric incompatibility even when only one context lists the other.
Two different named contexts do not inherit each other's facts.
When their ranges overlap, neither context's claims establish an unconditional claim.
Assess each selected context separately.
An unconditional assertion and a contrary assertion for the selected context conflict in that assessment.
An unselected conditional assertion remains stored but cannot establish the selected result.

An influence with several `from` entries asserts each individual effect.
It does not assert an interaction that occurs only when all listed DPs change together.
Version 1 has no interaction-only cell form.
Do not encode an interaction-only argument as several individual effects.

For each applicable exact record, evaluate all `holds_if` premises.
An invalidated premise disables its claimed effect and leaves an unresolved obligation.
An assumed or unevaluated premise supports only a conditional effect.
This condition does not prevent a `basis = "source"` assertion from passing on applicable authored effects.
Such a result does not validate its premises or establish a supported conclusion.
A validated premise still requires applicable support for a supported conclusion.
DP premises qualify all control claims that depend on that DP.

Merge compatible exact records without discarding their sources.
Two known contradictory effects produce a conflict, regardless of their support levels.
An explicit unknown remains visible beside a known assertion.
It prevents complete classification until its uncertainty is resolved.
It does not erase the known assertion from structural reports.

Apply a `cells` default only when no applicable exact record exists.
An exact unknown therefore prevents a default zero.
Defaults never overwrite exact facts.
Conflicting applicable defaults produce a conflict.
With neither exact facts nor a default, resolve the cell as unknown.
Keep conditional effects, conflicts, disabled records, and support gaps in every projection.

## Pairing and classification

Use `controls` to build a bipartite graph between sibling FRs and their DPs.
A pairing assigns each FR a distinct controlling DP.
Use it only when exactly one complete pairing exists.
Multiple complete pairings produce `ambiguous_pairing`.
No complete pairing produces `uncontrolled_requirement` or `redundant_control`, as applicable.
Never select a pairing because its matrix looks simpler.

Classify a square mapping only when every paired diagonal has a material effect.
A zero diagonal is an ineffective control.
An unknown diagonal leaves classification unresolved.
Rectangular mappings retain their full axes and coverage findings.
Do not remove rows or columns to obtain a square mapping.

An uncoupled mapping has material diagonal cells and justified zero off-diagonal cells.
A decoupled mapping admits a triangular order with at least one material off-diagonal cell.
Reorder paired rows and columns together, retaining their labels.
For each off-diagonal material effect from DP A to the FR paired with DP B, require A before B.
A cycle in these precedence relations establishes structural coupling when all required effects are known.
Unknown cells prevent a complete classification, even when a known cycle is reported.
Use label order only to display otherwise equivalent valid sequences.
An upstream change invalidates affected downstream conclusions.
A design sequence does not automatically specify runtime execution order.

## Hierarchy and support

The master axes contain every active leaf FR and every distinct active leaf DP.
A DP that controls several FRs still occupies one column.
Retain exact cross-branch effects on these axes.
Do not expand parent material effects or parent zeros into leaf facts.
An apparent ancestor disagreement creates `unresolved_refinement`.
Report a contradiction only when an applicable outcome argument proves incompatible claims.

An outcome argument is a claim-specific support record.
It identifies parent and child criteria, the allocation rule, premises, operating range, and source revisions.
For version 1, record these arguments in a separate review document linked through `source` and `cites`.
No new core argument block is introduced.
A citation alone does not establish that an argument exists or is sound.

Distinguish provenance from support.
A source citation identifies where a statement came from.
Qualitative support requires an applicable analytical, contractual, or observed argument for the exact claim.
Empirical probability evidence remains subject to the separate trial contract.
A supported conclusion requires human or external review of substantive arguments.
The analyzer can report structural completeness without claiming that it proves argument truth.
Changed claims, premises, criteria, workloads, or cited revisions invalidate dependent support.
The source profile preserves support gaps as warnings.
The architecture profile requires complete criteria and contracts, but completeness alone does not establish support.

### Review input and supported conclusions

Version 1 defines review documents as provenance, not as trusted machine-readable attestations.
It defines no review importer, review signature, or field that marks an argument as accepted.
The core loader must not fetch a source URL or interpret cited prose as executable review input.
An implementation cannot infer substantive approval from a title, citation, `because`, or `status = "validated"`.

A review document must name the exact claim, criteria, premises, operating range, and revisions that it reviews.
For an influence, it must identify the source and target and the particular claimed effect.
For an assumption, it must explain why that premise holds within the named range.
For allocation, it must explain how the child predicates jointly establish the parent predicate.
These records let a person review an argument without turning the citation into automatic support.

With only core version 1 and its initial attachments, machine-assessed qualitative support remains unresolved.
Report `support_gap` for claims that need such support.
An implementation with no separately specified trusted review input cannot emit supported independence, supported allocation, or documentary feasibility success.
This restriction applies equally to both profiles.
The architecture profile requires complete criteria and module contracts, not proof that their claims are true.
A future review-input contract must define authority, claim matching, revision matching, and invalidation before it can promote these conclusions.

Qualified empirical evidence still produces trial estimates under the evidence rules.
It does not by itself establish a causal zero, validate a premise, or authorize pooling.
Without an accepted review-input contract, keep separate evidence blocks unpooled.
An observed criterion failure can establish infeasibility without establishing qualitative independence.

### Expectation results

An expectation tests its named sibling scope, not the whole design.
Its `under` field selects the applicability for that assertion only.
It does not change another report's requested snapshot or the candidate's operating conditions.
An assertion cannot select conditions incompatible with its candidate.
An omitted `basis` means `source`.

Use the following basis rules:

| Basis | Required result |
|---|---|
| `source` | The applicable authored effects establish the asserted structure. Assumed or unevaluated premises do not fail this basis. Invalidated premises still disable their effects. Proposed control links alone do not supply diagonal effects or missing zeros. |
| `conditional` | The applicable effects establish the structure with every relied-on premise stated. Assumed or unevaluated premises remain named conditions. |
| `supported` | The structure and every relied-on effect, premise, and allocation have applicable accepted support. |

All bases retain exact unknowns and conflicts.
Neither `source` nor `conditional` permits replacing a missing effect with a pairing.
Report `failed_expectation` when the asserted classification differs or its requested basis cannot be established.
A conflict cannot satisfy a classification assertion.
An unknown result can satisfy only an explicit `classification = "unresolved"` assertion without a conflict.
A supported assertion still requires accepted support for the claims it makes.
Thus changing its classification to unresolved does not turn absent review input into support.

When `order` is present, require every active DP of the asserted sibling scope exactly once.
Reject an extra, missing, inactive, or repeated DP.
Require each known precedence edge to run forward in that order.
An order cannot pass as a complete design sequence while unknown cells leave precedence unresolved.
Report the author's valid order even when the display order differs.

### Profile completeness

Assess active requirements and realizations, not inactive alternatives.
A criterion is operational only when `metric`, `op`, and `value` form a valid schema predicate.
Every active FR needs at least one applicable operational criterion in the architecture profile.
Qualitative criteria remain visible as `incomplete_criterion` findings.
An unresolved operating condition cannot count as applicable criterion coverage.

Every active leaf DP needs exactly one realization module.
That module must declare `inputs`, `outputs`, `owns_state`, `dependencies`, `invariants`, `failure_behavior`, and `acceptance_checks`.
Explicit empty input and output lists are permitted for responsibilities with no such exchange.
Explicit empty state and dependency lists are permitted.
Require nonempty `invariants` and `acceptance_checks`, and a nonblank failure contract.
Each active realized leaf criterion must occur in its module's checks, directly or through a referenced scenario's `verifies`.
Composite modules do not replace leaf contracts.
Report `incomplete_module_contract` when this coverage is absent.

Each active leaf FR needs an applicable scenario that verifies all its applicable criteria.
Require a nonempty `steps` list for that scenario.
One scenario can cover several leaves if it names all their criteria.
An empty scenario declaration does not satisfy coverage.
Report `missing_leaf_scenario` for a leaf without this coverage.

When a residual material cycle requires iteration, require an applicable `coupling` record for that scope.
Its `affects` must include every paired FR and DP on the reported cycle.
Require `iteration_bound`, `convergence`, `stopping`, and `failure_behavior`.
A runtime flow loop alone does not satisfy these fields.
Report `incomplete_feedback_contract` when the record or any required field is absent.
Do not infer a material cycle from a runtime loop alone.

All these completeness findings are warnings in the source profile and errors in the architecture profile.
Neither profile treats complete declarations as evidence of a successful run.

## Feasibility and selection

Assess each candidate before comparing information content.
Report `infeasible` when a required constraint or criterion demonstrably fails.
Report `unresolved` when feasibility lacks evidence, coverage, or an operational criterion.
Report `feasible` only when all required constraints and criteria have applicable success evidence.
List the reasons and affected references beside the result.
An unresolved independence result blocks a supported Information Axiom recommendation.

Compare candidates against identical required FRs, criteria, contexts, workloads, and horizons.
Different acceptance ranges define different comparison problems.
Keep cost and user preferences separate from information content.
A selected candidate can remain provisional while evidence is missing.

Trials must be positive integers.
Failures must be integers from zero through trials.
Confidence must be strictly between zero and one.
Use the two-sided Wilson score interval without continuity correction.
Let n be trials, p the observed success fraction, and z the standard normal quantile at (1 + confidence) / 2.
The center is (p + z²/(2n)) / (1 + z²/n).
The half-width is z sqrt(p(1-p)/n + z²/(4n²)) / (1 + z²/n).
Clip bounds to [0,1] only for numerical roundoff.
All-success and all-failure samples retain their Wilson uncertainty.
Do not pool separate evidence blocks automatically.
Report each compatible block separately unless a reviewed joint sampling argument authorizes pooling.
In machine-readable output, represent infinite information as null with `status = "infinite"`.
Represent missing information as null with `status = "unknown"`.
Never emit nonfinite JSON numbers.

Version 1 intentionally limits numerical probability estimation to qualified joint binomial trials.
It does not accept an authored probability or an analytical distribution as a substitute for those trials.
Numerical bounds alone do not prove that one candidate is better.
In particular, overlapping marginal intervals do not establish a decisive ranking.
Without an accepted comparison argument, report insufficient evidence to rank and retain provisional selection.

## Identity and operation results

Core-only input selects core version 1.
Future core versions require an explicit manifest selection.
Record source identity separately from semantic identity.
Source identity includes relative paths and exact source bytes.
Semantic identity includes admitted values, typed references, and versioned contracts.
Exclude whitespace, comments, file order, and source positions from semantic identity.
A file move changes source identity but preserves semantic identity when references still resolve identically.
A semantic change invalidates dependent assessments and evidence applicability.
Do not specify a hash algorithm until the machine-readable interchange contract defines canonical encoding.

Publish separate statuses for core assessment, each owner, composition, and rendering.
An owner failure cannot erase independent core findings.
Missing external observations produce unknown owner results.
Invalid required attachments make the requested project operation fail.
The selected profile determines diagnostic severity, not the truth of a claim.
No report can promote conditional support to established independence.
