// Typed client for /api/v1. The UI has no private endpoints.

import type { ServerProbe } from '#lib/extensions.js';

export class ApiError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

export interface Probe {
	at: string;
	version_num: number;
	provider: string;
	read_only: boolean;
	in_recovery: boolean;
}

/** One inventory cluster, without secrets. */
export interface Cluster {
	id: string;
	environment: string;
	name: string;
	kind: string;
	provider?: string;
	host?: string;
	port?: number;
	database?: string;
	user?: string;
	tls?: string;
	last_probe?: Probe | null;
}

export interface Statement {
	query_id: number;
	query: string;
	calls: number;
	total_exec_time_ms: number;
	rows: number;
	shared_blks_hit: number;
	shared_blks_read: number;
	wal_bytes: number;
}

export interface Delta {
	query_id: number;
	query: string;
	calls: number;
	calls_reset?: boolean;
	total_exec_time_ms: number;
	exec_reset?: boolean;
	rows: number;
	shared_blks_hit: number;
	shared_blks_read: number;
	wal_bytes: number;
}

export interface QueriesResponse {
	ref: string;
	server_version_num: number;
	statements: Statement[];
	/** Null until a previous sample exists for this cluster. */
	deltas: Delta[] | null;
}

export interface Session {
	pid: number;
	leader_pid?: number;
	user: string;
	database: string;
	application: string;
	client_addr: string;
	backend_type: string;
	state: string;
	wait_event_type: string;
	wait_event: string;
	xact_age_ns: number;
	query_age_ns: number;
	state_age_ns: number;
	query_id?: number;
	query: string;
	blocked_by?: number[];
}

export interface Wait {
	type: string;
	event: string;
	sessions: number;
}

export interface Lock {
	pid: number;
	locktype: string;
	mode: string;
	granted: boolean;
	relation?: string;
	detail?: string;
	wait_age_ns?: number;
}

export interface BlockingNode {
	pid: number;
	session?: Session;
	blocked?: BlockingNode[];
}

export interface ActivityResponse {
	ref: string;
	at: string;
	sessions: Session[];
	waits: Wait[];
	locks: Lock[];
	blocking: BlockingNode[];
}

export interface Finding {
	check: string;
	severity: 'info' | 'warn';
	summary: string;
	evidence: string;
	fix: string;
}

export interface HealthResponse {
	ref: string;
	taken: string;
	server_version_num: number;
	findings: Finding[];
}

async function getJSON<T>(path: string, fetchImpl: typeof fetch): Promise<T> {
	const response = await fetchImpl(path, { headers: { Accept: 'application/json' } });
	let body: unknown = null;
	try {
		body = await response.json();
	} catch {
		/* a non-JSON error page leaves body null */
	}
	if (!response.ok) {
		const message =
			body && typeof body === 'object' && 'error' in body && typeof body.error === 'string'
				? body.error
				: response.statusText;
		throw new ApiError(response.status, message);
	}
	return body as T;
}

async function sendJSON<T>(path: string, payload: unknown, fetchImpl: typeof fetch): Promise<T> {
	const response = await fetchImpl(path, {
		method: 'POST',
		headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
		body: JSON.stringify(payload)
	});
	let body: unknown = null;
	try {
		body = await response.json();
	} catch {
		/* a non-JSON error page leaves body null */
	}
	if (!response.ok) {
		const message =
			body && typeof body === 'object' && 'error' in body && typeof body.error === 'string'
				? body.error
				: response.statusText;
		throw new ApiError(response.status, message);
	}
	return body as T;
}

/** Split an inventory ref `env/name`. Names do not contain slashes. */
export function splitRef(ref: string): { env: string; name: string } {
	const slash = ref.indexOf('/');
	if (slash <= 0 || slash === ref.length - 1 || ref.indexOf('/', slash + 1) !== -1) {
		throw new ApiError(400, `invalid ref ${ref}`);
	}
	return { env: ref.slice(0, slash), name: ref.slice(slash + 1) };
}

function clusterPath(env: string, name: string, leaf: string): string {
	return `/api/v1/clusters/${encodeURIComponent(env)}/${encodeURIComponent(name)}/${leaf}`;
}

export function serverHealth(fetchImpl: typeof fetch = fetch): Promise<{ status: string }> {
	return getJSON('/api/v1/health', fetchImpl);
}

export function serverVersion(
	fetchImpl: typeof fetch = fetch
): Promise<{ version: string; commit: string }> {
	return getJSON('/api/v1/version', fetchImpl);
}

export async function listClusters(fetchImpl: typeof fetch = fetch): Promise<Cluster[]> {
	const body = await getJSON<{ clusters: Cluster[] | null }>('/api/v1/clusters', fetchImpl);
	return body.clusters ?? [];
}

export interface SavedCluster extends Cluster {
	saved: boolean;
	note?: string;
}

/** Probe and save a cluster. password is sent once and must not be kept by the caller. */
export function saveCluster(
	body: {
		environment: string;
		name: string;
		host: string;
		port: number;
		database: string;
		user: string;
		tls: string;
		password_env?: string;
		password?: string;
	},
	fetchImpl: typeof fetch = fetch
): Promise<SavedCluster> {
	return sendJSON('/api/v1/clusters', body, fetchImpl);
}

export function clusterQueries(
	env: string,
	name: string,
	fetchImpl: typeof fetch = fetch
): Promise<QueriesResponse> {
	return getJSON(clusterPath(env, name, 'queries'), fetchImpl);
}

export function clusterActivity(
	env: string,
	name: string,
	fetchImpl: typeof fetch = fetch
): Promise<ActivityResponse> {
	return getJSON(clusterPath(env, name, 'activity'), fetchImpl);
}

export function clusterHealth(
	env: string,
	name: string,
	fetchImpl: typeof fetch = fetch
): Promise<HealthResponse> {
	return getJSON(clusterPath(env, name, 'health'), fetchImpl);
}

/** Installed and available extensions, plus shared_preload_libraries. */
export function clusterProbe(
	env: string,
	name: string,
	fetchImpl: typeof fetch = fetch
): Promise<ServerProbe> {
	return getJSON(clusterPath(env, name, 'probe'), fetchImpl);
}

export interface QueryResult {
	columns: string[];
	rows: unknown[][];
	truncated: boolean;
	duration_ms: number;
}

/** Apply one newline-delimited query event to an accumulator. */
export function consumeQueryLine(line: string, acc: QueryResult): 'row' | 'end' | 'meta' {
	const value: unknown = JSON.parse(line);
	if (Array.isArray(value)) {
		acc.rows.push(value);
		return 'row';
	}
	if (!value || typeof value !== 'object')
		throw new ApiError(400, 'The result stream was not JSON.');
	const record = value as Record<string, unknown>;
	if (typeof record.error === 'string') throw new ApiError(400, record.error);
	if (Array.isArray(record.columns)) acc.columns = record.columns.map((column) => String(column));
	if (record.end === true) {
		acc.truncated = record.truncated === true;
		acc.duration_ms = typeof record.duration_ms === 'number' ? record.duration_ms : 0;
		return 'end';
	}
	return 'meta';
}

/** Run one statement. Rows arrive as newline-delimited JSON. */
export async function runQuery(
	env: string,
	name: string,
	sql: string,
	fetchImpl: typeof fetch = fetch,
	signal?: AbortSignal,
	onRows?: (partial: QueryResult) => void
): Promise<QueryResult> {
	const response = await fetchImpl(clusterPath(env, name, 'query'), {
		method: 'POST',
		headers: { Accept: 'application/x-ndjson', 'Content-Type': 'application/json' },
		body: JSON.stringify({ sql }),
		signal
	});
	if (!response.ok) {
		let message = response.statusText;
		try {
			const body: unknown = await response.json();
			if (body && typeof body === 'object' && 'error' in body && typeof body.error === 'string') {
				message = body.error;
			}
		} catch {
			/* non-JSON */
		}
		throw new ApiError(response.status, message);
	}
	if (!response.body) throw new ApiError(502, 'The result stream was empty.');
	const acc: QueryResult = { columns: [], rows: [], truncated: false, duration_ms: 0 };
	const reader = response.body.getReader();
	const decoder = new TextDecoder();
	let pending = '';
	let ended = false;
	const take = (line: string) => {
		if (!line) return;
		const kind = consumeQueryLine(line, acc);
		if (kind === 'end') ended = true;
		if (onRows && (kind === 'end' || (kind === 'row' && acc.rows.length % 100 === 0))) {
			onRows({ ...acc, rows: acc.rows.slice() });
		}
	};
	for (;;) {
		const chunk = await reader.read();
		if (chunk.done) break;
		pending += decoder.decode(chunk.value, { stream: true });
		const lines = pending.split('\n');
		pending = lines.pop() ?? '';
		for (const line of lines) take(line);
	}
	pending += decoder.decode();
	if (pending.trim()) take(pending.trim());
	if (!ended) throw new ApiError(502, 'The result stream ended early.');
	return acc;
}

export interface AdviceItem {
	name: string;
	current: string;
	proposed: string;
	apply: string;
	summary: string;
	evidence?: string;
	diff: string;
}

export interface AdviseReport {
	ref: string;
	server_version_num: number;
	advice: AdviceItem[];
	notes: string[];
}

export interface AdviseQuery {
	memory?: string;
	cpus?: number;
	storage?: string;
	workload?: string;
	dbSize?: string;
	os?: string;
	connections?: number;
	pooler?: boolean;
}

/** Preview configuration changes. The server does not apply them. */
export function clusterAdvise(
	env: string,
	name: string,
	query: AdviseQuery = {},
	fetchImpl: typeof fetch = fetch
): Promise<AdviseReport> {
	const params = new URLSearchParams();
	if (query.memory) params.set('memory', query.memory);
	if (query.cpus) params.set('cpus', String(query.cpus));
	if (query.storage) params.set('storage', query.storage);
	if (query.workload) params.set('workload', query.workload);
	if (query.dbSize) params.set('db_size', query.dbSize);
	if (query.os) params.set('os', query.os);
	if (query.connections) params.set('connections', String(query.connections));
	if (query.pooler) params.set('pooler', '1');
	const qs = params.toString();
	const path = clusterPath(env, name, 'advise') + (qs ? `?${qs}` : '');
	return getJSON(path, fetchImpl);
}
