import { useMemo, useState } from "react";
import { ChevronDown, ChevronRight, Cpu, Download, Gauge, HardDrive, MemoryStick, RotateCw, Shield, SlidersHorizontal, Zap } from "lucide-react";
import { useApp } from "../store";
import { Badge, Card, Modal, PageHeader, Ring, Segmented, tint } from "../components/ui";
import { CodeTabs, DemoTag, download } from "../components/setup-ui";
import { CATEGORIES, CONF_FINDINGS, ConfFinding, HOST } from "../setup-data";

const W = { crit: 30, warn: 15, info: 5 } as const;
const sevColor = (s: ConfFinding["sev"]) => (s === "crit" ? "var(--crit)" : s === "warn" ? "var(--warn)" : "var(--accent)");
const scoreColor = (n: number) => (n >= 85 ? "var(--ok)" : n >= 60 ? "var(--warn)" : "var(--crit)");
const CAT_LABEL: Record<string, string> = { Memory: "Memory & planner", WAL: "WAL", Checkpoints: "Checkpoints", Autovacuum: "Autovacuum", Connections: "Connections" };

export default function Advisor() {
  const { cluster, toast } = useApp();
  const [profile, setProfile] = useState<"oltp" | "mixed" | "olap">("oltp");
  const [cat, setCat] = useState<string>("all");
  const [sel, setSel] = useState<string[]>(CONF_FINDINGS.map((f) => f.id));
  const [applied, setApplied] = useState<string[]>([]);
  const [open, setOpen] = useState<string | null>("c1");
  const [modal, setModal] = useState(false);
  const [typed, setTyped] = useState("");
  const [ticket, setTicket] = useState("");
  const prod = cluster.env === "prod";
  const managed = cluster.provider === "AWS" && cluster.ha === "RDS Multi-AZ";
  const patroni = cluster.ha === "Patroni";

  const pending = CONF_FINDINGS.filter((f) => !applied.includes(f.id));
  const chosen = pending.filter((f) => sel.includes(f.id));
  const catScore = (c: string) => Math.max(0, 100 - pending.filter((f) => f.cat === c).reduce((a, f) => a + W[f.sev], 0));
  const overall = Math.round(CATEGORIES.reduce((a, c) => a + catScore(c), 0) / CATEGORIES.length);
  const shown = pending.filter((f) => cat === "all" || f.cat === cat);
  const restartN = chosen.filter((f) => f.apply === "restart").length;

  const outputs = useMemo(() => {
    const diff = `--- a/postgresql.conf   (${cluster.name}, current)\n+++ b/postgresql.conf   (recommended, ${HOST.profile} · ${HOST.ramGb} GB · ${HOST.vcpu} vCPU)\n` +
      CATEGORIES.map((c) => { const fs = chosen.filter((f) => f.cat === c); return fs.length ? `@@ ${CAT_LABEL[c]} @@\n` + fs.map((f) => `-${f.param} = ${f.current}\n+${f.param} = ${f.rec}${" ".repeat(Math.max(1, 30 - f.param.length - f.rec.length))}# ${f.apply}`).join("\n") : ""; }).filter(Boolean).join("\n");
    const alter = `-- ALTER SYSTEM writes postgresql.auto.conf. ${patroni ? "This cluster is Patroni-managed: prefer the patronictl tab." : ""}\n` +
      chosen.map((f) => `ALTER SYSTEM SET ${f.param} = '${f.rec}';${f.apply === "restart" ? "   -- needs restart" : ""}`).join("\n") +
      `\n\nSELECT pg_reload_conf();\n-- Still pending a restart:\nSELECT name, setting FROM pg_settings WHERE pending_restart;`;
    const patch = `# tuskline.conf — include from postgresql.conf:  include_if_exists = 'tuskline.conf'\n` + chosen.map((f) => `${f.param} = '${f.rec}'${" ".repeat(Math.max(1, 34 - f.param.length - f.rec.length))}# was ${f.current} · ${f.apply}`).join("\n");
    const t = [
      { id: "diff", label: "diff", text: diff || "# Nothing selected", diff: true, file: "postgresql.conf.diff" },
      { id: "alter", label: "ALTER SYSTEM", text: alter, file: "tuskline-alter-system.sql" },
      { id: "patch", label: "conf patch", text: patch, file: "tuskline.conf" },
    ];
    if (patroni) t.push({ id: "patroni", label: "patronictl", text: `$ patronictl -c /etc/patroni/patroni.yml edit-config ${cluster.name} \\\n${chosen.map((f) => `    -s 'postgresql.parameters.${f.param}=${f.rec}'`).join(" \\\n")}\n$ patronictl restart ${cluster.name} --pending   # only if restart-type changes`, file: "patroni.sh", diff: false });
    if (managed) t.push({ id: "aws", label: "aws cli", text: `$ aws rds modify-db-parameter-group --db-parameter-group-name demo-${cluster.name}-pg17 \\\n    --parameters \\\n${chosen.map((f) => `      "ParameterName=${f.param},ParameterValue=${f.rec},ApplyMethod=${f.apply === "restart" ? "pending-reboot" : "immediate"}"`).join(" \\\n")}\n# RDS expects some memory parameters in 8 kB units: Tuskline converts on apply.`, file: "rds.sh", diff: false });
    return t;
  }, [chosen, cluster.name, patroni, managed]);

  const doApply = () => {
    setModal(false); setTyped(""); setTicket("");
    const ids = chosen.map((f) => f.id);
    setApplied((a) => [...a, ...ids.filter((id) => CONF_FINDINGS.find((f) => f.id === id)!.apply === "reload")]);
    toast({ title: `Applied ${chosen.length - restartN} reload change${chosen.length - restartN === 1 ? "" : "s"} (demo)`, body: restartN ? `${restartN} restart change${restartN === 1 ? "" : "s"} scheduled for the maintenance window, Sun 02:00 Europe/London` : "pg_reload_conf() succeeded on all members", tone: "ok" });
  };
  const canConfirm = !prod || (typed === cluster.name && ticket.trim().length > 2);

  return (
    <div>
      <PageHeader icon={<SlidersHorizontal size={18} className="text-accent" />} title="Config advisor"
        sub={<span className="flex items-center gap-2 flex-wrap"><span className="mono">{cluster.name}</span><span className="text-fg3">·</span><span className="flex items-center gap-1 text-fg3"><Cpu size={12} />{HOST.vcpu} vCPU</span><span className="flex items-center gap-1 text-fg3"><MemoryStick size={12} />{HOST.ramGb} GB</span><span className="flex items-center gap-1 text-fg3"><HardDrive size={12} />{HOST.storage}</span><DemoTag /></span>}
        actions={<><Segmented value={profile} onChange={setProfile} options={[{ value: "oltp", label: "OLTP" }, { value: "mixed", label: "Mixed" }, { value: "olap", label: "Analytics" }]} />
          <button className="btn" onClick={() => toast({ title: "Re-analysed settings (demo)", body: "pg_settings, pg_stat_bgwriter, pg_stat_checkpointer and 30 days of metrics", tone: "info" })}><RotateCw size={13} /> Re-analyse</button></>} />

      <div className="card p-4 mb-4 flex items-center gap-6">
        <div className="flex items-center gap-3 pr-6 border-r border-line">
          <Ring value={overall} color={scoreColor(overall)} size={56} stroke={5} />
          <div><div className="label">Config score</div><div className="text-[12.5px] text-fg2 mt-0.5">{pending.length} findings · {pending.filter((f) => f.sev === "crit").length} critical</div></div>
        </div>
        <div className="flex-1 grid grid-cols-5 gap-5">
          {CATEGORIES.map((c) => {
            const s = catScore(c); const n = pending.filter((f) => f.cat === c).length;
            return (
              <button key={c} onClick={() => setCat(cat === c ? "all" : c)} className={`text-left rounded-md p-1.5 -m-1.5 transition-colors ${cat === c ? "bg-hover" : "hover:bg-hover/50"}`}>
                <div className="flex items-center justify-between text-[12px]"><span className="font-medium">{CAT_LABEL[c]}</span><span className="tnum font-semibold" style={{ color: scoreColor(s) }}>{s}</span></div>
                <div className="h-1.5 rounded-full bg-subtle mt-1.5 overflow-hidden"><div className="h-full rounded-full transition-all duration-500" style={{ width: `${s}%`, background: scoreColor(s) }} /></div>
                <div className="text-[11px] text-fg3 mt-1">{n ? `${n} finding${n > 1 ? "s" : ""}` : "no issues"}</div>
              </button>
            );
          })}
        </div>
      </div>

      <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-4 items-start">
        <Card title="Findings" sub={cat === "all" ? "all categories" : CAT_LABEL[cat]} pad={false}
          actions={<button className="btn btn-ghost !h-6 text-[11.5px]" onClick={() => setSel(sel.length === pending.length ? [] : pending.map((f) => f.id))}>{sel.length >= pending.length ? "Select none" : "Select all"}</button>}>
          {shown.length === 0 && <div className="p-8 text-center text-fg3">Nothing left to fix here.</div>}
          {shown.map((f) => {
            const isOpen = open === f.id;
            return (
              <div key={f.id} className="border-b border-line/70 last:border-0">
                <div className="flex items-center gap-3 px-4 py-2.5 hover:bg-hover/40">
                  <input type="checkbox" className="accent-[var(--brand)]" checked={sel.includes(f.id)} onChange={() => setSel(sel.includes(f.id) ? sel.filter((x) => x !== f.id) : [...sel, f.id])} aria-label={`Include ${f.param}`} />
                  <button className="flex-1 min-w-0 flex items-center gap-2.5 text-left" onClick={() => setOpen(isOpen ? null : f.id)}>
                    <span className="w-1.5 h-1.5 rounded-full shrink-0" style={{ background: sevColor(f.sev) }} />
                    <span className="mono text-[12.5px] font-medium truncate">{f.param}</span>
                    <span className="mono text-[12px] text-fg3 line-through decoration-fg3/50">{f.current}</span>
                    <span className="text-fg3">→</span>
                    <span className="mono text-[12px] font-medium" style={{ color: "var(--ok)" }}>{f.rec}</span>
                    <span className="ml-auto flex items-center gap-1.5">
                      <Badge color={f.apply === "restart" ? "var(--warn)" : "var(--fg3)"}>{f.apply === "restart" ? <RotateCw size={10} /> : <Zap size={10} />}{f.apply}</Badge>
                      {isOpen ? <ChevronDown size={14} className="text-fg3" /> : <ChevronRight size={14} className="text-fg3" />}
                    </span>
                  </button>
                </div>
                {isOpen && (
                  <div className="px-4 pb-3 pl-[52px] animate-fadeIn">
                    <div className="text-[12.5px] text-fg2 leading-relaxed">{f.why}</div>
                    <div className="mt-1.5 text-[11.5px] text-fg3 flex items-center gap-1.5"><Gauge size={11} /> {f.evidence}</div>
                  </div>
                )}
              </div>
            );
          })}
        </Card>
        <div className="sticky top-4 space-y-3">
          <Card title="Recommended changes" sub={`${chosen.length} selected · ${restartN} need restart`}>
            <CodeTabs key={chosen.map((f) => f.id).join()} tabs={outputs} maxH={340} />
          </Card>
          <div className="flex items-center gap-2">
            <button className="btn" onClick={() => download(`${cluster.name}-tuskline.conf`, outputs[2].text)}><Download size={13} /> Export conf patch</button>
            <button className="btn" onClick={() => download(`${cluster.name}-alter-system.sql`, outputs[1].text)}><Download size={13} /> Export SQL</button>
            <button className={`btn ${prod ? "btn-danger" : "btn-primary"} ml-auto`} disabled={!chosen.length} onClick={() => setModal(true)}>{prod && <Shield size={13} />} Apply {chosen.length} to {cluster.env}…</button>
          </div>
        </div>
      </div>

      <Modal open={modal} onClose={() => setModal(false)} tone={prod ? "danger" : undefined} width={540}
        title={<span className="flex items-center gap-2">{prod && <Shield size={15} className="text-crit" />}Apply {chosen.length} setting changes to {cluster.name}</span>}
        footer={<><button className="btn" onClick={() => setModal(false)}>Cancel</button><button className={`btn ${prod ? "btn-danger" : "btn-primary"}`} disabled={!canConfirm} onClick={doApply}>{prod ? "Apply to production" : "Apply"}</button></>}>
        <div className="space-y-3 text-[12.5px]">
          <div className="grid grid-cols-2 gap-2">
            <div className="rounded-lg border border-line p-3"><div className="label mb-1">Reload now</div><div className="text-[18px] font-semibold tnum">{chosen.length - restartN}</div><div className="text-fg3 text-[11.5px]">pg_reload_conf(), no downtime</div></div>
            <div className="rounded-lg border p-3" style={{ borderColor: tint("var(--warn)", 40) }}><div className="label mb-1">Restart required</div><div className="text-[18px] font-semibold tnum" style={{ color: "var(--warn)" }}>{restartN}</div><div className="text-fg3 text-[11.5px]">scheduled · Sun 02:00 Europe/London</div></div>
          </div>
          {prod ? (
            <>
              <div className="rounded-md p-2.5 text-[12px]" style={{ background: tint("var(--crit)", 10), color: "var(--crit)" }}>Production change. A second approver is notified and every statement is audited. Restart-type settings roll replicas first, then switch over.</div>
              <label className="block"><div className="text-fg2 mb-1">Change ticket</div><input className="input w-full" placeholder="CHG-2041 (demo)" value={ticket} onChange={(e) => setTicket(e.target.value)} /></label>
              <label className="block"><div className="text-fg2 mb-1">Type <span className="mono text-fg">{cluster.name}</span> to confirm</div><input className="input w-full mono" value={typed} onChange={(e) => setTyped(e.target.value)} /></label>
            </>
          ) : <div className="text-fg2">Non-production: changes apply immediately to all members. (Prototype: nothing is changed.)</div>}
        </div>
      </Modal>
    </div>
  );
}
