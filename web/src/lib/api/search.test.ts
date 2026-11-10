import { get } from 'svelte/store';
import { beforeEach, expect, test, vi } from 'vitest';
import {
	createSearchClient,
	filtersFromSearchParams,
	isQueryMode,
	searchPage,
	searchParamsFromFilters,
	searchRequestOf,
	type SearchFilters
} from './search';
import { sessionStatus } from '$lib/auth/session.svelte';
import type { AssetEvent } from '$lib/events';

// The search surface's data client: the URL<->filters mapping (the
// deep-link state), the request builder (the contract's parameter
// discipline: a sub-two-rune query takes no text path, the year set
// maps onto the from/to range, the id lists ride CSV form params),
// the page read with its structural validation and error mapping,
// and the store-backed client the search page drives -- run (the
// authoritative restart), loadMore (the keyset walk) and applyEvent
// (the pinned SSE convergence: the refetch IS the convergence,
// collapsed single-flight).

const itemDefaults = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image' as const,
	originalFilename: 'IMG_0001.jpg',
	fileSize: 1000,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d659a2feaa0c55ad015',
	createdAt: '2026-10-29T09:12:00Z',
	takenAtSource: 'exif' as const,
	isTrashed: false,
	isFavorite: false,
	isReady: true,
	width: 4000,
	height: 3000,
	durationMs: null,
	blurhash: 'UHL6aAof00WBWB00ayWB00',
	livePhotoVideoId: null
};

function item(id: string, takenAt: string, overrides: Record<string, unknown> = {}) {
	return { id, takenAt, ...itemDefaults, ...overrides };
}

const pageOne = [item('a1', '2026-11-02T10:00:00Z'), item('a2', '2026-11-01T09:00:00Z')];
const pageTwo = [item('b1', '2026-09-05T08:00:00Z')];

const facetsBody = {
	people: [{ id: '40000000-0000-0000-0000-000000000001', name: 'Mom', assetCount: 3 }],
	places: [{ placeId: 2988507, name: 'Paris', country: 'FR', assetCount: 2 }],
	years: [{ year: 2026, assetCount: 3 }],
	cameras: [{ id: 'Apple|iPhone 15', make: 'Apple', model: 'iPhone 15', assetCount: 2 }],
	kinds: [{ kind: 'image', assetCount: 3 }],
	albums: [{ id: '20000000-0000-0000-0000-000000000001', title: 'Trip', assetCount: 2 }],
	tags: [{ id: '30000000-0000-0000-0000-000000000001', name: 'beach', assetCount: 2 }]
};

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, () => Response>;

function stubFetch(route: FetchRoute): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL): Promise<Response> => {
		const url = String(input);
		const answer = route[url];
		if (answer === undefined) {
			throw new Error(`unexpected fetch ${url}`);
		}
		return answer();
	});
}

/** One asset.* event of the SSE stream (the convergence trigger). */
function event(id: number, name: AssetEvent['name'], assetId: string): AssetEvent {
	return { id, name, assetId, kind: 'image' };
}

const emptyFilters: SearchFilters = {
	query: '',
	personIds: [],
	placeIds: [],
	years: [],
	cameras: [],
	kinds: [],
	albumIds: [],
	tagIds: []
};

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

test('the mode rule mirrors the contract: a trimmed query of two or more runes takes the text path', () => {
	expect(isQueryMode('')).toBe(false);
	expect(isQueryMode('   ')).toBe(false);
	expect(isQueryMode('x')).toBe(false);
	expect(isQueryMode(' x ')).toBe(false);
	expect(isQueryMode('海')).toBe(false);
	expect(isQueryMode('xy')).toBe(true);
	expect(isQueryMode(' 海边 ')).toBe(true);
});

test('filtersFromSearchParams reads the URL state, dropping malformed entries', () => {
	const params = new URLSearchParams(
		'query=%E6%B5%B7%E8%BE%B9&personIds=p1,p2&placeIds=2988507,1816670&years=2026,2025' +
			'&cameras=Apple%7CiPhone%2015&kinds=image,video&albumIds=al1&tagIds=t1,t2'
	);
	expect(filtersFromSearchParams(params)).toEqual({
		query: '海边',
		personIds: ['p1', 'p2'],
		placeIds: [2988507, 1816670],
		years: [2026, 2025],
		cameras: ['Apple|iPhone 15'],
		kinds: ['image', 'video'],
		albumIds: ['al1'],
		tagIds: ['t1', 't2']
	});

	// A hand-written URL's junk (non-numeric ids, unknown kinds,
	// empty segments) drops out instead of poisoning the request.
	const junk = new URLSearchParams('personIds=p1,,&placeIds=x,4&years=2024&kinds=image,movie');
	expect(filtersFromSearchParams(junk)).toEqual({
		query: '',
		personIds: ['p1'],
		placeIds: [4],
		years: [2024],
		cameras: [],
		kinds: ['image'],
		albumIds: [],
		tagIds: []
	});

	expect(filtersFromSearchParams(new URLSearchParams())).toEqual(emptyFilters);
});

test('searchParamsFromFilters writes the canonical deep-link URL', () => {
	const filters: SearchFilters = {
		...emptyFilters,
		query: '海边',
		personIds: ['p1', 'p2'],
		placeIds: [2988507],
		years: [2026],
		cameras: ['Apple|iPhone 15'],
		kinds: ['image'],
		albumIds: ['al1'],
		tagIds: ['t1']
	};
	const params = searchParamsFromFilters(filters);
	expect(params.toString()).toBe(
		'query=%E6%B5%B7%E8%BE%B9&personIds=p1%2Cp2&placeIds=2988507&years=2026' +
			'&cameras=Apple%7CiPhone+15&kinds=image&albumIds=al1&tagIds=t1'
	);

	// A one-rune query stays in the URL (the box shows what was
	// typed) but only while non-empty; an empty state writes nothing.
	expect(searchParamsFromFilters({ ...emptyFilters, query: 'x' }).toString()).toBe('query=x');
	expect(searchParamsFromFilters(emptyFilters).toString()).toBe('');
});

test('searchRequestOf builds the contract request: the sub-two-rune query drops, the years become the range', () => {
	// The full filter set: the text path, every CSV dimension, the
	// year set's bounding range, the locale, the limit, the cursor.
	const request = searchRequestOf(
		{ ...emptyFilters, query: '海边', personIds: ['p1'], years: [2026], kinds: ['image'] },
		'zh-Hans',
		100,
		'C1'
	);
	expect(request).toBe(
		'query=%E6%B5%B7%E8%BE%B9&personIds=p1&from=2026-01-01T00%3A00%3A00Z' +
			'&to=2027-01-01T00%3A00%3A00Z&kinds=image&locale=zh-Hans&limit=100&cursor=C1'
	);

	// A one-rune query takes NO text path: the request is the pure
	// filter browse (the server's own rule, mirrored client-side).
	expect(searchRequestOf({ ...emptyFilters, query: 'x' }, 'en', 100, null)).toBe('locale=en&limit=100');

	// A discontiguous year set maps onto the bounding range (the
	// contract's only time filter: min year start, max year end
	// exclusive).
	const range = searchRequestOf({ ...emptyFilters, years: [2026, 2024] }, 'en', 100, null);
	expect(range).toContain('from=2024-01-01T00%3A00%3A00Z');
	expect(range).toContain('to=2027-01-01T00%3A00%3A00Z');
});

test('searchPage reads one page with items, cursor and facets', async () => {
	const fetchMock = stubFetch({
		'/api/v1/search?locale=en&limit=100': () =>
			jsonResponse(200, { items: pageOne, nextCursor: 'C1', facets: facetsBody })
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await searchPage(emptyFilters, { locale: 'en' });

	expect(outcome).toEqual({
		ok: true,
		page: { items: pageOne, nextCursor: 'C1', facets: facetsBody }
	});
});

test('searchPage passes the cursor and the locale through', async () => {
	const fetchMock = stubFetch({
		'/api/v1/search?query=%E6%B5%B7%E8%BE%B9&locale=zh-Hans&limit=100&cursor=C1': () =>
			jsonResponse(200, { items: pageTwo, facets: facetsBody })
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await searchPage({ ...emptyFilters, query: '海边' }, { cursor: 'C1', locale: 'zh-Hans' });

	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.page.nextCursor).toBeNull();
	}
});

test('searchPage maps the failure arms: dead session, error codes, broken bodies', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/search?locale=en&limit=100': () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })
		})
	);
	expect(await searchPage(emptyFilters)).toEqual({ ok: false, errorKey: 'session_expired' });

	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/search?locale=en&limit=100': () =>
				jsonResponse(400, { code: 'VALIDATION_FAILED', message: 'invalid search cursor' })
		})
	);
	expect(await searchPage(emptyFilters)).toEqual({ ok: false, errorKey: 'validation_failed' });

	const mangles = [
		jsonResponse(200, { items: pageOne }),
		jsonResponse(200, { items: 'no', facets: facetsBody }),
		jsonResponse(200, { items: [{ id: 'a1' }], facets: facetsBody }),
		jsonResponse(200, {
			items: pageOne,
			facets: { ...facetsBody, people: [{ id: 'p1', name: 'Mom', assetCount: '3' }] }
		}),
		jsonResponse(200, { items: pageOne, nextCursor: 7, facets: facetsBody }),
		new Response('not json', { status: 200, headers: { 'Content-Type': 'application/json' } })
	];
	for (const mangle of mangles) {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => mangle)
		);
		expect(await searchPage(emptyFilters), String(mangle.body)).toEqual({
			ok: false,
			errorKey: 'internal'
		});
	}

	vi.stubGlobal(
		'fetch',
		vi.fn(async () => new Response('oops', { status: 500 }))
	);
	expect(await searchPage(emptyFilters)).toEqual({ ok: false, errorKey: 'internal' });
});

test('the client runs page one, walks the cursor, and dedups the append', async () => {
	const fetchMock = stubFetch({
		'/api/v1/search?locale=en&limit=100': () =>
			jsonResponse(200, { items: pageOne, nextCursor: 'C1', facets: facetsBody }),
		'/api/v1/search?locale=en&limit=100&cursor=C1': () =>
			jsonResponse(200, { items: [...pageTwo, pageOne[0]], facets: facetsBody })
	});
	vi.stubGlobal('fetch', fetchMock);
	const client = createSearchClient();

	await client.run(emptyFilters, 'en');
	expect(get(client)).toEqual({
		items: pageOne,
		facets: facetsBody,
		nextCursor: 'C1',
		status: 'ready',
		errorKey: ''
	});

	await client.loadMore();
	const state = get(client);
	expect(state.items.map((entry) => entry.id)).toEqual(['a1', 'a2', 'b1']);
	expect(state.nextCursor).toBeNull();
	expect(state.status).toBe('ready');

	// The exhausted walk no-ops.
	await client.loadMore();
	expect(fetchMock).toHaveBeenCalledTimes(2);
});

test('the client parks a failed first run in error, and the walk retry re-uses the same cursor', async () => {
	let failFirst = true;
	const fetchMock = vi.fn(async (input: RequestInfo | URL): Promise<Response> => {
		if (String(input).includes('cursor')) {
			return jsonResponse(200, { items: pageTwo, facets: facetsBody });
		}
		return failFirst
			? jsonResponse(500, { code: 'INTERNAL' })
			: jsonResponse(200, { items: pageOne, nextCursor: 'C1', facets: facetsBody });
	});
	vi.stubGlobal('fetch', fetchMock);
	const client = createSearchClient();

	await client.run(emptyFilters, 'en');
	expect(get(client)).toEqual({
		items: [],
		facets: null,
		nextCursor: null,
		status: 'error',
		errorKey: 'internal'
	});

	// The page's retry is a fresh run (the first-load failure's
	// affordance); the walk then continues from the cursor.
	failFirst = false;
	await client.run(emptyFilters, 'en');
	await client.loadMore();
	expect(get(client).items.map((entry) => entry.id)).toEqual(['a1', 'a2', 'b1']);
});

test('a superseded run discards its stale response (a URL change mid-flight)', async () => {
	let resolveFirst: ((response: Response) => void) | undefined;
	const fetchMock = vi.fn(async (input: RequestInfo | URL): Promise<Response> => {
		if (String(input).includes('personIds')) {
			return jsonResponse(200, { items: pageTwo, facets: facetsBody });
		}
		return new Promise<Response>((resolve) => {
			resolveFirst = resolve;
		});
	});
	vi.stubGlobal('fetch', fetchMock);
	const client = createSearchClient();

	const first = client.run(emptyFilters, 'en');
	const second = client.run({ ...emptyFilters, personIds: ['p1'] }, 'en');
	resolveFirst?.(jsonResponse(200, { items: pageOne, facets: facetsBody }));
	await Promise.all([first, second]);

	// The stale page-one answer never lands: the fresh filters own
	// the wall.
	expect(get(client).items.map((entry) => entry.id)).toEqual(['b1']);
});

test('applyEvent re-executes the current search, collapsed single-flight', async () => {
	let pageOneCalls = 0;
	const fetchMock = vi.fn(async (input: RequestInfo | URL): Promise<Response> => {
		if (String(input).includes('cursor')) {
			return jsonResponse(200, { items: pageTwo, facets: facetsBody });
		}
		pageOneCalls += 1;
		return jsonResponse(
			200,
			pageOneCalls > 1
				? { items: [item('fresh', '2026-11-06T08:00:00Z')], facets: facetsBody }
				: { items: pageOne, nextCursor: 'C1', facets: facetsBody }
		);
	});
	vi.stubGlobal('fetch', fetchMock);
	const client = createSearchClient();

	// Before any run, events are inert.
	client.applyEvent(event(1, 'asset.ready', 'a1'));
	expect(fetchMock).not.toHaveBeenCalled();

	await client.run({ ...emptyFilters, query: '海边' }, 'en');
	await client.loadMore();
	expect(get(client).items.map((entry) => entry.id)).toEqual(['a1', 'a2', 'b1']);

	// A burst of events costs ONE trailing re-execution (the refetch
	// IS the convergence: the fresh page replaces the walk wholesale).
	client.applyEvent(event(2, 'asset.ready', 'a1'));
	client.applyEvent(event(3, 'asset.trashed', 'a2'));
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise((resolve) => setTimeout(resolve, 0));

	const state = get(client);
	expect(state.items.map((entry) => entry.id)).toEqual(['fresh']);
	// The trailing run: the initial run, the loadMore, one burst
	// re-execution.
	expect(pageOneCalls).toBe(2);
	expect(state.status).toBe('ready');
});

test('a failed convergence keeps the last good state silently', async () => {
	let failNext = false;
	const fetchMock = vi.fn(async (input: RequestInfo | URL): Promise<Response> => {
		if (String(input).includes('cursor')) {
			return jsonResponse(200, { items: pageTwo, facets: facetsBody });
		}
		return failNext
			? jsonResponse(500, { code: 'INTERNAL' })
			: jsonResponse(200, { items: pageOne, nextCursor: 'C1', facets: facetsBody });
	});
	vi.stubGlobal('fetch', fetchMock);
	const client = createSearchClient();

	await client.run(emptyFilters, 'en');
	failNext = true;
	client.applyEvent(event(1, 'asset.updated', 'a1'));
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise((resolve) => setTimeout(resolve, 0));

	const state = get(client);
	expect(state.status).toBe('ready');
	expect(state.items.map((entry) => entry.id)).toEqual(['a1', 'a2']);
});

test('reset drops the state and makes events inert again', async () => {
	const fetchMock = stubFetch({
		'/api/v1/search?locale=en&limit=100': () => jsonResponse(200, { items: pageOne, facets: facetsBody })
	});
	vi.stubGlobal('fetch', fetchMock);
	const client = createSearchClient();

	await client.run(emptyFilters, 'en');
	client.reset();
	expect(get(client)).toEqual({
		items: [],
		facets: null,
		nextCursor: null,
		status: 'idle',
		errorKey: ''
	});
	client.applyEvent(event(1, 'asset.ready', 'a1'));
	await new Promise((resolve) => setTimeout(resolve, 0));
	expect(fetchMock).toHaveBeenCalledTimes(1);
});
