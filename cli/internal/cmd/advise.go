// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/advisor"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

func newConfigCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and advise on PostgreSQL configuration",
	}
	cmd.AddCommand(newAdviseCmd(a))
	return cmd
}

func newAdviseCmd(a *app) *cobra.Command {
	var (
		output      string
		memory      string
		cpus        int
		storage     string
		workload    string
		dbSize      string
		osName      string
		connections int
		pooler      bool
	)
	c := &cobra.Command{
		Use:   "advise <env/cluster>",
		Short: "Preview configuration changes",
		Long: `Compare the server's settings with a starting point and print only the
changes. Nothing is applied.

--memory, --cpus, --storage and --workload are the PgTune inputs. GB is
1024-based. Without --memory, memory settings are omitted rather than
guessed. --pooler records that a connection pooler is already in front
of the server, which changes the max_connections note.

See docs/adr/0014-config-advisor.md for which PgTune suggestions are
refused.`,
		Example: `  tl config advise dev/pg17
  tl config advise dev/pg17 --memory 16GB --cpus 8 --storage ssd --workload web`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if err := checkOutput(output); err != nil {
				return err
			}
			hw := advisor.Hardware{
				CPUs: cpus, Storage: storage, Workload: workload, DBSize: dbSize,
				OS: osName, Connections: connections, Pooler: pooler,
			}
			if memory != "" {
				n, err := advisor.ParseMemory(memory)
				if err != nil {
					return err
				}
				hw.Memory = n
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
			current, ev, err := advisor.Collect(c.Context(), db, version)
			if err != nil {
				return err
			}
			report, err := advisor.Advise(version, hw, current, ev)
			if err != nil {
				return err
			}
			if output == outputJSON {
				return writeJSON(c.OutOrStdout(), struct {
					Ref    string         `json:"ref"`
					Report advisor.Report `json:"report"`
				}{ref.String(), report})
			}
			_, err = fmt.Fprint(c.OutOrStdout(), report.Text(ref.String()))
			return err
		},
	}
	c.Flags().StringVarP(&output, "output", "o", outputText, "output format: text or json")
	c.Flags().StringVar(&memory, "memory", "", "RAM, for example 16GB (1024-based)")
	c.Flags().IntVar(&cpus, "cpus", 0, "CPUs PostgreSQL may use")
	c.Flags().StringVar(&storage, "storage", "", "storage: hdd, ssd, nvme or san")
	c.Flags().StringVar(&workload, "workload", "", "workload: web, oltp, dw, desktop or mixed")
	c.Flags().StringVar(&dbSize, "db-size", "", "database size against RAM: less, mid or greater")
	c.Flags().StringVar(&osName, "os", "", "operating system: linux, windows or mac")
	c.Flags().IntVar(&connections, "connections", 0, "connection count to size work_mem for")
	c.Flags().BoolVar(&pooler, "pooler", false, "a connection pooler is already in front of this server")
	return c
}
