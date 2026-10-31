// SPDX-License-Identifier: Apache-2.0

// Package catalog is the extension catalogue and the monitoring-role script.
// Each entry records what the extension unlocks, which PostgreSQL 14–18
// builds exist, whether it needs a preload and a restart, the PGDG package,
// and which managed providers allow it. Steps merge shared_preload_libraries
// with the current list and use patronictl when Patroni owns the
// configuration. MonitorPlan previews the tuskline_monitor role and does
// not set a password.
package catalog
