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
// the batch bar with optimistic convergence, and SSE events patch
// the loaded wall. The events module is mocked with a captured
// handler registry; $app/navigation is not used by this page, but
// the tile anchors rely on the router only in a real app.

const { handlers } = vi.hoisted(() => ({
	handlers: [] as Array<(event: unknown) => void>
}));

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
const pageTwo = [item('b1', '2026-09-05T08:00:00Z'), item('b2', '2026-08-22T18:00:00Z')];

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, () => Response>;

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
	return vi.fn(async (input: RequestInfo | URL) => {
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

function textOf(selector: string): string {
	return document.body.querySelector(selector)?.textContent ?? '';
}

function tiles(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.tile'));
}

function buttonByText(text: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
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

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	vi.clearAllMocks();
	clearWallItems();
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

	const monthHeaders = Array.from(document.body.querySelectorAll('.timeline__month-header'));
	expect(monthHeaders.map((header) => header.textContent)).toEqual([
		'November 2026',
		'October 2026'
	]);
	const dayHeaders = Array.from(document.body.querySelectorAll('.timeline__day-header'));
	expect(dayHeaders.map((header) => header.textContent)).toEqual([
		'November 2',
		'November 1',
		'October 13'
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
	const months = Array.from(document.body.querySelectorAll('.timeline__month-header'));
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

test('selection: the bar rises with the count, a batch trash removes the tiles', async () => {
	const fetchMock = stubFetch(
		timelineRoutes({
			'/api/v1/assets/batch': () => jsonResponse(200, { affected: 2, missing: [] })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.batchbar')).toBeNull();

	// Enter the select mode (the touch-friendly path) and pick two.
	buttonByText('Select').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
	clickTile(0);
	clickTile(1);

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

	buttonByText('Select').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
	clickTile(0);

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

	buttonByText('Select').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
	clickTile(0);

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
	expect(handlers).toHaveLength(1);

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
