<script lang="ts">
	import AlertTriangle from '@lucide/svelte/icons/alert-triangle';
	import Archive from '@lucide/svelte/icons/archive';
	import ArrowUpRight from '@lucide/svelte/icons/arrow-up-right';
	import Filter from '@lucide/svelte/icons/filter';
	import Plus from '@lucide/svelte/icons/plus';
	import Rocket from '@lucide/svelte/icons/rocket';
	import Search from '@lucide/svelte/icons/search';
	import ServerOff from '@lucide/svelte/icons/server-off';
	import Shield from '@lucide/svelte/icons/shield';
	import { ENV_LABEL, envRailClass, type EnvKind } from '#lib/env.js';
	import { CLUSTERS } from '#lib/fixtures/demo.js';
	import { filterFleet, sortByScore, type FleetFilter } from '#lib/fixtures/fleet.js';
	import { healthColor, healthDotClass } from '#lib/fixtures/clusters.js';
	import { fmt, fmtBytes } from '#lib/format.js';
	import { shell } from '#lib/shell.svelte.js';
	import ScoreRing from './score-ring.svelte';
	import Sparkline from './sparkline.svelte';

	let query = $state('');
	let filter = $state<FleetFilter>('all');

	const demoList = $derived(sortByScore(filterFleet(CLUSTERS, query, filter)));
	const counts = $derived({
		ok: CLUSTERS.filter((cluster) => cluster.health === 'ok').length,
		warn: CLUSTERS.filter((cluster) => cluster.health === 'warn').length,
		crit: CLUSTERS.filter((cluster) => cluster.health === 'crit').length
	});
	const totalTps = $derived(CLUSTERS.reduce((sum, cluster) => sum + cluster.tps, 0));
	const totalStorage = $derived(CLUSTERS.reduce((sum, cluster) => sum + cluster.storageGb, 0));
	const backupsInSla = $derived(CLUSTERS.filter((cluster) => cluster.backupAgeH <= 24).length);
	const lanes: EnvKind[] = ['prod', 'staging', 'dev', 'local'];

	const liveList = $derived(
		shell.clusters.filter((cluster) => {
			const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
			const hay =
				`${cluster.name} ${cluster.region} ${cluster.ha} ${cluster.database}`.toLowerCase();
			return terms.every((term) => hay.includes(term));
		})
	);

	function openCluster(id: string) {
		shell.setCluster(id);
		shell.go('cluster');
	}
</script>

<div class="mb-5 flex flex-wrap items-end justify-between gap-3">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">Fleet</h1>
		<p class="mt-1 text-[13px] text-tl-fg2">
			{#if shell.demo}
				{CLUSTERS.length} clusters across 3 environments ·
				<span class="text-tl-fg3">prototype fixtures</span>
			{:else}
				{shell.clusters.length === 1 ? '1 cluster' : `${shell.clusters.length} clusters`} from the inventory
			{/if}
		</p>
	</div>
	<div class="flex flex-wrap gap-2">
		<button
			class="inline-flex h-7 items-center gap-1.5 rounded-md border border-tl-line bg-tl-elev px-2.5 text-[12.5px] font-medium hover:bg-tl-hover"
			title="Deploy a new cluster (g d)"
			onclick={() => shell.go('deploy')}
		>
			<Rocket size={13} /> Deploy cluster
		</button>
		<button
			class="inline-flex h-7 items-center gap-1.5 rounded-md bg-brand px-2.5 text-[12.5px] font-medium text-white"
			title="Add connection (n)"
			onclick={() => shell.go('connect')}
		>
			<Plus size={13} /> Add connection
		</button>
	</div>
</div>

{#if shell.demo}
	<div
		class="mb-5 grid grid-cols-2 divide-tl-line overflow-hidden rounded-xl border border-tl-line bg-tl-elev sm:grid-cols-3 xl:grid-cols-6 xl:divide-x"
	>
		{#each [{ label: 'Healthy', value: counts.ok, health: 'ok' as const }, { label: 'Degraded', value: counts.warn, health: 'warn' as const }, { label: 'Critical', value: counts.crit, health: 'crit' as const }] as stat (stat.label)}
			<div class="border-tl-line px-5 py-4 xl:border-0">
				<div
					class="mb-1.5 flex items-center gap-1.5 text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase"
				>
					<span class="size-1.5 rounded-full {healthDotClass(stat.health)}"></span>
					{stat.label}
				</div>
				<div class="text-[22px] leading-none font-semibold tabular-nums">{stat.value}</div>
			</div>
		{/each}
		<div class="px-5 py-4">
			<div class="mb-1.5 text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">
				Fleet TPS
			</div>
			<div class="text-[22px] leading-none font-semibold tabular-nums">{fmt(totalTps)}</div>
		</div>
		<div class="px-5 py-4">
			<div class="mb-1.5 text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">
				Storage
			</div>
			<div class="text-[22px] leading-none font-semibold tabular-nums">
				{fmtBytes(totalStorage * 1e9)}
			</div>
		</div>
		<div class="px-5 py-4">
			<div
				class="mb-1.5 flex items-center gap-1.5 text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase"
			>
				<Archive size={11} /> Backups in SLA
			</div>
			<div class="text-[22px] leading-none font-semibold tabular-nums">
				{backupsInSla}<span class="text-[15px] text-tl-fg3">/{CLUSTERS.length}</span>
			</div>
		</div>
	</div>
{/if}

<div class="mb-4 flex flex-wrap items-center gap-2">
	<label class="relative">
		<span class="sr-only">Filter clusters</span>
		<Search
			size={13}
			class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-tl-fg3"
		/>
		<input
			data-filter
			bind:value={query}
			placeholder="Filter clusters, regions, databases…"
			class="h-7 w-[min(300px,70vw)] rounded-md border border-tl-line bg-tl-subtle pr-8 pl-8 text-[12.5px] outline-none focus:border-tl-accent"
		/>
		<kbd class="kbd pointer-events-none absolute top-1/2 right-2 -translate-y-1/2">/</kbd>
	</label>
	{#if shell.demo}
		<div class="inline-flex rounded-md border border-tl-line bg-tl-subtle p-0.5">
			<button
				class="h-6 rounded-[5px] px-2.5 text-[12px] font-medium {filter === 'all'
					? 'bg-tl-elev text-tl-fg ring-1 ring-tl-line2'
					: 'text-tl-fg3 hover:text-tl-fg'}"
				onclick={() => (filter = 'all')}>All</button
			>
			<button
				class="inline-flex h-6 items-center gap-1.5 rounded-[5px] px-2.5 text-[12px] font-medium {filter ===
				'issues'
					? 'bg-tl-elev text-tl-fg ring-1 ring-tl-line2'
					: 'text-tl-fg3 hover:text-tl-fg'}"
				onclick={() => (filter = 'issues')}><Filter size={11} /> Needs attention</button
			>
		</div>
	{/if}
	{#if shell.demo}
		<div class="ml-auto text-[12px] text-tl-fg3">Sorted by health, then load</div>
	{/if}
</div>

{#if shell.demo}
	{#if demoList.length === 0}
		<div class="rounded-xl border border-tl-line bg-tl-elev px-6 py-12 text-center">
			<div
				class="mx-auto mb-3 grid size-10 place-items-center rounded-xl border border-tl-line bg-tl-subtle text-tl-fg3"
			>
				<ServerOff size={18} />
			</div>
			<div class="font-medium">No clusters match</div>
			<p class="mx-auto mt-1 max-w-xs text-[12.5px] text-tl-fg3">
				Nothing matches “{query}”. Try a region like eu-west or an engine like Patroni.
			</p>
			<button
				class="mt-4 inline-flex h-7 items-center rounded-md border border-tl-line bg-tl-elev px-2.5 text-[12.5px] font-medium hover:bg-tl-hover"
				onclick={() => {
					query = '';
					filter = 'all';
				}}>Clear filters</button
			>
		</div>
	{:else}
		<div class="space-y-6">
			{#each lanes as kind (kind)}
				{@const list = demoList.filter((cluster) => cluster.env === kind)}
				{#if list.length}
					<section>
						<div class="mb-2.5 flex items-center gap-2.5">
							<span class="h-4 w-1 rounded-full {envRailClass(kind)}"></span>
							<h2 class="text-[13.5px] font-semibold">{ENV_LABEL[kind]}</h2>
							<span class="text-[12px] text-tl-fg3">{list.length} clusters</span>
							{#if kind === 'prod'}
								<span class="flex items-center gap-1 text-[11px] text-tl-fg3">
									<Shield size={11} class="text-env-prod" /> Read-only by default
								</span>
							{/if}
						</div>
						<div class="grid grid-cols-[repeat(auto-fill,minmax(330px,1fr))] gap-3">
							{#each list as cluster (cluster.id)}
								{@const color = healthColor(cluster.health)}
								{@const connPct = Math.round((cluster.conns / cluster.maxConns) * 100)}
								<button
									class="group relative flex flex-col overflow-hidden rounded-xl border border-tl-line bg-tl-elev p-4 text-left hover:border-tl-line2"
									onclick={() => openCluster(cluster.id)}
								>
									{#if cluster.health !== 'ok'}
										<span class="absolute inset-x-0 top-0 h-0.5" style:background={color}></span>
									{/if}
									<div class="flex w-full items-start gap-3">
										<ScoreRing value={cluster.score} {color} />
										<div class="min-w-0 flex-1">
											<div class="flex items-center gap-1.5">
												<span class="truncate font-mono text-[13px] font-medium"
													>{cluster.name}</span
												>
												<ArrowUpRight
													size={13}
													class="text-tl-fg3 opacity-0 group-hover:opacity-100"
												/>
											</div>
											<div class="mt-0.5 truncate text-[11.5px] text-tl-fg3">
												{cluster.provider} · {cluster.region}
											</div>
										</div>
										<Sparkline data={cluster.load} {color} />
									</div>
									<div class="mt-3 flex flex-wrap gap-1.5">
										<span class="rounded-[5px] bg-tl-subtle px-1.5 py-0.5 text-[11px] font-medium"
											>PG {cluster.version}</span
										>
										<span class="rounded-[5px] bg-tl-subtle px-1.5 py-0.5 text-[11px] font-medium"
											>{cluster.ha}</span
										>
										<span class="rounded-[5px] bg-tl-subtle px-1.5 py-0.5 text-[11px] font-medium"
											>{cluster.primary}P{cluster.replicas ? ` + ${cluster.replicas}R` : ''}</span
										>
									</div>
									{#if cluster.issue}
										<div
											class="mt-3 flex items-center gap-1.5 rounded-md px-2 py-1.5 text-[12px]"
											style:background="color-mix(in srgb, {color} 10%, transparent)"
											style:color
										>
											<AlertTriangle size={12} class="shrink-0" />
											<span class="truncate">{cluster.issue}</span>
										</div>
									{/if}
									<div
										class="mt-3 grid w-full grid-cols-4 gap-2 border-t border-tl-line pt-3 text-[11.5px]"
									>
										<div>
											<div class="text-tl-fg3">TPS</div>
											<div class="mt-0.5 font-medium tabular-nums">{fmt(cluster.tps)}</div>
										</div>
										<div>
											<div class="text-tl-fg3">Conns</div>
											<div
												class="mt-0.5 font-medium tabular-nums"
												style:color={connPct > 90 ? 'var(--env-prod)' : undefined}
											>
												{connPct}%
											</div>
										</div>
										<div>
											<div class="text-tl-fg3">Lag</div>
											<div
												class="mt-0.5 font-medium tabular-nums"
												style:color={cluster.lagSec > 5 ? 'var(--env-staging)' : undefined}
											>
												{cluster.replicas ? `${cluster.lagSec}s` : '—'}
											</div>
										</div>
										<div>
											<div class="text-tl-fg3">Backup</div>
											<div
												class="mt-0.5 font-medium tabular-nums"
												style:color={cluster.backupAgeH > 24
													? 'var(--env-prod)'
													: cluster.backupAgeH > 12
														? 'var(--env-staging)'
														: undefined}
											>
												{cluster.backupAgeH === 0 ? 'n/a' : `${cluster.backupAgeH}h`}
											</div>
										</div>
									</div>
								</button>
							{/each}
						</div>
					</section>
				{/if}
			{/each}
		</div>
	{/if}
{:else if liveList.length === 0}
	<div class="rounded-xl border border-tl-line bg-tl-elev px-6 py-12 text-center">
		<div class="font-medium">No clusters match</div>
		<p class="mx-auto mt-1 max-w-xs text-[12.5px] text-tl-fg3">Nothing matches “{query}”.</p>
	</div>
{:else}
	<div class="space-y-6">
		{#each lanes as kind (kind)}
			{@const list = liveList.filter((cluster) => cluster.env === kind)}
			{#if list.length}
				<section>
					<div class="mb-2.5 flex items-center gap-2.5">
						<span class="h-4 w-1 rounded-full {envRailClass(kind)}"></span>
						<h2 class="text-[13.5px] font-semibold">{ENV_LABEL[kind]}</h2>
						<span class="text-[12px] text-tl-fg3">{list.length}</span>
					</div>
					<div class="grid grid-cols-[repeat(auto-fill,minmax(330px,1fr))] gap-3">
						{#each list as cluster (cluster.id)}
							<button
								class="rounded-xl border border-tl-line bg-tl-elev p-4 text-left hover:border-tl-line2"
								onclick={() => openCluster(cluster.id)}
							>
								<div class="flex items-center gap-2">
									<span class="size-1.5 rounded-full {healthDotClass(cluster.health)}"></span>
									<span class="truncate font-mono text-[13px] font-medium">{cluster.name}</span>
								</div>
								<div class="mt-1 truncate text-[11.5px] text-tl-fg3">
									{cluster.ha} · {cluster.region || 'no host yet'}
								</div>
								<div class="mt-3 flex flex-wrap gap-1.5 text-[11px] font-medium">
									<span class="rounded-[5px] bg-tl-subtle px-1.5 py-0.5">PG {cluster.version}</span>
									{#if cluster.database}
										<span class="rounded-[5px] bg-tl-subtle px-1.5 py-0.5 font-mono"
											>{cluster.database}</span
										>
									{/if}
								</div>
								<p class="mt-3 text-[12px] text-tl-fg3">
									Health, load and backups appear once this cluster is collected.
								</p>
							</button>
						{/each}
					</div>
				</section>
			{/if}
		{/each}
	</div>
{/if}
