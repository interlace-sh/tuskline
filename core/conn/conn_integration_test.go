// SPDX-License-Identifier: Apache-2.0

//go:build integration

package conn_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/interlace-sh/tuskline/core/conn"
)

// Run with: moon run core:integration (needs Docker).
// PG_VERSIONS can narrow the matrix, e.g. PG_VERSIONS=17.
func TestProbeAcrossVersions(t *testing.T) {
	for _, major := range pgVersions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			ctx := context.Background()
			ctr, err := postgres.Run(ctx, fmt.Sprintf("postgres:%d-alpine", major),
				postgres.WithDatabase("tuskline"),
				postgres.WithUsername("tuskline"),
				postgres.WithPassword("tuskline"),
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
			info, err := conn.Probe(ctx, dsn)
			if err != nil {
				t.Fatal(err)
			}
			if info.Major() != major {
				t.Fatalf("major = %d, want %d", info.Major(), major)
			}
			if err := conn.CheckSupported(info.VersionNum); err != nil {
				t.Fatal(err)
			}
		})
	}
}
