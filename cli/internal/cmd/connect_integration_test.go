// SPDX-License-Identifier: Apache-2.0

//go:build integration

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/pgtest"
)

func TestConnectAgainstPostgres(t *testing.T) {
	// Not parallel: t.Setenv, and one container at a time is enough.
	t.Setenv("TL_TEST_PASSWORD", pgtest.Password)
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			srv := pgtest.Start(t, major)
			path := filepath.Join(t.TempDir(), "inventory.yaml")

			out, errOut, err := runApp(t, &app{}, "connect", "--dsn", srv.DSN, "prod/orders", "--password-env", "TL_TEST_PASSWORD", "--inventory", path)
			if err != nil {
				t.Fatalf("connect --dsn: %v\n%s", err, errOut)
			}
			for _, want := range []string{"connected: prod/orders (prod)", fmt.Sprintf("PostgreSQL %d", major), "read-only session", "superuser", "pg_wait_sampling", "saved prod/orders"} {
				if !strings.Contains(out, want) {
					t.Errorf("summary lacks %q:\n%s", want, out)
				}
			}
			if !strings.Contains(errOut, "sslmode=disable") {
				t.Errorf("no TLS warning on stderr: %q", errOut)
			}
			file, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range []string{out, errOut, string(file)} {
				if strings.Contains(s, pgtest.Password) {
					t.Fatalf("password leaked:\n%s", s)
				}
			}
			if !strings.Contains(string(file), "env: TL_TEST_PASSWORD") {
				t.Errorf("inventory lacks the password reference:\n%s", file)
			}

			// Probe the saved entry again; the password comes from the reference.
			out, errOut, err = runApp(t, &app{}, "connect", "prod/orders", "-o", "json", "--inventory", path)
			if err != nil {
				t.Fatalf("connect prod/orders: %v\n%s", err, errOut)
			}
			var res struct {
				Ref         string `json:"ref"`
				Saved       bool   `json:"saved"`
				PasswordRef string `json:"password_ref"`
				Server      struct {
					VersionNum int  `json:"version_num"`
					ReadOnly   bool `json:"read_only"`
				} `json:"server"`
				Missing []struct {
					Name string `json:"name"`
				} `json:"missing_extensions"`
			}
			if err := json.Unmarshal([]byte(out), &res); err != nil {
				t.Fatalf("%v:\n%s", err, out)
			}
			if res.Ref != "prod/orders" || !res.Saved || res.PasswordRef != "env:TL_TEST_PASSWORD" || res.Server.VersionNum/10000 != major || !res.Server.ReadOnly || len(res.Missing) == 0 {
				t.Errorf("unexpected JSON: %+v", res)
			}

			// A dev entry for the same server gets a read-write session, and
			// the DSN password is not saved without --password-env.
			out, errOut, err = runApp(t, &app{}, "connect", "--dsn", srv.DSN, "--env", "dev", "--name", "scratch", "--inventory", path)
			if err != nil {
				t.Fatalf("connect dev: %v\n%s", err, errOut)
			}
			if !strings.Contains(out, "read-write session") || !strings.Contains(errOut, "not saved") {
				t.Errorf("dev connect:\n%s\n%s", out, errOut)
			}

			// Roadmap M1 acceptance: tl ls shows what connect saved.
			out, _, err = runApp(t, &app{}, "ls", "--inventory", path)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"scratch", "orders", fmt.Sprintf(" %d.", major), "community"} {
				if !strings.Contains(out, want) {
					t.Errorf("ls lacks %q:\n%s", want, out)
				}
			}
			if strings.Contains(out, pgtest.Password) {
				t.Fatalf("password in ls:\n%s", out)
			}
		})
	}
}
