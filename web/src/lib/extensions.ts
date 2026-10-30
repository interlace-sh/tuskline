export interface ServerExtension {
	name: string;
	installed_version?: string;
	available: boolean;
}

export interface ServerProbe {
	ref: string;
	version: string;
	version_num: number;
	database: string;
	read_only: boolean;
	preload_known: boolean;
	shared_preload_libraries: string[];
	extensions: ServerExtension[];
}

/** Installed in this database, then available on the server but not installed. */
export function partitionExtensions(extensions: ServerExtension[]): {
	installed: ServerExtension[];
	available: ServerExtension[];
} {
	const installed = extensions.filter((extension) => extension.installed_version);
	const available = extensions.filter(
		(extension) => !extension.installed_version && extension.available
	);
	return { installed, available };
}

/** Shown only while the shell is on prototype fixtures. */
export const DEMO_PROBE: ServerProbe = {
	ref: 'demo',
	version: 'PostgreSQL 17.2',
	version_num: 170002,
	database: 'orders',
	read_only: true,
	preload_known: true,
	shared_preload_libraries: ['pg_stat_statements'],
	extensions: [
		{ name: 'pg_stat_statements', installed_version: '1.11', available: true },
		{ name: 'plpgsql', installed_version: '1.0', available: true },
		{ name: 'hypopg', available: true },
		{ name: 'pg_trgm', available: true },
		{ name: 'vector', available: true }
	]
};
