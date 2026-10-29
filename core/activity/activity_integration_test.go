// SPDX-License-Identifier: Apache-2.0

//go:build integration

package activity_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/activity"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/pgtest"
)

// prodLike is the session a prod observer gets: read-only with timeouts.
var prodLike = conn.SessionSettings{ReadOnly: true, StatementTimeout: 30 * time.Second, LockTimeout: 2 * time.Second}

func connect(t *testing.T, p conn.Profile, s conn.SessionSettings) *pgx.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := conn.Connect(ctx, p, s)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c
}

func pid(t *testing.T, c *pgx.Conn) int32 {
	t.Helper()
	var n int32
	if err := c.QueryRow(context.Background(), "SELECT pg_backend_pid()").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func find(ss []activity.Session, pid int32) (activity.Session, bool) {
	i := slices.IndexFunc(ss, func(s activity.Session) bool { return s.PID == pid })
	if i < 0 {
		return activity.Session{}, false
	}
	return ss[i], true
}

func TestSnapshotShowsBlocking(t *testing.T) {
	t.Parallel()
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			t.Parallel()
			super := pgtest.Start(t, major).Profile
			pgtest.Exec(t, super,
				"CREATE TABLE t (id int PRIMARY KEY)",
				"INSERT INTO t VALUES (1)",
				"CREATE ROLE mon LOGIN PASSWORD 'mon-test-only' IN ROLE pg_monitor",
				"CREATE ROLE plain LOGIN PASSWORD 'plain-test-only'",
			)

			holder := connect(t, super, conn.SessionSettings{})
			holderPID := pid(t, holder)
			ctx := context.Background()
			tx, err := holder.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, "LOCK TABLE t IN ACCESS EXCLUSIVE MODE"); err != nil {
				t.Fatal(err)
			}

			waiter := connect(t, super, conn.SessionSettings{})
			waiterPID := pid(t, waiter)
			done := make(chan error, 1)
			go func() {
				wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				_, err := waiter.Exec(wctx, "SELECT count(*) FROM t")
				done <- err
			}()

			observer := connect(t, pgtest.As(super, "mon", "mon-test-only"), prodLike)
			observerPID := pid(t, observer)
			var snap activity.Snapshot
			deadline := time.Now().Add(15 * time.Second)
			for {
				if snap, err = activity.Take(ctx, observer, major*10000); err != nil {
					t.Fatal(err)
				}
				if w, ok := find(snap.Sessions, waiterPID); ok && slices.Contains(w.BlockedBy, holderPID) {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("waiter never showed as blocked: %+v", snap.Sessions)
				}
				<-time.After(50 * time.Millisecond)
			}

			if _, ok := find(snap.Sessions, observerPID); ok {
				t.Error("snapshot includes the observer's own backend")
			}
			w, _ := find(snap.Sessions, waiterPID)
			if w.State != "active" || w.WaitEventType != "Lock" || w.WaitEvent != "relation" || !strings.Contains(w.Query, "count(*) FROM t") || w.QueryAge <= 0 {
				t.Errorf("waiter: %+v", w)
			}
			h, ok := find(snap.Sessions, holderPID)
			if !ok || h.State != "idle in transaction" || !strings.Contains(h.Query, "LOCK TABLE") || h.XactAge <= 0 {
				t.Errorf("holder: %+v (found %v)", h, ok)
			}

			var waiting, holding bool
			for _, l := range snap.Locks {
				switch {
				case l.PID == waiterPID && !l.Granted && l.Mode == "AccessShareLock" && l.Relation == "t":
					waiting = true
				case l.PID == holderPID && l.Granted && l.Mode == "AccessExclusiveLock" && l.Relation == "t":
					holding = true
				}
			}
			if !waiting || !holding {
				t.Errorf("locks lack the wait (%v) or the holder (%v): %+v", waiting, holding, snap.Locks)
			}

			tree := activity.BlockingTree(snap.Sessions)
			if len(tree) != 1 || tree[0].PID != holderPID || len(tree[0].Blocked) != 1 || tree[0].Blocked[0].PID != waiterPID {
				t.Errorf("blocking tree: %+v", tree)
			}
			if !slices.ContainsFunc(snap.Waits, func(x activity.Wait) bool {
				return x.Type == "Lock" && x.Event == "relation" && x.Sessions >= 1
			}) {
				t.Errorf("waits: %+v", snap.Waits)
			}

			// A role without pg_monitor still gets a snapshot, with other
			// users' details hidden by the server.
			plain := connect(t, pgtest.As(super, "plain", "plain-test-only"), prodLike)
			psnap, err := activity.Take(ctx, plain, major*10000)
			if err != nil {
				t.Fatalf("plain role: %v", err)
			}
			if ph, ok := find(psnap.Sessions, holderPID); ok && strings.Contains(ph.Query, "LOCK TABLE") {
				t.Errorf("plain role sees another user's query: %+v", ph)
			}

			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatalf("waiter: %v", err)
			}
		})
	}
}

func TestTakeRefusesOldServers(t *testing.T) {
	t.Parallel()
	if _, err := activity.Take(context.Background(), nil, 130000); err == nil {
		t.Fatal("no error for PostgreSQL 13")
	}
}
