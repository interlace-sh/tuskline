// App shell state. An empty inventory stays on the prototype fixtures (demo mode).
import { listClusters } from '#lib/api.js';
import { defaultReadOnly, ENV_LABEL, type EnvKind } from '#lib/env.js';
import {
	clusterFromAPI,
	demoForced,
	fixtureClusters,
	selectClusters,
	type Health,
	type ShellCluster
} from '#lib/fixtures/clusters.js';

export type { Health, ShellCluster };

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

export type Theme = 'dark' | 'light';

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

export function filterCommands<T extends Command>(commands: T[], query: string): T[] {
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

class Shell {
	theme = $state<Theme>('dark');
	screen = $state<Screen>('fleet');
	env = $state<EnvKind>('prod');
	clusterId = $state('prod-eu-west');
	/** True while the shell is showing prototype fixtures rather than the inventory. */
	demo = $state(true);
	clusters = $state<ShellCluster[]>(fixtureClusters());
	/** Icon-only left rail. */
	navCollapsed = $state(false);
	paletteOpen = $state(false);
	envMenuOpen = $state(false);
	helpOpen = $state(false);
	range = $state<Range>('1h');
	/** Set when a production switch is waiting for confirmation. */
	pendingProd = $state(false);

	cluster = $derived(
		this.clusters.find((c) => c.id === this.clusterId) ?? this.clusters[0] ?? fixtureClusters()[0]
	);
	clustersInEnv = $derived(this.clusters.filter((c) => c.env === this.env));
	unhealthy = $derived(
		this.clusters.filter((c) => c.health === 'warn' || c.health === 'crit').length
	);
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
			const next = this.clusters.find((c) => c.env === env);
			if (next) this.clusterId = next.id;
		}
	}

	setCluster(id: string) {
		const cluster = this.clusters.find((c) => c.id === id) ?? this.clusters[0];
		this.clusterId = cluster.id;
		this.env = cluster.env;
		this.envMenuOpen = false;
		this.pendingProd = false;
	}

	/** Load the inventory. An empty list, a failed request, or ?demo=1 stays on fixtures. */
	async refresh(fetchImpl: typeof fetch = fetch) {
		const force = demoForced(typeof location === 'undefined' ? '' : location.search);
		let live: ShellCluster[] = [];
		if (!force) {
			try {
				live = (await listClusters(fetchImpl)).map(clusterFromAPI);
			} catch {
				live = [];
			}
		}
		const next = selectClusters(live, force);
		this.demo = next.demo;
		this.clusters = next.clusters;
		if (!this.clusters.some((c) => c.id === this.clusterId)) {
			this.clusterId = this.clusters[0].id;
			this.env = this.clusters[0].env;
		}
	}

	toggleTheme() {
		this.theme = this.theme === 'dark' ? 'light' : 'dark';
		this.persist();
	}

	toggleNav() {
		this.navCollapsed = !this.navCollapsed;
		try {
			localStorage.setItem('tl-nav', this.navCollapsed ? 'icons' : 'full');
		} catch {
			/* private mode */
		}
	}

	stepRange(direction: -1 | 1) {
		this.range = stepRange(this.range, direction);
	}

	/** Apply the saved theme onto the document. */
	hydrate() {
		try {
			const theme = localStorage.getItem('tl-theme');
			if (theme === 'light' || theme === 'dark') this.theme = theme;
			this.navCollapsed = localStorage.getItem('tl-nav') === 'icons';
		} catch {
			/* private mode */
		}
		this.applyDocument();
	}

	persist() {
		this.applyDocument();
		try {
			localStorage.setItem('tl-theme', this.theme);
		} catch {
			/* private mode */
		}
	}

	applyDocument() {
		document.documentElement.classList.toggle('dark', this.theme === 'dark');
	}
}

export const shell = new Shell();

export function envLabel(kind: EnvKind): string {
	return ENV_LABEL[kind];
}
