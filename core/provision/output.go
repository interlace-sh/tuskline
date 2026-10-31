// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"fmt"
	"os"
	"path/filepath"
)

// composeFiles are the names a Compose output directory may already hold.
// Anything else means the directory is not one of ours.
var composeFiles = map[string]bool{
	"compose.yaml":          true,
	"haproxy.cfg":           true,
	"pgbackrest.conf":       true,
	".env":                  true,
	".tuskline-apply.jsonl": true,
	"pgbouncer":             true,
}

// CheckOutputDir allows a missing directory, an empty one, or one that only
// holds files from an earlier Compose apply. It refuses a directory that
// already has something else in it.
func CheckOutputDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("output directory is required")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("output directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !composeFiles[name] {
			return fmt.Errorf("output directory already contains %s", name)
		}
		if name != "pgbouncer" {
			if entry.IsDir() {
				return fmt.Errorf("output directory already contains directory %s", name)
			}
			continue
		}
		if !entry.IsDir() {
			return fmt.Errorf("pgbouncer is not a directory")
		}
		nested, err := os.ReadDir(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("output directory: %w", err)
		}
		for _, child := range nested {
			if child.Name() != "userlist.txt" || child.IsDir() {
				return fmt.Errorf("output directory already contains pgbouncer/%s", child.Name())
			}
		}
	}
	return nil
}
