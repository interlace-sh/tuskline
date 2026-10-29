// SPDX-License-Identifier: Apache-2.0

package sqlpack

import (
	"errors"
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
