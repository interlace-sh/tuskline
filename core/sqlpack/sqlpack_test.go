// SPDX-License-Identifier: Apache-2.0

package sqlpack

import (
	"errors"
	"strings"
	"testing"
)

func TestPick(t *testing.T) {
	t.Parallel()
	old := Query{Name: "q", MinVersion: PG14, MaxVersion: 170000, SQL: "old"}
	cur := Query{Name: "q", MinVersion: 170000, SQL: "new"}
	cases := []struct {
		v    int
		want string
	}{
		{140000, "old"},
		{160009, "old"},
		{170000, "new"},
		{180001, "new"},
	}
	for _, c := range cases {
		got, err := Pick(c.v, old, cur)
		if err != nil || got.SQL != c.want {
			t.Errorf("Pick(%d) = %q, %v; want %q", c.v, got.SQL, err, c.want)
		}
	}
	if _, err := Pick(130000, old, cur); !errors.Is(err, ErrNoVariant) {
		t.Errorf("Pick(130000) err = %v, want ErrNoVariant", err)
	}
}

func TestStatsPickFollowsTheViewChanges(t *testing.T) {
	t.Parallel()
	statements := []Query{StatsStatements14, StatsStatements15, StatsStatements17}
	tables := []Query{StatsTables14, StatsTables16, StatsTables18}
	for _, tc := range []struct {
		v    int
		sql  string
		pack []Query
	}{
		{PG14, "blk_read_time AS shared_blk_read_time", statements},
		{PG15, "temp_blk_read_time", statements},
		{160004, "temp_blk_read_time", statements},
		{PG17, "\tshared_blk_read_time,", statements},
		{180001, "\tshared_blk_read_time,", statements},
		{PG14, "seq_scan,\n\tseq_tup_read", tables},
		{PG16, "n_tup_newpage_upd", tables},
		{PG18, "total_vacuum_time", tables},
	} {
		got, err := Pick(tc.v, tc.pack...)
		if err != nil || !strings.Contains(got.SQL, tc.sql) {
			t.Errorf("Pick(%d) err %v, sql has %q? %.80s", tc.v, err, tc.sql, got.SQL)
		}
		if tc.v < PG15 && strings.Contains(got.SQL, "temp_blk_read_time") {
			t.Errorf("Pick(%d) included temporary block time", tc.v)
		}
		if tc.v < PG17 && strings.Contains(got.SQL, "\tshared_blk_read_time,") {
			t.Errorf("Pick(%d) used the PostgreSQL 17 column names", tc.v)
		}
	}
	if _, err := Pick(PG14, StatsIO16, StatsIO18); !errors.Is(err, ErrNoVariant) {
		t.Errorf("io on 14: %v", err)
	}
	if _, err := Pick(PG14, StatsCheckpointer17, StatsCheckpointer18); !errors.Is(err, ErrNoVariant) {
		t.Errorf("checkpointer on 14: %v", err)
	}
	if q, err := Pick(PG18, StatsWAL14, StatsWAL18); err != nil || strings.Contains(q.SQL, "wal_write") {
		t.Errorf("wal 18 = %v %.40q", err, q.SQL)
	}
}

// Every query in the pack must be documented and runnable on 14+.
func TestPackQueriesAreDocumented(t *testing.T) {
	t.Parallel()
	for _, q := range All() {
		if q.Name == "" || q.Doc == "" || len(q.Sources) == 0 || q.SQL == "" {
			t.Errorf("%q: name, doc, sources and SQL are required", q.Name)
		}
		if q.MinVersion < PG14 {
			t.Errorf("%s: MinVersion %d is below PostgreSQL 14", q.Name, q.MinVersion)
		}
	}
}
