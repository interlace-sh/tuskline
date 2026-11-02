// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"strings"
	"testing"
)

func TestCNPGManifestsHaveNoSecretValues(t *testing.T) {
	t.Parallel()
	plan, err := CNPGApply(CNPGSpec{Name: "demo", Instances: 3, Pooler: true, Major: 17})
	if err != nil {
		t.Fatal(err)
	}
	cluster := stepBody(t, plan, "cluster.yaml")
	if !strings.Contains(cluster, "instances: 3") || !strings.Contains(cluster, "barman-cloud.cloudnative-pg.io") {
		t.Fatal(cluster)
	}
	store := stepBody(t, plan, "objectstore.yaml")
	if strings.Contains(store, "AKIA") || strings.Contains(store, "secret: ") {
		t.Fatal("object store embedded a credential")
	}
	if !strings.Contains(store, "secretKeyRef") {
		t.Fatal(store)
	}
	apply := findStep(t, plan, "kubectl apply")
	if strings.Contains(strings.Join(apply.Argv, " "), "SECRET") {
		t.Fatal(apply.Argv)
	}
	if _, err := CNPG(CNPGSpec{Name: "Demo"}); err == nil {
		t.Fatal("uppercase name was accepted")
	}
}

func TestSSHPlanUsesPGDGAndRefusesAShellHost(t *testing.T) {
	t.Parallel()
	plan, err := SSHApply(SSHSpec{Name: "demo", Major: 17, Hosts: []string{"10.0.1.11", "db.example"}, Autobase: true})
	if err != nil {
		t.Fatal(err)
	}
	packages := stepBody(t, plan, "packages.txt")
	if !strings.Contains(packages, "postgresql-17-pg-stat-kcache") || !strings.Contains(packages, "Nothing has been installed") {
		t.Fatal(packages)
	}
	if !strings.Contains(stepBody(t, plan, "inventory.yml"), "playbook") && !strings.Contains(stepBody(t, plan, "inventory.yml"), "does not run") {
		t.Fatal("autobase inventory was not labelled as unrun")
	}
	argv := findStep(t, plan, "10.0.1.11").Argv
	if argv[0] != "ssh" || argv[len(argv)-1] != "packages.txt" {
		t.Fatalf("%v", argv)
	}
	if _, err := SSHArgv("10.0.1.11;rm", []string{"true"}); err == nil {
		t.Fatal("a shell host was accepted")
	}
	if _, err := SSH(SSHSpec{Name: "demo"}); err == nil {
		t.Fatal("a plan with no hosts was accepted")
	}
}
