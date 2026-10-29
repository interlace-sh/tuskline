// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"
)

// Sleep waits for d or until ctx is cancelled. A non-positive d returns
// at once.
func Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Runner repeats a pass until ctx is cancelled. The first pass starts
// immediately; later passes wait for the schedule, plus jitter.
type Runner struct {
	// Now is read around each query. Nil means time.Now.
	Now func() time.Time
	// Sleep waits between passes. Nil means Sleep.
	Sleep func(ctx context.Context, d time.Duration) error
	// Rand supplies jitter. Nil means a new random source.
	Rand *rand.Rand
	// OnPass, when set, is called after every pass.
	OnPass func(Result)
}

// Run collects until ctx is cancelled. The first pass does not wait.
func (r Runner) Run(ctx context.Context, spec Spec, queries []Query) error {
	spec, err := spec.resolve()
	if err != nil {
		return err
	}
	sleep := r.Sleep
	if sleep == nil {
		sleep = Sleep
	}
	rng := r.Rand
	if rng == nil {
		rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	sched := Schedule{Base: spec.Interval, Current: spec.Interval, Max: spec.MaxDelay}
	for {
		res := RunPass(ctx, spec.Budget, queries, r.Now)
		if r.OnPass != nil {
			r.OnPass(res)
		}
		if res.Stopped || ctx.Err() != nil {
			return ctx.Err()
		}
		delay := Jitter(sched.Note(res.Overloaded()), spec.Jitter, rng)
		if err := sleep(ctx, delay); err != nil {
			return fmt.Errorf("collector: %w", err)
		}
	}
}
