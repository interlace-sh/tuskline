# Licensing

Tuskline uses a split licence (ADR [0003](docs/adr/0003-licence-split.md) and [0010](docs/adr/0010-licence-boundary.md)).

| Directory | Licence | File | SPDX header |
|---|---|---|---|
| `core/` | Apache License 2.0 | [core/LICENSE](core/LICENSE) | `// SPDX-License-Identifier: Apache-2.0` |
| `cli/` | Apache License 2.0 | [cli/LICENSE](cli/LICENSE) | `// SPDX-License-Identifier: Apache-2.0` |
| `agent/` | Apache License 2.0 | [agent/LICENSE](agent/LICENSE) | `// SPDX-License-Identifier: Apache-2.0` |
| Everything else (`server/`, `web/`, `desktop/`, `deploy/`, `prototype/`, build files) | GNU AGPL v3 only | [LICENSE](LICENSE) | `// SPDX-License-Identifier: AGPL-3.0-only` |
| `docs/` | Creative Commons Attribution 4.0 | [docs/LICENSE](docs/LICENSE) | n/a |
| `brand/` | Licensed under AGPL-3.0 as files, but the name and logo are trademarks (see below) | | |

The nearest `LICENSE` file up the directory tree applies.

## Why the split

- **The server is AGPL-3.0.** This covers the multi-user API, RBAC, fleet history, alerting, provisioning orchestration, the web UI and the desktop app, which bundles the web UI. Anyone who offers a modified Tuskline server to users over a network must publish their changes. Tuskline Cloud runs this same AGPL server, and every Cloud feature is in the self-hosted build.
- **The CLI and agent are Apache-2.0**, so they can be used on company laptops, in CI and inside customer networks where AGPL is banned.
- **The core is Apache-2.0** because the CLI and agent link it. An Apache binary cannot link AGPL code without the result becoming AGPL.

## Rules that keep the boundary intact

1. Apache modules (`core`, `cli`, `agent`) must **never import** `server`, `web` or `desktop`. CI and review enforce this. AGPL code may import the core, never the reverse.
2. `tl serve` does not link the server: it **executes the separate `tuskline` binary** as a child process (ADR 0010).
3. Code moves from AGPL to Apache only if the copyright holder relicenses it explicitly. Never copy AGPL code into `core/`, `cli/` or `agent/`.
4. Third-party dependencies of Apache modules must be permissive (MIT, BSD, Apache-2.0, ISC, MPL-2.0 used as a library). Do not add GPL or AGPL dependencies there.
5. Every new Go file starts with the SPDX header for its directory.

## What this means for you

- Using Tuskline (CLI, desktop, self-hosted server) personally or inside your company: no obligations.
- Embedding the core or CLI in your product: Apache-2.0 terms (notice and licence; patent grant included).
- Running a **modified** server for others over a network: offer them the modified source (AGPL §13).
- The desktop app is not distributed through the Mac App Store, whose terms are widely seen as incompatible with GPL-family licences.

## Contributions

Use the [Developer Certificate of Origin](https://developercertificate.org/): sign off every commit with `git commit -s`. There is no CLA. Contributions are accepted under the licence of the directory they change.

## Trademarks

"Tuskline" and the Tusk Arc logo identify the official project. The licences above cover code and docs, not the right to present a fork as Tuskline. A trademark search (UKIPO, EUIPO and USPTO, classes 9 and 42) is still to be done before launch.
