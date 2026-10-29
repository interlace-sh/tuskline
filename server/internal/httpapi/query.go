// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/store"
)

// localActor is the operator of a loopback server until M9 adds sign-in.
const localActor = "local"

// maxQueryBytes caps the statement body. It is not a result-size limit.
const maxQueryBytes = 1 << 20

type queryRequest struct {
	SQL string `json:"sql"`
}

type queryResponse struct {
	Columns   []string `json:"columns"`
	Rows      [][]any  `json:"rows"`
	Truncated bool     `json:"truncated"`
	Duration  int64    `json:"duration_ms"`
}

func (a *api) clusterQuery(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxQueryBytes+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(body) > maxQueryBytes {
		writeErr(w, http.StatusRequestEntityTooLarge, fmt.Errorf("statement is larger than %d bytes", maxQueryBytes))
		return
	}
	var req queryRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid query request"))
		return
	}
	statement, err := oneStatement(req.SQL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ref, err := inventory.ParseRef(r.PathValue("env") + "/" + r.PathValue("name"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	inv, err := a.inventory()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	env, ok := inv.Environment(ref.Env)
	if !ok {
		writeErr(w, http.StatusNotFound, fmt.Errorf("%s: environment is missing", ref))
		return
	}
	limit := env.Policy.RowLimit
	if limit <= 0 {
		limit = 10000
	}
	var (
		columns   []string
		rows      [][]any
		truncated bool
		elapsed   time.Duration
		queryErr  error
	)
	a.withCluster(w, r, func(ctx context.Context, _ inventory.Ref, db *pgx.Conn, _ int) {
		start := time.Now()
		columns, rows, truncated, queryErr = readQuery(ctx, db, statement, limit)
		elapsed = time.Since(start)
		if env.Kind == inventory.EnvProd {
			message := ""
			if queryErr != nil {
				message = queryErr.Error()
			}
			if err := a.recordProdQuery(ctx, ref, statement, len(rows), message); err != nil {
				writeErr(w, http.StatusInternalServerError, fmt.Errorf("audit store: %w", err))
				return
			}
		}
		if queryErr != nil {
			writeErr(w, http.StatusBadRequest, queryErr)
			return
		}
		writeJSON(w, http.StatusOK, queryResponse{
			Columns: columns, Rows: rows, Truncated: truncated, Duration: elapsed.Milliseconds(),
		})
	})
}

func readQuery(ctx context.Context, db *pgx.Conn, statement string, limit int) ([]string, [][]any, bool, error) {
	pgRows, err := db.Query(ctx, statement)
	if err != nil {
		return nil, nil, false, err
	}
	defer pgRows.Close()
	fields := pgRows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, field := range fields {
		columns[i] = string(field.Name)
	}
	out := make([][]any, 0)
	truncated := false
	for pgRows.Next() {
		if len(out) >= limit {
			truncated = true
			break
		}
		values, err := pgRows.Values()
		if err != nil {
			return nil, nil, false, err
		}
		row := make([]any, len(values))
		for i, value := range values {
			row[i] = jsonCell(value)
		}
		out = append(out, row)
	}
	if err := pgRows.Err(); err != nil {
		return nil, nil, false, err
	}
	return columns, out, truncated, nil
}

func jsonCell(value any) any {
	switch v := value.(type) {
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano)
	case []byte:
		if utf8.Valid(v) {
			return string(v)
		}
		return fmt.Sprintf("%x", v)
	default:
		return v
	}
}

func (a *api) recordProdQuery(ctx context.Context, ref inventory.Ref, statement string, rows int, message string) error {
	path := a.storePath
	if path == "" {
		var err error
		path, err = store.DefaultPath()
		if err != nil {
			return fmt.Errorf("audit store: %w", err)
		}
	}
	db, err := store.Open(ctx, path)
	if err != nil {
		return fmt.Errorf("audit store: %w", err)
	}
	defer func() { _ = db.Close() }()
	return db.RecordQuery(ctx, store.QueryAudit{
		Actor: localActor, Environment: ref.Env, Cluster: ref.Cluster,
		SQL: statement, RowCount: rows, Error: message,
	})
}
