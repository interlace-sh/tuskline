// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/migrate"
)

func newMigrateCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Lint, preview and apply a folder of SQL migrations",
	}
	cmd.AddCommand(newMigrateLintCmd(), newMigratePlanCmd(), newMigrateApplyCmd(a), newMigrateDiffCmd(a))
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

func newMigrateApplyCmd(a *app) *cobra.Command {
	var dir, env, cluster, confirm string
	var approve, run bool
	c := &cobra.Command{
		Use:   "apply",
		Short: "Preview an apply. --run executes it on an inventory cluster.",
		Long: `Without --run, print the scripts in order. Production needs --approve.
Nothing is executed.

With --run, apply the scripts that tuskline.schema_migration has not recorded
on --cluster. Each script runs in one transaction. A script that cannot run
inside a transaction, such as CREATE INDEX CONCURRENTLY, runs on its own and
is recorded afterwards. An advisory lock stops a second apply on the same
database. Production needs --approve and --confirm equal to the cluster name.
The run is audited.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			folder, err := migrate.Load(dir)
			if err != nil {
				return err
			}
			if !run {
				text, err := folder.Apply(env, approve)
				if err != nil {
					return err
				}
				fmt.Fprint(c.OutOrStdout(), text)
				return nil
			}
			if cluster == "" {
				return errors.New("--run requires --cluster")
			}
			ref, err := inventory.ParseRef(cluster)
			if err != nil {
				return err
			}
			sc, err := a.loadCluster(c, ref)
			if err != nil {
				return err
			}
			if c.Flags().Changed("env") && inventory.EnvKind(env) != sc.env.Kind {
				return fmt.Errorf("--env %s does not match %s (%s)", env, ref.Env, sc.env.Kind)
			}
			if sc.env.Kind == inventory.EnvProd && !approve {
				return migrate.ErrApproval
			}
			if sc.env.Policy.RequireTypedConfirmation && confirm != sc.cluster.Name {
				return errors.New("type the cluster name to confirm")
			}
			db, err := conn.Connect(c.Context(), sc.profile, migrateSession(sc.env.Policy))
			if err != nil {
				return err
			}
			defer func() { _ = db.Close(context.Background()) }()
			if err := allowMigrationWrites(c.Context(), db); err != nil {
				return err
			}
			result, runErr := migrate.Run(c.Context(), db, folder)
			if runErr == nil || len(result.Applied) > 0 || len(result.Skipped) > 0 {
				fmt.Fprint(c.OutOrStdout(), result.String())
			}
			failure := ""
			if runErr != nil {
				failure = runErr.Error()
			}
			auditErr := recordAction(c.Context(), a, ref.Env, ref.Cluster, "migrate", folder.List(), failure)
			if runErr != nil && auditErr != nil {
				return fmt.Errorf("%w; audit: %w", runErr, auditErr)
			}
			if runErr != nil {
				return runErr
			}
			return auditErr
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "migration directory")
	c.Flags().StringVar(&env, "env", "dev", "environment kind for a preview: local, dev, staging or prod")
	c.Flags().StringVar(&cluster, "cluster", "", "inventory cluster to apply to, as env/name")
	c.Flags().StringVar(&confirm, "confirm", "", "cluster name, required when the environment asks for it")
	c.Flags().BoolVar(&approve, "approve", false, "approve a production apply")
	c.Flags().BoolVar(&run, "run", false, "apply the scripts on --cluster")
	_ = c.MarkFlagRequired("dir")
	return c
}

// migrateSession keeps the environment lock timeout and leaves the session
// writable. The production read-only default is not changed. This command
// turns read-only off itself, after approval, because a migration is a write.
func migrateSession(p inventory.Policy) conn.SessionSettings {
	return conn.SessionSettings{LockTimeout: p.LockTimeout}
}

func allowMigrationWrites(ctx context.Context, db *pgx.Conn) error {
	if _, err := db.Exec(ctx, `SELECT set_config('default_transaction_read_only', 'off', false)`); err != nil {
		return fmt.Errorf("allow migration writes: %w", err)
	}
	if _, err := db.Exec(ctx, `SELECT set_config('statement_timeout', '0', false)`); err != nil {
		return fmt.Errorf("clear migration statement timeout: %w", err)
	}
	return nil
}
