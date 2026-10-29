// SPDX-License-Identifier: Apache-2.0

package activity

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWaits(t *testing.T) {
	t.Parallel()
	got := Waits([]Session{
		{State: "active", WaitEventType: "Lock", WaitEvent: "relation"},
		{State: "active", WaitEventType: "Lock", WaitEvent: "relation"},
		{State: "active"},
		{State: "idle", WaitEventType: "Client", WaitEvent: "ClientRead"},
		{State: "idle in transaction", WaitEventType: "Client", WaitEvent: "ClientRead"},
	})
	want := []Wait{{"Lock", "relation", 2}, {"", "", 1}, {"Client", "ClientRead", 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSeconds(t *testing.T) {
	t.Parallel()
	f := func(v float64) *float64 { return &v }
	for _, tc := range []struct {
		in   *float64
		want time.Duration
	}{{nil, 0}, {f(-0.002), 0}, {f(math.NaN()), 0}, {f(1.5), 1500 * time.Millisecond}} {
		if got := seconds(tc.in); got != tc.want {
			t.Errorf("seconds(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// render prints a forest as "pid(child child)" for comparison.
func render(nodes []*Node) string {
	var parts []string
	for _, n := range nodes {
		s := itoa(n.PID)
		if n.Session == nil {
			s += "?"
		}
		if len(n.Blocked) > 0 {
			s += "(" + render(n.Blocked) + ")"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

func itoa(n int32) string { return strconv.Itoa(int(n)) }

func TestBlockingTree(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		sessions []Session
		want     string
	}{
		{"no blocking", []Session{{PID: 1}, {PID: 2}}, ""},
		{
			"chain and fan-out, largest first",
			[]Session{{PID: 10}, {PID: 11, BlockedBy: []int32{10}}, {PID: 12, BlockedBy: []int32{11}}, {PID: 13, BlockedBy: []int32{10}}, {PID: 20}, {PID: 21, BlockedBy: []int32{20}}},
			"10(11(12) 13) 20(21)",
		},
		{"two blockers", []Session{{PID: 1}, {PID: 2}, {PID: 3, BlockedBy: []int32{1, 2}}}, "1(3) 2(3)"},
		{"blocker not in snapshot", []Session{{PID: 5, BlockedBy: []int32{0}}}, "0?(5)"},
		{"cycle", []Session{{PID: 7, BlockedBy: []int32{8}}, {PID: 8, BlockedBy: []int32{7}}}, "7(8)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := render(BlockingTree(tc.sessions)); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
