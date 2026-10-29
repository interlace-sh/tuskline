// All data in this file is FAKE demo data for the Tuskline prototype.
// Hostnames use the reserved .example TLD; account IDs, ARNs and keys are placeholders.
import type { Env } from "./data";

export type DeployTarget = "compose" | "cnpg" | "vms";
export type DeployStatus = "succeeded" | "failed" | "running" | "rolled-back";
export interface Deployment {
  id: string; name: string; env: Env; target: DeployTarget; status: DeployStatus;
  pg: string; topology: string; startedAt: string; durationS: number; by: string; note?: string; log?: string[];
}
export const TARGET_LABEL: Record<DeployTarget, string> = { compose: "Docker Compose", cnpg: "Kubernetes · CloudNativePG", vms: "VMs via SSH" };

export const DEPLOYMENTS: Deployment[] = [
  { id: "dep-0019", name: "staging-k8s", env: "staging", target: "cnpg", status: "succeeded", pg: "17.2", topology: "1P + 2R · Pooler · Barman Cloud → GCS", startedAt: "2026-10-26 14:02", durationS: 412, by: "Mark Alford" },
  { id: "dep-0018", name: "dev-preview-pr-1423", env: "dev", target: "cnpg", status: "succeeded", pg: "18.0", topology: "1P · ephemeral (kind)", startedAt: "2026-10-25 09:41", durationS: 96, by: "ci-bot" },
  { id: "dep-0017", name: "perf-lab-eu", env: "staging", target: "vms", status: "failed", pg: "17.2", topology: "1P + 2R · etcd ×3 · HAProxy · vip", startedAt: "2026-10-23 17:15", durationS: 538, by: "Priya N. (demo)", note: "Pre-flight passed; etcd quorum timed out on pg-lab-3 (port 2380 blocked by host firewall)" },
  { id: "dep-0016", name: "local-docker", env: "dev", target: "compose", status: "succeeded", pg: "17.2", topology: "1P · single host", startedAt: "2026-10-20 11:03", durationS: 58, by: "Mark Alford" },
  { id: "dep-0015", name: "staging-eu-west", env: "staging", target: "vms", status: "rolled-back", pg: "17.0 → 17.2", topology: "minor upgrade · rolling", startedAt: "2026-10-14 22:30", durationS: 1290, by: "Jonas K. (demo)", note: "Replica pg-stg-2 failed health check after restart; automation rolled back and kept the old leader" },
  { id: "dep-0014", name: "dev-shared", env: "dev", target: "compose", status: "succeeded", pg: "17.2", topology: "1P + 1R · PgBouncer", startedAt: "2026-10-02 10:12", durationS: 141, by: "Mark Alford" },
];

export interface Ext { name: string; purpose: string; preload: boolean; restart: boolean; rds: "ok" | "missing" | "na"; self: "ok" | "missing" | "preloaded" }
export const EXTENSIONS: Ext[] = [
  { name: "pg_stat_statements", purpose: "Per-query latency, calls and I/O — powers Query insights", preload: true, restart: true, rds: "ok", self: "preloaded" },
  { name: "auto_explain", purpose: "Logs plans of slow statements for plan-regression detection", preload: true, restart: true, rds: "missing", self: "missing" },
  { name: "pg_wait_sampling", purpose: "High-frequency wait-event sampling for the DB-load chart", preload: true, restart: true, rds: "na", self: "missing" },
  { name: "pg_stat_kcache", purpose: "Real CPU and disk reads per query (needs pg_stat_statements)", preload: true, restart: true, rds: "na", self: "missing" },
  { name: "hypopg", purpose: "Hypothetical indexes to validate index advice before building", preload: false, restart: false, rds: "missing", self: "missing" },
];

export interface ConfFinding {
  id: string; cat: "Memory" | "WAL" | "Checkpoints" | "Autovacuum" | "Connections";
  param: string; current: string; rec: string; apply: "restart" | "reload"; sev: "crit" | "warn" | "info";
  why: string; evidence: string;
}
// Host facts for the advisor demo: 8 vCPU, 32 GB RAM, NVMe SSD, OLTP profile.
export const HOST = { vcpu: 8, ramGb: 32, storage: "NVMe SSD (gp3, 6,000 IOPS)", profile: "OLTP" };
export const CONF_FINDINGS: ConfFinding[] = [
  { id: "c1", cat: "Memory", param: "shared_buffers", current: "128MB", rec: "8GB", apply: "restart", sev: "crit", why: "The default 128MB is sized for a laptop. ~25% of RAM is the usual starting point on a dedicated host; the OS page cache covers the rest.", evidence: "Buffer cache hit ratio 91.4% on orders (demo) · 32 GB RAM detected" },
  { id: "c2", cat: "Memory", param: "effective_cache_size", current: "4GB", rec: "24GB", apply: "reload", sev: "warn", why: "Planner hint only (allocates nothing). Set to roughly the memory available for caching so index scans are costed realistically.", evidence: "~75% of 32 GB" },
  { id: "c3", cat: "Memory", param: "work_mem", current: "4MB", rec: "16MB", apply: "reload", sev: "info", why: "Per sort/hash node, per query, so raise cautiously. Raised modestly because temp-file spills are frequent but max_connections is high.", evidence: "1,284 temp files / hour, median 9 MB (demo)" },
  { id: "c4", cat: "WAL", param: "max_wal_size", current: "1GB", rec: "8GB", apply: "reload", sev: "warn", why: "WAL volume forces checkpoints before checkpoint_timeout, causing write spikes and extra full-page images.", evidence: "71% of checkpoints were requested, not timed (demo)" },
  { id: "c5", cat: "WAL", param: "wal_compression", current: "off", rec: "lz4", apply: "reload", sev: "info", why: "Compresses full-page images; cuts WAL volume and replica bandwidth at small CPU cost.", evidence: "Full-page images ≈ 38% of WAL bytes (demo)" },
  { id: "c6", cat: "Checkpoints", param: "checkpoint_completion_target", current: "0.5", rec: "0.9", apply: "reload", sev: "warn", why: "Spreads checkpoint writes across the interval instead of bursting. 0.9 has been the default since PostgreSQL 14, so 0.5 was set explicitly.", evidence: "Set in postgresql.conf line 231" },
  { id: "c7", cat: "Memory", param: "random_page_cost", current: "4", rec: "1.1", apply: "reload", sev: "warn", why: "4 models spinning disks. On SSD/NVMe, random reads cost close to sequential ones; a lower value lets the planner pick index scans.", evidence: "Storage class: NVMe SSD" },
  { id: "c12", cat: "Checkpoints", param: "checkpoint_timeout", current: "5min", rec: "15min", apply: "reload", sev: "info", why: "Longer intervals mean fewer full-page images and less write amplification, at the cost of longer crash recovery.", evidence: "Recovery-time target in profile: < 5 min (demo)" },
  { id: "c8", cat: "Autovacuum", param: "autovacuum_vacuum_scale_factor", current: "0.2", rec: "0.05", apply: "reload", sev: "warn", why: "20% of a large table must be dead before vacuum starts; on big, hot tables that is millions of dead rows.", evidence: "order_events: 41 M dead tuples before last vacuum (demo)" },
  { id: "c9", cat: "Autovacuum", param: "autovacuum_vacuum_cost_limit", current: "200", rec: "2000", apply: "reload", sev: "warn", why: "The default throttle is conservative for SSDs; vacuum falls behind write rate.", evidence: "Average vacuum on order_events: 3 h 12 m (demo)" },
  { id: "c10", cat: "Connections", param: "max_connections", current: "1000", rec: "300", apply: "restart", sev: "warn", why: "Each backend costs memory and contention; peak use is far lower. Put PgBouncer in front rather than raising this.", evidence: "Peak 212 connections in 30 days · no pooler detected (demo)" },
  { id: "c11", cat: "Connections", param: "idle_in_transaction_session_timeout", current: "0", rec: "60s", apply: "reload", sev: "crit", why: "Idle-in-transaction sessions hold locks and block vacuum. A timeout limits the damage from leaked transactions.", evidence: "pid 48211 idle in transaction for 4 m 12 s (demo)" },
];
export const CATEGORIES: ConfFinding["cat"][] = ["Memory", "WAL", "Checkpoints", "Autovacuum", "Connections"];

// ---------- Connection wizard generators ----------
export type Provider = "rds" | "cloudsql" | "azure" | "self";
export const PROVIDER_LABEL: Record<Provider, string> = { rds: "Amazon RDS for PostgreSQL", cloudsql: "Google Cloud SQL", azure: "Azure Database for PostgreSQL – Flexible Server", self: "Self-managed (Patroni detected)" };

export function preloadSql(exts: string[], provider: Provider) {
  const preload = EXTENSIONS.filter((e) => e.preload && exts.includes(e.name)).map((e) => e.name);
  const head = provider === "self"
    ? `-- 1. Merge into shared_preload_libraries (keeps anything already listed).\n--    Patroni manages config here, so change it with patronictl, not ALTER SYSTEM:\n--    $ patronictl -c /etc/patroni/patroni.yml edit-config \\\n--        -s 'postgresql.parameters.shared_preload_libraries=${["pg_stat_statements", ...preload.filter((x) => x !== "pg_stat_statements")].join(",")}'\n--    then: patronictl restart <cluster> --pending\n`
    : `-- 1. shared_preload_libraries cannot be set with SQL on a managed service.\n--    Use the parameter-group / flags command in the next tab, then reboot.\n`;
  const creates = exts.map((e) => `CREATE EXTENSION IF NOT EXISTS ${e};`).filter((l) => !l.includes("auto_explain"));
  return `${head}\n-- 2. After the restart, in each database you want to monitor:\n${creates.join("\n")}${exts.includes("auto_explain") ? "\n-- auto_explain is a module only: no CREATE EXTENSION needed." : ""}\n\n-- 3. Verify\nSELECT name, setting FROM pg_settings WHERE name = 'shared_preload_libraries';\nSELECT extname, extversion FROM pg_extension ORDER BY 1;`;
}
export function confPatch(exts: string[]) {
  const preload = EXTENSIONS.filter((e) => e.preload && exts.includes(e.name)).map((e) => e.name);
  const lib = ["pg_stat_statements", ...preload.filter((x) => x !== "pg_stat_statements")];
  return `# postgresql.conf (or postgresql.auto.conf via ALTER SYSTEM on self-managed hosts)\nshared_preload_libraries = '${lib.join(",")}'   # restart required\npg_stat_statements.track = top\npg_stat_statements.max = 10000\ntrack_io_timing = on${exts.includes("auto_explain") ? "\nauto_explain.log_min_duration = '500ms'\nauto_explain.log_format = json\nauto_explain.sample_rate = 0.1" : ""}${exts.includes("pg_wait_sampling") ? "\npg_wait_sampling.profile_period = 10" : ""}`;
}
export function restartSteps(provider: Provider) {
  if (provider === "self") return `# Rolling restart, replicas first (Patroni)\n$ patronictl -c /etc/patroni/patroni.yml list\n$ patronictl restart demo-cluster pg-demo-2 --pending\n$ patronictl restart demo-cluster pg-demo-3 --pending\n$ patronictl switchover demo-cluster --leader pg-demo-1 --candidate pg-demo-2\n$ patronictl restart demo-cluster pg-demo-1 --pending\n\n# Without Patroni\n$ sudo systemctl restart postgresql@17-main`;
  if (provider === "rds") return `# Parameters with ApplyMethod=pending-reboot apply at the next reboot.\n# Multi-AZ: reboot with failover to keep the outage to the failover time.\n$ aws rds reboot-db-instance --db-instance-identifier demo-orders-db --force-failover`;
  if (provider === "cloudsql") return `# Cloud SQL restarts the instance automatically when flags that need it change.\n# Schedule it inside the maintenance window.`;
  return `$ az postgres flexible-server restart --resource-group demo-rg --name demo-orders-db`;
}
export function cloudCommand(provider: Provider, exts: string[]) {
  const preload = EXTENSIONS.filter((e) => e.preload && exts.includes(e.name) && e.rds !== "na").map((e) => e.name);
  const lib = ["pg_stat_statements", ...preload.filter((x) => x !== "pg_stat_statements")].join(",");
  if (provider === "rds") return `$ aws rds modify-db-parameter-group \\\n    --db-parameter-group-name demo-orders-pg17 \\\n    --parameters \\\n      "ParameterName=shared_preload_libraries,ParameterValue='${lib}',ApplyMethod=pending-reboot" \\\n      "ParameterName=track_io_timing,ParameterValue=1,ApplyMethod=immediate"\n\n# Custom parameter group required: the default group cannot be modified.`;
  if (provider === "cloudsql") return `# WARNING: --database-flags replaces ALL existing flags. Include current ones.\n$ gcloud sql instances patch demo-orders-db \\\n    --database-flags=track_io_timing=on\n\n# pg_stat_statements is then enabled per database with CREATE EXTENSION.`;
  if (provider === "azure") return `$ az postgres flexible-server parameter set --resource-group demo-rg \\\n    --server-name demo-orders-db --name shared_preload_libraries --value '${lib}'\n$ az postgres flexible-server parameter set --resource-group demo-rg \\\n    --server-name demo-orders-db --name azure.extensions --value '${exts.join(",").toUpperCase()}'`;
  return "# Not a managed service — use the postgresql.conf tab.";
}
export function roleSql(o: { role: string; db: string; iam: boolean; operator: boolean; provider: Provider }) {
  return `-- Least-privilege monitoring role for Tuskline (read-only)\nCREATE ROLE ${o.role} WITH LOGIN${o.iam ? "" : " PASSWORD :'tuskline_password'"} CONNECTION LIMIT 5;\nGRANT pg_monitor TO ${o.role};            -- pg_stat_*, pg_settings, wait events\nGRANT CONNECT ON DATABASE ${o.db} TO ${o.role};\n${o.iam && o.provider === "rds" ? `GRANT rds_iam TO ${o.role};             -- IAM token auth, no stored password\n` : ""}\nALTER ROLE ${o.role} SET default_transaction_read_only = on;\nALTER ROLE ${o.role} SET statement_timeout = '5s';\nALTER ROLE ${o.role} SET lock_timeout = '1s';\nALTER ROLE ${o.role} SET idle_in_transaction_session_timeout = '10s';\n${o.operator ? `\n-- Optional, separate operator role: can cancel/terminate backends, nothing else.\n-- Tuskline only uses it after elevation + typed confirmation.\nCREATE ROLE tuskline_operator WITH LOGIN${o.iam ? "" : " PASSWORD :'operator_password'"} CONNECTION LIMIT 2;\nGRANT pg_monitor, pg_signal_backend TO tuskline_operator;\n` : ""}`;
}

// ---------- Deploy wizard ----------
export interface DeployCfg {
  name: string; env: Env; target: DeployTarget; replicas: number; etcd: number; pgbouncer: boolean;
  repo: "volume" | "s3" | "gcs" | "azure"; entry: "haproxy" | "vip" | "service"; sync: boolean;
  size: "dev" | "small" | "medium" | "large"; pg: string; exts: string[];
  fullCron: string; diffCron: string; retentionDays: number; hosts: string;
}
export const SIZES = {
  dev: { label: "Dev", cpu: 2, ram: 4, disk: 20, sb: "1GB" },
  small: { label: "Small", cpu: 4, ram: 16, disk: 100, sb: "4GB" },
  medium: { label: "Medium", cpu: 8, ram: 32, disk: 500, sb: "8GB" },
  large: { label: "Large", cpu: 16, ram: 64, disk: 2000, sb: "16GB" },
} as const;
export const DEPLOY_EXTS = ["pg_stat_statements", "auto_explain", "pg_wait_sampling", "pg_stat_kcache", "hypopg", "pgvector", "postgis", "pg_partman"];

const img = (n: string) => `ghcr.io/tuskline/${n}`; // proposed image names — not published
export function genCompose(c: DeployCfg) {
  const s = SIZES[c.size]; const nodes = 1 + c.replicas;
  const etcdSvc = Array.from({ length: c.etcd }, (_, i) => `  etcd${i + 1}:\n    image: quay.io/coreos/etcd:v3.5.17\n    command: >\n      etcd --name etcd${i + 1} --initial-advertise-peer-urls http://etcd${i + 1}:2380\n      --listen-peer-urls http://0.0.0.0:2380 --listen-client-urls http://0.0.0.0:2379\n      --advertise-client-urls http://etcd${i + 1}:2379\n      --initial-cluster ${Array.from({ length: c.etcd }, (_, j) => `etcd${j + 1}=http://etcd${j + 1}:2380`).join(",")}\n    volumes: [etcd${i + 1}:/etcd-data]`).join("\n");
  const pg = Array.from({ length: nodes }, (_, i) => `  pg${i + 1}:\n    image: ${img(`postgres-patroni:${c.pg}`)}   # proposed\n    hostname: pg${i + 1}\n    environment:\n      PATRONI_NAME: pg${i + 1}\n      PATRONI_SCOPE: ${c.name}\n      PATRONI_ETCD3_HOSTS: ${Array.from({ length: c.etcd }, (_, j) => `etcd${j + 1}:2379`).join(",")}\n      PATRONI_POSTGRESQL_PARAMETERS_SHARED_BUFFERS: ${s.sb}\n      PATRONI_POSTGRESQL_PARAMETERS_SHARED_PRELOAD_LIBRARIES: ${c.exts.filter((e) => ["pg_stat_statements", "auto_explain", "pg_wait_sampling", "pg_stat_kcache"].includes(e)).join(",")}\n${c.sync ? "      PATRONI_SYNCHRONOUS_MODE: \"true\"\n" : ""}    deploy: { resources: { limits: { cpus: "${s.cpu}", memory: ${s.ram}G } } }\n    volumes: [pg${i + 1}:/var/lib/postgresql/data, pgbackrest-repo:/var/lib/pgbackrest]\n    depends_on: [${Array.from({ length: c.etcd }, (_, j) => `etcd${j + 1}`).join(", ")}]`).join("\n");
  return `# compose.yaml — generated by Tuskline (preview, demo)\n# NOTE: one Docker host is a single point of failure. Compose is for dev,\n# demos and small installs; use CloudNativePG or VMs for real HA.\nname: ${c.name}\nservices:\n${etcdSvc}\n${pg}\n${c.entry === "haproxy" ? `  haproxy:\n    image: haproxy:3.0\n    ports: ["5000:5000", "5001:5001"]   # 5000 → leader (rw), 5001 → replicas (ro)\n    volumes: [./haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro]\n` : ""}${c.pgbouncer ? `  pgbouncer:\n    image: ${img("pgbouncer:1.23")}   # proposed\n    environment: { DB_HOST: haproxy, DB_PORT: "5000", POOL_MODE: transaction, MAX_CLIENT_CONN: "2000" }\n    ports: ["6432:6432"]\n` : ""}  tuskline:\n    image: ${img("tuskline:latest")}   # proposed\n    ports: ["7432:7432"]\n    environment: { TL_TARGET: "postgres://tuskline_monitor@haproxy:5000/postgres" }\nvolumes:\n${[...Array.from({ length: c.etcd }, (_, i) => `etcd${i + 1}`), ...Array.from({ length: nodes }, (_, i) => `pg${i + 1}`), "pgbackrest-repo"].map((v) => `  ${v}: {}`).join("\n")}\n# Backups: pgbackrest full ${c.fullCron} · diff ${c.diffCron} · retain ${c.retentionDays} d → ${c.repo === "volume" ? "local volume" : c.repo.toUpperCase()}`;
}
export function genCnpg(c: DeployCfg) {
  const s = SIZES[c.size];
  const store = c.repo === "s3" ? "s3://demo-backups/" + c.name : c.repo === "gcs" ? "gs://demo-backups/" + c.name : c.repo === "azure" ? "https://demoacct.blob.core.windows.net/backups/" + c.name : "s3://demo-minio/" + c.name;
  return `# CloudNativePG manifests — generated by Tuskline (preview, demo)\napiVersion: postgresql.cnpg.io/v1\nkind: Cluster\nmetadata:\n  name: ${c.name}\n  labels: { tuskline.dev/env: ${c.env} }\nspec:\n  instances: ${1 + c.replicas}\n  imageName: ghcr.io/cloudnative-pg/postgresql:${c.pg}\n  primaryUpdateStrategy: unsupervised\n${c.sync ? "  postgresql:\n    synchronous: { method: any, number: 1 }\n" : "  postgresql:\n"}    shared_preload_libraries: [${c.exts.filter((e) => ["auto_explain", "pg_wait_sampling", "pg_stat_kcache"].includes(e)).join(", ")}]\n    parameters:\n      shared_buffers: ${s.sb}\n      pg_stat_statements.track: top\n  storage: { size: ${s.disk}Gi }\n  resources:\n    requests: { cpu: "${s.cpu}", memory: ${s.ram}Gi }\n    limits: { memory: ${s.ram}Gi }\n  monitoring: { enablePodMonitor: true }\n  plugins:\n    - name: barman-cloud.cloudnative-pg.io\n      isWALArchiver: true\n      parameters: { barmanObjectName: ${c.name}-store }\n---\napiVersion: barmancloud.cnpg.io/v1\nkind: ObjectStore\nmetadata: { name: ${c.name}-store }\nspec:\n  configuration:\n    destinationPath: ${store}\n  retentionPolicy: "${c.retentionDays}d"\n---\napiVersion: postgresql.cnpg.io/v1\nkind: ScheduledBackup\nmetadata: { name: ${c.name}-daily }\nspec:\n  schedule: "0 0 2 * * *"   # six-field cron (seconds first)\n  cluster: { name: ${c.name} }\n  method: plugin\n  pluginConfiguration: { name: barman-cloud.cloudnative-pg.io }\n${c.pgbouncer ? `---\napiVersion: postgresql.cnpg.io/v1\nkind: Pooler\nmetadata: { name: ${c.name}-pooler-rw }\nspec:\n  cluster: { name: ${c.name} }\n  instances: 2\n  type: rw\n  pgbouncer:\n    poolMode: transaction\n    parameters: { max_client_conn: "2000", default_pool_size: "20" }\n` : ""}`;
}
export function genInventory(c: DeployCfg) {
  const hosts = c.hosts.split(/[\s,]+/).filter(Boolean);
  const nodes = Array.from({ length: 1 + c.replicas }, (_, i) => hosts[i] ?? `10.0.1.${11 + i}`);
  return `# inventory.yml — Ansible inventory, Autobase-style (preview, demo IPs)\nall:\n  vars:\n    patroni_cluster_name: ${c.name}\n    postgresql_version: ${c.pg.split(".")[0]}\n    dcs_type: etcd\n    with_haproxy_load_balancing: ${c.entry === "haproxy"}\n    cluster_vip: ${c.entry === "vip" ? "10.0.1.100" : '""'}\n    pgbouncer_install: ${c.pgbouncer}\n    synchronous_mode: ${c.sync}\n    pgbackrest_install: true\n    pgbackrest_repo_type: ${c.repo === "volume" ? "posix" : c.repo}\n    pgbackrest_retention_days: ${c.retentionDays}\n    postgresql_parameters:\n      - { option: shared_buffers, value: ${SIZES[c.size].sb} }\n      - { option: shared_preload_libraries, value: "${c.exts.filter((e) => ["pg_stat_statements", "auto_explain", "pg_wait_sampling", "pg_stat_kcache"].includes(e)).join(",")}" }\n  children:\n    etcd_cluster:\n      hosts:\n${nodes.slice(0, Math.max(c.etcd, 1)).map((h) => `        ${h}:`).join("\n")}\n    master:\n      hosts:\n        ${nodes[0]}: { hostname: ${c.name}-1 }\n    replica:\n      hosts:\n${nodes.slice(1).map((h, i) => `        ${h}: { hostname: ${c.name}-${i + 2} }`).join("\n") || "        {}"}\n    postgres_cluster:\n      children: { master: {}, replica: {} }\n# Run: ansible-playbook -i inventory.yml deploy_pgcluster.yml`;
}
export function deployLog(c: DeployCfg): { step: string; line: string }[] {
  const n = 1 + c.replicas; const L: { step: string; line: string }[] = [];
  const add = (step: string, ...lines: string[]) => lines.forEach((line) => L.push({ step, line }));
  add("Pre-flight", `checking ${c.target === "compose" ? "docker engine 27.3 on demo-host" : c.target === "cnpg" ? "kube context demo-gke (CNPG operator 1.25 found)" : `${n} hosts over SSH (ubuntu@10.0.1.x)`}`, "disk ≥ " + SIZES[c.size].disk + " GB free ✓", "ports 5432, 2379, 2380, 8008 free ✓", "object-store credentials present (placeholder) ✓");
  add("Plan", `${n} PostgreSQL ${c.pg} node(s), ${c.target === "cnpg" ? "operator-managed failover" : `${c.etcd} etcd member(s)`}${c.pgbouncer ? ", PgBouncer" : ""}, backups → ${c.repo}`, "0 to change · " + (n + c.etcd + 3) + " to add · 0 to destroy");
  if (c.target !== "cnpg") add("Consensus", ...Array.from({ length: c.etcd }, (_, i) => `etcd${i + 1} started`), `etcd quorum healthy (${c.etcd}/${c.etcd})`);
  add("Primary", `bootstrapping ${c.name}-1 (initdb, data checksums on)`, "applied shared_buffers=" + SIZES[c.size].sb, "leader lock acquired by " + c.name + "-1");
  for (let i = 2; i <= n; i++) add("Replicas", `${c.name}-${i}: pg_basebackup from leader…`, `${c.name}-${i}: streaming, lag 0 B${c.sync && i === 2 ? " (synchronous)" : ""}`);
  add("Pooling & routing", c.entry === "haproxy" ? "haproxy: 5000 → leader, 5001 → replicas" : c.entry === "vip" ? "vip-manager: 10.0.1.100 follows leader" : `Service ${c.name}-rw → primary`, ...(c.pgbouncer ? ["pgbouncer: transaction pooling on :6432"] : []));
  add("Backups", "stanza created · first full backup 2.1 GB in 48 s", `WAL archiving verified · schedule full ${c.fullCron}`);
  add("Extensions", ...c.exts.filter((e) => e !== "auto_explain").map((e) => `CREATE EXTENSION ${e === "pgvector" ? "vector" : e} ✓`), "tuskline_monitor role created (pg_monitor, read-only)");
  add("Hand-over", "health score 100 · registered in Fleet", "deployment complete");
  return L;
}
