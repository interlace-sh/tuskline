// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

func handlerWithInventory(t *testing.T) (http.Handler, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := &inventory.Inventory{}
	if _, err := inv.PutEnvironment("dev", inventory.EnvDev); err != nil {
		t.Fatal(err)
	}
	p, err := conn.ParseDSN("postgres://app@db.example.com:5432/orders")
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.PutCluster(inventory.Cluster{
		Name: "orders", Environment: "dev", Provider: "rds", Profile: p.WithPassword(""),
		Password: &inventory.SecretRef{Env: "ORDERS_PASSWORD"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	return newHandler(&api{inventoryPath: path}), path
}

func TestListClustersOmitsSecrets(t *testing.T) {
	h, _ := handlerWithInventory(t)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/clusters", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
	body := rr.Body.String()
	if strings.Contains(body, "ORDERS_PASSWORD") || strings.Contains(body, "password") {
		t.Fatalf("secret in response: %s", body)
	}
	var out struct {
		Clusters []clusterView `json:"clusters"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Clusters) != 1 || out.Clusters[0].ID != "dev/orders" || out.Clusters[0].Host != "db.example.com" || out.Clusters[0].Kind != "dev" {
		t.Fatalf("%+v", out.Clusters)
	}
}

func TestClusterRoutesRejectUnknown(t *testing.T) {
	h, _ := handlerWithInventory(t)
	for _, path := range []string{
		"/api/v1/clusters/dev/missing/queries",
		"/api/v1/clusters/dev/missing/activity",
		"/api/v1/clusters/dev/missing/health",
	} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("%s: %d %s", path, rr.Code, rr.Body)
		}
	}
}

func TestEmptyInventory(t *testing.T) {
	h := newHandler(&api{inventoryPath: filepath.Join(t.TempDir(), "none.yaml")})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/clusters", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"clusters": []`) && !strings.Contains(rr.Body.String(), `"clusters":[]`) {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}
