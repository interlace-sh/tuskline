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

// localActor is the operator of an unauthenticated server. Sign-in is a later feature.
const localActor = "local"

// maxQueryBytes caps the statement body. It is not a result-size limit.
const maxQueryBytes = 1 << 20

type queryRequest struct {
	SQL string `json:"sql"`
}

type queryHead struct {
	Columns []string `json:"columns"`
}

type queryTail struct {
	End       bool  `json:"end"`
	Truncated bool  `json:"truncated"`
	Duration  int64 `json:"duration_ms"`
	Rows      int   `json:"rows"`
}

type queryStreamError struct {
	Error string `json:"error"`
}

// writeQueryEvent writes one newline-delimited JSON value.
func writeQueryEvent(w io.Writer, v any) error {
	encoded, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(append(encoded, '\n'))
	return err
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
	a.withCluster(w, r, func(ctx context.Context, _ inventory.Ref, db *pgx.Conn, _ int) {
		if env.Kind == inventory.EnvProd {
			if err := a.auditReady(ctx); err != nil {
				writeErr(w, http.StatusInternalServerError, fmt.Errorf("audit store: %w", err))
				return
			}
		}
		start := time.Now()
		pgRows, err := db.Query(ctx, statement)
		if err != nil {
			a.finishQueryError(ctx, w, env.Kind, ref, statement, err)
			return
		}
		defer pgRows.Close()

		fields := pgRows.FieldDescriptions()
		columns := make([]string, len(fields))
		for i, field := range fields {
			columns[i] = field.Name
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flush := flusher(w)
		if err := writeQueryEvent(w, queryHead{Columns: columns}); err != nil {
			return
		}
		flush()

		sent := 0
		truncated := false
		var streamErr error
		for pgRows.Next() {
			if sent >= limit {
				truncated = true
				break
			}
			values, err := pgRows.Values()
			if err != nil {
				streamErr = err
				break
			}
			row := make([]any, len(values))
			for i, value := range values {
				row[i] = jsonCell(value)
			}
			if err := writeQueryEvent(w, row); err != nil {
				return
			}
			sent++
			if sent%64 == 0 {
				flush()
			}
		}
		if streamErr == nil {
			streamErr = pgRows.Err()
		}
		elapsed := time.Since(start)
		if env.Kind == inventory.EnvProd {
			message := ""
			if streamErr != nil {
				message = streamErr.Error()
			}
			if err := a.recordProdQuery(ctx, ref, statement, sent, message); err != nil {
				_ = writeQueryEvent(w, queryStreamError{Error: "audit store: " + err.Error()})
				flush()
				return
			}
		}
		if streamErr != nil {
			_ = writeQueryEvent(w, queryStreamError{Error: streamErr.Error()})
			flush()
			return
		}
		_ = writeQueryEvent(w, queryTail{
			End: true, Truncated: truncated, Duration: elapsed.Milliseconds(), Rows: sent,
		})
		flush()
	})
}

func flusher(w http.ResponseWriter) func() {
	f, ok := w.(http.Flusher)
	if !ok {
		return func() {}
	}
	return f.Flush
}

func (a *api) finishQueryError(ctx context.Context, w http.ResponseWriter, kind inventory.EnvKind, ref inventory.Ref, statement string, queryErr error) {
	if kind == inventory.EnvProd {
		if err := a.recordProdQuery(ctx, ref, statement, 0, queryErr.Error()); err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Errorf("audit store: %w", err))
			return
		}
	}
	writeErr(w, http.StatusBadRequest, queryErr)
}

func (a *api) auditReady(ctx context.Context) error {
	db, err := a.openAudit(ctx)
	if err != nil {
		return err
	}
	return db.Close()
}

func (a *api) openAudit(ctx context.Context) (*store.Store, error) {
	path := a.storePath
	if path == "" {
		var err error
		path, err = store.DefaultPath()
		if err != nil {
			return nil, fmt.Errorf("audit store: %w", err)
		}
	}
	db, err := store.Open(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("audit store: %w", err)
	}
	return db, nil
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
	db, err := a.openAudit(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	return db.RecordQuery(ctx, store.QueryAudit{
		Actor: localActor, Environment: ref.Env, Cluster: ref.Cluster,
		SQL: statement, RowCount: rows, Error: message,
	})
}
