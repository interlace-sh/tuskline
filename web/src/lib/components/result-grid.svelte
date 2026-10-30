<script lang="ts">
	import {
		Virtualizer,
		elementScroll,
		observeElementOffset,
		observeElementRect
	} from '@tanstack/svelte-virtual';

	let {
		columns,
		rows
	}: {
		columns: { name: string; type: string }[];
		rows: unknown[][];
	} = $props();

	let scrollEl = $state<HTMLDivElement | undefined>();
	let revision = $state(0);
	const rowHeight = 28;
	const template = $derived(`2.5rem repeat(${columns.length}, minmax(7.5rem, 1fr))`);

	const virtualizer = new Virtualizer<HTMLDivElement, Element>({
		count: 0,
		getScrollElement: () => scrollEl ?? null,
		estimateSize: () => rowHeight,
		overscan: 12,
		observeElementRect,
		observeElementOffset,
		scrollToFn: elementScroll,
		onChange: () => {
			revision += 1;
		}
	});

	$effect(() => {
		const count = rows.length;
		const element = scrollEl;
		virtualizer.setOptions({
			count,
			getScrollElement: () => element ?? null,
			estimateSize: () => rowHeight,
			overscan: 12,
			observeElementRect,
			observeElementOffset,
			scrollToFn: elementScroll,
			onChange: () => {
				revision += 1;
			}
		});
		virtualizer._willUpdate();
		return virtualizer._didMount();
	});

	const items = $derived.by(() => {
		void revision;
		return virtualizer.getVirtualItems();
	});
	const total = $derived.by(() => {
		void revision;
		return virtualizer.getTotalSize();
	});
</script>

<div class="flex min-h-0 flex-1 flex-col overflow-x-auto">
	<div class="flex min-h-0 min-w-full flex-1 flex-col" style:width="max-content">
		<div
			class="sticky top-0 z-10 grid shrink-0 border-b border-tl-line bg-tl-elev font-mono text-[12px]"
			style:grid-template-columns={template}
		>
			<div class="border-r border-tl-line px-2 py-1.5 text-right font-normal text-tl-fg3">#</div>
			{#each columns as column (column.name)}
				<div class="border-r border-tl-line px-3 py-1.5 text-left font-medium whitespace-nowrap">
					<span class="text-tl-fg">{column.name}</span>
					{#if column.type}<span class="text-[10.5px] font-normal text-tl-fg3">
							{column.type}</span
						>{/if}
				</div>
			{/each}
		</div>
		<div bind:this={scrollEl} class="min-h-0 flex-1 overflow-y-auto font-mono text-[12px]">
			<div class="relative" style:height="{total}px">
				{#each items as item (item.index)}
					{@const row = rows[item.index]}
					<div
						class="absolute top-0 grid w-full border-b border-tl-line hover:bg-tl-hover/60"
						style:height="{rowHeight}px"
						style:transform="translateY({item.start}px)"
						style:grid-template-columns={template}
					>
						<div class="border-r border-tl-line px-2 py-1 text-right text-tl-fg3">
							{item.index + 1}
						</div>
						{#each row ?? [] as value, cellIndex (cellIndex)}
							<div class="truncate border-r border-tl-line px-3 py-1 whitespace-nowrap">
								{#if value == null}
									<span class="text-tl-fg3 italic">null</span>
								{:else}
									{value}
								{/if}
							</div>
						{/each}
					</div>
				{/each}
			</div>
		</div>
	</div>
</div>
