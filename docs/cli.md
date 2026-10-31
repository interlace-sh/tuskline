# CLI: `tl`

`tl` is the Apache-2.0 CLI and TUI. It uses the same core and inventory as the UI. References take the form `environment/cluster`, for example `prod/orders`.

Conventions:

- Every data command takes `--output text|json` (`-o`; the default is `text`).
- Errors go to stderr with exit code 1. `tl serve` exits with the server's status and does not print it again. Usage errors exit with 2 once Cobra's handling is customised.
- Commands that change anything print a preview and need `--yes` or a typed confirmation in prod.
- Credentials are never printed.

## Implemented

| Command | Status |
|---|---|
| `tl version` | Prints `tl <version> (commit <sha>, built <date>)`; see [below](#tl-version) |
| `tl connect` | Probes a server and adds it to the inventory, or probes an inventory entry again; see [below](#tl-connect) |
| `tl ls` | Lists the inventory; see [below](#tl-ls) |
| `tl top <ref>` | Live sessions, waits, locks and blocking chains; see [below](#tl-top) |
| `tl snapshot <ref>`, `tl diff` | Workload snapshots; see [below](#tl-snapshot) |
| `tl diagnose <ref>` | Health checks; see [below](#tl-diagnose) |
| `tl serve [args]` | Runs the separate `tuskline serve`; see [below](#tl-serve) |
| `tl agent` | Stub; see [below](#tl-agent) |

Global flag: `--inventory <file>` (connect, ls, top, snapshot and diagnose) overrides the inventory location (default `$TUSKLINE_INVENTORY`, else `$XDG_CONFIG_HOME/tuskline/inventory.yaml`).

### `tl version`

| Flag | Meaning |
|---|---|
| `--output text\|json`, `-o` | Text is one line, `tl <version> (commit <sha>, built <date>)`. JSON is `{version, commit, date, go, os, arch}`. |

### `tl connect`

```sh
tl connect --dsn "postgres://app@db.example.com/orders" prod/orders   # probe and save
tl connect --dsn "$DSN" --env dev --name pg17                          # the same, for scripts
tl connect --dsn "$DSN"                                                # probe only (asks on a terminal)
tl connect prod/orders --output json                                   # probe a saved entry
```

| Flag | Meaning |
|---|---|
| `--dsn <url>` | libpq URL or keyword/value string. `service=` is resolved from `pg_service.conf`. |
| `--env`, `--name` | Target environment and cluster name; the same as the `env/cluster` argument. |
| `--kind local\|dev\|staging\|prod` | Kind of a new environment. Defaults to the environment name when that is a kind; asked for on a terminal. An existing environment's kind is never changed by `connect`. |
| `--password-env <VAR>` | Save a reference to an environment variable holding the password. |
| `--output text\|json`, `-o` | Summary format. JSON holds the profile (no password), the server probe and the missing extensions. |

Behaviour:

- **Probe.** It prints the version, provider, primary or standby, session read-only status, TLS mode, login-role privileges, `shared_preload_libraries`, and recommended extensions that are missing with what each unlocks. Those five come from `core/catalog`, which also builds the setup steps (a merged preload list, or `patronictl edit-config` when Patroni owns the configuration). It refuses PostgreSQL older than 14. With a target environment, the probe runs under that environment's policy, so a prod probe is a read-only session with prod timeouts.
- **Save.** With `--dsn` and a target, it adds or updates the entry and records the probe. Nothing is saved if the probe fails. On a terminal, missing values are asked for on stderr; leave them blank to only probe. When stdin is not a terminal, it never prompts.
- **Passwords** are never stored or printed. The password is taken from the DSN, then the saved `--password-env` reference, then `~/.pgpass` (ignored if group- or world-readable). A DSN password is used for that probe only, with a note on stderr.
- **TLS.** A `sslmode` weaker than `verify-full` gives a warning on stderr.

### `tl ls`

```text
ENVIRONMENT   CLUSTER  TARGET                      VERSION  PROVIDER  ROLE     LAST PROBE
prod          orders   db.example.com:6432/orders  17.6     rds       standby  1h ago
team-a (dev)  -
```

One row per cluster, from the last successful `tl connect` probe; it does not connect. An environment without clusters gets a `-` row. The kind is shown in brackets when it differs from the environment's name. On a terminal the environment is coloured by kind (prod red, staging amber, dev green, local grey, as in [design](design.md)); `NO_COLOR` or a pipe turns colour off. `--output json` (`-o json`) gives `{inventory, environments, clusters}`, with each environment's policy and no passwords.

### `tl top`

```text
tl top prod/orders · PostgreSQL 17.6 · 12:00:01 UTC · 5 sessions, 2 active · sort: query age
waits: not waiting 3 · Lock/relation 1
blocking:
  100  app  idle in transaction  xact 1m15s  LOCK TABLE t IN ACCESS EXCLUSIVE MODE
  └─ 101  app  active  SELECT count(*) FROM t
lock waits:
  101 waits AccessShareLock relation on t for 3.0s

PID  USER    DATABASE  APP  STATE                WAIT           XACT   QUERY AGE  BLOCKED BY  QUERY
102  report  orders         active                                     12m00s                 SELECT * FROM big
...
```

A full-screen view of one inventory entry. It is read-only and has no actions: it runs the activity pack (`core/sqlpack/activity.go`) in a session with the environment's policy, and never runs a refresh while the previous one is still going. Refreshes slow down to at least four times the last refresh's duration, and double after each failure, up to 30 s. A failed refresh keeps the last snapshot on screen with the error, and a broken connection is reopened.

| Key | Action |
|---|---|
| `q`, `ctrl+c` | Quit |
| `/` | Filter (pid, user, database, application, state, wait or query text); `enter` keeps it, `esc` clears it |
| `s` | Sort: query age, transaction age, pid, state (idle sessions always last) |

| Flag | Meaning |
|---|---|
| `--once` | Print one snapshot and exit; works without a terminal (CI, scripts) |
| `--output json`, `-o` | With `--once`: `{ref, server_version_num, snapshot, blocking}` |
| `--interval <d>` | Refresh interval, default `1s`, minimum `500ms` |

A role without `pg_monitor` sees other users' sessions with query text and waits hidden by the server. Query text is shown live and never saved.

### `tl snapshot`

```sh
tl snapshot dev/pg17 --file before.json
tl snapshot dev/pg17 --file after.json
tl diff before.json after.json
```

`tl snapshot` reads `pg_stat_statements` for the current database, under the environment's session policy, and writes JSON (mode 0600). Query text is included because the snapshot was requested; it is truncated to `--query-text` characters (default 500) and at most `--limit` statements are kept (default 100). The extension has to be installed.

`tl diff <older> <newer>` prints the change in calls, execution time, rows and WAL bytes. A decrease is a statistics reset and is marked `*`. `--output json` gives `{deltas}`.

### `tl diagnose`

```sh
tl diagnose dev/pg17
tl diagnose prod/orders --output markdown
tl diagnose dev/pg17 --lag 30s --output json
```

Connects with the environment's session policy and runs the health checks: missing `pg_stat_statements`, cache hit ratio, idle-in-transaction sessions, wraparound age, unused and duplicate indexes, dead-tuple bloat, and replication lag. A check the role cannot read is reported and the rest still run.

| Flag | Meaning |
|---|---|
| `--output text\|json\|markdown`, `-o` | Report format. The default is `text`. JSON is `{ref, taken, server_version_num, findings}`. |
| `--lag <d>` | Replay lag that counts as a finding. The default is `1m`. |

### `tl serve`

`tl serve [args]` runs `tuskline serve [args]`, the separate AGPL server binary ([ADR 0010](adr/0010-licence-boundary.md)); `tl` never links it. It looks for `tuskline` on `PATH`, then next to the `tl` executable, so `./bin/tl serve` finds `./bin/tuskline`. If neither has it, it exits 1 with install hints: a release binary, `moon run server:bin`, or building and running the container image.

Arguments after `serve` are the server's. `tl serve -h` prints this help; `tl serve -addr :8080` is passed through (`TUSKLINE_ADDR` also sets the listen address; the server default is `127.0.0.1:8080`). While the server runs, `SIGINT`, `SIGTERM` and `SIGHUP` sent to `tl` are passed on to it (`os.Interrupt` on Windows), and `tl` exits with the server's exit code.

### `tl agent`

| Flag | Meaning |
|---|---|
| `--server <url>` | Tuskline server or Cloud URL. Required. |

The enrolment token is read from `TUSKLINE_AGENT_TOKEN` and never printed. The command then exits 1: the agent is not built yet (M9).

## Target command set

| Command | What it does | Milestone |
|---|---|---|
| `tl serve` | Start the web UI and API (what the Docker image runs, via `tuskline serve`) | M0/M3 |
| `tl connect prod/orders` | Add or probe a connection (M1); psql-style shell through the inventory, with tunnels and IAM handled (later) | M1 |
| `tl ls` | List the inventory with environment, version, provider and last probe | M1 |
| `tl top prod/orders` | Live activity, waits and locks TUI (`--once` for a text snapshot) | M1 |
| `tl diagnose prod/orders` | Health-check report in text, JSON or Markdown (M2) | M2 |
| `tl snapshot` / `tl diff` | Workload snapshots and deltas (M2); richer comparison later | M2 |
| `tl explain file.sql` | Plan with plain-English summary and warnings | M3 |
| `tl config advise prod/orders` | postgresql.conf linter and tuning advisor, with diffs, marked reload or restart | M4 |
| `tl provision compose plan\|up` | Generate and start a Compose HA stack (non-production) | M5 |
| `tl backup status\|verify\|restore --to <time>` | Read and drive pgBackRest and CNPG backups | M6 |
| `tl ha status\|switchover` | Patroni and operator topology; guarded actions | M6 |
| `tl migrate plan\|lint\|apply --env staging` | Run Flyway, Sqitch or SQL folders with lock analysis | M7 |
| `tl inventory import --from aws,gcp,azure,k8s` | Discover clusters from clouds and Kubernetes | M9 |
| `tl agent` | Outbound collector over mTLS | M9 |
| `tl advise indexes` | Index candidates validated with hypopg | Later |

## Server binary: `tuskline` (AGPL)

| Command | What it does |
|---|---|
| `tuskline serve [-addr 127.0.0.1:8080]` | HTTP server: `/api/v1/health`, `/api/v1/version`, `/api/v1/clusters` and `/api/v1/clusters/{env}/{name}/queries`, `/activity`, `/health`, plus `GET .../probe` (installed and available extensions, and `shared_preload_libraries`) and `POST .../query` (one statement, environment policy, row limit; production statements are audited; a successful body is newline-delimited JSON). `POST /api/v1/clusters` probes a server and saves it the way `tl connect` does; a typed password is not stored. The server is unauthenticated. The embedded SPA is served alongside. `TUSKLINE_ADDR` overrides the address. The inventory is `TUSKLINE_INVENTORY` or the XDG file. The audit database is `TUSKLINE_STORE` or `$XDG_DATA_HOME/tuskline/meta.db` |
| `tuskline version` | Build info |

The container image (`Dockerfile`) runs `tuskline serve` as the distroless `nonroot` user, with `TUSKLINE_ADDR=:8080` so the port can be published, and a `/data` volume for the metadata store. `moon run scripts:image-smoke` builds it and checks `/api/v1/health`.
