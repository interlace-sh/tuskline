<script lang="ts">
	import AlertOctagon from '@lucide/svelte/icons/alert-octagon';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import Database from '@lucide/svelte/icons/database';
	import Download from '@lucide/svelte/icons/download';
	import FileCode2 from '@lucide/svelte/icons/file-code-2';
	import FolderPlus from '@lucide/svelte/icons/folder-plus';
	import PanelRight from '@lucide/svelte/icons/panel-right';
	import Hash from '@lucide/svelte/icons/hash';
	import Key from '@lucide/svelte/icons/key';
	import Link2 from '@lucide/svelte/icons/link-2';
	import Lock from '@lucide/svelte/icons/lock';
	import Maximize2 from '@lucide/svelte/icons/maximize-2';
	import Minimize2 from '@lucide/svelte/icons/minimize-2';
	import Play from '@lucide/svelte/icons/play';
	import Plus from '@lucide/svelte/icons/plus';
	import Eye from '@lucide/svelte/icons/eye';
	import Search from '@lucide/svelte/icons/search';
	import Square from '@lucide/svelte/icons/square';
	import Table2 from '@lucide/svelte/icons/table-2';
	import Timer from '@lucide/svelte/icons/timer';
	import X from '@lucide/svelte/icons/x';
	import Zap from '@lucide/svelte/icons/zap';
	import { ApiError, runQuery, splitRef } from '#lib/api.js';
	import { scheduleProfile } from '#lib/column-profile-cache.js';
	import {
		plannerStatsSQL,
		statsFromPlanner,
		statsFromProfiles,
		type OverviewColumn
	} from '#lib/overview-stats.js';
	import { analyzeNeedsConfirm, explainSQL, writeVerb } from '#lib/explain.js';
	import { CLUSTERS, RESULT_ROWS, SCHEMA } from '#lib/fixtures/demo.js';
	import { canPickFolder, pickSqlFolder, readSqlFile, type SqlNode } from '#lib/sql-files.js';
	import { shell } from '#lib/shell.svelte.js';
	import ColumnProfile from './column-profile.svelte';
	import ExplainView from './explain-view.svelte';
	import OverviewColumns from './overview-columns.svelte';
	import ResultGrid from './result-grid.svelte';
	import ServerExtensions from './server-extensions.svelte';
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

	type Cell = { id: number; name: string; sql: string; result: CellResult; running: boolean };
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

	function cell(id: number, sql: string, name = ''): Cell {
		return { id, name, sql, result: { kind: 'empty' }, running: false };
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
	const profileCell = $derived(
		[...page.cells].reverse().find((entry) => entry.result.kind === 'rows' && !entry.running)
	);

	let schemaOpen = $state<Record<string, boolean>>({ public: true });
	let schemaQuery = $state('');
	let selectedRelation = $state('');
	let dbMenu = $state(false);
	const inspectorOpenHeight = 300;
	const inspectorMaxHeight = 420;
	const inspectorMinHeight = 120;
	let inspectorHeight = $state(inspectorOpenHeight);
	let inspectorSized = $state(false);
	let inspectorBody: HTMLDivElement | undefined = $state();
	let overviewOpen = $state(true);
	let indexesOpen = $state(false);
	const inspectorOpen = $derived(overviewOpen || indexesOpen);
	let filesOpen = $state(false);
	let expanded = $state<number | null>(null);
	let filesNote = $state('');
	let naming = $state('');
	let nameDraft = $state('');
	let folders = $state<{ id: string; name: string; open: boolean; tree: SqlNode[] }[]>([]);
	let openDirs = $state<Record<string, boolean>>({});

	const databases = $derived.by(() => {
		if (shell.demo) {
			const fixture = CLUSTERS.find((cluster) => cluster.id === shell.cluster.id);
			if (fixture?.databases.length) return fixture.databases;
		}
		return [shell.cluster.database || 'postgres'];
	});
	const schemaForDatabase = $derived(
		!shell.demo || shell.cluster.database === databases[0] ? SCHEMA : []
	);

	const relation = $derived.by(() => {
		for (const schema of SCHEMA) {
			const table = schema.tables.find(
				(item) => `${schema.schema}.${item.name}` === selectedRelation
			);
			if (!table) continue;
			const kind = 'kind' in table && table.kind === 'view' ? 'view' : 'table';
			const indexes = 'indexes' in table ? table.indexes : [];
			return { ...table, schema: schema.schema, kind, indexes };
		}
		return null;
	});
	let overview = $state<OverviewColumn[]>([]);
	let overviewLoading = $state(false);
	let overviewNote = $state('');

	$effect(() => {
		const current = relation;
		if (!current) {
			overview = [];
			overviewLoading = false;
			overviewNote = '';
			return;
		}
		let cancelled = false;
		overviewLoading = true;
		overviewNote = '';
		const blank = current.cols.map((column) => ({
			name: column[0],
			nullPct: null,
			distinct: null,
			bars: [],
			depth: false
		}));
		overview = blank;

		if (!shell.demo && shell.cluster.id.includes('/')) {
			const sql = plannerStatsSQL(current.schema, current.name);
			if (!sql) {
				overviewLoading = false;
				overviewNote = 'Statistics stay unread for this name.';
				return;
			}
			const ref = splitRef(shell.cluster.id);
			const controller = new AbortController();
			runQuery(ref.env, ref.name, sql, fetch, controller.signal)
				.then((result) => {
					if (cancelled) return;
					overview = statsFromPlanner(result.rows);
					overviewNote = 'Planner estimates. Not a full scan.';
					overviewLoading = false;
				})
				.catch((err: unknown) => {
					if (cancelled || (err instanceof DOMException && err.name === 'AbortError')) return;
					overviewNote = err instanceof ApiError ? err.message : 'Could not read statistics.';
					overviewLoading = false;
				});
			return () => {
				cancelled = true;
				controller.abort();
			};
		}

		if (current.schema === 'public' && current.name === 'orders') {
			const task = scheduleProfile(fixtureColumns, RESULT_ROWS, (profiles) => {
				if (cancelled) return;
				overview = statsFromProfiles(profiles);
				overviewNote = 'Sample of the demo rows, not the whole table.';
				overviewLoading = false;
			});
			return () => {
				cancelled = true;
				task.cancel();
			};
		}

		overviewLoading = false;
		overviewNote = 'No statistics for this relation.';
		return () => {
			cancelled = true;
		};
	});
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

	let confirm = $state<{ pageId: number; entry: Cell; typed: string } | null>(null);

	function newPage(sql = '', notebook = false, name?: string) {
		const id = nextId++;
		pages = [
			...pages,
			{
				id,
				name: name ?? (notebook ? `notebook-${id}.sqlnb` : `untitled-${id}.sql`),
				notebook,
				cells: [cell(1, sql)]
			}
		];
		active = id;
	}

	function focusOnMount(node: HTMLInputElement) {
		node.focus();
		node.select();
	}

	function commitSnippetName(cellId: number) {
		if (naming !== `${page.id}:${cellId}`) return;
		patchCell(page.id, cellId, { name: nameDraft.trim() });
		naming = '';
	}

	async function addFolder() {
		filesNote = '';
		if (!canPickFolder()) {
			filesNote = 'This browser cannot open a local folder.';
			return;
		}
		const picked = await pickSqlFolder();
		if (!picked.ok) {
			if (picked.reason === 'unsupported') filesNote = 'This browser cannot open a local folder.';
			return;
		}
		folders = [
			...folders,
			{ id: `${picked.name}-${folders.length}`, name: picked.name, open: true, tree: picked.tree }
		];
		if (picked.tree.length === 0) filesNote = `${picked.name} has no .sql files.`;
	}

	async function openSqlFile(id: string, name: string) {
		const text = await readSqlFile(id);
		if (text == null) {
			filesNote = 'That file is no longer available.';
			return;
		}
		newPage(text, name.toLowerCase().endsWith('.sqlnb'), name);
	}

	function closePage(id: number) {
		if (pages.length === 1) return;
		const rest = pages.filter((item) => item.id !== id);
		pages = rest;
		if (active === id) active = rest[0].id;
	}

	function setFormat(notebook: boolean) {
		const item = page;
		if (item.notebook === notebook) return;
		expanded = null;
		pages = pages.map((entry) => {
			if (entry.id !== item.id) return entry;
			const name = notebook
				? entry.name.replace(/\.sql$/, '.sqlnb')
				: entry.name.endsWith('.sqlnb')
					? entry.name.slice(0, -2)
					: entry.name;
			return { ...entry, notebook, name };
		});
	}

	function toggleExpand(cellId: number) {
		expanded = expanded === cellId ? null : cellId;
	}

	function onEscape(event: KeyboardEvent) {
		if (event.key !== 'Escape' || expanded == null) return;
		expanded = null;
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

	function runCell(entry: Cell, explain = false) {
		if (explain && analyzeNeedsConfirm(entry.sql, shell.readOnly)) {
			confirm = { pageId: page.id, entry, typed: '' };
			return;
		}
		void execute(page.id, entry, explain);
	}

	async function execute(pageId: number, entry: Cell, explain = false) {
		resultTab = 'results';
		const verb = writeVerb(entry.sql);
		if (verb && shell.readOnly && !explain) {
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
			if (explain && verb && shell.readOnly) {
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
		const sql = explain ? explainSQL(entry.sql) : entry.sql;
		try {
			const ref = splitRef(shell.cluster.id);
			const result = await runQuery(
				ref.env,
				ref.name,
				sql,
				fetch,
				next.signal,
				explain
					? undefined
					: (partial) => {
							patchCell(pageId, entry.id, {
								running: true,
								result: {
									kind: 'rows',
									columns: partial.columns.map((name) => ({ name, type: '' })),
									rows: partial.rows,
									ms: partial.duration_ms,
									note: partial.truncated ? `Stopped after ${partial.rows.length} rows` : ''
								}
							});
						}
			);
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

	function toggleOverview() {
		overviewOpen = !overviewOpen;
		if (overviewOpen && !inspectorSized) inspectorHeight = inspectorOpenHeight;
	}

	function toggleIndexes() {
		indexesOpen = !indexesOpen;
		if (indexesOpen && !inspectorSized) inspectorHeight = inspectorOpenHeight;
	}

	function resizeInspector(event: PointerEvent) {
		const handle = event.currentTarget;
		if (!(handle instanceof HTMLElement) || !inspectorBody) return;
		event.preventDefault();
		handle.setPointerCapture(event.pointerId);
		const startY = event.clientY;
		const startH = inspectorBody.getBoundingClientRect().height;
		const aside = handle.closest('aside');
		const room = (aside?.getBoundingClientRect().height ?? 640) - 160;
		const maxH = Math.max(inspectorMinHeight, Math.min(inspectorMaxHeight, room));
		const move = (next: PointerEvent) => {
			inspectorSized = true;
			inspectorHeight = Math.round(
				Math.min(maxH, Math.max(inspectorMinHeight, startH - (next.clientY - startY)))
			);
		};
		const stop = () => {
			handle.removeEventListener('pointermove', move);
			handle.removeEventListener('pointerup', stop);
			handle.removeEventListener('pointercancel', stop);
		};
		handle.addEventListener('pointermove', move);
		handle.addEventListener('pointerup', stop);
		handle.addEventListener('pointercancel', stop);
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

<svelte:window onkeydown={onEscape} />

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
			<div class="ml-auto flex items-center gap-3 pr-1 text-[11.5px] text-tl-fg3">
				{#if entry.result.kind === 'rows'}
					<span class="tabular-nums"
						>{entry.result.rows.length} rows{entry.running ? '' : ` · ${entry.result.ms} ms`}
						{entry.result.note}</span
					>
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
				<div class="flex items-center gap-1.5">
					<span>Format</span>
					<div
						class="flex overflow-hidden rounded-md border border-tl-line"
						role="group"
						aria-label="Format"
					>
						<button
							class="h-6 px-2 text-[11px] {!page.notebook
								? 'bg-tl-elev text-tl-fg'
								: 'text-tl-fg3 hover:text-tl-fg'}"
							aria-pressed={!page.notebook}
							onclick={() => setFormat(false)}>SQL</button
						>
						<button
							class="h-6 border-l border-tl-line px-2 text-[11px] {page.notebook
								? 'bg-tl-elev text-tl-fg'
								: 'text-tl-fg3 hover:text-tl-fg'}"
							aria-pressed={page.notebook}
							onclick={() => setFormat(true)}>Notebook</button
						>
					</div>
				</div>
			</div>
		</div>
		<div class="flex min-h-[140px] min-w-0 flex-1 flex-col overflow-hidden">
			{#if entry.running && entry.result.kind !== 'rows'}
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
				<ExplainView text={entry.result.text} />
			{:else if entry.result.kind === 'rows'}
				<ResultGrid columns={entry.result.columns} rows={entry.result.rows} />
			{:else}
				<p class="p-4 text-[12.5px] text-tl-fg3">
					Run a statement with ⌘↵. Results stay with this snippet.
				</p>
			{/if}
		</div>
	</div>
{/snippet}

<div
	class="flex min-h-0 flex-1 flex-col overflow-hidden border border-tl-line bg-tl-elev"
>
	<div class="flex min-h-0 flex-1">
		{#if expanded == null}
			<aside class="flex w-[248px] shrink-0 flex-col border-r border-tl-line">
				<div class="border-b border-tl-line p-2">
					<div class="relative mb-2">
						<button
							class="flex h-7 w-full items-center gap-2 rounded px-1.5 text-left text-[12px] hover:bg-tl-hover"
							aria-expanded={dbMenu}
							aria-haspopup="listbox"
							onclick={() => (dbMenu = !dbMenu)}
						>
							<Database size={13} class="shrink-0 text-tl-fg3" />
							<span class="min-w-0 flex-1 truncate font-mono"
								>{shell.cluster.database || 'postgres'}</span
							>
							<ChevronDown size={12} class="shrink-0 text-tl-fg3" />
						</button>
						{#if dbMenu}
							<div
								class="absolute top-8 right-0 left-0 z-40 rounded-md border border-tl-line bg-tl-elev py-1"
								style="box-shadow: var(--tl-shadow)"
								role="listbox"
								aria-label="Database"
							>
								{#each databases as name (name)}
									<button
										class="flex h-7 w-full items-center px-2 text-left font-mono text-[12px] hover:bg-tl-hover {name ===
										shell.cluster.database
											? 'text-tl-fg'
											: 'text-tl-fg2'}"
										role="option"
										aria-selected={name === shell.cluster.database}
										onclick={() => {
											shell.setDatabase(name);
											selectedRelation = '';
											dbMenu = false;
										}}>{name}</button
									>
								{/each}
							</div>
						{/if}
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
				<ServerExtensions />
				<div class="min-h-0 flex-1 overflow-auto p-1.5 text-[12px]">
					{#if schemaForDatabase.length === 0}
						<p class="px-1.5 py-2 text-[12px] text-tl-fg3">
							Schema for {shell.cluster.database} is not loaded yet.
						</p>
					{/if}
					{#each schemaForDatabase as item (item.schema)}
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
								{@const view = 'kind' in table && table.kind === 'view'}
								<div
									class="flex h-6 w-full items-center gap-0.5 rounded pr-1 pl-3 {selectedRelation ===
									key
										? 'bg-brand text-white'
										: 'hover:bg-tl-hover'}"
								>
									<button
										class="grid size-4 shrink-0 place-items-center rounded"
										aria-expanded={!!schemaOpen[key]}
										aria-label={schemaOpen[key] ? 'Collapse columns' : 'Expand columns'}
										onclick={() => (schemaOpen[key] = !schemaOpen[key])}
									>
										{#if schemaOpen[key]}<ChevronDown size={11} />{:else}<ChevronRight
												size={11}
											/>{/if}
									</button>
									<button
										class="flex min-w-0 flex-1 items-center gap-1.5 text-left"
										title={view ? 'View. Double-click to query' : 'Table. Double-click to query'}
										aria-pressed={selectedRelation === key}
										onclick={() => (selectedRelation = key)}
										ondblclick={() => newPage(`SELECT *\nFROM ${key}\nLIMIT 100;\n`)}
									>
										{#if view}<Eye size={12} class="shrink-0" />{:else}<Table2
												size={12}
												class="shrink-0 {selectedRelation === key ? '' : 'text-tl-accent'}"
											/>{/if}
										<span class="truncate font-mono">{table.name}</span>
										<span class="ml-auto text-[10.5px] {selectedRelation === key ? '' : 'text-tl-fg3'}"
											>{view ? 'view' : table.size}</span
										>
									</button>
								</div>
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
				{#if relation || profileCell?.result.kind === 'rows'}
					<div class="flex shrink-0 flex-col border-t border-tl-line">
						<div
							class="h-2 shrink-0 cursor-row-resize touch-none border-b border-tl-line bg-tl-bg/40 hover:bg-tl-hover"
							role="separator"
							aria-orientation="horizontal"
							aria-label="Resize inspector"
							aria-valuemin={inspectorMinHeight}
							aria-valuemax={inspectorMaxHeight}
							aria-valuenow={inspectorOpen ? inspectorHeight : undefined}
							onpointerdown={resizeInspector}
						></div>
						<div
							bind:this={inspectorBody}
							class="overflow-auto"
							style:height={inspectorOpen ? `${inspectorHeight}px` : undefined}
						>
							<button
								class="flex h-7 w-full items-center gap-1 px-2 text-left text-[12px] text-tl-fg2 hover:bg-tl-hover"
								aria-expanded={overviewOpen}
								onclick={toggleOverview}
							>
								{#if overviewOpen}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
								<span class="font-medium">Overview</span>
								<span class="ml-auto text-[11px] text-tl-fg3">counts</span>
							</button>
							{#if overviewOpen}
								{#if relation}
									<OverviewColumns
										columns={overview}
										loading={overviewLoading}
										note={overviewNote}
									/>
								{:else if profileCell?.result.kind === 'rows'}
									<ColumnProfile
										embedded
										columns={profileCell.result.columns}
										rows={profileCell.result.rows}
									/>
								{/if}
							{/if}
							{#if relation}
								<button
									class="flex h-7 w-full items-center gap-1 border-t border-tl-line px-2 text-left text-[12px] text-tl-fg2 hover:bg-tl-hover"
									aria-expanded={indexesOpen}
									onclick={toggleIndexes}
								>
									{#if indexesOpen}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
									<span class="font-medium">Indexes</span>
									<span class="ml-auto text-[11px] text-tl-fg3 tabular-nums"
										>{relation.indexes?.length ?? 0}</span
									>
								</button>
								{#if indexesOpen}
									<ul class="px-2 pb-2 text-[12px]">
										{#each relation.indexes ?? [] as index (index.name)}
											<li class="flex h-[22px] items-center gap-1.5 px-1">
												{#if index.unique}<Key size={10} style="color: var(--c2)" />{:else}<Hash
														size={10}
														class="text-tl-fg3"
													/>{/if}
												<span class="truncate font-mono">{index.name}</span>
											</li>
										{:else}
											<li class="px-1 text-tl-fg3">No indexes.</li>
										{/each}
									</ul>
								{/if}
							{/if}
						</div>
					</div>
				{/if}
			</aside>
		{/if}
		<div class="flex min-w-0 flex-1 flex-col">
			{#if expanded == null}
				<div class="flex h-9 shrink-0 items-center border-b border-tl-line bg-tl-subtle/60">
					{#each pages as item (item.id)}
						<div
							class="group flex h-full cursor-pointer items-center gap-1.5 border-r border-tl-line pr-1.5 pl-3 text-[12px] {active ===
							item.id
								? 'bg-tl-elev text-tl-fg shadow-[inset_0_2px_0_var(--tl-accent)]'
								: 'text-tl-fg3 hover:text-tl-fg'}"
							role="tab"
							tabindex="0"
							onclick={() => {
								active = item.id;
								expanded = null;
							}}
							onkeydown={(event) => {
								if (event.key === 'Enter') {
									active = item.id;
									expanded = null;
								}
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
						class="ml-auto h-full px-2.5 text-tl-fg3 hover:text-tl-fg {filesOpen
							? 'text-tl-fg'
							: ''}"
						title="Files"
						aria-pressed={filesOpen}
						onclick={() => (filesOpen = !filesOpen)}><PanelRight size={14} /></button
					>
				</div>
			{/if}
			{#if page.notebook}
				{#if expanded == null}
					<div
						class="flex h-10 shrink-0 items-center gap-2 border-b border-tl-line px-3 text-[12px] text-tl-fg3"
					>
						Notebook · each snippet runs on its own
						<button class="{btn} ml-auto" onclick={addCell}><Plus size={12} /> Add snippet</button>
					</div>
				{/if}
				<div
					class={expanded == null
						? 'min-h-0 flex-1 space-y-3 overflow-auto p-3'
						: 'flex min-h-0 flex-1 flex-col'}
				>
					{#each page.cells.filter((entry) => expanded == null || entry.id === expanded) as entry (entry.id)}
						<section
							class={expanded == null
								? 'overflow-hidden rounded-lg border border-tl-line'
								: 'flex min-h-0 flex-1 flex-col'}
						>
							<div class="flex h-9 items-center gap-2 border-b border-tl-line bg-tl-subtle/50 px-2">
								{#if naming === `${page.id}:${entry.id}`}
									<input
										class="h-6 w-28 rounded border border-tl-line bg-tl-bg px-1 font-mono text-[11px] text-tl-fg outline-none"
										aria-label="Snippet name"
										value={nameDraft}
										use:focusOnMount
										oninput={(event) => (nameDraft = event.currentTarget.value)}
										onblur={(event) => {
											nameDraft = event.currentTarget.value;
											commitSnippetName(entry.id);
										}}
										onkeydown={(event) => {
											if (event.key === 'Enter') event.currentTarget.blur();
											if (event.key === 'Escape') {
												event.stopPropagation();
												naming = '';
											}
										}}
									/>
								{:else}
									<button
										class="font-mono text-[11px] text-tl-fg3 hover:text-tl-fg"
										title="Rename snippet"
										onclick={() => {
											naming = `${page.id}:${entry.id}`;
											nameDraft = entry.name;
										}}>{entry.name || `[${entry.id}]`}</button
									>
								{/if}
								<button
									class="{btn} border-transparent bg-brand text-white"
									onclick={() => runCell(entry)}
									disabled={entry.running}
								>
									<Play size={12} /> Run
								</button>
								<button class={btn} onclick={() => runCell(entry, true)}
									><Zap size={12} /> Analyse</button
								>
								<div class="ml-auto flex items-center">
									<button
										class="grid size-6 place-items-center text-tl-fg3 hover:text-tl-fg"
										title={expanded === entry.id ? 'Exit full screen' : 'Full screen'}
										aria-pressed={expanded === entry.id}
										onclick={() => toggleExpand(entry.id)}
									>
										{#if expanded === entry.id}<Minimize2 size={13} />{:else}<Maximize2
												size={13}
											/>{/if}
									</button>
									{#if page.cells.length > 1}
										<button
											class="grid size-6 place-items-center text-tl-fg3 hover:text-tl-fg"
											title="Remove snippet"
											onclick={() => {
												if (expanded === entry.id) expanded = null;
												pages = pages.map((item) =>
													item.id === page.id
														? {
																...item,
																cells: item.cells.filter((cell) => cell.id !== entry.id)
															}
														: item
												);
											}}><X size={13} /></button
										>
									{/if}
								</div>
							</div>
							<div class={expanded === entry.id ? 'min-h-0 flex-1' : ''}>
								{#key `${page.id}-${entry.id}`}
									<SqlEditor
										sql={entry.sql}
										schema={completion}
										minHeight={expanded === entry.id ? 0 : 120}
										onchange={(sql) => patchCell(page.id, entry.id, { sql })}
										onrun={() => runCell(entry)}
										onexplain={() => runCell(entry, true)}
									/>
								{/key}
							</div>
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
						<Zap size={12} /> Analyse
						<span class="ml-0.5 text-[11px] text-tl-fg3">⌘⇧↵</span>
					</button>
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
						<button
							class="grid size-6 place-items-center text-tl-fg3 hover:text-tl-fg"
							title={expanded === page.cells[0].id ? 'Exit full screen' : 'Full screen'}
							aria-pressed={expanded === page.cells[0].id}
							onclick={() => toggleExpand(page.cells[0].id)}
						>
							{#if expanded === page.cells[0].id}<Minimize2 size={14} />{:else}<Maximize2
									size={14}
								/>{/if}
						</button>
					</div>
				</div>
				<div class="min-h-[180px] flex-[1.1]">
					{#key page.id}
						<SqlEditor
							sql={page.cells[0].sql}
							schema={completion}
							onchange={(sql) => patchCell(page.id, page.cells[0].id, { sql })}
							onrun={() => runCell(page.cells[0])}
							onexplain={() => runCell(page.cells[0], true)}
						/>
					{/key}
				</div>
				{@render results(page.cells[0])}
			{/if}
		</div>
		{#if filesOpen && expanded == null}
			<aside class="flex w-[240px] shrink-0 flex-col border-l border-tl-line">
				<div class="flex h-9 shrink-0 items-center gap-2 border-b border-tl-line px-2">
					<span class="text-[12px] font-medium">Files</span>
					<button class="{btn} ml-auto" onclick={addFolder}
						><FolderPlus size={12} /> Add folder</button
					>
				</div>
				<div class="min-h-0 flex-1 overflow-auto p-1.5 text-[12px]">
					{#if filesNote}<p class="px-1.5 py-1 text-tl-fg3">{filesNote}</p>{/if}
					{#each folders as folder (folder.id)}
						<button
							class="flex h-6 w-full items-center gap-1 rounded px-1 text-left hover:bg-tl-hover"
							onclick={() =>
								(folders = folders.map((item) =>
									item.id === folder.id ? { ...item, open: !item.open } : item
								))}
						>
							{#if folder.open}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
							<span class="truncate font-medium">{folder.name}</span>
						</button>
						{#if folder.open}
							{@render fileTree(folder.tree, folder.id)}
						{/if}
					{:else}
						<p class="px-1.5 py-2 text-tl-fg3">Add a folder to open .sql files from disk.</p>
					{/each}
				</div>
			</aside>
		{/if}
	</div>
</div>

{#snippet fileTree(nodes: SqlNode[], prefix: string)}
	<ul class="pl-3">
		{#each nodes as node (`${prefix}/${node.name}`)}
			<li>
				{#if node.kind === 'folder'}
					<button
						class="flex h-6 w-full items-center gap-1 rounded px-1 text-left text-tl-fg2 hover:bg-tl-hover"
						onclick={() =>
							(openDirs[`${prefix}/${node.name}`] = !openDirs[`${prefix}/${node.name}`])}
					>
						{#if openDirs[`${prefix}/${node.name}`]}<ChevronDown size={12} />{:else}<ChevronRight
								size={12}
							/>{/if}
						<span class="truncate">{node.name}</span>
					</button>
					{#if openDirs[`${prefix}/${node.name}`]}
						{@render fileTree(node.children, `${prefix}/${node.name}`)}
					{/if}
				{:else}
					<button
						class="flex h-6 w-full items-center gap-1.5 rounded px-1 text-left hover:bg-tl-hover"
						title="Open {node.name}"
						onclick={() => openSqlFile(node.id, node.name)}
					>
						<FileCode2 size={12} class="shrink-0 text-tl-fg3" />
						<span class="truncate font-mono">{node.name}</span>
					</button>
				{/if}
			</li>
		{/each}
	</ul>
{/snippet}

{#if confirm}
	{@const verb = writeVerb(confirm.entry.sql) ?? 'statement'}
	<div class="fixed inset-0 z-50 grid place-items-center bg-black/50 p-6">
		<div
			class="w-[440px] rounded-xl border border-tl-line bg-tl-elev p-4"
			role="dialog"
			aria-labelledby="analyze-title"
		>
			<h2 id="analyze-title" class="text-sm font-semibold">
				EXPLAIN ANALYZE would run this {verb}
			</h2>
			<p class="mt-2 text-[13px] leading-relaxed text-tl-fg2">
				Production stays read-only, so PostgreSQL rejects the statement. Type
				<span class="font-mono text-tl-fg">{shell.cluster.name}</span> to send it anyway.
			</p>
			<label class="mt-3 block text-[12px] text-tl-fg3">
				Cluster name
				<input
					class="mt-1 h-8 w-full rounded-md border border-tl-line bg-tl-subtle px-2 font-mono text-[13px] text-tl-fg outline-none"
					value={confirm.typed}
					oninput={(event) => {
						if (confirm) confirm.typed = event.currentTarget.value;
					}}
				/>
			</label>
			<div class="mt-4 flex justify-end gap-2">
				<button class={btn} onclick={() => (confirm = null)}>Cancel</button>
				<button
					class="{btn} border-transparent bg-env-prod text-white"
					disabled={confirm.typed !== shell.cluster.name}
					onclick={() => {
						const pending = confirm;
						confirm = null;
						if (pending) void execute(pending.pageId, pending.entry, true);
					}}>Send</button
				>
			</div>
		</div>
	</div>
{/if}
