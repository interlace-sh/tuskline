// SPDX-License-Identifier: AGPL-3.0-only

// Package httpapi is the server's HTTP surface: the REST API under /api and
// the embedded web UI everywhere else. The desktop app mounts the same
// handler, so the UI only ever talks HTTP to the core.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/interlace-sh/tuskline/core/version"
	"github.com/interlace-sh/tuskline/server/internal/webui"
)

// New returns the root handler.
func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": version.Version, "commit": version.Commit})
	})
	mux.Handle("/", webui.Handler())
	return securityHeaders(mux)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
