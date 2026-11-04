// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInProgress is returned when another session holds the migration lock.
var ErrInProgress = errors.New("another migration is already running")

// Advisory lock keys for pg_try_advisory_lock. They are fixed so two
// Tuskline sessions on the same database exclude each other.
const (
	advisoryKey1 int32 = 0x7475736b // tusk
	advisoryKey2 int32 = 0x6d696772 // migr
)

// Result is what Run applied and what it left alone because the checksum
// already matched.
type Result struct {
	Applied []string
	Skipped []string
}

// String reports the outcome, one script per line.
func (r Result) String() string {
	var b strings.Builder
	for _, name := range r.Skipped {
		fmt.Fprintf(&b, "%s already applied\n", name)
	}
	for _, name := range r.Applied {
		fmt.Fprintf(&b, "applied %s\n", name)
	}
	if len(r.Applied) == 0 {
		b.WriteString("Nothing to apply.\n")
	}
	return b.String()
}

// Run applies folder on conn. It takes a session advisory lock, creates
// tuskline.schema_migration when that table is missing, and runs each
// pending script in one transaction. A script that PostgreSQL cannot run
// inside a transaction, such as CREATE INDEX CONCURRENTLY, is run on its
// own and recorded afterwards. The first error stops the run. Scripts
// that already committed stay applied.
func Run(ctx context.Context, conn *pgx.Conn, folder Folder) (Result, error) {
	if conn == nil {
		return Result{}, errors.New("migrate: no connection")
	}
	unlock, err := lockSession(ctx, conn)
	if err != nil {
		return Result{}, err
	}
	defer unlock()

	if err := ensureHistory(ctx, conn); err != nil {
		return Result{}, err
	}
	applied, err := readHistory(ctx, conn)
	if err != nil {
		return Result{}, err
	}
	pending, err := Pending(folder.Files, applied)
	if err != nil {
		return Result{}, err
	}
	var result Result
	appliedName := map[string]bool{}
	for _, row := range applied {
		if row.Success {
			appliedName[row.Name] = true
		}
	}
	for _, file := range folder.Files {
		if appliedName[file.Name] {
			result.Skipped = append(result.Skipped, file.Name)
		}
	}
	for _, file := range pending {
		if err := applyFile(ctx, conn, file); err != nil {
			return result, err
		}
		result.Applied = append(result.Applied, file.Name)
	}
	return result, nil
}

func applyFile(ctx context.Context, conn *pgx.Conn, file File) error {
	script, err := prepare(file)
	if err != nil {
		return err
	}
	started := time.Now()
	if script.outside {
		for _, stmt := range script.stmts {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				_ = record(ctx, conn, file, false, time.Since(started))
				return fmt.Errorf("apply %s: %w", file.Name, err)
			}
		}
		if err := record(ctx, conn, file, true, time.Since(started)); err != nil {
			return fmt.Errorf("record %s: %w", file.Name, err)
		}
		return nil
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("apply %s: %w", file.Name, err)
	}
	for _, stmt := range script.stmts {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", file.Name, err)
		}
	}
	if err := recordTx(ctx, tx, file, true, time.Since(started)); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("record %s: %w", file.Name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("apply %s: %w", file.Name, err)
	}
	return nil
}

func lockSession(ctx context.Context, conn *pgx.Conn) (func(), error) {
	var got bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1, $2)`, advisoryKey1, advisoryKey2).Scan(&got); err != nil {
		return nil, fmt.Errorf("migration lock: %w", err)
	}
	if !got {
		return nil, ErrInProgress
	}
	return func() {
		// Unlock even when the apply context is already cancelled.
		unlockCtx := context.WithoutCancel(ctx)
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1, $2)`, advisoryKey1, advisoryKey2)
	}, nil
}

func ensureHistory(ctx context.Context, conn *pgx.Conn) error {
	if _, err := conn.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS tuskline`); err != nil {
		return fmt.Errorf("create migration schema: %w", err)
	}
	_, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS tuskline.schema_migration (
		name text PRIMARY KEY,
		checksum text NOT NULL,
		applied_at timestamptz NOT NULL,
		execution_ms bigint NOT NULL,
		success boolean NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create migration history: %w", err)
	}
	return nil
}

func readHistory(ctx context.Context, conn *pgx.Conn) ([]Applied, error) {
	rows, err := conn.Query(ctx, `SELECT name, checksum, success FROM tuskline.schema_migration`)
	if err != nil {
		return nil, fmt.Errorf("read migration history: %w", err)
	}
	defer rows.Close()
	var out []Applied
	for rows.Next() {
		var row Applied
		if err := rows.Scan(&row.Name, &row.Checksum, &row.Success); err != nil {
			return nil, fmt.Errorf("read migration history: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read migration history: %w", err)
	}
	return out, nil
}

func record(ctx context.Context, conn *pgx.Conn, file File, success bool, elapsed time.Duration) error {
	_, err := conn.Exec(ctx, insertHistory, file.Name, file.Checksum, time.Now().UTC(), elapsed.Milliseconds(), success)
	return err
}

func recordTx(ctx context.Context, tx pgx.Tx, file File, success bool, elapsed time.Duration) error {
	_, err := tx.Exec(ctx, insertHistory, file.Name, file.Checksum, time.Now().UTC(), elapsed.Milliseconds(), success)
	return err
}

const insertHistory = `INSERT INTO tuskline.schema_migration
	(name, checksum, applied_at, execution_ms, success)
	VALUES ($1, $2, $3, $4, $5)`
