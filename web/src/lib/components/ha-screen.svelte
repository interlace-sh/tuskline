<script lang="ts">
	import { shell } from '#lib/shell.svelte.js';
	import HaLive from './ha-live.svelte';

	interface Node {
		id: string;
		role: 'leader' | 'replica';
	}

	let nodes = $state<Node[]>([
		{ id: 'pg-euw-1', role: 'leader' },
		{ id: 'pg-euw-2', role: 'replica' },
		{ id: 'pg-euw-3', role: 'replica' }
	]);
	let open = $state(false);
	let confirm = $state('');
	let notice = $state('');

	const leader = $derived(nodes.find((node) => node.role === 'leader') ?? nodes[0]);
	const candidate = $derived(nodes.find((node) => node.role !== 'leader') ?? nodes[1]);
	const needsName = $derived(shell.env === 'prod');
	const confirmed = $derived(!needsName || confirm === shell.cluster.name);

	const btn =
		'inline-flex h-8 items-center rounded-md border border-tl-line bg-tl-elev px-3 text-[13px] font-medium hover:bg-tl-hover';

	function switchover() {
		if (!confirmed) return;
		const from = leader.id;
		const to = candidate.id;
		nodes = nodes.map((node) => {
			if (node.id === to) return { ...node, role: 'leader' as const };
			if (node.id === from) return { ...node, role: 'replica' as const };
			return node;
		});
		notice = `${to} is now leader`;
		open = false;
		confirm = '';
	}
</script>

{#if !shell.demo && shell.cluster.id.includes('/')}
	<HaLive />
{:else}
	<div class="mb-5 flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="text-xl font-semibold tracking-tight">HA topology</h1>
			<p class="mt-1 text-[13px] text-tl-fg2">
				{shell.cluster.name} · {shell.cluster.ha} · timeline 7
			</p>
		</div>
		<button class="{btn} border-transparent bg-brand text-white" onclick={() => (open = true)}
			>Switchover…</button
		>
	</div>

	{#if notice}
		<p class="mb-3 text-[13px] text-tl-fg2">{notice}</p>
	{/if}

	<ul class="grid max-w-3xl gap-3 sm:grid-cols-3">
		{#each nodes as node (node.id)}
			<li class="rounded-xl border border-tl-line bg-tl-elev p-4">
				<div class="font-mono text-[13px] font-medium">{node.id}</div>
				<div class="mt-1 text-[12px] text-tl-fg3">{node.role}</div>
			</li>
		{/each}
	</ul>

	{#if open}
		<div class="fixed inset-0 z-40 grid place-items-center bg-black/40 p-4">
			<div
				class="w-full max-w-md rounded-xl border border-tl-line bg-tl-elev p-4"
				role="dialog"
				aria-labelledby="switchover-title"
			>
				<h2 id="switchover-title" class="text-[15px] font-semibold">Switchover</h2>
				<p class="mt-1 text-[13px] text-tl-fg2">
					{leader.id} to {candidate.id}. This demo does not call Patroni.
				</p>
				{#if needsName}
					<label class="mt-3 block text-[12px] text-tl-fg3">
						Type {shell.cluster.name}
						<input
							class="mono mt-1 w-full rounded-md border border-tl-line bg-tl-bg px-2 py-1"
							bind:value={confirm}
						/>
					</label>
				{/if}
				<div class="mt-4 flex gap-2">
					<button class={btn} onclick={() => (open = false)}>Cancel</button>
					<button
						class="{btn} border-transparent bg-brand text-white disabled:opacity-40"
						disabled={!confirmed}
						onclick={switchover}>Switch over now</button
					>
				</div>
			</div>
		</div>
	{/if}
{/if}
