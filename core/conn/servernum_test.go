// SPDX-License-Identifier: Apache-2.0

package conn

import "testing"

func TestParseServerVersion(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]int{
		"17.6":                      170006,
		"18beta1":                   180000,
		"16.4 (Debian 16.4-1.pgdg)": 160004,
		"14.12":                     140012,
		"15":                        150000,
	} {
		if got, err := parseServerVersion(in); err != nil || got != want {
			t.Errorf("parseServerVersion(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "beta", ".5"} {
		if _, err := parseServerVersion(bad); err == nil {
			t.Errorf("parseServerVersion(%q): no error", bad)
		}
	}
}
