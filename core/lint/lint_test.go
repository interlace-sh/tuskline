// SPDX-License-Identifier: Apache-2.0

package lint

import (
	"os"
	"strings"
	"testing"
)

func TestCheckLocks(t *testing.T) {
	t.Parallel()
	sql := strings.TrimSpace(`
ALTER TABLE orders ADD COLUMN paid_at timestamptz DEFAULT now();
ALTER TABLE orders ADD COLUMN token uuid DEFAULT gen_random_uuid();
ALTER TABLE orders ADD COLUMN note text DEFAULT 'x';
CREATE INDEX CONCURRENTLY orders_paid ON orders (paid_at);
CREATE INDEX orders_note ON orders (note);
ALTER TABLE orders ALTER COLUMN note TYPE varchar(40);
ALTER TABLE orders ALTER COLUMN note SET NOT NULL;
ALTER TABLE small_ok ALTER COLUMN id SET NOT NULL;
`)
	stats := Stats{Tables: map[string]TableStat{
		"orders":   {Bytes: 200 << 20, Active: 3, Known: true},
		"small_ok": {Bytes: 4096, Active: 0, Known: true},
	}}
	got, err := Check(sql, stats)
	if err != nil {
		t.Fatal(err)
	}
	report := Report(got)
	want, err := os.ReadFile("testdata/locks.txt")
	if err != nil {
		t.Fatal(err)
	}
	if report != string(want) {
		t.Fatalf("report:\n%s\nwant:\n%s", report, want)
	}
}

func TestUnknownSizeAndActivity(t *testing.T) {
	t.Parallel()
	got, err := Check("ALTER TABLE public.events ALTER COLUMN id SET NOT NULL;", Stats{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Summary, "not known") {
		t.Fatalf("%+v", got)
	}
	busy, err := Check("ALTER TABLE events ALTER COLUMN id SET NOT NULL;", Stats{Tables: map[string]TableStat{
		"public.events": {Bytes: 10, Active: 2, Known: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(busy) != 1 || !strings.Contains(busy[0].Summary, "2 sessions") {
		t.Fatalf("%+v", busy)
	}
}

func TestUnknownFunction(t *testing.T) {
	t.Parallel()
	got, err := Check("ALTER TABLE orders ADD COLUMN token text DEFAULT app_token();", Stats{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Summary, "app_token") {
		t.Fatalf("%+v", got)
	}
}

func TestParseError(t *testing.T) {
	t.Parallel()
	_, err := Check("ALTER TABLE", Stats{})
	if err == nil {
		t.Fatal("expected a parse error")
	}
}
