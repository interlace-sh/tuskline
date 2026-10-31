// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/interlace-sh/tuskline/core/inventory"
)

func TestHandoverKeepsThePasswordOutOfThePlan(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 10, 31, 22, 0, 0, 0, time.UTC)
	handed, err := Up(ComposeSpec{
		Name: "demo", Nodes: 1, Etcd: 1, Pooler: false, Env: "dev",
		Now: when, MonitorPassword: "monitor-secret",
		Superuser: "super-secret", Replication: "repl-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := stepBody(t, handed.Plan, "handover.yaml")
	for _, want := range []string{
		"tuskline_monitor", "sslmode: disable", "http://127.0.0.1:8008",
		"pgbackrest", "local-volume", "tl backup verify", "2026-11-07",
		"keychain: tuskline/dev/demo",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("handover.yaml lacks %q", want)
		}
	}
	if strings.Contains(body, "monitor-secret") || strings.Contains(body, "super-secret") {
		t.Fatal("handover.yaml contains a password")
	}
	sql := findStep(t, handed.Plan, "monitor.sql")
	if !sql.Secret || !strings.Contains(sql.After, "monitor-secret") || !strings.Contains(sql.After, "IF NOT EXISTS") {
		t.Fatal("monitor.sql is not a secret idempotent script")
	}
	role := findStep(t, handed.Plan, "tuskline_monitor")
	joined := strings.Join(role.Argv, " ")
	if role.StdinPath != "monitor.sql" || strings.Contains(joined, "monitor-secret") || strings.Contains(joined, "PASSWORD") {
		t.Fatalf("monitor command leaked or has no stdin: %+v", role)
	}
	changes, err := Changes(handed.Plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range changes {
		if strings.Contains(change.Diff, "monitor-secret") {
			t.Fatalf("diff leaked the monitor password:\n%s", change.Diff)
		}
	}
	if _, err := Up(ComposeSpec{Name: "demo", Env: "prod"}); err == nil {
		t.Fatal("prod was accepted")
	}
}

func TestSaveHandoverWritesAReference(t *testing.T) {
	inventory.UseKeyring(inventory.NewMemoryKeyring())
	t.Cleanup(func() { inventory.UseKeyring(nil) })
	path := t.TempDir() + "/inventory.yaml"
	handed, err := Up(ComposeSpec{
		Name: "demo", Nodes: 1, Etcd: 1, Env: "dev",
		Now: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), MonitorPassword: "monitor-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	handed.Cluster.LastProbe = nil
	if err := SaveHandover(path, handed.Cluster, handed.MonitorPassword); err != nil {
		t.Fatal(err)
	}
	inv, err := inventory.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := inv.Cluster(inventory.Ref{Env: "dev", Cluster: "demo"})
	if !ok || got.Profile.User != "tuskline_monitor" || got.Patroni == nil || got.Patroni.URL != "http://127.0.0.1:8008" {
		t.Fatalf("%+v", got)
	}
	if got.Password == nil || got.Password.Keychain != "tuskline/dev/demo" {
		t.Fatalf("password ref %+v", got.Password)
	}
	value, err := got.Password.Resolve()
	if err != nil || value != "monitor-secret" {
		t.Fatalf("resolve %q %v", value, err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved), "monitor-secret") {
		t.Fatal("inventory file contains the password")
	}
}
