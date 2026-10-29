// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

const serverBinary = "tuskline"

// exitCodeError makes tl exit with a child's code without printing more.
type exitCodeError struct{ code int }

func (e exitCodeError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve [args]",
		Short: "Start the web UI and API (runs the separate tuskline server binary)",
		Long: `Run "tuskline serve [args]". The server is a separate AGPL binary that tl
never links (docs/adr/0010-licence-boundary.md); tl looks for it on PATH,
then next to tl itself. Signals sent to tl are passed on to the server, and
tl exits with the server's exit code.`,
		DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
				return c.Help()
			}
			bin, err := findServer()
			if err != nil {
				return err
			}
			// The command context is not cancelled by the signals we forward, so
			// the child exits on its own. A cancelled caller still stops it.
			p := exec.CommandContext(c.Context(), bin, append([]string{"serve"}, args...)...)
			p.Stdin, p.Stdout, p.Stderr = os.Stdin, c.OutOrStdout(), c.ErrOrStderr()
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, forwardedSignals...)
			defer signal.Stop(sigs)
			return runChild(p, sigs)
		},
	}
}

// findServer looks on PATH, then beside the running tl.
func findServer() (string, error) {
	if bin, err := exec.LookPath(serverBinary); err == nil {
		return bin, nil
	}
	name := serverBinary
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var beside string
	if self, err := os.Executable(); err == nil {
		beside = filepath.Join(filepath.Dir(self), name)
		if st, err := os.Stat(beside); err == nil && !st.IsDir() {
			return beside, nil
		}
	}
	msg := "tl serve runs the separate tuskline server binary, which is not on PATH"
	if beside != "" {
		msg += " or in " + filepath.Dir(beside)
	}
	return "", errors.New(msg + `. Either:
  - install it from the same release as tl and put it on PATH, or
  - build it from source: moon run server:bin (writes ./bin/tuskline), or
  - run it in a container, from a source checkout:
      docker build -t tuskline . && docker run --rm -p 8080:8080 -v tuskline:/data tuskline`)
}

// runChild starts p, passes every signal from sigs on to it until it
// exits, and returns its exit code as an exitCodeError.
func runChild(p *exec.Cmd, sigs <-chan os.Signal) error {
	if err := p.Start(); err != nil {
		return fmt.Errorf("start %s: %w", p.Path, err)
	}
	done := make(chan error, 1)
	go func() { done <- p.Wait() }()
	for {
		select {
		case s := <-sigs:
			_ = p.Process.Signal(s) // the child may already be exiting
		case err := <-done:
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				if code := ee.ExitCode(); code > 0 {
					return exitCodeError{code: code}
				}
				return exitCodeError{code: 1} // killed by a signal
			}
			return err
		}
	}
}
