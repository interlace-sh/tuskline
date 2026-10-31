// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
)

func readyReport() Report {
	return Report{
		FreeBytes:     20 << 30,
		DockerServer:  "27.3.1",
		DockerCompose: "2.29.7",
	}
}

func TestRequirementsForASingleNode(t *testing.T) {
	t.Parallel()
	need, err := Requirements(ComposeSpec{Name: "demo", Nodes: 1, Etcd: 1, Pooler: false})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(ints(need.Ports), ",") != "5000,5001" {
		t.Fatalf("ports %v", need.Ports)
	}
	if strings.Join(need.Volumes, ",") != "demo_etcd1,demo_pg1,demo_pgbackrest" {
		t.Fatalf("volumes %v", need.Volumes)
	}
	if need.MinFree != 3<<30 {
		t.Fatalf("min free %d", need.MinFree)
	}
}

func TestEvaluatePreflight(t *testing.T) {
	t.Parallel()
	need, err := Requirements(ComposeSpec{Name: "demo", Nodes: 1, Etcd: 1, Pooler: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := Evaluate(need, readyReport()); err != nil {
		t.Fatal(err)
	}
	short := readyReport()
	short.FreeBytes = 1 << 30
	short.BusyPorts = map[int]bool{5000: true}
	short.Volumes = []string{"demo_pg1", "other"}
	short.DockerCompose = "2.19.0"
	err = Evaluate(need, short)
	if err == nil {
		t.Fatal("a short disk, an old Compose, a busy port and a leftover volume were accepted")
	}
	for _, want := range []string{"older than 2.20.0", "disk has 1 GiB free, need 3 GiB", "port 5000 is in use", "volume demo_pg1 already exists"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "other") {
		t.Fatalf("an unrelated volume was refused: %v", err)
	}

	resume := short
	resume.FreeBytes = 20 << 30
	resume.DockerCompose = "2.20.0"
	resume.Resume = true
	resume.OwnPorts = map[int]bool{5000: true, 5001: true, 6432: true}
	resume.BusyPorts = map[int]bool{5000: true, 5001: true, 6432: true}
	if err := Evaluate(need, resume); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(Summary(need, resume), "already held") {
		t.Fatal(Summary(need, resume))
	}
}

func TestEvaluateRejectsAnOldEngine(t *testing.T) {
	t.Parallel()
	need, err := Requirements(ComposeSpec{Name: "demo", Nodes: 1, Etcd: 1})
	if err != nil {
		t.Fatal(err)
	}
	report := readyReport()
	report.DockerServer = "23.0.6"
	if err := Evaluate(need, report); err == nil || !strings.Contains(err.Error(), "older than 24.0.0") {
		t.Fatal(err)
	}
	report.DockerError = "docker is not installed"
	if err := Evaluate(need, report); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatal(err)
	}
}

func TestPublishedPorts(t *testing.T) {
	t.Parallel()
	got := PublishedPorts("0.0.0.0:5000->5000/tcp, :::5000->5000/tcp, 0.0.0.0:6432->6432/tcp")
	if !got[5000] || !got[6432] || got[5432] {
		t.Fatalf("%v", got)
	}
}

func TestPortFree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	// 127.0.0.1 does not block 0.0.0.0, so hold the same address family.
	held, err := lc.Listen(ctx, "tcp4", "0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		t.Fatal(err)
	}
	free, err := PortFree(ctx, port)
	if err != nil || free {
		t.Fatalf("free=%v %v", free, err)
	}
	if err := held.Close(); err != nil {
		t.Fatal(err)
	}
	free, err = PortFree(ctx, port)
	if err != nil || !free {
		t.Fatalf("after close free=%v %v", free, err)
	}
}

func TestRefuseDataDirs(t *testing.T) {
	t.Parallel()
	fs := newMem()
	fs.files["pgdata/PG_VERSION"] = "17\n"
	plan := Plan{Steps: []Step{{ID: "data", Kind: KindFile, Path: "pgdata", DataDir: true}}}
	if err := RefuseDataDirs(context.Background(), plan, fs); err == nil || !strings.Contains(err.Error(), "non-empty") {
		t.Fatal(err)
	}
	empty := Plan{Steps: []Step{{ID: "cfg", Kind: KindFile, Path: "compose.yaml"}}}
	if err := RefuseDataDirs(context.Background(), empty, nil); err != nil {
		t.Fatal(err)
	}
}

func ints(ports []int) []string {
	out := make([]string, len(ports))
	for i, port := range ports {
		out[i] = strconv.Itoa(port)
	}
	return out
}
