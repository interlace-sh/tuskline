// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"context"
)

// ProbeSession is Probe inside a session with s applied, so probing a prod
// cluster runs under the prod policy like every other prod session.
func ProbeSession(ctx context.Context, p Profile, s SessionSettings) (ServerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()
	c, err := Connect(ctx, p, s)
	if err != nil {
		return ServerInfo{}, err
	}
	defer func() { _ = c.Close(context.Background()) }() // best effort; the probe result is already known
	return probe(ctx, c)
}
