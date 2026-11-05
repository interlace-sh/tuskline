<script lang="ts">
	import { ApiError, provisionPlan, type PlanChange } from '#lib/api.js';
	import { recordDeployment } from '#lib/deploy-log.svelte.js';
	import { shell } from '#lib/shell.svelte.js';

	const steps = ['Target', 'Topology', 'Sizing & backups', 'Review'];
	let step = $state(0);
	let phase = $state<'form' | 'running' | 'done'>('form');
	let target = $state<'compose' | 'cnpg' | 'ssh'>('compose');
	let planName = $state('demo');
	let hosts = $state('db1,db2');
	let plan = $state<PlanChange[]>([]);
	let planError = $state('');

	async function generate() {
		planError = '';
		plan = [];
		try {
			const body = await provisionPlan(target, planName.trim() || 'demo', hosts);
			plan = body.changes ?? [];
		} catch (err) {
			planError = err instanceof ApiError ? err.message : 'Could not build the plan.';
		}
	}

	const btn =
		'inline-flex h-8 items-center rounded-md border border-tl-line bg-tl-elev px-3 text-[13px] font-medium hover:bg-tl-hover';

	function next() {
		step = Math.min(step + 1, steps.length - 1);
	}

	function deploy() {
		phase = 'running';
	}

	function finish() {
		recordDeployment({
			name: 'orders-demo',
			target: 'compose',
			env: 'staging',
			when: 'just now'
		});
		shell.addCluster({
			id: 'orders-demo',
			name: 'orders-demo',
			env: 'staging',
			health: 'ok',
			version: '17.0',
			ha: 'Patroni',
			region: 'docker · demo-host',
			database: 'postgres'
		});
		phase = 'done';
	}
</script>

<div class="mb-5">
	<h1 class="text-xl font-semibold tracking-tight">Deploy cluster</h1>
	<p class="mt-1 text-[13px] text-tl-fg2">
		A preview of a non-production Compose stack. Deploy here records a demo history row and does not
		start Docker.
	</p>
</div>

{#if !shell.demo}
	<section class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4">
		<h2 class="text-[13px] font-semibold">Start the stack from the machine with Docker</h2>
		<p class="mt-1 text-[13px] text-tl-fg2">
			Generate a Compose, CloudNativePG or SSH plan. Secrets are redacted. Starting Docker stays on
			<code class="font-mono">tl provision compose up</code>.
		</p>
		<div class="mt-3 flex flex-wrap gap-2">
			<select
				class="h-8 rounded-md border border-tl-line bg-tl-bg px-2 text-[13px]"
				bind:value={target}
			>
				<option value="compose">Compose</option>
				<option value="cnpg">CloudNativePG</option>
				<option value="ssh">SSH</option>
			</select>
			<input
				class="h-8 rounded-md border border-tl-line bg-tl-bg px-2 font-mono text-[13px]"
				bind:value={planName}
			/>
			{#if target === 'ssh'}
				<input
					class="h-8 rounded-md border border-tl-line bg-tl-bg px-2 font-mono text-[13px]"
					bind:value={hosts}
				/>
			{/if}
			<button class={btn} onclick={generate}>Generate plan</button>
		</div>
		{#if planError}<p class="mt-2 text-[13px] text-env-prod">{planError}</p>{/if}
		{#each plan as change (change.ID)}
			<pre
				class="mt-3 overflow-auto font-mono text-[11px] whitespace-pre-wrap">{change.Path}{'\n'}{change.Diff}</pre>
		{/each}
	</section>
{:else if phase === 'form'}
	<ol class="mb-4 flex flex-wrap gap-2 text-[12px]">
		{#each steps as label, index (label)}
			<li
				class="rounded-md px-2 py-1 {index === step
					? 'bg-tl-subtle font-medium text-tl-fg'
					: 'text-tl-fg3'}"
			>
				{label}
			</li>
		{/each}
	</ol>
	<section class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4">
		{#if step === 0}
			<h2 class="text-[13px] font-semibold">Docker Compose</h2>
			<p class="mt-1 text-[13px] text-tl-fg2">
				One host, Patroni and etcd. Not real HA: the host is a single point of failure.
			</p>
		{:else if step === 1}
			<h2 class="text-[13px] font-semibold">3 nodes · etcd ×3</h2>
			<p class="mt-1 text-[13px] text-tl-fg2">
				PostgreSQL 17, HAProxy, PgBouncer, local pgBackRest.
			</p>
		{:else if step === 2}
			<h2 class="text-[13px] font-semibold">Small</h2>
			<p class="mt-1 text-[13px] text-tl-fg2">A starting size for a demo. Retention 14 days.</p>
		{:else}
			<h2 class="text-[13px] font-semibold">Review</h2>
			<p class="mt-1 font-mono text-[12.5px] text-tl-fg2">
				compose.yaml · haproxy.cfg · pgbackrest.conf
			</p>
		{/if}
		<div class="mt-4">
			{#if step < 3}
				<button class="{btn} border-transparent bg-brand text-white" onclick={next}>Continue</button
				>
			{:else}
				<button class="{btn} border-transparent bg-brand text-white" onclick={deploy}
					>Deploy orders-demo</button
				>
			{/if}
		</div>
	</section>
{:else if phase === 'running'}
	<section class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4">
		<h2 class="text-[13px] font-semibold">Deploying</h2>
		<p class="mt-1 text-[13px] text-tl-fg2">
			The real stack is started with tl provision compose up.
		</p>
		<button class="{btn} mt-4" onclick={finish}>Fast-forward</button>
	</section>
{:else}
	<section class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4">
		<h2 class="text-[13px] font-semibold">Deployment complete</h2>
		<p class="mt-1 text-[13px] text-tl-fg2">
			orders-demo is in the demo fleet and the history table.
		</p>
		<button
			class="{btn} mt-4 border-transparent bg-brand text-white"
			onclick={() => shell.go('fleet')}>View in Fleet</button
		>
	</section>
{/if}
