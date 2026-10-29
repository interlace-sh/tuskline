// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

var lsNow = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

func lsInventory(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := &inventory.Inventory{}
	for name, kind := range map[string]inventory.EnvKind{"prod": inventory.EnvProd, "team-a": inventory.EnvDev} {
		if _, err := inv.PutEnvironment(name, kind); err != nil {
			t.Fatal(err)
		}
	}
	p, err := conn.ParseDSN("postgres://app@db.example.com:6432/orders")
	if err != nil {
		t.Fatal(err)
	}
	probe := &inventory.ProbeRecord{At: lsNow.Add(-90 * time.Minute), VersionNum: 170006, Provider: conn.ProviderRDS, InRecovery: true}
	for _, cl := range []inventory.Cluster{
		{Name: "orders", Environment: "prod", Profile: p, LastProbe: probe},
		{Name: "billing", Environment: "prod", Profile: conn.Profile{Service: "billing"}},
	} {
		if err := inv.PutCluster(cl); err != nil {
			t.Fatal(err)
		}
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLsTable(t *testing.T) {
	path := lsInventory(t)
	out, _, err := runApp(t, &app{now: func() time.Time { return lsNow }}, "ls", "--inventory", path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"ENVIRONMENT   CLUSTER  TARGET                      VERSION  PROVIDER  ROLE     LAST PROBE",
		"prod          billing  service=billing                                         never",
		"prod          orders   db.example.com:6432/orders  17.6     rds       standby  1h ago",
		"team-a (dev)  -",
		"",
	}, "\n")
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("colour without a terminal")
	}
}

func TestLsColour(t *testing.T) {
	path := lsInventory(t)
	out, _, err := runApp(t, &app{colour: true}, "ls", "--inventory", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\x1b[1;38;2;229;72;77mprod\x1b[0m") || !strings.Contains(out, "\x1b[1;38;2;48;164;108mteam-a (dev)\x1b[0m") {
		t.Fatalf("no environment colours:\n%q", out)
	}
}

func TestLsJSON(t *testing.T) {
	path := lsInventory(t)
	out, _, err := runApp(t, &app{}, "ls", "-o", "json", "--inventory", path)
	if err != nil {
		t.Fatal(err)
	}
	var res lsResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v:\n%s", err, out)
	}
	if res.Inventory != path || len(res.Environments) != 2 || len(res.Clusters) != 2 || res.Clusters[1].LastProbe.VersionNum != 170006 {
		t.Fatalf("unexpected: %+v", res)
	}
	if !res.Environments[0].Policy.ReadOnly {
		t.Error("prod policy not in JSON")
	}
}

func TestLsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "none.yaml")
	out, _, err := runApp(t, &app{}, "ls", "--inventory", path)
	if err != nil || !strings.Contains(out, "is empty") || !strings.Contains(out, "tl connect --dsn") {
		t.Fatalf("got %q, %v", out, err)
	}
	out, _, err = runApp(t, &app{}, "ls", "-o", "json", "--inventory", path)
	if err != nil || !strings.Contains(out, `"clusters": []`) {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestAge(t *testing.T) {
	for d, want := range map[time.Duration]string{-time.Second: "just now", 5 * time.Second: "5s ago", 4 * time.Minute: "4m ago", 47 * time.Hour: "47h ago", 72 * time.Hour: "3d ago"} {
		if got := age(d); got != want {
			t.Errorf("age(%v) = %q, want %q", d, got, want)
		}
	}
}
