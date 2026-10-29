<script lang="ts">
	import ClusterHealth from '#lib/components/cluster-health.svelte';
	import Fleet from '#lib/components/fleet.svelte';
	import { SCREEN_LABEL, shell } from '#lib/shell.svelte.js';
</script>

{#if shell.screen === 'fleet'}
	<Fleet />
{:else if shell.screen === 'cluster'}
	<ClusterHealth />
{:else}
	<h1 class="text-xl font-semibold tracking-tight">{SCREEN_LABEL[shell.screen]}</h1>
	<p class="mt-1 max-w-xl text-sm text-tl-fg2">
		This screen is part of a later milestone. The shell, environment and time range already apply.
	</p>
{/if}
