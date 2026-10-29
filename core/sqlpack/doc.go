// SPDX-License-Identifier: Apache-2.0

// Package sqlpack holds every SQL statement Tuskline runs against a target,
// as documented, version-aware Query values for PostgreSQL 14-18. Callers
// choose a variant with Pick by server_version_num. Each query names its
// source views and any extension it needs. Packs: probe (M1.2); activity,
// waits and locks (M1.7); statistics and metrics (M2).
package sqlpack
