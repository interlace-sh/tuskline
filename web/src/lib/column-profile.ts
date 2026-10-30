export type ColumnKind = 'category' | 'number' | 'time';

export interface ValueCount {
	label: string;
	count: number;
	pct: number;
	null: boolean;
}

export interface HistogramBucket {
	label: string;
	count: number;
}

export interface ColumnProfile {
	name: string;
	type: string;
	rows: number;
	nulls: number;
	nullPct: number;
	distinct: number;
	kind: ColumnKind;
	/** Most common values, null included when present. */
	values: ValueCount[];
	buckets: HistogramBucket[];
	min?: string;
	max?: string;
}

const NUMBER = /^-?\d[\d,]*(?:\.\d+)?$/;

function asNumber(value: unknown): number | null {
	if (typeof value === 'number' && Number.isFinite(value)) return value;
	if (typeof value !== 'string') return null;
	const cleaned = value.replace(/,/g, '');
	if (!NUMBER.test(cleaned)) return null;
	const n = Number(cleaned);
	return Number.isFinite(n) ? n : null;
}

function asTime(value: unknown): number | null {
	if (value instanceof Date) return value.getTime();
	if (typeof value !== 'string' || value.length < 8) return null;
	let normalised = value.includes('T') ? value : value.replace(' ', 'T');
	normalised = normalised.replace(/([+-]\d{2})$/, '$1:00');
	const parsed = Date.parse(normalised);
	return Number.isNaN(parsed) ? null : parsed;
}

function kindOf(type: string, present: unknown[]): ColumnKind {
	const declared = type.toLowerCase();
	if (/int|numeric|float|double|real|decimal|money/.test(declared)) return 'number';
	if (/time|date/.test(declared)) return 'time';
	if (present.length === 0) return 'category';
	if (present.every((value) => asNumber(value) !== null)) return 'number';
	if (present.every((value) => asTime(value) !== null)) return 'time';
	return 'category';
}

function histogram(points: number[], buckets = 8): HistogramBucket[] {
	if (points.length === 0) return [];
	const min = Math.min(...points);
	const max = Math.max(...points);
	if (min === max) return [{ label: String(min), count: points.length }];
	const width = (max - min) / buckets;
	const counts = Array.from({ length: buckets }, () => 0);
	for (const point of points) {
		const index = Math.min(buckets - 1, Math.floor((point - min) / width));
		counts[index] += 1;
	}
	return counts.map((count, index) => {
		const start = min + index * width;
		return { label: formatTick(start), count };
	});
}

function formatTick(value: number): string {
	if (Math.abs(value) >= 1000) return `${Math.round(value / 100) / 10}k`;
	return String(Math.round(value * 100) / 100);
}

function formatTime(ms: number): string {
	return new Date(ms).toISOString().slice(0, 16).replace('T', ' ');
}

/** Summarise one result column. Counts stay in the browser; nothing is stored. */
export function profileColumn(name: string, type: string, values: unknown[]): ColumnProfile {
	const rows = values.length;
	const nulls = values.filter((value) => value == null).length;
	const present = values.filter((value) => value != null);
	const kind = kindOf(type, present);
	const counts = new Map<string, number>();
	for (const value of present) counts.set(String(value), (counts.get(String(value)) ?? 0) + 1);
	const ranked = [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
	const valueCounts: ValueCount[] = ranked.slice(0, 8).map(([label, count]) => ({
		label,
		count,
		pct: rows === 0 ? 0 : Math.round((count / rows) * 1000) / 10,
		null: false
	}));
	if (nulls > 0) {
		valueCounts.push({
			label: 'null',
			count: nulls,
			pct: rows === 0 ? 0 : Math.round((nulls / rows) * 1000) / 10,
			null: true
		});
	}

	let buckets: HistogramBucket[] = [];
	let min: string | undefined;
	let max: string | undefined;
	if (kind === 'number') {
		const points = present
			.map((value) => asNumber(value))
			.filter((value): value is number => value !== null);
		buckets = histogram(points);
		if (points.length) {
			min = String(Math.min(...points));
			max = String(Math.max(...points));
		}
	} else if (kind === 'time') {
		const points = present
			.map((value) => asTime(value))
			.filter((value): value is number => value !== null);
		buckets = histogram(points);
		if (points.length) {
			min = formatTime(Math.min(...points));
			max = formatTime(Math.max(...points));
		}
	}

	return {
		name,
		type,
		rows,
		nulls,
		nullPct: rows === 0 ? 0 : Math.round((nulls / rows) * 1000) / 10,
		distinct: counts.size,
		kind,
		values: valueCounts,
		buckets,
		min,
		max
	};
}

export function profileColumns(
	columns: { name: string; type: string }[],
	rows: unknown[][]
): ColumnProfile[] {
	return columns.map((column, index) =>
		profileColumn(
			column.name,
			column.type,
			rows.map((row) => row[index])
		)
	);
}
