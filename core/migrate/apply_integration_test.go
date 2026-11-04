// SPDX-License-Identifier: Apache-2.0

//go:build integration

package migrate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/pgtest"
)

func TestRunAcrossVersions(t *testing.T) {
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			profile := pgtest.Start(t, major).Profile
			t.Run("applies once and refuses an edit", func(t *testing.T) {
				db := connect(t, profile)
				reset(t, db)
				dir := t.TempDir()
				writeScript(t, dir, "1_table.sql", "CREATE TABLE orders (id int);\n")
				writeScript(t, dir, "2_fn.sql", "CREATE FUNCTION order_tag() RETURNS int AS $$ SELECT 1; $$ LANGUAGE sql;\nSELECT order_tag();\n")
				folder := load(t, dir)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				first, err := Run(ctx, db, folder)
				if err != nil {
					t.Fatal(err)
				}
				if first.String() != "applied 1_table.sql\napplied 2_fn.sql\n" {
					t.Fatalf("%s", first.String())
				}
				var n int
				if err := db.QueryRow(ctx, `SELECT count(*) FROM tuskline.schema_migration WHERE success`).Scan(&n); err != nil {
					t.Fatal(err)
				}
				if n != 2 {
					t.Fatalf("history rows %d", n)
				}
				second, err := Run(ctx, db, folder)
				if err != nil {
					t.Fatal(err)
				}
				if len(second.Applied) != 0 || len(second.Skipped) != 2 {
					t.Fatalf("%+v", second)
				}
				writeScript(t, dir, "2_fn.sql", "CREATE FUNCTION order_tag() RETURNS int AS $$ SELECT 2; $$ LANGUAGE sql;\n")
				edited := load(t, dir)
				_, err = Run(ctx, db, edited)
				if !errors.Is(err, ErrChecksumChanged) {
					t.Fatalf("%v", err)
				}
			})
			t.Run("rolls back a failed script", func(t *testing.T) {
				db := connect(t, profile)
				reset(t, db)
				dir := t.TempDir()
				writeScript(t, dir, "1.sql", "CREATE TABLE rolled (id int); SELECT 1/0;\n")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if _, err := Run(ctx, db, load(t, dir)); err == nil {
					t.Fatal("expected the script to fail")
				}
				var exists bool
				if err := db.QueryRow(ctx, `SELECT to_regclass('public.rolled') IS NOT NULL`).Scan(&exists); err != nil {
					t.Fatal(err)
				}
				if exists {
					t.Fatal("failed script left a table")
				}
				writeScript(t, dir, "1.sql", "CREATE TABLE rolled (id int);\n")
				if _, err := Run(ctx, db, load(t, dir)); err != nil {
					t.Fatal(err)
				}
			})
			t.Run("builds an index outside a transaction", func(t *testing.T) {
				db := connect(t, profile)
				reset(t, db)
				dir := t.TempDir()
				writeScript(t, dir, "1.sql", "CREATE TABLE indexed (id int);\n")
				writeScript(t, dir, "2.sql", "CREATE INDEX CONCURRENTLY indexed_id ON indexed (id);\n")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if _, err := Run(ctx, db, load(t, dir)); err != nil {
					t.Fatal(err)
				}
				var indisvalid bool
				if err := db.QueryRow(ctx, `SELECT indisvalid FROM pg_index WHERE indexrelid = 'indexed_id'::regclass`).Scan(&indisvalid); err != nil {
					t.Fatal(err)
				}
				if !indisvalid {
					t.Fatal("index was not valid")
				}
			})
			t.Run("waits while another session holds the lock", func(t *testing.T) {
				holder := connect(t, profile)
				db := connect(t, profile)
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				var got bool
				if err := holder.QueryRow(ctx, `SELECT pg_try_advisory_lock($1, $2)`, advisoryKey1, advisoryKey2).Scan(&got); err != nil {
					t.Fatal(err)
				}
				if !got {
					t.Fatal("lock was not acquired")
				}
				dir := t.TempDir()
				writeScript(t, dir, "1.sql", "SELECT 1;\n")
				_, err := Run(ctx, db, load(t, dir))
				if !errors.Is(err, ErrInProgress) {
					t.Fatalf("%v", err)
				}
			})
		})
	}
}

func connect(t *testing.T, profile conn.Profile) *pgx.Conn {
	t.Helper()
	db, err := conn.Connect(context.Background(), profile, conn.SessionSettings{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(context.Background()) })
	return db
}

func load(t *testing.T, dir string) Folder {
	t.Helper()
	folder, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return folder
}

func reset(t *testing.T, db *pgx.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := db.Exec(ctx, `DROP SCHEMA IF EXISTS tuskline CASCADE`); err != nil {
		t.Fatal(err)
	}
}

func writeScript(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
