import { useRef, useState } from "react";
import { AlertTriangle, Archive, CheckCircle2, Copy, FlaskConical, History, Play, ShieldCheck, XCircle } from "lucide-react";
import { BACKUPS, Backup, WAL_GAP } from "../data";
import { useApp } from "../store";
import { Card, EnvBadge, PageHeader, Stat, tint } from "../components/ui";
import { SqlText } from "./Queries";

const NOW = Date.UTC(2026, 9, 27, 12, 0);
const SPAN = 14 * 24;
const fmtT = (h: number, withYear = false) => {
  const d = new Date(NOW + h * 3600_000);
  return d.toLocaleString("en-GB", { timeZone: "Europe/London", day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", ...(withYear ? { year: "numeric" } : {}) }) ;
};
const iso = (h: number) => new Date(NOW + h * 3600_000).toISOString().replace("T", " ").slice(0, 19) + "+00";
const kindColor = { full: "var(--c1)", diff: "var(--c3)", incr: "var(--c4)" } as const;
const kindLabel = { full: "Full", diff: "Differential", incr: "Incremental" } as const;

export default function Backups() {
  const { cluster: c, toast } = useApp();
  const [target, setTarget] = useState(-30.25);
  const [hover, setHover] = useState<number | null>(null);
  const [dragging, setDragging] = useState(false);
  const trackRef = useRef<HTMLDivElement>(null);
  const toH = (clientX: number) => {
    const r = trackRef.current!.getBoundingClientRect();
    const p = Math.max(0, Math.min(1, (clientX - r.left) / r.width));
    return Math.round((p * SPAN - SPAN) * 4) / 4;
  };
  const x = (h: number) => `${((h + SPAN) / SPAN) * 100}%`;
  const base: Backup = [...BACKUPS].reverse().find((b) => b.start + b.durMin / 60 <= target) ?? BACKUPS[0];
  const fullBase = [...BACKUPS].reverse().find((b) => b.kind === "full" && b.start <= base.start)!;
  const inGap = target >= WAL_GAP.from && target <= WAL_GAP.to;
  const gapBeforeTarget = WAL_GAP.from > base.start && WAL_GAP.to < target;
  const walHours = target - (base.start + base.durMin / 60);
  const estMin = Math.round(fullBase.sizeGb / 9 + walHours * 1.6);
  const cmd = `pgbackrest --stanza=${c.databases[0]} --delta \\\n  --type=time --target="${iso(target)}" \\\n  --target-action=promote --set=${base.id} \\\n  restore`;

  return (
    <div>
      <PageHeader title="Backups & point-in-time recovery" sub={<span className="flex items-center gap-2"><EnvBadge env={c.env} /><span className="mono">{c.name}</span> · pgBackRest 2.56 · stanza <span className="mono">{c.databases[0]}</span> · 2 repositories</span>}
        actions={<><button className="btn" onClick={() => toast({ title: "pgbackrest verify started", body: "Checking checksums for 14 backups and 2,812 WAL segments", tone: "info" })}><ShieldCheck size={13} /> Verify</button><button className="btn btn-primary" onClick={() => toast({ title: "Incremental backup started", body: `${c.name} → repo1 (s3) · ETA 9 min`, tone: "ok" })}><Play size={13} /> Back up now</button></>} />

      <div className="card grid grid-cols-5 divide-x divide-line mb-4">
        <div className="p-4"><Stat label="Last backup" value={`${c.backupAgeH || 5} h ago`} color={c.backupAgeH > 24 ? "var(--crit)" : undefined} sub="incremental · verified" /></div>
        <div className="p-4"><Stat label="Recovery window" value="14 d" sub="since 13 Oct 2026, 06:00" /></div>
        <div className="p-4"><Stat label="RPO (WAL archive lag)" value="2 s" sub="archive_mode = on · async push" /></div>
        <div className="p-4"><Stat label="Repo size" value="1.21 TB" sub="zstd · 3.4× compression" /></div>
        <div className="p-4"><Stat label="Last restore drill" value="Passed" color="var(--ok)" sub="Sun 25 Oct · 41 min to ready" /></div>
      </div>

      <Card title="Timeline" sub="click or drag to choose a recovery target"
        actions={<div className="flex items-center gap-3 text-[11.5px] text-fg3">{(["full", "diff", "incr"] as const).map((k) => <span key={k} className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-sm" style={{ background: kindColor[k] }} />{kindLabel[k]}</span>)}<span className="flex items-center gap-1.5"><span className="w-4 h-1.5 rounded-full" style={{ background: "var(--ok)" }} />WAL</span><span className="flex items-center gap-1.5"><span className="w-2.5 h-1.5 rounded-sm" style={{ background: "var(--crit)" }} />Gap</span></div>}>
        <div className="flex gap-3">
          <div className="w-[88px] shrink-0 text-[11.5px] text-fg3 pt-[22px] space-y-[18px]"><div>Base backups</div><div>WAL archive</div></div>
          <div className="flex-1 select-none">
            <div className="relative h-5 text-[10.5px] text-fg3">
              {Array.from({ length: 15 }, (_, i) => { const h = -SPAN + i * 24; return <span key={i} className="absolute -translate-x-1/2 whitespace-nowrap" style={{ left: x(h) }}>{i === 14 ? "now" : fmtT(h).split(",")[0]}</span>; })}
            </div>
            <div ref={trackRef} className="relative h-[72px] mb-7 rounded-lg border border-line bg-subtle cursor-crosshair"
              onMouseDown={(e) => { setDragging(true); setTarget(toH(e.clientX)); }}
              onMouseMove={(e) => { const h = toH(e.clientX); setHover(h); if (dragging) setTarget(h); }}
              onMouseUp={() => setDragging(false)} onMouseLeave={() => { setHover(null); setDragging(false); }}>
              {Array.from({ length: 13 }, (_, i) => <div key={i} className="absolute top-0 bottom-0 w-px bg-line" style={{ left: x(-SPAN + (i + 1) * 24) }} />)}
              {BACKUPS.map((b) => (
                <div key={b.id} title={`${kindLabel[b.kind]} · ${b.id} · ${b.sizeGb} GB`} className="absolute top-2.5 h-7 rounded-[4px] transition-transform hover:scale-y-110"
                  style={{ left: x(b.start), width: `max(6px, ${(b.durMin / 60 / SPAN) * 100}%)`, background: kindColor[b.kind], opacity: base.id === b.id || fullBase.id === b.id ? 1 : 0.7, boxShadow: base.id === b.id ? "0 0 0 2px var(--elev), 0 0 0 3px var(--fg)" : undefined }}>
                  {!b.verified && <span className="absolute -top-1.5 -right-1.5 w-3 h-3 rounded-full grid place-items-center text-[8px] font-bold text-white" style={{ background: "var(--warn)" }}>!</span>}
                </div>
              ))}
              <div className="absolute h-2 rounded-full" style={{ left: x(-SPAN + 1), right: 0, top: 50, background: tint("var(--ok)", 65) }} />
              <div className="absolute h-3.5 rounded-sm" style={{ left: x(WAL_GAP.from), width: `max(4px, ${((WAL_GAP.to - WAL_GAP.from) / SPAN) * 100}%)`, top: 47, background: "var(--crit)" }} title="WAL gap: archive-push failed (S3 503) 23 Oct 09:30–10:48" />
              {hover !== null && !dragging && <div className="absolute top-0 bottom-0 w-px pointer-events-none" style={{ left: x(hover), background: "var(--fg3)" }} />}
              <div className="absolute -top-1 -bottom-1 w-0.5 pointer-events-none" style={{ left: x(target), background: inGap ? "var(--crit)" : "var(--accent)" }}>
                <div className="absolute -bottom-7 left-1/2 -translate-x-1/2 px-1.5 h-5 rounded text-[10.5px] font-medium text-white whitespace-nowrap grid place-items-center tnum" style={{ background: inGap ? "var(--crit)" : "var(--accent)" }}>{fmtT(target)}</div>
                <div className="absolute -top-1 left-1/2 -translate-x-1/2 w-2.5 h-2.5 rounded-full border-2" style={{ borderColor: inGap ? "var(--crit)" : "var(--accent)", background: "var(--elev)" }} />
              </div>
            </div>
          </div>
        </div>
      </Card>

      <div className="grid grid-cols-[1fr_440px] gap-4 mt-4">
        <Card title={<span className="flex items-center gap-2"><History size={14} className="text-accent" />Recovery plan</span>} sub={`target ${fmtT(target, true)} (Europe/London)`}>
          <div className="flex gap-2 mb-4">
            {[["−1 h", -1], ["−15 m", -0.25], ["+15 m", 0.25], ["+1 h", 1]].map(([l, d]) => <button key={l} className="btn" onClick={() => setTarget((t) => Math.min(0, Math.max(-SPAN + 1, t + (d as number))))}>{l}</button>)}
            <button className="btn btn-ghost" onClick={() => setTarget(-0.05)}>Latest</button>
            <button className="btn btn-ghost" onClick={() => setTarget(-26.75)}>Before migration 0140 (26 Oct 09:15)</button>
          </div>
          {inGap ? (
            <div className="rounded-lg p-3 mb-4 flex gap-2.5 text-[12.5px]" style={{ background: tint("var(--crit)", 10), border: `1px solid ${tint("var(--crit)", 30)}` }}>
              <XCircle size={16} style={{ color: "var(--crit)" }} className="shrink-0" /><div><b style={{ color: "var(--crit)" }}>Target is inside a WAL gap.</b> <span className="text-fg2">WAL from 23 Oct 09:30 to 10:48 is missing from both repositories. Pick a time before 09:30 or after 10:48.</span></div>
            </div>
          ) : gapBeforeTarget ? (
            <div className="rounded-lg p-3 mb-4 flex gap-2.5 text-[12.5px]" style={{ background: tint("var(--warn)", 10), border: `1px solid ${tint("var(--warn)", 30)}` }}>
              <AlertTriangle size={16} style={{ color: "var(--warn)" }} className="shrink-0" /><span className="text-fg2">A WAL gap lies between the base backup and target. Tuskline will use a later base backup.</span>
            </div>
          ) : (
            <div className="rounded-lg p-3 mb-4 flex gap-2.5 text-[12.5px]" style={{ background: tint("var(--ok)", 10), border: `1px solid ${tint("var(--ok)", 25)}` }}>
              <CheckCircle2 size={16} style={{ color: "var(--ok)" }} className="shrink-0" /><span className="text-fg2"><b style={{ color: "var(--ok)" }}>Recoverable.</b> Continuous WAL from base backup to target · all checksums verified.</span>
            </div>
          )}
          <div className="grid grid-cols-4 gap-3 mb-4">
            {[["Base backup", `${kindLabel[base.kind]} · ${fmtT(base.start).split(",")[0]}`], ["Chain", fullBase.id === base.id ? "full only" : `full → ${base.kind}`], ["WAL to replay", `${Math.max(0, walHours).toFixed(1)} h · ≈${Math.round(Math.max(0, walHours) * 11)} GB`], ["Est. time to ready", `≈ ${estMin} min`]].map(([l, v]) => (
              <div key={l} className="rounded-md border border-line p-2.5"><div className="text-fg3 text-[11px]">{l}</div><div className="font-medium mt-0.5 text-[12.5px]">{v}</div></div>
            ))}
          </div>
          <div className="label mb-1.5">Generated command</div>
          <div className="relative rounded-md bg-subtle border border-line p-3 text-[12px] leading-relaxed whitespace-pre"><SqlText sql={cmd} /><button className="absolute top-2 right-2 text-fg3 hover:text-fg" onClick={() => toast({ title: "Copied", tone: "ok" })}><Copy size={12} /></button></div>
          <div className="flex gap-2 mt-4">
            <button className="btn btn-primary" disabled={inGap} onClick={() => toast({ title: "Restore to new cluster queued", body: `${c.name}-pitr-1027 · ${estMin} min · original untouched`, tone: "ok" })}><Archive size={13} /> Restore to new cluster</button>
            <button className="btn" disabled={inGap} onClick={() => toast({ title: "Dry run passed", body: "Manifest and WAL chain resolved; no data written.", tone: "ok" })}><FlaskConical size={13} /> Dry run</button>
            <span className="ml-auto self-center text-[11.5px] text-fg3">In-place restore on prod requires two approvers</span>
          </div>
        </Card>
        <Card title="Backup sets" sub={`${BACKUPS.length} in retention`} pad={false}>
          <div className="max-h-[420px] overflow-auto">
            <table className="w-full text-[12px]">
              <thead className="sticky top-0 bg-elev"><tr className="text-fg3 text-left border-b border-line">{["Set", "Type", "Started", "Size", ""].map((h) => <th key={h} className={`font-medium px-3 py-2 ${h === "Size" ? "text-right" : ""}`}>{h}</th>)}</tr></thead>
              <tbody>
                {[...BACKUPS].reverse().map((b) => (
                  <tr key={b.id} onClick={() => setTarget(Math.min(-0.05, b.start + b.durMin / 60 + 0.25))} className={`border-b border-line last:border-0 cursor-pointer hover:bg-hover/50 ${b.id === base.id ? "bg-hover" : ""}`}>
                    <td className="px-3 py-2 mono text-[11px] whitespace-nowrap">{b.id}</td>
                    <td className="px-3 py-2"><span className="inline-flex items-center gap-1.5 whitespace-nowrap"><span className="w-2 h-2 rounded-sm" style={{ background: kindColor[b.kind] }} />{b.kind === "diff" ? "Diff" : kindLabel[b.kind]}</span></td>
                    <td className="px-3 py-2 text-fg2 tnum whitespace-nowrap">{fmtT(b.start)}</td>
                    <td className="px-3 py-2 tnum text-right whitespace-nowrap">{b.sizeGb} GB</td>
                    <td className="px-3 py-2 text-right">{b.verified ? <CheckCircle2 size={13} style={{ color: "var(--ok)" }} className="inline" /> : <span title="Checksum verification pending"><AlertTriangle size={13} style={{ color: "var(--warn)" }} className="inline" /></span>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </div>
    </div>
  );
}
