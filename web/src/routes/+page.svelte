<script lang="ts">
	import { serverHealth } from '#lib/api.js';
	import Fleet from '#lib/components/fleet.svelte';
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

{#if shell.screen === 'fleet'}
	<Fleet />
	<div class="mt-6 flex items-center gap-3">
		<Button onclick={check}>Check API health</Button>
		<Badge variant={health === 'ok' ? 'default' : 'secondary'}>API: {health}</Badge>
	</div>
{:else}
	<h1 class="text-xl font-semibold tracking-tight">{SCREEN_LABEL[shell.screen]}</h1>
	<p class="mt-1 max-w-xl text-sm text-tl-fg2">
		This screen is part of a later milestone. The shell, environment and time range already apply.
	</p>
{/if}
