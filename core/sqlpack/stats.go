// SPDX-License-Identifier: Apache-2.0

//nolint:misspell // PostgreSQL column names keep the spelling "analyze".
package sqlpack

// Statistics pack (roadmap M2.3). Column lists were taken from PostgreSQL
// 14, 15, 16, 17 and 18. Where a view changed shape, Pick selects the
// variant. Every query is read-only and takes no locks beyond the
// statistics views. Counter resets are not handled in SQL: collector.Delta
// compares two samples and treats a decrease as a reset.

// Version numbers for the statistics-view changes.
const (
	PG15 = 150000
	PG16 = 160000
	PG17 = 170000
	PG18 = 180000
)

// StatsDatabase is one row per database, for servers 14 through 18. The
// columns are the ones present on every supported major. PostgreSQL 18 also
// has parallel-worker counters; they are not in this query.
var StatsDatabase = Query{
	Name:       "stats.database",
	Doc:        "Per-database transaction, block, tuple and session counters from pg_stat_database.",
	Sources:    []string{"pg_stat_database"},
	MinVersion: PG14,
	SQL: `SELECT
	datname,
	numbackends,
	xact_commit,
	xact_rollback,
	blks_read,
	blks_hit,
	tup_returned,
	tup_fetched,
	tup_inserted,
	tup_updated,
	tup_deleted,
	conflicts,
	temp_files,
	temp_bytes,
	deadlocks,
	blk_read_time,
	blk_write_time,
	session_time,
	active_time,
	idle_in_transaction_time,
	sessions,
	sessions_abandoned,
	sessions_fatal,
	sessions_killed,
	stats_reset
FROM pg_stat_database
WHERE datname IS NOT NULL
ORDER BY datname`,
}

// StatsTables14 lists user tables on PostgreSQL 14 and 15. $1 is the row limit.
var StatsTables14 = Query{
	Name:       "stats.tables",
	Doc:        "User-table scan, tuple and vacuum counters. PostgreSQL 14 and 15.",
	Sources:    []string{"pg_stat_user_tables"},
	MinVersion: PG14,
	MaxVersion: PG16,
	SQL: `SELECT
	schemaname,
	relname,
	seq_scan,
	seq_tup_read,
	idx_scan,
	idx_tup_fetch,
	n_tup_ins,
	n_tup_upd,
	n_tup_del,
	n_tup_hot_upd,
	n_live_tup,
	n_dead_tup,
	n_mod_since_analyze,
	n_ins_since_vacuum,
	vacuum_count,
	autovacuum_count,
	analyze_count,
	autoanalyze_count
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC NULLS LAST, schemaname, relname
LIMIT $1`,
}

// StatsTables16 adds the PostgreSQL 16 scan timestamps and new-page updates.
// It also runs on 17. $1 is the row limit.
var StatsTables16 = Query{
	Name:       "stats.tables",
	Doc:        "User-table counters including last scan times and new-page updates. PostgreSQL 16 and 17.",
	Sources:    []string{"pg_stat_user_tables"},
	MinVersion: PG16,
	MaxVersion: PG18,
	SQL: `SELECT
	schemaname,
	relname,
	seq_scan,
	last_seq_scan,
	seq_tup_read,
	idx_scan,
	last_idx_scan,
	idx_tup_fetch,
	n_tup_ins,
	n_tup_upd,
	n_tup_del,
	n_tup_hot_upd,
	n_tup_newpage_upd,
	n_live_tup,
	n_dead_tup,
	n_mod_since_analyze,
	n_ins_since_vacuum,
	vacuum_count,
	autovacuum_count,
	analyze_count,
	autoanalyze_count
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC NULLS LAST, schemaname, relname
LIMIT $1`,
}

// StatsTables18 adds the PostgreSQL 18 cumulative vacuum and analyse times.
// $1 is the row limit.
var StatsTables18 = Query{
	Name:       "stats.tables",
	Doc:        "User-table counters including cumulative vacuum and analyse time. PostgreSQL 18.",
	Sources:    []string{"pg_stat_user_tables"},
	MinVersion: PG18,
	SQL: `SELECT
	schemaname,
	relname,
	seq_scan,
	last_seq_scan,
	seq_tup_read,
	idx_scan,
	last_idx_scan,
	idx_tup_fetch,
	n_tup_ins,
	n_tup_upd,
	n_tup_del,
	n_tup_hot_upd,
	n_tup_newpage_upd,
	n_live_tup,
	n_dead_tup,
	n_mod_since_analyze,
	n_ins_since_vacuum,
	vacuum_count,
	autovacuum_count,
	analyze_count,
	autoanalyze_count,
	total_vacuum_time,
	total_autovacuum_time,
	total_analyze_time,
	total_autoanalyze_time
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC NULLS LAST, schemaname, relname
LIMIT $1`,
}

// StatsWAL14 is the single pg_stat_wal row on PostgreSQL 14 to 17, including
// WAL write and sync counters. Those moved off this view in 18.
var StatsWAL14 = Query{
	Name:       "stats.wal",
	Doc:        "WAL generation, write and sync counters. PostgreSQL 14 to 17.",
	Sources:    []string{"pg_stat_wal"},
	MinVersion: PG14,
	MaxVersion: PG18,
	SQL: `SELECT
	wal_records,
	wal_fpi,
	wal_bytes,
	wal_buffers_full,
	wal_write,
	wal_sync,
	wal_write_time,
	wal_sync_time,
	stats_reset
FROM pg_stat_wal`,
}

// StatsWAL18 is the single pg_stat_wal row on PostgreSQL 18.
var StatsWAL18 = Query{
	Name:       "stats.wal",
	Doc:        "WAL generation counters. PostgreSQL 18; write and sync counters no longer live on this view.",
	Sources:    []string{"pg_stat_wal"},
	MinVersion: PG18,
	SQL: `SELECT
	wal_records,
	wal_fpi,
	wal_bytes,
	wal_buffers_full,
	stats_reset
FROM pg_stat_wal`,
}

// StatsBgwriter14 is the bgwriter row on PostgreSQL 14 to 16, when it still
// included checkpoint counters.
var StatsBgwriter14 = Query{
	Name:       "stats.bgwriter",
	Doc:        "Background writer and checkpoint counters. PostgreSQL 14 to 16.",
	Sources:    []string{"pg_stat_bgwriter"},
	MinVersion: PG14,
	MaxVersion: PG17,
	SQL: `SELECT
	checkpoints_timed,
	checkpoints_req,
	checkpoint_write_time,
	checkpoint_sync_time,
	buffers_checkpoint,
	buffers_clean,
	maxwritten_clean,
	buffers_backend,
	buffers_backend_fsync,
	buffers_alloc,
	stats_reset
FROM pg_stat_bgwriter`,
}

// StatsBgwriter17 is the bgwriter row once checkpoints have their own view.
var StatsBgwriter17 = Query{
	Name:       "stats.bgwriter",
	Doc:        "Background writer counters. PostgreSQL 17 and later; checkpoints are stats.checkpointer.",
	Sources:    []string{"pg_stat_bgwriter"},
	MinVersion: PG17,
	SQL: `SELECT
	buffers_clean,
	maxwritten_clean,
	buffers_alloc,
	stats_reset
FROM pg_stat_bgwriter`,
}

// StatsCheckpointer17 is the checkpointer row added in PostgreSQL 17.
var StatsCheckpointer17 = Query{
	Name:       "stats.checkpointer",
	Doc:        "Checkpointer counters. PostgreSQL 17.",
	Sources:    []string{"pg_stat_checkpointer"},
	MinVersion: PG17,
	MaxVersion: PG18,
	SQL: `SELECT
	num_timed,
	num_requested,
	restartpoints_timed,
	restartpoints_req,
	restartpoints_done,
	write_time,
	sync_time,
	buffers_written,
	stats_reset
FROM pg_stat_checkpointer`,
}

// StatsCheckpointer18 adds completion and SLRU counts.
var StatsCheckpointer18 = Query{
	Name:       "stats.checkpointer",
	Doc:        "Checkpointer counters, including completed checkpoints and SLRU writes. PostgreSQL 18.",
	Sources:    []string{"pg_stat_checkpointer"},
	MinVersion: PG18,
	SQL: `SELECT
	num_timed,
	num_requested,
	num_done,
	restartpoints_timed,
	restartpoints_req,
	restartpoints_done,
	write_time,
	sync_time,
	buffers_written,
	slru_written,
	stats_reset
FROM pg_stat_checkpointer`,
}

// StatsIO16 is pg_stat_io on PostgreSQL 16 and 17. There is no such view on 14 or 15.
var StatsIO16 = Query{
	Name:       "stats.io",
	Doc:        "Block I/O by backend type, object and context. PostgreSQL 16 and 17.",
	Sources:    []string{"pg_stat_io"},
	MinVersion: PG16,
	MaxVersion: PG18,
	SQL: `SELECT
	backend_type,
	object,
	context,
	reads,
	read_time,
	writes,
	write_time,
	writebacks,
	writeback_time,
	extends,
	extend_time,
	op_bytes,
	hits,
	evictions,
	reuses,
	fsyncs,
	fsync_time,
	stats_reset
FROM pg_stat_io
ORDER BY backend_type, object, context`,
}

// StatsIO18 counts bytes as well as operations, and drops op_bytes.
var StatsIO18 = Query{
	Name:       "stats.io",
	Doc:        "Block I/O by backend type, object and context, in operations and bytes. PostgreSQL 18.",
	Sources:    []string{"pg_stat_io"},
	MinVersion: PG18,
	SQL: `SELECT
	backend_type,
	object,
	context,
	reads,
	read_bytes,
	read_time,
	writes,
	write_bytes,
	write_time,
	writebacks,
	writeback_time,
	extends,
	extend_bytes,
	extend_time,
	hits,
	evictions,
	reuses,
	fsyncs,
	fsync_time,
	stats_reset
FROM pg_stat_io
ORDER BY backend_type, object, context`,
}

// statementSelect is the part of pg_stat_statements that exists on every
// supported major. Variants append the block-time columns their version has.
// $1 is the maximum query-text length. $2 is the row limit. Only top-level
// statements in the current database are returned, busiest first.
const statementSelect = `SELECT
	userid,
	dbid,
	queryid,
	left(query, $1) AS query,
	calls,
	total_exec_time,
	rows,
	shared_blks_hit,
	shared_blks_read,
	wal_bytes,
	`

const statementTail = `
FROM pg_stat_statements
WHERE dbid = (SELECT oid FROM pg_database WHERE datname = current_database())
	AND toplevel
	AND queryid IS NOT NULL
ORDER BY total_exec_time DESC
LIMIT $2`

// StatsStatements14 reads pg_stat_statements on PostgreSQL 14. Block time is
// one pair of columns, aliased to the names used from PostgreSQL 17.
var StatsStatements14 = Query{
	Name:       "stats.statements",
	Doc:        "Top statements by total execution time in the current database. PostgreSQL 14. Needs pg_stat_statements.",
	Sources:    []string{"pg_stat_statements", "pg_database"},
	MinVersion: PG14,
	MaxVersion: PG15,
	Extension:  "pg_stat_statements",
	SQL: statementSelect + `blk_read_time AS shared_blk_read_time,
	blk_write_time AS shared_blk_write_time` + statementTail,
}

// StatsStatements15 is the PostgreSQL 15 and 16 statement list, which adds
// temporary-file block time.
var StatsStatements15 = Query{
	Name:       "stats.statements",
	Doc:        "Top statements by total execution time, including temporary-file block time. PostgreSQL 15 and 16.",
	Sources:    []string{"pg_stat_statements", "pg_database"},
	MinVersion: PG15,
	MaxVersion: PG17,
	Extension:  "pg_stat_statements",
	SQL: statementSelect + `blk_read_time AS shared_blk_read_time,
	blk_write_time AS shared_blk_write_time,
	temp_blk_read_time,
	temp_blk_write_time` + statementTail,
}

// StatsStatements17 is the statement list once block time is split into
// shared, local and temporary. It also runs on PostgreSQL 18.
var StatsStatements17 = Query{
	Name:       "stats.statements",
	Doc:        "Top statements by total execution time, with shared, local and temporary block time. PostgreSQL 17 and 18.",
	Sources:    []string{"pg_stat_statements", "pg_database"},
	MinVersion: PG17,
	Extension:  "pg_stat_statements",
	SQL: statementSelect + `shared_blk_read_time,
	shared_blk_write_time,
	temp_blk_read_time,
	temp_blk_write_time` + statementTail,
}
