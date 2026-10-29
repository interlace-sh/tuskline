// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrOverloaded means the target is too busy for another query. Collectors
// return it when the server reports saturation (a statement timeout, a
// cancelled lock wait, or a connection limit). The scheduler then stops
// the pass and backs off.
var ErrOverloaded = errors.New("target is overloaded")

// Query is one unit of work in a pass. Run must return when ctx is done.
type Query struct {
	// Name identifies the query in results and errors.
	Name string
	// Timeout caps this query. Zero means "whatever the budget has left".
	Timeout time.Duration
	// Run does the work.
	Run func(ctx context.Context) error
}

// Result is what one pass did. Ran lists queries that were called, including
// those that returned an error. Skipped lists queries that were not called.
type Result struct {
	Used    time.Duration
	Ran     []string
	Skipped []string
	// Err joins the errors from queries that ran. A skip is not an error.
	Err error
	// Overrun is true when the queries took longer than the budget.
	Overrun bool
	// Load is true when a query timed out or returned ErrOverloaded.
	Load bool
	// Stopped is true when the parent context was cancelled.
	Stopped bool
}

// Overloaded reports whether the next pass should wait longer.
func (r Result) Overloaded() bool { return r.Overrun || r.Load }

// RunPass calls queries in order until the budget (limit) is spent, a query
// reports load, or ctx is cancelled. now is read before and after each
// query so tests can advance it; nil means time.Now. Time spent is the
// sum of those spans, not the wall clock between queries.
func RunPass(ctx context.Context, limit time.Duration, queries []Query, now func() time.Time) Result {
	if now == nil {
		now = time.Now
	}
	var res Result
	var errs []error
	for i, q := range queries {
		if ctx.Err() != nil {
			res.Stopped = true
			res.Skipped = append(res.Skipped, names(queries[i:])...)
			break
		}
		remain := limit - res.Used
		if remain <= 0 {
			res.Skipped = append(res.Skipped, names(queries[i:])...)
			break
		}
		if q.Run == nil {
			errs = append(errs, fmt.Errorf("query %q has no function", q.Name))
			res.Skipped = append(res.Skipped, names(queries[i+1:])...)
			break
		}
		timeout := q.Timeout
		if timeout <= 0 || timeout > remain {
			timeout = remain
		}
		qctx, cancel := context.WithTimeout(ctx, timeout)
		start := now()
		err := q.Run(qctx)
		elapsed := now().Sub(start)
		cancel()
		if elapsed < 0 {
			elapsed = 0
		}
		res.Used += elapsed
		res.Ran = append(res.Ran, q.Name)
		if res.Used > limit {
			res.Overrun = true
		}
		switch {
		case ctx.Err() != nil:
			res.Stopped = true
		case errors.Is(err, ErrOverloaded) || errors.Is(err, context.DeadlineExceeded) || errors.Is(qctx.Err(), context.DeadlineExceeded):
			res.Load = true
		}
		if err != nil && !errors.Is(err, context.Canceled) {
			errs = append(errs, fmt.Errorf("%s: %w", q.Name, err))
		}
		if res.Stopped || res.Load || res.Overrun {
			res.Skipped = append(res.Skipped, names(queries[i+1:])...)
			break
		}
	}
	res.Err = errors.Join(errs...)
	return res
}

func names(qs []Query) []string {
	out := make([]string, len(qs))
	for i, q := range qs {
		out[i] = q.Name
	}
	return out
}
