// SPDX-License-Identifier: Apache-2.0

package sqlpack

// Session pack (roadmap M1.4). conn.Connect runs these straight after
// connecting to apply an environment policy, then reads the result back.
// set_config(..., false) changes the session default, so it holds for every
// later transaction on the connection. Values are $1 parameters.

// SessionReadOnly makes later transactions read-only by default.
var SessionReadOnly = Query{
	Name:       "session.read_only",
	Doc:        "Sets default_transaction_read_only = on for the session (environment policy).",
	Sources:    []string{"set_config()"},
	MinVersion: PG14,
	SQL:        `SELECT set_config('default_transaction_read_only', 'on', false)`,
}

// SessionStatementTimeout sets statement_timeout; $1 is milliseconds as text.
var SessionStatementTimeout = Query{
	Name:       "session.statement_timeout",
	Doc:        "Sets statement_timeout for the session; $1 is the timeout in milliseconds.",
	Sources:    []string{"set_config()"},
	MinVersion: PG14,
	SQL:        `SELECT set_config('statement_timeout', $1, false)`,
}

// SessionLockTimeout sets lock_timeout; $1 is milliseconds as text.
var SessionLockTimeout = Query{
	Name:       "session.lock_timeout",
	Doc:        "Sets lock_timeout for the session; $1 is the timeout in milliseconds.",
	Sources:    []string{"set_config()"},
	MinVersion: PG14,
	SQL:        `SELECT set_config('lock_timeout', $1, false)`,
}

// SessionVerify reads the effective values back, in pg_settings' base units
// (milliseconds for the timeouts): read_only ('on'/'off'),
// statement_timeout_ms, lock_timeout_ms.
var SessionVerify = Query{
	Name:       "session.verify",
	Doc:        "Reads back default_transaction_read_only, statement_timeout and lock_timeout as applied to the session.",
	Sources:    []string{"pg_settings"},
	MinVersion: PG14,
	SQL: `SELECT
	(SELECT setting FROM pg_settings WHERE name = 'default_transaction_read_only') AS read_only,
	(SELECT setting::bigint FROM pg_settings WHERE name = 'statement_timeout') AS statement_timeout_ms,
	(SELECT setting::bigint FROM pg_settings WHERE name = 'lock_timeout') AS lock_timeout_ms`,
}
