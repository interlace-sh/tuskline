// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"os"
	"testing"
)

func TestDiffReport(t *testing.T) {
	t.Parallel()
	before := Snapshot{Objects: []Object{
		{Kind: KindColumn, Name: "public.orders.currency", Detail: "character(3) nullable"},
		{Kind: KindEnum, Name: "billing.refund_reason", Detail: "damaged, late, other"},
		{Kind: KindIndex, Name: "public.orders_note", Detail: "CREATE INDEX orders_note ON public.orders USING btree (note)"},
	}}
	after := Snapshot{Objects: []Object{
		{Kind: KindColumn, Name: "public.orders.currency", Detail: "character(3) not null"},
		{Kind: KindColumn, Name: "public.orders.note", Detail: "text nullable"},
		{Kind: KindEnum, Name: "billing.refund_reason", Detail: "damaged, late, other, fraud"},
	}}
	got := Report(Diff(before, after))
	want, err := os.ReadFile("testdata/drift.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("report:\n%s\nwant:\n%s", got, want)
	}
	if Report(Diff(before, before)) != "No drift.\n" {
		t.Fatal(Report(Diff(before, before)))
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	if _, err := Parse([]byte(`{"objects":[{"kind":"column","name":"a","detail":"int","extra":1}]}`)); err == nil {
		t.Fatal("expected unknown field to fail")
	}
}
