// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/sqlpack"
)

// SetupTimeout bounds connecting plus applying the session settings.
const SetupTimeout = 10 * time.Second

// SessionSettings are the session defaults an environment policy imposes
// (inventory.Policy.Session). Zero values impose nothing: a false ReadOnly
// does not switch read-only off, so a role that is read-only by default
// stays that way.
type SessionSettings struct {
	ReadOnly         bool
	StatementTimeout time.Duration
	LockTimeout      time.Duration
}

// ErrSessionNotApplied is returned when the server does not report the
// requested settings after they were applied, for example behind a pooler
// that resets session state. The connection is closed: Tuskline fails
// closed rather than run without the policy.
var ErrSessionNotApplied = errors.New("session policy was not applied")

// Connect opens a connection with p and applies s before returning it, so
// no statement runs before the policy holds. The caller owns the connection.
// Errors never contain the password.
func Connect(ctx context.Context, p Profile, s SessionSettings) (*pgx.Conn, error) {
	cfg, err := p.Config()
	if err != nil {
		return nil, err
	}
	sctx, cancel := context.WithTimeout(ctx, SetupTimeout)
	defer cancel()
	c, err := pgx.ConnectConfig(sctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", p, err)
	}
	if err := ApplySession(sctx, c, s); err != nil {
		_ = c.Close(context.Background()) // the setup error is what matters
		return nil, fmt.Errorf("connect %s: %w", p, err)
	}
	return c, nil
}

// ApplySession applies s to an open connection and verifies it took effect.
func ApplySession(ctx context.Context, c *pgx.Conn, s SessionSettings) error {
	if s.ReadOnly {
		if _, err := c.Exec(ctx, sqlpack.SessionReadOnly.SQL); err != nil {
			return fmt.Errorf("apply session read-only: %w", err)
		}
	}
	if s.StatementTimeout > 0 {
		if _, err := c.Exec(ctx, sqlpack.SessionStatementTimeout.SQL, millis(s.StatementTimeout)); err != nil {
			return fmt.Errorf("apply statement_timeout: %w", err)
		}
	}
	if s.LockTimeout > 0 {
		if _, err := c.Exec(ctx, sqlpack.SessionLockTimeout.SQL, millis(s.LockTimeout)); err != nil {
			return fmt.Errorf("apply lock_timeout: %w", err)
		}
	}
	var got appliedSettings
	if err := c.QueryRow(ctx, sqlpack.SessionVerify.SQL).Scan(&got.readOnly, &got.statementTimeoutMS, &got.lockTimeoutMS); err != nil {
		return fmt.Errorf("verify session settings: %w", err)
	}
	return got.check(s)
}

type appliedSettings struct {
	readOnly           string
	statementTimeoutMS int64
	lockTimeoutMS      int64
}

// check compares what the server reports with what s requires.
func (a appliedSettings) check(s SessionSettings) error {
	var errs []error
	if s.ReadOnly && a.readOnly != "on" {
		errs = append(errs, fmt.Errorf("default_transaction_read_only is %q, want on", a.readOnly))
	}
	if want := ms(s.StatementTimeout); want > 0 && a.statementTimeoutMS != want {
		errs = append(errs, fmt.Errorf("statement_timeout is %d ms, want %d ms", a.statementTimeoutMS, want))
	}
	if want := ms(s.LockTimeout); want > 0 && a.lockTimeoutMS != want {
		errs = append(errs, fmt.Errorf("lock_timeout is %d ms, want %d ms", a.lockTimeoutMS, want))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrSessionNotApplied, errors.Join(errs...))
	}
	return nil
}

// ms converts d to whole milliseconds, rounding a positive sub-millisecond
// value up so a tiny timeout never becomes 0 ("no timeout").
func ms(d time.Duration) int64 {
	n := d.Milliseconds()
	if d > 0 && n == 0 {
		n = 1
	}
	return n
}

// millis renders d for set_config.
func millis(d time.Duration) string { return strconv.FormatInt(ms(d), 10) }
