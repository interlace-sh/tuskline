// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileJournal appends one JSON object per completed step. The file mode is
// 0600. It stores ids and hashes, not step bodies.
type FileJournal struct {
	Path string
}

type journalLine struct {
	ID   string `json:"id"`
	Hash string `json:"hash"`
}

// Load reads the journal. A missing file is an empty journal.
func (j FileJournal) Load(ctx context.Context) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(j.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var line journalLine
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("journal %s: %w", j.Path, err)
		}
		if line.ID != "" {
			out[line.ID] = line.Hash
		}
	}
	if err := sc.Err(); err != nil {
		_ = f.Close()
		return nil, err
	}
	return out, f.Close()
}

// Append records one completed step.
func (j FileJournal) Append(ctx context.Context, id, hash string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(j.Path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(j.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(journalLine{ID: id, Hash: hash})
	if err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(append(raw, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
