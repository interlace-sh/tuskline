// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/advisor"
	"github.com/interlace-sh/tuskline/core/catalog"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/provision"
	"github.com/interlace-sh/tuskline/core/store"
)

var settingName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func (a *api) sampleCluster(ctx context.Context, ref inventory.Ref, db *pgx.Conn) {
	var hit, read int64
	var sessions int
	var lag float64
	err := db.QueryRow(ctx, `SELECT
		COALESCE((SELECT SUM(blks_hit) FROM pg_stat_database), 0),
		COALESCE((SELECT SUM(blks_read) FROM pg_stat_database), 0),
		(SELECT count(*)::int FROM pg_stat_activity),
		COALESCE((SELECT EXTRACT(EPOCH FROM MAX(replay_lag)) FROM pg_stat_replication), 0)`).Scan(&hit, &read, &sessions, &lag)
	if err != nil {
		return
	}
	ratio := 1.0
	if hit+read > 0 {
		ratio = float64(hit) / float64(hit+read)
	}
	now := time.Now().UTC()
	root, err := metricsRoot(a.storePath)
	if err != nil {
		return
	}
	_ = (store.Segments{Root: root}).Write(ref.String(), []store.Sample{
		store.Gauge(now, "cache_hit_ratio", ratio),
		store.Gauge(now, "sessions", float64(sessions)),
		store.Gauge(now, "replica_lag_seconds", lag),
	})
}

type applyAdviceRequest struct {
	Names   []string `json:"names"`
	Confirm string   `json:"confirm"`
	Ticket  string   `json:"ticket"`
}

// applyAdvice runs ALTER SYSTEM for advice the server just recomputed.
// A reload or restart is still required. Production needs the cluster name.
func (a *api) applyAdvice(w http.ResponseWriter, r *http.Request) {
	var req applyAdviceRequest
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid advice apply"))
		return
	}
	hw, err := adviseHardware(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	cl, env, err := a.savedCluster(r)
	if err != nil {
		writeMissing(w, err)
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
	current, ev, err := advisor.Collect(ctx, db, version)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	report, err := advisor.Advise(version, hw, current, ev)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	want := map[string]bool{}
	for _, name := range req.Names {
		want[name] = true
	}
	var statements []string
	restart := false
	for _, item := range report.Advice {
		if len(want) > 0 && !want[item.Name] {
			continue
		}
		stmt, err := alterSystem(item.Name, item.Proposed)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		statements = append(statements, stmt)
		if item.Apply == "restart" {
			restart = true
		}
	}
	if len(statements) == 0 {
		writeErr(w, http.StatusBadRequest, errors.New("none of the named settings are in the current advice"))
		return
	}
	if _, err := db.Exec(ctx, `SELECT set_config('default_transaction_read_only', 'off', false)`); err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	for _, stmt := range statements {
		if _, err := db.Exec(ctx, stmt); err != nil {
			_ = a.recordAction(ctx, cl.Environment, cl.Name, "config", strings.Join(statements, "\n"), err.Error())
			writeErr(w, http.StatusBadGateway, err)
			return
		}
	}
	note := "ALTER SYSTEM recorded and the configuration was reloaded."
	if restart {
		note = "ALTER SYSTEM recorded. At least one setting needs a restart. This server did not restart PostgreSQL."
	} else if _, err := db.Exec(ctx, `SELECT pg_reload_conf()`); err != nil {
		note = "ALTER SYSTEM recorded. pg_reload_conf failed: " + err.Error()
	}
	_ = a.recordAction(ctx, cl.Environment, cl.Name, "config", strings.Join(statements, "\n")+"\n"+req.Ticket, "")
	writeJSON(w, http.StatusOK, map[string]any{"applied": statements, "note": note})
}

func alterSystem(name, value string) (string, error) {
	if !settingName.MatchString(name) {
		return "", fmt.Errorf("refusing setting %q", name)
	}
	if value == "" || strings.ContainsAny(value, "'\n;\x00") {
		return "", fmt.Errorf("refusing value for %s", name)
	}
	return fmt.Sprintf("ALTER SYSTEM SET %s = '%s'", name, value), nil
}

type setupApplyRequest struct {
	Confirm string `json:"confirm"`
}

// applySetup runs CREATE EXTENSION for looked-for extensions that are missing
// and that the catalogue installs with SQL. Package installs stay in the preview.
func (a *api) applySetup(w http.ResponseWriter, r *http.Request) {
	var req setupApplyRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid setup apply"))
		return
	}
	cl, env, err := a.savedCluster(r)
	if err != nil {
		writeMissing(w, err)
		return
	}
	if env.Policy.RequireTypedConfirmation && req.Confirm != cl.Name {
		writeErr(w, http.StatusBadRequest, errors.New("type the cluster name to confirm"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), clusterCallTimeout)
	defer cancel()
	db, _, err := a.open(ctx, cl.Ref())
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	defer func() { _ = db.Close(context.Background()) }()
	info, err := conn.Inspect(ctx, db)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	installed := map[string]bool{}
	for _, ext := range info.Extensions {
		if ext.Installed() {
			installed[ext.Name] = true
		}
	}
	if _, err := db.Exec(ctx, `SELECT set_config('default_transaction_read_only', 'off', false)`); err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	var applied []string
	for _, ext := range catalog.LookedFor() {
		if !ext.Create || installed[ext.Name] {
			continue
		}
		if !settingName.MatchString(ext.Name) {
			continue
		}
		stmt := "CREATE EXTENSION IF NOT EXISTS " + ext.Name
		if _, err := db.Exec(ctx, stmt); err != nil {
			_ = a.recordAction(ctx, cl.Environment, cl.Name, "setup", stmt, err.Error())
			writeErr(w, http.StatusBadGateway, fmt.Errorf("%s: %w", stmt, err))
			return
		}
		applied = append(applied, stmt)
	}
	if applied == nil {
		applied = []string{}
	}
	_ = a.recordAction(ctx, cl.Environment, cl.Name, "setup", strings.Join(applied, "\n"), "")
	writeJSON(w, http.StatusOK, map[string]any{
		"applied": applied,
		"note":    "Only CREATE EXTENSION ran. Package installs and preload changes are still in the preview.",
	})
}

func (a *api) provisionPlan(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "demo"
	}
	var plan provision.Plan
	var err error
	switch target {
	case "compose":
		plan, err = provision.Compose(provision.ComposeSpec{Name: name, Env: "dev"})
	case "cnpg":
		plan, err = provision.CNPG(provision.CNPGSpec{Name: name})
	case "ssh":
		hosts := strings.Split(r.URL.Query().Get("hosts"), ",")
		plan, err = provision.SSH(provision.SSHSpec{Name: name, Hosts: hosts})
	default:
		writeErr(w, http.StatusBadRequest, errors.New("target must be compose, cnpg or ssh"))
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	changes, err := provision.Changes(plan)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"target": target, "changes": changes})
}

func (a *api) listDeployments(w http.ResponseWriter, r *http.Request) {
	inv, err := a.inventory()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	type row struct {
		Name   string `json:"name"`
		Target string `json:"target"`
		Env    string `json:"env"`
		When   string `json:"when"`
	}
	var out []row
	for _, cl := range inv.List() {
		if cl.Patroni == nil && cl.Backup == nil {
			continue
		}
		target := cl.HAEngine
		if target == "" && cl.Backup != nil {
			target = cl.Backup.Tool
		}
		when := ""
		if cl.LastProbe != nil {
			when = cl.LastProbe.At.UTC().Format(time.RFC3339)
		}
		out = append(out, row{Name: cl.Name, Target: target, Env: cl.Environment, When: when})
	}
	if out == nil {
		out = []row{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deployments": out})
}
