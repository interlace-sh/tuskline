// SPDX-License-Identifier: Apache-2.0

//go:build integration

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/pgtest"
	"github.com/interlace-sh/tuskline/core/snapshot"
)

func TestSnapshotAndDiff(t *testing.T) {
	t.Setenv("TL_TEST_PASSWORD", pgtest.Password)
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			srv := pgtest.Start(t, major)
			pgtest.Exec(t, srv.Profile,
				"CREATE EXTENSION IF NOT EXISTS pg_stat_statements",
				"SELECT 1 /* tl-snapshot-a */",
			)
			dir := t.TempDir()
			path := filepath.Join(dir, "inventory.yaml")
			before := filepath.Join(dir, "before.json")
			after := filepath.Join(dir, "after.json")
			if _, errOut, err := runApp(t, &app{}, "connect", "--dsn", srv.DSN, "dev/pg", "--password-env", "TL_TEST_PASSWORD", "--inventory", path); err != nil {
				t.Fatalf("connect: %v\n%s", err, errOut)
			}
			if _, errOut, err := runApp(t, &app{}, "snapshot", "dev/pg", "--file", before, "--inventory", path); err != nil {
				t.Fatalf("snapshot: %v\n%s", err, errOut)
			}
			// A different statement shape: pg_stat_statements folds both
			// "SELECT 1" and "SELECT 2" into "SELECT $1".
			pgtest.Exec(t, srv.Profile, "SELECT count(*) FROM pg_class /* tl-snapshot-b */")
			if _, errOut, err := runApp(t, &app{}, "snapshot", "dev/pg", "--file", after, "--inventory", path); err != nil {
				t.Fatalf("snapshot after: %v\n%s", err, errOut)
			}
			out, errOut, err := runApp(t, &app{}, "diff", before, after)
			if err != nil {
				t.Fatalf("diff: %v\n%s", err, errOut)
			}
			if !strings.Contains(out, "tl-snapshot") {
				t.Fatalf("diff has no statement text:\n%s", out)
			}
			newer, err := snapshot.ReadFile(after)
			if err != nil {
				t.Fatal(err)
			}
			var sawB bool
			for _, s := range newer.Statements {
				if strings.Contains(s.Query, "tl-snapshot-b") {
					sawB = true
				}
			}
			if !sawB {
				t.Fatalf("second snapshot missed the new statement: %+v", newer.Statements)
			}
		})
	}
}
