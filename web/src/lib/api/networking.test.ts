// The networking clients' tests: the map config read (both successful
// shapes and the shape failures) and the release check's two calls
// (the cached read and the manual run, the failure mapping, the
// unparseable-body refusal). Every fetch is a stub; nothing reaches a
// network.

import { describe, expect, test, vi } from 'vitest';
import { getMapConfig } from './mapConfig';
import { getUpdateCheck, runUpdateCheck } from './updateCheck';

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function stub(response: () => Response | Promise<Response>) {
	return vi.fn(async () => response());
}

describe('getMapConfig', () => {
	test('a configured source answers the URL', async () => {
		const fetcher = stub(() => jsonResponse(200, { tilejsonUrl: 'https://tiles.example.net/style.json' }));
		const outcome = await getMapConfig(fetcher as unknown as typeof fetch);
		expect(outcome).toEqual({ ok: true, tilejsonUrl: 'https://tiles.example.net/style.json' });
	});

	test('null and the empty string are both the unconfigured state', async () => {
		for (const body of [{ tilejsonUrl: null }, { tilejsonUrl: '' }]) {
			const fetcher = stub(() => jsonResponse(200, body));
			const outcome = await getMapConfig(fetcher as unknown as typeof fetch);
			expect(outcome).toEqual({ ok: true, tilejsonUrl: null });
		}
	});

	test('a failure maps onto the shared error keys', async () => {
		const cases: Array<[() => Response, string]> = [
			[() => jsonResponse(500, { code: 'INTERNAL' }), 'internal'],
			[() => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }), 'session_expired'],
			[() => jsonResponse(200, { nope: true }), 'internal'],
			[() => new Response('not json', { status: 200 }), 'internal']
		];
		for (const [respond, errorKey] of cases) {
			const fetcher = stub(respond);
			expect(await getMapConfig(fetcher as unknown as typeof fetch)).toEqual({ ok: false, errorKey });
		}

		// A transport failure is the generic internal error.
		const offline = vi.fn(async () => {
			throw new Error('offline');
		});
		expect(await getMapConfig(offline as unknown as typeof fetch)).toEqual({
			ok: false,
			errorKey: 'internal'
		});
	});
});

describe('the release check client', () => {
	const state = {
		enabled: true,
		status: 'update_available',
		currentVersion: '0.4.1',
		latestVersion: 'v0.5.0',
		checkedAt: '2026-11-14T09:00:00Z'
	};

	test('the read and the manual run hit the same endpoint, GET versus POST', async () => {
		const fetcher = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
			expect(init?.method).toBeDefined();
			return jsonResponse(200, state);
		});
		const read = await getUpdateCheck(fetcher as unknown as typeof fetch);
		expect(read).toEqual({ ok: true, state });
		expect((fetcher.mock.calls[0]?.[1] as RequestInit).method).toBe('GET');

		const run = await runUpdateCheck(fetcher as unknown as typeof fetch);
		expect(run).toEqual({ ok: true, state });
		expect((fetcher.mock.calls[1]?.[1] as RequestInit).method).toBe('POST');
	});

	test('the unknown status is a successful answer, not an error', async () => {
		const fetcher = stub(() =>
			jsonResponse(200, { ...state, status: 'unknown', latestVersion: null, checkedAt: null })
		);
		const outcome = await runUpdateCheck(fetcher as unknown as typeof fetch);
		expect(outcome.ok).toBe(true);
		if (outcome.ok) {
			expect(outcome.state.status).toBe('unknown');
			expect(outcome.state.latestVersion).toBeNull();
		}
	});

	test('a drifted body or a failed request maps onto the error keys', async () => {
		const drifted = stub(() => jsonResponse(200, { enabled: true, status: 'brand_new' }));
		expect(await getUpdateCheck(drifted as unknown as typeof fetch)).toEqual({
			ok: false,
			errorKey: 'internal'
		});

		const forbidden = stub(() => jsonResponse(403, { code: 'AUTH_FORBIDDEN' }));
		expect(await getUpdateCheck(forbidden as unknown as typeof fetch)).toEqual({
			ok: false,
			errorKey: 'auth_forbidden'
		});
	});
});
