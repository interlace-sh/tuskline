// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckOutputDir(t *testing.T) {
	t.Parallel()
	if err := CheckOutputDir(""); err == nil {
		t.Fatal("empty directory was accepted")
	}
	missing := filepath.Join(t.TempDir(), "fresh")
	if err := CheckOutputDir(missing); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte("name: demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "pgbouncer"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pgbouncer", "userlist.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckOutputDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckOutputDir(dir); err == nil {
		t.Fatal("a foreign file was accepted")
	}
}
