export interface AreaPath {
	key: string;
	area: string;
	line: string;
}

/** Stacked area paths in a width×height view box. Later keys sit on top. */
export function stackedAreas(
	rows: Record<string, number | string>[],
	keys: string[],
	width: number,
	height: number
): AreaPath[] {
	if (rows.length < 2 || keys.length === 0) return [];
	const columns = rows.map((row) => keys.map((key) => Number(row[key]) || 0));
	const max = Math.max(...columns.map((cols) => cols.reduce((sum, value) => sum + value, 0)), 1);
	const x = (index: number) => (index / (rows.length - 1)) * width;
	const y = (value: number) => height - 2 - (value / max) * (height - 4);
	const floor = columns.map(() => 0);
	return keys.map((key, series) => {
		const top = columns.map((cols, index) => floor[index] + cols[series]);
		const line = top
			.map((value, index) => `${index ? 'L' : 'M'}${x(index).toFixed(1)},${y(value).toFixed(1)}`)
			.join('');
		const back = [...floor]
			.reverse()
			.map((value, index) => {
				const at = floor.length - 1 - index;
				return `L${x(at).toFixed(1)},${y(value).toFixed(1)}`;
			})
			.join('');
		top.forEach((value, index) => {
			floor[index] = value;
		});
		return { key, area: `${line}${back}Z`, line };
	});
}

/** One line, scaled to the series or to an explicit domain. */
export function linePath(
	rows: Record<string, number | string>[],
	key: string,
	width: number,
	height: number,
	domain?: [number, number]
): string {
	if (rows.length < 2) return '';
	const values = rows.map((row) => Number(row[key]) || 0);
	const min = domain?.[0] ?? Math.min(...values);
	const max = domain?.[1] ?? Math.max(...values);
	const span = max - min || 1;
	return values
		.map((value, index) => {
			const x = (index / (values.length - 1)) * width;
			const y = height - 2 - ((value - min) / span) * (height - 4);
			return `${index ? 'L' : 'M'}${x.toFixed(1)},${y.toFixed(1)}`;
		})
		.join('');
}
