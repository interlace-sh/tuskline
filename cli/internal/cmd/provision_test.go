// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interlace-sh/tuskline/core/provision"
)

func TestComposeUpStopsWhenPreflightFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "stack")
	previous := probeCompose
	t.Cleanup(func() { probeCompose = previous })
	probeCompose = func(context.Context, provision.Need, string) (provision.Report, error) {
		return provision.Report{FreeBytes: 1, DockerServer: "27.0.0", DockerCompose: "2.29.0"}, nil
	}
	_, err := run(t, "provision", "compose", "up", "--name", "demo", "--dir", dir, "--nodes", "1", "--etcd", "1", "--pooler=false")
	if err == nil || !strings.Contains(err.Error(), "disk has") {
		t.Fatalf("got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "compose.yaml")); !os.IsNotExist(statErr) {
		t.Fatal("compose.yaml was written")
	}
}

func TestGuardedActionsStayPreview(t *testing.T) {
	out, err := run(t, "ha", "switchover", "--cluster", "orders", "--from", "pg1", "--to", "pg2", "--api", "http://127.0.0.1:9", "--prod", "--apply")
	if err == nil || !strings.Contains(err.Error(), "cluster name") {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(out, "Nothing has been sent") {
		t.Fatalf("preview missing:\n%s", out)
	}
	out, err = run(t, "backup", "verify", "--stanza", "demo")
	if err != nil || !strings.Contains(out, "Nothing has been started") {
		t.Fatalf("%v\n%s", err, out)
	}
	out, err = run(t, "provision", "cnpg", "plan", "--name", "demo", "--pooler=false")
	if err != nil || !strings.Contains(out, "Nothing has been applied") || !strings.Contains(out, "kind: Cluster") {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestComposeRejectsProd(t *testing.T) {
	_, err := run(t, "provision", "compose", "plan", "--name", "demo", "--env", "prod")
	if err == nil || !strings.Contains(err.Error(), "prod") {
		t.Fatalf("got %v", err)
	}
}

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
