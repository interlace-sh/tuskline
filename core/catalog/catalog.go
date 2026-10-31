// SPDX-License-Identifier: Apache-2.0

package catalog

import "fmt"

// Provider names match core/conn. Community is a self-managed server.
const (
	ProviderCommunity = "community"
	ProviderRDS       = "rds"
	ProviderAurora    = "aurora"
	ProviderCloudSQL  = "cloudsql"
	ProviderAlloyDB   = "alloydb"
	ProviderAzure     = "azure"
	ProviderNeon      = "neon"
)

// managed is every provider that publishes an extension allow-list.
var managed = []string{
	ProviderRDS, ProviderAurora, ProviderCloudSQL, ProviderAlloyDB, ProviderAzure, ProviderNeon,
}

// Setting is one postgresql.conf parameter the extension wants.
type Setting struct {
	Name  string
	Value string
}

// Extension is one catalogue entry. PackageSuffix is empty when the files
// ship in the postgresql-{major} server package (Contrib).
type Extension struct {
	Name          string
	Unlocks       string
	Required      bool
	LookedFor     bool
	Preload       bool
	Restart       bool
	Create        bool
	Contrib       bool
	PackageSuffix string
	// Repo is "pigsty" when PGDG does not ship the package. Empty means PGDG, or the server package when Contrib is set.
	Repo     string
	MinMajor int
	MaxMajor int
	// Providers lists managed services that allow the extension.
	// Self-managed servers are not listed: Offered treats them separately.
	Providers []string
	Depends   []string
	Settings  []Setting
}

// All returns the catalogue, most important first. The five extensions
// Tuskline looks for come before the ordinary single-node ones.
func All() []Extension {
	out := make([]Extension, len(entries))
	copy(out, entries)
	return out
}

// LookedFor returns the extensions a probe reports when they are not ready.
func LookedFor() []Extension {
	var out []Extension
	for _, e := range entries {
		if e.LookedFor {
			out = append(out, e)
		}
	}
	return out
}

// ByName returns the entry for the SQL or library name.
func ByName(name string) (Extension, bool) {
	for _, e := range entries {
		if e.Name == name {
			return e, true
		}
	}
	return Extension{}, false
}

// Offered reports whether this server can turn the extension on. A zero
// major skips the version check. Self-managed servers can use every entry.
func (e Extension) Offered(provider string, major int) bool {
	if major != 0 && (major < e.MinMajor || major > e.MaxMajor) {
		return false
	}
	if provider == "" || provider == ProviderCommunity {
		return true
	}
	for _, p := range e.Providers {
		if p == provider {
			return true
		}
	}
	return false
}

// PGDGPackage is the Debian package name for major, such as
// postgresql-16-hypopg. Contrib entries use the server package.
func (e Extension) PGDGPackage(major int) string {
	if major < 1 {
		return ""
	}
	if e.Contrib {
		return fmt.Sprintf("postgresql-%d", major)
	}
	return fmt.Sprintf("postgresql-%d-%s", major, e.PackageSuffix)
}

// tier1 is the managed services the onboarding guide treats as offering
// pg_stat_statements.
var tier1 = managed

var entries = []Extension{
	{
		Name: "pg_stat_statements", Unlocks: "query insights, load attribution and workload snapshots",
		Required: true, LookedFor: true, Preload: true, Restart: true, Create: true, Contrib: true,
		MinMajor: 14, MaxMajor: 18, Providers: tier1,
		Settings: []Setting{{Name: "pg_stat_statements.track", Value: "top"}, {Name: "track_io_timing", Value: "on"}},
	},
	{
		Name: "auto_explain", Unlocks: "captured plans for slow queries",
		LookedFor: true, Preload: true, Restart: true, Contrib: true,
		MinMajor: 14, MaxMajor: 18, Providers: tier1,
		Settings: []Setting{{Name: "auto_explain.log_min_duration", Value: "500ms"}, {Name: "auto_explain.log_format", Value: "json"}},
	},
	{
		Name: "pg_wait_sampling", Unlocks: "wait-event history between samples",
		LookedFor: true, Preload: true, Restart: true, Create: true, PackageSuffix: "pg-wait-sampling",
		MinMajor: 14, MaxMajor: 18, Providers: []string{ProviderCloudSQL},
	},
	{
		Name: "pg_stat_kcache", Unlocks: "CPU and real disk I/O per query",
		LookedFor: true, Preload: true, Restart: true, Create: true, PackageSuffix: "pg-stat-kcache",
		MinMajor: 14, MaxMajor: 18, Depends: []string{"pg_stat_statements"},
	},
	{
		Name: "hypopg", Unlocks: "index advice validated with hypothetical indexes",
		LookedFor: true, Create: true, PackageSuffix: "hypopg",
		MinMajor: 14, MaxMajor: 18,
		Providers: []string{ProviderRDS, ProviderAurora, ProviderCloudSQL, ProviderAzure, ProviderNeon},
	},
	{
		Name: "vector", Unlocks: "pgvector similarity search",
		Create: true, PackageSuffix: "pgvector",
		MinMajor: 14, MaxMajor: 18, Providers: tier1,
	},
	{
		Name: "postgis", Unlocks: "geographic objects and spatial indexes",
		Create: true, PackageSuffix: "postgis-3",
		MinMajor: 14, MaxMajor: 18, Providers: tier1,
	},
	{
		Name: "pgrouting", Unlocks: "routing functions on top of PostGIS",
		Create: true, PackageSuffix: "pgrouting",
		MinMajor: 14, MaxMajor: 18, Depends: []string{"postgis"},
		Providers: []string{ProviderRDS, ProviderAurora, ProviderCloudSQL, ProviderAzure},
	},
	{
		Name: "pg_cron", Unlocks: "scheduled jobs inside the database",
		Preload: true, Restart: true, Create: true, PackageSuffix: "cron",
		MinMajor: 14, MaxMajor: 18, Providers: tier1,
	},
	{
		Name: "pgaudit", Unlocks: "session and object audit logging",
		Preload: true, Restart: true, Create: true, PackageSuffix: "pgaudit",
		MinMajor: 14, MaxMajor: 18, Providers: tier1,
	},
	{
		Name: "pg_partman", Unlocks: "partition creation and maintenance",
		Create: true, PackageSuffix: "partman",
		MinMajor: 14, MaxMajor: 18,
		Providers: []string{ProviderRDS, ProviderAurora, ProviderCloudSQL, ProviderAzure, ProviderNeon},
	},
	{
		Name: "pg_repack", Unlocks: "repack tables online",
		Create: true, PackageSuffix: "repack",
		MinMajor: 14, MaxMajor: 18,
		Providers: []string{ProviderRDS, ProviderAurora, ProviderCloudSQL, ProviderAzure},
	},
}
