// SPDX-License-Identifier: Apache-2.0

package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTargetKey(t *testing.T) {
	t.Parallel()
	got, err := TargetKey("prod/orders")
	if err != nil || got != "prod__orders" {
		t.Fatalf("%q %v", got, err)
	}
	for _, bad := range []string{"orders", "prod/a/b", "../x", "prod/..", ""} {
		if _, err := TargetKey(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestSegmentWriteReadAndRollup(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	at := time.Date(2026, 10, 29, 11, 4, 10, 0, time.UTC)
	now := at
	s := Segments{Root: root, Now: func() time.Time { return now }}
	samples := []Sample{
		Gauge(at, "cache_hit", 1),
		Gauge(at.Add(time.Second), "cache_hit", 3),
		Gauge(at.Add(2*time.Second), "cache_hit", 5),
		Counter(at, "xact_commit", 10),
		Counter(at.Add(time.Second), "xact_commit", 12),
		Counter(at.Add(2*time.Second), "xact_commit", 3),
	}
	if err := s.Write("dev/pg", samples); err != nil {
		t.Fatal(err)
	}
	if err := s.Write("dev/pg", []Sample{Gauge(at.Add(3*time.Second), "cache_hit", 7)}); err != nil {
		t.Fatal(err)
	}
	if err := s.Write("dev/pg", []Sample{Gauge(at.Add(time.Minute), "cache_hit", 1)}); err == nil {
		t.Fatal("wrote into a closed minute")
	}
	got, err := s.Read("dev/pg", Res1s, at, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 7 {
		t.Fatalf("got %d samples", len(got))
	}

	now = at.Add(2 * time.Hour)
	if err := s.Retain("dev/pg", now); err != nil {
		t.Fatal(err)
	}
	raw, err := s.Read("dev/pg", Res1s, at, at.Add(time.Minute))
	if err != nil || len(raw) != 0 {
		t.Fatalf("1s file still readable: %d %v", len(raw), err)
	}
	rolled, err := s.Read("dev/pg", Res1m, at.Truncate(time.Hour), at.Truncate(time.Hour).Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rolled) != 2 {
		t.Fatalf("rollup %d: %+v", len(rolled), rolled)
	}
	for _, sample := range rolled {
		switch sample.Name {
		case "cache_hit":
			if sample.Count != 4 || sample.Min != 1 || sample.Max != 7 || sample.Average() != 4 {
				t.Fatalf("gauge %+v", sample)
			}
		case "xact_commit":
			// 10 -> 12 is +2, 12 -> 3 is a reset worth 3.
			if sample.Sum != 5 {
				t.Fatalf("counter %+v", sample)
			}
		default:
			t.Fatalf("unexpected %s", sample.Name)
		}
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "*", "*", "*.tmp")); len(matches) != 0 {
		t.Fatalf("temp files left: %v", matches)
	}
}

func TestRollupSumsCoarserCounters(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 29, 11, 0, 0, 0, time.UTC)
	got := rollup([]Sample{
		{At: at, Name: "xact", Kind: KindCounter, Sum: 5, Count: 2},
		{At: at.Add(time.Minute), Name: "xact", Kind: KindCounter, Sum: 4, Count: 1},
	}, Res1m)
	if len(got) != 1 || got[0].Sum != 9 || got[0].Count != 3 || !got[0].At.Equal(at.Truncate(time.Hour)) {
		t.Fatalf("%+v", got)
	}
}
