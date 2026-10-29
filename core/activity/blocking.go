// SPDX-License-Identifier: Apache-2.0

package activity

import (
	"slices"
)

// Node is a session in a blocking tree; Blocked are the sessions waiting
// on it.
type Node struct {
	PID     int32    `json:"pid"`
	Session *Session `json:"session,omitempty"` // nil when the blocker is not in the snapshot (for example a prepared transaction)
	Blocked []*Node  `json:"blocked,omitempty"`
}

// Size is the number of sessions in the tree, including the root.
func (n *Node) Size() int {
	total := 1
	for _, c := range n.Blocked {
		total += c.Size()
	}
	return total
}

// BlockingTree returns blocking chains: one tree per root blocker (a
// session that blocks others but is not itself blocked), largest first.
// A session blocked by several pids appears under each. Lock cycles that
// the deadlock detector has not broken yet have no root; each such cycle
// is reported from its lowest pid, and a pid is never repeated on one
// path, so the result is always finite.
func BlockingTree(sessions []Session) []*Node {
	byPID := make(map[int32]*Session, len(sessions))
	blocks := map[int32][]int32{} // blocker -> waiters
	for i := range sessions {
		s := &sessions[i]
		byPID[s.PID] = s
		for _, b := range s.BlockedBy {
			blocks[b] = append(blocks[b], s.PID)
		}
	}
	if len(blocks) == 0 {
		return nil
	}
	for _, w := range blocks {
		slices.Sort(w)
	}

	var build func(pid int32, path map[int32]bool) *Node
	build = func(pid int32, path map[int32]bool) *Node {
		n := &Node{PID: pid, Session: byPID[pid]}
		path[pid] = true
		for _, w := range blocks[pid] {
			if !path[w] {
				n.Blocked = append(n.Blocked, build(w, path))
			}
		}
		delete(path, pid)
		return n
	}

	var roots []*Node
	covered := map[int32]bool{}
	var mark func(n *Node)
	mark = func(n *Node) {
		covered[n.PID] = true
		for _, c := range n.Blocked {
			mark(c)
		}
	}
	blockers := make([]int32, 0, len(blocks))
	for b := range blocks {
		blockers = append(blockers, b)
	}
	slices.Sort(blockers)
	for _, b := range blockers {
		if s, ok := byPID[b]; ok && len(s.BlockedBy) > 0 {
			continue
		}
		r := build(b, map[int32]bool{})
		mark(r)
		roots = append(roots, r)
	}
	// Whatever is left is in a cycle.
	for _, b := range blockers {
		if !covered[b] {
			r := build(b, map[int32]bool{})
			mark(r)
			roots = append(roots, r)
		}
	}
	slices.SortStableFunc(roots, func(a, b *Node) int { return b.Size() - a.Size() })
	return roots
}
