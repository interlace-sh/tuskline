// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"strings"
	"testing"
)

func TestMonitorPlanHasNoPassword(t *testing.T) {
	t.Parallel()
	lines, err := MonitorPlan(ProviderCommunity)
	if err != nil {
		t.Fatal(err)
	}
	text := planText(lines)
	for _, stmt := range MonitorStatements() {
		if strings.Contains(strings.ToUpper(stmt), "PASSWORD") {
			t.Fatalf("statement sets a password: %s", stmt)
		}
	}
	for _, want := range []string{
		"CREATE ROLE tuskline_monitor LOGIN CONNECTION LIMIT 5;",
		"GRANT pg_monitor TO tuskline_monitor;",
		"ALTER ROLE tuskline_monitor SET default_transaction_read_only = on;",
		"ALTER ROLE tuskline_monitor SET statement_timeout = '5s';",
		"ALTER ROLE tuskline_monitor SET lock_timeout = '1s';",
		"ALTER ROLE tuskline_monitor SET idle_in_transaction_session_timeout = '10s';",
		"GRANT pg_signal_backend TO tuskline_operator;",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %s\n%s", want, text)
		}
	}
	script, err := MonitorScript()
	if err != nil || script != text {
		t.Fatalf("MonitorScript = %q, err %v", script, err)
	}
}

func TestMonitorPlanNamesTheAdminRole(t *testing.T) {
	t.Parallel()
	cases := []struct {
		provider string
		want     string
	}{
		{ProviderRDS, "rds_superuser"},
		{ProviderAurora, "rds_superuser"},
		{ProviderCloudSQL, "cloudsqlsuperuser"},
		{ProviderAlloyDB, "alloydbsuperuser"},
		{ProviderAzure, "azure_pg_admin"},
		{ProviderNeon, "Neon role"},
	}
	for _, tc := range cases {
		lines, err := MonitorPlan(tc.provider)
		if err != nil {
			t.Fatal(err)
		}
		text := planText(lines)
		if !strings.Contains(text, tc.want) || !strings.Contains(text, "GRANT pg_monitor TO tuskline_monitor;") {
			t.Errorf("%s:\n%s", tc.provider, text)
		}
	}
	if _, err := MonitorPlan("not-a-provider"); err == nil {
		t.Fatal("unknown provider was planned")
	}
}

func TestInstallPlanCloudSnippets(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		req  Request
		want []string
		ban  []string
	}{
		{
			name: "aurora parameter group",
			req: Request{
				Major: 16, Provider: ProviderAurora, PreloadKnown: true,
				Want: []string{"pg_stat_statements"},
			},
			want: []string{
				"aws rds modify-db-cluster-parameter-group",
				"--db-cluster-parameter-group-name <cluster-parameter-group>",
				"ApplyMethod=pending-reboot",
				"CREATE EXTENSION IF NOT EXISTS pg_stat_statements;",
			},
			ban: []string{"apt-get", "ALTER SYSTEM"},
		},
		{
			name: "cloud sql flags",
			req: Request{
				Major: 16, Provider: ProviderCloudSQL, PreloadKnown: true,
				CurrentPreload: []string{"pg_stat_statements"},
				Want:           []string{"auto_explain"},
			},
			want: []string{
				"gcloud sql instances patch <instance> --database-flags=",
				"shared_preload_libraries=pg_stat_statements,auto_explain",
				"auto_explain.log_min_duration=500ms",
				"--database-flags replaces the instance flag list",
			},
			ban: []string{"apt-get", "ALTER SYSTEM", "CREATE EXTENSION"},
		},
		{
			name: "azure az commands",
			req: Request{
				Major: 16, Provider: ProviderAzure, PreloadKnown: true,
				Want: []string{"pg_stat_statements", "hypopg"},
			},
			want: []string{
				"az postgres flexible-server parameter set",
				"--name shared_preload_libraries --value 'pg_stat_statements'",
				"--name azure.extensions --value 'pg_stat_statements,hypopg'",
				"CREATE EXTENSION IF NOT EXISTS hypopg;",
			},
			ban: []string{"apt-get", "ALTER SYSTEM"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lines, err := InstallPlan(tc.req)
			if err != nil {
				t.Fatal(err)
			}
			text := planText(lines)
			for _, want := range tc.want {
				if !strings.Contains(text, want) {
					t.Errorf("missing %s\n%s", want, text)
				}
			}
			for _, ban := range tc.ban {
				if strings.Contains(text, ban) {
					t.Errorf("plan contains %s\n%s", ban, text)
				}
			}
		})
	}
}
