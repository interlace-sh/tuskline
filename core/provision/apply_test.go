// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type memFS struct {
	files map[string]string
	fail  string
}

func newMem() *memFS { return &memFS{files: map[string]string{}} }

func (m *memFS) Read(context.Context, string) (string, bool, error) { return "", false, nil }

func (m *memFS) Write(_ context.Context, path, content string) error {
	if path == m.fail {
		return errors.New("disk full")
	}
	m.files[path] = content
	return nil
}

func (m *memFS) NonEmpty(_ context.Context, path string) (bool, error) {
	if _, ok := m.files[path]; ok && m.files[path] != "" {
		return true, nil
	}
	prefix := path + "/"
	for name := range m.files {
		if strings.HasPrefix(name, prefix) {
			return true, nil
		}
	}
	return false, nil
}

type memJournal struct {
	done map[string]string
}

func (m *memJournal) Load(context.Context) (map[string]string, error) {
	if m.done == nil {
		m.done = map[string]string{}
	}
	out := map[string]string{}
	for id, hash := range m.done {
		out[id] = hash
	}
	return out, nil
}

func (m *memJournal) Append(_ context.Context, id, hash string) error {
	if m.done == nil {
		m.done = map[string]string{}
	}
	m.done[id] = hash
	return nil
}

func TestApplyWritesThenSkipsOnResume(t *testing.T) {
	t.Parallel()
	plan := Plan{Steps: []Step{
		{ID: "a", Kind: KindFile, Path: "a.yml", After: "one\n"},
		{ID: "b", Kind: KindUnit, Path: "b.service", After: "two\n"},
	}}
	fs := newMem()
	journal := &memJournal{}
	var log []Event
	if err := Apply(context.Background(), plan, journal, fs, nil, func(ev Event) { log = append(log, ev) }); err != nil {
		t.Fatal(err)
	}
	if fs.files["a.yml"] != "one\n" || fs.files["b.service"] != "two\n" {
		t.Fatalf("files: %#v", fs.files)
	}
	log = nil
	if err := Apply(context.Background(), plan, journal, fs, nil, func(ev Event) { log = append(log, ev) }); err != nil {
		t.Fatal(err)
	}
	if len(log) != 2 || log[0].Status != "skip" || log[1].Status != "skip" {
		t.Fatalf("resume log: %+v", log)
	}
}

func TestApplyResumesAfterAFailure(t *testing.T) {
	t.Parallel()
	plan := Plan{Steps: []Step{
		{ID: "a", Kind: KindFile, Path: "a.yml", After: "one\n"},
		{ID: "b", Kind: KindFile, Path: "b.yml", After: "two\n"},
		{ID: "c", Kind: KindFile, Path: "c.yml", After: "three\n"},
	}}
	fs := newMem()
	fs.fail = "b.yml"
	journal := &memJournal{}
	err := Apply(context.Background(), plan, journal, fs, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "b") {
		t.Fatalf("got %v", err)
	}
	if _, ok := journal.done["a"]; !ok {
		t.Fatal("completed step was not recorded")
	}
	if _, ok := journal.done["b"]; ok {
		t.Fatal("failed step was recorded")
	}
	fs.fail = ""
	if err := Apply(context.Background(), plan, journal, fs, nil, nil); err != nil {
		t.Fatal(err)
	}
	if fs.files["a.yml"] != "one\n" || fs.files["b.yml"] != "two\n" || fs.files["c.yml"] != "three\n" {
		t.Fatalf("files: %#v", fs.files)
	}
}

func TestApplyRefusesANonEmptyDataDir(t *testing.T) {
	t.Parallel()
	fs := newMem()
	fs.files["pgdata/PG_VERSION"] = "17\n"
	plan := Plan{Steps: []Step{{ID: "data", Kind: KindFile, Path: "pgdata", After: "", DataDir: true}}}
	err := Apply(context.Background(), plan, &memJournal{}, fs, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "non-empty") {
		t.Fatalf("got %v", err)
	}
}

func TestApplyDoesNotWriteASecret(t *testing.T) {
	t.Parallel()
	fs := newMem()
	plan := Plan{Steps: []Step{{ID: "env", Kind: KindFile, Path: ".env", Secret: true, After: "PASSWORD=secret"}}}
	err := Apply(context.Background(), plan, &memJournal{}, fs, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "secret store") {
		t.Fatalf("got %v", err)
	}
	if len(fs.files) != 0 {
		t.Fatalf("wrote %#v", fs.files)
	}
}

func TestApplyRefusesACommandWithoutAnExecutor(t *testing.T) {
	t.Parallel()
	plan := Plan{Steps: []Step{{ID: "up", Kind: KindCommand, Path: "compose", After: "docker compose up"}}}
	err := Apply(context.Background(), plan, &memJournal{}, newMem(), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "executor") {
		t.Fatalf("got %v", err)
	}
}
