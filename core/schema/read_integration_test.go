// SPDX-License-Identifier: Apache-2.0

//go:build integration

package schema

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/pgtest"
)

func TestReadAndDiffAcrossVersions(t *testing.T) {
	for _, major := range pgtest.Versions(t) {
		t.Run(fmt.Sprintf("pg%d", major), func(t *testing.T) {
			profile := pgtest.Start(t, major).Profile
			db := connect(t, profile)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			pgtest.Exec(t, profile,
				"CREATE SCHEMA billing",
				"CREATE TYPE billing.refund_reason AS ENUM ('damaged', 'late', 'other')",
				"CREATE TABLE public.orders (id int PRIMARY KEY, note text, reason billing.refund_reason)",
				"CREATE INDEX orders_note ON public.orders (note)",
				"CREATE VIEW public.open_orders AS SELECT id FROM public.orders",
				"CREATE FUNCTION public.order_tag() RETURNS int LANGUAGE sql AS $$ SELECT 1 $$",
				"CREATE SCHEMA tuskline",
				"CREATE TABLE tuskline.hidden (id int)",
			)
			before, err := Read(ctx, db)
			if err != nil {
				t.Fatal(err)
			}
			if hasName(before, "tuskline.hidden.id") {
				t.Fatal("migration schema was included")
			}
			pgtest.Exec(t, profile,
				"ALTER TYPE billing.refund_reason ADD VALUE 'fraud'",
				"ALTER TABLE public.orders ALTER COLUMN note SET NOT NULL",
				"DROP INDEX public.orders_note",
			)
			after, err := Read(ctx, db)
			if err != nil {
				t.Fatal(err)
			}
			report := Report(Diff(before, after))
			for _, want := range []string{
				"~ column public.orders.note  text nullable -> text not null",
				"~ enum billing.refund_reason  damaged, late, other -> damaged, late, other, fraud",
				"- index public.orders_note  ",
			} {
				if !strings.Contains(report, want) {
					t.Fatalf("missing %q in\n%s", want, report)
				}
			}
			if strings.Contains(report, "tuskline.") {
				t.Fatalf("migration schema drifted:\n%s", report)
			}
			if !hasName(before, "public.open_orders") || !hasName(before, "public.order_tag()") {
				t.Fatalf("view or function missing: %+v", before.Objects)
			}
		})
	}
}

func connect(t *testing.T, profile conn.Profile) *pgx.Conn {
	t.Helper()
	db, err := conn.Connect(context.Background(), profile, conn.SessionSettings{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(context.Background()) })
	return db
}

func hasName(snap Snapshot, name string) bool {
	for _, object := range snap.Objects {
		if object.Name == name {
			return true
		}
	}
	return false
}
