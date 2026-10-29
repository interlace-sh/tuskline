<script lang="ts">
	import Activity from '@lucide/svelte/icons/activity';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Copy from '@lucide/svelte/icons/copy';
	import Stethoscope from '@lucide/svelte/icons/stethoscope';
	import Zap from '@lucide/svelte/icons/zap';
	import {
		ApiError,
		clusterActivity,
		clusterHealth,
		splitRef,
		type ActivityResponse,
		type HealthResponse
	} from '#lib/api.js';
	import { findingsFromReport, sessionsFromActivity } from '#lib/activity-view.js';
	import { envPillClass } from '#lib/env.js';
	import { healthColor, healthDotClass } from '#lib/fixtures/clusters.js';
	import { CLUSTERS, FINDINGS, SESSIONS, WAIT_KEYS, loadSeries } from '#lib/fixtures/demo.js';
	import { fmt, fmtBytes } from '#lib/format.js';
	import { shell } from '#lib/shell.svelte.js';
	import SeriesChart from './series-chart.svelte';

	const fixture = $derived(CLUSTERS.find((cluster) => cluster.id === shell.cluster.id));
	const series = $derived.by(() => {
		if (!fixture) return [];
		const seed = [...fixture.id].reduce((sum, char) => sum + char.charCodeAt(0), 0);
		return loadSeries(seed);
	});
	const scale = $derived(fixture ? fixture.tps / 4800 : 1);

	let hidden = $state<string[]>([]);
	let open = $state<string | null>('f1');
	let notice = $state('');

	let report = $state<HealthResponse | null>(null);
	let activity = $state<ActivityResponse | null>(null);
	let loading = $state(false);
	let error = $state('');

	const totals = $derived(
		WAIT_KEYS.map((wait) => ({
			...wait,
			sum: series.reduce((sum, row) => sum + Number(row[wait.key] || 0), 0)
		}))
	);
	const waitTotal = $derived(totals.reduce((sum, wait) => sum + wait.sum, 0) || 1);
	const visibleWaits = $derived(totals.filter((wait) => !hidden.includes(wait.key)));

	const liveFindings = $derived(findingsFromReport(report?.findings ?? []));
	const liveSessions = $derived(sessionsFromActivity(activity?.sessions ?? []));

	$effect(() => {
		const id = shell.cluster.id;
		if (shell.demo || !id.includes('/')) return;
		let cancelled = false;
		loading = true;
		error = '';
		const ref = splitRef(id);
		Promise.all([clusterHealth(ref.env, ref.name), clusterActivity(ref.env, ref.name)])
			.then(([health, snapshot]) => {
				if (cancelled) return;
				report = health;
				activity = snapshot;
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				report = null;
				activity = null;
				error = err instanceof ApiError ? err.message : 'Could not read this cluster.';
			})
			.finally(() => {
				if (!cancelled) loading = false;
			});
		return () => {
			cancelled = true;
		};
	});

	function toggleWait(key: string) {
		hidden = hidden.includes(key) ? hidden.filter((item) => item !== key) : [...hidden, key];
	}

	async function copyFix(text: string) {
		try {
			await navigator.clipboard.writeText(text);
			notice = 'Copied the suggested statement. It has not been run.';
		} catch {
			notice = 'Could not copy. Select the statement instead.';
		}
	}

	function investigate(screen: string | undefined) {
		if (screen === 'queries' || screen === 'ha' || screen === 'sql' || screen === 'cluster') {
			shell.go(screen);
		}
	}

	const btn =
		'inline-flex h-7 items-center gap-1.5 rounded-md border border-tl-line bg-tl-elev px-2.5 text-[12.5px] font-medium hover:bg-tl-hover';
</script>

{#snippet diagnosis(
	items: {
		id: string;
		sev: 'ok' | 'warn' | 'crit';
		title: string;
		detail: string;
		fix?: string;
		screen?: string;
	}[]
)}
	<section class="overflow-hidden rounded-xl border border-tl-line bg-tl-elev">
		<header class="flex h-11 items-center gap-2 border-b border-tl-line px-4">
			<Stethoscope size={14} class="text-tl-accent" />
			<h3 class="text-[13px] font-semibold">Diagnosis</h3>
			<span class="text-[12px] text-tl-fg3">ranked by what the checks found</span>
		</header>
		{#if items.length === 0}
			<p class="px-4 py-6 text-[13px] text-tl-fg2">No findings. The checks that ran were clean.</p>
		{:else}
			<div class="divide-y divide-tl-line">
				{#each items as item (item.id)}
					<div class="px-4 py-3">
						<button
							class="flex w-full items-start gap-2.5 text-left"
							onclick={() => (open = open === item.id ? null : item.id)}
						>
							<span class="mt-1 size-1.5 shrink-0 rounded-full {healthDotClass(item.sev)}"></span>
							<span class="flex-1 leading-snug font-medium">{item.title}</span>
							{#if open === item.id}
								<ChevronDown size={14} class="mt-0.5 text-tl-fg3" />
							{:else}
								<ChevronRight size={14} class="mt-0.5 text-tl-fg3" />
							{/if}
						</button>
						{#if open === item.id}
							<div class="mt-2 pl-[18px]">
								<p class="text-[12.5px] leading-relaxed text-tl-fg2">{item.detail}</p>
								{#if item.fix}
									<div class="relative mt-2.5 rounded-md border border-tl-line bg-tl-subtle p-2.5">
										<code class="block pr-6 font-mono text-[11.5px] whitespace-pre-wrap"
											>{item.fix}</code
										>
										<button
											class="absolute top-2 right-2 text-tl-fg3 hover:text-tl-fg"
											title="Copy the statement"
											onclick={() => copyFix(item.fix ?? '')}
										>
											<Copy size={12} />
										</button>
									</div>
								{/if}
								{#if item.screen}
									<button class="{btn} mt-2.5" onclick={() => investigate(item.screen)}>
										{#if item.screen === 'ha'}<Activity size={12} />{:else}<Zap size={12} />{/if}
										Investigate
									</button>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</section>
{/snippet}

{#snippet sessions(
	rows: {
		pid: number;
		user: string;
		app: string;
		state: string;
		wait: string;
		dur: string;
		query: string;
		blocked: boolean;
		blocking: number;
	}[]
)}
	<section class="overflow-hidden rounded-xl border border-tl-line bg-tl-elev">
		<header class="flex h-11 items-center justify-between border-b border-tl-line px-4">
			<div class="flex items-baseline gap-2">
				<h3 class="text-[13px] font-semibold">Active sessions</h3>
				<span class="text-[12px] text-tl-fg3">pg_stat_activity</span>
			</div>
			<button class={btn} onclick={() => shell.go('sql')}>Open in SQL</button>
		</header>
		<div class="overflow-x-auto">
			<table class="w-full text-left text-[12px]">
				<thead>
					<tr class="border-b border-tl-line text-tl-fg3">
						{#each ['PID', 'User / app', 'State', 'Wait event', 'Duration', 'Query'] as heading (heading)}
							<th class="px-4 py-2 font-medium">{heading}</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each rows as row (row.pid)}
						<tr class="border-b border-tl-line last:border-0">
							<td class="px-4 py-2 font-mono tabular-nums">
								{#if row.blocked}<span class="mr-1 text-tl-fg3">└</span>{/if}{row.pid}
							</td>
							<td class="px-4 py-2">
								<div>{row.user}</div>
								<div class="text-[11px] text-tl-fg3">{row.app}</div>
							</td>
							<td class="px-4 py-2">
								<span
									class="rounded-[5px] px-1.5 py-0.5 text-[11px] font-medium"
									style:background="color-mix(in srgb, {row.state === 'active'
										? 'var(--env-dev)'
										: 'var(--env-staging)'} 13%, transparent)"
									style:color={row.state === 'active' ? 'var(--env-dev)' : 'var(--env-staging)'}
									>{row.state}</span
								>
							</td>
							<td
								class="px-4 py-2 font-mono text-[11.5px]"
								style:color={row.wait.startsWith('Lock') ? 'var(--env-prod)' : 'var(--tl-fg2)'}
								>{row.wait}</td
							>
							<td class="px-4 py-2 whitespace-nowrap tabular-nums">{row.dur}</td>
							<td class="max-w-[300px] truncate px-4 py-2 font-mono text-[11.5px] text-tl-fg2"
								>{row.query}</td
							>
						</tr>
					{:else}
						<tr><td class="px-4 py-6 text-tl-fg3" colspan="6">No client sessions.</td></tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if rows.some((row) => row.blocking > 0)}
			<p class="border-t border-tl-line px-4 py-2 text-[11.5px] text-tl-fg3">
				{#each rows.filter((row) => row.blocking > 0) as row (row.pid)}
					pid {row.pid} is blocking {row.blocking}.
				{/each}
				Cancelling a backend needs a preview and a confirmation. That is a later milestone.
			</p>
		{/if}
	</section>
{/snippet}

<div class="mb-4 flex flex-wrap items-end justify-between gap-3">
	<div class="min-w-0">
		<div class="flex items-center gap-2">
			<span class="size-2 rounded-full {healthDotClass(shell.cluster.health)}"></span>
			<h1 class="truncate font-mono text-xl font-semibold tracking-tight">{shell.cluster.name}</h1>
		</div>
		<p class="mt-1 flex flex-wrap items-center gap-2 text-[13px] text-tl-fg2">
			<span class="rounded-md px-1.5 py-0.5 text-[11px] font-medium {envPillClass(shell.env)}"
				>{shell.env}</span
			>
			PostgreSQL {shell.cluster.version} · {shell.cluster.ha} · {shell.cluster.region}
			{#if fixture}
				· {fixture.primary} primary, {fixture.replicas} replicas
			{/if}
			· last {shell.range}
		</p>
	</div>
	<button class="{btn} border-transparent bg-brand text-white" onclick={() => shell.go('sql')}>
		Open SQL
	</button>
</div>

{#if notice}
	<p class="mb-3 text-[12.5px] text-tl-fg2">{notice}</p>
{/if}

{#if shell.demo && fixture}
	<div
		class="mb-4 grid grid-cols-2 overflow-hidden rounded-xl border border-tl-line bg-tl-elev sm:grid-cols-3 xl:grid-cols-6 xl:divide-x xl:divide-tl-line"
	>
		<div class="px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">Health score</div>
			<div
				class="mt-1 text-[20px] leading-none font-semibold tabular-nums"
				style:color={healthColor(fixture.health)}
			>
				{fixture.score}
			</div>
		</div>
		<div class="px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">
				Transactions/s
			</div>
			<div class="mt-1 text-[20px] leading-none font-semibold tabular-nums">{fmt(fixture.tps)}</div>
		</div>
		<div class="px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">
				Avg active sessions
			</div>
			<div class="mt-1 text-[20px] leading-none font-semibold tabular-nums">
				{fmt(6.1 * scale, 1)}
			</div>
		</div>
		<div class="px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">Connections</div>
			<div class="mt-1 text-[20px] leading-none font-semibold tabular-nums">
				{fixture.conns}<span class="text-[14px] text-tl-fg3">/{fixture.maxConns}</span>
			</div>
		</div>
		<div class="px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">Cache hit</div>
			<div class="mt-1 text-[20px] leading-none font-semibold tabular-nums">99.4%</div>
			<p class="mt-1 text-[11.5px] text-tl-fg3">fixture series</p>
		</div>
		<div class="px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">
				Max replica lag
			</div>
			<div
				class="mt-1 text-[20px] leading-none font-semibold tabular-nums"
				style:color={fixture.lagSec > 5 ? 'var(--env-staging)' : undefined}
			>
				{fixture.replicas ? `${fixture.lagSec}s` : '—'}
			</div>
			<p class="mt-1 text-[11.5px] text-tl-fg3">
				{fixture.replicas ? `${fmtBytes(fixture.lagBytes)} behind` : 'no replicas'}
			</p>
		</div>
	</div>

	<div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
		<div class="min-w-0 space-y-4">
			<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
				<div class="mb-3 flex flex-wrap items-baseline justify-between gap-2">
					<div>
						<h3 class="text-[13px] font-semibold">Database load</h3>
						<p class="text-[12px] text-tl-fg3">
							Average active sessions by wait event · prototype series
						</p>
					</div>
				</div>
				<div class="mb-3 flex flex-wrap gap-1.5">
					{#each totals as wait (wait.key)}
						<button
							class="flex h-6 items-center gap-1.5 rounded-md border px-2 text-[11.5px] {hidden.includes(
								wait.key
							)
								? 'border-tl-line opacity-40'
								: 'border-tl-line2 bg-tl-subtle'}"
							onclick={() => toggleWait(wait.key)}
						>
							<span class="size-2 rounded-sm" style:background={wait.color}></span>
							<span class="font-mono">{wait.key}</span>
							<span class="text-tl-fg3 tabular-nums"
								>{Math.round((wait.sum / waitTotal) * 100)}%</span
							>
						</button>
					{/each}
				</div>
				<SeriesChart rows={series} series={visibleWaits} height={240} stacked />
			</section>
			<div class="grid gap-4 md:grid-cols-2">
				<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
					<h3 class="mb-2 text-[13px] font-semibold">
						Throughput <span class="font-normal text-tl-fg3">commits/s</span>
					</h3>
					<SeriesChart rows={series} series={[{ key: 'tps', color: 'var(--c1)' }]} />
				</section>
				<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
					<h3 class="mb-2 text-[13px] font-semibold">
						Connections <span class="font-normal text-tl-fg3">by state</span>
					</h3>
					<SeriesChart
						rows={series}
						series={[
							{ key: 'active', color: 'var(--c1)' },
							{ key: 'idleTx', color: 'var(--c5)' },
							{ key: 'idle', color: 'var(--c6)' }
						]}
						height={150}
						stacked
					/>
				</section>
				<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
					<h3 class="mb-2 text-[13px] font-semibold">Buffer cache hit ratio</h3>
					<SeriesChart
						rows={series}
						series={[{ key: 'hit', color: 'var(--c3)' }]}
						domain={[96, 100]}
					/>
				</section>
				<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
					<h3 class="mb-2 text-[13px] font-semibold">
						Replication lag <span class="font-normal text-tl-fg3">seconds</span>
					</h3>
					<SeriesChart
						rows={series}
						series={[
							{ key: 'lag2', color: 'var(--c4)' },
							{ key: 'lag3', color: 'var(--c2)' }
						]}
					/>
				</section>
			</div>
			{@render sessions(
				SESSIONS.map((row) => ({
					pid: row.pid,
					user: row.user,
					app: row.app,
					state: row.state,
					wait: row.wait,
					dur: row.dur,
					query: row.query,
					blocked: 'blockedBy' in row,
					blocking: 'blocking' in row && typeof row.blocking === 'number' ? row.blocking : 0
				}))
			)}
		</div>
		<div class="space-y-4">
			{@render diagnosis(FINDINGS)}
			<section class="rounded-xl border border-tl-line bg-tl-elev p-4">
				<h3 class="text-[13px] font-semibold">Top waits</h3>
				<p class="mb-3 text-[12px] text-tl-fg3">prototype series</p>
				<div class="space-y-2.5">
					{#each [...totals].sort((a, b) => b.sum - a.sum) as wait (wait.key)}
						<div>
							<div class="mb-1 flex justify-between text-[12px]">
								<span class="font-mono text-tl-fg2">{wait.key}</span>
								<span class="tabular-nums">{((wait.sum / waitTotal) * 100).toFixed(1)}%</span>
							</div>
							<div class="h-1.5 overflow-hidden rounded-full bg-tl-subtle">
								<div
									class="h-full rounded-full"
									style:width="{(wait.sum / waitTotal) * 100}%"
									style:background={wait.color}
								></div>
							</div>
						</div>
					{/each}
				</div>
			</section>
		</div>
	</div>
{:else if loading}
	<p class="text-sm text-tl-fg2">Reading health and activity…</p>
{:else if error}
	<section class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4">
		<h2 class="text-[13px] font-semibold">Could not read this cluster</h2>
		<p class="mt-1 text-[13px] text-tl-fg2">{error}</p>
	</section>
{:else}
	<div class="mb-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
		<div class="rounded-xl border border-tl-line bg-tl-elev px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">Findings</div>
			<div class="mt-1 text-[20px] leading-none font-semibold tabular-nums">
				{liveFindings.length}
			</div>
		</div>
		<div class="rounded-xl border border-tl-line bg-tl-elev px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">Sessions</div>
			<div class="mt-1 text-[20px] leading-none font-semibold tabular-nums">
				{liveSessions.length}
			</div>
		</div>
		<div class="rounded-xl border border-tl-line bg-tl-elev px-4 py-4">
			<div class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">Server</div>
			<div class="mt-1 text-[20px] leading-none font-semibold tabular-nums">
				{report ? Math.floor(report.server_version_num / 10000) : shell.cluster.version}
			</div>
		</div>
	</div>
	<p class="mb-4 max-w-xl text-[13px] text-tl-fg3">
		Load charts use stored metric history, which this screen does not read yet. The findings and
		sessions below are from this cluster.
	</p>
	<div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
		{@render sessions(liveSessions)}
		{@render diagnosis(liveFindings)}
	</div>
{/if}
