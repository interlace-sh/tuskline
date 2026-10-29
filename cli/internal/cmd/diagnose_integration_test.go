// SPDX-License-Identifier: Apache-2.0

//go:build integration

package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/pgtest"
)

func TestDiagnoseAgainstPostgres(t *testing.T) {
	t.Setenv("TL_TEST_PASSWORD", pgtest.Password)
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			srv := pgtest.Start(t, major)
			dir := t.TempDir()
			path := filepath.Join(dir, "inventory.yaml")
			if _, errOut, err := runApp(t, &app{}, "connect", "--dsn", srv.DSN, "dev/pg", "--password-env", "TL_TEST_PASSWORD", "--inventory", path); err != nil {
				t.Fatalf("connect: %v\n%s", err, errOut)
			}
			out, errOut, err := runApp(t, &app{}, "diagnose", "dev/pg", "--inventory", path)
			if err != nil {
				t.Fatalf("diagnose: %v\n%s", err, errOut)
			}
			if !strings.Contains(out, "pg_stat_statements") || !strings.Contains(out, "not installed") {
				t.Fatalf("missing extension was not reported:\n%s", out)
			}

			pgtest.Exec(t, srv.Profile,
				"CREATE EXTENSION IF NOT EXISTS pg_stat_statements",
				"CREATE TABLE diag (id int)",
				"CREATE INDEX diag_a ON diag (id)",
				"CREATE INDEX diag_b ON diag (id)",
			)
			out, errOut, err = runApp(t, &app{}, "diagnose", "dev/pg", "-o", "json", "--inventory", path)
			if err != nil {
				t.Fatalf("diagnose json: %v\n%s", err, errOut)
			}
			var report struct {
				Ref      string `json:"ref"`
				Findings []struct {
					Check   string `json:"check"`
					Summary string `json:"summary"`
					Fix     string `json:"fix"`
				} `json:"findings"`
			}
			if err := json.Unmarshal([]byte(out), &report); err != nil {
				t.Fatalf("%v:\n%s", err, out)
			}
			if report.Ref != "dev/pg" {
				t.Fatalf("ref %s", report.Ref)
			}
			var dup bool
			for _, f := range report.Findings {
				if f.Check == "pg_stat_statements" {
					t.Fatalf("extension still reported missing: %+v", report.Findings)
				}
				if f.Check == "duplicate_index" && strings.Contains(f.Summary, "diag_b") && f.Fix != "" {
					dup = true
				}
			}
			if !dup {
				t.Fatalf("no duplicate index:\n%s", out)
			}
			md, errOut, err := runApp(t, &app{}, "diagnose", "dev/pg", "-o", "markdown", "--inventory", path)
			if err != nil || !strings.Contains(md, "## warn: duplicate_index") {
				t.Fatalf("markdown: %v\n%s\n%s", err, errOut, md)
			}
		})
	}
}
