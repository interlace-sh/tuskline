# Contributing to Tuskline

Thanks for helping. Tuskline is pre-alpha and run by one maintainer, so small, focused pull requests that follow the roadmap get merged fastest.

## Before you start

1. Read [AGENTS.md](AGENTS.md). It holds the rules for humans as well as AI agents.
2. Pick a task from [docs/roadmap.md](docs/roadmap.md), or open an issue first for anything not on it.
3. Architectural changes need a new ADR in `docs/adr/` (copy [0000-template.md](docs/adr/0000-template.md)).

## Set up

Tool versions are pinned in [`.prototools`](.prototools) and installed by [proto](https://moonrepo.dev/proto); tasks run through [moon](https://moonrepo.dev/moon) ([ADR 0012](docs/adr/0012-proto-and-moon.md)).

1. Install proto: `bash <(curl -fsSL https://moonrepo.dev/install/proto.sh)` (or see the [install guide](https://moonrepo.dev/docs/proto/install), including Windows).
2. From the repository root:

```sh
proto trust              # once: allows the vendored golangci-lint plugin in .prototools
proto install            # Go, Node.js, npm, moon and golangci-lint at the pinned versions
moon run :check          # build, test and lint everything (installs web deps on first run)
moon run core:integration   # needs Docker; PostgreSQL 14-18
```

You also need Docker for integration tests and `make dev-db`. `make` targets still work as short aliases for the moon commands (`make help`).

## Pull requests

- One logical change per PR. Use a [Conventional Commits](https://www.conventionalcommits.org/) title, for example `feat(cli): add tl ls`.
- **Sign off every commit** with `git commit -s` under the [DCO](https://developercertificate.org/). There is no CLA.
- Tests and docs come in the same PR. Tick the roadmap checkbox when a task is complete.
- `moon run :check` (or `make check`) must pass locally. CI also runs integration tests against PostgreSQL 14-18.
- UI changes include a screenshot (dark theme) and follow [docs/design.md](docs/design.md).
- Respect the licence boundary in [LICENSING.md](LICENSING.md): `core/`, `cli/` and `agent/` never import AGPL code.

## Style

British English in docs, UI text and comments ("licence" as a noun, "colour", "organisation"). Code identifiers follow the language's conventions: use `color` in CSS, and match the spelling of existing APIs.

## Conduct

Be kind and assume good faith. Harassment is not tolerated. The maintainer may remove content or contributors that break this rule.
