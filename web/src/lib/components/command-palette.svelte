<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import iconDark from '#lib/assets/tusk-arc-icon-dark.svg';
	import { DEMO_CLUSTERS, filterCommands, shell, type Command } from '#lib/shell.svelte.js';

	let query = $state('');
	let index = $state(0);
	let input = $state<HTMLInputElement | null>(null);

	const commands = $derived<Command[]>([
		{
			id: 'n-fleet',
			group: 'Navigate',
			label: 'Fleet overview',
			keys: ['g', 'f'],
			run: () => shell.go('fleet')
		},
		{
			id: 'n-cluster',
			group: 'Navigate',
			label: 'Cluster health',
			keys: ['g', 'c'],
			run: () => shell.go('cluster')
		},
		{
			id: 'n-queries',
			group: 'Navigate',
			label: 'Query insights',
			keys: ['g', 'q'],
			run: () => shell.go('queries')
		},
		{
			id: 'n-sql',
			group: 'Navigate',
			label: 'SQL editor',
			keys: ['g', 's'],
			run: () => shell.go('sql')
		},
		{
			id: 'n-ha',
			group: 'Navigate',
			label: 'HA topology',
			keys: ['g', 'h'],
			run: () => shell.go('ha')
		},
		{
			id: 'n-backups',
			group: 'Navigate',
			label: 'Backups & PITR',
			keys: ['g', 'b'],
			run: () => shell.go('backups')
		},
		{
			id: 'n-mig',
			group: 'Navigate',
			label: 'Migrations',
			keys: ['g', 'm'],
			run: () => shell.go('migrations')
		},
		{
			id: 's-deploy',
			group: 'Setup & deploy',
			label: 'Deploy a new cluster…',
			hint: 'tl deploy init',
			keys: ['g', 'd'],
			run: () => shell.go('deploy')
		},
		{
			id: 's-connect',
			group: 'Setup & deploy',
			label: 'New connection…',
			hint: 'tl connect add',
			keys: ['n'],
			run: () => shell.go('connect')
		},
		{
			id: 's-advisor',
			group: 'Setup & deploy',
			label: 'Review configuration (config advisor)',
			hint: 'tl advise config',
			keys: ['g', 'v'],
			run: () => shell.go('advisor')
		},
		{
			id: 'a-theme',
			group: 'Actions',
			label: 'Toggle light / dark theme',
			keys: ['t'],
			run: () => shell.toggleTheme()
		},
		{
			id: 'a-density',
			group: 'Actions',
			label: 'Toggle comfortable / compact density',
			run: () => shell.toggleDensity()
		},
		...(['prod', 'staging', 'dev'] as const).map((env) => ({
			id: `e-${env}`,
			group: 'Environments',
			label: `Switch to ${env}`,
			run: () => shell.requestEnv(env)
		})),
		...DEMO_CLUSTERS.map((cluster) => ({
			id: `c-${cluster.id}`,
			group: 'Clusters',
			label: cluster.name,
			hint: `${cluster.env} · PG ${cluster.version} · ${cluster.region}`,
			run: () => {
				shell.setCluster(cluster.id);
				shell.go('cluster');
			}
		}))
	]);

	const filtered = $derived(filterCommands(commands, query));

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
	<div class="fixed inset-0 z-50 flex justify-center bg-black/50 pt-[12vh]">
		<button
			class="absolute inset-0"
			aria-label="Close command palette"
			onclick={() => (shell.paletteOpen = false)}
		></button>
		<div
			class="relative z-10 flex max-h-[480px] w-[620px] flex-col overflow-hidden rounded-xl border border-tl-line bg-tl-elev shadow-lg"
			role="dialog"
			aria-label="Command palette"
			tabindex="-1"
		>
			<div class="flex h-12 items-center gap-2.5 border-b border-tl-line px-4">
				<img src={iconDark} alt="" class="size-5" />
				<input
					bind:this={input}
					bind:value={query}
					class="flex-1 bg-transparent text-[14px] outline-none placeholder:text-tl-fg3"
					placeholder="Type a command, cluster or query…"
					onkeydown={onKey}
				/>
				<kbd class="text-[10.5px] text-tl-fg3">esc</kbd>
			</div>
			<div class="overflow-auto p-1.5">
				{#if filtered.length === 0}
					<p class="py-10 text-center text-tl-fg3">No results for “{query}”.</p>
				{/if}
				{#each filtered as command, i (command.id)}
					{#if i === 0 || filtered[i - 1].group !== command.group}
						<div
							class="px-2.5 pt-2.5 pb-1 text-[10.5px] font-medium tracking-wider text-tl-fg3 uppercase"
						>
							{command.group}
						</div>
					{/if}
					<button
						class="flex h-9 w-full items-center gap-2.5 rounded-md px-2.5 text-left {i === index
							? 'bg-tl-hover'
							: ''}"
						onmouseenter={() => (index = i)}
						onclick={() => {
							shell.paletteOpen = false;
							command.run();
						}}
					>
						<span
							class="flex-1 truncate {command.group === 'Clusters'
								? 'font-mono text-[12.5px]'
								: ''}">{command.label}</span
						>
						{#if command.hint}<span
								class="max-w-[200px] truncate font-mono text-[11.5px] text-tl-fg3"
								>{command.hint}</span
							>{/if}
						{#if command.keys}<span class="text-[10.5px] text-tl-fg3">{command.keys.join(' ')}</span
							>{/if}
						{#if i === index}<ArrowRight size={13} class="text-tl-fg3" />{/if}
					</button>
				{/each}
			</div>
			<div
				class="flex h-9 items-center gap-4 border-t border-tl-line bg-tl-subtle px-4 text-[11px] text-tl-fg3"
			>
				<span>↑↓ navigate</span>
				<span>↵ run</span>
				<span class="ml-auto font-mono">tl · {filtered.length} results</span>
			</div>
		</div>
	</div>
{/if}
