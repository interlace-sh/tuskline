// SPDX-License-Identifier: Apache-2.0

package activity

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/sqlpack"
)

// Rower runs a single-row query. *pgx.Conn satisfies it.
type Rower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Signal asks PostgreSQL to cancel or terminate one backend. pid must be a
// positive backend id. terminate ends the session; otherwise only the
// current query is cancelled. The boolean is what the server function
// returned.
func Signal(ctx context.Context, db Rower, version int, pid int32, terminate bool) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("activity: no connection")
	}
	if pid <= 0 {
		return false, fmt.Errorf("activity: backend pid must be positive")
	}
	query := sqlpack.CancelBackend
	if terminate {
		query = sqlpack.TerminateBackend
	}
	picked, err := sqlpack.Pick(version, query)
	if err != nil {
		return false, err
	}
	var ok bool
	if err := db.QueryRow(ctx, picked.SQL, pid).Scan(&ok); err != nil {
		return false, fmt.Errorf("%s: %w", picked.Name, err)
	}
	return ok, nil
}

// Preview is the statement a signal would send. It does not connect.
func Preview(pid int32, terminate bool) string {
	name := "pg_cancel_backend"
	if terminate {
		name = "pg_terminate_backend"
	}
	return fmt.Sprintf("SELECT %s(%d)\nNothing has been sent.", name, pid)
}
