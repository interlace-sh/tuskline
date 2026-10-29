// SPDX-License-Identifier: Apache-2.0

package conn

import "testing"

func TestCheckSupported(t *testing.T) {
	for v, ok := range map[int]bool{130015: false, 140000: true, 170004: true, 180000: true} {
		if err := CheckSupported(v); (err == nil) != ok {
			t.Errorf("CheckSupported(%d) err = %v, want ok=%v", v, err, ok)
		}
	}
}

func TestMajor(t *testing.T) {
	if got := (ServerInfo{VersionNum: 160003}).Major(); got != 16 {
		t.Fatalf("Major() = %d, want 16", got)
	}
}
