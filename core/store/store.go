// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver" // registers the pure-Go "sqlite3" driver
)

// Store is the metadata database. It is safe for concurrent use.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens the database at path, creating it and its parent directory if
// needed, and applies any pending migrations. The file is mode 0600 and a
// directory this function creates is mode 0700.
func Open(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store: path is empty")
	}
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("store: create %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return nil, fmt.Errorf("store: chmod %s: %w", dir, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("store: %s: %w", dir, err)
	}
	db, err := sql.Open("sqlite3", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	// One writer. Reads can queue behind a short busy timeout.
	db.SetMaxOpenConns(1)
	s := &Store{db: db, path: path}
	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: chmod %s: %w", path, err)
	}
	return s, nil
}

func dsn(path string) string {
	q := url.Values{}
	q.Set("_txlock", "immediate")
	q.Set("_timefmt", "rfc3339")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	return (&url.URL{Scheme: "file", Path: path, RawQuery: q.Encode()}).String()
}

// Close closes the database.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("store: close: %w", err)
	}
	return nil
}

// Path is the database file.
func (s *Store) Path() string { return s.path }

// Version is the latest applied migration, or 0 when there are none.
func (s *Store) Version(ctx context.Context) (int, error) {
	var v int
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migration`).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("store: version: %w", err)
	}
	return v, nil
}

// Cursor is when metric was last sampled for target. ok is false when no
// sample has been recorded.
func (s *Store) Cursor(ctx context.Context, target, metric string) (at time.Time, ok bool, err error) {
	if err := names(target, metric); err != nil {
		return time.Time{}, false, err
	}
	var raw string
	err = s.db.QueryRowContext(ctx, `SELECT sampled_at FROM collection_cursor WHERE target = ? AND metric = ?`, target, metric).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("store: cursor %s %s: %w", target, metric, err)
	}
	at, err = time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("store: cursor %s %s: %w", target, metric, err)
	}
	return at, true, nil
}

// SetCursor records when metric was last sampled for target.
func (s *Store) SetCursor(ctx context.Context, target, metric string, at time.Time) error {
	if err := names(target, metric); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("store: sample time is zero")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO collection_cursor (target, metric, sampled_at) VALUES (?, ?, ?)
		ON CONFLICT (target, metric) DO UPDATE SET sampled_at = excluded.sampled_at`,
		target, metric, at.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: set cursor %s %s: %w", target, metric, err)
	}
	return nil
}

func names(target, metric string) error {
	if target == "" || metric == "" {
		return errors.New("store: target and metric are required")
	}
	return nil
}
