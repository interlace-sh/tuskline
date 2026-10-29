import type { Cluster } from './demo.js';

export type FleetFilter = 'all' | 'issues';

/** Keep clusters whose name, place, engine or database matches every part of the query. */
export function filterFleet(clusters: Cluster[], query: string, filter: FleetFilter): Cluster[] {
	const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
	return clusters.filter((cluster) => {
		if (filter === 'issues' && cluster.health === 'ok') return false;
		const hay =
			`${cluster.name} ${cluster.region} ${cluster.ha} ${cluster.provider} ${cluster.databases.join(' ')}`.toLowerCase();
		return terms.every((term) => hay.includes(term));
	});
}

/** Worse scores first, so a critical cluster leads its lane. */
export function sortByScore(clusters: Cluster[]): Cluster[] {
	return [...clusters].sort((a, b) => a.score - b.score);
}
