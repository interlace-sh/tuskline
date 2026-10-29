import type { Finding, Session } from '#lib/api.js';

export interface SessionRow {
	pid: number;
	user: string;
	app: string;
	state: string;
	wait: string;
	dur: string;
	query: string;
	blocked: boolean;
	blocking: number;
}

export interface DiagnosisItem {
	id: string;
	sev: 'ok' | 'warn' | 'crit';
	title: string;
	detail: string;
	fix: string;
}

/** Go encodes time.Duration as nanoseconds. */
export function formatAge(ns: number): string {
	if (!Number.isFinite(ns) || ns <= 0) return '0ms';
	const ms = ns / 1e6;
	if (ms < 1000) return `${Math.round(ms)}ms`;
	const seconds = ms / 1000;
	if (seconds < 60) return `${seconds < 10 ? seconds.toFixed(1) : Math.round(seconds)}s`;
	const minutes = Math.floor(seconds / 60);
	const rest = Math.round(seconds % 60);
	if (minutes < 60) return `${minutes}m ${rest}s`;
	const hours = Math.floor(minutes / 60);
	return `${hours}h ${minutes % 60}m`;
}

export function sessionsFromActivity(sessions: Session[]): SessionRow[] {
	return sessions.map((session) => ({
		pid: session.pid,
		user: session.user,
		app: session.application,
		state: session.state || session.backend_type || 'unknown',
		wait: [session.wait_event_type, session.wait_event].filter(Boolean).join(':') || '—',
		dur: formatAge(session.query_age_ns || session.xact_age_ns || session.state_age_ns),
		query: session.query,
		blocked: (session.blocked_by?.length ?? 0) > 0,
		blocking: sessions.filter((other) => other.blocked_by?.includes(session.pid)).length
	}));
}

/** Analyser severities are info and warn. Info is a note, not a failure. */
export function findingsFromReport(findings: Finding[]): DiagnosisItem[] {
	return findings.map((finding) => ({
		id: finding.check,
		sev: finding.severity === 'warn' ? 'warn' : 'ok',
		title: finding.summary,
		detail: finding.evidence,
		fix: finding.fix
	}));
}
