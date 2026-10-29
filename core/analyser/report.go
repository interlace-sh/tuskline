// SPDX-License-Identifier: Apache-2.0

package analyser

import (
	"fmt"
	"strings"
)

// Text renders r for a terminal. ref is the env/cluster the report is about.
func (r Report) Text(ref string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s · PostgreSQL %d · %s\n", ref, r.VersionNum/10000, counts(r.Findings))
	if len(r.Findings) == 0 {
		b.WriteString("No findings. The checks that ran were clean.\n")
		return b.String()
	}
	b.WriteByte('\n')
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "%s  %s\n  %s\n  evidence: %s\n  fix: %s\n\n", f.Severity, f.Check, f.Summary, one(f.Evidence), one(f.Fix))
	}
	return b.String()
}

// Markdown renders r as a Markdown document.
func (r Report) Markdown(ref string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nPostgreSQL %d. %s\n\n", ref, r.VersionNum/10000, counts(r.Findings))
	if len(r.Findings) == 0 {
		b.WriteString("No findings. The checks that ran were clean.\n")
		return b.String()
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "## %s: %s\n\n%s\n\n**Evidence:** %s\n\n**Fix:** %s\n\n", f.Severity, f.Check, f.Summary, one(f.Evidence), one(f.Fix))
	}
	return b.String()
}

func counts(findings []Finding) string {
	var warnings, notes int
	for _, f := range findings {
		if f.Severity == SeverityWarn {
			warnings++
		} else {
			notes++
		}
	}
	return fmt.Sprintf("%s, %s", plural(warnings, "warning"), plural(notes, "note"))
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func one(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
