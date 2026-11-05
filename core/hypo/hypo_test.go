// SPDX-License-Identifier: Apache-2.0

package hypo

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()
	if err := validate("CREATE INDEX orders_note ON orders (note)", "SELECT note FROM orders"); err != nil {
		t.Fatal(err)
	}
	if err := validate("CREATE INDEX CONCURRENTLY orders_note ON orders (note)", "SELECT 1"); err == nil {
		t.Fatal("expected concurrently to be rejected")
	}
	if err := validate("CREATE INDEX orders_note ON orders (note)", "DELETE FROM orders"); err == nil {
		t.Fatal("expected a non-select to be rejected")
	}
}
