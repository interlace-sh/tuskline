// SPDX-License-Identifier: Apache-2.0

//go:build integration

package conn_test

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
	testUser     = "tuskline"
	testPassword = "tuskline-test-only"
)

func pgVersions(t *testing.T) []int {
	t.Helper()
	s := os.Getenv("PG_VERSIONS")
	if s == "" {
		return []int{14, 15, 16, 17, 18}
	}
	var out []int
	// Accept "17", "14,18" or "14 15 16".
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			t.Fatalf("PG_VERSIONS: %v", err)
		}
		out = append(out, n)
	}
	return out
}

// startPG starts postgres:<major>-alpine with pg_stat_statements preloaded
// and returns a superuser profile for it. The container is removed when the
// test ends.
func startPG(t *testing.T, major int) conn.Profile {
	t.Helper()
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, fmt.Sprintf("postgres:%d-alpine", major),
		postgres.WithDatabase("tuskline"),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPassword),
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
	return p
}

// execAll runs setup statements as the profile's user (the container's
// superuser), with a timeout.
func execAll(t *testing.T, p conn.Profile, stmts ...string) {
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

// as returns p logged in as another role.
func as(p conn.Profile, user, password string) conn.Profile {
	p.User = user
	return p.WithPassword(password)
}
