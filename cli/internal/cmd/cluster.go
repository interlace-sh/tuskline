// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// savedCluster is an inventory entry ready to connect to.
type savedCluster struct {
	path    string
	inv     *inventory.Inventory
	cluster inventory.Cluster
	env     inventory.Environment
	// profile has the service resolved and the password filled in.
	profile conn.Profile
}

// loadCluster looks ref up in the inventory and resolves how to connect.
func (a *app) loadCluster(c *cobra.Command, ref inventory.Ref) (savedCluster, error) {
	path, err := a.inventoryFile()
	if err != nil {
		return savedCluster{}, err
	}
	inv, err := inventory.Load(path)
	if err != nil {
		return savedCluster{}, err
	}
	cl, ok := inv.Cluster(ref)
	if !ok {
		return savedCluster{}, fmt.Errorf("%s is not in the inventory (%s); add it with: tl connect --dsn <url> %s", ref, path, ref)
	}
	env, _ := inv.Environment(ref.Env)
	p, err := withPassword(c, cl.Profile, cl.Password)
	if err != nil {
		return savedCluster{}, err
	}
	if p, err = conn.ResolveService(conn.ServicefilePaths(), p); err != nil {
		return savedCluster{}, err
	}
	warnTLS(c, p)
	return savedCluster{path: path, inv: inv, cluster: cl, env: env, profile: p}, nil
}
