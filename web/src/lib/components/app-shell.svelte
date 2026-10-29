<script lang="ts">
	import { onMount, type Snippet } from 'svelte';
	import Activity from '@lucide/svelte/icons/activity';
	import Archive from '@lucide/svelte/icons/archive';
	import Bell from '@lucide/svelte/icons/bell';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Check from '@lucide/svelte/icons/check';
	import Command from '@lucide/svelte/icons/command';
	import GitPullRequestArrow from '@lucide/svelte/icons/git-pull-request-arrow';
	import History from '@lucide/svelte/icons/history';
	import Info from '@lucide/svelte/icons/info';
	import LayoutGrid from '@lucide/svelte/icons/layout-grid';
	import Lock from '@lucide/svelte/icons/lock';
	import Moon from '@lucide/svelte/icons/moon';
	import Network from '@lucide/svelte/icons/network';
	import PlugZap from '@lucide/svelte/icons/plug-zap';
	import Rocket from '@lucide/svelte/icons/rocket';
	import Search from '@lucide/svelte/icons/search';
	import Settings from '@lucide/svelte/icons/settings';
	import Shield from '@lucide/svelte/icons/shield';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import Sun from '@lucide/svelte/icons/sun';
	import Terminal from '@lucide/svelte/icons/terminal';
	import Wrench from '@lucide/svelte/icons/wrench';
	import Zap from '@lucide/svelte/icons/zap';
	import iconDark from '#lib/assets/tusk-arc-icon-dark.svg';
	import iconLight from '#lib/assets/tusk-arc-icon-light.svg';
	import { ENV_LABEL, envPillClass, envRailClass, envTextClass, type EnvKind } from '#lib/env.js';
	import { healthDotClass } from '#lib/fixtures/clusters.js';
	import { GLOBAL_SCREENS, SCREEN_LABEL, shell, type Screen } from '#lib/shell.svelte.js';
	import CommandPalette from '#lib/components/command-palette.svelte';

	let { children }: { children: Snippet } = $props();

	const nav: { id: Screen; label: string; keys: string }[] = [
		{ id: 'cluster', label: 'Health', keys: 'g c' },
		{ id: 'queries', label: 'Query insights', keys: 'g q' },
		{ id: 'sql', label: 'SQL editor', keys: 'g s' },
		{ id: 'ha', label: 'HA topology', keys: 'g h' },
		{ id: 'backups', label: 'Backups', keys: 'g b' },
		{ id: 'migrations', label: 'Migrations', keys: 'g m' },
		{ id: 'advisor', label: 'Config advisor', keys: 'g v' }
	];
	const setup: { id: Screen; label: string; keys: string }[] = [
		{ id: 'setup', label: 'Overview', keys: 'g u' },
		{ id: 'connect', label: 'Add connection', keys: 'n' },
		{ id: 'deploy', label: 'Deploy cluster', keys: 'g d' },
		{ id: 'deployments', label: 'Deployments', keys: 'g l' }
	];

	const kinds: EnvKind[] = ['prod', 'staging', 'dev'];
	let chord = '';

	const global = $derived(GLOBAL_SCREENS.includes(shell.screen));
	const rail = $derived(
		global ? 'bg-linear-to-r from-env-dev via-env-staging to-env-prod' : envRailClass(shell.env)
	);
	$effect(() => {
		shell.hydrate();
	});

	onMount(() => {
		void shell.refresh();
	});

	function onKey(event: KeyboardEvent) {
		const target = event.target as HTMLElement | null;
		const typing =
			target &&
			(target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable);
		if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
			event.preventDefault();
			shell.paletteOpen = !shell.paletteOpen;
			return;
		}
		if (typing || shell.paletteOpen) return;
		if (event.key === 'Escape') {
			shell.envMenuOpen = false;
			shell.helpOpen = false;
			shell.cancelProd();
			return;
		}
		if (event.key === 't') shell.toggleTheme();
		if (event.key === 'e') shell.envMenuOpen = !shell.envMenuOpen;
		if (event.key === '?') shell.helpOpen = !shell.helpOpen;
		if (event.key === '/') {
			const field = document.querySelector<HTMLInputElement>('[data-filter]');
			if (field) {
				event.preventDefault();
				field.focus();
			}
			return;
		}
		if (event.key === '[') shell.stepRange(-1);
		if (event.key === ']') shell.stepRange(1);
		if (event.key === 'n') shell.go('connect');
		if (event.key === 'g') {
			chord = 'g';
			return;
		}
		if (chord === 'g') {
			const next: Partial<Record<string, Screen>> = {
				f: 'fleet',
				c: 'cluster',
				q: 'queries',
				s: 'sql',
				h: 'ha',
				b: 'backups',
				m: 'migrations',
				d: 'deploy',
				v: 'advisor',
				u: 'setup',
				l: 'deployments',
				a: 'fleet'
			};
			if (next[event.key]) shell.go(next[event.key]!);
			chord = '';
		}
	}

	function iconFor(id: Screen) {
		switch (id) {
			case 'cluster':
				return Activity;
			case 'queries':
				return Zap;
			case 'sql':
				return Terminal;
			case 'ha':
				return Network;
			case 'backups':
				return Archive;
			case 'migrations':
				return GitPullRequestArrow;
			case 'advisor':
				return SlidersHorizontal;
			case 'setup':
				return Wrench;
			case 'connect':
				return PlugZap;
			case 'deploy':
				return Rocket;
			default:
				return History;
		}
	}
</script>

<svelte:window onkeydown={onKey} />

<div class="flex h-dvh flex-col bg-tl-bg text-tl-fg">
	<div class="h-[3px] shrink-0 {rail}"></div>
	{#if shell.env === 'prod' && !global}
		<div
			class="flex h-6 shrink-0 items-center justify-center gap-2 border-b border-env-prod/30 bg-env-prod/10 text-[11px] font-medium tracking-wide text-env-prod"
		>
			<Shield size={11} />
			PRODUCTION · read-only session · changes require elevation and typed confirmation
		</div>
	{/if}
	<div class="flex min-h-0 flex-1">
		<aside
			class="flex w-[232px] shrink-0 flex-col overflow-y-auto border-r border-tl-line bg-tl-elev/40"
		>
			<div class="flex h-12 items-center gap-2.5 border-b border-tl-line px-4">
				<img
					src={shell.theme === 'dark' ? iconDark : iconLight}
					alt=""
					class="size-6 shrink-0 rounded-[6px] ring-1 ring-tl-line2"
				/>
				<div class="min-w-0 leading-none">
					<div class="text-sm font-semibold tracking-tight">Tuskline</div>
					<div class="mt-1 text-[10.5px] text-tl-fg3">
						{shell.demo ? 'Acme Retail · demo' : 'Inventory'}
					</div>
				</div>
			</div>
			<nav class="space-y-0.5 p-2" aria-label="Primary">
				<button
					class="nav"
					class:active={shell.screen === 'fleet'}
					onclick={() => shell.go('fleet')}
				>
					<LayoutGrid size={15} />
					<span class="flex-1 text-left">Fleet</span>
					<span class="keys"><kbd>g</kbd><kbd>f</kbd></span>
				</button>
				<button class="nav" onclick={() => shell.go('fleet')} title="Alerts (g a)">
					<Bell size={15} />
					<span class="flex-1 text-left">Alerts</span>
					{#if shell.unhealthy}
						<span
							class="rounded-full bg-env-prod/20 px-1.5 text-[10.5px] text-env-prod tabular-nums"
							>{shell.unhealthy}</span
						>
					{/if}
				</button>
			</nav>
			<div class="px-4 pt-2 pb-1.5 text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">
				Setup & deploy
			</div>
			<div class="space-y-0.5 px-2">
				{#each setup as item (item.id)}
					{@const Icon = iconFor(item.id)}
					<button
						class="nav"
						class:active={shell.screen === item.id}
						onclick={() => shell.go(item.id)}
					>
						<Icon size={15} />
						<span class="flex-1 text-left">{item.label}</span>
						<span class="keys"
							>{#each item.keys.split(' ') as k (k)}<kbd>{k}</kbd>{/each}</span
						>
					</button>
				{/each}
			</div>
			<div class="flex items-center justify-between px-4 pt-3 pb-1.5">
				<span class="text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase">Cluster</span>
				<span
					class="rounded px-1.5 py-0.5 text-[10px] font-medium text-tl-bg {envRailClass(shell.env)}"
					>{shell.env}</span
				>
			</div>
			<div class="px-2">
				<div
					class="mb-1 flex items-center gap-2 rounded-md border border-tl-line bg-tl-subtle px-2.5 py-2"
				>
					<span class="size-1.5 rounded-full {healthDotClass(shell.cluster.health)}"></span>
					<div class="min-w-0 flex-1">
						<div class="truncate font-mono text-[12px]">{shell.cluster.name}</div>
						<div class="truncate text-[10.5px] text-tl-fg3">
							PG {shell.cluster.version} · {shell.cluster.ha}
						</div>
					</div>
				</div>
				<div class="space-y-0.5">
					{#each nav as item (item.id)}
						{@const Icon = iconFor(item.id)}
						<button
							class="nav"
							class:active={shell.screen === item.id}
							onclick={() => shell.go(item.id)}
						>
							<Icon size={15} />
							<span class="flex-1 text-left">{item.label}</span>
							<span class="keys"
								>{#each item.keys.split(' ') as k (k)}<kbd>{k}</kbd>{/each}</span
							>
						</button>
					{/each}
				</div>
			</div>
			<div class="mt-auto space-y-2 p-3">
				<div class="rounded-lg border border-tl-line bg-tl-subtle p-3">
					<div class="mb-1.5 flex items-center gap-1.5 text-[11.5px] text-tl-fg2">
						<Terminal size={12} /> Same thing, from your terminal
					</div>
					<code class="block truncate font-mono text-[11px]"
						>$ tl connect {shell.env}/{shell.cluster.database}</code
					>
				</div>
				<div class="flex items-center gap-1">
					<button class="nav min-w-0 flex-1" onclick={() => shell.go('fleet')}
						><Settings size={14} /> Settings</button
					>
					<button class="nav icon" title="About Tuskline" onclick={() => (shell.helpOpen = true)}
						><Info size={14} /></button
					>
					<button class="nav icon" title="Toggle theme (t)" onclick={() => shell.toggleTheme()}>
						{#if shell.theme === 'dark'}<Sun size={14} />{:else}<Moon size={14} />{/if}
					</button>
				</div>
			</div>
		</aside>
		<div class="flex min-w-0 flex-1 flex-col">
			<header
				class="sticky top-0 z-30 flex h-12 shrink-0 items-center gap-3 border-b border-tl-line bg-tl-bg/80 px-4 backdrop-blur"
			>
				<nav
					class="flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden text-[12.5px] whitespace-nowrap text-tl-fg3"
					aria-label="Breadcrumb"
				>
					<span>Acme Retail</span>
					<ChevronRight size={12} />
					{#if shell.screen === 'fleet'}
						<span class="text-tl-fg">Fleet</span>
					{:else if GLOBAL_SCREENS.includes(shell.screen)}
						<button class="hover:text-tl-fg" onclick={() => shell.go('setup')}>Setup</button>
						<ChevronRight size={12} />
						<span class="text-tl-fg">{SCREEN_LABEL[shell.screen]}</span>
					{:else}
						<span class={envTextClass(shell.env)}>{ENV_LABEL[shell.env]}</span>
						<ChevronRight size={12} />
						<button class="font-mono hover:text-tl-fg" onclick={() => shell.go('cluster')}
							>{shell.cluster.name}</button
						>
						<ChevronRight size={12} />
						<span class="font-mono">{shell.cluster.database}</span>
					{/if}
				</nav>
				<button
					class="ml-auto flex h-7 w-[min(300px,34vw)] shrink-0 items-center gap-2 rounded-md border border-tl-line bg-tl-subtle px-2.5 text-tl-fg3 hover:border-tl-line2"
					onclick={() => (shell.paletteOpen = true)}
				>
					<Search size={13} />
					<span class="flex-1 text-left text-[12.5px]">Search clusters, queries, actions…</span>
					<span class="flex gap-0.5"
						><kbd class="kbd"><Command size={10} /></kbd><kbd class="kbd">K</kbd></span
					>
				</button>
				<label class="sr-only" for="range">Time range</label>
				<select
					id="range"
					class="h-7 rounded-md border border-tl-line bg-tl-subtle px-2 text-[12px]"
					bind:value={shell.range}
					title="Global time range ([ and ] to step)"
				>
					{#each ['15m', '1h', '6h', '24h', '7d', '30d'] as r (r)}
						<option value={r}>Last {r}</option>
					{/each}
				</select>
				<div class="relative">
					<button
						class="flex h-7 shrink-0 items-center gap-2 rounded-md pr-1.5 pl-2 text-[12.5px] font-medium {envPillClass(
							shell.env
						)}"
						title="Switch environment (e)"
						onclick={() => (shell.envMenuOpen = !shell.envMenuOpen)}
					>
						{#if shell.env === 'prod'}<Shield size={13} />{:else}<span
								class="size-1.5 rounded-full bg-current"
							></span>{/if}
						<span class="text-[11px] tracking-wide uppercase">{shell.env}</span>
						<span class="max-w-28 truncate font-mono text-[11.5px] font-normal text-tl-fg2"
							>{shell.cluster.name}</span
						>
						<ChevronDown size={13} />
					</button>
					{#if shell.envMenuOpen}
						<div
							class="absolute top-9 right-0 z-40 w-[330px] rounded-xl border border-tl-line bg-tl-elev p-1.5"
							style="box-shadow: var(--tl-shadow)"
						>
							<div
								class="flex items-center justify-between px-2 py-1.5 text-[10.5px] tracking-wider text-tl-fg3 uppercase"
							>
								Switch environment <kbd class="kbd">e</kbd>
							</div>
							{#each kinds as kind (kind)}
								<button
									class="flex h-8 w-full items-center gap-2 rounded-md px-2 hover:bg-tl-hover"
									onclick={() => shell.requestEnv(kind)}
								>
									<span class="h-4 w-1 rounded-full {envRailClass(kind)}"></span>
									<span class="flex-1 text-left font-medium">{ENV_LABEL[kind]}</span>
									{#if kind === 'prod'}<span
											class="flex items-center gap-1 text-[10.5px] text-tl-fg3"
											><Lock size={10} /> read-only by default</span
										>{/if}
									{#if shell.env === kind}<Check size={14} class="text-tl-accent" />{/if}
								</button>
								{#if shell.env === kind}
									<div class="space-y-0.5 pr-1 pb-1 pl-5">
										{#each shell.clusters.filter((c) => c.env === kind) as cluster (cluster.id)}
											<button
												class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-[12px] hover:bg-tl-hover {cluster.id ===
												shell.clusterId
													? 'text-tl-fg'
													: 'text-tl-fg2'}"
												onclick={() => shell.setCluster(cluster.id)}
											>
												<span class="size-1.5 rounded-full {healthDotClass(cluster.health)}"></span>
												<span class="flex-1 text-left font-mono">{cluster.name}</span>
												<span class="text-[11px] text-tl-fg3">{cluster.region}</span>
											</button>
										{/each}
									</div>
								{/if}
							{/each}
							{#if shell.pendingProd}
								<div
									class="mt-1 rounded-md border border-env-prod/40 bg-env-prod/10 p-2 text-[12px] text-env-prod"
								>
									<p>Switch to production? The session is read-only until you elevate.</p>
									<div class="mt-2 flex gap-2">
										<button
											class="rounded-md bg-env-prod px-2 py-1 text-tl-bg"
											onclick={() => shell.confirmProd()}>Switch</button
										>
										<button class="rounded-md px-2 py-1" onclick={() => shell.cancelProd()}
											>Cancel</button
										>
									</div>
								</div>
							{/if}
						</div>
					{/if}
				</div>
				<div class="avatar" title="Account">TL</div>
			</header>
			<main class="flex-1 overflow-auto">
				<div class="mx-auto max-w-[1600px] p-6">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
</div>

<CommandPalette />

{#if shell.helpOpen}
	<div class="fixed inset-0 z-50 grid place-items-center bg-black/50 p-6">
		<button
			class="absolute inset-0"
			aria-label="Close keyboard shortcuts"
			onclick={() => (shell.helpOpen = false)}
		></button>
		<div
			class="relative z-10 w-[560px] rounded-xl border border-tl-line bg-tl-elev p-4"
			role="dialog"
			aria-label="Keyboard shortcuts"
			tabindex="-1"
		>
			<h2 class="mb-3 text-sm font-semibold">Keyboard shortcuts</h2>
			<div class="grid grid-cols-2 gap-x-8 gap-y-2 text-[13px]">
				{#each [['⌘/Ctrl K', 'Command palette'], ['g f', 'Fleet'], ['g c', 'Cluster health'], ['g q', 'Query insights'], ['g s', 'SQL editor'], ['e', 'Switch environment'], ['t', 'Toggle theme'], ['[ ]', 'Step the time range'], ['?', 'This help']] as row (row[1])}
					<div class="flex items-center justify-between border-b border-tl-line/60 py-1">
						<span class="text-tl-fg2">{row[1]}</span>
						<kbd>{row[0]}</kbd>
					</div>
				{/each}
			</div>
		</div>
	</div>
{/if}

<style>
	.nav {
		display: flex;
		height: 2rem;
		width: 100%;
		align-items: center;
		gap: 0.625rem;
		border-radius: 0.375rem;
		padding-inline: 0.625rem;
		font-size: 13px;
		color: var(--tl-fg2);
	}
	.nav:hover,
	.nav.active {
		background: var(--tl-hover);
		color: var(--tl-fg);
	}
	.nav.icon {
		width: auto;
		flex: none;
		justify-content: center;
		padding-inline: 0.5rem;
	}
	.nav :global(svg) {
		flex-shrink: 0;
	}
	.keys {
		display: none;
		gap: 0.125rem;
	}
	.nav:hover .keys,
	.nav:focus-visible .keys {
		display: flex;
	}
	kbd {
		display: inline-flex;
		min-width: 1.125rem;
		align-items: center;
		justify-content: center;
		border-radius: 0.25rem;
		border: 1px solid var(--tl-line2);
		background: var(--tl-subtle);
		padding-inline: 0.25rem;
		font-family: 'JetBrains Mono', ui-monospace, monospace;
		font-size: 10.5px;
		line-height: 1.125rem;
		color: var(--tl-fg2);
	}
	button:focus-visible,
	select:focus-visible {
		outline: 2px solid var(--tl-accent);
		outline-offset: 2px;
	}
</style>
