// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/interlace-sh/tuskline/core/activity"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// sortKey orders the session table; `s` cycles through them.
type sortKey int

const (
	sortQueryAge sortKey = iota
	sortXactAge
	sortPID
	sortState
	sortKeys
)

func (k sortKey) String() string {
	return [...]string{"query age", "transaction age", "pid", "state"}[k]
}

// topView is everything one frame of `tl top` shows. render is pure, so
// --once and the TUI print the same thing.
type topView struct {
	ref        string
	kind       inventory.EnvKind
	versionNum int
	snap       activity.Snapshot
	err        error // the last refresh failed; the previous snapshot stays
	filter     string
	filtering  bool
	sort       sortKey
	// width and height bound the frame; zero means unbounded (--once).
	width, height int
	interactive   bool
}

// Section limits, so sessions always get most of the screen.
const (
	maxWaitsShown     = 6
	maxLockWaitsShown = 5
)

func (v topView) render(a *app) string {
	var lines []string
	add := func(s string) { lines = append(lines, s) }

	title := a.envColour(v.kind, v.ref)
	if v.kind != "" && string(v.kind) != strings.SplitN(v.ref, "/", 2)[0] {
		title += " (" + string(v.kind) + ")"
	}
	sessions := v.visible()
	active := 0
	for _, s := range v.snap.Sessions {
		if s.State == "active" {
			active++
		}
	}
	head := fmt.Sprintf("tl top %s · PostgreSQL %d.%d · %s · %d sessions, %d active · sort: %s",
		title, v.versionNum/10000, v.versionNum%10000, v.snap.At.Format("15:04:05 UTC"), len(v.snap.Sessions), active, v.sort)
	if v.filter != "" || v.filtering {
		head += fmt.Sprintf(" · filter: %s (%d shown)", v.filter, len(sessions))
		if v.filtering {
			head += "▏"
		}
	}
	add(head)
	if v.err != nil {
		add("refresh failed, showing the last snapshot: " + v.err.Error())
	}

	add("waits: " + waitSummary(v.snap.Waits))

	if tree := activity.BlockingTree(v.snap.Sessions); len(tree) > 0 {
		add("blocking:")
		var walk func(n *activity.Node, prefix string)
		walk = func(n *activity.Node, prefix string) {
			add(prefix + nodeLine(n))
			for _, c := range n.Blocked {
				walk(c, strings.Replace(prefix, "└─ ", "   ", 1)+"└─ ")
			}
		}
		for _, r := range tree {
			walk(r, "  ")
		}
	}

	var lockWaits []activity.Lock
	for _, l := range v.snap.Locks {
		if !l.Granted {
			lockWaits = append(lockWaits, l)
		}
	}
	if len(lockWaits) > 0 {
		add("lock waits:")
		for i, l := range lockWaits {
			if i == maxLockWaitsShown {
				add(fmt.Sprintf("  … %d more", len(lockWaits)-i))
				break
			}
			add(fmt.Sprintf("  %d waits %s %s%s for %s", l.PID, l.Mode, l.LockType, lockTarget(l), dur(l.WaitAge)))
		}
	}
	add("")

	table := sessionTable(sessions)
	footer := ""
	if v.interactive {
		footer = "q quit  / filter  s sort  esc clear filter · read-only"
	}
	if v.height > 0 {
		room := v.height - len(lines) - 1
		if footer != "" {
			room--
		}
		if room < 1 {
			room = 1
		}
		if len(table) > room+1 {
			hidden := len(table) - room
			table = append(table[:room], fmt.Sprintf("… %d more sessions", hidden))
		}
	}
	lines = append(lines, table...)
	if footer != "" {
		add(footer)
	}
	if v.width > 0 {
		for i, l := range lines {
			lines[i] = truncate(l, v.width)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// visible applies the filter and sort; idle sessions always come last.
func (v topView) visible() []activity.Session {
	f := strings.ToLower(v.filter)
	out := make([]activity.Session, 0, len(v.snap.Sessions))
	for _, s := range v.snap.Sessions {
		if f == "" || strings.Contains(strings.ToLower(strings.Join([]string{
			strconv.Itoa(int(s.PID)), s.User, s.Database, s.Application, s.State, s.WaitEventType, s.WaitEvent, s.Query,
		}, " ")), f) {
			out = append(out, s)
		}
	}
	slices.SortStableFunc(out, func(a, b activity.Session) int {
		if c := cmp.Compare(boolInt(a.State == "idle"), boolInt(b.State == "idle")); c != 0 {
			return c
		}
		switch v.sort {
		case sortXactAge:
			return cmp.Or(cmp.Compare(b.XactAge, a.XactAge), cmp.Compare(a.PID, b.PID))
		case sortPID:
			return cmp.Compare(a.PID, b.PID)
		case sortState:
			return cmp.Or(cmp.Compare(a.State, b.State), cmp.Compare(a.PID, b.PID))
		default:
			return cmp.Or(cmp.Compare(b.QueryAge, a.QueryAge), cmp.Compare(a.PID, b.PID))
		}
	})
	return out
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func sessionTable(ss []activity.Session) []string {
	header := []string{"PID", "USER", "DATABASE", "APP", "STATE", "WAIT", "XACT", "QUERY AGE", "BLOCKED BY", "QUERY"}
	rows := [][]string{header}
	for _, s := range ss {
		blocked := make([]string, len(s.BlockedBy))
		for i, b := range s.BlockedBy {
			blocked[i] = strconv.Itoa(int(b))
		}
		state := s.State
		if state == "" {
			state = s.BackendType
		}
		rows = append(rows, []string{
			strconv.Itoa(int(s.PID)), s.User, s.Database, s.Application, state, wait(s),
			dur(s.XactAge), dur(s.QueryAge), strings.Join(blocked, ","), oneLine(s.Query),
		})
	}
	widths := make([]int, len(header))
	for _, r := range rows {
		for i, c := range r[:len(r)-1] {
			widths[i] = max(widths[i], utf8.RuneCountInString(c))
		}
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		var b strings.Builder
		for j, c := range r {
			if j == len(r)-1 {
				b.WriteString(c)
				break
			}
			b.WriteString(c + strings.Repeat(" ", widths[j]-utf8.RuneCountInString(c)+2))
		}
		out[i] = strings.TrimRight(b.String(), " ")
	}
	return out
}

func wait(s activity.Session) string {
	if s.WaitEventType == "" {
		return ""
	}
	return s.WaitEventType + "/" + s.WaitEvent
}

func waitSummary(ws []activity.Wait) string {
	if len(ws) == 0 {
		return "none"
	}
	parts := make([]string, 0, maxWaitsShown)
	for i, w := range ws {
		if i == maxWaitsShown {
			parts = append(parts, fmt.Sprintf("+%d more", len(ws)-i))
			break
		}
		name := "not waiting"
		if w.Type != "" {
			name = w.Type + "/" + w.Event
		}
		parts = append(parts, fmt.Sprintf("%s %d", name, w.Sessions))
	}
	return strings.Join(parts, " · ")
}

func nodeLine(n *activity.Node) string {
	if n.Session == nil {
		if n.PID == 0 {
			return "0 (prepared transaction)"
		}
		return strconv.Itoa(int(n.PID)) + " (not visible)"
	}
	s := n.Session
	parts := []string{strconv.Itoa(int(s.PID)), s.User, s.State}
	if s.XactAge > 0 {
		parts = append(parts, "xact "+dur(s.XactAge))
	}
	if q := oneLine(s.Query); q != "" {
		parts = append(parts, q)
	}
	return strings.Join(slices.DeleteFunc(parts, func(p string) bool { return p == "" }), "  ")
}

func lockTarget(l activity.Lock) string {
	s := ""
	if l.Relation != "" {
		s += " on " + l.Relation
	}
	if l.Detail != "" {
		s += " (" + l.Detail + ")"
	}
	return s
}

// dur formats an age compactly: 0.4s, 12s, 3m04s, 2h03m, 3d04h.
func dur(d time.Duration) string {
	switch {
	case d <= 0:
		return ""
	case d < 10*time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd%02dh", int(d.Hours()/24), int(d.Hours())%24)
	}
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// truncate cuts s to width runes, skipping ANSI colour sequences.
func truncate(s string, width int) string {
	var b strings.Builder
	n := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := strings.IndexByte(s[i:], 'm')
			if j < 0 {
				break
			}
			b.WriteString(s[i : i+j+1])
			i += j + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if n == width-1 && utf8.RuneCountInString(stripANSI(s[i:])) > 1 {
			b.WriteRune('…')
			if strings.Contains(s, "\x1b[") {
				b.WriteString("\x1b[0m")
			}
			return b.String()
		}
		b.WriteRune(r)
		n++
		i += size
	}
	return b.String()
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			if j := strings.IndexByte(s[i:], 'm'); j >= 0 {
				i += j
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
