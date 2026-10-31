// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"testing"
	"time"

	"github.com/interlace-sh/tuskline/core/conn"
)

func TestHandoverRejectsAPasswordInThePatroniURL(t *testing.T) {
	t.Parallel()
	inv := &Inventory{Version: FileVersion}
	if _, err := inv.PutEnvironment("dev", EnvDev); err != nil {
		t.Fatal(err)
	}
	cluster := Cluster{
		Name: "demo", Environment: "dev",
		Profile: conn.Profile{Host: "127.0.0.1", User: "tuskline_monitor", Database: "postgres", TLS: conn.TLSDisable},
		Patroni: &Patroni{URL: "http://user:secret@127.0.0.1:8008"},
	}
	if err := inv.PutCluster(cluster); err == nil {
		t.Fatal("a Patroni URL with a password was stored")
	}
	cluster.Patroni = &Patroni{URL: "http://127.0.0.1:8008"}
	cluster.Backup = &Backup{Tool: "pgbackrest", Stanza: "demo", Repo: "local-volume", Path: "/var/lib/pgbackrest"}
	cluster.RestoreDrill = &RestoreDrill{Status: "scheduled", Due: time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC), Command: "tl backup verify"}
	if err := inv.PutCluster(cluster); err != nil {
		t.Fatal(err)
	}
}
