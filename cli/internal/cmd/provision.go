// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/provision"
)

func newProvisionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provision",
		Short: "Plan a deployment",
	}
	cmd.AddCommand(newComposeCmd())
	return cmd
}

func newComposeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compose",
		Short: "Non-production Patroni stack for one Docker host",
	}
	var spec provision.ComposeSpec
	var pooler bool
	var dir string
	flags := cmd.PersistentFlags()
	flags.StringVar(&spec.Name, "name", "", "cluster name (lowercase letters, digits and hyphens)")
	flags.IntVar(&spec.Major, "pg", 17, "PostgreSQL major, 14–18")
	flags.IntVar(&spec.Nodes, "nodes", 3, "Postgres nodes, 1, 2 or 3")
	flags.IntVar(&spec.Etcd, "etcd", 0, "etcd members, 1 or 3 (default 1 for one node, otherwise 3)")
	flags.BoolVar(&pooler, "pooler", true, "include PgBouncer")
	flags.BoolVar(&spec.Sync, "sync", false, "ask Patroni for synchronous replication")
	flags.StringVar(&dir, "dir", "", "directory to write the stack into")

	plan := &cobra.Command{
		Use:   "plan",
		Short: "Print the Compose files. Nothing is started.",
		Long: `Generate compose.yaml, haproxy.cfg, pgbackrest.conf and a secret .env
for a Patroni cluster on one Docker host. The diff is a preview.
Compose is non-production: one host is a single point of failure.

up writes those files and starts the stack.`,
		Example: `  tl provision compose plan --name demo
  tl provision compose plan --name demo --pg 16 --nodes 1 --etcd 1`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			spec.Pooler = pooler
			built, err := provision.Compose(spec)
			if err != nil {
				return err
			}
			return showCompose(c.OutOrStdout(), built, "Compose plan. Non-production. Nothing has been applied.")
		},
	}
	up := &cobra.Command{
		Use:   "up",
		Short: "Write the stack and start it. Waits for health checks.",
		Long: `Write the Compose files into --dir and run docker compose up.
The command waits until etcd and Patroni health checks pass, up to 10 minutes
after the containers have started. Before anything is written it checks Docker,
free disk, the published ports and leftover volumes. Passwords are written
into the directory as mode 0600 files and are not printed.

Compose is non-production: one host is a single point of failure.`,
		Example: `  tl provision compose up --name demo --dir ~/clusters/demo --nodes 1 --etcd 1`,
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			spec.Pooler = pooler
			if err := provision.CheckOutputDir(dir); err != nil {
				return err
			}
			need, err := provision.Requirements(spec)
			if err != nil {
				return err
			}
			probeCtx, cancelProbe := context.WithTimeout(c.Context(), 15*time.Second)
			report, err := probeCompose(probeCtx, need, dir)
			cancelProbe()
			if err != nil {
				return err
			}
			if err := provision.Evaluate(need, report); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), provision.Summary(need, report))
			built, err := provision.ComposeStart(spec)
			if err != nil {
				return err
			}
			if err := showCompose(c.OutOrStdout(), built, "Compose plan. Non-production."); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), "Applying. Non-production. Passwords stay in the directory.")
			if err := provision.RefuseDataDirs(c.Context(), built, provision.OSFS{Root: dir}); err != nil {
				return err
			}
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return fmt.Errorf("create %s: %w", dir, err)
			}
			ctx, cancel := context.WithTimeout(c.Context(), 15*time.Minute)
			defer cancel()
			fs := provision.OSFS{Root: dir}
			return provision.Apply(ctx, built, provision.FileJournal{Path: filepath.Join(dir, ".tuskline-apply.jsonl")}, fs, dockerExec{dir: dir, stdout: c.OutOrStdout(), stderr: c.ErrOrStderr()}, func(ev provision.Event) {
				fmt.Fprintf(c.OutOrStdout(), "%s  %s  %s\n", ev.Status, ev.Kind, ev.Text)
			}, fs)
		},
	}
	_ = plan.MarkFlagRequired("name")
	_ = up.MarkFlagRequired("name")
	_ = up.MarkFlagRequired("dir")
	cmd.AddCommand(plan, up)
	return cmd
}

func showCompose(w io.Writer, plan provision.Plan, header string) error {
	changes, err := provision.Changes(plan)
	if err != nil {
		return err
	}
	return printComposePlan(w, header, changes)
}

func printComposePlan(w io.Writer, header string, changes []provision.Change) error {
	fmt.Fprintln(w, header)
	for _, change := range changes {
		fmt.Fprintf(w, "\n# %s  %s\n", change.Kind, change.Path)
		if strings.TrimSpace(change.Diff) == "" {
			fmt.Fprintln(w, "# unchanged")
			continue
		}
		fmt.Fprint(w, change.Diff)
	}
	return nil
}

// dockerExec runs a command step in the output directory. It does not put
// passwords on the command line; Compose reads them from .env.
type dockerExec struct {
	dir    string
	stdout io.Writer
	stderr io.Writer
}

func (d dockerExec) Run(ctx context.Context, step provision.Step) error {
	cmd := exec.CommandContext(ctx, step.Argv[0], step.Argv[1:]...)
	cmd.Dir = d.dir
	cmd.Stdout = d.stdout
	cmd.Stderr = d.stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}
	return nil
}
