import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, ArrowRightLeft, CheckCircle2, Circle, Cpu, Crown, Database, HardDrive, Loader2, RefreshCw, Server, Users, XCircle } from "lucide-react";
import { useApp } from "../store";
import { Badge, Card, Dot, EnvBadge, Modal, PageHeader, fmtBytes, tint } from "../components/ui";

interface Node { id: string; role: "leader" | "sync" | "async"; az: string; lagBytes: number; lagSec: number; cpu: number; conns: number; tl: number; lsn: string }

function buildNodes(prefix: string, replicas: number, laggy: boolean, version: number): Node[] {
  const azs = ["a", "b", "c"];
  const out: Node[] = [{ id: `${prefix}-1`, role: "leader", az: azs[0], lagBytes: 0, lagSec: 0, cpu: 62, conns: 212, tl: 7 + version, lsn: "4A/9C3F21D8" }];
  for (let i = 0; i < replicas; i++) {
    const bad = laggy && i === replicas - 1;
    out.push({ id: `${prefix}-${i + 2}`, role: i === 0 ? "sync" : "async", az: azs[(i + 1) % 3], lagBytes: bad ? 48_200_000 : 180_000 + i * 90_000, lagSec: bad ? 6.4 : 0.1 + i * 0.1, cpu: bad ? 91 : 34 + i * 8, conns: 40 + i * 22, tl: 7 + version, lsn: bad ? "4A/96E0A000" : "4A/9C3F1F00" });
  }
  return out;
}

const W = 1100, H = 420;
const pct = (v: number, of: number) => `${(v / of) * 100}%`;

function NodeCard({ n, selected, onClick, leaderName }: { n: Node; selected: boolean; onClick: () => void; leaderName: string }) {
  const lagColor = n.lagSec > 5 ? "var(--warn)" : "var(--ok)";
  const isLeader = n.role === "leader";
  return (
    <button onClick={onClick} className={`w-full text-left card p-3 transition-all focus-ring ${selected ? "ring-2 ring-accent" : "hover:border-line2"}`}
      style={{ boxShadow: isLeader ? `0 0 0 1px ${tint("var(--accent)", 40)}, 0 8px 30px ${tint("var(--accent)", 15)}` : undefined }}>
      <div className="flex items-center gap-2">
        {isLeader ? <Crown size={14} style={{ color: "var(--c2)" }} /> : <Server size={13} className="text-fg3" />}
        <span className="mono font-medium text-[12.5px]">{n.id}</span>
        <span className="ml-auto"><Badge color={isLeader ? "var(--accent)" : n.role === "sync" ? "var(--ok)" : "var(--fg3)"}>{isLeader ? "Leader" : n.role === "sync" ? "Sync standby" : "Async replica"}</Badge></span>
      </div>
      <div className="flex items-center gap-3 mt-2 text-[11px] text-fg3">
        <span>AZ {n.az}</span><span className="mono">TL {n.tl}</span><span className="mono">{n.lsn}</span>
      </div>
      {isLeader ? (
        <div className="grid grid-cols-2 gap-2 mt-2.5 text-[11.5px]">
          <div className="flex items-center gap-1.5"><Cpu size={11} className="text-fg3" /> <span className="tnum">{n.cpu}%</span> CPU</div>
          <div className="flex items-center gap-1.5"><Users size={11} className="text-fg3" /> <span className="tnum">{n.conns}</span> conns</div>
        </div>
      ) : (
        <div className="mt-2.5">
          <div className="flex justify-between text-[11.5px] mb-1"><span className="text-fg3">Replay lag from {leaderName.split("-").pop()}</span><span className="tnum font-medium" style={{ color: lagColor }}>{n.lagSec}s · {fmtBytes(n.lagBytes)}</span></div>
          <div className="h-1.5 rounded-full bg-subtle overflow-hidden"><div className="h-full rounded-full transition-all" style={{ width: `${Math.min(100, n.lagSec * 12 + 3)}%`, background: lagColor }} /></div>
        </div>
      )}
    </button>
  );
}

export default function Ha() {
  const { cluster: c, toast } = useApp();
  const prefix = c.id.startsWith("prod-eu-west") ? "pg-euw" : c.id.split("-").slice(0, 2).join("-").replace("prod", "pg").replace("staging", "stg").replace("dev", "dev");
  const initial = useMemo(() => buildNodes(prefix, c.replicas, c.lagSec > 5, 0), [prefix, c.replicas, c.lagSec]);
  const [nodes, setNodes] = useState<Node[]>(initial);
  const [sel, setSel] = useState(initial[0].id);
  const [open, setOpen] = useState(false);
  const [history, setHistory] = useState([
    { t: "14 Oct 2026 03:12 BST", kind: "Switchover", from: `${prefix}-2`, to: `${prefix}-1`, by: "mark.a", dur: "3.8 s" },
    { t: "29 Aug 2026 22:47 BST", kind: "Failover", from: `${prefix}-1`, to: `${prefix}-2`, by: "Patroni (automatic)", dur: "11.2 s" },
  ]);
  useEffect(() => { setNodes(initial); setSel(initial[0].id); }, [initial]);
  useEffect(() => { const on = () => setOpen(true); window.addEventListener("tl:switchover", on); return () => window.removeEventListener("tl:switchover", on); }, []);

  const leader = nodes.find((n) => n.role === "leader")!;
  const reps = nodes.filter((n) => n.role !== "leader");
  const repY = reps.length === 1 ? [210] : reps.length === 2 ? [105, 315] : [];
  const P = { x: 440, w: 250, y: 210 }, B = { x: 210, w: 160, y: 210 }, R = { x: 800, w: 290 };
  const engine = c.ha === "CloudNativePG" ? "CloudNativePG operator" : c.ha === "Patroni" ? "etcd · 3 members · quorum OK" : c.ha;
  const selected = nodes.find((n) => n.id === sel) ?? leader;

  if (c.replicas === 0) {
    return (
      <div>
        <PageHeader title="HA topology" sub={<span className="flex items-center gap-2"><EnvBadge env={c.env} /><span className="mono">{c.name}</span> · standalone</span>} />
        <Card><div className="py-14 text-center"><div className="w-11 h-11 rounded-xl mx-auto grid place-items-center bg-subtle border border-line text-fg3 mb-3"><Database size={18} /></div><div className="font-medium">No replicas configured</div><div className="text-fg3 mt-1 text-[12.5px]">This cluster is a single instance. Add a streaming replica or adopt it into Patroni or CloudNativePG.</div><div className="mt-4 flex justify-center gap-2"><button className="btn">Read the HA guide</button><button className="btn btn-primary">Add replica…</button></div></div></Card>
      </div>
    );
  }

  return (
    <div>
      <PageHeader title="HA topology" sub={<span className="flex items-center gap-2"><EnvBadge env={c.env} /><span className="mono">{c.name}</span> · {c.ha} · {nodes.length} members · timeline {leader.tl}</span>}
        actions={<><button className="btn" onClick={() => toast({ title: "Topology refreshed", body: `${c.ha} REST API polled 0.3 s ago`, tone: "ok" })}><RefreshCw size={13} /> Refresh</button><button className="btn btn-primary" onClick={() => setOpen(true)}><ArrowRightLeft size={13} /> Switchover…</button></>} />

      <Card pad={false} className="mb-4 overflow-hidden">
        <div className="relative w-full" style={{ aspectRatio: `${W}/${H}`, backgroundImage: "radial-gradient(var(--line) 1px, transparent 1px)", backgroundSize: "18px 18px" }}>
          <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="absolute inset-0 w-full h-full">
            <path d={`M150,${B.y} L${B.x},${B.y}`} stroke="var(--line2)" strokeWidth={1.5} fill="none" vectorEffect="non-scaling-stroke" />
            <path d={`M${B.x + B.w},${B.y} L${P.x},${P.y}`} stroke="var(--accent)" strokeWidth={2} fill="none" className="flow-line" vectorEffect="non-scaling-stroke" />
            <path d={`M${P.x + P.w / 2},60 L${P.x + P.w / 2},${P.y - 50}`} stroke="var(--line2)" strokeWidth={1.5} strokeDasharray="3 4" fill="none" vectorEffect="non-scaling-stroke" />
            <path d={`M${P.x + P.w / 2},${P.y + 50} L${P.x + P.w / 2},350`} stroke="var(--c4)" strokeWidth={1.5} fill="none" className="flow-line" vectorEffect="non-scaling-stroke" />
            {reps.map((r, i) => {
              const y = repY[i]; const color = r.lagSec > 5 ? "var(--warn)" : r.role === "sync" ? "var(--ok)" : "var(--fg3)";
              return <path key={r.id} d={`M${P.x + P.w},${P.y} C${P.x + P.w + 60},${P.y} ${R.x - 60},${y} ${R.x},${y}`} stroke={color} strokeWidth={2} fill="none" className="flow-line" vectorEffect="non-scaling-stroke" />;
            })}
          </svg>
          {/* clients */}
          <div className="absolute -translate-y-1/2" style={{ left: pct(20, W), top: pct(B.y, H), width: pct(130, W) }}>
            <div className="card px-3 py-2.5 text-[12px]"><div className="flex items-center gap-1.5 font-medium"><Users size={12} className="text-fg3" />Clients</div><div className="text-fg3 text-[11px] mt-0.5">orders-api, billing, sync</div></div>
          </div>
          <div className="absolute -translate-y-1/2" style={{ left: pct(B.x, W), top: pct(B.y, H), width: pct(B.w, W) }}>
            <div className="card px-3 py-2.5 text-[12px]"><div className="font-medium">PgBouncer</div><div className="text-fg3 text-[11px] mt-0.5">transaction pool · 2 pods</div><div className="text-[11px] mt-1 tnum">312 clients → 60 servers</div></div>
          </div>
          <div className="absolute -translate-x-1/2" style={{ left: pct(P.x + P.w / 2, W), top: pct(18, H) }}>
            <div className="flex items-center gap-1.5 px-2.5 h-7 rounded-full border border-line bg-elev text-[11.5px] text-fg2 whitespace-nowrap"><Dot color="var(--ok)" size={6} /> {engine}</div>
          </div>
          <div className="absolute -translate-y-1/2" style={{ left: pct(P.x, W), top: pct(P.y, H), width: pct(P.w, W) }}>
            <NodeCard n={leader} selected={sel === leader.id} onClick={() => setSel(leader.id)} leaderName={leader.id} />
          </div>
          {reps.map((r, i) => (
            <div key={r.id} className="absolute -translate-y-1/2" style={{ left: pct(R.x, W), top: pct(repY[i], H), width: pct(R.w, W) }}>
              <NodeCard n={r} selected={sel === r.id} onClick={() => setSel(r.id)} leaderName={leader.id} />
            </div>
          ))}
          <div className="absolute -translate-x-1/2" style={{ left: pct(P.x + P.w / 2, W), top: pct(352, H) }}>
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg border border-line bg-elev text-[11.5px] whitespace-nowrap"><HardDrive size={12} style={{ color: "var(--c4)" }} /> pgBackRest repo1 · s3 <span className="text-fg3">archive-push lag 2 s</span></div>
          </div>
          <div className="absolute bottom-3 left-4 flex items-center gap-3 text-[11px] text-fg3">
            <span className="flex items-center gap-1"><span className="w-3 h-0.5" style={{ background: "var(--ok)" }} /> sync</span>
            <span className="flex items-center gap-1"><span className="w-3 h-0.5" style={{ background: "var(--fg3)" }} /> async</span>
            <span className="flex items-center gap-1"><span className="w-3 h-0.5" style={{ background: "var(--warn)" }} /> lagging</span>
            <span className="flex items-center gap-1"><span className="w-3 h-0.5" style={{ background: "var(--c4)" }} /> WAL archive</span>
          </div>
        </div>
      </Card>

      <div className="grid grid-cols-3 gap-4">
        <Card title={<span className="mono">{selected.id}</span>} sub={selected.role === "leader" ? "leader" : "replica"}>
          <dl className="grid grid-cols-2 gap-y-2 text-[12px]">
            {[["Host", `${selected.id}.internal:5432`], ["Availability zone", `${c.region}${selected.az}`], ["Timeline", selected.tl], ["Current LSN", selected.lsn], ["Lag (bytes)", fmtBytes(selected.lagBytes)], ["Lag (seconds)", `${selected.lagSec}s`], ["CPU", `${selected.cpu}%`], ["Connections", selected.conns]].map(([k, v]) => (
              <div key={k as string} className="contents"><dt className="text-fg3">{k}</dt><dd className="mono text-right">{v}</dd></div>
            ))}
          </dl>
        </Card>
        <Card title="Failover policy" sub={c.ha}>
          <dl className="space-y-2 text-[12px]">
            {[["Synchronous mode", "on (1 standby)"], ["maximum_lag_on_failover", "1 MB"], ["ttl / loop_wait", "30 s / 10 s"], ["Fencing", "watchdog + pg_rewind"], ["Last DCS heartbeat", "2 s ago"]].map(([k, v]) => <div key={k} className="flex justify-between"><span className="text-fg3">{k}</span><span className="mono">{v}</span></div>)}
          </dl>
        </Card>
        <Card title="Role changes" sub="last 90 days" pad={false}>
          <div className="divide-y divide-line">
            {history.map((h, i) => (
              <div key={i} className="px-4 py-2.5 text-[12px] flex items-center gap-2.5">
                <Badge color={h.kind === "Failover" ? "var(--warn)" : "var(--accent)"}>{h.kind}</Badge>
                <span className="mono text-[11.5px]">{h.from.split("-").pop()} → {h.to.split("-").pop()}</span>
                <span className="text-fg3 truncate">{h.by}</span>
                <span className="ml-auto text-fg3 text-[11px] whitespace-nowrap">{h.t.split(" ").slice(0, 2).join(" ")} · {h.dur}</span>
              </div>
            ))}
          </div>
        </Card>
      </div>

      <SwitchoverModal open={open} onClose={() => setOpen(false)} nodes={nodes} clusterName={c.name} isProd={c.env === "prod"}
        onDone={(to) => {
          const from = leader.id;
          setNodes((ns) => ns.map((n) => (n.id === to ? { ...n, role: "leader", lagBytes: 0, lagSec: 0, tl: n.tl + 1, cpu: 58, conns: 205 } : n.id === from ? { ...n, role: "sync", lagBytes: 120_000, lagSec: 0.1, tl: n.tl + 1, cpu: 31, conns: 38 } : { ...n, tl: n.tl + 1 })));
          setSel(to);
          setHistory((h) => [{ t: "27 Oct 2026 12:00 GMT", kind: "Switchover", from, to, by: "mark.a", dur: "3.1 s" }, ...h]);
          toast({ title: `Switchover complete: ${to} is now leader`, body: `Timeline ${leader.tl} → ${leader.tl + 1} · 3.1 s write unavailability · PgBouncer resumed`, tone: "ok" });
        }} />
    </div>
  );
}

function SwitchoverModal({ open, onClose, nodes, clusterName, isProd, onDone }: { open: boolean; onClose: () => void; nodes: Node[]; clusterName: string; isProd: boolean; onDone: (to: string) => void }) {
  const reps = nodes.filter((n) => n.role !== "leader");
  const [cand, setCand] = useState(reps[0]?.id);
  const [checked, setChecked] = useState(0);
  const [confirm, setConfirm] = useState("");
  const [phase, setPhase] = useState<"plan" | "running">("plan");
  const [step, setStep] = useState(0);
  const candNode = reps.find((r) => r.id === cand) ?? reps[0];
  useEffect(() => { if (open) { setCand(reps.find((r) => r.role === "sync")?.id ?? reps[0]?.id); setConfirm(""); setPhase("plan"); setStep(0); } }, [open]); // eslint-disable-line
  useEffect(() => { if (!open || phase !== "plan") return; setChecked(0); const t = setInterval(() => setChecked((x) => (x >= 6 ? x : x + 1)), 220); return () => clearInterval(t); }, [open, cand, phase]);
  if (!candNode) return null;
  const lagOk = candNode.lagBytes < 1_000_000;
  const checks: { label: string; detail: string; ok: boolean; warn?: boolean }[] = [
    { label: "Candidate replication lag under 1 MB", detail: `${fmtBytes(candNode.lagBytes)} · ${candNode.lagSec}s`, ok: lagOk },
    { label: "No transactions older than 5 minutes", detail: "oldest: 4m 12s (pid 48211, idle in txn) — will be terminated", ok: true, warn: true },
    { label: "DCS quorum healthy", detail: "etcd 3/3 members", ok: true },
    { label: "Candidate on same timeline", detail: `TL ${candNode.tl}`, ok: true },
    { label: "WAL archiving current", detail: "last archived 2 s ago", ok: true },
    { label: "Recent verified backup", detail: "incr 5 h ago · verified", ok: true },
  ];
  const allOk = checks.every((c) => c.ok);
  const steps = ["PAUSE PgBouncer pools", "CHECKPOINT on leader", "Demote leader (patronictl switchover)", `Promote ${candNode.id}`, "Repoint PgBouncer · RESUME", "Rewind old leader as standby"];
  const go = () => {
    setPhase("running");
    let s = 0;
    const t = setInterval(() => { s++; setStep(s); if (s >= steps.length) { clearInterval(t); setTimeout(() => { onDone(candNode.id); onClose(); }, 500); } }, 480);
  };
  const canGo = allOk && checked >= 6 && (!isProd || confirm === clusterName);
  return (
    <Modal open={open} onClose={phase === "running" ? () => {} : onClose} tone={isProd ? "danger" : undefined} width={600} title={<span className="flex items-center gap-2"><ArrowRightLeft size={15} /> Guarded switchover · <span className="mono">{clusterName}</span></span>}
      footer={phase === "plan" ? <><span className="mr-auto text-[11.5px] text-fg3 self-center mono">tl ha switchover --to {candNode.id}</span><button className="btn" onClick={onClose}>Cancel</button><button className="btn btn-danger" disabled={!canGo} onClick={go}>Switch over now</button></> : undefined}>
      {phase === "plan" ? (
        <div className="space-y-4">
          <div>
            <div className="label mb-2">Promote candidate</div>
            <div className="space-y-1.5">
              {reps.map((r) => (
                <label key={r.id} className={`flex items-center gap-3 p-2.5 rounded-lg border cursor-pointer transition-colors ${cand === r.id ? "border-accent bg-hover" : "border-line hover:bg-hover/50"}`}>
                  <input type="radio" className="accent-[var(--accent)]" checked={cand === r.id} onChange={() => setCand(r.id)} />
                  <span className="mono text-[12.5px]">{r.id}</span>
                  <Badge color={r.role === "sync" ? "var(--ok)" : "var(--fg3)"}>{r.role === "sync" ? "sync" : "async"}</Badge>
                  <span className="ml-auto tnum text-[12px]" style={{ color: r.lagSec > 5 ? "var(--warn)" : "var(--fg2)" }}>lag {r.lagSec}s · {fmtBytes(r.lagBytes)}</span>
                </label>
              ))}
            </div>
          </div>
          <div>
            <div className="label mb-2">Pre-flight checks</div>
            <div className="rounded-lg border border-line divide-y divide-line">
              {checks.map((ch, i) => {
                const done = i < checked;
                const Icon = !done ? Loader2 : !ch.ok ? XCircle : ch.warn ? AlertTriangle : CheckCircle2;
                const col = !done ? "var(--fg3)" : !ch.ok ? "var(--crit)" : ch.warn ? "var(--warn)" : "var(--ok)";
                return (
                  <div key={ch.label} className="flex items-center gap-2.5 px-3 py-2 text-[12.5px]">
                    <Icon size={14} style={{ color: col }} className={!done ? "animate-spin" : ""} />
                    <span className="flex-1">{ch.label}</span>
                    <span className="text-fg3 text-[11.5px] text-right max-w-[250px] truncate" title={ch.detail}>{done ? ch.detail : "checking…"}</span>
                  </div>
                );
              })}
            </div>
            {!lagOk && checked >= 6 && <div className="mt-2 text-[12px] flex items-center gap-1.5" style={{ color: "var(--crit)" }}><XCircle size={13} /> {candNode.id} is too far behind. Choose the sync standby or wait for it to catch up.</div>}
          </div>
          {isProd && (
            <label className="block"><div className="label mb-1.5">Type <span className="mono normal-case text-fg">{clusterName}</span> to confirm</div>
              <input value={confirm} onChange={(e) => setConfirm(e.target.value)} className="input w-full mono" placeholder={clusterName} /></label>
          )}
          <p className="text-[12px] text-fg3">Expected write unavailability ≈ 3 s. Reads continue on replicas. Clients reconnect through PgBouncer automatically.</p>
        </div>
      ) : (
        <div className="space-y-2 py-1">
          {steps.map((s, i) => (
            <div key={s} className="flex items-center gap-2.5 text-[13px]">
              {i < step ? <CheckCircle2 size={15} style={{ color: "var(--ok)" }} /> : i === step ? <Loader2 size={15} className="animate-spin text-accent" /> : <Circle size={15} className="text-fg3" />}
              <span className={i <= step ? "" : "text-fg3"}>{s}</span>
              {i < step && <span className="ml-auto text-fg3 text-[11.5px] tnum">{(0.2 + i * 0.4).toFixed(1)} s</span>}
            </div>
          ))}
        </div>
      )}
    </Modal>
  );
}
