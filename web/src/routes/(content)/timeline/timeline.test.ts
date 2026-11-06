import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems, wallItems } from '$lib/timeline/wall.svelte';

// The timeline wall page: first page renders grouped tiles, the
// empty and error states carry their affordances, the infinite
// scroll burst collapses into one fetch (the M2d review focus), the
// manual pager covers hosts without an observer, selection raises
// the batch bar with optimistic convergence, the batch bar's
// add-to-album dialog adds the selection to an existing album or
// creates one first, and SSE events patch the loaded wall (the
// album.* events restart the walk: the union's membership arms).
// $app/state is mocked for the ?album= preselect; the events module
// is mocked with a captured handler registry; $app/navigation is not
// used by this page, but the tile anchors rely on the router only in
// a real app.

const { pageState, handlers } = vi.hoisted(() => ({
	pageState: {
		params: {} as Record<string, string>,
		url: new URL('http://localhost/timeline')
	},
	handlers: [] as Array<(event: unknown) => void>
}));

vi.mock('$app/state', () => ({ page: pageState }));
vi.mock('$lib/events', () => ({
	subscribeAssetEvents: (listener: (event: unknown) => void) => {
		handlers.push(listener);
		return () => {
			const index = handlers.indexOf(listener);
			if (index !== -1) {
				handlers.splice(index, 1);
			}
		};
	},
	subscribeAlbumEvents: (listener: (event: unknown) => void) => {
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
const pageTwo = [item('b1', '2026-09-05T08:00:00Z'), item('b2', '2026-08-22T18:00:00Z')];

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

function timelineRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/assets/timeline?limit=100': () => jsonResponse(200, { items: pageOne, nextCursor: 'C1' }),
		'/api/v1/assets/timeline?limit=100&cursor=C1': () =>
			jsonResponse(200, { items: pageTwo }),
		'/api/v1/assets/batch': () => jsonResponse(200, { affected: 0, missing: [] }),
		...overrides
	};
}

function stubFetch(route: FetchRoute): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
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

function textOf(selector: string): string {
	return document.body.querySelector(selector)?.textContent ?? '';
}

function tiles(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.tile'));
}

function buttonByText(text: string, scope: ParentNode = document.body): HTMLButtonElement {
	const buttons = Array.from(scope.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

function clickTile(index: number): void {
	const link = tiles()[index]?.querySelector('a.tile__link');
	link?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

/**
 * Click one tile's check affordance -- R2-11's selection entry: the
 * check reveals on hover/keyboard focus on every tile (the toolbar's
 * select-mode button is gone), and the first click enters the
 * selection.
 */
function clickTileCheck(index: number): void {
	const check = tiles()[index]?.querySelector<HTMLButtonElement>('.tile__check');
	check?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	vi.clearAllMocks();
	clearWallItems();
	pageState.url = new URL('http://localhost/timeline');
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

test('the first page renders sticky month and day groups of tiles', async () => {
	const fetchMock = stubFetch(timelineRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The wall's month and day headers: the shared justified wall
	// renderer (MonthGroups) owns the header classes now -- the
	// selectors follow the wall__ prefix the curated views always
	// used. R2-12: no dividing hairline, the tightened vertical
	// rhythm (the month band's ~py-1.5/mb-2), the sticky positions
	// and the opaque bg-page stays.
	const monthHeader = document.body.querySelector('.wall__month-header');
	expect(monthHeader?.className).not.toContain('border-b');
	expect(monthHeader?.className).toContain('py-1.5');
	expect(monthHeader?.className).toContain('mb-2');
	expect(monthHeader?.className).toContain('sticky');
	expect(monthHeader?.className).toContain('bg-page');
	expect(monthHeader?.className).toContain('mt-(--wall-month-gap)');
	const dayHeader = document.body.querySelector('.wall__day-header');
	expect(dayHeader?.className).not.toContain('border-b');
	expect(dayHeader?.className).toContain('sticky');
	expect(dayHeader?.className).toContain('bg-page');
	// The day header stacks under the tightened month band exactly
	// (44px -> 38px: py-2.5 -> py-1.5 over the same 26px line).
	expect(dayHeader?.className).toContain('top-[38px]');
	const monthHeaders = Array.from(document.body.querySelectorAll('.wall__month-header'));
	// The rich date labels: the day header carries the weekday; the
	// month header stays the bare month label.
	expect(monthHeaders.map((header) => header.textContent)).toEqual([
		'November 2026',
		'October 2026'
	]);
	const dayHeaders = Array.from(document.body.querySelectorAll('.wall__day-header'));
	expect(dayHeaders.map((header) => header.textContent)).toEqual([
		'Monday, November 2',
		'Sunday, November 1',
		'Tuesday, October 13'
	]);
	expect(tiles()).toHaveLength(3);
	expect(tiles()[0]?.querySelector('img')?.getAttribute('src')).toBe(
		'/api/v1/assets/a1/thumbnail?size=tile'
	);

	// The scrubber carries the loaded months.
	const stops = Array.from(document.body.querySelectorAll('.scrubber__stop'));
	expect(stops.map((stop) => stop.textContent)).toEqual(['November 2026', 'October 2026']);

	unmount(page);
	flushSync();
});

test('tiles pack into justified rows at the engine-computed sizes', async () => {
	vi.stubGlobal('fetch', stubFetch(timelineRoutes()));
	const page = mountPage();
	await settle();

	// jsdom has neither a ResizeObserver nor box metrics, so the wall
	// packs against the content-column fallback width (1200px): every
	// day group holds ONE 4:3 item, whose natural row height (900px)
	// is above the 216px target, so each row is the clamped tail --
	// height exactly the target, tile width 216 * 4/3 = 288px.
	const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.wall__row'));
	expect(rows).toHaveLength(3);
	for (const row of rows) {
		expect(row.style.height).toBe('216px');
		expect(row.style.gap).toBe('2px');
	}
	const cells = Array.from(document.body.querySelectorAll<HTMLElement>('.wall__cell'));
	expect(cells).toHaveLength(3);
	expect(cells[0]?.style.width).toBe('288px');

	// The tiles fill their cells at the true aspect: the engine's
	// width over the row height is the item's sensor ratio.
	const tile = cells[0]?.querySelector<HTMLElement>('.tile');
	expect(tile).not.toBeNull();

	// The wall passes the engine's exact cell width as the tile's
	// sizes hint (the srcset's honest width source).
	const image = cells[0]?.querySelector('img');
	expect(image?.getAttribute('sizes')).toBe('288px');
	unmount(page);
	flushSync();
});

test('an empty timeline renders the empty state, no tiles, no scrubber', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/assets/timeline?limit=100': () => jsonResponse(200, { items: [] }) })
	);
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.timeline__empty')?.textContent).toContain(
		'Your timeline is empty.'
	);
	expect(tiles()).toHaveLength(0);
	expect(document.body.querySelector('.scrubber')).toBeNull();
	unmount(page);
	flushSync();
});

test('a failed first load shows the error box with a working retry', async () => {
	let failing = true;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (failing && String(input).startsWith('/api/v1/assets/timeline')) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return jsonResponse(200, { items: pageOne });
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.timeline__failure');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('Something went wrong.');
	expect(failure?.textContent).toContain('Try again');

	failing = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(tiles()).toHaveLength(3);
	unmount(page);
	flushSync();
});

test('an unknown error code renders the generic message, never raw code jargon', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/assets/timeline?limit=100': () => jsonResponse(500, { code: 'PROXY_TEAPOT' })
		})
	);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.timeline__failure');
	expect(failure?.textContent).toContain('Something went wrong.');
	expect(failure?.textContent).not.toContain('proxy_teapot');
	unmount(page);
	flushSync();
});

test('a dead session renders the session message, never credential text', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/assets/timeline?limit=100': () =>
				jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })
		})
	);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.timeline__failure');
	expect(failure?.textContent).toContain('Your session has ended.');
	expect(failure?.textContent).not.toContain('Incorrect email or password');
	unmount(page);
	flushSync();
});

test('a burst of sentinel intersections collapses into one fetch (the guard)', async () => {
	type Callback = IntersectionObserverCallback;
	const observers: Array<{ callback: Callback }> = [];
	class MockIntersectionObserver {
		constructor(callback: Callback) {
			observers.push({ callback });
		}
		observe(): void {}
		unobserve(): void {}
		disconnect(): void {}
	}
	vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);

	const fetchMock = stubFetch(timelineRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(fetchMock).toHaveBeenCalledTimes(1);

	// Fast scroll: the sentinel re-intersects twice in the same
	// breath while the first follow-up page is still in flight.
	expect(observers).toHaveLength(1);
	const observer = observers[0];
	const entry = { isIntersecting: true } as IntersectionObserverEntry;
	observer?.callback([entry], {} as IntersectionObserver);
	observer?.callback([entry], {} as IntersectionObserver);
	await settle();

	const timelineCalls = fetchMock.mock.calls.filter(([input]) =>
		String(input).startsWith('/api/v1/assets/timeline')
	);
	// The initial page plus exactly ONE follow-up: the two
	// intersections of the burst collapsed into a single fetch (a
	// missing guard would show three).
	expect(timelineCalls).toHaveLength(2);
	expect(String(timelineCalls[1]?.[0])).toBe('/api/v1/assets/timeline?limit=100&cursor=C1');
	unmount(page);
	flushSync();
});

test('without an observer the manual pager loads the next page', async () => {
	// jsdom has no IntersectionObserver: the fallback button shows.
	expect(typeof globalThis.IntersectionObserver).toBe('undefined');
	const fetchMock = stubFetch(timelineRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	const more = buttonByText('Load more');
	more.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	expect(tiles()).toHaveLength(5);
	// (The wall__ prefix: the shared MonthGroups renderer owns the
	// headers of every wall.)
	const months = Array.from(document.body.querySelectorAll('.wall__month-header'));
	expect(months.map((header) => header.textContent)).toEqual([
		'November 2026',
		'October 2026',
		'September 2026',
		'August 2026'
	]);
	// The last page carries no cursor: the pager is gone.
	expect(() => buttonByText('Load more')).toThrow();
	unmount(page);
	flushSync();
});

test('selection: the check entry raises the bar, a batch trash removes the tiles', async () => {
	const fetchMock = stubFetch(
		timelineRoutes({
			'/api/v1/assets/batch': () => jsonResponse(200, { affected: 2, missing: [] })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// R2-11: the toolbar is gone -- no hidden-view link, no
	// select-mode button -- and the check affordance is the entry.
	expect(document.body.querySelector('.timeline__toolbar')).toBeNull();
	expect(() => buttonByText('Select')).toThrow();
	expect(() => buttonByText('Hidden')).toThrow();
	expect(document.body.querySelector('a[href="/hidden"]')).toBeNull();

	// Enter the selection through the tiles' checks and pick two (the
	// affordance's click toggles the id; the bar rises at >=1).
	clickTileCheck(0);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');
	clickTileCheck(1);

	const bar = document.body.querySelector('.batchbar');
	expect(bar?.textContent).toContain('2 selected');

	buttonByText('Trash').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/batch');
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({ action: 'trash', ids: ['a1', 'a2'] });
	expect(tiles().map((tile) => tile.querySelector('img')?.getAttribute('src'))).toEqual([
		'/api/v1/assets/a3/thumbnail?size=tile'
	]);
	// The successful action cleared the selection: the bar is gone.
	expect(document.body.querySelector('.batchbar')).toBeNull();
	unmount(page);
	flushSync();
});

test('a failed batch keeps the tiles, the selection and shows the message', async () => {
	const fetchMock = stubFetch(
		timelineRoutes({
			'/api/v1/assets/batch': () => jsonResponse(500, { code: 'INTERNAL' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// R2-11: the check entry (the select-mode button is gone from the
	// toolbar).
	clickTileCheck(0);

	buttonByText('Hide').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	expect(tiles()).toHaveLength(3);
	expect(document.body.querySelector('.batchbar__error')?.textContent).toBe(
		'Something went wrong. Please try again.'
	);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');

	// Escape clears the selection and exits the select mode.
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
	await tick();
	expect(document.body.querySelector('.batchbar')).toBeNull();
	expect(tiles()[0]?.className).not.toContain('tile--selectable');
	unmount(page);
	flushSync();
});

test('favorite toggles the selected tiles optimistically', async () => {
	const fetchMock = stubFetch(
		timelineRoutes({
			'/api/v1/assets/batch': () => jsonResponse(200, { affected: 1, missing: [] })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// R2-11: the check entry (the select-mode button is gone from the
	// toolbar).
	clickTileCheck(0);

	buttonByText('Favorite').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	expect(tiles()[0]?.querySelector('.tile__badge--favorite')).not.toBeNull();
	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/batch');
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({ action: 'favorite', ids: ['a1'] });
	unmount(page);
	flushSync();
});

test('SSE events patch the wall: a trashed event removes the tile', async () => {
	vi.stubGlobal('fetch', stubFetch(timelineRoutes()));
	const page = mountPage();
	await settle();
	expect(tiles()).toHaveLength(3);

	for (const handler of [...handlers]) {
		handler({ id: 1, name: 'asset.trashed', assetId: 'a2', kind: 'image' });
	}
	await settle();

	expect(tiles().map((tile) => tile.querySelector('img')?.getAttribute('src'))).toEqual([
		'/api/v1/assets/a1/thumbnail?size=tile',
		'/api/v1/assets/a3/thumbnail?size=tile'
	]);
	unmount(page);
	flushSync();
});

test('SSE events patch the wall: a ready event refetches the detail', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			timelineRoutes({
				'/api/v1/assets/a2': () =>
					jsonResponse(200, {
						...item('a2', '2026-11-01T09:00:00Z', { isReady: true }),
						isHidden: false,
						description: '',
						exif: {}
					})
			})
		)
	);
	const page = mountPage();
	await settle();

	for (const handler of [...handlers]) {
		handler({ id: 1, name: 'asset.ready', assetId: 'a2', kind: 'image' });
	}
	await settle();

	const tile = tiles()[1];
	expect(tile?.querySelector('img')?.getAttribute('src')).toBe(
		'/api/v1/assets/a2/thumbnail?size=tile'
	);
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from the event stream', async () => {
	vi.stubGlobal('fetch', stubFetch(timelineRoutes()));
	const page = mountPage();
	await settle();
	// The asset listener and the album listener.
	expect(handlers).toHaveLength(2);

	unmount(page);
	flushSync();
	expect(handlers).toHaveLength(0);
});

test('the wall publishes its loaded page items as the viewer sequence', async () => {
	// The /a/{id} viewer walks the wall's published snapshot (see
	// $lib/timeline/wall.svelte): the page publishes while mounted,
	// and the snapshot outlives the route so the viewer reads it
	// after the navigation.
	clearWallItems();
	vi.stubGlobal('fetch', stubFetch(timelineRoutes()));
	const page = mountPage();
	await settle();
	expect(wallItems().map((wallItem) => wallItem.id)).toEqual(['a1', 'a2', 'a3']);

	unmount(page);
	flushSync();
	expect(wallItems().map((wallItem) => wallItem.id)).toEqual(['a1', 'a2', 'a3']);
	clearWallItems();
});

test('the scrubber jumps to a month group', async () => {
	vi.stubGlobal('fetch', stubFetch(timelineRoutes()));
	const scrolled: HTMLElement[] = [];
	Element.prototype.scrollIntoView = function (this: HTMLElement) {
		scrolled.push(this);
	} as unknown as Element['scrollIntoView'];
	const page = mountPage();
	await settle();

	const stops = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop'));
	stops[1]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();

	expect(scrolled).toHaveLength(1);
	expect(scrolled[0]?.dataset.monthKey).toBe('2026-10');
	Reflect.deleteProperty(Element.prototype, 'scrollIntoView');
	unmount(page);
	flushSync();
});

test('the scrubber tracking observes the scroll root through the capture phase', async () => {
	// R2-1's risky edge: the frame moved scrolling from the document
	// into the shell's <main>, and element scroll events DO NOT
	// BUBBLE -- a window-level onscroll would silently die. The
	// tracking must observe the scroll root some other way, and the
	// capture-phase registration is the one that reaches every
	// container under the document (a jsdom mount has no <main>, so
	// a scroll dispatched on any descendant stands in for it).
	vi.stubGlobal('fetch', stubFetch(timelineRoutes()));
	const page = mountPage();
	await settle();

	const sections = Array.from(document.body.querySelectorAll<HTMLElement>('[data-month-key]'));
	expect(sections.map((section) => section.dataset.monthKey)).toEqual(['2026-11', '2026-10']);

	// A scrolled-forward geometry: both groups above the anchor, the
	// older one last-past (the layout-less host's rects are zero, so
	// the rects are stubbed the way the wall's own test does).
	const rects = [
		{ top: -300, height: 600 } as DOMRect,
		{ top: -50, height: 600 } as DOMRect
	];
	sections.forEach((section, index) => {
		vi.spyOn(section, 'getBoundingClientRect').mockReturnValue(rects[index]!);
	});

	// The non-bubbling scroll event of the scroll container.
	sections[1]!.dispatchEvent(new Event('scroll'));
	await new Promise((resolve) => requestAnimationFrame(resolve));
	flushSync();

	const active = document.body.querySelector('.scrubber__stop[aria-current="true"]');
	expect(active?.textContent).toBe('October 2026');
	unmount(page);
	flushSync();
});

// --- the batch bar's add-to-album flow (M3c Task 5) ---------------------------

function modal(): HTMLElement | null {
	return document.body.querySelector<HTMLElement>('.modal__panel');
}

function fillDialogField(scope: ParentNode, value: string): void {
	const field = scope.querySelector<HTMLInputElement>("input:not([type='radio'])");
	if (field === null) {
		throw new Error('missing dialog input');
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

function chooseDialogOption(scope: ParentNode, value: string): void {
	const select = scope.querySelector<HTMLSelectElement>('select');
	if (select === null) {
		throw new Error('missing dialog select');
	}
	select.value = value;
	select.dispatchEvent(new Event('change', { bubbles: true }));
}

const ownAlbumsBody = {
	albums: [
		{
			id: 'al1',
			title: 'Road trip 2026',
			description: '',
			type: 'virtual',
			locked: false,
			isShared: false,
			coverAssetIds: [],
			itemCount: 0,
			updatedAt: '2026-11-01T10:00:00Z',
			callerRole: 'owner'
		},
		{
			id: 'al2',
			title: 'Weekend',
			description: '',
			type: 'virtual',
			locked: false,
			isShared: false,
			coverAssetIds: [],
			itemCount: 0,
			updatedAt: '2026-10-30T10:00:00Z',
			callerRole: 'owner'
		},
		{
			id: 'al4',
			title: 'Shared with me',
			description: '',
			type: 'virtual',
			locked: false,
			isShared: true,
			coverAssetIds: [],
			itemCount: 3,
			updatedAt: '2026-10-29T10:00:00Z',
			callerRole: 'viewer',
			owner: { id: 'u2', email: 'grace@example.com' }
		},
		{
			id: 'al5',
			title: 'The archive',
			description: '',
			type: 'directory',
			locked: false,
			isShared: false,
			coverAssetIds: [],
			itemCount: 9,
			updatedAt: '2026-10-28T10:00:00Z',
			callerRole: 'owner'
		}
	]
};

function selectTwoTiles(): void {
	// R2-11: the check entry (the select-mode button is gone from the
	// toolbar -- the tiles' checks are the only way in).
	clickTileCheck(0);
	clickTileCheck(1);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('2 selected');
}

test('add to album: picking an existing album POSTs the selection, the report lands', async () => {
	const fetchMock = stubFetch(
		timelineRoutes({
			'/api/v1/albums': () => jsonResponse(200, ownAlbumsBody),
			'/api/v1/albums/al1/items': () => jsonResponse(200, { added: 2, skipped: 0 })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	selectTwoTiles();
	buttonByText('Add to album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Add to album');
	// The pick list offers the caller's own virtual albums only.
	const select = dialog?.querySelector<HTMLSelectElement>('select');
	const options = Array.from(select?.options ?? []);
	expect(options.map((option) => option.value)).toEqual(['al1', 'al2']);
	expect(dialog?.textContent).toContain('Road trip 2026');

	chooseDialogOption(dialog as ParentNode, 'al1');
	buttonByText('Add photos', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/albums/al1/items');
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({ assetIds: ['a1', 'a2'] });
	// The honest report renders inside the dialog.
	expect(modal()?.textContent).toContain('Added 2, skipped 0.');
	buttonByText('Done', modal() as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	// The successful add cleared the selection: the bar is gone.
	expect(document.body.querySelector('.batchbar')).toBeNull();
	unmount(page);
	flushSync();
});

test('add to album: the create-new path creates the album first, then adds', async () => {
	const fetchMock = stubFetch(
		timelineRoutes({
			'/api/v1/albums': (init?: RequestInit) => {
				if (init?.method === 'POST') {
					return jsonResponse(201, {
						id: 'al-new',
						title: 'Weekend in the hills',
						description: '',
						type: 'virtual',
						locked: false,
						isShared: false,
						coverAssetIds: [],
						itemCount: 0,
						updatedAt: '2026-11-05T10:15:00Z',
						callerRole: 'owner'
					});
				}
				return jsonResponse(200, ownAlbumsBody);
			},
			'/api/v1/albums/al-new/items': () => jsonResponse(200, { added: 1, skipped: 1 })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	selectTwoTiles();
	buttonByText('Add to album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const dialog = modal();
	// The new-album mode: name it, create, then add. (A raw click
	// event does not flip a radio in jsdom -- the viewer assign
	// dialog's tests follow the same click + change pair.)
	const radios = Array.from(dialog?.querySelectorAll<HTMLInputElement>('input[type="radio"]') ?? []);
	radios.find((radio) => radio.value === 'new')?.dispatchEvent(
		new MouseEvent('click', { bubbles: true, cancelable: true })
	);
	radios.find((radio) => radio.value === 'new')?.dispatchEvent(
		new Event('change', { bubbles: true })
	);
	flushSync();
	fillDialogField(dialog as ParentNode, 'Weekend in the hills');
	buttonByText('Create and add', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const posts = fetchMock.mock.calls.filter(([input, init]) => String(input) === '/api/v1/albums' && init?.method === 'POST');
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		title: 'Weekend in the hills',
		description: ''
	});
	const adds = fetchMock.mock.calls.filter(
		([input]) => String(input) === '/api/v1/albums/al-new/items'
	);
	expect(adds).toHaveLength(1);
	expect(JSON.parse(String(adds[0]?.[1]?.body))).toEqual({ assetIds: ['a1', 'a2'] });
	expect(modal()?.textContent).toContain('Added 1, skipped 1.');
	unmount(page);
	flushSync();
});

test('a failed add keeps the selection and surfaces the message in the dialog', async () => {
	const fetchMock = stubFetch(
		timelineRoutes({
			'/api/v1/albums': () => jsonResponse(200, ownAlbumsBody),
			'/api/v1/albums/al1/items': () =>
				jsonResponse(409, { code: 'ALBUM_LOCKED', message: 'the album is locked' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	selectTwoTiles();
	buttonByText('Add to album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const dialog = modal();
	chooseDialogOption(dialog as ParentNode, 'al1');
	buttonByText('Add photos', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	expect(modal()?.textContent).toContain('This album is locked.');
	// The dialog stays on its choice; the selection survives.
	expect(modal()?.querySelector('select')).not.toBeNull();
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('2 selected');
	unmount(page);
	flushSync();
});

test('the ?album= param arms the select mode and preselects the album', async () => {
	pageState.url = new URL('http://localhost/timeline?album=al2');
	const fetchMock = stubFetch(
		timelineRoutes({
			'/api/v1/albums': () => jsonResponse(200, ownAlbumsBody),
			'/api/v1/albums/al2/items': () => jsonResponse(200, { added: 2, skipped: 0 })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The select mode armed on arrival: the checkboxes are revealed.
	expect(tiles()[0]?.className).toContain('tile--selectable');

	clickTile(0);
	clickTile(1);
	buttonByText('Add to album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The dialog preselected the route's album.
	const select = modal()?.querySelector<HTMLSelectElement>('select');
	expect(select?.value).toBe('al2');
	buttonByText('Add photos', modal() as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/albums/al2/items');
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({ assetIds: ['a1', 'a2'] });
	unmount(page);
	flushSync();
});

test('an album membership event restarts the walk: the fresh page one applies', async () => {
	let shared = false;
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (url === '/api/v1/assets/timeline?limit=100') {
			return jsonResponse(200, {
				items: shared
					? [item('s1', '2026-11-04T10:00:00Z'), ...pageOne]
					: pageOne,
				nextCursor: 'C1'
			});
		}
		if (url === '/api/v1/assets/timeline?limit=100&cursor=C1') {
			return jsonResponse(200, { items: pageTwo });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(tiles()).toHaveLength(3);

	// Someone shared an album with the caller: the union grew, and the
	// restart's fresh page one carries the new tile.
	shared = true;
	for (const handler of [...handlers]) {
		handler({ id: 2, name: 'album.shared', albumId: 'al4', userId: 'grantee-me' });
	}
	await settle();

	expect(tiles()).toHaveLength(4);
	expect(tiles()[0]?.querySelector('img')?.getAttribute('src')).toBe(
		'/api/v1/assets/s1/thumbnail?size=tile'
	);
	unmount(page);
	flushSync();
});
