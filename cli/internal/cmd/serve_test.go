// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestExitCode(t *testing.T) {
	var buf bytes.Buffer
	if got := exitCode(nil, &buf); got != 0 || buf.Len() != 0 {
		t.Fatalf("nil: %d %q", got, buf.String())
	}
	if got := exitCode(exitCodeError{code: 4}, &buf); got != 4 || buf.Len() != 0 {
		t.Fatalf("child: %d %q", got, buf.String())
	}
	if got := exitCode(errors.New("nope"), &buf); got != 1 || !strings.Contains(buf.String(), "tl: nope") {
		t.Fatalf("other: %d %q", got, buf.String())
	}
}

func TestServeMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	out, err := run(t, "serve")
	if err == nil {
		t.Fatal("serve found a server binary")
	}
	for _, want := range []string{"not on PATH", "moon run server:bin", "docker build -t tuskline"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q:\n%s", want, err)
		}
	}
	if out != "" {
		t.Errorf("stdout %q", out)
	}
}

func TestServeHelpDoesNotNeedTheBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	out, err := run(t, "serve", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "tuskline serve") {
		t.Fatalf("help:\n%s", out)
	}
}
