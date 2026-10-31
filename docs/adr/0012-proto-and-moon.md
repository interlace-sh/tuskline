# 0012. proto + moon for toolchain and tasks

- Status: Accepted
- Date: 2026-10-29

## Context

Tuskline is a Go workspace (`core`, `agent`, `cli`, `server`) plus a SvelteKit app (`web`) whose build is embedded into the server. Until now a `Makefile` looped over the Go modules and CI repeated the same steps by hand, with its own Go, Node.js and golangci-lint versions. That had three problems:

- **Drift.** Versions lived in four places (`go.mod`, `actions/setup-go`, `actions/setup-node`, `golangci-lint-action`) and contributors installed whatever they had. AI agents in particular run whatever `go` or `node` is on `PATH`.
- **Implicit ordering.** "Build the web UI before the server" was a convention (`make web` then `make bin`), not something the tooling knew.
- **No caching or affected detection.** Every change ran everything.

## Decision

- **proto** ([moonrepo.dev/proto](https://moonrepo.dev/proto)) pins every tool in the root `.prototools`: Go, Node.js, npm, moon itself and golangci-lint. `proto install` sets up a machine; proto's shims select the pinned version inside the repository.
  - Go, Node.js, npm and moon are proto built-in plugins.
  - golangci-lint has no built-in or official community plugin. It is listed in proto's third-party registry ([b4nst/proto-plugins](https://github.com/b4nst/proto-plugins)); we **vendor** that TOML plugin as `.proto/plugins/golangci-lint.toml` so it is pinned and reviewable rather than fetched from someone's `main` branch. It downloads golangci-lint's official GitHub release archives and verifies them against the release checksums. Since proto 0.63, third-party plugins in a repository config only load after `proto trust`, so contributors run it once per clone; CI is trusted automatically.
- **moon** ([moonrepo.dev/moon](https://moonrepo.dev/moon), v2) runs every task.
  - `.moon/workspace.yml` lists the five projects explicitly. `.moon/toolchains.yml` enables the `go`, `javascript`, `node` and `npm` toolchains and inherits their versions from `.prototools`, so there is one place to bump a version.
  - Go is a first-class moon v2 toolchain (a WASM plugin maintained by moonrepo), not the generic `system` toolchain: moon understands `go.mod`, `go.sum` and `go.work`, hashes dependencies, runs `go mod download`, and installs Go through proto. Relationship inference from `go list` is turned off; `dependsOn` in each `moon.yml` states the graph (`cli → core, agent`; `server → core`, and `server → web` at build scope), which keeps the licence boundary from [0010](0010-licence-boundary.md) visible in configuration.
  - Go tasks shared by every module live in `.moon/tasks/go.yml` (`build`, `test`, `vet`, `fmt-check`, `golangci`, `lint`, `fmt`, `tidy`, `check`). Each declares inputs, including the sources of upstream projects, so results are cached and `--affected` works. Projects add their own tasks: `core:integration` (tagged `integration`, `docker`; never cached; skipped by `moon ci`), `cli:bin`, `server:bin`, `cli:cross`, `server:cross`, `server:embed-web` and the web tasks.
  - `server:build` and `server:bin` depend on `server:embed-web`, which depends on `web:build` (outputs `web/build/`). The UI is therefore always rebuilt when it changed and reused from cache when it did not.
  - moon's npm toolchain installs `web/` dependencies on demand (`npm ci` in CI), so a fresh clone needs no separate install step.
- **CI** uses `moonrepo/setup-toolchain` (which runs `proto install`) and `moon ci`, which runs only affected tasks with `runInCI`. The PostgreSQL 14–18 integration matrix stays as separate jobs running `moon run core:integration` with `PG_VERSIONS` set per job.
- **The `Makefile` stays as a thin alias layer.** Every target is a single moon command and adds no behaviour. It keeps muscle memory, existing docs and `make help` working, and costs almost nothing to maintain. If it ever needs logic, that logic belongs in a moon task instead.

## Consequences

- One `.prototools` change bumps a tool everywhere: laptops, agents and CI. Renovate can update it.
- Contributors need proto installed first (one `curl | bash` or the Windows installer), and must run `proto trust` once for the golangci-lint plugin. Without trust, the `golangci` tasks (and so `lint` and `check`) stop with a hint to run `proto trust && proto install`; everything else still works.
- moon adds a cache directory (`.moon/cache/`, ignored by git) and a second configuration format to learn. Tasks must keep accurate `inputs`; an input missing from a task means a stale cached pass. When in doubt, run with `--force` or `-u` to bypass the cache.
- `go.mod` keeps `go 1.25.0` as the minimum language version for library consumers; `.prototools` pins the toolchain developers and CI actually use.
- Not adopted yet: `.protolock` checksums for every platform, moon remote caching and moon-managed Git hooks. Revisit once CI timings justify them (roadmap M0.4a).

## Notes

- 2026-10-29 (M0.4): on a runner, `runInCI: false` also stops an explicit `moon run`, so the first integration matrix reported "No tasks found". `core:integration` now uses `runInCI: 'always'`, and the moon job runs `moon ci :check :cross` so the Docker tests stay out of it. A task that needs Docker therefore uses `runInCI: 'always'` and is left out of the `moon ci` targets.
- 2026-10-29 (M0.4a): moon remote caching and `.protolock` stay off. A local `moon run :check` finishes in a few seconds once the cache is warm, which does not justify a remote cache service or checksums for every platform. Revisit when a CI run is slow enough that the cache misses cost more than the service.
- 2026-10-31: the dev databases are `moon run :up` and `moon run :down` (the `compose` project). `make dev-db` only calls `:up`. Integration tests still start their own containers.
