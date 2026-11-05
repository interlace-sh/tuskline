// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAlterSystemRefusesInjection(t *testing.T) {
	t.Parallel()
	if _, err := alterSystem("shared_buffers", "4GB"); err != nil {
		t.Fatal(err)
	}
	if _, err := alterSystem("shared_buffers", "4GB'; DROP TABLE t; --"); err == nil {
		t.Fatal("expected a rejected value")
	}
	if _, err := alterSystem("Not A Setting", "on"); err == nil {
		t.Fatal("expected a rejected name")
	}
}

func TestProvisionPlanRedactsSecrets(t *testing.T) {
	h := newHandler(&api{})
	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/provision/plan?target=compose&name=demo", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
	if !strings.Contains(rr.Body.String(), "compose.yaml") || !strings.Contains(rr.Body.String(), "contents not shown") {
		t.Fatalf("%s", rr.Body)
	}
	if strings.Contains(rr.Body.String(), "PGPASSWORD_SUPERUSER=") {
		t.Fatal("a generated password was included in the plan")
	}
}

func TestDeploymentsEmpty(t *testing.T) {
	h, _ := handlerWithInventory(t)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/deployments", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"deployments":[]`) {
		t.Fatalf("%d %s", rr.Code, rr.Body)
	}
}
