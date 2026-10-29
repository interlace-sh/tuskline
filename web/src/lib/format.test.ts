import { describe, expect, it } from 'vitest';
import { fmtMs } from './format';

describe('fmtMs', () => {
	it('picks microseconds, milliseconds or seconds', () => {
		expect(fmtMs(0.4)).toBe('400 µs');
		expect(fmtMs(9.8)).toBe('9.8 ms');
		expect(fmtMs(182.4)).toBe('182 ms');
		expect(fmtMs(7420)).toBe('7.4 s');
	});
});
