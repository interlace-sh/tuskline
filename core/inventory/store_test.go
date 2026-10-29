// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/interlace-sh/tuskline/core/conn"
)

const secretValue = "hunter2-S3cret"

func mustProfile(t *testing.T, dsn string) conn.Profile {
	t.Helper()
	p, err := conn.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func sample(t *testing.T) *Inventory {
	t.Helper()
	inv := &Inventory{Version: FileVersion}
	for name, kind := range map[string]EnvKind{"prod": EnvProd, "dev": EnvDev} {
		if _, err := inv.PutEnvironment(name, kind); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []Cluster{
		{Name: "orders", Environment: "prod", Provider: "rds", Profile: mustProfile(t, "postgres://app@orders.example.com/orders"), Password: &SecretRef{Env: "ORDERS_PASSWORD"}},
		{Name: "pg17", Environment: "dev", Profile: mustProfile(t, "host=localhost port=5417 dbname=tuskline user=tuskline sslmode=disable")},
	} {
		if err := inv.PutCluster(c); err != nil {
			t.Fatal(err)
		}
	}
	return inv
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tuskline", "inventory.yaml")
	inv := sample(t)
	c, _ := inv.Cluster(Ref{"dev", "pg17"})
	c.LastProbe = RecordProbe(conn.ServerInfo{VersionNum: 170006, Provider: conn.ProviderCommunity}, time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC))
	if err := inv.PutCluster(c); err != nil {
		t.Fatal(err)
	}
	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm() != 0o600 {
			t.Errorf("file mode = %v, want 0600", st.Mode().Perm())
		}
		dst, _ := os.Stat(filepath.Dir(path))
		if dst.Mode().Perm() != 0o700 {
			t.Errorf("dir mode = %v, want 0700", dst.Mode().Perm())
		}
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Environments) != 2 || len(got.Clusters) != 2 {
		t.Fatalf("loaded %d environments, %d clusters", len(got.Environments), len(got.Clusters))
	}
	orders, ok := got.Cluster(Ref{"prod", "orders"})
	if !ok || orders.Profile.Host != "orders.example.com" || orders.Provider != "rds" || orders.Password == nil || orders.Password.Env != "ORDERS_PASSWORD" {
		t.Fatalf("prod/orders = %+v", orders)
	}
	if orders.Profile.TLSMode() != conn.TLSVerifyFull {
		t.Errorf("TLS = %s, want the verify-full default", orders.Profile.TLSMode())
	}
	pg17, _ := got.Cluster(Ref{"dev", "pg17"})
	if pg17.LastProbe == nil || pg17.LastProbe.VersionNum != 170006 {
		t.Errorf("last probe = %+v", pg17.LastProbe)
	}
	prod, _ := got.Environment("prod")
	if !prod.Policy.ReadOnly {
		t.Error("prod loaded from the file must be read-only")
	}
}

func TestInventoryNeverStoresSecrets(t *testing.T) {
	t.Setenv("ORDERS_PASSWORD", secretValue)
	path := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := sample(t)

	withPw := Cluster{Name: "leaky", Environment: "dev", Profile: mustProfile(t, "postgres://app:"+secretValue+"@db/app")}
	if err := inv.PutCluster(withPw); !errors.Is(err, ErrPlaintextSecret) {
		t.Fatalf("PutCluster with a password in the profile: err = %v, want ErrPlaintextSecret", err)
	}
	inv.Clusters = append(inv.Clusters, withPw)
	if err := inv.Save(path); !errors.Is(err, ErrPlaintextSecret) {
		t.Fatalf("Save with a password in a profile: err = %v, want ErrPlaintextSecret", err)
	}
	inv.Clusters = inv.Clusters[:len(inv.Clusters)-1]

	if err := inv.Save(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secretValue) {
		t.Fatalf("inventory file contains the secret:\n%s", raw)
	}
	if !strings.Contains(string(raw), "ORDERS_PASSWORD") {
		t.Fatalf("inventory file lacks the secret reference:\n%s", raw)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	orders, _ := got.Cluster(Ref{"prod", "orders"})
	v, err := orders.Password.Resolve()
	if err != nil || v != secretValue {
		t.Fatalf("Resolve() = matched %v, %v", v == secretValue, err)
	}
}

func TestParseRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, doc string
		is        error
		mention   string
	}{
		{
			name: "plaintext password",
			doc:  "environments: [{name: dev, kind: dev}]\nclusters:\n  - name: app\n    environment: dev\n    profile: {host: db}\n    password: " + secretValue + "\n",
			is:   ErrPlaintextSecret,
		},
		{
			name:    "password inside the profile",
			doc:     "environments: [{name: dev, kind: dev}]\nclusters:\n  - name: app\n    environment: dev\n    profile: {host: db, password: " + secretValue + "}\n",
			mention: "password",
		},
		{
			name:    "policy override in the file",
			doc:     "environments:\n  - name: prod\n    kind: prod\n    policy: {read_only: false}\n",
			mention: "policy",
		},
		{
			name:    "unknown key",
			doc:     "environments: [{name: dev, kind: dev, colour: red}]\n",
			mention: "colour",
		},
		{
			name:    "unknown environment kind",
			doc:     "environments: [{name: qa, kind: qa}]\n",
			mention: "invalid kind",
		},
		{
			name: "cluster in a missing environment",
			doc:  "environments: [{name: dev, kind: dev}]\nclusters: [{name: app, environment: prod, profile: {host: db}}]\n",
			is:   ErrNotFound,
		},
		{
			name:    "duplicate cluster",
			doc:     "environments: [{name: dev, kind: dev}]\nclusters: [{name: app, environment: dev, profile: {host: a}}, {name: app, environment: dev, profile: {host: b}}]\n",
			mention: "defined twice",
		},
		{
			name:    "bad sslmode",
			doc:     "environments: [{name: dev, kind: dev}]\nclusters: [{name: app, environment: dev, profile: {host: db, sslmode: off}}]\n",
			mention: "sslmode",
		},
		{
			name:    "bad secret reference",
			doc:     "environments: [{name: dev, kind: dev}]\nclusters: [{name: app, environment: dev, profile: {host: db}, password: {env: '1-BAD'}}]\n",
			mention: "environment variable name",
		},
		{
			name:    "future version",
			doc:     "version: 9\n",
			mention: "unsupported inventory version",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(strings.NewReader(c.doc))
			if err == nil {
				t.Fatal("want an error")
			}
			if c.is != nil && !errors.Is(err, c.is) {
				t.Errorf("err = %v, want %v", err, c.is)
			}
			if c.mention != "" && !strings.Contains(err.Error(), c.mention) {
				t.Errorf("err = %v, want it to mention %q", err, c.mention)
			}
			if strings.Contains(err.Error(), secretValue) {
				t.Errorf("error leaks the secret: %v", err)
			}
		})
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	t.Parallel()
	inv, err := Load(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil || len(inv.Clusters) != 0 || inv.Version != FileVersion {
		t.Fatalf("Load(absent) = %+v, %v", inv, err)
	}
}

func TestCRUD(t *testing.T) {
	t.Parallel()
	inv := sample(t)
	if _, err := inv.PutEnvironment("qa", "qa"); err == nil {
		t.Error("PutEnvironment with an invalid kind: want an error")
	}
	if _, err := inv.PutEnvironment("Prod", EnvProd); !errors.Is(err, ErrInvalidName) {
		t.Errorf("PutEnvironment(Prod): err = %v, want ErrInvalidName", err)
	}
	if err := inv.PutCluster(Cluster{Name: "x", Environment: "staging", Profile: conn.Profile{Host: "db"}}); !errors.Is(err, ErrNotFound) {
		t.Errorf("PutCluster into a missing environment: err = %v, want ErrNotFound", err)
	}
	if err := inv.PutCluster(Cluster{Name: "x", Environment: "dev"}); err == nil {
		t.Error("PutCluster without host or service: want an error")
	}
	if err := inv.PutCluster(Cluster{Name: "x", Environment: "dev", Profile: conn.Profile{Host: "db"}, Password: &SecretRef{}}); err == nil {
		t.Error("PutCluster with an empty secret reference: want an error")
	}

	updated := Cluster{Name: "pg17", Environment: "dev", Provider: "community", Profile: conn.Profile{Host: "127.0.0.1", Port: 5417}}
	if err := inv.PutCluster(updated); err != nil {
		t.Fatal(err)
	}
	if got, _ := inv.Cluster(Ref{"dev", "pg17"}); got.Profile.Host != "127.0.0.1" || len(inv.Clusters) != 2 {
		t.Errorf("PutCluster should replace in place: %+v (%d clusters)", got, len(inv.Clusters))
	}

	if _, err := inv.PutEnvironment("dev", EnvStaging); err != nil {
		t.Fatal(err)
	}
	if e, _ := inv.Environment("dev"); e.Kind != EnvStaging || e.Policy != DefaultPolicy(EnvStaging) {
		t.Errorf("changing the kind must reset the policy to the kind's default: %+v", e)
	}

	if err := inv.RemoveEnvironment("dev"); !errors.Is(err, ErrEnvNotEmpty) {
		t.Errorf("RemoveEnvironment(dev): err = %v, want ErrEnvNotEmpty", err)
	}
	if err := inv.RemoveCluster(Ref{"dev", "pg17"}); err != nil {
		t.Fatal(err)
	}
	if err := inv.RemoveCluster(Ref{"dev", "pg17"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("second RemoveCluster: err = %v, want ErrNotFound", err)
	}
	if err := inv.RemoveEnvironment("dev"); err != nil {
		t.Fatal(err)
	}
	if err := inv.RemoveEnvironment("dev"); !errors.Is(err, ErrNotFound) {
		t.Errorf("second RemoveEnvironment: err = %v, want ErrNotFound", err)
	}
}

func TestListIsSorted(t *testing.T) {
	t.Parallel()
	inv := sample(t)
	var refs []string
	for _, c := range inv.List() {
		refs = append(refs, c.Ref().String())
	}
	if strings.Join(refs, ",") != "dev/pg17,prod/orders" {
		t.Fatalf("List() = %v", refs)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("TUSKLINE_INVENTORY", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if got, _ := DefaultPath(); got != filepath.Join("/xdg", "tuskline", "inventory.yaml") {
		t.Errorf("DefaultPath() with XDG_CONFIG_HOME = %q", got)
	}
	t.Setenv("TUSKLINE_INVENTORY", "/custom/inv.yaml")
	if got, _ := DefaultPath(); got != "/custom/inv.yaml" {
		t.Errorf("DefaultPath() with TUSKLINE_INVENTORY = %q", got)
	}
}

func TestSecretRef(t *testing.T) {
	t.Setenv("TL_TEST_SET", secretValue)
	if v, err := (SecretRef{Env: "TL_TEST_SET"}).Resolve(); err != nil || v != secretValue {
		t.Errorf("env reference: matched %v, %v", v == secretValue, err)
	}
	if _, err := (SecretRef{Env: "TL_TEST_UNSET_VAR"}).Resolve(); !errors.Is(err, ErrSecretUnavailable) {
		t.Errorf("unset env: err = %v, want ErrSecretUnavailable", err)
	}
	if _, err := (SecretRef{Keychain: "tuskline/prod/orders"}).Resolve(); !errors.Is(err, ErrKeychainNotImplemented) {
		t.Errorf("keychain: err = %v, want ErrKeychainNotImplemented", err)
	}
	if err := (SecretRef{Env: "A", Keychain: "b"}).Validate(); err == nil {
		t.Error("both fields set: want an error")
	}
}
