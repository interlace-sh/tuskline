import { profileColumns, type ColumnProfile } from './column-profile.js';

const MAX = 12;
const cache = new Map<string, ColumnProfile[]>();

/** Below this many cells, profiling stays on the main thread, just after paint. */
const WORKER_CELLS = 2_000;

export function fingerprint(columns: { name: string; type: string }[], rows: unknown[][]): string {
	let hash = 2166136261;
	const mix = (text: string) => {
		for (let i = 0; i < text.length; i++) hash = Math.imul(hash ^ text.charCodeAt(i), 16777619);
	};
	mix(String(rows.length));
	mix('\0');
	for (const column of columns) {
		mix(column.name);
		mix('\0');
		mix(column.type);
		mix('\0');
	}
	const sample = (row: unknown[] | undefined) => {
		if (!row) return;
		for (let i = 0; i < columns.length; i++) mix(row[i] == null ? '\u0001' : String(row[i]));
		mix('\0');
	};
	sample(rows[0]);
	sample(rows[rows.length - 1]);
	const stride = Math.max(1, Math.floor(rows.length / 24));
	for (let i = 0; i < rows.length; i += stride) sample(rows[i]);
	return (hash >>> 0).toString(16);
}

export function readProfileCache(key: string): ColumnProfile[] | undefined {
	const hit = cache.get(key);
	if (!hit) return undefined;
	cache.delete(key);
	cache.set(key, hit);
	return hit;
}

export function writeProfileCache(key: string, profiles: ColumnProfile[]) {
	cache.delete(key);
	cache.set(key, profiles);
	while (cache.size > MAX) {
		const oldest = cache.keys().next().value;
		if (oldest === undefined) break;
		cache.delete(oldest);
	}
}

interface ProfileResponse {
	id: number;
	profiles: ColumnProfile[];
}

let worker: Worker | undefined;
let nextId = 0;
const pending = new Map<number, (profiles: ColumnProfile[]) => void>();

function ensureWorker(): Worker | undefined {
	if (worker) return worker;
	if (typeof Worker === 'undefined') return undefined;
	try {
		worker = new Worker(new URL('./column-profile.worker.ts', import.meta.url), { type: 'module' });
	} catch {
		return undefined;
	}
	worker.onmessage = (event: MessageEvent<ProfileResponse>) => {
		const done = pending.get(event.data.id);
		pending.delete(event.data.id);
		done?.(event.data.profiles);
	};
	return worker;
}

/** Profile after the current frame. Large results go to a worker. Cached by the caller. */
export function scheduleProfile(
	columns: { name: string; type: string }[],
	rows: unknown[][],
	done: (profiles: ColumnProfile[]) => void
): { cancel: () => void } {
	const cells = rows.length * Math.max(columns.length, 1);
	const thread = cells >= WORKER_CELLS ? ensureWorker() : undefined;
	if (!thread) {
		const timer = setTimeout(() => done(profileColumns(columns, rows)), 0);
		return { cancel: () => clearTimeout(timer) };
	}
	const id = ++nextId;
	pending.set(id, done);
	thread.postMessage({ id, columns, rows });
	return {
		cancel: () => pending.delete(id)
	};
}
