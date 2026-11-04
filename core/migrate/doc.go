// SPDX-License-Identifier: Apache-2.0

// Package migrate reads a folder of SQL migrations and reports what applying
// them would do. It recognises a plain SQL directory, a Flyway directory and
// a Sqitch project. Apply does not open a connection and does not start
// Flyway or Sqitch.
package migrate
