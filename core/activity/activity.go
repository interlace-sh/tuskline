// SPDX-License-Identifier: Apache-2.0

package activity

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/sqlpack"
)

// Limits for one snapshot.
const (
	// Timeout bounds a whole snapshot.
	Timeout = 5 * time.Second
	// MaxQueryText is how much of each query's text is fetched.
	MaxQueryText = 2048
	// MaxRows caps sessions and locks, matching the prod grid row limit.
	MaxRows = 1000
)

// Querier runs a query; *pgx.Conn and pgxpool.Pool satisfy it.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Session is one backend from pg_stat_activity.
type Session struct {
	PID           int32  `json:"pid"`
	LeaderPID     int32  `json:"leader_pid,omitempty"` // parallel worker's leader
	User          string `json:"user"`
	Database      string `json:"database"`
	Application   string `json:"application"`
	ClientAddr    string `json:"client_addr"`
	BackendType   string `json:"backend_type"`
	State         string `json:"state"` // "" when hidden or not a client backend
	WaitEventType string `json:"wait_event_type"`
	WaitEvent     string `json:"wait_event"`
	// Ages are zero when the timestamp is unset or hidden.
	XactAge  time.Duration `json:"xact_age_ns"`
	QueryAge time.Duration `json:"query_age_ns"`
	StateAge time.Duration `json:"state_age_ns"`
	QueryID  int64         `json:"query_id,omitempty"`
	Query    string        `json:"query"`
	// BlockedBy lists the pids holding locks this session waits for; 0
	// stands for a prepared transaction.
	BlockedBy []int32 `json:"blocked_by,omitempty"`
}

// Waiting reports whether the session is waiting on anything.
func (s Session) Waiting() bool { return s.WaitEventType != "" }

// Wait counts non-idle sessions per wait event. An empty Type means the
// session was not waiting (running on CPU, or details hidden).
type Wait struct {
	Type     string `json:"type"`
	Event    string `json:"event"`
	Sessions int    `json:"sessions"`
}

// Lock is a waiting lock request or a granted lock on the same object.
type Lock struct {
	PID      int32         `json:"pid"`
	LockType string        `json:"locktype"`
	Mode     string        `json:"mode"`
	Granted  bool          `json:"granted"`
	Relation string        `json:"relation,omitempty"`
	Detail   string        `json:"detail,omitempty"`
	WaitAge  time.Duration `json:"wait_age_ns,omitempty"`
}

// Snapshot is one moment of server activity.
type Snapshot struct {
	At       time.Time `json:"at"`
	Sessions []Session `json:"sessions"`
	Waits    []Wait    `json:"waits"`
	Locks    []Lock    `json:"locks"`
}

// Take reads a snapshot from a server with the given server_version_num.
// It runs only the read-only activity pack, under Timeout.
func Take(ctx context.Context, q Querier, versionNum int) (Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	sq, err := sqlpack.Pick(versionNum, sqlpack.ActivitySessions)
	if err != nil {
		return Snapshot{}, err
	}
	lq, err := sqlpack.Pick(versionNum, sqlpack.ActivityLocks)
	if err != nil {
		return Snapshot{}, err
	}
	snap := Snapshot{At: time.Now().UTC()}
	if snap.Sessions, err = sessions(ctx, q, sq); err != nil {
		return Snapshot{}, err
	}
	if snap.Locks, err = locks(ctx, q, lq); err != nil {
		return Snapshot{}, err
	}
	snap.Waits = Waits(snap.Sessions)
	return snap, nil
}

func sessions(ctx context.Context, q Querier, query sqlpack.Query) ([]Session, error) {
	rows, err := q.Query(ctx, query.SQL, MaxQueryText, MaxRows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", query.Name, err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Session, error) {
		var (
			s                Session
			leader           *int32
			xact, qry, state *float64
			queryID          *int64
		)
		err := r.Scan(&s.PID, &leader, &s.User, &s.Database, &s.Application, &s.ClientAddr, &s.BackendType,
			&s.State, &s.WaitEventType, &s.WaitEvent, &xact, &qry, &state, &queryID, &s.Query, &s.BlockedBy)
		if leader != nil {
			s.LeaderPID = *leader
		}
		if queryID != nil {
			s.QueryID = *queryID
		}
		s.XactAge, s.QueryAge, s.StateAge = seconds(xact), seconds(qry), seconds(state)
		if len(s.BlockedBy) == 0 {
			s.BlockedBy = nil
		}
		return s, err
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", query.Name, err)
	}
	return out, nil
}

func locks(ctx context.Context, q Querier, query sqlpack.Query) ([]Lock, error) {
	rows, err := q.Query(ctx, query.SQL, MaxRows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", query.Name, err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Lock, error) {
		var (
			l    Lock
			wait *float64
		)
		err := r.Scan(&l.PID, &l.LockType, &l.Mode, &l.Granted, &l.Relation, &l.Detail, &wait)
		l.WaitAge = seconds(wait)
		return l, err
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", query.Name, err)
	}
	return out, nil
}

// seconds converts an age in seconds to a duration. NULL and the small
// negative ages from clock skew between now() and a later start become 0.
func seconds(v *float64) time.Duration {
	if v == nil || *v <= 0 || math.IsNaN(*v) {
		return 0
	}
	return time.Duration(*v * float64(time.Second))
}

// Waits summarises non-idle sessions by wait event, most common first.
func Waits(sessions []Session) []Wait {
	counts := map[[2]string]int{}
	for _, s := range sessions {
		if s.State == "idle" {
			continue
		}
		counts[[2]string{s.WaitEventType, s.WaitEvent}]++
	}
	out := make([]Wait, 0, len(counts))
	for k, n := range counts {
		out = append(out, Wait{Type: k[0], Event: k[1], Sessions: n})
	}
	slices.SortFunc(out, func(a, b Wait) int {
		return cmp.Or(cmp.Compare(b.Sessions, a.Sessions), cmp.Compare(a.Type, b.Type), cmp.Compare(a.Event, b.Event))
	})
	return out
}
