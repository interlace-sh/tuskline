// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestOpenMigratesAndRemembersTheCursor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "meta", "tuskline.sqlite")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.Version(ctx)
	if err != nil || v != 1 {
		t.Fatalf("version %d, %v", v, err)
	}
	if _, ok, err := s.Cursor(ctx, "prod/orders", "stats.database"); err != nil || ok {
		t.Fatalf("missing cursor ok %v err %v", ok, err)
	}
	at := time.Date(2026, 10, 29, 11, 0, 0, 123, time.UTC)
	if err := s.SetCursor(ctx, "prod/orders", "stats.database", at); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	got, ok, err := s.Cursor(ctx, "prod/orders", "stats.database")
	if err != nil || !ok || !got.Equal(at) {
		t.Fatalf("cursor %v ok %v err %v", got, ok, err)
	}
	later := at.Add(time.Minute)
	if err := s.SetCursor(ctx, "prod/orders", "stats.database", later); err != nil {
		t.Fatal(err)
	}
	got, _, err = s.Cursor(ctx, "prod/orders", "stats.database")
	if err != nil || !got.Equal(later) {
		t.Fatalf("updated cursor %v err %v", got, err)
	}
	if v, err = s.Version(ctx); err != nil || v != 1 {
		t.Fatalf("version after reopen %d, %v", v, err)
	}
}

func TestOpenFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes")
	}
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "private")
	path := filepath.Join(dir, "tuskline.sqlite")
	s, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{dir, path} {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		want := os.FileMode(0o600)
		if p == dir {
			want = 0o700
		}
		if st.Mode().Perm() != want {
			t.Errorf("%s mode %o, want %o", p, st.Mode().Perm(), want)
		}
	}
}

func TestOpenRejectsBadInput(t *testing.T) {
	t.Parallel()
	if _, err := Open(context.Background(), ""); err == nil {
		t.Fatal("empty path")
	}
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	if err := s.SetCursor(ctx, "", "m", time.Now()); err == nil {
		t.Fatal("empty target")
	}
	if err := s.SetCursor(ctx, "t", "m", time.Time{}); err == nil {
		t.Fatal("zero time")
	}
}
