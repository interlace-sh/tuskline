<script lang="ts">
	let {
		data,
		color = 'var(--tl-accent)',
		w = 80,
		h = 30
	}: {
		data: number[];
		color?: string;
		w?: number;
		h?: number;
	} = $props();

	const gid = `spark-${Math.random().toString(36).slice(2)}`;

	const d = $derived.by(() => {
		if (data.length < 2) return '';
		const min = Math.min(...data);
		const max = Math.max(...data);
		const span = max - min || 1;
		return data
			.map((value, i) => {
				const x = (i / (data.length - 1)) * w;
				const y = h - 2 - ((value - min) / span) * (h - 4);
				return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
			})
			.join('');
	});
</script>

<svg width={w} height={h} class="shrink-0 overflow-visible" aria-hidden="true">
	<defs>
		<linearGradient id={gid} x1="0" x2="0" y1="0" y2="1">
			<stop offset="0" stop-color={color} stop-opacity="0.28" />
			<stop offset="1" stop-color={color} stop-opacity="0" />
		</linearGradient>
	</defs>
	{#if d}
		<path d="{d}L{w},{h}L0,{h}Z" fill="url(#{gid})" />
		<path
			{d}
			fill="none"
			stroke={color}
			stroke-width="1.5"
			stroke-linejoin="round"
			stroke-linecap="round"
		/>
	{/if}
</svg>
