<script lang="ts">
	import Hash from '@lucide/svelte/icons/hash';
	import type { OverviewColumn } from '#lib/overview-stats.js';

	let {
		columns,
		loading = false,
		note = ''
	}: {
		columns: OverviewColumn[];
		loading?: boolean;
		note?: string;
	} = $props();

	let selected = $state('');
	let anchor = $state<{ top: number; left: number } | null>(null);
	const current = $derived(columns.find((column) => column.name === selected));

	function toggle(column: OverviewColumn, event: MouseEvent) {
		if (selected === column.name) {
			selected = '';
			anchor = null;
			return;
		}
		const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
		selected = column.name;
		anchor = { top: rect.top, left: rect.right + 8 };
	}

	function fmtPct(value: number | null): string {
		return value == null ? '—' : `${value}%`;
	}

	function fmtDistinct(value: number | null): string {
		return value == null ? '—' : String(value);
	}
</script>

<div class="px-3 pb-1 text-[11px] text-tl-fg3">
	{#if loading}Reading statistics…{:else if note}{note}{/if}
</div>
{#each columns as column (column.name)}
	<button
		class="flex h-7 w-full items-center gap-2 px-2 text-left hover:bg-tl-hover {current?.name ===
		column.name
			? 'bg-tl-hover'
			: ''}"
		aria-expanded={current?.name === column.name}
		onclick={(event) => toggle(column, event)}
	>
		<Hash size={12} class="shrink-0 text-tl-fg3" />
		<span class="min-w-0 flex-1 truncate font-mono text-[12px]">{column.name}</span>
		<span class="shrink-0 text-[10.5px] text-tl-fg3 tabular-nums"
			>{fmtDistinct(column.distinct)}</span
		>
		<span class="w-9 shrink-0 text-right text-[10.5px] text-tl-fg3 tabular-nums"
			>{fmtPct(column.nullPct)}</span
		>
	</button>
{/each}

{#if current && anchor}
	<div
		class="fixed z-40 w-64 rounded-lg border border-tl-line bg-tl-elev p-3"
		style:top="{anchor.top}px"
		style:left="{anchor.left}px"
		style:box-shadow="var(--tl-shadow)"
	>
		<div class="font-mono text-[12px]">{current.name}</div>
		<p class="mt-0.5 text-[10.5px] text-tl-fg3">
			{fmtDistinct(current.distinct)} distinct · {fmtPct(current.nullPct)} null
		</p>
		{#if current.bars.length}
			<p class="mt-2 text-[10.5px] text-tl-fg3">
				{current.depth ? 'Equal-depth histogram' : 'Most common values'}
			</p>
			<ul class="mt-1.5 space-y-1.5">
				{#each current.bars as bar (bar.label)}
					{@const peak = Math.max(...current.bars.map((item) => item.weight), 1)}
					<li>
						<div class="flex items-center justify-between gap-2 text-[11px]">
							<span class="truncate font-mono">{bar.label}</span>
							{#if !current.depth}
								<span class="shrink-0 text-tl-fg3 tabular-nums">{bar.weight}</span>
							{/if}
						</div>
						<div class="mt-0.5 h-1.5 overflow-hidden rounded-full bg-tl-subtle">
							<div
								class="h-full rounded-full bg-tl-accent"
								style:width="{Math.max(8, (bar.weight / peak) * 100)}%"
							></div>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
{/if}
