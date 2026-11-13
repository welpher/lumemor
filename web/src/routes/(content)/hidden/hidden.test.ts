import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems, wallItems } from '$lib/timeline/wall-sequence.svelte';

// The hidden page through its curated wall: the listing renders and
// publishes the wall sequence, the selection bar carries exactly the
// unhide action, the unhide batch ends the selected rows'
// membership, and the SSE convergence follows the view's membership
// rule -- an unhidden row (the detail answered live) leaves the
// wall, a row that stayed hidden keeps its place. The events module
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

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function detailOf(timelineItem: ReturnType<typeof item>, overrides: Record<string, unknown> = {}): Response {
	return jsonResponse(200, {
		description: '',
		exif: {},
		isHidden: true,
		...timelineItem,
		...overrides
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, () => Response>;

function hiddenRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/assets/hidden?limit=100': () => jsonResponse(200, { items: pageOne }),
		'/api/v1/assets/batch': () => jsonResponse(200, { affected: 1, missing: [] }),
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

test('the listing renders tiles and publishes the wall sequence', async () => {
	vi.stubGlobal('fetch', stubFetch(hiddenRoutes()));
	const page = mountPage();
	await settle();

	expect(tiles()).toHaveLength(2);
	expect(wallItems().map((entry) => entry.id)).toEqual(['a1', 'a2']);

	unmount(page);
	flushSync();
});

test('an empty hidden view renders the empty state', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/assets/hidden?limit=100': () => jsonResponse(200, { items: [] })
		})
	);
	const page = mountPage();
	await settle();

	expect(tiles()).toHaveLength(0);
	expect(document.body.textContent).toContain('Nothing hidden');

	unmount(page);
	flushSync();
});

test('the selection bar carries exactly the unhide action, and it ends the membership', async () => {
	const fetchMock = stubFetch(hiddenRoutes());
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
	// The subset: only the unhide action of the timeline's set, and
	// the wall-selection entry (a hidden row is still in the
	// caller's sight here -- the entry's ground).
	expect(hasButtonByLabel('Unhide')).toBe(true);
	expect(hasButtonByLabel('Add to memory')).toBe(true);
	expect(hasButtonByLabel('Favorite')).toBe(false);
	expect(hasButtonByLabel('Hide')).toBe(false);
	expect(hasButtonByLabel('Trash')).toBe(false);
	expect(hasButtonByLabel('Restore')).toBe(false);

	buttonByLabel('Unhide').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(bodies).toEqual([{ action: 'unhide', ids: ['a1'] }]);
	// The membership ended: the tile returned to the live views and
	// left this wall.
	expect(tiles()).toHaveLength(1);

	unmount(page);
	flushSync();
});

test('the convergence removes an unhidden row and keeps a hidden one', async () => {
	const details = new Map<string, boolean>();
	const fetchMock = stubFetch(
		hiddenRoutes({
			'/api/v1/assets/a1': () => detailOf(pageOne[0]!, { isHidden: details.get('a1') ?? true })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	type DeliveredEvent = { id: number; name: string; assetId: string; kind: 'image' | 'video' };
	const deliver = (event: DeliveredEvent): void => {
		for (const handler of handlers) {
			handler(event);
		}
		flushSync();
	};

	// The edit of a row that stayed hidden (the detail answered
	// isHidden true) converges to its own place.
	details.set('a1', true);
	deliver({ id: 1, name: 'asset.updated', assetId: 'a1', kind: 'image' });
	await settle();
	expect(tiles()).toHaveLength(2);

	// The unhide elsewhere (another session's batch, say) leaves the
	// row live: it leaves this wall through the detail's membership
	// rule.
	details.set('a1', false);
	deliver({ id: 2, name: 'asset.updated', assetId: 'a1', kind: 'image' });
	await settle();
	expect(tiles()).toHaveLength(1);

	unmount(page);
	flushSync();
});
