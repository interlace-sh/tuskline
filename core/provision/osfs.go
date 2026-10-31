// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// OSFS reads and writes a directory on the local machine. Paths in the
// plan are relative to Root.
type OSFS struct {
	Root string
}

func (f OSFS) full(p string) (string, error) {
	clean, err := cleanPath(p)
	if err != nil {
		return "", err
	}
	return filepath.Join(f.Root, filepath.FromSlash(clean)), nil
}

// Read returns the file body. ok is false when the file is absent.
func (f OSFS) Read(ctx context.Context, p string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	full, err := f.full(p)
	if err != nil {
		return "", false, err
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(raw), true, nil
}

// Write replaces the file, creating parent directories.
func (f OSFS) Write(ctx context.Context, p, content string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	full, err := f.full(p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0o644)
}

// NonEmpty reports whether p is an existing directory that already has entries.
func (f OSFS) NonEmpty(ctx context.Context, p string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	full, err := f.full(p)
	if err != nil {
		return false, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read %s: %w", p, err)
	}
	return len(entries) > 0, nil
}
