# Decisions

The eight product decisions below were recorded on 28 Oct 2026. Each one, and each later architectural choice, has an ADR in [adr/](adr/). To change a decision, write a new ADR that supersedes it.

| ADR | Question | Decision | Status |
|---|---|---|---|
| [0001](adr/0001-launch-personas.md) | Launch persona | Both: developers (IDE first) and DBAs/SREs (fleet first); one core, two front doors | Accepted |
| [0002](adr/0002-hosted-control-plane.md) | Funding | Hosted control plane (Tuskline Cloud) from v1; full self-hosting always | Accepted |
| [0003](adr/0003-licence-split.md) | Licence | AGPL-3.0 server; Apache-2.0 core, CLI and agent | Accepted |
| [0004](adr/0004-surfaces.md) | Surfaces | Desktop app as well as web and CLI | Accepted |
| [0005](adr/0005-compatible-systems.md) | Compatible systems | Only those needing the least effort, in three tiers | Accepted |
| [0006](adr/0006-minimum-postgresql-14.md) | Minimum version | PostgreSQL 14+ | Accepted |
| [0007](adr/0007-build-vs-reuse.md) | Build or reuse | Build collectors, metrics, linter and advisor; drive Patroni, etcd, PgBouncer, pgBackRest, CNPG | Accepted |
| [0008](adr/0008-solo-founder-scope.md) | Team | Solo founder plus AI tooling; three releases and a cut list | Accepted |
| [0009](adr/0009-desktop-wails-v3.md) | Desktop framework | Wails v3; Tauri v2 as the exit route | Accepted |
| [0010](adr/0010-licence-boundary.md) | Licence boundary in practice | Separate binaries, import rules, CC BY docs, DCO | Accepted |
| [0011](adr/0011-frontend-sveltekit-spa.md) | Web stack | SvelteKit SPA + Svelte 5 + Tailwind + shadcn-svelte; React prototype is the spec | Accepted |
| [0012](adr/0012-proto-and-moon.md) | Toolchain and tasks | proto pins Go, Node.js, npm, moon and golangci-lint; moon runs and caches every task; Makefile is a thin alias | Accepted |

Template: [adr/0000-template.md](adr/0000-template.md).
