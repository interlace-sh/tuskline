// SPDX-License-Identifier: Apache-2.0

package inventory

import "testing"

func TestProdIsReadOnlyByDefault(t *testing.T) {
	p := DefaultPolicy(EnvProd)
	if !p.ReadOnly {
		t.Fatal("prod must be read-only by default (safety invariant)")
	}
	if !p.RequireTypedConfirmation {
		t.Fatal("prod must require typed confirmation")
	}
}

func TestParseRef(t *testing.T) {
	cases := []struct {
		in      string
		want    Ref
		wantErr bool
	}{
		{"prod/orders", Ref{"prod", "orders"}, false},
		{"staging/eu-west-1", Ref{"staging", "eu-west-1"}, false},
		{"orders", Ref{}, true},
		{"Prod/orders", Ref{}, true},
		{"prod/", Ref{}, true},
	}
	for _, c := range cases {
		got, err := ParseRef(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseRef(%q) err = %v, wantErr %v", c.in, err, c.wantErr)
			continue
		}
		if got != c.want {
			t.Errorf("ParseRef(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestEnvKindValid(t *testing.T) {
	for _, k := range []EnvKind{EnvLocal, EnvDev, EnvStaging, EnvProd} {
		if !k.Valid() {
			t.Errorf("%s should be valid", k)
		}
	}
	if EnvKind("qa").Valid() {
		t.Error("qa should not be valid")
	}
}
