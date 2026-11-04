<script lang="ts">
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import Unplug from '@lucide/svelte/icons/unplug';
	import { ApiError, clusterSetup, splitRef, type SetupReport } from '#lib/api.js';
	import { DEMO_PROBE, partitionExtensions } from '#lib/extensions.js';
	import { shell } from '#lib/shell.svelte.js';

	let report = $state<SetupReport | null>(null);
	let error = $state('');
	let loading = $state(false);

	const sample = partitionExtensions(DEMO_PROBE.extensions);
	const btn =
		'inline-flex h-8 items-center gap-1.5 rounded-md border border-transparent bg-brand px-2.5 text-[12.5px] font-medium text-white';

	$effect(() => {
		const id = shell.cluster.id;
		if (shell.demo || !id.includes('/')) {
			report = null;
			return;
		}
		let cancelled = false;
		loading = true;
		error = '';
		const ref = splitRef(id);
		clusterSetup(ref.env, ref.name)
			.then((next) => {
				if (!cancelled) report = next;
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				report = null;
				error = err instanceof ApiError ? err.message : 'Could not read setup.';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});
</script>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-4">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Setup</h1>
		<p class="mt-1 max-w-xl text-sm text-tl-fg2">
			See which extensions this server is missing, and the install preview. Nothing here is applied.
		</p>
	</div>
	<div class="grid gap-3 sm:grid-cols-2">
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
			<h2 class="flex items-center gap-2 text-[14px] font-semibold">
				<Unplug size={16} class="text-brand" /> Add connection
			</h2>
			<p class="mt-1 text-[12.5px] leading-relaxed text-tl-fg2">
				One form. Create probes the server and saves a typed password in the OS keychain.
			</p>
			<button class="{btn} mt-3" onclick={() => shell.startConnect()}>Add connection</button>
		</section>
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
			<h2 class="flex items-center gap-2 text-[14px] font-semibold">
				<SlidersHorizontal size={16} class="text-brand" /> Config advisor
			</h2>
			<p class="mt-1 text-[12.5px] leading-relaxed text-tl-fg2">
				A preview of settings to change. Confirming does not run the SQL.
			</p>
			<button class="{btn} mt-3" onclick={() => shell.go('advisor')}>Open config advisor</button>
		</section>
	</div>

	{#if shell.demo}
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
			<h2 class="text-[13px] font-semibold">Extensions</h2>
			<p class="mt-1 text-[12.5px] text-tl-fg3">Sample for the orders demo. Not from a server.</p>
			<ul class="mt-3 space-y-1 text-[13px]">
				{#each sample.installed as extension (extension.name)}
					<li>{extension.name} {extension.installed_version}</li>
				{/each}
				{#each sample.available as extension (extension.name)}
					<li class="text-tl-fg2">{extension.name} is available and not installed</li>
				{/each}
			</ul>
		</section>
	{:else if loading}
		<p class="text-sm text-tl-fg2">Reading extensions…</p>
	{:else if error}
		<p class="text-sm text-env-prod">{error}</p>
	{:else if report}
		<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
			<h2 class="text-[13px] font-semibold">{report.version}</h2>
			<p class="mt-1 text-[12.5px] text-tl-fg3">{report.provider}</p>
			{#if report.missing.length === 0}
				<p class="mt-3 text-[13px]">The extensions Tuskline looks for are installed.</p>
			{:else}
				<p class="mt-3 text-[13px]">Missing: {report.missing.join(', ')}</p>
				<pre
					class="mt-3 overflow-auto rounded-md border border-tl-line bg-tl-subtle p-3 font-mono text-[11.5px] whitespace-pre-wrap">{report.plan.join(
						'\n'
					)}</pre>
				<p class="mt-2 text-[12px] text-tl-fg3">Preview only. Nothing has been installed.</p>
			{/if}
		</section>
	{/if}
</div>
