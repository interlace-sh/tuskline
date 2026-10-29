// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/snapshot"
)

func newSnapshotCmd(a *app) *cobra.Command {
	var (
		file  string
		limit int
		text  int
	)
	c := &cobra.Command{
		Use:   "snapshot <env/cluster>",
		Short: "Save a workload snapshot from pg_stat_statements",
		Long: `Read the current database's top statements and write them to a JSON file.
Query text is stored because you asked for the snapshot. The file is mode 0600.
Diff two files with tl diff.`,
		Example: `  tl snapshot dev/pg17 --file before.json
  tl snapshot dev/pg17 --file after.json
  tl diff before.json after.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("give --file for the snapshot")
			}
			if limit < 1 || text < 0 {
				return fmt.Errorf("--limit must be at least 1 and --query-text must not be negative")
			}
			ref, err := inventory.ParseRef(args[0])
			if err != nil {
				return err
			}
			sc, err := a.loadCluster(c, ref)
			if err != nil {
				return err
			}
			db, err := conn.Connect(c.Context(), sc.profile, sc.env.Policy.Session())
			if err != nil {
				return err
			}
			defer func() { _ = db.Close(context.Background()) }()
			version, err := conn.ServerVersionNum(db)
			if err != nil {
				return err
			}
			snap, err := snapshot.Capture(c.Context(), db, version, text, limit)
			if err != nil {
				return err
			}
			if err := snapshot.WriteFile(file, snap); err != nil {
				return err
			}
			_, err = fmt.Fprintf(c.OutOrStdout(), "wrote %s (%d statements, %s)\n", file, len(snap.Statements), snap.Taken.Format(time.RFC3339))
			return err
		},
	}
	c.Flags().StringVar(&file, "file", "", "path of the snapshot file (required)")
	c.Flags().IntVar(&limit, "limit", 100, "maximum statements to save")
	c.Flags().IntVar(&text, "query-text", 500, "maximum query text length, in characters")
	return c
}

func newDiffCmd() *cobra.Command {
	var output string
	c := &cobra.Command{
		Use:   "diff <older> <newer>",
		Short: "Show how a workload changed between two snapshots",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			if err := checkOutput(output); err != nil {
				return err
			}
			older, err := snapshot.ReadFile(args[0])
			if err != nil {
				return err
			}
			newer, err := snapshot.ReadFile(args[1])
			if err != nil {
				return err
			}
			deltas := snapshot.Diff(older, newer)
			if output == outputJSON {
				return writeJSON(c.OutOrStdout(), map[string]any{"deltas": deltas})
			}
			w := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "QUERY ID\tCALLS\tEXEC MS\tROWS\tWAL\tQUERY")
			for _, d := range deltas {
				calls := fmt.Sprintf("%+d", d.Calls)
				if d.CallsReset {
					calls += "*"
				}
				fmt.Fprintf(w, "%d\t%s\t%+.1f\t%+d\t%+d\t%s\n", d.QueryID, calls, d.TotalExec, d.Rows, d.WALBytes, oneLine(d.Query))
			}
			if len(deltas) == 0 {
				fmt.Fprintln(w, "-\t-\t-\t-\t-\tno statements in the newer snapshot")
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if hasReset(deltas) {
				fmt.Fprintln(c.OutOrStdout(), "* counter reset between the snapshots")
			}
			return nil
		},
	}
	addOutputFlag(c, &output)
	return c
}

func hasReset(deltas []snapshot.Delta) bool {
	for _, d := range deltas {
		if d.CallsReset || d.ExecReset {
			return true
		}
	}
	return false
}
