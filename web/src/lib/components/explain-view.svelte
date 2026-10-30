<script lang="ts">
	import { hierarchy, tree, type HierarchyPointNode } from 'd3-hierarchy';
	import { fmt, fmtMs } from '#lib/format.js';
	import { parseExplain, type ExplainNode } from '#lib/explain.js';

	let { text }: { text: string } = $props();

	const plan = $derived(parseExplain(text));
	let selectedId = $state('');
	const selected = $derived.by(() => {
		if (!plan) return undefined;
		const found = find(plan.root, selectedId);
		return found ?? plan.root;
	});

	const laid = $derived.by(() => {
		if (!plan) return undefined;
		const root = hierarchy(plan.root);
		const laidOut = tree<ExplainNode>().nodeSize([48, 228])(root);
		let minX = Infinity;
		let maxX = -Infinity;
		let maxY = 0;
		laidOut.each((node) => {
			minX = Math.min(minX, node.x);
			maxX = Math.max(maxX, node.x);
			maxY = Math.max(maxY, node.y);
		});
		return { root: laidOut, minX, width: maxY + 210, height: maxX - minX + 56 };
	});

	function find(node: ExplainNode, id: string): ExplainNode | undefined {
		if (node.id === id) return node;
		for (const child of node.children) {
			const found = find(child, id);
			if (found) return found;
		}
		return undefined;
	}

	function link(source: HierarchyPointNode<ExplainNode>, target: HierarchyPointNode<ExplainNode>) {
		const sy = source.y;
		const ty = target.y;
		const sx = source.x - (laid?.minX ?? 0) + 28;
		const tx = target.x - (laid?.minX ?? 0) + 28;
		const mid = (sy + ty) / 2;
		return `M${sy},${sx}C${mid},${sx} ${mid},${tx} ${ty},${tx}`;
	}

	function nodeY(node: HierarchyPointNode<ExplainNode>) {
		return node.x - (laid?.minX ?? 0) + 8;
	}
</script>

{#if plan && laid && selected}
	<div class="space-y-3 p-4">
		<p class="text-[13px] leading-relaxed text-tl-fg">{plan.summary}</p>
		{#if plan.warnings.length}
			<ul class="flex flex-wrap gap-1.5">
				{#each plan.warnings as warning (warning.text)}
					<li
						class="rounded-md px-2 py-1 text-[12px] {warning.tone === 'bad'
							? 'bg-env-prod/15 text-env-prod'
							: 'bg-tl-subtle text-tl-fg2'}"
					>
						{warning.text}
					</li>
				{/each}
			</ul>
		{/if}
		<div class="overflow-auto rounded-lg border border-tl-line">
			<svg
				width={laid.width}
				height={laid.height}
				role="img"
				aria-label="Query plan"
				class="text-tl-fg"
			>
				{#each laid.root.links() as edge (edge.target.data.id)}
					<path
						d={link(edge.source, edge.target)}
						fill="none"
						stroke="var(--tl-line)"
						stroke-width="1.5"
					/>
				{/each}
				{#each laid.root.descendants() as node (node.data.id)}
					{@const hot = /seq scan/i.test(node.data.op)}
					{@const caption = node.data.rel
						? `${node.data.totalMs > 0 ? fmtMs(node.data.selfMs) : `${fmt(node.data.estRows)} est.`} · ${node.data.rel}`
						: node.data.totalMs > 0
							? fmtMs(node.data.selfMs)
							: `${fmt(node.data.estRows)} est.`}
					<g
						transform="translate({node.y},{nodeY(node)})"
						class="cursor-pointer"
						role="button"
						tabindex="0"
						aria-pressed={selected.id === node.data.id}
						onclick={() => (selectedId = node.data.id)}
						onkeydown={(event) => {
							if (event.key === 'Enter') selectedId = node.data.id;
						}}
					>
						<rect
							width="196"
							height="36"
							rx="6"
							fill={selected.id === node.data.id ? 'var(--tl-hover)' : 'var(--tl-elev)'}
							stroke={hot ? 'var(--env-prod)' : 'var(--tl-line)'}
						/>
						<text x="10" y="15" fill={hot ? 'var(--env-prod)' : 'currentColor'} font-size="12">
							{node.data.op}
						</text>
						<text x="10" y="28" fill="var(--tl-fg3)" font-size="10">{caption}</text>
					</g>
				{/each}
			</svg>
		</div>
		<div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
			{#each [['Est. rows', fmt(selected.estRows)], ['Actual rows', fmt(selected.actRows)], ['Loops', fmt(selected.loops)], ['Buffers read / hit', `${fmt(selected.bufRead)} / ${fmt(selected.bufHit)}`]] as cell (cell[0])}
				<div class="rounded-md border border-tl-line bg-tl-subtle px-2.5 py-2">
					<div class="text-[11px] text-tl-fg3">{cell[0]}</div>
					<div class="mt-0.5 font-mono text-[12px] font-medium tabular-nums">{cell[1]}</div>
				</div>
			{/each}
		</div>
		{#if selected.detail}
			<pre
				class="rounded-md border border-tl-line bg-tl-subtle p-2.5 font-mono text-[11.5px] whitespace-pre-wrap text-tl-fg2">{selected.detail}</pre>
		{/if}
		<details class="text-[12px] text-tl-fg3">
			<summary class="cursor-pointer">Plan text</summary>
			<pre class="mt-2 font-mono text-[12px] leading-relaxed whitespace-pre-wrap">{text}</pre>
		</details>
	</div>
{:else}
	<pre
		class="p-4 font-mono text-[12px] leading-relaxed whitespace-pre-wrap text-tl-fg2">{text}</pre>
{/if}
