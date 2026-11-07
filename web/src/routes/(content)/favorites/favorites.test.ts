import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems, wallItems } from '$lib/timeline/wall-sequence.svelte';

// The favorites page through its curated wall: keyset pages stream
// in through the load-more surface, the selection bar carries
// exactly the unfavorite action (the subset the view makes sense
// of), the unfavorite batch ends the selected rows' membership, the
// SSE events converge the wall, and the loaded items publish into
// the wall snapshot the asset viewer navigates. The events module
// is mocked with a captured handler registry, the same pattern as
// the timeline page test.

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
	checksum: '9f86d081884c7d65a2feaa0c55ad015',
	createdAt: '2026-10-29T09:12:00Z',
	takenAtSource: 'exif' as const,
	isTrashed: false,
	isFavorite: true,
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
	item('a2', '2026-11-01T09:00:00Z')
];
const pageTwo = [item('b1', '2026-09-05T08:00:00Z')];

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, () => Response>;

function favoritesRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/assets/favorites?limit=100': () =>
			jsonResponse(200, { items: pageOne, nextCursor: 'C1' }),
		'/api/v1/assets/favorites?limit=100&cursor=C1': () =>
			jsonResponse(200, { items: pageTwo }),
		'/api/v1/assets/batch': () => jsonResponse(200, { affected: 0, missing: [] }),
		...overrides
	};
}

function stubFetch(route: FetchRoute): typeof fetch {
	return async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
		const url = String(input);
		const answer = route[url];
		if (answer === undefined) {
			throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
		}
		return answer();
	};
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

function buttonByText(text: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

function hasButton(text: string): boolean {
	return Array.from(document.body.querySelectorAll<HTMLButtonElement>('button')).some(
		(button) => button.textContent?.trim() === text
	);
}

/** The icon-only actions (the bar, the toolbar toggle) name themselves by aria-label. */
function buttonByLabel(label: string): HTMLButtonElement {
	const found = document.body.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing button labeled ${label}`);
	}
	return found;
}

function hasButtonByLabel(label: string): boolean {
	return document.body.querySelector(`button[aria-label="${label}"]`) !== null;
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

test('the first page renders grouped tiles and publishes the wall sequence', async () => {
	vi.stubGlobal('fetch', stubFetch(favoritesRoutes()));
	const page = mountPage();
	await settle();

	expect(tiles()).toHaveLength(2);
	// The viewer's navigation sequence is the wall's own items.
	expect(wallItems().map((entry) => entry.id)).toEqual(['a1', 'a2']);

	unmount(page);
	flushSync();
});

test('the manual pager walks the next page', async () => {
	const fetchMock = stubFetch(favoritesRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Load more').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(tiles()).toHaveLength(3);
	expect(wallItems().map((entry) => entry.id)).toEqual(['a1', 'a2', 'b1']);
	// The exhausted wall drops the pager.
	expect(hasButton('Load more')).toBe(false);

	unmount(page);
	flushSync();
});

test('an empty favorites view renders the empty state', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/assets/favorites?limit=100': () => jsonResponse(200, { items: [] })
		})
	);
	const page = mountPage();
	await settle();

	expect(tiles()).toHaveLength(0);
	expect(document.body.textContent).toContain('No favorites yet');

	unmount(page);
	flushSync();
});

test('a failed page load renders the mapped error with a retry', async () => {
	let failed = true;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (String(input).startsWith('/api/v1/assets/favorites') && failed) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return jsonResponse(200, { items: pageOne, nextCursor: 'C1' });
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(document.body.textContent).toContain('Something went wrong');

	failed = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(tiles()).toHaveLength(2);

	unmount(page);
	flushSync();
});

test('the selection bar carries exactly the unfavorite action, and it ends the membership', async () => {
	const fetchMock = stubFetch(favoritesRoutes());
	// Capture the batch bodies the runner posts.
	const bodies: Array<{ action: string; ids: string[] }> = [];
	const capturing: typeof fetch = async (input, init) => {
		if (String(input) === '/api/v1/assets/batch' && init?.body !== undefined) {
			bodies.push(JSON.parse(String(init.body)));
		}
		return fetchMock(input, init);
	};
	vi.stubGlobal('fetch', capturing);
	const page = mountPage();
	await settle();

	// Enter the select mode and pick the first tile (the toolbar's
	// toggle is icon-only, named by its aria-label).
	buttonByLabel('Select').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	clickTile(0);
	expect(document.body.querySelector('.batchbar')).not.toBeNull();
	// The subset: the unfavorite toggle renders, none of the other
	// timeline actions.
	expect(hasButtonByLabel('Unfavorite')).toBe(true);
	expect(hasButtonByLabel('Hide')).toBe(false);
	expect(hasButtonByLabel('Unhide')).toBe(false);
	expect(hasButtonByLabel('Trash')).toBe(false);
	expect(hasButtonByLabel('Restore')).toBe(false);

	buttonByLabel('Unfavorite').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(bodies).toEqual([{ action: 'unfavorite', ids: ['a1'] }]);
	// The membership ended: the tile left the wall.
	expect(tiles()).toHaveLength(1);

	unmount(page);
	flushSync();
});

test('SSE restored and deleted events converge the wall', async () => {
	vi.stubGlobal('fetch', stubFetch(favoritesRoutes()));
	const page = mountPage();
	await settle();
	expect(tiles()).toHaveLength(2);

	type DeliveredEvent = { id: number; name: string; assetId: string; kind: 'image' | 'video' };
	const deliver = (event: DeliveredEvent): void => {
		for (const handler of handlers) {
			handler(event);
		}
		flushSync();
	};

	// A favorite trashed elsewhere leaves the view directly.
	deliver({ id: 1, name: 'asset.trashed', assetId: 'a1', kind: 'image' });
	expect(tiles()).toHaveLength(1);

	// A purged favorite leaves through the 404 convergence.
	deliver({ id: 2, name: 'asset.deleted', assetId: 'a2', kind: 'image' });
	expect(tiles()).toHaveLength(0);

	unmount(page);
	flushSync();
});
