<script lang="ts">
	import { shell } from '#lib/shell.svelte.js';

	let choice = $state<'none' | 'latest' | 'gap'>('none');

	const btn =
		'inline-flex h-8 items-center rounded-md border border-tl-line bg-tl-elev px-3 text-[13px] font-medium hover:bg-tl-hover';
</script>

<div class="mb-5">
	<h1 class="text-xl font-semibold tracking-tight">Backups</h1>
	<p class="mt-1 text-[13px] text-tl-fg2">
		{shell.cluster.name} · pgBackRest · stanza {shell.cluster.database || 'postgres'}
	</p>
</div>

<div class="mb-4 flex gap-2">
	<button class={btn} onclick={() => (choice = 'latest')}>Latest</button>
	<button class={btn} onclick={() => (choice = 'gap')}>Inside the gap</button>
</div>

{#if choice === 'latest'}
	<p class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4 text-[13px] text-tl-fg2">
		Recoverable. Continuous WAL from the latest base backup to the target. Nothing has been
		restored.
	</p>
{:else if choice === 'gap'}
	<p class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4 text-[13px] text-tl-fg2">
		Target is inside a WAL gap. Pick a time before the gap or after it.
	</p>
{/if}
