// SPDX-License-Identifier: Apache-2.0

package sqlpack

// AdviseSettings reads the configuration rows the advisor compares.
// enumvals is cast to text so the caller can see which values this build allows.
var AdviseSettings = Query{
	Name:       "advise.settings",
	Doc:        "Selected pg_settings rows: value, unit, context and the allowed values. Read-only.",
	Sources:    []string{"pg_settings"},
	MinVersion: PG14,
	SQL: `SELECT name, setting, COALESCE(unit, ''), context, COALESCE(enumvals::text, '')
FROM pg_settings
WHERE name = ANY($1::text[])`,
}

// AdviseCache is the cluster-wide cache hit ratio.
var AdviseCache = Query{
	Name:       "advise.cache",
	Doc:        "Blocks hit and read across every database, for the cache hit ratio. Read-only.",
	Sources:    []string{"pg_stat_database"},
	MinVersion: PG14,
	SQL:        `SELECT COALESCE(sum(blks_hit), 0), COALESCE(sum(blks_read), 0) FROM pg_stat_database`,
}

// AdviseTemp is the volume spilled to temporary files.
var AdviseTemp = Query{
	Name:       "advise.temp",
	Doc:        "Bytes written to temporary files across every database. Read-only.",
	Sources:    []string{"pg_stat_database"},
	MinVersion: PG14,
	SQL:        `SELECT COALESCE(sum(temp_bytes), 0) FROM pg_stat_database`,
}

// AdviseCheckpoints14 is timed and requested checkpoints before they moved view.
var AdviseCheckpoints14 = Query{
	Name:       "advise.checkpoints",
	Doc:        "Timed and requested checkpoints from pg_stat_bgwriter. PostgreSQL 14 to 16. Read-only.",
	Sources:    []string{"pg_stat_bgwriter"},
	MinVersion: PG14,
	MaxVersion: PG17,
	SQL:        `SELECT checkpoints_timed, checkpoints_req FROM pg_stat_bgwriter`,
}

// AdviseCheckpoints17 is the same pair from the checkpointer view.
var AdviseCheckpoints17 = Query{
	Name:       "advise.checkpoints",
	Doc:        "Timed and requested checkpoints from pg_stat_checkpointer. PostgreSQL 17 and later. Read-only.",
	Sources:    []string{"pg_stat_checkpointer"},
	MinVersion: PG17,
	SQL:        `SELECT num_timed, num_requested FROM pg_stat_checkpointer`,
}
