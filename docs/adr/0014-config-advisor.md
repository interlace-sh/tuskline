# 0014. Config advisor

- Status: Accepted
- Date: 2026-10-31

## Context

`tl config advise` has to suggest `postgresql.conf` changes with a reason, the current value, a proposed value, and whether the change needs a reload or a restart. Two public sources already cover parts of this:

- [PgTune](https://github.com/le0pard/pgtune) (the calculator at [pgtune.leopard.in.ua](https://pgtune.leopard.in.ua/)) turns RAM, CPU count, storage, workload and database size into a starting configuration. The arithmetic lives in `configurationSelectors.js`. It does not look at the running server, and it prints a full configuration, including values PostgreSQL 14+ already uses.
- [postgresqlco.nf](https://postgresqlco.nf/tuning-guide) publishes the parameter reference: type, unit, context (reload or restart) and the manual's guidance. The parts that change a recommendation are specific. `shared_buffers` starts at 25% of RAM on a dedicated server and is unlikely to help above 40%, because PostgreSQL also uses the operating system cache. `effective_cache_size` allocates nothing; it is the planner's estimate of that cache. `autovacuum` stays on. `checkpoint_completion_target` stays at 0.9, which has been the default since PostgreSQL 14.

A starting configuration and a dictionary are not yet advice. Advice is a change that the current setting and, where we have them, the live statistics support.

## Decision

`core/advisor` reimplements the PgTune formulas for PostgreSQL 14–18 and compares them with `pg_settings`. A setting is left alone when it is already within 20% of the candidate, or when it is a `shared_buffers` value between the candidate and 40% of RAM.

The formulas run only for the inputs the operator actually gave. `--memory` enables the memory settings. `--storage` enables the I/O cost settings. `--cpus` enables parallel workers and, on a large machine, autovacuum workers. With none of those, the command still reports settings that are simply wrong on a server Tuskline is asked to operate: `autovacuum` off, `track_io_timing` off, `idle_in_transaction_session_timeout` at 0, `checkpoint_completion_target` below 0.9, and `wal_compression` off.

Live statistics are evidence, not a second formula. A low cache hit ratio is reported beside a `shared_buffers` change and is not treated as proof. Temporary-file bytes are reported beside `work_mem`. `max_wal_size` moves only when requested checkpoints outnumber timed ones.

Divergences from PgTune, on purpose:

- `wal_level = minimal` is never suggested. It disables replication and point-in-time recovery.
- `max_connections` is not raised. `work_mem` is sized for the larger of the live setting and the workload's connection count, so a later increase does not silently exhaust RAM. A setting above 200 without `--pooler` is a note. A setting that does not leave 4MB of `work_mem` per connection is a restart recommendation to lower it, after a pooler is in place.
- `wal_buffers` stays at `-1`. PostgreSQL already derives it from `shared_buffers` and caps it at 16MB, which is what PgTune would write explicitly.
- `io_method = io_uring` is suggested on PostgreSQL 18 only when the running build lists it. The note says containers often block it.
- `wal_compression = lz4` is suggested on PostgreSQL 15+ only when the build lists `lz4`. An existing `zstd` value is left as it is.

Nothing in this package is applied. The text is a preview.

## Consequences

- `tl config advise` is the command. The config advisor screen (M4.5) consumes the same report.
- The formulas are a snapshot of PgTune as of 2026-10-31, checked by a fixture: 16GB RAM, 8 CPUs, SSD, web workload, PostgreSQL 17. A future PgTune change is adopted only by changing that fixture deliberately.
- Host RAM is not visible from SQL. Without `--memory`, memory settings are omitted rather than guessed.
- Per-table autovacuum scale factors and index advice stay later work. They need table statistics and hypopg, not a hardware formula.
