// SPDX-License-Identifier: Apache-2.0

//go:build linux

package provision

import (
	"fmt"
	"syscall"
)

// FreeBytes is the free space available to the current user on the
// filesystem that holds path.
func FreeBytes(path string) (int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, fmt.Errorf("disk space: %w", err)
	}
	return int64(st.Bavail) * st.Bsize, nil
}
