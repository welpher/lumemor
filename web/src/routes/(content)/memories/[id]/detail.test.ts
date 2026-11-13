import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { clearWallItems } from '$lib/timeline/wall-sequence.svelte';
import type { AlbumDetail } from '$lib/api/albums';
import type { TimelineItem } from '$lib/api/timeline';
import Page from './+page.svelte';

// The memory detail: the reference-set's own face. The contextual
// bar's role-split arms -- the owner holds the edit dialog (title +
// audience alone: no source row, no autonomy block), the cover
// picker and the delete behind its light-cascade confirm; the member
// holds the owner attribution and the read-only badge alone, no
// action at all. The owner's wall curates the REMOVE-ONLY subset
// (the minus action alone -- no favorite, no trash: the membership
// tidy-up, never a photo-state write); the member's wall is pure
// navigation (no check renders). No rescan and no remove-flow
// anywhere: a memory owns no source to walk and no files to
// un-ingest. The album.* SSE events for THIS memory refetch the
// metadata and restart the items walk on membership changes.
// $app/state and $app/navigation are mocked because the real router
// needs a running SvelteKit app; the events module is mocked with a
// captured handler registry.

const { pageState, albumHandlers, assetHandlers } = vi.hoisted(() => ({
	pageState: {
		params: { id: 'mem1' } as { id: string },
		url: new URL('http://localhost/memories/mem1')
	},
	albumHandlers: [] as Array<(event: unknown) => void>,
	assetHandlers: [] as Array<(event: unknown) => void>
}));

vi.mock('$app/state', () => ({ page: pageState }));
vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));
vi.mock('$lib/events', () => ({
	subscribeAlbumEvents: (listener: (event: unknown) => void) => {
		albumHandlers.push(listener);
		return () => {
			const index = albumHandlers.indexOf(listener);
			if (index !== -1) {
				albumHandlers.splice(index, 1);
			}
		};
	},
	subscribeAssetEvents: (listener: (event: unknown) => void) => {
		assetHandlers.push(listener);
		return () => {
			const index = assetHandlers.indexOf(listener);
			if (index !== -1) {
				assetHandlers.splice(index, 1);
			}
		};
	}
}));

import { goto } from '$app/navigation';

let detailState: AlbumDetail = {
	id: 'mem1',
	title: 'Paris trip',
	type: 'memory',
	coverAssetId: null,
	callerRole: 'owner',
	itemCount: 2,
	items: [],
	shares: [{ userId: 'u2', email: 'grace@example.com', grantedAt: '2026-11-05T09:30:00Z' }]
};

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

function item(id: string, takenAt: string, overrides: Record<string, unknown> = {}): TimelineItem {
	return { id, takenAt, ...itemDefaults, ...overrides } as TimelineItem;
}

const wallItems = [item('a1', '2026-11-02T10:00:00Z'), item('a2', '2026-10-13T12:30:00Z')];

const usersBody = {
	users: [
		{ id: 'u1', email: 'ada@example.com', role: 'admin', isDisabled: false },
		{ id: 'u2', email: 'grace@example.com', role: 'user', isDisabled: false },
		{ id: 'u3', email: 'ivy@example.com', role: 'user', isDisabled: false }
	]
};

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

function detailRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/albums/mem1?limit=1': () => jsonResponse(200, detailState),
		'/api/v1/assets/timeline?albumId=mem1&limit=100': () => jsonResponse(200, { items: wallItems }),
		'/api/v1/users': () => jsonResponse(200, usersBody),
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

/** The icon-only action buttons carry their name on the aria-label. */
function buttonByLabel(label: string, scope: ParentNode = document.body): HTMLButtonElement {
	const found = scope.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing button labeled ${label}`);
	}
	return found;
}

function fillField(scope: ParentNode, value: string, id: string): void {
	const field = scope.querySelector<HTMLInputElement>(`#${id}`);
	if (field === null) {
		throw new Error(`missing dialog input ${id}`);
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

function modal(): HTMLElement | null {
	return document.body.querySelector<HTMLElement>('.modal__panel');
}

/** Fire one parsed album event to every live album subscriber. */
function fireAlbumEvent(event: Record<string, unknown>): void {
	for (const handler of [...albumHandlers]) {
		handler(event);
	}
}

beforeEach(async () => {
	document.body.textContent = '';
	albumHandlers.length = 0;
	assetHandlers.length = 0;
	vi.clearAllMocks();
	detailState = {
		id: 'mem1',
		title: 'Paris trip',
		type: 'memory',
		coverAssetId: null,
		callerRole: 'owner',
		itemCount: 2,
		items: [],
		shares: [{ userId: 'u2', email: 'grace@example.com', grantedAt: '2026-11-05T09:30:00Z' }]
	};
	vi.mocked(goto).mockClear();
	clearWallItems();
	await initI18n();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('a deleted or withdrawn memory renders the one unreachable face', async () => {
	// A memory the caller can no longer read answers the contract's
	// byte-identical 404: the same card an unknown address gets, with
	// none of the memory's own face on the page.
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums/mem1?limit=1': () => jsonResponse(404, { code: 'ALBUM_NOT_FOUND' }),
			'/api/v1/assets/timeline?albumId=mem1&limit=100': () => jsonResponse(200, { items: [] })
		})
	);
	const page = mountPage();
	await settle();

	const card = document.body.querySelector('.unreachable-card');
	expect(card?.textContent).toContain('This page will not open');
	expect(card?.querySelector('a')?.getAttribute('href')).toBe('/timeline');
	expect(document.body.querySelector('.state-card--error')).toBeNull();
	expect(document.body.querySelector('.memory-bar__title')).toBeNull();
	expect(document.body.querySelector('.memory-detail__wall')).toBeNull();
	unmount(page);
	flushSync();
});

test('the owner detail renders the bar, the count, the write arms and the wall', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The contextual bar: back into the memories wall, the title, the
	// honest count.
	const back = document.body.querySelector<HTMLAnchorElement>('.memory-bar__back');
	expect(back?.getAttribute('href')).toBe('/memories');
	expect(back?.getAttribute('aria-label')).toBe('Back to memories');
	expect(document.body.querySelector('.memory-bar__title')?.textContent).toBe('Paris trip');
	expect(document.body.querySelector('.memory-bar__count')?.textContent).toBe('2 photos');

	// The owner's write set: the edit dialog, the cover, the delete.
	for (const label of ['Edit memory', 'Change cover', 'Delete memory']) {
		expect(() => buttonByLabel(label), label).not.toThrow();
	}
	// No rescan, no remove-flow, no exclusion toggle: a memory owns
	// no source and no autonomy.
	for (const label of ['Rescan', 'Remove album', 'Keep out of timeline', 'Keep in timeline']) {
		expect(() => buttonByLabel(label), label).toThrow();
	}

	// The wall: the month-grouped walk over the album narrow.
	expect(tiles()).toHaveLength(2);
	expect(tiles()[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/assets/a1/thumbnail?size=view');

	unmount(page);
	flushSync();
});

test('the owner wall curates the removal subset: the minus action alone', async () => {
	const removals: string[][] = [];
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/mem1/items': (init?: RequestInit) => {
				expect(init?.method).toBe('DELETE');
				const body = JSON.parse(String(init?.body)) as { assetIds: string[] };
				removals.push(body.assetIds);
				return jsonResponse(200, { removed: body.assetIds.length, skipped: 0 });
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The checks render (the removal runner's presence is the
	// capability); a selection raises the bar with the REMOVE-ONLY
	// subset -- no favorite, no trash, no restore.
	const check = tiles()[0]?.querySelector<HTMLElement>('.tile__check');
	expect(check).not.toBeNull();
	check?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const bar = document.body.querySelector<HTMLElement>('.batchbar');
	expect(bar).not.toBeNull();
	expect(bar?.textContent).toContain('1 selected');
	for (const label of ['Favorite', 'Hide', 'Unhide', 'Restore', 'Trash', 'Add tag', 'Add to memory']) {
		expect(() => buttonByLabel(label, bar as ParentNode), label).toThrow();
	}
	const remove = buttonByLabel('Remove from memory', bar as ParentNode);
	expect(remove.classList.contains('icon-btn--danger')).toBe(true);

	// The direct write runs (no confirm dialog: the site shows none,
	// the light semantics need none) and the success arm clears the
	// selection.
	remove.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(removals).toEqual([['a1']]);
	expect(document.body.querySelector('.batchbar')).toBeNull();
	// The membership's own convergence: the landing write's event
	// restarts the walk (the fresh page answer owns the tiles).
	fireAlbumEvent({ name: 'album.items-changed', albumId: 'mem1', userId: 'u1' });
	await settle();
	expect(
		fetchMock.mock.calls.filter(([input]) => String(input).startsWith('/api/v1/assets/timeline')).length
	).toBeGreaterThanOrEqual(2);

	unmount(page);
	flushSync();
});

test('a failed removal keeps the selection and maps the error in the bar', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			detailRoutes({
				'/api/v1/albums/mem1/items': () =>
					jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'the sharee cannot curate' })
			})
		)
	);
	const page = mountPage();
	await settle();

	const check = tiles()[0]?.querySelector<HTMLElement>('.tile__check');
	check?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	buttonByLabel('Remove from memory').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The failure arm: the selection stays (the retry affordance) and
	// the mapped error renders in the bar.
	expect(document.body.querySelector('.batchbar')).not.toBeNull();
	expect(document.body.querySelector('.batchbar__error')?.textContent).toBe("You don't have permission to do this.");

	unmount(page);
	flushSync();
});

test('a viewer sees read-only: the attribution, the badge, no checks at all', async () => {
	detailState = {
		...detailState,
		callerRole: 'viewer',
		shares: undefined
	};
	// The attribution's source: the detail read carries no owner
	// face, so the viewer's bar reads it off the matching shared
	// wall card.
	const sharedCard = {
		id: 'mem1',
		title: 'Paris trip',
		type: 'memory',
		isShared: true,
		coverAssetIds: [],
		itemCount: 2,
		updatedAt: '2026-11-05T10:15:00Z',
		callerRole: 'viewer',
		owner: { id: 'u2', name: 'Grace Hopper', email: 'grace@example.com' }
	};
	vi.stubGlobal(
		'fetch',
		stubFetch(detailRoutes({ '/api/v1/albums': () => jsonResponse(200, { albums: [sharedCard] }) }))
	);
	const page = mountPage();
	await settle();

	// Read-only is visual: every write affordance is absent, never
	// disabled -- the audience is owner-domain.
	for (const label of ['Edit memory', 'Change cover', 'Delete memory']) {
		expect(() => buttonByLabel(label), label).toThrow();
	}
	// The member's arm: the honest owner attribution and the
	// read-only badge.
	expect(document.body.querySelector('.memory-bar__owner')?.textContent).toBe('Shared by Grace Hopper');
	expect(document.body.querySelector('.memory-bar__badge')?.textContent?.trim()).toBe('Read-only');

	// THE NO-SELECTION WALL: the member's tiles are pure navigation
	// -- no hover checkbox, no batch bar ever rises.
	expect(tiles()).toHaveLength(2);
	expect(document.body.querySelector('.tile__check')).toBeNull();
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(page);
	flushSync();
});

test('the edit dialog edits the title and PATCHes it alone', async () => {
	let titleSaved = false;
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/mem1': (init?: RequestInit) => {
				expect(init?.method).toBe('PATCH');
				const body = JSON.parse(String(init?.body));
				expect(body).toEqual({ title: 'Paris 2026' });
				titleSaved = true;
				return jsonResponse(200, {
					id: 'mem1',
					title: 'Paris 2026',
					type: 'memory',
					isShared: true,
					coverAssetIds: [],
					itemCount: 2,
					updatedAt: '2026-11-09T09:00:00Z',
					callerRole: 'owner'
				});
			},
			'/api/v1/albums/mem1?limit=1': () =>
				titleSaved ? jsonResponse(200, { ...detailState, title: 'Paris 2026' }) : jsonResponse(200, detailState)
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Edit memory').click();
	await settle();
	const dialog = modal();
	if (dialog === null) {
		throw new Error('missing edit dialog');
	}
	expect(dialog.getAttribute('aria-label')).toBe('Edit memory');

	// The dialog's shape: the title and the audience alone -- no
	// source row, no autonomy toggles (a memory owns neither).
	fillField(dialog, 'Paris 2026', 'album-title');
	expect(dialog.querySelector('.album-form__source')).toBeNull();
	expect(dialog.querySelector('[data-testid="album-timeline-toggle"]')).toBeNull();
	expect(dialog.querySelector('[data-testid="album-deletable-toggle"]')).toBeNull();

	buttonByText('Save changes', dialog).click();
	await settle();

	// The title PATCH rode alone (the audience set did not change);
	// the converged detail carries the new title into the bar.
	expect(document.body.querySelector('.memory-bar__title')?.textContent).toBe('Paris 2026');
	expect(modal()).toBeNull();

	unmount(page);
	flushSync();
});

test('the cover dialog picks from the grid members and PATCHes the pin', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/mem1': (init?: RequestInit) => {
				expect(JSON.parse(String(init?.body))).toEqual({ coverAssetId: 'a2' });
				return jsonResponse(200, {
					id: 'mem1',
					title: 'Paris trip',
					type: 'memory',
					isShared: true,
					coverAssetIds: ['a2'],
					itemCount: 2,
					updatedAt: '2026-11-09T09:00:00Z',
					callerRole: 'owner'
				});
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Change cover').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Set the cover');
	const options = Array.from(dialog?.querySelectorAll<HTMLButtonElement>('.cover-option') ?? []);
	expect(options).toHaveLength(2);
	expect(options[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/assets/a1/thumbnail?size=tile');
	// The current pin (none yet) marks nothing; picking a2 selects it.
	options[1]?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(options[1]?.getAttribute('aria-pressed')).toBe('true');
	buttonByText('Set cover', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const patches = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/mem1' && init?.method === 'PATCH'
	);
	expect(patches).toHaveLength(1);
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the cover dialog renders the current pin and the empty state', async () => {
	detailState = { ...detailState, coverAssetId: 'a1', itemCount: 0 };
	const routes = detailRoutes({
		'/api/v1/assets/timeline?albumId=mem1&limit=100': () => jsonResponse(200, { items: [] })
	});
	vi.stubGlobal('fetch', stubFetch(routes));
	const page = mountPage();
	await settle();

	buttonByLabel('Change cover').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.textContent).toContain('This album has no photos yet.');
	unmount(page);
	flushSync();
});

test('the delete confirm states the light cascade and leaves for the wall', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/mem1': (init?: RequestInit) => {
				if (init?.method === 'DELETE') {
					return new Response(null, { status: 204 });
				}
				return jsonResponse(200, detailState);
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Delete memory').click();
	await settle();
	const dialog = modal();
	if (dialog === null) {
		throw new Error('missing delete confirm');
	}
	// The light cascade's own statement: the references and the
	// audience die, the photos stay.
	expect(dialog.textContent).toContain('"Paris trip" goes');
	expect(dialog.textContent).toContain('The photos themselves stay in your library.');
	// Not the un-ingest language.
	expect(dialog.textContent).not.toContain('source folder');

	buttonByText('Delete memory', dialog).click();
	await settle();

	expect(
		fetchMock.mock.calls.some(
			([input, init]) => String(input) === '/api/v1/albums/mem1' && init?.method === 'DELETE'
		)
	).toBe(true);
	expect(vi.mocked(goto)).toHaveBeenCalledWith('/memories');
	expect(modal()).toBeNull();

	unmount(page);
	flushSync();
});

test('SSE membership changes restart the walk and refetch the detail', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	const detailCalls = () =>
		fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/albums/mem1?limit=1').length;
	const walkCalls = () =>
		fetchMock.mock.calls.filter(([input]) => String(input).startsWith('/api/v1/assets/timeline')).length;
	expect(detailCalls()).toBe(1);
	expect(walkCalls()).toBe(1);

	// A foreign memory's event is not this page's business.
	fireAlbumEvent({ name: 'album.items-changed', albumId: 'mem2', userId: 'u1' });
	await settle();
	expect(detailCalls()).toBe(1);

	// THIS memory's membership change restarts the walk and refetches
	// the metadata; every other album event refetches alone.
	fireAlbumEvent({ name: 'album.items-changed', albumId: 'mem1', userId: 'u1' });
	await settle();
	expect(detailCalls()).toBe(2);
	expect(walkCalls()).toBe(2);
	fireAlbumEvent({ name: 'album.updated', albumId: 'mem1' });
	await settle();
	expect(detailCalls()).toBe(3);
	expect(walkCalls()).toBe(2);

	unmount(page);
	flushSync();
});
