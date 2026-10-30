import type { EnvKind } from '#lib/env.js';

export const WIZARD_STEPS = ['Connection', 'Test', 'Finish'] as const;

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
}

export const PROVIDER_LABEL: Record<WizardProvider, string> = {
	rds: 'Amazon RDS for PostgreSQL',
	cloudsql: 'Google Cloud SQL',
	azure: 'Azure Database for PostgreSQL',
	self: 'Self-managed PostgreSQL'
};

/** Demo hosts only. The test step never opens a connection to them. */
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
	env: 'staging'
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
	env: 'dev'
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
	if (draft.ssh && !draft.bastion.trim())
		problems.push('Bastion host is required for an SSH tunnel.');
	return problems;
}

export interface DemoTestLine {
	tone: 'ok' | 'warn';
	text: string;
}

/** A local report. It does not dial the host. */
export function demoTestReport(draft: ConnectionDraft): DemoTestLine[] {
	const provider = detectProvider(draft);
	const lines: DemoTestLine[] = [
		{ tone: 'ok', text: `Provider guess: ${PROVIDER_LABEL[provider]}.` },
		{
			tone: 'ok',
			text: `Would use ${draft.user.trim()} at ${draft.host.trim()}:${draft.port}/${draft.database.trim()}.`
		},
		{ tone: 'ok', text: 'No connection was opened.' }
	];
	if (draft.tls === 'disable') {
		lines.push({
			tone: 'warn',
			text: 'TLS is disabled. That is only suitable for a local disposable database.'
		});
	} else if (draft.tls === 'require') {
		lines.push({
			tone: 'warn',
			text: 'TLS is require: the certificate is not checked against the host name.'
		});
	} else {
		lines.push({ tone: 'ok', text: 'TLS mode is verify-full.' });
	}
	if (draft.env === 'prod') {
		lines.push({ tone: 'ok', text: 'Production sessions open read-only.' });
	}
	return lines;
}
