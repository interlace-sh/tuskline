// SPDX-License-Identifier: Apache-2.0

// Package store is the embedded metadata database: SQLite through a pure-Go
// driver, so the server stays free of cgo. Metrics history is not stored
// here; that is the Parquet segments in ADR 0013 (roadmap M2.4b).
package store
