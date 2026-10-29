// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const secret = "s3cret-Pa55"

func TestParseDSN(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want Profile
		pw   bool
	}{
		{
			name: "url",
			in:   "postgres://app:" + secret + "@db.example.com:6432/orders?sslmode=require&application_name=tl",
			want: Profile{Host: "db.example.com", Port: 6432, Database: "orders", User: "app", TLS: TLSRequire, ApplicationName: "tl"},
			pw:   true,
		},
		{
			name: "url defaults",
			in:   "postgresql://db/orders",
			want: Profile{Host: "db", Database: "orders"},
		},
		{
			name: "keyword",
			in:   "host=db port=5417 dbname=tuskline user=tuskline password='" + secret + "' sslmode=disable",
			want: Profile{Host: "db", Port: 5417, Database: "tuskline", User: "tuskline", TLS: TLSDisable},
			pw:   true,
		},
		{
			name: "keyword quoted spaces",
			in:   `host=db application_name='my app' sslrootcert='/etc/ssl/root ca.pem'`,
			want: Profile{Host: "db", ApplicationName: "my app", SSLRootCert: "/etc/ssl/root ca.pem"},
		},
		{
			name: "service",
			in:   "service=orders-prod",
			want: Profile{Service: "orders-prod"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseDSN(c.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.HasPassword() != c.pw {
				t.Fatalf("HasPassword() = %v, want %v", got.HasPassword(), c.pw)
			}
			got.password = ""
			if got != c.want {
				t.Fatalf("ParseDSN() = %#v\nwant %#v", got, c.want)
			}
		})
	}
}

func TestParseDSNErrorsNeverContainPassword(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		"postgres://app:" + secret + "@db:99999/orders",
		"postgres://app:" + secret + "@db/orders?sslmode=bogus",
		"postgres://app:" + secret + "@db/orders?nosuchparam=1",
		"postgres://app:" + secret + "@db\x7f/orders",
		"host=db password=" + secret + " sslmode=bogus",
		"host=db password='" + secret,
		"host=db password=" + secret + " junk",
	} {
		_, err := ParseDSN(in)
		if err == nil {
			t.Errorf("ParseDSN(%q): want an error", Redact(in))
			continue
		}
		if strings.Contains(err.Error(), secret) {
			t.Errorf("error leaks the password: %v", err)
		}
	}
}

func TestTLSDefaultsToVerifyFull(t *testing.T) {
	t.Parallel()
	p, err := ParseDSN("postgres://app@db.example.com/orders")
	if err != nil {
		t.Fatal(err)
	}
	if p.TLSMode() != TLSVerifyFull {
		t.Fatalf("TLSMode() = %q, want verify-full", p.TLSMode())
	}
	if p.TLSMode().Downgraded() {
		t.Fatal("verify-full must not count as downgraded")
	}
	cfg, err := p.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TLSConfig == nil || cfg.TLSConfig.InsecureSkipVerify && cfg.TLSConfig.VerifyConnection == nil {
		t.Fatal("verify-full must verify the server certificate")
	}
	if cfg.TLSConfig.ServerName != "db.example.com" {
		t.Fatalf("ServerName = %q, want the host name for verification", cfg.TLSConfig.ServerName)
	}
	if len(cfg.Fallbacks) != 0 {
		t.Fatalf("verify-full must not fall back to plain text, got %d fallbacks", len(cfg.Fallbacks))
	}
}

func TestEnvironmentCannotDowngradeTLS(t *testing.T) {
	t.Setenv("PGSSLMODE", "disable")
	p, err := ParseDSN("host=db.example.com dbname=orders")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := p.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TLSConfig == nil {
		t.Fatal("PGSSLMODE=disable overrode the verify-full default")
	}
}

func TestDowngradedModesWarn(t *testing.T) {
	t.Parallel()
	for _, m := range []TLSMode{TLSDisable, TLSAllow, TLSPrefer, TLSRequire, TLSVerifyCA} {
		if !m.Downgraded() || m.Warning() == "" {
			t.Errorf("%s: Downgraded()=%v Warning()=%q, want true and a warning", m, m.Downgraded(), m.Warning())
		}
	}
	if TLSVerifyFull.Warning() != "" {
		t.Error("verify-full must not warn")
	}
}

func TestParseTLSMode(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]TLSMode{"": TLSVerifyFull, "Verify-Full": TLSVerifyFull, "disable": TLSDisable, " require ": TLSRequire} {
		got, err := ParseTLSMode(in)
		if err != nil || got != want {
			t.Errorf("ParseTLSMode(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseTLSMode("off"); !errors.Is(err, ErrInvalidTLSMode) {
		t.Errorf("ParseTLSMode(off) err = %v, want ErrInvalidTLSMode", err)
	}
}

func TestProfileNeverRendersPassword(t *testing.T) {
	t.Parallel()
	p, err := ParseDSN("postgres://app:" + secret + "@db/orders")
	if err != nil {
		t.Fatal(err)
	}
	js, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for name, s := range map[string]string{
		"String": p.String(),
		"%v":     fmt.Sprintf("%v", p),
		"%+v":    fmt.Sprintf("%+v", p),
		"%#v":    fmt.Sprintf("%#v", p),
		"json":   string(js),
	} {
		if strings.Contains(s, secret) {
			t.Errorf("%s leaks the password: %s", name, s)
		}
	}
	if !strings.Contains(p.String(), "password=***") {
		t.Errorf("String() = %q, want a password=*** marker", p.String())
	}
}

func TestConfigCarriesPassword(t *testing.T) {
	t.Parallel()
	p, err := ParseDSN("host=db user=app dbname=orders sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := p.WithPassword(secret).Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Password != secret || cfg.Host != "db" || cfg.Port != DefaultPort || cfg.Database != "orders" || cfg.User != "app" {
		t.Fatalf("Config() = host %q port %d db %q user %q, password set %v", cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password == secret)
	}
	if cfg.TLSConfig != nil {
		t.Fatal("sslmode=disable must not configure TLS")
	}
}

func TestConfigRejectsUnresolvedServiceAndEmptyHost(t *testing.T) {
	t.Parallel()
	if _, err := (Profile{Service: "orders"}).Config(); err == nil || !strings.Contains(err.Error(), "ResolveService") {
		t.Errorf("unresolved service: got %v", err)
	}
	if _, err := (Profile{Database: "orders"}).Config(); err == nil {
		t.Error("profile without host or service: want an error")
	}
}
