import { Virtualizer } from '@tanstack/svelte-virtual';
import { describe, expect, it } from 'vitest';

describe('result grid window', () => {
	it('materialises a small window of a 100k-row result', () => {
		const virtualizer = new Virtualizer({
			count: 100_000,
			getScrollElement: () => null,
			estimateSize: () => 28,
			overscan: 8,
			initialRect: { width: 640, height: 280 },
			initialOffset: 50_000 * 28,
			observeElementRect: () => () => {},
			observeElementOffset: () => () => {},
			scrollToFn: () => {}
		});
		const items = virtualizer.getVirtualItems();
		expect(items.length).toBeGreaterThan(0);
		expect(items.length).toBeLessThan(40);
		expect(items[0]?.index).toBeGreaterThan(49_000);
		expect(items[items.length - 1]?.index).toBeLessThan(51_000);
	});
});
