// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/schema"
)

func newMigrateDiffCmd(a *app) *cobra.Command {
	var fromFile, toFile, fromCluster, toCluster string
	c := &cobra.Command{
		Use:   "diff",
		Short: "Compare two schema snapshots. Nothing is changed.",
		Long: `Compare the structure of two databases, or two saved snapshots.
The report lists columns, constraints, enums, indexes, views and routines
that were added, removed or redefined. Table rows are not read.

A difference exits 1. System catalogues and the tuskline migration schema
are not compared.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			before, fromLabel, err := loadSide(c, a, fromFile, fromCluster, "from")
			if err != nil {
				return err
			}
			after, toLabel, err := loadSide(c, a, toFile, toCluster, "to")
			if err != nil {
				return err
			}
			report := schema.Report(schema.Diff(before, after))
			fmt.Fprintf(c.OutOrStdout(), "%s -> %s\n%s", fromLabel, toLabel, report)
			if report != "No drift.\n" {
				return errors.New("schema drift")
			}
			return nil
		},
	}
	c.Flags().StringVar(&fromFile, "from-file", "", "baseline snapshot JSON")
	c.Flags().StringVar(&toFile, "to-file", "", "later snapshot JSON")
	c.Flags().StringVar(&fromCluster, "from", "", "baseline inventory cluster, as env/name")
	c.Flags().StringVar(&toCluster, "to", "", "later inventory cluster, as env/name")
	return c
}

func loadSide(c *cobra.Command, a *app, file, cluster, which string) (schema.Snapshot, string, error) {
	switch {
	case file != "" && cluster != "":
		return schema.Snapshot{}, "", fmt.Errorf("--%s-file and --%s are both set", which, which)
	case file != "":
		snap, err := schema.ReadFile(file)
		return snap, file, err
	case cluster != "":
		ref, err := inventory.ParseRef(cluster)
		if err != nil {
			return schema.Snapshot{}, "", err
		}
		sc, err := a.loadCluster(c, ref)
		if err != nil {
			return schema.Snapshot{}, "", err
		}
		db, err := conn.Connect(c.Context(), sc.profile, sc.env.Policy.Session())
		if err != nil {
			return schema.Snapshot{}, "", err
		}
		defer func() { _ = db.Close(context.Background()) }()
		snap, err := schema.Read(c.Context(), db)
		return snap, ref.String(), err
	default:
		return schema.Snapshot{}, "", fmt.Errorf("pass --%s-file or --%s", which, which)
	}
}
