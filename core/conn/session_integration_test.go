// SPDX-License-Identifier: Apache-2.0

//go:build integration

package conn_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// SQLSTATEs the policy must produce.
const (
	readOnlySQLTransaction = "25006"
	queryCanceled          = "57014"
	lockNotAvailable       = "55P03"
)

func sqlState(err error) string {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return pg.Code
	}
	return ""
}

func connect(t *testing.T, p conn.Profile, s conn.SessionSettings) *pgx.Conn {
	t.Helper()
	c, err := conn.Connect(context.Background(), p, s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c
}

func TestSessionPolicyAcrossVersions(t *testing.T) {
	for _, major := range pgVersions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			super := startPG(t, major)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("prod rejects writes", func(t *testing.T) {
				c := connect(t, super, inventory.DefaultPolicy(inventory.EnvProd).Session())
				_, err := c.Exec(ctx, "CREATE TABLE prod_write (id int)")
				if got := sqlState(err); got != readOnlySQLTransaction {
					t.Fatalf("CREATE TABLE in a prod session: err = %v (SQLSTATE %q), want %s", err, got, readOnlySQLTransaction)
				}
				var ro, st, lt string
				if err := c.QueryRow(ctx, "SELECT current_setting('default_transaction_read_only'), current_setting('statement_timeout'), current_setting('lock_timeout')").Scan(&ro, &st, &lt); err != nil {
					t.Fatal(err)
				}
				if ro != "on" || st != "30s" || lt != "2s" {
					t.Fatalf("prod session: read_only=%s statement_timeout=%s lock_timeout=%s, want on, 30s, 2s", ro, st, lt)
				}
			})

			t.Run("dev allows the same write", func(t *testing.T) {
				c := connect(t, super, inventory.DefaultPolicy(inventory.EnvDev).Session())
				if _, err := c.Exec(ctx, "CREATE TABLE dev_write (id int)"); err != nil {
					t.Fatalf("CREATE TABLE in a dev session: %v", err)
				}
				if _, err := c.Exec(ctx, "INSERT INTO dev_write VALUES ($1)", 1); err != nil {
					t.Fatalf("INSERT in a dev session: %v", err)
				}
			})

			t.Run("statement timeout cancels", func(t *testing.T) {
				c := connect(t, super, conn.SessionSettings{StatementTimeout: 100 * time.Millisecond})
				_, err := c.Exec(ctx, "SELECT pg_sleep(5)")
				if got := sqlState(err); got != queryCanceled {
					t.Fatalf("pg_sleep(5) with a 100 ms statement_timeout: err = %v (SQLSTATE %q), want %s", err, got, queryCanceled)
				}
			})

			t.Run("lock timeout gives up", func(t *testing.T) {
				holder := connect(t, super, conn.SessionSettings{})
				if _, err := holder.Exec(ctx, "CREATE TABLE IF NOT EXISTS locked (id int)"); err != nil {
					t.Fatal(err)
				}
				tx, err := holder.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = tx.Rollback(context.Background()) }()
				if _, err := tx.Exec(ctx, "LOCK TABLE locked IN ACCESS EXCLUSIVE MODE"); err != nil {
					t.Fatal(err)
				}
				waiter := connect(t, super, conn.SessionSettings{LockTimeout: 100 * time.Millisecond})
				_, err = waiter.Exec(ctx, "SELECT count(*) FROM locked")
				if got := sqlState(err); got != lockNotAvailable {
					t.Fatalf("read of a locked table with a 100 ms lock_timeout: err = %v (SQLSTATE %q), want %s", err, got, lockNotAvailable)
				}
			})

			t.Run("dev keeps a read-only role read-only", func(t *testing.T) {
				execAll(t, super,
					"CREATE ROLE ro_login LOGIN PASSWORD 'ro-test-only'",
					"GRANT CREATE ON SCHEMA public TO ro_login",
					"ALTER ROLE ro_login SET default_transaction_read_only = on",
				)
				c := connect(t, as(super, "ro_login", "ro-test-only"), inventory.DefaultPolicy(inventory.EnvDev).Session())
				_, err := c.Exec(ctx, "CREATE TABLE ro_write (id int)")
				if got := sqlState(err); got != readOnlySQLTransaction {
					t.Fatalf("dev session for a read-only role: err = %v (SQLSTATE %q), want %s", err, got, readOnlySQLTransaction)
				}
			})
		})
	}
}
