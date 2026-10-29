<script lang="ts">
	import { linePath, stackedAreas } from '#lib/charts.js';

	let {
		rows,
		series,
		height = 160,
		stacked = false,
		domain
	}: {
		rows: Record<string, number | string>[];
		series: { key: string; color: string; name?: string }[];
		height?: number;
		stacked?: boolean;
		domain?: [number, number];
	} = $props();

	const width = 640;
	const areas = $derived(
		stacked
			? stackedAreas(
					rows,
					series.map((item) => item.key),
					width,
					height
				)
			: []
	);
	const lines = $derived(
		series.map((item) => ({
			...item,
			d: linePath(rows, item.key, width, height, domain)
		}))
	);

	function colorFor(key: string): string {
		return series.find((item) => item.key === key)?.color ?? 'var(--tl-fg3)';
	}
</script>

<svg viewBox="0 0 {width} {height}" class="w-full" style:height="{height}px" role="img">
	{#if stacked}
		{#each areas as area (area.key)}
			<path d={area.area} fill={colorFor(area.key)} fill-opacity="0.45" />
			<path d={area.line} fill="none" stroke={colorFor(area.key)} stroke-width="1.25" />
		{/each}
	{:else}
		{#each lines as line (line.key)}
			<path d={line.d} fill="none" stroke={line.color} stroke-width="1.75" />
		{/each}
	{/if}
</svg>
