// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

func TestTopologyReadsPatroni(t *testing.T) {
	var posted string
	patroni := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/switchover" && r.Method == http.MethodPost {
			posted = r.URL.Path
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(`{"members":[{"name":"pg1","role":"leader","state":"running"},{"name":"pg2","role":"replica","state":"streaming","lag":0}]}`))
	}))
	t.Cleanup(patroni.Close)

	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := &inventory.Inventory{}
	if _, err := inv.PutEnvironment("dev", inventory.EnvDev); err != nil {
		t.Fatal(err)
	}
	profile, err := conn.ParseDSN("postgres://app@127.0.0.1:1/postgres?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.PutCluster(inventory.Cluster{
		Name: "orders", Environment: "dev", Profile: profile.WithPassword(""),
		Patroni: &inventory.Patroni{URL: patroni.URL},
	}); err != nil {
		t.Fatal(err)
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	storePath := filepath.Join(t.TempDir(), "meta.db")
	h := newHandler(&api{inventoryPath: path, storePath: storePath})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/clusters/dev/orders/topology", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"pg1"`) {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}

	rr = httptest.NewRecorder()
	body := strings.NewReader(`{"from":"pg1","to":"pg2","run":false}`)
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/clusters/dev/orders/switchover", body))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "Nothing has been sent") || posted != "" {
		t.Fatalf("preview %d %s posted %q", rr.Code, rr.Body, posted)
	}

	rr = httptest.NewRecorder()
	body = strings.NewReader(`{"from":"pg1","to":"pg2","run":true}`)
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/clusters/dev/orders/switchover", body))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"sent":true`) || posted != "/switchover" {
		t.Fatalf("send %d %s posted %q", rr.Code, rr.Body, posted)
	}
}

func TestLintSQL(t *testing.T) {
	h := newHandler(&api{})
	rr := httptest.NewRecorder()
	body := strings.NewReader(`{"sql":"CREATE INDEX orders_note ON orders (note);"}`)
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/lint", body))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "create-index") {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}

func TestSwitchoverOnProdNeedsTheName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := &inventory.Inventory{}
	if _, err := inv.PutEnvironment("prod", inventory.EnvProd); err != nil {
		t.Fatal(err)
	}
	profile, err := conn.ParseDSN("postgres://app@127.0.0.1:1/postgres?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.PutCluster(inventory.Cluster{
		Name: "orders", Environment: "prod", Profile: profile.WithPassword(""),
		Patroni: &inventory.Patroni{URL: "http://127.0.0.1:9"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	h := newHandler(&api{inventoryPath: path, storePath: filepath.Join(t.TempDir(), "meta.db")})
	rr := httptest.NewRecorder()
	body := strings.NewReader(`{"from":"a","to":"b","run":true}`)
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/clusters/prod/orders/switchover", body))
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "cluster name") {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}
