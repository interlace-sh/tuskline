// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeUpRefusesAForeignDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := run(t, "provision", "compose", "up", "--name", "demo", "--dir", dir, "--nodes", "1", "--etcd", "1", "--pooler=false")
	if err == nil || !strings.Contains(err.Error(), "notes.txt") {
		t.Fatalf("got %v", err)
	}
}
