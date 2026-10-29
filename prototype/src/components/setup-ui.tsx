import { ReactNode, useState } from "react";
import { Check, Copy, Download, Loader2, AlertTriangle, XCircle, Circle } from "lucide-react";
import { useApp } from "../store";
import { tint } from "./ui";

export function Stepper({ steps, at, onJump }: { steps: string[]; at: number; onJump?: (i: number) => void }) {
  return (
    <ol className="flex items-center gap-1 mb-5" aria-label="Wizard steps">
      {steps.map((s, i) => {
        const done = i < at, cur = i === at;
        return (
          <li key={s} className="flex items-center gap-1 min-w-0">
            <button disabled={!onJump || i > at} onClick={() => onJump?.(i)}
              className={`flex items-center gap-2 h-7 pl-1 pr-2.5 rounded-full text-[12px] font-medium transition-colors ${cur ? "bg-hover text-fg" : done ? "text-fg2 hover:text-fg" : "text-fg3"}`}>
              <span className="w-5 h-5 rounded-full grid place-items-center text-[10.5px] tnum shrink-0"
                style={done ? { background: "var(--brand)", color: "#fff" } : cur ? { boxShadow: "inset 0 0 0 1.5px var(--accent)", color: "var(--accent)" } : { boxShadow: "inset 0 0 0 1px var(--line2)" }}>
                {done ? <Check size={11} strokeWidth={3} /> : i + 1}
              </span>
              <span className="truncate">{s}</span>
            </button>
            {i < steps.length - 1 && <span className="w-5 h-px shrink-0" style={{ background: done ? "var(--accent)" : "var(--line2)" }} />}
          </li>
        );
      })}
    </ol>
  );
}

export function useCopy() {
  const { toast } = useApp();
  return (text: string, what = "Copied to clipboard") => {
    const fallback = () => { const t = document.createElement("textarea"); t.value = text; document.body.appendChild(t); t.select(); try { document.execCommand("copy"); } catch { /* ignore */ } t.remove(); };
    if (navigator.clipboard?.writeText) navigator.clipboard.writeText(text).catch(fallback); else fallback();
    toast({ title: what, tone: "ok" });
  };
}
export function download(name: string, text: string) {
  const a = document.createElement("a"); a.href = URL.createObjectURL(new Blob([text], { type: "text/plain" })); a.download = name; a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000);
}

/** Syntax-light code block: comments dimmed, diff lines coloured. */
export function Code({ text, maxH = 360, diff }: { text: string; maxH?: number; diff?: boolean }) {
  return (
    <pre className="code" style={{ maxHeight: maxH }}>
      {text.split("\n").map((l, i) => {
        let color: string | undefined, bg: string | undefined;
        const t = l.trimStart();
        if (diff && l.startsWith("+") && !l.startsWith("+++")) { color = "var(--ok)"; bg = tint("var(--ok)", 9); }
        else if (diff && l.startsWith("-") && !l.startsWith("---")) { color = "var(--crit)"; bg = tint("var(--crit)", 9); }
        else if (diff && l.startsWith("@@")) color = "var(--accent)";
        else if (t.startsWith("#") || t.startsWith("--") || t.startsWith("//")) color = "var(--fg3)";
        else if (t.startsWith("$ ")) color = "var(--fg)";
        return <div key={i} style={{ color, background: bg }} className="min-h-[1.6em] px-1 -mx-1 rounded-sm">{l || " "}</div>;
      })}
    </pre>
  );
}

export function CodeTabs({ tabs, filename, maxH = 360, right }: { tabs: { id: string; label: string; text: string; diff?: boolean; file?: string }[]; filename?: string; maxH?: number; right?: ReactNode }) {
  const [t, setT] = useState(tabs[0].id);
  const cur = tabs.find((x) => x.id === t) ?? tabs[0];
  const copy = useCopy();
  return (
    <div className="min-w-0">
      <div className="flex items-center gap-1 mb-2 flex-wrap">
        {tabs.map((x) => (
          <button key={x.id} onClick={() => setT(x.id)} className={`h-6 px-2 rounded-md text-[11.5px] mono transition-colors ${cur.id === x.id ? "bg-hover text-fg ring-1 ring-line2" : "text-fg3 hover:text-fg"}`}>{x.label}</button>
        ))}
        <span className="ml-auto flex items-center gap-1">
          {right}
          <button className="btn btn-ghost !h-6 !px-1.5 text-[11.5px]" onClick={() => download(cur.file ?? filename ?? "tuskline.txt", cur.text)} title="Download"><Download size={12} /></button>
          <button className="btn !h-6 !px-2 text-[11.5px]" onClick={() => copy(cur.text, `Copied ${cur.label}`)}><Copy size={12} /> Copy</button>
        </span>
      </div>
      <Code text={cur.text} maxH={maxH} diff={cur.diff} />
    </div>
  );
}

export type CheckState = "pending" | "running" | "ok" | "warn" | "fail";
export function CheckRow({ state, label, detail }: { state: CheckState; label: string; detail?: string }) {
  const icon = { pending: <Circle size={15} className="text-fg3 opacity-50" />, running: <Loader2 size={15} className="spin text-accent" />, ok: <Check size={15} className="text-ok" strokeWidth={2.75} />, warn: <AlertTriangle size={15} className="text-warn" />, fail: <XCircle size={15} className="text-crit" /> }[state];
  return (
    <div className={`flex items-start gap-3 py-2.5 border-b border-line/70 last:border-0 transition-opacity ${state === "pending" ? "opacity-50" : ""}`}>
      <span className="mt-0.5 w-4 grid place-items-center">{icon}</span>
      <div className="min-w-0 flex-1">
        <div className="font-medium text-[13px]">{label}</div>
        {detail && state !== "pending" && state !== "running" && <div className="text-fg3 text-[12px] mt-0.5 mono truncate">{detail}</div>}
      </div>
    </div>
  );
}

export function Field({ label, hint, children, className = "" }: { label: string; hint?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <label className={`block ${className}`}>
      <div className="text-[12px] text-fg2 font-medium mb-1">{label}</div>
      {children}
      {hint && <div className="text-[11.5px] text-fg3 mt-1">{hint}</div>}
    </label>
  );
}
export function Toggle({ on, onChange, label }: { on: boolean; onChange: (b: boolean) => void; label?: ReactNode }) {
  return (
    <button type="button" role="switch" aria-checked={on} onClick={() => onChange(!on)} className="inline-flex items-center gap-2 text-[12.5px] focus-ring rounded">
      <span className="w-7 h-4 rounded-full relative transition-colors" style={{ background: on ? "var(--brand)" : "var(--line2)" }}>
        <span className="absolute top-0.5 w-3 h-3 rounded-full bg-white transition-all" style={{ left: on ? 14 : 2 }} />
      </span>
      {label}
    </button>
  );
}
export function DemoTag() {
  return <span className="text-[10.5px] px-1.5 h-[18px] inline-flex items-center rounded border border-dashed border-line2 text-fg3 font-medium uppercase tracking-wide">fake demo data</span>;
}
