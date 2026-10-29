import { useEffect, useMemo, useState } from "react";
import { ArrowDown, ArrowUp, CheckCircle2, Copy, FlaskConical, GitCompareArrows, Lightbulb, Play, Search, TrendingUp, Zap } from "lucide-react";
import { PLAN_AFTER, PLAN_BEFORE, PlanNode, QUERIES, Query } from "../data";
import { useApp } from "../store";
import { Badge, Card, Kbd, PageHeader, Segmented, Sparkline, fmt, fmtMs, tint } from "../components/ui";

type SortKey = "share" | "callsPerSec" | "meanMs" | "p95Ms";

export function SqlText({ sql, className = "" }: { sql: string; className?: string }) {
  const kw = /\b(SELECT|FROM|WHERE|JOIN|ON|AND|OR|UPDATE|SET|INSERT|INTO|VALUES|DELETE|WITH|AS|GROUP|BY|ORDER|LIMIT|DESC|ASC|ANY|COUNT|SUM|NOW|INTERVAL|ALTER|TABLE|ADD|COLUMN|CREATE|INDEX|CONCURRENTLY|CONSTRAINT|FOREIGN|KEY|REFERENCES|NOT|NULL|LIKE|INCLUDING|ALL|PARTITION|RANGE|TYPE|ENUM|DROP|EXPLAIN|ANALYZE|BUFFERS|BEGIN|COMMIT|ROLLBACK|LOWER|IS|IN|CASE|WHEN|THEN|ELSE|END|DISTINCT|HAVING|LEFT|INNER|OUTER|RETURNING)\b/gi;
  const parts = sql.split(/(--[^\n]*|'[^']*'|\$\d+|\b\d+(?:\.\d+)?\b)/g);
  return (
    <code className={`mono ${className}`}>
      {parts.map((p, i) => {
        if (!p) return null;
        if (p.startsWith("--")) return <span key={i} style={{ color: "var(--fg3)", fontStyle: "italic" }}>{p}</span>;
        if (p.startsWith("'")) return <span key={i} style={{ color: "var(--c2)" }}>{p}</span>;
        if (p.startsWith("$")) return <span key={i} style={{ color: "var(--c4)" }}>{p}</span>;
        if (/^\d/.test(p)) return <span key={i} style={{ color: "var(--c3)" }}>{p}</span>;
        return <span key={i}>{p.split(kw).map((t, j) => (j % 2 ? <span key={j} style={{ color: "var(--accent)", fontWeight: 500 }}>{t}</span> : t))}</span>;
      })}
    </code>
  );
}

function flatten(n: PlanNode, depth = 0, out: { n: PlanNode; d: number }[] = []) {
  out.push({ n, d: depth });
  n.children?.forEach((c) => flatten(c, depth + 1, out));
  return out;
}

function PlanTree({ root }: { root: PlanNode }) {
  const rows = flatten(root);
  const [sel, setSel] = useState(rows[1]?.n.id ?? rows[0].n.id);
  const selected = rows.find((r) => r.n.id === sel)!.n;
  return (
    <div>
      <div className="rounded-lg border border-line overflow-hidden">
        {rows.map(({ n, d }) => {
          const pct = (n.selfMs / root.totalMs) * 100;
          const mis = Math.max(n.actRows / Math.max(n.estRows, 1), n.estRows / Math.max(n.actRows, 1));
          const hot = pct > 50;
          return (
            <button key={n.id} onClick={() => setSel(n.id)} className={`w-full text-left flex items-center gap-3 px-3 py-2 border-b border-line last:border-0 transition-colors ${sel === n.id ? "bg-hover" : "hover:bg-hover/50"}`}>
              <div className="flex items-center gap-2 min-w-0 flex-1" style={{ paddingLeft: d * 18 }}>
                {d > 0 && <span className="text-fg3 mono">↳</span>}
                <span className="font-medium whitespace-nowrap" style={{ color: n.op.includes("Seq") ? "var(--crit)" : undefined }}>{n.op}</span>
                {n.rel && <span className="mono text-[11.5px] text-fg2 truncate">{n.rel}</span>}
                {n.idx && <span className="mono text-[11px] text-fg3 truncate">using {n.idx}</span>}
              </div>
              {mis > 10 && <Badge color="var(--warn)">rows ×{Math.round(mis)}</Badge>}
              <div className="w-[120px] shrink-0">
                <div className="flex justify-between text-[11px] mb-0.5"><span className="tnum">{fmtMs(n.selfMs)}</span><span className="text-fg3 tnum">{pct.toFixed(0)}%</span></div>
                <div className="h-1.5 rounded-full bg-subtle overflow-hidden"><div className="h-full rounded-full" style={{ width: `${Math.max(pct, 2)}%`, background: hot ? "var(--crit)" : "var(--accent)" }} /></div>
              </div>
            </button>
          );
        })}
      </div>
      <div className="grid grid-cols-4 gap-2 mt-3">
        {[["Est. rows", fmt(selected.estRows)], ["Actual rows", fmt(selected.actRows)], ["Loops", fmt(selected.loops)], ["Buffers read / hit", `${fmt(selected.bufRead)} / ${fmt(selected.bufHit)}`]].map(([l, v]) => (
          <div key={l} className="rounded-md bg-subtle border border-line px-2.5 py-2"><div className="text-fg3 text-[11px]">{l}</div><div className="tnum font-medium mono text-[12px] mt-0.5">{v}</div></div>
        ))}
      </div>
      {selected.detail && <pre className="mono text-[11.5px] text-fg2 mt-2 p-2.5 rounded-md bg-subtle border border-line whitespace-pre-wrap">{selected.detail}</pre>}
    </div>
  );
}

function Flame({ root }: { root: PlanNode }) {
  const levels: { n: PlanNode; x: number; w: number }[][] = [];
  const walk = (n: PlanNode, x: number, w: number, d: number) => {
    (levels[d] ||= []).push({ n, x, w });
    let cx = x;
    n.children?.forEach((c) => { const cw = (c.totalMs / n.totalMs) * w; walk(c, cx, cw, d + 1); cx += cw; });
  };
  walk(root, 0, 100, 0);
  return (
    <div className="space-y-1">
      {levels.map((lvl, d) => (
        <div key={d} className="relative h-8">
          {lvl.map(({ n, x, w }) => (
            <div key={n.id} title={`${n.op} ${n.rel ?? ""} · ${fmtMs(n.totalMs)}`} className="absolute top-0 h-8 rounded-[5px] px-2 flex items-center text-[11.5px] overflow-hidden whitespace-nowrap text-white font-medium"
              style={{ left: `${x}%`, width: `calc(${Math.max(w, 1.2)}% - 2px)`, background: n.op.includes("Seq") ? "var(--crit)" : n.op.includes("Index") ? "var(--ok)" : "var(--accent)", opacity: 0.9 }}>
              {w > 8 && <>{n.op} {n.rel && <span className="opacity-80 ml-1 mono">{n.rel}</span>}<span className="ml-auto pl-2 opacity-80 tnum">{fmtMs(n.totalMs)}</span></>}
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

function Detail({ q }: { q: Query }) {
  const { toast, cluster } = useApp();
  const [tab, setTab] = useState<"plan" | "diff" | "advice">("plan");
  const [view, setView] = useState<"tree" | "flame">("tree");
  const [tested, setTested] = useState(false);
  useEffect(() => { setTab("plan"); setTested(false); }, [q.id]);
  const isHot = q.id === "a91f3c";
  return (
    <Card pad={false} className="sticky top-0">
      <div className="p-4 border-b border-line">
        <div className="flex items-center gap-2 mb-2">
          <span className="mono text-[12px] text-fg3">#{q.id}</span><Badge>{q.db}</Badge><Badge>{q.user}</Badge>
          {q.regressed && <Badge color="var(--crit)"><TrendingUp size={11} /> regressed 31% since deploy 2026.10.26</Badge>}
        </div>
        <div className="rounded-md bg-subtle border border-line p-3 text-[12px] leading-relaxed max-h-[88px] overflow-auto"><SqlText sql={q.sql} /></div>
        <div className="grid grid-cols-4 gap-3 mt-3">
          {[["Share of load", `${q.share}%`], ["Calls/s", fmt(q.callsPerSec, 1)], ["Mean", fmtMs(q.meanMs)], ["p95", fmtMs(q.p95Ms)]].map(([l, v]) => (
            <div key={l}><div className="text-fg3 text-[11px]">{l}</div><div className="font-semibold tnum text-[15px]">{v}</div></div>
          ))}
        </div>
      </div>
      <div className="flex items-center gap-1 px-3 pt-2 border-b border-line">
        {([["plan", "EXPLAIN plan", Zap], ["diff", "Plan diff", GitCompareArrows], ["advice", "Index advice", Lightbulb]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id)} className={`flex items-center gap-1.5 h-8 px-2.5 text-[12.5px] border-b-2 -mb-px transition-colors ${tab === id ? "border-accent text-fg font-medium" : "border-transparent text-fg3 hover:text-fg"}`}>
            <Icon size={13} />{label}{id === "advice" && isHot && <span className="w-1.5 h-1.5 rounded-full bg-accent" />}
          </button>
        ))}
        {tab === "plan" && <div className="ml-auto pb-1.5"><Segmented size="xs" value={view} onChange={setView} options={[{ value: "tree", label: "Tree" }, { value: "flame", label: "Flame" }]} /></div>}
      </div>
      <div className="p-4">
        {tab === "plan" && (isHot ? (
          <div className="animate-fadeIn">
            <div className="flex flex-wrap gap-1.5 mb-3">
              <Badge color="var(--crit)">Seq Scan on 12.4 GB table</Badge><Badge color="var(--warn)">Row misestimate ×412</Badge><Badge color="var(--fg3)">Execution 181.9 ms · Planning 0.4 ms</Badge>
            </div>
            {view === "tree" ? <PlanTree root={PLAN_BEFORE} /> : <Flame root={PLAN_BEFORE} />}
          </div>
        ) : (
          <div className="text-fg2 text-[12.5px] py-6 text-center">Plan captured by auto_explain 4 min ago · no issues detected.<div className="mt-3"><button className="btn" onClick={() => toast({ title: "EXPLAIN (ANALYZE, BUFFERS) queued", body: `Runs on a replica of ${cluster.name} inside a rolled-back transaction.`, tone: "info" })}><Play size={12} /> Re-run EXPLAIN ANALYZE</button></div></div>
        ))}
        {tab === "diff" && (
          <div className="grid grid-cols-2 gap-3 animate-fadeIn">
            {[["Before · 26 Oct", PLAN_BEFORE, "var(--crit)"], ["With idx_order_items_order_sku", PLAN_AFTER, "var(--ok)"]].map(([l, p, col]) => (
              <div key={l as string} className="rounded-lg border border-line p-3" style={{ borderTop: `2px solid ${col}` }}>
                <div className="flex justify-between items-baseline mb-2"><span className="text-[12px] text-fg2">{l as string}</span><span className="font-semibold tnum" style={{ color: col as string }}>{fmtMs((p as PlanNode).totalMs)}</span></div>
                {flatten(p as PlanNode).map(({ n, d }) => (
                  <div key={n.id} className="text-[11.5px] py-1 flex gap-1.5" style={{ paddingLeft: d * 12 }}>
                    <span className="font-medium whitespace-nowrap" style={{ color: n.op.includes("Seq") ? "var(--crit)" : undefined }}>{n.op}</span><span className="mono text-fg3 truncate">{n.rel}</span>
                  </div>
                ))}
              </div>
            ))}
            <div className="col-span-2 flex items-center gap-2 text-[12.5px] rounded-md px-3 py-2" style={{ background: tint("var(--ok)", 10), color: "var(--ok)" }}><ArrowDown size={13} /> 58× faster · 1.6 M fewer buffer reads per call</div>
          </div>
        )}
        {tab === "advice" && (isHot ? (
          <div className="animate-fadeIn space-y-3">
            <div className="flex items-start gap-2.5">
              <div className="w-8 h-8 rounded-lg grid place-items-center shrink-0" style={{ background: tint("var(--accent)", 15), color: "var(--accent)" }}><Lightbulb size={15} /></div>
              <div><div className="font-medium">Add a composite index on order_items (order_id, sku)</div><div className="text-fg2 text-[12.5px] mt-0.5">Covers this query and 2 others (#c7d9e1, #19f2aa reads). Predicted DB load reduction: 31%. Write overhead: +0.3 ms per insert on order_items.</div></div>
            </div>
            <div className="rounded-md bg-subtle border border-line p-3 relative">
              <SqlText className="text-[12px] whitespace-pre-wrap" sql={"CREATE INDEX CONCURRENTLY idx_order_items_order_sku\n  ON public.order_items (order_id, sku);"} />
              <button className="absolute top-2 right-2 text-fg3 hover:text-fg" onClick={() => toast({ title: "Copied", tone: "ok" })}><Copy size={12} /></button>
            </div>
            <div className="grid grid-cols-3 gap-2 text-[12px]">
              <div className="rounded-md border border-line p-2.5"><div className="text-fg3 text-[11px]">Est. size</div><div className="font-medium tnum">410 MB</div></div>
              <div className="rounded-md border border-line p-2.5"><div className="text-fg3 text-[11px]">Build time</div><div className="font-medium tnum">≈ 6 min</div></div>
              <div className="rounded-md border border-line p-2.5"><div className="text-fg3 text-[11px]">Lock</div><div className="font-medium">SHARE UPDATE EXCL.</div></div>
            </div>
            {tested ? (
              <div className="flex items-center gap-2 text-[12.5px] rounded-md px-3 py-2 animate-fadeIn" style={{ background: tint("var(--ok)", 10), color: "var(--ok)" }}><CheckCircle2 size={14} /> hypopg: planner picks the index · cost 182,410 → 3,120 (−98%)</div>
            ) : null}
            <div className="flex gap-2">
              <button className="btn" onClick={() => setTested(true)}><FlaskConical size={13} /> Validate with hypopg</button>
              <button className="btn btn-primary" onClick={() => toast({ title: "Migration 0143 drafted", body: "Opened as a change request for review in Migrations.", tone: "ok" })}>Create migration</button>
            </div>
          </div>
        ) : <div className="text-fg2 text-[12.5px] py-6 text-center">No index recommendations for this statement.</div>)}
      </div>
    </Card>
  );
}

export default function Queries() {
  const [sort, setSort] = useState<SortKey>("share");
  const [q, setQ] = useState("");
  const [sel, setSel] = useState("a91f3c");
  useEffect(() => { const on = (e: Event) => setSel((e as CustomEvent).detail); window.addEventListener("tl:query", on); return () => window.removeEventListener("tl:query", on); }, []);
  const rows = useMemo(() => QUERIES.filter((x) => `${x.sql} ${x.db} ${x.user} ${x.id}`.toLowerCase().includes(q.toLowerCase())).sort((a, b) => b[sort] - a[sort]), [q, sort]);
  const selected = QUERIES.find((x) => x.id === sel)!;
  const th = (k: SortKey, l: string) => (
    <th className="font-medium px-3 py-2 text-right cursor-pointer select-none hover:text-fg whitespace-nowrap" onClick={() => setSort(k)}>
      <span className="inline-flex items-center gap-1">{l}{sort === k ? <ArrowDown size={11} /> : <ArrowUp size={11} className="opacity-0" />}</span>
    </th>
  );
  return (
    <div>
      <PageHeader title="Query insights" sub="pg_stat_statements · auto_explain · normalised and fingerprinted across all databases"
        actions={<div className="relative"><Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-fg3" /><input data-filter value={q} onChange={(e) => setQ(e.target.value)} placeholder="Filter statements…" className="input pl-8 w-[260px]" /><span className="absolute right-2 top-1/2 -translate-y-1/2"><Kbd>/</Kbd></span></div>} />
      <div className="grid grid-cols-[minmax(0,1fr)_540px] gap-4 items-start">
        <Card pad={false}>
          <table className="w-full text-[12px]">
            <thead><tr className="text-fg3 text-left border-b border-line">
              <th className="font-medium px-3 py-2">Statement</th>{th("share", "Load")}{th("callsPerSec", "Calls/s")}{th("meanMs", "Mean")}{th("p95Ms", "p95")}<th className="font-medium px-3 py-2">24h</th>
            </tr></thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id} onClick={() => setSel(r.id)} className={`border-b border-line last:border-0 cursor-pointer transition-colors ${sel === r.id ? "bg-hover" : "hover:bg-hover/50"}`}
                  style={sel === r.id ? { boxShadow: "inset 2px 0 0 var(--accent)" } : undefined}>
                  <td className="px-3 py-2.5 max-w-0 w-full">
                    <div className="truncate text-[11.5px]"><SqlText sql={r.sql} /></div>
                    <div className="flex items-center gap-1.5 mt-1 text-[11px] text-fg3 whitespace-nowrap overflow-hidden"><span className="mono">#{r.id}</span><span className="truncate">{r.db} · {r.user}</span>{r.flags.slice(0, 1).map((f) => <Badge key={f} color={f.includes("Seq") || f.includes("Lock") ? "var(--crit)" : "var(--warn)"} className="!h-[17px] !text-[10.5px]">{f}</Badge>)}</div>
                  </td>
                  <td className="px-3 py-2.5 text-right"><div className="tnum font-medium">{r.share}%</div><div className="h-1 mt-1 rounded-full bg-subtle w-12 ml-auto overflow-hidden"><div className="h-full bg-accent" style={{ width: `${r.share * 2.5}%` }} /></div></td>
                  <td className="px-3 py-2.5 text-right tnum whitespace-nowrap">{fmt(r.callsPerSec, 1)}</td>
                  <td className="px-3 py-2.5 text-right tnum whitespace-nowrap">{fmtMs(r.meanMs)}</td>
                  <td className="px-3 py-2.5 text-right tnum text-fg2 whitespace-nowrap">{fmtMs(r.p95Ms)}</td>
                  <td className="px-3 py-2.5 pr-4"><Sparkline data={r.trend} w={56} h={20} color={r.regressed ? "var(--crit)" : "var(--accent)"} /></td>
                </tr>
              ))}
              {rows.length === 0 && <tr><td colSpan={6} className="text-center text-fg3 py-10">No statements match “{q}”.</td></tr>}
            </tbody>
          </table>
        </Card>
        <Detail q={selected} />
      </div>
    </div>
  );
}
