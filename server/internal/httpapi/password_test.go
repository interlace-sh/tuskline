// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

func TestResolvePasswordFallsThroughToPgpass(t *testing.T) {
	os.Unsetenv("TL_TEST_UNSET_PASSWORD")
	dir := t.TempDir()
	path := filepath.Join(dir, "pgpass")
	if err := os.WriteFile(path, []byte("127.0.0.1:5417:tuskline:tuskline:from-pgpass\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := conn.Profile{Host: "127.0.0.1", Port: 5417, Database: "tuskline", User: "tuskline"}
	got, err := resolvePassword(profile, &inventory.SecretRef{Env: "TL_TEST_UNSET_PASSWORD"}, path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasPassword() {
		t.Fatal("pgpass password was not applied")
	}
}

func TestResolvePasswordKeepsOtherSecretErrors(t *testing.T) {
	t.Parallel()
	_, err := resolvePassword(conn.Profile{Host: "db", User: "u"}, &inventory.SecretRef{Keychain: "item"}, "")
	if err == nil {
		t.Fatal("keychain reference was ignored")
	}
}
