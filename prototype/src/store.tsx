import { createContext, useCallback, useContext, useEffect, useMemo, useState, ReactNode } from "react";
import { CLUSTERS, Cluster, Env } from "./data";
import { faviconSvg } from "./components/Brand";
import { DEPLOYMENTS, Deployment } from "./setup-data";

export type Screen = "fleet" | "cluster" | "queries" | "sql" | "ha" | "backups" | "migrations" | "setup" | "connect" | "advisor" | "deploy" | "deployments";
export const SCREENS: Screen[] = ["fleet", "cluster", "queries", "sql", "ha", "backups", "migrations", "setup", "connect", "advisor", "deploy", "deployments"];
/** Screens that are not scoped to the selected cluster (no env rail / prod banner). */
export const GLOBAL_SCREENS: Screen[] = ["fleet", "setup", "connect", "deploy", "deployments"];

interface Toast { id: number; title: string; body?: string; tone?: "ok" | "warn" | "crit" | "info" }
interface Ctx {
  screen: Screen; go: (s: Screen) => void;
  theme: "dark" | "light"; toggleTheme: () => void;
  env: Env; setEnv: (e: Env) => void;
  cluster: Cluster; setClusterId: (id: string) => void;
  range: string; setRange: (r: string) => void;
  paletteOpen: boolean; setPaletteOpen: (b: boolean) => void;
  envMenuOpen: boolean; setEnvMenuOpen: (b: boolean) => void;
  helpOpen: boolean; setHelpOpen: (b: boolean) => void;
  toasts: Toast[]; toast: (t: Omit<Toast, "id">) => void;
  elevatedUntil: number | null; setElevatedUntil: (n: number | null) => void;
  aboutOpen: boolean; setAboutOpen: (b: boolean) => void;
  fleetVersion: number; freshInstall: boolean; setFreshInstall: (b: boolean) => void;
  newClusterId: string | null; addCluster: (c: Cluster) => void;
  deployments: Deployment[]; upsertDeployment: (d: Deployment) => void;
}
const C = createContext<Ctx>(null as unknown as Ctx);
export const useApp = () => useContext(C);

function parseHash(): { screen: Screen; cluster?: string } {
  const [, s, c] = window.location.hash.split("/");
  const screen = (SCREENS as string[]).includes(s) ? (s as Screen) : "fleet";
  return { screen, cluster: c };
}

export function AppProvider({ children }: { children: ReactNode }) {
  const initial = parseHash();
  const params = new URLSearchParams(window.location.search);
  const [screen, setScreen] = useState<Screen>(initial.screen);
  const [theme, setTheme] = useState<"dark" | "light">(params.get("theme") === "light" ? "light" : "dark");
  const [clusterId, setClusterIdRaw] = useState(initial.cluster && CLUSTERS.some((c) => c.id === initial.cluster) ? initial.cluster : "prod-eu-west");
  const cluster = CLUSTERS.find((c) => c.id === clusterId)!;
  const [env, setEnvRaw] = useState<Env>(cluster.env);
  const [range, setRange] = useState("6h");
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [envMenuOpen, setEnvMenuOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [elevatedUntil, setElevatedUntil] = useState<number | null>(null);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [fleetVersion, setFleetVersion] = useState(0);
  const [freshInstall, setFreshInstall] = useState(params.get("empty") === "1");
  const [newClusterId, setNewClusterId] = useState<string | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>(DEPLOYMENTS);
  const addCluster = useCallback((c: Cluster) => {
    const i = CLUSTERS.findIndex((x) => x.id === c.id);
    if (i >= 0) CLUSTERS[i] = c; else CLUSTERS.push(c);
    setFleetVersion((v) => v + 1); setNewClusterId(c.id); setFreshInstall(false);
  }, []);
  const upsertDeployment = useCallback((d: Deployment) => setDeployments((xs) => [d, ...xs.filter((x) => x.id !== d.id)]), []);

  useEffect(() => {
    document.documentElement.className = theme;
    const link = document.querySelector<HTMLLinkElement>("link[rel=icon]");
    if (link) link.href = "data:image/svg+xml," + encodeURIComponent(faviconSvg(theme));
  }, [theme]);
  useEffect(() => {
    const h = `#/${screen}/${clusterId}`;
    if (window.location.hash !== h) history.replaceState(null, "", h);
  }, [screen, clusterId]);
  useEffect(() => {
    const on = () => { const p = parseHash(); setScreen(p.screen); const cl = CLUSTERS.find((c) => c.id === p.cluster); if (cl) { setClusterIdRaw(cl.id); setEnvRaw(cl.env); } };
    window.addEventListener("hashchange", on);
    return () => window.removeEventListener("hashchange", on);
  }, []);

  const toast = useCallback((t: Omit<Toast, "id">) => {
    const id = Date.now() + Math.random();
    setToasts((x) => [...x, { ...t, id }]);
    setTimeout(() => setToasts((x) => x.filter((y) => y.id !== id)), 4200);
  }, []);
  const setClusterId = useCallback((id: string) => {
    const c = CLUSTERS.find((x) => x.id === id)!;
    setClusterIdRaw(id); setEnvRaw(c.env); setElevatedUntil(null);
  }, []);
  const setEnv = useCallback((e: Env) => {
    setEnvRaw(e);
    const first = CLUSTERS.find((c) => c.env === e)!;
    setClusterIdRaw((cur) => (CLUSTERS.find((c) => c.id === cur)!.env === e ? cur : first.id));
    setElevatedUntil(null);
  }, []);

  const value = useMemo<Ctx>(() => ({
    screen, go: setScreen, theme, toggleTheme: () => setTheme((t) => (t === "dark" ? "light" : "dark")),
    env, setEnv, cluster, setClusterId, range, setRange, paletteOpen, setPaletteOpen, envMenuOpen, setEnvMenuOpen,
    helpOpen, setHelpOpen, toasts, toast, elevatedUntil, setElevatedUntil,
    aboutOpen, setAboutOpen, fleetVersion, freshInstall, setFreshInstall, newClusterId, addCluster, deployments, upsertDeployment,
  }), [screen, theme, env, cluster, range, paletteOpen, envMenuOpen, helpOpen, toasts, elevatedUntil, setEnv, setClusterId, toast, aboutOpen, fleetVersion, freshInstall, newClusterId, addCluster, deployments, upsertDeployment]);
  return <C.Provider value={value}>{children}</C.Provider>;
}
