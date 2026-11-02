// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/interlace-sh/tuskline/core/catalog"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// SSHSpec is a Patroni cluster on virtual machines. The plan is files and
// package lines. Apply does not open a connection unless the caller runs
// the command steps.
type SSHSpec struct {
	Name     string
	Major    int
	Hosts    []string
	Autobase bool
}

var sshHost = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,253}$`)

// SSH builds the unit, Patroni and pgBackRest files and the PGDG package list.
func SSH(spec SSHSpec) (Plan, error) {
	return sshPlan(spec, false)
}

// SSHApply is SSH plus one ssh command per host. The remote command is the
// package list. It does not open a shell on the local machine.
func SSHApply(spec SSHSpec) (Plan, error) {
	return sshPlan(spec, true)
}

func sshPlan(spec SSHSpec, apply bool) (Plan, error) {
	spec, err := normaliseSSH(spec)
	if err != nil {
		return Plan{}, err
	}
	packages, err := packageText(spec)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		Target: "ssh",
		Steps: []Step{
			{ID: "patroni", Kind: KindFile, Path: "patroni.yml", After: sshPatroni(spec)},
			{ID: "unit", Kind: KindUnit, Path: "patroni.service", After: sshUnit(spec)},
			{ID: "pgbackrest", Kind: KindFile, Path: "pgbackrest.conf", After: sshBackrest(spec)},
			{ID: "packages", Kind: KindFile, Path: "packages.txt", After: packages},
		},
	}
	if spec.Autobase {
		plan.Steps = append(plan.Steps, Step{ID: "autobase", Kind: KindFile, Path: "inventory.yml", After: sshAutobase(spec)})
	}
	if apply {
		for _, host := range spec.Hosts {
			argv, err := SSHArgv(host, []string{"sudo", "bash", "packages.txt"})
			if err != nil {
				return Plan{}, err
			}
			plan.Steps = append(plan.Steps, Step{
				ID: "apply-" + hostID(host), Kind: KindCommand, Path: host,
				After: strings.Join(argv, " "), Argv: argv, Rerun: true,
			})
		}
	}
	return plan, nil
}

func normaliseSSH(spec SSHSpec) (SSHSpec, error) {
	if err := inventory.ValidateName(spec.Name); err != nil {
		return spec, fmt.Errorf("cluster name: %w", err)
	}
	if spec.Major == 0 {
		spec.Major = 17
	}
	if spec.Major < 14 || spec.Major > 18 {
		return spec, fmt.Errorf("PostgreSQL %d is outside 14–18", spec.Major)
	}
	if len(spec.Hosts) == 0 {
		return spec, fmt.Errorf("ssh target needs at least one host")
	}
	if len(spec.Hosts) > 5 {
		return spec, fmt.Errorf("ssh target accepts at most 5 hosts")
	}
	seen := map[string]bool{}
	for _, host := range spec.Hosts {
		if !sshHost.MatchString(host) || seen[host] {
			return spec, fmt.Errorf("host %q is not a single hostname or address", host)
		}
		seen[host] = true
	}
	return spec, nil
}

func packageText(spec SSHSpec) (string, error) {
	lines, err := catalog.InstallPlan(catalog.Request{
		Major: spec.Major, Provider: catalog.ProviderCommunity, Patroni: true, PreloadKnown: true,
		Want: []string{"pg_stat_statements", "auto_explain", "pg_wait_sampling", "pg_stat_kcache", "hypopg"},
	})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# Package preview. Nothing has been installed.\n")
	for _, line := range lines {
		b.WriteString(line.Text)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

// SSHArgv is the local argv that runs remote on host. Host is a single
// token, so the local command is not a shell.
func SSHArgv(host string, remote []string) ([]string, error) {
	if !sshHost.MatchString(host) {
		return nil, fmt.Errorf("host %q is not a single hostname or address", host)
	}
	if len(remote) == 0 {
		return nil, fmt.Errorf("ssh command is empty")
	}
	argv := []string{"ssh", "-o", "BatchMode=yes", host}
	return append(argv, remote...), nil
}

func hostID(host string) string {
	return strings.NewReplacer(".", "-", ":", "-").Replace(host)
}

func sshPatroni(spec SSHSpec) string {
	var hosts []string
	for _, host := range spec.Hosts {
		hosts = append(hosts, host+":2379")
	}
	return fmt.Sprintf(`scope: %s
namespace: /service/
etcd3:
  hosts: %s
bootstrap:
  dcs:
    postgresql:
      parameters:
        wal_level: replica
postgresql:
  data_dir: /var/lib/postgresql/%d/main
  bin_dir: /usr/lib/postgresql/%d/bin
`, spec.Name, strings.Join(hosts, ","), spec.Major, spec.Major)
}

func sshUnit(spec SSHSpec) string {
	return fmt.Sprintf(`[Unit]
Description=Patroni for %s
After=network-online.target

[Service]
Type=simple
User=postgres
ExecStart=/usr/bin/patroni /etc/patroni/patroni.yml
Restart=on-failure

[Install]
WantedBy=multi-user.target
`, spec.Name)
}

func sshBackrest(spec SSHSpec) string {
	return fmt.Sprintf(`# Local or object repository. No credentials are written here.
[global]
repo1-path=/var/lib/pgbackrest
repo1-retention-full=2

[%s]
pg1-path=/var/lib/postgresql/%d/main
`, spec.Name, spec.Major)
}

func sshAutobase(spec SSHSpec) string {
	var b strings.Builder
	b.WriteString("# Optional Autobase inventory. Tuskline does not run the playbook.\n")
	fmt.Fprintf(&b, "all:\n  vars:\n    patroni_cluster_name: %s\n    postgresql_version: %d\n  hosts:\n", spec.Name, spec.Major)
	for _, host := range spec.Hosts {
		fmt.Fprintf(&b, "    %s:\n", host)
	}
	return b.String()
}
