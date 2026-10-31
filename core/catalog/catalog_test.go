// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"strings"
	"testing"
)

func TestCatalogueShape(t *testing.T) {
	t.Parallel()
	want := []string{
		"pg_stat_statements", "auto_explain", "pg_wait_sampling", "pg_stat_kcache", "hypopg",
		"vector", "postgis", "pgrouting", "pg_cron", "pgaudit", "pg_partman", "pg_repack",
	}
	all := All()
	if len(all) != len(want) {
		t.Fatalf("catalogue has %d entries, want %d", len(all), len(want))
	}
	for i, name := range want {
		e := all[i]
		if e.Name != name {
			t.Fatalf("entry %d = %s, want %s", i, e.Name, name)
		}
		if e.Unlocks == "" || e.MinMajor != 14 || e.MaxMajor != 18 {
			t.Fatalf("%s: unlocks %q majors %d–%d", e.Name, e.Unlocks, e.MinMajor, e.MaxMajor)
		}
		if e.Contrib == (e.PackageSuffix != "") {
			t.Fatalf("%s: contrib %v suffix %q", e.Name, e.Contrib, e.PackageSuffix)
		}
		if !e.Offered(ProviderCommunity, 16) {
			t.Fatalf("%s is not offered self-managed", e.Name)
		}
	}
	looked := LookedFor()
	if len(looked) != 5 || !looked[0].Required || looked[0].Name != "pg_stat_statements" {
		t.Fatalf("looked-for: %+v", looked)
	}
	kcache, ok := ByName("pg_stat_kcache")
	if !ok || kcache.PGDGPackage(16) != "postgresql-16-pg-stat-kcache" {
		t.Fatalf("package: %+v", kcache)
	}
	statements, _ := ByName("pg_stat_statements")
	if kcache.Offered(ProviderRDS, 16) || !statements.Offered(ProviderRDS, 16) {
		t.Fatal("RDS allow-list")
	}
	waits, _ := ByName("pg_wait_sampling")
	if !waits.Offered(ProviderCloudSQL, 17) {
		t.Fatal("Cloud SQL should allow pg_wait_sampling")
	}
	vector, _ := ByName("vector")
	if vector.Offered(ProviderCommunity, 13) {
		t.Fatal("PostgreSQL 13 is outside the catalogue")
	}
}

func TestMergePreloadKeepsTheCurrentList(t *testing.T) {
	t.Parallel()
	got := MergePreload([]string{"neon", "pg_cron"}, []string{"pg_stat_statements", "pg_cron", "auto_explain"})
	want := "neon,pg_cron,pg_stat_statements,auto_explain"
	if strings.Join(got, ",") != want {
		t.Fatalf("merge = %s, want %s", strings.Join(got, ","), want)
	}
}

func TestStepsMergePreloadAndPatroni(t *testing.T) {
	t.Parallel()
	steps, err := Steps(Request{
		Major: 16, Provider: ProviderCommunity, PreloadKnown: true,
		CurrentPreload: []string{"neon"},
		Want:           []string{"pg_stat_statements", "auto_explain", "hypopg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := join(steps)
	if strings.Contains(text, "patronictl") {
		t.Fatalf("self-managed steps used patronictl:\n%s", text)
	}
	if !strings.Contains(text, "ALTER SYSTEM SET shared_preload_libraries = 'neon,pg_stat_statements,auto_explain';") {
		t.Fatalf("preload was replaced or dropped:\n%s", text)
	}
	if strings.Contains(text, "CREATE EXTENSION IF NOT EXISTS auto_explain") {
		t.Fatalf("auto_explain is not CREATE EXTENSION:\n%s", text)
	}
	if !strings.Contains(text, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;") {
		t.Fatalf("missing create:\n%s", text)
	}
	if !strings.Contains(text, "-- package postgresql-16-hypopg (PGDG)") {
		t.Fatalf("missing package:\n%s", text)
	}
	if !strings.Contains(text, "ALTER SYSTEM SET pg_stat_statements.track = 'top';") {
		t.Fatalf("missing setting:\n%s", text)
	}

	patroni, err := Steps(Request{
		Major: 16, Provider: ProviderCommunity, Patroni: true, PreloadKnown: true,
		CurrentPreload: []string{"neon"},
		Want:           []string{"pg_stat_statements"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ptext := join(patroni)
	if strings.Contains(ptext, "ALTER SYSTEM") {
		t.Fatalf("Patroni steps used ALTER SYSTEM:\n%s", ptext)
	}
	if !strings.Contains(ptext, "patronictl edit-config") || !strings.Contains(ptext, "shared_preload_libraries: 'neon,pg_stat_statements'") {
		t.Fatalf("patronictl block:\n%s", ptext)
	}
}

func TestStepsDoNotReplaceAnUnknownPreloadList(t *testing.T) {
	t.Parallel()
	steps, err := Steps(Request{
		Major: 16, Provider: ProviderCommunity,
		Want: []string{"pg_stat_statements"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := join(steps)
	if strings.Contains(text, "ALTER SYSTEM SET shared_preload_libraries") {
		t.Fatalf("unknown preload list was replaced:\n%s", text)
	}
	if !strings.Contains(text, "could not be read") {
		t.Fatalf("missing warning:\n%s", text)
	}
}

func TestStepsMarkManagedGaps(t *testing.T) {
	t.Parallel()
	steps, err := Steps(Request{
		Major: 16, Provider: ProviderRDS, PreloadKnown: true,
		Want: []string{"pg_stat_statements", "pg_stat_kcache"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := join(steps)
	if !strings.Contains(text, "pg_stat_kcache is not offered on Amazon RDS") {
		t.Fatalf("kcache should be refused:\n%s", text)
	}
	if strings.Contains(text, "pg-stat-kcache") {
		t.Fatalf("RDS step offered a PGDG package:\n%s", text)
	}
	if !strings.Contains(text, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;") {
		t.Fatalf("statements should still be created:\n%s", text)
	}
}

func TestStepsRejectUnknownName(t *testing.T) {
	t.Parallel()
	if _, err := Steps(Request{Want: []string{"not_an_extension"}}); err == nil {
		t.Fatal("unknown extension was accepted")
	}
}

func join(steps []Step) string {
	var b strings.Builder
	for _, s := range steps {
		b.WriteString(s.Text)
		b.WriteByte('\n')
	}
	return b.String()
}
