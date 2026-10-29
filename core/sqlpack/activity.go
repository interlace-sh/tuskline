// SPDX-License-Identifier: Apache-2.0

package sqlpack

// Activity pack (roadmap M1.7). core/activity runs these for `tl top` and
// the activity screen. Both are read-only, take no heavyweight locks
// (reading pg_locks only briefly holds lock-manager partition locks) and
// exclude Tuskline's own backend. Roles without pg_read_all_stats (part of
// pg_monitor) see other users' sessions with query text and wait details
// hidden; nothing errors. Nothing in these views that the queries use
// changed between PostgreSQL 14 and 18, so each has one variant.

// ActivitySessions lists backends, busiest first, with $1 the maximum
// query text length and $2 the row limit. Columns: pid, leader_pid,
// usename, datname, application_name, client_addr, backend_type, state,
// wait_event_type, wait_event, xact_seconds, query_seconds,
// state_seconds, query_id, query, blocked_by.
//
// blocked_by is pg_blocking_pids(), computed only for sessions waiting on
// a heavyweight lock because the function is expensive under contention.
// query_id is NULL unless compute_query_id is on.
var ActivitySessions = Query{
	Name:       "activity.sessions",
	Doc:        "Backends with state, waits, transaction and query age, query text (truncated) and the pids blocking them.",
	Sources:    []string{"pg_stat_activity", "pg_blocking_pids()"},
	MinVersion: PG14,
	SQL: `SELECT
	a.pid,
	a.leader_pid,
	coalesce(a.usename::text, '') AS usename,
	coalesce(a.datname::text, '') AS datname,
	coalesce(a.application_name, '') AS application_name,
	coalesce(host(a.client_addr), '') AS client_addr,
	coalesce(a.backend_type, '') AS backend_type,
	coalesce(a.state, '') AS state,
	coalesce(a.wait_event_type, '') AS wait_event_type,
	coalesce(a.wait_event, '') AS wait_event,
	extract(epoch FROM now() - a.xact_start)::float8 AS xact_seconds,
	extract(epoch FROM now() - a.query_start)::float8 AS query_seconds,
	extract(epoch FROM now() - a.state_change)::float8 AS state_seconds,
	a.query_id,
	left(coalesce(a.query, ''), $1) AS query,
	CASE WHEN a.wait_event_type = 'Lock' THEN pg_blocking_pids(a.pid) ELSE '{}'::int[] END AS blocked_by
FROM pg_stat_activity a
WHERE a.pid <> pg_backend_pid()
ORDER BY coalesce(a.state, '') = 'idle', a.query_start NULLS LAST, a.pid
LIMIT $2`,
}

// ActivityLocks lists lock requests that are waiting, and the granted
// locks on the same targets (the holders they wait for), with $1 the row
// limit. Columns: pid (0 for a prepared transaction), locktype, mode,
// granted, relation, detail, wait_seconds.
//
// relation is resolved with regclass only for the current database (or
// shared catalogues); other databases' relations show their OID, because
// an OID from another database could name an unrelated local table.
var ActivityLocks = Query{
	Name:       "activity.locks",
	Doc:        "Waiting lock requests and the granted locks on the same objects, with mode, target and wait time.",
	Sources:    []string{"pg_locks", "pg_database"},
	MinVersion: PG14,
	SQL: `WITH waiting AS (
	SELECT * FROM pg_locks WHERE NOT granted
)
SELECT
	coalesce(l.pid, 0) AS pid,
	l.locktype,
	l.mode,
	l.granted,
	CASE
		WHEN l.relation IS NULL THEN ''
		WHEN l.database = 0 OR l.database = (SELECT oid FROM pg_database WHERE datname = current_database()) THEN l.relation::regclass::text
		ELSE 'oid ' || l.relation
	END AS relation,
	concat_ws(' ',
		'page ' || l.page,
		'tuple ' || l.tuple,
		'xid ' || l.transactionid,
		'vxid ' || l.virtualxid,
		CASE WHEN l.locktype = 'advisory' THEN 'key ' || l.classid || ',' || l.objid END
	) AS detail,
	extract(epoch FROM now() - l.waitstart)::float8 AS wait_seconds
FROM pg_locks l
WHERE l.pid IS DISTINCT FROM pg_backend_pid()
	AND (NOT l.granted OR EXISTS (
		SELECT 1 FROM waiting w
		WHERE w.locktype = l.locktype
			AND w.database IS NOT DISTINCT FROM l.database
			AND w.relation IS NOT DISTINCT FROM l.relation
			AND w.page IS NOT DISTINCT FROM l.page
			AND w.tuple IS NOT DISTINCT FROM l.tuple
			AND w.virtualxid IS NOT DISTINCT FROM l.virtualxid
			AND w.transactionid IS NOT DISTINCT FROM l.transactionid
			AND w.classid IS NOT DISTINCT FROM l.classid
			AND w.objid IS NOT DISTINCT FROM l.objid
			AND w.objsubid IS NOT DISTINCT FROM l.objsubid
	))
ORDER BY l.granted, l.waitstart NULLS LAST, l.pid
LIMIT $1`,
}
