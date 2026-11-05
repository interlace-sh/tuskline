<script lang="ts">
	import { ApiError, lintSQL, schemaDiff, splitRef } from '#lib/api.js';
	import { MIGRATIONS, type Migration } from '#lib/fixtures/demo.js';
	import { shell } from '#lib/shell.svelte.js';

	let sql = $state('CREATE INDEX orders_note ON orders (note);\n');
	let lintReport = $state('');
	let lintError = $state('');
	let linting = $state(false);
	let fromId = $state('');
	let toId = $state('');
	let driftReport = $state('');
	let driftError = $state('');
	let diffing = $state(false);
	let migrateNote = $state('');
	let migrateConfirm = $state('');

	async function upload(run: boolean) {
		const input = document.getElementById('migrate-files') as HTMLInputElement | null;
		const files = input?.files;
		if (!files?.length || !shell.cluster.id.includes('/')) return;
		const ref = splitRef(shell.cluster.id);
		const body = new FormData();
		for (const file of files) body.append('file', file);
		body.set('run', run ? 'true' : 'false');
		body.set('approve', 'true');
		body.set('confirm', migrateConfirm.trim());
		migrateNote = '';
		try {
			const response = await fetch(
				`/api/v1/clusters/${encodeURIComponent(ref.env)}/${encodeURIComponent(ref.name)}/migrate`,
				{ method: 'POST', body }
			);
			const payload = (await response.json()) as {
				error?: string;
				preview?: string;
				lint?: string;
				applied?: string[];
			};
			if (!response.ok) {
				migrateNote = payload.error ?? response.statusText;
				return;
			}
			migrateNote =
				`${payload.lint ?? ''}\n${payload.preview ?? ''}\n${(payload.applied ?? []).map((name) => `applied ${name}`).join('\n')}`.trim();
		} catch {
			migrateNote = 'Could not upload the scripts.';
		}
	}

	const liveClusters = $derived(shell.clusters.filter((cluster) => cluster.id.includes('/')));

	async function lint() {
		linting = true;
		lintError = '';
		lintReport = '';
		try {
			const result = await lintSQL(sql);
			lintReport = result.report || 'No lock findings.';
		} catch (err) {
			lintError = err instanceof ApiError ? err.message : 'Could not lint the script.';
		} finally {
			linting = false;
		}
	}

	async function compare() {
		if (!fromId || !toId) return;
		diffing = true;
		driftError = '';
		driftReport = '';
		try {
			const result = await schemaDiff(fromId, toId);
			driftReport = result.report;
		} catch (err) {
			driftError = err instanceof ApiError ? err.message : 'Could not compare the schemas.';
		} finally {
			diffing = false;
		}
	}

	let selectedId = $state('0142');
	let rewritten = $state(false);
	let approved = $state(false);

	const selected = $derived(MIGRATIONS.find((item) => item.id === selectedId) ?? MIGRATIONS[0]);
	const is142 = $derived(selected.id === '0142');
	const canApprove = $derived(is142 && rewritten && !approved);

	const safeSql = `-- 0142_add_fulfilment_centre.sql
SET lock_timeout = '3s';

ALTER TABLE public.orders
  ADD COLUMN fulfilment_centre_id bigint;

ALTER TABLE public.orders
  ADD CONSTRAINT orders_fulfilment_centre_fk
  FOREIGN KEY (fulfilment_centre_id)
  REFERENCES public.fulfilment_centres (id) NOT VALID;
ALTER TABLE public.orders VALIDATE CONSTRAINT orders_fulfilment_centre_fk;

CREATE INDEX CONCURRENTLY idx_orders_fulfilment_centre
  ON public.orders (fulfilment_centre_id);

ALTER TABLE public.orders
  ADD CONSTRAINT orders_currency_nn CHECK (currency IS NOT NULL) NOT VALID;
ALTER TABLE public.orders VALIDATE CONSTRAINT orders_currency_nn;
ALTER TABLE public.orders ALTER COLUMN currency SET NOT NULL;`;

	const risks = [
		{
			stmt: 'ADD CONSTRAINT … FOREIGN KEY',
			note: 'Validates every row while blocking writes.'
		},
		{
			stmt: 'CREATE INDEX',
			note: 'Blocks writes for the whole build. Use CONCURRENTLY.'
		},
		{
			stmt: 'SET NOT NULL',
			note: 'Scans the table under a lock.'
		}
	];
	const safeRisks = [
		{ stmt: 'NOT VALID, then VALIDATE', note: 'Validation does not block writes.' },
		{
			stmt: 'CREATE INDEX CONCURRENTLY',
			note: 'Does not block writes. Cannot run inside a transaction.'
		},
		{ stmt: 'CHECK, then SET NOT NULL', note: 'The scan is skipped once the check is valid.' }
	];

	const btn =
		'inline-flex h-8 items-center rounded-md border border-tl-line bg-tl-elev px-3 text-[13px] font-medium hover:bg-tl-hover disabled:opacity-40';

	function statusOf(item: Migration): string {
		if (item.id === '0142' && approved) return 'approved';
		return item.status;
	}
</script>

{#if !shell.demo}
	<div class="mb-5">
		<h1 class="text-xl font-semibold tracking-tight">Migrations</h1>
		<p class="mt-1 max-w-2xl text-[13px] text-tl-fg2">
			Lint a script and compare two clusters. Applying a directory stays on
			<code class="font-mono">tl migrate apply --run</code>, because the files live on the machine
			that runs the command.
		</p>
	</div>
	<div class="grid gap-4 lg:grid-cols-2">
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
			<h2 class="text-[13px] font-semibold">Lock check</h2>
			<textarea
				class="mt-2 h-40 w-full rounded-md border border-tl-line bg-tl-bg p-2 font-mono text-[12px]"
				bind:value={sql}></textarea>
			<button
				class="mt-2 inline-flex h-8 items-center rounded-md border border-tl-line px-3 text-[13px]"
				onclick={lint}
				disabled={linting}>{linting ? 'Checking…' : 'Lint'}</button
			>
			{#if lintError}<p class="mt-2 text-[13px] text-env-prod">{lintError}</p>{/if}
			{#if lintReport}
				<pre class="mt-2 overflow-auto font-mono text-[12px] whitespace-pre-wrap">{lintReport}</pre>
			{/if}
		</section>
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
			<h2 class="text-[13px] font-semibold">Schema drift</h2>
			<div class="mt-2 flex flex-wrap gap-2">
				<select
					class="h-8 rounded-md border border-tl-line bg-tl-bg px-2 text-[13px]"
					bind:value={fromId}
				>
					<option value="">From</option>
					{#each liveClusters as cluster (cluster.id)}
						<option value={cluster.id}>{cluster.id}</option>
					{/each}
				</select>
				<select
					class="h-8 rounded-md border border-tl-line bg-tl-bg px-2 text-[13px]"
					bind:value={toId}
				>
					<option value="">To</option>
					{#each liveClusters as cluster (cluster.id)}
						<option value={cluster.id}>{cluster.id}</option>
					{/each}
				</select>
				<button
					class="inline-flex h-8 items-center rounded-md border border-tl-line px-3 text-[13px] disabled:opacity-40"
					onclick={compare}
					disabled={diffing || !fromId || !toId}>{diffing ? 'Comparing…' : 'Compare'}</button
				>
			</div>
			{#if driftError}<p class="mt-2 text-[13px] text-env-prod">{driftError}</p>{/if}
			{#if driftReport}
				<pre
					class="mt-2 overflow-auto font-mono text-[12px] whitespace-pre-wrap">{driftReport}</pre>
			{/if}
		</section>
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4 lg:col-span-2">
			<h2 class="text-[13px] font-semibold">Apply a directory to {shell.cluster.name}</h2>
			<p class="mt-1 text-[13px] text-tl-fg2">
				Upload the .sql files. Preview does not connect. Apply writes them in name order after the
				environment's confirmation.
			</p>
			<input id="migrate-files" class="mt-3 block text-[13px]" type="file" accept=".sql" multiple />
			{#if shell.env === 'prod' || shell.env === 'staging'}
				<input
					class="mt-2 h-8 rounded-md border border-tl-line bg-tl-bg px-2 font-mono text-[13px]"
					placeholder={shell.cluster.name}
					bind:value={migrateConfirm}
				/>
			{/if}
			<div class="mt-3 flex gap-2">
				<button
					class="inline-flex h-8 items-center rounded-md border border-tl-line px-3 text-[13px]"
					onclick={() => upload(false)}>Preview</button
				>
				<button
					class="inline-flex h-8 items-center rounded-md border border-tl-line px-3 text-[13px]"
					onclick={() => upload(true)}>Apply</button
				>
			</div>
			{#if migrateNote}
				<pre
					class="mt-3 overflow-auto font-mono text-[12px] whitespace-pre-wrap">{migrateNote}</pre>
			{/if}
		</section>
	</div>
{:else}
	<div class="mb-5">
		<h1 class="text-xl font-semibold tracking-tight">Migrations</h1>
		<p class="mt-1 max-w-2xl text-[13px] text-tl-fg2">
			Sample review for orders. Approving here does not run SQL. Apply a directory with
			<code class="font-mono">tl migrate apply --run</code>.
		</p>
	</div>

	<div class="grid items-start gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
		<ul class="space-y-2">
			{#each MIGRATIONS as item (item.id)}
				<li>
					<button
						class="w-full rounded-xl border px-3 py-2 text-left {item.id === selected.id
							? 'border-tl-accent bg-tl-elev'
							: 'border-tl-line bg-tl-elev hover:bg-tl-hover'}"
						onclick={() => (selectedId = item.id)}
					>
						<div class="font-mono text-[12px] text-tl-fg3">#{item.id}</div>
						<div class="text-[13px] font-medium">{item.title}</div>
						<div class="mt-0.5 text-[12px] text-tl-fg3">{statusOf(item)} · {item.age}</div>
					</button>
				</li>
			{/each}
		</ul>

		<section class="min-w-0 space-y-3">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<h2 class="text-[15px] font-semibold">#{selected.id} {selected.title}</h2>
				<div class="flex gap-2">
					{#if is142 && !rewritten}
						<button class={btn} onclick={() => (rewritten = true)}>Apply suggested rewrite</button>
					{/if}
					{#if is142}
						<button
							class="{btn} border-transparent bg-brand text-white"
							disabled={!canApprove}
							onclick={() => (approved = true)}
						>
							Approve
						</button>
					{/if}
				</div>
			</div>

			{#if is142 && approved}
				<p class="text-[13px] font-medium">Approved #0142</p>
			{/if}

			{#if selected.envs.some((env) => env.state === 'drift')}
				<p class="rounded-xl border border-tl-line bg-tl-subtle px-3 py-2 text-[13px] text-tl-fg2">
					Drift detected on staging. Type <span class="font-mono">billing.refund_reason</span> has
					an extra value <span class="font-mono">fraud</span> that is not in this migration. Compare
					the schemas with
					<code class="font-mono">tl migrate diff</code>.
				</p>
			{/if}

			<pre
				class="overflow-x-auto rounded-xl border border-tl-line bg-tl-elev p-3 font-mono text-[12px] leading-5">{is142 &&
				rewritten
					? safeSql
					: selected.sql}</pre>

			{#if is142}
				<ul class="space-y-2">
					{#each rewritten ? safeRisks : risks as risk (risk.stmt)}
						<li class="rounded-xl border border-tl-line px-3 py-2">
							<div class="font-mono text-[12px]">{risk.stmt}</div>
							<p class="text-[13px] text-tl-fg2">{risk.note}</p>
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	</div>
{/if}
