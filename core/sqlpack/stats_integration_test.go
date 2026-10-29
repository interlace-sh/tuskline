// SPDX-License-Identifier: Apache-2.0

//go:build integration

package sqlpack_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/pgtest"
	"github.com/interlace-sh/tuskline/core/sqlpack"
)

func TestStatsQueries(t *testing.T) {
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			t.Parallel()
			p := pgtest.Start(t, major).Profile
			pgtest.Exec(t, p,
				"CREATE EXTENSION IF NOT EXISTS pg_stat_statements",
				"CREATE TABLE t (id int)",
				"INSERT INTO t VALUES (1)",
				"SELECT count(*) FROM t",
			)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cfg, err := p.Config()
			if err != nil {
				t.Fatal(err)
			}
			c, err := pgx.ConnectConfig(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close(context.Background()) }()
			vn, err := conn.ServerVersionNum(c)
			if err != nil {
				t.Fatal(err)
			}

			must := func(q sqlpack.Query, minRows int, args ...any) {
				t.Helper()
				rows, err := c.Query(ctx, q.SQL, args...)
				if err != nil {
					t.Fatalf("%s: %v", q.Name, err)
				}
				n := 0
				for rows.Next() {
					if _, err := rows.Values(); err != nil {
						t.Fatalf("%s: %v", q.Name, err)
					}
					n++
				}
				if err := rows.Err(); err != nil {
					t.Fatalf("%s: %v", q.Name, err)
				}
				if n < minRows {
					t.Errorf("%s: %d rows, want at least %d", q.Name, n, minRows)
				}
			}
			one := func(variants ...sqlpack.Query) sqlpack.Query {
				t.Helper()
				q, err := sqlpack.Pick(vn, variants...)
				if err != nil {
					t.Fatal(err)
				}
				return q
			}

			must(sqlpack.StatsDatabase, 1)
			must(one(sqlpack.StatsTables14, sqlpack.StatsTables16, sqlpack.StatsTables18), 1, 100)
			must(one(sqlpack.StatsWAL14, sqlpack.StatsWAL18), 1)
			must(one(sqlpack.StatsBgwriter14, sqlpack.StatsBgwriter17), 1)
			must(one(sqlpack.StatsStatements14, sqlpack.StatsStatements15, sqlpack.StatsStatements17), 1, 200, 50)

			io, err := sqlpack.Pick(vn, sqlpack.StatsIO16, sqlpack.StatsIO18)
			cp, cpErr := sqlpack.Pick(vn, sqlpack.StatsCheckpointer17, sqlpack.StatsCheckpointer18)
			if major < 16 {
				if err == nil || cpErr == nil {
					t.Fatalf("pg%d should have no io (%v) or checkpointer (%v) query", major, err, cpErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			must(io, 1)
			if major < 17 {
				if cpErr == nil {
					t.Fatal("checkpointer query exists before PostgreSQL 17")
				}
				return
			}
			if cpErr != nil {
				t.Fatal(cpErr)
			}
			must(cp, 1)
		})
	}
}
