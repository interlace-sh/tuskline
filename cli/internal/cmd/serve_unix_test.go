// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fakeServer puts an executable named tuskline on PATH. body is a shell script.
func fakeServer(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tuskline")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestServePassesArgsAndExitCode(t *testing.T) {
	out := filepath.Join(t.TempDir(), "args")
	t.Setenv("TL_SERVE_OUT", out)
	fakeServer(t, `printf '%s\n' "$*" > "$TL_SERVE_OUT"; exit 4`)

	var stdout, stderr bytes.Buffer
	root := NewRoot(&stdout, &stderr)
	root.SetArgs([]string{"serve", "--addr", "127.0.0.1:9"})
	err := root.Execute()
	if exitCode(err, &stderr) != 4 {
		t.Fatalf("exit %v, stderr %q", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("child status was printed: %q", stderr.String())
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "serve --addr 127.0.0.1:9\n" {
		t.Fatalf("args %q", got)
	}
}

func TestRunChildForwardsSignals(t *testing.T) {
	for _, tc := range []struct {
		sig  syscall.Signal
		mark string
		code int
	}{
		{syscall.SIGTERM, "TERM", 9},
		{syscall.SIGHUP, "HUP", 8},
	} {
		t.Run(tc.mark, func(t *testing.T) {
			dir := t.TempDir()
			script := filepath.Join(dir, "child.sh")
			body := `#!/bin/sh
trap 'echo TERM >> "$OUT"; exit 9' TERM
trap 'echo HUP >> "$OUT"; exit 8' HUP
echo ready > "$OUT"
while true; do sleep 0.05; done
`
			if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(dir, "out")
			p := exec.CommandContext(context.Background(), script)
			p.Env = append(os.Environ(), "OUT="+out)
			sigs := make(chan os.Signal, 1)
			errc := make(chan error, 1)
			go func() { errc <- runChild(p, sigs) }()

			deadline := time.Now().Add(2 * time.Second)
			for {
				b, err := os.ReadFile(out)
				if err == nil && strings.Contains(string(b), "ready") {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("child did not start")
				}
				time.Sleep(5 * time.Millisecond)
			}
			sigs <- tc.sig

			select {
			case err := <-errc:
				var ec exitCodeError
				if !errors.As(err, &ec) || ec.code != tc.code {
					t.Fatalf("got %v, want exit %d", err, tc.code)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("child did not exit")
			}
			b, err := os.ReadFile(out)
			if err != nil || !strings.Contains(string(b), tc.mark) {
				t.Fatalf("mark %q not recorded: %q (%v)", tc.mark, b, err)
			}
		})
	}
}
