// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"strings"
	"testing"
)

func TestPlanRejectsComposeInProduction(t *testing.T) {
	t.Parallel()
	err := (Plan{Target: "compose", Production: true}).Check()
	if err == nil || !strings.Contains(err.Error(), "non-production") {
		t.Fatalf("got %v", err)
	}
}

func TestPlanRejectsEscapingPaths(t *testing.T) {
	t.Parallel()
	err := (Plan{Steps: []Step{{ID: "a", Kind: KindFile, Path: "../etc/passwd", After: "x"}}}).Check()
	if err == nil {
		t.Fatal("escaping path was accepted")
	}
	err = (Plan{Steps: []Step{{ID: "a", Kind: KindFile, Path: "/etc/passwd", After: "x"}}}).Check()
	if err == nil {
		t.Fatal("absolute path was accepted")
	}
}

func TestChangesShowsADiffAndHidesSecrets(t *testing.T) {
	t.Parallel()
	changes, err := Changes(Plan{Steps: []Step{
		{ID: "cfg", Kind: KindFile, Path: "patroni.yml", Before: "scope: old\n", After: "scope: demo\n"},
		{ID: "same", Kind: KindUnit, Path: "patroni.service", Before: "same\n", After: "same\n"},
		{ID: "pw", Kind: KindFile, Path: ".env", Secret: true, After: "PASSWORD=secret"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 3 {
		t.Fatalf("%d changes", len(changes))
	}
	if !strings.Contains(changes[0].Diff, "-scope: old") || !strings.Contains(changes[0].Diff, "+scope: demo") {
		t.Fatalf("diff:\n%s", changes[0].Diff)
	}
	if changes[1].Diff != "" {
		t.Fatalf("unchanged diff:\n%s", changes[1].Diff)
	}
	if strings.Contains(changes[2].Diff, "PASSWORD") || !strings.Contains(changes[2].Diff, "secret; contents not shown") {
		t.Fatalf("secret diff:\n%s", changes[2].Diff)
	}
}
