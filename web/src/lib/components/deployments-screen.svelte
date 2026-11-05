<script lang="ts">
	import { listDeployments, type DeploymentRow } from '#lib/api.js';
	import { recordedDeployments } from '#lib/deploy-log.svelte.js';
	import { shell } from '#lib/shell.svelte.js';

	let remote = $state<DeploymentRow[]>([]);

	$effect(() => {
		if (shell.demo) return;
		let cancelled = false;
		listDeployments()
			.then((body) => {
				if (!cancelled) remote = body.deployments ?? [];
			})
			.catch(() => {
				if (!cancelled) remote = [];
			});
		return () => {
			cancelled = true;
		};
	});

	const rows = $derived(shell.demo ? recordedDeployments() : remote);
</script>

<div class="mb-5">
	<h1 class="text-xl font-semibold tracking-tight">Deployments</h1>
	<p class="mt-1 text-[13px] text-tl-fg2">
		{shell.demo
			? 'Clusters recorded from Deploy in this session.'
			: 'Clusters handed over with a Patroni URL or a backup repository.'}
	</p>
</div>

<table class="w-full text-left text-[13px]">
	<thead>
		<tr class="border-b border-tl-line text-tl-fg3">
			<th class="px-3 py-2 font-medium">Name</th>
			<th class="px-3 py-2 font-medium">Target</th>
			<th class="px-3 py-2 font-medium">Environment</th>
			<th class="px-3 py-2 font-medium">When</th>
		</tr>
	</thead>
	<tbody>
		{#each rows as row (row.name)}
			<tr class="border-b border-tl-line">
				<td class="px-3 py-2 font-mono">{row.name}</td>
				<td class="px-3 py-2">{row.target}</td>
				<td class="px-3 py-2">{row.env}</td>
				<td class="px-3 py-2">{row.when}</td>
			</tr>
		{:else}
			<tr>
				<td class="px-3 py-4 text-tl-fg3" colspan="4">No deployments in this session.</td>
			</tr>
		{/each}
	</tbody>
</table>
