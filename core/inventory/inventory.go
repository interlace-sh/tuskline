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

	"github.com/interlace-sh/tuskline/core/conn"
)

// EnvKind classifies an environment. It drives colour and default policy.
type EnvKind string

// The environment kinds, from least to most protected.
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
	ReadOnly bool `json:"read_only"`
	// StatementTimeout and LockTimeout are applied to every session.
	StatementTimeout time.Duration `json:"statement_timeout"`
	LockTimeout      time.Duration `json:"lock_timeout"`
	// RowLimit caps result grids. Zero means no limit.
	RowLimit int `json:"row_limit"`
	// RequireTypedConfirmation gates destructive actions behind typing the
	// cluster name.
	RequireTypedConfirmation bool `json:"require_typed_confirmation"`
	// MaxElevation bounds just-in-time write elevation. Zero disables it.
	MaxElevation time.Duration `json:"max_elevation"`
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

// Session returns the session defaults conn.Connect applies for p.
func (p Policy) Session() conn.SessionSettings {
	return conn.SessionSettings{ReadOnly: p.ReadOnly, StatementTimeout: p.StatementTimeout, LockTimeout: p.LockTimeout}
}

// Environment groups clusters that share a policy.
type Environment struct {
	Name string  `json:"name" yaml:"name"`
	Kind EnvKind `json:"kind" yaml:"kind"`
	// Policy is always DefaultPolicy(Kind). It is not read from or written
	// to the inventory file, so editing the file cannot weaken it.
	Policy Policy `json:"policy" yaml:"-"`
}

// Cluster is one PostgreSQL cluster (a primary plus any replicas).
type Cluster struct {
	Name        string            `json:"name" yaml:"name"`
	Environment string            `json:"environment" yaml:"environment"`
	Provider    string            `json:"provider,omitempty" yaml:"provider,omitempty"`   // community, rds, aurora, cloudsql, azure, ...
	HAEngine    string            `json:"ha_engine,omitempty" yaml:"ha_engine,omitempty"` // none, patroni, cnpg
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	// Profile says how to connect. It never holds a password.
	Profile conn.Profile `json:"profile" yaml:"profile"`
	// Password references the login password, if the server needs one and
	// ~/.pgpass does not provide it.
	Password *SecretRef `json:"password,omitempty" yaml:"password,omitempty"`
	// LastProbe is the latest successful probe, for `tl ls`.
	LastProbe *ProbeRecord `json:"last_probe,omitempty" yaml:"last_probe,omitempty"`
}

// Ref returns the cluster's env/cluster reference.
func (c Cluster) Ref() Ref { return Ref{Env: c.Environment, Cluster: c.Name} }

// ProbeRecord is a summary of the last successful probe.
type ProbeRecord struct {
	At         time.Time     `json:"at" yaml:"at"`
	VersionNum int           `json:"version_num" yaml:"version_num"`
	Provider   conn.Provider `json:"provider" yaml:"provider"`
	ReadOnly   bool          `json:"read_only" yaml:"read_only"`
	InRecovery bool          `json:"in_recovery" yaml:"in_recovery"`
}

// RecordProbe summarises info, taken at time at.
func RecordProbe(info conn.ServerInfo, at time.Time) *ProbeRecord {
	return &ProbeRecord{At: at.UTC(), VersionNum: info.VersionNum, Provider: info.Provider, ReadOnly: info.ReadOnly, InRecovery: info.InRecovery}
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
