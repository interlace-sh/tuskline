import { ReactNode, useEffect } from "react";
import { X } from "lucide-react";
import { Env, Health, envColor } from "../data";

export const tint = (color: string, pct = 14) => `color-mix(in srgb, ${color} ${pct}%, transparent)`;
export const healthColor = (h: Health) => (h === "ok" ? "var(--ok)" : h === "warn" ? "var(--warn)" : "var(--crit)");
export const fmt = (n: number, d = 0) => n.toLocaleString("en-GB", { maximumFractionDigits: d, minimumFractionDigits: d });
export function fmtBytes(b: number) {
  if (b === 0) return "0 B";
  const u = ["B", "kB", "MB", "GB", "TB"]; let i = 0; let v = b;
  while (v >= 1000 && i < u.length - 1) { v /= 1000; i++; }
  return `${v.toFixed(v < 10 && i > 0 ? 1 : 0)} ${u[i]}`;
}
export function fmtMs(ms: number) {
  if (ms >= 1000) return `${fmt(ms / 1000, 1)} s`;
  if (ms < 1) return `${fmt(ms * 1000, 0)} µs`;
  return `${fmt(ms, ms < 10 ? 1 : 0)} ms`;
}

export function Badge({ children, color = "var(--fg3)", solid, className = "" }: { children: ReactNode; color?: string; solid?: boolean; className?: string }) {
  return (
    <span className={`inline-flex items-center gap-1 h-[19px] px-1.5 rounded-[5px] text-[11px] font-medium whitespace-nowrap ${className}`}
      style={solid ? { background: color, color: "white" } : { background: tint(color, 13), color, boxShadow: `inset 0 0 0 1px ${tint(color, 22)}` }}>
      {children}
    </span>
  );
}
export function EnvBadge({ env, className = "" }: { env: Env; className?: string }) {
  return <Badge color={envColor(env)} className={`uppercase tracking-wide !text-[10px] ${className}`}>{env}</Badge>;
}
export function Dot({ color, pulse, size = 7 }: { color: string; pulse?: boolean; size?: number }) {
  return <span className={`inline-block rounded-full shrink-0 ${pulse ? "pulse-dot" : ""}`} style={{ width: size, height: size, background: color, boxShadow: `0 0 0 3px ${tint(color, 18)}` }} />;
}
export function Kbd({ children }: { children: ReactNode }) { return <kbd className="kbd">{children}</kbd>; }

export function Card({ title, actions, children, className = "", pad = true, sub }: { title?: ReactNode; sub?: ReactNode; actions?: ReactNode; children: ReactNode; className?: string; pad?: boolean }) {
  return (
    <section className={`card ${className}`}>
      {title && (
        <header className="flex items-center justify-between gap-2 px-4 h-11 border-b border-line">
          <div className="flex items-baseline gap-2 min-w-0"><h3 className="font-semibold text-[13px] truncate">{title}</h3>{sub && <span className="text-fg3 text-[12px] truncate">{sub}</span>}</div>
          <div className="flex items-center gap-1.5">{actions}</div>
        </header>
      )}
      <div className={pad ? "p-4" : ""}>{children}</div>
    </section>
  );
}

export function Stat({ label, value, sub, color }: { label: string; value: ReactNode; sub?: ReactNode; color?: string }) {
  return (
    <div className="min-w-0">
      <div className="label mb-1">{label}</div>
      <div className="text-[20px] font-semibold tnum leading-none tracking-tight" style={{ color }}>{value}</div>
      {sub && <div className="text-[11.5px] text-fg3 mt-1.5 truncate">{sub}</div>}
    </div>
  );
}

export function Sparkline({ data, color = "var(--accent)", w = 96, h = 26, fill = true }: { data: number[]; color?: string; w?: number; h?: number; fill?: boolean }) {
  const min = Math.min(...data), max = Math.max(...data);
  const pts = data.map((v, i) => [(i / (data.length - 1)) * w, h - 2 - ((v - min) / (max - min || 1)) * (h - 4)]);
  const d = pts.map((p, i) => `${i ? "L" : "M"}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join("");
  const gid = `g${Math.abs(Math.round(data[0] * 1000 + data.length + w))}${color.length}`;
  return (
    <svg width={w} height={h} className="overflow-visible shrink-0">
      <defs><linearGradient id={gid} x1="0" x2="0" y1="0" y2="1"><stop offset="0" stopColor={color} stopOpacity=".28" /><stop offset="1" stopColor={color} stopOpacity="0" /></linearGradient></defs>
      {fill && <path d={`${d}L${w},${h}L0,${h}Z`} fill={`url(#${gid})`} />}
      <path d={d} fill="none" stroke={color} strokeWidth={1.5} strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  );
}

export function Ring({ value, color, size = 40, stroke = 4 }: { value: number; color: string; size?: number; stroke?: number }) {
  const r = (size - stroke) / 2, c = 2 * Math.PI * r;
  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={r} stroke="var(--line)" strokeWidth={stroke} fill="none" />
        <circle cx={size / 2} cy={size / 2} r={r} stroke={color} strokeWidth={stroke} fill="none" strokeDasharray={c} strokeDashoffset={c * (1 - value / 100)} strokeLinecap="round" style={{ transition: "stroke-dashoffset .6s" }} />
      </svg>
      <span className="absolute inset-0 grid place-items-center text-[11.5px] font-semibold tnum">{value}</span>
    </div>
  );
}

export function Segmented<T extends string>({ value, options, onChange, size = "sm" }: { value: T; options: { value: T; label: ReactNode }[]; onChange: (v: T) => void; size?: "sm" | "xs" }) {
  return (
    <div className="inline-flex p-0.5 rounded-md bg-subtle border border-line">
      {options.map((o) => (
        <button key={o.value} onClick={() => onChange(o.value)}
          className={`${size === "xs" ? "h-[22px] px-2 text-[11.5px]" : "h-6 px-2.5 text-[12px]"} rounded-[5px] font-medium transition-all focus-ring ${value === o.value ? "bg-elev text-fg shadow-sm ring-1 ring-line2" : "text-fg3 hover:text-fg"}`}>
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function Modal({ open, onClose, title, children, footer, width = 520, tone }: { open: boolean; onClose: () => void; title: ReactNode; children: ReactNode; footer?: ReactNode; width?: number; tone?: "danger" }) {
  useEffect(() => {
    if (!open) return;
    const on = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    window.addEventListener("keydown", on);
    return () => window.removeEventListener("keydown", on);
  }, [open, onClose]);
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 grid place-items-center p-6 animate-fadeIn" style={{ background: "rgba(5,7,10,.55)", backdropFilter: "blur(3px)" }} onMouseDown={onClose}>
      <div className="card w-full animate-pop overflow-hidden" style={{ maxWidth: width, boxShadow: "var(--shadow)", borderTop: tone === "danger" ? "2px solid var(--crit)" : undefined }} onMouseDown={(e) => e.stopPropagation()}>
        <header className="flex items-center justify-between px-5 h-12 border-b border-line">
          <h2 className="font-semibold text-[14px]">{title}</h2>
          <button className="btn btn-ghost !px-1.5" onClick={onClose} aria-label="Close"><X size={15} /></button>
        </header>
        <div className="p-5">{children}</div>
        {footer && <footer className="flex justify-end gap-2 px-5 py-3 border-t border-line bg-subtle">{footer}</footer>}
      </div>
    </div>
  );
}

export function Empty({ icon, title, body, action }: { icon: ReactNode; title: string; body: string; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center text-center py-12 px-6">
      <div className="w-10 h-10 rounded-xl grid place-items-center bg-subtle border border-line text-fg3 mb-3">{icon}</div>
      <div className="font-medium">{title}</div>
      <div className="text-fg3 text-[12.5px] mt-1 max-w-xs">{body}</div>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

export function PageHeader({ title, sub, actions, icon }: { title: ReactNode; sub?: ReactNode; actions?: ReactNode; icon?: ReactNode }) {
  return (
    <div className="flex items-end justify-between gap-4 mb-5">
      <div className="min-w-0">
        <h1 className="text-[20px] font-semibold tracking-tight flex items-center gap-2.5">{icon}{title}</h1>
        {sub && <div className="text-fg2 mt-1 text-[13px]">{sub}</div>}
      </div>
      <div className="flex items-center gap-2 shrink-0">{actions}</div>
    </div>
  );
}

export const tooltipStyle = {
  contentStyle: { background: "var(--elev)", border: "1px solid var(--line2)", borderRadius: 8, fontSize: 12, boxShadow: "var(--shadow)", color: "var(--fg)" },
  labelStyle: { color: "var(--fg3)", marginBottom: 4 },
  itemStyle: { padding: 0, color: "var(--fg)" },
};
