// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/backup"
)

func newBackupCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{Use: "backup", Short: "Read backups and preview a restore"}
	cmd.AddCommand(newBackupStatusCmd(), newBackupRestoreCmd(a), newBackupVerifyCmd(a))
	return cmd
}

func newBackupStatusCmd() *cobra.Command {
	var file, kind string
	c := &cobra.Command{
		Use:   "status",
		Short: "Print pgBackRest or CloudNativePG backups. Nothing is restored.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			raw, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if kind == "cnpg" {
				items, err := backup.ParseCNPG(raw)
				if err != nil {
					return err
				}
				for _, item := range items {
					fmt.Fprintf(c.OutOrStdout(), "%s %s\n", item.Name, item.Phase)
				}
				return nil
			}
			report, err := backup.ParseInfo(raw)
			if err != nil {
				return err
			}
			fmt.Fprint(c.OutOrStdout(), report.Text())
			return nil
		},
	}
	c.Flags().StringVar(&file, "file", "", "pgbackrest info JSON, or CloudNativePG backup JSON with --kind cnpg")
	c.Flags().StringVar(&kind, "kind", "pgbackrest", "pgbackrest or cnpg")
	_ = c.MarkFlagRequired("file")
	return c
}

func newBackupRestoreCmd(a *app) *cobra.Command {
	var file, stanza, target, env, cluster, confirm string
	var prod, apply, run bool
	c := &cobra.Command{
		Use:   "restore",
		Short: "Preview a point-in-time restore. --apply records it; --run starts pgbackrest.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			raw, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			report, err := backup.ParseInfo(raw)
			if err != nil {
				return err
			}
			when, err := time.Parse(time.RFC3339, target)
			if err != nil {
				return fmt.Errorf("target time: %w", err)
			}
			action := backup.Restore{Stanza: stanza, To: when, Prod: prod, Confirm: confirm, Cluster: cluster}
			text, err := action.Preview(report)
			if err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), text)
			if !apply && !run {
				return nil
			}
			if err := recordAction(c.Context(), a, env, cluster, "restore", text, ""); err != nil {
				return err
			}
			if !run {
				fmt.Fprintln(c.OutOrStdout(), "Recorded. Pass --run to start pgbackrest.")
				return nil
			}
			cmd := exec.CommandContext(c.Context(), "pgbackrest", "--stanza="+stanza, "restore", "--type=time", "--target="+when.UTC().Format(time.RFC3339))
			cmd.Stdout = c.OutOrStdout()
			cmd.Stderr = c.ErrOrStderr()
			return cmd.Run()
		},
	}
	c.Flags().StringVar(&file, "file", "", "pgbackrest info JSON")
	c.Flags().StringVar(&stanza, "stanza", "", "stanza name")
	c.Flags().StringVar(&target, "to", "", "target time, RFC3339")
	c.Flags().StringVar(&env, "env", "dev", "environment name")
	c.Flags().StringVar(&cluster, "cluster", "", "cluster name")
	c.Flags().BoolVar(&prod, "prod", false, "production: --confirm must be the cluster name")
	c.Flags().StringVar(&confirm, "confirm", "", "cluster name, required on prod")
	c.Flags().BoolVar(&apply, "apply", false, "record the restore in the audit log")
	c.Flags().BoolVar(&run, "run", false, "run pgbackrest restore")
	_ = c.MarkFlagRequired("file")
	_ = c.MarkFlagRequired("stanza")
	_ = c.MarkFlagRequired("to")
	_ = c.MarkFlagRequired("cluster")
	return c
}

func newBackupVerifyCmd(a *app) *cobra.Command {
	var stanza, image, env, cluster string
	var run bool
	c := &cobra.Command{
		Use:   "verify",
		Short: "Preview a restore drill in a scratch container. --run starts it.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			argv, err := backup.VerifyCommand(stanza, image)
			if err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), "Restore drill. The original cluster is not changed.")
			fmt.Fprintln(c.OutOrStdout(), join(argv))
			if !run {
				fmt.Fprintln(c.OutOrStdout(), "Nothing has been started.")
				return nil
			}
			if cluster == "" {
				return fmt.Errorf("verify --run needs --cluster")
			}
			if err := recordAction(c.Context(), a, env, cluster, "verify", join(argv), ""); err != nil {
				return err
			}
			cmd := exec.CommandContext(c.Context(), argv[0], argv[1:]...)
			cmd.Stdout = c.OutOrStdout()
			cmd.Stderr = c.ErrOrStderr()
			return cmd.Run()
		},
	}
	c.Flags().StringVar(&stanza, "stanza", "", "stanza name")
	c.Flags().StringVar(&image, "image", "postgres:17", "scratch image")
	c.Flags().StringVar(&env, "env", "dev", "environment name")
	c.Flags().StringVar(&cluster, "cluster", "", "cluster name, recorded when --run is set")
	c.Flags().BoolVar(&run, "run", false, "start the scratch container")
	_ = c.MarkFlagRequired("stanza")
	return c
}

func join(argv []string) string {
	out := argv[0]
	for _, arg := range argv[1:] {
		out += " " + arg
	}
	return out
}
