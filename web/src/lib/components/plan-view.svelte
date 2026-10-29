<script lang="ts">
	import type { PlanNode } from '#lib/fixtures/demo.js';
	import { fmt, fmtMs } from '#lib/format.js';
	import { flattenPlan } from '#lib/query-view.js';

	let {
		root,
		flame = false
	}: {
		root: PlanNode;
		flame?: boolean;
	} = $props();

	const rows = $derived(flattenPlan(root));
	let selectedId = $state('');
	const selected = $derived(
		rows.find((row) => row.node.id === selectedId)?.node ?? rows[1]?.node ?? rows[0].node
	);

	const levels = $derived.by(() => {
		const out: { node: PlanNode; x: number; w: number }[][] = [];
		const walk = (node: PlanNode, x: number, w: number, depth: number) => {
			(out[depth] ??= []).push({ node, x, w });
			let cursor = x;
			for (const child of node.children ?? []) {
				const width = node.totalMs === 0 ? 0 : (child.totalMs / node.totalMs) * w;
				walk(child, cursor, width, depth + 1);
				cursor += width;
			}
		};
		walk(root, 0, 100, 0);
		return out;
	});

	function bar(node: PlanNode): string {
		if (node.op.includes('Seq')) return 'var(--env-prod)';
		if (node.op.includes('Index')) return 'var(--env-dev)';
		return 'var(--tl-accent)';
	}
</script>

{#if flame}
	<div class="space-y-1">
		{#each levels as level, depth (depth)}
			<div class="relative h-8">
				{#each level as item (item.node.id)}
					<div
						class="absolute top-0 flex h-8 items-center overflow-hidden rounded-[5px] px-2 text-[11.5px] font-medium whitespace-nowrap text-white"
						style:left="{item.x}%"
						style:width="calc({Math.max(item.w, 1.2)}% - 2px)"
						style:background={bar(item.node)}
						title="{item.node.op} {item.node.rel ?? ''} · {fmtMs(item.node.totalMs)}"
					>
						{#if item.w > 8}
							{item.node.op}
							<span class="ml-auto pl-2 tabular-nums opacity-80">{fmtMs(item.node.totalMs)}</span>
						{/if}
					</div>
				{/each}
			</div>
		{/each}
	</div>
{:else}
	<div class="overflow-hidden rounded-lg border border-tl-line">
		{#each rows as row (row.node.id)}
			{@const pct = root.totalMs === 0 ? 0 : (row.node.selfMs / root.totalMs) * 100}
			<button
				class="flex w-full items-center gap-3 border-b border-tl-line px-3 py-2 text-left last:border-0 {selected.id ===
				row.node.id
					? 'bg-tl-hover'
					: 'hover:bg-tl-hover/50'}"
				onclick={() => (selectedId = row.node.id)}
			>
				<div class="flex min-w-0 flex-1 items-center gap-2" style:padding-left="{row.depth * 18}px">
					{#if row.depth > 0}<span class="font-mono text-tl-fg3">↳</span>{/if}
					<span
						class="font-medium whitespace-nowrap"
						style:color={row.node.op.includes('Seq') ? 'var(--env-prod)' : undefined}
						>{row.node.op}</span
					>
					{#if row.node.rel}<span class="truncate font-mono text-[11.5px] text-tl-fg2"
							>{row.node.rel}</span
						>{/if}
				</div>
				<div class="w-[120px] shrink-0">
					<div class="mb-0.5 flex justify-between text-[11px]">
						<span class="tabular-nums">{fmtMs(row.node.selfMs)}</span>
						<span class="text-tl-fg3 tabular-nums">{pct.toFixed(0)}%</span>
					</div>
					<div class="h-1.5 overflow-hidden rounded-full bg-tl-subtle">
						<div
							class="h-full rounded-full"
							style:width="{Math.max(pct, 2)}%"
							style:background={pct > 50 ? 'var(--env-prod)' : 'var(--tl-accent)'}
						></div>
					</div>
				</div>
			</button>
		{/each}
	</div>
	<div class="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
		{#each [['Est. rows', fmt(selected.estRows)], ['Actual rows', fmt(selected.actRows)], ['Loops', fmt(selected.loops)], ['Buffers read / hit', `${fmt(selected.bufRead)} / ${fmt(selected.bufHit)}`]] as cell (cell[0])}
			<div class="rounded-md border border-tl-line bg-tl-subtle px-2.5 py-2">
				<div class="text-[11px] text-tl-fg3">{cell[0]}</div>
				<div class="mt-0.5 font-mono text-[12px] font-medium tabular-nums">{cell[1]}</div>
			</div>
		{/each}
	</div>
	{#if selected.detail}
		<pre
			class="mt-2 rounded-md border border-tl-line bg-tl-subtle p-2.5 font-mono text-[11.5px] whitespace-pre-wrap text-tl-fg2">{selected.detail}</pre>
	{/if}
{/if}
