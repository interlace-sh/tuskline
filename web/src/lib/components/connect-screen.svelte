<script lang="ts">
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import PlugZap from '@lucide/svelte/icons/plug-zap';
	import { envPillClass, type EnvKind } from '#lib/env.js';
	import {
		PROVIDER_LABEL,
		RDS_PRESET,
		SELF_PRESET,
		WIZARD_STEPS,
		demoTestReport,
		detectProvider,
		draftProblems,
		type ConnectionDraft,
		type DemoTestLine,
		type TlsMode,
		type WizardAuth
	} from '#lib/connect-wizard.js';
	import { shell } from '#lib/shell.svelte.js';

	let step = $state(0);
	let draft = $state<ConnectionDraft>({ ...RDS_PRESET });
	let testing = $state(false);
	let tested = $state(false);
	let report = $state<DemoTestLine[]>([]);
	let generation = 0;

	const problems = $derived(draftProblems(draft));
	const provider = $derived(detectProvider(draft));
	const ready = $derived(problems.length === 0);
	const envs: EnvKind[] = ['dev', 'staging', 'prod'];
	const tlsModes: TlsMode[] = ['verify-full', 'require', 'disable'];

	function patch<K extends keyof ConnectionDraft>(key: K, value: ConnectionDraft[K]) {
		generation += 1;
		draft[key] = value;
		tested = false;
		testing = false;
		report = [];
	}

	function usePreset(next: ConnectionDraft) {
		generation += 1;
		draft = { ...next };
		tested = false;
		testing = false;
		report = [];
	}

	async function runTest() {
		if (!ready) return;
		const ticket = ++generation;
		testing = true;
		tested = false;
		await new Promise((resolve) => setTimeout(resolve, 350));
		if (ticket !== generation) return;
		report = demoTestReport(draft);
		tested = true;
		testing = false;
	}

	const field =
		'h-8 w-full rounded-md border border-tl-line bg-tl-subtle px-2 font-mono text-[13px] text-tl-fg outline-none';
	const btn =
		'inline-flex h-7 items-center gap-1.5 rounded-md border border-tl-line bg-tl-elev px-2.5 text-[12.5px] font-medium hover:bg-tl-hover disabled:opacity-50';
</script>

<div class="mx-auto max-w-[980px]">
	<div class="mb-4 flex items-start justify-between gap-3">
		<div>
			<h1 class="flex items-center gap-2 text-xl font-semibold tracking-tight">
				<PlugZap size={18} class="text-tl-accent" /> Add connection
			</h1>
			<p class="mt-1 text-[13px] text-tl-fg2">
				Describe a PostgreSQL 14+ server. The test stays on this machine and does not open a
				connection. Saving it comes later.
			</p>
		</div>
		<button class={btn} onclick={() => shell.go('fleet')}>Cancel</button>
	</div>

	<ol class="mb-4 flex gap-2 text-[12px]">
		{#each WIZARD_STEPS as label, index (label)}
			<li>
				<button
					class="rounded-full px-2.5 py-1 {index === step
						? 'bg-tl-accent text-white'
						: 'text-tl-fg3 hover:text-tl-fg'}"
					onclick={() => {
						if (index === 0 || (index === 1 && ready) || (index === 2 && tested)) step = index;
					}}
				>
					{index + 1}. {label}
				</button>
			</li>
		{/each}
	</ol>

	<div class="rounded-xl border border-tl-line bg-tl-elev p-4">
		{#if step === 0}
			<div class="mb-3 flex flex-wrap items-center gap-2 text-[12px] text-tl-fg3">
				Fill demo
				<button class={btn} onclick={() => usePreset(RDS_PRESET)}>Amazon RDS</button>
				<button class={btn} onclick={() => usePreset(SELF_PRESET)}>Self-managed</button>
			</div>
			<div class="grid gap-3 sm:grid-cols-2">
				<label class="text-[12px] text-tl-fg3"
					>Display name
					<input
						class="{field} mt-1"
						value={draft.name}
						oninput={(e) => patch('name', e.currentTarget.value)}
					/>
				</label>
				<label class="text-[12px] text-tl-fg3"
					>Host
					<input
						class="{field} mt-1"
						value={draft.host}
						oninput={(e) => patch('host', e.currentTarget.value)}
					/>
				</label>
				<label class="text-[12px] text-tl-fg3"
					>Port
					<input
						class="{field} mt-1"
						value={draft.port}
						oninput={(e) => patch('port', e.currentTarget.value)}
					/>
				</label>
				<label class="text-[12px] text-tl-fg3"
					>Database
					<input
						class="{field} mt-1"
						value={draft.database}
						oninput={(e) => patch('database', e.currentTarget.value)}
					/>
				</label>
				<label class="text-[12px] text-tl-fg3"
					>User
					<input
						class="{field} mt-1"
						value={draft.user}
						oninput={(e) => patch('user', e.currentTarget.value)}
					/>
				</label>
				<label class="text-[12px] text-tl-fg3"
					>Authentication
					<select
						class="{field} mt-1"
						value={draft.auth}
						onchange={(e) => patch('auth', e.currentTarget.value as WizardAuth)}
					>
						<option value="password">Password (secret store)</option>
						<option value="aws-iam">AWS IAM token</option>
						<option value="gcp-iam">Google Cloud IAM</option>
						<option value="azure-entra">Microsoft Entra ID</option>
					</select>
				</label>
			</div>
			<div class="mt-4">
				<div class="mb-1.5 text-[12px] text-tl-fg3">TLS</div>
				<div class="flex flex-wrap gap-1.5">
					{#each tlsModes as mode (mode)}
						<button
							class="{btn} {draft.tls === mode ? 'border-tl-accent text-tl-accent' : ''}"
							aria-pressed={draft.tls === mode}
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
							class="h-7 rounded-md px-3 text-[12px] font-medium uppercase {draft.env === kind
								? envPillClass(kind)
								: 'text-tl-fg3 ring-1 ring-tl-line ring-inset'}"
							aria-pressed={draft.env === kind}
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
						oninput={(e) => patch('bastion', e.currentTarget.value)}
					/>
				</label>
			{/if}
			<p class="mt-4 text-[12.5px] text-tl-fg2">
				Detected from the host: {PROVIDER_LABEL[provider]}.
			</p>
			{#if problems.length}
				<ul class="mt-2 list-disc pl-5 text-[12.5px] text-env-prod">
					{#each problems as problem (problem)}
						<li>{problem}</li>
					{/each}
				</ul>
			{/if}
		{:else if step === 1}
			<p class="text-[13px] text-tl-fg2">
				This check looks at the form only. It does not dial {draft.host}.
			</p>
			<button
				class="{btn} mt-3 border-transparent bg-brand text-white"
				onclick={runTest}
				disabled={testing}
			>
				{testing ? 'Checking…' : 'Test connection'}
			</button>
			{#if report.length}
				<ul class="mt-4 space-y-1.5 text-[13px]">
					{#each report as line (line.text)}
						<li class={line.tone === 'warn' ? 'text-env-prod' : 'text-tl-fg'}>{line.text}</li>
					{/each}
				</ul>
			{/if}
		{:else}
			<h2 class="text-[15px] font-semibold">Nothing was saved</h2>
			<p class="mt-2 text-[13px] leading-relaxed text-tl-fg2">
				{draft.name} would be a {draft.env} connection to {PROVIDER_LABEL[provider]}. The inventory
				write and a live probe are the next step, so this draft stays in the browser.
			</p>
			<p class="mt-2 font-mono text-[12.5px] text-tl-fg2">
				{draft.user}@{draft.host}:{draft.port}/{draft.database} · {draft.tls}
			</p>
		{/if}
	</div>

	<div class="mt-4 flex items-center gap-2 border-t border-tl-line pt-4">
		{#if step > 0}
			<button class={btn} onclick={() => (step -= 1)}><ArrowLeft size={13} /> Back</button>
		{/if}
		<span class="text-[12px] text-tl-fg3">Step {step + 1} of {WIZARD_STEPS.length}</span>
		{#if step < 2}
			<button
				class="{btn} ml-auto border-transparent bg-brand text-white"
				disabled={(step === 0 && !ready) || (step === 1 && !tested)}
				onclick={() => (step += 1)}
			>
				{step === 0 ? 'Test connection' : 'Continue'}
				<ArrowRight size={13} />
			</button>
		{:else}
			<button
				class="{btn} ml-auto border-transparent bg-brand text-white"
				onclick={() => shell.go('fleet')}
			>
				Back to fleet
			</button>
		{/if}
	</div>
</div>
