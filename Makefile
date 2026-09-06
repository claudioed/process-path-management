# Makefile — the local quality gate for process-path-management.
#
# Every target below mirrors a sensor in .github/workflows/ci.yml, so the
# same feedback CI gives you post-push is available locally, pre-commit.

GO                 ?= go
GOLANGCI_LINT      ?= golangci-lint
GOLANGCI_VERSION   := v2.13.1

COVERAGE_OUT       := coverage.out
COVERAGE_PKGS      := ./internal/domain/...,./internal/application/...
COVERAGE_THRESHOLD := 90

.DEFAULT_GOAL := help

.PHONY: help build vet fmt fmt-check lint test coverage check check-all mutation

help:
	@echo "process-path-management — local quality gate (targets mirror .github/workflows/ci.yml)"
	@echo ""
	@echo "  help              Print this list of targets (default target)"
	@echo "  build             go build ./..."
	@echo "  vet               go vet ./..."
	@echo "  fmt               gofmt -w . — format the tree in place"
	@echo "  fmt-check         Fail if gofmt -l . is non-empty (the CI-style check)"
	@echo "  lint              golangci-lint run ./... (pinned $(GOLANGCI_VERSION) in CI)"
	@echo "  test              go test ./... -race"
	@echo "  coverage          CI coverage command + the $(COVERAGE_THRESHOLD)% gate"
	@echo "  bdd               go test ./... -run TestFeatures -v (godog/Gherkin)"
	@echo "  arch-test         go test ./internal/architecture/... -v"
	@echo "  integration       go test -tags=integration ./... -race -count=1 (needs DATABASE_URL)"
	@echo "  mutation          gremlins unleash ./internal/domain (see .gremlins.yaml)"
	@echo ""
	@echo "  check             FAST bundle: fmt-check vet build lint test"
	@echo "  check-all         check + coverage — run this before pushing"

build:
	$(GO) build ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

fmt-check:
	@files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then \
		echo "gofmt: the following files are not formatted:"; \
		echo "$$files" | sed 's/^/  /'; \
		echo "run 'make fmt' to fix them"; \
		exit 1; \
	fi; \
	echo "gofmt: clean"

lint:
	@if ! command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		echo "golangci-lint is not installed (or not on PATH)."; \
		echo "Install the exact version CI pins:"; \
		echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)"; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run ./...

test:
	$(GO) test ./... -race

coverage:
	$(GO) test ./... -race -coverprofile=$(COVERAGE_OUT) -coverpkg=$(COVERAGE_PKGS)
	@COVERAGE=$$($(GO) tool cover -func=$(COVERAGE_OUT) | awk '/^total:/ {print $$3}' | tr -d '%'); \
	echo "Coverage: $${COVERAGE}% (gate: $(COVERAGE_THRESHOLD)%)"; \
	if awk -v c="$$COVERAGE" -v t="$(COVERAGE_THRESHOLD)" 'BEGIN { exit !(c < t) }'; then \
		echo "coverage $${COVERAGE}% is below the $(COVERAGE_THRESHOLD)% gate"; \
		exit 1; \
	fi

bdd:
	$(GO) test ./... -run TestFeatures -v

arch-test:
	$(GO) test ./internal/architecture/... -v

# Requires a running Postgres: docker compose up -d postgres, and
# DATABASE_URL pointed at it.
integration:
	$(GO) test -tags=integration ./... -race -count=1

mutation:
	$(GO) run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0 unleash ./internal/domain --workers 1 --timeout-coefficient 30

# The fast self-correction loop: run this after every change, before committing.
check: fmt-check vet build lint test

# The fuller gate a human runs before pushing.
check-all: check coverage
