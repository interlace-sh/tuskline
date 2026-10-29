# Thin wrapper over moon (https://moonrepo.dev/moon). moon owns every task,
# its inputs, outputs and cache; see .moon/ and each project's moon.yml.
# Nothing here adds behaviour: every target maps to one moon command, so
# `make check` and `moon run :check` are always the same thing.
# Toolchain: run `proto install` (reads .prototools) once. ADR 0012.

MOON ?= moon
PG_VERSIONS ?= 14 15 16 17 18

.PHONY: help check build bin test test-integration test-web lint lint-go lint-web fmt tidy web web-install dev dev-db dev-db-down clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

check: ## Build, test and lint every project, Go and web (moon run :check)
	$(MOON) run :check

build: ## Compile every project; server embeds the web build (moon run :build)
	$(MOON) run :build

bin: ## Build ./bin/tl and ./bin/tuskline (web UI embedded)
	$(MOON) run cli:bin server:bin

test: ## Go and web unit tests (moon run :test)
	$(MOON) run :test

test-integration: ## Integration tests against PostgreSQL 14-18 (needs Docker)
	PG_VERSIONS="$(PG_VERSIONS)" $(MOON) run core:integration

test-web: ## Web unit tests only
	$(MOON) run web:test

lint: ## Every linter (moon run :lint)
	$(MOON) run :lint

lint-go: ## gofmt check, go vet and golangci-lint for the Go modules
	$(MOON) run core:lint agent:lint cli:lint server:lint

lint-web: ## svelte-check, prettier and eslint
	$(MOON) run web:typecheck web:lint

fmt: ## Format Go and web code in place
	$(MOON) run :fmt

tidy: ## go mod tidy in every module
	$(MOON) run :tidy

web: ## Build the web UI and copy it into the server's embed directory
	$(MOON) run server:embed-web

web-install: ## Install web dependencies (moon also does this on demand)
	cd web && npm ci

dev: ## Vite dev server on :8090 (proxies /api to :8080)
	$(MOON) run web:dev

dev-db: ## Start PostgreSQL 14-18 on ports 5414-5418 (Docker)
	docker compose -f deploy/compose/compose.yaml up -d --wait

dev-db-down: ## Stop the dev databases and delete their volumes
	docker compose -f deploy/compose/compose.yaml down -v

clean: ## Remove build output and moon's cache
	rm -rf bin web/build web/.svelte-kit .moon/cache
	find server/internal/webui/dist -mindepth 1 ! -name placeholder.html -exec rm -rf {} +
