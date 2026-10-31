// SPDX-License-Identifier: Apache-2.0

//go:build linux || darwin

package provision

import "testing"

func TestFreeBytes(t *testing.T) {
	t.Parallel()
	n, err := FreeBytes(t.TempDir())
	if err != nil || n <= 0 {
		t.Fatalf("free %d %v", n, err)
	}
}
