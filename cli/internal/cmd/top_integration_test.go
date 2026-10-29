// SPDX-License-Identifier: Apache-2.0

//go:build integration

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/pgtest"
)

// Roadmap M1 acceptance: tl top <ref> --once lists sessions on 14-18.
func TestTopOnceAgainstPostgres(t *testing.T) {
	t.Setenv("TL_TEST_PASSWORD", pgtest.Password)
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			srv := pgtest.Start(t, major)
			path := filepath.Join(t.TempDir(), "inventory.yaml")
			for _, ref := range []string{"dev/pg", "prod/pg"} {
				if _, errOut, err := runApp(t, &app{}, "connect", "--dsn", srv.DSN, ref, "--password-env", "TL_TEST_PASSWORD", "--inventory", path); err != nil {
					t.Fatalf("connect %s: %v\n%s", ref, err, errOut)
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			sleeper, err := conn.Connect(ctx, srv.Profile, conn.SessionSettings{})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = sleeper.Close(context.Background()) }()
			done := make(chan struct{})
			go func() {
				defer close(done)
				_, _ = sleeper.Exec(ctx, "SELECT pg_sleep(60) /* tl-top-probe */")
			}()
			defer func() { cancel(); <-done }()

			var out string
			deadline := time.Now().Add(15 * time.Second)
			for {
				o, errOut, err := runApp(t, &app{}, "top", "prod/pg", "--once", "--inventory", path)
				if err != nil {
					t.Fatalf("top --once: %v\n%s", err, errOut)
				}
				if out = o; strings.Contains(out, "tl-top-probe") {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("sleeping session never listed:\n%s", out)
				}
				<-time.After(100 * time.Millisecond)
			}
			for _, want := range []string{fmt.Sprintf("tl top prod/pg · PostgreSQL %d.", major), "PID  USER", "active", "Timeout/PgSleep"} {
				if !strings.Contains(out, want) {
					t.Errorf("top lacks %q:\n%s", want, out)
				}
			}

			out, _, err = runApp(t, &app{}, "top", "dev/pg", "--once", "-o", "json", "--inventory", path)
			if err != nil {
				t.Fatal(err)
			}
			var res topJSON
			if err := json.Unmarshal([]byte(out), &res); err != nil {
				t.Fatalf("%v:\n%s", err, out)
			}
			if res.VersionNum/10000 != major || len(res.Snapshot.Sessions) == 0 || !strings.Contains(out, "tl-top-probe") {
				t.Errorf("unexpected JSON: %+v", res)
			}
		})
	}
}
