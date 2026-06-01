SHELL := /bin/bash

GOFMT ?= gofmt
GOLANGCI_LINT ?= golangci-lint
GITLEAKS ?= gitleaks
GOVULNCHECK ?= govulncheck
COVERAGE_MIN ?= 90

LEFTHOOK_VERSION ?= 1.7.10
LEFTHOOK_DIR ?= $(CURDIR)/.bin
LEFTHOOK_BIN ?= $(LEFTHOOK_DIR)/lefthook

MUTATION_PACKAGES ?= ./internal/...
MUTATION_THRESHOLD ?= 60

CONTAINER_COMMAND ?= podman
IMAGE_TAG ?= local
IMAGE_NAME ?= ffreis-latex-compiler


.PHONY: mutation-test help \
	fmt fmt-check lint validate test test-race coverage-gate quality-gates \
	hook-generated-drift secrets-scan-staged \
	lefthook-bootstrap lefthook-install lefthook-run lefthook setup \

	container-build docker-build \

	install build build-native validate-articles promote doctor \
	ci-list install-act ci-local

## mutation-test: run mutation testing with gremlins (slow — CI only)
mutation-test:
	@which gremlins >/dev/null 2>&1 || go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
	gremlins unleash --threshold-efficacy $(MUTATION_THRESHOLD) $(MUTATION_PACKAGES)

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "; printf "Targets:\n"} /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format all Go files in place
	$(GOFMT) -w .

fmt-check: ## Fail if Go files are not gofmt-formatted
	@./scripts/hooks/check_required_tools.sh $(GOFMT)
	@out="$$(find . -type f -name '*.go' -not -path './vendor/*' -not -path './.git/*' -print0 | xargs -0 -r $(GOFMT) -l)"; \
	if [ -n "$$out" ]; then \
		echo "Unformatted Go files:"; \
		echo "$$out"; \
		echo "Run: $(GOFMT) -w <files>"; \
		exit 1; \
	fi

lint: ## Run golangci-lint
	@./scripts/hooks/check_required_tools.sh $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

validate: ## Static analysis and compilation check (go vet + build)
	go vet ./...
	go build -o /dev/null ./...

test: ## Run unit tests (race + shuffle per workspace Go invariant)
	go test -race -shuffle=on ./...

test-race: ## Run tests with race detector
	go test -race ./...

coverage-gate: ## Run tests with coverage and fail if below COVERAGE_MIN
	@COVERAGE_MIN="$(COVERAGE_MIN)" ./scripts/hooks/check_coverage_gate.sh

quality-gates: ## Run strict pre-push quality gates (test + race + coverage + govulncheck)
	@./scripts/hooks/check_required_tools.sh $(GOVULNCHECK)
	$(MAKE) test
	$(MAKE) test-race
	$(MAKE) coverage-gate
	$(GOVULNCHECK) ./...

hook-generated-drift: ## Run generate target if present and fail on drift
	@set -euo pipefail; \
	if $(MAKE) -n generate >/dev/null 2>&1; then \
		$(MAKE) generate; \
		if ! git diff --quiet -- .; then \
			echo "Generated files are out of date. Run 'make generate' and commit updates."; \
			git status --short; \
			exit 1; \
		fi; \
	else \
		echo "No 'generate' target found; skipping generated drift check."; \
	fi

secrets-scan-staged: ## Scan staged diff for secrets
	@./scripts/hooks/check_required_tools.sh $(GITLEAKS)
	$(GITLEAKS) protect --staged --redact


container-build: ## Build container image
	$(CONTAINER_COMMAND) build -t "$(IMAGE_NAME):$(IMAGE_TAG)" -f containers/Dockerfile.cli .

docker-build: container-build ## Backward-compatible alias


# ── Compiler usage targets ───────────────────────────────────────────────────
# `build` runs the full LaTeX toolchain inside the container image (tectonic +
# tex4ht + pandoc), so nothing needs to be installed on the host beyond podman.
# `promote`/`doctor`/`validate-articles` are pure Go (no LaTeX tools) and run
# natively via `go run`.
ARTICLES_ROOT ?= ../ffreis-articles
SNIPPETS_ROOT ?= ../ffreis-snippets
POSTS_DIR     ?= ../ffreis-posts
OUT           ?= dist
SLUG          ?=
FORMATS       ?= pdf,html,md
GO_PKG        := ./cmd/ffreis-latex-compiler
SLUG_FLAG     := $(if $(SLUG),-slug $(SLUG),)

install: ## Install the compiler binary into GOPATH/bin
	go install $(GO_PKG)

build: container-build ## Compile article(s) via the toolchain container -> OUT/ (SLUG=… optional)
	@mkdir -p "$(OUT)" "$(HOME)/.cache/tectonic"
	$(CONTAINER_COMMAND) run --rm \
		-v "$(abspath $(OUT))":/work/out \
		-v "$(abspath $(ARTICLES_ROOT))":/work/articles:ro \
		-v "$(abspath $(SNIPPETS_ROOT))":/work/snippets:ro \
		-v "$(HOME)/.cache/tectonic":/root/.cache/Tectonic \
		"$(IMAGE_NAME):$(IMAGE_TAG)" \
		build -articles-root /work/articles -snippets-root /work/snippets \
		-out /work/out $(SLUG_FLAG) -formats $(FORMATS)

build-native: ## Compile natively (requires tectonic, make4ht, pandoc on PATH)
	go run $(GO_PKG) build -articles-root "$(ARTICLES_ROOT)" -snippets-root "$(SNIPPETS_ROOT)" \
		-out "$(OUT)" $(SLUG_FLAG) -formats $(FORMATS)

validate-articles: ## Validate article sources (no LaTeX tools needed)
	go run $(GO_PKG) validate -articles-root "$(ARTICLES_ROOT)" -snippets-root "$(SNIPPETS_ROOT)" $(SLUG_FLAG)

promote: ## Stage a compiled article into POSTS_DIR (SLUG=… ; OPEN_PR=1 to open a PR; DRY_RUN=1 to preview)
	go run $(GO_PKG) promote -articles-root "$(ARTICLES_ROOT)" -out "$(OUT)" -posts-dir "$(POSTS_DIR)" \
		-slug "$(SLUG)" $(if $(filter 1,$(OPEN_PR)),-open-pr,) $(if $(filter 1,$(DRY_RUN)),-dry-run,)

doctor: ## Report toolchain availability (tectonic, make4ht, pandoc)
	go run $(GO_PKG) doctor

lefthook-bootstrap: ## Download lefthook binary into ./.bin
	LEFTHOOK_VERSION="$(LEFTHOOK_VERSION)" BIN_DIR="$(LEFTHOOK_DIR)" bash ./scripts/bootstrap_lefthook.sh

lefthook-install: lefthook-bootstrap ## Install git hooks if missing
	@if [ -x "$(LEFTHOOK_BIN)" ] && [ -x ".git/hooks/pre-commit" ] && [ -x ".git/hooks/pre-push" ] && [ -x ".git/hooks/commit-msg" ]; then \
		echo "lefthook hooks already installed"; \
		exit 0; \
	fi
	LEFTHOOK="$(LEFTHOOK_BIN)" "$(LEFTHOOK_BIN)" install

lefthook-run: lefthook-bootstrap ## Run hooks (pre-commit + commit-msg + pre-push)
	LEFTHOOK="$(LEFTHOOK_BIN)" "$(LEFTHOOK_BIN)" run pre-commit
	@tmp_msg="$$(mktemp)"; \
	echo "chore(hooks): validate commit-msg hook" > "$$tmp_msg"; \
	LEFTHOOK="$(LEFTHOOK_BIN)" "$(LEFTHOOK_BIN)" run commit-msg -- "$$tmp_msg"; \
	rm -f "$$tmp_msg"
	LEFTHOOK="$(LEFTHOOK_BIN)" "$(LEFTHOOK_BIN)" run pre-push

lefthook: lefthook-bootstrap lefthook-install lefthook-run ## Install hooks and run them

setup: lefthook ## Install hooks and verify dev tools
	@./scripts/hooks/check_required_tools.sh $(GOLANGCI_LINT) $(GITLEAKS) $(GOVULNCHECK) || true

ci-list: ## List local CI workflows
	@ls -1 .github/workflows | sort

# ── Local CI (act-based fallback when GH Actions quota is hit) ───────────────
PLATFORM_STANDARDS_SHA ?= 3c787edb4e96ddea2e86b2add2c32139685e8db7  # v1.2.1
PLATFORM_STANDARDS_RAW ?= https://raw.githubusercontent.com/FelipeFuhr/ffreis-platform-standards

install-act: ## Download pinned act binary into .bin/
	@mkdir -p scripts
	@curl -fsSL "$(PLATFORM_STANDARDS_RAW)/$(PLATFORM_STANDARDS_SHA)/scripts/install_act.sh" \
		-o scripts/install_act.sh && chmod +x scripts/install_act.sh
	@bash ./scripts/install_act.sh

ci-local: ## Run workflows locally via act (GH Actions quota fallback). Args via ARGS=...
	@mkdir -p scripts
	@curl -fsSL "$(PLATFORM_STANDARDS_RAW)/$(PLATFORM_STANDARDS_SHA)/scripts/run-ci-local.sh" \
		-o scripts/run-ci-local.sh && chmod +x scripts/run-ci-local.sh
	@PATH="$(CURDIR)/.bin:$(PATH)" bash ./scripts/run-ci-local.sh $(ARGS)
