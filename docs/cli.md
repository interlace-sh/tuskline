# CLI: `tl`

`tl` is the Apache-2.0 CLI and TUI. It uses the same core and inventory as the UI. References take the form `environment/cluster`, for example `prod/orders`.

Conventions:

- Every data command takes `--output text|json` (the default is `text`).
- Errors go to stderr with exit code 1. Usage errors exit with 2 once Cobra's handling is customised.
- Commands that change anything print a preview and need `--yes` or a typed confirmation in prod.
- Credentials are never printed.

## Implemented in the scaffold

| Command | Status |
|---|---|
| `tl version` | Prints `tl <version> (commit <sha>, built <date>)` |
| `tl connect` | Probes a server and adds it to the inventory, or probes an inventory entry again; see [below](#tl-connect) |
| `tl ls` | Lists the inventory; see [below](#tl-ls) |
| `tl top <ref>` | Stub that exits 1 with "not implemented yet (roadmap M1)" |

Global flag: `--inventory <file>` overrides the inventory location (default `$TUSKLINE_INVENTORY`, else `$XDG_CONFIG_HOME/tuskline/inventory.yaml`).

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

- **Probe.** It prints the version, provider, primary or standby, session read-only status, TLS mode, login-role privileges, `shared_preload_libraries`, and recommended extensions that are missing with what each unlocks. It refuses PostgreSQL older than 14. With a target environment, the probe runs under that environment's policy, so a prod probe is a read-only session with prod timeouts.
- **Save.** With `--dsn` and a target, it adds or updates the entry and records the probe. Nothing is saved if the probe fails. On a terminal, missing values are asked for on stderr; leave them blank to only probe. When stdin is not a terminal, it never prompts.
- **Passwords** are never stored or printed. The password is taken from the DSN, then the saved `--password-env` reference, then `~/.pgpass` (ignored if group- or world-readable). A DSN password is used for that probe only, with a note on stderr.
- **TLS.** A `sslmode` weaker than `verify-full` gives a warning on stderr.

### `tl ls`

```text
ENVIRONMENT   CLUSTER  TARGET                      VERSION  PROVIDER  ROLE     LAST PROBE
prod          orders   db.example.com:6432/orders  17.6     rds       standby  1h ago
team-a (dev)  -
```

One row per cluster, from the last successful `tl connect` probe; it does not connect. An environment without clusters gets a `-` row. The kind is shown in brackets when it differs from the environment's name. On a terminal the environment is coloured by kind (prod red, staging amber, dev green, local grey, as in [design](design.md)); `NO_COLOR` or a pipe turns colour off. `--output json` gives `{inventory, environments, clusters}`, with each environment's policy and no passwords.
| `tl serve [args]` | Runs the separate AGPL `tuskline serve` binary from `PATH` ([ADR 0010](adr/0010-licence-boundary.md)) |
| `tl agent --server <url>` | Stub; token from `TUSKLINE_AGENT_TOKEN` (M9) |

## Target command set

| Command | What it does | Milestone |
|---|---|---|
| `tl serve` | Start the web UI and API (what the Docker image runs, via `tuskline serve`) | M0/M3 |
| `tl connect prod/orders` | Add or probe a connection (M1); psql-style shell through the inventory, with tunnels and IAM handled (later) | M1 |
| `tl ls` | List the inventory with environment, version, provider and last probe | M1 |
| `tl top prod/orders` | Live activity, waits and locks TUI (`--once` for a text snapshot) | M1 |
| `tl diagnose prod/orders` | Run all health checks; text, JSON or Markdown report | M2 |
| `tl snapshot` / `tl diff` | Workload snapshots and deltas | M2 |
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
| `tuskline serve [-addr 127.0.0.1:8080]` | HTTP server: `/api/v1/health`, `/api/v1/version`, and the embedded SPA. `TUSKLINE_ADDR` overrides the address |
| `tuskline version` | Build info |

The container image (`Dockerfile`) runs `tuskline serve` as the distroless `nonroot` user, with `TUSKLINE_ADDR=:8080` so the port can be published, and a `/data` volume for the metadata store. `moon run scripts:image-smoke` builds it and checks `/api/v1/health`.
