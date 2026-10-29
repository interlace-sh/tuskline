import { describe, expect, it } from 'vitest';
import type { Finding, Session } from './api';
import { findingsFromReport, formatAge, sessionsFromActivity } from './activity-view';

const session = (partial: Partial<Session> & Pick<Session, 'pid'>): Session => ({
	user: 'api',
	database: 'orders',
	application: 'orders-api',
	client_addr: '',
	backend_type: 'client backend',
	state: 'active',
	wait_event_type: '',
	wait_event: '',
	xact_age_ns: 0,
	query_age_ns: 0,
	state_age_ns: 0,
	query: 'SELECT 1',
	...partial
});

describe('activity view', () => {
	it('formats durations from nanoseconds', () => {
		expect(formatAge(180e6)).toBe('180ms');
		expect(formatAge(4.2e9)).toBe('4.2s');
		expect(formatAge(252e9)).toBe('4m 12s');
	});

	it('marks who is blocked and who is blocking', () => {
		const rows = sessionsFromActivity([
			session({ pid: 10, state: 'idle in transaction', query_age_ns: 252e9 }),
			session({ pid: 11, wait_event_type: 'Lock', wait_event: 'transactionid', blocked_by: [10] })
		]);
		expect(rows[0]).toMatchObject({ blocking: 1, blocked: false, dur: '4m 12s' });
		expect(rows[1]).toMatchObject({ blocked: true, wait: 'Lock:transactionid' });
	});

	it('keeps info findings as notes', () => {
		const findings: Finding[] = [
			{ check: 'wraparound', severity: 'info', summary: 'Fine', evidence: 'low', fix: '' },
			{ check: 'bloat', severity: 'warn', summary: 'Bloated', evidence: 'big', fix: 'vacuum' }
		];
		expect(findingsFromReport(findings).map((item) => item.sev)).toEqual(['ok', 'warn']);
	});
});
