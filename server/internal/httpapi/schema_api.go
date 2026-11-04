// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/lint"
	"github.com/interlace-sh/tuskline/core/schema"
)

func (a *api) clusterSchema(w http.ResponseWriter, r *http.Request) {
	a.withCluster(w, r, func(ctx context.Context, ref inventory.Ref, db *pgx.Conn, _ int) {
		snap, err := schema.Read(ctx, db)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Ref string `json:"ref"`
			schema.Snapshot
		}{Ref: ref.String(), Snapshot: snap})
	})
}

// schemaDiff compares two inventory clusters. Both sessions stay on the
// environment policy, so production is read-only. Rows are not read.
func (a *api) schemaDiff(w http.ResponseWriter, r *http.Request) {
	fromRef, err := inventory.ParseRef(r.URL.Query().Get("from"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	toRef, err := inventory.ParseRef(r.URL.Query().Get("to"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx := r.Context()
	before, err := a.readSchema(ctx, fromRef)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	after, err := a.readSchema(ctx, toRef)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	changes := schema.Diff(before, after)
	out := make([]schemaChange, 0, len(changes))
	for _, change := range changes {
		out = append(out, schemaChange{Op: change.Op, Kind: change.Kind, Name: change.Name, From: change.From, To: change.To})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from":    fromRef.String(),
		"to":      toRef.String(),
		"report":  schema.Report(changes),
		"changes": out,
	})
}

type schemaChange struct {
	Op   string `json:"op"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

func (a *api) readSchema(ctx context.Context, ref inventory.Ref) (schema.Snapshot, error) {
	db, _, err := a.open(ctx, ref)
	if err != nil {
		return schema.Snapshot{}, err
	}
	defer func() { _ = db.Close(context.Background()) }()
	return schema.Read(ctx, db)
}

type lintRequest struct {
	SQL string `json:"sql"`
}

// lintSQL runs the lock rules on one script. It does not connect.
func (a *api) lintSQL(w http.ResponseWriter, r *http.Request) {
	var req lintRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid lint request"))
		return
	}
	found, err := lint.Check(req.SQL, lint.Stats{})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if found == nil {
		found = []lint.Finding{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": found, "report": lint.Report(found)})
}
