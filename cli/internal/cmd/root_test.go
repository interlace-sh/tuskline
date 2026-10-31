// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/interlace-sh/tuskline/core/version"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	root := NewRoot(&out, &out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

// runApp runs tl with a test app, a fresh inventory file and no libpq
// files from the developer's home. It returns stdout and stderr apart.
func runApp(t *testing.T, a *app, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	isolatePGFiles(t)
	if a.now == nil {
		a.now = func() time.Time { return time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC) }
	}
	if a.in == nil {
		a.in = strings.NewReader("")
	}
	var out, errOut bytes.Buffer
	root := newRoot(&out, &errOut, a)
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func isolatePGFiles(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PGPASSFILE", filepath.Join(dir, "pgpass"))
	t.Setenv("PGSERVICEFILE", filepath.Join(dir, "pg_service.conf"))
	t.Setenv("PGSYSCONFDIR", "")
}

func TestVersion(t *testing.T) {
	want := "tl " + version.String() + "\n"
	for _, args := range [][]string{{"-v"}, {"--version"}} {
		out, err := run(t, args...)
		if err != nil {
			t.Fatal(err)
		}
		if out != want {
			t.Fatalf("%v: got %q", args, out)
		}
	}
	if _, err := run(t, "version"); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("tl version: %v", err)
	}
}

func TestConnectRejectsBadRef(t *testing.T) {
	if _, err := run(t, "connect", "orders"); err == nil || !strings.Contains(err.Error(), "env/cluster") {
		t.Fatalf("got %v", err)
	}
}
