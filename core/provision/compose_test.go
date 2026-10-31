// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestComposePlanIsNonProduction(t *testing.T) {
	t.Parallel()
	plan, err := Compose(ComposeSpec{
		Name: "demo", Major: 17, Nodes: 3, Etcd: 3, Pooler: true,
		Superuser: "super-secret", Replication: "repl-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Target != "compose" || plan.Production {
		t.Fatalf("plan %+v", plan)
	}
	refused := plan
	refused.Production = true
	if err := refused.Check(); err == nil {
		t.Fatal("compose was accepted as production")
	}
	body := stepBody(t, plan, "compose.yaml")
	for _, want := range []string{
		"Non-production", "etcd1:", "etcd3:", "pg1:", "pg3:", "haproxy:", "pgbouncer:",
		"ALLOW_NOSSL", "ghcr.io/zalando/spilo-17:4.0-p3", "pgbackrest.conf",
		"service_healthy", "8008/health", "127.0.0.1:8008:8008",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("compose.yaml lacks %q", want)
		}
	}
	if strings.Contains(body, "super-secret") || strings.Contains(body, "repl-secret") {
		t.Fatal("compose.yaml contains a password")
	}
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("compose.yaml: %v", err)
	}
	haproxy := stepBody(t, plan, "haproxy.cfg")
	if !strings.Contains(haproxy, "/primary") || !strings.Contains(haproxy, "/replica") {
		t.Fatalf("haproxy:\n%s", haproxy)
	}
	env := findStep(t, plan, ".env")
	if !env.Secret || !strings.Contains(env.After, "super-secret") {
		t.Fatal(".env is not a secret step holding the password")
	}
	changes, err := Changes(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range changes {
		if strings.Contains(change.Diff, "super-secret") || strings.Contains(change.Diff, "repl-secret") {
			t.Fatalf("diff leaked a password:\n%s", change.Diff)
		}
	}
}

func TestComposeRejectsASplitBrain(t *testing.T) {
	t.Parallel()
	if _, err := Compose(ComposeSpec{Name: "demo", Nodes: 3, Etcd: 1}); err == nil {
		t.Fatal("one etcd was accepted for three Postgres nodes")
	}
	if _, err := Compose(ComposeSpec{Name: "Demo"}); err == nil {
		t.Fatal("uppercase name was accepted")
	}
	if _, err := Compose(ComposeSpec{Name: "demo", Major: 13}); err == nil {
		t.Fatal("PostgreSQL 13 was accepted")
	}
}

func stepBody(t *testing.T, plan Plan, path string) string {
	t.Helper()
	return findStep(t, plan, path).After
}

func findStep(t *testing.T, plan Plan, path string) Step {
	t.Helper()
	for _, step := range plan.Steps {
		if step.Path == path {
			return step
		}
	}
	t.Fatalf("no step %s", path)
	return Step{}
}
