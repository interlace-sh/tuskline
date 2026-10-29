import { useEffect, useMemo, useRef, useState } from "react";
import { PlugZap, Rocket, SlidersHorizontal, History, Wrench, Container, Ship, ExternalLink } from "lucide-react";
import { TuskMark, TuskGlyph, Wordmark } from "./Brand";
import { Activity, Archive, ArrowRight, CheckCircle2, AlertTriangle, XCircle, Info, GitPullRequestArrow, LayoutGrid, Moon, Network, Shield, Stethoscope, Terminal, Zap, Keyboard, RefreshCw, Server, FileCode2 } from "lucide-react";
import { useApp, Screen } from "../store";
import { CLUSTERS, ENVS, QUERIES } from "../data";
import { Dot, Kbd, Modal, healthColor } from "./ui";

interface Cmd { id: string; group: string; label: string; hint?: string; icon: typeof Activity; keys?: string[]; run: () => void; color?: string }

export function CommandPalette() {
  const app = useApp();
  const { paletteOpen, setPaletteOpen, go, setClusterId, setEnv, toggleTheme, toast, setHelpOpen } = app;
  const [q, setQ] = useState("");
  const [i, setI] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  useEffect(() => { if (paletteOpen) { setQ(""); setI(0); setTimeout(() => inputRef.current?.focus(), 0); } }, [paletteOpen]);

  const cmds = useMemo<Cmd[]>(() => {
    const nav = (s: Screen) => () => go(s);
    return [
      { id: "n-fleet", group: "Navigate", label: "Fleet overview", icon: LayoutGrid, keys: ["g", "f"], run: nav("fleet") },
      { id: "n-cluster", group: "Navigate", label: "Cluster health", icon: Activity, keys: ["g", "c"], run: nav("cluster") },
      { id: "n-queries", group: "Navigate", label: "Query insights", icon: Zap, keys: ["g", "q"], run: nav("queries") },
      { id: "n-sql", group: "Navigate", label: "SQL editor", icon: Terminal, keys: ["g", "s"], run: nav("sql") },
      { id: "n-ha", group: "Navigate", label: "HA topology", icon: Network, keys: ["g", "h"], run: nav("ha") },
      { id: "n-backups", group: "Navigate", label: "Backups & PITR", icon: Archive, keys: ["g", "b"], run: nav("backups") },
      { id: "n-mig", group: "Navigate", label: "Migrations", icon: GitPullRequestArrow, keys: ["g", "m"], run: nav("migrations") },
      { id: "s-new", group: "Setup & deploy", label: "New connection…", hint: "tl connect add", icon: PlugZap, keys: ["n"], run: nav("connect") },
      { id: "s-deploy", group: "Setup & deploy", label: "Deploy a new cluster…", hint: "tl deploy init", icon: Rocket, keys: ["g", "d"], run: nav("deploy") },
      { id: "s-compose", group: "Setup & deploy", label: "Deploy with Docker Compose (single host)", hint: "--target compose", icon: Container, run: nav("deploy") },
      { id: "s-cnpg", group: "Setup & deploy", label: "Deploy on Kubernetes with CloudNativePG", hint: "--target cnpg", icon: Ship, run: () => { go("deploy"); setTimeout(() => window.dispatchEvent(new CustomEvent("tl:target", { detail: "cnpg" })), 50); } },
      { id: "s-advisor", group: "Setup & deploy", label: "Review configuration (config advisor)", hint: "tl advise config", icon: SlidersHorizontal, keys: ["g", "v"], run: nav("advisor") },
      { id: "s-history", group: "Setup & deploy", label: "Deployment history", hint: "tl deploy history", icon: History, keys: ["g", "l"], run: nav("deployments") },
      { id: "s-hub", group: "Setup & deploy", label: "Setup overview", icon: Wrench, keys: ["g", "u"], run: nav("setup") },
      { id: "a-diag", group: "Actions", label: "Run diagnosis on current cluster", hint: "tl diagnose", icon: Stethoscope, run: () => { go("cluster"); toast({ title: "Diagnosis complete", body: "3 findings · 1 critical (lock contention on orders)", tone: "warn" }); } },
      { id: "a-switch", group: "Actions", label: "Start guarded switchover…", hint: "tl ha switchover", icon: RefreshCw, run: () => { go("ha"); setTimeout(() => window.dispatchEvent(new Event("tl:switchover")), 50); } },
      { id: "a-pitr", group: "Actions", label: "Plan point-in-time restore…", hint: "tl backup restore --to", icon: Archive, run: nav("backups") },
      { id: "a-advise", group: "Actions", label: "Advise indexes", hint: "tl advise indexes", icon: Zap, run: nav("queries") },
      { id: "a-theme", group: "Actions", label: "Toggle light / dark theme", icon: Moon, keys: ["t"], run: toggleTheme },
      { id: "a-help", group: "Actions", label: "Keyboard shortcuts", icon: Keyboard, keys: ["?"], run: () => setHelpOpen(true) },
      { id: "a-about", group: "Actions", label: "About Tuskline", icon: Info, run: () => app.setAboutOpen(true) },
      ...ENVS.map((e) => ({ id: `e-${e.id}`, group: "Environments", label: `Switch to ${e.label}`, icon: e.id === "prod" ? Shield : Server, color: e.color, run: () => setEnv(e.id) })),
      ...CLUSTERS.map((c) => ({ id: `c-${c.id}`, group: "Clusters", label: c.name, hint: `${c.env} · PG ${c.version} · ${c.region}`, icon: Server, color: healthColor(c.health), run: () => { setClusterId(c.id); go("cluster"); } })),
      ...QUERIES.slice(0, 6).map((qq) => ({ id: `q-${qq.id}`, group: "Queries", label: qq.sql.slice(0, 70), hint: `#${qq.id}`, icon: FileCode2, run: () => { go("queries"); setTimeout(() => window.dispatchEvent(new CustomEvent("tl:query", { detail: qq.id })), 50); } })),
    ];
  }, [go, setClusterId, setEnv, toggleTheme, toast, setHelpOpen]);

  const filtered = useMemo(() => {
    if (!q.trim()) return cmds.filter((c) => c.group !== "Queries");
    const terms = q.toLowerCase().split(/\s+/);
    return cmds.filter((c) => terms.every((t) => `${c.label} ${c.hint ?? ""} ${c.group}`.toLowerCase().includes(t)));
  }, [q, cmds]);
  useEffect(() => setI(0), [q]);
  useEffect(() => { listRef.current?.querySelector(`[data-i="${i}"]`)?.scrollIntoView({ block: "nearest" }); }, [i]);

  if (!paletteOpen) return null;
  const run = (c: Cmd) => { setPaletteOpen(false); c.run(); };
  let lastGroup = "";
  return (
    <div className="fixed inset-0 z-50 flex justify-center pt-[12vh] animate-fadeIn" style={{ background: "rgba(5,7,10,.5)", backdropFilter: "blur(2px)" }} onMouseDown={() => setPaletteOpen(false)}>
      <div className="card w-[620px] h-fit max-h-[480px] flex flex-col overflow-hidden animate-pop" style={{ boxShadow: "var(--shadow)" }} onMouseDown={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-2.5 px-4 h-12 border-b border-line">
          <TuskMark size={20} />
          <input ref={inputRef} value={q} onChange={(e) => setQ(e.target.value)} placeholder="Type a command, cluster or query…"
            className="flex-1 bg-transparent outline-none text-[14px] placeholder:text-fg3"
            onKeyDown={(e) => {
              if (e.key === "ArrowDown") { e.preventDefault(); setI((x) => Math.min(filtered.length - 1, x + 1)); }
              if (e.key === "ArrowUp") { e.preventDefault(); setI((x) => Math.max(0, x - 1)); }
              if (e.key === "Enter" && filtered[i]) run(filtered[i]);
              if (e.key === "Escape") setPaletteOpen(false);
            }} />
          <Kbd>esc</Kbd>
        </div>
        <div ref={listRef} className="overflow-auto p-1.5">
          {filtered.length === 0 && <div className="py-10 text-center text-fg3">No results for “{q}”. Try “switchover” or “prod”.</div>}
          {filtered.map((c, idx) => {
            const head = c.group !== lastGroup; lastGroup = c.group;
            const Icon = c.icon;
            return (
              <div key={c.id}>
                {head && <div className="label px-2.5 pt-2.5 pb-1">{c.group}</div>}
                <button data-i={idx} onMouseMove={() => setI(idx)} onClick={() => run(c)}
                  className={`w-full flex items-center gap-2.5 h-9 px-2.5 rounded-md text-left ${idx === i ? "bg-hover" : ""}`}>
                  {c.group === "Clusters" ? <Dot color={c.color!} size={6} /> : <Icon size={15} style={{ color: c.color ?? "var(--fg3)" }} />}
                  <span className={`flex-1 truncate ${c.group === "Clusters" || c.group === "Queries" ? "mono text-[12.5px]" : ""}`}>{c.label}</span>
                  {c.hint && <span className="text-fg3 text-[11.5px] mono truncate max-w-[200px]">{c.hint}</span>}
                  {c.keys && <span className="flex gap-0.5">{c.keys.map((k) => <Kbd key={k}>{k}</Kbd>)}</span>}
                  {idx === i && <ArrowRight size={13} className="text-fg3" />}
                </button>
              </div>
            );
          })}
        </div>
        <div className="flex items-center gap-4 px-4 h-9 border-t border-line text-[11px] text-fg3 bg-subtle">
          <span className="flex items-center gap-1"><Kbd>↑</Kbd><Kbd>↓</Kbd> navigate</span>
          <span className="flex items-center gap-1"><Kbd>↵</Kbd> run</span>
          <span className="ml-auto flex items-center gap-1.5"><TuskGlyph size={11} color="var(--fg3)" /><span className="mono">tl · {filtered.length} results</span></span>
        </div>
      </div>
    </div>
  );
}

export function ShortcutHelp() {
  const { helpOpen, setHelpOpen } = useApp();
  const rows: [string[], string][] = [
    [["⌘/Ctrl", "K"], "Command palette"], [["g", "f"], "Go to fleet"], [["g", "c"], "Go to cluster health"], [["g", "q"], "Go to query insights"],
    [["g", "s"], "Go to SQL editor"], [["g", "h"], "Go to HA topology"], [["g", "b"], "Go to backups"], [["g", "m"], "Go to migrations"],
    [["n"], "New connection"], [["g", "d"], "Deploy a new cluster"], [["g", "v"], "Config advisor"], [["g", "u"], "Setup overview"], [["g", "l"], "Deployment history"],
    [["e"], "Switch environment"], [["t"], "Toggle theme"], [["/"], "Focus filter"], [["[", "]"], "Step time range back / forward"],
    [["⌘/Ctrl", "↵"], "Run query (SQL editor)"], [["⌘/Ctrl", "⇧", "↵"], "EXPLAIN ANALYZE (SQL editor)"], [["?"], "This help"],
  ];
  return (
    <Modal open={helpOpen} onClose={() => setHelpOpen(false)} title={<span className="flex items-center gap-2"><Keyboard size={15} /> Keyboard shortcuts</span>} width={560}>
      <div className="grid grid-cols-2 gap-x-8 gap-y-2">
        {rows.map(([k, l]) => (
          <div key={l} className="flex items-center justify-between gap-3 py-1 border-b border-line/60">
            <span className="text-fg2">{l}</span><span className="flex gap-1">{k.map((x) => <Kbd key={x}>{x}</Kbd>)}</span>
          </div>
        ))}
      </div>
    </Modal>
  );
}

export function Toasts() {
  const { toasts } = useApp();
  const icon = { ok: CheckCircle2, warn: AlertTriangle, crit: XCircle, info: Info };
  return (
    <div className="fixed bottom-4 right-4 z-[60] flex flex-col gap-2 w-[360px]">
      {toasts.map((t) => {
        const tone = t.tone ?? "info"; const Icon = icon[tone];
        const color = tone === "info" ? "var(--accent)" : `var(--${tone})`;
        return (
          <div key={t.id} className="card p-3 flex gap-2.5 animate-slideIn" style={{ boxShadow: "var(--shadow)", borderLeft: `3px solid ${color}` }}>
            <Icon size={16} style={{ color }} className="mt-0.5 shrink-0" />
            <div className="min-w-0"><div className="font-medium">{t.title}</div>{t.body && <div className="text-fg2 text-[12px] mt-0.5">{t.body}</div>}</div>
          </div>
        );
      })}
    </div>
  );
}

export function useGlobalKeys() {
  const app = useApp();
  const ref = useRef(app); ref.current = app;
  useEffect(() => {
    let pendingG = 0;
    const ranges = ["15m", "1h", "6h", "24h", "7d", "30d"];
    const on = (e: KeyboardEvent) => {
      const a = ref.current;
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") { e.preventDefault(); a.setPaletteOpen(!a.paletteOpen); return; }
      const el = e.target as HTMLElement;
      if (el && (el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.tagName === "SELECT" || el.isContentEditable)) return;
      if (a.paletteOpen || e.metaKey || e.ctrlKey || e.altKey) return;
      const k = e.key;
      if (Date.now() - pendingG < 900) {
        const map: Record<string, Screen> = { f: "fleet", a: "fleet", c: "cluster", q: "queries", s: "sql", h: "ha", b: "backups", m: "migrations", d: "deploy", v: "advisor", u: "setup", l: "deployments" };
        pendingG = 0;
        if (map[k]) { e.preventDefault(); a.go(map[k]); }
        return;
      }
      if (k === "g") { pendingG = Date.now(); return; }
      if (k === "n") { e.preventDefault(); a.go("connect"); return; }
      if (k === "?") { a.setHelpOpen(!a.helpOpen); return; }
      if (k === "e") { e.preventDefault(); a.setEnvMenuOpen(!a.envMenuOpen); return; }
      if (k === "t") { a.toggleTheme(); return; }
      if (k === "/") { const f = document.querySelector<HTMLInputElement>("[data-filter]"); if (f) { e.preventDefault(); f.focus(); } return; }
      if (k === "[" || k === "]") {
        const idx = ranges.indexOf(a.range); const n = Math.max(0, Math.min(ranges.length - 1, idx + (k === "]" ? -1 : 1)));
        a.setRange(ranges[n]); a.toast({ title: `Time range: last ${ranges[n]}`, tone: "info" });
      }
    };
    window.addEventListener("keydown", on);
    return () => window.removeEventListener("keydown", on);
  }, []);
}


export function AboutDialog() {
  const { aboutOpen, setAboutOpen, theme } = useApp();
  return (
    <Modal open={aboutOpen} onClose={() => setAboutOpen(false)} title="About" width={440}>
      <div className="flex flex-col items-center text-center pt-2">
        <TuskMark size={72} variant={theme} />
        <div className="mt-4"><Wordmark height={26} /></div>
        <div className="text-fg2 mt-2 text-[12.5px]">A keyboard-first PostgreSQL workbench: monitor, diagnose, set up and deploy.</div>
        <div className="mono text-[11.5px] text-fg3 mt-3">v0.2.0-prototype · build 2026-10-28 · React (Svelte 5 port planned)</div>
        <div className="grid grid-cols-2 gap-2 w-full mt-5 text-left text-[12px]">
          {[["Licence", "AGPL-3.0 server · Apache-2.0 core (planned)"], ["Data", "All demo data is fake"], ["Brand", "Tusk Arc · #3E63DD"], ["Fonts", "Inter, JetBrains Mono"]].map(([k, v]) => (
            <div key={k} className="rounded-md bg-subtle border border-line px-3 py-2"><div className="label">{k}</div><div className="mt-0.5">{v}</div></div>
          ))}
        </div>
        <div className="flex items-center gap-1.5 text-[11.5px] text-fg3 mt-4"><ExternalLink size={11} /> tuskline is a working name; trademark checks pending.</div>
      </div>
    </Modal>
  );
}

/** Brief launch splash. ?splash=hold keeps it on screen (for screenshots); ?nosplash skips it. */
export function Splash() {
  const params = new URLSearchParams(window.location.search);
  const hold = params.get("splash") === "hold";
  const light = useApp().theme === "light";
  const [phase, setPhase] = useState<"show" | "out" | "gone">(params.has("nosplash") ? "gone" : "show");
  useEffect(() => {
    if (hold || phase !== "show") return;
    const t1 = setTimeout(() => setPhase("out"), 650);
    const t2 = setTimeout(() => setPhase("gone"), 1000);
    return () => { clearTimeout(t1); clearTimeout(t2); };
  }, [hold, phase]);
  if (phase === "gone") return null;
  return (
    <div className={`fixed inset-0 z-[100] grid place-items-center bg-bg ${phase === "out" ? "splash-out" : ""}`} aria-hidden>
      <div className="flex flex-col items-center">
        <svg width="88" height="88" viewBox="0 0 64 64">
          <rect width="64" height="64" rx="14" fill={light ? "#3E63DD" : "#0E1116"} />
          {!light && <rect x="0.75" y="0.75" width="62.5" height="62.5" rx="13.25" fill="none" stroke="#2A303A" strokeWidth="1.5" />}
          <path d="M13 45.5 C13 27.5 26.5 13.5 51 12 C34 19 23 30.5 23 45.5 A5 5 0 0 1 13 45.5 Z" fill={light ? "#FFFFFF" : "#FAFAF9"} className="animate-fadeIn" />
          <path d="M30 45.5 H40.5" stroke={light ? "#FFFFFF" : "#FAFAF9"} strokeWidth="5" strokeLinecap="round" fill="none" className="draw-arc" />
          <circle cx="48.5" cy="45.5" r="4.25" fill={light ? "#FFFFFF" : "#7B96FF"} className="pulse-dot" />
        </svg>
        <div className="mt-5"><Wordmark height={22} /></div>
        <div className="mt-6 w-40 h-1 rounded-full bg-subtle overflow-hidden"><div className="h-full rounded-full" style={{ background: "var(--brand)", animation: "grow 0.65s ease-out forwards" }} /></div>
        <div className="mono text-[11px] text-fg3 mt-3">loading demo workspace…</div>
      </div>
    </div>
  );
}
