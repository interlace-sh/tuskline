// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultPath is $TUSKLINE_STORE, or $XDG_DATA_HOME/tuskline/meta.db, or
// ~/.local/share/tuskline/meta.db.
func DefaultPath() (string, error) {
	if path := os.Getenv("TUSKLINE_STORE"); path != "" {
		return path, nil
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("store: home directory: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "tuskline", "meta.db"), nil
}
