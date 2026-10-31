import { describe, expect, it } from 'vitest';
import { flyoutAnchor, flyoutBox } from './flyout';

describe('flyoutBox', () => {
	it('shifts a panel up so it is not cut off below the window', () => {
		const box = flyoutBox(1022, 329, 980);
		expect(box.top + Math.min(329, box.maxHeight)).toBeLessThanOrEqual(980 - 8);
		expect(box.top).toBeGreaterThanOrEqual(8);
	});

	it('clips a panel that is taller than the window', () => {
		const box = flyoutBox(800, 1200, 900);
		expect(box.top).toBe(8);
		expect(box.top + box.maxHeight).toBeLessThanOrEqual(900 - 8);
	});
});

describe('flyoutAnchor', () => {
	it('flips to the left when the panel would leave the window', () => {
		const anchor = flyoutAnchor({ top: 40, left: 700, right: 900 }, 1000, 256);
		expect(anchor.left + 256).toBeLessThanOrEqual(1000 - 8);
		expect(anchor.left).toBe(700 - 256 - 8);
	});
});
