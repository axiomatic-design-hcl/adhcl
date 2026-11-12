# Core version 1 conformance cases

These assets help decoder and assessment implementers test `requirements_hcl/core/1`.
They contain proposed acceptance expectations, not measured semantic results.
There is no RequirementsHCL validator in this repository.
Native HCL parsing alone cannot establish these expectations.

The authorities are the [field schema](../../docs/schema.md),
[assessment contract](../../docs/assessment-contract.md), and
[diagnostic contract](../../docs/diagnostics.md).
The JSON fixture format is `requirements_hcl/conformance/1`.
It is test data, not the tool's report interchange format.
When you copy these assets into an implementation, pin the complete specification
checkout revision with them. The directory name pins the core language version,
not a Git revision or a semantic test result.

## Select a case

Use the exact `input` from [expected.json](expected.json).
Resolve that path relative to this directory.
Do not assess this directory as one merged core model: its cases are mutually
exclusive inputs. Every manifest selects only contained files and names every
attachment explicitly.

The cases cover these decisions:

| Case | Required behavior |
|---|---|
| `baseline` | Derive a complete proposed diagonal matrix without claiming reviewed support. |
| `invalid_schema` | Reject an unknown field, null, an empty required list, and a reference-shaped string. |
| `wrong_reference_kind` | Reject a need where an FR traversal is required. |
| `cross_file_duplicate` | Retain both duplicate label locations; never let file order select a winner. |
| `exact_unknown_over_default` | Keep an exact unknown instead of applying the default zero. |
| `ambiguous_pairing` | Report both possible pairings without choosing either. |
| `invalid_required_attachment` | Reject a false `show_source_locations`; retain the independent core matrix. |
| `starter_core` | Derive the complete proposed triangular receipt design. |
| `starter_project_external_unknown` | Assess all three payloads; keep binding and behavior outcomes unknown without external inputs. |

Every HCL file is intended to parse as native HCL, including the negative cases.
Their errors concern the language schema or assessment rules, not malformed HCL.
The duplicate label is deliberately in another file.
Schema errors, reference errors, and duplicate identity errors use separate
manifests so that one failed prerequisite cannot hide another case's finding.

## Read the expected data

Each case declares both profile outcomes and separate stage statuses.
`complete` means processing finished, not that the operation succeeded.
A completed assessment can contain errors.
A blocked core assessment has `conditional_classification = "not_assessed"` in
these test records; that marker is not a fourth AD classification.
A completed but unresolved classification uses `unresolved`.
Text delivery is assumed available; HTML rendering is not requested.

Use these matching rules:

- Match each required finding by its code and supplied subject or source locator.
  A locator names a block and field rather than a line that changes with formatting.
- `related_files` requires the duplicate finding to retain both declarations.
- Match severity separately for each profile. Message wording and diagnostic
  grouping are not pinned; one finding can retain several affected locations.
- Additional findings may use only `permitted_additional_codes`.
  In `invalid_schema`, additional structure findings must concern the listed
  invalid fields, not invented downstream graph or pairing errors.
- `support_gap` and `unresolved_feasibility` can appear for any affected claim
  or candidate. Their number is not fixed, but their catalog severity is.
- Any other code fails the case. No permitted warning can change a case's
  required operation or stage outcome.

The `matrix` fields give complete resolved cells for the stated scope.
The base fixture has the same leaf and sibling axes, so its master cells agree.
The ambiguity case pins the two possible matchings, not an invented chosen order.
The exact-unknown case requires all three other cells to remain unchanged.

The assumed registry contains the three fixed owners.
No observation set, exact external behavior revision, or trusted qualitative
review import is available.
An owner therefore cannot fabricate `match` or `resolved`.
The view payload can be valid without an HTML render.
Supported independence, implementation conformance, feasibility, and information
remain separate from a conditional structural classification.
No fixture supplies empirical trials or establishes a success probability.

## Inspect the self-design separately

[self-design-inspection.json](self-design-inspection.json) records expected gaps
from source inspection of the existing language self-design.
It is not a complete conformance oracle or an observed validator report.
It preserves the authored criteria and conclusions rather than changing the
example to make an assertion pass.
