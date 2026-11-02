// SPDX-License-Identifier: Apache-2.0

package ha

import (
	"strings"
	"testing"
	"time"
)

func TestParsePatroni(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"members":[
		{"name":"pg1","role":"leader","state":"running","timeline":7,"host":"pg1"},
		{"name":"pg2","role":"sync_standby","state":"streaming","timeline":7,"lag":1200}
	]}`)
	topo, err := ParsePatroni(raw)
	if err != nil {
		t.Fatal(err)
	}
	if topo.Leader().Name != "pg1" || topo.Timeline != 7 || topo.Members[1].LagBytes != 1200 {
		t.Fatalf("%+v", topo)
	}
	if !strings.Contains(topo.Text(), "pg2 sync_standby") {
		t.Fatal(topo.Text())
	}
}

func TestParseCNPG(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"phase":"Cluster in healthy state","instances":[
		{"name":"demo-1","ready":true,"primary":true},
		{"name":"demo-2","ready":true,"role":"replica"}
	]}`)
	topo, err := ParseCNPG(raw)
	if err != nil {
		t.Fatal(err)
	}
	if topo.Source != "cnpg" || topo.Leader().Name != "demo-1" || topo.Members[1].Role != "replica" {
		t.Fatalf("%+v", topo)
	}
}

func TestSwitchoverNeedsTheClusterNameOnProd(t *testing.T) {
	t.Parallel()
	action := Switchover{Environment: "prod", Cluster: "orders", From: "pg1", To: "pg2", Prod: true, API: "http://127.0.0.1:8008"}
	if err := action.Check(); err == nil {
		t.Fatal("prod switchover ran without the cluster name")
	}
	action.Confirm = "orders"
	if err := action.Check(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(action.Preview(), "secret") || !strings.Contains(action.Preview(), "Nothing has been sent") {
		t.Fatal(action.Preview())
	}
	restart := Restart{Cluster: "orders", API: "http://127.0.0.1:8008", Prod: true}
	if restart.Check() == nil {
		t.Fatal("prod restart ran without the cluster name")
	}
	if !strings.Contains(RestartBody(time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)), `"role":"replica"`) {
		t.Fatal(RestartBody(time.Time{}))
	}
}
