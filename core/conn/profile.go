// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// TLSMode is a libpq sslmode. Tuskline defaults to TLSVerifyFull, unlike
// libpq's "prefer"; anything weaker must be chosen explicitly per profile.
type TLSMode string

// The libpq sslmode values, weakest first.
const (
	TLSDisable    TLSMode = "disable"
	TLSAllow      TLSMode = "allow"
	TLSPrefer     TLSMode = "prefer"
	TLSRequire    TLSMode = "require"
	TLSVerifyCA   TLSMode = "verify-ca"
	TLSVerifyFull TLSMode = "verify-full"
)

// DefaultTLSMode is used when a DSN or profile does not name an sslmode.
const DefaultTLSMode = TLSVerifyFull

// ErrInvalidTLSMode is returned for an unknown sslmode.
var ErrInvalidTLSMode = errors.New("invalid sslmode: use disable, allow, prefer, require, verify-ca or verify-full")

// ParseTLSMode validates s. An empty string yields DefaultTLSMode.
func ParseTLSMode(s string) (TLSMode, error) {
	switch m := TLSMode(strings.ToLower(strings.TrimSpace(s))); m {
	case "":
		return DefaultTLSMode, nil
	case TLSDisable, TLSAllow, TLSPrefer, TLSRequire, TLSVerifyCA, TLSVerifyFull:
		return m, nil
	}
	return "", fmt.Errorf("%q: %w", s, ErrInvalidTLSMode)
}

// Downgraded reports whether m is weaker than verify-full, so the UI and CLI
// must warn about it.
func (m TLSMode) Downgraded() bool { return m != TLSVerifyFull && m != "" }

// Warning explains the risk of a downgraded mode, or returns "" for verify-full.
func (m TLSMode) Warning() string {
	switch m {
	case TLSDisable:
		return "TLS is disabled (sslmode=disable): credentials and data cross the network in plain text"
	case TLSAllow, TLSPrefer:
		return fmt.Sprintf("TLS is optional (sslmode=%s): the connection may fall back to plain text", m)
	case TLSRequire:
		return "TLS is not verified (sslmode=require): the server's certificate and host name are not checked"
	case TLSVerifyCA:
		return "TLS host name is not verified (sslmode=verify-ca): any server with a certificate from the CA is accepted"
	}
	return ""
}

// DefaultPort is PostgreSQL's default TCP port.
const DefaultPort = 5432

// Profile is everything needed to reach one server, except secrets in clear
// text. The password, if any, is held in memory only: it has no exported
// field and is never serialised, printed or returned in errors.
type Profile struct {
	// Host is a host name, IP address or absolute Unix socket directory.
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	// Port defaults to DefaultPort when zero.
	Port     uint16 `json:"port,omitempty" yaml:"port,omitempty"`
	Database string `json:"database,omitempty" yaml:"database,omitempty"`
	User     string `json:"user,omitempty" yaml:"user,omitempty"`
	// TLS is the sslmode; empty means DefaultTLSMode.
	TLS         TLSMode `json:"sslmode,omitempty" yaml:"sslmode,omitempty"`
	SSLRootCert string  `json:"sslrootcert,omitempty" yaml:"sslrootcert,omitempty"`
	SSLCert     string  `json:"sslcert,omitempty" yaml:"sslcert,omitempty"`
	SSLKey      string  `json:"sslkey,omitempty" yaml:"sslkey,omitempty"`
	// Service names a pg_service.conf entry; see ResolveService.
	Service         string `json:"service,omitempty" yaml:"service,omitempty"`
	ApplicationName string `json:"application_name,omitempty" yaml:"application_name,omitempty"`

	password string
}

// TLSMode returns the effective sslmode.
func (p Profile) TLSMode() TLSMode {
	if p.TLS == "" {
		return DefaultTLSMode
	}
	return p.TLS
}

// EffectivePort returns Port, or DefaultPort when unset.
func (p Profile) EffectivePort() uint16 {
	if p.Port == 0 {
		return DefaultPort
	}
	return p.Port
}

// HasPassword reports whether a password is held in memory.
func (p Profile) HasPassword() bool { return p.password != "" }

// WithPassword returns a copy of p holding password in memory.
func (p Profile) WithPassword(password string) Profile {
	p.password = password
	return p
}

// String returns a redacted keyword/value form, safe to log or print.
func (p Profile) String() string { return p.connString(true) }

// GoString keeps the password out of %#v output too.
func (p Profile) GoString() string { return "conn.Profile{" + p.String() + "}" }

// Validate checks a profile built by hand or loaded from a file: a known
// sslmode, and a host or service to connect to.
func (p Profile) Validate() error {
	if _, err := ParseTLSMode(string(p.TLS)); err != nil {
		return err
	}
	if p.Host == "" && p.Service == "" {
		return errors.New("profile needs a host or a service")
	}
	return nil
}

// dsnKeys maps libpq keywords to profile fields; anything else is rejected,
// so a typo never silently connects with defaults.
var dsnKeys = map[string]func(*Profile, string) error{
	"host":             func(p *Profile, v string) error { p.Host = v; return nil },
	"hostaddr":         func(p *Profile, v string) error { p.Host = v; return nil },
	"port":             func(p *Profile, v string) error { return p.setPort(v) },
	"dbname":           func(p *Profile, v string) error { p.Database = v; return nil },
	"user":             func(p *Profile, v string) error { p.User = v; return nil },
	"password":         func(p *Profile, v string) error { p.password = v; return nil },
	"sslmode":          func(p *Profile, v string) error { return p.setTLS(v) },
	"sslrootcert":      func(p *Profile, v string) error { p.SSLRootCert = v; return nil },
	"sslcert":          func(p *Profile, v string) error { p.SSLCert = v; return nil },
	"sslkey":           func(p *Profile, v string) error { p.SSLKey = v; return nil },
	"service":          func(p *Profile, v string) error { p.Service = v; return nil },
	"application_name": func(p *Profile, v string) error { p.ApplicationName = v; return nil },
}

func (p *Profile) setPort(v string) error {
	n, err := strconv.ParseUint(v, 10, 16)
	if err != nil || n == 0 {
		return fmt.Errorf("invalid port %q", v)
	}
	p.Port = uint16(n)
	return nil
}

func (p *Profile) setTLS(v string) error {
	m, err := ParseTLSMode(v)
	if err != nil {
		return err
	}
	p.TLS = m
	return nil
}

// ParseDSN parses a libpq URL (postgres:// or postgresql://) or a
// keyword/value string. It does not read the environment, ~/.pgpass or
// pg_service.conf; see ResolveService and LookupPassword. Errors never
// contain the password.
func ParseDSN(dsn string) (Profile, error) {
	var p Profile
	var err error
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		err = p.parseURL(dsn)
	} else {
		err = p.parseKeywords(dsn)
	}
	if err != nil {
		return Profile{}, fmt.Errorf("parse DSN %s: %w", Redact(dsn), err)
	}
	return p, nil
}

func (p *Profile) parseURL(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		// url.Error repeats the input, which may hold the password.
		return errors.New("malformed URL")
	}
	if u.User != nil {
		p.User = u.User.Username()
		p.password, _ = u.User.Password()
	}
	if host := u.Hostname(); host != "" {
		p.Host = host
	}
	if port := u.Port(); port != "" {
		if err := p.setPort(port); err != nil {
			return err
		}
	}
	if strings.Contains(u.Host, ",") {
		return errors.New("multiple hosts are not supported yet")
	}
	p.Database = strings.TrimPrefix(u.Path, "/")
	q := u.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := p.set(k, q.Get(k)); err != nil {
			return err
		}
	}
	return nil
}

func (p *Profile) parseKeywords(dsn string) error {
	s := strings.TrimSpace(dsn)
	for s != "" {
		eq := strings.IndexByte(s, '=')
		if eq <= 0 {
			return errors.New("want keyword=value pairs or a postgres:// URL")
		}
		key := strings.TrimSpace(s[:eq])
		s = strings.TrimLeft(s[eq+1:], " \t")
		var val string
		if strings.HasPrefix(s, "'") {
			var b strings.Builder
			i := 1
			for ; i < len(s); i++ {
				c := s[i]
				if c == '\\' && i+1 < len(s) {
					i++
					b.WriteByte(s[i])
					continue
				}
				if c == '\'' {
					break
				}
				b.WriteByte(c)
			}
			if i >= len(s) {
				return fmt.Errorf("unterminated quoted value for %q", key)
			}
			val, s = b.String(), s[i+1:]
		} else {
			end := strings.IndexAny(s, " \t\n")
			if end < 0 {
				end = len(s)
			}
			val, s = s[:end], s[end:]
		}
		if err := p.set(key, val); err != nil {
			return err
		}
		s = strings.TrimLeft(s, " \t\n")
	}
	return nil
}

func (p *Profile) set(key, val string) error {
	f, ok := dsnKeys[key]
	if !ok {
		return fmt.Errorf("unsupported parameter %q", key)
	}
	return f(p, val)
}

// connString renders keyword/value pairs. The password is never rendered: it
// is shown as *** when showRedacted is set, and omitted otherwise.
func (p Profile) connString(showRedacted bool) string {
	var parts []string
	add := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+quote(v))
		}
	}
	add("host", p.Host)
	if p.Port != 0 {
		add("port", strconv.Itoa(int(p.Port)))
	}
	add("dbname", p.Database)
	add("user", p.User)
	if showRedacted && p.password != "" {
		add("password", redacted)
	}
	add("sslmode", string(p.TLSMode()))
	add("sslrootcert", p.SSLRootCert)
	add("sslcert", p.SSLCert)
	add("sslkey", p.SSLKey)
	add("service", p.Service)
	add("application_name", p.ApplicationName)
	return strings.Join(parts, " ")
}

func quote(v string) string {
	if v != "" && !strings.ContainsAny(v, " \t\n'\\") {
		return v
	}
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
}

// Config builds a pgconn configuration from p, applying the TLS default. It
// does not resolve services or look up passwords; callers do that first so
// the steps stay explicit and testable. Environment variables such as
// PGSSLMODE never override the profile, because every field is passed
// explicitly. Errors never contain the password.
func (p Profile) Config() (*pgconn.Config, error) {
	if p.Service != "" {
		return nil, fmt.Errorf("profile %s: service %q not resolved; call ResolveService first", p, p.Service)
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("profile %s: %w", p, err)
	}
	cfg, err := pgconn.ParseConfig(p.connString(false))
	if err != nil {
		return nil, fmt.Errorf("profile %s: %w", p, err)
	}
	// Set after parsing so the password never sits in a string pgx could
	// echo. This also discards PGPASSWORD and pgx's own ~/.pgpass fallback:
	// the only password sources are the profile and LookupPassword.
	cfg.Password = p.password
	return cfg, nil
}
