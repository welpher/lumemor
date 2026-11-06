import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems, wallItems } from '$lib/timeline/wall.svelte';

// The search page: the URL owns the state (the deep-link contract),
// the two result modes render through the wall vocabulary (QUERY
// mode's flat justified wall, BROWSE mode's month-grouped
// day-block wall exactly as the timeline), the facet chips toggle
// the seven dimensions and re-execute through a URL replacement,
// the manual pager walks the cursor, the empty and error states
// carry their affordances, the SSE events re-execute the current
// search (the pinned convergence), and the loaded items publish as
// the viewer's navigation sequence. $app/state is mocked for the
// URL; $app/navigation captures the goto the URL sync issues; the
// events module is mocked with a captured handler registry.

const { pageState, handlers, gotoMock } = vi.hoisted(() => ({
	pageState: {
		params: {} as Record<string, string>,
		url: new URL('http://localhost/search')
	},
	handlers: [] as Array<(event: unknown) => void>,
	gotoMock: vi.fn()
}));

vi.mock('$app/state', () => ({ page: pageState }));
vi.mock('$app/navigation', () => ({ goto: gotoMock }));
vi.mock('$lib/events', () => ({
	subscribeAssetEvents: (listener: (event: unknown) => void) => {
		handlers.push(listener);
		return () => {
			const index = handlers.indexOf(listener);
			if (index !== -1) {
				handlers.splice(index, 1);
			}
		};
	}
}));

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

const pageOne = [
	item('a1', '2026-11-02T10:00:00Z'),
	item('a2', '2026-11-01T09:00:00Z'),
	item('a3', '2026-10-13T12:30:00Z')
];
const pageTwo = [item('b1', '2026-09-05T08:00:00Z')];

const facetsBody = {
	people: [{ id: 'p1', name: 'Mom', assetCount: 3 }],
	places: [{ placeId: 2988507, name: 'Paris', country: 'FR', assetCount: 2 }],
	years: [{ year: 2026, assetCount: 3 }],
	cameras: [{ id: 'Apple|iPhone 15', make: 'Apple', model: 'iPhone 15', assetCount: 2 }],
	kinds: [{ kind: 'image', assetCount: 3 }],
	albums: [{ id: 'al1', title: 'Trip', assetCount: 2 }],
	tags: [{ id: 't1', name: 'beach', assetCount: 2 }]
};

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, () => Response>;

function searchRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/search?locale=en&limit=100': () =>
			jsonResponse(200, { items: pageOne, nextCursor: 'C1', facets: facetsBody }),
		'/api/v1/search?locale=en&limit=100&cursor=C1': () =>
			jsonResponse(200, { items: pageTwo, facets: facetsBody }),
		...overrides
	};
}

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

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
	await tick();
}

function mountPage(): ReturnType<typeof mount> {
	const page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
}

function tiles(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.tile'));
}

function buttonByText(text: string, scope: ParentNode = document.body): HTMLButtonElement {
	const buttons = Array.from(scope.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.includes(text));
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	vi.clearAllMocks();
	gotoMock.mockClear();
	clearWallItems();
	pageState.url = new URL('http://localhost/search');
	// Boot the session store unauthenticated (the interceptor stays
	// inert; these routes answer 200s).
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('no query opens the filter browse: the month-grouped wall, the chips, the count line', async () => {
	const fetchMock = stubFetch(searchRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The request is the pure filter browse (no query parameter).
	expect(fetchMock).toHaveBeenCalledWith('/api/v1/search?locale=en&limit=100', undefined);
	// BROWSE mode: the timeline's exact month-grouped day-block wall.
	// (The fixture spans two months: November and October 2026.)
	expect(document.body.querySelectorAll('.wall__month-header')).toHaveLength(2);
	expect(document.body.querySelectorAll('.wall__day-header')).toHaveLength(3);
	expect(tiles()).toHaveLength(3);
	// The loaded page items publish as the viewer's sequence.
	expect(wallItems().map((entry) => entry.id)).toEqual(['a1', 'a2', 'a3']);

	// The chips of every dimension, each carrying the facet's count.
	const chips = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.search-chip'));
	expect(chips.map((chip) => chip.textContent)).toEqual([
		'People · Mom 3',
		'Places · Paris 2',
		'Years · 2026 3',
		'Cameras · Apple iPhone 15 2',
		'Types · Photos 3',
		'Albums · Trip 2',
		'Tags · beach 2'
	]);
	for (const chip of chips) {
		expect(chip.getAttribute('aria-pressed')).toBe('false');
	}

	// The honest count line: the loaded count (a page follows, so
	// "so far") and the facet note.
	const meta = document.body.querySelector('.search-meta');
	expect(meta?.textContent).toContain('3 results so far');
	expect(meta?.textContent).toContain(
		"Each filter's count shows the photos the other filters leave visible."
	);

	unmount(page);
	flushSync();
});

test('a text query renders the flat justified wall: rank order, no month or day headers', async () => {
	const fetchMock = stubFetch(
		searchRoutes({
			'/api/v1/search?query=%E6%B5%B7%E8%BE%B9&locale=en&limit=100': () =>
				jsonResponse(200, { items: pageOne, facets: facetsBody })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	pageState.url = new URL('http://localhost/search?query=%E6%B5%B7%E8%BE%B9');
	const page = mountPage();
	await settle();

	expect(fetchMock).toHaveBeenCalledWith(
		'/api/v1/search?query=%E6%B5%B7%E8%BE%B9&locale=en&limit=100',
		undefined
	);
	// QUERY mode: the flat wall -- no month or day grouping (the
	// grouping would lie about the rank order).
	expect(document.body.querySelectorAll('.flat-wall')).toHaveLength(1);
	expect(document.body.querySelectorAll('.wall__month-header')).toHaveLength(0);
	expect(document.body.querySelectorAll('.wall__day-header')).toHaveLength(0);
	expect(tiles()).toHaveLength(3);
	// The rank order is the wall's sequence (the viewer walks it).
	expect(wallItems().map((entry) => entry.id)).toEqual(['a1', 'a2', 'a3']);
	// The settled count line drops the "so far".
	expect(document.body.querySelector('.search-meta')?.textContent).toContain('3 results');

	unmount(page);
	flushSync();
});

test('a one-rune query takes no text path: the request and the wall mirror the browse mode', async () => {
	const fetchMock = stubFetch(searchRoutes());
	vi.stubGlobal('fetch', fetchMock);
	pageState.url = new URL('http://localhost/search?query=x');
	const page = mountPage();
	await settle();

	// The server's own rule, mirrored: the request carries no query.
	expect(fetchMock).toHaveBeenCalledWith('/api/v1/search?locale=en&limit=100', undefined);
	// The honest render: the time-ordered grouped wall.
	expect(document.body.querySelectorAll('.flat-wall')).toHaveLength(0);
	expect(document.body.querySelectorAll('.wall__month-header')).toHaveLength(2);
	// The typed text stays in the box (the URL state).
	expect((document.body.querySelector('input.search-query') as HTMLInputElement).value).toBe('x');

	unmount(page);
	flushSync();
});

test('chips toggle their dimension through the URL (deep-linkable state)', async () => {
	vi.stubGlobal('fetch', stubFetch(searchRoutes()));
	const page = mountPage();
	await settle();

	buttonByText('People · Mom').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(gotoMock).toHaveBeenCalledWith('/search?personIds=p1', { replaceState: true });

	buttonByText('Years · 2026').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(gotoMock).toHaveBeenCalledWith('/search?years=2026', { replaceState: true });

	buttonByText('Types · Photos').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(gotoMock).toHaveBeenCalledWith('/search?kinds=image', { replaceState: true });

	buttonByText('Cameras · Apple iPhone 15').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	// The canonical URL the params writer builds (a space encodes as
	// the form-style '+').
	expect(gotoMock).toHaveBeenCalledWith('/search?cameras=Apple%7CiPhone+15', { replaceState: true });

	unmount(page);
	flushSync();
});

test('a deep link with filters carries them into the request and marks the chips selected', async () => {
	const fetchMock = stubFetch(
		searchRoutes({
			'/api/v1/search?personIds=p1&locale=en&limit=100': () =>
				jsonResponse(200, { items: pageOne, facets: facetsBody })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	pageState.url = new URL('http://localhost/search?personIds=p1');
	const page = mountPage();
	await settle();

	expect(fetchMock).toHaveBeenCalledWith('/api/v1/search?personIds=p1&locale=en&limit=100', undefined);
	const peopleChip = document.body.querySelector<HTMLButtonElement>('.search-chip');
	expect(peopleChip?.textContent).toContain('People · Mom');
	expect(peopleChip?.getAttribute('aria-pressed')).toBe('true');

	unmount(page);
	flushSync();
});

test('submitting the query box navigates with the query parameter', async () => {
	vi.stubGlobal('fetch', stubFetch(searchRoutes()));
	const page = mountPage();
	await settle();

	const box = document.body.querySelector<HTMLInputElement>('input.search-query');
	if (box === null) {
		throw new Error('missing query box');
	}
	box.value = '海边 日落';
	box.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	const form = document.body.querySelector('form.search-form');
	form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));

	expect(gotoMock).toHaveBeenCalledWith('/search?query=%E6%B5%B7%E8%BE%B9+%E6%97%A5%E8%90%BD', {
		replaceState: true
	});

	unmount(page);
	flushSync();
});

test('the manual pager walks the next page and appends the tiles', async () => {
	const fetchMock = stubFetch(searchRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Load more').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(tiles()).toHaveLength(4);
	expect(wallItems().map((entry) => entry.id)).toEqual(['a1', 'a2', 'a3', 'b1']);
	// The exhausted walk drops the pager and settles the count.
	expect(document.body.querySelector('.search-meta')?.textContent).toContain('4 results');

	unmount(page);
	flushSync();
});

test('no results render the empty state with the honest suggestion', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/search?locale=en&limit=100': () =>
				jsonResponse(200, {
					items: [],
					facets: { people: [], places: [], years: [], cameras: [], kinds: [], albums: [], tags: [] }
				})
		})
	);
	const page = mountPage();
	await settle();

	expect(tiles()).toHaveLength(0);
	const empty = document.body.querySelector('.search-empty');
	expect(empty?.textContent).toContain('No matching results');
	expect(empty?.textContent).toContain('Try different words, or clear some filters.');

	unmount(page);
	flushSync();
});

test('a failed first load shows the error box with a working retry', async () => {
	let failed = true;
	const fetchMock = vi.fn(async (input: RequestInfo | URL): Promise<Response> => {
		if (String(input).startsWith('/api/v1/search') && failed) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return jsonResponse(200, { items: pageOne, facets: facetsBody });
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(document.body.textContent).toContain('Something went wrong');

	failed = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(tiles()).toHaveLength(3);

	unmount(page);
	flushSync();
});

test('an SSE event re-executes the current search (the refetch is the convergence)', async () => {
	let calls = 0;
	const fetchMock = vi.fn(async (input: RequestInfo | URL): Promise<Response> => {
		if (String(input).includes('cursor')) {
			return jsonResponse(200, { items: pageTwo, facets: facetsBody });
		}
		calls += 1;
		return jsonResponse(
			200,
			calls > 1
				? { items: [item('fresh', '2026-11-06T08:00:00Z')], facets: facetsBody }
				: { items: pageOne, nextCursor: 'C1', facets: facetsBody }
		);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(tiles()).toHaveLength(3);

	for (const handler of handlers) {
		handler({ id: 1, name: 'asset.ready', assetId: 'a1', kind: 'image' });
	}
	flushSync();
	await settle();

	// The fresh page one replaced the walk wholesale (no patching).
	expect(tiles()).toHaveLength(1);
	expect(wallItems().map((entry) => entry.id)).toEqual(['fresh']);
	expect(calls).toBe(2);

	unmount(page);
	flushSync();
});

test('the locale rides the request (the places facet display names)', async () => {
	const fetchMock = stubFetch(
		searchRoutes({
			'/api/v1/search?locale=zh-Hans&limit=100': () =>
				jsonResponse(200, { items: pageOne, facets: facetsBody })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	setLocale('zh-Hans');
	const page = mountPage();
	await settle();

	expect(fetchMock).toHaveBeenCalledWith('/api/v1/search?locale=zh-Hans&limit=100', undefined);
	setLocale('en');

	unmount(page);
	flushSync();
});

test('the zh-Hans locale renders through the shared table', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			searchRoutes({
				'/api/v1/search?locale=zh-Hans&limit=100': () =>
					jsonResponse(200, { items: pageOne, facets: facetsBody })
			})
		)
	);
	setLocale('zh-Hans');
	const page = mountPage();
	await settle();

	const chips = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.search-chip'));
	expect(chips[0]?.textContent).toContain('人物 · Mom');
	expect(document.body.querySelector('.search-meta')?.textContent).toContain('每');
	setLocale('en');

	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from the event stream', async () => {
	vi.stubGlobal('fetch', stubFetch(searchRoutes()));
	const page = mountPage();
	await settle();
	expect(handlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(handlers).toHaveLength(0);
});
