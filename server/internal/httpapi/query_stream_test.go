// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"bytes"
	"strings"
	"testing"
)

func TestQueryStreamEvents(t *testing.T) {
	var buf bytes.Buffer
	if err := writeQueryEvent(&buf, queryHead{Columns: []string{"n"}}); err != nil {
		t.Fatal(err)
	}
	if err := writeQueryEvent(&buf, []any{1}); err != nil {
		t.Fatal(err)
	}
	if err := writeQueryEvent(&buf, queryTail{End: true, Duration: 3, Rows: 1}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines: %q", lines)
	}
	if lines[0] != `{"columns":["n"]}` || lines[1] != `[1]` || !strings.Contains(lines[2], `"end":true`) {
		t.Fatalf("stream: %q", lines)
	}
}
