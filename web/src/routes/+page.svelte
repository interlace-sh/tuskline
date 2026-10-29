<script lang="ts">
	import { serverHealth } from '#lib/api.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { SCREEN_LABEL, shell } from '#lib/shell.svelte.js';

	let health = $state<'unknown' | 'ok' | 'down'>('unknown');

	async function check() {
		try {
			const body = await serverHealth();
			health = body.status === 'ok' ? 'ok' : 'down';
		} catch {
			health = 'down';
		}
	}
</script>

<h1 class="text-xl font-semibold tracking-tight">{SCREEN_LABEL[shell.screen]}</h1>
<p class="mt-1 max-w-xl text-sm text-tl-fg2">
	{#if shell.screen === 'fleet'}
		{#if shell.demo}
			Demo fleet from the prototype fixtures. An empty inventory stays here;
			<code class="font-mono">?demo=1</code> keeps the fixtures when clusters are connected.
		{:else}
			{shell.clusters.length === 1 ? '1 cluster' : `${shell.clusters.length} clusters`} from the inventory.
			The API behind this UI is the same one
			<code class="font-mono">tl</code> uses.
		{/if}
	{:else}
		This screen is part of a later milestone. The shell, environment and time range already apply.
	{/if}
</p>
{#if shell.screen === 'fleet'}
	<div class="mt-4 flex items-center gap-3">
		<Button onclick={check}>Check API health</Button>
		<Badge variant={health === 'ok' ? 'default' : 'secondary'}>API: {health}</Badge>
	</div>
{/if}
