# Changelog

All notable changes to this project are documented in this file. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## v0.2.1

### Changed

- Building from source now requires Go 1.26 or later. CI verifies against Go 1.26.x and 1.27.x instead of the unsupported Go 1.25 series.

### Security

- Use Go 1.26.9 or later to build binaries with fixes for the `html/template` vulnerabilities GO-2026-6599 and GO-2026-6600. Go 1.25 has no fixes for these vulnerabilities. The release toolchain is Go 1.26.11.

## v0.2.0

### Added

- Add `adhcl diagram`. The command runs the same assessment as `adhcl validate` and writes one self-contained HTML page. It accepts the same `--profile`, `--candidate`, `--context`, and `--` options.
- The command writes the page to stdout. Use `-o FILE`, `--output FILE`, or `--output=FILE` to write it to a file. A blocked assessment writes its report to stderr and does not change the output file.
- Exit code 0 means that the command delivered the page, also when the page shows error findings. Exit code 1 means that the core assessment was blocked or the delivery failed. Use `adhcl validate` as the gate in CI.
- The page has one tab for each reader: start, business view, product view, architecture view, engineering view, trace and impact, risks and evidence, and a guide to the method.
- The business and product views list the needs, constraints, stakeholders, capabilities, FRs, acceptance criteria, and scenarios. FRs and DPs have hierarchy numbers such as `1.2.1`, and the alias from the source document.
- The architecture view shows each design matrix with its classification, design order, and expectation. A cell off the diagonal has its own color. A table under each matrix gives the reason for each cell and marks it as stated in the source or inferred.
- The engineering view lists the process variables, the module contracts, the flows, and the module junctions. A module junction is S, C, or F. The page derives it from the classification of the matrix.
- The trace view has a trace map, a change impact table, and a trace table from each need to its leaf FRs, DPs, modules, and scenarios.
- The risks view lists the open items, the design risks, the assumptions, the evidence, and the findings grouped by code. The guide gives a meaning and an action for each finding code.
- A child FR without `need` takes the needs of its parent FRs. The lists, the trace map, and the change impact use this rule from the schema. Change impact also lists the scenarios that test a criterion of a parent FR.
- Each record has one home row on the page. Every other mention of the record links to that row.
- Without JavaScript, the page shows all tabs in order. When you print the page, each tab starts on a new page.
- Add `examples/architecture-diagram.html`, made from `examples/architecture.ad.hcl`. The `make diagram-example-check` target makes sure that the committed page matches the output of the command.
- The `validate-examples` target also makes a diagram from each example.

## v0.1.1

### Added

- Add a sample architecture diagram and a plan for a future `adhcl diagram` command.

### Changed

- `validate.Result.Snapshot` gives the selected design when the core assessment completes. It is nil when the assessment is blocked.

## v0.1.0

### Fixed

- Address implementation-review findings R1–R15. Ambiguous control matching retains at most two witness pairings, and matrices retain sparse facts and scoped defaults without allocating implicit unknown cells.
- Enforce logical leaf state ownership, nonempty cell scopes, exact candidate evidence references, cross-decision DP separation, and attachment label uniqueness even when envelopes or core resolution fail.
- Qualify measured estimates against explicit information requests, applicable scenarios, and complete operational predicate coverage. Observed required failures produce `infeasible_candidate`; finite all-success samples retain uncertainty.
- Preserve independent candidate estimates and their sampling, confidence, exposure, and information bounds through composition and text output.
- Retain qualitative criterion and shared-PV review gaps. Keep unsupported structural results and other unresolved obligations visible in the summary.
- Include owner contracts, implementation identity, source ranges, core and attachment revisions, snapshot identity, and invocation input identity in owner output.

### Changed

- `validate.Run(validate.Request)` returns a composed `validate.Result` without a writer or CLI dependency. The command adapter renders the result and translates operation failure into a process exit code.

### Added

- Add `adhcl version`. It prints `adhcl dev` for development builds and the version that the linker injects for release builds.
- Add `adhcl validate` with `--profile source|architecture`, `--candidate REF`, `--context REF`, and `--` to end option parsing.
- Add command help through `adhcl --help`, `adhcl -h`, and `validate --help` or `version --help` when given as the only argument.
- Add exit code 2 for invalid command lines.
- Add input selection for one `.ad.hcl` file, a directory of immediate `*.ad.hcl` files, and an explicit `.ad.project.hcl` manifest. Symbolic links resolve before identity and containment checks.
- Add syntax admission, schema decoding, and graph resolution for core files.
- Add snapshot selection for candidates and contexts.
- Add core assessment. It covers sibling-set classification, cell records, hierarchy refinement, expectations, support gaps, candidate evidence bounds, and profile completeness.
- Add attachment owners `terraform_binding`, `event_model_link`, and `html_view`. Binding and behavior results stay unknown when no observations or behavior models are supplied.
- Add the text report on stdout. It lists findings, scopes, owners, and a summary. Exit code 0 means validation completed without error findings. Exit code 1 means an input, parse, assessment, required attachment, or output step failed.
- Add the `source` and `architecture` profiles. The profiles change finding severity only.
- Add the examples `examples/minimal.ad.hcl`, `examples/architecture.ad.hcl`, and `examples/project/model.ad.project.hcl`.
- Pin the core language schema, diagnostic contract, and conformance cases to spec revision `01c50bf3d141624c8f541a4797f0224b7658d003` under `testdata/spec`.
- Add the Makefile verification targets, the CI workflows, and the GoReleaser release configuration.

### Security

- Update the development, CI, and release toolchain to Go 1.26.11 to fix GO-2026-6859.
