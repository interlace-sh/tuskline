// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

func TestSignalPreviewDoesNotConnect(t *testing.T) {
	h, _ := handlerWithInventory(t)
	rr := httptest.NewRecorder()
	body := strings.NewReader(`{"pid":42,"run":false}`)
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/clusters/dev/orders/signal", body))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "Nothing has been sent") {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}

func TestSignalOnProdNeedsTheName(t *testing.T) {
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
	}); err != nil {
		t.Fatal(err)
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	h := newHandler(&api{inventoryPath: path, storePath: filepath.Join(t.TempDir(), "meta.db")})
	rr := httptest.NewRecorder()
	body := strings.NewReader(`{"pid":42,"run":true}`)
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/clusters/prod/orders/signal", body))
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "cluster name") {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}

func TestParseBackupInfo(t *testing.T) {
	h := newHandler(&api{})
	raw := `[{"name":"demo","backup":[{"label":"20261031-010000F","type":"full","timestamp":{"start":100,"stop":200}}],"archive":[{"min":"01","max":"02"},{"min":"03","max":"04"}]}]`
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/backups/parse", strings.NewReader(raw))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "demo") {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}

func TestMigrateUploadPreviews(t *testing.T) {
	h, _ := handlerWithInventory(t)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", "001.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("SELECT 1;\n")); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/clusters/dev/orders/migrate", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "Nothing has been applied") {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}

func TestMetricsMissingStoreIsEmpty(t *testing.T) {
	h := newHandler(&api{storePath: filepath.Join(t.TempDir(), "meta.db")})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/clusters/dev/orders/metrics", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"samples":[]`) {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}
