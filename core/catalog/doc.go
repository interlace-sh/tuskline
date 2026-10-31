// SPDX-License-Identifier: Apache-2.0

// Package catalog is the extension catalogue. Each entry records what the
// extension unlocks, which PostgreSQL 14–18 builds exist, whether it needs
// a preload and a restart, the PGDG package, and which managed providers
// allow it. Steps merge shared_preload_libraries with the current list and
// use patronictl when Patroni owns the configuration.
package catalog
