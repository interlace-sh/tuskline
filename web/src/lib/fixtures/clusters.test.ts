import { describe, expect, it } from 'vitest';
import {
	clusterFromAPI,
	demoForced,
	fixtureClusters,
	formatPgVersion,
	selectClusters
} from './clusters';

describe('fixture clusters', () => {
	it('keeps the prototype fleet, with the first database as the shell default', () => {
		const clusters = fixtureClusters();
		expect(clusters).toHaveLength(10);
		expect(clusters.find((cluster) => cluster.id === 'prod-eu-west')).toMatchObject({
			env: 'prod',
			health: 'warn',
			database: 'orders',
			version: '17.2'
		});
	});

	it('formats server_version_num and leaves an unprobed cluster unknown', () => {
		expect(formatPgVersion(170002)).toBe('17.2');
		expect(formatPgVersion(160006)).toBe('16.6');
		expect(formatPgVersion(0)).toBe('unknown');
		expect(
			clusterFromAPI({
				id: 'dev/pg17',
				environment: 'dev',
				name: 'pg17',
				kind: 'dev',
				provider: 'local',
				host: '127.0.0.1',
				port: 5417,
				database: 'postgres'
			})
		).toMatchObject({
			id: 'dev/pg17',
			env: 'dev',
			health: 'unknown',
			version: 'unknown',
			ha: 'local',
			region: '127.0.0.1:5417',
			database: 'postgres'
		});
	});

	it('uses fixtures when the inventory is empty or demo is forced', () => {
		const live = fixtureClusters().slice(0, 1);
		expect(selectClusters([], false).demo).toBe(true);
		expect(selectClusters([], false).clusters).toHaveLength(10);
		expect(selectClusters(live, true).demo).toBe(true);
		expect(selectClusters(live, false)).toEqual({ demo: false, clusters: live });
		expect(demoForced('?demo=1')).toBe(true);
		expect(demoForced('')).toBe(false);
	});
});
