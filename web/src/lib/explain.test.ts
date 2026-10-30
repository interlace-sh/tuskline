import { describe, expect, it } from 'vitest';
import { analyzeNeedsConfirm, explainSQL, parseExplain, writeVerb } from './explain.js';

const TEXT = `Limit  (cost=0.56..412.80 rows=100 width=40) (actual time=0.031..0.912 rows=100 loops=1)
  ->  Index Scan Backward using idx_orders_created_at on orders o  (cost=0.42..400.00 rows=100 width=40) (actual time=0.030..0.874 rows=100 loops=1)
        Index Cond: (created_at > (now() - '00:15:00'::interval))
        Buffers: shared hit=212
Planning Time: 0.188 ms
Execution Time: 0.951 ms`;

const JSON_PLAN = JSON.stringify([
	{
		Plan: {
			'Node Type': 'Seq Scan',
			'Relation Name': 'order_items',
			Alias: 'oi',
			'Startup Cost': 0,
			'Total Cost': 100,
			'Plan Rows': 1,
			'Plan Width': 8,
			'Actual Total Time': 171.4,
			'Actual Rows': 412,
			'Actual Loops': 1,
			'Shared Hit Blocks': 3220,
			'Shared Read Blocks': 1612004,
			Filter: '(order_id = $1)'
		},
		'Planning Time': 0.4,
		'Execution Time': 181.9
	}
]);

describe('parseExplain', () => {
	it('reads a text plan, its buffers and a plain-English summary', () => {
		const plan = parseExplain(TEXT);
		expect(plan?.root.op).toBe('Limit');
		expect(plan?.root.children[0]?.op).toBe('Index Scan Backward');
		expect(plan?.root.children[0]?.idx).toBe('idx_orders_created_at');
		expect(plan?.root.children[0]?.rel).toBe('orders o');
		expect(plan?.root.children[0]?.bufHit).toBe(212);
		expect(plan?.executionMs).toBe(0.951);
		expect(plan?.summary).toContain('100 rows');
		expect(plan?.summary).toContain('Index Scan Backward');
		expect(plan?.warnings).toEqual([]);
	});

	it('warns on a sequential scan and a row misestimate', () => {
		const plan = parseExplain(JSON_PLAN);
		expect(plan?.root.op).toBe('Seq Scan');
		expect(plan?.root.rel).toBe('order_items oi');
		expect(plan?.warnings.map((warning) => warning.tone)).toEqual(['bad', 'warn']);
		expect(plan?.summary).toContain('Seq Scan');
	});

	it('returns null for an empty string', () => {
		expect(parseExplain('   ')).toBeNull();
	});
});

describe('explainSQL', () => {
	it('wraps a statement once, in JSON', () => {
		expect(explainSQL('SELECT 1;')).toBe('EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) SELECT 1');
		expect(explainSQL('EXPLAIN SELECT 1')).toBe('EXPLAIN SELECT 1');
	});

	it('asks for confirmation only when a production write would be analysed', () => {
		expect(writeVerb('UPDATE t SET a = 1')).toBe('UPDATE');
		expect(writeVerb('SELECT 1')).toBeNull();
		expect(analyzeNeedsConfirm('UPDATE t SET a = 1', true)).toBe(true);
		expect(analyzeNeedsConfirm('SELECT 1', true)).toBe(false);
		expect(analyzeNeedsConfirm('UPDATE t SET a = 1', false)).toBe(false);
	});
});
