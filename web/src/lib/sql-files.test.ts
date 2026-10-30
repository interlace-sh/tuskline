import { describe, expect, it } from 'vitest';
import { treeFromPaths } from './sql-files';

describe('treeFromPaths', () => {
	it('nests files under folders and sorts folders first', () => {
		const tree = treeFromPaths([
			{ id: 'b', path: 'queries/locks.sql' },
			{ id: 'a', path: 'readme.sql' },
			{ id: 'c', path: 'queries/daily/orders.sqlnb' }
		]);
		expect(tree.map((node) => node.name)).toEqual(['queries', 'readme.sql']);
		const queries = tree[0];
		expect(queries?.kind).toBe('folder');
		if (queries?.kind !== 'folder') return;
		expect(queries.children.map((node) => node.name)).toEqual(['daily', 'locks.sql']);
	});
});
