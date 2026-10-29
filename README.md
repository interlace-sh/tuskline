<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="brand/svg/tusk-arc-lockup-dark.svg">
    <img src="brand/svg/tusk-arc-lockup-light.svg" alt="Tuskline" height="64">
  </picture>
</p>

<p align="center"><strong>The calm, trustworthy cockpit for every PostgreSQL you run, from a laptop container to a multi-cloud fleet.</strong></p>

> **Status: pre-alpha.** This repository holds a scaffold, the plan and a design prototype. Nothing here is ready to point at a database you care about yet. Follow progress in [docs/roadmap.md](docs/roadmap.md).

Tuskline is an open-source PostgreSQL workbench and control plane. It is one Go core with four surfaces: a desktop app, a web UI, the `tl` CLI/TUI and an agent. It brings together the jobs that today need a dozen tools: SQL IDE, monitoring, query insights, config advice, HA and backups, migrations and guided deployment.

- **Two front doors.** Developers start local-first with the desktop app or `tl connect`. DBAs and SREs start with a self-hosted server or Tuskline Cloud and see the whole fleet.
- **Diagnoses, not dashboards.** Every finding comes with evidence, the SQL behind it and a fix.
- **Safe by default.** Every screen shows which environment you are in. Production is read-only unless policy allows otherwise, and every action is previewed and audited.
- **Orchestrates, does not replace.** Tuskline drives Patroni, etcd, PgBouncer, pgBackRest and CloudNativePG; it does not reimplement them.
- **PostgreSQL 14+**: community PostgreSQL, Amazon RDS and Aurora, Cloud SQL and Azure Flexible Server first. See [compatibility](docs/compatibility.md).

## Licence

| Component | Licence |
|---|---|
| `core/`: the shared engine | Apache-2.0 |
| `cli/`: the `tl` CLI and TUI | Apache-2.0 |
| `agent/`: the `tl agent` collector | Apache-2.0 |
| `server/`, `web/`, `desktop/`: the multi-user server, web UI and desktop app | AGPL-3.0-only |
| `docs/` | CC BY 4.0 |

See [LICENSING.md](LICENSING.md) for why the line is drawn there and what it means for you. Contributions use the DCO (`git commit -s`), with no CLA.

## Quickstart

> Placeholder: none of this works end to end yet. The commands are the target UX.

```sh
# Developer door (target): single binary
brew install tuskline            # not published yet
tl connect "postgres://me@localhost:5432/app"
tl top local/app

# Team server (target)
docker run -p 8080:8080 -v tuskline:/data ghcr.io/interlace-sh/tuskline   # not published yet
```

### Build from source (works today)

Requirements: [proto](https://moonrepo.dev/proto), which installs the pinned Go, Node.js, npm, [moon](https://moonrepo.dev/moon) and golangci-lint from [`.prototools`](.prototools), and Docker if you want to run the integration tests.

```sh
proto trust && proto install    # once per clone
moon run :check                 # build, test and lint every project
moon run cli:bin server:bin     # ./bin/tl and ./bin/tuskline, with the UI embedded
./bin/tl version
./bin/tuskline serve  # http://127.0.0.1:8080
make dev-db           # PostgreSQL 14-18 on ports 5414-5418
./bin/tl connect --dsn "postgres://tuskline:tuskline@localhost:5417/tuskline?sslmode=disable"
```

## Repository map

| Path | What |
|---|---|
| [AGENTS.md](AGENTS.md) | Instructions for AI coding agents and humans: start here to contribute |
| [docs/](docs/) | Vision, architecture, decisions (ADRs), roadmap, design, CLI spec, security |
| `core/` `cli/` `agent/` `server/` | Go modules (one `go.work`) |
| `web/` | SvelteKit + Svelte 5 + Tailwind + shadcn-svelte UI |
| `desktop/` | Wails v3 desktop app (placeholder until M8) |
| `deploy/` | Dev databases now; generated deployment templates later |
| `prototype/` | Frozen React prototype: the design reference for `web/` |
| `brand/` | Tusk Arc logo, icons and favicon |
| `.prototools`, `.moon/`, `*/moon.yml` | Toolchain pins (proto) and tasks (moon) |

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md). Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

PostgreSQL is a registered trademark of the PostgreSQL Community Association. Tuskline is not affiliated with or endorsed by it.
