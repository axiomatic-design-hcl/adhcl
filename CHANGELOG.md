# Changelog

All notable changes to this project are documented in this file. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
