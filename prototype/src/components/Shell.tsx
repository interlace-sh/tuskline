import { ReactNode, useEffect, useRef, useState } from "react";
import { History, Info, PlugZap, Rocket, SlidersHorizontal, Wrench, Activity, Archive, Bell, ChevronDown, ChevronRight, Command, GitPullRequestArrow, LayoutGrid, Moon, Network, Search, Settings, Shield, Sun, Terminal, Zap, Check, Lock } from "lucide-react";
import { useApp, Screen, GLOBAL_SCREENS } from "../store";
import { TuskMark, Wordmark } from "./Brand";
import { CLUSTERS, ENVS, envColor } from "../data";
import { Dot, EnvBadge, Kbd, healthColor, tint } from "./ui";

export const NAV: { id: Screen; label: string; icon: typeof Activity; keys: string }[] = [
  { id: "cluster", label: "Health", icon: Activity, keys: "g c" },
  { id: "queries", label: "Query insights", icon: Zap, keys: "g q" },
  { id: "sql", label: "SQL editor", icon: Terminal, keys: "g s" },
  { id: "ha", label: "HA topology", icon: Network, keys: "g h" },
  { id: "backups", label: "Backups", icon: Archive, keys: "g b" },
  { id: "migrations", label: "Migrations", icon: GitPullRequestArrow, keys: "g m" },
];

export const SETUP_NAV: { id: Screen; label: string; icon: typeof Activity; keys: string }[] = [
  { id: "setup", label: "Overview", icon: Wrench, keys: "g u" },
  { id: "connect", label: "Add connection", icon: PlugZap, keys: "n" },
  { id: "deploy", label: "Deploy cluster", icon: Rocket, keys: "g d" },
  { id: "deployments", label: "Deployments", icon: History, keys: "g l" },
];

/** Kept for compatibility with earlier imports: now the Tusk Arc tile. */
export function Logo({ size = 22 }: { size?: number }) { return <TuskMark size={size} />; }

function Sidebar() {
  const { screen, go, cluster, env, theme, toggleTheme, setAboutOpen, deployments, freshInstall } = useApp();
  const running = deployments.filter((d) => d.status === "running").length;
  const item = (id: Screen, label: string, Icon: typeof Activity, keys: string, badge?: ReactNode, noActive?: boolean) => {
    const active = screen === id && !noActive;
    return (
      <button key={label} onClick={() => go(id)} title={`${label} (${keys})`}
        className={`group w-full flex items-center gap-2.5 h-8 px-2.5 rounded-md text-[13px] transition-colors focus-ring ${active ? "bg-hover text-fg font-medium" : "text-fg2 hover:text-fg hover:bg-hover/60"}`}>
        <Icon size={15} className={active ? "text-accent" : "text-fg3 group-hover:text-fg2"} />
        <span className="flex-1 text-left truncate">{label}</span>
        {badge}
        <span className="hidden group-hover:flex gap-0.5 opacity-70">{keys.split(" ").map((k) => <Kbd key={k}>{k}</Kbd>)}</span>
      </button>
    );
  };
  const crit = CLUSTERS.filter((c) => c.health !== "ok").length;
  return (
    <aside className="w-[232px] shrink-0 border-r border-line bg-elev/40 flex flex-col overflow-y-auto">
      <button onClick={() => setAboutOpen(true)} title="About Tuskline" className="h-12 flex items-center gap-2.5 px-4 border-b border-line text-left hover:bg-hover/40 transition-colors">
        <TuskMark size={24} />
        <div className="leading-none min-w-0">
          <Wordmark height={14} />
          <div className="text-[10.5px] text-fg3 mt-1">Acme Retail · demo</div>
        </div>
      </button>
      <nav className="p-2 space-y-0.5">
        {item("fleet", "Fleet", LayoutGrid, "g f")}
        {item("fleet", "Alerts", Bell, "g a", freshInstall ? undefined : <span className="text-[10.5px] px-1.5 rounded-full tnum" style={{ background: tint("var(--crit)", 18), color: "var(--crit)" }}>{crit}</span>, true)}
      </nav>
      <div className="px-4 pt-2 pb-1.5"><span className="label">Setup &amp; deploy</span></div>
      <div className="px-2 space-y-0.5">{SETUP_NAV.map((n) => item(n.id, n.label, n.icon, n.keys, n.id === "deployments" && running ? <span className="text-[10.5px] px-1.5 rounded-full tnum" style={{ background: tint("var(--accent)", 18), color: "var(--accent)" }}>{running}</span> : undefined))}</div>
      {!freshInstall && <><div className="px-4 pt-3 pb-1.5 flex items-center justify-between">
        <span className="label">Cluster</span>
        <EnvBadge env={env} />
      </div>
      <div className="px-2">
        <div className="flex items-center gap-2 px-2.5 py-2 mb-1 rounded-md bg-subtle border border-line">
          <Dot color={healthColor(cluster.health)} pulse={cluster.health !== "ok"} />
          <div className="min-w-0 flex-1">
            <div className="mono text-[12px] truncate">{cluster.name}</div>
            <div className="text-[10.5px] text-fg3 truncate">PG {cluster.version} · {cluster.ha}</div>
          </div>
        </div>
        <div className="space-y-0.5">{NAV.map((n) => item(n.id, n.label, n.icon, n.keys))}{item("advisor", "Config advisor", SlidersHorizontal, "g v")}</div>
      </div></>}
      <div className="mt-auto p-3 space-y-2">
        <div className="rounded-lg border border-line bg-subtle p-3">
          <div className="flex items-center gap-1.5 text-[11.5px] text-fg2 mb-1.5"><Terminal size={12} /> Same thing, from your terminal</div>
          <code className="mono text-[11px] text-fg block truncate">$ tl connect {env}/{cluster.databases[0]}</code>
        </div>
        <div className="flex items-center gap-1">
          <button className="btn btn-ghost flex-1 justify-start" onClick={() => go("fleet")}><Settings size={14} /> Settings</button>
          <button className="btn btn-ghost !px-2" onClick={() => setAboutOpen(true)} title="About Tuskline"><Info size={14} /></button>
          <button className="btn btn-ghost !px-2" onClick={toggleTheme} title="Toggle theme (t)">{theme === "dark" ? <Sun size={14} /> : <Moon size={14} />}</button>
        </div>
      </div>
    </aside>
  );
}

function EnvSwitcher() {
  const { env, setEnv, envMenuOpen, setEnvMenuOpen, cluster, setClusterId } = useApp();
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!envMenuOpen) return;
    const on = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) setEnvMenuOpen(false); };
    const key = (e: KeyboardEvent) => { if (e.key === "Escape") setEnvMenuOpen(false); };
    window.addEventListener("mousedown", on); window.addEventListener("keydown", key);
    return () => { window.removeEventListener("mousedown", on); window.removeEventListener("keydown", key); };
  }, [envMenuOpen, setEnvMenuOpen]);
  const color = envColor(env);
  return (
    <div className="relative" ref={ref}>
      <button onClick={() => setEnvMenuOpen(!envMenuOpen)} className="flex items-center gap-2 h-7 pl-2 pr-1.5 rounded-md text-[12.5px] font-medium focus-ring transition-colors"
        style={{ background: tint(color, 14), color, boxShadow: `inset 0 0 0 1px ${tint(color, 35)}` }} title="Switch environment (e)">
        {env === "prod" ? <Shield size={13} /> : <Dot color={color} size={6} />}
        <span className="uppercase tracking-wide text-[11px]">{env}</span>
        <span className="text-fg2 mono text-[11.5px] font-normal">{cluster.name}</span>
        <ChevronDown size={13} />
      </button>
      {envMenuOpen && (
        <div className="absolute right-0 top-9 z-40 w-[330px] card p-1.5 animate-pop" style={{ boxShadow: "var(--shadow)" }}>
          <div className="px-2 py-1.5 flex items-center justify-between"><span className="label">Switch environment</span><Kbd>e</Kbd></div>
          {ENVS.map((e) => (
            <div key={e.id} className="mb-1">
              <button onClick={() => { setEnv(e.id); }} className={`w-full flex items-center gap-2 px-2 h-8 rounded-md hover:bg-hover ${env === e.id ? "bg-hover" : ""}`}>
                <span className="w-1 h-4 rounded-full" style={{ background: e.color }} />
                <span className="font-medium flex-1 text-left">{e.label}</span>
                {e.id === "prod" && <span className="text-[10.5px] text-fg3 flex items-center gap-1"><Lock size={10} /> read-only by default</span>}
                {env === e.id && <Check size={14} className="text-accent" />}
              </button>
              {env === e.id && (
                <div className="pl-5 pr-1 pb-1 space-y-0.5">
                  {CLUSTERS.filter((c) => c.env === e.id).map((c) => (
                    <button key={c.id} onClick={() => { setClusterId(c.id); setEnvMenuOpen(false); }} className={`w-full flex items-center gap-2 px-2 h-7 rounded-md text-[12px] hover:bg-hover ${c.id === cluster.id ? "text-fg" : "text-fg2"}`}>
                      <Dot color={healthColor(c.health)} size={6} /><span className="mono flex-1 text-left">{c.name}</span><span className="text-fg3 text-[11px]">{c.region}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function Topbar() {
  const { env, cluster, screen, setPaletteOpen, range, setRange, theme, toggleTheme, go } = useApp();
  const envLabel = ENVS.find((e) => e.id === env)!.label;
  const crumbs: ReactNode[] = [<span key="p">Acme Retail</span>];
  const SETUP_LABEL: Partial<Record<Screen, string>> = { setup: "Overview", connect: "Add connection", deploy: "Deploy cluster", deployments: "Deployments" };
  if (SETUP_LABEL[screen]) crumbs.push(<button key="s" className="hover:text-fg" onClick={() => go("setup")}>Setup</button>, <span key="x" className="text-fg">{SETUP_LABEL[screen]}</span>);
  else if (screen !== "fleet") crumbs.push(<span key="e" style={{ color: envColor(env) }}>{envLabel}</span>, <button key="c" className="mono hover:text-fg" onClick={() => go("cluster")}>{cluster.name}</button>, <span key="d" className="mono">{cluster.databases[0]}</span>);
  else crumbs.push(<span key="f" className="text-fg">Fleet</span>);
  return (
    <header className="h-12 shrink-0 flex items-center gap-3 px-4 border-b border-line bg-bg/80 backdrop-blur sticky top-0 z-30">
      <div className="flex items-center gap-1.5 text-fg3 text-[12.5px] min-w-0">
        {crumbs.map((c, i) => <span key={i} className="flex items-center gap-1.5 truncate">{i > 0 && <ChevronRight size={12} className="opacity-60" />}{c}</span>)}
      </div>
      <button onClick={() => setPaletteOpen(true)} className="ml-auto flex items-center gap-2 h-7 w-[300px] px-2.5 rounded-md bg-subtle border border-line text-fg3 hover:border-line2 transition-colors focus-ring">
        <Search size={13} /><span className="flex-1 text-left text-[12.5px]">Search clusters, queries, actions…</span>
        <span className="flex gap-0.5"><Kbd><Command size={10} /></Kbd><Kbd>K</Kbd></span>
      </button>
      <select value={range} onChange={(e) => setRange(e.target.value)} className="input !h-7 !pr-6 text-[12px] cursor-pointer" title="Global time range ([ and ] to step)">
        {["15m", "1h", "6h", "24h", "7d", "30d"].map((r) => <option key={r} value={r}>Last {r}</option>)}
      </select>
      <EnvSwitcher />
      <button className="btn btn-ghost !px-2" onClick={toggleTheme} title="Toggle theme (t)">{theme === "dark" ? <Sun size={14} /> : <Moon size={14} />}</button>
      <div className="w-7 h-7 rounded-full grid place-items-center text-[11px] font-semibold text-white" style={{ background: "linear-gradient(135deg,#3E63DD,#8E4EC6)" }} title="Mark Alford · Owner">MA</div>
    </header>
  );
}

export function AppShell({ children }: { children: ReactNode }) {
  const { env, screen, elevatedUntil } = useApp();
  const color = envColor(env);
  const [key, setKey] = useState(0);
  useEffect(() => setKey((k) => k + 1), [screen]);
  return (
    <div className="h-full flex flex-col">
      <div className="h-[3px] shrink-0 transition-colors duration-300" style={{ background: GLOBAL_SCREENS.includes(screen) ? "linear-gradient(90deg,var(--dev),var(--staging),var(--prod))" : color }} />
      {env === "prod" && !GLOBAL_SCREENS.includes(screen) && (
        <div className="shrink-0 h-6 flex items-center justify-center gap-2 text-[11px] font-medium tracking-wide" style={{ background: tint("var(--prod)", 12), color: "var(--prod)", borderBottom: `1px solid ${tint("var(--prod)", 25)}` }}>
          <Shield size={11} /> {elevatedUntil ? "PRODUCTION · write access elevated · every statement audited" : "PRODUCTION · read-only session · changes require elevation and typed confirmation"}
        </div>
      )}
      <div className="flex-1 flex min-h-0">
        <Sidebar />
        <div className="flex-1 flex flex-col min-w-0">
          <Topbar />
          <main key={key} className="flex-1 overflow-auto animate-fadeIn"><div className="p-6 max-w-[1600px] mx-auto">{children}</div></main>
        </div>
      </div>
    </div>
  );
}

