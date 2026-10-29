# Contributing to Tuskline

Thanks for helping. Tuskline is pre-alpha and run by one maintainer, so small, focused pull requests that follow the roadmap get merged fastest.

## Before you start

1. Read [AGENTS.md](AGENTS.md). It holds the rules for humans as well as AI agents.
2. Pick a task from [docs/roadmap.md](docs/roadmap.md), or open an issue first for anything not on it.
3. Architectural changes need a new ADR in `docs/adr/` (copy [0000-template.md](docs/adr/0000-template.md)).

## Set up

- Go 1.25+.
- Node.js 22.17+ and npm.
- Docker, for integration tests and `make dev-db`.
- Optional: golangci-lint v2.

```sh
make build test          # Go
make web-install lint-web test-web
make test-integration    # needs Docker; PostgreSQL 14-18
```

## Pull requests

- One logical change per PR. Use a [Conventional Commits](https://www.conventionalcommits.org/) title, for example `feat(cli): add tl ls`.
- **Sign off every commit** with `git commit -s` under the [DCO](https://developercertificate.org/). There is no CLA.
- Tests and docs come in the same PR. Tick the roadmap checkbox when a task is complete.
- `make check` must pass locally. CI also runs integration tests against PostgreSQL 14-18.
- UI changes include a screenshot (dark theme) and follow [docs/design.md](docs/design.md).
- Respect the licence boundary in [LICENSING.md](LICENSING.md): `core/`, `cli/` and `agent/` never import AGPL code.

## Style

British English in docs, UI text and comments ("licence" as a noun, "colour", "organisation"). Code identifiers follow the language's conventions: use `color` in CSS, and match the spelling of existing APIs.

## Conduct

Be kind and assume good faith. Harassment is not tolerated. The maintainer may remove content or contributors that break this rule.
