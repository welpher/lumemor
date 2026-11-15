import { beforeEach, expect, test, vi } from 'vitest';
import { addTagAssets, createTag, listTags, removeTagAssets, type TagSummary } from './tags';
import { sessionStatus } from '$lib/auth/session.svelte';

// The tags surface's data client: the vocabulary listing, the
// idempotent create (200 existing / 201 fresh, both the same summary
// shape), and the batch assignment add with its honest
// {added, skipped} report -- each behind the repo-wide error-code
// convention. Every case drives one call through an injected fetch
// mock.

const beach: TagSummary = {
	id: '0192f6a1-7b8c-9d0e-a1b2-3c4d5e6f7081',
	name: 'Beach',
	assetCount: 42,
	createdAt: '2026-11-06T09:00:00Z',
	updatedAt: '2026-11-06T09:00:00Z'
};

const sunrise: TagSummary = {
	id: '0192f6a1-7b8c-9d0e-a1b2-3c4d5e6f7082',
	name: 'Sunrise',
	assetCount: 0,
	createdAt: '2026-11-06T09:30:00Z',
	updatedAt: '2026-11-06T09:30:00Z'
};

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

beforeEach(async () => {
	// Boot the session store unauthenticated: the 401 interceptor
	// stays inert for these tests (no mid-session flips unless a
	// case arranges one).
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
});

test("listTags reads the caller's vocabulary", async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, [beach, sunrise]));
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await listTags();

	expect(outcome).toEqual({ ok: true, tags: [beach, sunrise] });
	expect(fetchMock).toHaveBeenCalledWith('/api/v1/tags', undefined);
});

test('listTags accepts the empty vocabulary', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, []))
	);

	expect(await listTags()).toEqual({ ok: true, tags: [] });
});

test('createTag POSTs the name and answers the fresh 201 summary', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		expect(String(input)).toBe('/api/v1/tags');
		expect(init?.method).toBe('POST');
		expect(JSON.parse(String(init?.body))).toEqual({ name: 'Roadtrip' });
		return jsonResponse(201, { ...beach, name: 'Roadtrip', assetCount: 0 });
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await createTag('Roadtrip');

	expect(outcome).toEqual({ ok: true, tag: { ...beach, name: 'Roadtrip', assetCount: 0 } });
});

test('createTag answers the idempotent 200 with the existing tag', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, beach))
	);

	const outcome = await createTag('Beach');

	expect(outcome).toEqual({ ok: true, tag: beach });
});

test('addTagAssets POSTs the batch and answers the honest report', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		expect(String(input)).toBe(`/api/v1/tags/${beach.id}/assets`);
		expect(init?.method).toBe('POST');
		expect(JSON.parse(String(init?.body))).toEqual({ assetIds: ['a1', 'a2'] });
		return jsonResponse(200, { added: 2, skipped: 1 });
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await addTagAssets(beach.id, ['a1', 'a2']);

	expect(outcome).toEqual({ ok: true, report: { added: 2, skipped: 1 } });
});

test('the tags surface maps the failure arms: dead session, codes, broken bodies', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	expect(await listTags()).toEqual({ ok: false, errorKey: 'session_expired' });

	// The tag 404 (unknown id or another account's -- one shape).
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(404, { code: 'TAG_NOT_FOUND', message: 'no such tag for this caller' }))
	);
	expect(await addTagAssets(beach.id, ['a1'])).toEqual({ ok: false, errorKey: 'tag_not_found' });

	// The rename collision code exists for the write faces that can
	// answer it; the create converges instead, but the mapping is
	// the convention's own.
	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			jsonResponse(409, { code: 'TAG_NAME_TAKEN', message: 'another tag already carries this name' })
		)
	);
	expect(await createTag('Beach')).toEqual({ ok: false, errorKey: 'tag_name_taken' });

	const mangles = [
		jsonResponse(200, 'no'),
		jsonResponse(200, [
			{
				id: 't1',
				name: 'Beach',
				assetCount: '1',
				createdAt: '2026-11-06T09:00:00Z',
				updatedAt: '2026-11-06T09:00:00Z'
			}
		]),
		jsonResponse(201, { id: 't1' }),
		jsonResponse(200, { added: '2', skipped: 1 }),
		jsonResponse(200, { added: 2 }),
		new Response('not json', { status: 200, headers: { 'Content-Type': 'application/json' } })
	];
	for (const mangle of mangles) {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => mangle)
		);
		expect(await listTags(), `list ${String(mangle.body)}`).toEqual({ ok: false, errorKey: 'internal' });
	}

	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(500, { code: 'INTERNAL' }))
	);
	expect(await createTag('Beach')).toEqual({ ok: false, errorKey: 'internal' });

	vi.stubGlobal(
		'fetch',
		vi.fn(async () => new Response('oops', { status: 502 }))
	);
	expect(await addTagAssets(beach.id, ['a1'])).toEqual({ ok: false, errorKey: 'internal' });
});

test('the untag half posts the DELETE body and maps the outcomes', async () => {
	let seen: { url: string; init: RequestInit | undefined } | undefined;
	const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		seen = { url: String(input), init };
		return jsonResponse(200, { removed: 1, skipped: 0 });
	});
	expect(await removeTagAssets('t1', ['a1'], fetcher)).toEqual({
		ok: true,
		report: { removed: 1, skipped: 0 }
	});
	expect(seen?.url).toBe('/api/v1/tags/t1/assets');
	expect(seen?.init?.method).toBe('DELETE');
	expect(JSON.parse(String(seen?.init?.body))).toEqual({ assetIds: ['a1'] });

	// The refusal and the failure mappings ride the same error
	// convention as every other client of this module.
	expect(
		await removeTagAssets(
			't1',
			['a1'],
			vi.fn(async () => jsonResponse(404, { code: 'TAG_NOT_FOUND' }))
		)
	).toEqual({ ok: false, errorKey: 'tag_not_found' });
	expect(
		await removeTagAssets(
			't1',
			['a1'],
			vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
		)
	).toEqual({ ok: false, errorKey: 'session_expired' });
	expect(
		await removeTagAssets(
			't1',
			['a1'],
			vi.fn(async () => Promise.reject(new Error('offline')))
		)
	).toEqual({ ok: false, errorKey: 'internal' });
});
