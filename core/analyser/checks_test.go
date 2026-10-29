// SPDX-License-Identifier: Apache-2.0

package analyser

import (
	"testing"
	"time"
)

func TestMissingStatements(t *testing.T) {
	t.Parallel()
	if MissingStatements(true) != nil {
		t.Fatal("installed extension produced a finding")
	}
	if got := MissingStatements(false); got == nil || got.Check != "pg_stat_statements" || got.Fix == "" {
		t.Fatalf("%+v", got)
	}
}

func TestCacheHit(t *testing.T) {
	t.Parallel()
	if CacheHit("app", 10, 10) != nil {
		t.Fatal("too few blocks")
	}
	if CacheHit("app", 10000, 10) != nil {
		t.Fatal("healthy ratio")
	}
	got := CacheHit("app", 1000, 1000)
	if got == nil || got.Severity != SeverityWarn {
		t.Fatalf("%+v", got)
	}
}

func TestIdleWraparoundLagAndBloat(t *testing.T) {
	t.Parallel()
	if IdleInTransaction(1, "app", "select 1", time.Minute) != nil {
		t.Fatal("short idle")
	}
	if got := IdleInTransaction(7, "app", "select 1", 10*time.Minute); got == nil || got.Check != "idle_in_transaction" {
		t.Fatalf("%+v", got)
	}
	if Wraparound("app", 100) != nil {
		t.Fatal("young database")
	}
	if got := Wraparound("app", WraparoundWarn); got == nil {
		t.Fatal("no wraparound finding")
	}
	if ReplicationLag("standby", time.Second, time.Minute) != nil {
		t.Fatal("lag under the threshold")
	}
	if got := ReplicationLag("standby", 2*time.Minute, time.Minute); got == nil || got.Check != "replication_lag" {
		t.Fatalf("%+v", got)
	}
	if Bloat("public", "t", 100, 50) != nil {
		t.Fatal("too few dead tuples")
	}
	if got := Bloat("public", "t", 1000, 1000); got == nil || got.Check != "bloat" {
		t.Fatalf("%+v", got)
	}
}

func TestIndexes(t *testing.T) {
	t.Parallel()
	indexes := []Index{
		{Schema: "public", Table: "t", Name: "t_a_idx", Key: "a", Scans: 0, Bytes: MinIndexBytes},
		{Schema: "public", Table: "t", Name: "t_a_idx2", Key: "a", Scans: 3, Bytes: MinIndexBytes},
		{Schema: "public", Table: "t", Name: "t_b_idx", Key: "b", Scans: 0, Bytes: 10},
	}
	unused := UnusedIndexes(indexes)
	if len(unused) != 1 || unused[0].Summary == "" {
		t.Fatalf("unused %+v", unused)
	}
	dups := DuplicateIndexes(indexes)
	if len(dups) != 1 || dups[0].Check != "duplicate_index" {
		t.Fatalf("dups %+v", dups)
	}
}
