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
| `tl connect --dsn <url>` | Probes the server: version, read-only status. Refuses PostgreSQL < 14 |
| `tl connect <env/cluster>` | Validates the reference; inventory lookup is M1 |
| `tl ls`, `tl top <ref>` | Stubs that exit 1 with "not implemented yet (roadmap M1)" |
| `tl serve [args]` | Runs the separate AGPL `tuskline serve` binary from `PATH` ([ADR 0010](adr/0010-licence-boundary.md)) |
| `tl agent --server <url>` | Stub; token from `TUSKLINE_AGENT_TOKEN` (M9) |

## Target command set

| Command | What it does | Milestone |
|---|---|---|
| `tl serve` | Start the web UI and API (what the Docker image runs, via `tuskline serve`) | M0/M3 |
| `tl connect prod/orders` | Add or open a connection; psql-style shell through the inventory, with tunnels and IAM handled | M1 |
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
