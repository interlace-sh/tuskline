// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/interlace-sh/tuskline/core/activity"
	"github.com/interlace-sh/tuskline/core/inventory"
)

func topSnapshot() activity.Snapshot {
	ss := []activity.Session{
		{PID: 100, User: "app", Database: "orders", State: "idle in transaction", XactAge: 75 * time.Second, QueryAge: 70 * time.Second, Query: "LOCK TABLE t\n  IN ACCESS EXCLUSIVE MODE"},
		{PID: 101, User: "app", Database: "orders", State: "active", WaitEventType: "Lock", WaitEvent: "relation", QueryAge: 3 * time.Second, Query: "SELECT count(*) FROM t", BlockedBy: []int32{100}},
		{PID: 102, User: "report", Database: "orders", State: "active", QueryAge: 12 * time.Minute, Query: "SELECT * FROM big"},
		{PID: 103, User: "app", Database: "orders", State: "idle", QueryAge: time.Hour, Query: "COMMIT"},
		{PID: 50, BackendType: "autovacuum launcher"},
	}
	return activity.Snapshot{
		At:       time.Date(2026, 10, 29, 12, 0, 1, 0, time.UTC),
		Sessions: ss,
		Waits:    activity.Waits(ss),
		Locks: []activity.Lock{
			{PID: 101, LockType: "relation", Mode: "AccessShareLock", Relation: "t", WaitAge: 3 * time.Second},
			{PID: 100, LockType: "relation", Mode: "AccessExclusiveLock", Granted: true, Relation: "t"},
		},
	}
}

func TestTopRender(t *testing.T) {
	v := topView{ref: "prod/orders", kind: inventory.EnvProd, versionNum: 170006, snap: topSnapshot()}
	out := v.render(&app{})
	for _, want := range []string{
		"tl top prod/orders · PostgreSQL 17.6 · 12:00:01 UTC · 5 sessions, 2 active · sort: query age",
		"waits: not waiting 3 · Lock/relation 1",
		"blocking:\n  100  app  idle in transaction  xact 1m15s  LOCK TABLE t IN ACCESS EXCLUSIVE MODE\n  └─ 101  app  active  SELECT count(*) FROM t\n",
		"lock waits:\n  101 waits AccessShareLock relation on t for 3.0s\n",
		"PID  USER    DATABASE  APP  STATE",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render lacks %q:\n%s", want, out)
		}
	}
	// Busiest first by query age; idle last.
	table := out[strings.Index(out, "PID  USER"):]
	order := []string{"102  report", "100  app", "101  app", "50 ", "103  app"}
	last := -1
	for _, s := range order {
		i := strings.Index(table, s)
		if i <= last {
			t.Fatalf("row %q out of order:\n%s", s, out)
		}
		last = i
	}
	if strings.Contains(out, "q quit") {
		t.Error("--once output has the key help")
	}
}

func TestTopRenderFitsTheScreen(t *testing.T) {
	v := topView{ref: "prod/orders", kind: inventory.EnvProd, versionNum: 170006, snap: topSnapshot(), width: 40, height: 12, interactive: true}
	out := strings.TrimSuffix(v.render(&app{colour: true}), "\n")
	lines := strings.Split(out, "\n")
	if len(lines) > v.height {
		t.Errorf("%d lines for height %d:\n%s", len(lines), v.height, out)
	}
	for _, l := range lines {
		if n := utf8.RuneCountInString(stripANSI(l)); n > v.width {
			t.Errorf("line of %d runes for width %d: %q", n, v.width, l)
		}
	}
	if !strings.Contains(out, "more sessions") || !strings.HasPrefix(lines[len(lines)-1], "q quit") {
		t.Errorf("no overflow marker or footer:\n%s", out)
	}
	if !strings.Contains(lines[0], "\x1b[1;38;2;229;72;77mprod/orders\x1b[0m") {
		t.Errorf("title not coloured: %q", lines[0])
	}
}

func TestTopFilterAndSort(t *testing.T) {
	v := topView{snap: topSnapshot(), filter: "REPORT"}
	if got := v.visible(); len(got) != 1 || got[0].PID != 102 {
		t.Fatalf("filter: %+v", got)
	}
	v = topView{snap: topSnapshot(), sort: sortPID}
	got := v.visible()
	if got[0].PID != 50 || got[len(got)-1].PID != 103 {
		t.Fatalf("pid sort with idle last: %+v", got)
	}
}

type fakeSource struct{}

func (fakeSource) take(context.Context) (activity.Snapshot, error) { return topSnapshot(), nil }

func press(m *topModel, keys ...tea.KeyPressMsg) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range keys {
		_, cmd = m.Update(k)
	}
	return cmd
}

func char(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestTopKeys(t *testing.T) {
	m := &topModel{app: &app{}, view: topView{ref: "dev/pg17"}, interval: time.Second, ctx: context.Background(), source: fakeSource{}}
	press(m, char('/'), char('a'), char('q'), char('x'), tea.KeyPressMsg{Code: tea.KeyBackspace})
	if !m.view.filtering || m.view.filter != "aq" {
		t.Fatalf("filtering %v %q: q while filtering must type, not quit", m.view.filtering, m.view.filter)
	}
	press(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.view.filtering || m.view.filter != "aq" {
		t.Fatalf("enter: %v %q", m.view.filtering, m.view.filter)
	}
	press(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.view.filter != "" {
		t.Fatal("esc did not clear the filter")
	}
	press(m, char('s'), char('s'))
	if m.view.sort != sortPID {
		t.Fatalf("sort = %v", m.view.sort)
	}
	press(m, char('s'), char('s'))
	if m.view.sort != sortQueryAge {
		t.Fatalf("sort did not wrap: %v", m.view.sort)
	}
	if !isQuit(press(m, char('q'))) {
		t.Fatal("q did not quit")
	}
	m.view.filtering = true
	if !isQuit(press(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})) {
		t.Fatal("ctrl+c did not quit while filtering")
	}
}

func TestTopRefreshKeepsLastSnapshotOnError(t *testing.T) {
	m := &topModel{app: &app{}, view: topView{ref: "dev/pg17"}, interval: time.Second, ctx: context.Background(), source: fakeSource{}}
	m.Update(m.fetch())
	if !m.loaded || len(m.view.snap.Sessions) != 5 {
		t.Fatal("first snapshot not shown")
	}
	m.Update(snapMsg{err: errors.New("connection reset")})
	if len(m.view.snap.Sessions) != 5 || m.view.err == nil || m.failures != 1 {
		t.Fatal("error replaced the snapshot or was not shown")
	}
	if !strings.Contains(m.View().Content, "refresh failed") {
		t.Errorf("view hides the error:\n%s", m.View().Content)
	}
}

func TestTopBackoff(t *testing.T) {
	m := &topModel{interval: time.Second}
	cases := []struct {
		failures int
		took     time.Duration
		want     time.Duration
	}{
		{0, 10 * time.Millisecond, time.Second},
		{0, time.Second, 4 * time.Second},
		{2, 0, 4 * time.Second},
		{10, 0, maxTopBackoff},
	}
	for _, tc := range cases {
		m.failures = tc.failures
		if got := m.nextDelay(tc.took); got != tc.want {
			t.Errorf("failures %d took %v: %v, want %v", tc.failures, tc.took, got, tc.want)
		}
	}
}

func TestTopArguments(t *testing.T) {
	path := seedInventory(t)
	cases := []struct {
		a    *app
		args []string
		want string
	}{
		{&app{}, []string{"top", "prod/orders", "-o", "json"}, "needs --once"},
		{&app{}, []string{"top", "prod/orders", "--once", "--interval", "100ms"}, "minimum is 500ms"},
		{&app{}, []string{"top", "prod/orders"}, "needs a terminal"},
		{&app{stdoutTTY: true}, []string{"top", "prod/orders"}, "not in the inventory"},
		{&app{}, []string{"top", "orders"}, "env/cluster"},
	}
	for _, tc := range cases {
		_, _, err := runApp(t, tc.a, append(tc.args, "--inventory", path)...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: got %v, want %q", tc.args, err, tc.want)
		}
	}
}

func TestDur(t *testing.T) {
	for d, want := range map[time.Duration]string{0: "", 400 * time.Millisecond: "0.4s", 12 * time.Second: "12s", 184 * time.Second: "3m04s", 123 * time.Minute: "2h03m", 76 * time.Hour: "3d04h"} {
		if got := dur(d); got != want {
			t.Errorf("dur(%v) = %q, want %q", d, got, want)
		}
	}
}
