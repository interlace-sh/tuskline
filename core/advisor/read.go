// SPDX-License-Identifier: Apache-2.0

package advisor

import (
	"context"
	"strconv"
	"strings"

	"github.com/interlace-sh/tuskline/core/activity"
	"github.com/interlace-sh/tuskline/core/sqlpack"
)

// settingNames are the rows Collect reads. A name that does not exist
// on this server is simply absent.
func settingNames() []string {
	return []string{
		"autovacuum",
		"autovacuum_max_workers",
		"autovacuum_vacuum_cost_limit",
		"autovacuum_work_mem",
		"checkpoint_completion_target",
		"default_statistics_target",
		"effective_cache_size",
		"effective_io_concurrency",
		"huge_pages",
		"idle_in_transaction_session_timeout",
		"io_method",
		"jit",
		"maintenance_io_concurrency",
		"maintenance_work_mem",
		"max_connections",
		"max_parallel_maintenance_workers",
		"max_parallel_workers",
		"max_parallel_workers_per_gather",
		"max_wal_size",
		"max_worker_processes",
		"min_wal_size",
		"random_page_cost",
		"shared_buffers",
		"track_io_timing",
		"wal_buffers",
		"wal_compression",
		"work_mem",
	}
}

// Collect reads the settings and the optional evidence. A failure to
// read a statistics view leaves that evidence unset. A failure to read
// pg_settings is returned.
func Collect(ctx context.Context, q activity.Querier, versionNum int) (map[string]Observed, Evidence, error) {
	rows, err := q.Query(ctx, sqlpack.AdviseSettings.SQL, settingNames())
	if err != nil {
		return nil, Evidence{}, err
	}
	defer rows.Close()
	current := map[string]Observed{}
	for rows.Next() {
		var name, setting, unit, context, enums string
		if err := rows.Scan(&name, &setting, &unit, &context, &enums); err != nil {
			return nil, Evidence{}, err
		}
		obs := parseSetting(setting, unit, context, enums)
		current[name] = obs
	}
	if err := rows.Err(); err != nil {
		return nil, Evidence{}, err
	}
	return current, evidence(ctx, q, versionNum), nil
}

func evidence(ctx context.Context, q activity.Querier, versionNum int) Evidence {
	var ev Evidence
	if rows, err := q.Query(ctx, sqlpack.AdviseCache.SQL); err == nil {
		if rows.Next() {
			var hit, read int64
			if err := rows.Scan(&hit, &read); err == nil && hit+read > 0 {
				ev.HaveCache = true
				ev.CacheHit = float64(hit) / float64(hit+read)
			}
		}
		rows.Close()
	}
	if rows, err := q.Query(ctx, sqlpack.AdviseTemp.SQL); err == nil {
		if rows.Next() {
			var n int64
			if err := rows.Scan(&n); err == nil {
				ev.HaveTemp = true
				ev.TempBytes = n
			}
		}
		rows.Close()
	}
	query, err := sqlpack.Pick(versionNum, sqlpack.AdviseCheckpoints14, sqlpack.AdviseCheckpoints17)
	if err != nil {
		return ev
	}
	if rows, err := q.Query(ctx, query.SQL); err == nil {
		if rows.Next() {
			var timed, requested int64
			if err := rows.Scan(&timed, &requested); err == nil {
				ev.HaveCheckpoints = true
				ev.CheckpointsTimed = timed
				ev.CheckpointsRequested = requested
			}
		}
		rows.Close()
	}
	return ev
}

func parseSetting(setting, unit, context, enums string) Observed {
	obs := Observed{Text: setting, Context: context, Enums: parseEnums(enums)}
	n, err := strconv.ParseInt(setting, 10, 64)
	switch unit {
	case "B":
		if err == nil && n >= 0 {
			obs.KB = n / kb
			obs.HasKB = true
			obs.Text = FormatKB(obs.KB)
		}
	case "8kB":
		if err == nil && n >= 0 {
			obs.KB = n * 8
			obs.HasKB = true
			obs.Text = FormatKB(obs.KB)
		}
	case "kB":
		if err == nil && n >= 0 {
			obs.KB = n
			obs.HasKB = true
			obs.Text = FormatKB(obs.KB)
		}
	case "MB":
		if err == nil && n >= 0 {
			obs.KB = n * 1024
			obs.HasKB = true
			obs.Text = FormatKB(obs.KB)
		}
	case "GB":
		if err == nil && n >= 0 {
			obs.KB = n * 1024 * 1024
			obs.HasKB = true
			obs.Text = FormatKB(obs.KB)
		}
	case "ms":
		if err == nil {
			obs.HasInt = true
			obs.Int = int(n)
			obs.Text = setting
		}
	default:
		if f, ferr := strconv.ParseFloat(setting, 64); ferr == nil {
			obs.HasReal = true
			obs.Real = f
		}
		if err == nil && !strings.Contains(setting, ".") {
			obs.HasInt = true
			obs.Int = int(n)
		}
	}
	return obs
}

func parseEnums(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "{}")
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.Trim(part, ` "`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
