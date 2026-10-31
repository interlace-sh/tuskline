// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/advisor"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// clusterAdvise previews configuration changes. It does not apply them.
func (a *api) clusterAdvise(w http.ResponseWriter, r *http.Request) {
	hw, err := adviseHardware(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	a.withCluster(w, r, func(ctx context.Context, ref inventory.Ref, db *pgx.Conn, version int) {
		current, ev, err := advisor.Collect(ctx, db, version)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		report, err := advisor.Advise(version, hw, current, ev)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Ref string `json:"ref"`
			advisor.Report
		}{Ref: ref.String(), Report: report})
	})
}

func adviseHardware(r *http.Request) (advisor.Hardware, error) {
	q := r.URL.Query()
	hw := advisor.Hardware{
		Storage:  q.Get("storage"),
		Workload: q.Get("workload"),
		DBSize:   q.Get("db_size"),
		OS:       q.Get("os"),
		Pooler:   q.Get("pooler") == "1" || q.Get("pooler") == "true",
	}
	if s := q.Get("memory"); s != "" {
		n, err := advisor.ParseMemory(s)
		if err != nil {
			return hw, err
		}
		hw.Memory = n
	}
	if s := q.Get("cpus"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return hw, fmt.Errorf("cpus %q: want a non-negative integer", s)
		}
		hw.CPUs = n
	}
	if s := q.Get("connections"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return hw, fmt.Errorf("connections %q: want a non-negative integer", s)
		}
		hw.Connections = n
	}
	return hw, nil
}
