# 0010. Keeping the licence boundary intact

- Status: Accepted
- Date: 2026-10-29

## Context

[0003](0003-licence-split.md) splits AGPL-3.0 (server, web, desktop) from Apache-2.0 (core, CLI, agent). The page's CLI spec includes `tl serve`, which starts the web UI and API. If the Apache CLI linked the AGPL server, the `tl` binary would become AGPL as a whole.

## Decision

1. **Separate Go modules** per licence, each with its own `LICENSE`: `core`, `cli` and `agent` are Apache; `server` is AGPL. Web and desktop are AGPL under the root `LICENSE`.
2. **Import rule:** Apache modules never import AGPL modules. AGPL modules may import `core`. Review enforces this; add an import check to CI in M0.
3. **Two binaries:** `tl` (Apache) and `tuskline` (AGPL server). `tl serve` finds `tuskline` on `PATH` and runs it as a child process, passing arguments and signals through. The Docker image runs `tuskline serve` directly. Distributions ship both binaries.
4. **SPDX headers** on every source file.
5. **Docs** are CC BY 4.0 (`docs/LICENSE`).
6. **Contributions** use the DCO (`Signed-off-by`), with no CLA. This means the project cannot relicense contributors' code unilaterally, which is a deliberate trust signal.
7. Apache modules take only permissively licensed dependencies.

## Consequences

- Users who install only `tl` get a clear message from `tl serve` if `tuskline` is missing.
- The desktop app (AGPL) may link both the core and the server handler in-process.
- Relicensing later would need every contributor's consent. We accept that.
