import { describe, expect, it } from 'vitest';
import { CLUSTERS } from './demo';
import { filterFleet, sortByScore } from './fleet';

describe('fleet filter', () => {
	it('hides healthy clusters when only issues are requested', () => {
		const issues = filterFleet(CLUSTERS, '', 'issues');
		expect(issues.every((cluster) => cluster.health !== 'ok')).toBe(true);
		expect(issues.map((cluster) => cluster.id)).toContain('prod-eu-central');
	});

	it('matches a region and an engine together', () => {
		expect(filterFleet(CLUSTERS, 'eu-west patroni', 'all').map((cluster) => cluster.id)).toEqual([
			'prod-eu-west',
			'staging-eu-west'
		]);
	});

	it('sorts a lane so the lowest score is first', () => {
		const prod = sortByScore(CLUSTERS.filter((cluster) => cluster.env === 'prod'));
		expect(prod[0]?.id).toBe('prod-eu-central');
	});
});
