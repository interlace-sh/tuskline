// SPDX-License-Identifier: Apache-2.0

//go:build integration

package conn_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/conn"
)

// Run with: moon run core:integration (needs Docker).
// PG_VERSIONS can narrow the matrix, e.g. PG_VERSIONS=17.
func TestProbeAcrossVersions(t *testing.T) {
	for _, major := range pgVersions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			ctx := context.Background()
			super := startPG(t, major)

			t.Run("superuser", func(t *testing.T) {
				execAll(t, super, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
				info, err := conn.Probe(ctx, super)
				if err != nil {
					t.Fatal(err)
				}
				if info.Major() != major {
					t.Fatalf("major = %d, want %d", info.Major(), major)
				}
				if info.Provider != conn.ProviderCommunity {
					t.Errorf("provider = %s, want community", info.Provider)
				}
				if info.ReadOnly || info.InRecovery {
					t.Errorf("read_only=%v in_recovery=%v, want a writable primary", info.ReadOnly, info.InRecovery)
				}
				if info.Database != "tuskline" || info.LoginRole != testUser {
					t.Errorf("database=%q login_role=%q", info.Database, info.LoginRole)
				}
				if !info.Role.Superuser || !info.Role.CanMonitor() {
					t.Errorf("role = %+v, want superuser", info.Role)
				}
				if !info.PreloadKnown || !info.Preloaded("pg_stat_statements") {
					t.Errorf("preload known=%v libs=%v, want pg_stat_statements", info.PreloadKnown, info.SharedPreloadLibraries)
				}
				pss, ok := info.Extension("pg_stat_statements")
				if !ok || !pss.Installed() || !pss.Available {
					t.Errorf("pg_stat_statements = %+v, want installed and available", pss)
				}
				if e, ok := info.Extension("plpgsql"); !ok || !e.Installed() {
					t.Errorf("plpgsql = %+v, want installed", e)
				}
				for _, m := range info.MissingExtensions() {
					if m.Name == "pg_stat_statements" {
						t.Errorf("pg_stat_statements reported missing: %+v", m)
					}
				}
			})

			t.Run("monitoring role", func(t *testing.T) {
				execAll(t, super,
					"CREATE ROLE tuskline_monitor LOGIN PASSWORD 'monitor-test-only'",
					"GRANT pg_monitor TO tuskline_monitor",
				)
				info, err := conn.Probe(ctx, as(super, "tuskline_monitor", "monitor-test-only"))
				if err != nil {
					t.Fatal(err)
				}
				r := info.Role
				if r.Superuser || !r.PGMonitor || !r.PGReadAllSettings || !r.PGReadAllStats || !r.CanMonitor() {
					t.Errorf("role = %+v, want pg_monitor and its members, not superuser", r)
				}
				if r.PGSignalBackend {
					t.Error("pg_monitor must not imply pg_signal_backend")
				}
				if !info.PreloadKnown || !info.Preloaded("pg_stat_statements") {
					t.Errorf("pg_monitor should read shared_preload_libraries: known=%v libs=%v", info.PreloadKnown, info.SharedPreloadLibraries)
				}
			})

			t.Run("plain role", func(t *testing.T) {
				execAll(t, super, "CREATE ROLE plain_login LOGIN PASSWORD 'plain-test-only'")
				info, err := conn.Probe(ctx, as(super, "plain_login", "plain-test-only"))
				if err != nil {
					t.Fatal(err)
				}
				if info.Role.CanMonitor() || info.Role.PGReadAllSettings {
					t.Errorf("role = %+v, want no monitoring privileges", info.Role)
				}
				if info.PreloadKnown {
					t.Errorf("a plain role cannot read shared_preload_libraries, got %v", info.SharedPreloadLibraries)
				}
				if len(info.Extensions) == 0 {
					t.Error("a plain role should still list extensions")
				}
			})

			t.Run("provider fingerprints", func(t *testing.T) {
				steps := []struct {
					setup []string
					want  conn.Provider
				}{
					{[]string{"CREATE ROLE azure_pg_admin NOLOGIN"}, conn.ProviderAzure},
					{[]string{"CREATE ROLE cloudsqlsuperuser NOLOGIN"}, conn.ProviderCloudSQL},
					{[]string{"CREATE ROLE alloydbsuperuser NOLOGIN"}, conn.ProviderAlloyDB},
					{[]string{"CREATE ROLE rds_superuser NOLOGIN"}, conn.ProviderRDS},
					{[]string{"CREATE FUNCTION aurora_version() RETURNS text LANGUAGE sql AS $$SELECT '16.4.0'$$"}, conn.ProviderAurora},
				}
				for _, s := range steps {
					execAll(t, super, s.setup...)
					info, err := conn.Probe(ctx, super)
					if err != nil {
						t.Fatal(err)
					}
					if info.Provider != s.want {
						t.Errorf("after %q: provider = %s, want %s", s.setup, info.Provider, s.want)
					}
				}
			})

			t.Run("wrong password is not echoed", func(t *testing.T) {
				const bad = "definitely-wrong-pw-123"
				_, err := conn.Probe(ctx, super.WithPassword(bad))
				if err == nil {
					t.Fatal("want an authentication error")
				}
				if strings.Contains(err.Error(), bad) || strings.Contains(err.Error(), testPassword) {
					t.Fatalf("error leaks a password: %v", err)
				}
			})
		})
	}
}
