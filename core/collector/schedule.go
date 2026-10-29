// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"errors"
	"math/rand/v2"
	"time"
)

// Defaults for a statistics pass. Activity sampling passes a shorter
// interval of its own; the budget stays small either way.
const (
	DefaultInterval = 10 * time.Second
	DefaultBudget   = 2 * time.Second
	// DefaultJitter is the fraction of the interval added at random, so
	// targets do not all start a pass together.
	DefaultJitter   = 0.1
	DefaultMaxDelay = 5 * time.Minute
)

// Spec is how often a target is collected and how much query time it may cost.
type Spec struct {
	// Interval is the gap after a healthy pass. Zero means DefaultInterval.
	Interval time.Duration
	// Budget is the query time one pass may spend. Zero means DefaultBudget.
	Budget time.Duration
	// Jitter is the fraction of the current delay added at random, in
	// [0, 1]. Zero means no jitter.
	Jitter float64
	// MaxDelay caps back-off. Zero means DefaultMaxDelay.
	MaxDelay time.Duration
}

// DefaultSpec is a statistics pass: every 10 s, at most 2 s of query time.
func DefaultSpec() Spec {
	return Spec{Interval: DefaultInterval, Budget: DefaultBudget, Jitter: DefaultJitter, MaxDelay: DefaultMaxDelay}
}

func (s Spec) resolve() (Spec, error) {
	if s.Interval == 0 {
		s.Interval = DefaultInterval
	}
	if s.Budget == 0 {
		s.Budget = DefaultBudget
	}
	if s.MaxDelay == 0 {
		s.MaxDelay = DefaultMaxDelay
	}
	if s.Interval < 0 || s.Budget < 0 || s.MaxDelay < 0 || s.Jitter < 0 {
		return Spec{}, errors.New("collector: interval, budget, max delay and jitter must not be negative")
	}
	if s.MaxDelay < s.Interval {
		s.MaxDelay = s.Interval
	}
	return s, nil
}

// Schedule is the delay before the next pass. It starts at Base and moves
// toward Max while the target is overloaded, then back toward Base.
type Schedule struct {
	Base    time.Duration
	Current time.Duration
	Max     time.Duration
}

// Note records one pass and returns the delay before the next one, before jitter.
func (s *Schedule) Note(overloaded bool) time.Duration {
	if s.Current < s.Base {
		s.Current = s.Base
	}
	if overloaded {
		s.Current *= 2
		if s.Max > 0 && s.Current > s.Max {
			s.Current = s.Max
		}
		return s.Current
	}
	if s.Current > s.Base {
		s.Current /= 2
		if s.Current < s.Base {
			s.Current = s.Base
		}
	}
	return s.Current
}

// Jitter returns d plus a random fraction of d in [0, fraction). A
// non-positive fraction or a nil rng returns d unchanged.
func Jitter(d time.Duration, fraction float64, rng *rand.Rand) time.Duration {
	if d <= 0 || fraction <= 0 || rng == nil {
		return d
	}
	extra := time.Duration(rng.Float64() * fraction * float64(d))
	return d + extra
}
