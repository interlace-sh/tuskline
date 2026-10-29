import { useEffect, useState } from "react";
import { AlertTriangle, Check, CheckCircle2, ChevronRight, Circle, Clock, GitBranch, GitPullRequestArrow, Lock, MessageSquare, ShieldAlert, Sparkles, XCircle, Wand2 } from "lucide-react";
import { MIGRATIONS, Migration, envColor } from "../data";
import { useApp } from "../store";
import { Badge, Card, PageHeader, Segmented, tint } from "../components/ui";
import { SqlText } from "./Queries";

const statusColor = { review: "var(--accent)", approved: "var(--ok)", applied: "var(--fg3)", blocked: "var(--crit)" } as const;

const SAFE_SQL = `-- 0142_add_fulfilment_centre.sql  (rewritten by tl migrate lint --fix)
SET lock_timeout = '3s';

ALTER TABLE public.orders
  ADD COLUMN fulfilment_centre_id bigint;

ALTER TABLE public.orders
  ADD CONSTRAINT orders_fulfilment_centre_fk
  FOREIGN KEY (fulfilment_centre_id)
  REFERENCES public.fulfilment_centres (id) NOT VALID;
ALTER TABLE public.orders VALIDATE CONSTRAINT orders_fulfilment_centre_fk;

CREATE INDEX CONCURRENTLY idx_orders_fulfilment_centre
  ON public.orders (fulfilment_centre_id);

ALTER TABLE public.orders
  ADD CONSTRAINT orders_currency_nn CHECK (currency IS NOT NULL) NOT VALID;
ALTER TABLE public.orders VALIDATE CONSTRAINT orders_currency_nn;
ALTER TABLE public.orders ALTER COLUMN currency SET NOT NULL;`;

type DiffLine = [" " | "+" | "-" | "~", string];
const DIFF: DiffLine[] = [
  [" ", "Table \"public.orders\"   ·   48.2 M rows · 21 GB"],
  [" ", "  id                    bigint         not null  pk"],
  [" ", "  customer_id           bigint         not null"],
  [" ", "  status                order_status   not null"],
  [" ", "  total                 numeric(12,2)  not null"],
  ["-", "  currency              char(3)"],
  ["+", "  currency              char(3)        not null"],
  [" ", "  created_at            timestamptz    not null"],
  [" ", "  updated_at            timestamptz    not null"],
  ["+", "  fulfilment_centre_id  bigint"],
  [" ", "Indexes:"],
  [" ", "  \"orders_pkey\" PRIMARY KEY, btree (id)"],
  [" ", "  \"idx_orders_created_at\" btree (created_at)"],
  ["+", "  \"idx_orders_fulfilment_centre\" btree (fulfilment_centre_id)"],
  [" ", "Foreign-key constraints:"],
  [" ", "  \"orders_customer_fk\" FOREIGN KEY (customer_id) REFERENCES customers(id)"],
  ["+", "  \"orders_fulfilment_centre_fk\" FOREIGN KEY (fulfilment_centre_id) REFERENCES fulfilment_centres(id)"],
];

interface Risk { stmt: string; lock: string; dur: string; level: "high" | "med" | "low"; note: string; fix?: string }
const RISKS: Risk[] = [
  { stmt: "ADD COLUMN fulfilment_centre_id", lock: "ACCESS EXCLUSIVE", dur: "< 10 ms", level: "low", note: "Metadata-only (nullable, no default). Brief lock, but queues behind long transactions." },
  { stmt: "ADD CONSTRAINT … FOREIGN KEY", lock: "SHARE ROW EXCLUSIVE", dur: "≈ 48 s", level: "high", note: "Validates 48.2 M rows while blocking writes to orders.", fix: "Add as NOT VALID, then VALIDATE CONSTRAINT separately." },
  { stmt: "CREATE INDEX idx_orders_fulfilment_centre", lock: "SHARE", dur: "≈ 2 m 10 s", level: "high", note: "Blocks INSERT/UPDATE/DELETE on orders for the whole build.", fix: "Use CREATE INDEX CONCURRENTLY." },
  { stmt: "ALTER COLUMN currency SET NOT NULL", lock: "ACCESS EXCLUSIVE", dur: "≈ 19 s", level: "med", note: "Full table scan under ACCESS EXCLUSIVE; blocks reads too.", fix: "Add CHECK (…) NOT VALID, VALIDATE, then SET NOT NULL (PG 12+ skips the scan)." },
];
const SAFE_RISKS: Risk[] = [
  { stmt: "ADD COLUMN fulfilment_centre_id", lock: "ACCESS EXCLUSIVE", dur: "< 10 ms", level: "low", note: "Metadata-only · lock_timeout 3s guards against queueing." },
  { stmt: "ADD CONSTRAINT … NOT VALID + VALIDATE", lock: "SHARE UPDATE EXCL.", dur: "≈ 52 s", level: "low", note: "Validation runs without blocking writes." },
  { stmt: "CREATE INDEX CONCURRENTLY", lock: "SHARE UPDATE EXCL.", dur: "≈ 3 m 40 s", level: "low", note: "Non-blocking. Cannot run inside a transaction block." },
  { stmt: "CHECK NOT VALID → SET NOT NULL", lock: "ACCESS EXCLUSIVE", dur: "< 10 ms", level: "low", note: "Scan skipped thanks to validated CHECK constraint." },
];
const levelColor = { high: "var(--crit)", med: "var(--warn)", low: "var(--ok)" } as const;

function Pipeline({ m }: { m: Migration }) {
  return (
    <div className="flex items-center gap-1.5">
      {m.envs.map((e, i) => {
        const col = e.state === "applied" ? "var(--ok)" : e.state === "failed" || e.state === "drift" ? "var(--crit)" : "var(--fg3)";
        const Icon = e.state === "applied" ? CheckCircle2 : e.state === "drift" ? AlertTriangle : e.state === "failed" ? XCircle : Circle;
        return (
          <div key={e.env} className="flex items-center gap-1.5">
            {i > 0 && <ChevronRight size={12} className="text-fg3" />}
            <span className="flex items-center gap-1.5 h-6 px-2 rounded-md border text-[11.5px]" style={{ borderColor: tint(envColor(e.env), 35), background: tint(envColor(e.env), 8) }}>
              <span className="w-1.5 h-1.5 rounded-full" style={{ background: envColor(e.env) }} /><span className="font-medium">{e.env}</span><Icon size={12} style={{ color: col }} /><span className="text-fg3">{e.state}</span>
            </span>
          </div>
        );
      })}
    </div>
  );
}

export default function Migrations() {
  const { toast } = useApp();
  const [list, setList] = useState(MIGRATIONS);
  const [sel, setSel] = useState("0142");
  const [tab, setTab] = useState<"diff" | "sql">("diff");
  const [safe, setSafe] = useState(false);
  const [approvals, setApprovals] = useState(1);
  const [filter, setFilter] = useState<"open" | "all">("open");
  const m = list.find((x) => x.id === sel)!;
  useEffect(() => { setTab("diff"); }, [sel]);
  const risks = safe ? SAFE_RISKS : RISKS;
  const high = risks.filter((r) => r.level === "high").length, med = risks.filter((r) => r.level === "med").length;
  const overall = high ? "high" : med ? "med" : "low";
  const is142 = m.id === "0142";
  const shown = list.filter((x) => filter === "all" || x.status !== "applied");

  return (
    <div>
      <PageHeader title="Migrations" sub="Schema changes as reviewable change requests · lint, lock analysis and promotion dev → staging → prod"
        actions={<><button className="btn"><GitBranch size={13} /> Connect repo</button><button className="btn btn-primary"><GitPullRequestArrow size={13} /> New migration</button></>} />
      <div className="grid grid-cols-[290px_minmax(0,1fr)_350px] gap-4 items-start">
        <Card pad={false}>
          <div className="p-2 border-b border-line"><Segmented size="xs" value={filter} onChange={setFilter} options={[{ value: "open", label: "Open" }, { value: "all", label: "All" }]} /></div>
          <div className="divide-y divide-line">
            {shown.map((x) => (
              <button key={x.id} onClick={() => setSel(x.id)} className={`w-full text-left px-3 py-3 transition-colors ${sel === x.id ? "bg-hover" : "hover:bg-hover/50"}`} style={sel === x.id ? { boxShadow: "inset 2px 0 0 var(--accent)" } : undefined}>
                <div className="flex items-center gap-2"><span className="mono text-fg3 text-[11px]">{x.id}</span><Badge color={statusColor[x.status]}>{x.status}</Badge><span className="ml-auto text-fg3 text-[11px]">{x.age}</span></div>
                <div className="font-medium mt-1.5 leading-snug">{x.title}</div>
                <div className="text-fg3 text-[11.5px] mt-1 flex items-center gap-1.5">{x.author} · <span className="mono">{x.db}</span>{x.envs.some((e) => e.state === "drift") && <Badge color="var(--crit)">drift</Badge>}</div>
              </button>
            ))}
          </div>
        </Card>

        <div className="space-y-4 min-w-0">
          <Card pad={false}>
            <div className="p-4 border-b border-line">
              <div className="flex items-center gap-2 mb-1"><span className="mono text-fg3">#{m.id}</span><h2 className="font-semibold text-[15px]">{m.title}</h2></div>
              <div className="text-fg3 text-[12px] mb-3">{m.author} wants to apply 4 statements to <span className="mono text-fg2">{m.db}</span> · opened {m.age} ago</div>
              <Pipeline m={m} />
            </div>
            <div className="flex items-center gap-1 px-3 border-b border-line">
              {([["diff", "Schema diff"], ["sql", "Migration SQL"]] as const).map(([id, l]) => (
                <button key={id} onClick={() => setTab(id)} className={`h-9 px-2.5 text-[12.5px] border-b-2 -mb-px ${tab === id ? "border-accent font-medium" : "border-transparent text-fg3 hover:text-fg"}`}>{l}</button>
              ))}
              {is142 && <span className="ml-auto text-[11.5px] text-fg3">{safe ? "showing lint-fixed version" : "as submitted"}</span>}
            </div>
            {tab === "diff" ? (
              is142 ? (
                <div className="mono text-[12px] leading-[22px] py-2 overflow-x-auto">
                  {DIFF.map(([k, t], i) => {
                    const bg = k === "+" ? tint("var(--ok)", 10) : k === "-" ? tint("var(--crit)", 10) : "transparent";
                    const col = k === "+" ? "var(--ok)" : k === "-" ? "var(--crit)" : undefined;
                    const header = t.startsWith("Table") || t.endsWith(":");
                    return (
                      <div key={i} className="flex whitespace-pre" style={{ background: bg }}>
                        <span className="w-10 text-right pr-3 text-fg3 select-none shrink-0">{i + 1}</span>
                        <span className="w-5 shrink-0 select-none" style={{ color: col }}>{k === " " ? "" : k}</span>
                        <span className={header ? "text-fg2 font-medium" : ""} style={{ color: k !== " " ? col : undefined }}>{t}</span>
                      </div>
                    );
                  })}
                </div>
              ) : <div className="p-4 text-[12.5px] whitespace-pre-wrap leading-relaxed"><SqlText sql={m.sql} /></div>
            ) : (
              <div className="p-4 text-[12px] whitespace-pre leading-[20px] overflow-x-auto"><SqlText sql={is142 && safe ? SAFE_SQL : m.sql} /></div>
            )}
          </Card>
          {m.envs.some((e) => e.state === "drift") && (
            <div className="card p-4 flex gap-3" style={{ borderColor: tint("var(--crit)", 40) }}>
              <AlertTriangle size={16} style={{ color: "var(--crit)" }} className="shrink-0 mt-0.5" />
              <div className="text-[12.5px]"><div className="font-medium">Drift detected on staging-eu-west</div><div className="text-fg2 mt-0.5">Type <span className="mono">billing.refund_reason</span> exists with an extra value <span className="mono">'fraud'</span> not in any migration. Someone ran DDL by hand on 22 Oct 2026.</div><div className="flex gap-2 mt-2.5"><button className="btn">Generate reconcile migration</button><button className="btn btn-ghost">View audit log</button></div></div>
            </div>
          )}
          <Card title={<span className="flex items-center gap-2"><MessageSquare size={14} />Review</span>}>
            <div className="space-y-3 text-[12.5px]">
              <div className="flex gap-2.5"><div className="w-6 h-6 rounded-full grid place-items-center text-[10px] font-semibold text-white shrink-0" style={{ background: "#8E4EC6" }}>TB</div><div><span className="font-medium">tom.b</span> <span className="text-fg3">approved · 1 h ago</span><div className="text-fg2 mt-0.5">Looks good on staging, took 2m 03s. Happy for prod in the Thursday window.</div></div></div>
              <div className="flex gap-2.5"><div className="w-6 h-6 rounded-full grid place-items-center text-[10px] font-semibold text-white shrink-0" style={{ background: "#3E63DD" }}><Sparkles size={11} /></div><div><span className="font-medium">Tuskline lint</span> <span className="text-fg3">bot · 2 h ago</span><div className="text-fg2 mt-0.5">{safe ? "All findings resolved by suggested rewrite." : "3 findings would block writes to a 21 GB table on prod. See risk panel."}</div></div></div>
            </div>
          </Card>
        </div>

        <div className="space-y-4">
          <Card title="Risk assessment" sub="prod-eu-west · orders" actions={<Badge color={levelColor[overall]} solid>{overall === "high" ? "High risk" : overall === "med" ? "Medium" : "Low risk"}</Badge>} pad={false}>
            {is142 ? (
              <>
                <div className="divide-y divide-line">
                  {risks.map((r) => (
                    <div key={r.stmt} className="px-4 py-3 animate-fadeIn">
                      <div className="flex items-start gap-2"><span className="mt-1.5 w-1.5 h-1.5 rounded-full shrink-0" style={{ background: levelColor[r.level] }} /><span className="mono text-[11.5px] flex-1 leading-snug">{r.stmt}</span></div>
                      <div className="flex flex-wrap items-center gap-1.5 mt-1.5 pl-3.5">
                        <Badge color={r.lock === "ACCESS EXCLUSIVE" ? "var(--crit)" : r.lock.startsWith("SHARE U") ? "var(--ok)" : "var(--warn)"}><Lock size={10} />{r.lock}</Badge>
                        <Badge><Clock size={10} />{r.dur}</Badge>
                      </div>
                      <div className="text-fg2 text-[12px] mt-1.5 pl-3.5">{r.note}</div>
                      {r.fix && <div className="text-[12px] mt-1 pl-3.5 flex gap-1" style={{ color: "var(--accent)" }}><Wand2 size={12} className="mt-0.5 shrink-0" />{r.fix}</div>}
                    </div>
                  ))}
                </div>
                <div className="p-4 border-t border-line space-y-2">
                  <div className="flex justify-between text-[12px]"><span className="text-fg3">Est. total duration</span><span className="tnum font-medium">{safe ? "≈ 4 m 32 s (online)" : "≈ 3 m 17 s (writes blocked)"}</span></div>
                  <div className="flex justify-between text-[12px]"><span className="text-fg3">Table rewrite</span><span>No</span></div>
                  <div className="flex justify-between text-[12px]"><span className="text-fg3">Backup taken before apply</span><span style={{ color: "var(--ok)" }}>Yes (auto)</span></div>
                  {!safe && <button className="btn w-full justify-center mt-2" onClick={() => { setSafe(true); setTab("sql"); toast({ title: "Applied lint fixes", body: "3 statements rewritten to online-safe equivalents.", tone: "ok" }); }}><Wand2 size={13} /> Apply suggested rewrite</button>}
                </div>
              </>
            ) : <div className="p-4 text-fg3 text-[12.5px]">{m.status === "applied" ? "Applied to all environments. No outstanding risk." : "Low risk · metadata-only changes."}</div>}
          </Card>
          <Card title="Approvals" sub="prod requires 2">
            <div className="flex items-center gap-2 mb-3">
              {[0, 1].map((i) => <div key={i} className="flex-1 h-1.5 rounded-full" style={{ background: i < approvals ? "var(--ok)" : "var(--line)" }} />)}
              <span className="text-[12px] tnum text-fg2">{approvals}/2</span>
            </div>
            {m.status === "review" && is142 ? (
              <div className="space-y-2">
                {high > 0 && <div className="text-[12px] flex gap-1.5" style={{ color: "var(--warn)" }}><ShieldAlert size={13} className="shrink-0 mt-0.5" />Policy: high-risk statements must be fixed or explicitly overridden.</div>}
                <div className="flex gap-2">
                  <button className="btn flex-1 justify-center" onClick={() => toast({ title: "Changes requested", tone: "warn" })}>Request changes</button>
                  <button className="btn btn-primary flex-1 justify-center" disabled={high > 0} onClick={() => { setApprovals(2); setList((l) => l.map((x) => (x.id === "0142" ? { ...x, status: "approved" } : x))); toast({ title: "Approved #0142", body: "Scheduled for prod window Thu 29 Oct 2026, 06:00 GMT", tone: "ok" }); }}><Check size={13} /> Approve</button>
                </div>
              </div>
            ) : m.status === "approved" ? (
              <div className="text-[12.5px] flex items-center gap-2" style={{ color: "var(--ok)" }}><CheckCircle2 size={14} /> Approved · <span className="mono text-fg2">tl migrate apply --env prod</span></div>
            ) : <div className="text-fg3 text-[12.5px]">No action needed.</div>}
          </Card>
        </div>
      </div>
    </div>
  );
}
