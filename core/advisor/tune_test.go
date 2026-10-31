// SPDX-License-Identifier: Apache-2.0

package advisor

import "testing"

func TestTargetsMatchPgTune16GBWeb(t *testing.T) {
	t.Parallel()
	plan, err := Targets(170000, Hardware{
		Memory: 16 * gb, CPUs: 8, Storage: "ssd", Workload: "web", DBSize: "mid", OS: "linux",
	}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Connections != 200 {
		t.Fatalf("work_mem was sized for %d connections, want the web default 200", plan.Connections)
	}
	want := map[string]string{
		"shared_buffers":                   "4GB",
		"effective_cache_size":             "12GB",
		"maintenance_work_mem":             "1GB",
		"work_mem":                         "19MB",
		"random_page_cost":                 "1.1",
		"effective_io_concurrency":         "200",
		"maintenance_io_concurrency":       "200",
		"max_worker_processes":             "8",
		"max_parallel_workers_per_gather":  "4",
		"max_parallel_workers":             "8",
		"max_parallel_maintenance_workers": "4",
		"jit":                              "off",
	}
	got := map[string]Param{}
	for _, p := range plan.Params {
		got[p.Name] = p
		if p.Name == "wal_buffers" || p.Name == "wal_level" || p.Name == "max_connections" {
			t.Errorf("refused setting was generated: %s", p.Name)
		}
	}
	for name, text := range want {
		if got[name].Text != text {
			t.Errorf("%s = %q, want %q", name, got[name].Text, text)
		}
	}
}

func TestTargetsDesktopAndWarehouse(t *testing.T) {
	t.Parallel()
	desktop, err := Targets(170000, Hardware{Memory: 16 * gb, Workload: "desktop"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if text := param(desktop, "shared_buffers"); text != "1GB" {
		t.Fatalf("desktop shared_buffers = %s, want 1GB", text)
	}
	wh, err := Targets(170000, Hardware{Memory: 16 * gb, Storage: "ssd", Workload: "dw"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if text := param(wh, "random_page_cost"); text != "4" {
		t.Fatalf("warehouse random_page_cost = %s, want 4", text)
	}
	if _, ok := find(wh, "jit"); ok {
		t.Fatal("a warehouse should keep JIT")
	}
}

func TestTargetsRefuseUnknownInput(t *testing.T) {
	t.Parallel()
	if _, err := Targets(130000, Hardware{}, 0); err == nil {
		t.Fatal("PostgreSQL 13 was accepted")
	}
	if _, err := Targets(170000, Hardware{Storage: "tape"}, 0); err == nil {
		t.Fatal("tape was accepted")
	}
}

func param(plan Plan, name string) string {
	p, ok := find(plan, name)
	if !ok {
		return ""
	}
	return p.Text
}

func find(plan Plan, name string) (Param, bool) {
	for _, p := range plan.Params {
		if p.Name == name {
			return p, true
		}
	}
	return Param{}, false
}
