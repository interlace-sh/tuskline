// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/migrate"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Lint and preview a folder of SQL migrations",
	}
	cmd.AddCommand(newMigrateLintCmd(), newMigratePlanCmd(), newMigrateApplyCmd())
	return cmd
}

func newMigrateLintCmd() *cobra.Command {
	var dir string
	c := &cobra.Command{
		Use:   "lint",
		Short: "Report lock and rewrite risks. Nothing is applied.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			folder, err := migrate.Load(dir)
			if err != nil {
				return err
			}
			report, err := folder.Lint()
			if err != nil {
				return err
			}
			if report == "" {
				fmt.Fprintln(c.OutOrStdout(), "No lock findings.")
				return nil
			}
			fmt.Fprint(c.OutOrStdout(), report)
			return errors.New("lock findings")
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "migration directory")
	_ = c.MarkFlagRequired("dir")
	return c
}

func newMigratePlanCmd() *cobra.Command {
	var dir string
	c := &cobra.Command{
		Use:   "plan",
		Short: "Print the migration order. Nothing is applied.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			folder, err := migrate.Load(dir)
			if err != nil {
				return err
			}
			fmt.Fprint(c.OutOrStdout(), folder.Preview())
			return nil
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "migration directory")
	_ = c.MarkFlagRequired("dir")
	return c
}

func newMigrateApplyCmd() *cobra.Command {
	var dir, env string
	var approve bool
	c := &cobra.Command{
		Use:   "apply",
		Short: "Preview an apply. Production requires --approve. Nothing is executed.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			folder, err := migrate.Load(dir)
			if err != nil {
				return err
			}
			text, err := folder.Apply(env, approve)
			if err != nil {
				return err
			}
			fmt.Fprint(c.OutOrStdout(), text)
			return nil
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "migration directory")
	c.Flags().StringVar(&env, "env", "dev", "environment kind: local, dev, staging or prod")
	c.Flags().BoolVar(&approve, "approve", false, "approve a production apply")
	_ = c.MarkFlagRequired("dir")
	return c
}
