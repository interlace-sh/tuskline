<script lang="ts">
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import Copy from '@lucide/svelte/icons/copy';
	import FlaskConical from '@lucide/svelte/icons/flask-conical';
	import GitCompareArrows from '@lucide/svelte/icons/git-compare-arrows';
	import Lightbulb from '@lucide/svelte/icons/lightbulb';
	import Search from '@lucide/svelte/icons/search';
	import TrendingUp from '@lucide/svelte/icons/trending-up';
	import Zap from '@lucide/svelte/icons/zap';
	import { clusterQueries, splitRef, type Statement } from '#lib/api.js';
	import { PLAN_AFTER, PLAN_BEFORE, QUERIES, type Query } from '#lib/fixtures/demo.js';
	import { fmt, fmtMs } from '#lib/format.js';
	import { filterQueries, flattenPlan, sortQueries, type QuerySort } from '#lib/query-view.js';
	import { shell } from '#lib/shell.svelte.js';
	import type { Component } from 'svelte';
	import PlanView from './plan-view.svelte';
	import Sparkline from './sparkline.svelte';
	import SqlText from './sql-text.svelte';

	let text = $state('');
	let sort = $state<QuerySort>('share');
	let selectedId = $state('a91f3c');
	let tab = $state<'plan' | 'diff' | 'advice'>('plan');
	let flame = $state(false);
	let tested = $state(false);
	let notice = $state('');

	let statements = $state<Statement[] | null>(null);
	let deltasReady = $state(false);
	let error = $state('');
	let loading = $state(false);

	const rows = $derived(sortQueries(filterQueries(QUERIES, text), sort));
	const selected = $derived(QUERIES.find((query) => query.id === selectedId) ?? rows[0]);
	const hot = $derived(selected?.id === 'a91f3c');

	const liveRows = $derived(
		(statements ?? []).filter((statement) =>
			statement.query.toLowerCase().includes(text.trim().toLowerCase())
		)
	);

	$effect(() => {
		const id = shell.cluster.id;
		if (shell.demo || !id.includes('/')) return;
		let cancelled = false;
		loading = true;
		error = '';
		const ref = splitRef(id);
		clusterQueries(ref.env, ref.name)
			.then((body) => {
				if (cancelled) return;
				statements = body.statements ?? [];
				deltasReady = body.deltas != null;
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				statements = null;
				error = err instanceof Error ? err.message : 'Could not read statements.';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});

	function pick(query: Query) {
		selectedId = query.id;
		tab = 'plan';
		tested = false;
		notice = '';
	}

	async function copy(sql: string) {
		try {
			await navigator.clipboard.writeText(sql);
			notice = 'Copied. Nothing has been run.';
		} catch {
			notice = 'Could not copy.';
		}
	}

	const adviceSql =
		'CREATE INDEX CONCURRENTLY idx_order_items_order_sku\n  ON public.order_items (order_id, sku);';

	const tabs: { id: 'plan' | 'diff' | 'advice'; label: string; icon: Component }[] = [
		{ id: 'plan', label: 'EXPLAIN plan', icon: Zap },
		{ id: 'diff', label: 'Plan diff', icon: GitCompareArrows },
		{ id: 'advice', label: 'Index advice', icon: Lightbulb }
	];

	const diffPanes = [
		{ label: 'Before', plan: PLAN_BEFORE, color: 'var(--env-prod)' },
		{ label: 'With the hypothetical index', plan: PLAN_AFTER, color: 'var(--env-dev)' }
	];
</script>

<div class="mb-4 flex flex-wrap items-end justify-between gap-3">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Query insights</h1>
		<p class="mt-1 text-[13px] text-tl-fg2">
			{#if shell.demo}
				pg_stat_statements · prototype statements, not this server
			{:else}
				pg_stat_statements from {shell.cluster.name}
				{#if !deltasReady}· deltas appear on the next sample{/if}
			{/if}
		</p>
	</div>
	<label class="relative">
		<span class="sr-only">Filter statements</span>
		<Search
			size={13}
			class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-tl-fg3"
		/>
		<input
			data-filter
			bind:value={text}
			placeholder="Filter statements…"
			class="h-7 w-[min(260px,70vw)] rounded-md border border-tl-line bg-tl-subtle pr-8 pl-8 text-[12.5px] outline-none focus:border-tl-accent"
		/>
		<kbd class="kbd pointer-events-none absolute top-1/2 right-2 -translate-y-1/2">/</kbd>
	</label>
</div>

{#if notice}<p class="mb-3 text-[12.5px] text-tl-fg2">{notice}</p>{/if}

{#if shell.demo && selected}
	<div class="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_540px]">
		<div class="overflow-hidden rounded-xl border border-tl-line bg-tl-elev">
			<table class="w-full text-[12px]">
				<thead>
					<tr class="border-b border-tl-line text-left text-tl-fg3">
						<th class="px-3 py-2 font-medium">Statement</th>
						{#each [['share', 'Load'], ['callsPerSec', 'Calls/s'], ['meanMs', 'Mean'], ['p95Ms', 'p95']] as column (column[0])}
							<th class="px-3 py-2 text-right font-medium">
								<button
									class="inline-flex items-center gap-1"
									onclick={() => (sort = column[0] as QuerySort)}
								>
									{column[1]}
									{#if sort === column[0]}<ArrowDown size={11} />{/if}
								</button>
							</th>
						{/each}
						<th class="px-3 py-2 font-medium">24h</th>
					</tr>
				</thead>
				<tbody>
					{#each rows as query (query.id)}
						<tr
							class="cursor-pointer border-b border-tl-line last:border-0 {selected.id === query.id
								? 'bg-tl-hover shadow-[inset_2px_0_0_var(--tl-accent)]'
								: 'hover:bg-tl-hover/50'}"
							onclick={() => pick(query)}
						>
							<td class="w-full max-w-0 px-3 py-2.5">
								<div class="truncate text-[11.5px]"><SqlText sql={query.sql} /></div>
								<div
									class="mt-1 flex items-center gap-1.5 overflow-hidden text-[11px] whitespace-nowrap text-tl-fg3"
								>
									<span class="font-mono">#{query.id}</span>
									<span class="truncate">{query.db} · {query.user}</span>
									{#if query.regressed}
										<span class="inline-flex items-center gap-1 text-env-prod"
											><TrendingUp size={11} /> regressed</span
										>
									{/if}
								</div>
							</td>
							<td class="px-3 py-2.5 text-right">
								<div class="font-medium tabular-nums">{query.share}%</div>
							</td>
							<td class="px-3 py-2.5 text-right whitespace-nowrap tabular-nums"
								>{fmt(query.callsPerSec, 1)}</td
							>
							<td class="px-3 py-2.5 text-right whitespace-nowrap tabular-nums"
								>{fmtMs(query.meanMs)}</td
							>
							<td class="px-3 py-2.5 text-right whitespace-nowrap text-tl-fg2 tabular-nums"
								>{fmtMs(query.p95Ms)}</td
							>
							<td class="px-3 py-2.5 pr-4">
								<Sparkline
									data={query.trend}
									w={56}
									h={20}
									color={query.regressed ? 'var(--env-prod)' : 'var(--tl-accent)'}
								/>
							</td>
						</tr>
					{:else}
						<tr
							><td class="px-3 py-10 text-center text-tl-fg3" colspan="6"
								>No statements match “{text}”.</td
							></tr
						>
					{/each}
				</tbody>
			</table>
		</div>

		<section class="overflow-hidden rounded-xl border border-tl-line bg-tl-elev xl:sticky xl:top-0">
			<div class="border-b border-tl-line p-4">
				<div class="mb-2 flex flex-wrap items-center gap-2">
					<span class="font-mono text-[12px] text-tl-fg3">#{selected.id}</span>
					<span class="rounded-[5px] bg-tl-subtle px-1.5 py-0.5 text-[11px]">{selected.db}</span>
					<span class="rounded-[5px] bg-tl-subtle px-1.5 py-0.5 text-[11px]">{selected.user}</span>
				</div>
				<div
					class="max-h-[88px] overflow-auto rounded-md border border-tl-line bg-tl-subtle p-3 text-[12px] leading-relaxed"
				>
					<SqlText sql={selected.sql} />
				</div>
				<div class="mt-3 grid grid-cols-4 gap-3">
					{#each [['Share of load', `${selected.share}%`], ['Calls/s', fmt(selected.callsPerSec, 1)], ['Mean', fmtMs(selected.meanMs)], ['p95', fmtMs(selected.p95Ms)]] as cell (cell[0])}
						<div>
							<div class="text-[11px] text-tl-fg3">{cell[0]}</div>
							<div class="text-[15px] font-semibold tabular-nums">{cell[1]}</div>
						</div>
					{/each}
				</div>
			</div>
			<div class="flex items-center gap-1 border-b border-tl-line px-3 pt-2">
				{#each tabs as item (item.id)}
					{@const Icon = item.icon}
					<button
						class="flex h-8 items-center gap-1.5 border-b-2 px-2.5 text-[12.5px] {tab === item.id
							? 'border-tl-accent font-medium text-tl-fg'
							: 'border-transparent text-tl-fg3 hover:text-tl-fg'}"
						onclick={() => (tab = item.id)}
					>
						<Icon size={13} />
						{item.label}
					</button>
				{/each}
				{#if tab === 'plan' && hot}
					<div class="mb-1 ml-auto inline-flex rounded-md border border-tl-line bg-tl-subtle p-0.5">
						<button
							class="h-[22px] rounded-[5px] px-2 text-[11.5px] {flame
								? 'text-tl-fg3'
								: 'bg-tl-elev ring-1 ring-tl-line2'}"
							onclick={() => (flame = false)}>Tree</button
						>
						<button
							class="h-[22px] rounded-[5px] px-2 text-[11.5px] {flame
								? 'bg-tl-elev ring-1 ring-tl-line2'
								: 'text-tl-fg3'}"
							onclick={() => (flame = true)}>Flame</button
						>
					</div>
				{/if}
			</div>
			<div class="p-4">
				{#if tab === 'plan'}
					{#if hot}
						<PlanView root={PLAN_BEFORE} {flame} />
					{:else}
						<p class="py-6 text-center text-[12.5px] text-tl-fg2">
							This fixture statement has no captured plan.
						</p>
					{/if}
				{:else if tab === 'diff'}
					<div class="grid gap-3 sm:grid-cols-2">
						{#each diffPanes as pane (pane.label)}
							<div
								class="rounded-lg border border-tl-line p-3"
								style:border-top="2px solid {pane.color}"
							>
								<div class="mb-2 flex items-baseline justify-between">
									<span class="text-[12px] text-tl-fg2">{pane.label}</span>
									<span class="font-semibold tabular-nums" style:color={pane.color}
										>{fmtMs(pane.plan.totalMs)}</span
									>
								</div>
								{#each flattenPlan(pane.plan) as row (row.node.id)}
									<div
										class="flex gap-1.5 py-1 text-[11.5px]"
										style:padding-left="{row.depth * 12}px"
									>
										<span
											class="font-medium"
											style:color={row.node.op.includes('Seq') ? 'var(--env-prod)' : undefined}
											>{row.node.op}</span
										>
										<span class="truncate font-mono text-tl-fg3">{row.node.rel}</span>
									</div>
								{/each}
							</div>
						{/each}
					</div>
					<p class="mt-3 text-[12.5px] text-tl-fg3">Fixture comparison. hypopg has not been run.</p>
				{:else if hot}
					<div class="space-y-3">
						<div class="flex items-start gap-2.5">
							<Lightbulb size={15} class="mt-0.5 text-tl-accent" />
							<div>
								<div class="font-medium">Composite index on order_items (order_id, sku)</div>
								<p class="mt-0.5 text-[12.5px] text-tl-fg2">
									Fixture advice for this statement. It has not been validated on a server.
								</p>
							</div>
						</div>
						<div class="relative rounded-md border border-tl-line bg-tl-subtle p-3">
							<SqlText class="text-[12px] whitespace-pre-wrap" sql={adviceSql} />
							<button
								class="absolute top-2 right-2 text-tl-fg3 hover:text-tl-fg"
								onclick={() => copy(adviceSql)}
								title="Copy"
							>
								<Copy size={12} />
							</button>
						</div>
						<button
							class="inline-flex h-7 items-center gap-1.5 rounded-md border border-tl-line px-2.5 text-[12.5px] font-medium hover:bg-tl-hover"
							onclick={() => (tested = true)}
						>
							<FlaskConical size={13} /> Show the fixture prediction
						</button>
						{#if tested}
							<p class="text-[12.5px] text-tl-fg2">
								Fixture only: planner cost 182,410 → 3,120. This click did not call hypopg.
							</p>
						{/if}
					</div>
				{:else}
					<p class="py-6 text-center text-[12.5px] text-tl-fg2">
						No index recommendation in the fixture for this statement.
					</p>
				{/if}
			</div>
		</section>
	</div>
{:else if loading}
	<p class="text-sm text-tl-fg2">Reading pg_stat_statements…</p>
{:else if error}
	<section class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4">
		<h2 class="text-[13px] font-semibold">Could not read statements</h2>
		<p class="mt-1 text-[13px] text-tl-fg2">{error}</p>
	</section>
{:else}
	<div class="overflow-hidden rounded-xl border border-tl-line bg-tl-elev">
		<table class="w-full text-left text-[12px]">
			<thead>
				<tr class="border-b border-tl-line text-tl-fg3">
					<th class="px-3 py-2 font-medium">Statement</th>
					<th class="px-3 py-2 text-right font-medium">Calls</th>
					<th class="px-3 py-2 text-right font-medium">Total</th>
					<th class="px-3 py-2 text-right font-medium">Mean</th>
					<th class="px-3 py-2 text-right font-medium">Rows</th>
				</tr>
			</thead>
			<tbody>
				{#each liveRows as statement (statement.query_id)}
					<tr class="border-b border-tl-line last:border-0">
						<td class="max-w-[480px] truncate px-3 py-2 font-mono text-[11.5px]"
							>{statement.query}</td
						>
						<td class="px-3 py-2 text-right tabular-nums">{fmt(statement.calls)}</td>
						<td class="px-3 py-2 text-right tabular-nums">{fmtMs(statement.total_exec_time_ms)}</td>
						<td class="px-3 py-2 text-right tabular-nums">
							{statement.calls > 0 ? fmtMs(statement.total_exec_time_ms / statement.calls) : '—'}
						</td>
						<td class="px-3 py-2 text-right tabular-nums">{fmt(statement.rows)}</td>
					</tr>
				{:else}
					<tr
						><td class="px-3 py-10 text-center text-tl-fg3" colspan="5">No statements match.</td
						></tr
					>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
