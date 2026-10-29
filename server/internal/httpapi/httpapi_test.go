// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	rr := httptest.NewRecorder()
	New().ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/health", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "ok") {
		t.Fatalf("got %d %q", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing security headers")
	}
}

func TestSPAFallback(t *testing.T) {
	rr := httptest.NewRecorder()
	New().ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/fleet/prod", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "<html") {
		t.Fatalf("got %d", rr.Code)
	}
}
