// SPDX-License-Identifier: Apache-2.0

package hypo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	pg_query "github.com/wasilibs/go-pgquery"
)

// ErrUnavailable is returned when hypopg is not installed.
var ErrUnavailable = errors.New("hypopg is not installed")

// Result is the planner cost before and after the hypothetical index.
type Result struct {
	Before float64 `json:"before"`
	After  float64 `json:"after"`
}

// Compare plans query, creates indexSQL with hypopg, and plans query again.
// indexSQL must be one CREATE INDEX without CONCURRENTLY. query must be one
// SELECT. Neither statement is stored.
func Compare(ctx context.Context, db *pgx.Conn, indexSQL, query string) (Result, error) {
	if db == nil {
		return Result{}, fmt.Errorf("hypo: no connection")
	}
	if err := validate(indexSQL, query); err != nil {
		return Result{}, err
	}
	var installed bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'hypopg')`).Scan(&installed); err != nil {
		return Result{}, fmt.Errorf("hypo: %w", err)
	}
	if !installed {
		return Result{}, ErrUnavailable
	}
	before, err := planCost(ctx, db, query)
	if err != nil {
		return Result{}, err
	}
	if _, err := db.Exec(ctx, `SELECT hypopg_reset()`); err != nil {
		return Result{}, fmt.Errorf("hypo: reset: %w", err)
	}
	if _, err := db.Exec(ctx, `SELECT hypopg_create_index($1)`, strings.TrimSpace(indexSQL)); err != nil {
		return Result{}, fmt.Errorf("hypo: create: %w", err)
	}
	after, err := planCost(ctx, db, query)
	if err != nil {
		return Result{}, err
	}
	_, _ = db.Exec(ctx, `SELECT hypopg_reset()`)
	return Result{Before: before, After: after}, nil
}

func validate(indexSQL, query string) error {
	indexTree, err := pg_query.Parse(indexSQL)
	if err != nil {
		return fmt.Errorf("hypo: index: %w", err)
	}
	if len(indexTree.GetStmts()) != 1 || indexTree.GetStmts()[0].GetStmt().GetIndexStmt() == nil {
		return fmt.Errorf("hypo: index statement must be one CREATE INDEX")
	}
	if indexTree.GetStmts()[0].GetStmt().GetIndexStmt().GetConcurrent() {
		return fmt.Errorf("hypo: CONCURRENTLY is not hypothetical")
	}
	queryTree, err := pg_query.Parse(query)
	if err != nil {
		return fmt.Errorf("hypo: query: %w", err)
	}
	if len(queryTree.GetStmts()) != 1 || queryTree.GetStmts()[0].GetStmt().GetSelectStmt() == nil {
		return fmt.Errorf("hypo: query must be one SELECT")
	}
	return nil
}

func planCost(ctx context.Context, db *pgx.Conn, query string) (float64, error) {
	var raw []byte
	if err := db.QueryRow(ctx, `EXPLAIN (FORMAT JSON) `+strings.TrimRight(strings.TrimSpace(query), ";")).Scan(&raw); err != nil {
		return 0, fmt.Errorf("hypo: explain: %w", err)
	}
	var docs []struct {
		Plan struct {
			TotalCost float64 `json:"Total Cost"`
		} `json:"Plan"`
	}
	if err := json.Unmarshal(raw, &docs); err != nil || len(docs) == 0 {
		return 0, fmt.Errorf("hypo: explain json: %w", err)
	}
	return docs[0].Plan.TotalCost, nil
}
