// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
)

// Step kinds. A command is not run unless the caller supplies an Executor.
const (
	KindFile     = "file"
	KindUnit     = "unit"
	KindManifest = "manifest"
	KindCommand  = "command"
)

// Step is one change. ID is stable for the lifetime of that change: a new
// body is a new hash, so a later apply runs it again.
type Step struct {
	ID   string
	Kind string
	// Path is the file, unit or manifest path. Command steps use it as a label.
	Path string
	// Before is the current body when the caller already knows it.
	Before string
	// After is the desired body. Command steps store the preview, not a shell line to run blindly.
	After string
	// DataDir refuses the step when Path is an existing non-empty directory.
	DataDir bool
	// Secret means the body is not shown and is not written to the filesystem.
	Secret bool
}

// Plan is a preview. Apply is a separate call.
type Plan struct {
	Target string
	// Production is refused for the compose target.
	Production bool
	Steps      []Step
}

// Change is the diff of one step.
type Change struct {
	ID   string
	Kind string
	Path string
	// Diff is a unified diff. Empty when the body is unchanged.
	Diff string
}

// Check reports a plan that must not be applied.
func (p Plan) Check() error {
	if p.Target == "compose" && p.Production {
		return fmt.Errorf("compose deployments are non-production")
	}
	seen := map[string]bool{}
	for _, step := range p.Steps {
		if step.ID == "" {
			return fmt.Errorf("step has no id")
		}
		if seen[step.ID] {
			return fmt.Errorf("duplicate step %s", step.ID)
		}
		seen[step.ID] = true
		switch step.Kind {
		case KindFile, KindUnit, KindManifest, KindCommand:
		default:
			return fmt.Errorf("step %s: unknown kind %q", step.ID, step.Kind)
		}
		if step.Kind != KindCommand {
			if _, err := cleanPath(step.Path); err != nil {
				return fmt.Errorf("step %s: %w", step.ID, err)
			}
		}
	}
	return nil
}

// Changes builds a diff for every step. Unchanged bodies produce an empty Diff.
func Changes(p Plan) ([]Change, error) {
	if err := p.Check(); err != nil {
		return nil, err
	}
	out := make([]Change, 0, len(p.Steps))
	for _, step := range p.Steps {
		name := step.Path
		if name == "" {
			name = step.ID
		}
		diff := ""
		if step.Secret {
			if step.Before != step.After {
				diff = fmt.Sprintf("--- a/%s\n+++ b/%s\n# secret; contents not shown\n", name, name)
			}
		} else {
			diff = Unified(name, step.Before, step.After)
		}
		out = append(out, Change{ID: step.ID, Kind: step.Kind, Path: name, Diff: diff})
	}
	return out, nil
}

// Unified is a line diff of before and after. An unchanged body is empty.
func Unified(name, before, after string) string {
	if before == after {
		return ""
	}
	a := splitLines(before)
	b := splitLines(after)
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	old := a[prefix : len(a)-suffix]
	newLines := b[prefix : len(b)-suffix]
	var buf strings.Builder
	fmt.Fprintf(&buf, "--- a/%s\n+++ b/%s\n@@ -%d,%d +%d,%d @@\n", name, name, prefix+1, len(old), prefix+1, len(newLines))
	for _, line := range old {
		buf.WriteString("-")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	for _, line := range newLines {
		buf.WriteString("+")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func cleanPath(p string) (string, error) {
	if p == "" || strings.Contains(p, "\x00") {
		return "", fmt.Errorf("empty path")
	}
	if path.IsAbs(p) {
		return "", fmt.Errorf("path %s is absolute", p)
	}
	c := path.Clean(p)
	if c == ".." || strings.HasPrefix(c, "../") || c == "." {
		return "", fmt.Errorf("path %s escapes the plan directory", p)
	}
	return c, nil
}

func stepHash(step Step) string {
	sum := sha256.Sum256([]byte(step.Kind + "\n" + step.Path + "\n" + step.After))
	return hex.EncodeToString(sum[:])
}
