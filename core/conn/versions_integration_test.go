// SPDX-License-Identifier: Apache-2.0

//go:build integration

package conn_test

import (
	"testing"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/pgtest"
)

const (
	testUser     = pgtest.User
	testPassword = pgtest.Password
)

func pgVersions(t *testing.T) []int { t.Helper(); return pgtest.Versions(t) }

// startPG returns a superuser profile for a fresh postgres:<major>-alpine.
func startPG(t *testing.T, major int) conn.Profile {
	t.Helper()
	return pgtest.Start(t, major).Profile
}

func execAll(t *testing.T, p conn.Profile, stmts ...string) {
	t.Helper()
	pgtest.Exec(t, p, stmts...)
}

func as(p conn.Profile, user, password string) conn.Profile { return pgtest.As(p, user, password) }
