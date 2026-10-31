// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !darwin

package provision

import "fmt"

// FreeBytes is not implemented on this OS.
func FreeBytes(string) (int64, error) {
	return 0, fmt.Errorf("disk space check is not available on this OS")
}
