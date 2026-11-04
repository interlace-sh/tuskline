// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"errors"
	"fmt"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
	pg_query "github.com/wasilibs/go-pgquery"
)

// ErrMixedScript is returned when one file contains a statement that cannot
// run inside a transaction and a statement that can. The file has to be
// split, because a failure in the middle would leave part of it applied.
var ErrMixedScript = errors.New("script mixes statements that cannot share a transaction")

// prepared is one script split into statements. outside is true when none
// of them can run inside a transaction block.
type prepared struct {
	file    File
	stmts   []string
	outside bool
}

func prepare(file File) (prepared, error) {
	parts, err := splitSQL(file.SQL)
	if err != nil {
		return prepared{}, fmt.Errorf("%s: %w", file.Name, err)
	}
	var stmts []string
	outsideN := 0
	for _, part := range parts {
		tree, err := pg_query.Parse(part)
		if err != nil {
			return prepared{}, fmt.Errorf("%s: %w", file.Name, err)
		}
		if len(tree.GetStmts()) == 0 {
			continue
		}
		outside := false
		for _, raw := range tree.GetStmts() {
			if outsideTransaction(raw.GetStmt()) {
				outside = true
				break
			}
		}
		if outside {
			outsideN++
		}
		stmts = append(stmts, part)
	}
	if outsideN > 0 && outsideN != len(stmts) {
		return prepared{}, fmt.Errorf("%w: %s", ErrMixedScript, file.Name)
	}
	return prepared{file: file, stmts: stmts, outside: outsideN > 0}, nil
}

func splitSQL(sql string) ([]string, error) {
	scanned, err := pg_query.Scan(sql)
	if err != nil {
		return nil, err
	}
	var out []string
	start := 0
	for _, tok := range scanned.GetTokens() {
		if tok.GetToken() != pgquery.Token_ASCII_59 {
			continue
		}
		end := int(tok.GetStart())
		if end < start || end > len(sql) {
			return nil, fmt.Errorf("statement boundary %d is outside the script", end)
		}
		if stmt := strings.TrimSpace(sql[start:end]); stmt != "" {
			out = append(out, stmt)
		}
		start = int(tok.GetEnd())
		if start > len(sql) {
			return nil, fmt.Errorf("statement boundary %d is outside the script", start)
		}
	}
	if tail := strings.TrimSpace(sql[start:]); tail != "" {
		out = append(out, tail)
	}
	return out, nil
}

// outsideTransaction reports statements PostgreSQL refuses inside a
// transaction block. REINDEX CONCURRENTLY arrives as a parameter named
// concurrent rather than a flag on the statement.
func outsideTransaction(stmt *pgquery.Node) bool {
	if stmt == nil {
		return false
	}
	if s := stmt.GetIndexStmt(); s != nil && s.GetConcurrent() {
		return true
	}
	if s := stmt.GetDropStmt(); s != nil && s.GetConcurrent() {
		return true
	}
	if s := stmt.GetRefreshMatViewStmt(); s != nil && s.GetConcurrent() {
		return true
	}
	if s := stmt.GetVacuumStmt(); s != nil && s.GetIsVacuumcmd() {
		return true
	}
	if stmt.GetCreatedbStmt() != nil || stmt.GetDropdbStmt() != nil || stmt.GetAlterSystemStmt() != nil {
		return true
	}
	if s := stmt.GetReindexStmt(); s != nil {
		for _, param := range s.GetParams() {
			def := param.GetDefElem()
			if def != nil && strings.EqualFold(def.GetDefname(), "concurrent") {
				return true
			}
		}
	}
	return false
}
