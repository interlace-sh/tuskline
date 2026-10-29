# Compatibility

**PostgreSQL 14 and newer** ([ADR 0006](adr/0006-minimum-postgresql-14.md)). Systems are tiered by least effort ([ADR 0005](adr/0005-compatible-systems.md)): if a system keeps PostgreSQL's catalogues, statistics views and planner, it is in.

| System | Tier | Why | Adaptation needed |
|---|---|---|---|
| Community PostgreSQL 14–18 (self-managed, Patroni, CNPG) | 1 | The reference target | None |
| Amazon RDS for PostgreSQL | 1 | Stock PostgreSQL, no superuser; extensions through parameter groups | `rds_superuser` grants, parameter-group snippets, RDS IAM auth, CloudWatch logs |
| Aurora PostgreSQL | 1 | PostgreSQL front end on Aurora storage; catalogue and stats views mostly behave | Hide checkpoint and WAL-volume advice; replica lag from Aurora functions, not `pg_stat_replication` |
| Cloud SQL for PostgreSQL | 1 | Stock PostgreSQL; settings through database flags | `cloudsqlsuperuser`, flag snippets, Cloud SQL IAM auth |
| Azure Database for PostgreSQL Flexible Server | 1 | Stock PostgreSQL; server parameters through the Azure API | Allow-listed extensions, `az` snippets, Entra ID tokens |
| AlloyDB | 2 | PostgreSQL-compatible with Google's storage and columnar engine; supports `pg_stat_statements` and `hypopg` | Storage and I/O metrics differ; skip storage advice |
| Neon | 2 | Stock PostgreSQL compute over Neon storage | `pg_stat_statements` is lost when compute suspends, so snapshot often; no Patroni or backups to manage |
| TimescaleDB | 2 | An extension on normal PostgreSQL | Later: hypertable and chunk awareness |
| Citus | 2 | An extension; coordinator and workers are normal instances | Monitor each node; distributed views later |
| YugabyteDB | Out | Own distributed storage; documented gaps | Too much special-casing |
| CockroachDB, Redshift, other wire-compatible systems | Out | Speak the protocol, not the catalogues or planner | Not planned |

- Tier 1 is tested in CI from release 1: containers for community PostgreSQL 14–18, plus a nightly run against low-cost managed instances once M2 lands.
- Tier 2 arrives in release 2, with gaps documented here.

## Provider detection

`conn.Probe` (SQL in `core/sqlpack/probe.go`) classifies a server from fingerprints that managed services create and self-managed servers normally lack. More specific services win, because Aurora also has `rds_superuser`. These are heuristics, and the inventory can override the result.

| Provider | Fingerprints, checked in this order |
|---|---|
| Aurora | function `aurora_version()` |
| RDS | role `rds_superuser`, or setting `rds.extensions` |
| AlloyDB | role `alloydbsuperuser`, or setting `alloydb.iam_authentication` |
| Cloud SQL | role `cloudsqlsuperuser`, or setting `cloudsql.iam_authentication` |
| Azure Flexible Server | role `azure_pg_admin`, or setting `azure.extensions` |
| Neon | setting `neon.timeline_id`, or `neon` in `shared_preload_libraries` |
| Community | none of the above |

The probe also records `shared_preload_libraries` (readable only with `pg_read_all_settings`, which `pg_monitor` includes; otherwise reported as unknown), the installed and available extensions, and the login role's attributes and memberships (`pg_monitor`, `pg_read_all_settings`, `pg_read_all_stats`, `pg_signal_backend`).

## Version-specific features SQL packs rely on

| Feature | Available from |
|---|---|
| `pg_stat_wal`, `pg_stat_replication_slots`, `compute_query_id`, `pg_locks.waitstart` | 14 |
| `pg_stat_io` | 16 |
| `pg_stat_checkpointer` (replaces checkpoint columns of `pg_stat_bgwriter`) | 17 |

Add a row whenever a SQL pack branches on a version, and cover the branch with an integration test on every major.

## Client platforms

- `tl` and `tuskline`: Linux, macOS and Windows, on amd64 and arm64 (pure Go, cross-compiled in CI).
- Desktop: macOS (Intel and Apple Silicon), Windows x64/arm64, Linux x64/arm64 (WebKitGTK).
- Browsers: current Chrome, Edge, Firefox and Safari.
