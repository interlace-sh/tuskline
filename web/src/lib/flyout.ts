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

/** Position the panel from the row, not from wherever it was last shifted. */
export function applyFlyout(
	node: HTMLElement,
	anchor: { top: number; left: number },
	viewportHeight: number
) {
	node.style.left = `${anchor.left}px`;
	node.style.maxHeight = 'none';
	node.style.top = `${anchor.top}px`;
	const box = flyoutBox(anchor.top, node.scrollHeight, viewportHeight);
	node.style.top = `${box.top}px`;
	node.style.maxHeight = `${box.maxHeight}px`;
}

/**
 * Hang the panel on document.body and keep it inside the window.
 * The SQL card is overflow:hidden with a radius, which becomes the
 * containing block for position:fixed and clips the panel.
 * Passing the anchor again recalculates when another row is chosen
 * without closing the panel first.
 */
export function mountFlyout(node: HTMLElement, anchor: { top: number; left: number }) {
	document.body.appendChild(node);
	let origin = anchor;
	const place = () => applyFlyout(node, origin, window.innerHeight);
	let frame = 0;
	place();
	const onResize = () => place();
	window.addEventListener('resize', onResize);
	return {
		update(next: { top: number; left: number }) {
			origin = next;
			place();
			cancelAnimationFrame(frame);
			frame = requestAnimationFrame(place);
		},
		destroy() {
			cancelAnimationFrame(frame);
			window.removeEventListener('resize', onResize);
			node.remove();
		}
	};
}
