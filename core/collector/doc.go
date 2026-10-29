// SPDX-License-Identifier: Apache-2.0

// Package collector schedules read-only collection passes. It does not open
// database connections: the caller supplies query functions. A pass stops
// when its query-time budget is spent, each query gets a timeout no longer
// than the time still left, and a pass that hits that budget or finds the
// target overloaded waits longer before the next one.
package collector
