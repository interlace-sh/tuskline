<script lang="ts">
	import { ApiError, clusterHealth, splitRef, type Finding } from '#lib/api.js';
	import { FINDINGS } from '#lib/fixtures/demo.js';
	import { shell } from '#lib/shell.svelte.js';

	let rows = $state<{ cluster: string; finding: Finding }[]>([]);
	let error = $state('');
	let loading = $state(false);

	$effect(() => {
		if (shell.demo) return;
		const clusters = shell.clusters.filter((cluster) => cluster.id.includes('/'));
		let cancelled = false;
		loading = true;
		error = '';
		Promise.all(
			clusters.map(async (cluster) => {
				const ref = splitRef(cluster.id);
				const report = await clusterHealth(ref.env, ref.name);
				return report.findings.map((finding) => ({ cluster: cluster.name, finding }));
			})
		)
			.then((groups) => {
				if (!cancelled) rows = groups.flat();
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				rows = [];
				error = err instanceof ApiError ? err.message : 'Could not read alerts.';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});
</script>

<div class="mb-5">
	<h1 class="text-xl font-semibold tracking-tight">Alerts</h1>
	<p class="mt-1 text-[13px] text-tl-fg2">Findings from the clusters in this inventory.</p>
</div>

{#if shell.demo}
	<ul class="max-w-2xl space-y-2">
		{#each FINDINGS as item (item.id)}
			<li class="rounded-xl border border-tl-line bg-tl-elev p-4">
				<div class="text-[13px] font-medium">{item.title}</div>
				<p class="mt-1 text-[12.5px] text-tl-fg2">{item.detail}</p>
			</li>
		{/each}
	</ul>
{:else if loading}
	<p class="text-sm text-tl-fg2">Reading findings…</p>
{:else if error}
	<p class="text-sm text-env-prod">{error}</p>
{:else if rows.length === 0}
	<p class="text-sm text-tl-fg2">No findings. The checks that ran were clean.</p>
{:else}
	<ul class="max-w-2xl space-y-2">
		{#each rows as row (`${row.cluster}:${row.finding.check}:${row.finding.summary}`)}
			<li class="rounded-xl border border-tl-line bg-tl-elev p-4">
				<div class="text-[11px] text-tl-fg3">{row.cluster}</div>
				<div class="text-[13px] font-medium">{row.finding.summary}</div>
				<p class="mt-1 text-[12.5px] text-tl-fg2">{row.finding.evidence}</p>
			</li>
		{/each}
	</ul>
{/if}
