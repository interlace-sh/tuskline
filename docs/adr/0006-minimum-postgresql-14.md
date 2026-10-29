# 0006. Minimum version: PostgreSQL 14

- Status: Accepted
- Date: 2026-10-28

## Context

PostgreSQL 13 reached community end of life in November 2025. Each extra supported major adds branches to SQL packs and integration tests.

## Decision

Support **PostgreSQL 14 and newer** (14–18 today). Refuse older servers with a clear message (`conn.MinServerVersionNum = 140000`).

## Consequences

- PostgreSQL 14 gives `pg_stat_wal`, `pg_stat_replication_slots`, query ID in core (`compute_query_id`) and `pg_locks.waitstart`, so SQL packs are simpler.
- The CI matrix tests 14, 15, 16, 17 and 18 in containers. New majors are added when released, and the oldest is dropped at its end of life (by ADR).
