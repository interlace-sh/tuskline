# Vision

**The calm, trustworthy cockpit for every PostgreSQL you run, from a laptop container to a multi-cloud fleet.**

Teams running PostgreSQL today juggle many tools:

- A SQL client.
- A monitoring stack.
- pganalyze or Performance Insights.
- Patroni and pgBackRest command lines.
- A migration tool.
- Ad hoc scripts.

Tuskline is one open-source product that brings these together around one inventory. It is one Go core with four surfaces: desktop app, web UI, `tl` CLI/TUI, and agent. A hosted control plane runs the same server.

## Personas

| Persona | Job to be done | First screens they live in |
|---|---|---|
| Application developer | Write and tune queries; understand why prod is slow | SQL IDE, plan visualiser, query insights |
| DBA / DBRE | Keep many clusters healthy; tune, vacuum, index, recover | Fleet, cluster health, advisors, backups |
| Platform / SRE | Operate HA and backups across clouds and Kubernetes; get paged less | Fleet, HA topology, alerts, Prometheus export |
| Tech lead / reviewer | Ship schema changes safely | Migration review, drift |
| Consultant / MSP | Diagnose unfamiliar estates fast | `tl diagnose` reports, snapshots, read-only mode |
| Data analyst | Explore data safely on read replicas | SQL IDE, charts, saved queries |

## One product, two front doors

| | Developer door | DBA / SRE door |
|---|---|---|
| Way in | Desktop app or `tl connect`; one database, local-first, no server | `docker run`, Helm or Tuskline Cloud; many clusters, shared with the team |
| Home screen | Database view: SQL editor, recent queries, health summary | Fleet overview by environment |
| First wow | EXPLAIN visualiser and query insights on their own app's queries | Fleet health and diagnosis across every environment in five seconds |

- The home screen follows what you connect: one database opens the database view, and three or more clusters switch the default to Fleet.
- Features appear when their data exists. The HA tab appears only when Patroni or CNPG is detected, and Backups only when a repository is found.
- One navigation and one command palette. There is no persona switch and there are no separate editions.

## Differentiators

- One inventory, many surfaces: UI, CLI, TUI, API and MCP share one core and one set of permissions.
- Diagnoses, not dashboards: every finding has evidence, the SQL behind it and a fix.
- Environment-aware safety is designed in, not bolted on.
- Snapshots and history together: instant value with zero storage, and deeper value when left running.
- Orchestrates the best open-source engines (Patroni, CNPG, pgBackRest, Flyway) rather than competing with them.
- Truly open: AGPL-3.0 server and Apache-2.0 core, CLI and agent. Everything in Tuskline Cloud is in the self-hosted build.
- Design quality on a par with Linear and Raycast.

## Design principles

1. **Calm by default, loud when it matters.** Healthy things recede; problems rise to the top with words, not just red.
2. **Every number explains itself.** Hover for its definition, source view and SQL; click through to the evidence.
3. **Always know where you are.** An environment colour rail and breadcrumb on every screen; prod never looks like dev.
4. **Keyboard first, mouse friendly.** Command palette, mnemonic shortcuts, everything reachable without a mouse.
5. **Fast is a feature.** Interactions under 100 ms, virtualised grids, optimistic UI, streaming results.
6. **One mental model.** The same nouns and verbs in the UI, CLI and API.
7. **Progressive disclosure.** A simple first view, with depth one click away.
8. **Accessible.** WCAG 2.2 AA, colour-blind-safe palettes, reduced motion, full screen-reader labels.

## Business model

Tuskline Cloud runs the same AGPL server for teams, starting in v1. Agents in customer networks dial out, so database credentials stay with the customer.

- The free tier covers a few clusters.
- Paid tiers are priced by cluster count and retention.
- Enterprise adds SAML, SCIM, audit streaming and SLAs.

Prices will be set after the beta. Self-hosting is always complete.

## Release 1 success measure

From install to a useful diagnosis of an unfamiliar database in **under five minutes**, needing no extension beyond `pg_stat_statements`.
