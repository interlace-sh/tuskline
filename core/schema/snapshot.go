// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Object kinds stored in a snapshot.
const (
	KindColumn     = "column"
	KindConstraint = "constraint"
	KindEnum       = "enum"
	KindIndex      = "index"
	KindMatview    = "matview"
	KindRoutine    = "routine"
	KindView       = "view"
)

// Object is one named definition. Detail is the part that is compared.
type Object struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// Snapshot is the structure of one database at one moment.
type Snapshot struct {
	Objects []Object `json:"objects"`
}

const maxSnapshot = 8 << 20

// Parse decodes a snapshot. Object order does not matter.
func Parse(raw []byte) (Snapshot, error) {
	if len(raw) > maxSnapshot {
		return Snapshot{}, fmt.Errorf("schema snapshot is larger than %d bytes", maxSnapshot)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var snap Snapshot
	if err := dec.Decode(&snap); err != nil {
		return Snapshot{}, fmt.Errorf("parse schema snapshot: %w", err)
	}
	if dec.More() {
		return Snapshot{}, errors.New("parse schema snapshot: extra data")
	}
	if snap.Objects == nil {
		snap.Objects = []Object{}
	}
	sort.Slice(snap.Objects, func(i, j int) bool {
		return lessObject(snap.Objects[i], snap.Objects[j])
	})
	return snap, nil
}

// ReadFile reads a snapshot from path.
func ReadFile(path string) (Snapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Snapshot{}, err
	}
	if info.Size() > maxSnapshot {
		return Snapshot{}, fmt.Errorf("%s is larger than %d bytes", path, maxSnapshot)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	return Parse(raw)
}

// JSON encodes the snapshot with objects in a stable order.
func (s Snapshot) JSON() ([]byte, error) {
	if s.Objects == nil {
		s.Objects = []Object{}
	}
	sort.Slice(s.Objects, func(i, j int) bool {
		return lessObject(s.Objects[i], s.Objects[j])
	})
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	return raw, nil
}

func lessObject(a, b Object) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return a.Name < b.Name
}

// Change is one difference between two snapshots. Op is "+", "-" or "~".
// From is the baseline detail and To is the later detail.
type Change struct {
	Op   string
	Kind string
	Name string
	From string
	To   string
}

// Diff reports what changed to get from before to after.
func Diff(before, after Snapshot) []Change {
	old := byKey(before)
	neu := byKey(after)
	var out []Change
	seen := make(map[string]bool, len(old))
	for key, item := range old {
		seen[key] = true
		next, ok := neu[key]
		if !ok {
			out = append(out, Change{Op: "-", Kind: item.Kind, Name: item.Name, From: item.Detail})
			continue
		}
		if item.Detail != next.Detail {
			out = append(out, Change{Op: "~", Kind: item.Kind, Name: item.Name, From: item.Detail, To: next.Detail})
		}
	}
	for key, item := range neu {
		if seen[key] {
			continue
		}
		out = append(out, Change{Op: "+", Kind: item.Kind, Name: item.Name, To: item.Detail})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Op < out[j].Op
	})
	return out
}

// Report renders changes, one per line. An empty diff says there is none.
func Report(changes []Change) string {
	if len(changes) == 0 {
		return "No drift.\n"
	}
	var b strings.Builder
	for _, change := range changes {
		switch change.Op {
		case "+":
			fmt.Fprintf(&b, "+ %s %s  %s\n", change.Kind, change.Name, change.To)
		case "-":
			fmt.Fprintf(&b, "- %s %s  %s\n", change.Kind, change.Name, change.From)
		default:
			fmt.Fprintf(&b, "~ %s %s  %s -> %s\n", change.Kind, change.Name, change.From, change.To)
		}
	}
	return b.String()
}

func byKey(snap Snapshot) map[string]Object {
	out := make(map[string]Object, len(snap.Objects))
	for _, item := range snap.Objects {
		out[item.Kind+"\x00"+item.Name] = item
	}
	return out
}
