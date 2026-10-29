// Shell-shaped clusters: prototype fixtures, or a live inventory row.
import type { Cluster as ApiCluster } from '#lib/api.js';
import type { EnvKind } from '#lib/env.js';
import { CLUSTERS, type Health as FixtureHealth } from './demo.js';

export type Health = FixtureHealth | 'unknown';

export interface ShellCluster {
	id: string;
	name: string;
	env: EnvKind;
	health: Health;
	version: string;
	ha: string;
	region: string;
	database: string;
}

const KINDS: readonly EnvKind[] = ['local', 'dev', 'staging', 'prod'];

export function isEnvKind(value: string): value is EnvKind {
	return (KINDS as readonly string[]).includes(value);
}

/** PostgreSQL server_version_num is major * 10000 + minor. */
export function formatPgVersion(versionNum: number): string {
	if (!Number.isFinite(versionNum) || versionNum <= 0) return 'unknown';
	const major = Math.floor(versionNum / 10000);
	const minor = versionNum % 10000;
	return `${major}.${minor}`;
}

export function fixtureClusters(): ShellCluster[] {
	return CLUSTERS.map((cluster) => ({
		id: cluster.id,
		name: cluster.name,
		env: cluster.env,
		health: cluster.health,
		version: cluster.version,
		ha: cluster.ha,
		region: cluster.region,
		database: cluster.databases[0] ?? ''
	}));
}

/** A live inventory row has no health score until a screen asks for /health. */
export function clusterFromAPI(cluster: ApiCluster): ShellCluster {
	const env = isEnvKind(cluster.kind) ? cluster.kind : 'local';
	const host = cluster.host ?? '';
	const region = host && cluster.port ? `${host}:${cluster.port}` : host;
	return {
		id: cluster.id,
		name: cluster.name,
		env,
		health: 'unknown',
		version: formatPgVersion(cluster.last_probe?.version_num ?? 0),
		ha: cluster.provider || 'unknown',
		region,
		database: cluster.database ?? ''
	};
}

/** Empty inventory, a failed fetch, or ?demo=1 keeps the prototype fixtures. */
export function selectClusters(
	live: ShellCluster[],
	forceDemo: boolean
): { demo: boolean; clusters: ShellCluster[] } {
	if (forceDemo || live.length === 0) return { demo: true, clusters: fixtureClusters() };
	return { demo: false, clusters: live };
}

export function demoForced(search: string): boolean {
	return new URLSearchParams(search).get('demo') === '1';
}

export function healthDotClass(health: Health): string {
	if (health === 'warn') return 'bg-env-staging';
	if (health === 'crit') return 'bg-env-prod';
	if (health === 'unknown') return 'bg-tl-fg3';
	return 'bg-env-dev';
}
