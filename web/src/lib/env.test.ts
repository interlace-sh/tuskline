import { describe, expect, it } from 'vitest';
import { defaultReadOnly, envRailClass, ENV_LABEL } from './env';

describe('env', () => {
	it('keeps production read-only by default', () => {
		expect(defaultReadOnly('prod')).toBe(true);
		expect(defaultReadOnly('dev')).toBe(false);
	});
	it('maps kinds to rail classes and labels', () => {
		expect(envRailClass('staging')).toBe('bg-env-staging');
		expect(ENV_LABEL.prod).toBe('Production');
	});
});
