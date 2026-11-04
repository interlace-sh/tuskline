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

func TestMigrateApplyRefusesProd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "001.sql"), []byte("SELECT 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, "migrate", "apply", "--dir", dir, "--env", "prod")
	if err == nil || !strings.Contains(err.Error(), "prod requires approval") {
		t.Fatalf("%v", err)
	}
	out, err := run(t, "migrate", "apply", "--dir", dir, "--env", "prod", "--approve")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Nothing has been applied.") {
		t.Fatalf("%s", out)
	}
}

func TestMigrateRunStopsBeforeConnect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "001.sql"), []byte("SELECT 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, "migrate", "apply", "--dir", dir, "--run")
	if err == nil || !strings.Contains(err.Error(), "--cluster") {
		t.Fatalf("%v", err)
	}

	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := &inventory.Inventory{}
	if _, err := inv.PutEnvironment("prod", inventory.EnvProd); err != nil {
		t.Fatal(err)
	}
	profile, err := conn.ParseDSN("postgres://app@127.0.0.1:1/postgres?sslmode=disable")
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
	args := []string{"migrate", "apply", "--dir", dir, "--run", "--cluster", "prod/orders", "--inventory", path}
	_, _, err = runApp(t, &app{}, args...)
	if err == nil || !strings.Contains(err.Error(), "prod requires approval") {
		t.Fatalf("%v", err)
	}
	_, _, err = runApp(t, &app{}, append(args, "--approve")...)
	if err == nil || !strings.Contains(err.Error(), "type the cluster name") {
		t.Fatalf("%v", err)
	}
}

func TestMigrateLintExit(t *testing.T) {
	dir := t.TempDir()
	body := "CREATE INDEX orders_note ON orders (note);\n"
	if err := os.WriteFile(filepath.Join(dir, "001.sql"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "migrate", "lint", "--dir", dir)
	if err == nil || !strings.Contains(err.Error(), "lock findings") {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(out, "create-index") {
		t.Fatalf("%s", out)
	}
}
