// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/collector"
	"github.com/interlace-sh/tuskline/core/sqlpack"
)

// Querier runs a query. *pgx.Conn satisfies it.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Statement is one top-level statement in the current database.
type Statement struct {
	QueryID    int64   `json:"query_id"`
	Query      string  `json:"query"`
	Calls      int64   `json:"calls"`
	TotalExec  float64 `json:"total_exec_time_ms"`
	Rows       int64   `json:"rows"`
	SharedHit  int64   `json:"shared_blks_hit"`
	SharedRead int64   `json:"shared_blks_read"`
	WALBytes   int64   `json:"wal_bytes"`
}

// Snapshot is a workload sample.
type Snapshot struct {
	Taken      time.Time   `json:"taken"`
	VersionNum int         `json:"server_version_num"`
	Statements []Statement `json:"statements"`
}

// Capture reads pg_stat_statements through the statistics pack. maxText
// truncates query text and limit caps the number of statements.
func Capture(ctx context.Context, q Querier, versionNum, maxText, limit int) (Snapshot, error) {
	query, err := sqlpack.Pick(versionNum, sqlpack.StatsStatements14, sqlpack.StatsStatements15, sqlpack.StatsStatements17)
	if err != nil {
		return Snapshot{}, err
	}
	rows, err := q.Query(ctx, query.SQL, maxText, limit)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%s: %w", query.Name, err)
	}
	defer rows.Close()
	names := rows.FieldDescriptions()
	var statements []Statement
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return Snapshot{}, fmt.Errorf("%s: %w", query.Name, err)
		}
		col := map[string]any{}
		for i, fd := range names {
			col[fd.Name] = vals[i]
		}
		statements = append(statements, Statement{
			QueryID: asInt(col["queryid"]), Query: asString(col["query"]),
			Calls: asInt(col["calls"]), TotalExec: asFloat(col["total_exec_time"]),
			Rows: asInt(col["rows"]), SharedHit: asInt(col["shared_blks_hit"]),
			SharedRead: asInt(col["shared_blks_read"]), WALBytes: asInt(col["wal_bytes"]),
		})
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("%s: %w", query.Name, err)
	}
	if statements == nil {
		statements = []Statement{}
	}
	return Snapshot{Taken: time.Now().UTC(), VersionNum: versionNum, Statements: statements}, nil
}

// WriteFile saves snap as JSON, mode 0600.
func WriteFile(path string, snap Snapshot) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snap); err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	return nil
}

// ReadFile loads a snapshot written by WriteFile.
func ReadFile(path string) (Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot: %w", err)
	}
	defer func() { _ = f.Close() }()
	return Read(f)
}

// Read decodes a snapshot.
func Read(r io.Reader) (Snapshot, error) {
	var snap Snapshot
	if err := json.NewDecoder(r).Decode(&snap); err != nil {
		return Snapshot{}, fmt.Errorf("snapshot: %w", err)
	}
	if snap.Statements == nil {
		snap.Statements = []Statement{}
	}
	return snap, nil
}

// Delta is one statement's change from an older snapshot to a newer one.
// Gone statements are not listed; a statement only in the newer snapshot
// has its whole value as the delta and Reset set.
type Delta struct {
	QueryID    int64   `json:"query_id"`
	Query      string  `json:"query"`
	Calls      int64   `json:"calls"`
	CallsReset bool    `json:"calls_reset,omitempty"`
	TotalExec  float64 `json:"total_exec_time_ms"`
	ExecReset  bool    `json:"exec_reset,omitempty"`
	Rows       int64   `json:"rows"`
	SharedHit  int64   `json:"shared_blks_hit"`
	SharedRead int64   `json:"shared_blks_read"`
	WALBytes   int64   `json:"wal_bytes"`
}

// Diff subtracts older from newer, matched on query id. A counter that
// fell is a statistics reset (collector.Delta).
func Diff(older, newer Snapshot) []Delta {
	prev := map[int64]Statement{}
	for _, s := range older.Statements {
		prev[s.QueryID] = s
	}
	var out []Delta
	for _, s := range newer.Statements {
		old, ok := prev[s.QueryID]
		if !ok {
			out = append(out, Delta{QueryID: s.QueryID, Query: s.Query, Calls: s.Calls, TotalExec: s.TotalExec, Rows: s.Rows, SharedHit: s.SharedHit, SharedRead: s.SharedRead, WALBytes: s.WALBytes})
			continue
		}
		calls, callsReset := collector.Delta(old.Calls, s.Calls)
		exec, execReset := collector.DeltaFloat(old.TotalExec, s.TotalExec)
		rows, _ := collector.Delta(old.Rows, s.Rows)
		hit, _ := collector.Delta(old.SharedHit, s.SharedHit)
		read, _ := collector.Delta(old.SharedRead, s.SharedRead)
		wal, _ := collector.Delta(old.WALBytes, s.WALBytes)
		out = append(out, Delta{QueryID: s.QueryID, Query: s.Query, Calls: calls, CallsReset: callsReset, TotalExec: exec, ExecReset: execReset, Rows: rows, SharedHit: hit, SharedRead: read, WALBytes: wal})
	}
	if out == nil {
		out = []Delta{}
	}
	return out
}

func asInt(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func asFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
