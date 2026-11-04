<script lang="ts">
	import { recordDeployment } from '#lib/deploy-log.svelte.js';
	import { shell } from '#lib/shell.svelte.js';

	const steps = ['Target', 'Topology', 'Sizing & backups', 'Review'];
	let step = $state(0);
	let phase = $state<'form' | 'running' | 'done'>('form');

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
			<code class="font-mono">tl provision compose plan</code> writes the files.
			<code class="font-mono">tl provision compose up</code> starts them. This server does not launch
			containers from the browser.
		</p>
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
