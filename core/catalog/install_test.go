// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"strings"
	"testing"
)

func TestInstallPlanSelfManaged(t *testing.T) {
	t.Parallel()
	lines, err := InstallPlan(Request{
		Major: 16, Provider: ProviderCommunity, PreloadKnown: true,
		CurrentPreload: []string{"neon"},
		Want:           []string{"pg_stat_statements", "hypopg", "pg_wait_sampling"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := planText(lines)
	if !strings.HasPrefix(text, "# Preview only") {
		t.Fatalf("plan is not marked as a preview:\n%s", text)
	}
	if strings.Contains(text, "git clone") || strings.Contains(text, "make install") || strings.Contains(text, "pgxs") {
		t.Fatalf("plan compiles from source:\n%s", text)
	}
	if !strings.Contains(text, "sudo apt-get install -y postgresql-16-hypopg") {
		t.Fatalf("missing PGDG package:\n%s", text)
	}
	if !strings.Contains(text, "Pigsty") || !strings.Contains(text, "postgresql-16-pg-wait-sampling") {
		t.Fatalf("missing Pigsty package:\n%s", text)
	}
	if !strings.Contains(text, "ALTER SYSTEM SET shared_preload_libraries = 'neon,pg_stat_statements,pg_wait_sampling';") {
		t.Fatalf("preload was not merged:\n%s", text)
	}
	if !strings.Contains(text, "CREATE EXTENSION IF NOT EXISTS hypopg;") {
		t.Fatalf("missing SQL:\n%s", text)
	}
}

func TestInstallPlanPatroniSkipsAlterSystem(t *testing.T) {
	t.Parallel()
	lines, err := InstallPlan(Request{
		Major: 16, Provider: ProviderCommunity, Patroni: true, PreloadKnown: true,
		Want: []string{"pg_stat_statements"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := planText(lines)
	if strings.Contains(text, "ALTER SYSTEM") || !strings.Contains(text, "patronictl edit-config") {
		t.Fatalf("patroni plan:\n%s", text)
	}
}

func TestInstallPlanManaged(t *testing.T) {
	t.Parallel()
	lines, err := InstallPlan(Request{
		Major: 16, Provider: ProviderRDS, PreloadKnown: true,
		Want: []string{"pg_stat_statements", "pg_stat_kcache"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := planText(lines)
	if !strings.Contains(text, "pg_stat_kcache is not offered here") {
		t.Fatalf("missing refusal:\n%s", text)
	}
	if strings.Contains(text, "apt-get") || strings.Contains(text, "ALTER SYSTEM") {
		t.Fatalf("RDS plan used a package or ALTER SYSTEM:\n%s", text)
	}
	if !strings.Contains(text, "aws rds modify-db-parameter-group") || !strings.Contains(text, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;") {
		t.Fatalf("RDS plan:\n%s", text)
	}
}

func TestInstallPlanRejectsUnknown(t *testing.T) {
	t.Parallel()
	if _, err := InstallPlan(Request{Want: []string{"not_an_extension"}}); err == nil {
		t.Fatal("unknown extension was planned")
	}
}

func planText(lines []PlanLine) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line.Text)
		b.WriteByte('\n')
	}
	return b.String()
}
