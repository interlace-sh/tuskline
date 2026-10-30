import { describe, expect, it } from 'vitest';
import { fingerprint, readProfileCache, writeProfileCache } from './column-profile-cache';
import { profileColumn } from './column-profile';

describe('column profile', () => {
	it('counts nulls and the most common text values', () => {
		const profile = profileColumn('status', 'text', ['paid', 'paid', null, 'pending', 'paid']);
		expect(profile.kind).toBe('category');
		expect(profile.nulls).toBe(1);
		expect(profile.nullPct).toBe(20);
		expect(profile.distinct).toBe(2);
		expect(profile.values[0]).toMatchObject({ label: 'paid', count: 3 });
		expect(profile.values.find((value) => value.null)?.count).toBe(1);
	});

	it('builds a numeric range from the values', () => {
		const profile = profileColumn('total', 'numeric', [1, 2, 3, null, 9]);
		expect(profile.kind).toBe('number');
		expect(profile.min).toBe('1');
		expect(profile.max).toBe('9');
		expect(profile.buckets.length).toBeGreaterThan(1);
		expect(profile.nullPct).toBe(20);
	});

	it('reuses a cached summary for the same rows', () => {
		const columns = [{ name: 'status', type: 'text' }];
		const rows = [['paid'], ['pending']];
		const key = fingerprint(columns, rows);
		expect(fingerprint(columns, rows)).toBe(key);
		expect(fingerprint(columns, [['paid'], [null]])).not.toBe(key);
		const profiles = [profileColumn('status', 'text', ['paid', 'pending'])];
		writeProfileCache(key, profiles);
		expect(readProfileCache(key)).toBe(profiles);
	});

	it('treats timestamps as a time range', () => {
		const profile = profileColumn('created_at', 'timestamptz', [
			'2026-10-27 11:41:00+00',
			'2026-10-27 11:42:00+00'
		]);
		expect(profile.kind).toBe('time');
		expect(profile.min?.startsWith('2026-10-27')).toBe(true);
		expect(profile.max?.startsWith('2026-10-27')).toBe(true);
	});
});
