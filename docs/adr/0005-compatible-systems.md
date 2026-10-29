# 0005. Compatible systems: least effort only

- Status: Accepted
- Date: 2026-10-28

## Context

Many systems claim PostgreSQL compatibility. Supporting systems that replace the storage engine or planner would multiply the special cases in every SQL pack and advisor.

## Decision

The rule: if a system keeps PostgreSQL's catalogues, statistics views and planner, it is in. If it replaces the storage engine's behaviour or the planner, it is out.

- **Tier 1** (CI from release 1): community PostgreSQL 14–18, Amazon RDS, Aurora PostgreSQL, Cloud SQL, Azure Flexible Server.
- **Tier 2** (release 2, gaps documented): AlloyDB, Neon, TimescaleDB, Citus.
- **Out:** YugabyteDB, CockroachDB, Redshift and other wire-compatible systems.

## Consequences

- Provider detection is part of `conn.Probe`. Advice that does not apply is hidden, for example checkpoint advice on Aurora.
- See [compatibility.md](../compatibility.md) for the adaptations each system needs.
