import { useEffect, useMemo, useRef, useState } from "react";
import { AlertTriangle, Archive, ArrowUpRight, Filter, Plus, Search, ServerOff, Shield, Upload, Rocket, PlugZap, Sparkles, ArrowRight, Container, Ship, Server } from "lucide-react";
import { TuskMark } from "../components/Brand";
import { CLUSTERS, Cluster, ENVS, Env } from "../data";
import { useApp } from "../store";
import { Badge, Dot, Kbd, PageHeader, Ring, Segmented, Sparkline, Empty, fmt, fmtBytes, healthColor, tint } from "../components/ui";

function ClusterCard({ c }: { c: Cluster }) {
  const { setClusterId, go, newClusterId } = useApp();
  const isNew = newClusterId === c.id;
  const ref = useRef<HTMLButtonElement>(null);
  useEffect(() => { if (isNew) setTimeout(() => ref.current?.scrollIntoView({ block: "center", behavior: "smooth" }), 150); }, [isNew]);
  const hc = healthColor(c.health);
  const backupColor = c.backupAgeH > 24 ? "var(--crit)" : c.backupAgeH > 12 ? "var(--warn)" : "var(--fg)";
  const connPct = Math.round((c.conns / c.maxConns) * 100);
  return (
    <button ref={ref} onClick={() => { setClusterId(c.id); go("cluster"); }}
      className="group card text-left p-4 flex flex-col justify-start hover:border-line2 hover:-translate-y-px transition-all focus-ring relative overflow-hidden"
      style={isNew ? { borderColor: "var(--accent)", boxShadow: `0 0 0 3px ${tint("var(--accent)", 18)}` } : c.health === "crit" ? { borderColor: tint("var(--crit)", 45) } : undefined}>
      {c.health !== "ok" && <div className="absolute inset-x-0 top-0 h-[2px]" style={{ background: hc }} />}
      <div className="flex items-start gap-3 w-full">
        <Ring value={c.score} color={hc} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="mono font-medium text-[13px] truncate">{c.name}</span>
            {isNew && <Badge color="var(--accent)"><Sparkles size={10} />New</Badge>}
            <ArrowUpRight size={13} className="text-fg3 opacity-0 group-hover:opacity-100 transition-opacity" />
          </div>
          <div className="text-[11.5px] text-fg3 mt-0.5 truncate">{c.provider} · {c.region}</div>
        </div>
        <Sparkline data={c.load} color={hc} w={80} h={30} />
      </div>
      <div className="flex flex-wrap gap-1.5 mt-3">
        <Badge>PG {c.version}</Badge>
        <Badge>{c.ha}</Badge>
        <Badge>{c.primary}P{c.replicas ? ` + ${c.replicas}R` : ""}</Badge>
        {c.advisories > 0 && <Badge color="var(--accent)">{c.advisories} advisories</Badge>}
      </div>
      {c.issue && (
        <div className="mt-3 flex items-center gap-1.5 text-[12px] px-2 py-1.5 rounded-md" style={{ background: tint(hc, 10), color: hc }}>
          <AlertTriangle size={12} className="shrink-0" /><span className="truncate">{c.issue}</span>
        </div>
      )}
      <div className="flex-1" />
      <div className="grid grid-cols-4 gap-2 mt-3 pt-3 border-t border-line text-[11.5px] w-full">
        <div><div className="text-fg3">TPS</div><div className="tnum font-medium mt-0.5">{fmt(c.tps)}</div></div>
        <div><div className="text-fg3">Conns</div><div className="tnum font-medium mt-0.5" style={{ color: connPct > 90 ? "var(--crit)" : undefined }}>{connPct}%</div></div>
        <div><div className="text-fg3">Lag</div><div className="tnum font-medium mt-0.5" style={{ color: c.lagSec > 5 ? "var(--warn)" : undefined }}>{c.replicas ? `${c.lagSec}s` : "—"}</div></div>
        <div><div className="text-fg3">Backup</div><div className="tnum font-medium mt-0.5" style={{ color: backupColor }}>{c.backupAgeH === 0 ? "n/a" : `${c.backupAgeH}h`}</div></div>
      </div>
    </button>
  );
}

export default function Fleet() {
  const { go, fleetVersion, freshInstall } = useApp();
  if (freshInstall) return <FleetEmpty />;
  return <FleetList v={fleetVersion} go={go} />;
}

function FleetList({ v, go }: { v: number; go: ReturnType<typeof useApp>["go"] }) {
  const [q, setQ] = useState("");
  const [filter, setFilter] = useState<"all" | "issues">("all");
  const filtered = useMemo(() => CLUSTERS.filter((c) => (filter === "all" || c.health !== "ok") && `${c.name} ${c.region} ${c.ha} ${c.provider} ${c.databases.join(" ")}`.toLowerCase().includes(q.toLowerCase())), [q, filter, v]);
  const counts = { ok: CLUSTERS.filter((c) => c.health === "ok").length, warn: CLUSTERS.filter((c) => c.health === "warn").length, crit: CLUSTERS.filter((c) => c.health === "crit").length };
  const totalTps = CLUSTERS.reduce((a, c) => a + c.tps, 0);
  const totalStorage = CLUSTERS.reduce((a, c) => a + c.storageGb, 0);
  return (
    <div>
      <PageHeader title="Fleet" sub={<>{CLUSTERS.length} clusters across 3 environments · <span className="text-fg3">fake demo data</span></>}
        actions={<><button className="btn" title="Import inventory (demo)"><Upload size={13} /> Import</button><button className="btn" onClick={() => go("deploy")} title="Deploy a new cluster (g d)"><Rocket size={13} /> Deploy cluster</button><button className="btn btn-primary" onClick={() => go("connect")} title="Add connection (n)"><Plus size={13} /> Add connection</button></>} />

      <div className="card grid grid-cols-6 divide-x divide-line mb-5">
        {[
          ["Healthy", counts.ok, "var(--ok)"], ["Degraded", counts.warn, "var(--warn)"], ["Critical", counts.crit, "var(--crit)"],
        ].map(([l, v, col]) => (
          <div key={l as string} className="px-5 py-4"><div className="label mb-1.5 flex items-center gap-1.5"><Dot color={col as string} size={6} />{l}</div><div className="text-[22px] font-semibold tnum leading-none">{v}</div></div>
        ))}
        <div className="px-5 py-4"><div className="label mb-1.5">Fleet TPS</div><div className="text-[22px] font-semibold tnum leading-none">{fmt(totalTps)}</div></div>
        <div className="px-5 py-4"><div className="label mb-1.5">Storage</div><div className="text-[22px] font-semibold tnum leading-none">{fmtBytes(totalStorage * 1e9)}</div></div>
        <div className="px-5 py-4"><div className="label mb-1.5 flex items-center gap-1.5"><Archive size={11} />Backups in SLA</div><div className="text-[22px] font-semibold tnum leading-none">{CLUSTERS.filter((c) => c.backupAgeH <= 24).length}<span className="text-fg3 text-[15px]">/{CLUSTERS.length}</span></div></div>
      </div>

      <div className="flex items-center gap-2 mb-4">
        <div className="relative">
          <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-fg3" />
          <input data-filter value={q} onChange={(e) => setQ(e.target.value)} placeholder="Filter clusters, regions, databases…" className="input pl-8 w-[300px]" />
          <span className="absolute right-2 top-1/2 -translate-y-1/2"><Kbd>/</Kbd></span>
        </div>
        <Segmented value={filter} onChange={setFilter} options={[{ value: "all", label: "All" }, { value: "issues", label: <span className="flex items-center gap-1.5"><Filter size={11} />Needs attention</span> }]} />
        <div className="ml-auto text-fg3 text-[12px]">Sorted by health, then load</div>
      </div>

      {filtered.length === 0 ? (
        <div className="card"><Empty icon={<ServerOff size={18} />} title="No clusters match" body={`Nothing matches “${q}”. Try a region like eu-west or an engine like Patroni.`} action={<button className="btn" onClick={() => { setQ(""); setFilter("all"); }}>Clear filters</button>} /></div>
      ) : (
        <div className="space-y-6">
          {ENVS.map((e) => {
            const list = filtered.filter((c) => c.env === e.id).sort((a, b) => a.score - b.score);
            if (!list.length) return null;
            return <Lane key={e.id} env={e.id} label={e.label} color={e.color} list={list} />;
          })}
        </div>
      )}
    </div>
  );
}

function Lane({ env, label, color, list }: { env: Env; label: string; color: string; list: Cluster[] }) {
  return (
    <section>
      <div className="flex items-center gap-2.5 mb-2.5">
        <span className="w-1 h-4 rounded-full" style={{ background: color }} />
        <h2 className="font-semibold text-[13.5px]">{label}</h2>
        <span className="text-fg3 text-[12px]">{list.length} clusters</span>
        {env === "prod" && <span className="flex items-center gap-1 text-[11px] text-fg3"><Shield size={11} style={{ color }} /> Read-only by default · 4-eyes approvals</span>}
      </div>
      <div className="grid grid-cols-[repeat(auto-fill,minmax(330px,1fr))] gap-3">
        {list.map((c) => <ClusterCard key={c.id} c={c} />)}
      </div>
    </section>
  );
}

function FleetEmpty() {
  const { go, setFreshInstall } = useApp();
  const paths: { icon: typeof Rocket; title: string; body: string; cta: string; run: () => void; primary?: boolean; keys: string[] }[] = [
    { icon: PlugZap, title: "Connect an existing database", body: "RDS, Cloud SQL, Azure, Patroni or a plain server. SSH tunnels, TLS and cloud IAM supported.", cta: "Add connection", run: () => go("connect"), primary: true, keys: ["n"] },
    { icon: Rocket, title: "Deploy a new cluster", body: "Primary + replicas with failover, pooling and backups, on Docker Compose, Kubernetes or your own VMs.", cta: "Deploy cluster", run: () => go("deploy"), keys: ["g", "d"] },
  ];
  return (
    <div className="max-w-[880px] mx-auto pt-10">
      <div className="flex flex-col items-center text-center">
        <TuskMark size={56} />
        <h1 className="text-[24px] font-semibold tracking-tight mt-5">Your fleet is empty</h1>
        <p className="text-fg2 mt-2 max-w-md">Add the PostgreSQL you already run, or let Tuskline stand up a new highly available cluster. You can do both later from <b className="text-fg">Setup</b>.</p>
      </div>
      <div className="grid grid-cols-2 gap-3 mt-8">
        {paths.map((p) => {
          const Icon = p.icon;
          return (
            <button key={p.title} onClick={p.run} className="card p-5 text-left group hover:border-line2 hover:-translate-y-px transition-all focus-ring">
              <div className="flex items-center justify-between"><div className="w-10 h-10 rounded-xl grid place-items-center" style={{ background: tint("var(--accent)", 14), color: "var(--accent)" }}><Icon size={19} /></div><span className="flex gap-0.5">{p.keys.map((k) => <Kbd key={k}>{k}</Kbd>)}</span></div>
              <div className="font-semibold text-[14.5px] mt-4">{p.title}</div>
              <div className="text-fg2 text-[12.5px] mt-1.5 leading-relaxed">{p.body}</div>
              <span className={`btn mt-4 ${p.primary ? "btn-primary" : ""}`}>{p.cta} <ArrowRight size={13} /></span>
            </button>
          );
        })}
      </div>
      <div className="card mt-3 px-5 py-3.5 flex items-center gap-4 text-[12.5px]">
        <span className="text-fg3">Deploy targets</span>
        <span className="flex items-center gap-1.5"><Container size={13} className="text-accent" /> Docker Compose</span>
        <span className="flex items-center gap-1.5"><Ship size={13} className="text-fg2" /> Kubernetes · CloudNativePG</span>
        <span className="flex items-center gap-1.5"><Server size={13} className="text-fg2" /> VMs via SSH</span>
        <button className="btn btn-ghost ml-auto" onClick={() => setFreshInstall(false)}>Load fake demo fleet instead</button>
      </div>
    </div>
  );
}
