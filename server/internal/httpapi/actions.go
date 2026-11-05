// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/activity"
	"github.com/interlace-sh/tuskline/core/backup"
	"github.com/interlace-sh/tuskline/core/ha"
	"github.com/interlace-sh/tuskline/core/hypo"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/migrate"
	"github.com/interlace-sh/tuskline/core/store"
)

type signalRequest struct {
	PID       int32  `json:"pid"`
	Terminate bool   `json:"terminate"`
	Confirm   string `json:"confirm"`
	Run       bool   `json:"run"`
}

func (a *api) clusterSignal(w http.ResponseWriter, r *http.Request) {
	var req signalRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid signal"))
		return
	}
	cl, env, err := a.savedCluster(r)
	if err != nil {
		writeMissing(w, err)
		return
	}
	preview := activity.Preview(req.PID, req.Terminate)
	if !req.Run {
		writeJSON(w, http.StatusOK, map[string]any{"preview": preview, "sent": false})
		return
	}
	if req.PID <= 0 {
		writeErr(w, http.StatusBadRequest, errors.New("backend pid must be positive"))
		return
	}
	if env.Policy.RequireTypedConfirmation && req.Confirm != cl.Name {
		writeErr(w, http.StatusBadRequest, errors.New("type the cluster name to confirm"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), clusterCallTimeout)
	defer cancel()
	db, version, err := a.open(ctx, cl.Ref())
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	defer func() { _ = db.Close(context.Background()) }()
	ok, sigErr := activity.Signal(ctx, db, version, req.PID, req.Terminate)
	failure := ""
	if sigErr != nil {
		failure = sigErr.Error()
	}
	action := "cancel"
	if req.Terminate {
		action = "terminate"
	}
	if err := a.recordAction(ctx, cl.Environment, cl.Name, action, preview, failure); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if sigErr != nil {
		writeErr(w, http.StatusBadGateway, sigErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preview": preview, "sent": ok})
}

func (a *api) clusterRestart(w http.ResponseWriter, r *http.Request) {
	var req switchRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid restart"))
		return
	}
	cl, env, err := a.savedCluster(r)
	if err != nil {
		writeMissing(w, err)
		return
	}
	if cl.Patroni == nil || cl.Patroni.URL == "" {
		writeErr(w, http.StatusNotFound, errors.New("this cluster has no Patroni URL"))
		return
	}
	action := ha.Restart{
		Environment: cl.Environment, Cluster: cl.Name, API: cl.Patroni.URL,
		Prod: env.Policy.RequireTypedConfirmation, Confirm: req.Confirm,
	}
	if !req.Run {
		writeJSON(w, http.StatusOK, map[string]any{"preview": action.Preview(), "sent": false})
		return
	}
	if err := action.Check(); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	sendErr := postLimited(ctx, strings.TrimRight(action.API, "/")+"/restart", ha.RestartBody(time.Now().UTC()))
	failure := ""
	if sendErr != nil {
		failure = sendErr.Error()
	}
	if err := a.recordAction(ctx, cl.Environment, cl.Name, "restart", action.Preview(), failure); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if sendErr != nil {
		writeErr(w, http.StatusBadGateway, sendErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preview": action.Preview(), "sent": true})
}

func (a *api) parseBackup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	report, err := backup.ParseInfo(raw)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

type hypoRequest struct {
	Index string `json:"index"`
	Query string `json:"query"`
}

func (a *api) clusterHypo(w http.ResponseWriter, r *http.Request) {
	var req hypoRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid hypopg request"))
		return
	}
	a.withCluster(w, r, func(ctx context.Context, _ inventory.Ref, db *pgx.Conn, _ int) {
		result, err := hypo.Compare(ctx, db, req.Index, req.Query)
		if err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, hypo.ErrUnavailable) {
				status = http.StatusConflict
			}
			writeErr(w, status, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
}

func (a *api) clusterMigrate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid migration upload"))
		return
	}
	cl, env, err := a.savedCluster(r)
	if err != nil {
		writeMissing(w, err)
		return
	}
	dir, err := os.MkdirTemp("", "tuskline-migrate-")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer func() { _ = os.RemoveAll(dir) }()
	files := r.MultipartForm.File["file"]
	if len(files) == 0 || len(files) > 32 {
		writeErr(w, http.StatusBadRequest, errors.New("upload between 1 and 32 sql files"))
		return
	}
	for _, header := range files {
		name := filepath.Base(header.Filename)
		if !strings.HasSuffix(strings.ToLower(name), ".sql") || strings.Contains(name, "..") {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("%s is not a sql file name", header.Filename))
			return
		}
		if header.Size > 1<<20 {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("%s is larger than 1MB", name))
			return
		}
		src, err := header.Open()
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(src, 1<<20+1))
		_ = src.Close()
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	folder, err := migrate.Load(dir)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	report, err := folder.Lint()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	run := r.FormValue("run") == "true"
	if !run {
		writeJSON(w, http.StatusOK, map[string]any{"preview": folder.Preview(), "lint": report, "applied": []string{}})
		return
	}
	if env.Kind == inventory.EnvProd && r.FormValue("approve") != "true" {
		writeErr(w, http.StatusBadRequest, migrate.ErrApproval)
		return
	}
	if env.Policy.RequireTypedConfirmation && r.FormValue("confirm") != cl.Name {
		writeErr(w, http.StatusBadRequest, errors.New("type the cluster name to confirm"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	db, err := a.openWritable(ctx, cl.Ref())
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	defer func() { _ = db.Close(context.Background()) }()
	result, runErr := migrate.Run(ctx, db, folder)
	failure := ""
	if runErr != nil {
		failure = runErr.Error()
	}
	if err := a.recordAction(ctx, cl.Environment, cl.Name, "migrate", folder.List(), failure); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if runErr != nil {
		writeErr(w, http.StatusBadGateway, runErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preview": folder.Preview(), "lint": report, "applied": result.Applied})
}

func (a *api) clusterMetrics(w http.ResponseWriter, r *http.Request) {
	ref, err := inventory.ParseRef(r.PathValue("env") + "/" + r.PathValue("name"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	res := r.URL.Query().Get("res")
	if res == "" {
		res = store.Res1m
	}
	if res != store.Res1s && res != store.Res1m && res != store.Res1h {
		writeErr(w, http.StatusBadRequest, errors.New("res must be 1s, 1m or 1h"))
		return
	}
	root, err := metricsRoot(a.storePath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	to := time.Now().UTC()
	from := to.Add(-24 * time.Hour)
	samples, err := (store.Segments{Root: root}).Read(ref.String(), res, from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	name := r.URL.Query().Get("name")
	out := make([]sampleView, 0, len(samples))
	for _, sample := range samples {
		if name != "" && sample.Name != name {
			continue
		}
		out = append(out, sampleView{At: sample.At, Name: sample.Name, Value: sample.Average()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"samples": out})
}

type sampleView struct {
	At    time.Time `json:"at"`
	Name  string    `json:"name"`
	Value float64   `json:"value"`
}

func metricsRoot(storePath string) (string, error) {
	if path := os.Getenv("TUSKLINE_METRICS"); path != "" {
		return path, nil
	}
	if storePath == "" {
		var err error
		storePath, err = store.DefaultPath()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(filepath.Dir(storePath), "metrics"), nil
}

func (a *api) openWritable(ctx context.Context, ref inventory.Ref) (*pgx.Conn, error) {
	db, _, err := a.open(ctx, ref)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(ctx, `SELECT set_config('default_transaction_read_only', 'off', false)`); err != nil {
		_ = db.Close(context.Background())
		return nil, fmt.Errorf("allow migration writes: %w", err)
	}
	if _, err := db.Exec(ctx, `SELECT set_config('statement_timeout', '0', false)`); err != nil {
		_ = db.Close(context.Background())
		return nil, fmt.Errorf("clear migration statement timeout: %w", err)
	}
	return db, nil
}
