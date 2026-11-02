// SPDX-License-Identifier: Apache-2.0

package ha

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Member is one Postgres node in a topology.
type Member struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	State    string `json:"state,omitempty"`
	Timeline int    `json:"timeline,omitempty"`
	LagBytes int64  `json:"lag_bytes,omitempty"`
	Host     string `json:"host,omitempty"`
}

// Topology is a point-in-time view of a cluster. It is read-only.
type Topology struct {
	Source   string   `json:"source"`
	Timeline int      `json:"timeline,omitempty"`
	Members  []Member `json:"members"`
}

// Leader is the current primary, or the zero member.
func (t Topology) Leader() Member {
	for _, member := range t.Members {
		if member.Role == "leader" || member.Role == "primary" {
			return member
		}
	}
	return Member{}
}

// patroniCluster is the subset of GET /cluster this package reads.
type patroniCluster struct {
	Members []struct {
		Name     string          `json:"name"`
		Role     string          `json:"role"`
		State    string          `json:"state"`
		Timeline int             `json:"timeline"`
		Lag      json.RawMessage `json:"lag"`
		Host     string          `json:"host"`
	} `json:"members"`
}

// ParsePatroni reads a Patroni /cluster document.
func ParsePatroni(raw []byte) (Topology, error) {
	var doc patroniCluster
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Topology{}, fmt.Errorf("patroni topology: %w", err)
	}
	if len(doc.Members) == 0 {
		return Topology{}, fmt.Errorf("patroni topology: no members")
	}
	out := Topology{Source: "patroni"}
	for _, member := range doc.Members {
		if member.Name == "" {
			return Topology{}, fmt.Errorf("patroni topology: a member has no name")
		}
		role := member.Role
		if role == "primary" {
			role = "leader"
		}
		item := Member{Name: member.Name, Role: role, State: member.State, Timeline: member.Timeline, Host: member.Host, LagBytes: lagBytes(member.Lag)}
		out.Members = append(out.Members, item)
		if role == "leader" && item.Timeline > out.Timeline {
			out.Timeline = item.Timeline
		}
	}
	return out, nil
}

func lagBytes(raw json.RawMessage) int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	return 0
}

// cnpgStatus is the subset of a CloudNativePG cluster status this package reads.
type cnpgStatus struct {
	Phase     string `json:"phase"`
	Instances []struct {
		Name    string `json:"name"`
		Ready   bool   `json:"ready"`
		Primary bool   `json:"primary"`
		Role    string `json:"role"`
		Lag     string `json:"lag"`
	} `json:"instances"`
}

// ParseCNPG reads a CloudNativePG status document.
func ParseCNPG(raw []byte) (Topology, error) {
	var doc cnpgStatus
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Topology{}, fmt.Errorf("cnpg status: %w", err)
	}
	if len(doc.Instances) == 0 {
		return Topology{}, fmt.Errorf("cnpg status: no instances")
	}
	out := Topology{Source: "cnpg"}
	for _, instance := range doc.Instances {
		if instance.Name == "" {
			return Topology{}, fmt.Errorf("cnpg status: an instance has no name")
		}
		role := instance.Role
		if instance.Primary || role == "primary" {
			role = "leader"
		} else if role == "" {
			role = "replica"
		}
		state := "not ready"
		if instance.Ready {
			state = "ready"
		}
		out.Members = append(out.Members, Member{Name: instance.Name, Role: role, State: state})
	}
	if doc.Phase != "" {
		out.Members[0].State = strings.TrimSpace(doc.Phase)
	}
	return out, nil
}

// Text renders a topology for the CLI.
func (t Topology) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s", t.Source)
	if t.Timeline > 0 {
		fmt.Fprintf(&b, " timeline %d", t.Timeline)
	}
	b.WriteByte('\n')
	for _, member := range t.Members {
		fmt.Fprintf(&b, "%s %s", member.Name, member.Role)
		if member.State != "" {
			fmt.Fprintf(&b, " %s", member.State)
		}
		if member.LagBytes > 0 {
			fmt.Fprintf(&b, " lag %d bytes", member.LagBytes)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Switchover is a request to ask Patroni or CloudNativePG to move the leader.
type Switchover struct {
	Environment string
	Cluster     string
	From        string
	To          string
	// Prod requires Confirm to equal Cluster.
	Prod    bool
	Confirm string
	API     string
}

// Preview is the action that would be sent. It contains no password.
func (s Switchover) Preview() string {
	body := fmt.Sprintf(`{"leader":%q,"candidate":%q}`, s.From, s.To)
	return fmt.Sprintf("POST %s/switchover\n%s\nNothing has been sent.", strings.TrimRight(s.API, "/"), body)
}

// Check reports why the action must not be sent.
func (s Switchover) Check() error {
	if s.Cluster == "" || s.From == "" || s.To == "" {
		return fmt.Errorf("switchover needs a cluster, the current leader and a candidate")
	}
	if s.From == s.To {
		return fmt.Errorf("switchover candidate is already the leader")
	}
	if strings.ContainsAny(s.From, " \t\n") || strings.ContainsAny(s.To, " \t\n") {
		return fmt.Errorf("switchover names cannot contain spaces")
	}
	if s.API == "" || strings.Contains(s.API, " ") {
		return fmt.Errorf("switchover needs an API URL")
	}
	if s.Prod && s.Confirm != s.Cluster {
		return fmt.Errorf("type the cluster name to confirm the switchover")
	}
	return nil
}

// Body is the JSON sent to the API.
func (s Switchover) Body() string {
	return fmt.Sprintf(`{"leader":%q,"candidate":%q}`, s.From, s.To)
}

// Restart is a request to restart members through the HA API, replicas first.
type Restart struct {
	Environment string
	Cluster     string
	Member      string
	Prod        bool
	Confirm     string
	API         string
}

// Preview describes the restart. Replicas are named before the leader.
func (r Restart) Preview() string {
	target := r.Member
	if target == "" {
		target = "the leader"
	}
	return fmt.Sprintf("POST %s/restart\nreplicas first, then %s\nNothing has been sent.", strings.TrimRight(r.API, "/"), target)
}

// Check reports why the restart must not be sent.
func (r Restart) Check() error {
	if r.Cluster == "" || r.API == "" {
		return fmt.Errorf("restart needs a cluster and an API URL")
	}
	if r.Prod && r.Confirm != r.Cluster {
		return fmt.Errorf("type the cluster name to confirm the restart")
	}
	return nil
}

// RestartBody is the JSON sent to Patroni. The schedule is supplied by the caller.
func RestartBody(schedule time.Time) string {
	if schedule.IsZero() {
		schedule = time.Now().UTC()
	}
	return fmt.Sprintf(`{"schedule":%q,"role":"replica"}`, schedule.UTC().Format(time.RFC3339))
}
