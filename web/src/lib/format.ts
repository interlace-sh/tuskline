/** British-grouped number, for metrics. */
export function fmt(n: number, digits = 0): string {
	return n.toLocaleString('en-GB', {
		maximumFractionDigits: digits,
		minimumFractionDigits: digits
	});
}

/** Decimal byte size (1 000-based), matching the prototype. */
export function fmtBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const units = ['B', 'kB', 'MB', 'GB', 'TB'];
	let index = 0;
	let value = bytes;
	while (value >= 1000 && index < units.length - 1) {
		value /= 1000;
		index += 1;
	}
	const digits = value < 10 && index > 0 ? 1 : 0;
	return `${value.toFixed(digits)} ${units[index]}`;
}

/** Duration for a statement mean or a plan node. */
export function fmtMs(ms: number): string {
	if (ms >= 1000) return `${fmt(ms / 1000, 1)} s`;
	if (ms < 1) return `${fmt(ms * 1000, 0)} µs`;
	return `${fmt(ms, ms < 10 ? 1 : 0)} ms`;
}
