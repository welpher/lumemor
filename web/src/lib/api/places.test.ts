import { beforeEach, expect, test, vi } from 'vitest';
import { listPlaces, type PlaceSummary } from './places';
import { sessionStatus } from '$lib/auth/session.svelte';

// The places surface's data client: the city grouping read with its
// locale parameter (the display-name join), the structural
// validation of the place summaries, and the repo-wide error-code
// convention. Every case drives one call through an injected fetch
// mock.

const paris: PlaceSummary = {
	placeId: 2988507,
	name: 'Paris',
	country: 'FR',
	assetCount: 17,
	latitude: 48.85341,
	longitude: 2.3488,
	representativeCoverAssetIds: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f72', '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f73']
};

const beijing: PlaceSummary = {
	placeId: 1816670,
	name: '北京',
	country: 'CN',
	assetCount: 42,
	latitude: 39.9075,
	longitude: 116.39723,
	representativeCoverAssetIds: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70']
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

test('listPlaces reads the city grouping with the locale join', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, [beijing, paris]));
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await listPlaces('zh-Hans');

	expect(outcome).toEqual({ ok: true, places: [beijing, paris] });
	expect(fetchMock).toHaveBeenCalledWith('/api/v1/places?locale=zh-Hans', undefined);
});

test('listPlaces defaults the locale to en', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, [paris]));
	vi.stubGlobal('fetch', fetchMock);

	await listPlaces();

	expect(fetchMock).toHaveBeenCalledWith('/api/v1/places?locale=en', undefined);
});

test('listPlaces accepts the empty grouping (nothing resolved yet)', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, []))
	);

	const outcome = await listPlaces('en');

	expect(outcome).toEqual({ ok: true, places: [] });
});

test('listPlaces maps the failure arms: dead session, error codes, broken bodies', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	expect(await listPlaces('en')).toEqual({ ok: false, errorKey: 'session_expired' });

	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			jsonResponse(400, { code: 'VALIDATION_FAILED', message: 'locale must be one of: en, zh-Hans' })
		)
	);
	expect(await listPlaces('en')).toEqual({ ok: false, errorKey: 'validation_failed' });

	const mangles = [
		jsonResponse(200, 'no'),
		jsonResponse(200, [
			{ placeId: 'x', name: 'Paris', country: 'FR', assetCount: 1, representativeCoverAssetIds: [] }
		]),
		jsonResponse(200, [{ placeId: 1, name: 7, country: 'FR', assetCount: 1, representativeCoverAssetIds: [] }]),
		jsonResponse(200, [
			{ placeId: 1, name: 'Paris', country: 'FR', assetCount: -2, representativeCoverAssetIds: [] }
		]),
		jsonResponse(200, [
			{ placeId: 1, name: 'Paris', country: 'FR', assetCount: 1, representativeCoverAssetIds: 'no' }
		]),
		new Response('not json', { status: 200, headers: { 'Content-Type': 'application/json' } })
	];
	for (const mangle of mangles) {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => mangle)
		);
		expect(await listPlaces('en'), String(mangle.body)).toEqual({ ok: false, errorKey: 'internal' });
	}

	vi.stubGlobal(
		'fetch',
		vi.fn(async () => new Response('oops', { status: 500 }))
	);
	expect(await listPlaces('en')).toEqual({ ok: false, errorKey: 'internal' });
});
