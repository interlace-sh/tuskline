// SPDX-License-Identifier: Apache-2.0

package conn

import (
	"errors"
	"testing"
	"time"
)

func TestAppliedSettingsCheck(t *testing.T) {
	t.Parallel()
	prod := SessionSettings{ReadOnly: true, StatementTimeout: 30 * time.Second, LockTimeout: 2 * time.Second}
	cases := []struct {
		name string
		got  appliedSettings
		want SessionSettings
		ok   bool
	}{
		{"prod applied", appliedSettings{"on", 30000, 2000}, prod, true},
		{"read-only lost", appliedSettings{"off", 30000, 2000}, prod, false},
		{"statement timeout lost", appliedSettings{"on", 0, 2000}, prod, false},
		{"lock timeout differs", appliedSettings{"on", 30000, 5000}, prod, false},
		{"dev imposes nothing", appliedSettings{"off", 0, 0}, SessionSettings{}, true},
		{"dev keeps a read-only role", appliedSettings{"on", 5000, 1000}, SessionSettings{}, true},
		{"sub-millisecond rounds up", appliedSettings{"off", 1, 0}, SessionSettings{StatementTimeout: time.Microsecond}, true},
	}
	for _, c := range cases {
		err := c.got.check(c.want)
		if (err == nil) != c.ok {
			t.Errorf("%s: check() = %v, want ok=%v", c.name, err, c.ok)
		}
		if err != nil && !errors.Is(err, ErrSessionNotApplied) {
			t.Errorf("%s: err = %v, want ErrSessionNotApplied", c.name, err)
		}
	}
}

func TestMillis(t *testing.T) {
	t.Parallel()
	for d, want := range map[time.Duration]string{30 * time.Second: "30000", 2 * time.Second: "2000", time.Microsecond: "1", 1500 * time.Microsecond: "1"} {
		if got := millis(d); got != want {
			t.Errorf("millis(%v) = %s, want %s", d, got, want)
		}
	}
}
