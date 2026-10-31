// SPDX-License-Identifier: Apache-2.0

package advisor

import (
	"fmt"
	"strings"
)

// Observed is one live pg_settings row, already parsed.
type Observed struct {
	Text    string
	KB      int64
	HasKB   bool
	Real    float64
	HasReal bool
	Int     int
	HasInt  bool
	Enums   []string
	Context string // postmaster, sighup, user, ...
}

// Evidence is optional live statistics. A false Have field means the
// view could not be read, not that the value is zero.
type Evidence struct {
	HaveCache            bool
	CacheHit             float64 // 0 to 1
	HaveTemp             bool
	TempBytes            int64
	HaveCheckpoints      bool
	CheckpointsTimed     int64
	CheckpointsRequested int64
}

// Advice is one change to preview. It has not been run.
type Advice struct {
	Name     string `json:"name"`
	Current  string `json:"current"`
	Proposed string `json:"proposed"`
	Apply    string `json:"apply"`
	Summary  string `json:"summary"`
	Evidence string `json:"evidence,omitempty"`
	Diff     string `json:"diff"`
}

// Report is a preview for one server.
type Report struct {
	VersionNum int      `json:"server_version_num"`
	Advice     []Advice `json:"advice"`
	Notes      []string `json:"notes"`
}

// Advise builds a preview. current may be nil when only the candidate
// list is wanted; a nil entry means the setting was not read, and it is
// then omitted rather than treated as a default.
func Advise(versionNum int, hw Hardware, current map[string]Observed, ev Evidence) (Report, error) {
	var currentMax int
	if o, ok := current["max_connections"]; ok && o.HasInt {
		currentMax = o.Int
	}
	plan, err := Targets(versionNum, hw, currentMax)
	if err != nil {
		return Report{}, err
	}
	report := Report{VersionNum: versionNum}
	for _, p := range plan.Params {
		if a, ok := consider(p, current[p.Name], plan, ev); ok {
			report.Advice = append(report.Advice, a)
		}
	}
	report.Advice = append(report.Advice, standing(versionNum, current, plan, ev)...)
	report.Notes = notes(plan, current, report.Advice)
	if report.Advice == nil {
		report.Advice = []Advice{}
	}
	if report.Notes == nil {
		report.Notes = []string{}
	}
	return report, nil
}

func consider(p Param, cur Observed, plan Plan, ev Evidence) (Advice, bool) {
	if cur.Text == "" {
		return Advice{}, false
	}
	switch p.Name {
	case "wal_buffers":
		return Advice{}, false
	case "io_method":
		if len(cur.Enums) == 0 || !contains(cur.Enums, p.Text) || cur.Text == p.Text {
			return Advice{}, false
		}
	case "shared_buffers":
		if cur.HasKB && plan.Memory > 0 && cur.KB >= p.KB && cur.KB <= plan.Memory/kb*2/5 {
			return Advice{}, false
		}
		if cur.HasKB && near(cur.KB, p.KB) {
			return Advice{}, false
		}
	case "work_mem":
		if !cur.HasKB || (cur.KB >= p.KB/2 && cur.KB <= p.KB*2) {
			return Advice{}, false
		}
	default:
		if !differs(p, cur) {
			return Advice{}, false
		}
		if raiseOnly(p.Name) && cur.HasInt && cur.Int >= p.Int {
			return Advice{}, false
		}
	}
	evidence := evidenceFor(p.Name, ev)
	return Advice{
		Name: p.Name, Current: display(cur), Proposed: p.Text,
		Apply: applyOf(p, cur), Summary: p.Why, Evidence: evidence,
		Diff: p.Name + " = " + p.Text,
	}, true
}

func standing(versionNum int, current map[string]Observed, plan Plan, ev Evidence) []Advice {
	var out []Advice
	if a, ok := whenOff(current, "autovacuum", "on", "reload",
		"Autovacuum has to stay on. Switching it off lets dead tuples accumulate; wraparound vacuum still runs, and it is not a substitute."); ok {
		out = append(out, a)
	}
	if a, ok := whenOff(current, "track_io_timing", "on", "reload",
		"Block read and write times are what make I/O waits visible. The overhead on current hardware is small."); ok {
		out = append(out, a)
	}
	if cur, ok := current["idle_in_transaction_session_timeout"]; ok && cur.Text != "" && zeroSetting(cur) {
		out = append(out, Advice{
			Name: "idle_in_transaction_session_timeout", Current: display(cur), Proposed: "5min",
			Apply:   applyOf(Param{Apply: "reload"}, cur),
			Summary: "A transaction left idle holds a snapshot and blocks vacuum. Five minutes is for application sessions. The collector role keeps its own 10 second timeout.",
			Diff:    "idle_in_transaction_session_timeout = 5min",
		})
	}
	if cur, ok := current["checkpoint_completion_target"]; ok && cur.HasReal && cur.Real < 0.9 {
		out = append(out, Advice{
			Name: "checkpoint_completion_target", Current: display(cur), Proposed: "0.9",
			Apply:   applyOf(Param{Apply: "reload"}, cur),
			Summary: "A checkpoint should spread its writes across the interval. 0.9 has been the default since PostgreSQL 14. A lower value finishes in a burst.",
			Diff:    "checkpoint_completion_target = 0.9",
		})
	}
	if a, ok := walCompression(versionNum/10000, current["wal_compression"]); ok {
		out = append(out, a)
	}
	shared := plan.SharedKB
	if cur, ok := current["shared_buffers"]; ok && cur.HasKB && cur.KB > shared {
		shared = cur.KB
	}
	if cur, ok := current["huge_pages"]; ok && cur.Text == "off" && shared >= 2*gb/kb {
		out = append(out, Advice{
			Name: "huge_pages", Current: "off", Proposed: "try",
			Apply:    applyOf(Param{Apply: "restart"}, cur),
			Summary:  "Shared buffers are large enough that the page tables of each connection cost real RAM. try uses huge pages when the operating system has reserved them.",
			Diff:     "huge_pages = try",
			Evidence: evidenceFor("shared_buffers", ev),
		})
	}
	if plan.Memory > 0 && plan.ConnectionFit > 0 {
		if cur, ok := current["max_connections"]; ok && cur.HasInt && cur.Int > plan.ConnectionFit {
			out = append(out, Advice{
				Name: "max_connections", Current: display(cur), Proposed: fmt.Sprintf("%d", plan.ConnectionFit),
				Apply:   applyOf(Param{Apply: "restart"}, cur),
				Summary: "This many connections does not leave 4MB of work_mem each after shared buffers. Put a pooler in front, then lower the limit. Lowering it under a live application will refuse logins.",
				Diff:    fmt.Sprintf("max_connections = %d", plan.ConnectionFit),
			})
		}
	}
	if ev.HaveCheckpoints && ev.CheckpointsRequested > ev.CheckpointsTimed && plan.Memory > 0 {
		if cur, ok := current["max_wal_size"]; ok && cur.HasKB {
			proposed := walTarget(plan.Workload)
			if cur.KB < proposed && !near(cur.KB, proposed) {
				out = append(out, Advice{
					Name: "max_wal_size", Current: display(cur), Proposed: FormatKB(proposed),
					Apply:    applyOf(Param{Apply: "reload"}, cur),
					Summary:  "Requested checkpoints outnumber timed ones, so the WAL budget is being exhausted early. A larger max_wal_size spreads those writes.",
					Evidence: fmt.Sprintf("Requested checkpoints %d, timed %d.", ev.CheckpointsRequested, ev.CheckpointsTimed),
					Diff:     "max_wal_size = " + FormatKB(proposed),
				})
			}
		}
	}
	return out
}

func walCompression(major int, cur Observed) (Advice, bool) {
	if cur.Text == "" || cur.Text == "zstd" || cur.Text == "lz4" {
		return Advice{}, false
	}
	proposed := "on"
	if major >= 15 {
		proposed = "lz4"
	}
	if proposed == "lz4" && len(cur.Enums) > 0 && !contains(cur.Enums, "lz4") {
		if cur.Text == "on" || cur.Text == "pglz" {
			return Advice{}, false
		}
		proposed = "on"
	}
	if cur.Text == proposed {
		return Advice{}, false
	}
	return Advice{
		Name: "wal_compression", Current: display(cur), Proposed: proposed,
		Apply:   applyOf(Param{Apply: "reload"}, cur),
		Summary: "Compressing WAL cuts archive and replication volume. lz4 is the fast method on PostgreSQL 15 and newer. zstd, if already selected, is left as it is.",
		Diff:    "wal_compression = " + proposed,
	}, true
}

func walTarget(workload string) int64 {
	switch workload {
	case "oltp":
		return 8 * gb / kb
	case "dw":
		return 16 * gb / kb
	case "desktop":
		return 2 * gb / kb
	default:
		return 4 * gb / kb
	}
}

func whenOff(current map[string]Observed, name, proposed, apply, why string) (Advice, bool) {
	cur, ok := current[name]
	if !ok || cur.Text == "" || cur.Text == proposed {
		return Advice{}, false
	}
	if cur.Text != "off" {
		return Advice{}, false
	}
	return Advice{
		Name: name, Current: "off", Proposed: proposed,
		Apply: applyOf(Param{Apply: apply}, cur), Summary: why,
		Diff: name + " = " + proposed,
	}, true
}

func notes(plan Plan, current map[string]Observed, advice []Advice) []string {
	var out []string
	if plan.Memory > 0 && plan.Memory < gb {
		out = append(out, "These formulas are a poor fit below 1GB of RAM. Leave room for the operating system.")
	}
	if plan.Memory > 100*gb {
		out = append(out, "These formulas are a poor fit above 100GB of RAM. Treat the candidates as a starting point and measure.")
	}
	shared := plan.SharedKB
	if cur, ok := current["shared_buffers"]; ok && cur.HasKB && cur.KB > shared {
		shared = cur.KB
	}
	if shared >= 2*gb/kb && plan.OS != "mac" {
		out = append(out, "Shared buffers are large. Reserve huge pages (vm.nr_hugepages) or each connection pays for its page tables. With the server stopped: postgres -D $PGDATA -C shared_memory_size_in_huge_pages")
	}
	if cur, ok := current["max_connections"]; ok && cur.HasInt && cur.Int > 200 && !plan.Pooler {
		out = append(out, fmt.Sprintf("max_connections is %d and no pooler was declared. A pooler keeps this memory math honest.", cur.Int))
	}
	if plan.Workload == "dw" && plan.Storage != "" && plan.Storage != "hdd" && plan.DBSize != "less" {
		out = append(out, "A warehouse on solid-state storage keeps random_page_cost at the default, so a large index scan is not preferred over a sequential read.")
	}
	for _, a := range advice {
		if a.Name == "wal_compression" && a.Proposed == "lz4" {
			out = append(out, "wal_compression = lz4 needs a build with lz4. SELECT enumvals FROM pg_settings WHERE name = 'wal_compression' lists it when the build has it.")
		}
		if a.Name == "io_method" {
			out = append(out, "io_method = io_uring needs a build with liburing, and containers often block io_uring in their seccomp profile. The server will not start if this build cannot use it.")
		}
	}
	return out
}

func differs(p Param, cur Observed) bool {
	switch p.Kind {
	case "memory":
		return cur.HasKB && !near(cur.KB, p.KB)
	case "int":
		return cur.HasInt && !near(int64(cur.Int), int64(p.Int))
	case "real":
		if !cur.HasReal {
			return false
		}
		d := cur.Real - p.Real
		if d < 0 {
			d = -d
		}
		return d >= 0.2
	default:
		return cur.Text != p.Text
	}
}

func near(current, target int64) bool {
	if target <= 0 {
		return current == target
	}
	d := current - target
	if d < 0 {
		d = -d
	}
	return d*5 <= target
}

func raiseOnly(name string) bool {
	switch name {
	case "max_worker_processes", "max_parallel_workers", "max_parallel_workers_per_gather",
		"max_parallel_maintenance_workers", "autovacuum_max_workers", "autovacuum_vacuum_cost_limit",
		"default_statistics_target", "effective_io_concurrency", "maintenance_io_concurrency":
		return true
	default:
		return false
	}
}

func evidenceFor(name string, ev Evidence) string {
	switch name {
	case "shared_buffers":
		if ev.HaveCache && ev.CacheHit > 0 {
			return fmt.Sprintf("Cache hit ratio is %.0f%%. That includes the operating system cache, so it is context, not proof.", ev.CacheHit*100)
		}
	case "work_mem":
		if ev.HaveTemp && ev.TempBytes > 0 {
			return fmt.Sprintf("Queries have spilled %s to temporary files.", FormatBytes(ev.TempBytes))
		}
	}
	return ""
}

func applyOf(p Param, cur Observed) string {
	if cur.Context == "postmaster" {
		return "restart"
	}
	if cur.Context != "" {
		return "reload"
	}
	return p.Apply
}

func display(cur Observed) string {
	if cur.HasKB {
		return FormatKB(cur.KB)
	}
	if cur.Text != "" {
		return cur.Text
	}
	return "unset"
}

func zeroSetting(cur Observed) bool {
	return cur.Text == "0" || (cur.HasInt && cur.Int == 0) || (cur.HasKB && cur.KB == 0)
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// FormatKB renders a kilobyte count the way a configuration file would.
func FormatKB(valueKB int64) string {
	if valueKB <= 0 {
		return "0"
	}
	switch {
	case valueKB%(gb/kb) == 0:
		return fmt.Sprintf("%dGB", valueKB/(gb/kb))
	case valueKB%1024 == 0:
		return fmt.Sprintf("%dMB", valueKB/1024)
	default:
		return fmt.Sprintf("%dkB", valueKB)
	}
}

// FormatBytes renders a byte count for evidence.
func FormatBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	return FormatKB(n / kb)
}

// ParseMemory parses a size such as 16GB or 512MB. GB is 1024-based, as in PgTune.
func ParseMemory(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.TrimSuffix(s, "IB")
	var mult int64
	switch {
	case strings.HasSuffix(s, "TB"):
		mult, s = 1024*1024*1024*1024, strings.TrimSuffix(s, "TB")
	case strings.HasSuffix(s, "GB"), strings.HasSuffix(s, "G"):
		mult, s = 1024*1024*1024, strings.TrimSuffix(strings.TrimSuffix(s, "GB"), "G")
	case strings.HasSuffix(s, "MB"), strings.HasSuffix(s, "M"):
		mult, s = 1024*1024, strings.TrimSuffix(strings.TrimSuffix(s, "MB"), "M")
	case strings.HasSuffix(s, "KB"), strings.HasSuffix(s, "K"):
		mult, s = 1024, strings.TrimSuffix(strings.TrimSuffix(s, "KB"), "K")
	default:
		return 0, fmt.Errorf("memory %q: want a size such as 16GB", s)
	}
	var n int64
	if _, err := fmt.Sscan(s, &n); err != nil || n <= 0 {
		return 0, fmt.Errorf("memory %q: want a positive size such as 16GB", s)
	}
	return n * mult, nil
}
