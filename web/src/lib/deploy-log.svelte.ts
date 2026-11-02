export interface RecordedDeployment {
	name: string;
	target: string;
	env: string;
	when: string;
}

let items = $state<RecordedDeployment[]>([]);

/** Deployments recorded in this browser session. Demo history, not the inventory. */
export function recordedDeployments(): RecordedDeployment[] {
	return items;
}

export function recordDeployment(item: RecordedDeployment) {
	items = [item, ...items.filter((existing) => existing.name !== item.name)];
}
