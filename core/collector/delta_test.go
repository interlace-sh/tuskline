// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"math"
	"testing"
)

func TestDelta(t *testing.T) {
	t.Parallel()
	cases := []struct {
		prev, next int64
		want       int64
		reset      bool
	}{
		{0, 0, 0, false},
		{10, 15, 5, false},
		{10, 10, 0, false},
		{100, 4, 4, true},
		{5, 0, 0, true},
		{5, -1, 0, true},
	}
	for _, tc := range cases {
		got, reset := Delta(tc.prev, tc.next)
		if got != tc.want || reset != tc.reset {
			t.Errorf("Delta(%d, %d) = %d, %v; want %d, %v", tc.prev, tc.next, got, reset, tc.want, tc.reset)
		}
	}
}

func TestDeltaFloat(t *testing.T) {
	t.Parallel()
	got, reset := DeltaFloat(1.5, 4)
	if got != 2.5 || reset {
		t.Fatalf("growth %v %v", got, reset)
	}
	got, reset = DeltaFloat(10, 1.25)
	if got != 1.25 || !reset {
		t.Fatalf("reset %v %v", got, reset)
	}
	if got, reset = DeltaFloat(1, math.NaN()); got != 0 || !reset {
		t.Fatalf("nan %v %v", got, reset)
	}
}
