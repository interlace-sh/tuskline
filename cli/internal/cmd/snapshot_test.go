// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/interlace-sh/tuskline/core/snapshot"
)

func TestDiffCommand(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "older.json")
	newer := filepath.Join(dir, "newer.json")
	if err := snapshot.WriteFile(older, snapshot.Snapshot{Taken: time.Now().UTC(), Statements: []snapshot.Statement{{QueryID: 7, Query: "select 1", Calls: 10}}}); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.WriteFile(newer, snapshot.Snapshot{Taken: time.Now().UTC(), Statements: []snapshot.Statement{{QueryID: 7, Query: "select 1", Calls: 4}}}); err != nil {
		t.Fatal(err)
	}
	out, _, err := runApp(t, &app{}, "diff", older, newer)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "7") || !strings.Contains(out, "+4*") || !strings.Contains(out, "counter reset") {
		t.Fatalf("%s", out)
	}
}
