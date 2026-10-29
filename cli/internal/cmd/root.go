// SPDX-License-Identifier: Apache-2.0

// Package cmd defines the tl command tree. See docs/cli.md for the spec.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/agent"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/version"
)

// errNotYet marks commands that are specified but not built.
func errNotYet(milestone string) error {
	return fmt.Errorf("not implemented yet (roadmap %s, see docs/roadmap.md)", milestone)
}

// NewRoot builds the command tree. out/errOut are injectable for tests.
func NewRoot(out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "tl",
		Short:         "Tuskline: a keyboard-first PostgreSQL workbench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(out)
	root.SetErr(errOut)

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		Run: func(c *cobra.Command, _ []string) {
			fmt.Fprintf(c.OutOrStdout(), "tl %s\n", version.String())
		},
	})

	var dsn string
	connect := &cobra.Command{
		Use:   "connect <env/cluster>",
		Short: "Open a psql-style session through the inventory (tunnels and IAM handled)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if dsn != "" {
				info, err := conn.Probe(c.Context(), dsn)
				if err != nil {
					return err
				}
				if err := conn.CheckSupported(info.VersionNum); err != nil {
					return err
				}
				fmt.Fprintf(c.OutOrStdout(), "connected: PostgreSQL %d (read-only=%v)\n%s\n", info.Major(), info.ReadOnly, info.Version)
				return nil
			}
			if len(args) == 1 {
				if _, err := inventory.ParseRef(args[0]); err != nil {
					return err
				}
			}
			return errNotYet("M1")
		},
	}
	connect.Flags().StringVar(&dsn, "dsn", "", "probe a libpq URL directly, bypassing the inventory")
	root.AddCommand(connect)

	root.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List environments and clusters in the inventory",
		Args:  cobra.NoArgs,
		RunE:  func(*cobra.Command, []string) error { return errNotYet("M1") },
	})
	root.AddCommand(&cobra.Command{
		Use:   "top <env/cluster>",
		Short: "Live activity, waits and locks (TUI)",
		Args:  cobra.ExactArgs(1),
		RunE:  func(*cobra.Command, []string) error { return errNotYet("M1") },
	})

	root.AddCommand(&cobra.Command{
		Use:                "serve",
		Short:              "Start the web UI and API (runs the separate tuskline server binary)",
		DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error {
			bin, err := exec.LookPath("tuskline")
			if err != nil {
				return errors.New("the tuskline server binary was not found on PATH; install it or run the Docker image")
			}
			// A child process rather than exec(2) so this works on Windows too.
			p := exec.CommandContext(c.Context(), bin, append([]string{"serve"}, args...)...)
			p.Stdin, p.Stdout, p.Stderr = os.Stdin, c.OutOrStdout(), c.ErrOrStderr()
			return p.Run()
		},
	})

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
	if err := root.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "tl:", err)
		return 1
	}
	return 0
}
