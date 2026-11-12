# Pinned language and conformance assets

The specification snapshot under `testdata/spec` comes from the committed sibling
`spec` repository. The snapshot includes the files listed below. The corpus records
expected behavior. It does not report measured validator results.

## Specification provenance

The pinned specification revision is
`01c50bf3d141624c8f541a4797f0224b7658d003` from the sibling `spec` repository.
It includes the schema and diagnostic contracts committed in
`402132b98d24c4fbc24a044590c7432b9ce42fa3` and the source assertion clarification.
Every copied file matches its path at the pinned revision.

### Expected snapshot manifest

Specification documents copied separately:

- `testdata/spec/docs/language.md`
- `testdata/spec/docs/assessment-contract.md`
- `testdata/spec/docs/schema.md`
- `testdata/spec/docs/diagnostics.md`
- `testdata/spec/docs/review-2026-11-11.md`
- `testdata/spec/LICENSE`

Conformance fixtures:

- `testdata/spec/conformance/core-1/README.md`
- `testdata/spec/conformance/core-1/expected.json`
- `testdata/spec/conformance/core-1/self-design-inspection.json`
- `testdata/spec/conformance/core-1/ambiguous.ad.hcl`
- `testdata/spec/conformance/core-1/base.ad.hcl`
- `testdata/spec/conformance/core-1/duplicate.ad.hcl`
- `testdata/spec/conformance/core-1/duplicate.ad.project.hcl`
- `testdata/spec/conformance/core-1/exact-unknown.ad.hcl`
- `testdata/spec/conformance/core-1/exact-unknown.ad.project.hcl`
- `testdata/spec/conformance/core-1/invalid-attachment.ad.project.hcl`
- `testdata/spec/conformance/core-1/invalid-fields.ad.hcl`
- `testdata/spec/conformance/core-1/invalid-schema.ad.project.hcl`
- `testdata/spec/conformance/core-1/invalid-view.ad.ext.hcl`
- `testdata/spec/conformance/core-1/wrong-kind.ad.hcl`
- `testdata/spec/conformance/core-1/wrong-kind.ad.project.hcl`

Examples:

- `testdata/spec/examples/requirements_hcl.ad.hcl`
- `testdata/spec/examples/starter/README.md`
- `testdata/spec/examples/starter/design.ad.hcl`
- `testdata/spec/examples/starter/bindings.ad.ext.hcl`
- `testdata/spec/examples/starter/behavior.ad.ext.hcl`
- `testdata/spec/examples/starter/views.ad.ext.hcl`
- `testdata/spec/examples/starter/model.ad.project.hcl`

## Authority and fixture roles

`language.md` defines syntax and core language behavior. `schema.md` defines
field rules. `assessment-contract.md` defines assessment behavior and takes
precedence when assessment rules differ. `diagnostics.md` defines diagnostic
codes and profile outcomes. These files are the normative authorities.

`review-2026-11-11.md` records review findings. It is supporting context, not a
replacement for normative rules. `LICENSE` retains the source repository's
license.

The `core-1` directory contains conformance inputs and authored expectations.
Resolve each `input` in `expected.json` relative to that file's directory. Keep
the directory under `testdata/spec` so authority paths such as
`../../docs/schema.md` resolve. The starter example remains intact at
`testdata/spec/examples/starter/` so manifest-relative paths resolve.

`self-design-inspection.json` records source-inspection findings for
`examples/requirements_hcl.ad.hcl`. It is a separate inspection record, not a
conformance oracle. Do not edit authored expectations to fit an implementation.
The conformance README defines case matching and scope.
