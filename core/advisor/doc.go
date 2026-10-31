// SPDX-License-Identifier: Apache-2.0

// Package advisor compares a PostgreSQL configuration with a starting
// point and reports only the settings that should change. The formulas
// follow PgTune; the decision of when to speak, and which PgTune
// suggestions to refuse, is ADR 0014. Advice is a preview and is never
// applied.
package advisor
