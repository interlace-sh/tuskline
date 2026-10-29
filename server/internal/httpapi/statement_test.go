// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import "testing"

func TestOneStatement(t *testing.T) {
	t.Parallel()
	ok, err := oneStatement("  SELECT 1;  ")
	if err != nil || ok != "SELECT 1" {
		t.Fatalf("got %q %v", ok, err)
	}
	if _, err := oneStatement("SELECT 1; SELECT 2"); err == nil {
		t.Fatal("stacked statements were accepted")
	}
	quoted, err := oneStatement("SELECT ';'")
	if err != nil || quoted != "SELECT ';'" {
		t.Fatalf("quoted semicolon: %q %v", quoted, err)
	}
	if _, err := oneStatement("   "); err == nil {
		t.Fatal("empty statement was accepted")
	}
}
