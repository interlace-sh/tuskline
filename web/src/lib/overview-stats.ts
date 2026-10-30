import type { ColumnProfile } from '#lib/column-profile.js';

export interface OverviewBar {
	label: string;
	weight: number;
}

export interface OverviewColumn {
	name: string;
	nullPct: number | null;
	distinct: number | null;
	/** Frequency bars, or equal-depth histogram buckets. */
	bars: OverviewBar[];
	/** True when bars are an equal-depth histogram, not frequencies. */
	depth: boolean;
}

const IDENT = /^[A-Za-z_][A-Za-z0-9_$]*$/;

function sqlString(value: string): string | null {
	if (value.length === 0 || value.length > 63 || value.includes('\0')) return null;
	if (!IDENT.test(value)) return null;
	return `'${value.replaceAll("'", "''")}'`;
}

/** Read planner statistics for one relation. Null when the name is not a plain identifier. */
export function plannerStatsSQL(schema: string, table: string): string | null {
	const schemaLit = sqlString(schema);
	const tableLit = sqlString(table);
	if (!schemaLit || !tableLit) return null;
	return `SELECT a.attname, s.null_frac, s.n_distinct,
	s.most_common_vals::text, s.most_common_freqs::text, s.histogram_bounds::text,
	c.reltuples::float8
FROM pg_attribute a
JOIN pg_class c ON c.oid = a.attrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_stats s
	ON s.schemaname = n.nspname AND s.tablename = c.relname AND s.attname = a.attname
WHERE n.nspname = ${schemaLit} AND c.relname = ${tableLit}
	AND a.attnum > 0 AND NOT a.attisdropped
ORDER BY a.attnum`;
}

/** Parse a PostgreSQL array literal such as `{1,2}` or `{"a b",c}`. */
export function parsePgArray(text: string | null | undefined): string[] {
	if (!text || text === '{}' || text === 'NULL') return [];
	const trimmed = text.trim();
	if (!trimmed.startsWith('{') || !trimmed.endsWith('}')) return [];
	const body = trimmed.slice(1, -1);
	if (!body) return [];
	const out: string[] = [];
	let index = 0;
	while (index < body.length) {
		if (body[index] === '"') {
			let cursor = index + 1;
			let value = '';
			while (cursor < body.length) {
				if (body[cursor] === '\\') {
					value += body[cursor + 1] ?? '';
					cursor += 2;
					continue;
				}
				if (body[cursor] === '"') {
					cursor += 1;
					break;
				}
				value += body[cursor];
				cursor += 1;
			}
			out.push(value);
			if (body[cursor] === ',') cursor += 1;
			index = cursor;
			continue;
		}
		const comma = body.indexOf(',', index);
		const end = comma === -1 ? body.length : comma;
		const token = body.slice(index, end);
		if (token !== 'NULL') out.push(token);
		index = end + 1;
	}
	return out;
}

function num(value: unknown): number | null {
	if (typeof value === 'number' && Number.isFinite(value)) return value;
	if (typeof value === 'string' && value.trim() !== '') {
		const parsed = Number(value);
		return Number.isFinite(parsed) ? parsed : null;
	}
	return null;
}

function distinctEstimate(nDistinct: number | null, reltuples: number | null): number | null {
	if (nDistinct == null || nDistinct === 0) return null;
	if (nDistinct > 0) return Math.round(nDistinct);
	if (reltuples == null || reltuples <= 0) return null;
	return Math.round(-nDistinct * reltuples);
}

/** Turn `pg_stats` rows into overview columns. Estimates, not a table scan. */
export function statsFromPlanner(rows: unknown[][]): OverviewColumn[] {
	return rows.map((row) => {
		const name = String(row[0] ?? '');
		const nullFrac = num(row[1]);
		const nDistinct = num(row[2]);
		const reltuples = num(row[6]);
		const freqs = parsePgArray(typeof row[4] === 'string' ? row[4] : null)
			.map((item) => Number(item))
			.filter((item) => Number.isFinite(item) && item > 0);
		const values = parsePgArray(typeof row[3] === 'string' ? row[3] : null);
		const bounds = parsePgArray(typeof row[5] === 'string' ? row[5] : null);
		let bars: OverviewBar[] = [];
		let depth = false;
		if (freqs.length) {
			bars = freqs.map((weight, index) => ({
				label: values[index] ?? String(index + 1),
				weight
			}));
		} else if (bounds.length > 1) {
			depth = true;
			bars = bounds.slice(0, -1).map((label, index) => ({
				label: `${label} – ${bounds[index + 1]}`,
				weight: 1
			}));
		}
		return {
			name,
			nullPct: nullFrac == null ? null : Math.round(nullFrac * 1000) / 10,
			distinct: distinctEstimate(nDistinct, reltuples),
			bars,
			depth
		};
	});
}

/** Map a profiled sample onto the same overview shape. */
export function statsFromProfiles(profiles: ColumnProfile[]): OverviewColumn[] {
	return profiles.map((profile) => ({
		name: profile.name,
		nullPct: profile.nullPct,
		distinct: profile.distinct,
		depth: false,
		bars:
			profile.kind === 'category'
				? profile.values
						.filter((value) => !value.null)
						.slice(0, 8)
						.map((value) => ({ label: value.label, weight: value.pct }))
				: profile.buckets.map((bucket) => ({ label: bucket.label, weight: bucket.count }))
	}));
}
