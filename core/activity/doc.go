// SPDX-License-Identifier: Apache-2.0

// Package activity takes live snapshots of sessions, waits, locks and
// blocking chains with the activity pack in core/sqlpack. It is what
// `tl top` and the activity screen show. Snapshots are read-only and kept
// in memory only; query text is never persisted.
package activity
