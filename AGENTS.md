# AGENTS.md: instructions for AI coding agents (and humans)

This file is authoritative. If it conflicts with anything else, follow this file and propose a fix to the other document. Path-specific rules live in `.cursor/rules/*.mdc`; the reasoning lives in `docs/`.

## 1. What you are building

Tuskline is an open-source PostgreSQL workbench and control plane: **one Go core, four surfaces** (desktop app, web UI, `tl` CLI/TUI, agent), plus a hosted control plane (Tuskline Cloud) that runs the same server. Read `docs/vision.md` (5 min) and `docs/architecture.md` before your first change.

Fixed decisions (do not relitigate without a new ADR; see `docs/decisions.md`):

| # | Decision |
|---|---|
| 1 | Both personas: developers (desktop, `tl`) and DBAs/SREs (server, fleet). One core and one inventory. |
| 2 | Hosted control plane from v1. Self-hosting always complete. Agents dial out; credentials stay with the customer. |
| 3 | AGPL-3.0 server/web/desktop; Apache-2.0 core, CLI, agent. |
| 4 | Surfaces: desktop (Wails v3), web, CLI. |
| 5 | Compatible systems: least effort only (tiers in `docs/compatibility.md`). |
| 6 | PostgreSQL 14+ only. |
| 7 | Build collectors, metric definitions, linter and advisor; *drive* Patroni, etcd, PgBouncer, pgBackRest, CNPG. |
| 8 | Solo founder + AI tooling: strict roadmap and cut list. |

## 2. Stack

| Layer | Choice |
|---|---|
| Core, server, CLI, agent | **Go** (1.25+, `go.work` workspace), `pgx/v5`, `pg_query_go` (from M7), Cobra (+ Bubble Tea for TUI) |
| Metadata | **Embedded SQLite** (pure-Go driver, no cgo in server); optional external PostgreSQL for team HA |
| Metrics history | Embedded columnar segments / Parquet (decided in M2 ADR); no external TSDB required |
| Targets | **PostgreSQL 14–18**, tier 1 managed services |
| Web UI | **SvelteKit** (`adapter-static`, SPA, `ssr = false`), **Svelte 5 runes**, **Tailwind CSS 4**, **shadcn-svelte** (Bits UI), TanStack Table/Virtual, CodeMirror 6, uPlot, `@lucide/svelte` |
| Desktop | **Wails v3**: Go host, core in-process, same web build (M8) |
| Tests | Go `testing` + **testcontainers-go** (PostgreSQL 14–18 in Docker), vitest, Playwright (UI acceptance) |
| Packaging | `go:embed` UI, distroless image, Homebrew, signed desktop builds |

## 3. Repository layout

```
core/      Apache-2.0  Go module github.com/interlace-sh/tuskline/core   (engine; no I/O surfaces)
  version/ inventory/ conn/ collector/ sqlpack/ analyser/ advisor/ lint/ provision/ store/
cli/       Apache-2.0  module …/cli    cmd/tl (main), internal/cmd (Cobra commands)
agent/     Apache-2.0  module …/agent  outbound collector (`tl agent`)
server/    AGPL-3.0    module …/server cmd/tuskline, internal/httpapi, internal/webui (go:embed)
web/       AGPL-3.0    SvelteKit app; `make web` copies web/build → server/internal/webui/dist
desktop/   AGPL-3.0    Wails v3 app (placeholder until M8)
deploy/    dev databases (compose) now; generated templates later
docs/      CC BY 4.0   vision, architecture, decisions + adr/, roadmap, design, cli, security …
prototype/ frozen React prototype = UI spec; port to web/, then delete
brand/     Tusk Arc SVG/PNG/ICO
```

Dependency direction (enforced in review): `cli, agent, server, desktop → core`. **Never** `core → anything`, and Apache modules never import AGPL modules. `tl serve` execs the `tuskline` binary; it does not import `server`.

## 4. How to pick the next task

1. Open `docs/roadmap.md`. Find the **lowest-numbered milestone with unchecked boxes**.
2. Take the **first unchecked task** in it whose dependencies are done (tasks are ordered). If the user named a task, do that one instead.
3. If a task is bigger than about one day or about 400 changed lines, split it in the roadmap first (as its own commit) and do the first part.
4. If a task needs a decision not covered by the ADRs, write a short ADR (`docs/adr/0000-template.md`) with status `Proposed`, stop, and ask.
5. Never start a later milestone while an earlier one has open tasks, unless the roadmap marks the task `(parallel)`.

## 5. Coding standards

**Go**
- `gofmt`/`goimports`; pass `go vet` and golangci-lint (`.golangci.yml`). SPDX header on every file (Apache-2.0 in core/cli/agent, AGPL-3.0-only elsewhere).
- Small packages with a `doc.go`. Accept interfaces, return structs. Every exported identifier is documented.
- `context.Context` is the first parameter of anything that does I/O; always apply a timeout to database calls.
- Errors: wrap with `%w` and context (`fmt.Errorf("probe %s: %w", ref, err)`); sentinel errors via `errors.Is`. No panics outside `main`.
- Logging: `log/slog`, structured. **Never log DSNs, passwords, tokens, query parameters or result rows.**
- SQL lives in `core/sqlpack` as versioned, documented queries, keyed by `server_version_num` range. Always use parameters, never string concatenation of values. Quote identifiers with `pgx.Identifier`.
- No cgo in `core`, `cli`, `agent` or `server` (pure-Go SQLite driver). cgo is allowed only in `desktop` (for example DuckDB).
- Add dependencies sparingly. Apache modules need permissive licences. Run `make tidy`.

**Web**
- Svelte 5 runes only (`$state`, `$derived`, `$effect`, `$props`); no legacy `export let` or stores for new code. Shared state in `*.svelte.ts` modules.
- shadcn-svelte components in `src/lib/components/ui` (add with `npx shadcn-svelte@latest add`); app components in `src/lib/components`. Use `#lib/...` imports with explicit `.js` or `/index.js`.
- Tailwind utility classes and the design tokens in `src/routes/layout.css`; no hard-coded hex values in components (use `bg-env-prod`, `text-primary`, and so on).
- All data comes from the server's REST API (`/api/v1`); there are no private endpoints. The UI must work identically in the browser and in Wails.
- Accessibility (WCAG 2.2 AA): labels, focus rings, keyboard paths, reduced motion.

**Docs and text**: British English. Concise. Update the relevant `docs/*.md` in the same change.

## 6. Testing

- `make test`: unit tests for every Go module (race detector). Must be fast and need no Docker.
- `make test-integration`: files tagged `//go:build integration`. These start **PostgreSQL 14, 15, 16, 17 and 18** with testcontainers-go (`postgres:<v>-alpine`); `PG_VERSIONS=17` narrows the run. Every SQL pack query and every version-specific branch needs an integration test across the matrix. CI runs each version as a separate job.
- `make test-web`: vitest for pure TypeScript logic. UI screens are accepted by Playwright smoke checks ported from `prototype/scripts/smoke.mjs` plus a screenshot comparison.
- Table-driven tests; `t.Parallel()` where safe; no sleeps (poll with a deadline).
- Bug fix ⇒ regression test first.
- `make dev-db` starts PostgreSQL 14–18 on ports 5414–5418 for manual testing.

## 7. Safety rules (non-negotiable)

1. **Production is read-only by default.**
   - Sessions to `prod` environments start with `default_transaction_read_only = on`, a statement timeout (30 s), a lock timeout (2 s) and grid row limits (1,000).
   - Elevation is explicit, time-limited (15 min maximum by default), carries a reason and is audited.
   - These defaults live in `core/inventory.DefaultPolicy`. Do not weaken them without an ADR.
2. Every mutating action (cancel, terminate, switchover, restart, restore, DDL, provisioning) has: a **dry run / SQL preview**, pre-flight checks, a **typed confirmation** in prod, and an audit entry.
3. Collectors are read-only and use the least-privilege `tuskline_monitor` role (`pg_monitor`). They respect a per-target query budget and back off under load.
4. **Secrets**:
   - Never write credentials to logs, error messages, SQLite in plaintext, test fixtures or git.
   - Store references to a secret provider, or envelope-encrypt at rest.
   - TLS defaults to `verify-full`.
5. Never store query parameters or result data unless a policy allows it. Redact PII in captured plans. **No telemetry** by default.
6. Provisioning refuses to touch non-empty data directories. Compose deployments are labelled non-production.
7. AI and MCP features are read-only by default and never auto-execute writes or DDL.
8. When an agent runs commands itself: never connect to a non-local database, never run destructive commands against anything but disposable test containers, and never push, release or publish without being asked.

## 8. Definition of done

A task is done when **all** of these are true:

- [ ] Code meets section 5 and the relevant `.cursor/rules`; SPDX headers are present; the licence boundary holds.
- [ ] Unit tests added or updated. Integration tests added for anything that touches PostgreSQL, passing on 14–18 (CI).
- [ ] `make check` passes (Go build, test, lint), and so does `make lint-web test-web` if `web/` changed.
- [ ] User-visible behaviour is documented: `docs/cli.md` for commands, `docs/design.md` for screens, an ADR for decisions.
- [ ] The roadmap checkbox is ticked in the same PR; follow-ups are added as new unchecked tasks.
- [ ] No secrets, no generated artefacts (`web/build`, `server/internal/webui/dist/*` except the placeholder, `bin/`) committed.
- [ ] UI work: matches the prototype screenshot, dark and light themes, keyboard reachable.

## 9. Commits and pull requests

- **Conventional Commits**: `type(scope): summary`. Types are `feat fix docs test refactor perf build ci chore`. Scopes are `core cli agent server web desktop deploy docs`. Use the imperative mood, with a summary of 72 characters or fewer, and give the reason in the body.
- **Sign off** (`git commit -s`, DCO). No CLA.
- One logical change per commit or PR. Reference the roadmap task (for example `Roadmap: M1.3`).
- The PR description covers what and why, how it was tested (commands and output), screenshots for UI, and risks or follow-ups.
- Branches: `m1/tl-connect`, `fix/conn-timeout`, and so on. Do not force-push shared branches. `main` stays green.

## 10. Useful commands

```sh
make help              # list targets
make build test lint   # Go
make test-integration  # Docker + PostgreSQL 14-18
make web-install web   # build UI into the server
make bin && ./bin/tuskline serve
cd web && npm run dev  # UI dev server on :5173, proxies /api to :8080
```

Node.js 22.17 or newer is required for `web/`. In a workspace (`go.work`), run Go commands per module or through `make`; `go build ./...` from the repository root does not work.
