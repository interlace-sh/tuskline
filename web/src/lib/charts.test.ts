import { describe, expect, it } from 'vitest';
import { linePath, stackedAreas } from './charts';

const rows = [
	{ t: '08:00', a: 1, b: 2 },
	{ t: '08:05', a: 2, b: 2 },
	{ t: '08:10', a: 1, b: 3 }
];

describe('charts', () => {
	it('stacks each series into a closed area', () => {
		const areas = stackedAreas(rows, ['a', 'b'], 100, 40);
		expect(areas.map((area) => area.key)).toEqual(['a', 'b']);
		expect(areas[0]?.area.endsWith('Z')).toBe(true);
		expect(areas[1]?.line.startsWith('M')).toBe(true);
	});

	it('draws a line across the view box', () => {
		const path = linePath(rows, 'a', 100, 40);
		expect(path.startsWith('M0.0,')).toBe(true);
		expect(path).toContain('L100.0,');
	});
});
