// SPDX-License-Identifier: Apache-2.0

package sqlpack

// Probe pack (roadmap M1.2). conn.Probe runs these on connect to learn what
// a server is and what Tuskline can do on it. All are read-only, take no
// locks beyond catalogue reads, and need no extension. Every column must be
// readable by an ordinary login role: anything privileged is wrapped so it
// yields NULL instead of an error.

// ProbeServer returns version, session and provider signals in one row:
// server_version_num, version(), default_transaction_read_only,
// pg_is_in_recovery(), current_database(), current_user, then the
// shared_preload_libraries value (NULL when the role may not read it:
// pg_settings hides superuser-only settings from roles without
// pg_read_all_settings), then booleans used by conn.DetectProvider.
var ProbeServer = Query{
	Name: "probe.server",
	Doc:  "Server version, session read-only state, recovery state, preload libraries and managed-service fingerprints.",
	Sources: []string{
		"current_setting()", "version()", "pg_is_in_recovery()", "pg_settings",
		"pg_proc", "pg_roles",
	},
	MinVersion: PG14,
	SQL: `SELECT
	current_setting('server_version_num')::int AS version_num,
	version() AS version,
	current_setting('default_transaction_read_only') = 'on' AS read_only,
	pg_is_in_recovery() AS in_recovery,
	current_database() AS database,
	current_user AS login_role,
	(SELECT setting FROM pg_settings WHERE name = 'shared_preload_libraries') AS preload,
	EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'aurora_version') AS has_aurora_version,
	EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'rds_superuser') AS has_rds_superuser,
	current_setting('rds.extensions', true) IS NOT NULL AS has_rds_settings,
	EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cloudsqlsuperuser') AS has_cloudsql_superuser,
	current_setting('cloudsql.iam_authentication', true) IS NOT NULL AS has_cloudsql_settings,
	EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'alloydbsuperuser') AS has_alloydb_superuser,
	current_setting('alloydb.iam_authentication', true) IS NOT NULL AS has_alloydb_settings,
	EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'azure_pg_admin') AS has_azure_admin,
	current_setting('azure.extensions', true) IS NOT NULL AS has_azure_settings,
	current_setting('neon.timeline_id', true) IS NOT NULL AS has_neon_settings`,
}

// ProbeExtensions lists extensions installed in the current database and
// those available to install, one row per extension name:
// name, installed version (NULL if not installed), default version
// (NULL if the server has no control file for it).
var ProbeExtensions = Query{
	Name:       "probe.extensions",
	Doc:        "Installed extensions in the current database, plus every extension available on the server.",
	Sources:    []string{"pg_extension", "pg_available_extensions"},
	MinVersion: PG14,
	SQL: `SELECT
	coalesce(e.extname, a.name::text) AS name,
	e.extversion AS installed_version,
	a.default_version AS default_version
FROM pg_extension e
FULL JOIN pg_available_extensions a ON a.name = e.extname
ORDER BY 1`,
}

// ProbeRole returns the login role's attributes and the predefined roles
// Tuskline cares about, checked with pg_has_role(..., 'USAGE') so inherited
// membership counts and the role's privileges are those actually usable.
var ProbeRole = Query{
	Name:       "probe.role",
	Doc:        "Attributes and predefined-role memberships of the login role.",
	Sources:    []string{"pg_roles", "pg_has_role()"},
	MinVersion: PG14,
	SQL: `SELECT
	r.rolsuper, r.rolcreaterole, r.rolcreatedb, r.rolreplication, r.rolbypassrls,
	pg_has_role('pg_monitor', 'USAGE') AS pg_monitor,
	pg_has_role('pg_read_all_settings', 'USAGE') AS pg_read_all_settings,
	pg_has_role('pg_read_all_stats', 'USAGE') AS pg_read_all_stats,
	pg_has_role('pg_signal_backend', 'USAGE') AS pg_signal_backend
FROM pg_roles r
WHERE r.rolname = current_user`,
}
