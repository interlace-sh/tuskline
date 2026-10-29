// SPDX-License-Identifier: Apache-2.0

package sqlpack

// Diagnose pack (roadmap M2.7). These are the reads behind `tl diagnose`
// that the statistics pack does not already cover. All are read-only.

// DiagnoseStatementsInstalled is true when pg_stat_statements is installed
// in the current database.
var DiagnoseStatementsInstalled = Query{
	Name:       "diagnose.statements_installed",
	Doc:        "Whether pg_stat_statements is installed in the current database.",
	Sources:    []string{"pg_extension"},
	MinVersion: PG14,
	SQL:        `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements') AS installed`,
}

// DiagnoseWraparound is the frozen-xid age of each database.
var DiagnoseWraparound = Query{
	Name:       "diagnose.wraparound",
	Doc:        "age(datfrozenxid) per database, the wraparound horizon.",
	Sources:    []string{"pg_database"},
	MinVersion: PG14,
	SQL: `SELECT datname, age(datfrozenxid)::bigint AS xid_age
FROM pg_database
WHERE datname IS NOT NULL
ORDER BY datname`,
}

// DiagnoseIndexes lists user indexes with the key, predicate, scan count
// and size needed to spot unused and duplicate indexes.
var DiagnoseIndexes = Query{
	Name:       "diagnose.indexes",
	Doc:        "User indexes with key, predicate, scans, size, and whether they are unique.",
	Sources:    []string{"pg_index", "pg_class", "pg_namespace", "pg_stat_user_indexes"},
	MinVersion: PG14,
	SQL: `SELECT
	n.nspname AS schema,
	t.relname AS table_name,
	i.relname AS index_name,
	ix.indkey::text AS index_key,
	coalesce(pg_get_expr(ix.indpred, ix.indrelid), '') AS predicate,
	coalesce(s.idx_scan, 0) AS idx_scan,
	pg_relation_size(i.oid) AS bytes,
	ix.indisunique AS is_unique
FROM pg_index ix
JOIN pg_class i ON i.oid = ix.indexrelid
JOIN pg_class t ON t.oid = ix.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
LEFT JOIN pg_stat_user_indexes s ON s.indexrelid = ix.indexrelid
WHERE n.nspname <> 'pg_catalog' AND n.nspname <> 'information_schema'
ORDER BY n.nspname, t.relname, i.relname`,
}

// DiagnoseReplication is the replay lag of each streaming standby.
// replay_seconds is NULL when the standby has not reported a lag yet.
var DiagnoseReplication = Query{
	Name:       "diagnose.replication",
	Doc:        "Replay lag of each streaming standby, in seconds.",
	Sources:    []string{"pg_stat_replication"},
	MinVersion: PG14,
	SQL: `SELECT
	coalesce(application_name, '') AS application_name,
	extract(epoch FROM replay_lag)::float8 AS replay_seconds
FROM pg_stat_replication
ORDER BY application_name`,
}
