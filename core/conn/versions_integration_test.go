// SPDX-License-Identifier: Apache-2.0

//go:build integration

package conn_test

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func pgVersions(t *testing.T) []int {
	t.Helper()
	s := os.Getenv("PG_VERSIONS")
	if s == "" {
		return []int{14, 15, 16, 17, 18}
	}
	var out []int
	// Accept "17", "14,18" or "14 15 16".
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			t.Fatalf("PG_VERSIONS: %v", err)
		}
		out = append(out, n)
	}
	return out
}
