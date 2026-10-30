import type { EnvKind } from '#lib/env.js';

export type TlsMode = 'verify-full' | 'require' | 'disable';
export type WizardAuth = 'password' | 'aws-iam' | 'gcp-iam' | 'azure-entra';
export type WizardProvider = 'rds' | 'cloudsql' | 'azure' | 'self';

export interface ConnectionDraft {
	name: string;
	host: string;
	port: string;
	database: string;
	user: string;
	auth: WizardAuth;
	tls: TlsMode;
	ssh: boolean;
	bastion: string;
	env: EnvKind;
	/** Environment variable name holding the password. Never the password itself. */
	passwordEnv: string;
}

export const PROVIDER_LABEL: Record<WizardProvider, string> = {
	rds: 'Amazon RDS for PostgreSQL',
	cloudsql: 'Google Cloud SQL',
	azure: 'Azure Database for PostgreSQL',
	self: 'Self-managed PostgreSQL'
};

/** Demo hosts only. Create checks the form locally and does not dial them. */
export const RDS_PRESET: ConnectionDraft = {
	name: 'orders-eu-demo',
	host: 'demo-orders.example.eu-west-2.rds.amazonaws.com',
	port: '5432',
	database: 'orders',
	user: 'tuskline_monitor',
	auth: 'aws-iam',
	tls: 'verify-full',
	ssh: false,
	bastion: '',
	env: 'staging',
	passwordEnv: ''
};

export const SELF_PRESET: ConnectionDraft = {
	name: 'billing-onprem-demo',
	host: 'pg-demo-1.example',
	port: '5432',
	database: 'billing',
	user: 'tuskline_monitor',
	auth: 'password',
	tls: 'verify-full',
	ssh: false,
	bastion: '',
	env: 'dev',
	passwordEnv: ''
};

/** Guess a provider from the host name. IAM auth is only a hint for Cloud SQL. */
export function detectProvider(draft: Pick<ConnectionDraft, 'host' | 'auth'>): WizardProvider {
	const host = draft.host.trim().toLowerCase();
	if (host.endsWith('.rds.amazonaws.com') || host.includes('.rds.amazonaws.com')) return 'rds';
	if (host.includes('.postgres.database.azure.com')) return 'azure';
	if (draft.auth === 'gcp-iam' || host.includes('cloudsql') || host.endsWith('.googleapis.com')) {
		return 'cloudsql';
	}
	return 'self';
}

/** Problems that block the test step. Empty means the draft can be tested. */
export function draftProblems(draft: ConnectionDraft): string[] {
	const problems: string[] = [];
	if (!draft.name.trim()) problems.push('Name is required.');
	if (!draft.host.trim()) problems.push('Host is required.');
	if (!draft.database.trim()) problems.push('Database is required.');
	if (!draft.user.trim()) problems.push('User is required.');
	const port = Number(draft.port);
	if (!Number.isInteger(port) || port < 1 || port > 65535) problems.push('Port must be 1–65535.');
	if (draft.ssh) problems.push('SSH tunnels cannot be saved yet.');
	if (draft.ssh && !draft.bastion.trim())
		problems.push('Bastion host is required for an SSH tunnel.');
	if (draft.passwordEnv.trim() && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(draft.passwordEnv.trim())) {
		problems.push(
			'Password environment variable must be a name such as ORDERS_PASSWORD, not the password itself.'
		);
	}
	if (!/^[a-z0-9](?:[a-z0-9-]{0,62})$/.test(draft.name.trim())) {
		problems.push('Name must be lowercase letters, digits and hyphens.');
	}
	return problems;
}

function localHost(host: string): boolean {
	return host === 'localhost' || host === '127.0.0.1' || host === '::1';
}

/**
 * Decide whether Create can proceed. This does not open a socket: a live probe
 * waits until the connection is saved.
 */
export function verifyDraft(draft: ConnectionDraft): string | null {
	const problems = draftProblems(draft);
	if (problems.length) return problems[0];
	const host = draft.host.trim().toLowerCase();
	if (draft.tls === 'disable' && !localHost(host)) {
		return 'Refusing an unencrypted connection to a remote host.';
	}
	if (host.includes('unreachable') || host.includes('fail.example')) {
		return `Could not reach ${draft.host.trim()}:${draft.port.trim()}.`;
	}
	return null;
}
