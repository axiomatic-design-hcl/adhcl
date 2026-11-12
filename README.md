# adhcl

adhcl validates authored design files written in the `.ad.hcl` language. It reads the input, checks it against the pinned language rules, runs the core assessment, and prints a text report. It does not render HTML.

Use adhcl to check that a design file parses, resolves, and passes the core assessment under a chosen profile.

## Requirements

Build adhcl with Go 1.25 or later. The `go.mod` file names `go1.26.10` as the toolchain. CI tests with Go 1.25.x and 1.26.10.

## Build

Run `make build` from the repository root. The command writes the binary to `bin/adhcl`.

```sh
make build
```

Run `make help` to list every Makefile target.

## Commands

```sh
adhcl version
adhcl validate [--profile source|architecture] [--candidate REF] [--context REF] [--] PATH
adhcl --help
adhcl -h
```

`adhcl version` prints the tool version. A development build prints `adhcl dev`. A release build prints the version that the linker injects. For example, a build with `-ldflags '-X main.buildVersion=v0.1.0'` prints `adhcl v0.1.0`.

`adhcl validate` checks one input path and writes a report to stdout.

Use these options with `validate`:

- `--profile source|architecture` selects the profile. The default is `source`. See [Profiles](#profiles).
- `--candidate REF` selects a candidate. The reference must match `decision.NAME.candidate.NAME`.
- `--context REF` selects a context. The reference must match `context.NAME`.
- `--` ends option parsing. Use it when the path starts with `-`.

Follow these rules for options:

- Put options before the path.
- Give exactly one path.
- Write a value as `--flag value` or `--flag=value`.
- A value cannot start with `-`.
- Repeating an option is an error.

Use `adhcl validate --help` or `adhcl version --help` to print help for one command. Give the help option as the only argument.

## Examples

Run these commands from the repository root:

```sh
adhcl validate examples/minimal.ad.hcl
adhcl validate --profile architecture examples/architecture.ad.hcl
adhcl validate examples/project/model.ad.project.hcl
```

See [examples/README.md](examples/README.md) for what each example shows.

## Input

The input path selects the design files:

- A `.ad.hcl` file selects that one file.
- A directory selects its immediate `*.ad.hcl` files. adhcl sorts the files by name and merges them. adhcl does not read subdirectories. A directory with no `*.ad.hcl` core files fails.
- A `.ad.project.hcl` manifest selects only the core and attachment files that it lists.

A file or directory input does not load attachments. Only a manifest loads attachments. Attachment files use the `.ad.ext.hcl` suffix.

adhcl resolves symbolic links before it checks file identity and location. It rejects these inputs:

- absolute paths in a manifest
- paths that escape the manifest directory
- two entries that name the same physical file
- listed files that do not exist
- files with the wrong suffix
- directories in a file list

## Profiles

The profile sets the severity of some findings. It does not change the claim that validation makes.

- `source` is the default profile. Under this profile, these codes are warnings: `incomplete_criterion`, `incomplete_module_contract`, `missing_leaf_scenario`, `incomplete_feedback_contract`, `unresolved_refinement`, `unresolved_applicability`, `unresolved_influence`, and `shared_resource_review_gap`.
- `architecture` makes the same codes errors.

Both profiles keep `support_gap`, `unresolved_feasibility`, `incomparable_evidence`, and `unavailable_external_input` as warnings.

## Owners

Owners check attachments. adhcl has three built-in owners. The manifest declares each owner in an `extension` block, and the extension label selects the built-in owner. Each attachment names its owner with an `extension.LABEL` reference.

| Owner | Role | Subject | Required fields |
| --- | --- | --- | --- |
| `terraform_binding` | binding | `pv` | `address` and `observation_source`, both non-blank strings |
| `event_model_link` | behavior link | `module` | `model`, `revision`, `target_kind` set to `slice`, and `target_label` |
| `html_view` | view | `design` | `layout` set to `requirements_first` or `matrices_first`, and `show_source_locations` set to `true` |

The binding and behavior results stay unknown. adhcl does not receive observations or behavior models. Each attachment for these owners reports the outcome `unknown` with a stated cause and gets an `unavailable_external_input` warning.

A valid `html_view` attachment reports `valid`. adhcl does not render the view.

## Output

The report goes to stdout. It has these sections, in this order:

1. A header with the report title, input, tool version, contract, profile, candidate, context, and workload.
2. Findings, each with its severity, code, location, and message.
3. Scopes, with the classification, pairing, order, and any cycles for each sibling set.
4. Owners, with the status and the outcome of each attachment.
5. A summary with the error count, the warning count, the stage statuses, and the unresolved causes.

The report closes by stating that exit code 0 means validation completed under the profile, not that the design is independent or implemented.

Usage errors and delivery failures go to stderr. A delivery failure looks like this: `adhcl: cannot deliver ...`.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Validation completed without error findings. |
| 1 | An input, parse, assessment, required attachment, or output step failed. |
| 2 | The command line is invalid. |

A zero exit code does not prove that the design is independent, implemented, or supported by observations.

## Language and conformance

The language rules come from revision `01c50bf3d141624c8f541a4797f0224b7658d003` of the sibling `spec` repository. The copied snapshot lives under `testdata/spec`, and that snapshot is the contract for this tool. See [docs/conformance/pin.md](docs/conformance/pin.md) for the pin details.

The diagnostic codes and their default severities come from `testdata/spec/docs/diagnostics.md`.

## Method

adhcl follows the design method in [docs/axiomatic-design.md](docs/axiomatic-design.md). That document describes the command contract and the attachment contract.

## Release

Release builds and archives come from CI with GoReleaser. You do not build release archives locally. See `.github/workflows/release.yml` and [CONTRIBUTING.md](CONTRIBUTING.md) for the release gates.

Release notes are in [CHANGELOG.md](CHANGELOG.md).
