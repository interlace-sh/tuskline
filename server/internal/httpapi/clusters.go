// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/activity"
	"github.com/interlace-sh/tuskline/core/analyser"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/snapshot"
)

// clusterCallTimeout bounds one clusters request. Prod sessions also set a
// statement timeout; this stops a dev session with no timeout from hanging
// the handler.
const clusterCallTimeout = 30 * time.Second

// api serves the cluster routes. previous remembers the last statement
// sample per cluster so the next /queries response can carry deltas. It
// lives in memory only: query text is not written to the metadata store.
// storePath is the SQLite file for the production query audit. Empty uses
// store.DefaultPath.
type api struct {
	inventoryPath string
	storePath     string
	mu            sync.Mutex
	previous      map[string]snapshot.Snapshot
}

func (a *api) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/clusters", a.listClusters)
	mux.HandleFunc("POST /api/v1/clusters", a.saveCluster)
	mux.HandleFunc("GET /api/v1/clusters/{env}/{name}/queries", a.clusterQueries)
	mux.HandleFunc("GET /api/v1/clusters/{env}/{name}/activity", a.clusterActivity)
	mux.HandleFunc("GET /api/v1/clusters/{env}/{name}/health", a.clusterHealth)
	mux.HandleFunc("GET /api/v1/clusters/{env}/{name}/probe", a.clusterProbe)
	mux.HandleFunc("POST /api/v1/clusters/{env}/{name}/query", a.clusterQuery)
}

type clusterView struct {
	ID          string                 `json:"id"`
	Environment string                 `json:"environment"`
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Provider    string                 `json:"provider,omitempty"`
	Host        string                 `json:"host,omitempty"`
	Port        uint16                 `json:"port,omitempty"`
	Database    string                 `json:"database,omitempty"`
	User        string                 `json:"user,omitempty"`
	TLS         string                 `json:"tls,omitempty"`
	LastProbe   *inventory.ProbeRecord `json:"last_probe,omitempty"`
}

func (a *api) listClusters(w http.ResponseWriter, r *http.Request) {
	inv, err := a.inventory()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]clusterView, 0, len(inv.List()))
	for _, cl := range inv.List() {
		env, _ := inv.Environment(cl.Environment)
		p := cl.Profile
		out = append(out, clusterView{
			ID: cl.Ref().String(), Environment: cl.Environment, Name: cl.Name, Kind: string(env.Kind),
			Provider: cl.Provider, Host: p.Host, Port: p.EffectivePort(), Database: p.Database,
			User: p.User, TLS: string(p.TLSMode()), LastProbe: cl.LastProbe,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"clusters": out})
}

func (a *api) clusterQueries(w http.ResponseWriter, r *http.Request) {
	a.withCluster(w, r, func(ctx context.Context, ref inventory.Ref, db *pgx.Conn, version int) {
		snap, err := snapshot.Capture(ctx, db, version, snapshotMaxText, snapshotLimit)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		a.mu.Lock()
		prev, ok := a.previous[ref.String()]
		a.previous[ref.String()] = snap
		a.mu.Unlock()
		var deltas []snapshot.Delta
		if ok {
			deltas = snapshot.Diff(prev, snap)
		}
		writeJSON(w, http.StatusOK, struct {
			Ref        string               `json:"ref"`
			VersionNum int                  `json:"server_version_num"`
			Statements []snapshot.Statement `json:"statements"`
			Deltas     []snapshot.Delta     `json:"deltas"`
		}{ref.String(), version, snap.Statements, deltas})
	})
}

func (a *api) clusterActivity(w http.ResponseWriter, r *http.Request) {
	a.withCluster(w, r, func(ctx context.Context, ref inventory.Ref, db *pgx.Conn, version int) {
		snap, err := activity.Take(ctx, db, version)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Ref string `json:"ref"`
			activity.Snapshot
			Blocking []*activity.Node `json:"blocking"`
		}{Ref: ref.String(), Snapshot: snap, Blocking: activity.BlockingTree(snap.Sessions)})
	})
}

func (a *api) clusterProbe(w http.ResponseWriter, r *http.Request) {
	a.withCluster(w, r, func(ctx context.Context, ref inventory.Ref, db *pgx.Conn, _ int) {
		info, err := conn.Inspect(ctx, db)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		if info.Extensions == nil {
			info.Extensions = []conn.Extension{}
		}
		if info.SharedPreloadLibraries == nil {
			info.SharedPreloadLibraries = []string{}
		}
		writeJSON(w, http.StatusOK, struct {
			Ref string `json:"ref"`
			conn.ServerInfo
		}{Ref: ref.String(), ServerInfo: info})
	})
}

func (a *api) clusterHealth(w http.ResponseWriter, r *http.Request) {
	a.withCluster(w, r, func(ctx context.Context, ref inventory.Ref, db *pgx.Conn, version int) {
		report, err := analyser.Diagnose(ctx, db, version, time.Minute)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Ref   string    `json:"ref"`
			Taken time.Time `json:"taken"`
			analyser.Report
		}{Ref: ref.String(), Taken: time.Now().UTC(), Report: report})
	})
}

const (
	snapshotMaxText = 500
	snapshotLimit   = 100
)

func (a *api) withCluster(w http.ResponseWriter, r *http.Request, fn func(context.Context, inventory.Ref, *pgx.Conn, int)) {
	ref, err := inventory.ParseRef(r.PathValue("env") + "/" + r.PathValue("name"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), clusterCallTimeout)
	defer cancel()
	db, version, err := a.open(ctx, ref)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errNotInInventory) {
			status = http.StatusNotFound
		}
		writeErr(w, status, err)
		return
	}
	defer func() { _ = db.Close(context.Background()) }()
	fn(ctx, ref, db, version)
}

var errNotInInventory = errors.New("not in the inventory")

func (a *api) inventoryFile() (string, error) {
	if a.inventoryPath != "" {
		return a.inventoryPath, nil
	}
	return inventory.DefaultPath()
}

func (a *api) inventory() (*inventory.Inventory, error) {
	path, err := a.inventoryFile()
	if err != nil {
		return nil, err
	}
	return inventory.Load(path)
}

func (a *api) open(ctx context.Context, ref inventory.Ref) (*pgx.Conn, int, error) {
	inv, err := a.inventory()
	if err != nil {
		return nil, 0, err
	}
	cl, ok := inv.Cluster(ref)
	if !ok {
		return nil, 0, fmt.Errorf("%s: %w", ref, errNotInInventory)
	}
	env, ok := inv.Environment(ref.Env)
	if !ok {
		return nil, 0, fmt.Errorf("%s: environment is missing", ref)
	}
	p, err := resolvePassword(cl.Profile, cl.Password, conn.PassfilePath())
	if err != nil {
		return nil, 0, err
	}
	p, err = conn.ResolveService(conn.ServicefilePaths(), p)
	if err != nil {
		return nil, 0, err
	}
	db, err := conn.Connect(ctx, p, env.Policy.Session())
	if err != nil {
		return nil, 0, err
	}
	version, err := conn.ServerVersionNum(db)
	if err != nil {
		_ = db.Close(context.Background())
		return nil, 0, err
	}
	return db, version, nil
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
