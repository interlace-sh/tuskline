// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"errors"
	"testing"
)

func TestKeychainRoundTrip(t *testing.T) {
	ring := NewMemoryKeyring()
	UseKeyring(ring)
	t.Cleanup(func() { UseKeyring(nil) })

	ref, err := StorePassword("dev", "pg17", secretValue)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Keychain != "tuskline/dev/pg17" {
		t.Fatalf("item %q", ref.Keychain)
	}
	got, err := ref.Resolve()
	if err != nil || got != secretValue {
		t.Fatalf("resolve matched %v, %v", got == secretValue, err)
	}
	if err := DeletePassword(ref.Keychain); err != nil {
		t.Fatal(err)
	}
	if _, err := ref.Resolve(); !errors.Is(err, ErrKeychainNotFound) {
		t.Fatalf("after delete: %v", err)
	}
}

func TestKeychainRejectsOddNames(t *testing.T) {
	t.Parallel()
	if _, err := StorePassword("Dev", "pg", "x"); err == nil {
		t.Fatal("uppercase environment was accepted")
	}
	if err := (SecretRef{Keychain: "other/app"}).Validate(); err == nil {
		t.Fatal("foreign keychain item was accepted")
	}
	if _, err := (SecretRef{Keychain: "not a name"}).Resolve(); err == nil {
		t.Fatal("invalid item was resolved")
	}
}
