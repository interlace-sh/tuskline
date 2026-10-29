# 0003. Licence split: AGPL-3.0 server, Apache-2.0 core, CLI and agent

- Status: Accepted
- Date: 2026-10-28

## Context

We want to stop a cloud provider from offering a modified Tuskline server without contributing back. We also want the CLI and agent usable where AGPL is banned: company laptops, CI, and customer networks.

## Decision

- **AGPL-3.0-only:** the server (multi-user API, RBAC, fleet history, alerting, provisioning orchestration), the web UI and the desktop app, which bundles the web UI.
- **Apache-2.0:** the `tl` CLI/TUI and `tl agent`.
- **Apache-2.0 for the core engine as well**: inventory, connections, collectors, SQL packs, analysers, linter, config advisor and provisioning planner. The Apache CLI and agent link it, so it cannot be AGPL.
- **Docs:** CC BY 4.0.

## Consequences

- Others may embed the core's checks and SQL packs, with a patent grant.
- The boundary must be enforced mechanically. See [0010](0010-licence-boundary.md).
- Some companies avoid AGPL on their servers. Tuskline Cloud and the Apache CLI and agent serve them.
- The desktop app is not distributed through the Mac App Store.
