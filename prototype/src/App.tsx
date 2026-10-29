import { AppProvider, useApp } from "./store";
import { AppShell } from "./components/Shell";
import { AboutDialog, CommandPalette, ShortcutHelp, Splash, Toasts, useGlobalKeys } from "./components/Overlays";
import Fleet from "./screens/Fleet";
import Cluster from "./screens/Cluster";
import Queries from "./screens/Queries";
import Sql from "./screens/Sql";
import Ha from "./screens/Ha";
import Backups from "./screens/Backups";
import Migrations from "./screens/Migrations";
import Setup, { Deployments } from "./screens/Setup";
import Connect from "./screens/Connect";
import Advisor from "./screens/Advisor";
import Deploy from "./screens/Deploy";

function Inner() {
  const { screen, cluster } = useApp();
  useGlobalKeys();
  const S = { fleet: Fleet, cluster: Cluster, queries: Queries, sql: Sql, ha: Ha, backups: Backups, migrations: Migrations, setup: Setup, connect: Connect, advisor: Advisor, deploy: Deploy, deployments: Deployments }[screen];
  const k = screen === "fleet" || screen === "setup" || screen === "connect" || screen === "deploy" || screen === "deployments" ? screen : `${screen}-${cluster.id}`;
  return (
    <>
      <AppShell><S key={k} /></AppShell>
      <CommandPalette /><ShortcutHelp /><AboutDialog /><Toasts /><Splash />
    </>
  );
}
export default function App() { return <AppProvider><Inner /></AppProvider>; }
