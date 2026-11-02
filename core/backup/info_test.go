// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"strings"
	"testing"
	"time"
)

func TestParseInfoFindsAWalGap(t *testing.T) {
	t.Parallel()
	raw := []byte(`[
		{"name":"demo","backup":[{"label":"20261031-010000F","type":"full","timestamp":{"start":1761868800,"stop":1761868900}}],
		 "archive":[{"min":"000000010000000000000001","max":"000000010000000000000002"},{"min":"000000010000000000000010","max":"000000010000000000000011"}]}
	]`)
	report, err := ParseInfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Stanzas[0].Gaps) != 1 || report.Stanzas[0].Backups[0].Type != "full" {
		t.Fatalf("%+v", report.Stanzas[0])
	}
	if !strings.Contains(report.Text(), "gap") {
		t.Fatal(report.Text())
	}
	restore := Restore{Stanza: "demo", To: time.Unix(1761868900, 0).UTC(), Cluster: "demo"}
	if _, err := restore.Preview(report); err == nil || !strings.Contains(err.Error(), "WAL gap") {
		t.Fatal(err)
	}
}

func TestRestorePreviewWithoutAGap(t *testing.T) {
	t.Parallel()
	raw := []byte(`[{"name":"demo","backup":[{"label":"20261031-010000F","type":"full","timestamp":{"start":100,"stop":200}}],"archive":[{"min":"01","max":"02"},{"min":"03","max":"04"}]}]`)
	report, err := ParseInfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Stanzas[0].Gaps) != 0 {
		t.Fatalf("contiguous archive looked like a gap: %+v", report.Stanzas[0].Gaps)
	}
	text, err := Restore{Stanza: "demo", To: time.Unix(200, 0).UTC(), Cluster: "demo"}.Preview(report)
	if err != nil || !strings.Contains(text, "Nothing has been restored") {
		t.Fatal(text, err)
	}
	prod := Restore{Stanza: "demo", To: time.Unix(200, 0).UTC(), Cluster: "orders", Prod: true}
	if _, err := prod.Preview(report); err == nil {
		t.Fatal("prod restore ran without the cluster name")
	}
}

func TestVerifyCommandHasNoPassword(t *testing.T) {
	t.Parallel()
	argv, err := VerifyCommand("demo", "")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, "tuskline-verify-demo") || strings.Contains(joined, "PASSWORD") {
		t.Fatal(joined)
	}
	backups, err := ParseCNPG([]byte(`{"items":[{"name":"demo-daily","phase":"completed"}]}`))
	if err != nil || backups[0].Phase != "completed" {
		t.Fatal(err, backups)
	}
}
