// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileJournalAndOSFS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	journal := FileJournal{Path: filepath.Join(dir, "state", "apply.jsonl")}
	fs := OSFS{Root: dir}
	plan := Plan{Target: "compose", Steps: []Step{
		{ID: "cfg", Kind: KindFile, Path: "compose/patroni.yml", After: "scope: demo\n"},
	}}
	ctx := context.Background()
	if err := Apply(ctx, plan, journal, fs, nil, nil); err != nil {
		t.Fatal(err)
	}
	body, ok, err := fs.Read(ctx, "compose/patroni.yml")
	if err != nil || !ok || body != "scope: demo\n" {
		t.Fatalf("read %q ok=%v %v", body, ok, err)
	}
	if err := os.Mkdir(filepath.Join(dir, "pgdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pgdata", "PG_VERSION"), []byte("17\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nonEmpty, err := fs.NonEmpty(ctx, "pgdata")
	if err != nil || !nonEmpty {
		t.Fatalf("nonempty %v %v", nonEmpty, err)
	}
	again := Plan{Steps: []Step{{ID: "data", Kind: KindFile, Path: "pgdata", DataDir: true}}}
	if err := Apply(ctx, again, journal, fs, nil, nil); err == nil {
		t.Fatal("wrote into a non-empty data directory")
	}
	loaded, err := journal.Load(ctx)
	if err != nil || loaded["cfg"] == "" || loaded["data"] != "" {
		t.Fatalf("journal %#v %v", loaded, err)
	}
	info, err := os.Stat(journal.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("journal mode %o", info.Mode().Perm())
	}
}
