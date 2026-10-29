// SPDX-License-Identifier: Apache-2.0

// Package cmd defines the tl command tree. See docs/cli.md for the spec.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/agent"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/version"
)

// app holds what commands need from the outside world, so tests can
// replace it.
type app struct {
	in          io.Reader
	interactive bool             // stdin is a terminal, so prompts are allowed
	stdoutTTY   bool             // stdout is a terminal, so the TUI can run
	colour      bool             // stdout is a terminal and the no-colour variable is unset
	now         func() time.Time // for last-probe timestamps
	// inventoryPath is set by --inventory; empty means inventory.DefaultPath.
	inventoryPath string
}

func (a *app) inventoryFile() (string, error) {
	if a.inventoryPath != "" {
		return a.inventoryPath, nil
	}
	return inventory.DefaultPath()
}

// NewRoot builds the command tree. out/errOut are injectable for tests.
func NewRoot(out, errOut io.Writer) *cobra.Command {
	_, noColour := os.LookupEnv("NO_COLOR") //nolint:misspell // the no-color.org variable name
	return newRoot(out, errOut, &app{
		in:          os.Stdin,
		interactive: isTerminal(os.Stdin),
		stdoutTTY:   isTerminal(out),
		colour:      !noColour && isTerminal(out),
		now:         time.Now,
	})
}

func isTerminal(v any) bool {
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func newRoot(out, errOut io.Writer, a *app) *cobra.Command {
	root := &cobra.Command{
		Use:           "tl",
		Short:         "Tuskline: a keyboard-first PostgreSQL workbench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetIn(a.in)
	root.PersistentFlags().StringVar(&a.inventoryPath, "inventory", "", "inventory file (default $TUSKLINE_INVENTORY or $XDG_CONFIG_HOME/tuskline/inventory.yaml)")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newConnectCmd(a))
	root.AddCommand(newLsCmd(a))
	root.AddCommand(newTopCmd(a))
	root.AddCommand(newSnapshotCmd(a))
	root.AddCommand(newDiagnoseCmd(a))
	root.AddCommand(newDiffCmd())
	root.AddCommand(newServeCmd())

	var server string
	ag := &cobra.Command{
		Use:   "agent",
		Short: "Run the outbound collector agent",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return agent.Run(c.Context(), agent.Config{ServerURL: server, Token: os.Getenv("TUSKLINE_AGENT_TOKEN")})
		},
	}
	ag.Flags().StringVar(&server, "server", "", "Tuskline server or Cloud URL")
	root.AddCommand(ag)
	return root
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	root := NewRoot(os.Stdout, os.Stderr)
	return exitCode(root.ExecuteContext(context.Background()), os.Stderr)
}

// exitCode maps an error to the process exit code, printing it unless it
// only carries a child's exit status.
func exitCode(err error, errOut io.Writer) int {
	var ec exitCodeError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ec):
		return ec.code
	default:
		fmt.Fprintln(errOut, "tl:", err)
		return 1
	}
}

func newVersionCmd() *cobra.Command {
	var output string
	c := &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if err := checkOutput(output); err != nil {
				return err
			}
			if output == outputJSON {
				return writeJSON(c.OutOrStdout(), map[string]string{
					"version": version.Version, "commit": version.Commit, "date": version.Date,
					"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH,
				})
			}
			_, err := fmt.Fprintf(c.OutOrStdout(), "tl %s\n", version.String())
			return err
		},
	}
	addOutputFlag(c, &output)
	return c
}
