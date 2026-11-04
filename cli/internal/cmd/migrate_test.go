// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
