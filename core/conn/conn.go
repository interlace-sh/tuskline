// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/sqlpack"
)

// MinServerVersionNum is the lowest supported server_version_num
// (PostgreSQL 14, see docs/adr/0006-minimum-postgresql-14.md).
const MinServerVersionNum = 140000

// ProbeTimeout bounds a whole probe: connect plus every probe query.
const ProbeTimeout = 10 * time.Second

// ServerInfo is what Probe learns about a server.
type ServerInfo struct {
	VersionNum int      `json:"version_num"` // e.g. 170004
	Version    string   `json:"version"`     // full version() string
	ReadOnly   bool     `json:"read_only"`   // session default_transaction_read_only
	InRecovery bool     `json:"in_recovery"` // a standby
	Database   string   `json:"database"`
	LoginRole  string   `json:"login_role"`
	Provider   Provider `json:"provider"`
	// SharedPreloadLibraries is nil when PreloadKnown is false, because the
	// login role may not read the setting (it needs pg_read_all_settings).
	SharedPreloadLibraries []string    `json:"shared_preload_libraries"`
	PreloadKnown           bool        `json:"preload_known"`
	Extensions             []Extension `json:"extensions"`
	Role                   Role        `json:"role"`
}

// Extension is an extension that is installed in the current database or
// available on the server.
type Extension struct {
	Name             string `json:"name"`
	InstalledVersion string `json:"installed_version,omitempty"` // "" when not installed
	Available        bool   `json:"available"`                   // a control file exists
}

// Installed reports whether the extension is installed in the database.
func (e Extension) Installed() bool { return e.InstalledVersion != "" }

// Role describes the login role's usable privileges.
type Role struct {
	Superuser         bool `json:"superuser"`
	CreateRole        bool `json:"create_role"`
	CreateDB          bool `json:"create_db"`
	Replication       bool `json:"replication"`
	BypassRLS         bool `json:"bypass_rls"`
	PGMonitor         bool `json:"pg_monitor"`
	PGReadAllSettings bool `json:"pg_read_all_settings"`
	PGReadAllStats    bool `json:"pg_read_all_stats"`
	PGSignalBackend   bool `json:"pg_signal_backend"`
}

// CanMonitor reports whether the role sees every session and setting, as
// Tuskline's collectors need (pg_monitor or superuser).
func (r Role) CanMonitor() bool { return r.Superuser || r.PGMonitor }

// Major returns the major version, e.g. 17.
func (s ServerInfo) Major() int { return s.VersionNum / 10000 }

// Extension returns the named extension, if the server knows it.
func (s ServerInfo) Extension(name string) (Extension, bool) {
	for _, e := range s.Extensions {
		if e.Name == name {
			return e, true
		}
	}
	return Extension{}, false
}

// Preloaded reports whether lib is in shared_preload_libraries. It returns
// false when the setting could not be read; check PreloadKnown.
func (s ServerInfo) Preloaded(lib string) bool {
	return slices.Contains(s.SharedPreloadLibraries, lib)
}

// CheckSupported returns an error for servers older than PostgreSQL 14.
func CheckSupported(versionNum int) error {
	if versionNum < MinServerVersionNum {
		return fmt.Errorf("PostgreSQL %d is not supported: Tuskline needs PostgreSQL 14 or newer", versionNum/10000)
	}
	return nil
}

// Inspect reads version, preload libraries and extensions from an open
// connection. It runs only the read-only probe pack.
func Inspect(ctx context.Context, db *pgx.Conn) (ServerInfo, error) {
	return probe(ctx, db)
}

// Probe connects with p, reads what Tuskline needs to know about the server
// and disconnects. It runs only the read-only probe pack, so it never
// writes; it refuses servers older than PostgreSQL 14 before running
// anything version-specific. Errors never contain the password.
func Probe(ctx context.Context, p Profile) (ServerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()
	cfg, err := p.Config()
	if err != nil {
		return ServerInfo{}, err
	}
	c, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("connect %s: %w", p, err)
	}
	defer func() { _ = c.Close(context.Background()) }() // best effort; the probe result is already known
	return probe(ctx, c)
}

func probe(ctx context.Context, c *pgx.Conn) (ServerInfo, error) {
	var info ServerInfo
	var preload sql.NullString
	var sig ProviderSignals
	err := c.QueryRow(ctx, sqlpack.ProbeServer.SQL).Scan(
		&info.VersionNum, &info.Version, &info.ReadOnly, &info.InRecovery,
		&info.Database, &info.LoginRole, &preload,
		&sig.AuroraVersion, &sig.RDSSuperuser, &sig.RDSSettings,
		&sig.CloudSQLSuperuser, &sig.CloudSQLSettings,
		&sig.AlloyDBSuperuser, &sig.AlloyDBSettings,
		&sig.AzureAdmin, &sig.AzureSettings, &sig.NeonSettings,
	)
	if err != nil {
		return ServerInfo{}, fmt.Errorf("probe server: %w", err)
	}
	if err := CheckSupported(info.VersionNum); err != nil {
		return info, err
	}
	if preload.Valid {
		info.PreloadKnown = true
		info.SharedPreloadLibraries = splitList(preload.String)
	}
	sig.PreloadsNeon = info.Preloaded("neon")
	info.Provider = DetectProvider(sig)

	q, err := sqlpack.Pick(info.VersionNum, sqlpack.ProbeExtensions)
	if err != nil {
		return info, err
	}
	rows, err := c.Query(ctx, q.SQL)
	if err != nil {
		return info, fmt.Errorf("probe extensions: %w", err)
	}
	info.Extensions, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Extension, error) {
		var e Extension
		var installed, def sql.NullString
		if err := r.Scan(&e.Name, &installed, &def); err != nil {
			return e, err
		}
		e.InstalledVersion, e.Available = installed.String, def.Valid
		return e, nil
	})
	if err != nil {
		return info, fmt.Errorf("probe extensions: %w", err)
	}

	q, err = sqlpack.Pick(info.VersionNum, sqlpack.ProbeRole)
	if err != nil {
		return info, err
	}
	r := &info.Role
	if err := c.QueryRow(ctx, q.SQL).Scan(
		&r.Superuser, &r.CreateRole, &r.CreateDB, &r.Replication, &r.BypassRLS,
		&r.PGMonitor, &r.PGReadAllSettings, &r.PGReadAllStats, &r.PGSignalBackend,
	); err != nil {
		return info, fmt.Errorf("probe role: %w", err)
	}
	return info, nil
}

// splitList parses a GUC list such as shared_preload_libraries
// ("pg_stat_statements, auto_explain" or with quoted entries).
func splitList(s string) []string {
	var out []string
	for _, f := range strings.Split(s, ",") {
		f = strings.Trim(strings.TrimSpace(f), `"`)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}
