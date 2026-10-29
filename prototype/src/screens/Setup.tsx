import { useState } from "react";
import { ArrowRight, CheckCircle2, Circle, History, PlugZap, Rocket, SlidersHorizontal, Upload, XCircle, Loader2, RotateCcw, Terminal } from "lucide-react";
import { useApp, Screen } from "../store";
import { ENVS } from "../data";
import { Badge, Card, Kbd, PageHeader, tint } from "../components/ui";
import { DemoTag, Code } from "../components/setup-ui";
import { DeployStatus, TARGET_LABEL, Deployment } from "../setup-data";
import { TuskMark } from "../components/Brand";

export const STATUS: Record<DeployStatus, { label: string; color: string; icon: typeof Circle }> = {
  succeeded: { label: "Succeeded", color: "var(--ok)", icon: CheckCircle2 },
  failed: { label: "Failed", color: "var(--crit)", icon: XCircle },
  running: { label: "Running", color: "var(--accent)", icon: Loader2 },
  "rolled-back": { label: "Rolled back", color: "var(--warn)", icon: RotateCcw },
};
const dur = (s: number) => (s >= 60 ? `${Math.floor(s / 60)} m ${s % 60} s` : `${s} s`);

export default function Setup() {
  const { go, deployments, cluster } = useApp();
  const actions: { id: Screen; icon: typeof Rocket; title: string; body: string; keys: string[]; cta: string; primary?: boolean }[] = [
    { id: "connect", icon: PlugZap, title: "Connect an existing database", body: "Host, SSH tunnel, TLS or cloud IAM. Tests the connection, detects RDS / Cloud SQL / Azure / Patroni and generates the setup SQL.", keys: ["n"], cta: "Add connection", primary: true },
    { id: "deploy", icon: Rocket, title: "Deploy a new cluster", body: "Docker Compose, Kubernetes (CloudNativePG) or VMs over SSH. Primary + replicas, etcd, PgBouncer and pgBackRest, previewed as files.", keys: ["g", "d"], cta: "Start deploy wizard" },
    { id: "advisor", icon: SlidersHorizontal, title: "Review configuration", body: `Scored findings for memory, WAL, checkpoints, autovacuum and connections on ${cluster.name}, with a diff you can apply or export.`, keys: ["g", "v"], cta: "Open config advisor" },
  ];
  const checklist: [string, boolean, Screen][] = [["Connect your first database", true, "connect"], ["Install monitoring extensions", false, "connect"], ["Create the tuskline_monitor role", true, "connect"], ["Review configuration", false, "advisor"], ["Configure backups and test a restore", false, "backups"]];
  const doneN = checklist.filter((c) => c[1]).length;
  return (
    <div>
      <PageHeader icon={<TuskMark size={24} />} title="Setup & deploy" sub={<>Connect what you already run, or stand up a new highly available cluster. <DemoTag /></>}
        actions={<button className="btn" onClick={() => go("deployments")}><History size={13} /> Deployments</button>} />
      <div className="grid grid-cols-3 gap-3 mb-5">
        {actions.map((a) => {
          const Icon = a.icon;
          return (
            <button key={a.id} onClick={() => go(a.id)} className="card p-5 text-left group hover:border-line2 hover:-translate-y-px transition-all flex flex-col focus-ring">
              <div className="flex items-center justify-between">
                <div className="w-10 h-10 rounded-xl grid place-items-center" style={{ background: tint("var(--accent)", 14), color: "var(--accent)" }}><Icon size={19} /></div>
                <span className="flex gap-0.5">{a.keys.map((k) => <Kbd key={k}>{k}</Kbd>)}</span>
              </div>
              <div className="font-semibold text-[14.5px] mt-4">{a.title}</div>
              <div className="text-fg2 text-[12.5px] mt-1.5 leading-relaxed flex-1">{a.body}</div>
              <span className={`btn mt-4 self-start ${a.primary ? "btn-primary" : ""}`}>{a.cta} <ArrowRight size={13} className="group-hover:translate-x-0.5 transition-transform" /></span>
            </button>
          );
        })}
      </div>
      <div className="grid grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)] gap-4 items-start">
        <Card title="Recent deployments" pad={false} actions={<button className="btn btn-ghost !h-6 text-[11.5px]" onClick={() => go("deployments")}>View all <ArrowRight size={12} /></button>}>
          <DeployTable rows={deployments.slice(0, 5)} compact />
        </Card>
        <div className="space-y-4">
          <Card title="Getting started" sub={`${doneN} of ${checklist.length}`}>
            <div className="h-1.5 rounded-full bg-subtle overflow-hidden mb-3"><div className="h-full rounded-full" style={{ width: `${(doneN / checklist.length) * 100}%`, background: "var(--brand)" }} /></div>
            {checklist.map(([l, d, s]) => (
              <button key={l} onClick={() => go(s)} className="w-full flex items-center gap-2.5 h-8 px-1.5 -mx-1.5 rounded-md hover:bg-hover/60 text-left">
                {d ? <CheckCircle2 size={15} className="text-ok" /> : <Circle size={15} className="text-fg3" />}
                <span className={d ? "text-fg3 line-through" : ""}>{l}</span>
              </button>
            ))}
          </Card>
          <Card title="Or from the terminal">
            <Code text={"$ tl connect add --interactive\n$ tl deploy init --target compose\n$ tl deploy plan && tl deploy apply\n$ tl advise config --cluster " + cluster.name} maxH={140} />
            <div className="text-[11.5px] text-fg3 mt-2 flex items-center gap-1.5"><Terminal size={11} /> Same engine, same files. <button className="underline hover:text-fg" onClick={() => go("fleet")}><Upload size={10} className="inline" /> Import inventory</button></div>
          </Card>
        </div>
      </div>
    </div>
  );
}

export function DeployTable({ rows, onOpen, openId, compact, noTopology }: { rows: Deployment[]; onOpen?: (d: Deployment) => void; openId?: string | null; compact?: boolean; noTopology?: boolean }) {
  const cols = compact ? ["Status", "Cluster", "Target", "Started"] : ["Status", "Cluster", "Target", ...(noTopology ? [] : ["Topology"]), "Started", "Duration", "By"];
  return (
    <table className="w-full text-[12.5px]">
      <thead><tr className="text-left text-fg3 border-b border-line">{cols.map((h) => <th key={h} className="font-medium label px-4 h-8">{h}</th>)}</tr></thead>
      <tbody>
        {rows.map((d) => {
          const st = STATUS[d.status]; const Icon = st.icon; const env = ENVS.find((e) => e.id === d.env)!;
          return (
            <tr key={d.id} onClick={() => onOpen?.(d)} className={`border-b border-line/60 last:border-0 ${onOpen ? "cursor-pointer hover:bg-hover/50" : ""} ${openId === d.id ? "bg-hover" : ""}`}>
              <td className="px-4 py-2.5"><Badge color={st.color}><Icon size={11} className={d.status === "running" ? "spin" : ""} />{st.label}</Badge></td>
              <td className="px-4"><div className="flex items-center gap-2"><span className="w-1 h-3.5 rounded-full" style={{ background: env.color }} /><span className="mono whitespace-nowrap">{d.name}</span></div></td>
              <td className="px-4 text-fg2 whitespace-nowrap">{TARGET_LABEL[d.target].replace("Kubernetes · ", "")}</td>
              {!compact && !noTopology && <td className="px-4 text-fg3 truncate max-w-[220px]">{d.topology}</td>}
              <td className="px-4 text-fg2 tnum whitespace-nowrap">{d.startedAt}</td>
              {!compact && <td className="px-4 text-fg2 tnum whitespace-nowrap">{d.status === "running" ? "…" : dur(d.durationS)}</td>}
              {!compact && <td className="px-4 text-fg3 whitespace-nowrap">{d.by}</td>}
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

export function Deployments() {
  const { deployments, go } = useApp();
  const [open, setOpen] = useState<string | null>(deployments.find((d) => d.status !== "succeeded")?.id ?? null);
  const [f, setF] = useState<"all" | DeployStatus>("all");
  const rows = deployments.filter((d) => f === "all" || d.status === f);
  const cur = deployments.find((d) => d.id === open);
  const counts = (s: DeployStatus) => deployments.filter((d) => d.status === s).length;
  return (
    <div>
      <PageHeader icon={<History size={18} className="text-accent" />} title="Deployments" sub={<>Every deploy, upgrade and rollback, with its plan and log. Times Europe/London. <DemoTag /></>}
        actions={<button className="btn btn-primary" onClick={() => go("deploy")}><Rocket size={13} /> New deployment</button>} />
      <div className="flex items-center gap-1.5 mb-3">
        {(["all", "succeeded", "running", "failed", "rolled-back"] as const).map((s) => (
          <button key={s} onClick={() => setF(s)} className={`h-7 px-2.5 rounded-md text-[12px] font-medium ${f === s ? "bg-hover text-fg ring-1 ring-line2" : "text-fg3 hover:text-fg"}`}>
            {s === "all" ? `All ${deployments.length}` : `${STATUS[s].label} ${counts(s)}`}
          </button>
        ))}
      </div>
      <div className="grid grid-cols-[minmax(0,1fr)_360px] gap-4 items-start">
        <div className="card overflow-hidden"><DeployTable rows={rows} onOpen={(d) => setOpen(d.id)} openId={open} noTopology /></div>
        {cur ? (
          <Card title={<span className="mono">{cur.id}</span>} sub={cur.name} className="sticky top-4">
            <div className="flex items-center gap-2 mb-3"><Badge color={STATUS[cur.status].color}>{STATUS[cur.status].label}</Badge><span className="text-fg3 text-[12px]">{TARGET_LABEL[cur.target]} · PG {cur.pg}</span></div>
            <dl className="text-[12px] mb-3 space-y-1">{([["Topology", cur.topology], ["Started", `${cur.startedAt} Europe/London`], ["By", cur.by]] as [string, string][]).map(([k, v]) => <div key={k} className="flex gap-3"><dt className="w-16 text-fg3 shrink-0">{k}</dt><dd className="min-w-0">{v}</dd></div>)}</dl>
            {cur.note && <div className="rounded-md p-2.5 text-[12px] mb-3" style={{ background: tint(STATUS[cur.status].color, 10), color: STATUS[cur.status].color }}>{cur.note}</div>}
            <Code text={(cur.log ?? [`[Pre-flight] all checks passed`, `[Plan] ${cur.topology}`, cur.status === "failed" ? "[Consensus] etcd3: waiting for quorum… timeout after 300 s" : cur.status === "rolled-back" ? "[Rolling] pg-stg-2 health check failed → rollback" : "[Hand-over] registered in Fleet", cur.status === "succeeded" ? "deployment complete" : "deployment stopped; no data was changed on the leader"]).join("\n")} maxH={300} />
            <div className="flex gap-2 mt-3">
              <button className="btn" onClick={() => go("deploy")}><RotateCcw size={13} /> {cur.status === "failed" ? "Retry with same plan" : "Redeploy"}</button>
            </div>
          </Card>
        ) : <div className="card p-6 text-fg3 text-center">Select a deployment to see its log.</div>}
      </div>
    </div>
  );
}
