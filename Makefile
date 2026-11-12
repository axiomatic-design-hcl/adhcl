GO ?= go
BINARY := bin/adhcl

.DEFAULT_GOAL := help

.PHONY: help build test test-race vet fmt-check tidy-check staticcheck vulncheck validate-examples release-tag-check release-annotation-check release-version-check verify clean

help: ## List available commands.
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "%-28s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: $(BINARY) ## Build the validator into bin/.

$(BINARY): $(shell find cmd internal -type f -name '*.go') go.mod go.sum
	@mkdir -p $(@D)
	$(GO) build -o $@ ./cmd/adhcl

test: ## Run unit and fixture tests.
	$(GO) test ./...

test-race: ## Run tests with the race detector.
	$(GO) test -race ./...

vet: ## Run go vet.
	$(GO) vet ./...

fmt-check: ## Fail if tracked Go files are not gofmt-formatted.
	@unformatted=$$($(GO)fmt -l $$(git ls-files -co --exclude-standard '*.go')); \
	test -z "$$unformatted" || { printf '%s\n' "$$unformatted"; exit 1; }

tidy-check: ## Fail if go.mod or go.sum need tidying.
	$(GO) mod tidy -diff

staticcheck: ## Run Staticcheck.
	$(GO) run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...

vulncheck: ## Run the Go vulnerability checker.
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

validate-examples: build ## Validate the shipped core files and manifest.
	$(BINARY) validate examples/minimal.ad.hcl
	$(BINARY) validate --profile architecture examples/architecture.ad.hcl
	$(BINARY) validate examples/project/model.ad.project.hcl

release-tag-check: ## Test the stable semantic-version tag validator.
	sh scripts/test-require-stable-tag.sh

release-annotation-check: ## Test annotated-tag verification from a tagless checkout.
	sh scripts/test-require-annotated-tag.sh

release-version-check: ## Verify the linker-injected release version is reported by adhcl.
	@temporary=$$(mktemp -d); \
	trap 'rm -rf "$$temporary"' EXIT; \
	$(GO) build -ldflags '-X main.buildVersion=v0.1.0' -o "$$temporary/adhcl" ./cmd/adhcl; \
	test "$$($$temporary/adhcl version)" = 'adhcl v0.1.0'

verify: fmt-check tidy-check vet test test-race staticcheck vulncheck validate-examples release-tag-check release-annotation-check release-version-check ## Run the complete local verification suite.

clean: ## Remove locally built artifacts.
	rm -rf bin dist

# -- pre-commit --
.PHONY: pre-commit-install pre-commit-run

pre-commit-install: ## Install the repository pre-commit hook.
	pre-commit install

pre-commit-run: ## Run all repository pre-commit hooks.
	pre-commit run --all-files

