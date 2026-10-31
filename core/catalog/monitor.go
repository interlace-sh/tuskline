// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"fmt"
	"strings"
)

// MonitorRole is the login role collectors use. It is not a superuser.
const MonitorRole = "tuskline_monitor"

// MonitorStatements is the SQL that creates MonitorRole. It sets no
// password: the operator uses \password or a cloud IAM login. Applying
// the statements is still a manual step.
func MonitorStatements() []string {
	role := MonitorRole
	return []string{
		"CREATE ROLE " + role + " LOGIN CONNECTION LIMIT 5",
		"GRANT pg_monitor TO " + role,
		"ALTER ROLE " + role + " SET default_transaction_read_only = on",
		"ALTER ROLE " + role + " SET statement_timeout = '5s'",
		"ALTER ROLE " + role + " SET lock_timeout = '1s'",
		"ALTER ROLE " + role + " SET idle_in_transaction_session_timeout = '10s'",
	}
}

// MonitorPlan is the preview for creating MonitorRole on provider.
// An empty provider means a self-managed server. Managed providers get a
// note naming the admin role that must run the SQL. Extension parameter
// groups, database flags and az commands stay in InstallPlan.
func MonitorPlan(provider string) ([]PlanLine, error) {
	note, err := monitorAdminNote(provider)
	if err != nil {
		return nil, err
	}
	lines := []PlanLine{{
		Kind: "note",
		Text: "# Preview only. Nothing here has been run. The script does not set a password.",
	}}
	if note != "" {
		lines = append(lines, PlanLine{Name: provider, Kind: "note", Text: note})
	}
	stmts := MonitorStatements()
	for i, stmt := range stmts {
		text := stmt + ";"
		if i == 0 {
			text += " -- password via \\password or IAM"
		}
		lines = append(lines, PlanLine{Name: MonitorRole, Kind: "sql", Text: text})
	}
	lines = append(lines, PlanLine{
		Name: MonitorRole, Kind: "note",
		Text: "-- Optional role for actions: GRANT pg_signal_backend TO tuskline_operator;",
	})
	return lines, nil
}

func monitorAdminNote(provider string) (string, error) {
	switch provider {
	case "", ProviderCommunity:
		return "", nil
	case ProviderRDS, ProviderAurora:
		return "# Run these statements as a member of rds_superuser. Extension parameters stay in the parameter group from InstallPlan.", nil
	case ProviderCloudSQL:
		return "# Run these statements as cloudsqlsuperuser. Extension flags stay in the gcloud --database-flags command from InstallPlan.", nil
	case ProviderAlloyDB:
		return "# Run these statements as alloydbsuperuser. Extension flags stay in the gcloud --database-flags command from InstallPlan.", nil
	case ProviderAzure:
		return "# Run these statements as azure_pg_admin. Extension parameters stay in the az postgres flexible-server commands from InstallPlan.", nil
	case ProviderNeon:
		return "# Run these statements as the Neon role that owns the database.", nil
	default:
		return "", fmt.Errorf("unknown provider %q", provider)
	}
}

// MonitorScript is MonitorPlan rendered as text for a self-managed server.
func MonitorScript() (string, error) {
	lines, err := MonitorPlan(ProviderCommunity)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line.Text)
		b.WriteByte('\n')
	}
	return b.String(), nil
}
