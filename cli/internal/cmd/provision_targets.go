// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/provision"
)

func newCNPGCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cnpg",
		Short: "CloudNativePG manifests",
	}
	var spec provision.CNPGSpec
	plan := &cobra.Command{
		Use:   "plan",
		Short: "Print the Cluster, Pooler and Barman Cloud manifests. Nothing is applied.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			built, err := provision.CNPG(spec)
			if err != nil {
				return err
			}
			return showCompose(c.OutOrStdout(), built, "CloudNativePG plan. Nothing has been applied.")
		},
	}
	plan.Flags().StringVar(&spec.Name, "name", "", "cluster name")
	plan.Flags().StringVar(&spec.Namespace, "namespace", "default", "Kubernetes namespace")
	plan.Flags().IntVar(&spec.Major, "pg", 17, "PostgreSQL major, 14–18")
	plan.Flags().IntVar(&spec.Instances, "instances", 3, "instances, 1 to 5")
	plan.Flags().BoolVar(&spec.Pooler, "pooler", true, "include a Pooler")
	plan.Flags().StringVar(&spec.Storage, "storage", "10Gi", "volume size")
	plan.Flags().StringVar(&spec.Destination, "destination", "", "Barman object URL (default s3://demo-backups/<name>)")
	_ = plan.MarkFlagRequired("name")
	cmd.AddCommand(plan)
	return cmd
}

func newSSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh",
		Short: "Patroni on virtual machines",
	}
	var spec provision.SSHSpec
	var hosts string
	plan := &cobra.Command{
		Use:   "plan",
		Short: "Print Patroni, systemd and pgBackRest files. Nothing is installed.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			spec.Hosts = splitHosts(hosts)
			built, err := provision.SSH(spec)
			if err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), "SSH plan. Nothing has been installed.")
			return showCompose(c.OutOrStdout(), built, "Files:")
		},
	}
	plan.Flags().StringVar(&spec.Name, "name", "", "cluster name")
	plan.Flags().IntVar(&spec.Major, "pg", 17, "PostgreSQL major, 14–18")
	plan.Flags().StringVar(&hosts, "hosts", "", "comma-separated hostnames or addresses")
	plan.Flags().BoolVar(&spec.Autobase, "autobase", false, "also write an Autobase inventory; the playbook is not run")
	_ = plan.MarkFlagRequired("name")
	_ = plan.MarkFlagRequired("hosts")
	cmd.AddCommand(plan)
	return cmd
}

func splitHosts(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
