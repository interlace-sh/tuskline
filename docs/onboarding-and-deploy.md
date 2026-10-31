# Onboarding and deploy

The fastest way to lose a new user is a dashboard full of "extension not installed". Onboarding detects what is missing and generates the exact steps for that server and provider. Deploy offers a guided plan-and-apply path that flows straight into monitoring, backups and HA.

## Connection wizard (M4)

1. **Connect.** Paste a URL, pick from `~/.pgpass` or `pg_service.conf`, or discover from AWS, GCP, Azure or Kubernetes. Choose the TLS mode, an SSH hop or cloud IAM.
2. **Classify.** Name the environment (local, dev, staging, prod) and set its policy. Prod defaults to read-only.
3. **Detect.** Version, provider (from settings and functions such as `aurora_version()`), HA manager, pooler, extensions, preload libraries, and the login role's privileges.
4. **Fix.** For each gap, show what it unlocks, the generated SQL and conf change, whether a restart is needed, and the provider-specific route (parameter group, database flag, server parameter).
5. **Verify.** Re-check after the change and show the first diagnosis.

## Extensions Tuskline looks for

| Extension | Used for | Preload? | Notes |
|---|---|---|---|
| `pg_stat_statements` | Query insights, load attribution, snapshots | Yes (restart) | **Required.** Offered by all tier 1 and 2 services |
| `auto_explain` | Captured plans for slow queries | Yes, or `LOAD` per session | Recommended; threshold such as 500 ms |
| `pg_wait_sampling` | Wait-event history between samples | Yes (restart) | Optional; Cloud SQL behind a flag |
| `pg_stat_kcache` | CPU and real disk I/O per query | Yes (restart); needs `pg_stat_statements` | Optional; mostly self-managed |
| `hypopg` | Validate index advice with hypothetical indexes | No | Optional; per database |

## Generated setup (self-managed example)

`core/catalog` holds the entries and builds the steps. The steps merge the current `shared_preload_libraries` value, because `ALTER SYSTEM` replaces the whole list, and they do not emit that statement when the current list could not be read. They switch to `patronictl edit-config` when Patroni manages the configuration. `InstallPlan` turns that into the package commands and the SQL preview.

```sql
ALTER SYSTEM SET shared_preload_libraries = '<current>,pg_stat_statements,auto_explain';
ALTER SYSTEM SET pg_stat_statements.track = 'top';
ALTER SYSTEM SET track_io_timing = on;
ALTER SYSTEM SET auto_explain.log_min_duration = '500ms';
ALTER SYSTEM SET auto_explain.log_format = 'json';
-- restart, then in each monitored database:
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE EXTENSION IF NOT EXISTS hypopg;   -- optional
```

## Monitoring role

`catalog.MonitorPlan` previews the collector role. The script does not set a password. On RDS and Aurora it is run as a member of `rds_superuser`, on Cloud SQL as `cloudsqlsuperuser`, on AlloyDB as `alloydbsuperuser`, and on Azure as `azure_pg_admin`. Extension parameter groups, Cloud SQL flags and `az postgres flexible-server parameter set` stay in `InstallPlan`; the role script does not repeat them.

```sql
CREATE ROLE tuskline_monitor LOGIN CONNECTION LIMIT 5; -- password via \password or IAM
GRANT pg_monitor TO tuskline_monitor;
ALTER ROLE tuskline_monitor SET default_transaction_read_only = on;
ALTER ROLE tuskline_monitor SET statement_timeout = '5s';
ALTER ROLE tuskline_monitor SET lock_timeout = '1s';
ALTER ROLE tuskline_monitor SET idle_in_transaction_session_timeout = '10s';
-- Optional role for actions: GRANT pg_signal_backend TO tuskline_operator;
```

## How extensions are installed

Tuskline keeps a catalog and drives the install. It does not build extension binaries or run its own package repository. [Autobase](https://autobase.tech/docs/extensions/install) and [Pigsty](https://pigsty.io/ext/) already solved delivery: PGDG packages where they exist, the Pigsty repository for the rest of the ecosystem, then `shared_preload_libraries` and `CREATE EXTENSION`. Tuskline uses that order.

1. **Package, then SQL**, on anything with a host package manager (Compose, a VM over SSH, Patroni). `catalog.InstallPlan` previews that sequence and does not run it. The package name is the PGDG name, `postgresql-<major>-<ext>` (for example `postgresql-16-pg-stat-kcache`). If a catalogue entry says PGDG does not ship it, the plan says to add the Pigsty DEB or RPM repository, which uses the same package names. `hypopg`, `pg_stat_kcache` and `pg_wait_sampling` are PGDG packages. Autobase's "auto-setup" switches are this same sequence for a short list, not a different installer. Managed clouds get the provider command, or a line that the extension is not offered. The plan never compiles from source.
2. **Allow-list, then SQL**, on managed PostgreSQL. RDS, Aurora, Cloud SQL, Azure, Neon and AlloyDB do not accept `apt` or `dnf`. The catalog records which of them offer the extension. The step is their parameter group, database flag or `azure.extensions` change, then `CREATE EXTENSION`, or a straight "not offered here".
3. **No compiling from source** in the default path. Autobase allows a pre-deploy script that builds an extension; that is slow and hard to repeat. If neither PGDG nor Pigsty has a package, the plan says so and stops.

A catalog entry is metadata: what the extension unlocks, the PostgreSQL 14–18 builds that exist, whether it needs a preload and a restart, the package name, the provider allow-list, and any settings (`pg_stat_statements.track`, `auto_explain.log_min_duration`). The first cut is the five extensions in the table above, plus the ordinary single-node extensions Autobase turns on with a flag: pgvector, PostGIS, pgRouting, pg_cron, pgaudit, pg_partman and pg_repack. TimescaleDB and Citus stay compatible systems (tier 2), not one-click installs. Pigsty's catalog of several hundred extensions is the index for everything else; Tuskline does not copy that list in.

`moon run :up` builds that image for PostgreSQL 14–18: the official image plus the PGDG packages for `hypopg`, `pg_stat_kcache` and `pg_wait_sampling`, preloaded where they must be. The 14–18 integration tests stay on the stock images.

Managed services have no superuser and no `postgresql.conf`. Tuskline emits the provider's own commands, warns about static parameters that need a reboot, and offers Terraform equivalents:

- RDS and Aurora: parameter groups and `rds_superuser`.
- Cloud SQL: database flags and `cloudsqlsuperuser`.
- Azure: `az` server parameters and allow-listed extensions.

## Config advisor (`tl config advise`, M4)

The rules are [ADR 0014](adr/0014-config-advisor.md). PgTune supplies the candidate values from RAM, CPUs, storage and workload. The parameter reference supplies the reload or restart context, the 25% and 40% `shared_buffers` bounds, and the fact that `effective_cache_size` allocates nothing. Live statistics are quoted beside a change and are not a second formula. `wal_level = minimal` is never suggested, and `max_connections` is never raised. Without `--memory` the command still reports `autovacuum` off, `track_io_timing` off, a zero idle-in-transaction timeout, a low checkpoint target and `wal_compression` off. Per-table autovacuum scale factors are later work.

## Deploy targets (M5)

| Target | What Tuskline deploys | How | Production HA? |
|---|---|---|---|
| Docker Compose (R2) | Patroni ×3, etcd ×3, PgBouncer, HAProxy, pgBackRest as containers | Generated `compose.yaml`, `.env`, `haproxy.cfg`; `tl provision compose up` with health-gated start-up | **No:** one host is a single point of failure. For dev, CI, staging and demos |
| Kubernetes (R3) | CloudNativePG `Cluster` (failover through the Kubernetes API, so no Patroni or etcd), CNPG `Pooler`, Barman Cloud plugin or volume snapshots | Generated manifests or Helm values; server-side apply; status read back | Yes |
| VMs and bare metal over SSH (R3) | Patroni + etcd (3 nodes) + PgBouncer + HAProxy or vip-manager + pgBackRest to S3, GCS, Azure or NFS | Go SSH executor running idempotent, versioned steps (PGDG packages, systemd units, templates); optional Autobase-playbook mode | Yes, with 3+ hosts across failure domains |

## Guided flow

1. **Target:** hosts (SSH), local Docker or a Kubernetes context.
2. **Topology:** PostgreSQL 14–18; 1, 2 or 3 nodes; synchronous or asynchronous replicas; pooler on or off.
3. **Backups:** repository (S3, GCS, Azure, NFS or local volume), schedule, retention, encryption.
4. **Pre-flight:** SSH reachability, OS and package repositories, ports, clock sync, disk space and latency, existing data directories.
5. **Plan:** every file, package, unit and manifest shown as a diff before anything runs.
6. **Apply:** streamed log, resumable on failure; secrets written to the chosen secret store.
7. **Hand-over:** the cluster is added to the inventory with a monitoring role, the Patroni API and backup repository are registered, and the first restore drill is scheduled.

`core/provision` is the plan and apply engine. `Changes` is the diff of every file, unit and manifest. `Apply` writes those paths, streams a log, and records each finished step so a later call resumes after a failure. A step whose body has changed runs again. Compose marked as production is refused. A step that names a data directory is refused when that directory already has files. A secret is redacted in the diff and is not written: there is no secret store in this engine yet. Nothing runs until `Apply` is called, and a command step does not start a shell unless the caller supplies an executor.

Safety: refuse to touch non-empty data directories, show diffs before applying, and label Compose as non-production.

`deploy/compose/compose.yaml` in this repository is only for dev and test databases. It is not the generated HA topology.
