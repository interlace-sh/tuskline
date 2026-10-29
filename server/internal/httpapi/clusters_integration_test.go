// SPDX-License-Identifier: AGPL-3.0-only

//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/pgtest"
	"github.com/interlace-sh/tuskline/core/snapshot"
	"github.com/interlace-sh/tuskline/core/store"
)

func TestClusterAPIAgainstPostgres(t *testing.T) {
	t.Setenv("TL_TEST_PASSWORD", pgtest.Password)
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			srv := pgtest.Start(t, major)
			pgtest.Exec(t, srv.Profile, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements", "SELECT 1")
			path := filepath.Join(t.TempDir(), "inventory.yaml")
			inv := &inventory.Inventory{}
			if _, err := inv.PutEnvironment("dev", inventory.EnvDev); err != nil {
				t.Fatal(err)
			}
			if _, err := inv.PutEnvironment("prod", inventory.EnvProd); err != nil {
				t.Fatal(err)
			}
			for _, env := range []string{"dev", "prod"} {
				if err := inv.PutCluster(inventory.Cluster{
					Name: "pg", Environment: env, Profile: srv.Profile.WithPassword(""),
					Password: &inventory.SecretRef{Env: "TL_TEST_PASSWORD"},
				}); err != nil {
					t.Fatal(err)
				}
			}
			pgtest.Exec(t, srv.Profile, "CREATE TABLE tl_editor (n int)", "INSERT INTO tl_editor VALUES (1)")
			if err := inv.Save(path); err != nil {
				t.Fatal(err)
			}
			storePath := filepath.Join(t.TempDir(), "meta.db")
			h := newHandler(&api{inventoryPath: path, storePath: storePath, previous: map[string]snapshot.Snapshot{}})
			get := func(url string) *httptest.ResponseRecorder {
				t.Helper()
				rr := httptest.NewRecorder()
				h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, url, nil))
				if strings.Contains(rr.Body.String(), pgtest.Password) {
					t.Fatalf("%s leaked the password", url)
				}
				return rr
			}
			first := get("/api/v1/clusters/dev/pg/queries")
			if first.Code != http.StatusOK {
				t.Fatalf("queries: %d %s", first.Code, first.Body)
			}
			var once struct {
				Deltas     []snapshot.Delta `json:"deltas"`
				Statements []struct {
					Query string `json:"query"`
				} `json:"statements"`
			}
			if err := json.Unmarshal(first.Body.Bytes(), &once); err != nil {
				t.Fatal(err)
			}
			if once.Deltas != nil || len(once.Statements) == 0 {
				t.Fatalf("first call: %+v", once)
			}
			second := get("/api/v1/clusters/dev/pg/queries")
			if err := json.Unmarshal(second.Body.Bytes(), &once); err != nil {
				t.Fatal(err)
			}
			if once.Deltas == nil {
				t.Fatal("second call has no deltas")
			}
			if rr := get("/api/v1/clusters/dev/pg/activity"); rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"sessions"`) {
				t.Fatalf("activity: %d %s", rr.Code, rr.Body)
			}
			if rr := get("/api/v1/clusters/dev/pg/health"); rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"findings"`) {
				t.Fatalf("health: %d %s", rr.Code, rr.Body)
			}
			post := func(url, sql string) *httptest.ResponseRecorder {
				t.Helper()
				payload, err := json.Marshal(map[string]string{"sql": sql})
				if err != nil {
					t.Fatal(err)
				}
				rr := httptest.NewRecorder()
				req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(payload))
				h.ServeHTTP(rr, req)
				if strings.Contains(rr.Body.String(), pgtest.Password) {
					t.Fatalf("%s leaked the password", url)
				}
				return rr
			}
			selected := post("/api/v1/clusters/dev/pg/query", "SELECT n FROM tl_editor")
			if selected.Code != http.StatusOK || !strings.Contains(selected.Body.String(), `"rows":[[1]]`) {
				t.Fatalf("select: %d %s", selected.Code, selected.Body)
			}
			if stacked := post("/api/v1/clusters/dev/pg/query", "SELECT 1; SELECT 2"); stacked.Code != http.StatusBadRequest {
				t.Fatalf("stacked statement: %d %s", stacked.Code, stacked.Body)
			}
			updated := post("/api/v1/clusters/prod/pg/query", "UPDATE tl_editor SET n = 2")
			if updated.Code == http.StatusOK {
				t.Fatalf("prod write succeeded: %s", updated.Body)
			}
			again := post("/api/v1/clusters/dev/pg/query", "SELECT n FROM tl_editor")
			if !strings.Contains(again.Body.String(), `"rows":[[1]]`) {
				t.Fatalf("prod write changed the row: %s", again.Body)
			}
			meta, err := store.Open(context.Background(), storePath)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = meta.Close() })
			count, err := meta.QueryAuditCount(context.Background())
			if err != nil || count != 1 {
				t.Fatalf("prod audit count %d, %v", count, err)
			}
		})
	}
}
