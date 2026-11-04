// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlainLintAndApply(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("10_ten.sql", "SELECT 10;\n")
	write("2_two.sql", "CREATE INDEX orders_note ON orders (note);\n")
	write("1_one.sql", "ALTER TABLE orders ADD COLUMN note text DEFAULT 'x';\n")
	write("notes.txt", "not a migration\n")
	folder, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if folder.Files[0].Name != "1_one.sql" || folder.Files[1].Name != "2_two.sql" || folder.Files[2].Name != "10_ten.sql" {
		t.Fatalf("%+v", names(folder))
	}
	if folder.Files[0].Checksum == "" || folder.Files[0].Checksum == folder.Files[1].Checksum {
		t.Fatalf("checksums %+v", folder.Files)
	}
	report, err := folder.Lint()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report, "2_two.sql") || !strings.Contains(report, "create-index") {
		t.Fatalf("%s", report)
	}
	if strings.Contains(report, "1_one.sql") {
		t.Fatalf("constant default was flagged:\n%s", report)
	}
	text, err := folder.Apply("staging", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Nothing has been applied.") || !strings.Contains(text, "1_one.sql") {
		t.Fatalf("%s", text)
	}
	if _, err := folder.Apply("prod", false); !errors.Is(err, ErrApproval) {
		t.Fatalf("prod: %v", err)
	}
	ok, err := folder.Apply("prod", true)
	if err != nil || !strings.Contains(ok, "Nothing has been applied.") {
		t.Fatalf("%s %v", ok, err)
	}
}

func TestUnknownEnvironment(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "001.sql"), []byte("SELECT 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	folder, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := folder.Apply("production", true); err == nil {
		t.Fatal("expected an unknown environment")
	}
}

func TestPendingRefusesDivergence(t *testing.T) {
	t.Parallel()
	files := []File{
		{Name: "1.sql", Checksum: "a"},
		{Name: "2.sql", Checksum: "b"},
		{Name: "3.sql", Checksum: "c"},
	}
	pending, err := Pending(files, []Applied{{Name: "1.sql", Checksum: "a", Success: true}})
	if err != nil || len(pending) != 2 || pending[0].Name != "2.sql" {
		t.Fatalf("%v %+v", err, pending)
	}
	_, err = Pending(files, []Applied{{Name: "1.sql", Checksum: "changed", Success: true}})
	if !errors.Is(err, ErrChecksumChanged) {
		t.Fatalf("%v", err)
	}
	_, err = Pending(files, []Applied{{Name: "3.sql", Checksum: "c", Success: true}})
	if !errors.Is(err, ErrOutOfOrder) {
		t.Fatalf("%v", err)
	}
	_, err = Pending(files[:2], []Applied{{Name: "3.sql", Checksum: "c", Success: true}})
	if !errors.Is(err, ErrMissingScript) {
		t.Fatalf("%v", err)
	}
	_, err = Pending(files, []Applied{{Name: "2.sql", Checksum: "b", Success: false}})
	if !errors.Is(err, ErrPreviousFailure) {
		t.Fatalf("%v", err)
	}
}

func TestScriptSplitting(t *testing.T) {
	t.Parallel()
	sql := "CREATE FUNCTION f() RETURNS int AS $$ SELECT 1; $$ LANGUAGE sql;\nSELECT 'a;b';\n"
	parts, err := splitSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 2 || !strings.Contains(parts[0], "SELECT 1;") || parts[1] != "SELECT 'a;b'" {
		t.Fatalf("%#v", parts)
	}
	script, err := prepare(File{Name: "1.sql", SQL: "CREATE TABLE t (id int); INSERT INTO t VALUES (1);"})
	if err != nil || script.outside || len(script.stmts) != 2 {
		t.Fatalf("%+v %v", script, err)
	}
	script, err = prepare(File{Name: "2.sql", SQL: "CREATE INDEX CONCURRENTLY t_id ON t (id);"})
	if err != nil || !script.outside {
		t.Fatalf("%+v %v", script, err)
	}
	_, err = prepare(File{Name: "3.sql", SQL: "CREATE TABLE t (id int);\nCREATE INDEX CONCURRENTLY t_id ON t (id);"})
	if !errors.Is(err, ErrMixedScript) {
		t.Fatalf("%v", err)
	}
	script, err = prepare(File{Name: "4.sql", SQL: "VACUUM t;"})
	if err != nil || !script.outside {
		t.Fatalf("%+v %v", script, err)
	}
}

func names(folder Folder) []string {
	out := make([]string, len(folder.Files))
	for i, file := range folder.Files {
		out[i] = file.Name
	}
	return out
}
