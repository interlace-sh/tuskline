<script lang="ts">
	import AdvisorScreen from '#lib/components/advisor-screen.svelte';
	import AlertsScreen from '#lib/components/alerts-screen.svelte';
	import BackupsScreen from '#lib/components/backups-screen.svelte';
	import ClusterHealth from '#lib/components/cluster-health.svelte';
	import ConnectScreen from '#lib/components/connect-screen.svelte';
	import DeployScreen from '#lib/components/deploy-screen.svelte';
	import DeploymentsScreen from '#lib/components/deployments-screen.svelte';
	import Fleet from '#lib/components/fleet.svelte';
	import HaScreen from '#lib/components/ha-screen.svelte';
	import MigrationsScreen from '#lib/components/migrations-screen.svelte';
	import Queries from '#lib/components/queries.svelte';
	import SetupScreen from '#lib/components/setup-screen.svelte';
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
{:else if shell.screen === 'alerts'}
	<AlertsScreen />
{:else if shell.screen === 'advisor'}
	<AdvisorScreen />
{:else if shell.screen === 'setup'}
	<SetupScreen />
{:else if shell.screen === 'deploy'}
	<DeployScreen />
{:else if shell.screen === 'deployments'}
	<DeploymentsScreen />
{:else if shell.screen === 'ha'}
	<HaScreen />
{:else if shell.screen === 'backups'}
	<BackupsScreen />
{:else if shell.screen === 'migrations'}
	<MigrationsScreen />
{:else}
	<h1 class="text-xl font-semibold tracking-tight">{SCREEN_LABEL[shell.screen]}</h1>
	<p class="mt-1 max-w-xl text-sm text-tl-fg2">
		This screen is part of a later milestone. The shell, environment and time range already apply.
	</p>
{/if}
