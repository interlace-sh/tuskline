// Demo config advice. It is the 16GB web fixture from the advisor golden
// file, shortened to the changes a reader sees first. Nothing here is live.
import type { AdviceItem } from '#lib/api.js';

export const DEMO_ADVICE: AdviceItem[] = [
	{
		name: 'shared_buffers',
		current: '128MB',
		proposed: '4GB',
		apply: 'restart',
		summary:
			'25% of RAM is the starting point on a dedicated server. More than 40% is unlikely to help, because PostgreSQL also uses the operating system cache.',
		evidence:
			'Cache hit ratio is 97%. That includes the operating system cache, so it is context, not proof.',
		diff: 'shared_buffers = 4GB'
	},
	{
		name: 'work_mem',
		current: '4MB',
		proposed: '19MB',
		apply: 'reload',
		summary:
			'Sized from the RAM left after shared buffers, divided by connections and parallel workers, with a margin so a spike does not exhaust the machine.',
		diff: 'work_mem = 19MB'
	},
	{
		name: 'random_page_cost',
		current: '4',
		proposed: '1.1',
		apply: 'reload',
		summary:
			'Solid-state storage does not pay a spinning-disk seek, so a random read is close to a sequential one.',
		diff: 'random_page_cost = 1.1'
	},
	{
		name: 'track_io_timing',
		current: 'off',
		proposed: 'on',
		apply: 'reload',
		summary:
			'Block read and write times are what make I/O waits visible. The overhead on current hardware is small.',
		diff: 'track_io_timing = on'
	}
];

export const DEMO_NOTES = [
	'Demo fixture for a 16GB web server. Nothing is read from a database, and nothing is applied.'
];
