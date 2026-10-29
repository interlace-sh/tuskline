// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeFile(t *testing.T, name, body string, perm os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), perm); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLookupPassword(t *testing.T) {
	t.Parallel()
	path := writeFile(t, ".pgpass", strings.Join([]string{
		"# comment",
		"db.example.com:5432:orders:app:" + secret,
		"localhost:5417:*:tuskline:local-pw",
		"*:*:*:readonly:fallback-pw",
	}, "\n")+"\n", 0o600)
	cases := []struct {
		name string
		p    Profile
		want string
	}{
		{"exact match on default port", Profile{Host: "db.example.com", Database: "orders", User: "app"}, secret},
		{"wildcard database", Profile{Host: "localhost", Port: 5417, Database: "any", User: "tuskline"}, "local-pw"},
		{"unix socket matches localhost", Profile{Host: "/var/run/postgresql", Port: 5417, User: "tuskline"}, "local-pw"},
		{"wildcard everything", Profile{Host: "other", User: "readonly"}, "fallback-pw"},
		{"no match", Profile{Host: "db.example.com", Database: "orders", User: "nobody"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := LookupPassword(path, c.p)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("LookupPassword() matched %v, want match %v", got != "", c.want != "")
			}
		})
	}
}

func TestLookupPasswordMissingFile(t *testing.T) {
	t.Parallel()
	got, err := LookupPassword(filepath.Join(t.TempDir(), "absent"), Profile{Host: "db"})
	if err != nil || got != "" {
		t.Fatalf("missing file: got %q, %v; want no password and no error", got, err)
	}
}

func TestLookupPasswordIgnoresInsecureFile(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("libpq does not check password file permissions on Windows")
	}
	path := writeFile(t, ".pgpass", "*:*:*:*:"+secret+"\n", 0o644)
	got, err := LookupPassword(path, Profile{Host: "db", User: "app"})
	if !errors.Is(err, ErrPassfileInsecure) {
		t.Fatalf("err = %v, want ErrPassfileInsecure", err)
	}
	if got != "" {
		t.Fatal("an insecure password file must be ignored")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaks the password: %v", err)
	}
}

func TestPassfilePathHonoursEnv(t *testing.T) {
	t.Setenv("PGPASSFILE", "/custom/pgpass")
	if got := PassfilePath(); got != "/custom/pgpass" {
		t.Fatalf("PassfilePath() = %q", got)
	}
}

func TestServicefilePathsHonourEnv(t *testing.T) {
	t.Setenv("PGSERVICEFILE", "/custom/services.conf")
	t.Setenv("PGSYSCONFDIR", "/etc/postgresql-common")
	got := ServicefilePaths()
	want := []string{"/custom/services.conf", filepath.Join("/etc/postgresql-common", "pg_service.conf")}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ServicefilePaths() = %v, want %v", got, want)
	}
}

func TestResolveService(t *testing.T) {
	t.Parallel()
	user := writeFile(t, "user.conf", "[orders]\nhost=db.example.com\nport=6432\ndbname=orders\nuser=app\nsslmode=verify-ca\n", 0o600)
	system := writeFile(t, "system.conf", "[orders]\nhost=ignored\n[reports]\nhost=reports.example.com\ndbname=reports\n", 0o600)
	paths := []string{filepath.Join(t.TempDir(), "absent.conf"), user, system}

	p, err := ParseDSN("service=orders user=override")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveService(paths, p)
	if err != nil {
		t.Fatal(err)
	}
	want := Profile{Host: "db.example.com", Port: 6432, Database: "orders", User: "override", TLS: TLSVerifyCA}
	if got != want {
		t.Fatalf("ResolveService() = %#v\nwant %#v", got, want)
	}

	got, err = ResolveService(paths, Profile{Service: "reports"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != "reports.example.com" || got.TLSMode() != TLSVerifyFull {
		t.Fatalf("system service: got %#v, want reports.example.com with the verify-full default", got)
	}

	if _, err := ResolveService(paths, Profile{Service: "missing"}); !errors.Is(err, ErrServiceNotFound) {
		t.Fatalf("missing service: err = %v, want ErrServiceNotFound", err)
	}
	if got, err := ResolveService(paths, Profile{Host: "plain"}); err != nil || got.Host != "plain" {
		t.Fatalf("no service: got %#v, %v; want unchanged", got, err)
	}
}

func TestResolveServiceRejectsUnknownSetting(t *testing.T) {
	t.Parallel()
	path := writeFile(t, "svc.conf", "[orders]\nhost=db\nsslmod=disable\n", 0o600)
	if _, err := ResolveService([]string{path}, Profile{Service: "orders"}); err == nil || !strings.Contains(err.Error(), "sslmod") {
		t.Fatalf("got %v, want an error naming the unknown setting", err)
	}
}
