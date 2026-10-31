// SPDX-License-Identifier: Apache-2.0

package advisor

import (
	"fmt"
	"strings"
)

// Text renders r for a terminal. ref is the env/cluster the report is about.
func (r Report) Text(ref string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s · PostgreSQL %d · %s\n", ref, r.VersionNum/10000, tally(r))
	if len(r.Advice) == 0 && len(r.Notes) == 0 {
		b.WriteString("No changes. The settings that were checked are already in range.\n")
		return b.String()
	}
	for _, a := range r.Advice {
		fmt.Fprintf(&b, "\n%s  %s\n  %s → %s\n  %s\n  %s\n", a.Apply, a.Name, a.Current, a.Proposed, a.Diff, a.Summary)
		if a.Evidence != "" {
			fmt.Fprintf(&b, "  evidence: %s\n", a.Evidence)
		}
	}
	if len(r.Notes) > 0 {
		b.WriteString("\nNotes\n")
		for _, n := range r.Notes {
			fmt.Fprintf(&b, "  %s\n", n)
		}
	}
	return b.String()
}

func tally(r Report) string {
	switch n := len(r.Advice); n {
	case 0:
		return "no changes"
	case 1:
		return "1 change"
	default:
		return fmt.Sprintf("%d changes", n)
	}
}
