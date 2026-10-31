// SPDX-License-Identifier: Apache-2.0

package advisor

import (
	"os"
	"strings"
	"testing"
)

func TestAdviseStockWebServer(t *testing.T) {
	t.Parallel()
	report, err := Advise(170000, Hardware{
		Memory: 16 * gb, CPUs: 8, Storage: "ssd", Workload: "web", OS: "linux",
	}, stockSettings(), Evidence{HaveCache: true, CacheHit: 0.97})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Advice{}
	for _, a := range report.Advice {
		got[a.Name] = a
	}
	for _, name := range []string{
		"shared_buffers", "effective_cache_size", "work_mem", "maintenance_work_mem",
		"random_page_cost", "effective_io_concurrency", "maintenance_io_concurrency",
		"max_parallel_workers_per_gather", "max_parallel_maintenance_workers",
		"jit", "track_io_timing", "idle_in_transaction_session_timeout", "wal_compression",
	} {
		if _, ok := got[name]; !ok {
			t.Errorf("missing %s", name)
		}
	}
	if got["shared_buffers"].Proposed != "4GB" || got["shared_buffers"].Apply != "restart" {
		t.Fatalf("shared_buffers: %+v", got["shared_buffers"])
	}
	if !strings.Contains(got["shared_buffers"].Evidence, "97%") {
		t.Fatalf("evidence: %s", got["shared_buffers"].Evidence)
	}
	if got["work_mem"].Proposed != "19MB" {
		t.Fatalf("work_mem: %+v", got["work_mem"])
	}
	if got["wal_compression"].Proposed != "lz4" || got["jit"].Proposed != "off" {
		t.Fatalf("wal %q jit %q", got["wal_compression"].Proposed, got["jit"].Proposed)
	}
	for _, banned := range []string{"wal_level", "wal_buffers", "max_connections"} {
		if _, ok := got[banned]; ok {
			t.Errorf("%s should not change on this server", banned)
		}
	}
	text := report.Text("dev/pg17")
	golden, err := os.ReadFile("testdata/web-16gb.txt")
	if err != nil {
		t.Fatal(err)
	}
	if text != string(golden) {
		t.Fatalf("fixture report changed:\n%s", text)
	}
	if !strings.Contains(text, "huge pages") {
		t.Fatalf("missing huge-page note:\n%s", text)
	}
}

func TestAdviseLeavesAReasonableSharedBuffers(t *testing.T) {
	t.Parallel()
	current := stockSettings()
	current["shared_buffers"] = memObs("5GB", 5*gb/kb, "postmaster")
	report, err := Advise(170000, Hardware{Memory: 16 * gb, Workload: "web"}, current, Evidence{})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range report.Advice {
		if a.Name == "shared_buffers" {
			t.Fatalf("5GB is between 25%% and 40%% of 16GB and should stay: %+v", a)
		}
	}
}

func TestAdviseDoesNotRaiseConnections(t *testing.T) {
	t.Parallel()
	report, err := Advise(170000, Hardware{Memory: 16 * gb, Workload: "web"}, stockSettings(), Evidence{})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range report.Advice {
		if a.Name == "max_connections" {
			t.Fatalf("raised connections: %+v", a)
		}
	}
}

func TestAdviseWithoutHardwareStillLints(t *testing.T) {
	t.Parallel()
	report, err := Advise(170000, Hardware{}, stockSettings(), Evidence{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, a := range report.Advice {
		got[a.Name] = true
		if a.Name == "shared_buffers" || a.Name == "work_mem" {
			t.Fatalf("memory advice without RAM: %+v", a)
		}
	}
	for _, name := range []string{"track_io_timing", "idle_in_transaction_session_timeout", "wal_compression"} {
		if !got[name] {
			t.Errorf("missing %s", name)
		}
	}
}

func TestParseMemory(t *testing.T) {
	t.Parallel()
	n, err := ParseMemory("16GB")
	if err != nil || n != 16*gb {
		t.Fatalf("16GB = %d, %v", n, err)
	}
	if _, err := ParseMemory("lots"); err == nil {
		t.Fatal("lots was accepted")
	}
}

func stockSettings() map[string]Observed {
	return map[string]Observed{
		"shared_buffers":                      memObs("128MB", 128*mb/kb, "postmaster"),
		"effective_cache_size":                memObs("4GB", 4*gb/kb, "user"),
		"maintenance_work_mem":                memObs("64MB", 64*mb/kb, "user"),
		"work_mem":                            memObs("4MB", 4*mb/kb, "user"),
		"max_wal_size":                        memObs("1GB", gb/kb, "sighup"),
		"random_page_cost":                    {Text: "4", Real: 4, HasReal: true, Context: "user"},
		"effective_io_concurrency":            {Text: "1", Int: 1, HasInt: true, Context: "user"},
		"maintenance_io_concurrency":          {Text: "10", Int: 10, HasInt: true, Context: "user"},
		"checkpoint_completion_target":        {Text: "0.9", Real: 0.9, HasReal: true, Context: "sighup"},
		"wal_compression":                     {Text: "off", Enums: []string{"off", "pglz", "lz4", "zstd"}, Context: "sighup"},
		"jit":                                 {Text: "on", Context: "user"},
		"autovacuum":                          {Text: "on", Context: "sighup"},
		"track_io_timing":                     {Text: "off", Context: "sighup"},
		"idle_in_transaction_session_timeout": {Text: "0", Int: 0, HasInt: true, Context: "user"},
		"huge_pages":                          {Text: "try", Context: "postmaster"},
		"max_connections":                     {Text: "100", Int: 100, HasInt: true, Context: "postmaster"},
		"max_worker_processes":                {Text: "8", Int: 8, HasInt: true, Context: "postmaster"},
		"max_parallel_workers":                {Text: "8", Int: 8, HasInt: true, Context: "user"},
		"max_parallel_workers_per_gather":     {Text: "2", Int: 2, HasInt: true, Context: "user"},
		"max_parallel_maintenance_workers":    {Text: "2", Int: 2, HasInt: true, Context: "user"},
		"autovacuum_max_workers":              {Text: "3", Int: 3, HasInt: true, Context: "postmaster"},
		"wal_buffers":                         {Text: "-1", Context: "postmaster"},
	}
}

func memObs(text string, valueKB int64, context string) Observed {
	return Observed{Text: text, KB: valueKB, HasKB: true, Context: context}
}
