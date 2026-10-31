// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"reflect"
	"testing"

	"github.com/interlace-sh/tuskline/core/catalog"
)

func TestDetectProvider(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		s    ProviderSignals
		want Provider
	}{
		{"nothing", ProviderSignals{}, ProviderCommunity},
		{"aurora wins over rds", ProviderSignals{AuroraVersion: true, RDSSuperuser: true, RDSSettings: true}, ProviderAurora},
		{"rds role", ProviderSignals{RDSSuperuser: true}, ProviderRDS},
		{"rds setting", ProviderSignals{RDSSettings: true}, ProviderRDS},
		{"alloydb wins over cloud sql", ProviderSignals{AlloyDBSuperuser: true, CloudSQLSuperuser: true}, ProviderAlloyDB},
		{"alloydb setting", ProviderSignals{AlloyDBSettings: true}, ProviderAlloyDB},
		{"cloud sql role", ProviderSignals{CloudSQLSuperuser: true}, ProviderCloudSQL},
		{"cloud sql setting", ProviderSignals{CloudSQLSettings: true}, ProviderCloudSQL},
		{"azure role", ProviderSignals{AzureAdmin: true}, ProviderAzure},
		{"azure setting", ProviderSignals{AzureSettings: true}, ProviderAzure},
		{"neon setting", ProviderSignals{NeonSettings: true}, ProviderNeon},
		{"neon preload", ProviderSignals{PreloadsNeon: true}, ProviderNeon},
	}
	for _, c := range cases {
		if got := DetectProvider(c.s); got != c.want {
			t.Errorf("%s: DetectProvider() = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestSplitList(t *testing.T) {
	t.Parallel()
	for in, want := range map[string][]string{
		"":                                 nil,
		"pg_stat_statements":               {"pg_stat_statements"},
		"pg_stat_statements, auto_explain": {"pg_stat_statements", "auto_explain"},
		`"pg_stat_statements",neon ,  "pg_cron" `: {"pg_stat_statements", "neon", "pg_cron"},
	} {
		if got := splitList(in); !reflect.DeepEqual(got, want) {
			t.Errorf("splitList(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRecommendedMatchesCatalogue(t *testing.T) {
	t.Parallel()
	if string(ProviderRDS) != catalog.ProviderRDS || string(ProviderCommunity) != catalog.ProviderCommunity {
		t.Fatal("provider names drifted from core/catalog")
	}
	looked := catalog.LookedFor()
	if len(Recommended) != len(looked) {
		t.Fatalf("recommended %d, catalogue %d", len(Recommended), len(looked))
	}
	for i := range looked {
		if Recommended[i].Name != looked[i].Name || Recommended[i].Create != looked[i].Create {
			t.Fatalf("recommended[%d] = %+v, catalogue %+v", i, Recommended[i], looked[i])
		}
	}
}

func TestMissingExtensions(t *testing.T) {
	t.Parallel()
	names := func(ms []Missing) []string {
		var out []string
		for _, m := range ms {
			out = append(out, m.Name)
		}
		return out
	}
	all := []string{"pg_stat_statements", "auto_explain", "pg_wait_sampling", "pg_stat_kcache", "hypopg"}

	bare := ServerInfo{PreloadKnown: true}
	if got := names(bare.MissingExtensions()); !reflect.DeepEqual(got, all) {
		t.Fatalf("bare server: missing %v, want %v", got, all)
	}

	ready := ServerInfo{
		PreloadKnown:           true,
		SharedPreloadLibraries: []string{"pg_stat_statements"},
		Extensions: []Extension{
			{Name: "pg_stat_statements", InstalledVersion: "1.11", Available: true},
			{Name: "hypopg", InstalledVersion: "1.4.1", Available: true},
			{Name: "pg_wait_sampling", Available: true},
		},
	}
	got := ready.MissingExtensions()
	if want := []string{"auto_explain", "pg_wait_sampling", "pg_stat_kcache"}; !reflect.DeepEqual(names(got), want) {
		t.Fatalf("ready server: missing %v, want %v", names(got), want)
	}
	if !got[0].Available || !got[0].NeedsPreload || got[0].Create || !got[1].Available || !got[1].NeedsPreload || got[2].Available {
		t.Fatalf("availability or preload flags wrong: %+v", got)
	}

	installedNotPreloaded := ServerInfo{
		PreloadKnown: true,
		Extensions:   []Extension{{Name: "pg_stat_statements", InstalledVersion: "1.11", Available: true}},
	}
	m := installedNotPreloaded.MissingExtensions()[0]
	if m.Name != "pg_stat_statements" || !m.Installed || !m.NeedsPreload {
		t.Fatalf("installed but not preloaded: got %+v", m)
	}

	unknownPreload := ServerInfo{
		Extensions: []Extension{{Name: "pg_stat_statements", InstalledVersion: "1.11", Available: true}},
	}
	if m := unknownPreload.MissingExtensions()[0]; m.Name != "pg_stat_statements" || !m.NeedsPreload {
		t.Fatalf("unreadable preload list must count as needing a preload: got %+v", m)
	}
}
