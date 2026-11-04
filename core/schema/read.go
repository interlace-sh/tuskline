// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/sqlpack"
)

// Read loads the schema of the database db is connected to.
func Read(ctx context.Context, db *pgx.Conn) (Snapshot, error) {
	if db == nil {
		return Snapshot{}, fmt.Errorf("schema: no connection")
	}
	version, err := conn.ServerVersionNum(db)
	if err != nil {
		return Snapshot{}, err
	}
	var objects []Object
	columns, err := readColumns(ctx, db, version)
	if err != nil {
		return Snapshot{}, err
	}
	objects = append(objects, columns...)
	constraints, err := readConstraints(ctx, db, version)
	if err != nil {
		return Snapshot{}, err
	}
	objects = append(objects, constraints...)
	enums, err := readEnums(ctx, db, version)
	if err != nil {
		return Snapshot{}, err
	}
	objects = append(objects, enums...)
	indexes, err := readIndexes(ctx, db, version)
	if err != nil {
		return Snapshot{}, err
	}
	objects = append(objects, indexes...)
	views, err := readViews(ctx, db, version)
	if err != nil {
		return Snapshot{}, err
	}
	objects = append(objects, views...)
	routines, err := readRoutines(ctx, db, version)
	if err != nil {
		return Snapshot{}, err
	}
	objects = append(objects, routines...)
	snap := Snapshot{Objects: objects}
	if snap.Objects == nil {
		snap.Objects = []Object{}
	}
	return snap, nil
}

func readColumns(ctx context.Context, db *pgx.Conn, version int) ([]Object, error) {
	query, err := sqlpack.Pick(version, sqlpack.SchemaColumns)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, query.SQL)
	if err != nil {
		return nil, fmt.Errorf("read columns: %w", err)
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		var schemaName, table, column, dataType, identity string
		var notNull bool
		var def *string
		if err := rows.Scan(&schemaName, &table, &column, &dataType, &notNull, &def, &identity); err != nil {
			return nil, fmt.Errorf("read columns: %w", err)
		}
		out = append(out, Object{
			Kind:   KindColumn,
			Name:   qualified(schemaName, table, column),
			Detail: columnDetail(dataType, notNull, def, identity),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read columns: %w", err)
	}
	return out, nil
}

func columnDetail(dataType string, notNull bool, def *string, identity string) string {
	null := "nullable"
	if notNull {
		null = "not null"
	}
	detail := dataType + " " + null
	switch identity {
	case "a":
		detail += " identity always"
	case "d":
		detail += " identity by default"
	}
	if identity == "" && def != nil && *def != "" {
		detail += " default " + *def
	}
	return detail
}

func readEnums(ctx context.Context, db *pgx.Conn, version int) ([]Object, error) {
	query, err := sqlpack.Pick(version, sqlpack.SchemaEnums)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, query.SQL)
	if err != nil {
		return nil, fmt.Errorf("read enums: %w", err)
	}
	defer rows.Close()
	var out []Object
	var current string
	var labels []string
	flush := func() {
		if current == "" {
			return
		}
		out = append(out, Object{Kind: KindEnum, Name: current, Detail: strings.Join(labels, ", ")})
	}
	for rows.Next() {
		var schemaName, typeName, label string
		if err := rows.Scan(&schemaName, &typeName, &label); err != nil {
			return nil, fmt.Errorf("read enums: %w", err)
		}
		name := qualified(schemaName, typeName)
		if name != current {
			flush()
			current = name
			labels = nil
		}
		labels = append(labels, label)
	}
	flush()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read enums: %w", err)
	}
	return out, nil
}

func readIndexes(ctx context.Context, db *pgx.Conn, version int) ([]Object, error) {
	query, err := sqlpack.Pick(version, sqlpack.SchemaIndexes)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, query.SQL)
	if err != nil {
		return nil, fmt.Errorf("read indexes: %w", err)
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		var schemaName, name, def string
		if err := rows.Scan(&schemaName, &name, &def); err != nil {
			return nil, fmt.Errorf("read indexes: %w", err)
		}
		out = append(out, Object{Kind: KindIndex, Name: qualified(schemaName, name), Detail: def})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read indexes: %w", err)
	}
	return out, nil
}

func readConstraints(ctx context.Context, db *pgx.Conn, version int) ([]Object, error) {
	query, err := sqlpack.Pick(version, sqlpack.SchemaConstraints)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, query.SQL)
	if err != nil {
		return nil, fmt.Errorf("read constraints: %w", err)
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		var schemaName, table, name, def string
		if err := rows.Scan(&schemaName, &table, &name, &def); err != nil {
			return nil, fmt.Errorf("read constraints: %w", err)
		}
		out = append(out, Object{Kind: KindConstraint, Name: qualified(schemaName, table, name), Detail: def})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read constraints: %w", err)
	}
	return out, nil
}

func readViews(ctx context.Context, db *pgx.Conn, version int) ([]Object, error) {
	query, err := sqlpack.Pick(version, sqlpack.SchemaViews)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, query.SQL)
	if err != nil {
		return nil, fmt.Errorf("read views: %w", err)
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		var schemaName, name, kind, def string
		if err := rows.Scan(&schemaName, &name, &kind, &def); err != nil {
			return nil, fmt.Errorf("read views: %w", err)
		}
		objectKind := KindView
		if kind == "m" {
			objectKind = KindMatview
		}
		out = append(out, Object{Kind: objectKind, Name: qualified(schemaName, name), Detail: strings.TrimSpace(def)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read views: %w", err)
	}
	return out, nil
}

func readRoutines(ctx context.Context, db *pgx.Conn, version int) ([]Object, error) {
	query, err := sqlpack.Pick(version, sqlpack.SchemaRoutines)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, query.SQL)
	if err != nil {
		return nil, fmt.Errorf("read routines: %w", err)
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		var schemaName, signature, def string
		if err := rows.Scan(&schemaName, &signature, &def); err != nil {
			return nil, fmt.Errorf("read routines: %w", err)
		}
		out = append(out, Object{Kind: KindRoutine, Name: qualified(schemaName, signature), Detail: strings.TrimSpace(def)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read routines: %w", err)
	}
	return out, nil
}

func qualified(parts ...string) string {
	return strings.Join(parts, ".")
}
