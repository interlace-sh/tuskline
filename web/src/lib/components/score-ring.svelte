<script lang="ts">
	let {
		value,
		color,
		size = 40,
		stroke = 4
	}: {
		value: number;
		color: string;
		size?: number;
		stroke?: number;
	} = $props();

	const radius = $derived((size - stroke) / 2);
	const circumference = $derived(2 * Math.PI * radius);
</script>

<div class="relative shrink-0" style:width="{size}px" style:height="{size}px">
	<svg width={size} height={size} class="-rotate-90" aria-hidden="true">
		<circle
			cx={size / 2}
			cy={size / 2}
			r={radius}
			stroke="var(--tl-line)"
			stroke-width={stroke}
			fill="none"
		/>
		<circle
			cx={size / 2}
			cy={size / 2}
			r={radius}
			stroke={color}
			stroke-width={stroke}
			fill="none"
			stroke-dasharray={circumference}
			stroke-dashoffset={circumference * (1 - value / 100)}
			stroke-linecap="round"
		/>
	</svg>
	<span class="absolute inset-0 grid place-items-center text-[11.5px] font-semibold tabular-nums"
		>{value}</span
	>
</div>
