// SPDX-License-Identifier: Apache-2.0

package analyser

import (
	"strings"
	"testing"
)

func TestReportTextAndMarkdown(t *testing.T) {
	t.Parallel()
	empty := Report{VersionNum: 170006, Findings: []Finding{}}
	if !strings.Contains(empty.Text("dev/pg"), "No findings") || !strings.Contains(empty.Markdown("dev/pg"), "# dev/pg") {
		t.Fatalf("empty text:\n%s\nmarkdown:\n%s", empty.Text("dev/pg"), empty.Markdown("dev/pg"))
	}
	r := Report{VersionNum: 170006, Findings: []Finding{{
		Check: "duplicate_index", Severity: SeverityWarn,
		Summary: "b duplicates a on t", Evidence: "both index 1", Fix: "Drop the duplicate.",
	}, {
		Check: "bloat", Severity: SeverityInfo, Summary: "public.t is about 50% dead tuples", Evidence: "n_dead_tup 10", Fix: "VACUUM.",
	}}}
	text := r.Text("dev/pg")
	if !strings.Contains(text, "1 warning, 1 note") || !strings.Contains(text, "warn  duplicate_index") {
		t.Fatalf("%s", text)
	}
	md := r.Markdown("dev/pg")
	if !strings.Contains(md, "## warn: duplicate_index") || !strings.Contains(md, "**Fix:** VACUUM.") {
		t.Fatalf("%s", md)
	}
}
