// SPDX-License-Identifier: Apache-2.0

// Package agent is "tl agent": an optional collector that runs inside a
// customer network, reads OS metrics and local logs, and dials out to a
// Tuskline server or Tuskline Cloud over mTLS. Database credentials never
// leave it. Milestone M9 (Cloud) and M2 (collectors).
package agent

import (
	"context"
	"errors"
)

// Config configures the agent.
type Config struct {
	// ServerURL is the Tuskline server or Cloud endpoint to dial out to.
	ServerURL string
	// Token is the enrolment token. It is never logged.
	Token string
}

// ErrNotImplemented is returned until the agent lands (see docs/roadmap.md).
var ErrNotImplemented = errors.New("tl agent is not implemented yet (see docs/roadmap.md, M9)")

// Run starts the agent and blocks until ctx is cancelled.
func Run(ctx context.Context, cfg Config) error {
	if cfg.ServerURL == "" {
		return errors.New("agent: server URL is required")
	}
	_ = ctx
	return ErrNotImplemented
}
