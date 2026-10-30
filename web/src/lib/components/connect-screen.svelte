<script lang="ts">
	import LoaderCircle from '@lucide/svelte/icons/loader-circle';
	import PlugZap from '@lucide/svelte/icons/plug-zap';
	import { ApiError, saveCluster } from '#lib/api.js';
	import { envPillClass, type EnvKind } from '#lib/env.js';
	import { demoForced, type ShellCluster } from '#lib/fixtures/clusters.js';
	import {
		PROVIDER_LABEL,
		RDS_PRESET,
		SELF_PRESET,
		detectProvider,
		draftProblems,
		verifyDraft,
		type ConnectionDraft,
		type TlsMode,
		type WizardAuth
	} from '#lib/connect-wizard.js';
	import { shell } from '#lib/shell.svelte.js';

	let draft = $state<ConnectionDraft>({ ...RDS_PRESET });
	let password = $state('');
	let verifying = $state(false);
	let success = $state(false);
	let error = $state('');
	let generation = 0;

	const provider = $derived(detectProvider(draft));
	const rehearsal = $derived(demoForced(typeof location === 'undefined' ? '' : location.search));
	const envs: EnvKind[] = ['dev', 'staging', 'prod'];
	const tlsModes: TlsMode[] = ['verify-full', 'require', 'disable'];

	function patch<K extends keyof ConnectionDraft>(key: K, value: ConnectionDraft[K]) {
		generation += 1;
		draft[key] = value;
		if (key === 'host' || key === 'port' || key === 'database' || key === 'user') password = '';
		verifying = false;
		success = false;
		error = '';
	}

	function usePreset(next: ConnectionDraft) {
		generation += 1;
		draft = { ...next };
		password = '';
		verifying = false;
		success = false;
		error = '';
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

	async function create() {
		const ticket = ++generation;
		error = '';
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
				environment: draft.env,
				name: draft.name.trim(),
				host: draft.host.trim(),
				port: Number(draft.port),
				database: draft.database.trim(),
				user: draft.user.trim(),
				tls: draft.tls,
				password_env: draft.passwordEnv.trim() || undefined,
				password: once || undefined
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
			<PlugZap size={18} class="text-tl-accent" /> Add connection
		</h1>
		<p class="mt-1 text-[13px] text-tl-fg2">
			{#if rehearsal}
				This demo checks the form and does not dial out or save anything.
			{:else}
				Create probes the server and saves it. A password is used for that probe only and is not
				stored.
			{/if}
		</p>
	</div>

	<div class="rounded-xl border border-tl-line bg-tl-elev p-4">
		<div class="mb-3 flex flex-wrap items-center gap-2 text-[12px] text-tl-fg3">
			Fill demo
			<button class={btn} onclick={() => usePreset(RDS_PRESET)} disabled={verifying}
				>Amazon RDS</button
			>
			<button class={btn} onclick={() => usePreset(SELF_PRESET)} disabled={verifying}
				>Self-managed</button
			>
		</div>
		<div class="grid gap-3 sm:grid-cols-2">
			<label class="text-[12px] text-tl-fg3"
				>Display name
				<input
					class="{field} mt-1"
					value={draft.name}
					disabled={verifying || success}
					oninput={(e) => patch('name', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>Host
				<input
					class="{field} mt-1"
					value={draft.host}
					disabled={verifying || success}
					oninput={(e) => patch('host', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>Port
				<input
					class="{field} mt-1"
					value={draft.port}
					disabled={verifying || success}
					oninput={(e) => patch('port', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>Database
				<input
					class="{field} mt-1"
					value={draft.database}
					disabled={verifying || success}
					oninput={(e) => patch('database', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>User
				<input
					class="{field} mt-1"
					value={draft.user}
					disabled={verifying || success}
					oninput={(e) => patch('user', e.currentTarget.value)}
				/>
			</label>
			<label class="text-[12px] text-tl-fg3"
				>Authentication
				<select
					class="{field} mt-1"
					value={draft.auth}
					disabled={verifying || success}
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
						disabled={verifying || success}
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
						disabled={verifying || success}
						oninput={(e) => {
							generation += 1;
							password = e.currentTarget.value;
							error = '';
						}}
					/>
				</label>
				<p class="text-[12px] text-tl-fg3 sm:col-span-2">
					Leave both empty to use ~/.pgpass. The password is not written to the inventory or kept in
					this page.
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
						disabled={verifying || success}
						onclick={() => patch('tls', mode)}>{mode}</button
					>
				{/each}
			</div>
			{#if draft.tls === 'disable'}
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
						disabled={verifying || success}
						onclick={() => patch('env', kind)}>{kind}</button
					>
				{/each}
			</div>
			{#if draft.env === 'prod'}
				<p class="mt-1.5 text-[12px] text-tl-fg2">
					Production sessions open read-only. Writes need a typed confirmation.
				</p>
			{/if}
		</div>
		<label class="mt-4 flex items-center gap-2 text-[13px]">
			<input
				type="checkbox"
				checked={draft.ssh}
				disabled={verifying || success}
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
					disabled={verifying || success}
					oninput={(e) => patch('bastion', e.currentTarget.value)}
				/>
			</label>
		{/if}
		<p class="mt-4 text-[12.5px] text-tl-fg2">
			Detected from the host: {PROVIDER_LABEL[provider]}.
		</p>
		{#if error}
			<p class="mt-3 text-[13px] text-env-prod" role="alert">{error}</p>
		{/if}
	</div>

	<div class="mt-4 flex items-center gap-2 border-t border-tl-line pt-4">
		<button class={btn} onclick={() => shell.go('fleet')} disabled={verifying || success}
			>Cancel</button
		>
		<button
			class="{btn} ml-auto border-transparent text-white {success ? 'bg-env-dev' : 'bg-brand'}"
			onclick={create}
			disabled={verifying || success}
		>
			{#if verifying}
				<LoaderCircle size={14} class="animate-spin" /> Verifying…
			{:else if success}
				Connected
			{:else}
				Create connection
			{/if}
		</button>
	</div>
</div>
