import { useEffect, useRef, useState } from "react";
import { AlertOctagon, ChevronDown, ChevronRight, Database, Download, FileCode2, Hash, Key, Link2, Lock, Play, Plus, Search, Shield, ShieldAlert, Table2, Timer, Unlock, Wand2, X, Zap } from "lucide-react";
import { RESULT_ROWS, SCHEMA } from "../data";
import { useApp } from "../store";
import { Badge, Modal, tint } from "../components/ui";
import { SqlText } from "./Queries";

interface Tab { id: number; name: string; sql: string }
const INITIAL: Tab[] = [
  { id: 1, name: "recent-orders.sql", sql: "-- Most recent orders with customer country\nSELECT o.id, o.customer_id, o.status, o.total, o.currency, o.created_at\nFROM public.orders o\nWHERE o.created_at > now() - interval '15 minutes'\nORDER BY o.created_at DESC\nLIMIT 100;" },
  { id: 2, name: "lock-tree.sql", sql: "SELECT pid, pg_blocking_pids(pid) AS blocked_by, wait_event_type, state, query\nFROM pg_stat_activity\nWHERE cardinality(pg_blocking_pids(pid)) > 0;" },
  { id: 3, name: "fix-status.sql", sql: "-- Hotfix: un-stick invoiced orders\nUPDATE public.orders\nSET status = 'paid'\nWHERE status = 'invoiced' AND updated_at < now() - interval '1 day';" },
];
const COLS: [string, string][] = [["id", "int8"], ["customer_id", "int8"], ["status", "order_status"], ["total", "numeric"], ["currency", "bpchar"], ["created_at", "timestamptz"]];

type Result = { kind: "rows"; ms: number } | { kind: "error"; msg: string; stmt: string } | { kind: "explain"; ms: number } | { kind: "ok"; ms: number; affected: number } | null;

function SchemaTree({ onInsert }: { onInsert: (t: string) => void }) {
  const { cluster } = useApp();
  const [open, setOpen] = useState<Record<string, boolean>>({ public: true, "public.orders": true });
  const [q, setQ] = useState("");
  const toggle = (k: string) => setOpen((o) => ({ ...o, [k]: !o[k] }));
  return (
    <div className="w-[248px] shrink-0 border-r border-line flex flex-col min-h-0">
      <div className="p-2 border-b border-line">
        <div className="flex items-center gap-2 px-1.5 py-1 mb-2 text-[12px]"><Database size={13} className="text-fg3" /><span className="mono flex-1 truncate">{cluster.databases[0]}</span><ChevronDown size={12} className="text-fg3" /></div>
        <div className="relative"><Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg3" /><input data-filter value={q} onChange={(e) => setQ(e.target.value)} placeholder="Filter objects" className="input w-full pl-7 !h-[26px] text-[12px]" /></div>
      </div>
      <div className="flex-1 overflow-auto p-1.5 text-[12px]">
        {SCHEMA.map((s) => (
          <div key={s.schema}>
            <button onClick={() => toggle(s.schema)} className="w-full flex items-center gap-1 h-6 px-1 rounded hover:bg-hover text-fg2">
              {open[s.schema] ? <ChevronDown size={12} /> : <ChevronRight size={12} />}<span className="font-medium">{s.schema}</span><span className="ml-auto text-fg3 text-[11px]">{s.tables.length}</span>
            </button>
            {open[s.schema] && s.tables.filter((t) => t.name.includes(q.toLowerCase())).map((t) => {
              const k = `${s.schema}.${t.name}`;
              return (
                <div key={k}>
                  <button onClick={() => toggle(k)} onDoubleClick={() => onInsert(`SELECT *\nFROM ${k}\nLIMIT 100;`)} title="Double-click to query"
                    className="group w-full flex items-center gap-1.5 h-6 pl-4 pr-1 rounded hover:bg-hover">
                    {open[k] ? <ChevronDown size={11} className="text-fg3" /> : <ChevronRight size={11} className="text-fg3" />}
                    <Table2 size={12} className="text-accent" /><span className="mono truncate">{t.name}</span>
                    <span className="ml-auto text-fg3 text-[10.5px] group-hover:hidden">{t.size}</span>
                    <span className="ml-auto text-fg3 text-[10.5px] hidden group-hover:inline">{t.rows} rows</span>
                  </button>
                  {open[k] && t.cols.map(([n, ty, key]) => (
                    <div key={n} className="flex items-center gap-1.5 h-[22px] pl-10 pr-1 text-fg2 hover:bg-hover/60 rounded">
                      {key === "pk" ? <Key size={10} style={{ color: "var(--c2)" }} /> : key === "fk" ? <Link2 size={10} style={{ color: "var(--c3)" }} /> : <Hash size={10} className="text-fg3" />}
                      <span className="mono truncate">{n}</span><span className="ml-auto mono text-fg3 text-[10.5px]">{ty}</span>
                    </div>
                  ))}
                </div>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function Sql() {
  const { cluster, env, elevatedUntil, setElevatedUntil, toast } = useApp();
  const [tabs, setTabs] = useState<Tab[]>(INITIAL);
  const [active, setActive] = useState(1);
  const [result, setResult] = useState<Result>({ kind: "rows", ms: 18 });
  const [running, setRunning] = useState(false);
  const [elevOpen, setElevOpen] = useState(false);
  const [confirmText, setConfirmText] = useState("");
  const [reason, setReason] = useState("");
  const [now, setNow] = useState(Date.now());
  const taRef = useRef<HTMLTextAreaElement>(null);
  const preRef = useRef<HTMLDivElement>(null);
  const tab = tabs.find((t) => t.id === active)!;
  const isProd = env === "prod";
  const elevated = !!elevatedUntil && elevatedUntil > now;
  const readOnly = isProd && !elevated;
  useEffect(() => { if (!elevatedUntil) return; const t = setInterval(() => setNow(Date.now()), 1000); return () => clearInterval(t); }, [elevatedUntil]);

  const setSql = (sql: string) => setTabs((ts) => ts.map((t) => (t.id === active ? { ...t, sql } : t)));
  const run = (explain = false) => {
    setRunning(true);
    setTimeout(() => {
      setRunning(false);
      const write = /^\s*(--[^\n]*\n\s*)*(update|delete|insert|alter|drop|truncate|create)\b/i.exec(tab.sql);
      if (write && readOnly) { setResult({ kind: "error", stmt: write[2].toUpperCase(), msg: `cannot execute ${write[2].toUpperCase()} in a read-only transaction` }); return; }
      if (write) { setResult({ kind: "ok", ms: 42, affected: 37 }); toast({ title: `${write[2].toUpperCase()} committed · 37 rows`, body: "Recorded in the audit log with your elevation reason.", tone: "warn" }); return; }
      setResult(explain ? { kind: "explain", ms: 21 } : { kind: "rows", ms: 12 + Math.round(Math.random() * 20) });
    }, 380);
  };
  const newTab = (sql = "") => { const id = Math.max(...tabs.map((t) => t.id)) + 1; setTabs([...tabs, { id, name: `untitled-${id}.sql`, sql }]); setActive(id); };
  const closeTab = (id: number) => { if (tabs.length === 1) return; const rest = tabs.filter((t) => t.id !== id); setTabs(rest); if (active === id) setActive(rest[0].id); };
  const lines = tab.sql.split("\n").length;
  const remaining = elevatedUntil ? Math.max(0, Math.round((elevatedUntil - now) / 1000)) : 0;

  return (
    <div className="card overflow-hidden flex flex-col" style={{ height: `calc(100vh - ${isProd ? 150 : 126}px)` }}>
      {isProd && (readOnly ? (
        <div className="flex items-center gap-2.5 px-4 h-10 shrink-0 text-[12.5px]" style={{ background: tint("var(--prod)", 11), color: "var(--prod)", borderBottom: `1px solid ${tint("var(--prod)", 28)}` }}>
          <Shield size={14} /><span className="font-semibold">Production · read-only</span>
          <span className="text-fg2">Session runs with <span className="mono">default_transaction_read_only = on</span> on <span className="mono">{cluster.name}</span>. Writes are blocked.</span>
          <button className="btn ml-auto !h-[26px]" style={{ borderColor: tint("var(--prod)", 40), color: "var(--prod)" }} onClick={() => setElevOpen(true)}><Unlock size={12} /> Request write access</button>
        </div>
      ) : (
        <div className="flex items-center gap-2.5 px-4 h-10 shrink-0 text-[12.5px] animate-fadeIn" style={{ background: "var(--prod)", color: "white" }}>
          <ShieldAlert size={14} /><span className="font-semibold">WRITE ACCESS ON PRODUCTION</span><span className="opacity-85">Every statement is audited · auto-revokes in</span>
          <span className="mono font-semibold tnum">{Math.floor(remaining / 60)}:{String(remaining % 60).padStart(2, "0")}</span>
          <button className="btn ml-auto !h-[26px] !bg-white/15 !border-white/30 !text-white hover:!bg-white/25" onClick={() => { setElevatedUntil(null); toast({ title: "Back to read-only", tone: "ok" }); }}><Lock size={12} /> Drop elevation</button>
        </div>
      ))}
      <div className="flex flex-1 min-h-0">
        <SchemaTree onInsert={(s) => newTab(s)} />
        <div className="flex-1 flex flex-col min-w-0">
          <div className="flex items-center h-9 border-b border-line bg-subtle/60 shrink-0">
            {tabs.map((t) => (
              <div key={t.id} onClick={() => setActive(t.id)} className={`group h-full flex items-center gap-1.5 pl-3 pr-1.5 border-r border-line cursor-pointer text-[12px] ${active === t.id ? "bg-elev text-fg" : "text-fg3 hover:text-fg"}`}
                style={active === t.id ? { boxShadow: "inset 0 2px 0 var(--accent)" } : undefined}>
                <FileCode2 size={12} /><span className="mono">{t.name}</span>
                <button onClick={(e) => { e.stopPropagation(); closeTab(t.id); }} className="w-4 h-4 grid place-items-center rounded opacity-0 group-hover:opacity-100 hover:bg-hover"><X size={11} /></button>
              </div>
            ))}
            <button onClick={() => newTab()} className="h-full px-2.5 text-fg3 hover:text-fg" title="New tab"><Plus size={14} /></button>
          </div>
          <div className="flex items-center gap-2 px-3 h-10 border-b border-line shrink-0">
            <button className="btn btn-primary" onClick={() => run(false)} disabled={running}><Play size={12} /> {running ? "Running…" : "Run"} <span className="opacity-70 text-[11px] ml-0.5">⌘↵</span></button>
            <button className="btn" onClick={() => run(true)}><Zap size={12} /> Explain analyse <span className="text-fg3 text-[11px] ml-0.5">⌘⇧↵</span></button>
            <button className="btn btn-ghost" onClick={() => toast({ title: "Formatted with pg_format rules", tone: "ok" })}><Wand2 size={12} /> Format</button>
            <div className="ml-auto flex items-center gap-2 text-[11.5px] text-fg3">
              <span className="flex items-center gap-1"><Timer size={12} /> statement_timeout 30s</span>
              <span>·</span><span>Limit 1,000</span><span>·</span>
              {readOnly ? <Badge color="var(--prod)"><Lock size={10} /> read-only</Badge> : isProd ? <Badge color="var(--prod)" solid>read-write</Badge> : <Badge color="var(--ok)">read-write</Badge>}
            </div>
          </div>
          <div className="relative flex-[1.1] min-h-0 overflow-hidden flex text-[12.5px] leading-[20px]">
            <div className="w-11 shrink-0 text-right pr-3 pt-3 text-fg3 mono select-none bg-subtle/40 border-r border-line">{Array.from({ length: Math.max(lines, 12) }, (_, i) => <div key={i} className={i < lines ? "" : "opacity-0"}>{i + 1}</div>)}</div>
            <div className="relative flex-1 min-w-0 overflow-auto">
              <div ref={preRef} aria-hidden className="absolute inset-0 p-3 whitespace-pre pointer-events-none"><SqlText sql={tab.sql + "\n"} /></div>
              <textarea ref={taRef} value={tab.sql} spellCheck={false} onChange={(e) => setSql(e.target.value)}
                onKeyDown={(e) => { if ((e.metaKey || e.ctrlKey) && e.key === "Enter") { e.preventDefault(); run(e.shiftKey); } }}
                onScroll={(e) => { if (preRef.current) { preRef.current.scrollTop = e.currentTarget.scrollTop; preRef.current.scrollLeft = e.currentTarget.scrollLeft; } }}
                className="absolute inset-0 w-full h-full p-3 mono bg-transparent resize-none outline-none whitespace-pre text-transparent leading-[20px]" style={{ caretColor: "var(--accent)" }} />
            </div>
          </div>
          <ResultsPanel result={result} running={running} onElevate={() => setElevOpen(true)} />
        </div>
      </div>
      <Modal open={elevOpen} onClose={() => setElevOpen(false)} tone="danger" title={<span className="flex items-center gap-2"><ShieldAlert size={15} style={{ color: "var(--crit)" }} /> Elevate to write access on production</span>}
        footer={<><button className="btn" onClick={() => setElevOpen(false)}>Cancel</button><button className="btn btn-danger" disabled={confirmText !== cluster.name || reason.trim().length < 4}
          onClick={() => { setElevatedUntil(Date.now() + 15 * 60 * 1000); setElevOpen(false); setConfirmText(""); setReason(""); toast({ title: "Write access granted for 15 minutes", body: "Approved by policy prod-hotfix · audit id AUD-20261027-0912", tone: "warn" }); }}>Elevate for 15 min</button></>}>
        <div className="space-y-4">
          <p className="text-fg2 text-[12.5px] leading-relaxed">Your session on <span className="mono text-fg">{cluster.name}</span> will switch to read-write for <b className="text-fg">15 minutes</b>. All statements are recorded with your reason. Policy may require a second approver for DDL.</p>
          <label className="block"><div className="label mb-1.5">Reason / ticket</div><input autoFocus value={reason} onChange={(e) => setReason(e.target.value)} placeholder="e.g. INC-4821 un-stick invoiced orders" className="input w-full" /></label>
          <label className="block"><div className="label mb-1.5">Type <span className="mono normal-case text-fg">{cluster.name}</span> to confirm</div><input value={confirmText} onChange={(e) => setConfirmText(e.target.value)} className="input w-full mono" style={confirmText && confirmText !== cluster.name ? { borderColor: "var(--crit)" } : undefined} /></label>
        </div>
      </Modal>
    </div>
  );
}

function ResultsPanel({ result, running, onElevate }: { result: Result; running: boolean; onElevate: () => void }) {
  const { setClusterId } = useApp();
  const [tab, setTab] = useState<"results" | "messages">("results");
  return (
    <div className="flex-1 min-h-0 border-t border-line flex flex-col">
      <div className="flex items-center gap-1 px-2 h-9 border-b border-line shrink-0 bg-subtle/60">
        {(["results", "messages"] as const).map((t) => <button key={t} onClick={() => setTab(t)} className={`h-7 px-2.5 rounded text-[12px] capitalize ${tab === t ? "bg-elev text-fg ring-1 ring-line" : "text-fg3 hover:text-fg"}`}>{t}</button>)}
        <div className="ml-auto flex items-center gap-3 text-[11.5px] text-fg3 pr-2">
          {result?.kind === "rows" && <><span className="tnum">{RESULT_ROWS.length} rows · {result.ms} ms</span><button className="flex items-center gap-1 hover:text-fg"><Download size={12} /> CSV</button></>}
          {result?.kind === "explain" && <span>EXPLAIN (ANALYZE, BUFFERS) · rolled back</span>}
        </div>
      </div>
      <div className="flex-1 overflow-auto">
        {running ? (
          <div className="p-4 space-y-2">{Array.from({ length: 6 }, (_, i) => <div key={i} className="h-5 rounded bg-subtle animate-pulse" style={{ width: `${90 - i * 7}%` }} />)}</div>
        ) : tab === "messages" ? (
          <pre className="mono text-[12px] text-fg2 p-4">{`SET default_transaction_read_only = on;\nSET statement_timeout = '30s';\n${result?.kind === "error" ? "ERROR:  25006: " + result.msg : "OK"}`}</pre>
        ) : result?.kind === "error" ? (
          <div className="m-4 rounded-lg p-4 flex gap-3 animate-fadeIn" style={{ background: tint("var(--crit)", 9), border: `1px solid ${tint("var(--crit)", 30)}` }}>
            <AlertOctagon size={18} style={{ color: "var(--crit)" }} className="shrink-0 mt-0.5" />
            <div className="flex-1">
              <div className="mono text-[12.5px]" style={{ color: "var(--crit)" }}>ERROR: 25006: {result.msg}</div>
              <div className="text-fg2 text-[12.5px] mt-1.5">Tuskline blocked this {result.stmt} because production sessions are read-only. Nothing was executed.</div>
              <div className="flex gap-2 mt-3"><button className="btn" onClick={onElevate}><Unlock size={12} /> Request write access</button><button className="btn btn-ghost" onClick={() => setClusterId("staging-eu-west")}>Switch to staging-eu-west</button></div>
            </div>
          </div>
        ) : result?.kind === "ok" ? (
          <div className="p-4 mono text-[12.5px]" style={{ color: "var(--ok)" }}>UPDATE 37 · {result.ms} ms · committed</div>
        ) : result?.kind === "explain" ? (
          <pre className="mono text-[12px] p-4 leading-relaxed text-fg2">{`Limit  (cost=0.56..412.80 rows=100) (actual time=0.031..0.912 rows=100 loops=1)
  ->  Index Scan Backward using idx_orders_created_at on orders o  (actual time=0.030..0.874 rows=100 loops=1)
        Index Cond: (created_at > (now() - '00:15:00'::interval))
        Buffers: shared hit=212
Planning Time: 0.188 ms
Execution Time: 0.951 ms`}</pre>
        ) : result?.kind === "rows" ? (
          <table className="text-[12px] mono min-w-full">
            <thead className="sticky top-0 bg-elev z-10"><tr className="border-b border-line">
              <th className="w-10 px-2 py-1.5 text-fg3 font-normal text-right border-r border-line">#</th>
              {COLS.map(([c, t]) => <th key={c} className="px-3 py-1.5 text-left font-medium border-r border-line whitespace-nowrap"><span className="text-fg">{c}</span> <span className="text-fg3 font-normal text-[10.5px]">{t}</span></th>)}
            </tr></thead>
            <tbody>
              {RESULT_ROWS.map((r, i) => (
                <tr key={i} className="border-b border-line hover:bg-hover/60">
                  <td className="px-2 py-1 text-right text-fg3 border-r border-line">{i + 1}</td>
                  {r.map((v, j) => <td key={j} className={`px-3 py-1 border-r border-line whitespace-nowrap ${j === 3 ? "text-right tnum" : ""}`}>{j === 2 ? <StatusCell v={v} /> : v}</td>)}
                </tr>
              ))}
            </tbody>
          </table>
        ) : null}
      </div>
    </div>
  );
}
function StatusCell({ v }: { v: string }) {
  const c = v === "paid" ? "var(--ok)" : v === "cancelled" || v === "refunded" ? "var(--crit)" : v === "pending" ? "var(--warn)" : "var(--accent)";
  return <span style={{ color: c }}>{v}</span>;
}
