// SPDX-License-Identifier: AGPL-3.0-only

// Package webui embeds the static SvelteKit build. "make web" builds web/
// and copies web/build into dist/. Without a build, a placeholder page is
// served so the server always compiles and runs.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// Handler serves the UI, falling back to index.html for client-side routes
// (SvelteKit adapter-static SPA mode).
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	index := "index.html"
	if _, err := fs.Stat(sub, index); err != nil {
		index = "placeholder.html"
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" || p == "." {
			p = index
		}
		if st, err := fs.Stat(sub, p); err != nil || st.IsDir() {
			http.ServeFileFS(w, r, sub, index)
			return
		}
		files.ServeHTTP(w, r)
	})
}
