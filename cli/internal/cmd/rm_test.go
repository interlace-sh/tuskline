// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

func TestRmYesRemovesADevCluster(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := &inventory.Inventory{}
	if _, err := inv.PutEnvironment("dev", inventory.EnvDev); err != nil {
		t.Fatal(err)
	}
	profile, err := conn.ParseDSN("postgres://app@127.0.0.1:5432/postgres?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.PutCluster(inventory.Cluster{
		Name: "scratch", Environment: "dev", Profile: profile.WithPassword(""),
	}); err != nil {
		t.Fatal(err)
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	out, _, err := runApp(t, &app{}, "rm", "dev/scratch", "--yes", "--inventory", path)
	if err != nil || !strings.Contains(out, "removed dev/scratch") {
		t.Fatalf("%v %s", err, out)
	}
	loaded, err := inventory.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Cluster(inventory.Ref{Env: "dev", Cluster: "scratch"}); ok {
		t.Fatal("cluster remained")
	}
}

func TestRmYesRefusesProd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := &inventory.Inventory{}
	if _, err := inv.PutEnvironment("prod", inventory.EnvProd); err != nil {
		t.Fatal(err)
	}
	profile, err := conn.ParseDSN("postgres://app@db.example.com:5432/orders")
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.PutCluster(inventory.Cluster{
		Name: "orders", Environment: "prod", Profile: profile.WithPassword(""),
	}); err != nil {
		t.Fatal(err)
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	_, _, err = runApp(t, &app{}, "rm", "prod/orders", "--yes", "--inventory", path)
	if err == nil || !strings.Contains(err.Error(), "--yes is not accepted") {
		t.Fatalf("%v", err)
	}
}
