import { describe, expect, it } from 'vitest';
import {
	filterCommands,
	prodSwitchNeedsConfirm,
	screenAvailable,
	stepRange,
	type Command
} from './shell.svelte';

const commands: Command[] = [
	{ id: '1', group: 'Navigate', label: 'Fleet overview', run: () => {} },
	{
		id: '2',
		group: 'Setup & deploy',
		label: 'Deploy a new cluster…',
		hint: 'tl deploy init',
		run: () => {}
	},
	{ id: '3', group: 'Queries', label: 'SELECT 1', hint: 'orders', run: () => {} }
];

describe('shell commands', () => {
	it('hides queries until the user types', () => {
		expect(filterCommands(commands, '').map((c) => c.id)).toEqual(['1', '2']);
	});
	it('matches every word across the label and hint', () => {
		expect(filterCommands(commands, 'deploy cluster').map((c) => c.id)).toEqual(['2']);
		expect(filterCommands(commands, 'select orders').map((c) => c.id)).toEqual(['3']);
	});
	it('asks before entering production', () => {
		expect(prodSwitchNeedsConfirm('dev', 'prod')).toBe(true);
		expect(prodSwitchNeedsConfirm('prod', 'prod')).toBe(false);
		expect(prodSwitchNeedsConfirm('prod', 'dev')).toBe(false);
	});
	it('keeps the base screens when the cluster has no HA or backup', () => {
		expect(screenAvailable('sql', false)).toBe(true);
		expect(screenAvailable('migrations', false)).toBe(true);
		expect(screenAvailable('ha', false)).toBe(false);
		expect(screenAvailable('deploy', false)).toBe(false);
		expect(screenAvailable('ha', true)).toBe(true);
	});
	it('steps the time range and stops at the ends', () => {
		expect(stepRange('1h', 1)).toBe('6h');
		expect(stepRange('15m', -1)).toBe('15m');
		expect(stepRange('30d', 1)).toBe('30d');
	});
});
