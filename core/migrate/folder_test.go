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
	write("002_index.sql", "CREATE INDEX orders_note ON orders (note);\n")
	write("001_init.sql", "ALTER TABLE orders ADD COLUMN note text DEFAULT 'x';\n")
	folder, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if folder.Kind != KindPlain {
		t.Fatalf("kind %s", folder.Kind)
	}
	if folder.Files[0].Name != "001_init.sql" || folder.Files[1].Name != "002_index.sql" {
		t.Fatalf("%+v", folder.Files)
	}
	report, err := folder.Lint()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report, "002_index.sql") || !strings.Contains(report, "create-index") {
		t.Fatalf("%s", report)
	}
	if strings.Contains(report, "001_init.sql") {
		t.Fatalf("constant default was flagged:\n%s", report)
	}
	text, err := folder.Apply("staging", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Nothing has been applied.") || !strings.HasPrefix(text, "kind: plain\n") {
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

func TestFlywayOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{"V10__ten.sql", "V2__two.sql", "V1__one.sql", "R__view.sql", "U1__undo.sql"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("SELECT 1;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	folder, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if folder.Kind != KindFlyway {
		t.Fatal(folder.Kind)
	}
	var names []string
	for _, f := range folder.Files {
		names = append(names, f.Name)
	}
	want := []string{"V1__one.sql", "V2__two.sql", "V10__ten.sql", "R__view.sql"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("%v", names)
	}
}

func TestSqitchPlanOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	plan := "%project=orders\n\n# first\nusers 2026-01-01T00:00:00Z Mark <m@example.com>\norders 2026-01-02T00:00:00Z Mark <m@example.com>\n"
	if err := os.WriteFile(filepath.Join(dir, "sqitch.plan"), []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "deploy"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "CREATE INDEX orders_note ON orders (note);\n"
	if err := os.WriteFile(filepath.Join(dir, "deploy", "orders.sql"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	folder, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if folder.Kind != KindSqitch || len(folder.Files) != 2 {
		t.Fatalf("%+v", folder)
	}
	if folder.Files[0].Name != filepath.Join("deploy", "users.sql") || folder.Files[0].Err == nil {
		t.Fatalf("users: %+v", folder.Files[0])
	}
	report, err := folder.Lint()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report, "create-index") || !strings.Contains(report, "users.sql") {
		t.Fatalf("%s", report)
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
