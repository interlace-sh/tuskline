// SPDX-License-Identifier: Apache-2.0

//go:build integration

package advisor_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/advisor"
	"github.com/interlace-sh/tuskline/core/pgtest"
)

func TestAdviseAgainstPostgres(t *testing.T) {
	for _, major := range pgtest.Versions(t) {
		t.Run("pg"+strconv.Itoa(major), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			p := pgtest.Start(t, major).Profile
			cfg, err := p.Config()
			if err != nil {
				t.Fatal(err)
			}
			db, err := pgx.ConnectConfig(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close(context.Background()) })
			version, err := serverVersion(ctx, db)
			if err != nil {
				t.Fatal(err)
			}
			current, ev, err := advisor.Collect(ctx, db, version)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := current["shared_buffers"]; !ok {
				t.Fatal("shared_buffers was not read")
			}
			plain, err := advisor.Advise(version, advisor.Hardware{}, current, ev)
			if err != nil {
				t.Fatal(err)
			}
			if !has(plain, "track_io_timing") {
				t.Fatalf("stock server did not report track_io_timing:\n%s", plain.Text("test"))
			}
			sized, err := advisor.Advise(version, advisor.Hardware{Memory: 16 * 1024 * 1024 * 1024, CPUs: 8, Storage: "ssd", Workload: "web"}, current, ev)
			if err != nil {
				t.Fatal(err)
			}
			if !has(sized, "shared_buffers") && !has(sized, "work_mem") {
				t.Fatalf("16GB advice had neither memory change:\n%s", sized.Text("test"))
			}
			for _, a := range sized.Advice {
				if a.Name == "wal_level" {
					t.Fatal("wal_level was suggested")
				}
			}
		})
	}
}

func has(r advisor.Report, name string) bool {
	for _, a := range r.Advice {
		if a.Name == name {
			return true
		}
	}
	return false
}

func serverVersion(ctx context.Context, db *pgx.Conn) (int, error) {
	var s string
	if err := db.QueryRow(ctx, `SHOW server_version_num`).Scan(&s); err != nil {
		return 0, err
	}
	return strconv.Atoi(s)
}
