// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"strings"

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
	plan := &cobra.Command{
		Use:   "plan",
		Short: "Print the Compose files. Nothing is started.",
		Long: `Generate compose.yaml, haproxy.cfg, pgbackrest.conf and a secret .env
for a Patroni cluster on one Docker host. The diff is a preview.
Compose is non-production: one host is a single point of failure.

Starting the stack is a later step.`,
		Example: `  tl provision compose plan --name demo
  tl provision compose plan --name demo --pg 16 --nodes 1 --etcd 1`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			spec.Pooler = pooler
			built, err := provision.Compose(spec)
			if err != nil {
				return err
			}
			changes, err := provision.Changes(built)
			if err != nil {
				return err
			}
			return printComposePlan(c.OutOrStdout(), changes)
		},
	}
	plan.Flags().StringVar(&spec.Name, "name", "", "cluster name (lowercase letters, digits and hyphens)")
	plan.Flags().IntVar(&spec.Major, "pg", 17, "PostgreSQL major, 14–18")
	plan.Flags().IntVar(&spec.Nodes, "nodes", 3, "Postgres nodes, 1, 2 or 3")
	plan.Flags().IntVar(&spec.Etcd, "etcd", 0, "etcd members, 1 or 3 (default 1 for one node, otherwise 3)")
	plan.Flags().BoolVar(&pooler, "pooler", true, "include PgBouncer")
	plan.Flags().BoolVar(&spec.Sync, "sync", false, "ask Patroni for synchronous replication")
	_ = plan.MarkFlagRequired("name")
	cmd.AddCommand(plan)
	return cmd
}

func printComposePlan(w io.Writer, changes []provision.Change) error {
	fmt.Fprintln(w, "Compose plan. Non-production. Nothing has been applied.")
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
