<script lang="ts">
	import LoaderCircle from '@lucide/svelte/icons/loader-circle';
	import PlugZap from '@lucide/svelte/icons/plug-zap';
	import { ApiError, saveCluster, testCluster } from '#lib/api.js';
	import { envPillClass, type EnvKind } from '#lib/env.js';
	import { demoForced, type ShellCluster } from '#lib/fixtures/clusters.js';
	import {
		PROVIDER_LABEL,
		RDS_PRESET,
		SELF_PRESET,
		connectionProblems,
		detectProvider,
		draftProblems,
		localTlsHint,
		verifyDraft,
		type ConnectionDraft,
		type TlsMode,
		type WizardAuth
	} from '#lib/connect-wizard.js';
	import { shell } from '#lib/shell.svelte.js';

	let draft = $state<ConnectionDraft>({ ...RDS_PRESET });
	let password = $state('');
	let verifying = $state(false);
	let testing = $state(false);
	let success = $state(false);
	let error = $state('');
	let tested = $state('');
	let confirmName = $state('');
	let generation = 0;

	const editing = $derived(shell.editing);
	const prodBlocked = $derived(draft.env === 'prod' && confirmName.trim() !== draft.name.trim());

	const provider = $derived(detectProvider(draft));
	const rehearsal = $derived(demoForced(typeof location === 'undefined' ? '' : location.search));
	const busy = $derived(verifying || testing || success);
	const tlsHint = $derived(localTlsHint(draft));

	$effect(() => {
		const token = shell.connectToken;
		const current = shell.editing;
		if (!current) {
			if (token > 0) usePreset(RDS_PRESET);
			return;
		}
		const env = current.kind === 'staging' || current.kind === 'prod' ? current.kind : 'dev';
		const tls =
			current.tls === 'require' || current.tls === 'disable' ? current.tls : 'verify-full';
		generation += 1;
		draft = {
			name: current.name,
			host: current.host ?? '',
			port: String(current.port || 5432),
			database: current.database ?? 'postgres',
			user: current.user ?? '',
			auth: 'password',
			tls,
			ssh: false,
			bastion: '',
			env,
			passwordEnv: current.secret_env ?? ''
		};
		password = '';
		confirmName = '';
		error = '';
		tested = '';
		success = false;
	});
	const envs: EnvKind[] = ['dev', 'staging', 'prod'];
	const tlsModes: TlsMode[] = ['verify-full', 'require', 'disable'];

	function patch<K extends keyof ConnectionDraft>(key: K, value: ConnectionDraft[K]) {
		generation += 1;
		draft[key] = value;
		if (key === 'host' || key === 'port' || key === 'database' || key === 'user') password = '';
		verifying = false;
		testing = false;
		success = false;
		error = '';
		tested = '';
	}

	function usePreset(next: ConnectionDraft) {
		generation += 1;
		draft = { ...next };
		password = '';
		confirmName = '';
		verifying = false;
		testing = false;
		success = false;
		error = '';
		tested = '';
	}

	function wait(ms: number) {
		return new Promise((resolve) => setTimeout(resolve, ms));
	}

	function openDatabase() {
		const name = draft.name.trim();
		const id =
			name
				.toLowerCase()
				.replace(/[^a-z0-9-]+/g, '-')
				.replace(/^-|-$/g, '') || 'connection';
		const cluster: ShellCluster = {
			id,
			name,
			env: draft.env,
			health: 'ok',
			version: 'unknown',
			ha: PROVIDER_LABEL[provider],
			region: `${draft.host.trim()}:${draft.port.trim()}`,
			database: draft.database.trim()
		};
		shell.clusters = [cluster, ...shell.clusters.filter((item) => item.id !== id)];
		shell.setCluster(id);
		shell.go('sql');
	}

	function payload() {
		return {
			environment: draft.env,
			host: draft.host.trim(),
			port: Number(draft.port),
			database: draft.database.trim(),
			user: draft.user.trim(),
			tls: draft.tls,
			password_env: draft.passwordEnv.trim() || undefined,
			password: password || undefined
		};
	}

	async function test() {
		const ticket = ++generation;
		error = '';
		tested = '';
		const invalid = connectionProblems(draft);
		if (invalid.length) {
			error = invalid[0];
			return;
		}
		if (rehearsal) {
			error = 'This demo does not dial out.';
			return;
		}
		testing = true;
		try {
			const result = await testCluster(payload());
			if (ticket !== generation) return;
			const where = result.in_recovery
				? 'standby'
				: result.read_only
					? 'read-only'
					: 'writable primary';
			tested = `Reached ${result.version} on ${result.host}:${result.port}, database ${result.database}, as ${result.user}. ${where}. Nothing was saved.`;
		} catch (err) {
			if (ticket !== generation) return;
			const message = err instanceof ApiError ? err.message : 'Could not reach the server.';
			error =
				tlsHint && /tls|ssl|certificate|handshake/i.test(message)
					? `${message} ${tlsHint}`
					: message;
		} finally {
			if (ticket === generation) testing = false;
		}
	}

	async function create() {
		const ticket = ++generation;
		error = '';
		tested = '';
		success = false;
		const invalid = draftProblems(draft);
		if (invalid.length) {
			error = invalid[0];
			return;
		}
		verifying = true;
		if (rehearsal) {
			await wait(700);
			if (ticket !== generation) return;
			const failed = verifyDraft(draft);
			verifying = false;
			if (failed) {
				error = failed;
				return;
			}
			success = true;
			await wait(700);
			if (ticket !== generation) return;
			openDatabase();
			return;
		}
		const once = password;
		password = '';
		try {
			const saved = await saveCluster({
				...payload(),
				name: draft.name.trim(),
				password: once || undefined,
				confirm: draft.env === 'prod' ? confirmName.trim() : undefined
			});
			if (ticket !== generation) return;
			verifying = false;
			success = true;
			await wait(700);
			if (ticket !== generation) return;
			await shell.refresh();
			shell.setCluster(saved.id);
			shell.go('sql');
		} catch (err) {
			if (ticket !== generation) return;
			verifying = false;
			error = err instanceof ApiError ? err.message : 'Could not save the connection.';
		}
	}

	const field =
		'h-8 w-full rounded-md border border-tl-line bg-tl-subtle px-2 font-mono text-[13px] text-tl-fg outline-none disabled:opacity-60';
	const btn =
		'inline-flex h-7 items-center gap-1.5 rounded-md border border-tl-line bg-tl-elev px-2.5 text-[12.5px] font-medium hover:bg-tl-hover disabled:opacity-50';
	const okBtn = 'border-transparent bg-env-dev text-white';
</script>

<div class="mx-auto max-w-[980px]">
	<div class="mb-4">
		<h1 class="flex items-center gap-2 text-xl font-semibold tracking-tight">
			<PlugZap size={18} class="text-tl-accent" />
			{editing ? 'Edit connection' : 'Add connection'}
		</h1>
		<p class="mt-1 text-[13px] text-tl-fg2">
			{#if rehearsal}
				This demo checks the form and does not dial out or save anything.
			{:else if editing}
				Environment and name stay as they are. An empty password keeps the current one. Test
				connection still saves nothing.
			{:else}
				Test connection checks the server and saves nothing. Create does the same check, then saves
				a typed password in the OS keychain.
			{/if}
		</p>
	</div>

	<div class="rounded-xl border border-tl-line bg-tl-elev p-4">
		<div class="mb-3 flex flex-wrap items-center gap-2 text-[12px] text-tl-fg3">
			Fill demo
			<button class={btn} onclick={() => usePreset(RDS_PRESET)} disabled={busy}>Amazon RDS</button>
			<button class={btn} onclick={() => usePreset(SELF_PRESET)} disabled={busy}
				>Self-managed</button
			>
		</div>
		<div class="grid gap-3 sm:grid-cols-2">
			<label class="text-[12px] text-tl-fg3"
				>Display name
				<input
					class="{field} mt-1"
					value={draft.name}
					disabled={busy || editing !== null}
					oninput={(e) => patch('name', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>Host
				<input
					class="{field} mt-1"
					value={draft.host}
					disabled={busy}
					oninput={(e) => patch('host', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>Port
				<input
					class="{field} mt-1"
					value={draft.port}
					disabled={busy}
					oninput={(e) => patch('port', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>Database
				<input
					class="{field} mt-1"
					value={draft.database}
					disabled={busy}
					oninput={(e) => patch('database', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>User
				<input
					class="{field} mt-1"
					value={draft.user}
					disabled={busy}
					oninput={(e) => patch('user', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>Authentication
				<select
					class="{field} mt-1"
					value={draft.auth}
					disabled={busy}
					onchange={(e) => patch('auth', e.currentTarget.value as WizardAuth)}
				>
					<option value="password">Password (~/.pgpass or an environment variable)</option>
					<option value="aws-iam">AWS IAM token</option>
					<option value="gcp-iam">Google Cloud IAM</option>
					<option value="azure-entra">Microsoft Entra ID</option>
				</select>
			</label>
			{#if draft.auth === 'password' && !rehearsal}
				<label class="text-[12px] text-tl-fg3 sm:col-span-2"
					>Password environment variable
					<input
						class="{field} mt-1"
						value={draft.passwordEnv}
						autocomplete="off"
						spellcheck="false"
						placeholder="ORDERS_PASSWORD"
						disabled={busy}
						oninput={(e) => patch('passwordEnv', e.currentTarget.value)}
					/>
				</label>
				<label class="text-[12px] text-tl-fg3 sm:col-span-2"
					>Password for this probe only
					<input
						class="{field} mt-1"
						type="password"
						value={password}
						autocomplete="off"
						disabled={busy}
						oninput={(e) => {
							generation += 1;
							password = e.currentTarget.value;
							error = '';
						}}
					/>
				</label>
				<p class="text-[12px] text-tl-fg3 sm:col-span-2">
					A typed password is saved in the OS keychain when you create the connection. The inventory
					stores only the reference. Leave the password empty to use the environment variable or
					~/.pgpass. It is not kept in this page.
				</p>
			{/if}
		</div>
		<div class="mt-4">
			<div class="mb-1.5 text-[12px] text-tl-fg3">TLS</div>
			<div class="flex flex-wrap gap-1.5">
				{#each tlsModes as mode (mode)}
					<button
						class="{btn} {success && draft.tls === mode
							? okBtn
							: draft.tls === mode
								? 'border-tl-accent text-tl-accent'
								: ''}"
						aria-pressed={draft.tls === mode}
						disabled={busy}
						onclick={() => patch('tls', mode)}>{mode}</button
					>
				{/each}
			</div>
			{#if tlsHint}
				<p class="mt-1.5 text-[12px] text-tl-fg2">{tlsHint}</p>
			{:else if draft.tls === 'disable'}
				<p class="mt-1.5 text-[12px] text-env-prod">
					Unencrypted. Only for a local disposable database.
				</p>
			{:else if draft.tls === 'require'}
				<p class="mt-1.5 text-[12px] text-tl-fg2">
					Encrypted, but the server certificate is not checked.
				</p>
			{/if}
		</div>
		<div class="mt-4">
			<div class="mb-1.5 text-[12px] text-tl-fg3">Environment</div>
			<div class="flex flex-wrap gap-1.5">
				{#each envs as kind (kind)}
					<button
						class="h-7 rounded-md px-3 text-[12px] font-medium uppercase {success &&
						draft.env === kind
							? okBtn
							: draft.env === kind
								? envPillClass(kind)
								: 'text-tl-fg3 ring-1 ring-tl-line ring-inset'}"
						aria-pressed={draft.env === kind}
						disabled={busy || editing !== null}
						onclick={() => patch('env', kind)}>{kind}</button
					>
				{/each}
			</div>
			{#if draft.env === 'prod'}
				<p class="mt-1.5 text-[12px] text-tl-fg2">
					Production sessions open read-only. Save stays disabled until the cluster name is typed.
				</p>
				<label class="mt-2 block text-[12px] text-tl-fg3"
					>Type {draft.name || 'the cluster name'} to save
					<input
						class="{field} mt-1"
						value={confirmName}
						disabled={busy}
						oninput={(e) => (confirmName = e.currentTarget.value)}
					/>
				</label>
			{/if}
		</div>
		<label class="mt-4 flex items-center gap-2 text-[13px]">
			<input
				type="checkbox"
				checked={draft.ssh}
				disabled={busy}
				onchange={(e) => patch('ssh', e.currentTarget.checked)}
			/>
			Connect through an SSH bastion
		</label>
		{#if draft.ssh}
			<label class="mt-2 block text-[12px] text-tl-fg3"
				>Bastion host
				<input
					class="{field} mt-1"
					value={draft.bastion}
					disabled={busy}
					oninput={(e) => patch('bastion', e.currentTarget.value)}
				/>
			</label>
		{/if}
		<p class="mt-4 text-[12.5px] text-tl-fg2">
			Detected from the host: {PROVIDER_LABEL[provider]}.
		</p>
		{#if error}
			<p class="mt-3 text-[13px] text-env-prod" role="alert">{error}</p>
		{:else if tested}
			<p class="mt-3 text-[13px] text-env-dev" role="status">{tested}</p>
		{/if}
	</div>

	<div class="mt-4 flex items-center gap-2 border-t border-tl-line pt-4">
		<button class={btn} onclick={() => shell.go('fleet')} disabled={busy}>Cancel</button>
		<button class="{btn} ml-auto" onclick={test} disabled={busy}>
			{#if testing}
				<LoaderCircle size={14} class="animate-spin" /> Testing…
			{:else}
				Test connection
			{/if}
		</button>
		<button
			class="{btn} border-transparent text-white {success ? 'bg-env-dev' : 'bg-brand'}"
			onclick={create}
			disabled={busy || prodBlocked}
		>
			{#if verifying}
				<LoaderCircle size={14} class="animate-spin" /> Verifying…
			{:else if success}
				Connected
			{:else}
				{editing ? 'Save connection' : 'Create connection'}
			{/if}
		</button>
	</div>
</div>
