import { describe, expect, it } from 'vitest';
import { partitionExtensions } from './extensions';

describe('partitionExtensions', () => {
	it('splits installed extensions from those only available to install', () => {
		const parts = partitionExtensions([
			{ name: 'plpgsql', installed_version: '1.0', available: true },
			{ name: 'hypopg', available: true },
			{ name: 'orphan', available: false }
		]);
		expect(parts.installed.map((extension) => extension.name)).toEqual(['plpgsql']);
		expect(parts.available.map((extension) => extension.name)).toEqual(['hypopg']);
	});
});
