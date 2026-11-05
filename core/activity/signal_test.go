// SPDX-License-Identifier: Apache-2.0

package activity

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeRow struct {
	ok  bool
	err error
}

func (f fakeRow) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*(dest[0].(*bool)) = f.ok
	return nil
}

type fakeDB struct {
	sql string
	pid int32
	row fakeRow
}

func (f *fakeDB) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.sql = sql
	f.pid = args[0].(int32)
	return f.row
}

func TestSignalCancel(t *testing.T) {
	t.Parallel()
	db := &fakeDB{row: fakeRow{ok: true}}
	ok, err := Signal(context.Background(), db, 170000, 42, false)
	if err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	if db.sql != "SELECT pg_cancel_backend($1)" || db.pid != 42 {
		t.Fatalf("%s %d", db.sql, db.pid)
	}
	if _, err := Signal(context.Background(), db, 170000, 0, true); err == nil {
		t.Fatal("expected a pid error")
	}
}

func TestPreview(t *testing.T) {
	t.Parallel()
	if got := Preview(7, true); got != "SELECT pg_terminate_backend(7)\nNothing has been sent." {
		t.Fatal(got)
	}
}
