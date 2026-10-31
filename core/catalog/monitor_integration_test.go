// SPDX-License-Identifier: Apache-2.0

//go:build integration

package catalog_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/catalog"
	"github.com/interlace-sh/tuskline/core/pgtest"
)

// TestMonitorRoleApplied checks the generated script on a fresh server.
// The password is applied by the test, not by the script.
func TestMonitorRoleApplied(t *testing.T) {
	const password = "monitor-test-only"
	for _, major := range pgtest.Versions(t) {
		t.Run("pg"+strconv.Itoa(major), func(t *testing.T) {
			super := pgtest.Start(t, major).Profile
			for _, stmt := range catalog.MonitorStatements() {
				if strings.Contains(strings.ToUpper(stmt), "PASSWORD") {
					t.Fatalf("statement sets a password: %s", stmt)
				}
			}
			pgtest.Exec(t, super, catalog.MonitorStatements()...)
			pgtest.Exec(t, super, "ALTER ROLE "+catalog.MonitorRole+" PASSWORD '"+password+"'")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cfg, err := pgtest.As(super, catalog.MonitorRole, password).Config()
			if err != nil {
				t.Fatal(err)
			}
			c, err := pgx.ConnectConfig(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = c.Close(context.Background()) })

			var (
				connLimit int
				monitor   bool
				readOnly  string
				stmt      string
				lock      string
				idle      string
			)
			if err := c.QueryRow(ctx, `SELECT rolconnlimit FROM pg_roles WHERE rolname = current_user`).Scan(&connLimit); err != nil {
				t.Fatal(err)
			}
			if connLimit != 5 {
				t.Errorf("connection limit = %d, want 5", connLimit)
			}
			if err := c.QueryRow(ctx, `SELECT pg_has_role(current_user, 'pg_monitor', 'member')`).Scan(&monitor); err != nil {
				t.Fatal(err)
			}
			if !monitor {
				t.Error("role is not a member of pg_monitor")
			}
			if err := c.QueryRow(ctx, `SHOW default_transaction_read_only`).Scan(&readOnly); err != nil {
				t.Fatal(err)
			}
			if readOnly != "on" {
				t.Errorf("default_transaction_read_only = %s, want on", readOnly)
			}
			if err := c.QueryRow(ctx, `SHOW statement_timeout`).Scan(&stmt); err != nil || stmt != "5s" {
				t.Errorf("statement_timeout = %s, %v", stmt, err)
			}
			if err := c.QueryRow(ctx, `SHOW lock_timeout`).Scan(&lock); err != nil || lock != "1s" {
				t.Errorf("lock_timeout = %s, %v", lock, err)
			}
			if err := c.QueryRow(ctx, `SHOW idle_in_transaction_session_timeout`).Scan(&idle); err != nil || idle != "10s" {
				t.Errorf("idle_in_transaction_session_timeout = %s, %v", idle, err)
			}
			if _, err := c.Exec(ctx, `CREATE TEMP TABLE tuskline_monitor_write (id int)`); err == nil || !strings.Contains(err.Error(), "read-only") {
				t.Fatalf("write: %v", err)
			}
			var one int
			if err := c.QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil || one != 1 {
				t.Fatalf("read: %v", err)
			}
		})
	}
}
