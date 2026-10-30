<script lang="ts">
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { ApiError, clusterProbe, splitRef } from '#lib/api.js';
	import { DEMO_PROBE, partitionExtensions, type ServerProbe } from '#lib/extensions.js';
	import { shell } from '#lib/shell.svelte.js';

	let open = $state(false);
	let showAvailable = $state(false);
	let probe = $state<ServerProbe | null>(null);
	let loading = $state(false);
	let error = $state('');

	const live = $derived(!shell.demo && shell.cluster.id.includes('/'));
	const shown = $derived(live ? probe : DEMO_PROBE);
	const parts = $derived(partitionExtensions(shown?.extensions ?? []));
	const preload = $derived(
		shown?.preload_known ? (shown.shared_preload_libraries ?? []).join(', ') : 'not readable'
	);

	$effect(() => {
		const id = shell.cluster.id;
		if (shell.demo || !id.includes('/')) {
			probe = null;
			error = '';
			loading = false;
			return;
		}
		let cancelled = false;
		loading = true;
		error = '';
		const ref = splitRef(id);
		clusterProbe(ref.env, ref.name)
			.then((next) => {
				if (!cancelled) probe = next;
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				probe = null;
				error = err instanceof ApiError ? err.message : 'Could not read extensions.';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});
</script>

<div class="border-b border-tl-line">
	<button
		class="flex h-7 w-full items-center gap-1 px-2 text-left text-[12px] text-tl-fg2 hover:bg-tl-hover"
		aria-expanded={open}
		onclick={() => (open = !open)}
	>
		{#if open}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
		<span class="font-medium">Extensions</span>
		<span class="ml-auto text-[11px] text-tl-fg3 tabular-nums">
			{#if loading}
				reading…
			{:else if shown}
				{parts.installed.length} installed
			{/if}
		</span>
	</button>
	{#if open}
		<div class="max-h-40 overflow-auto px-2 pb-2 text-[12px]">
			{#if !live}
				<p class="px-1 pb-1 text-[11px] text-tl-fg3">
					Demo list. A saved cluster reads the server.
				</p>
			{/if}
			{#if error}
				<p class="px-1 text-env-prod">{error}</p>
			{:else if shown}
				<p class="truncate px-1 pb-1 font-mono text-[11px] text-tl-fg3" title={preload}>
					preload {preload || 'none'}
				</p>
				<ul>
					{#each parts.installed as extension (extension.name)}
						<li class="flex h-[22px] items-center gap-2 px-1">
							<span class="truncate font-mono">{extension.name}</span>
							<span class="ml-auto font-mono text-[10.5px] text-tl-fg3"
								>{extension.installed_version}</span
							>
						</li>
					{/each}
				</ul>
				<button
					class="mt-1 flex h-6 w-full items-center px-1 text-left text-[11px] text-tl-fg3 hover:text-tl-fg"
					aria-expanded={showAvailable}
					onclick={() => (showAvailable = !showAvailable)}
				>
					{showAvailable ? 'Hide' : 'Show'}
					{parts.available.length} available
				</button>
				{#if showAvailable}
					<ul>
						{#each parts.available as extension (extension.name)}
							<li class="flex h-[22px] items-center px-1 font-mono text-tl-fg3">
								{extension.name}
							</li>
						{/each}
					</ul>
				{/if}
			{/if}
		</div>
	{/if}
</div>
