export interface SqlFileNode {
	kind: 'file';
	id: string;
	name: string;
}

export interface SqlFolderNode {
	kind: 'folder';
	name: string;
	children: SqlNode[];
}

export type SqlNode = SqlFileNode | SqlFolderNode;

/** Build a folder tree from slash-separated paths. Folders sort before files. */
export function treeFromPaths(entries: { id: string; path: string }[]): SqlNode[] {
	const root: SqlFolderNode = { kind: 'folder', name: '', children: [] };
	for (const entry of entries) {
		const parts = entry.path.split('/').filter(Boolean);
		let cursor = root;
		parts.forEach((part, index) => {
			const last = index === parts.length - 1;
			if (last) {
				cursor.children.push({ kind: 'file', id: entry.id, name: part });
				return;
			}
			let next = cursor.children.find(
				(child): child is SqlFolderNode => child.kind === 'folder' && child.name === part
			);
			if (!next) {
				next = { kind: 'folder', name: part, children: [] };
				cursor.children.push(next);
			}
			cursor = next;
		});
	}
	const sort = (nodes: SqlNode[]) => {
		nodes.sort((a, b) => {
			if (a.kind !== b.kind) return a.kind === 'folder' ? -1 : 1;
			return a.name.localeCompare(b.name);
		});
		for (const node of nodes) if (node.kind === 'folder') sort(node.children);
	};
	sort(root.children);
	return root.children;
}

type FileHandle = { getFile: () => Promise<File> };
type DirHandle = {
	name: string;
	entries: () => AsyncIterable<[string, { kind: 'file' | 'directory' } & FileHandle & DirHandle]>;
};

const fileHandles = new Map<string, FileHandle>();
let nextFile = 0;

export function canPickFolder(): boolean {
	return typeof window !== 'undefined' && 'showDirectoryPicker' in window;
}

export async function readSqlFile(id: string): Promise<string | null> {
	const handle = fileHandles.get(id);
	if (!handle) return null;
	const file = await handle.getFile();
	return file.text();
}

export type PickFolder =
	{ ok: true; name: string; tree: SqlNode[] } | { ok: false; reason: 'unsupported' | 'cancelled' };

/** Ask for a local folder and keep handles for its .sql and .sqlnb files. */
export async function pickSqlFolder(): Promise<PickFolder> {
	const picker = (
		window as Window & {
			showDirectoryPicker?: (options?: { mode?: 'read' }) => Promise<DirHandle>;
		}
	).showDirectoryPicker;
	if (!picker) return { ok: false, reason: 'unsupported' };
	let dir: DirHandle;
	try {
		dir = await picker({ mode: 'read' });
	} catch (err) {
		if (err instanceof DOMException && err.name === 'AbortError') {
			return { ok: false, reason: 'cancelled' };
		}
		throw err;
	}
	const found: { id: string; path: string }[] = [];
	await walkSql(dir, '', found, { n: 0 });
	return { ok: true, name: dir.name, tree: treeFromPaths(found) };
}

async function walkSql(
	dir: DirHandle,
	prefix: string,
	found: { id: string; path: string }[],
	limit: { n: number }
) {
	for await (const [name, handle] of dir.entries()) {
		if (limit.n >= 400) return;
		if (handle.kind === 'directory') {
			await walkSql(handle, `${prefix}${name}/`, found, limit);
			continue;
		}
		if (!/\.(sql|sqlnb)$/i.test(name)) continue;
		limit.n += 1;
		const id = `sql-file-${++nextFile}`;
		fileHandles.set(id, handle);
		found.push({ id, path: `${prefix}${name}` });
	}
}
