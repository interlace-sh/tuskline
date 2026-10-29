// SPDX-License-Identifier: Apache-2.0

package sqlpack

import (
	"errors"
	"fmt"
)

// Query is one documented, read-only SQL statement. Values are always
// passed as $n parameters; queries never interpolate input.
type Query struct {
	// Name is unique within the pack, for example "probe.server".
	Name string
	// Doc says what the query returns and why it is safe to run.
	Doc string
	// Sources lists the views, catalogues and functions it reads.
	Sources []string
	// MinVersion is the lowest server_version_num it runs on (inclusive).
	MinVersion int
	// MaxVersion is the first server_version_num it no longer runs on
	// (exclusive); zero means no upper bound.
	MaxVersion int
	// Extension names an extension the query needs, or "".
	Extension string
	// SQL is the statement text.
	SQL string
}

// Supports reports whether q runs on versionNum.
func (q Query) Supports(versionNum int) bool {
	return versionNum >= q.MinVersion && (q.MaxVersion == 0 || versionNum < q.MaxVersion)
}

// ErrNoVariant is returned when no variant of a query supports a server.
var ErrNoVariant = errors.New("no query variant for this server version")

// Pick returns the first variant that supports versionNum.
func Pick(versionNum int, variants ...Query) (Query, error) {
	for _, q := range variants {
		if q.Supports(versionNum) {
			return q, nil
		}
	}
	name := ""
	if len(variants) > 0 {
		name = variants[0].Name
	}
	return Query{}, fmt.Errorf("%s on server_version_num %d: %w", name, versionNum, ErrNoVariant)
}

// PG14 is server_version_num for PostgreSQL 14.0, the minimum supported.
const PG14 = 140000

// All returns every query in every pack, for documentation and tests.
func All() []Query {
	return []Query{
		ProbeServer, ProbeExtensions, ProbeRole,
		SessionReadOnly, SessionStatementTimeout, SessionLockTimeout, SessionVerify,
		ActivitySessions, ActivityLocks,
		StatsDatabase,
		StatsTables14, StatsTables16, StatsTables18,
		StatsWAL14, StatsWAL18,
		StatsBgwriter14, StatsBgwriter17,
		StatsCheckpointer17, StatsCheckpointer18,
		StatsIO16, StatsIO18,
		StatsStatements14, StatsStatements15, StatsStatements17,
		DiagnoseStatementsInstalled, DiagnoseWraparound, DiagnoseIndexes, DiagnoseReplication,
	}
}
