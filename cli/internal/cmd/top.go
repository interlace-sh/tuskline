// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/activity"
	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// Refresh bounds for `tl top`.
const (
	minTopInterval = 500 * time.Millisecond
	// maxTopBackoff caps the delay after slow or failed refreshes.
	maxTopBackoff = 30 * time.Second
)

// topJSON is the --once --output json shape.
type topJSON struct {
	Ref        string            `json:"ref"`
	VersionNum int               `json:"server_version_num"`
	Snapshot   activity.Snapshot `json:"snapshot"`
	Blocking   []*activity.Node  `json:"blocking"`
}

func newTopCmd(a *app) *cobra.Command {
	var (
		once     bool
		interval time.Duration
		output   string
	)
	c := &cobra.Command{
		Use:   "top <env/cluster>",
		Short: "Live sessions, waits, locks and blocking chains (read-only TUI)",
		Long: `Show live sessions, wait events, lock waits and blocking chains,
refreshing every second. Keys: q quit, / filter, s sort, esc clear filter.
It has no actions; it runs read-only queries in a session with the
environment's policy, and slows down when the server is slow to answer.

--once prints one snapshot as text (or JSON with --output json) and exits,
for scripts and CI.`,
		Example: `  tl top prod/orders
  tl top dev/pg17 --once
  tl top dev/pg17 --once --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ref, err := inventory.ParseRef(args[0])
			if err != nil {
				return err
			}
			if err := checkOutput(output); err != nil {
				return err
			}
			if output == outputJSON && !once {
				return errors.New("--output json needs --once")
			}
			if interval < minTopInterval {
				return fmt.Errorf("--interval %s: the minimum is %s", interval, minTopInterval)
			}
			if !once && !a.stdoutTTY {
				return errors.New("tl top needs a terminal; use --once for a text snapshot")
			}
			sc, err := a.loadCluster(c, ref)
			if err != nil {
				return err
			}
			s := &topSource{profile: sc.profile, settings: sc.env.Policy.Session()}
			defer s.close()
			view := topView{ref: ref.String(), kind: sc.env.Kind}
			if once {
				return a.topOnce(c, s, view, output)
			}
			return a.topTUI(c, s, view, interval)
		},
	}
	c.Flags().BoolVar(&once, "once", false, "print one snapshot and exit")
	c.Flags().DurationVar(&interval, "interval", time.Second, "refresh interval (minimum 500ms)")
	addOutputFlag(c, &output)
	return c
}

// topSource keeps one connection and reconnects after it breaks.
type topSource struct {
	profile    conn.Profile
	settings   conn.SessionSettings
	c          *pgx.Conn
	versionNum int
}

func (s *topSource) take(ctx context.Context) (activity.Snapshot, error) {
	if s.c == nil || s.c.IsClosed() {
		c, err := conn.Connect(ctx, s.profile, s.settings)
		if err != nil {
			return activity.Snapshot{}, err
		}
		vn, err := conn.ServerVersionNum(c)
		if err != nil {
			_ = c.Close(context.Background()) // best effort; the version error is what matters
			return activity.Snapshot{}, err
		}
		s.c, s.versionNum = c, vn
	}
	return activity.Take(ctx, s.c, s.versionNum)
}

func (s *topSource) close() {
	if s.c != nil {
		_ = s.c.Close(context.Background()) // best effort on exit
	}
}

func (a *app) topOnce(c *cobra.Command, s *topSource, v topView, output string) error {
	snap, err := s.take(c.Context())
	if err != nil {
		return err
	}
	if output == outputJSON {
		if snap.Sessions == nil {
			snap.Sessions = []activity.Session{}
		}
		if snap.Locks == nil {
			snap.Locks = []activity.Lock{}
		}
		tree := activity.BlockingTree(snap.Sessions)
		if tree == nil {
			tree = []*activity.Node{}
		}
		return writeJSON(c.OutOrStdout(), topJSON{Ref: v.ref, VersionNum: s.versionNum, Snapshot: snap, Blocking: tree})
	}
	v.snap, v.versionNum = snap, s.versionNum
	_, err = fmt.Fprint(c.OutOrStdout(), v.render(a))
	return err
}

func (a *app) topTUI(c *cobra.Command, s *topSource, v topView, interval time.Duration) error {
	v.interactive = true
	m := &topModel{app: a, view: v, interval: interval, ctx: c.Context(), source: s}
	p := tea.NewProgram(m, tea.WithContext(c.Context()), tea.WithInput(c.InOrStdin()), tea.WithOutput(c.OutOrStdout()))
	_, err := p.Run()
	switch {
	case errors.Is(err, tea.ErrProgramPanic):
		return err
	case errors.Is(err, tea.ErrInterrupted), errors.Is(err, tea.ErrProgramKilled):
		return nil // ctrl+c or a cancelled context: a normal way out
	}
	return err
}

// snapMsg carries one refresh result.
type snapMsg struct {
	snap    activity.Snapshot
	version int
	err     error
	took    time.Duration
}

type tickMsg struct{}

// topModel is the Bubble Tea model. Refreshes never overlap: the next
// one is scheduled only after the previous result arrives.
type topModel struct {
	app      *app
	view     topView
	interval time.Duration
	ctx      context.Context
	source   interface {
		take(context.Context) (activity.Snapshot, error)
	}
	failures int
	loaded   bool
}

func (m *topModel) Init() tea.Cmd { return m.fetch }

func (m *topModel) fetch() tea.Msg {
	start := time.Now()
	snap, err := m.source.take(m.ctx)
	msg := snapMsg{snap: snap, err: err, took: time.Since(start)}
	if ts, ok := m.source.(*topSource); ok {
		msg.version = ts.versionNum
	}
	return msg
}

// nextDelay backs off when refreshes are slow (at least four times the
// last refresh) or failing (doubling per failure), up to maxTopBackoff.
func (m *topModel) nextDelay(took time.Duration) time.Duration {
	d := max(m.interval, 4*took)
	for i := 0; i < m.failures && d < maxTopBackoff; i++ {
		d *= 2
	}
	return min(d, maxTopBackoff)
}

func (m *topModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case snapMsg:
		if msg.err != nil {
			m.failures++
			m.view.err = msg.err
		} else {
			m.failures = 0
			m.view.err = nil
			m.view.snap = msg.snap
			m.loaded = true
			if msg.version != 0 {
				m.view.versionNum = msg.version
			}
		}
		return m, tea.Tick(m.nextDelay(msg.took), func(time.Time) tea.Msg { return tickMsg{} })
	case tickMsg:
		return m, m.fetch
	case tea.WindowSizeMsg:
		m.view.width, m.view.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		return m, m.key(msg)
	}
	return m, nil
}

func (m *topModel) key(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	if key == "ctrl+c" {
		return tea.Quit
	}
	if m.view.filtering {
		switch key {
		case "esc":
			m.view.filtering, m.view.filter = false, ""
		case "enter":
			m.view.filtering = false
		case "backspace":
			if r := []rune(m.view.filter); len(r) > 0 {
				m.view.filter = string(r[:len(r)-1])
			}
		default:
			if k.Text != "" {
				m.view.filter += k.Text
			}
		}
		return nil
	}
	switch key {
	case "q":
		return tea.Quit
	case "/":
		m.view.filtering = true
	case "s":
		m.view.sort = (m.view.sort + 1) % sortKeys
	case "esc":
		m.view.filter = ""
	}
	return nil
}

func (m *topModel) View() tea.View {
	var s string
	switch {
	case m.loaded:
		s = m.view.render(m.app)
	case m.view.err != nil:
		s = fmt.Sprintf("tl top %s: cannot read activity: %v\nretrying… (q to quit)\n", m.view.ref, m.view.err)
	default:
		s = fmt.Sprintf("tl top %s: connecting…\n", m.view.ref)
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
