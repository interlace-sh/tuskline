<script lang="ts">
	import Shield from '@lucide/svelte/icons/shield';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import { clusterAdvise, splitRef, type AdviceItem } from '#lib/api.js';
	import { DEMO_ADVICE, DEMO_NOTES } from '#lib/advisor-demo.js';
	import { shell } from '#lib/shell.svelte.js';

	let memory = $state('');
	let cpus = $state('');
	let storage = $state('');
	let workload = $state('');
	let pooler = $state(false);
	let query = $state({ memory: '', cpus: '', storage: '', workload: '', pooler: false });
	let live = $state<AdviceItem[] | null>(null);
	let notes = $state<string[]>([]);
	let loading = $state(false);
	let error = $state('');
	let picked = $state<string[] | null>(null);
	let modal = $state(false);
	let ticket = $state('');
	let typed = $state('');
	let result = $state('');

	const advice = $derived(shell.demo ? DEMO_ADVICE : (live ?? []));
	const shownNotes = $derived(shell.demo ? DEMO_NOTES : notes);
	const selected = $derived(
		advice.filter((item) => (picked ?? advice.map((row) => row.name)).includes(item.name))
	);
	const prod = $derived(shell.cluster.env === 'prod');
	const canConfirm = $derived(!prod || (typed === shell.cluster.name && ticket.trim().length > 2));

	$effect(() => {
		shell.cluster.id;
		picked = null;
		result = '';
		modal = false;
	});

	function reanalyse(event: SubmitEvent) {
		event.preventDefault();
		picked = null;
		query = { memory: memory.trim(), cpus: cpus.trim(), storage, workload, pooler };
	}

	$effect(() => {
		const id = shell.cluster.id;
		const requested = query;
		if (shell.demo || !id.includes('/')) return;
		let cancelled = false;
		loading = true;
		error = '';
		const ref = splitRef(id);
		clusterAdvise(ref.env, ref.name, {
			memory: requested.memory || undefined,
			cpus: Number(requested.cpus) || undefined,
			storage: requested.storage || undefined,
			workload: requested.workload || undefined,
			pooler: requested.pooler
		})
			.then((body) => {
				if (cancelled) return;
				live = body.advice ?? [];
				notes = body.notes ?? [];
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				live = null;
				error = err instanceof Error ? err.message : 'Could not read settings.';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});

	function toggle(name: string) {
		const current = picked ?? advice.map((item) => item.name);
		picked = current.includes(name) ? current.filter((item) => item !== name) : [...current, name];
	}

	function confirmApply() {
		modal = false;
		ticket = '';
		typed = '';
		result = 'Nothing was applied. The preview is the change; run it yourself.';
	}

	const field =
		'h-8 w-full rounded-md border border-tl-line bg-tl-subtle px-2 text-[13px] text-tl-fg outline-none';
	const btn =
		'inline-flex h-8 items-center gap-1.5 rounded-md border border-tl-line bg-tl-elev px-2.5 text-[12.5px] font-medium hover:bg-tl-hover disabled:opacity-50';
</script>

<div class="flex flex-col gap-4">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="flex items-center gap-2 text-xl font-semibold tracking-tight">
				<SlidersHorizontal size={18} class="text-brand" /> Config advisor
			</h1>
			<p class="mt-1 font-mono text-[12.5px] text-tl-fg2">{shell.cluster.name}</p>
		</div>
		{#if !shell.demo}
			<form class="flex flex-wrap items-end gap-2" onsubmit={reanalyse}>
				<label class="text-[12px] text-tl-fg2"
					>Memory
					<input class="{field} w-28 font-mono" bind:value={memory} placeholder="16GB" />
				</label>
				<label class="text-[12px] text-tl-fg2"
					>CPUs
					<input class="{field} w-16 font-mono" bind:value={cpus} inputmode="numeric" />
				</label>
				<label class="text-[12px] text-tl-fg2"
					>Storage
					<select class="{field} w-28" bind:value={storage}>
						<option value="">Unknown</option>
						<option value="ssd">ssd</option>
						<option value="nvme">nvme</option>
						<option value="hdd">hdd</option>
						<option value="san">san</option>
					</select>
				</label>
				<label class="text-[12px] text-tl-fg2"
					>Workload
					<select class="{field} w-32" bind:value={workload}>
						<option value="">Default</option>
						<option value="web">web</option>
						<option value="oltp">oltp</option>
						<option value="dw">warehouse</option>
						<option value="desktop">desktop</option>
						<option value="mixed">mixed</option>
					</select>
				</label>
				<label class="flex items-center gap-1.5 pb-1.5 text-[12.5px]">
					<input type="checkbox" bind:checked={pooler} /> Pooler
				</label>
				<button class={btn} type="submit">Re-analyse</button>
			</form>
		{/if}
	</div>

	{#if loading}
		<p class="text-sm text-tl-fg2">Reading settings…</p>
	{:else if error}
		<p class="text-sm text-env-prod">{error}</p>
	{:else}
		<div class="grid items-start gap-4 lg:grid-cols-2">
			<section class="rounded-xl border border-tl-line bg-tl-elev">
				<div class="border-b border-tl-line px-4 py-3 text-[13px] font-medium">
					Findings
					<span class="font-normal text-tl-fg3">· {advice.length}</span>
				</div>
				{#if advice.length === 0}
					<p class="p-6 text-sm text-tl-fg2">
						No changes. The settings that were checked are already in range.
					</p>
				{/if}
				{#each advice as item (item.name)}
					<div class="border-b border-tl-line/70 px-4 py-2.5 last:border-0">
						<div class="flex items-center gap-2.5">
							<input
								type="checkbox"
								checked={selected.some((row) => row.name === item.name)}
								onchange={() => toggle(item.name)}
								aria-label="Include {item.name}"
							/>
							<span class="font-mono text-[12.5px] font-medium">{item.name}</span>
							<span class="font-mono text-[12px] text-tl-fg3 line-through">{item.current}</span>
							<span class="text-tl-fg3">→</span>
							<span class="font-mono text-[12.5px]">{item.proposed}</span>
							<span
								class="ml-auto rounded px-1.5 py-0.5 text-[11px] {item.apply === 'restart'
									? 'bg-env-staging/15 text-env-staging'
									: 'bg-tl-subtle text-tl-fg2'}"
							>
								{item.apply}
							</span>
						</div>
						<p class="mt-1 pl-6 text-[12.5px] leading-relaxed text-tl-fg2">{item.summary}</p>
						{#if item.evidence}
							<p class="mt-1 pl-6 text-[11.5px] text-tl-fg3">{item.evidence}</p>
						{/if}
					</div>
				{/each}
			</section>
			<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
				<h2 class="text-[13px] font-medium">
					Preview
					<span class="font-normal text-tl-fg3"
						>· {selected.length} selected · nothing is applied</span
					>
				</h2>
				<pre
					class="mt-3 max-h-80 overflow-auto rounded-md bg-tl-subtle p-3 font-mono text-[12px] leading-relaxed">{selected
						.map((item) => item.diff)
						.join('\n') || '# Nothing selected'}</pre>
				{#if shownNotes.length}
					<ul class="mt-3 space-y-1 text-[12.5px] text-tl-fg2">
						{#each shownNotes as note, index (index)}
							<li>{note}</li>
						{/each}
					</ul>
				{/if}
				<div class="mt-3 flex items-center gap-2">
					<button
						class="{btn} ml-auto {prod
							? 'border-env-prod/40 text-env-prod'
							: 'border-transparent bg-brand text-white'}"
						disabled={selected.length === 0}
						onclick={() => (modal = true)}
					>
						{#if prod}<Shield size={13} />{/if}
						Apply {selected.length} to {shell.cluster.env}…
					</button>
				</div>
				{#if result}
					<p class="mt-3 text-[12.5px] text-tl-fg2">{result}</p>
				{/if}
			</section>
		</div>
	{/if}
</div>

{#if modal}
	<div class="fixed inset-0 z-50 grid place-items-center bg-black/50 p-6">
		<div
			class="w-full max-w-lg rounded-xl border border-tl-line bg-tl-elev p-4"
			role="dialog"
			aria-labelledby="advise-apply-title"
		>
			<h2 id="advise-apply-title" class="flex items-center gap-2 text-[15px] font-semibold">
				{#if prod}<Shield size={15} class="text-env-prod" />{/if}
				Apply {selected.length} setting changes to {shell.cluster.name}
			</h2>
			<p class="mt-2 text-[12.5px] text-tl-fg2">
				This does not run the change. Production still needs a ticket and the cluster name.
			</p>
			{#if prod}
				<label class="mt-3 block text-[12.5px] text-tl-fg2">
					Change ticket
					<input class="{field} mt-1" placeholder="CHG-2041 (demo)" bind:value={ticket} />
				</label>
				<label class="mt-3 block text-[12.5px] text-tl-fg2">
					Type <span class="font-mono text-tl-fg">{shell.cluster.name}</span> to confirm
					<input class="{field} mono mt-1 font-mono" bind:value={typed} />
				</label>
			{/if}
			<div class="mt-4 flex justify-end gap-2">
				<button class={btn} onclick={() => (modal = false)}>Cancel</button>
				<button
					class="{btn} {prod
						? 'border-env-prod/40 text-env-prod'
						: 'border-transparent bg-brand text-white'}"
					disabled={!canConfirm}
					onclick={confirmApply}
				>
					{prod ? 'Apply to production' : 'Apply'}
				</button>
			</div>
		</div>
	</div>
{/if}
