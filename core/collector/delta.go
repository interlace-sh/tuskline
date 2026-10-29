// SPDX-License-Identifier: Apache-2.0

package collector

import "math"

// Delta is how much a monotonic counter grew between two samples. reset is
// true when next is lower than prev, which is what a statistics reset or a
// wrap looks like: the drop is not counted as negative work, and delta is
// the value accumulated since the reset. A negative next is treated as no
// progress.
func Delta(prev, next int64) (delta int64, reset bool) {
	if next < 0 {
		return 0, true
	}
	if next < prev {
		return next, true
	}
	return next - prev, false
}

// DeltaFloat is Delta for double-precision counters such as total_exec_time.
// A NaN sample is treated as a reset with no progress.
func DeltaFloat(prev, next float64) (delta float64, reset bool) {
	if math.IsNaN(next) || math.IsNaN(prev) || next < 0 {
		return 0, true
	}
	if next < prev {
		return next, true
	}
	return next - prev, false
}
