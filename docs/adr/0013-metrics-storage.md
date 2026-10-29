# 0013. Metrics history is Parquet segments

- Status: Accepted
- Date: 2026-10-29

## Context

Collectors sample every target: activity about once a second, and statistics views (`pg_stat_statements`, database, tables, I/O, WAL, checkpointer and bgwriter) more slowly. One cluster at one sample a second is on the order of 10^5 activity rows a day before rollup, and statement statistics are wide. That history has to live in the server and the desktop app with no external time-series database ([AGENTS.md](../../AGENTS.md)). `core` and `server` cannot use cgo; DuckDB is allowed only in `desktop`.

Three shapes were considered:

1. **SQLite tables**, beside the metadata database. One file, one driver, and the pure-Go driver is already required. A row-oriented store of this volume shares a lock with the metadata the UI reads, and compaction is a long transaction.
2. **A private segment format.** Full control of the layout, and another format for every other tool to learn.
3. **Parquet files.** Columnar, compressed, and readable by DuckDB in the desktop app and by anything else later. Pure-Go readers and writers exist, so the server stays cgo-free.

Retention is already set in [architecture.md](../architecture.md): 1 s raw for 1 hour, 1 minute rollups for 14 days, 1 hour rollups for 13 months.

## Decision

- Metrics history is a directory of Parquet files. The metadata SQLite database does not store samples.
- `core` writes and reads them with [`github.com/parquet-go/parquet-go`](https://github.com/parquet-go/parquet-go) (Apache-2.0, no cgo). The desktop app may query the same files with DuckDB later (M8). DuckDB is not required to read history.
- Layout, under a metrics root:

  ```text
  {root}/{target}/1s/20060102T150405Z.parquet   # one minute of 1 s samples
  {root}/{target}/1m/20060102T150405Z.parquet   # one hour of 1 min rollups
  {root}/{target}/1h/20060102T000000Z.parquet   # one day of 1 h rollups
  ```

  `{target}` is the inventory environment and cluster (`prod/orders` with the slash replaced, so one path segment). The file name is the UTC start of the bucket. The directory listing is the catalogue; it is not copied into SQLite.
- A file is immutable once it is renamed into place. A crash mid-write leaves a temporary file, which the next open deletes. Compaction and retention delete or rename whole files, never rows inside one.
- Downsampling is compaction, not part of the collector. A gauge rolls up to average, minimum and maximum. A counter rolls up to the sum of non-negative deltas, with a reset (a decrease) dropped rather than treated as negative traffic. The same reset rule is what M2.3 uses for live deltas.
- Retention, per target, drops whole files whose bucket start is older than:

  | Resolution | Keep |
  |---|---|
  | 1 s | 1 hour |
  | 1 min | 14 days |
  | 1 h | 13 calendar months |

- Query text, parameters and result rows are not columns in these files. A statement is a numeric identifier here; its text stays on the server or in a workload snapshot the user asked to save (M2.5).
- Remote write and `/metrics` (M2.9) export the latest sample. They are not the store.

## Consequences

- M2.4 implements the directory, the writer, the reader, retention and compaction. Each metric family's Parquet schema lives next to the `sqlpack` query that produces it.
- The collector (M2.2) only decides when a target may be queried. It does not open segment files.
- SQLite stays the metadata store (inventory cache, users, audit) and is not on the metrics write path, so compaction cannot block the UI on a metrics lock.
- Changing the file layout or the retention defaults later is a new ADR. A reader must tolerate an unknown column so a later schema can add one.
