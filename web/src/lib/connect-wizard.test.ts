import { describe, expect, it } from 'vitest';
import {
	RDS_PRESET,
	detectProvider,
	demoTestReport,
	draftProblems,
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
	});
});

describe('demoTestReport', () => {
	it('does not claim a connection was opened', () => {
		const lines = demoTestReport({ ...RDS_PRESET, tls: 'disable', env: 'prod' });
		expect(lines.some((line) => line.text.includes('No connection was opened'))).toBe(true);
		expect(
			lines.some((line) => line.tone === 'warn' && line.text.includes('TLS is disabled'))
		).toBe(true);
		expect(lines.some((line) => line.text.includes('read-only'))).toBe(true);
	});
});
