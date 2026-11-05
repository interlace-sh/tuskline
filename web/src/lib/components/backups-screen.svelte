<script lang="ts">
	import { ApiError, parseBackup, type BackupReport } from '#lib/api.js';
	import { shell } from '#lib/shell.svelte.js';

	let choice = $state<'none' | 'latest' | 'gap'>('none');
	let parsed = $state<BackupReport | null>(null);
	let parseError = $state('');

	async function onFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		parseError = '';
		parsed = null;
		try {
			parsed = await parseBackup(await file.text());
		} catch (err) {
			parseError = err instanceof ApiError ? err.message : 'Could not parse that file.';
		}
	}

	const btn =
		'inline-flex h-8 items-center rounded-md border border-tl-line bg-tl-elev px-3 text-[13px] font-medium hover:bg-tl-hover';
</script>

<div class="mb-5">
	<h1 class="text-xl font-semibold tracking-tight">Backups</h1>
	<p class="mt-1 text-[13px] text-tl-fg2">
		{shell.cluster.name} · pgBackRest · stanza {shell.cluster.database || 'postgres'}
	</p>
</div>

{#if !shell.demo}
	<section class="max-w-xl rounded-xl border border-tl-line bg-tl-elev p-4 text-[13px] text-tl-fg2">
		<p>
			{shell.cluster.backup
				? `Registered repository: ${shell.cluster.backup}.`
				: 'No backup repository is registered on this cluster.'}
		</p>
		{#if shell.cluster.restoreDrill}
			<p class="mt-2">Restore drill: {shell.cluster.restoreDrill}.</p>
		{/if}
		<p class="mt-2">
			Choose a <code class="font-mono">pgbackrest info --output=json</code> file. Nothing is restored.
		</p>
		<input
			class="mt-3 block text-[13px]"
			type="file"
			accept="application/json,.json"
			onchange={onFile}
		/>
		{#if parseError}<p class="mt-2 text-env-prod">{parseError}</p>{/if}
		{#if parsed}
			<ul class="mt-3 space-y-1 font-mono text-[12px]">
				{#each parsed.Stanzas ?? [] as stanza (stanza.Name)}
					<li>
						{stanza.Name}: {stanza.Backups?.length ?? 0} backups, {stanza.Gaps?.length ?? 0} WAL gaps
					</li>
				{/each}
			</ul>
		{/if}
	</section>
{:else}
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
{/if}
