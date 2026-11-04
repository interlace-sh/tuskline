// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/inventory"
)

func newRmCmd(a *app) *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "rm <env/cluster>",
		Short: "Remove a saved connection. Does not drop anything on the server.",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ref, err := inventory.ParseRef(args[0])
			if err != nil {
				return err
			}
			path, err := a.inventoryFile()
			if err != nil {
				return err
			}
			inv, err := inventory.Load(path)
			if err != nil {
				return err
			}
			cl, ok := inv.Cluster(ref)
			if !ok {
				return fmt.Errorf("cluster %s: %w", ref, inventory.ErrNotFound)
			}
			env, _ := inv.Environment(ref.Env)
			if env.Kind == inventory.EnvProd && yes {
				return fmt.Errorf("prod requires the cluster name; --yes is not accepted")
			}
			if !yes || env.Kind == inventory.EnvProd {
				fmt.Fprintf(c.OutOrStdout(), "Type %s to remove it: ", ref.Cluster)
				line, err := bufio.NewReader(c.InOrStdin()).ReadString('\n')
				if err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if strings.TrimSpace(line) != ref.Cluster {
					return fmt.Errorf("type the cluster name to confirm")
				}
			}
			var keychain string
			if cl.Password != nil {
				keychain = cl.Password.Keychain
			}
			if err := inv.RemoveCluster(ref); err != nil {
				return err
			}
			if err := inv.Save(path); err != nil {
				return err
			}
			if keychain != "" {
				_ = inventory.DeletePassword(keychain)
			}
			fmt.Fprintf(c.OutOrStdout(), "removed %s\n", ref)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "skip the prompt when the environment is not prod")
	return c
}
