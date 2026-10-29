// SPDX-License-Identifier: Apache-2.0

// Package inventory models what Tuskline knows about: organisation ›
// project › environment (with a policy) › cluster › instance › database.
// See docs/architecture.md, "Inventory and environment model".
package inventory

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// EnvKind classifies an environment. It drives colour and default policy.
type EnvKind string

const (
	EnvLocal   EnvKind = "local"
	EnvDev     EnvKind = "dev"
	EnvStaging EnvKind = "staging"
	EnvProd    EnvKind = "prod"
)

// Valid reports whether k is a known environment kind.
func (k EnvKind) Valid() bool {
	switch k {
	case EnvLocal, EnvDev, EnvStaging, EnvProd:
		return true
	}
	return false
}

// Policy is the safety policy attached to an environment.
type Policy struct {
	// ReadOnly makes every session default_transaction_read_only.
	ReadOnly bool
	// StatementTimeout and LockTimeout are applied to every session.
	StatementTimeout time.Duration
	LockTimeout      time.Duration
	// RowLimit caps result grids. Zero means no limit.
	RowLimit int
	// RequireTypedConfirmation gates destructive actions behind typing the
	// cluster name.
	RequireTypedConfirmation bool
	// MaxElevation bounds just-in-time write elevation. Zero disables it.
	MaxElevation time.Duration
}

// DefaultPolicy returns the policy a new environment of kind k gets.
// Production is read-only by default; this is a safety invariant and is
// covered by tests. Do not weaken it without an ADR.
func DefaultPolicy(k EnvKind) Policy {
	switch k {
	case EnvProd:
		return Policy{ReadOnly: true, StatementTimeout: 30 * time.Second, LockTimeout: 2 * time.Second, RowLimit: 1000, RequireTypedConfirmation: true, MaxElevation: 15 * time.Minute}
	case EnvStaging:
		return Policy{StatementTimeout: 60 * time.Second, LockTimeout: 5 * time.Second, RowLimit: 5000, RequireTypedConfirmation: true, MaxElevation: time.Hour}
	default:
		return Policy{RowLimit: 10000}
	}
}

// Environment groups clusters that share a policy.
type Environment struct {
	Name   string
	Kind   EnvKind
	Policy Policy
}

// Cluster is one PostgreSQL cluster (a primary plus any replicas).
type Cluster struct {
	ID          string
	Name        string
	Environment string
	Provider    string // self-managed, rds, aurora, cloudsql, azure, ...
	HAEngine    string // none, patroni, cnpg
	Tags        map[string]string
}

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// ErrInvalidName is returned for names that are not DNS-label-like.
var ErrInvalidName = errors.New("invalid name: use lowercase letters, digits and hyphens (max 63)")

// ValidateName checks an environment or cluster name.
func ValidateName(s string) error {
	if !nameRE.MatchString(s) {
		return fmt.Errorf("%q: %w", s, ErrInvalidName)
	}
	return nil
}

// Ref is the "env/cluster" form used by the CLI, e.g. prod/orders.
type Ref struct{ Env, Cluster string }

func (r Ref) String() string { return r.Env + "/" + r.Cluster }

// ParseRef parses "env/cluster".
func ParseRef(s string) (Ref, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			r := Ref{Env: s[:i], Cluster: s[i+1:]}
			if err := ValidateName(r.Env); err != nil {
				return Ref{}, err
			}
			if err := ValidateName(r.Cluster); err != nil {
				return Ref{}, err
			}
			return r, nil
		}
	}
	return Ref{}, fmt.Errorf("%q: want env/cluster, e.g. prod/orders", s)
}
