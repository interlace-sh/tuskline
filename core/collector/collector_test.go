// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
	"testing"
	"time"
)

var errSyntax = errors.New("syntax")

// clock is a now function tests move forward from inside a query.
type clock struct{ t time.Time }

func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func (c *clock) now() time.Time { return c.t }

func query(name string, fn func(context.Context) error) Query {
	return Query{Name: name, Run: fn}
}

func TestRunPassStopsWhenTheBudgetIsSpent(t *testing.T) {
	t.Parallel()
	c := &clock{t: time.Unix(0, 0)}
	var called []string
	run := func(name string, cost time.Duration) Query {
		return query(name, func(context.Context) error {
			called = append(called, name)
			c.advance(cost)
			return nil
		})
	}
	res := RunPass(context.Background(), time.Second, []Query{
		run("a", time.Second),
		run("b", 0),
		run("c", 0),
	}, c.now)
	if !slices.Equal(res.Ran, []string{"a"}) || !slices.Equal(res.Skipped, []string{"b", "c"}) {
		t.Fatalf("ran %v skipped %v", res.Ran, res.Skipped)
	}
	if res.Used != time.Second || res.Overrun || res.Overloaded() || res.Err != nil {
		t.Fatalf("%+v", res)
	}
	if !slices.Equal(called, []string{"a"}) {
		t.Fatalf("called %v", called)
	}
}

func TestRunPassOverrunSkipsTheRest(t *testing.T) {
	t.Parallel()
	c := &clock{t: time.Unix(0, 0)}
	res := RunPass(context.Background(), time.Second, []Query{
		query("slow", func(context.Context) error { c.advance(2 * time.Second); return nil }),
		query("next", func(context.Context) error { t.Fatal("ran past an overrun"); return nil }),
	}, c.now)
	if !res.Overrun || !res.Overloaded() || len(res.Skipped) != 1 || res.Skipped[0] != "next" {
		t.Fatalf("%+v", res)
	}
}

func TestRunPassLoadSkipsTheRest(t *testing.T) {
	t.Parallel()
	res := RunPass(context.Background(), time.Minute, []Query{
		query("hot", func(context.Context) error { return ErrOverloaded }),
		query("next", func(context.Context) error { t.Fatal("ran against an overloaded target"); return nil }),
	}, nil)
	if !res.Load || res.Overrun || !errors.Is(res.Err, ErrOverloaded) || !slices.Equal(res.Skipped, []string{"next"}) {
		t.Fatalf("%+v", res)
	}
}

func TestRunPassKeepsGoingAfterAPlainError(t *testing.T) {
	t.Parallel()
	res := RunPass(context.Background(), time.Minute, []Query{
		query("bad", func(context.Context) error { return errSyntax }),
		query("ok", func(context.Context) error { return nil }),
	}, nil)
	if res.Overloaded() || !slices.Equal(res.Ran, []string{"bad", "ok"}) || res.Err == nil || !errors.Is(res.Err, errSyntax) {
		t.Fatalf("%+v", res)
	}
}

func TestRunPassTimeoutIsTheTimeLeft(t *testing.T) {
	t.Parallel()
	var got []time.Duration
	res := RunPass(context.Background(), time.Second, []Query{
		{Name: "short", Timeout: 50 * time.Millisecond, Run: func(ctx context.Context) error {
			d, ok := ctx.Deadline()
			if !ok {
				t.Fatal("no deadline")
			}
			got = append(got, time.Until(d))
			return nil
		}},
		{Name: "rest", Run: func(ctx context.Context) error {
			d, _ := ctx.Deadline()
			got = append(got, time.Until(d))
			return nil
		}},
	}, nil)
	if !slices.Equal(res.Ran, []string{"short", "rest"}) {
		t.Fatalf("%+v", res)
	}
	if got[0] <= 0 || got[0] > 50*time.Millisecond {
		t.Fatalf("query timeout %s", got[0])
	}
	if got[1] <= 50*time.Millisecond || got[1] > time.Second {
		t.Fatalf("remaining budget %s", got[1])
	}
}

func TestRunPassDeadlineIsLoad(t *testing.T) {
	t.Parallel()
	res := RunPass(context.Background(), time.Second, []Query{
		{Name: "stuck", Timeout: 20 * time.Millisecond, Run: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}},
		query("next", func(context.Context) error { t.Fatal("ran after a timeout"); return nil }),
	}, nil)
	if !res.Load || res.Stopped || !errors.Is(res.Err, context.DeadlineExceeded) || !slices.Equal(res.Skipped, []string{"next"}) {
		t.Fatalf("%+v err %v", res, res.Err)
	}
}

func TestRunPassParentCancelIsNotLoad(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := RunPass(ctx, time.Minute, []Query{
		query("a", func(context.Context) error { t.Fatal("ran after cancel"); return nil }),
	}, nil)
	if !res.Stopped || res.Load || res.Err != nil || !slices.Equal(res.Skipped, []string{"a"}) {
		t.Fatalf("%+v", res)
	}
}

func TestRunPassRejectsANilFunction(t *testing.T) {
	t.Parallel()
	res := RunPass(context.Background(), time.Second, []Query{{Name: "gap"}, {Name: "later"}}, nil)
	if res.Err == nil || !slices.Equal(res.Skipped, []string{"later"}) || len(res.Ran) != 0 {
		t.Fatalf("%+v", res)
	}
}

func TestSchedule(t *testing.T) {
	t.Parallel()
	s := Schedule{Base: time.Second, Current: time.Second, Max: 4 * time.Second}
	if d := s.Note(true); d != 2*time.Second {
		t.Fatalf("first back-off %s", d)
	}
	if d := s.Note(true); d != 4*time.Second {
		t.Fatalf("second back-off %s", d)
	}
	if d := s.Note(true); d != 4*time.Second {
		t.Fatalf("capped %s", d)
	}
	if d := s.Note(false); d != 2*time.Second {
		t.Fatalf("step down %s", d)
	}
	if d := s.Note(false); d != time.Second {
		t.Fatalf("back to base %s", d)
	}
	if d := s.Note(false); d != time.Second {
		t.Fatalf("stayed at base %s", d)
	}
}

func TestJitterStaysInRange(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(1, 2))
	for range 50 {
		got := Jitter(time.Second, 0.1, rng)
		if got < time.Second || got >= time.Second+100*time.Millisecond {
			t.Fatalf("%s", got)
		}
	}
	if Jitter(time.Second, 0, rng) != time.Second || Jitter(time.Second, 0.1, nil) != time.Second {
		t.Fatal("jitter was applied when it should not be")
	}
}

func TestRunnerBacksOffThenRecovers(t *testing.T) {
	t.Parallel()
	var passes int
	var waits []time.Duration
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := Runner{
		Rand:   rand.New(rand.NewPCG(1, 2)),
		OnPass: func(Result) { passes++ },
	}
	r.Sleep = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		if len(waits) == 3 {
			cancel()
			return ctx.Err()
		}
		return nil
	}
	n := 0
	err := r.Run(ctx, Spec{Interval: time.Second, Budget: time.Second, MaxDelay: 4 * time.Second}, []Query{
		query("q", func(context.Context) error {
			n++
			if n == 1 {
				return ErrOverloaded
			}
			return nil
		}),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v", err)
	}
	if passes != 3 {
		t.Fatalf("passes %d", passes)
	}
	// Overload doubles, a healthy pass halves back to the base, then the base again.
	want := []time.Duration{2 * time.Second, time.Second, time.Second}
	if !slices.Equal(waits, want) {
		t.Fatalf("waits %v", waits)
	}
}

func TestRunnerRejectsANegativeBudget(t *testing.T) {
	t.Parallel()
	err := Runner{}.Run(context.Background(), Spec{Budget: -time.Second}, nil)
	if err == nil {
		t.Fatal("no error")
	}
}

func TestDefaultSpec(t *testing.T) {
	t.Parallel()
	s := DefaultSpec()
	if s.Interval != 10*time.Second || s.Budget != 2*time.Second || s.Jitter != 0.1 || s.MaxDelay != 5*time.Minute {
		t.Fatalf("%+v", s)
	}
}
