<script lang="ts">
	import {
		ApiError,
		clusterTopology,
		requestSwitchover,
		splitRef,
		type Topology
	} from '#lib/api.js';
	import { shell } from '#lib/shell.svelte.js';

	let topo = $state<Topology | null>(null);
	let error = $state('');
	let open = $state(false);
	let confirm = $state('');
	let notice = $state('');
	let busy = $state(false);

	const leader = $derived(topo?.members.find((member) => member.role === 'leader'));
	const candidate = $derived(topo?.members.find((member) => member.role !== 'leader'));
	const needsName = $derived(shell.env === 'prod' || shell.env === 'staging');
	const confirmed = $derived(!needsName || confirm.trim() === shell.cluster.name);

	const btn =
		'inline-flex h-8 items-center rounded-md border border-tl-line bg-tl-elev px-3 text-[13px] font-medium hover:bg-tl-hover disabled:opacity-40';

	$effect(() => {
		const id = shell.cluster.id;
		if (!id.includes('/')) return;
		const ref = splitRef(id);
		let cancelled = false;
		topo = null;
		error = '';
		clusterTopology(ref.env, ref.name)
			.then((body) => {
				if (!cancelled) topo = body;
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				error = err instanceof ApiError ? err.message : 'Could not read the topology.';
			});
		return () => {
			cancelled = true;
		};
	});

	async function send() {
		if (!leader || !candidate || !confirmed) return;
		const ref = splitRef(shell.cluster.id);
		busy = true;
		notice = '';
		try {
			const result = await requestSwitchover(ref.env, ref.name, {
				from: leader.name,
				to: candidate.name,
				confirm: confirm.trim(),
				run: true
			});
			notice = result.sent
				? `Switchover sent. ${candidate.name} was the candidate.`
				: result.preview;
			open = false;
			confirm = '';
			topo = await clusterTopology(ref.env, ref.name);
		} catch (err) {
			notice = err instanceof ApiError ? err.message : 'Switchover was not sent.';
		} finally {
			busy = false;
		}
	}
</script>

<div class="mb-5 flex flex-wrap items-end justify-between gap-3">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">HA topology</h1>
		<p class="mt-1 text-[13px] text-tl-fg2">
			{shell.cluster.name}
			{#if topo}
				· {topo.source}{#if topo.timeline}
					· timeline {topo.timeline}{/if}
			{/if}
		</p>
	</div>
	<button
		class="{btn} border-transparent bg-brand text-white"
		disabled={!leader || !candidate}
		onclick={() => (open = true)}>Switchover…</button
	>
</div>

{#if error}
	<p class="max-w-xl text-[13px] text-tl-fg2">{error}</p>
{:else if !topo}
	<p class="text-[13px] text-tl-fg2">Reading the HA API…</p>
{:else}
	{#if notice}<p class="mb-3 text-[13px] text-tl-fg2">{notice}</p>{/if}
	<ul class="grid max-w-3xl gap-3 sm:grid-cols-3">
		{#each topo.members as member (member.name)}
			<li class="rounded-xl border border-tl-line bg-tl-elev p-4">
				<div class="font-mono text-[13px] font-medium">{member.name}</div>
				<div class="mt-1 text-[12px] text-tl-fg3">
					{member.role}{member.state ? ` · ${member.state}` : ''}
					{member.lag_bytes ? ` · lag ${member.lag_bytes} bytes` : ''}
				</div>
			</li>
		{/each}
	</ul>
{/if}

{#if open && leader && candidate}
	<div class="fixed inset-0 z-40 grid place-items-center bg-black/40 p-4">
		<div class="w-full max-w-md rounded-xl border border-tl-line bg-tl-elev p-4" role="dialog">
			<h2 class="text-[15px] font-semibold">Switchover</h2>
			<p class="mt-1 text-[13px] text-tl-fg2">
				POST the Patroni API: {leader.name} to {candidate.name}.
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
					class="{btn} border-transparent bg-brand text-white"
					disabled={!confirmed || busy}
					onclick={send}
				>
					Switch over now
				</button>
			</div>
		</div>
	</div>
{/if}
