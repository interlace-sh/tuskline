import { useEffect, useMemo, useState } from "react";
import { ArrowLeft, ArrowRight, Cable, CheckCircle2, Cloud, KeyRound, Lock, PlugZap, Server, Shield, ShieldCheck, Sparkles, Wand2 } from "lucide-react";
import { useApp } from "../store";
import { Env, ENVS, series, Cluster } from "../data";
import { Badge, Card, PageHeader, Segmented, tint } from "../components/ui";
import { CheckRow, CheckState, CodeTabs, DemoTag, Field, Stepper, Toggle } from "../components/setup-ui";
import { EXTENSIONS, PROVIDER_LABEL, Provider, cloudCommand, confPatch, preloadSql, restartSteps, roleSql } from "../setup-data";

const STEPS = ["Connection", "Test", "Capabilities", "Monitoring role", "Finish"];
type Auth = "password" | "aws-iam" | "gcp-iam" | "azure-entra";
interface Form { name: string; host: string; port: string; db: string; user: string; auth: Auth; tls: "verify-full" | "require" | "disable"; ssh: boolean; bastion: string; sshUser: string; env: Env }
const PRESETS: Record<"rds" | "self", Form> = {
  rds: { name: "orders-eu-demo", host: "demo-orders-db.c1a2b3c4d5e6.eu-west-2.rds.amazonaws.com", port: "5432", db: "orders", user: "tuskline_monitor", auth: "aws-iam", tls: "verify-full", ssh: true, bastion: "bastion.demo.example", sshUser: "ec2-user", env: "staging" },
  self: { name: "billing-onprem-demo", host: "pg-demo-1.dc1.demo.example", port: "5432", db: "billing", user: "tuskline_monitor", auth: "password", tls: "verify-full", ssh: false, bastion: "", sshUser: "", env: "prod" },
};
const detect = (f: Form): Provider => f.host.includes("rds.amazonaws.com") ? "rds" : f.host.includes("postgres.database.azure.com") ? "azure" : f.auth === "gcp-iam" ? "cloudsql" : "self";
const slug = (s: string) => s.toLowerCase().replace(/[^a-z0-9-]+/g, "-").replace(/^-|-$/g, "") || "new-connection";

export default function Connect() {
  const { go, addCluster, setClusterId, toast } = useApp();
  const [step, setStep] = useState(0);
  const [f, setF] = useState<Form>(PRESETS.rds);
  const set = <K extends keyof Form>(k: K, v: Form[K]) => setF((x) => ({ ...x, [k]: v }));
  const provider = detect(f);
  const version = provider === "self" ? "17.2" : "16.4";
  const missing = EXTENSIONS.filter((e) => (provider === "self" ? e.self !== "preloaded" && e.self !== "ok" : e.rds === "missing"));
  const [picked, setPicked] = useState<string[]>([]);
  useEffect(() => { setPicked(missing.map((e) => e.name)); }, [provider]); // eslint-disable-line react-hooks/exhaustive-deps
  const [role, setRole] = useState("tuskline_monitor");
  const [iam, setIam] = useState(true);
  const [operator, setOperator] = useState(false);
  const [roleVerified, setRoleVerified] = useState<CheckState>("pending");
  const [testDone, setTestDone] = useState(false);
  useEffect(() => setIam(f.auth !== "password"), [f.auth]);

  const valid = f.name.trim() && f.host.trim() && f.db.trim() && (!f.ssh || f.bastion.trim());
  const next = () => setStep((s) => Math.min(STEPS.length - 1, s + 1));
  const back = () => setStep((s) => Math.max(0, s - 1));

  const finish = () => {
    const id = slug(f.name);
    const c: Cluster = {
      id, name: id, env: f.env, health: picked.length ? "warn" : "ok", score: picked.length ? 84 : 97, version,
      ha: provider === "rds" ? "RDS Multi-AZ" : provider === "self" ? "Patroni" : "Standalone",
      provider: provider === "rds" ? "AWS" : provider === "cloudsql" ? "GCP" : provider === "azure" ? "Azure" : "Local",
      region: provider === "rds" ? "eu-west-2" : provider === "self" ? "dc1 · on-prem" : "—", primary: 1, replicas: provider === "self" ? 2 : 1,
      tps: 640, conns: 48, maxConns: 400, lagBytes: 120_000, lagSec: 0.1, backupAgeH: 6, advisories: picked.length + 3, storageGb: 210, storagePct: 41,
      load: series(id.length * 7, 40, 1.6, 0.8), databases: [f.db], issue: picked.length ? `${picked.length} monitoring extensions pending restart` : undefined,
    };
    addCluster(c); setClusterId(id);
    toast({ title: `${id} added to Fleet`, body: `${ENVS.find((e) => e.id === f.env)!.label} · PG ${version} · demo connection`, tone: "ok" });
    go("fleet");
  };

  return (
    <div className="max-w-[1180px] mx-auto">
      <PageHeader icon={<PlugZap size={18} className="text-accent" />} title="Add connection" sub={<>Connect an existing PostgreSQL 14+ server, check what it can do, and set it up for monitoring. <DemoTag /></>}
        actions={<button className="btn btn-ghost" onClick={() => go("setup")}>Cancel</button>} />
      <Stepper steps={STEPS} at={step} onJump={setStep} />
      {step === 0 && <Details f={f} set={set} setF={setF} provider={provider} />}
      {step === 1 && <TestStep f={f} provider={provider} version={version} onDone={setTestDone} />}
      {step === 2 && <Capabilities provider={provider} version={version} picked={picked} setPicked={setPicked} />}
      {step === 3 && <RoleStep provider={provider} db={f.db} role={role} setRole={setRole} iam={iam} setIam={setIam} operator={operator} setOperator={setOperator} verified={roleVerified} setVerified={setRoleVerified} />}
      {step === 4 && <FinishStep f={f} provider={provider} version={version} picked={picked} roleVerified={roleVerified === "ok"} />}
      <div className="flex items-center gap-2 mt-5 pt-4 border-t border-line">
        {step > 0 && <button className="btn" onClick={back}><ArrowLeft size={13} /> Back</button>}
        <span className="text-fg3 text-[12px] ml-2">Step {step + 1} of {STEPS.length}</span>
        <span className="ml-auto" />
        {step < STEPS.length - 1
          ? <button className="btn btn-primary" disabled={!valid || (step === 1 && !testDone)} onClick={next}>{step === 0 ? "Test connection" : "Continue"} <ArrowRight size={13} /></button>
          : <button className="btn btn-primary" onClick={finish}><CheckCircle2 size={13} /> Add to Fleet</button>}
      </div>
    </div>
  );
}

function Details({ f, set, setF, provider }: { f: Form; set: <K extends keyof Form>(k: K, v: Form[K]) => void; setF: (f: Form) => void; provider: Provider }) {
  return (
    <div className="grid grid-cols-[1fr_340px] gap-4">
      <Card title="Connection details" actions={<span className="flex items-center gap-1.5 text-[11.5px] text-fg3"><Wand2 size={12} /> Fill demo:
        <button className="btn !h-6 text-[11.5px]" onClick={() => setF(PRESETS.rds)}>Amazon RDS</button>
        <button className="btn !h-6 text-[11.5px]" onClick={() => setF(PRESETS.self)}>Self-managed</button></span>}>
        <div className="grid grid-cols-6 gap-3">
          <Field label="Display name" className="col-span-2"><input className="input w-full mono" value={f.name} onChange={(e) => set("name", e.target.value)} /></Field>
          <Field label="Host" className="col-span-3"><input className="input w-full mono" value={f.host} onChange={(e) => set("host", e.target.value)} placeholder="db.example.internal" /></Field>
          <Field label="Port"><input className="input w-full mono" value={f.port} onChange={(e) => set("port", e.target.value)} /></Field>
          <Field label="Database" className="col-span-2"><input className="input w-full mono" value={f.db} onChange={(e) => set("db", e.target.value)} /></Field>
          <Field label="User" className="col-span-2"><input className="input w-full mono" value={f.user} onChange={(e) => set("user", e.target.value)} /></Field>
          <Field label="Authentication" className="col-span-2">
            <select className="input w-full" value={f.auth} onChange={(e) => set("auth", e.target.value as Auth)}>
              <option value="password">Password (OS keychain)</option><option value="aws-iam">AWS IAM token</option><option value="gcp-iam">Google Cloud IAM</option><option value="azure-entra">Microsoft Entra ID</option>
            </select>
          </Field>
          <Field label="TLS" className="col-span-3" hint={f.tls === "disable" ? <span className="text-crit">Unencrypted. Only for local containers.</span> : f.tls === "require" ? "Encrypted, but the server certificate is not verified." : "Encrypted and the certificate is checked against the hostname."}>
            <Segmented value={f.tls} onChange={(v) => set("tls", v)} options={[{ value: "verify-full", label: <span className="flex items-center gap-1"><ShieldCheck size={11} />verify-full</span> }, { value: "require", label: "require" }, { value: "disable", label: "disable" }]} />
          </Field>
          <Field label="Environment tag" className="col-span-3" hint={f.env === "prod" ? "Production connections open read-only; writes need elevation and typed confirmation." : "Sets the colour rail, guards and default permissions."}>
            <div className="flex gap-1.5">
              {(["dev", "staging", "prod"] as Env[]).map((e) => {
                const col = ENVS.find((x) => x.id === e)!.color; const on = f.env === e;
                return <button key={e} onClick={() => set("env", e)} className="h-7 px-3 rounded-md text-[12px] font-medium uppercase tracking-wide transition-all" style={on ? { background: tint(col, 18), color: col, boxShadow: `inset 0 0 0 1px ${tint(col, 50)}` } : { color: "var(--fg3)", boxShadow: "inset 0 0 0 1px var(--line)" }}>{e === "prod" && <Shield size={11} className="inline -mt-0.5 mr-1" />}{e}</button>;
              })}
            </div>
          </Field>
        </div>
        <div className="mt-4 pt-4 border-t border-line">
          <Toggle on={f.ssh} onChange={(b) => set("ssh", b)} label={<span className="flex items-center gap-1.5"><Cable size={13} className="text-fg3" /> Connect through an SSH tunnel (bastion)</span>} />
          {f.ssh && (
            <div className="grid grid-cols-6 gap-3 mt-3 animate-fadeIn">
              <Field label="Bastion host" className="col-span-3"><input className="input w-full mono" value={f.bastion} onChange={(e) => set("bastion", e.target.value)} placeholder="bastion.example.com" /></Field>
              <Field label="SSH user" className="col-span-1"><input className="input w-full mono" value={f.sshUser} onChange={(e) => set("sshUser", e.target.value)} /></Field>
              <Field label="Key" className="col-span-2" hint="Uses your SSH agent; keys never leave this machine."><div className="input w-full flex items-center gap-1.5 text-fg2"><KeyRound size={12} /> ssh-agent · ed25519</div></Field>
            </div>
          )}
        </div>
      </Card>
      <div className="space-y-4">
        <Card title="Detected from host">
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-lg grid place-items-center bg-subtle border border-line">{provider === "self" ? <Server size={16} className="text-fg2" /> : <Cloud size={16} className="text-accent" />}</div>
            <div className="min-w-0"><div className="font-medium truncate">{PROVIDER_LABEL[provider]}</div><div className="text-fg3 text-[12px]">{provider === "self" ? "Generic PostgreSQL, full control of postgresql.conf" : "Managed service · Tier 1 compatibility"}</div></div>
          </div>
          <div className="mt-3 text-[12px] text-fg2 leading-relaxed">{provider === "rds" ? "Settings change through a DB parameter group, not ALTER SYSTEM. We will generate the AWS CLI commands." : provider === "self" ? "We will generate SQL, postgresql.conf changes and a rolling-restart plan." : "We will generate the provider CLI commands."}</div>
        </Card>
        <Card title="Same thing, from your terminal">
          <code className="mono text-[11.5px] block whitespace-pre-wrap break-all text-fg2">$ tl connect add {f.name} \{"\n"}  --host {f.host} --db {f.db} \{"\n"}  --env {f.env} --sslmode {f.tls}{f.ssh ? ` \\\n  --ssh ${f.sshUser}@${f.bastion}` : ""}{f.auth !== "password" ? ` \\\n  --auth ${f.auth}` : ""}</code>
        </Card>
        <div className="text-[11.5px] text-fg3 flex items-start gap-1.5 px-1"><Lock size={12} className="mt-0.5 shrink-0" /> Credentials are stored in the OS keychain. Nothing is sent to Tuskline servers. (Prototype: nothing connects anywhere.)</div>
      </div>
    </div>
  );
}

function useSequence(n: number, results: CheckState[], gap = 520, start = true) {
  const [i, setI] = useState(start ? 0 : -1);
  useEffect(() => {
    if (i < 0 || i >= n) return;
    const t = setTimeout(() => setI((x) => x + 1), gap);
    return () => clearTimeout(t);
  }, [i, n, gap]);
  const states: CheckState[] = Array.from({ length: n }, (_, k) => (k < i ? results[k] : k === i ? "running" : "pending"));
  return { states, done: i >= n, restart: () => setI(0) };
}

function TestStep({ f, provider, version, onDone }: { f: Form; provider: Provider; version: string; onDone?: (b: boolean) => void }) {
  const checks = useMemo(() => {
    const c: { label: string; detail: string; r: CheckState }[] = [];
    if (f.ssh) c.push({ label: "SSH tunnel", detail: `${f.sshUser}@${f.bastion} · host key matches known_hosts · local port 54321`, r: "ok" });
    c.push({ label: "Resolve and reach host", detail: `${f.host.split(".")[0]}… → 10.20.3.41:${f.port} (demo) · 18 ms`, r: "ok" });
    c.push({ label: f.tls === "disable" ? "Encryption" : `TLS handshake (${f.tls})`, detail: f.tls === "disable" ? "connection is NOT encrypted" : `TLSv1.3 · certificate valid for ${provider === "rds" ? "*.rds.amazonaws.com" : f.host} · expires 2027-03-14`, r: f.tls === "disable" ? "warn" : "ok" });
    c.push({ label: "Authenticate", detail: f.auth === "aws-iam" ? "IAM token from profile demo-readonly · 15 min lifetime" : f.auth === "password" ? "SCRAM-SHA-256 · password from keychain" : `${f.auth} token accepted`, r: "ok" });
    c.push({ label: "Server version", detail: `PostgreSQL ${version} — supported (14 or newer required)`, r: "ok" });
    c.push({ label: "Detect platform", detail: provider === "rds" ? "rds.extensions setting present → Amazon RDS · Multi-AZ instance" : provider === "self" ? "Patroni REST API on :8008 → 3-node cluster, leader pg-demo-1" : PROVIDER_LABEL[provider], r: "ok" });
    c.push({ label: "Monitoring privileges", detail: `${f.user} lacks pg_monitor — generated grant in step 4`, r: "warn" });
    c.push({ label: "Statistics extensions", detail: `${provider === "self" ? 4 : 2} of 5 recommended extensions missing — see next step`, r: "warn" });
    return c;
  }, [f, provider, version]);
  const { states, done, restart } = useSequence(checks.length, checks.map((c) => c.r));
  useEffect(() => onDone?.(done), [done, onDone]);
  const warns = checks.filter((c) => c.r === "warn").length;
  return (
    <div className="grid grid-cols-[1fr_340px] gap-4">
      <Card title="Testing connection" sub={<span className="mono">{f.name}</span>} actions={done && <button className="btn !h-6 text-[11.5px]" onClick={restart}>Run again</button>}>
        {checks.map((c, i) => <CheckRow key={c.label} state={states[i]} label={c.label} detail={c.detail} />)}
      </Card>
      <Card title="Result">
        {!done ? <div className="text-fg3 text-[12.5px]">Running {checks.length} checks…</div> : (
          <div className="animate-fadeIn">
            <div className="flex items-center gap-2 text-ok font-medium"><CheckCircle2 size={16} /> Connected</div>
            <div className="text-fg2 text-[12.5px] mt-2 leading-relaxed">The server is reachable and supported. {warns} item{warns === 1 ? "" : "s"} need setup before every feature works; the next steps generate exactly what to run.</div>
            <div className="flex flex-wrap gap-1.5 mt-3"><Badge>PG {version}</Badge><Badge color="var(--accent)">{provider === "self" ? "Patroni" : provider.toUpperCase()}</Badge><Badge color={f.tls === "disable" ? "var(--crit)" : "var(--ok)"}>{f.tls}</Badge></div>
          </div>
        )}
      </Card>
    </div>
  );
}

function Capabilities({ provider, version, picked, setPicked }: { provider: Provider; version: string; picked: string[]; setPicked: (x: string[]) => void }) {
  const managed = provider !== "self";
  const status = (e: (typeof EXTENSIONS)[number]) => {
    const s = managed ? e.rds : e.self;
    if (s === "ok" || s === "preloaded") return { label: "installed", color: "var(--ok)", can: false };
    if (s === "na") return { label: "not offered by provider", color: "var(--fg3)", can: false };
    return { label: "missing", color: "var(--warn)", can: true };
  };
  const caps = [
    { k: "PostgreSQL", v: `${version}`, ok: true, note: "14+ required" },
    { k: "Platform", v: managed ? PROVIDER_LABEL[provider].replace(" for PostgreSQL", "") : "Patroni 4.0 · etcd", ok: true, note: managed ? "managed · parameter groups" : "self-managed" },
    { k: "HA", v: managed ? "Multi-AZ standby" : "1 leader + 2 replicas", ok: true, note: managed ? "failover via provider" : "switchover supported" },
    { k: "Pooler", v: "none detected", ok: false, note: "PgBouncer recommended" },
  ];
  const toggle = (n: string) => setPicked(picked.includes(n) ? picked.filter((x) => x !== n) : [...picked, n]);
  const tabs = [
    { id: "sql", label: "setup.sql", text: preloadSql(picked, provider), file: "tuskline-setup.sql" },
    ...(managed ? [{ id: "cloud", label: provider === "rds" ? "aws cli" : provider === "cloudsql" ? "gcloud" : "az cli", text: cloudCommand(provider, picked), file: "tuskline-params.sh" }] : [{ id: "conf", label: "postgresql.conf", text: confPatch(picked), file: "tuskline.conf" }]),
    { id: "restart", label: "restart steps", text: restartSteps(provider), file: "restart.sh" },
  ];
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-4 gap-3">
        {caps.map((c) => (
          <div key={c.k} className="card p-3.5">
            <div className="label mb-1.5 flex items-center gap-1.5">{c.ok ? <CheckCircle2 size={11} className="text-ok" /> : <Sparkles size={11} className="text-warn" />}{c.k}</div>
            <div className="font-medium truncate">{c.v}</div><div className="text-fg3 text-[11.5px] mt-0.5">{c.note}</div>
          </div>
        ))}
      </div>
      <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)] gap-4">
        <Card title="Monitoring extensions" sub={`${picked.length} selected to install`} pad={false}>
          {EXTENSIONS.map((e) => {
            const st = status(e);
            return (
              <label key={e.name} className={`flex items-start gap-3 px-4 py-3 border-b border-line/70 last:border-0 ${st.can ? "cursor-pointer hover:bg-hover/40" : ""}`}>
                <input type="checkbox" className="mt-1 accent-[var(--brand)]" disabled={!st.can} checked={st.can ? picked.includes(e.name) : st.label === "installed"} onChange={() => toggle(e.name)} />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2"><span className="mono text-[12.5px] font-medium">{e.name}</span><Badge color={st.color}>{st.label}</Badge>{e.restart && st.can && <Badge color="var(--warn)">restart</Badge>}</div>
                  <div className="text-fg3 text-[12px] mt-0.5">{e.purpose}</div>
                </div>
              </label>
            );
          })}
        </Card>
        <Card title="Generated setup" sub={managed ? "managed service" : "self-managed"}>
          <CodeTabs key={provider + picked.join()} tabs={tabs} maxH={330} />
          {provider === "cloudsql" && <div className="text-[11.5px] text-warn mt-2">gcloud --database-flags replaces every existing flag: include the ones already set.</div>}
        </Card>
      </div>
    </div>
  );
}

function RoleStep({ provider, db, role, setRole, iam, setIam, operator, setOperator, verified, setVerified }: { provider: Provider; db: string; role: string; setRole: (s: string) => void; iam: boolean; setIam: (b: boolean) => void; operator: boolean; setOperator: (b: boolean) => void; verified: CheckState; setVerified: (s: CheckState) => void }) {
  const verify = () => { setVerified("running"); setTimeout(() => setVerified("ok"), 1100); };
  return (
    <div className="grid grid-cols-[320px_1fr] gap-4">
      <div className="space-y-4">
        <Card title="Role options">
          <div className="space-y-3.5">
            <Field label="Role name"><input className="input w-full mono" value={role} onChange={(e) => setRole(slug(e.target.value).replace(/-/g, "_"))} /></Field>
            <Toggle on={iam} onChange={setIam} label="IAM / token auth (no stored password)" />
            <Toggle on={operator} onChange={setOperator} label="Also create an operator role (cancel / terminate)" />
          </div>
          <div className="mt-4 pt-3 border-t border-line text-[12px] text-fg2 leading-relaxed space-y-1.5">
            <div className="flex gap-1.5"><Shield size={12} className="mt-0.5 shrink-0 text-ok" /> Read-only by default; cannot read table data beyond statistics views.</div>
            <div className="flex gap-1.5"><Shield size={12} className="mt-0.5 shrink-0 text-ok" /> Short statement and lock timeouts so monitoring never blocks your workload.</div>
          </div>
        </Card>
        <Card title="Verify">
          <CheckRow state={verified} label={`Log in as ${role} and read pg_stat_activity`} detail="pg_monitor granted · default_transaction_read_only = on" />
          {verified !== "ok" && <button className="btn w-full justify-center mt-2" onClick={verify} disabled={verified === "running"}>I’ve run it, verify now</button>}
        </Card>
      </div>
      <Card title="Least-privilege monitoring role" sub="run as a superuser or the provider’s admin role">
        <CodeTabs key={role + iam + operator} tabs={[{ id: "sql", label: "role.sql", text: roleSql({ role, db, iam, operator, provider }), file: "tuskline-role.sql" }, { id: "psql", label: "psql", text: `$ psql "host=… dbname=${db} user=postgres sslmode=verify-full" \\\n    ${iam ? "" : "-v tuskline_password=\"$(op read op://demo/tuskline/password)\" \\\n    "}-f tuskline-role.sql` }]} maxH={420} />
      </Card>
    </div>
  );
}

function FinishStep({ f, provider, version, picked, roleVerified }: { f: Form; provider: Provider; version: string; picked: string[]; roleVerified: boolean }) {
  const rows: [string, string][] = [
    ["Name", slug(f.name)], ["Environment", ENVS.find((e) => e.id === f.env)!.label], ["Endpoint", `${f.host}:${f.port}/${f.db}`],
    ["Transport", `${f.tls}${f.ssh ? ` · via ${f.bastion}` : ""}`], ["Platform", `${PROVIDER_LABEL[provider]} · PG ${version}`],
    ["Monitoring role", roleVerified ? "verified" : "not verified yet — limited metrics until it is"], ["Pending setup", picked.length ? `${picked.join(", ")} (restart)` : "none"],
  ];
  return (
    <div className="grid grid-cols-[1fr_340px] gap-4">
      <Card title="Ready to add">
        <dl className="divide-y divide-line/70">
          {rows.map(([k, v]) => <div key={k} className="flex gap-4 py-2"><dt className="w-36 text-fg3 shrink-0">{k}</dt><dd className={`min-w-0 break-all ${k === "Endpoint" || k === "Name" ? "mono text-[12.5px]" : ""}`} style={{ color: k === "Monitoring role" && !roleVerified ? "var(--warn)" : undefined }}>{v}</dd></div>)}
        </dl>
      </Card>
      <Card title="What happens next">
        <ol className="space-y-2.5 text-[12.5px] text-fg2 list-decimal pl-4">
          <li>The cluster appears in <b className="text-fg">Fleet</b> under {ENVS.find((e) => e.id === f.env)!.label}.</li>
          <li>Collection starts at 10 s resolution; history fills in over the first hour.</li>
          <li>Run the <b className="text-fg">Config advisor</b> for a scored review of memory, WAL and autovacuum settings.</li>
        </ol>
      </Card>
    </div>
  );
}
