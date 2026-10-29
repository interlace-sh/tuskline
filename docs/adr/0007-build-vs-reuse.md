# 0007. Build where it gives performance or independence; drive mature engines

- Status: Accepted
- Date: 2026-10-28

## Context

HA, consensus, pooling and backup engines are mature and trusted, so rewriting them would be reckless. Monitoring and advice, by contrast, are where Tuskline differentiates.

## Decision

| Area | Choice | Reason |
|---|---|---|
| Collectors and SQL packs | Build | Overhead control (query budgets, back-off) and one metric model shared by UI, CLI and alerts |
| Metric definitions | Build | One documented catalogue; Prometheus names exported for compatibility |
| Migration linter | Build on `pg_query_go` | Lock analysis needs table sizes and live activity |
| Config advisor, plan visualiser, provisioning planner | Build | Core differentiators; small, testable Go packages |
| SQL parser | Reuse libpg_query (`pg_query_go`) | Rewriting the grammar gains nothing |
| HA, consensus, pooling, backup | Drive Patroni, etcd, PgBouncer, pgBackRest, CloudNativePG | Configure, observe and operate them through their APIs |
| Editor, charts, analytics | Reuse CodeMirror 6, uPlot, DuckDB | |
| VM provisioning | Build a Go SSH executor, with an Autobase mode | Single binary; Autobase users can reuse their playbooks |

## Consequences

- No own HA manager, backup engine or Kubernetes operator, ever.
- Operator-managed clusters are acted on through the operator, never around it.
