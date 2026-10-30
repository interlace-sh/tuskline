import { describe, expect, it } from 'vitest';
import { parsePgArray, plannerStatsSQL, statsFromPlanner } from './overview-stats';

describe('plannerStatsSQL', () => {
	it('quotes plain names and refuses anything else', () => {
		const sql = plannerStatsSQL('public', 'orders');
		expect(sql).toContain(`n.nspname = 'public'`);
		expect(sql).toContain(`c.relname = 'orders'`);
		expect(plannerStatsSQL('public', 'orders;drop')).toBeNull();
		expect(plannerStatsSQL('bad name', 'orders')).toBeNull();
	});
});

describe('parsePgArray', () => {
	it('reads numbers and quoted text', () => {
		expect(parsePgArray('{0.4,0.1}')).toEqual(['0.4', '0.1']);
		expect(parsePgArray('{"paid","in transit"}')).toEqual(['paid', 'in transit']);
		expect(parsePgArray('{}')).toEqual([]);
	});
});

describe('statsFromPlanner', () => {
	it('turns null fraction and a negative n_distinct into estimates', () => {
		const [column] = statsFromPlanner([
			['status', 0.2, -0.5, '{paid,shipped}', '{0.6,0.4}', null, 1000]
		]);
		expect(column.nullPct).toBe(20);
		expect(column.distinct).toBe(500);
		expect(column.depth).toBe(false);
		expect(column.bars.map((bar) => bar.label)).toEqual(['paid', 'shipped']);
	});

	it('marks histogram bounds as equal-depth', () => {
		const [column] = statsFromPlanner([['total', 0, 12, null, null, '{1,5,9}', 12]]);
		expect(column.depth).toBe(true);
		expect(column.bars[0]?.label).toBe('1 – 5');
	});
});
