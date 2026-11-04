// SPDX-License-Identifier: Apache-2.0

package sqlpack

// Schema pack (roadmap M7.3). These read the catalogues that define a
// database and nothing else: no row data, no query text, and no locks
// beyond the catalogue reads. The migration bookkeeping schema tuskline
// is left out, as are the system schemas.

const schemaWhere = `
	n.nspname <> 'pg_catalog'
	AND n.nspname <> 'information_schema'
	AND n.nspname <> 'tuskline'
	AND n.nspname NOT LIKE 'pg_toast%'
	AND n.nspname NOT LIKE 'pg_temp%'`

// SchemaColumns returns one row per column of an ordinary or partitioned
// table: schema, table, column, type, not-null, default expression
// (NULL when there is none) and identity ("" , "a" or "d").
var SchemaColumns = Query{
	Name:       "schema.columns",
	Doc:        "Columns of ordinary and partitioned tables, excluding system and migration schemas.",
	Sources:    []string{"pg_attribute", "pg_class", "pg_namespace", "pg_attrdef", "format_type()", "pg_get_expr()"},
	MinVersion: PG14,
	SQL: `SELECT n.nspname, c.relname, a.attname,
		pg_catalog.format_type(a.atttypid, a.atttypmod),
		a.attnotnull,
		pg_catalog.pg_get_expr(d.adbin, d.adrelid),
		a.attidentity::text
	FROM pg_catalog.pg_attribute a
	JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
	JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
	WHERE c.relkind IN ('r', 'p')
		AND a.attnum > 0
		AND NOT a.attisdropped
		AND ` + schemaWhere + `
	ORDER BY n.nspname, c.relname, a.attnum`,
}

// SchemaEnums returns enum labels in definition order: schema, type, label.
var SchemaEnums = Query{
	Name:       "schema.enums",
	Doc:        "Enum labels in sort order, excluding system and migration schemas.",
	Sources:    []string{"pg_enum", "pg_type", "pg_namespace"},
	MinVersion: PG14,
	SQL: `SELECT n.nspname, t.typname, e.enumlabel
	FROM pg_catalog.pg_enum e
	JOIN pg_catalog.pg_type t ON t.oid = e.enumtypid
	JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
	WHERE ` + schemaWhere + `
	ORDER BY n.nspname, t.typname, e.enumsortorder`,
}

// SchemaIndexes returns schema, index name and the CREATE INDEX statement.
var SchemaIndexes = Query{
	Name:       "schema.indexes",
	Doc:        "Index definitions, excluding system and migration schemas.",
	Sources:    []string{"pg_indexes"},
	MinVersion: PG14,
	SQL: `SELECT schemaname, indexname, indexdef
	FROM pg_catalog.pg_indexes
	WHERE schemaname <> 'pg_catalog'
		AND schemaname <> 'information_schema'
		AND schemaname <> 'tuskline'
		AND schemaname NOT LIKE 'pg_toast%'
		AND schemaname NOT LIKE 'pg_temp%'
	ORDER BY schemaname, indexname`,
}

// SchemaConstraints returns schema, table, constraint name and definition.
var SchemaConstraints = Query{
	Name:       "schema.constraints",
	Doc:        "Primary, unique, foreign and check constraints, excluding system and migration schemas.",
	Sources:    []string{"pg_constraint", "pg_class", "pg_namespace", "pg_get_constraintdef()"},
	MinVersion: PG14,
	SQL: `SELECT n.nspname, r.relname, c.conname, pg_catalog.pg_get_constraintdef(c.oid, true)
	FROM pg_catalog.pg_constraint c
	JOIN pg_catalog.pg_class r ON r.oid = c.conrelid
	JOIN pg_catalog.pg_namespace n ON n.oid = r.relnamespace
	WHERE c.contype IN ('c', 'f', 'p', 'u')
		AND ` + schemaWhere + `
	ORDER BY n.nspname, r.relname, c.conname`,
}

// SchemaViews returns schema, view name, kind ("v" or "m") and the view body.
var SchemaViews = Query{
	Name:       "schema.views",
	Doc:        "View and materialised view bodies, excluding system and migration schemas.",
	Sources:    []string{"pg_class", "pg_namespace", "pg_get_viewdef()"},
	MinVersion: PG14,
	SQL: `SELECT n.nspname, c.relname, c.relkind::text, pg_catalog.pg_get_viewdef(c.oid, true)
	FROM pg_catalog.pg_class c
	JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relkind IN ('v', 'm')
		AND ` + schemaWhere + `
	ORDER BY n.nspname, c.relname`,
}

// SchemaRoutines returns schema, signature and CREATE text for functions
// and procedures.
var SchemaRoutines = Query{
	Name:       "schema.routines",
	Doc:        "Function and procedure definitions, excluding system and migration schemas.",
	Sources:    []string{"pg_proc", "pg_namespace", "pg_get_function_identity_arguments()", "pg_get_functiondef()"},
	MinVersion: PG14,
	SQL: `SELECT n.nspname,
		p.proname || '(' || pg_catalog.pg_get_function_identity_arguments(p.oid) || ')',
		pg_catalog.pg_get_functiondef(p.oid)
	FROM pg_catalog.pg_proc p
	JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
	WHERE p.prokind IN ('f', 'p')
		AND ` + schemaWhere + `
	ORDER BY n.nspname, 2`,
}
