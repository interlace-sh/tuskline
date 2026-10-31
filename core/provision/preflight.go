// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

const (
	// MinDockerServer is the oldest Engine release up accepts.
	MinDockerServer = "24.0.0"
	// MinCompose is the oldest Compose release that supports up --wait-timeout.
	MinCompose = "2.20.0"
)

// Need is what a Compose stack requires of the machine before it starts.
type Need struct {
	Name    string
	Ports   []int
	Volumes []string
	MinFree int64
}

// Report is what the machine looks like. The caller fills it; Evaluate does
// not probe Docker or the network.
type Report struct {
	FreeBytes     int64
	DockerServer  string
	DockerCompose string
	// DockerError is set when the engine or the Compose plugin could not be read.
	DockerError string
	// BusyPorts are host ports that did not accept a listen.
	BusyPorts map[int]bool
	// OwnPorts are host ports already published by this Compose project.
	OwnPorts map[int]bool
	// Volumes are Docker volume names that already exist.
	Volumes []string
	// Resume is true when this output directory has already been applied,
	// so its volumes and ports may already be held.
	Resume bool
}

// Requirements is the pre-flight for a Compose spec: published ports, the
// named volumes Compose will create, and a free-disk floor. The floor is
// 2 GiB per Postgres node plus 1 GiB, enough for the images and an empty
// data volume, not a capacity plan.
func Requirements(spec ComposeSpec) (Need, error) {
	spec, err := prepareCompose(spec)
	if err != nil {
		return Need{}, err
	}
	need := Need{
		Name:    spec.Name,
		Ports:   []int{5000, 5001, 8008},
		MinFree: int64(spec.Nodes*2+1) << 30,
	}
	if spec.Pooler {
		need.Ports = append(need.Ports, 6432)
	}
	for i := 1; i <= spec.Etcd; i++ {
		need.Volumes = append(need.Volumes, fmt.Sprintf("%s_etcd%d", spec.Name, i))
	}
	for i := 1; i <= spec.Nodes; i++ {
		need.Volumes = append(need.Volumes, fmt.Sprintf("%s_pg%d", spec.Name, i))
	}
	need.Volumes = append(need.Volumes, spec.Name+"_pgbackrest")
	return need, nil
}

// Evaluate reports every problem it finds. A resume may keep the project's
// own volumes and the ports those containers already publish.
func Evaluate(need Need, report Report) error {
	var problems []string
	switch {
	case report.DockerError != "":
		problems = append(problems, "docker: "+report.DockerError)
	case report.DockerServer == "" || report.DockerCompose == "":
		problems = append(problems, "Docker Compose is not available")
	default:
		if ok, err := atLeast(report.DockerServer, MinDockerServer); err != nil {
			problems = append(problems, fmt.Sprintf("Docker version %q: %v", report.DockerServer, err))
		} else if !ok {
			problems = append(problems, fmt.Sprintf("Docker %s is older than %s", report.DockerServer, MinDockerServer))
		}
		if ok, err := atLeast(report.DockerCompose, MinCompose); err != nil {
			problems = append(problems, fmt.Sprintf("Compose version %q: %v", report.DockerCompose, err))
		} else if !ok {
			problems = append(problems, fmt.Sprintf("Compose %s is older than %s", report.DockerCompose, MinCompose))
		}
	}
	if report.FreeBytes < need.MinFree {
		problems = append(problems, fmt.Sprintf("disk has %s free, need %s", fmtGiB(report.FreeBytes), fmtGiB(need.MinFree)))
	}
	for _, port := range need.Ports {
		if !report.BusyPorts[port] {
			continue
		}
		if report.Resume && report.OwnPorts[port] {
			continue
		}
		problems = append(problems, fmt.Sprintf("port %d is in use", port))
	}
	if !report.Resume {
		existing := map[string]bool{}
		for _, name := range report.Volumes {
			existing[name] = true
		}
		for _, name := range need.Volumes {
			if existing[name] {
				problems = append(problems, fmt.Sprintf("volume %s already exists", name))
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("pre-flight failed:\n- %s", strings.Join(problems, "\n- "))
}

// Summary is the one line printed when Evaluate succeeds.
func Summary(need Need, report Report) string {
	held := 0
	for _, port := range need.Ports {
		if report.BusyPorts[port] && report.OwnPorts[port] {
			held++
		}
	}
	ports := "ports free"
	if held > 0 && held == len(need.Ports) {
		ports = "ports already held by this project"
	}
	return fmt.Sprintf("Pre-flight passed. Docker %s, Compose %s, %s free, %s.", report.DockerServer, report.DockerCompose, fmtGiB(report.FreeBytes), ports)
}

// RefuseDataDirs fails before Apply when a plan step names a data directory
// that already has files. Compose keeps data in named volumes, so its plan
// has no such steps; the check is here for every target.
func RefuseDataDirs(ctx context.Context, plan Plan, fs FS) error {
	for _, step := range plan.Steps {
		if !step.DataDir {
			continue
		}
		if fs == nil {
			return fmt.Errorf("no filesystem")
		}
		nonEmpty, err := fs.NonEmpty(ctx, step.Path)
		if err != nil {
			return err
		}
		if nonEmpty {
			return fmt.Errorf("refuse non-empty data directory %s", step.Path)
		}
	}
	return nil
}

// PortFree reports whether a TCP listen on 0.0.0.0:port succeeds.
// A failed listen is treated as the port being in use, except when ctx is done.
func PortFree(ctx context.Context, port int) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, nil
	}
	return true, ln.Close()
}

var publishedPort = regexp.MustCompile(`:(\d+)->\d+/tcp`)

// PublishedPorts reads `docker ps` port columns and returns host ports.
func PublishedPorts(text string) map[int]bool {
	out := map[int]bool{}
	for _, match := range publishedPort.FindAllStringSubmatch(text, -1) {
		port, err := strconv.Atoi(match[1])
		if err == nil {
			out[port] = true
		}
	}
	return out
}

func atLeast(have, min string) (bool, error) {
	got, err := parseVersion(have)
	if err != nil {
		return false, err
	}
	want, err := parseVersion(min)
	if err != nil {
		return false, err
	}
	for i := range got {
		if got[i] > want[i] {
			return true, nil
		}
		if got[i] < want[i] {
			return false, nil
		}
	}
	return true, nil
}

func parseVersion(raw string) ([3]int, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")
	raw, _, _ = strings.Cut(raw, "-")
	parts := strings.Split(raw, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return [3]int{}, fmt.Errorf("not a version")
	}
	var out [3]int
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return [3]int{}, fmt.Errorf("not a version")
		}
		out[i] = n
	}
	return out, nil
}

func fmtGiB(n int64) string {
	const gib = 1 << 30
	if n < 0 {
		n = 0
	}
	whole := n / gib
	frac := (n % gib) * 10 / gib
	if whole >= 10 || frac == 0 {
		return fmt.Sprintf("%d GiB", whole)
	}
	return fmt.Sprintf("%d.%d GiB", whole, frac)
}
