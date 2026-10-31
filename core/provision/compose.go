// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/interlace-sh/tuskline/core/inventory"
)

// Published images for the non-production Compose stack.
const (
	etcdImage      = "quay.io/coreos/etcd:v3.5.17"
	haproxyImage   = "haproxy:3.0"
	pgbouncerImage = "edoburu/pgbouncer:v1.23.1-p3"
)

// Published Spilo tags, checked 2026-10-31. PostgreSQL 14 is the last
// release published on the Zalando registry; 15–18 are on GitHub's registry.
var spiloImages = map[int]string{
	14: "registry.opensource.zalan.do/acid/spilo-14:2.1-p7",
	15: "ghcr.io/zalando/spilo-15:3.2-p1",
	16: "ghcr.io/zalando/spilo-16:3.3-p3",
	17: "ghcr.io/zalando/spilo-17:4.0-p3",
	18: "ghcr.io/zalando/spilo-18:4.1-p2",
}

// ComposeSpec is a non-production Patroni stack on one Docker host.
type ComposeSpec struct {
	Name   string
	Major  int // PostgreSQL major, 14–18. Zero means 17.
	Nodes  int // 1, 2 or 3. Zero means 3.
	Etcd   int // 1 only when Nodes is 1; otherwise 3. Zero means 3.
	Pooler bool
	Sync   bool
	// Superuser and Replication are the generated role passwords. Empty
	// values are filled with random ones. They appear only in secret steps.
	Superuser   string
	Replication string
	// Env is the inventory environment for the hand-over. Empty means dev.
	// prod is refused: Compose is not a production target.
	Env string
	// Now is the clock for a new restore-drill date. Zero means the current time.
	Now time.Time
	// DrillDue keeps a drill that is already scheduled. Zero means seven days after Now.
	DrillDue time.Time
	// MonitorPassword is the tuskline_monitor password. Empty generates one.
	// It is written only to a secret file and the OS keychain.
	MonitorPassword string
}

// Compose builds a plan for compose.yaml, haproxy.cfg, pgBackRest config
// and a secret env file. The plan is not production, and nothing is started.
func Compose(spec ComposeSpec) (Plan, error) {
	plan, _, _, err := composePlan(spec, false)
	return plan, err
}

// ComposeStart is Compose plus a command that starts the stack and waits
// until the health checks pass. The command has no passwords in it.
func ComposeStart(spec ComposeSpec) (Plan, error) {
	plan, _, _, err := composePlan(spec, true)
	return plan, err
}

// HandedOver is a started stack plus the inventory entry and the monitor
// password. The password is not in the plan diff.
type HandedOver struct {
	Plan            Plan
	Cluster         inventory.Cluster
	MonitorPassword string
}

// Up is ComposeStart with the hand-over values the caller stores after Apply.
func Up(spec ComposeSpec) (HandedOver, error) {
	plan, cluster, password, err := composePlan(spec, true)
	if err != nil {
		return HandedOver{}, err
	}
	return HandedOver{Plan: plan, Cluster: cluster, MonitorPassword: password}, nil
}

func composePlan(spec ComposeSpec, start bool) (Plan, inventory.Cluster, string, error) {
	spec, err := normaliseCompose(spec)
	if err != nil {
		return Plan{}, inventory.Cluster{}, "", err
	}
	cluster, monitorPassword, sql, err := handover(spec)
	if err != nil {
		return Plan{}, inventory.Cluster{}, "", err
	}
	handoverYAML, err := handoverDocument(cluster)
	if err != nil {
		return Plan{}, inventory.Cluster{}, "", err
	}
	plan := Plan{
		Target:     "compose",
		Production: false,
		Steps: []Step{
			{ID: "compose", Kind: KindFile, Path: "compose.yaml", After: composeYAML(spec)},
			{ID: "haproxy", Kind: KindFile, Path: "haproxy.cfg", After: haproxyCfg(spec)},
			{ID: "pgbackrest", Kind: KindFile, Path: "pgbackrest.conf", After: pgBackRestConf(spec)},
			{ID: "env", Kind: KindFile, Path: ".env", Secret: true, After: composeEnv(spec)},
		},
	}
	if spec.Pooler {
		plan.Steps = append(plan.Steps, Step{
			ID: "pgbouncer-users", Kind: KindFile, Path: "pgbouncer/userlist.txt", Secret: true,
			After: fmt.Sprintf("\"postgres\" \"%s\"\n", spec.Superuser),
		})
	}
	plan.Steps = append(plan.Steps,
		Step{ID: "handover", Kind: KindFile, Path: "handover.yaml", After: handoverYAML},
		Step{ID: "monitor-sql", Kind: KindFile, Path: "monitor.sql", Secret: true, After: sql},
	)
	if start {
		plan.Steps = append(plan.Steps,
			Step{
				ID: "up", Kind: KindCommand, Path: "docker compose up",
				After: "docker compose --env-file .env up -d --wait --wait-timeout 600",
				Argv:  []string{"docker", "compose", "--env-file", ".env", "up", "-d", "--wait", "--wait-timeout", "600"},
				Rerun: true,
			},
			Step{
				ID: "monitor-role", Kind: KindCommand, Path: "tuskline_monitor",
				After:     "docker compose exec -T pg1 psql -U postgres -d postgres -v ON_ERROR_STOP=1",
				Argv:      []string{"docker", "compose", "exec", "-T", "pg1", "psql", "-U", "postgres", "-d", "postgres", "-v", "ON_ERROR_STOP=1"},
				StdinPath: "monitor.sql",
				Rerun:     true,
			},
		)
	}
	return plan, cluster, monitorPassword, nil
}

func prepareCompose(spec ComposeSpec) (ComposeSpec, error) {
	if err := inventory.ValidateName(spec.Name); err != nil {
		return spec, fmt.Errorf("cluster name: %w", err)
	}
	if spec.Major == 0 {
		spec.Major = 17
	}
	if spec.Major < 14 || spec.Major > 18 {
		return spec, fmt.Errorf("PostgreSQL %d is outside 14–18", spec.Major)
	}
	if spec.Nodes == 0 {
		spec.Nodes = 3
	}
	if spec.Nodes < 1 || spec.Nodes > 3 {
		return spec, fmt.Errorf("nodes must be 1, 2 or 3")
	}
	if spec.Etcd == 0 {
		if spec.Nodes == 1 {
			spec.Etcd = 1
		} else {
			spec.Etcd = 3
		}
	}
	if spec.Etcd != 1 && spec.Etcd != 3 {
		return spec, fmt.Errorf("etcd nodes must be 1 or 3")
	}
	if spec.Nodes > 1 && spec.Etcd != 3 {
		return spec, fmt.Errorf("more than one Postgres node needs a 3-member etcd")
	}
	return spec, nil
}

func normaliseCompose(spec ComposeSpec) (ComposeSpec, error) {
	spec, err := prepareCompose(spec)
	if err != nil {
		return spec, err
	}
	if spec.Superuser == "" {
		if spec.Superuser, err = randomSecret(); err != nil {
			return spec, err
		}
	}
	if spec.Replication == "" {
		if spec.Replication, err = randomSecret(); err != nil {
			return spec, err
		}
	}
	return spec, nil
}

func randomSecret() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func composeYAML(spec ComposeSpec) string {
	var b strings.Builder
	b.WriteString("# Generated by Tuskline. Non-production: one Docker host is a single point of failure.\n")
	b.WriteString("# For dev, CI and demos. Use CloudNativePG or VMs across hosts for real HA.\n")
	b.WriteString("# Nothing in this file is a secret. Passwords are in .env, which is not shown in the diff.\n")
	fmt.Fprintf(&b, "name: %s\n", spec.Name)
	b.WriteString("services:\n")
	hosts := make([]string, spec.Etcd)
	for i := 1; i <= spec.Etcd; i++ {
		hosts[i-1] = fmt.Sprintf("etcd%d=http://etcd%d:2380", i, i)
	}
	cluster := strings.Join(hosts, ",")
	for i := 1; i <= spec.Etcd; i++ {
		fmt.Fprintf(&b, `  etcd%d:
    image: %s
    environment:
      ETCD_NAME: etcd%d
      ETCDCTL_API: "3"
      ETCD_DATA_DIR: /etcd-data
      ETCD_INITIAL_CLUSTER_STATE: new
      ETCD_INITIAL_CLUSTER_TOKEN: %s
      ETCD_INITIAL_ADVERTISE_PEER_URLS: http://etcd%d:2380
      ETCD_LISTEN_PEER_URLS: http://0.0.0.0:2380
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
      ETCD_ADVERTISE_CLIENT_URLS: http://etcd%d:2379
      ETCD_INITIAL_CLUSTER: %s
    volumes:
      - etcd%d:/etcd-data
    command: [/usr/local/bin/etcd]
    healthcheck:
      test: ["CMD", "etcdctl", "endpoint", "health"]
      interval: 5s
      timeout: 5s
      retries: 20
      start_period: 5s
`, i, etcdImage, i, spec.Name, i, i, cluster, i)
	}
	var etcdClients []string
	for i := 1; i <= spec.Etcd; i++ {
		etcdClients = append(etcdClients, fmt.Sprintf(`"etcd%d:2379"`, i))
	}
	clientList := strings.Join(etcdClients, ",")
	for i := 1; i <= spec.Nodes; i++ {
		fmt.Fprintf(&b, `  pg%d:
    image: %s
    hostname: pg%d
    environment:
      SCOPE: %s
      PGVERSION: "%d"
      ETCD_HOSTS: '%s'
      ALLOW_NOSSL: "true"
      PGPASSWORD_SUPERUSER: ${PGPASSWORD_SUPERUSER}
      PGPASSWORD_STANDBY: ${PGPASSWORD_STANDBY}
`, i, spiloImages[spec.Major], i, spec.Name, spec.Major, clientList)
		if spec.Sync {
			b.WriteString("      # Synchronous replication was requested. Starting the stack confirms the image honours it.\n")
		}
		if i == 1 {
			b.WriteString("    ports:\n      - \"127.0.0.1:8008:8008\"\n")
		}
		fmt.Fprintf(&b, `    volumes:
      - pg%d:/home/postgres/pgdata
      - pgbackrest:/var/lib/pgbackrest
      - ./pgbackrest.conf:/etc/pgbackrest/pgbackrest.conf:ro
    healthcheck:
      test: ["CMD", "python3", "-c", "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8008/health')"]
      interval: 10s
      timeout: 5s
      retries: 30
      start_period: 40s
    depends_on:
`, i)
		for e := 1; e <= spec.Etcd; e++ {
			fmt.Fprintf(&b, "      etcd%d: { condition: service_healthy }\n", e)
		}
	}
	b.WriteString(`  haproxy:
    image: ` + haproxyImage + `
    ports:
      - "5000:5000"
      - "5001:5001"
    volumes:
      - ./haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro
    depends_on:
`)
	for i := 1; i <= spec.Nodes; i++ {
		fmt.Fprintf(&b, "      pg%d: { condition: service_healthy }\n", i)
	}
	if spec.Pooler {
		b.WriteString(`  pgbouncer:
    image: ` + pgbouncerImage + `
    environment:
      DB_HOST: haproxy
      DB_PORT: "5000"
      DB_USER: postgres
      DB_PASSWORD: ${PGPASSWORD_SUPERUSER}
      POOL_MODE: transaction
      MAX_CLIENT_CONN: "200"
    ports:
      - "6432:6432"
    volumes:
      - ./pgbouncer/userlist.txt:/etc/pgbouncer/userlist.txt:ro
    depends_on:
      - haproxy
`)
	}
	b.WriteString("volumes:\n")
	for i := 1; i <= spec.Etcd; i++ {
		fmt.Fprintf(&b, "  etcd%d:\n", i)
	}
	for i := 1; i <= spec.Nodes; i++ {
		fmt.Fprintf(&b, "  pg%d:\n", i)
	}
	b.WriteString("  pgbackrest:\n")
	return b.String()
}

func haproxyCfg(spec ComposeSpec) string {
	var b strings.Builder
	b.WriteString("# Non-production. 5000 is the leader, 5001 is the replicas.\n")
	b.WriteString("# Patroni answers /primary and /replica on port 8008.\n")
	b.WriteString("defaults\n  mode tcp\n  timeout connect 5s\n  timeout client 30m\n  timeout server 30m\n")
	writeBackend := func(name, port, check string) {
		fmt.Fprintf(&b, "frontend %s\n  bind *:%s\n  default_backend %s\nbackend %s\n  option httpchk\n  http-check send meth GET uri %s\n  http-check expect status 200\n", name, port, name, name, check)
		for i := 1; i <= spec.Nodes; i++ {
			fmt.Fprintf(&b, "  server pg%d pg%d:5432 check port 8008\n", i, i)
		}
	}
	writeBackend("rw", "5000", "/primary")
	writeBackend("ro", "5001", "/replica")
	return b.String()
}

func pgBackRestConf(spec ComposeSpec) string {
	return fmt.Sprintf(`# Local volume only. Not a production backup target.
[global]
repo1-path=/var/lib/pgbackrest
repo1-retention-full=2

[%s]
pg1-path=/home/postgres/pgdata/pgroot/data
pg1-port=5432
`, spec.Name)
}

func composeEnv(spec ComposeSpec) string {
	return fmt.Sprintf("PGPASSWORD_SUPERUSER=%s\nPGPASSWORD_STANDBY=%s\n", spec.Superuser, spec.Replication)
}
