import { fmt, fmtMs } from '#lib/format.js';

export interface ExplainNode {
	id: string;
	op: string;
	rel?: string;
	idx?: string;
	detail: string;
	/** Inclusive time, actual total × loops. Zero when the plan was not analysed. */
	totalMs: number;
	selfMs: number;
	estRows: number;
	actRows: number;
	loops: number;
	bufRead: number;
	bufHit: number;
	children: ExplainNode[];
}

export interface ExplainWarning {
	tone: 'warn' | 'bad';
	text: string;
}

export interface ExplainPlan {
	root: ExplainNode;
	planningMs?: number;
	executionMs?: number;
	summary: string;
	warnings: ExplainWarning[];
}

const WRITE = /^\s*(?:--[^\n]*\n\s*)*(update|delete|insert|alter|drop|truncate|create)\b/i;

/** Leading mutating verb, ignoring a comment line. */
export function writeVerb(sql: string): string | null {
	const match = WRITE.exec(sql);
	return match ? match[1].toUpperCase() : null;
}

/** Production must confirm before ANALYZE runs a write. The session stays read-only. */
export function analyzeNeedsConfirm(sql: string, readOnly: boolean): boolean {
	return readOnly && writeVerb(sql) !== null;
}

/** One EXPLAIN statement. An existing EXPLAIN is left alone. */
export function explainSQL(sql: string): string {
	const body = sql.trim().replace(/;\s*$/, '');
	if (/^explain\b/i.test(body)) return body;
	return `EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) ${body}`;
}

interface RawNode {
	indent: number;
	op: string;
	rel?: string;
	idx?: string;
	totalMs: number;
	estRows: number;
	actRows: number;
	loops: number;
	bufRead: number;
	bufHit: number;
	props: string[];
	children: RawNode[];
}

const WITH_BOTH =
	/^( *)(?:->\s+)?(.+?)\s+\(cost=([\d.]+)\.\.([\d.]+) rows=(\d+)(?: width=\d+)?\)\s+\(actual time=([\d.]+)\.\.([\d.]+) rows=(\d+) loops=(\d+)\)\s*$/;
const COST_ONLY =
	/^( *)(?:->\s+)?(.+?)\s+\(cost=([\d.]+)\.\.([\d.]+) rows=(\d+)(?: width=\d+)?\)\s*$/;
const ACTUAL_ONLY =
	/^( *)(?:->\s+)?(.+?)\s+\(actual time=([\d.]+)\.\.([\d.]+) rows=(\d+) loops=(\d+)\)\s*$/;

/** Parse EXPLAIN text or FORMAT JSON. Returns null when the text is not a plan. */
export function parseExplain(text: string): ExplainPlan | null {
	const trimmed = text.trim();
	if (!trimmed) return null;
	if (trimmed.startsWith('[') || trimmed.startsWith('{')) {
		try {
			return finish(fromJSON(JSON.parse(trimmed) as unknown));
		} catch {
			return null;
		}
	}
	return finish(fromText(trimmed));
}

function finish(
	raw: { root: RawNode; planningMs?: number; executionMs?: number } | null
): ExplainPlan | null {
	if (!raw) return null;
	const root = seal(raw.root, { n: 0 });
	exclusive(root);
	const plan: ExplainPlan = {
		root,
		planningMs: raw.planningMs,
		executionMs: raw.executionMs,
		summary: '',
		warnings: []
	};
	plan.warnings = collectWarnings(plan);
	plan.summary = summarise(plan);
	return plan;
}

function seal(raw: RawNode, seq: { n: number }): ExplainNode {
	seq.n += 1;
	return {
		id: `n${seq.n}`,
		op: raw.op,
		rel: raw.rel,
		idx: raw.idx,
		detail: raw.props.join('\n'),
		totalMs: raw.totalMs,
		selfMs: raw.totalMs,
		estRows: raw.estRows,
		actRows: raw.actRows,
		loops: raw.loops,
		bufRead: raw.bufRead,
		bufHit: raw.bufHit,
		children: raw.children.map((child) => seal(child, seq))
	};
}

function exclusive(node: ExplainNode) {
	for (const child of node.children) exclusive(child);
	const below = node.children.reduce((sum, child) => sum + child.totalMs, 0);
	node.selfMs = Math.max(0, round(node.totalMs - below));
}

function fromText(
	text: string
): { root: RawNode; planningMs?: number; executionMs?: number } | null {
	let planningMs: number | undefined;
	let executionMs: number | undefined;
	let root: RawNode | undefined;
	const stack: RawNode[] = [];
	for (const line of text.split('\n')) {
		if (!line.trim()) continue;
		const timing = /^(Planning|Execution) Time:\s+([\d.]+)\s*ms/i.exec(line.trim());
		if (timing) {
			const ms = Number(timing[2]);
			if (timing[1].toLowerCase() === 'planning') planningMs = ms;
			else executionMs = ms;
			continue;
		}
		const header = matchNode(line);
		if (header) {
			const named = splitOp(header.op);
			const node: RawNode = {
				indent: header.indent,
				op: named.op,
				rel: named.rel,
				idx: named.idx,
				totalMs: round(header.actual * header.loops),
				estRows: header.estRows,
				actRows: header.actRows,
				loops: header.loops,
				bufRead: 0,
				bufHit: 0,
				props: [],
				children: []
			};
			while (stack.length && stack[stack.length - 1].indent >= header.indent) stack.pop();
			const parent = stack[stack.length - 1];
			if (parent) parent.children.push(node);
			else root = node;
			stack.push(node);
			continue;
		}
		const current = stack[stack.length - 1];
		if (!current) continue;
		const prop = line.trim();
		current.props.push(prop);
		const buffers = /Buffers:/.test(prop) ? prop : '';
		const hit = /shared hit=(\d+)/.exec(buffers);
		const read = /(?:shared |local )?read=(\d+)/.exec(buffers);
		if (hit) current.bufHit += Number(hit[1]);
		if (read) current.bufRead += Number(read[1]);
	}
	if (!root) return null;
	return { root, planningMs, executionMs };
}

function matchNode(line: string): {
	indent: number;
	op: string;
	estRows: number;
	actual: number;
	actRows: number;
	loops: number;
} | null {
	const both = WITH_BOTH.exec(line);
	if (both) {
		return {
			indent: both[1].length,
			op: both[2],
			estRows: Number(both[5]),
			actual: Number(both[7]),
			actRows: Number(both[8]),
			loops: Number(both[9])
		};
	}
	const actual = ACTUAL_ONLY.exec(line);
	if (actual) {
		return {
			indent: actual[1].length,
			op: actual[2],
			estRows: 0,
			actual: Number(actual[4]),
			actRows: Number(actual[5]),
			loops: Number(actual[6])
		};
	}
	const cost = COST_ONLY.exec(line);
	if (cost) {
		return {
			indent: cost[1].length,
			op: cost[2],
			estRows: Number(cost[5]),
			actual: 0,
			actRows: 0,
			loops: 1
		};
	}
	return null;
}

function splitOp(op: string): { op: string; rel?: string; idx?: string } {
	const using = / using (\S+)/.exec(op);
	const on = / on (.+)$/.exec(op);
	let name = op;
	if (using) name = name.replace(/ using \S+/, '');
	if (on) name = name.replace(/ on .+$/, '');
	return { op: name.trim(), rel: on?.[1].trim(), idx: using?.[1] };
}

interface JsonNode {
	'Node Type'?: string;
	'Relation Name'?: string;
	Alias?: string;
	'Index Name'?: string;
	'Plan Rows'?: number;
	'Actual Total Time'?: number;
	'Actual Rows'?: number;
	'Actual Loops'?: number;
	'Shared Hit Blocks'?: number;
	'Shared Read Blocks'?: number;
	Plans?: JsonNode[];
	[key: string]: unknown;
}

function fromJSON(
	value: unknown
): { root: RawNode; planningMs?: number; executionMs?: number } | null {
	const list = Array.isArray(value) ? value : [value];
	const first = list[0];
	if (!first || typeof first !== 'object') return null;
	const record = first as { Plan?: JsonNode; 'Planning Time'?: number; 'Execution Time'?: number };
	if (!record.Plan) return null;
	return {
		root: jsonNode(record.Plan, 0),
		planningMs: typeof record['Planning Time'] === 'number' ? record['Planning Time'] : undefined,
		executionMs: typeof record['Execution Time'] === 'number' ? record['Execution Time'] : undefined
	};
}

const DETAIL_KEYS = [
	'Filter',
	'Index Cond',
	'Recheck Cond',
	'Hash Cond',
	'Join Filter',
	'Merge Cond',
	'Sort Key',
	'Group Key',
	'One-Time Filter'
];

function jsonNode(node: JsonNode, indent: number): RawNode {
	const loops = node['Actual Loops'] ?? 1;
	const actual = node['Actual Total Time'] ?? 0;
	const rel = [node['Relation Name'], node.Alias].filter(Boolean).join(' ');
	const props = DETAIL_KEYS.flatMap((key) => {
		const value = node[key];
		if (value == null) return [];
		const shown = Array.isArray(value) ? value.join(', ') : String(value);
		return [`${key}: ${shown}`];
	});
	return {
		indent,
		op: node['Node Type'] ?? 'Plan',
		rel: rel || undefined,
		idx: node['Index Name'],
		totalMs: round(actual * loops),
		estRows: node['Plan Rows'] ?? 0,
		actRows: node['Actual Rows'] ?? 0,
		loops,
		bufRead: node['Shared Read Blocks'] ?? 0,
		bufHit: node['Shared Hit Blocks'] ?? 0,
		props,
		children: (node.Plans ?? []).map((child) => jsonNode(child, indent + 2))
	};
}

function collectWarnings(plan: ExplainPlan): ExplainWarning[] {
	const warnings: ExplainWarning[] = [];
	const walk = (node: ExplainNode) => {
		const share = plan.root.totalMs > 0 ? node.selfMs / plan.root.totalMs : 0;
		if (/seq scan/i.test(node.op) && (share > 0.5 || node.bufRead > 1000)) {
			const where = node.rel ? ` of ${node.rel}` : '';
			const time = plan.root.totalMs > 0 ? ` takes ${Math.round(share * 100)}% of the time` : '';
			warnings.push({ tone: 'bad', text: `Sequential scan${where}${time}.` });
		}
		if (node.estRows > 0 && node.actRows > 0) {
			const ratio = Math.max(node.actRows / node.estRows, node.estRows / node.actRows);
			if (ratio > 10) {
				warnings.push({
					tone: 'warn',
					text: `${node.op} expected ${fmt(node.estRows)} rows and returned ${fmt(node.actRows)} (×${Math.round(ratio)}).`
				});
			}
		}
		if (/external merge|Disk:/i.test(node.detail)) {
			warnings.push({ tone: 'warn', text: `${node.op} spilled to disk.` });
		}
		for (const child of node.children) walk(child);
	};
	walk(plan.root);
	return warnings;
}

function summarise(plan: ExplainPlan): string {
	const rows = plan.root.actRows || plan.root.estRows;
	const ran = plan.executionMs != null || plan.root.totalMs > 0;
	const time = plan.executionMs ?? plan.root.totalMs;
	const verb = ran ? 'returned' : 'would read about';
	let sentence = `The statement ${verb} ${fmt(rows)} rows`;
	if (time > 0) sentence += ` in ${fmtMs(time)}`;
	sentence += '.';
	const hot = hottest(plan.root);
	if (hot && hot.selfMs > 0 && hot !== plan.root) {
		const target = hot.rel ? ` on ${hot.rel}` : '';
		sentence += ` Most of that time is ${hot.op}${target} (${fmtMs(hot.selfMs)}).`;
	} else if (hot && hot.selfMs > 0) {
		const target = hot.rel ? ` on ${hot.rel}` : '';
		sentence += ` That time is ${hot.op}${target}.`;
	}
	return sentence;
}

function hottest(node: ExplainNode): ExplainNode | undefined {
	let best: ExplainNode | undefined;
	const walk = (current: ExplainNode) => {
		if (!best || current.selfMs > best.selfMs) best = current;
		for (const child of current.children) walk(child);
	};
	walk(node);
	return best;
}

function round(value: number): number {
	return Math.round(value * 1000) / 1000;
}
