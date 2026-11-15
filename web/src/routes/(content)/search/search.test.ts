import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems, wallItems } from '$lib/timeline/wall-sequence.svelte';

// The search page: the URL owns the state (the deep-link contract),
// the two result states render ONE wall (the flat justified rank
// order -- the query's matches and the browse's recent feed are the
// same wall wearing different orders), the context bar carries
// the centered field and the honest count, an empty query is the
// recent-photos browse (never the no-results card), the manual
// pager walks the cursor, the empty and error states carry their
// affordances, the SSE events re-execute the current search (the
// pinned convergence), and the loaded items publish as the viewer's
// navigation sequence. $app/state is mocked for the URL;
// $app/navigation captures the goto the URL sync issues; the
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
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

function searchRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/search?locale=en&limit=100': () =>
			jsonResponse(200, { items: pageOne, nextCursor: 'C1', facets: facetsBody }),
		'/api/v1/search?locale=en&limit=100&cursor=C1': () => jsonResponse(200, { items: pageTwo, facets: facetsBody }),
		...overrides
	};
}

function stubFetch(route: FetchRoute): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
		const url = String(input);
		const answer = route[url];
		if (answer === undefined) {
			throw new Error(`unexpected fetch ${url}`);
		}
		return answer(init);
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

/** The context bar's count idiom: the wording lives on the
 * accessible name (glyph + digits render). */
function countLabel(): string | null {
	return document.body.querySelector<HTMLElement>('.search-bar .count')?.getAttribute('aria-label') ?? null;
}

function buttonByText(text: string, scope: ParentNode = document.body): HTMLButtonElement {
	const buttons = Array.from(scope.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.includes(text));
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

/** The icon-only action buttons carry their name on the aria-label. */
function buttonByLabel(label: string, scope: ParentNode = document.body): HTMLButtonElement {
	const found = scope.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing button labeled ${label}`);
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
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the empty query is the recent-photos browse: the one flat wall, the hint line, the total count', async () => {
	const fetchMock = stubFetch(searchRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The request is the pure browse (no query parameter).
	expect(fetchMock).toHaveBeenCalledWith('/api/v1/search?locale=en&limit=100', undefined);
	// BROWSE state: the SAME flat wall as the query state -- the
	// recent feed wears the rank order, no month or day grouping.
	expect(document.body.querySelectorAll('.wall__month-header')).toHaveLength(0);
	expect(document.body.querySelectorAll('.wall__day-header')).toHaveLength(0);
	expect(tiles()).toHaveLength(3);
	// The per-user subset rides the wall: the check affordance is
	// wired (the hover-reveal entry; a selection raises the bar) and
	// the bar carries no buttons while nothing is selected.
	expect(document.body.querySelector('.tile__check')).not.toBeNull();
	expect(document.body.querySelector('.batchbar')).toBeNull();
	// The loaded page items publish as the viewer's sequence.
	expect(wallItems().map((entry) => entry.id)).toEqual(['a1', 'a2', 'a3']);

	// The browse hint: the one line that says what the empty query
	// is looking at.
	expect(document.body.querySelector('.search__hint')?.textContent).toContain(
		'Without a keyword, these are the most recent photos.'
	);

	// The bar's count: the browse wording over the loaded count (a
	// page follows, so the "so far" arm).
	expect(countLabel()).toBe('3 photos so far');

	// The query is the filter: no facet affordance rides this page.
	expect(document.body.querySelectorAll('.search-chip')).toHaveLength(0);

	unmount(page);
	flushSync();
});

test('the context bar carries the identity, the centered field and the back link; no shell chrome', async () => {
	vi.stubGlobal('fetch', stubFetch(searchRoutes()));
	const page = mountPage();
	await settle();

	// The bar's own skeleton: back link, the 20px title, the field.
	expect(document.body.querySelector('.search-bar__back')?.getAttribute('aria-label')).toBe('Back to the timeline');
	expect(document.body.querySelector('.search-bar__title')?.textContent).toBe('Search');
	expect(document.body.querySelector('.search-bar__field input.search-query')).not.toBeNull();

	// The wall is the body: no facet chips, no caliber note.
	expect(document.body.querySelectorAll('.search-chip')).toHaveLength(0);
	expect(document.body.querySelector('.search-meta')).toBeNull();

	unmount(page);
	flushSync();
});

test('the walls curate the per-user subset: favorite + tag on the bar, the batch write lands, every role', async () => {
	// The curation ruling: BOTH search walls carry the action subset
	// [favorite, tag] for EVERY role -- the ownership actions
	// (hide/trash/restore) stay off the bar, the endpoint's
	// owner-per-caller resolution applies to the admin too, and the
	// honest missing report handles the edge.
	let batchBody: unknown = null;
	const fetchMock = stubFetch(
		searchRoutes({
			'/api/v1/assets/batch': (init?: RequestInit) => {
				batchBody = JSON.parse(String(init?.body));
				return jsonResponse(200, { affected: 1, missing: [] });
			},
			'/api/v1/tags': () => jsonResponse(200, { tags: [] }),
			'/api/v1/albums/mem1/items': () => jsonResponse(200, { added: 1, skipped: 0 }),
			'/api/v1/albums': () =>
				jsonResponse(200, {
					albums: [
						{
							id: 'mem1',
							title: 'Beach 2026',
							type: 'memory',
							isShared: false,
							coverAssetIds: [],
							itemCount: 4,
							photoCount: 4,
							updatedAt: '2026-11-05T10:15:00Z',
							callerRole: 'owner'
						}
					]
				})
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The check entry is the hover affordance; the first click
	// enters the selection and the bar rises.
	tiles()[0]
		?.querySelector<HTMLElement>('.tile__check')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const bar = document.body.querySelector<HTMLElement>('.batchbar');
	expect(bar).not.toBeNull();
	expect(bar?.getAttribute('role')).toBe('toolbar');
	// The subset: favorite + tag + the memory entry; the ownership
	// actions absent.
	expect(() => buttonByLabel('Favorite', bar as ParentNode)).not.toThrow();
	expect(() => buttonByLabel('Add tag', bar as ParentNode)).not.toThrow();
	expect(() => buttonByLabel('Add to memory', bar as ParentNode)).not.toThrow();
	for (const label of ['Hide', 'Unhide', 'Trash', 'Restore']) {
		expect(() => buttonByLabel(label, bar as ParentNode), label).toThrow();
	}

	// The favorite batch runs through the client: the optimistic
	// flip, the POST, and the success arm's clear (the bar drops).
	buttonByLabel('Favorite', bar as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(batchBody).toEqual({ action: 'favorite', ids: ['a1'] });
	expect(document.body.querySelector('.batchbar')).toBeNull();

	// The tag flow rides the opener: the dialog opens over the ids
	// captured at open (the vocabulary fetches fresh).
	tiles()[1]
		?.querySelector<HTMLElement>('.tile__check')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	buttonByLabel('Add tag').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(document.body.querySelector('.modal__panel')?.textContent).toContain('Add tag');
	expect(fetchMock.mock.calls.some(([input]) => String(input) === '/api/v1/tags')).toBe(true);
	// Close the tag dialog before the memory step (one modal at a
	// time; the wall keeps its selection through the cancel).
	buttonByText('Cancel').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The memory entry rides the same opener shape: the picker opens
	// over the ids captured at open (the memory list fetches fresh),
	// and a row pick adds them through the items face.
	tiles()[2]
		?.querySelector<HTMLElement>('.tile__check')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	buttonByLabel('Add to memory').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(document.body.querySelector('.modal__panel')?.textContent).toContain('Add to memory');
	expect(document.body.querySelector('.modal__panel')?.textContent).toContain('Beach 2026');
	const memoryRow = document.body.querySelector<HTMLElement>('.add-memory__row');
	memoryRow?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	await settle();
	expect(
		fetchMock.mock.calls.some(
			([input, init]) => String(input) === '/api/v1/albums/mem1/items' && init?.method === 'POST'
		)
	).toBe(true);
	// The pass's Done clears the selection (the success arm): the bar
	// drops.
	buttonByText('Done').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(page);
	flushSync();
});

test('a text query renders the flat justified wall: rank order, no month or day headers, the match count', async () => {
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

	expect(fetchMock).toHaveBeenCalledWith('/api/v1/search?query=%E6%B5%B7%E8%BE%B9&locale=en&limit=100', undefined);
	// QUERY state: the flat wall -- no month or day grouping (the
	// grouping would lie about the rank order).
	expect(document.body.querySelectorAll('.flat-wall')).toHaveLength(1);
	expect(document.body.querySelectorAll('.wall__month-header')).toHaveLength(0);
	expect(document.body.querySelectorAll('.wall__day-header')).toHaveLength(0);
	expect(tiles()).toHaveLength(3);
	// The flat wall carries the same per-user subset: the check
	// affordance is wired (its own organ), no bar while nothing is
	// selected.
	expect(document.body.querySelector('.tile__check')).not.toBeNull();
	expect(document.body.querySelector('.batchbar')).toBeNull();
	// The rank order is the wall's sequence (the viewer walks it).
	expect(wallItems().map((entry) => entry.id)).toEqual(['a1', 'a2', 'a3']);
	// The count follows the query: the match wording, settled (the
	// walk exhausted).
	expect(countLabel()).toBe('3 matches');
	// The hint line belongs to the browse state only.
	expect(document.body.querySelector('.search__hint')).toBeNull();

	unmount(page);
	flushSync();
});

test('a one-rune query takes no text path: the request and the wall mirror the browse state', async () => {
	const fetchMock = stubFetch(searchRoutes());
	vi.stubGlobal('fetch', fetchMock);
	pageState.url = new URL('http://localhost/search?query=x');
	const page = mountPage();
	await settle();

	// The server's own rule, mirrored: the request carries no query.
	expect(fetchMock).toHaveBeenCalledWith('/api/v1/search?locale=en&limit=100', undefined);
	// The honest render: the one flat wall (the browse order).
	expect(document.body.querySelectorAll('.flat-wall')).toHaveLength(1);
	expect(document.body.querySelectorAll('.wall__month-header')).toHaveLength(0);
	// The typed text stays in the box (the URL state).
	expect((document.body.querySelector('input.search-query') as HTMLInputElement).value).toBe('x');
	// The browse face: the hint line rides the wall.
	expect(document.body.querySelector('.search__hint')).not.toBeNull();

	unmount(page);
	flushSync();
});

test('a deep-linked filter rides the request (the URL state survives the chipless page)', async () => {
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
	// The filter narrows nothing on the page: no chip row renders,
	// the wall is the one flat wall over the filtered set.
	expect(document.body.querySelectorAll('.search-chip')).toHaveLength(0);
	expect(document.body.querySelectorAll('.flat-wall')).toHaveLength(1);
	// A filter-only deep link is a RESULT face: the match count (not
	// the bare browse's "All"), and no "recent photos" hint over a
	// narrowed wall.
	expect(countLabel()).toBe('3 matches');
	expect(document.body.querySelector('.search__hint')).toBeNull();

	unmount(page);
	flushSync();
});

test('a filter that answers nothing gets the no-results card, not the no-data lie', async () => {
	const fetchMock = stubFetch(
		searchRoutes({
			'/api/v1/search?personIds=p1&locale=en&limit=100': () =>
				jsonResponse(200, { items: [], facets: facetsBody })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	pageState.url = new URL('http://localhost/search?personIds=p1');
	const page = mountPage();
	await settle();

	// The library holds photos; THIS narrowing emptied it -- the
	// no-results face (never "No photos yet" over a filtered wall).
	const card = document.body.querySelector('.state-card');
	expect(card?.textContent).toContain('No matching photos');
	expect(card?.textContent).not.toContain('No photos yet');

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
	const form = document.body.querySelector('form.search-bar__field');
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
	// The exhausted walk drops the pager and settles the count: the
	// browse wording claims the whole library it just loaded.
	expect(countLabel()).toBe('All 4 photos');

	unmount(page);
	flushSync();
});

test('a query that answers nothing renders the no-results card; an empty browse renders the no-data card', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL): Promise<Response> => {
		const url = String(input);
		const empty = {
			items: [],
			facets: { people: [], places: [], years: [], cameras: [], kinds: [], albums: [], tags: [] }
		};
		if (url === '/api/v1/search?query=%E6%B5%B7%E8%BE%B9&locale=en&limit=100') {
			return jsonResponse(200, empty);
		}
		if (url === '/api/v1/search?locale=en&limit=100') {
			return jsonResponse(200, empty);
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);

	// QUERY state, zero answers: the no-results card.
	pageState.url = new URL('http://localhost/search?query=%E6%B5%B7%E8%BE%B9');
	const first = mountPage();
	await settle();
	expect(tiles()).toHaveLength(0);
	const empty = document.body.querySelector('.state-card');
	expect(empty?.textContent).toContain('No matching photos');
	expect(empty?.textContent).toContain('Try a different keyword, or drop a filter.');
	expect(countLabel()).toBe('0 matches');
	unmount(first);
	flushSync();

	// BROWSE state, nothing to show: the no-data face -- an empty
	// query is never told "no matching photos".
	document.body.textContent = '';
	pageState.url = new URL('http://localhost/search');
	const second = mountPage();
	await settle();
	const browseEmpty = document.body.querySelector('.state-card');
	expect(browseEmpty?.textContent).toContain('No photos yet');
	expect(browseEmpty?.textContent).not.toContain('No matching photos');
	unmount(second);
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

test('the locale rides the request (the display names)', async () => {
	const fetchMock = stubFetch(
		searchRoutes({
			'/api/v1/search?locale=zh-Hans&limit=100': () => jsonResponse(200, { items: pageOne, facets: facetsBody })
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

	// The hint line and the count wording come from the shared
	// table: the browse face reads as the recent photos, the whole
	// library it loaded.
	expect(document.body.querySelector('.search__hint')?.textContent).toContain('没输入关键词时');
	expect(countLabel()).toBe('全部 3 张');
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
