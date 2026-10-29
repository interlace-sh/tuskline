import { describe, expect, it } from 'vitest';
import { PLAN_BEFORE, QUERIES } from './fixtures/demo';
import { filterQueries, flattenPlan, sortQueries } from './query-view';

describe('query view', () => {
	it('filters on the statement text and sorts by load', () => {
		const matched = filterQueries(QUERIES, 'order_items');
		expect(matched.map((query) => query.id)).toContain('a91f3c');
		expect(sortQueries(QUERIES, 'share')[0]?.id).toBe('a91f3c');
	});

	it('flattens a plan parent before its children', () => {
		const rows = flattenPlan(PLAN_BEFORE);
		expect(rows.map((row) => row.node.op)).toEqual(['Nested Loop', 'Seq Scan', 'Index Scan']);
		expect(rows.map((row) => row.depth)).toEqual([0, 1, 1]);
	});
});
