<script lang="ts">
	import Play from '@lucide/svelte/icons/play';
	import Square from '@lucide/svelte/icons/square';
	import { ApiError, runQuery, splitRef } from '#lib/api.js';
	import { RESULT_ROWS, SCHEMA } from '#lib/fixtures/demo.js';
	import { shell } from '#lib/shell.svelte.js';
	import SqlEditor from './sql-editor.svelte';

	const sample = `SELECT o.id, o.customer_id, o.status, o.total, o.currency, o.created_at
FROM public.orders o
WHERE o.created_at > now() - interval '15 minutes'
ORDER BY o.created_at DESC
LIMIT 100;`;

	const fixtureColumns = ['id', 'customer_id', 'status', 'total', 'currency', 'created_at'];

	let tabs = $state([{ id: 1, name: 'query.sql', sql: sample }]);
	let active = $state(1);
	let editorKey = $state(0);
	const current = $derived(tabs.find((tab) => tab.id === active) ?? tabs[0]);

	const schema = $derived.by(() => {
		const out: Record<string, string[]> = {};
		if (!shell.demo) return out;
		for (const item of SCHEMA) {
			for (const table of item.tables) {
				const columns = table.cols.map((column) => column[0]);
				out[table.name] = columns;
				out[`${item.schema}.${table.name}`] = columns;
			}
		}
		return out;
	});

	let columns = $state<string[]>([]);
	let rows = $state<unknown[][]>([]);
	let error = $state('');
	let note = $state('');
	let running = $state(false);
	let history = $state<{ sql: string; detail: string }[]>([]);
	let controller: AbortController | null = null;

	function setSql(sql: string) {
		tabs = tabs.map((tab) => (tab.id === active ? { ...tab, sql } : tab));
	}

	function blocked(sql: string): boolean {
		if (!shell.readOnly) return false;
		const body = sql
			.replace(/\/\*[\s\S]*?\*\//g, '')
			.replace(/--.*$/gm, '')
			.trim();
		return body !== '' && !/^(select|with|explain|show|values)\b/i.test(body);
	}

	function remember(sql: string, detail: string) {
		history = [{ sql, detail }, ...history].slice(0, 20);
	}

	async function run() {
		const sql = current.sql;
		error = '';
		note = '';
		if (blocked(sql)) {
			error = 'Production is read-only. This statement was not sent.';
			columns = [];
			rows = [];
			remember(sql, 'blocked');
			return;
		}
		if (shell.demo || !shell.cluster.id.includes('/')) {
			columns = fixtureColumns;
			rows = RESULT_ROWS;
			note = 'Fixture rows. Demo mode does not send SQL to a server.';
			remember(sql, 'fixture');
			return;
		}
		controller?.abort();
		const next = new AbortController();
		controller = next;
		running = true;
		try {
			const ref = splitRef(shell.cluster.id);
			const result = await runQuery(ref.env, ref.name, sql, fetch, next.signal);
			columns = result.columns;
			rows = result.rows;
			note = result.truncated
				? `Stopped after ${result.rows.length} rows.`
				: `${result.rows.length} rows · ${result.duration_ms} ms`;
			remember(sql, note);
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') {
				note = 'Cancelled.';
				return;
			}
			error = err instanceof ApiError ? err.message : 'Could not run the statement.';
			columns = [];
			rows = [];
			remember(sql, 'error');
		} finally {
			running = false;
		}
	}

	function cancel() {
		controller?.abort();
	}

	function loadHistory(sql: string) {
		setSql(sql);
		editorKey += 1;
	}

	const btn =
		'inline-flex h-7 items-center gap-1.5 rounded-md border border-tl-line bg-tl-elev px-2.5 text-[12.5px] font-medium hover:bg-tl-hover disabled:opacity-50';
</script>

<div class="mb-3 flex flex-wrap items-end justify-between gap-3">
	<div>
		<h1 class="text-xl font-semibold tracking-tight">SQL editor</h1>
		<p class="mt-1 text-[13px] text-tl-fg2">
			{#if shell.readOnly}
				Production is read-only. A statement that is not a query is not sent.
			{:else if shell.demo}
				Demo mode shows fixture rows and does not run SQL.
			{:else}
				Runs on {shell.cluster.name} with the environment row limit. Ctrl/Cmd Enter runs.
			{/if}
		</p>
	</div>
	<div class="flex gap-2">
		<button class="{btn} border-transparent bg-brand text-white" onclick={run} disabled={running}>
			<Play size={13} /> Run
		</button>
		<button class={btn} onclick={cancel} disabled={!running}><Square size={12} /> Cancel</button>
	</div>
</div>

<section class="overflow-hidden rounded-xl border border-tl-line bg-tl-elev">
	{#key editorKey}
		<SqlEditor sql={current.sql} {schema} onchange={setSql} onrun={run} />
	{/key}
</section>

{#if error}
	<p class="mt-3 text-[13px] text-env-prod">{error}</p>
{:else if note}
	<p class="mt-3 text-[13px] text-tl-fg2">{note}</p>
{/if}

{#if columns.length > 0}
	<div class="mt-3 max-h-[360px] overflow-auto rounded-xl border border-tl-line bg-tl-elev">
		<table class="w-full text-left text-[12px]">
			<thead class="sticky top-0 bg-tl-elev">
				<tr class="border-b border-tl-line text-tl-fg3">
					{#each columns as column (column)}
						<th class="px-3 py-2 font-medium">{column}</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each rows as row, index (index)}
					<tr class="border-b border-tl-line last:border-0">
						{#each row as cell, cellIndex (cellIndex)}
							<td class="px-3 py-1.5 font-mono whitespace-nowrap tabular-nums">{cell ?? ''}</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

{#if history.length > 0}
	<section class="mt-4">
		<h2 class="mb-2 text-[12px] font-medium tracking-wider text-tl-fg3 uppercase">History</h2>
		<ul class="space-y-1">
			{#each history as item, index (index)}
				<li>
					<button
						class="w-full truncate text-left font-mono text-[12px] text-tl-fg2 hover:text-tl-fg"
						onclick={() => loadHistory(item.sql)}
					>
						<span class="mr-2 text-tl-fg3">{item.detail}</span>{item.sql.replaceAll('\n', ' ')}
					</button>
				</li>
			{/each}
		</ul>
	</section>
{/if}
