// SPDX-License-Identifier: Apache-2.0

// Package schema reads the structure of a PostgreSQL database and diffs
// two such snapshots. It records names and definitions only. It does not
// read table rows, and it does not change the database.
package schema
