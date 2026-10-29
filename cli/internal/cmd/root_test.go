// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	root := NewRoot(&out, &out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestVersion(t *testing.T) {
	out, err := run(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "tl ") {
		t.Fatalf("got %q", out)
	}
}

func TestConnectRejectsBadRef(t *testing.T) {
	if _, err := run(t, "connect", "orders"); err == nil || !strings.Contains(err.Error(), "env/cluster") {
		t.Fatalf("got %v", err)
	}
}

func TestStubsSayNotYet(t *testing.T) {
	for _, args := range [][]string{{"ls"}, {"top", "prod/orders"}, {"connect", "prod/orders"}} {
		if _, err := run(t, args...); err == nil || !strings.Contains(err.Error(), "not implemented yet") {
			t.Errorf("%v: got %v", args, err)
		}
	}
}
