import { beforeEach, expect, test, vi } from 'vitest';
import { fetchPersonCluster } from './people';
import { sessionStatus } from '$lib/auth/session.svelte';

// The people client's cluster read: the dead-end classification of
// the malformed-id edge. The named list and the unnamed pages carry
// their own page-level stories (the people pages' tests); this file
// pins the id-shape guard that spares the retryable face a request.

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

beforeEach(async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
});

test('a non-UUID cluster id is the dead end before any request', async () => {
	const fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
	const outcome = await fetchPersonCluster('12');
	expect(outcome).toEqual({ ok: false, errorKey: 'person_not_found', unreachable: true });
	expect(fetchMock).not.toHaveBeenCalled();
});
