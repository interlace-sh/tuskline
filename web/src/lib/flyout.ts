const margin = 8;

/** Place a fixed panel beside a row, flipping to the left when it would leave the window. */
export function flyoutAnchor(
	rect: { top: number; left: number; right: number },
	viewportWidth: number,
	width = 256
): { top: number; left: number } {
	let left = rect.right + margin;
	if (left + width > viewportWidth - margin) {
		left = Math.max(margin, rect.left - width - margin);
	}
	return { top: Math.max(margin, rect.top), left };
}

/**
 * Shift a panel up until its bottom sits inside the viewport. When the
 * content is taller than the window, maxHeight clips it so it can scroll.
 */
export function flyoutBox(
	top: number,
	height: number,
	viewportHeight: number
): { top: number; maxHeight: number } {
	const limit = viewportHeight - margin;
	const overflow = top + height - limit;
	const nextTop = overflow > 0 ? Math.max(margin, top - overflow) : Math.max(margin, top);
	return { top: nextTop, maxHeight: Math.max(120, viewportHeight - nextTop - margin) };
}

/** Keep a fixed flyout inside the window after it has been rendered. */
export function fitFlyout(node: HTMLElement) {
	const place = () => {
		const rect = node.getBoundingClientRect();
		const box = flyoutBox(rect.top, rect.height, window.innerHeight);
		node.style.top = `${box.top}px`;
		node.style.maxHeight = `${box.maxHeight}px`;
	};
	place();
	window.addEventListener('resize', place);
	return {
		destroy() {
			window.removeEventListener('resize', place);
		}
	};
}
