// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"fmt"
	"time"
)

// QueryAudit is one production statement run from the editor. It does not
// contain result rows.
type QueryAudit struct {
	At          time.Time
	Actor       string
	Environment string
	Cluster     string
	SQL         string
	RowCount    int
	Error       string
}

// RecordQuery appends an audit entry. Actor, environment, cluster and SQL
// are required.
func (s *Store) RecordQuery(ctx context.Context, entry QueryAudit) error {
	if entry.Actor == "" || entry.Environment == "" || entry.Cluster == "" || entry.SQL == "" {
		return fmt.Errorf("store: audit entry is incomplete")
	}
	at := entry.At
	if at.IsZero() {
		at = time.Now()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO query_audit
		(at, actor, environment, cluster, sql, row_count, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		at.UTC().Format(time.RFC3339Nano), entry.Actor, entry.Environment, entry.Cluster, entry.SQL, entry.RowCount, entry.Error)
	if err != nil {
		return fmt.Errorf("store: record query: %w", err)
	}
	return nil
}

// QueryAuditCount is the number of recorded statements. Tests use it.
func (s *Store) QueryAuditCount(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM query_audit`).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count query audit: %w", err)
	}
	return n, nil
}
