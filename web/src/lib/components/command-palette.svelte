<script lang="ts">
	import Activity from '@lucide/svelte/icons/activity';
	import Archive from '@lucide/svelte/icons/archive';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import GitPullRequestArrow from '@lucide/svelte/icons/git-pull-request-arrow';
	import Keyboard from '@lucide/svelte/icons/keyboard';
	import LayoutGrid from '@lucide/svelte/icons/layout-grid';
	import Moon from '@lucide/svelte/icons/moon';
	import Network from '@lucide/svelte/icons/network';
	import PlugZap from '@lucide/svelte/icons/plug-zap';
	import Rocket from '@lucide/svelte/icons/rocket';
	import Server from '@lucide/svelte/icons/server';
	import Shield from '@lucide/svelte/icons/shield';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import Terminal from '@lucide/svelte/icons/terminal';
	import Zap from '@lucide/svelte/icons/zap';
	import type { Component } from 'svelte';
	import iconDark from '#lib/assets/tusk-arc-icon-dark.svg';
	import iconLight from '#lib/assets/tusk-arc-icon-light.svg';
	import { healthDotClass } from '#lib/fixtures/clusters.js';
	import { filterCommands, shell, type Command } from '#lib/shell.svelte.js';

	type Row = Command & { icon: Component };

	let query = $state('');
	let index = $state(0);
	let input = $state<HTMLInputElement | null>(null);

	const commands = $derived<Row[]>([
		{
			id: 'n-fleet',
			group: 'Navigate',
			label: 'Fleet overview',
			keys: ['g', 'f'],
			icon: LayoutGrid,
			run: () => shell.go('fleet')
		},
		{
			id: 'n-cluster',
			group: 'Navigate',
			label: 'Cluster health',
			keys: ['g', 'c'],
			icon: Activity,
			run: () => shell.go('cluster')
		},
		{
			id: 'n-queries',
			group: 'Navigate',
			label: 'Query insights',
			keys: ['g', 'q'],
			icon: Zap,
			run: () => shell.go('queries')
		},
		{
			id: 'n-sql',
			group: 'Navigate',
			label: 'SQL editor',
			keys: ['g', 's'],
			icon: Terminal,
			run: () => shell.go('sql')
		},
		{
			id: 'n-ha',
			group: 'Navigate',
			label: 'HA topology',
			keys: ['g', 'h'],
			icon: Network,
			run: () => shell.go('ha')
		},
		{
			id: 'n-backups',
			group: 'Navigate',
			label: 'Backups & PITR',
			keys: ['g', 'b'],
			icon: Archive,
			run: () => shell.go('backups')
		},
		{
			id: 'n-mig',
			group: 'Navigate',
			label: 'Migrations',
			keys: ['g', 'm'],
			icon: GitPullRequestArrow,
			run: () => shell.go('migrations')
		},
		{
			id: 's-deploy',
			group: 'Setup & deploy',
			label: 'Deploy a new cluster…',
			hint: 'tl deploy init',
			keys: ['g', 'd'],
			icon: Rocket,
			run: () => shell.go('deploy')
		},
		{
			id: 's-connect',
			group: 'Setup & deploy',
			label: 'New connection…',
			hint: 'tl connect add',
			keys: ['n'],
			icon: PlugZap,
			run: () => shell.go('connect')
		},
		{
			id: 's-advisor',
			group: 'Setup & deploy',
			label: 'Review configuration (config advisor)',
			hint: 'tl advise config',
			keys: ['g', 'v'],
			icon: SlidersHorizontal,
			run: () => shell.go('advisor')
		},
		{
			id: 'a-theme',
			group: 'Actions',
			label: 'Toggle light / dark theme',
			keys: ['t'],
			icon: Moon,
			run: () => shell.toggleTheme()
		},
		{
			id: 'a-help',
			group: 'Actions',
			label: 'Keyboard shortcuts',
			keys: ['?'],
			icon: Keyboard,
			run: () => (shell.helpOpen = true)
		},
		...(['prod', 'staging', 'dev'] as const).map((env) => ({
			id: `e-${env}`,
			group: 'Environments',
			label: `Switch to ${env}`,
			icon: env === 'prod' ? Shield : Server,
			run: () => shell.requestEnv(env)
		})),
		...shell.clusters.map((cluster) => ({
			id: `c-${cluster.id}`,
			group: 'Clusters',
			label: cluster.name,
			hint: `${cluster.env} · PG ${cluster.version} · ${cluster.region}`,
			icon: Server,
			run: () => {
				shell.setCluster(cluster.id);
				shell.go('cluster');
			}
		}))
	]);

	const filtered = $derived(filterCommands(commands, query));

	function healthDot(id: string): string {
		const health = shell.clusters.find((cluster) => `c-${cluster.id}` === id)?.health;
		return healthDotClass(health ?? 'unknown');
	}

	function iconClass(id: string): string {
		if (id === 'e-prod') return 'text-env-prod';
		if (id === 'e-staging') return 'text-env-staging';
		if (id === 'e-dev') return 'text-env-dev';
		return 'text-tl-fg3';
	}

	$effect(() => {
		if (shell.paletteOpen) {
			query = '';
			index = 0;
			queueMicrotask(() => input?.focus());
		}
	});

	function onKey(event: KeyboardEvent) {
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			index = Math.min(filtered.length - 1, index + 1);
		}
		if (event.key === 'ArrowUp') {
			event.preventDefault();
			index = Math.max(0, index - 1);
		}
		if (event.key === 'Enter' && filtered[index]) {
			shell.paletteOpen = false;
			filtered[index].run();
		}
		if (event.key === 'Escape') shell.paletteOpen = false;
	}
</script>

{#if shell.paletteOpen}
	<div
		class="fixed inset-0 z-50 flex items-start justify-center bg-black/50 pt-[12vh] backdrop-blur-[2px]"
	>
		<button
			class="absolute inset-0"
			aria-label="Close command palette"
			onclick={() => (shell.paletteOpen = false)}
		></button>
		<div
			class="relative z-10 flex h-fit max-h-[480px] w-[min(620px,calc(100vw-2rem))] flex-col overflow-hidden rounded-xl border border-tl-line bg-tl-elev"
			style="box-shadow: var(--tl-shadow)"
			role="dialog"
			aria-label="Command palette"
			tabindex="-1"
		>
			<div class="flex h-12 items-center gap-2.5 border-b border-tl-line px-4">
				<img
					src={shell.theme === 'dark' ? iconDark : iconLight}
					alt=""
					class="size-5 shrink-0 rounded-[5px] ring-1 ring-tl-line2"
				/>
				<input
					bind:this={input}
					bind:value={query}
					class="flex-1 bg-transparent text-[14px] text-tl-fg outline-none placeholder:text-tl-fg3"
					placeholder="Type a command, cluster or query…"
					onkeydown={onKey}
				/>
				<kbd class="kbd">esc</kbd>
			</div>
			<div class="max-h-[396px] overflow-auto p-1.5">
				{#if filtered.length === 0}
					<p class="py-10 text-center text-tl-fg3">
						No results for “{query}”. Try “deploy” or “prod”.
					</p>
				{/if}
				{#each filtered as command, i (command.id)}
					{@const Icon = command.icon}
					{#if i === 0 || filtered[i - 1].group !== command.group}
						<div
							class="px-2.5 pt-2.5 pb-1 text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase"
						>
							{command.group}
						</div>
					{/if}
					<button
						class="flex h-9 w-full items-center gap-2.5 rounded-md px-2.5 text-left text-[13px] {i ===
						index
							? 'bg-tl-hover'
							: ''}"
						onmouseenter={() => (index = i)}
						onclick={() => {
							shell.paletteOpen = false;
							command.run();
						}}
					>
						{#if command.group === 'Clusters'}
							<span class="size-1.5 shrink-0 rounded-full {healthDot(command.id)}"></span>
						{:else}
							<Icon size={15} class="shrink-0 {iconClass(command.id)}" />
						{/if}
						<span
							class="flex-1 truncate {command.group === 'Clusters'
								? 'font-mono text-[12.5px]'
								: ''}">{command.label}</span
						>
						{#if command.hint}<span
								class="max-w-[200px] truncate font-mono text-[11.5px] text-tl-fg3"
								>{command.hint}</span
							>{/if}
						{#if command.keys}<span class="flex gap-0.5"
								>{#each command.keys as key (key)}<kbd class="kbd">{key}</kbd>{/each}</span
							>{/if}
						{#if i === index}<ArrowRight size={13} class="text-tl-fg3" />{/if}
					</button>
				{/each}
			</div>
			<div
				class="flex h-9 items-center gap-4 border-t border-tl-line bg-tl-subtle px-4 text-[11px] text-tl-fg3"
			>
				<span class="flex items-center gap-1"
					><kbd class="kbd">↑</kbd><kbd class="kbd">↓</kbd> navigate</span
				>
				<span class="flex items-center gap-1"><kbd class="kbd">↵</kbd> run</span>
				<span class="ml-auto font-mono">tl · {filtered.length} results</span>
			</div>
		</div>
	</div>
{/if}
