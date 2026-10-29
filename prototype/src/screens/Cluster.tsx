import { useMemo, useState } from "react";
import { Area, AreaChart, CartesianGrid, Line, LineChart, ReferenceArea, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Activity, ChevronDown, ChevronRight, Copy, Stethoscope, XCircle, Zap } from "lucide-react";
import { FINDINGS, SESSIONS, WAIT_KEYS, loadSeries } from "../data";
import { useApp, Screen } from "../store";
import { Badge, Card, Dot, EnvBadge, PageHeader, Stat, fmt, fmtBytes, healthColor, tint, tooltipStyle } from "../components/ui";

const hash = (s: string) => [...s].reduce((a, c) => a + c.charCodeAt(0), 0);

export default function Cluster() {
  const { cluster: c, go, toast, range } = useApp();
  const data = useMemo(() => loadSeries(hash(c.id)), [c.id]);
  const [hidden, setHidden] = useState<string[]>([]);
  const [open, setOpen] = useState<string | null>("f1");
  const scale = c.tps / 4800;
  const totals = WAIT_KEYS.map((w) => ({ ...w, sum: data.reduce((a, d) => a + (d[w.key] as number), 0) }));
  const all = totals.reduce((a, t) => a + t.sum, 0);
  return (
    <div>
      <PageHeader
        icon={<Dot color={healthColor(c.health)} pulse={c.health !== "ok"} size={9} />}
        title={<span className="mono">{c.name}</span>}
        sub={<span className="flex items-center gap-2"><EnvBadge env={c.env} /> PostgreSQL {c.version} · {c.ha} · {c.provider} {c.region} · {c.primary} primary, {c.replicas} replicas · last {range}</span>}
        actions={<><button className="btn" onClick={() => go("sql")}>Open SQL</button><button className="btn btn-primary" onClick={() => toast({ title: "Diagnosis complete", body: "4 findings · 1 critical. Ranked by impact on DB load.", tone: "warn" })}><Stethoscope size={13} /> Diagnose</button></>} />

      <div className="card grid grid-cols-6 divide-x divide-line mb-4">
        <div className="p-4"><Stat label="Health score" value={c.score} color={healthColor(c.health)} sub="7 checks, 2 failing" /></div>
        <div className="p-4"><Stat label="Transactions/s" value={fmt(c.tps)} sub="+4.2% vs last week" /></div>
        <div className="p-4"><Stat label="Avg active sessions" value={fmt(6.1 * scale, 1)} sub="8 vCPU on primary" /></div>
        <div className="p-4"><Stat label="Connections" value={<>{c.conns}<span className="text-fg3 text-[14px]">/{c.maxConns}</span></>} sub="via PgBouncer (transaction)" /></div>
        <div className="p-4"><Stat label="Cache hit" value="99.4%" sub="shared_buffers 16 GB" /></div>
        <div className="p-4"><Stat label="Max replica lag" value={c.replicas ? `${c.lagSec}s` : "—"} color={c.lagSec > 5 ? "var(--warn)" : undefined} sub={c.replicas ? fmtBytes(c.lagBytes) + " behind" : "no replicas"} /></div>
      </div>

      <div className="grid grid-cols-[1fr_380px] gap-4">
        <div className="space-y-4 min-w-0">
          <Card title="Database load" sub="Average active sessions by wait event"
            actions={<Badge color="var(--crit)">Lock spike 11:45–12:35</Badge>}>
            <div className="flex flex-wrap gap-1.5 mb-3">
              {totals.map((t) => (
                <button key={t.key} onClick={() => setHidden((h) => (h.includes(t.key) ? h.filter((x) => x !== t.key) : [...h, t.key]))}
                  className={`flex items-center gap-1.5 h-6 px-2 rounded-md border text-[11.5px] transition-opacity ${hidden.includes(t.key) ? "opacity-40 border-line" : "border-line2 bg-subtle"}`}>
                  <span className="w-2 h-2 rounded-sm" style={{ background: t.color }} /><span className="mono">{t.key}</span><span className="text-fg3 tnum">{Math.round((t.sum / all) * 100)}%</span>
                </button>
              ))}
            </div>
            <div className="h-[240px]">
              <ResponsiveContainer>
                <AreaChart data={data} margin={{ top: 4, right: 4, left: -18, bottom: 0 }}>
                  <CartesianGrid vertical={false} />
                  <XAxis dataKey="t" tickLine={false} axisLine={false} interval={11} />
                  <YAxis tickLine={false} axisLine={false} />
                  <Tooltip {...tooltipStyle} />
                  <ReferenceArea x1={data[45].t as string} x2={data[55].t as string} fill="var(--crit)" fillOpacity={0.06} />
                  <ReferenceLine y={8} stroke="var(--fg3)" strokeDasharray="4 4" label={{ value: "8 vCPU", position: "insideTopRight", fill: "var(--fg3)", fontSize: 11 }} />
                  {WAIT_KEYS.filter((w) => !hidden.includes(w.key)).map((w) => (
                    <Area key={w.key} type="monotone" dataKey={w.key} stackId="1" stroke={w.color} fill={w.color} fillOpacity={0.55} strokeWidth={1} isAnimationActive={false} />
                  ))}
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </Card>

          <div className="grid grid-cols-2 gap-4">
            <Card title="Throughput" sub="commits/s">
              <MiniLine data={data} lines={[{ k: "tps", c: "var(--c1)" }]} />
            </Card>
            <Card title="Connections" sub="by state">
              <div className="h-[150px]">
                <ResponsiveContainer>
                  <AreaChart data={data} margin={{ top: 4, right: 4, left: -18, bottom: 0 }}>
                    <CartesianGrid vertical={false} /><XAxis dataKey="t" tickLine={false} axisLine={false} interval={17} /><YAxis tickLine={false} axisLine={false} />
                    <Tooltip {...tooltipStyle} />
                    <Area dataKey="active" stackId="a" stroke="var(--c1)" fill="var(--c1)" fillOpacity={0.5} isAnimationActive={false} />
                    <Area dataKey="idleTx" name="idle in txn" stackId="a" stroke="var(--c5)" fill="var(--c5)" fillOpacity={0.5} isAnimationActive={false} />
                    <Area dataKey="idle" stackId="a" stroke="var(--c6)" fill="var(--c6)" fillOpacity={0.18} isAnimationActive={false} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </Card>
            <Card title="Buffer cache hit ratio" sub="%">
              <MiniLine data={data} lines={[{ k: "hit", c: "var(--c3)" }]} domain={[96, 100]} />
            </Card>
            <Card title="Replication lag" sub="seconds">
              <MiniLine data={data} lines={[{ k: "lag2", c: "var(--c4)", n: "pg-euw-2" }, { k: "lag3", c: "var(--c2)", n: "pg-euw-3" }]} />
            </Card>
          </div>

          <Card title="Active sessions" sub="pg_stat_activity · blocking tree" pad={false} actions={<button className="btn btn-ghost" onClick={() => go("sql")}>Open in SQL</button>}>
            <table className="w-full text-[12px]">
              <thead><tr className="text-fg3 text-left border-b border-line">{["PID", "User / app", "State", "Wait event", "Duration", "Query", ""].map((h) => <th key={h} className="font-medium px-4 py-2">{h}</th>)}</tr></thead>
              <tbody>
                {SESSIONS.map((s) => (
                  <tr key={s.pid} className="border-b border-line last:border-0 hover:bg-hover/50 group">
                    <td className="px-4 py-2 mono tnum">{s.blockedBy ? <span className="text-fg3 mr-1">└</span> : null}{s.pid}</td>
                    <td className="px-4 py-2"><div>{s.user}</div><div className="text-fg3 text-[11px]">{s.app}</div></td>
                    <td className="px-4 py-2"><Badge color={s.state === "active" ? "var(--ok)" : "var(--warn)"}>{s.state}</Badge></td>
                    <td className="px-4 py-2 mono text-[11.5px]" style={{ color: s.wait.startsWith("Lock") ? "var(--crit)" : "var(--fg2)" }}>{s.wait}</td>
                    <td className="px-4 py-2 tnum">{s.dur}</td>
                    <td className="px-4 py-2 mono text-[11.5px] text-fg2 max-w-[300px] truncate">{s.query}</td>
                    <td className="px-4 py-2 text-right whitespace-nowrap">
                      {s.blocking ? <Badge color="var(--crit)" solid>blocking {s.blocking}</Badge> : null}
                      <button className="btn btn-ghost !h-6 ml-1 opacity-0 group-hover:opacity-100" onClick={() => toast({ title: `Cancel requested for pid ${s.pid}`, body: c.env === "prod" ? "Prod: queued for approval by a second operator." : "pg_cancel_backend sent.", tone: c.env === "prod" ? "warn" : "ok" })}><XCircle size={12} /> Cancel</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        </div>

        <div className="space-y-4">
          <Card title={<span className="flex items-center gap-2"><Stethoscope size={14} className="text-accent" />Diagnosis</span>} sub="ranked by load impact" pad={false}>
            <div className="divide-y divide-line">
              {FINDINGS.map((f) => {
                const col = healthColor(f.sev); const isOpen = open === f.id;
                return (
                  <div key={f.id} className="px-4 py-3">
                    <button className="w-full flex items-start gap-2.5 text-left" onClick={() => setOpen(isOpen ? null : f.id)}>
                      <span className="mt-1"><Dot color={col} size={7} /></span>
                      <span className="flex-1 font-medium leading-snug">{f.title}</span>
                      {isOpen ? <ChevronDown size={14} className="text-fg3 mt-0.5" /> : <ChevronRight size={14} className="text-fg3 mt-0.5" />}
                    </button>
                    {isOpen && (
                      <div className="pl-[18px] mt-2 animate-fadeIn">
                        <p className="text-fg2 text-[12.5px] leading-relaxed">{f.detail}</p>
                        {f.fix && (
                          <div className="mt-2.5 rounded-md bg-subtle border border-line p-2.5 relative group">
                            <code className="mono text-[11.5px] block whitespace-pre-wrap pr-6">{f.fix}</code>
                            <button className="absolute top-2 right-2 text-fg3 hover:text-fg" onClick={() => toast({ title: "Copied to clipboard", tone: "ok" })}><Copy size={12} /></button>
                          </div>
                        )}
                        {f.screen && <button className="btn mt-2.5" onClick={() => go(f.screen as Screen)}>{f.screen === "ha" ? <Activity size={12} /> : <Zap size={12} />} Investigate</button>}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </Card>
          <Card title="Top waits" sub="last hour">
            <div className="space-y-2.5">
              {totals.slice().sort((a, b) => b.sum - a.sum).map((t) => {
                const pct = (t.sum / all) * 100;
                return (
                  <div key={t.key}>
                    <div className="flex justify-between text-[12px] mb-1"><span className="mono text-fg2">{t.key}</span><span className="tnum">{pct.toFixed(1)}%</span></div>
                    <div className="h-1.5 rounded-full bg-subtle overflow-hidden"><div className="h-full rounded-full" style={{ width: `${pct}%`, background: t.color }} /></div>
                  </div>
                );
              })}
            </div>
          </Card>
          <Card title="Configuration drift" sub="vs. staging-eu-west">
            <div className="space-y-2 text-[12px]">
              {[["work_mem", "4MB", "16MB"], ["random_page_cost", "4", "1.1"], ["idle_in_transaction_session_timeout", "0", "60s"]].map(([k, a, b]) => (
                <div key={k} className="flex items-center gap-2"><span className="mono flex-1 truncate text-fg2">{k}</span><span className="mono px-1.5 rounded" style={{ background: tint("var(--crit)", 12), color: "var(--crit)" }}>{a}</span><span className="text-fg3">→</span><span className="mono px-1.5 rounded" style={{ background: tint("var(--ok)", 12), color: "var(--ok)" }}>{b}</span></div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}

function MiniLine({ data, lines, domain }: { data: Record<string, number | string>[]; lines: { k: string; c: string; n?: string }[]; domain?: [number, number] }) {
  return (
    <div className="h-[150px]">
      <ResponsiveContainer>
        <LineChart data={data} margin={{ top: 4, right: 4, left: -18, bottom: 0 }}>
          <CartesianGrid vertical={false} /><XAxis dataKey="t" tickLine={false} axisLine={false} interval={17} /><YAxis tickLine={false} axisLine={false} domain={domain ?? ["auto", "auto"]} />
          <Tooltip {...tooltipStyle} />
          {lines.map((l) => <Line key={l.k} dataKey={l.k} name={l.n ?? l.k} stroke={l.c} dot={false} strokeWidth={1.75} isAnimationActive={false} />)}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
