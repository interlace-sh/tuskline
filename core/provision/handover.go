// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"fmt"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/interlace-sh/tuskline/core/catalog"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

const restoreDrillCommand = "tl backup verify"

func handover(spec ComposeSpec) (inventory.Cluster, string, string, error) {
	env := spec.Env
	if env == "" {
		env = "dev"
	}
	if env == string(inventory.EnvProd) {
		return inventory.Cluster{}, "", "", fmt.Errorf("compose cannot be added to a prod environment")
	}
	if err := inventory.ValidateName(env); err != nil {
		return inventory.Cluster{}, "", "", fmt.Errorf("environment: %w", err)
	}
	password := spec.MonitorPassword
	if password == "" {
		var err error
		password, err = randomSecret()
		if err != nil {
			return inventory.Cluster{}, "", "", err
		}
	}
	now := spec.Now
	if now.IsZero() {
		now = time.Now()
	}
	due := spec.DrillDue
	if due.IsZero() {
		due = now.UTC().Add(7 * 24 * time.Hour)
	}
	item, err := inventory.KeychainItem(env, spec.Name)
	if err != nil {
		return inventory.Cluster{}, "", "", err
	}
	sql, err := monitorSQL(password)
	if err != nil {
		return inventory.Cluster{}, "", "", err
	}
	cluster := inventory.Cluster{
		Name:        spec.Name,
		Environment: env,
		Provider:    "community",
		HAEngine:    "patroni",
		Profile: conn.Profile{
			Host:            "127.0.0.1",
			Port:            5000,
			Database:        "postgres",
			User:            catalog.MonitorRole,
			TLS:             conn.TLSDisable,
			ApplicationName: "tuskline",
		},
		Password: &inventory.SecretRef{Keychain: item},
		Patroni:  &inventory.Patroni{URL: "http://127.0.0.1:8008"},
		Backup: &inventory.Backup{
			Tool: "pgbackrest", Stanza: spec.Name, Repo: "local-volume", Path: "/var/lib/pgbackrest",
		},
		RestoreDrill: &inventory.RestoreDrill{
			Status: "scheduled", Due: due.UTC(), Command: restoreDrillCommand,
		},
	}
	return cluster, password, sql, nil
}

func handoverDocument(cluster inventory.Cluster) (string, error) {
	raw, err := yaml.Marshal(cluster)
	if err != nil {
		return "", err
	}
	return "# Non-production hand-over. The monitor password is not in this file.\n" + string(raw), nil
}

func monitorSQL(password string) (string, error) {
	if password == "" || strings.ContainsAny(password, "$\r\n\x00") {
		return "", fmt.Errorf("monitor password is empty or contains a newline or $")
	}
	stmts := catalog.MonitorStatements()
	quoted := "'" + strings.ReplaceAll(password, "'", "''") + "'"
	role := catalog.MonitorRole
	var b strings.Builder
	b.WriteString("-- Applied by tl provision compose up. Do not print this file.\n")
	fmt.Fprintf(&b, `DO $tuskline$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
    CREATE ROLE %s LOGIN CONNECTION LIMIT 5 PASSWORD %s;
  ELSE
    ALTER ROLE %s WITH LOGIN CONNECTION LIMIT 5 PASSWORD %s;
  END IF;
END
$tuskline$;
`, role, role, quoted, role, quoted)
	for _, stmt := range stmts[1:] {
		b.WriteString(stmt)
		b.WriteString(";\n")
	}
	return b.String(), nil
}

// SaveHandover stores the monitor password in the OS keychain and the cluster
// in the inventory. A prod environment is refused. An existing environment
// keeps its kind. LastProbe on an existing cluster is kept.
func SaveHandover(path string, cluster inventory.Cluster, password string) error {
	if cluster.Environment == string(inventory.EnvProd) || password == "" {
		return fmt.Errorf("compose hand-over needs a non-production environment and a monitor password")
	}
	if _, err := inventory.StorePassword(cluster.Environment, cluster.Name, password); err != nil {
		return err
	}
	inv, err := inventory.Load(path)
	if err != nil {
		return err
	}
	if env, ok := inv.Environment(cluster.Environment); ok {
		if env.Kind == inventory.EnvProd {
			return fmt.Errorf("compose cannot be added to a prod environment")
		}
	} else {
		kind := inventory.EnvDev
		switch cluster.Environment {
		case string(inventory.EnvLocal):
			kind = inventory.EnvLocal
		case string(inventory.EnvStaging):
			kind = inventory.EnvStaging
		}
		if _, err := inv.PutEnvironment(cluster.Environment, kind); err != nil {
			return err
		}
	}
	if old, ok := inv.Cluster(cluster.Ref()); ok {
		cluster.LastProbe = old.LastProbe
	}
	if err := inv.PutCluster(cluster); err != nil {
		return err
	}
	return inv.Save(path)
}
