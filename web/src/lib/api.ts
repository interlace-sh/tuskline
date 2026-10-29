// Typed client for /api/v1. The UI has no private endpoints.

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
