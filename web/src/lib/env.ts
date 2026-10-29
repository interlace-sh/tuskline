// Environment identity. Mirrors core/inventory.EnvKind in Go.
export type EnvKind = 'local' | 'dev' | 'staging' | 'prod';

export const ENV_LABEL: Record<EnvKind, string> = {
	local: 'Local',
	dev: 'Development',
	staging: 'Staging',
	prod: 'Production'
};

// Full class names, so Tailwind's scanner can see them (no string building).
const RAIL: Record<EnvKind, string> = {
	local: 'bg-env-local',
	dev: 'bg-env-dev',
	staging: 'bg-env-staging',
	prod: 'bg-env-prod'
};

/** Tailwind background class for the environment colour rail. */
export function envRailClass(kind: EnvKind): string {
	return RAIL[kind];
}

/** Production is read-only by default (safety invariant, see docs/security.md). */
export function defaultReadOnly(kind: EnvKind): boolean {
	return kind === 'prod';
}
