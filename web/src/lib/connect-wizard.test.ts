import { describe, expect, it } from 'vitest';
import {
	RDS_PRESET,
	detectProvider,
	draftProblems,
	verifyDraft,
	type ConnectionDraft
} from './connect-wizard';

describe('detectProvider', () => {
	it('recognises managed host names and treats everything else as self-managed', () => {
		expect(detectProvider(RDS_PRESET)).toBe('rds');
		expect(detectProvider({ host: 'db.postgres.database.azure.com', auth: 'password' })).toBe(
			'azure'
		);
		expect(detectProvider({ host: 'pg.example', auth: 'gcp-iam' })).toBe('cloudsql');
		expect(detectProvider({ host: '127.0.0.1', auth: 'password' })).toBe('self');
	});
});

describe('draftProblems', () => {
	it('accepts the RDS preset and rejects a missing bastion', () => {
		expect(draftProblems(RDS_PRESET)).toEqual([]);
		const tunneled: ConnectionDraft = { ...RDS_PRESET, ssh: true, bastion: '  ' };
		expect(draftProblems(tunneled)).toContain('Bastion host is required for an SSH tunnel.');
		expect(draftProblems({ ...RDS_PRESET, port: '0' })).toContain('Port must be 1–65535.');
		expect(draftProblems({ ...RDS_PRESET, name: 'Orders' })).toContain(
			'Name must be lowercase letters, digits and hyphens.'
		);
		expect(draftProblems({ ...RDS_PRESET, passwordEnv: 'not a name' })).toContain(
			'Password environment variable must be a name such as ORDERS_PASSWORD, not the password itself.'
		);
		expect(draftProblems({ ...RDS_PRESET, ssh: true, bastion: 'bastion.example' })).toContain(
			'SSH tunnels cannot be saved yet.'
		);
	});
});

describe('verifyDraft', () => {
	it('accepts the demo preset and refuses a remote host without TLS', () => {
		expect(verifyDraft(RDS_PRESET)).toBeNull();
		expect(verifyDraft({ ...RDS_PRESET, tls: 'disable' })).toMatch(/unencrypted/);
		expect(verifyDraft({ ...RDS_PRESET, host: 'db.fail.example' })).toMatch(/Could not reach/);
		expect(verifyDraft({ ...RDS_PRESET, tls: 'disable', host: '127.0.0.1' })).toBeNull();
	});
});
