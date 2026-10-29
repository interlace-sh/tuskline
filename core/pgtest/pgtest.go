// SPDX-License-Identifier: Apache-2.0

//go:build integration

package pgtest

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/interlace-sh/tuskline/core/conn"
)

// Throwaway credentials for disposable test containers only.
const (
	User     = "tuskline"
	Password = "tuskline-test-only"
)

// Versions returns the PostgreSQL majors to test: $PG_VERSIONS ("17",
// "14,18" or "14 15 16"), or 14 to 18.
func Versions(t *testing.T) []int {
	t.Helper()
	s := os.Getenv("PG_VERSIONS")
	if s == "" {
		return []int{14, 15, 16, 17, 18}
	}
	var out []int
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			t.Fatalf("PG_VERSIONS: %v", err)
		}
		out = append(out, n)
	}
	return out
}

// Server is a running test container.
type Server struct {
	// DSN is a URL with the superuser's throwaway password and
	// sslmode=disable.
	DSN string
	// Profile is DSN parsed.
	Profile conn.Profile
}

// Start starts postgres:<major>-alpine with pg_stat_statements preloaded.
// The container is removed when the test ends.
func Start(t *testing.T, major int) Server {
	t.Helper()
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, fmt.Sprintf("postgres:%d-alpine", major),
		postgres.WithDatabase("tuskline"),
		postgres.WithUsername(User),
		postgres.WithPassword(Password),
		testcontainers.WithCmdArgs("-c", "shared_preload_libraries=pg_stat_statements"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres %d: %v", major, err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })
	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	p, err := conn.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return Server{DSN: dsn, Profile: p}
}

// Exec runs setup statements as p's user, with a timeout.
func Exec(t *testing.T, p conn.Profile, stmts ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg, err := p.Config()
	if err != nil {
		t.Fatal(err)
	}
	c, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close(context.Background()) }()
	for _, s := range stmts {
		if _, err := c.Exec(ctx, s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

// As returns p logged in as another role.
func As(p conn.Profile, user, password string) conn.Profile {
	p.User = user
	return p.WithPassword(password)
}
