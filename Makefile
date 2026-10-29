# Tuskline developer tasks. Run `make help`.
# Go modules live in a go.work workspace, so `go build ./...` from the root does
# not work: every Go target loops over $(MODULES).

MODULES   := core agent cli server
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE      ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -s -w \
  -X github.com/interlace-sh/tuskline/core/version.Version=$(VERSION) \
  -X github.com/interlace-sh/tuskline/core/version.Commit=$(COMMIT) \
  -X github.com/interlace-sh/tuskline/core/version.Date=$(DATE)
BIN       := bin
WEBDIST   := server/internal/webui/dist
PG_VERSIONS ?= 14 15 16 17 18

.PHONY: help build bin test test-integration test-web lint lint-go lint-web fmt tidy web web-install dev-db dev-db-down clean check

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

build: ## Compile every Go module
	@for m in $(MODULES); do echo "==> build $$m"; (cd $$m && go build ./...) || exit 1; done

bin: ## Build the tl and tuskline binaries into ./bin (run `make web` first to embed the UI)
	@mkdir -p $(BIN)
	cd cli && go build -trimpath -ldflags "$(LDFLAGS)" -o ../$(BIN)/tl ./cmd/tl
	cd server && go build -trimpath -ldflags "$(LDFLAGS)" -o ../$(BIN)/tuskline ./cmd/tuskline

test: ## Unit tests for every Go module (race detector on)
	@for m in $(MODULES); do echo "==> test $$m"; (cd $$m && go test -race -count=1 ./...) || exit 1; done

test-integration: ## Integration tests against PostgreSQL 14-18 in Docker (testcontainers)
	cd core && PG_VERSIONS="$(PG_VERSIONS)" go test -tags integration -count=1 -timeout 15m ./...

test-web: ## Web unit tests (vitest)
	cd web && npm test

lint: lint-go lint-web ## All linters

lint-go: ## gofmt check, go vet and golangci-lint (if installed)
	@test -z "$$(gofmt -l $(MODULES))" || (echo "gofmt needed:"; gofmt -l $(MODULES); exit 1)
	@for m in $(MODULES); do echo "==> vet $$m"; (cd $$m && go vet ./... && go vet -tags integration ./...) || exit 1; done
	@if command -v golangci-lint >/dev/null; then \
	  for m in $(MODULES); do echo "==> golangci-lint $$m"; (cd $$m && golangci-lint run --config ../.golangci.yml ./...) || exit 1; done; \
	else echo "golangci-lint not installed; skipped (CI runs it)"; fi

lint-web: ## svelte-check, prettier and eslint
	cd web && npm run check && npm run lint

fmt: ## Format Go and web code
	gofmt -w $(MODULES)
	cd web && npm run format

tidy: ## go mod tidy in every module
	@for m in $(MODULES); do echo "==> tidy $$m"; (cd $$m && GOWORK=off go mod tidy) || exit 1; done

web-install: ## npm ci for the web UI
	cd web && npm ci

web: ## Build the web UI and copy it into the server's embed directory
	cd web && npm run build
	find $(WEBDIST) -mindepth 1 ! -name placeholder.html -exec rm -rf {} +
	cp -R web/build/. $(WEBDIST)/

check: build test lint-go ## Go build, unit tests and lint (web: make lint-web test-web)

dev-db: ## Start PostgreSQL 14-18 on ports 5414-5418
	docker compose -f deploy/compose/compose.yaml up -d --wait

dev-db-down: ## Stop the dev databases and delete their volumes
	docker compose -f deploy/compose/compose.yaml down -v

clean: ## Remove build output
	rm -rf $(BIN) web/build web/.svelte-kit
	find $(WEBDIST) -mindepth 1 ! -name placeholder.html -exec rm -rf {} +
