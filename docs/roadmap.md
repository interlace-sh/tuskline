# Roadmap

Milestones are sized for one founder with AI coding agents ([0008](adr/0008-solo-founder-scope.md)). Durations are planning estimates, not commitments. Work top to bottom: see `AGENTS.md` § 4. Tick a box in the same PR that completes the task. Tasks marked *(parallel)* may start before earlier milestones close.

| Release | Milestones | Theme |
|---|---|---|
| R1: developer door | M0–M4, M8 | Core, CLI, SQL IDE, onboarding, desktop |
| R2: fleet door and Cloud | M2 history, M5 (Compose), M6 (read-only), M9 | History, fleet, teams, Cloud beta, Compose HA |
| R3: guarded actions | M5 (CNPG, SSH), M6 actions, M7, Later | Switchover/restore, VM provisioning, migrations, lakehouse |

---

## M0: repository, CI and tooling (about 1 week)

- [x] M0.1 Monorepo scaffold: `go.work` with `core`, `cli`, `agent`, `server`; SvelteKit `web`; licences; docs; prototype; brand.
- [x] M0.2 Developer tasks: build, test, integration, lint, fmt, web embed, bin, dev-db. Now moon tasks with a thin `Makefile` alias layer ([0012](adr/0012-proto-and-moon.md)).
- [x] M0.2a Toolchain pins in `.prototools` (Go, Node.js, npm, moon, golangci-lint) installed by proto; moon workspace, per-project `moon.yml` and inherited Go tasks with inputs and outputs for caching.
- [x] M0.3 CI workflow: `moonrepo/setup-toolchain` + `moon ci` (gofmt, vet, unit tests, cross-compile, golangci-lint per module, web typecheck/lint/test/build), plus integration per PostgreSQL 14–18.
- [x] M0.4 Push to `github.com/interlace-sh/tuskline` and make the first CI run green. Fix anything that only fails on the runners (first run of `moon ci` and the integration matrix included).
- [x] M0.4a Optional: enable moon remote caching or `.protolock` once CI timings justify it; record the decision in ADR 0012. Decision (2026-10-29): not yet; a warm `moon run :check` is already a few seconds.
- [x] M0.5 CI licence-boundary check: a script that fails if `core`, `cli` or `agent` import `server`, or if any `.go` file lacks the SPDX header for its directory. (`scripts/check-licence-boundary.sh`, self-tested; `moon run scripts:check`.)
- [x] M0.6 Dependency licence check for the Apache modules (for example `go-licenses check` with an allow-list). (`moon run :licences`, `.moon/tasks/apache.yml`.)
- [x] M0.7 Repository hygiene: issue and PR templates (PR template includes the definition-of-done checklist), `CODEOWNERS`, Dependabot for Go modules, npm and actions (or Renovate, which can also bump `.prototools`), branch protection notes in CONTRIBUTING.
- [x] M0.7a Maintainer: apply the `main` ruleset described in CONTRIBUTING.md (required checks, code-owner review, DCO app) once CI is green; this needs repository admin rights. Decision (2026-10-29): not applied. One person fast-forwards `main`; the ruleset waits for a second committer.
- [x] M0.8 Release skeleton: GoReleaser config producing `tl` and `tuskline` for linux/darwin/windows × amd64/arm64 with checksums and an SBOM (no publishing yet). (`moon run scripts:release-snapshot`.)
- [ ] M0.8a *(parallel)* Release workflow: tag-triggered GoReleaser job with keyless signing (cosign) of checksums and SBOMs. Needs explicit approval before anything is published. Publishing stays disabled in `.goreleaser.yaml` until then.
- [x] M0.9 Container image: multi-stage Dockerfile (web build → Go build → distroless), `tuskline serve` on :8080, non-root.

**Acceptance:**
- A fresh clone passes `proto install`, `moon run :check` and `moon run cli:bin server:bin`.
- CI is green on `main`.
- `docker build` produces an image whose `/api/v1/health` returns `{"status":"ok"}`.
- The boundary check fails on a deliberately bad import.

## M1: core connection, inventory and `tl connect / ls / top` (about 3 weeks)

- [x] M1.1 `core/conn`: connection profiles. DSN parsing, a `Redact()` helper, TLS modes with `verify-full` as the default, and `~/.pgpass` and `pg_service.conf` lookup. Unit tests cover redaction and TLS defaults.
- [x] M1.2 `core/conn`: `Probe` extended with provider detection. Covers community, RDS, Aurora (`aurora_version()`), Cloud SQL, Azure, AlloyDB and Neon heuristics, plus `shared_preload_libraries`, installed extensions and login-role privileges. Integration-tested on 14–18.
- [x] M1.3 `core/inventory`: file-backed inventory (`$XDG_CONFIG_HOME/tuskline/inventory.yaml`) with environments, clusters and connection profiles. Secrets are stored as references (env var, keychain placeholder), never plaintext. Includes CRUD and validation.
- [x] M1.4 `core/conn`: session setup applies the environment policy (`default_transaction_read_only`, `statement_timeout`, `lock_timeout`). Integration test: a write in a prod session fails, the same write in dev succeeds.
- [x] M1.5 `tl connect <env/cluster>` and `tl connect --dsn`:
  - Adds or updates the inventory entry interactively (`--env`, `--name` flags for scripts).
  - Probes, prints the server summary and missing extensions.
  - `--output json`.
- [x] M1.6 `tl ls`: inventory as a table (environment colour, version, provider, last probe), `--output json`.
- [x] M1.7 `core/sqlpack`: first pack, for activity, waits, locks and blocking chains. Version-aware, documented, integration-tested on 14–18.
- [x] M1.8 `tl top <env/cluster>`: Bubble Tea TUI with live sessions, waits, locks and blocking tree, refreshing every 1 s. Keys: `q` quit, `/` filter, `s` sort. No actions. Has a `--once` text mode for CI and tests.
- [x] M1.9 `tl version` and `tl serve` polish: helpful message when `tuskline` is not on `PATH`; signals forwarded to the child.
- [x] M1.10 Docs: `docs/cli.md` updated for every flag; README quickstart switched from placeholders to commands that work.

**Acceptance:**
- Against `make dev-db`, `tl connect --dsn … --env dev --name pg17` then `tl ls` shows the cluster.
- `tl top dev/pg17 --once` lists sessions on every version 14–18 (integration test).
- Credentials never appear in output, logs or the inventory file (test).
- Prod sessions reject writes (test).

## M2: metrics collector and query insights (about 4 weeks)

- [x] M2.1 ADR: metrics storage format (embedded segments versus Parquet versus SQLite tables), downsampling, and the retention defaults (1 s raw for 1 h, 1 min for 14 days, 1 h for 13 months). ([0013](adr/0013-metrics-storage.md).)
- [x] M2.2 `core/collector`: scheduler with per-target query budget, jitter, timeouts and back-off under load. Unit tests cover budget enforcement.
- [x] M2.3 `core/sqlpack`: statements (`pg_stat_statements`, with deltas and counter-reset handling), database, tables, IO (`pg_stat_io` on 16+), WAL (`pg_stat_wal`), checkpointer (17+) / bgwriter. Integration-tested on 14–18.
- [x] M2.4 `core/store`: SQLite metadata store with migrations, including the collection cursor used to resume counter deltas. Pure-Go driver, no cgo.
- [x] M2.4b `core/store`: Parquet segment writer and reader, retention and compaction, per [ADR 0013](adr/0013-metrics-storage.md).
- [x] M2.5 `tl snapshot` / `tl diff`: workload snapshots to a file, and the delta between two.
- [x] M2.6 `core/analyser`: first health checks with evidence and a fix. Covers missing `pg_stat_statements`, cache hit ratio, idle in transaction, wraparound age, unused or duplicate indexes, bloat estimate and replication lag.
- [x] M2.7 `tl diagnose <ref>`: runs all checks and writes text, JSON or Markdown.
- [ ] M2.8 Server API: `/api/v1/clusters`, `/api/v1/clusters/{id}/queries` (top queries with deltas), `/activity`, `/health`.
- [ ] M2.9 `/metrics` Prometheus export of collected metrics *(parallel)*.

**Acceptance:**
- `tl diagnose` on an unfamiliar dev database produces a useful report in under five minutes from install, needing only `pg_stat_statements`.
- The collector's overhead is measured and published (no invented figures).
- Counter resets are handled (test).

## M3: web UI shell and SQL editor (about 5 weeks)

- [ ] M3.1 Web foundation: app shell from the prototype. Includes the top bar with environment switcher, left rail, breadcrumb, command palette (`Cmd/Ctrl K`), dark and light themes, density switch, and the environment colour rail.
- [ ] M3.2 Typed API client (`src/lib/api`), fixtures moved from `prototype/src/data.ts`, and a demo mode.
- [ ] M3.3 Port screens: Fleet, Cluster health, Query insights (from the M2 APIs).
- [ ] M3.4 SQL editor:
  - CodeMirror 6 with PostgreSQL highlighting and schema completion.
  - Run with `Cmd/Ctrl Enter`.
  - Virtualised result grid (TanStack Virtual) with the policy row limit.
  - History and cancel.
- [ ] M3.5 EXPLAIN visualiser (`Shift Cmd/Ctrl Enter` runs EXPLAIN ANALYZE). The plan tree uses D3, with a plain-English summary and warnings. Prod requires a confirmation before ANALYZE on writes.
- [ ] M3.6 Server: query execution API (streamed results), policy enforcement server-side, audit log of prod queries, local users plus sessions.
- [ ] M3.7 Port the prototype's Playwright smoke checks to run against `moon run server:bin && tuskline serve` in CI.

**Acceptance:**
- The ported smoke checks pass.
- Screens match the prototype screenshots (dark and light).
- A prod session cannot run a write without elevation (test).
- Grid scrolls 100k rows smoothly.

## M4: onboarding and config advisor (about 3 weeks)

- [ ] M4.1 Connection wizard (web and `tl connect`): connect → classify → detect → fix → verify.
- [ ] M4.2 Extension catalog and generated steps. A catalog in `core` covers the five extensions Tuskline uses (`pg_stat_statements`, `auto_explain`, `pg_wait_sampling`, `pg_stat_kcache`, `hypopg`) and the ordinary single-node ones Autobase auto-setup turns on (pgvector, PostGIS, pgRouting, pg_cron, pgaudit, pg_partman, pg_repack). Each entry records what it unlocks, PostgreSQL 14–18 availability, preload and restart, the PGDG package name, and which managed providers allow-list it. Generated steps merge `shared_preload_libraries` instead of replacing it, and use `patronictl edit-config` when Patroni manages config. See [onboarding](onboarding-and-deploy.md).
- [ ] M4.2a Install plan (preview only; apply is M5). Self-managed targets install the PGDG package, or the Pigsty repository when PGDG does not ship it, then the conf change and `CREATE EXTENSION`. Managed clouds get the provider command, or "not offered here". No compiling from source. The plan shows every package and SQL line before anything runs.
- [ ] M4.2b Dev database image: `make dev-db` installs `pg_wait_sampling`, `pg_stat_kcache` and `hypopg` from PGDG so a laptop matches the catalog. Integration tests stay on stock `postgres` images.
- [ ] M4.3 Monitoring-role script generator (`tuskline_monitor`) and managed-cloud snippets: RDS parameter groups, Cloud SQL flags, Azure `az` commands.
- [ ] M4.4 `core/advisor` and `tl config advise <ref>`: lints `postgresql.conf` and tunes memory, WAL and checkpoints, autovacuum, planner and I/O, and connections. Each recommendation gives its evidence, a diff, and whether it needs a reload or restart.
- [ ] M4.5 Port the Setup & deploy screens' onboarding parts (Add connection, Config advisor) from the prototype.

**Acceptance:**
- On a fresh `postgres:17` container, the wizard generates SQL that, once applied, makes the verify step pass (integration test).
- Advisor output on a fixture config matches a golden file.

## M5: guided deploy (Compose → CNPG → SSH/VM) (about 6 weeks, split across R2 and R3)

- [ ] M5.1 `core/provision`: plan/apply engine with typed steps, a diff of every file, unit and manifest, resumable apply, and a streamed log.
- [ ] M5.2 Compose target (R2): generate `compose.yaml`, `.env` and `haproxy.cfg` for Patroni + etcd ×3 + PgBouncer + HAProxy + pgBackRest. `tl provision compose plan|up` with health-gated start-up. Clearly labelled non-production.
- [ ] M5.3 Pre-flight checks: ports, disk, Docker version, existing volumes. Refuse non-empty data directories.
- [ ] M5.4 Hand-over: add the cluster to the inventory with a monitoring role, register the Patroni API and backup repository, and schedule the first restore drill.
- [ ] M5.5 CloudNativePG target (R3): `Cluster`, `Pooler` and Barman Cloud plugin manifests, server-side apply, status read back.
- [ ] M5.6 SSH/VM target (R3): Go SSH executor, PGDG packages (extension packages from the M4.2 catalog, Pigsty repo only when PGDG lacks the package), systemd units and config templates, idempotent versioned steps. Autobase-playbook mode.
- [ ] M5.7 Port the Deploy and Deployments screens.

**Acceptance:**
- `tl provision compose up` brings up a 3-node cluster that Tuskline then monitors.
- Killing the primary container fails over.
- Apply resumes after an interrupted run (integration test, CI-optional because it is slow).

## M6: HA and backups (Patroni, pgBackRest) (about 4 weeks)

- [ ] M6.1 Read-only (R2): Patroni REST topology, CNPG status, replication lag, timeline history.
- [ ] M6.2 Read-only (R2): pgBackRest `info` timeline (full/diff/incr, WAL coverage band, gaps), CNPG backups. `tl backup status`.
- [ ] M6.3 `tl ha status` and the HA and Backups screens ported from the prototype.
- [ ] M6.4 Guarded actions (R3): switchover, restart and PITR restore (`tl ha switchover`, `tl backup restore --to <time>`). Each has a dry run, pre-flight checks, typed confirmation and an audit record. Actions go through Patroni and CNPG APIs, never around them.
- [ ] M6.5 `tl backup verify`: restore drill into a scratch container.

**Acceptance:**
- Topology and backup views are correct against the M5 Compose stack.
- Switchover in a test stack completes with the audit entry.
- No action runs without a preview (test).

## M7: migrations (about 4 weeks, R3)

- [ ] M7.1 `core/lint` on `pg_query_go`: lock-aware rules (for example `ADD COLUMN … DEFAULT` volatility, `CREATE INDEX` without `CONCURRENTLY`, `ALTER TYPE`, `SET NOT NULL` on big tables), using live table sizes and activity.
- [ ] M7.2 `tl migrate lint|plan|apply --env staging`: drive Flyway, Sqitch or plain SQL folders. Prod needs approval.
- [ ] M7.3 Drift detection: schema snapshot diff between environments.
- [ ] M7.4 Migrations screen (PR-style review) ported from the prototype.

**Acceptance:** golden-file tests for lint rules; `migrate apply` on prod refuses without approval (test).

## M8: desktop app (Wails v3) (about 3 weeks; can start after M3)

- [ ] M8.1 Scaffold `desktop/` with Wails v3 and add it to `go.work`. Serve the web build and mount the server handler in-process.
- [ ] M8.2 Local-first mode: SQLite in the user config directory, no login. Optional sign-in to a server or Cloud.
- [ ] M8.3 Native extras: OS keychain secrets, SSH agent, `~/.pgpass` and `pg_service.conf` import, file dialogs.
- [ ] M8.4 Signed builds in CI: macOS (notarised, Intel and Apple Silicon), Windows x64/arm64, Linux x64/arm64. Updater.
- [ ] M8.5 WebKitGTK and WebKit rendering checks (smoke tests in the webview).

**Acceptance:** the same smoke checks pass in the desktop build on all three OSes; there are no UI forks.

## M9: hosted control plane (Tuskline Cloud) (about 6 weeks, R2)

- [ ] M9.1 Multi-tenancy: organisation scoping across inventory, storage and API.
- [ ] M9.2 OIDC SSO (Google, Entra, Okta, Keycloak) and RBAC roles viewer, analyst, operator and admin, scoped to project and environment. Just-in-time elevation.
- [ ] M9.3 Agent protocol: `tl agent` dials out over mTLS, enrols with a one-time token, resumes after disconnects, and sends metrics, redacted plans and metadata only.
- [ ] M9.4 Alerting: default rules; email, Slack and generic webhook.
- [ ] M9.5 Audit log export (SIEM) and the append-only store.
- [ ] M9.6 Cloud operations: deployment, backups of Tuskline's own metadata, status page, billing hooks. Pricing is decided after the beta, so do not invent prices.
- [ ] M9.7 Discovery: inventory import from AWS, GCP, Azure and Kubernetes (`tl inventory import`).

**Acceptance:** an agent behind NAT appears in a Cloud organisation, its cluster's insights load, and credentials never leave the customer network (test with network capture).

## Later

- [ ] Tier 2 systems with documented gaps: AlloyDB, Neon (frequent snapshots), TimescaleDB, Citus.
- [ ] hypopg-validated index advisor (`tl advise indexes`); per-table autovacuum tuning.
- [ ] **DuckDB / DuckLake:** DuckLake catalogue browser (`ducklake_*` tables); DuckDB as a second engine in the desktop IDE; explore DuckLake for the metrics archive; `pg_duckdb` awareness.
- [ ] **interlace.sh integration:**
  - Query attribution by `interlace_model` / `run_id` comments (the dbt comment format works too).
  - Grouping of fingerprinted tables and the `interlace gc` hint.
  - Lock-aware linting of terminal writes.
  - Pipeline health from `interlace serve`.
- [ ] MCP server (read-only by default); YAML SQL checks.
- [ ] AI (bring your own model, off by default): plan explanations and incident summaries.

## Cut or deferred (do not start without an ADR)

- gRPC API, Terraform provider, WASM plugin host, VS Code extension.
- PGO, StackGres, Barman, WAL-G, PagerDuty, Opsgenie, and Cloud Logging / Azure Monitor ingestion.
- Workload replay, thin clones, forecasting, cost insight, data compare and masking, SAML and SCIM.
- Not building: our own HA manager, backup engine or Kubernetes operator, or mobile apps.
