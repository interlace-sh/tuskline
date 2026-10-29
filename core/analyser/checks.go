// SPDX-License-Identifier: Apache-2.0

package analyser

import (
	"fmt"
	"time"
)

// Thresholds for the first checks. They are starting points, not tuning.
const (
	// MinCacheHit is the shared-buffer hit ratio below which a database
	// with enough reads is worth a warning.
	MinCacheHit = 0.99
	// MinBlocks is how many block reads plus hits a database needs before
	// the hit ratio means anything.
	MinBlocks = 1000
	// IdleInTransactionWarn is how long a transaction may sit idle.
	IdleInTransactionWarn = 5 * time.Minute
	// WraparoundWarn is the age of datfrozenxid, in transactions, that
	// deserves a warning. Autovacuum usually starts much earlier.
	WraparoundWarn = 1_000_000_000
	// MinDeadTuples is the dead-tuple count below which a bloat ratio is noise.
	MinDeadTuples = 1000
	// MaxDeadRatio is n_dead_tup / (n_live_tup + n_dead_tup) worth a warning.
	MaxDeadRatio = 0.2
	// MinIndexBytes ignores tiny unused indexes.
	MinIndexBytes = 1 << 20
)

// MissingStatements reports that pg_stat_statements is not installed.
// installed is false when the extension is absent from the database.
func MissingStatements(installed bool) *Finding {
	if installed {
		return nil
	}
	return &Finding{
		Check: "pg_stat_statements", Severity: SeverityWarn,
		Summary:  "pg_stat_statements is not installed",
		Evidence: "no row in pg_extension for pg_stat_statements",
		Fix:      "Add pg_stat_statements to shared_preload_libraries, restart, then CREATE EXTENSION pg_stat_statements. Query text and per-statement timings stay unavailable until then.",
	}
}

// CacheHit reports a low shared-buffer hit ratio. hits and reads are the
// pg_stat_database counters for one database.
func CacheHit(database string, hits, reads int64) *Finding {
	total := hits + reads
	if total < MinBlocks {
		return nil
	}
	ratio := float64(hits) / float64(total)
	if ratio >= MinCacheHit {
		return nil
	}
	return &Finding{
		Check: "cache_hit", Severity: SeverityWarn,
		Summary:  fmt.Sprintf("%s cache hit ratio is %.1f%%", database, ratio*100),
		Evidence: fmt.Sprintf("blks_hit %d, blks_read %d", hits, reads),
		Fix:      "A low hit ratio means the cache is smaller than the working set, or the workload reads cold data. Check shared_buffers and the largest sequential scans before raising memory.",
	}
}

// IdleInTransaction reports sessions that have been idle in a transaction
// for at least IdleInTransactionWarn. age is how long the state has lasted.
func IdleInTransaction(pid int32, user, query string, age time.Duration) *Finding {
	if age < IdleInTransactionWarn {
		return nil
	}
	return &Finding{
		Check: "idle_in_transaction", Severity: SeverityWarn,
		Summary:  fmt.Sprintf("session %d (%s) has been idle in a transaction for %s", pid, user, age.Truncate(time.Second)),
		Evidence: query,
		Fix:      "Commit or roll back the transaction. A session left idle in a transaction holds back vacuum and can block DDL.",
	}
}

// Wraparound reports a database whose frozen-xid age is past WraparoundWarn.
func Wraparound(database string, xidAge int64) *Finding {
	if xidAge < WraparoundWarn {
		return nil
	}
	return &Finding{
		Check: "wraparound", Severity: SeverityWarn,
		Summary:  fmt.Sprintf("%s frozen-xid age is %d", database, xidAge),
		Evidence: fmt.Sprintf("age(datfrozenxid) = %d", xidAge),
		Fix:      "VACUUM that database, and check that autovacuum is not stuck. Wraparound shutdown is still a long way off at this age, but the margin should not keep shrinking.",
	}
}

// ReplicationLag reports a standby whose replay lag is at least the given
// minimum. Zero or negative lag is not a finding.
func ReplicationLag(application string, lag, minimum time.Duration) *Finding {
	if lag < minimum || minimum <= 0 {
		return nil
	}
	return &Finding{
		Check: "replication_lag", Severity: SeverityWarn,
		Summary:  fmt.Sprintf("%s replay lag is %s", application, lag.Truncate(time.Second)),
		Evidence: fmt.Sprintf("replay_lag %s, threshold %s", lag, minimum),
		Fix:      "Look at replay on the standby and at WAL shipping. A growing lag means the standby cannot keep up or is not receiving WAL.",
	}
}

// UnusedIndexes reports indexes that have never been scanned and are at
// least MinIndexBytes. Primary keys and unique indexes are not passed in
// by the caller when they must be kept.
func UnusedIndexes(indexes []Index) []Finding {
	var out []Finding
	for _, idx := range indexes {
		if idx.Scans != 0 || idx.Bytes < MinIndexBytes {
			continue
		}
		out = append(out, Finding{
			Check: "unused_index", Severity: SeverityInfo,
			Summary:  fmt.Sprintf("%s.%s has never been scanned", idx.Table, idx.Name),
			Evidence: fmt.Sprintf("idx_scan 0, size %d bytes", idx.Bytes),
			Fix:      "Confirm the index is unused over a full workload period, then DROP INDEX. Unused indexes still cost writes and vacuum.",
		})
	}
	return out
}

// DuplicateIndexes reports later indexes that repeat an earlier index's
// key and predicate on the same table.
func DuplicateIndexes(indexes []Index) []Finding {
	type key struct{ schema, table, cols, pred string }
	seen := map[key]string{}
	var out []Finding
	for _, idx := range indexes {
		k := key{idx.Schema, idx.Table, idx.Key, idx.Predicate}
		if first, ok := seen[k]; ok {
			out = append(out, Finding{
				Check: "duplicate_index", Severity: SeverityWarn,
				Summary:  fmt.Sprintf("%s duplicates %s on %s", idx.Name, first, idx.Table),
				Evidence: fmt.Sprintf("both index %s", idx.Key),
				Fix:      "Drop the duplicate once nothing depends on its name. Keep the index that constraints use.",
			})
			continue
		}
		seen[k] = idx.Name
	}
	return out
}

// Bloat reports a table whose dead tuples are a large share of the table.
func Bloat(schema, table string, live, dead int64) *Finding {
	if dead < MinDeadTuples {
		return nil
	}
	total := live + dead
	if total <= 0 {
		return nil
	}
	ratio := float64(dead) / float64(total)
	if ratio < MaxDeadRatio {
		return nil
	}
	return &Finding{
		Check: "bloat", Severity: SeverityInfo,
		Summary:  fmt.Sprintf("%s.%s is about %.0f%% dead tuples", schema, table, ratio*100),
		Evidence: fmt.Sprintf("n_live_tup %d, n_dead_tup %d", live, dead),
		Fix:      "VACUUM the table. If autovacuum is not keeping up, lower its scale factor for this table rather than vacuuming by hand forever.",
	}
}
