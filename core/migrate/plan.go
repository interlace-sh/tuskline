// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"errors"
	"fmt"
)

// ErrChecksumChanged is returned when an applied script no longer matches
// the copy that was recorded.
var ErrChecksumChanged = errors.New("applied migration was edited")

// ErrOutOfOrder is returned when a new script sorts before one that is
// already applied.
var ErrOutOfOrder = errors.New("migration is ordered before one that is already applied")

// ErrMissingScript is returned when the history names a script the folder
// no longer contains.
var ErrMissingScript = errors.New("applied migration is missing from the folder")

// ErrPreviousFailure is returned when a script was recorded as failed and
// has not been repaired.
var ErrPreviousFailure = errors.New("a previous migration failed")

// Applied is one row from tuskline.schema_migration.
type Applied struct {
	Name     string
	Checksum string
	Success  bool
}

// Pending returns the scripts still to run, in folder order. It refuses a
// folder that has diverged from the history.
func Pending(files []File, applied []Applied) ([]File, error) {
	byName := make(map[string]Applied, len(applied))
	for _, row := range applied {
		byName[row.Name] = row
	}
	have := make(map[string]bool, len(files))
	for _, file := range files {
		have[file.Name] = true
	}
	for _, row := range applied {
		if row.Success && !have[row.Name] {
			return nil, fmt.Errorf("%w: %s", ErrMissingScript, row.Name)
		}
	}
	for _, file := range files {
		row, ok := byName[file.Name]
		if !ok {
			continue
		}
		if !row.Success {
			return nil, fmt.Errorf("%w: %s", ErrPreviousFailure, file.Name)
		}
		if row.Checksum != file.Checksum {
			return nil, fmt.Errorf("%w: %s", ErrChecksumChanged, file.Name)
		}
	}
	var pending []File
	for i, file := range files {
		if row, ok := byName[file.Name]; ok && row.Success {
			continue
		}
		for _, later := range files[i+1:] {
			if row, ok := byName[later.Name]; ok && row.Success {
				return nil, fmt.Errorf("%w: %s is ordered before applied %s", ErrOutOfOrder, file.Name, later.Name)
			}
		}
		pending = append(pending, file)
	}
	return pending, nil
}
