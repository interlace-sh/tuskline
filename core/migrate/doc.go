// SPDX-License-Identifier: Apache-2.0

// Package migrate reads a directory of SQL scripts and applies the ones
// the target database has not recorded. Scripts run in natural name order.
// Each applied script is stored, with its checksum, in
// tuskline.schema_migration. Apply does not run unless the caller asks.
package migrate
