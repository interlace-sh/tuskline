// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// MinServerVersionNum is the lowest supported server_version_num
// (PostgreSQL 14, see docs/adr/0006-minimum-postgresql-14.md).
const MinServerVersionNum = 140000

// ServerInfo is what Probe learns about a server.
type ServerInfo struct {
	VersionNum int    // e.g. 170004
	Version    string // full version() string
	ReadOnly   bool   // session default_transaction_read_only
}

// Major returns the major version, e.g. 17.
func (s ServerInfo) Major() int { return s.VersionNum / 10000 }

// CheckSupported returns an error for servers older than PostgreSQL 14.
func CheckSupported(versionNum int) error {
	if versionNum < MinServerVersionNum {
		return fmt.Errorf("PostgreSQL %d is not supported: Tuskline needs PostgreSQL 14 or newer", versionNum/10000)
	}
	return nil
}

// Probe connects with a libpq URL or keyword/value DSN, reads basic server
// information and disconnects. It never writes.
func Probe(ctx context.Context, dsn string) (ServerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	c, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = c.Close(context.Background()) }() // best effort; the probe result is already known
	var info ServerInfo
	var ro string
	err = c.QueryRow(ctx, "SELECT current_setting('server_version_num')::int, version(), current_setting('default_transaction_read_only')").Scan(&info.VersionNum, &info.Version, &ro)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("probe: %w", err)
	}
	info.ReadOnly = ro == "on"
	return info, nil
}
