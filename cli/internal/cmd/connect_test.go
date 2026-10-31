// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// Nothing listens on port 1, so probes fail fast without a network.
const deadDSN = "postgres://app:s3cret-in-dsn@127.0.0.1:1/orders?sslmode=disable"

func seedInventory(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := &inventory.Inventory{}
	if _, err := inv.PutEnvironment("prod", inventory.EnvProd); err != nil {
		t.Fatal(err)
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConnectArguments(t *testing.T) {
	path := seedInventory(t)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"nothing to connect to", []string{"connect"}, "give env/cluster"},
		{"ref and flags", []string{"connect", "--dsn", deadDSN, "--env", "dev", "prod/orders"}, "not both"},
		{"bad output", []string{"connect", "prod/orders", "-o", "yaml"}, "want text or json"},
		{"unknown cluster", []string{"connect", "prod/orders"}, "not in the inventory"},
		{"kind without dsn", []string{"connect", "prod/orders", "--kind", "dev"}, "only apply with --dsn"},
		{"half a target", []string{"connect", "--dsn", deadDSN, "--env", "dev"}, "both an environment and a cluster name"},
		{"bad cluster name", []string{"connect", "--dsn", deadDSN, "dev/Not_Valid"}, "invalid name"},
		{"bad name flag", []string{"connect", "--dsn", deadDSN, "--env", "dev", "--name", "Not_Valid"}, "cluster: "},
		{"bad password env", []string{"connect", "--dsn", deadDSN, "--password-env", "1BAD"}, "not a valid environment variable name"},
		{"prod cannot become dev", []string{"connect", "--dsn", deadDSN, "--kind", "dev", "prod/orders"}, "already exists with kind prod"},
		{"new env needs a kind", []string{"connect", "--dsn", deadDSN, "team-a/orders"}, "give --kind"},
		{"bad kind", []string{"connect", "--dsn", deadDSN, "--kind", "qa", "team-a/orders"}, "want local, dev, staging or prod"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runApp(t, &app{}, append(tc.args, "--inventory", path)...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

func TestConnectNeverLeaksTheDSNPassword(t *testing.T) {
	path := seedInventory(t)
	out, errOut, err := runApp(t, &app{}, "connect", "--dsn", deadDSN, "prod/orders", "--inventory", path)
	if err == nil {
		t.Fatal("probe of a dead server succeeded")
	}
	for _, s := range []string{out, errOut, err.Error()} {
		if strings.Contains(s, "s3cret-in-dsn") {
			t.Fatalf("password leaked: %q", s)
		}
	}
	if !strings.Contains(errOut, "sslmode=disable") {
		t.Errorf("no TLS downgrade warning on stderr: %q", errOut)
	}
	inv, err := inventory.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := inv.Cluster(inventory.Ref{Env: "prod", Cluster: "orders"}); ok {
		t.Error("a failed probe was saved to the inventory")
	}
}

func TestConnectPromptsOnATerminal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.yaml")
	a := &app{interactive: true, in: strings.NewReader("staging\n\n")}
	_, errOut, err := runApp(t, a, "connect", "--dsn", deadDSN, "--inventory", path)
	if err == nil {
		t.Fatal("probe of a dead server succeeded")
	}
	for _, want := range []string{"Environment [dev]: ", "Cluster name [orders]: "} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr %q lacks %q", errOut, want)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("inventory written after a failed probe: %v", err)
	}
}

func TestSuggestName(t *testing.T) {
	cases := []struct {
		p    conn.Profile
		want string
	}{
		{conn.Profile{Database: "Orders_DB", Host: "x"}, "orders-db"},
		{conn.Profile{Host: "db1.example.com"}, "db1"},
		{conn.Profile{Database: "__", Host: "--"}, ""},
	}
	for _, tc := range cases {
		if got := suggestName(tc.p); got != tc.want {
			t.Errorf("suggestName(%+v) = %q, want %q", tc.p, got, tc.want)
		}
	}
}

func TestMissingStatus(t *testing.T) {
	cases := []struct {
		m    conn.Missing
		want string
	}{
		{conn.Missing{}, "not available"},
		{conn.Missing{Recommendation: conn.Recommendation{Create: true}, Available: true}, "CREATE EXTENSION"},
		{conn.Missing{Recommendation: conn.Recommendation{Create: true}, Available: true, NeedsPreload: true}, "restart) and CREATE"},
		{conn.Missing{Available: true, NeedsPreload: true}, "shared_preload_libraries (restart)"},
		{conn.Missing{Available: true, NeedsPreload: true, Installed: true}, "installed;"},
	}
	for _, tc := range cases {
		if got := missingStatus(tc.m); !strings.Contains(got, tc.want) {
			t.Errorf("missingStatus(%+v) = %q, want %q", tc.m, got, tc.want)
		}
	}
}
