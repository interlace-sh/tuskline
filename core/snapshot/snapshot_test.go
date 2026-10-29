// SPDX-License-Identifier: Apache-2.0

package snapshot

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDiffHandlesGrowthAndReset(t *testing.T) {
	t.Parallel()
	older := Snapshot{Statements: []Statement{{QueryID: 1, Query: "select 1", Calls: 10, TotalExec: 5}}}
	newer := Snapshot{Statements: []Statement{
		{QueryID: 1, Query: "select 1", Calls: 4, TotalExec: 8},
		{QueryID: 2, Query: "select 2", Calls: 3, TotalExec: 1},
	}}
	got := Diff(older, newer)
	if len(got) != 2 || got[0].Calls != 4 || !got[0].CallsReset || got[0].TotalExec != 3 || got[0].ExecReset || got[1].Calls != 3 {
		t.Fatalf("%+v", got)
	}
}

func TestSnapshotFileRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "snap.json")
	snap := Snapshot{Taken: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC), VersionNum: 170006, Statements: []Statement{{QueryID: 9, Query: "select 1", Calls: 2}}}
	if err := WriteFile(path, snap); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.VersionNum != 170006 || len(got.Statements) != 1 || got.Statements[0].QueryID != 9 {
		t.Fatalf("%+v", got)
	}
}
