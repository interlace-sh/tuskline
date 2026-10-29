// SPDX-License-Identifier: Apache-2.0

package analyser

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/interlace-sh/tuskline/core/activity"
	"github.com/interlace-sh/tuskline/core/sqlpack"
)

// Report is one diagnose pass. Findings is empty when every check was clean.
type Report struct {
	VersionNum int       `json:"server_version_num"`
	Findings   []Finding `json:"findings"`
}

// Diagnose runs the health checks against q. versionNum selects the
// statistics-pack variant. lag is the replication lag that counts as a
// finding. A check the role may not read becomes an info finding; any
// other query error fails the report.
func Diagnose(ctx context.Context, q activity.Querier, versionNum int, lag time.Duration) (Report, error) {
	report := Report{VersionNum: versionNum, Findings: []Finding{}}
	var skipped []string

	installed, err := statementsInstalled(ctx, q)
	if err != nil {
		if denied(err) {
			skipped = append(skipped, "pg_extension")
		} else {
			return Report{}, err
		}
	} else if f := MissingStatements(installed); f != nil {
		report.Findings = append(report.Findings, *f)
	}

	snap, err := activity.Take(ctx, q, versionNum)
	if err != nil {
		if denied(err) {
			skipped = append(skipped, "pg_stat_activity")
		} else {
			return Report{}, err
		}
	} else {
		for _, s := range snap.Sessions {
			if s.State != "idle in transaction" && s.State != "idle in transaction (aborted)" {
				continue
			}
			if f := IdleInTransaction(s.PID, s.User, s.Query, s.StateAge); f != nil {
				report.Findings = append(report.Findings, *f)
			}
		}
	}

	if err := databases(ctx, q, &report); err != nil {
		if denied(err) {
			skipped = append(skipped, "pg_stat_database")
		} else {
			return Report{}, err
		}
	}
	if err := tables(ctx, q, versionNum, &report); err != nil {
		if denied(err) {
			skipped = append(skipped, "pg_stat_user_tables")
		} else {
			return Report{}, err
		}
	}
	if err := wraparound(ctx, q, &report); err != nil {
		if denied(err) {
			skipped = append(skipped, "pg_database")
		} else {
			return Report{}, err
		}
	}
	if err := indexes(ctx, q, &report); err != nil {
		if denied(err) {
			skipped = append(skipped, "pg_index")
		} else {
			return Report{}, err
		}
	}
	if err := replication(ctx, q, lag, &report); err != nil {
		if denied(err) {
			skipped = append(skipped, "pg_stat_replication")
		} else {
			return Report{}, err
		}
	}
	if len(skipped) > 0 {
		report.Findings = append(report.Findings, Finding{
			Check: "privilege", Severity: SeverityInfo,
			Summary:  "Some checks were skipped because the login role cannot read them",
			Evidence: fmt.Sprintf("denied: %s", join(skipped)),
			Fix:      "Grant pg_monitor to the login role. The monitoring role tuskline_monitor includes it.",
		})
	}
	return report, nil
}

func statementsInstalled(ctx context.Context, q activity.Querier) (bool, error) {
	rows, err := read(ctx, q, sqlpack.DiagnoseStatementsInstalled.SQL)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	installed, _ := rows[0]["installed"].(bool)
	return installed, nil
}

func databases(ctx context.Context, q activity.Querier, report *Report) error {
	rows, err := read(ctx, q, sqlpack.StatsDatabase.SQL)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if f := CacheHit(asString(row["datname"]), asInt(row["blks_hit"]), asInt(row["blks_read"])); f != nil {
			report.Findings = append(report.Findings, *f)
		}
	}
	return nil
}

func tables(ctx context.Context, q activity.Querier, versionNum int, report *Report) error {
	query, err := sqlpack.Pick(versionNum, sqlpack.StatsTables14, sqlpack.StatsTables16, sqlpack.StatsTables18)
	if err != nil {
		return err
	}
	rows, err := read(ctx, q, query.SQL, 1000)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if f := Bloat(asString(row["schemaname"]), asString(row["relname"]), asInt(row["n_live_tup"]), asInt(row["n_dead_tup"])); f != nil {
			report.Findings = append(report.Findings, *f)
		}
	}
	return nil
}

func wraparound(ctx context.Context, q activity.Querier, report *Report) error {
	rows, err := read(ctx, q, sqlpack.DiagnoseWraparound.SQL)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if f := Wraparound(asString(row["datname"]), asInt(row["xid_age"])); f != nil {
			report.Findings = append(report.Findings, *f)
		}
	}
	return nil
}

func indexes(ctx context.Context, q activity.Querier, report *Report) error {
	rows, err := read(ctx, q, sqlpack.DiagnoseIndexes.SQL)
	if err != nil {
		return err
	}
	indexes := make([]Index, 0, len(rows))
	for _, row := range rows {
		indexes = append(indexes, Index{
			Schema: asString(row["schema"]), Table: asString(row["table_name"]), Name: asString(row["index_name"]),
			Key: asString(row["index_key"]), Predicate: asString(row["predicate"]),
			Scans: asInt(row["idx_scan"]), Bytes: asInt(row["bytes"]), Unique: asBool(row["is_unique"]),
		})
	}
	report.Findings = append(report.Findings, UnusedIndexes(indexes)...)
	report.Findings = append(report.Findings, DuplicateIndexes(indexes)...)
	return nil
}

func replication(ctx context.Context, q activity.Querier, lag time.Duration, report *Report) error {
	rows, err := read(ctx, q, sqlpack.DiagnoseReplication.SQL)
	if err != nil {
		return err
	}
	for _, row := range rows {
		seconds, ok := row["replay_seconds"].(float64)
		if !ok {
			continue
		}
		name := asString(row["application_name"])
		if name == "" {
			name = "standby"
		}
		if f := ReplicationLag(name, time.Duration(seconds*float64(time.Second)), lag); f != nil {
			report.Findings = append(report.Findings, *f)
		}
	}
	return nil
}

func read(ctx context.Context, q activity.Querier, sql string, args ...any) ([]map[string]any, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fields := rows.FieldDescriptions()
	var out []map[string]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(fields))
		for i, fd := range fields {
			row[fd.Name] = vals[i]
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func denied(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "42501"
}

func join(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += ", " + p
	}
	return out
}

func asInt(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}
