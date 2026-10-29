// App shell state. Demo clusters stand in until M3.2 moves the prototype fixtures.
import { defaultReadOnly, ENV_LABEL, type EnvKind } from '#lib/env.js';

export type Screen =
	| 'fleet'
	| 'cluster'
	| 'queries'
	| 'sql'
	| 'ha'
	| 'backups'
	| 'migrations'
	| 'advisor'
	| 'setup'
	| 'connect'
	| 'deploy'
	| 'deployments';

export type Density = 'comfortable' | 'compact';
export type Theme = 'dark' | 'light';
export type Health = 'ok' | 'warn' | 'crit';

export interface DemoCluster {
	id: string;
	name: string;
	env: EnvKind;
	health: Health;
	version: string;
	ha: string;
	region: string;
	database: string;
}

export const RANGES = ['15m', '1h', '6h', '24h', '7d', '30d'] as const;
export type Range = (typeof RANGES)[number];

/** Screens that are not inside one environment, so the rail shows every colour. */
export const GLOBAL_SCREENS: Screen[] = ['fleet', 'setup', 'connect', 'deploy', 'deployments'];

export const SCREEN_LABEL: Record<Screen, string> = {
	fleet: 'Fleet',
	cluster: 'Health',
	queries: 'Query insights',
	sql: 'SQL editor',
	ha: 'HA topology',
	backups: 'Backups',
	migrations: 'Migrations',
	advisor: 'Config advisor',
	setup: 'Overview',
	connect: 'Add connection',
	deploy: 'Deploy cluster',
	deployments: 'Deployments'
};

export const DEMO_CLUSTERS: DemoCluster[] = [
	{
		id: 'prod-eu-west',
		name: 'prod-eu-west',
		env: 'prod',
		health: 'warn',
		version: '17.2',
		ha: 'Patroni',
		region: 'eu-west-2',
		database: 'orders'
	},
	{
		id: 'prod-eu-central',
		name: 'prod-eu-central-ledger',
		env: 'prod',
		health: 'crit',
		version: '16.6',
		ha: 'Patroni',
		region: 'europe-west3',
		database: 'ledger'
	},
	{
		id: 'staging-eu-west',
		name: 'staging-eu-west',
		env: 'staging',
		health: 'ok',
		version: '17.2',
		ha: 'Patroni',
		region: 'eu-west-2',
		database: 'orders'
	},
	{
		id: 'dev-shared',
		name: 'dev-shared',
		env: 'dev',
		health: 'ok',
		version: '17.2',
		ha: 'Standalone',
		region: 'uksouth',
		database: 'orders'
	}
];

export interface Command {
	id: string;
	group: string;
	label: string;
	hint?: string;
	keys?: string[];
	run: () => void;
}

/** Switching into production needs an explicit confirm. Leaving it does not. */
export function prodSwitchNeedsConfirm(from: EnvKind, to: EnvKind): boolean {
	return to === 'prod' && from !== 'prod';
}

export function filterCommands(commands: Command[], query: string): Command[] {
	const q = query.trim().toLowerCase();
	if (!q) return commands.filter((c) => c.group !== 'Queries');
	const terms = q.split(/\s+/);
	return commands.filter((c) =>
		terms.every((t) => `${c.label} ${c.hint ?? ''} ${c.group}`.toLowerCase().includes(t))
	);
}

export function stepRange(range: Range, direction: -1 | 1): Range {
	const i = RANGES.indexOf(range);
	const next = Math.min(RANGES.length - 1, Math.max(0, i + direction));
	return RANGES[next];
}

function clusterById(id: string): DemoCluster {
	return DEMO_CLUSTERS.find((c) => c.id === id) ?? DEMO_CLUSTERS[0];
}

class Shell {
	theme = $state<Theme>('dark');
	density = $state<Density>('comfortable');
	screen = $state<Screen>('fleet');
	env = $state<EnvKind>('prod');
	clusterId = $state('prod-eu-west');
	paletteOpen = $state(false);
	envMenuOpen = $state(false);
	helpOpen = $state(false);
	range = $state<Range>('1h');
	/** Set when a production switch is waiting for confirmation. */
	pendingProd = $state(false);

	cluster = $derived(clusterById(this.clusterId));
	clustersInEnv = $derived(DEMO_CLUSTERS.filter((c) => c.env === this.env));
	unhealthy = $derived(DEMO_CLUSTERS.filter((c) => c.health !== 'ok').length);
	readOnly = $derived(defaultReadOnly(this.env));

	go(screen: Screen) {
		this.screen = screen;
		this.paletteOpen = false;
		this.envMenuOpen = false;
	}

	requestEnv(env: EnvKind) {
		if (prodSwitchNeedsConfirm(this.env, env)) {
			this.pendingProd = true;
			return;
		}
		this.applyEnv(env);
	}

	confirmProd() {
		this.pendingProd = false;
		this.applyEnv('prod');
	}

	cancelProd() {
		this.pendingProd = false;
	}

	private applyEnv(env: EnvKind) {
		this.env = env;
		if (this.cluster.env !== env) {
			const next = DEMO_CLUSTERS.find((c) => c.env === env);
			if (next) this.clusterId = next.id;
		}
	}

	setCluster(id: string) {
		const cluster = clusterById(id);
		this.clusterId = cluster.id;
		this.env = cluster.env;
		this.envMenuOpen = false;
		this.pendingProd = false;
	}

	toggleTheme() {
		this.theme = this.theme === 'dark' ? 'light' : 'dark';
		this.persist();
	}

	toggleDensity() {
		this.density = this.density === 'comfortable' ? 'compact' : 'comfortable';
		this.persist();
	}

	stepRange(direction: -1 | 1) {
		this.range = stepRange(this.range, direction);
	}

	/** Apply the saved theme and density onto the document. */
	hydrate() {
		try {
			const theme = localStorage.getItem('tl-theme');
			const density = localStorage.getItem('tl-density');
			if (theme === 'light' || theme === 'dark') this.theme = theme;
			if (density === 'compact' || density === 'comfortable') this.density = density;
		} catch {
			/* private mode */
		}
		this.applyDocument();
	}

	persist() {
		this.applyDocument();
		try {
			localStorage.setItem('tl-theme', this.theme);
			localStorage.setItem('tl-density', this.density);
		} catch {
			/* private mode */
		}
	}

	applyDocument() {
		document.documentElement.classList.toggle('dark', this.theme === 'dark');
		document.documentElement.dataset.density = this.density;
	}
}

export const shell = new Shell();

export function envLabel(kind: EnvKind): string {
	return ENV_LABEL[kind];
}
