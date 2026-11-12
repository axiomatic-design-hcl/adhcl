# Contributing

This guide covers local setup, the checks that CI runs, and the release path for adhcl.

## Requirements

Install Go 1.25 or later. CI runs the verification suite on Go `1.25.x` and `1.26.11`.

Run `make help` to list the Makefile targets.

## Local checks

Run `make verify` before you open a pull request. It runs these targets in this order:

1. `fmt-check` fails if tracked Go files are not gofmt-formatted.
2. `tidy-check` fails if `go.mod` or `go.sum` need tidying.
3. `vet` runs `go vet`.
4. `test` runs unit and fixture tests.
5. `test-race` runs tests with the race detector.
6. `staticcheck` runs Staticcheck at a pinned version.
7. `vulncheck` runs the Go vulnerability checker at a pinned version.
8. `validate-examples` builds the binary and validates the shipped examples.
9. `release-tag-check` tests the stable semantic-version tag validator.
10. `release-annotation-check` tests annotated-tag verification from a tagless checkout.
11. `release-version-check` builds with a linker-injected version and checks that `adhcl version` prints `adhcl v0.1.0`.

Run a single target when you change only part of the code. For example, run `make test` after a change to a single package.

## Other targets

- `make build` builds `bin/adhcl` from `./cmd/adhcl`.
- `make clean` removes the `bin` and `dist` directories.
- `make pre-commit-install` installs the repository pre-commit hook with `pre-commit install`.
- `make pre-commit-run` runs all pre-commit hooks with `pre-commit run --all-files`.

The pre-commit targets need the `pre-commit` tool on your path.

## Test first

Write a behavior test before you write production code. Each test uses separate `// Given`, `// When`, and `// Then` comments that describe the setup, the action, and the expected result.

Run the test and confirm that it fails for the reason you expect before you write the fix.

## Language and conformance

The language rules come from the pinned spec snapshot under `testdata/spec`. See [docs/conformance/pin.md](docs/conformance/pin.md) for the pinned revision. Do not change the snapshot in this repository to fit a local change. Make language changes in the spec repository and update the pin.

## Changelog

Add a line to [CHANGELOG.md](CHANGELOG.md) for each user-visible change.

## Continuous integration

CI runs in `.github/workflows/ci.yml` on pull requests and on pushes to `main`.

- The `verify` job runs `make verify` on Go `1.25.x` and `1.26.11`. It sets `GOTOOLCHAIN` to `local`.
- The `build` job runs `go build ./cmd/adhcl` for these platforms on Go `1.26.11`: linux, darwin, and windows, each on amd64 and arm64.

## Releases

The release workflow in `.github/workflows/release.yml` runs when a tag that matches `v*` is pushed. The workflow does these steps in order:

1. It checks that the tag is annotated with `scripts/require-annotated-tag.sh`.
2. It checks that the tag is a stable semantic version with `scripts/require-stable-tag.sh`.
3. It runs `make verify`.
4. It runs GoReleaser with `release --clean`.
5. It attests the archives in `dist` and `dist/checksums.txt`.

GoReleaser, configured in `.goreleaser.yaml`, builds linux, darwin, and windows for amd64 and arm64. It disables cgo. It injects the tag as `main.buildVersion`. It writes `tar.gz` archives, and `zip` archives for windows. Each archive includes `LICENSE` and `README.md`.

CI produces the release archives. Do not build release archives on your machine.
