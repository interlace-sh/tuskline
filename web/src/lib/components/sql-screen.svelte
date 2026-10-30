<script lang="ts">
	import AlertOctagon from '@lucide/svelte/icons/alert-octagon';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Database from '@lucide/svelte/icons/database';
	import Download from '@lucide/svelte/icons/download';
	import FileCode2 from '@lucide/svelte/icons/file-code-2';
	import Hash from '@lucide/svelte/icons/hash';
	import Key from '@lucide/svelte/icons/key';
	import Link2 from '@lucide/svelte/icons/link-2';
	import Lock from '@lucide/svelte/icons/lock';
	import Play from '@lucide/svelte/icons/play';
	import Plus from '@lucide/svelte/icons/plus';
	import Search from '@lucide/svelte/icons/search';
	import Shield from '@lucide/svelte/icons/shield';
	import Square from '@lucide/svelte/icons/square';
	import Table2 from '@lucide/svelte/icons/table-2';
	import Timer from '@lucide/svelte/icons/timer';
	import X from '@lucide/svelte/icons/x';
	import Zap from '@lucide/svelte/icons/zap';
	import { ApiError, runQuery, splitRef } from '#lib/api.js';
	import { RESULT_ROWS, SCHEMA } from '#lib/fixtures/demo.js';
	import { shell } from '#lib/shell.svelte.js';
	import SqlEditor from './sql-editor.svelte';

	type RowResult = {
		kind: 'rows';
		columns: { name: string; type: string }[];
		rows: unknown[][];
		ms: number;
		note: string;
	};
	type CellResult =
		| RowResult
		| { kind: 'error'; message: string; stmt: string }
		| { kind: 'explain'; text: string; ms: number }
		| { kind: 'ok'; message: string; ms: number }
		| { kind: 'empty' };

	type Cell = { id: number; sql: string; result: CellResult; running: boolean };
	type Page = { id: number; name: string; notebook: boolean; cells: Cell[] };

	const fixtureColumns = [
		{ name: 'id', type: 'int8' },
		{ name: 'customer_id', type: 'int8' },
		{ name: 'status', type: 'order_status' },
		{ name: 'total', type: 'numeric' },
		{ name: 'currency', type: 'bpchar' },
		{ name: 'created_at', type: 'timestamptz' }
	];

	const explainText = `Limit  (cost=0.56..412.80 rows=100) (actual time=0.031..0.912 rows=100 loops=1)
  ->  Index Scan Backward using idx_orders_created_at on orders o  (actual time=0.030..0.874 rows=100 loops=1)
        Index Cond: (created_at > (now() - '00:15:00'::interval))
        Buffers: shared hit=212
Planning Time: 0.188 ms
Execution Time: 0.951 ms`;

	function cell(id: number, sql: string): Cell {
		return { id, sql, result: { kind: 'empty' }, running: false };
	}

	let pages = $state<Page[]>([
		{
			id: 1,
			name: 'recent-orders.sql',
			notebook: false,
			cells: [
				cell(
					1,
					`SELECT o.id, o.customer_id, o.status, o.total, o.currency, o.created_at
FROM public.orders o
WHERE o.created_at > now() - interval '15 minutes'
ORDER BY o.created_at DESC
LIMIT 100;`
				)
			]
		},
		{
			id: 2,
			name: 'lock-tree.sql',
			notebook: false,
			cells: [
				cell(
					1,
					`SELECT pid, pg_blocking_pids(pid) AS blocked_by, wait_event_type, state, query
FROM pg_stat_activity
WHERE cardinality(pg_blocking_pids(pid)) > 0;`
				)
			]
		},
		{
			id: 3,
			name: 'checks.sqlnb',
			notebook: true,
			cells: [
				cell(1, 'SELECT current_database() AS db, current_user AS role;'),
				cell(
					2,
					`SELECT pid, state, wait_event_type, left(query, 80) AS query
FROM pg_stat_activity
WHERE backend_type = 'client backend'
LIMIT 20;`
				)
			]
		}
	]);
	let active = $state(1);
	let nextId = 4;
	const page = $derived(pages.find((item) => item.id === active) ?? pages[0]);

	let schemaOpen = $state<Record<string, boolean>>({ public: true, 'public.orders': true });
	let schemaQuery = $state('');
	let resultTab = $state<'results' | 'messages'>('results');
	let controller: AbortController | null = null;

	const completion = $derived.by(() => {
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

	function patchCell(pageId: number, cellId: number, patch: Partial<Cell>) {
		pages = pages.map((item) =>
			item.id === pageId
				? {
						...item,
						cells: item.cells.map((entry) => (entry.id === cellId ? { ...entry, ...patch } : entry))
					}
				: item
		);
	}

	function writeVerb(sql: string): string | null {
		const match = /^\s*(?:--[^\n]*\n\s*)*(update|delete|insert|alter|drop|truncate|create)\b/i.exec(
			sql
		);
		return match ? match[1].toUpperCase() : null;
	}

	function newPage(sql = '', notebook = false) {
		const id = nextId++;
		pages = [
			...pages,
			{
				id,
				name: notebook ? `notebook-${id}.sqlnb` : `untitled-${id}.sql`,
				notebook,
				cells: [cell(1, sql)]
			}
		];
		active = id;
	}

	function closePage(id: number) {
		if (pages.length === 1) return;
		const rest = pages.filter((item) => item.id !== id);
		pages = rest;
		if (active === id) active = rest[0].id;
	}

	function addCell() {
		const item = page;
		const id = Math.max(...item.cells.map((entry) => entry.id)) + 1;
		pages = pages.map((entry) =>
			entry.id === item.id
				? { ...entry, notebook: true, cells: [...entry.cells, cell(id, '')] }
				: entry
		);
		if (!item.name.endsWith('.sqlnb')) {
			pages = pages.map((entry) =>
				entry.id === item.id ? { ...entry, name: entry.name.replace(/\.sql$/, '.sqlnb') } : entry
			);
		}
	}

	async function runCell(entry: Cell, explain = false) {
		const pageId = page.id;
		resultTab = 'results';
		const verb = writeVerb(entry.sql);
		if (verb && shell.readOnly) {
			patchCell(pageId, entry.id, {
				running: false,
				result: {
					kind: 'error',
					stmt: verb,
					message: `cannot execute ${verb} in a read-only transaction`
				}
			});
			return;
		}
		if (shell.demo || !shell.cluster.id.includes('/')) {
			patchCell(pageId, entry.id, {
				running: false,
				result: explain
					? { kind: 'explain', text: explainText, ms: 21 }
					: verb
						? { kind: 'ok', message: `${verb} 37 · fixture, not sent`, ms: 42 }
						: {
								kind: 'rows',
								columns: fixtureColumns,
								rows: RESULT_ROWS,
								ms: 18,
								note: 'Fixture rows'
							}
			});
			return;
		}
		controller?.abort();
		const next = new AbortController();
		controller = next;
		patchCell(pageId, entry.id, { running: true });
		const sql = explain ? `EXPLAIN (ANALYZE, BUFFERS) ${entry.sql}` : entry.sql;
		try {
			const ref = splitRef(shell.cluster.id);
			const result = await runQuery(ref.env, ref.name, sql, fetch, next.signal);
			if (explain) {
				const text = result.rows.map((row) => String(row[0] ?? '')).join('\n');
				patchCell(pageId, entry.id, {
					running: false,
					result: { kind: 'explain', text: text || 'No plan text.', ms: result.duration_ms }
				});
				return;
			}
			patchCell(pageId, entry.id, {
				running: false,
				result: {
					kind: 'rows',
					columns: result.columns.map((name) => ({ name, type: '' })),
					rows: result.rows,
					ms: result.duration_ms,
					note: result.truncated ? `Stopped after ${result.rows.length} rows` : ''
				}
			});
		} catch (err) {
			if (err instanceof DOMException && err.name === 'AbortError') {
				patchCell(pageId, entry.id, { running: false });
				return;
			}
			patchCell(pageId, entry.id, {
				running: false,
				result: {
					kind: 'error',
					stmt: verb ?? 'QUERY',
					message: err instanceof ApiError ? err.message : 'Could not run the statement.'
				}
			});
		}
	}

	function cancel() {
		controller?.abort();
	}

	function csv(result: RowResult) {
		const lines = [
			result.columns.map((column) => column.name).join(','),
			...result.rows.map((row) => row.map((cell) => JSON.stringify(cell ?? '')).join(','))
		];
		const blob = new Blob([lines.join('\n')], { type: 'text/csv' });
		const url = URL.createObjectURL(blob);
		const link = document.createElement('a');
		link.href = url;
		link.download = 'query.csv';
		link.click();
		URL.revokeObjectURL(url);
	}

	const btn =
		'inline-flex h-7 items-center gap-1.5 rounded-md border border-tl-line bg-tl-elev px-2.5 text-[12.5px] font-medium hover:bg-tl-hover disabled:opacity-50';
</script>

{#snippet results(entry: Cell)}
	<div class="flex min-h-0 flex-1 flex-col border-t border-tl-line">
		<div class="flex h-9 shrink-0 items-center gap-1 border-b border-tl-line bg-tl-subtle/60 px-2">
			{#each ['results', 'messages'] as tab (tab)}
				<button
					class="h-7 rounded px-2.5 text-[12px] capitalize {resultTab === tab
						? 'bg-tl-elev text-tl-fg ring-1 ring-tl-line'
						: 'text-tl-fg3 hover:text-tl-fg'}"
					onclick={() => (resultTab = tab as 'results' | 'messages')}>{tab}</button
				>
			{/each}
			<div class="ml-auto flex items-center gap-3 pr-2 text-[11.5px] text-tl-fg3">
				{#if entry.result.kind === 'rows'}
					<span class="tabular-nums">{entry.result.rows.length} rows · {entry.result.ms} ms</span>
					<button
						class="flex items-center gap-1 hover:text-tl-fg"
						onclick={() => csv(entry.result as RowResult)}
					>
						<Download size={12} /> CSV
					</button>
				{/if}
				{#if entry.result.kind === 'explain'}
					<span>EXPLAIN (ANALYZE, BUFFERS)</span>
				{/if}
			</div>
		</div>
		<div class="min-h-[140px] flex-1 overflow-auto">
			{#if entry.running}
				<div class="space-y-2 p-4">
					{#each [90, 80, 70, 60] as width (width)}
						<div class="h-5 animate-pulse rounded bg-tl-subtle" style:width="{width}%"></div>
					{/each}
				</div>
			{:else if resultTab === 'messages'}
				<pre class="p-4 font-mono text-[12px] text-tl-fg2">{entry.result.kind === 'error'
						? `ERROR:  ${entry.result.message}`
						: entry.result.kind === 'empty'
							? 'Run a statement with ⌘↵.'
							: 'OK'}</pre>
			{:else if entry.result.kind === 'error'}
				<div class="m-4 flex gap-3 rounded-lg border border-env-prod/40 bg-env-prod/10 p-4">
					<AlertOctagon size={18} class="mt-0.5 shrink-0 text-env-prod" />
					<div>
						<div class="font-mono text-[12.5px] text-env-prod">ERROR: {entry.result.message}</div>
						<p class="mt-1.5 text-[12.5px] text-tl-fg2">
							{shell.readOnly
								? `Tuskline did not send this ${entry.result.stmt}. Production sessions are read-only.`
								: 'The server rejected the statement. Nothing else was retried.'}
						</p>
					</div>
				</div>
			{:else if entry.result.kind === 'ok'}
				<p class="p-4 font-mono text-[12.5px] text-env-dev">
					{entry.result.message} · {entry.result.ms} ms
				</p>
			{:else if entry.result.kind === 'explain'}
				<pre
					class="p-4 font-mono text-[12px] leading-relaxed whitespace-pre-wrap text-tl-fg2">{entry
						.result.text}</pre>
			{:else if entry.result.kind === 'rows'}
				<table class="min-w-full font-mono text-[12px]">
					<thead class="sticky top-0 z-10 bg-tl-elev">
						<tr class="border-b border-tl-line">
							<th
								class="w-10 border-r border-tl-line px-2 py-1.5 text-right font-normal text-tl-fg3"
								>#</th
							>
							{#each entry.result.columns as column (column.name)}
								<th
									class="border-r border-tl-line px-3 py-1.5 text-left font-medium whitespace-nowrap"
								>
									<span class="text-tl-fg">{column.name}</span>
									{#if column.type}<span class="text-[10.5px] font-normal text-tl-fg3">
											{column.type}</span
										>{/if}
								</th>
							{/each}
						</tr>
					</thead>
					<tbody>
						{#each entry.result.rows as row, index (index)}
							<tr class="border-b border-tl-line hover:bg-tl-hover/60">
								<td class="border-r border-tl-line px-2 py-1 text-right text-tl-fg3">{index + 1}</td
								>
								{#each row as value, cellIndex (cellIndex)}
									<td class="border-r border-tl-line px-3 py-1 whitespace-nowrap">{value ?? ''}</td>
								{/each}
							</tr>
						{/each}
					</tbody>
				</table>
			{:else}
				<p class="p-4 text-[12.5px] text-tl-fg3">
					Run a statement with ⌘↵. Results stay with this snippet.
				</p>
			{/if}
		</div>
	</div>
{/snippet}

<div
	class="flex flex-col overflow-hidden rounded-xl border border-tl-line bg-tl-elev"
	style:height={shell.env === 'prod' ? 'calc(100dvh - 11.5rem)' : 'calc(100dvh - 9.5rem)'}
>
	{#if shell.readOnly}
		<div
			class="flex h-10 shrink-0 items-center gap-2.5 border-b border-env-prod/30 bg-env-prod/10 px-4 text-[12.5px] text-env-prod"
		>
			<Shield size={14} />
			<span class="font-semibold">Production · read-only</span>
			<span class="text-tl-fg2">
				Session runs with <span class="font-mono">default_transaction_read_only = on</span>. Writes
				are not sent.
			</span>
		</div>
	{/if}
	<div class="flex min-h-0 flex-1">
		<aside class="flex w-[248px] shrink-0 flex-col border-r border-tl-line">
			<div class="border-b border-tl-line p-2">
				<div class="mb-2 flex items-center gap-2 px-1.5 py-1 text-[12px]">
					<Database size={13} class="text-tl-fg3" />
					<span class="flex-1 truncate font-mono">{shell.cluster.database || 'postgres'}</span>
				</div>
				<label class="relative block">
					<span class="sr-only">Filter objects</span>
					<Search
						size={12}
						class="pointer-events-none absolute top-1/2 left-2 -translate-y-1/2 text-tl-fg3"
					/>
					<input
						bind:value={schemaQuery}
						placeholder="Filter objects"
						class="h-[26px] w-full rounded-md border border-tl-line bg-tl-subtle pr-2 pl-7 text-[12px] outline-none"
					/>
				</label>
			</div>
			<div class="flex-1 overflow-auto p-1.5 text-[12px]">
				{#each SCHEMA as item (item.schema)}
					<button
						class="flex h-6 w-full items-center gap-1 rounded px-1 text-tl-fg2 hover:bg-tl-hover"
						onclick={() => (schemaOpen[item.schema] = !schemaOpen[item.schema])}
					>
						{#if schemaOpen[item.schema]}<ChevronDown size={12} />{:else}<ChevronRight
								size={12}
							/>{/if}
						<span class="font-medium">{item.schema}</span>
						<span class="ml-auto text-[11px] text-tl-fg3">{item.tables.length}</span>
					</button>
					{#if schemaOpen[item.schema]}
						{#each item.tables.filter( (table) => table.name.includes(schemaQuery.toLowerCase()) ) as table (table.name)}
							{@const key = `${item.schema}.${table.name}`}
							<button
								class="flex h-6 w-full items-center gap-1.5 rounded pr-1 pl-4 hover:bg-tl-hover"
								title="Open a query for this table"
								onclick={() => (schemaOpen[key] = !schemaOpen[key])}
								ondblclick={() => newPage(`SELECT *\nFROM ${key}\nLIMIT 100;\n`)}
							>
								{#if schemaOpen[key]}<ChevronDown
										size={11}
										class="text-tl-fg3"
									/>{:else}<ChevronRight size={11} class="text-tl-fg3" />{/if}
								<Table2 size={12} class="text-tl-accent" />
								<span class="truncate font-mono">{table.name}</span>
								<span class="ml-auto text-[10.5px] text-tl-fg3">{table.size}</span>
							</button>
							{#if schemaOpen[key]}
								{#each table.cols as column (`${key}.${column[0]}`)}
									<div class="flex h-[22px] items-center gap-1.5 rounded pr-1 pl-10 text-tl-fg2">
										{#if column[2] === 'pk'}<Key size={10} style="color: var(--c2)" />
										{:else if column[2] === 'fk'}<Link2 size={10} style="color: var(--c3)" />
										{:else}<Hash size={10} class="text-tl-fg3" />{/if}
										<span class="truncate font-mono">{column[0]}</span>
										<span class="ml-auto font-mono text-[10.5px] text-tl-fg3">{column[1]}</span>
									</div>
								{/each}
							{/if}
						{/each}
					{/if}
				{/each}
			</div>
		</aside>
		<div class="flex min-w-0 flex-1 flex-col">
			<div class="flex h-9 shrink-0 items-center border-b border-tl-line bg-tl-subtle/60">
				{#each pages as item (item.id)}
					<div
						class="group flex h-full cursor-pointer items-center gap-1.5 border-r border-tl-line pr-1.5 pl-3 text-[12px] {active ===
						item.id
							? 'bg-tl-elev text-tl-fg shadow-[inset_0_2px_0_var(--tl-accent)]'
							: 'text-tl-fg3 hover:text-tl-fg'}"
						role="tab"
						tabindex="0"
						onclick={() => (active = item.id)}
						onkeydown={(event) => {
							if (event.key === 'Enter') active = item.id;
						}}
					>
						<FileCode2 size={12} />
						<span class="font-mono">{item.name}</span>
						<button
							class="grid size-4 place-items-center rounded opacity-0 group-hover:opacity-100 hover:bg-tl-hover"
							title="Close page"
							onclick={(event) => {
								event.stopPropagation();
								closePage(item.id);
							}}><X size={11} /></button
						>
					</div>
				{/each}
				<button
					class="h-full px-2.5 text-tl-fg3 hover:text-tl-fg"
					title="New SQL page"
					onclick={() => newPage()}><Plus size={14} /></button
				>
				<button
					class="h-full px-2.5 text-tl-fg3 hover:text-tl-fg"
					title="New notebook"
					onclick={() => newPage('SELECT 1;\n', true)}
					><span class="text-[12px]">+ notebook</span></button
				>
			</div>
			{#if page.notebook}
				<div
					class="flex h-10 shrink-0 items-center gap-2 border-b border-tl-line px-3 text-[12px] text-tl-fg3"
				>
					Notebook · each snippet runs on its own
					<button class="{btn} ml-auto" onclick={addCell}><Plus size={12} /> Add snippet</button>
				</div>
				<div class="min-h-0 flex-1 space-y-3 overflow-auto p-3">
					{#each page.cells as entry (entry.id)}
						<section class="overflow-hidden rounded-lg border border-tl-line">
							<div class="flex h-9 items-center gap-2 border-b border-tl-line bg-tl-subtle/50 px-2">
								<span class="font-mono text-[11px] text-tl-fg3">[{entry.id}]</span>
								<button
									class="{btn} border-transparent bg-brand text-white"
									onclick={() => runCell(entry)}
									disabled={entry.running}
								>
									<Play size={12} /> Run
								</button>
								<button class={btn} onclick={() => runCell(entry, true)}
									><Zap size={12} /> Explain</button
								>
								{#if page.cells.length > 1}
									<button
										class="ml-auto text-tl-fg3 hover:text-tl-fg"
										title="Remove snippet"
										onclick={() => {
											pages = pages.map((item) =>
												item.id === page.id
													? { ...item, cells: item.cells.filter((cell) => cell.id !== entry.id) }
													: item
											);
										}}><X size={13} /></button
									>
								{/if}
							</div>
							{#key `${page.id}-${entry.id}`}
								<SqlEditor
									sql={entry.sql}
									schema={completion}
									minHeight={120}
									onchange={(sql) => patchCell(page.id, entry.id, { sql })}
									onrun={() => runCell(entry)}
								/>
							{/key}
							{@render results(entry)}
						</section>
					{/each}
				</div>
			{:else}
				<div class="flex h-10 shrink-0 items-center gap-2 border-b border-tl-line px-3">
					<button
						class="{btn} border-transparent bg-brand text-white"
						onclick={() => runCell(page.cells[0])}
						disabled={page.cells[0].running}
					>
						<Play size={12} />
						{page.cells[0].running ? 'Running…' : 'Run'}
						<span class="ml-0.5 text-[11px] opacity-70">⌘↵</span>
					</button>
					<button class={btn} onclick={() => runCell(page.cells[0], true)}>
						<Zap size={12} /> Explain analyse
						<span class="ml-0.5 text-[11px] text-tl-fg3">⌘⇧↵</span>
					</button>
					<button class={btn} onclick={addCell}><Plus size={12} /> Add snippet</button>
					<button class={btn} onclick={cancel} disabled={!page.cells[0].running}
						><Square size={12} /> Cancel</button
					>
					<div class="ml-auto flex items-center gap-2 text-[11.5px] text-tl-fg3">
						<span class="flex items-center gap-1"><Timer size={12} /> statement timeout 30s</span>
						<span>·</span>
						<span>Limit {shell.readOnly ? '1,000' : '10,000'}</span>
						<span>·</span>
						<span class="inline-flex items-center gap-1 rounded-[5px] bg-tl-subtle px-1.5 py-0.5">
							{#if shell.readOnly}<Lock size={10} /> read-only{:else}read-write{/if}
						</span>
					</div>
				</div>
				<div class="min-h-[180px] flex-[1.1]">
					{#key page.id}
						<SqlEditor
							sql={page.cells[0].sql}
							schema={completion}
							onchange={(sql) => patchCell(page.id, page.cells[0].id, { sql })}
							onrun={() => runCell(page.cells[0])}
						/>
					{/key}
				</div>
				{@render results(page.cells[0])}
			{/if}
		</div>
	</div>
</div>
