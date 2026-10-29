import { useEffect, useMemo, useRef, useState } from "react";
import { ArrowLeft, ArrowRight, Box, CheckCircle2, Container, Database, HardDrive, Minus, Plus, Rocket, Server, Shield, Ship, AlertTriangle, Loader2, Check } from "lucide-react";
import { useApp } from "../store";
import { ENVS, Env, series, Cluster } from "../data";
import { Badge, Card, PageHeader, Segmented, tint } from "../components/ui";
import { CheckRow, CheckState, CodeTabs, DemoTag, Field, Stepper, Toggle } from "../components/setup-ui";
import { DEPLOY_EXTS, DeployCfg, DeployTarget, SIZES, TARGET_LABEL, deployLog, genCnpg, genCompose, genInventory } from "../setup-data";

const STEPS = ["Target", "Topology", "Sizing & backups", "Review", "Deploy"];
const DEFAULT: DeployCfg = { name: "orders-demo", env: "staging", target: "compose", replicas: 2, etcd: 3, pgbouncer: true, repo: "volume", entry: "haproxy", sync: false, size: "small", pg: "17", exts: ["pg_stat_statements", "auto_explain", "pg_wait_sampling", "hypopg"], fullCron: "Sun 01:00", diffCron: "daily 01:00", retentionDays: 14, hosts: "10.0.1.11 10.0.1.12 10.0.1.13" };
const TARGETS: { id: DeployTarget; icon: typeof Box; title: string; blurb: string; tag?: string; ha: string }[] = [
  { id: "compose", icon: Container, title: "Docker Compose", blurb: "One host, Patroni + etcd in containers. Fastest way to a working cluster for dev, demos and small installs.", tag: "Recommended to start", ha: "Not real HA: the host is a single point of failure" },
  { id: "cnpg", icon: Ship, title: "Kubernetes · CloudNativePG", blurb: "Operator-managed Cluster, Pooler and Barman Cloud backups. Failover handled by the operator.", ha: "HA across nodes / zones" },
  { id: "vms", icon: Server, title: "VMs via SSH", blurb: "Autobase-style Ansible: Patroni, etcd, HAProxy or a VIP, pgBackRest on your own hosts.", ha: "HA across hosts" },
];

export default function Deploy() {
  const { go } = useApp();
  const [step, setStep] = useState(0);
  const [c, setC] = useState<DeployCfg>(DEFAULT);
  const set = <K extends keyof DeployCfg>(k: K, v: DeployCfg[K]) => setC((x) => ({ ...x, [k]: v }));
  useEffect(() => {
    const on = (e: Event) => setC((x) => ({ ...x, target: (e as CustomEvent).detail as DeployTarget }));
    window.addEventListener("tl:target", on); return () => window.removeEventListener("tl:target", on);
  }, []);
  useEffect(() => { // keep options coherent with the target
    setC((x) => ({ ...x, entry: x.target === "cnpg" ? "service" : x.entry === "service" ? "haproxy" : x.entry, repo: x.target === "cnpg" && x.repo === "volume" ? "s3" : x.repo }));
  }, [c.target]);
  const files = useMemo(() => {
    const t = c.target === "compose" ? { id: "compose", label: "compose.yaml", text: genCompose(c), file: "compose.yaml" }
      : c.target === "cnpg" ? { id: "cnpg", label: "cluster.yaml", text: genCnpg(c), file: "cluster.yaml" }
      : { id: "inv", label: "inventory.yml", text: genInventory(c), file: "inventory.yml" };
    return [t, { id: "tl", label: "tuskline.yaml", text: `# Desired state, kept in git. tl deploy plan|apply reads this.\ncluster: ${c.name}\nenv: ${c.env}\ntarget: ${c.target}\npostgres: { version: "${c.pg}", size: ${c.size} }\ntopology: { replicas: ${c.replicas}, etcd: ${c.target === "cnpg" ? "operator" : c.etcd}, pooler: ${c.pgbouncer}, entry: ${c.entry}, synchronous: ${c.sync} }\nextensions: [${c.exts.join(", ")}]\nbackups: { tool: ${c.target === "cnpg" ? "barman-cloud" : "pgbackrest"}, repo: ${c.repo}, full: "${c.fullCron}", diff: "${c.diffCron}", retention_days: ${c.retentionDays} }`, file: "tuskline.yaml" }];
  }, [c]);
  const busy = step === 4;
  return (
    <div>
      <PageHeader icon={<Rocket size={18} className="text-accent" />} title="Deploy a new cluster" sub={<>Guided, reviewable deployment. Every step produces a file you can keep in git. <DemoTag /></>}
        actions={<button className="btn btn-ghost" onClick={() => go("deployments")}>Deployment history</button>} />
      <Stepper steps={STEPS} at={step} onJump={busy ? undefined : setStep} />
      {step < 3 && (
        <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,0.95fr)] gap-4 items-start">
          <div className="space-y-4">
            {step === 0 && <TargetStep c={c} set={set} />}
            {step === 1 && <TopologyStep c={c} set={set} />}
            {step === 2 && <SizingStep c={c} set={set} />}
          </div>
          <div className="sticky top-4"><Card title="Live preview" sub="regenerates as you change options"><CodeTabs key={c.target} tabs={files} maxH={560} /></Card></div>
        </div>
      )}
      {step === 3 && <ReviewStep c={c} files={files} />}
      {step === 4 && <RunStep c={c} />}
      {step < 4 && (
        <div className="flex items-center gap-2 mt-5 pt-4 border-t border-line">
          {step > 0 && <button className="btn" onClick={() => setStep(step - 1)}><ArrowLeft size={13} /> Back</button>}
          <span className="text-fg3 text-[12px] ml-2">Step {step + 1} of {STEPS.length}</span>
          <span className="ml-auto" />
          {step < 3 ? <button className="btn btn-primary" disabled={!c.name.trim()} onClick={() => setStep(step + 1)}>Continue <ArrowRight size={13} /></button>
            : <button className={`btn ${c.env === "prod" ? "btn-danger" : "btn-primary"}`} onClick={() => setStep(4)}><Rocket size={13} /> Deploy {c.name}</button>}
        </div>
      )}
    </div>
  );
}

type SetFn = <K extends keyof DeployCfg>(k: K, v: DeployCfg[K]) => void;
function EnvPick({ v, on }: { v: Env; on: (e: Env) => void }) {
  return (
    <div className="flex gap-1.5">
      {(["dev", "staging", "prod"] as Env[]).map((e) => {
        const col = ENVS.find((x) => x.id === e)!.color; const sel = v === e;
        return <button key={e} onClick={() => on(e)} className="h-7 px-3 rounded-md text-[12px] font-medium uppercase tracking-wide" style={sel ? { background: tint(col, 18), color: col, boxShadow: `inset 0 0 0 1px ${tint(col, 50)}` } : { color: "var(--fg3)", boxShadow: "inset 0 0 0 1px var(--line)" }}>{e}</button>;
      })}
    </div>
  );
}

function TargetStep({ c, set }: { c: DeployCfg; set: SetFn }) {
  return (
    <>
      <div className="grid grid-cols-1 gap-2.5">
        {TARGETS.map((t) => {
          const on = c.target === t.id; const Icon = t.icon;
          return (
            <button key={t.id} onClick={() => set("target", t.id)} className="card p-4 text-left flex gap-3.5 transition-all focus-ring"
              style={on ? { borderColor: "var(--accent)", boxShadow: `0 0 0 3px ${tint("var(--accent)", 15)}` } : undefined}>
              <div className="w-10 h-10 rounded-lg grid place-items-center shrink-0" style={{ background: on ? tint("var(--accent)", 15) : "var(--subtle)", color: on ? "var(--accent)" : "var(--fg2)" }}><Icon size={19} /></div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2"><span className="font-semibold">{t.title}</span>{t.tag && <Badge color="var(--accent)">{t.tag}</Badge>}</div>
                <div className="text-fg2 text-[12.5px] mt-1 leading-relaxed">{t.blurb}</div>
                <div className="text-[11.5px] mt-1.5 flex items-center gap-1" style={{ color: t.id === "compose" ? "var(--warn)" : "var(--ok)" }}>{t.id === "compose" ? <AlertTriangle size={11} /> : <Shield size={11} />}{t.ha}</div>
              </div>
              <span className="w-4 h-4 rounded-full mt-1 shrink-0 grid place-items-center" style={{ boxShadow: `inset 0 0 0 ${on ? 5 : 1.5}px ${on ? "var(--brand)" : "var(--line2)"}` }} />
            </button>
          );
        })}
      </div>
      <Card title="Cluster">
        <div className="grid grid-cols-2 gap-3">
          <Field label="Name" hint="Lower-case, used for hostnames and the Patroni scope."><input className="input w-full mono" value={c.name} onChange={(e) => set("name", e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "-"))} /></Field>
          <Field label="Environment tag"><EnvPick v={c.env} on={(e) => set("env", e)} /></Field>
          {c.target === "vms" && <Field label="Hosts (SSH)" className="col-span-2" hint="One per node; first host becomes the initial leader. Demo addresses."><input className="input w-full mono" value={c.hosts} onChange={(e) => set("hosts", e.target.value)} /></Field>}
          {c.target === "cnpg" && <Field label="Kube context" className="col-span-2"><div className="input w-full flex items-center mono text-fg2">demo-gke · namespace databases · CNPG operator 1.25 detected (demo)</div></Field>}
          {c.target === "compose" && <Field label="Docker host" className="col-span-2"><div className="input w-full flex items-center mono text-fg2">unix:///var/run/docker.sock · Docker 27.3 (demo)</div></Field>}
        </div>
      </Card>
    </>
  );
}

function Stepr({ v, min, max, on }: { v: number; min: number; max: number; on: (n: number) => void }) {
  return (
    <div className="inline-flex items-center rounded-md border border-line bg-subtle">
      <button className="w-7 h-7 grid place-items-center text-fg2 hover:text-fg disabled:opacity-40" disabled={v <= min} onClick={() => on(v - 1)} aria-label="Fewer"><Minus size={12} /></button>
      <span className="w-6 text-center tnum font-semibold">{v}</span>
      <button className="w-7 h-7 grid place-items-center text-fg2 hover:text-fg disabled:opacity-40" disabled={v >= max} onClick={() => on(v + 1)} aria-label="More"><Plus size={12} /></button>
    </div>
  );
}

function TopologyStep({ c, set }: { c: DeployCfg; set: SetFn }) {
  const k8s = c.target === "cnpg";
  return (
    <>
      <Card title="Topology" sub={`${1 + c.replicas} PostgreSQL node${c.replicas ? "s" : ""}`}><TopologyDiagram c={c} /></Card>
      <Card title="Components">
        <div className="grid grid-cols-2 gap-x-6 gap-y-4">
          <Field label="Replicas" hint={c.replicas === 0 ? "No standby: no failover." : `${c.replicas} streaming standby${c.replicas > 1 ? "s" : ""}`}><Stepr v={c.replicas} min={0} max={4} on={(n) => set("replicas", n)} /></Field>
          <Field label={k8s ? "Consensus" : "etcd members"} hint={k8s ? "The operator uses the Kubernetes API; no etcd to run." : c.etcd === 1 ? "No quorum tolerance — dev only." : `Tolerates ${Math.floor((c.etcd - 1) / 2)} member failure${c.etcd > 3 ? "s" : ""}`}>
            {k8s ? <div className="text-fg2 text-[12.5px] h-7 flex items-center">Kubernetes API</div> : <Segmented value={String(c.etcd)} onChange={(v) => set("etcd", +v)} options={[{ value: "1", label: "1" }, { value: "3", label: "3" }, { value: "5", label: "5" }]} />}
          </Field>
          <Field label="Client entry point">
            {k8s ? <div className="text-fg2 text-[12.5px] h-7 flex items-center mono">Service {c.name}-rw / -ro</div>
              : <Segmented value={c.entry} onChange={(v) => set("entry", v)} options={[{ value: "haproxy", label: "HAProxy" }, ...(c.target === "vms" ? [{ value: "vip" as const, label: "vip-manager" }] : [])]} />}
          </Field>
          <Field label="Backup repository">
            <select className="input w-full" value={c.repo} onChange={(e) => set("repo", e.target.value as DeployCfg["repo"])}>
              {!k8s && <option value="volume">{c.target === "compose" ? "Docker volume (pgbackrest-repo)" : "Dedicated repo host (posix)"}</option>}
              <option value="s3">Amazon S3 / MinIO</option><option value="gcs">Google Cloud Storage</option><option value="azure">Azure Blob</option>
            </select>
          </Field>
          <Toggle on={c.pgbouncer} onChange={(b) => set("pgbouncer", b)} label={k8s ? "CNPG Pooler (PgBouncer, transaction mode)" : "PgBouncer (transaction pooling)"} />
          <Toggle on={c.sync} onChange={(b) => set("sync", b)} label="Synchronous replication (zero data loss, higher latency)" />
        </div>
        {c.target === "compose" && c.replicas > 0 && <div className="mt-4 text-[12px] flex gap-1.5 rounded-md p-2.5" style={{ background: tint("var(--warn)", 10), color: "var(--warn)" }}><AlertTriangle size={13} className="mt-px shrink-0" /> All {1 + c.replicas} nodes run on one Docker host. Replicas protect against a crashed container, not a lost host.</div>}
      </Card>
    </>
  );
}

function TopologyDiagram({ c }: { c: DeployCfg }) {
  const n = 1 + c.replicas; const W = 560; const nodeW = 96; const gap = 14;
  const rowW = n * nodeW + (n - 1) * gap; const x0 = (W - 120 - rowW) / 2;
  const box = (x: number, y: number, w: number, label: string, sub: string, color: string, key: string, dashed?: boolean) => (
    <g key={key} className="animate-fadeIn">
      <rect x={x} y={y} width={w} height={40} rx={8} fill="var(--subtle)" stroke={color} strokeWidth={1.25} strokeDasharray={dashed ? "4 3" : undefined} />
      <text x={x + w / 2} y={y + 17} textAnchor="middle" fontSize="11.5" fontWeight="600" fill="var(--fg)">{label}</text>
      <text x={x + w / 2} y={y + 31} textAnchor="middle" fontSize="10" fill="var(--fg3)" className="mono">{sub}</text>
    </g>
  );
  const cx = x0 + rowW / 2; const pgY = c.pgbouncer ? 132 : 96;
  const entry = c.target === "cnpg" ? "Service" : c.entry === "vip" ? "vip-manager" : "HAProxy";
  return (
    <svg viewBox={`0 0 ${W} ${c.target === "cnpg" ? pgY + 60 : pgY + 110}`} className="w-full">
      {box(cx - 70, 8, 140, entry, c.target === "cnpg" ? `${c.name}-rw` : c.entry === "vip" ? "10.0.1.100" : ":5000 rw · :5001 ro", "var(--line2)", "e")}
      {c.pgbouncer && box(cx - 70, 70, 140, "PgBouncer", c.target === "cnpg" ? "Pooler ×2" : ":6432", "var(--line2)", "b")}
      <line x1={cx} y1={48} x2={cx} y2={c.pgbouncer ? 70 : pgY} stroke="var(--line2)" />
      {c.pgbouncer && <line x1={cx} y1={110} x2={cx} y2={pgY - 10} stroke="var(--line2)" />}
      <line x1={x0 + nodeW / 2} y1={pgY - 10} x2={x0 + rowW - nodeW / 2} y2={pgY - 10} stroke="var(--line2)" />
      {Array.from({ length: n }, (_, i) => {
        const x = x0 + i * (nodeW + gap);
        return (
          <g key={i}>
            <line x1={x + nodeW / 2} y1={pgY - 10} x2={x + nodeW / 2} y2={pgY} stroke="var(--line2)" />
            {i > 0 && <line x1={x - gap} y1={pgY + 20} x2={x} y2={pgY + 20} stroke="var(--accent)" strokeWidth={1.25} className="flow-line" />}
            {box(x, pgY, nodeW, i === 0 ? "primary" : `replica ${i}`, `${c.name.slice(0, 9)}-${i + 1}`, i === 0 ? "var(--accent)" : "var(--line2)", `n${i}`)}
            {i === 1 && c.sync && <text x={x + nodeW / 2} y={pgY + 54} textAnchor="middle" fontSize="9.5" fill="var(--accent)">sync</text>}
          </g>
        );
      })}
      {box(W - 112, pgY, 104, c.target === "cnpg" ? "Barman Cloud" : "pgBackRest", c.repo === "volume" ? "local repo" : c.repo.toUpperCase(), "var(--ok)", "r", true)}
      <line x1={x0 + rowW} y1={pgY + 20} x2={W - 112} y2={pgY + 20} stroke="var(--ok)" strokeDasharray="3 4" />
      {c.target !== "cnpg" && Array.from({ length: c.etcd }, (_, i) => {
        const ew = 70, eg = 10, ex0 = cx - (c.etcd * ew + (c.etcd - 1) * eg) / 2;
        return box(ex0 + i * (ew + eg), pgY + 64, ew, `etcd${i + 1}`, "DCS", "var(--c4)", `et${i}`);
      })}
    </svg>
  );
}

function SizingStep({ c, set }: { c: DeployCfg; set: SetFn }) {
  return (
    <>
      <Card title="Size per node">
        <div className="grid grid-cols-4 gap-2">
          {(Object.keys(SIZES) as (keyof typeof SIZES)[]).map((k) => {
            const s = SIZES[k]; const on = c.size === k;
            return (
              <button key={k} onClick={() => set("size", k)} className="rounded-lg border p-3 text-left transition-all" style={on ? { borderColor: "var(--accent)", background: tint("var(--accent)", 8) } : { borderColor: "var(--line)" }}>
                <div className="font-semibold">{s.label}</div>
                <div className="text-fg3 text-[11.5px] mt-1 tnum">{s.cpu} vCPU · {s.ram} GB</div>
                <div className="text-fg3 text-[11.5px] tnum">{s.disk >= 1000 ? `${s.disk / 1000} TB` : `${s.disk} GB`} · sb {s.sb}</div>
              </button>
            );
          })}
        </div>
      </Card>
      <Card title="PostgreSQL">
        <Field label="Major version"><Segmented value={c.pg} onChange={(v) => set("pg", v)} options={["14", "15", "16", "17", "18"].map((v) => ({ value: v, label: v }))} /></Field>
        <div className="text-[12px] text-fg2 font-medium mt-4 mb-1.5">Extensions</div>
        <div className="grid grid-cols-2 gap-1.5">
          {DEPLOY_EXTS.map((e) => {
            const on = c.exts.includes(e); const locked = e === "pg_stat_statements";
            return (
              <label key={e} className={`flex items-center gap-2 px-2.5 h-8 rounded-md border text-[12.5px] ${locked ? "opacity-80" : "cursor-pointer hover:bg-hover/50"}`} style={{ borderColor: on ? tint("var(--accent)", 40) : "var(--line)" }}>
                <input type="checkbox" className="accent-[var(--brand)]" checked={on} disabled={locked} onChange={() => set("exts", on ? c.exts.filter((x) => x !== e) : [...c.exts, e])} />
                <span className="mono">{e}</span>{locked && <span className="ml-auto text-[10.5px] text-fg3">required</span>}
              </label>
            );
          })}
        </div>
      </Card>
      <Card title="Backups" sub={c.target === "cnpg" ? "Barman Cloud plugin" : "pgBackRest"}>
        <div className="grid grid-cols-3 gap-3">
          <Field label="Full backup"><select className="input w-full" value={c.fullCron} onChange={(e) => set("fullCron", e.target.value)}><option>Sun 01:00</option><option>daily 01:00</option><option>Sat 23:00</option></select></Field>
          <Field label={c.target === "cnpg" ? "Base backups" : "Differential"}><select className="input w-full" value={c.diffCron} onChange={(e) => set("diffCron", e.target.value)}><option>daily 01:00</option><option>every 6 h</option><option>none</option></select></Field>
          <Field label="Retention"><select className="input w-full" value={c.retentionDays} onChange={(e) => set("retentionDays", +e.target.value)}>{[7, 14, 30, 90].map((d) => <option key={d} value={d}>{d} days</option>)}</select></Field>
        </div>
        <div className="text-[11.5px] text-fg3 mt-2">Continuous WAL archiving is always on, so point-in-time recovery works across the whole retention window. Times are Europe/London.</div>
      </Card>
    </>
  );
}

function ReviewStep({ c, files }: { c: DeployCfg; files: { id: string; label: string; text: string; file?: string }[] }) {
  const pre = useMemo(() => [
    { l: c.target === "compose" ? "Docker engine reachable" : c.target === "cnpg" ? "CNPG operator installed" : "SSH to all hosts", d: c.target === "compose" ? "Docker 27.3 · compose v2.29 (demo)" : c.target === "cnpg" ? "cnpg-system/cnpg-controller-manager 1.25 (demo)" : c.hosts, r: "ok" as CheckState },
    { l: "Capacity", d: `needs ${(1 + c.replicas) * SIZES[c.size].cpu} vCPU · ${(1 + c.replicas) * SIZES[c.size].ram} GB · ${(1 + c.replicas) * SIZES[c.size].disk} GB disk`, r: (c.target === "compose" && c.size === "large" ? "warn" : "ok") as CheckState },
    { l: "Ports / network", d: c.target === "cnpg" ? "namespace databases, NetworkPolicy allows 5432 from apps" : "5432, 5000-5001, 6432, 2379-2380, 8008 free", r: "ok" as CheckState },
    { l: "Backup repository", d: c.repo === "volume" ? "local volume — not off-host; add S3 for real DR" : `${c.repo.toUpperCase()} bucket demo-backups writable (placeholder credentials)`, r: (c.repo === "volume" ? "warn" : "ok") as CheckState },
    { l: "Environment guard", d: c.env === "prod" ? "production: second approver required (demo: auto-approved)" : `${c.env}: no approval needed`, r: (c.env === "prod" ? "warn" : "ok") as CheckState },
  ], [c]);
  const [i, setI] = useState(0);
  useEffect(() => { if (i < pre.length) { const t = setTimeout(() => setI(i + 1), 380); return () => clearTimeout(t); } }, [i, pre.length]);
  const plan: [string, string][] = [
    ["PostgreSQL nodes", `${1 + c.replicas} × PG ${c.pg} · ${SIZES[c.size].label} (${SIZES[c.size].cpu} vCPU / ${SIZES[c.size].ram} GB)`],
    ["Failover", c.target === "cnpg" ? "CloudNativePG operator" : `Patroni + etcd ×${c.etcd}`],
    ["Entry point", c.target === "cnpg" ? `Services ${c.name}-rw / -ro` : c.entry === "vip" ? "vip-manager 10.0.1.100" : "HAProxy :5000 rw / :5001 ro"],
    ["Pooling", c.pgbouncer ? "PgBouncer, transaction mode" : "none"],
    ["Backups", `${c.target === "cnpg" ? "Barman Cloud" : "pgBackRest"} → ${c.repo === "volume" ? "local" : c.repo.toUpperCase()} · full ${c.fullCron} · ${c.retentionDays} d`],
    ["Extensions", c.exts.join(", ")],
  ];
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-4 items-start">
      <div className="space-y-4">
        <Card title="Plan" sub={<span className="flex items-center gap-1.5"><span className="mono">{c.name}</span> · {TARGET_LABEL[c.target]}</span>} actions={<Badge color={ENVS.find((e) => e.id === c.env)!.color} className="uppercase">{c.env}</Badge>}>
          <dl className="divide-y divide-line/70">{plan.map(([k, v]) => <div key={k} className="flex gap-4 py-2"><dt className="w-32 text-fg3 shrink-0">{k}</dt><dd className="min-w-0">{v}</dd></div>)}</dl>
          <div className="mt-3 rounded-md bg-subtle border border-line px-3 py-2 mono text-[11.5px] text-fg2">Plan: <span className="text-ok">+{(1 + c.replicas) + (c.target === "cnpg" ? 2 : c.etcd + 2) + (c.pgbouncer ? 1 : 0)} to add</span>, 0 to change, 0 to destroy</div>
        </Card>
        <Card title="Pre-flight checks">{pre.map((p, k) => <CheckRow key={p.l} state={k < i ? p.r : k === i ? "running" : "pending"} label={p.l} detail={p.d} />)}</Card>
      </div>
      <Card title="Files to apply"><CodeTabs tabs={files} maxH={520} /></Card>
    </div>
  );
}

function RunStep({ c }: { c: DeployCfg }) {
  const { addCluster, upsertDeployment, setClusterId, go, toast, deployments } = useApp();
  const log = useMemo(() => deployLog(c), [c]);
  const phases = useMemo(() => Array.from(new Set(log.map((l) => l.step))), [log]);
  const [n, setN] = useState(0);
  const [fast, setFast] = useState(false);
  const depId = useRef(`dep-${String(20 + deployments.length - 6).padStart(4, "0")}`);
  const start = useRef(new Date());
  const ref = useRef<HTMLDivElement>(null);
  const done = n >= log.length;
  const id = c.name;
  useEffect(() => {
    upsertDeployment({ id: depId.current, name: c.name, env: c.env, target: c.target, status: "running", pg: c.pg, topology: `${1 + c.replicas === 1 ? "1P" : `1P + ${c.replicas}R`}${c.target !== "cnpg" ? ` · etcd ×${c.etcd}` : ""}${c.pgbouncer ? " · PgBouncer" : ""}`, startedAt: fmtNow(start.current), durationS: 0, by: "Mark Alford" });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (done) return;
    const t = setTimeout(() => setN((x) => x + 1), fast ? 40 : 230 + ((n * 37) % 160));
    return () => clearTimeout(t);
  }, [n, done, fast]);
  useEffect(() => { ref.current?.scrollTo({ top: ref.current.scrollHeight }); }, [n]);
  useEffect(() => {
    if (!done) return;
    const cl: Cluster = { id, name: id, env: c.env, health: "ok", score: 100, version: `${c.pg}.0`, ha: c.target === "cnpg" ? "CloudNativePG" : c.replicas ? "Patroni" : "Standalone", provider: c.target === "cnpg" ? "k8s" : "Local", region: c.target === "compose" ? "docker · demo-host" : c.target === "cnpg" ? "demo-gke · databases" : "dc1 · 10.0.1.0/24", primary: 1, replicas: c.replicas, tps: 0, conns: 6, maxConns: 300, lagBytes: 0, lagSec: 0, backupAgeH: 0, advisories: 0, storageGb: 2, storagePct: 1, load: series(id.length * 13, 40, 0.3, 0.2), databases: ["postgres"] };
    addCluster(cl);
    const dur = Math.max(1, Math.round((Date.now() - start.current.getTime()) / 1000));
    upsertDeployment({ id: depId.current, name: c.name, env: c.env, target: c.target, status: "succeeded", pg: c.pg, topology: `${c.replicas ? `1P + ${c.replicas}R` : "1P"}${c.target !== "cnpg" ? ` · etcd ×${c.etcd}` : ""}${c.pgbouncer ? " · PgBouncer" : ""}`, startedAt: fmtNow(start.current), durationS: dur, by: "Mark Alford", log: log.map((l) => `[${l.step}] ${l.line}`) });
    toast({ title: `${id} deployed (simulated)`, body: `Added to Fleet · ${TARGET_LABEL[c.target]}`, tone: "ok" });
  }, [done]); // eslint-disable-line react-hooks/exhaustive-deps
  const curPhase = done ? phases.length : phases.indexOf(log[Math.min(n, log.length - 1)].step);
  const pct = Math.round((n / log.length) * 100);
  return (
    <div className="grid grid-cols-[280px_minmax(0,1fr)] gap-4 items-start">
      <Card title="Progress" sub={`${pct}%`}>
        <div className="h-1.5 rounded-full bg-subtle overflow-hidden mb-4"><div className="h-full rounded-full transition-all duration-300" style={{ width: `${pct}%`, background: done ? "var(--ok)" : "var(--brand)" }} /></div>
        <ol className="space-y-1">
          {phases.map((p, i) => (
            <li key={p} className={`flex items-center gap-2.5 h-7 text-[12.5px] ${i > curPhase ? "text-fg3" : ""}`}>
              <span className="w-4 grid place-items-center">{i < curPhase ? <Check size={13} className="text-ok" strokeWidth={3} /> : i === curPhase ? <Loader2 size={13} className="spin text-accent" /> : <span className="w-1.5 h-1.5 rounded-full bg-line2" />}</span>
              {p}
            </li>
          ))}
        </ol>
        {!done && <button className="btn btn-ghost w-full justify-center mt-3 text-[11.5px]" onClick={() => setFast(true)} disabled={fast}>{fast ? "Fast-forwarding…" : "Fast-forward (demo)"}</button>}
      </Card>
      <div className="space-y-3">
        <section className="card overflow-hidden">
          <header className="flex items-center gap-2 px-4 h-10 border-b border-line">
            {done ? <CheckCircle2 size={14} className="text-ok" /> : <Loader2 size={14} className="spin text-accent" />}
            <span className="font-semibold text-[13px]">{done ? "Deployment complete" : "Deploying"}</span>
            <span className="mono text-fg3 text-[12px]">{depId.current} · tl deploy apply {c.name}</span>
            <span className="ml-auto"><DemoTag /></span>
          </header>
          <div ref={ref} className="mono text-[11.5px] leading-[1.7] p-3 h-[420px] overflow-auto bg-subtle">
            {log.slice(0, n).map((l, i) => (
              <div key={i} className="log-line flex gap-3">
                <span className="text-fg3 tnum shrink-0">{fmtClock(new Date(start.current.getTime() + i * 1700))}</span>
                <span className="w-[120px] shrink-0 truncate" style={{ color: "var(--accent)" }}>{l.step}</span>
                <span className={l.line.includes("complete") ? "text-ok" : "text-fg2"}>{l.line}</span>
              </div>
            ))}
            {!done && <div className="flex gap-3"><span className="text-fg3">…</span><span className="w-2 h-4 bg-fg3/60 animate-pulse" /></div>}
          </div>
        </section>
        {done && (
          <div className="card p-4 flex items-center gap-3 animate-pop" style={{ borderColor: tint("var(--ok)", 40) }}>
            <div className="w-9 h-9 rounded-lg grid place-items-center" style={{ background: tint("var(--ok)", 15), color: "var(--ok)" }}><Database size={17} /></div>
            <div className="min-w-0 flex-1"><div className="font-semibold"><span className="mono">{id}</span> is ready</div><div className="text-fg3 text-[12px]">Registered in Fleet with monitoring enabled. Connection: <span className="mono">postgres://app@{c.target === "cnpg" ? `${id}-rw.databases` : "localhost:5000"}/postgres</span></div></div>
            <button className="btn" onClick={() => go("deployments")}><HardDrive size={13} /> History</button>
            <button className="btn" onClick={() => go("fleet")}>View in Fleet</button>
            <button className="btn btn-primary" onClick={() => { setClusterId(id); go("cluster"); }}>Open cluster <ArrowRight size={13} /></button>
          </div>
        )}
      </div>
    </div>
  );
}
const pad = (x: number) => String(x).padStart(2, "0");
const fmtClock = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
const fmtNow = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
