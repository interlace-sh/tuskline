<script lang="ts">
	import { MIGRATIONS, type Migration } from '#lib/fixtures/demo.js';

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
				Drift detected on staging. Type <span class="font-mono">billing.refund_reason</span> has an
				extra value <span class="font-mono">fraud</span> that is not in this migration. Compare the
				schemas with
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
