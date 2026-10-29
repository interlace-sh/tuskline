// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/jackc/pgpassfile"
	"github.com/jackc/pgservicefile"
)

// ErrPassfileInsecure is returned when the password file is readable by the
// group or others. Like libpq, Tuskline then ignores it.
var ErrPassfileInsecure = errors.New("password file is group- or world-accessible; run chmod 0600 on it")

// ErrServiceNotFound is returned when no service file defines the service.
var ErrServiceNotFound = errors.New("service not found")

// PassfilePath returns libpq's password file location: $PGPASSFILE, else
// ~/.pgpass (%APPDATA%\postgresql\pgpass.conf on Windows).
func PassfilePath() string {
	if p := os.Getenv("PGPASSFILE"); p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "postgresql", "pgpass.conf")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pgpass")
}

// ServicefilePaths returns libpq's service files in lookup order: the user
// file ($PGSERVICEFILE, else ~/.pg_service.conf) and then the system file
// ($PGSYSCONFDIR/pg_service.conf) when PGSYSCONFDIR is set.
func ServicefilePaths() []string {
	var paths []string
	switch {
	case os.Getenv("PGSERVICEFILE") != "":
		paths = append(paths, os.Getenv("PGSERVICEFILE"))
	case runtime.GOOS == "windows":
		paths = append(paths, filepath.Join(os.Getenv("APPDATA"), "postgresql", ".pg_service.conf"))
	default:
		if home, err := os.UserHomeDir(); err == nil {
			paths = append(paths, filepath.Join(home, ".pg_service.conf"))
		}
	}
	if dir := os.Getenv("PGSYSCONFDIR"); dir != "" {
		paths = append(paths, filepath.Join(dir, "pg_service.conf"))
	}
	return paths
}

// LookupPassword finds the password for p in the password file at path. It
// returns "" with a nil error when the file does not exist or has no
// matching entry. On Unix a file readable by the group or others is ignored
// and ErrPassfileInsecure is returned. The password is never in an error.
func LookupPassword(path string, p Profile) (string, error) {
	if path == "" {
		return "", nil
	}
	st, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("password file %s: %w", path, err)
	}
	if runtime.GOOS != "windows" && st.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("%s: %w", path, ErrPassfileInsecure)
	}
	pf, err := pgpassfile.ReadPassfile(path)
	if err != nil {
		return "", fmt.Errorf("read password file %s: %w", path, err)
	}
	host := p.Host
	if host == "" || filepath.IsAbs(host) {
		// libpq matches Unix-socket connections as "localhost".
		host = "localhost"
	}
	db := p.Database
	if db == "" {
		db = p.User
	}
	return pf.FindPassword(host, strconv.Itoa(int(p.EffectivePort())), db, p.User), nil
}

// ResolveService fills p's unset fields from the first service file in paths
// that defines p.Service, and clears Service. Values already in p (from the
// DSN) win, as in libpq. A profile without a service is returned unchanged.
func ResolveService(paths []string, p Profile) (Profile, error) {
	if p.Service == "" {
		return p, nil
	}
	for _, path := range paths {
		sf, err := pgservicefile.ReadServicefile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return Profile{}, fmt.Errorf("read service file %s: %w", path, err)
		}
		svc, err := sf.GetService(p.Service)
		if err != nil {
			continue
		}
		var base Profile
		for k, v := range svc.Settings {
			if err := base.set(k, v); err != nil {
				return Profile{}, fmt.Errorf("service %q in %s: %w", p.Service, path, err)
			}
		}
		return overlay(base, p), nil
	}
	return Profile{}, fmt.Errorf("service %q: %w (looked in %v)", p.Service, ErrServiceNotFound, paths)
}

// overlay returns base with every non-empty field of top applied over it.
func overlay(base, top Profile) Profile {
	set := func(dst *string, v string) {
		if v != "" {
			*dst = v
		}
	}
	set(&base.Host, top.Host)
	if top.Port != 0 {
		base.Port = top.Port
	}
	set(&base.Database, top.Database)
	set(&base.User, top.User)
	if top.TLS != "" {
		base.TLS = top.TLS
	}
	set(&base.SSLRootCert, top.SSLRootCert)
	set(&base.SSLCert, top.SSLCert)
	set(&base.SSLKey, top.SSLKey)
	set(&base.ApplicationName, top.ApplicationName)
	set(&base.password, top.password)
	base.Service = ""
	return base
}
