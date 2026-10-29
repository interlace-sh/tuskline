import type { PlanNode, Query } from '#lib/fixtures/demo.js';

export type QuerySort = 'share' | 'callsPerSec' | 'meanMs' | 'p95Ms';

export function filterQueries(queries: Query[], text: string): Query[] {
	const needle = text.trim().toLowerCase();
	if (!needle) return queries;
	return queries.filter((query) =>
		`${query.sql} ${query.db} ${query.user} ${query.id}`.toLowerCase().includes(needle)
	);
}

export function sortQueries(queries: Query[], key: QuerySort): Query[] {
	return [...queries].sort((a, b) => b[key] - a[key]);
}

export function flattenPlan(
	node: PlanNode,
	depth = 0,
	out: { node: PlanNode; depth: number }[] = []
): { node: PlanNode; depth: number }[] {
	out.push({ node, depth });
	for (const child of node.children ?? []) flattenPlan(child, depth + 1, out);
	return out;
}
