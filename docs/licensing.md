# Licensing (developer notes)

The summary for users is in [../LICENSING.md](../LICENSING.md). The reasoning is in ADRs [0003](adr/0003-licence-split.md) and [0010](adr/0010-licence-boundary.md).

## Checklist when you add code

- New Go file in `core/`, `cli/` or `agent/`: start it with `// SPDX-License-Identifier: Apache-2.0`.
- New Go file in `server/` or `desktop/`: start it with `// SPDX-License-Identifier: AGPL-3.0-only`.
- New Go module: give it its own `LICENSE`, add it to `go.work`, to the Makefile's `MODULES`, and to the CI matrix.
- Never import `server` (or anything AGPL) from `core`, `cli` or `agent`. `tl serve` execs `tuskline`.
- Dependencies of Apache modules must be MIT, BSD, Apache-2.0, ISC or similar. Check the dependency's licence before adding it. M0.6 adds an automated check.
- Copying code from elsewhere: keep its notice, make sure its licence is compatible with the target directory, and mention it in the PR.
- Docs are CC BY 4.0.
- Commits carry `Signed-off-by` (DCO).

## Licence texts

The texts are the official ones, fetched verbatim:

| File | Text |
|---|---|
| `LICENSE` | GNU AGPL v3, from gnu.org |
| `core/LICENSE`, `cli/LICENSE`, `agent/LICENSE` | Apache 2.0, from apache.org |
| `docs/LICENSE` | CC BY 4.0 legal code, from creativecommons.org |
