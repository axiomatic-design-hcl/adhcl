# Receipt design starter

Use this project as a small example of a complete proposed architecture:
core facts, an explicit manifest, and one attachment for each initial owner.
It describes a design; it contains no receipt implementation or measured trials.

Select `design.ad.hcl` for a core-only assessment, or
`model.ad.project.hcl` for explicit project processing.
The manifest lists only files inside this directory.
No executable or validator command is implemented in this repository.

## Proposed behavior

The admission module accepts exactly integer amounts from 0 through 100000 cents.
The total module adds an integer fee from 0 through 1000 cents, or propagates
`invalid_amount` without producing a total.
Each leaf has an operational criterion, a full module contract, a proposed
realization, and an operational scenario.
Scenario instructions are acceptance checks, not observations.
A change scenario varies `exact_addition` while requiring admission to remain
unchanged under the fixed contract.
A failure scenario requires a rejected amount to propagate as `invalid_amount`
without a fabricated zero or fee-only total.
Neither scenario records runtime execution.

The conditional matrix has FR rows and DP columns:

| FR / DP | Amount admission | Exact addition |
|---|---|---|
| Admit amount | Material | None |
| Calculate total | Material | Material |

Establish `amount_admission` before `exact_addition`.
After an admission change, recompute the total acceptance checks.
The zero depends on an immutable admission result and a fixed semantic contract.
Runtime order is also admission then addition because the total module consumes
that result, not merely because the design matrix is triangular.
Shared process failure and resource exhaustion are outside this change range.

This is a proposed decoupled mapping, not supported independence.
The premise remains `to_validate`; no trusted reviewed argument or joint empirical
evidence is supplied. Feasibility and implementation conformance remain unresolved,
and information content remains unknown.
The selected candidate is explicitly provisional.

## Attachment boundaries

- `bindings.ad.ext.hcl` names a proposed Terraform configuration address.
  The core PV does not duplicate that address.
  Without caller-supplied observations, the binding outcome is `unknown`.
- `behavior.ad.ext.hcl` names a proposed external slice at one exact revision.
  No external model is bundled or resolved here, so its outcome is `unknown`.
- `views.ad.ext.hcl` selects a permitted layout and preserves source locations.
  Validating its payload does not render HTML.

The attachment identifiers are design declarations, not claims that external
resources or models exist.
Owner results cannot change the core cells.
See the [conformance expectations](../../conformance/core-1/expected.json) for
separate source and architecture profile outcomes, and the
[field schema](../../docs/schema.md) for the authoritative field contracts.
