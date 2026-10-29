import { describe, expect, it } from 'vitest';
import { ApiError, listClusters, serverHealth, splitRef } from './api';

function fakeFetch(routes: Record<string, { status?: number; body: unknown }>): typeof fetch {
	return (async (input) => {
		const url = String(input);
		const route = routes[url];
		if (!route) return new Response('missing', { status: 404 });
		return new Response(JSON.stringify(route.body), {
			status: route.status ?? 200,
			headers: { 'Content-Type': 'application/json' }
		});
	}) as typeof fetch;
}

describe('api client', () => {
	it('reads the health payload', async () => {
		const fetchImpl = fakeFetch({ '/api/v1/health': { body: { status: 'ok' } } });
		await expect(serverHealth(fetchImpl)).resolves.toEqual({ status: 'ok' });
	});

	it('returns an empty cluster list and surfaces API errors', async () => {
		const empty = fakeFetch({ '/api/v1/clusters': { body: { clusters: [] } } });
		await expect(listClusters(empty)).resolves.toEqual([]);

		const missing = fakeFetch({
			'/api/v1/clusters': { status: 500, body: { error: 'inventory unreadable' } }
		});
		await expect(listClusters(missing)).rejects.toMatchObject({
			name: 'ApiError',
			status: 500,
			message: 'inventory unreadable'
		});
		await expect(listClusters(missing)).rejects.toBeInstanceOf(ApiError);
	});

	it('splits an inventory ref on the first slash', () => {
		expect(splitRef('prod/orders')).toEqual({ env: 'prod', name: 'orders' });
		expect(() => splitRef('orders')).toThrow(ApiError);
		expect(() => splitRef('prod/a/b')).toThrow(ApiError);
	});
});
