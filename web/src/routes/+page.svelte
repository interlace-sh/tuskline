<script lang="ts">
	import ClusterHealth from '#lib/components/cluster-health.svelte';
	import ConnectScreen from '#lib/components/connect-screen.svelte';
	import Fleet from '#lib/components/fleet.svelte';
	import Queries from '#lib/components/queries.svelte';
	import SqlScreen from '#lib/components/sql-screen.svelte';
	import { SCREEN_LABEL, shell } from '#lib/shell.svelte.js';
</script>

{#if shell.screen === 'fleet'}
	<Fleet />
{:else if shell.screen === 'cluster'}
	<ClusterHealth />
{:else if shell.screen === 'queries'}
	<Queries />
{:else if shell.screen === 'sql'}
	<SqlScreen />
{:else if shell.screen === 'connect'}
	<ConnectScreen />
{:else}
	<h1 class="text-xl font-semibold tracking-tight">{SCREEN_LABEL[shell.screen]}</h1>
	<p class="mt-1 max-w-xl text-sm text-tl-fg2">
		This screen is part of a later milestone. The shell, environment and time range already apply.
	</p>
{/if}
