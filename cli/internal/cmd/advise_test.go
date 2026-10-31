// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"
)

func TestConfigAdviseRejectsBadMemory(t *testing.T) {
	_, err := run(t, "config", "advise", "dev/pg17", "--memory", "lots")
	if err == nil || !strings.Contains(err.Error(), "memory") {
		t.Fatalf("got %v", err)
	}
}

func TestConfigAdviseNeedsARef(t *testing.T) {
	_, err := run(t, "config", "advise")
	if err == nil {
		t.Fatal("missing ref was accepted")
	}
}
