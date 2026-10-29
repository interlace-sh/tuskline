// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/analyser"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

func newDiagnoseCmd(a *app) *cobra.Command {
	var (
		output string
		lag    time.Duration
	)
	c := &cobra.Command{
		Use:   "diagnose <env/cluster>",
		Short: "Run health checks and print the findings",
		Long: `Connect with the environment's session policy and run the health checks:
missing pg_stat_statements, cache hit ratio, idle-in-transaction sessions,
wraparound age, unused and duplicate indexes, dead-tuple bloat, and
replication lag.

A check the login role cannot read is reported and skipped. The other
checks still run.`,
		Example: `  tl diagnose dev/pg17
  tl diagnose prod/orders --output markdown
  tl diagnose dev/pg17 --lag 30s --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			switch output {
			case outputText, outputJSON, "markdown":
			default:
				return fmt.Errorf("--output %q: want text, json or markdown", output)
			}
			if lag <= 0 {
				return fmt.Errorf("--lag must be greater than 0")
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
			report, err := analyser.Diagnose(c.Context(), db, version, lag)
			if err != nil {
				return err
			}
			taken := time.Now().UTC()
			if a.now != nil {
				taken = a.now().UTC()
			}
			switch output {
			case outputJSON:
				return writeJSON(c.OutOrStdout(), struct {
					Ref        string             `json:"ref"`
					Taken      time.Time          `json:"taken"`
					VersionNum int                `json:"server_version_num"`
					Findings   []analyser.Finding `json:"findings"`
				}{Ref: ref.String(), Taken: taken, VersionNum: report.VersionNum, Findings: report.Findings})
			case "markdown":
				_, err = fmt.Fprint(c.OutOrStdout(), report.Markdown(ref.String()))
			default:
				_, err = fmt.Fprint(c.OutOrStdout(), report.Text(ref.String()))
			}
			return err
		},
	}
	c.Flags().StringVarP(&output, "output", "o", outputText, "output format: text, json or markdown")
	c.Flags().DurationVar(&lag, "lag", time.Minute, "replication replay lag that counts as a finding")
	return c
}
