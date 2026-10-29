// All data in this file is FAKE demo data for the Tuskline prototype.
export type Env = "prod" | "staging" | "dev";
export type Health = "ok" | "warn" | "crit";

export function rng(seed: number) {
  let s = seed >>> 0;
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0;
    return s / 4294967296;
  };
}
export function series(seed: number, n: number, base: number, amp: number, drift = 0) {
  const r = rng(seed);
  const out: number[] = [];
  let v = base;
  for (let i = 0; i < n; i++) {
    v += (r() - 0.5) * amp + drift;
    v = Math.max(base * 0.35, Math.min(base * 1.9, v));
    out.push(+v.toFixed(2));
  }
  return out;
}

export interface Cluster {
  id: string; name: string; env: Env; health: Health; score: number;
  version: string; ha: "Patroni" | "CloudNativePG" | "RDS Multi-AZ" | "Standalone" | "Crunchy PGO";
  provider: "AWS" | "GCP" | "Azure" | "k8s" | "Local";
  region: string; primary: number; replicas: number;
  tps: number; conns: number; maxConns: number; lagBytes: number; lagSec: number;
  backupAgeH: number; advisories: number; storageGb: number; storagePct: number;
  load: number[]; databases: string[]; issue?: string;
}

export const ENVS: { id: Env; label: string; color: string }[] = [
  { id: "prod", label: "Production", color: "var(--prod)" },
  { id: "staging", label: "Staging", color: "var(--staging)" },
  { id: "dev", label: "Development", color: "var(--dev)" },
];
export const envColor = (e: Env) => ENVS.find((x) => x.id === e)!.color;

export const CLUSTERS: Cluster[] = [
  { id: "prod-eu-west", name: "prod-eu-west", env: "prod", health: "warn", score: 78, version: "17.2", ha: "Patroni", provider: "AWS", region: "eu-west-2", primary: 1, replicas: 2, tps: 4820, conns: 312, maxConns: 500, lagBytes: 48_200_000, lagSec: 6.4, backupAgeH: 5, advisories: 4, storageGb: 1840, storagePct: 71, load: series(11, 40, 6, 2.2, 0.03), databases: ["orders", "inventory", "auth"], issue: "Replica pg-euw-3 lagging 6.4 s" },
  { id: "prod-us-east", name: "prod-us-east", env: "prod", health: "ok", score: 96, version: "17.2", ha: "CloudNativePG", provider: "k8s", region: "us-east-1 · eks", primary: 1, replicas: 2, tps: 6110, conns: 280, maxConns: 600, lagBytes: 1_200_000, lagSec: 0.2, backupAgeH: 2, advisories: 1, storageGb: 2210, storagePct: 54, load: series(22, 40, 5, 1.6), databases: ["payments", "ledger"] },
  { id: "prod-eu-central", name: "prod-eu-central-ledger", env: "prod", health: "crit", score: 41, version: "16.6", ha: "Patroni", provider: "GCP", region: "europe-west3", primary: 1, replicas: 1, tps: 1930, conns: 488, maxConns: 500, lagBytes: 3_400_000, lagSec: 0.6, backupAgeH: 31, advisories: 7, storageGb: 960, storagePct: 89, load: series(33, 40, 9, 3, 0.06), databases: ["ledger"], issue: "Connections at 98% · backup 31 h old" },
  { id: "prod-ap-south", name: "prod-ap-south", env: "prod", health: "ok", score: 92, version: "17.1", ha: "RDS Multi-AZ", provider: "AWS", region: "ap-south-1", primary: 1, replicas: 1, tps: 2140, conns: 140, maxConns: 400, lagBytes: 0, lagSec: 0, backupAgeH: 3, advisories: 2, storageGb: 720, storagePct: 48, load: series(44, 40, 3, 1.2), databases: ["catalogue", "search"] },
  { id: "staging-eu-west", name: "staging-eu-west", env: "staging", health: "ok", score: 94, version: "17.2", ha: "Patroni", provider: "AWS", region: "eu-west-2", primary: 1, replicas: 1, tps: 410, conns: 64, maxConns: 200, lagBytes: 210_000, lagSec: 0.1, backupAgeH: 9, advisories: 2, storageGb: 310, storagePct: 37, load: series(55, 40, 1.4, 0.8), databases: ["orders", "inventory"] },
  { id: "staging-k8s", name: "staging-k8s", env: "staging", health: "warn", score: 81, version: "17.2", ha: "CloudNativePG", provider: "k8s", region: "gke · europe-west2", primary: 1, replicas: 2, tps: 260, conns: 41, maxConns: 200, lagBytes: 900_000, lagSec: 0.4, backupAgeH: 26, advisories: 3, storageGb: 180, storagePct: 62, load: series(66, 40, 1.1, 0.7), databases: ["payments"], issue: "Last base backup 26 h ago" },
  { id: "staging-us-east", name: "staging-us-east", env: "staging", health: "ok", score: 97, version: "17.2", ha: "Crunchy PGO", provider: "k8s", region: "us-east-1", primary: 1, replicas: 1, tps: 190, conns: 22, maxConns: 200, lagBytes: 80_000, lagSec: 0.0, backupAgeH: 4, advisories: 0, storageGb: 140, storagePct: 21, load: series(77, 40, 0.9, 0.5), databases: ["ledger"] },
  { id: "dev-shared", name: "dev-shared", env: "dev", health: "ok", score: 90, version: "17.2", ha: "Standalone", provider: "Azure", region: "uksouth", primary: 1, replicas: 0, tps: 38, conns: 17, maxConns: 100, lagBytes: 0, lagSec: 0, backupAgeH: 22, advisories: 5, storageGb: 60, storagePct: 33, load: series(88, 40, 0.5, 0.4), databases: ["orders", "payments", "scratch"] },
  { id: "dev-preview-1423", name: "dev-preview-pr-1423", env: "dev", health: "ok", score: 99, version: "18.0", ha: "Standalone", provider: "k8s", region: "kind · ephemeral", primary: 1, replicas: 0, tps: 4, conns: 3, maxConns: 100, lagBytes: 0, lagSec: 0, backupAgeH: 0, advisories: 0, storageGb: 4, storagePct: 8, load: series(99, 40, 0.2, 0.2), databases: ["orders"] },
  { id: "dev-local", name: "local-docker", env: "dev", health: "ok", score: 100, version: "17.2", ha: "Standalone", provider: "Local", region: "localhost:5432", primary: 1, replicas: 0, tps: 1, conns: 2, maxConns: 100, lagBytes: 0, lagSec: 0, backupAgeH: 0, advisories: 1, storageGb: 2, storagePct: 3, load: series(101, 40, 0.1, 0.1), databases: ["postgres"] },
];

export const WAIT_KEYS = [
  { key: "CPU", color: "var(--c1)" },
  { key: "IO:DataFileRead", color: "var(--c3)" },
  { key: "Lock:transactionid", color: "var(--c5)" },
  { key: "LWLock:WALWrite", color: "var(--c4)" },
  { key: "Client:ClientRead", color: "var(--c2)" },
  { key: "Other", color: "var(--c6)" },
] as const;

export function loadSeries(seed: number, n = 72) {
  const r = rng(seed);
  const out: Record<string, number | string>[] = [];
  for (let i = 0; i < n; i++) {
    const t = new Date(Date.UTC(2026, 9, 27, 8, 0) + i * 5 * 60000);
    const spike = i > 44 && i < 56 ? 1 : 0;
    out.push({
      t: t.toISOString().slice(11, 16),
      CPU: +(2.4 + r() * 1.2 + Math.sin(i / 8) * 0.5).toFixed(2),
      "IO:DataFileRead": +(0.9 + r() * 0.8 + spike * 1.4).toFixed(2),
      "Lock:transactionid": +(0.1 + r() * 0.2 + spike * 2.1).toFixed(2),
      "LWLock:WALWrite": +(0.3 + r() * 0.3).toFixed(2),
      "Client:ClientRead": +(0.4 + r() * 0.3).toFixed(2),
      Other: +(0.2 + r() * 0.2).toFixed(2),
      tps: Math.round(4300 + r() * 900 - spike * 1100),
      active: Math.round(38 + r() * 12 + spike * 30),
      idle: Math.round(220 + r() * 25),
      idleTx: Math.round(4 + r() * 4 + spike * 14),
      hit: +(99.1 + r() * 0.7 - spike * 1.6).toFixed(2),
      lag3: +(0.3 + (i > 50 ? (i - 50) * 0.3 : 0) + r() * 0.3).toFixed(2),
      lag2: +(0.2 + r() * 0.3).toFixed(2),
    });
  }
  return out;
}

export interface Finding { id: string; sev: Health; title: string; detail: string; fix?: string; screen?: string }
export const FINDINGS: Finding[] = [
  { id: "f1", sev: "crit", title: "Lock contention on orders (transactionid)", detail: "Between 11:45 and 12:35, 31% of DB load was Lock:transactionid. Root blocker: pid 48211 (billing-worker) held a row lock for 4 m 12 s inside an idle-in-transaction session.", fix: "SET idle_in_transaction_session_timeout = '60s';", screen: "queries" },
  { id: "f2", sev: "warn", title: "Replica pg-euw-3 replay lag rising", detail: "Replay lag 6.4 s (48 MB) and climbing for 18 min. Replica is IO-bound: DataFileRead waits 3× baseline after the nightly VACUUM of order_events.", screen: "ha" },
  { id: "f3", sev: "warn", title: "Sequential scans on order_items (12.4 GB)", detail: "Query #a91f3c scans order_items 1,840×/min. A composite index on (order_id, sku) is predicted to cut its mean from 182 ms to 3.1 ms (hypopg-validated).", fix: "CREATE INDEX CONCURRENTLY idx_order_items_order_sku ON public.order_items (order_id, sku);", screen: "queries" },
  { id: "f4", sev: "ok", title: "Autovacuum keeping up", detail: "Oldest xid age is 182 M (9% of wraparound). Dead tuple ratio below 5% on all tables over 1 GB." },
];

export interface Query {
  id: string; sql: string; db: string; user: string; calls: number; callsPerSec: number;
  meanMs: number; p95Ms: number; rows: number; share: number; readMB: number; trend: number[]; regressed?: boolean; flags: string[];
}
export const QUERIES: Query[] = [
  { id: "a91f3c", sql: "SELECT oi.sku, oi.qty, oi.unit_price, p.title FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.order_id = $1 AND oi.sku = ANY($2)", db: "orders", user: "api", calls: 1_104_220, callsPerSec: 30.7, meanMs: 182.4, p95Ms: 410, rows: 4.2, share: 34.1, readMB: 18420, trend: series(1, 24, 140, 30, 2), regressed: true, flags: ["Seq Scan", "Rows misestimate ×412"] },
  { id: "4be012", sql: "UPDATE orders SET status = $1, updated_at = now() WHERE id = $2", db: "orders", user: "billing-worker", calls: 3_912_004, callsPerSec: 108.6, meanMs: 9.8, p95Ms: 61, rows: 1, share: 17.2, readMB: 210, trend: series(2, 24, 9, 3), flags: ["Lock waits"] },
  { id: "c7d9e1", sql: "SELECT * FROM inventory_levels WHERE warehouse_id = $1 AND updated_at > $2 ORDER BY updated_at DESC LIMIT $3", db: "inventory", user: "sync", calls: 480_118, callsPerSec: 13.3, meanMs: 41.2, p95Ms: 96, rows: 50, share: 9.1, readMB: 3110, trend: series(3, 24, 40, 6), flags: ["Sort spill 24 MB"] },
  { id: "19f2aa", sql: "INSERT INTO order_events (order_id, kind, payload, created_at) VALUES ($1, $2, $3, now())", db: "orders", user: "api", calls: 6_220_310, callsPerSec: 172.8, meanMs: 1.9, p95Ms: 4.2, rows: 1, share: 6.8, readMB: 12, trend: series(4, 24, 2, 0.4), flags: [] },
  { id: "e30b77", sql: "SELECT count(*) FROM sessions WHERE user_id = $1 AND expires_at > now()", db: "auth", user: "auth", calls: 2_010_554, callsPerSec: 55.8, meanMs: 4.1, p95Ms: 9.5, rows: 1, share: 4.7, readMB: 640, trend: series(5, 24, 4, 1), flags: [] },
  { id: "77ac40", sql: "WITH daily AS (SELECT date_trunc('day', created_at) d, sum(total) s FROM orders WHERE created_at >= $1 GROUP BY 1) SELECT * FROM daily ORDER BY d", db: "orders", user: "metabase", calls: 2_204, callsPerSec: 0.06, meanMs: 7420, p95Ms: 11800, rows: 30, share: 4.4, readMB: 52200, trend: series(6, 24, 7000, 900), flags: ["Parallel Seq Scan", "Hash spill 410 MB"] },
  { id: "0d81f5", sql: "SELECT id, email, display_name FROM users WHERE lower(email) = lower($1)", db: "auth", user: "auth", calls: 810_442, callsPerSec: 22.5, meanMs: 6.3, p95Ms: 14, rows: 1, share: 3.1, readMB: 980, trend: series(7, 24, 6, 1.4), flags: ["Function on indexed column"] },
  { id: "b5e6c2", sql: "DELETE FROM carts WHERE updated_at < now() - interval '30 days'", db: "orders", user: "cron", calls: 24, callsPerSec: 0, meanMs: 48200, p95Ms: 61000, rows: 182_004, share: 2.2, readMB: 9800, trend: series(8, 24, 45000, 6000), flags: ["Long-running"] },
  { id: "3f0a9d", sql: "SELECT p.* FROM products p WHERE p.search_vector @@ plainto_tsquery('english', $1) LIMIT 20", db: "inventory", user: "api", calls: 340_121, callsPerSec: 9.4, meanMs: 11.7, p95Ms: 33, rows: 20, share: 2.0, readMB: 1220, trend: series(9, 24, 12, 2), flags: [] },
  { id: "8c21de", sql: "SELECT pg_advisory_xact_lock($1)", db: "orders", user: "billing-worker", calls: 3_910_880, callsPerSec: 108.6, meanMs: 0.4, p95Ms: 2.1, rows: 1, share: 1.1, readMB: 0, trend: series(10, 24, 0.4, 0.1), flags: [] },
];

export interface PlanNode {
  id: string; op: string; rel?: string; idx?: string; detail?: string; selfMs: number; totalMs: number;
  estRows: number; actRows: number; loops: number; bufRead: number; bufHit: number; children?: PlanNode[]; badges?: string[];
}
export const PLAN_BEFORE: PlanNode = {
  id: "n1", op: "Nested Loop", selfMs: 2.1, totalMs: 181.9, estRows: 1, actRows: 412, loops: 1, bufRead: 0, bufHit: 6,
  badges: ["Rows ×412"],
  children: [
    { id: "n2", op: "Seq Scan", rel: "order_items oi", detail: "Filter: ((order_id = $1) AND (sku = ANY ($2)))\nRows Removed by Filter: 9,812,330", selfMs: 171.4, totalMs: 171.4, estRows: 1, actRows: 412, loops: 1, bufRead: 1_612_004, bufHit: 3_220, badges: ["Seq Scan", "Rows ×412", "94% of time"] },
    { id: "n3", op: "Index Scan", rel: "products p", idx: "products_pkey", detail: "Index Cond: (id = oi.product_id)", selfMs: 8.4, totalMs: 8.4, estRows: 1, actRows: 1, loops: 412, bufRead: 12, bufHit: 1_640 },
  ],
};
export const PLAN_AFTER: PlanNode = {
  id: "m1", op: "Nested Loop", selfMs: 0.3, totalMs: 3.1, estRows: 398, actRows: 412, loops: 1, bufRead: 0, bufHit: 4,
  children: [
    { id: "m2", op: "Index Scan", rel: "order_items oi", idx: "idx_order_items_order_sku (hypothetical)", detail: "Index Cond: ((order_id = $1) AND (sku = ANY ($2)))", selfMs: 1.9, totalMs: 1.9, estRows: 398, actRows: 412, loops: 1, bufRead: 0, bufHit: 418 },
    { id: "m3", op: "Index Scan", rel: "products p", idx: "products_pkey", detail: "Index Cond: (id = oi.product_id)", selfMs: 0.9, totalMs: 0.9, estRows: 1, actRows: 1, loops: 412, bufRead: 0, bufHit: 1_236 },
  ],
};

export const SESSIONS = [
  { pid: 48211, user: "billing-worker", app: "billing-worker@v3.14", state: "idle in transaction", wait: "Client:ClientRead", dur: "4m 12s", query: "UPDATE orders SET status = 'invoiced' WHERE id = 99120431", blocking: 14 },
  { pid: 48377, user: "billing-worker", app: "billing-worker@v3.14", state: "active", wait: "Lock:transactionid", dur: "3m 58s", query: "UPDATE orders SET status = $1, updated_at = now() WHERE id = $2", blockedBy: 48211 },
  { pid: 48402, user: "api", app: "orders-api", state: "active", wait: "Lock:transactionid", dur: "3m 41s", query: "UPDATE orders SET status = $1, updated_at = now() WHERE id = $2", blockedBy: 48211 },
  { pid: 51022, user: "metabase", app: "Metabase", state: "active", wait: "IO:DataFileRead", dur: "52s", query: "WITH daily AS (SELECT date_trunc('day', created_at) d, …" },
  { pid: 51190, user: "api", app: "orders-api", state: "active", wait: "CPU", dur: "180ms", query: "SELECT oi.sku, oi.qty, oi.unit_price, p.title FROM order_items oi …" },
];

export const SCHEMA = [
  { schema: "public", tables: [
    { name: "orders", rows: "48.2 M", size: "21 GB", cols: [["id", "bigint", "pk"], ["customer_id", "bigint", "fk"], ["status", "order_status"], ["total", "numeric(12,2)"], ["currency", "char(3)"], ["created_at", "timestamptz"], ["updated_at", "timestamptz"]] },
    { name: "order_items", rows: "9.8 M", size: "12.4 GB", cols: [["id", "bigint", "pk"], ["order_id", "bigint", "fk"], ["product_id", "bigint", "fk"], ["sku", "text"], ["qty", "integer"], ["unit_price", "numeric(12,2)"]] },
    { name: "order_events", rows: "412 M", size: "88 GB", cols: [["id", "bigint", "pk"], ["order_id", "bigint", "fk"], ["kind", "text"], ["payload", "jsonb"], ["created_at", "timestamptz"]] },
    { name: "customers", rows: "3.1 M", size: "1.9 GB", cols: [["id", "bigint", "pk"], ["email", "citext"], ["name", "text"], ["country", "char(2)"], ["created_at", "timestamptz"]] },
    { name: "products", rows: "84 k", size: "210 MB", cols: [["id", "bigint", "pk"], ["sku", "text"], ["title", "text"], ["price", "numeric(12,2)"], ["search_vector", "tsvector"]] },
  ] },
  { schema: "billing", tables: [
    { name: "invoices", rows: "22 M", size: "9.1 GB", cols: [["id", "bigint", "pk"], ["order_id", "bigint", "fk"], ["amount", "numeric(12,2)"], ["issued_at", "timestamptz"]] },
    { name: "refunds", rows: "410 k", size: "120 MB", cols: [["id", "bigint", "pk"], ["invoice_id", "bigint", "fk"], ["amount", "numeric(12,2)"], ["reason", "text"]] },
  ] },
];

export const RESULT_ROWS = [
  ["99120431", "4108221", "invoiced", "184.20", "GBP", "2026-10-27 11:42:08+00"],
  ["99120430", "2290114", "paid", "62.00", "GBP", "2026-10-27 11:42:07+00"],
  ["99120429", "5521907", "shipped", "1,240.99", "EUR", "2026-10-27 11:42:05+00"],
  ["99120428", "1180342", "paid", "19.99", "GBP", "2026-10-27 11:42:05+00"],
  ["99120427", "3302915", "pending", "75.40", "EUR", "2026-10-27 11:42:03+00"],
  ["99120426", "4108221", "cancelled", "12.00", "GBP", "2026-10-27 11:42:01+00"],
  ["99120425", "6610045", "paid", "349.00", "USD", "2026-10-27 11:41:59+00"],
  ["99120424", "2081776", "invoiced", "88.10", "GBP", "2026-10-27 11:41:58+00"],
  ["99120423", "7710283", "shipped", "23.45", "EUR", "2026-10-27 11:41:56+00"],
  ["99120422", "1092837", "paid", "540.00", "GBP", "2026-10-27 11:41:55+00"],
  ["99120421", "3349901", "paid", "17.25", "GBP", "2026-10-27 11:41:53+00"],
  ["99120420", "5590012", "refunded", "64.99", "USD", "2026-10-27 11:41:52+00"],
];

export type BackupKind = "full" | "diff" | "incr";
export interface Backup { id: string; kind: BackupKind; start: number; durMin: number; sizeGb: number; repo: string; verified: boolean }
// times are hours relative to now (negative = past)
export const BACKUPS: Backup[] = (() => {
  const out: Backup[] = [];
  for (let d = 13; d >= 0; d--) {
    const kind: BackupKind = d % 7 === 6 ? "full" : d % 7 === 3 ? "diff" : "incr";
    out.push({ id: `202610${String(27 - d).padStart(2, "0")}-010000${kind[0].toUpperCase()}`, kind, start: -d * 24 - 5, durMin: kind === "full" ? 74 : kind === "diff" ? 31 : 9, sizeGb: kind === "full" ? 412 : kind === "diff" ? 96 : 14 + (d % 4) * 3, repo: d % 2 ? "repo1 · s3" : "repo1 · s3 + repo2 · gcs", verified: d !== 4 });
  }
  return out;
})();
export const WAL_GAP = { from: -98.5, to: -97.2 };

export interface Migration {
  id: string; title: string; author: string; status: "review" | "approved" | "applied" | "blocked"; age: string; db: string;
  sql: string; envs: { env: Env; state: "applied" | "pending" | "failed" | "drift" }[];
}
export const MIGRATIONS: Migration[] = [
  { id: "0142", title: "Add fulfilment_centre to orders", author: "priya.n", status: "review", age: "2h", db: "orders",
    sql: `-- 0142_add_fulfilment_centre.sql
ALTER TABLE public.orders
  ADD COLUMN fulfilment_centre_id bigint;

ALTER TABLE public.orders
  ADD CONSTRAINT orders_fulfilment_centre_fk
  FOREIGN KEY (fulfilment_centre_id)
  REFERENCES public.fulfilment_centres (id);

CREATE INDEX idx_orders_fulfilment_centre
  ON public.orders (fulfilment_centre_id);

ALTER TABLE public.orders
  ALTER COLUMN currency SET NOT NULL;`,
    envs: [{ env: "dev", state: "applied" }, { env: "staging", state: "applied" }, { env: "prod", state: "pending" }] },
  { id: "0141", title: "Partition order_events by month", author: "tom.b", status: "approved", age: "1d", db: "orders", sql: "-- 0141_partition_order_events.sql\nCREATE TABLE public.order_events_p (LIKE public.order_events INCLUDING ALL)\n  PARTITION BY RANGE (created_at);", envs: [{ env: "dev", state: "applied" }, { env: "staging", state: "pending" }, { env: "prod", state: "pending" }] },
  { id: "0140", title: "Drop legacy carts.session_token", author: "mark.a", status: "applied", age: "3d", db: "orders", sql: "-- 0140_drop_session_token.sql\nALTER TABLE public.carts DROP COLUMN session_token;", envs: [{ env: "dev", state: "applied" }, { env: "staging", state: "applied" }, { env: "prod", state: "applied" }] },
  { id: "0139", title: "Add refunds.reason enum", author: "li.w", status: "blocked", age: "5d", db: "orders", sql: "-- 0139_refund_reason.sql\nCREATE TYPE billing.refund_reason AS ENUM ('damaged','late','other');", envs: [{ env: "dev", state: "applied" }, { env: "staging", state: "drift" }, { env: "prod", state: "pending" }] },
];
