// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/catalog"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

type setupResponse struct {
	Ref      string          `json:"ref"`
	Version  string          `json:"version"`
	Provider string          `json:"provider"`
	Missing  []string        `json:"missing"`
	Plan     []string        `json:"plan"`
	Probe    conn.ServerInfo `json:"probe"`
}

// clusterSetup reports looked-for extensions and the install preview.
// The preview is not applied.
func (a *api) clusterSetup(w http.ResponseWriter, r *http.Request) {
	a.withCluster(w, r, func(ctx context.Context, ref inventory.Ref, db *pgx.Conn, _ int) {
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
		var missing []string
		for _, ext := range catalog.LookedFor() {
			if !installed[ext.Name] {
				missing = append(missing, ext.Name)
			}
		}
		if missing == nil {
			missing = []string{}
		}
		lines, err := catalog.InstallPlan(catalog.Request{
			Major: info.VersionNum / 10000, Provider: string(info.Provider),
			PreloadKnown: info.PreloadKnown, CurrentPreload: info.SharedPreloadLibraries,
			Want: missing, Installed: installed,
		})
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		plan := make([]string, 0, len(lines))
		for _, line := range lines {
			plan = append(plan, line.Text)
		}
		if info.Extensions == nil {
			info.Extensions = []conn.Extension{}
		}
		writeJSON(w, http.StatusOK, setupResponse{
			Ref: ref.String(), Version: info.Version, Provider: string(info.Provider),
			Missing: missing, Plan: plan, Probe: info,
		})
	})
}
