// SPDX-License-Identifier: Apache-2.0

package advisor

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Kilobytes, matching PgTune's internal unit.
const (
	kb = 1024
	mb = 1024 * kb
	gb = 1024 * mb

	// minWorkMemKB is the floor under which a sort spills for a trivial query.
	minWorkMemKB int64 = 4 * mb / kb
)

// Hardware is the part of a machine SQL cannot see. Zero Memory skips
// every formula that needs RAM. Empty Workload means web once Memory
// is set, and "do not assume a workload" otherwise.
type Hardware struct {
	Memory      int64 // bytes
	CPUs        int
	Storage     string // hdd, ssd, nvme, san
	Workload    string // web, oltp, dw, desktop, mixed
	DBSize      string // less, mid, greater
	OS          string // linux, windows, mac
	Connections int    // overrides the connection count used for work_mem
	Pooler      bool
}

// Param is one candidate setting. KB is set for memory values.
type Param struct {
	Name  string
	Text  string
	Kind  string // memory, int, real, enum
	KB    int64
	Real  float64
	Int   int
	Apply string // restart or reload
	Why   string
}

// Plan is the candidate configuration for one server.
type Plan struct {
	Params []Param
	// Connections is the count work_mem was divided by.
	Connections int
	// ConnectionFit is how many connections leave 4MB of work_mem each.
	// Zero when RAM is unknown.
	ConnectionFit int
	SharedKB      int64
	Workload      string
	Storage       string
	DBSize        string
	OS            string
	Memory        int64
	CPUs          int
	Pooler        bool
}

// Targets computes the candidate settings. versionNum is
// server_version_num. currentMax is the live max_connections, or 0.
func Targets(versionNum int, hw Hardware, currentMax int) (Plan, error) {
	major := versionNum / 10000
	if major < 14 || major > 18 {
		return Plan{}, fmt.Errorf("PostgreSQL %d is outside 14–18", major)
	}
	hw, err := normalise(hw)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		Workload: hw.Workload, Storage: hw.Storage, DBSize: hw.DBSize,
		OS: hw.OS, Memory: hw.Memory, CPUs: hw.CPUs, Pooler: hw.Pooler,
	}
	conns := hw.Connections
	if conns == 0 {
		conns = workloadConnections(hw.Workload)
		if currentMax > conns {
			conns = currentMax
		}
	}
	if hw.Memory > 0 && conns == 0 {
		conns = workloadConnections(hw.Workload)
	}
	plan.Connections = conns

	var shared int64
	if hw.Memory > 0 {
		shared = sharedBuffers(hw)
		plan.SharedKB = shared
		plan.Params = append(plan.Params,
			memParam("shared_buffers", shared, "restart", sharedWhy(hw)),
			memParam("effective_cache_size", effectiveCache(hw), "reload",
				"This does not allocate memory. It tells the planner how large the cache is, including shared buffers and the operating system cache."),
			memParam("maintenance_work_mem", maintenanceWorkMem(hw, major), "reload",
				"Maintenance memory is a fraction of RAM, capped so one rebuild cannot take the machine."),
			memParam("work_mem", workMem(hw, shared, conns, parallelWorkers(hw)), "reload",
				"Sized from the RAM left after shared buffers, divided by connections and parallel workers, with a margin so a spike does not exhaust the machine."),
		)
		plan.ConnectionFit = connectionFit(hw.Memory/kb, shared, parallelWorkers(hw))
		if hw.CPUs >= 16 {
			workers := autovacuumWorkers(hw.CPUs)
			plan.Params = append(plan.Params,
				intParam("autovacuum_max_workers", workers, autovacuumWorkersApply(major),
					"The default of 3 workers falls behind on a machine with many CPUs. Each worker still has to be given a share of the cost limit."),
				intParam("autovacuum_vacuum_cost_limit", costLimit(workers), "reload",
					"The cost limit is shared by every autovacuum worker. Raising the worker count without it makes each worker slower."),
			)
			if maintenanceWorkMem(hw, major) > gb/kb {
				plan.Params = append(plan.Params, memParam("autovacuum_work_mem", gb/kb, "reload",
					"Autovacuum would otherwise inherit maintenance_work_mem. Several workers at that size can exhaust RAM, so each is capped at 1GB."))
			}
		}
		if hw.Workload == "dw" {
			plan.Params = append(plan.Params, intParam("default_statistics_target", 500, "reload",
				"A warehouse reads large tables. A higher statistics target gives the planner a better picture of them."))
		}
	}
	if hw.Storage != "" {
		plan.Params = append(plan.Params, realParam("random_page_cost", randomPageCost(hw), "reload", randomWhy(hw)))
		if n, ok := ioConcurrency(hw, major); ok {
			plan.Params = append(plan.Params,
				intParam("effective_io_concurrency", n, "reload",
					"How many concurrent reads the planner may expect. Solid-state storage and a SAN can keep more than one in flight."),
			)
			if major >= 13 {
				plan.Params = append(plan.Params, intParam("maintenance_io_concurrency", n, "reload",
					"Vacuum and index builds use the same storage, so they get the same concurrency as ordinary reads."))
			}
		}
	}
	if hw.CPUs >= 4 {
		plan.Params = append(plan.Params, parallelParams(hw, major)...)
	}
	if hw.Workload == "web" || hw.Workload == "oltp" || hw.Workload == "mixed" {
		plan.Params = append(plan.Params, enumParam("jit", "off", "reload",
			"JIT compilation spends more time planning a short query than it saves executing it. Leave it for a warehouse."))
	}
	if major >= 18 && hw.OS == "linux" {
		plan.Params = append(plan.Params, enumParam("io_method", "io_uring", "restart",
			"PostgreSQL 18 can submit I/O with io_uring. It is only suggested when this build lists the value."))
	}
	return plan, nil
}

func normalise(hw Hardware) (Hardware, error) {
	hw.Storage = strings.ToLower(strings.TrimSpace(hw.Storage))
	hw.Workload = strings.ToLower(strings.TrimSpace(hw.Workload))
	hw.DBSize = strings.ToLower(strings.TrimSpace(hw.DBSize))
	hw.OS = strings.ToLower(strings.TrimSpace(hw.OS))
	switch hw.DBSize {
	case "", "mid", "mid_ram":
		hw.DBSize = "mid"
	case "less", "less_ram":
		hw.DBSize = "less"
	case "greater", "greater_ram":
		hw.DBSize = "greater"
	default:
		return hw, fmt.Errorf("database size %q: want less, mid or greater", hw.DBSize)
	}
	switch hw.Storage {
	case "", "hdd", "ssd", "nvme", "san":
	default:
		return hw, fmt.Errorf("storage %q: want hdd, ssd, nvme or san", hw.Storage)
	}
	switch hw.Workload {
	case "", "web", "oltp", "dw", "desktop", "mixed":
	default:
		return hw, fmt.Errorf("workload %q: want web, oltp, dw, desktop or mixed", hw.Workload)
	}
	switch hw.OS {
	case "", "linux", "windows", "mac":
	default:
		return hw, fmt.Errorf("os %q: want linux, windows or mac", hw.OS)
	}
	if hw.OS == "" {
		hw.OS = "linux"
	}
	if hw.Memory > 0 && hw.Workload == "" {
		hw.Workload = "web"
	}
	if hw.Memory < 0 || hw.CPUs < 0 || hw.Connections < 0 {
		return hw, fmt.Errorf("memory, cpus and connections must be zero or positive")
	}
	return hw, nil
}

func sharedBuffers(hw Hardware) int64 {
	ram := hw.Memory / kb
	div := int64(4)
	if hw.Workload == "desktop" {
		div = 16
	}
	return roundMB(ram / div)
}

func sharedWhy(hw Hardware) string {
	if hw.Workload == "desktop" {
		return "A desktop install shares the machine, so the starting point is one sixteenth of RAM rather than a quarter."
	}
	return "25% of RAM is the starting point on a dedicated server. More than 40% is unlikely to help, because PostgreSQL also uses the operating system cache."
}

func effectiveCache(hw Hardware) int64 {
	ram := hw.Memory / kb
	if hw.Workload == "desktop" {
		return roundMB(ram / 4)
	}
	return roundMB(ram * 3 / 4)
}

func maintenanceWorkMem(hw Hardware, major int) int64 {
	ram := hw.Memory / kb
	div := int64(16)
	if hw.Workload == "dw" {
		div = 8
	}
	mem := roundMB(ram / div)
	limit := int64(8 * gb / kb)
	if hw.OS == "windows" && major <= 17 {
		limit = 2*gb/kb - mb/kb
	}
	if mem > limit {
		return limit
	}
	return mem
}

func workMem(hw Hardware, shared int64, conns, workers int) int64 {
	if conns < 1 {
		conns = 1
	}
	budget := float64(hw.Memory/kb-shared) / float64((conns+workers)*3)
	var mem float64
	switch hw.Workload {
	case "dw", "mixed":
		mem = budget / 2
	case "desktop":
		mem = budget / 6
	default:
		mem = budget
	}
	switch hw.DBSize {
	case "less":
		mem *= 1.3
	case "greater":
		mem *= 0.9
	}
	out := roundMB(int64(math.Floor(mem)))
	if out < minWorkMemKB {
		return minWorkMemKB
	}
	if hw.OS == "windows" && out > 2*gb/kb-mb/kb {
		return 2*gb/kb - mb/kb
	}
	return out
}

func connectionFit(ramKB, shared int64, workers int) int {
	if ramKB <= shared {
		return 0
	}
	fit := int((ramKB-shared)/(minWorkMemKB*3)) - workers
	if fit < 100 {
		return 100
	}
	return fit
}

func parallelWorkers(hw Hardware) int {
	if hw.CPUs < 4 {
		return 8 // PostgreSQL's own default, used when we are not changing it
	}
	if hw.CPUs > 1024 {
		return 1024
	}
	return hw.CPUs
}

func parallelParams(hw Hardware, major int) []Param {
	workers := parallelWorkers(hw)
	perGather := (workers + 1) / 2
	if hw.Workload == "dw" {
		keep := workers / 4
		if keep < 4 {
			keep = 4
		}
		if perGather > keep {
			perGather = keep
		}
	} else if perGather > 4 {
		perGather = 4
	}
	maint := (workers + 1) / 2
	if maint > 4 {
		maint = 4
	}
	procs := workers
	if procs < 8 {
		procs = 8
	}
	out := []Param{
		intParam("max_worker_processes", procs, "restart",
			"Background workers, including parallel query, share this pool. It is never set below the default of 8."),
		intParam("max_parallel_workers_per_gather", perGather, "reload",
			"How many workers one query may use. Extra workers past four rarely pay for themselves on short queries."),
	}
	if major >= 10 {
		out = append(out, intParam("max_parallel_workers", workers, "reload",
			"The total number of parallel workers at once. Sized to the CPUs PostgreSQL may use."))
	}
	if major >= 11 {
		out = append(out, intParam("max_parallel_maintenance_workers", maint, "reload",
			"Parallel vacuum and index builds. Capped at four, for the same reason as a single query."))
	}
	return out
}

func autovacuumWorkers(cpus int) int {
	if cpus >= 32 {
		return 5
	}
	if cpus >= 16 {
		return 4
	}
	return 3
}

func autovacuumWorkersApply(major int) string {
	if major >= 18 {
		return "reload"
	}
	return "restart"
}

func costLimit(workers int) int {
	return int(math.Round(float64(200*workers) / 3))
}

func randomPageCost(hw Hardware) float64 {
	if hw.DBSize == "less" {
		return 1.1
	}
	if hw.Storage == "hdd" || hw.Workload == "dw" {
		return 4
	}
	return 1.1
}

func randomWhy(hw Hardware) string {
	if hw.DBSize == "less" {
		return "The database fits in RAM, so a read is served from the operating system cache and a spinning disk's seek no longer dominates."
	}
	if hw.Workload == "dw" && hw.Storage != "hdd" {
		return "A warehouse larger than RAM keeps the default. A low random page cost makes a large index scan look cheaper than a sequential read."
	}
	if hw.Storage == "hdd" {
		return "A spinning disk still pays a seek on a random read, which is what the default cost describes."
	}
	return "Solid-state storage does not pay a spinning-disk seek, so a random read is close to a sequential one."
}

func ioConcurrency(hw Hardware, major int) (int, bool) {
	if hw.OS != "linux" && major < 18 {
		return 0, false
	}
	switch hw.Storage {
	case "hdd":
		return 2, true
	case "ssd":
		return 200, true
	case "san":
		return 300, true
	case "nvme":
		return 1000, true
	default:
		return 0, false
	}
}

func workloadConnections(workload string) int {
	switch workload {
	case "web":
		return 200
	case "oltp":
		return 300
	case "dw", "desktop", "mixed":
		return 100
	default:
		return 0
	}
}

func roundMB(valueKB int64) int64 {
	if valueKB < 0 {
		return 0
	}
	return (valueKB / 1024) * 1024
}

func memParam(name string, valueKB int64, apply, why string) Param {
	return Param{Name: name, Text: FormatKB(valueKB), Kind: "memory", KB: valueKB, Apply: apply, Why: why}
}

func intParam(name string, value int, apply, why string) Param {
	return Param{Name: name, Text: fmt.Sprintf("%d", value), Kind: "int", Int: value, Apply: apply, Why: why}
}

func realParam(name string, value float64, apply, why string) Param {
	return Param{Name: name, Text: formatReal(value), Kind: "real", Real: value, Apply: apply, Why: why}
}

func enumParam(name, value, apply, why string) Param {
	return Param{Name: name, Text: value, Kind: "enum", Apply: apply, Why: why}
}

func formatReal(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
