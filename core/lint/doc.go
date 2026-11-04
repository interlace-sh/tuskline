// SPDX-License-Identifier: Apache-2.0

// Package lint reports migration statements that lock or rewrite a table.
// Parsing uses the PostgreSQL parser through a pure-Go build, so the core
// stays free of cgo. Findings are a preview: nothing here connects or runs SQL.
package lint
