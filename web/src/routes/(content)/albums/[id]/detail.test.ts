import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems } from '$lib/timeline/wall.svelte';
import type { AlbumDetail, AlbumSummary } from '$lib/api/albums';
import type { TimelineItem } from '$lib/api/timeline';
import Page from './+page.svelte';

// The album detail (web IA §2-§3): the metadata header with its
// callerRole x type action set -- the owner of a virtual album holds
// rename / set cover / lock / share management / add photos / delete,
// the owner of a directory album the properties / exclusion toggle /
// disabled rescan placeholder, a viewer the exclusion toggle alone
// (read-only is visual: no write affordance renders) -- over the
// CuratedWall grid that pages the album's own timeline. Every write
// adopts its own response or converges through the detail refetch;
// the album.* SSE events for THIS album refetch the metadata and
// restart the items walk on membership changes. $app/state and
// $app/navigation are mocked because the real router needs a running
// SvelteKit app; the events module is mocked with a captured handler
// registry, the timeline page's pattern.

const { pageState, albumHandlers, assetHandlers } = vi.hoisted(() => ({
	pageState: {
		params: { id: 'al1' } as { id: string },
		url: new URL('http://localhost/albums/al1')
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

function summary(overrides: Partial<AlbumSummary> = {}): AlbumSummary {
	return {
		id: 'al1',
		title: 'Road trip 2026',
		description: 'The coastal week in the van.',
		type: 'virtual',
		locked: false,
		isShared: false,
		coverAssetIds: ['a1', 'a2'],
		itemCount: 2,
		updatedAt: '2026-11-04T18:20:00Z',
		callerRole: 'owner',
		...overrides
	};
}

let detailState: AlbumDetail = {
	id: 'al1',
	title: 'Road trip 2026',
	description: 'The coastal week in the van.',
	type: 'virtual',
	locked: false,
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

const albumItems = [item('a1', '2026-11-02T10:00:00Z'), item('a2', '2026-10-13T12:30:00Z')];

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
		'/api/v1/albums/al1?limit=1': () => jsonResponse(200, detailState),
		'/api/v1/assets/timeline?albumId=al1&limit=100': () =>
			jsonResponse(200, { items: albumItems }),
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

function fillField(scope: ParentNode, value: string): void {
	const field = scope.querySelector<HTMLInputElement>('input');
	if (field === null) {
		throw new Error('missing dialog input');
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

function chooseOption(scope: ParentNode, value: string): void {
	const select = scope.querySelector<HTMLSelectElement>('select');
	if (select === null) {
		throw new Error('missing dialog select');
	}
	select.value = value;
	select.dispatchEvent(new Event('change', { bubbles: true }));
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
		id: 'al1',
		title: 'Road trip 2026',
		description: 'The coastal week in the van.',
		type: 'virtual',
		locked: false,
		coverAssetId: null,
		callerRole: 'owner',
		itemCount: 2,
		items: [],
		shares: [{ userId: 'u2', email: 'grace@example.com', grantedAt: '2026-11-05T09:30:00Z' }]
	};
	// Boot the session store unauthenticated (the interceptor stays
	// inert; these routes answer 200s).
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
	clearWallItems();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the owner detail renders the header, the count and the items grid', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.album-detail__title')?.textContent).toBe('Road trip 2026');
	expect(document.body.querySelector('.album-detail__description')?.textContent).toBe(
		'The coastal week in the van.'
	);
	expect(document.body.querySelector('.album-detail__count')?.textContent).toBe('2 photos');
	expect(document.body.querySelector('.album-detail__back')?.getAttribute('href')).toBe('/albums');

	// The owner of a virtual album holds the full write set.
	for (const label of ['Rename', 'Set cover…', 'Lock', 'Share…', 'Add photos', 'Delete album']) {
		expect(() => buttonByText(label), label).not.toThrow();
	}

	// The items grid is the album's own timeline: the tiles come from
	// the album-narrowed page one.
	expect(tiles()).toHaveLength(2);
	expect(tiles()[0]?.querySelector('img')?.getAttribute('src')).toBe(
		'/api/v1/assets/a1/thumbnail?size=tile'
	);
	const wallCalls = fetchMock.mock.calls.filter(([input]) =>
		String(input).startsWith('/api/v1/assets/timeline?albumId=al1')
	);
	expect(wallCalls).toHaveLength(1);

	// The add-photos handoff arms the timeline's selection flow.
	buttonByText('Add photos').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(vi.mocked(goto)).toHaveBeenCalledWith('/timeline?album=al1');

	unmount(page);
	flushSync();
});

test('a viewer sees read-only: the exclusion toggle alone, no write affordances', async () => {
	detailState = {
		...detailState,
		callerRole: 'viewer',
		shares: undefined,
		inTimeline: true
	};
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	// Read-only is visual: every write affordance is absent, never
	// disabled.
	for (const label of [
		'Rename',
		'Set cover…',
		'Lock',
		'Unlock',
		'Share…',
		'Add photos',
		'Delete album'
	]) {
		expect(() => buttonByText(label), label).toThrow();
	}
	// The one action a viewer holds: their own timeline preference.
	expect(() => buttonByText('Keep out of timeline')).not.toThrow();
	expect(tiles()).toHaveLength(2);
	unmount(page);
	flushSync();
});

test('the own virtual album offers no exclusion toggle: it never feeds the own timeline', async () => {
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	expect(() => buttonByText('Keep out of timeline')).toThrow();
	expect(() => buttonByText('Keep in timeline')).toThrow();
	unmount(page);
	flushSync();
});

test('the directory detail: properties, the exclusion toggle and the disabled rescan placeholder', async () => {
	detailState = {
		...detailState,
		type: 'directory',
		inTimeline: true
	};
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	expect(() => buttonByText('Properties')).not.toThrow();
	expect(() => buttonByText('Keep out of timeline')).not.toThrow();

	const rescan = buttonByText('Rescan');
	expect(rescan.disabled).toBe(true);
	expect(document.body.querySelector('.album-detail__rescan-note')?.textContent).toContain(
		'Directory albums arrive in a later update.'
	);

	// The virtual album's write set is gone, and the directory album
	// holds neither curation nor removal.
	for (const label of ['Rename', 'Set cover…', 'Lock', 'Share…', 'Add photos', 'Delete album']) {
		expect(() => buttonByText(label), label).toThrow();
	}
	unmount(page);
	flushSync();
});

test('renaming PATCHes the properties and the header adopts the response', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1': (init?: RequestInit) => {
				expect(init?.method).toBe('PATCH');
				const body = JSON.parse(String(init?.body));
				expect(body).toEqual({
					title: 'Road trip 2026 -- rerun',
					description: 'The coastal week in the van.'
				});
				return jsonResponse(
					200,
					summary({ title: 'Road trip 2026 -- rerun', updatedAt: '2026-11-05T11:02:00Z' })
				);
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Rename').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Album properties');
	fillField(dialog as ParentNode, 'Road trip 2026 -- rerun');
	buttonByText('Save changes', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const patches = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'PATCH'
	);
	expect(patches).toHaveLength(1);
	expect(document.body.querySelector('.album-detail__title')?.textContent).toBe(
		'Road trip 2026 -- rerun'
	);
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the lock toggle PATCHes locked; the badge and notice follow and curation disables', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1': (init?: RequestInit) => {
				expect(JSON.parse(String(init?.body))).toEqual({ locked: true });
				return jsonResponse(200, summary({ locked: true }));
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Lock').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const patches = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'PATCH'
	);
	expect(patches).toHaveLength(1);
	expect(document.body.querySelector('.album-detail__badge--locked')?.textContent?.trim()).toBe('Locked');
	expect(document.body.querySelector('.album-detail__locked-notice')?.textContent).toContain(
		'until you unlock it'
	);
	// The locked gate is visual too: curation and property writes
	// disable until the unlock, while the delete and the share stay.
	expect(buttonByText('Rename').disabled).toBe(true);
	expect(buttonByText('Set cover…').disabled).toBe(true);
	expect(buttonByText('Add photos').disabled).toBe(true);
	expect(buttonByText('Delete album').disabled).toBe(false);
	expect(buttonByText('Share…').disabled).toBe(false);
	expect(() => buttonByText('Unlock')).not.toThrow();
	unmount(page);
	flushSync();
});

test('the cover dialog picks from the grid members and PATCHes the pin', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1': (init?: RequestInit) => {
				expect(JSON.parse(String(init?.body))).toEqual({ coverAssetId: 'a2' });
				return jsonResponse(200, summary({ coverAssetIds: ['a2'] }));
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Set cover…').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Set the cover');
	const options = Array.from(dialog?.querySelectorAll<HTMLButtonElement>('.cover-option') ?? []);
	expect(options).toHaveLength(2);
	expect(options[0]?.querySelector('img')?.getAttribute('src')).toBe(
		'/api/v1/assets/a1/thumbnail?size=tile'
	);
	// The current pin (none yet) marks nothing; picking a2 selects it.
	options[1]?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(options[1]?.getAttribute('aria-pressed')).toBe('true');
	buttonByText('Set cover', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const patches = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'PATCH'
	);
	expect(patches).toHaveLength(1);
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the cover dialog renders the current pin and the empty state', async () => {
	detailState = { ...detailState, coverAssetId: 'a1', itemCount: 0 };
	const routes = detailRoutes({
		'/api/v1/assets/timeline?albumId=al1&limit=100': () => jsonResponse(200, { items: [] })
	});
	vi.stubGlobal('fetch', stubFetch(routes));
	const page = mountPage();
	await settle();

	buttonByText('Set cover…').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.textContent).toContain('This album has no photos yet.');
	unmount(page);
	flushSync();
});

test('the share dialog grants and revokes through the detail refetch', async () => {
	let granted = false;
	let revoked = false;
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1/shares': (init?: RequestInit) => {
				expect(init?.method).toBe('POST');
				expect(JSON.parse(String(init?.body))).toEqual({ userId: 'u3' });
				granted = true;
				return jsonResponse(200, {
					albumId: 'al1',
					userId: 'u3',
					grantedAt: '2026-11-05T12:00:00Z'
				});
			},
			'/api/v1/albums/al1/shares/u2': (init?: RequestInit) => {
				expect(init?.method).toBe('DELETE');
				revoked = true;
				return new Response(null, { status: 204 });
			},
			'/api/v1/albums/al1?limit=1': () => {
				if (granted && !revoked) {
					return jsonResponse(200, {
						...detailState,
						isShared: undefined,
						shares: [
							{ userId: 'u2', email: 'grace@example.com', grantedAt: '2026-11-05T09:30:00Z' },
							{ userId: 'u3', email: 'ivy@example.com', grantedAt: '2026-11-05T12:00:00Z' }
						]
					});
				}
				if (revoked) {
					return jsonResponse(200, {
						...detailState,
						shares: [{ userId: 'u3', email: 'ivy@example.com', grantedAt: '2026-11-05T12:00:00Z' }]
					});
				}
				return jsonResponse(200, detailState);
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Share…').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Share this album');
	// The grantee list rides the owner's detail: one current grantee.
	expect(dialog?.textContent).toContain('grace@example.com');
	// The pick list offers the other accounts (current grantees excluded).
	chooseOption(dialog as ParentNode, 'u3');
	buttonByText('Share', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	// The refetched detail's grantee list carries the new row.
	expect(modal()?.textContent).toContain('ivy@example.com');

	// The revoke: the row's button leaves, the refetch drops the row
	// (the account legitimately re-enters the pick list).
	const revoke = buttonByText('Revoke', modal() as ParentNode);
	revoke.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const granteeList = modal()?.querySelector('.share-dialog__list');
	expect(granteeList?.textContent).not.toContain('grace@example.com');
	expect(granteeList?.textContent).toContain('ivy@example.com');
	unmount(page);
	flushSync();
});

test('the share dialog surfaces the honest owner-target message', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			detailRoutes({
				'/api/v1/albums/al1/shares': () =>
					jsonResponse(400, {
						code: 'VALIDATION_FAILED',
						message: 'an album owner cannot be its grantee',
						details: { key: 'album_share_to_owner' }
					})
			})
		)
	);
	const page = mountPage();
	await settle();

	buttonByText('Share…').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const dialog = modal();
	chooseOption(dialog as ParentNode, 'u1');
	buttonByText('Share', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	// The pick list cannot exclude the caller (no me source on the
	// client), so the server's own target rule answers the pick
	// honestly.
	expect(modal()?.textContent).toContain('This album is already yours');
	unmount(page);
	flushSync();
});

test('deleting the album confirms and leaves for the wall', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1': (init?: RequestInit) => {
				expect(init?.method).toBe('DELETE');
				return new Response(null, { status: 204 });
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Delete album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Delete this album?');
	expect(dialog?.textContent).toContain('The photos themselves stay in your library.');
	buttonByText('Delete album', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const deletes = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'DELETE'
	);
	expect(deletes).toHaveLength(1);
	expect(vi.mocked(goto)).toHaveBeenCalledWith('/albums');
	unmount(page);
	flushSync();
});

test('the exclusion toggle PUTs the preference and the badge and notice follow', async () => {
	detailState = { ...detailState, callerRole: 'viewer', shares: undefined, inTimeline: true };
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1/timeline-preference': (init?: RequestInit) => {
				expect(init?.method).toBe('PUT');
				expect(JSON.parse(String(init?.body))).toEqual({ inTimeline: false });
				return jsonResponse(200, { albumId: 'al1', inTimeline: false });
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Keep out of timeline').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(
		([input]) => String(input) === '/api/v1/albums/al1/timeline-preference'
	);
	expect(puts).toHaveLength(1);
	expect(document.body.querySelector('.album-detail__badge--excluded')?.textContent?.trim()).toBe(
		'Not in timeline'
	);
	expect(document.body.querySelector('.album-detail__excluded-notice')?.textContent).toContain(
		'stay out of your timeline'
	);
	// The toggle flipped: the action now offers the include.
	expect(() => buttonByText('Keep in timeline')).not.toThrow();
	unmount(page);
	flushSync();
});

test('an unknown or invisible album renders the not-found error with a retry', async () => {
	let failing = true;
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		if (String(input) === '/api/v1/albums/al1?limit=1') {
			if (failing) {
				return jsonResponse(404, { code: 'ALBUM_NOT_FOUND', message: 'no such album' });
			}
			return jsonResponse(200, detailState);
		}
		if (String(input).startsWith('/api/v1/assets/timeline')) {
			return jsonResponse(200, { items: [] });
		}
		throw new Error(`unexpected fetch ${String(input)}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.album-detail__failure');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('This album does not exist or is not visible to you.');
	// The wall stays gated behind the metadata: no tiles, no empty
	// state from the album's timeline page.
	expect(tiles()).toHaveLength(0);

	failing = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(document.body.querySelector('.album-detail__title')?.textContent).toBe('Road trip 2026');
	unmount(page);
	flushSync();
});

test('SSE album.updated converges the metadata through the refetch', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	const initial = fetchMock.mock.calls.filter(([input]) =>
		String(input).startsWith('/api/v1/albums/al1?limit=1')
	).length;

	detailState = { ...detailState, title: 'Road trip 2026 -- rerun' };
	fireAlbumEvent({ id: 2, name: 'album.updated', albumId: 'al1', userId: 'u1' });
	await settle();

	expect(
		fetchMock.mock.calls.filter(([input]) => String(input).startsWith('/api/v1/albums/al1?limit=1'))
			.length
	).toBe(initial + 1);
	expect(document.body.querySelector('.album-detail__title')?.textContent).toBe(
		'Road trip 2026 -- rerun'
	);
	unmount(page);
	flushSync();
});

test('SSE album.items-changed restarts the items walk', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	const initial = fetchMock.mock.calls.filter(([input]) =>
		String(input).startsWith('/api/v1/assets/timeline?albumId=al1')
	).length;

	fireAlbumEvent({ id: 3, name: 'album.items-changed', albumId: 'al1', userId: 'u1' });
	await settle();

	expect(
		fetchMock.mock.calls.filter(([input]) =>
			String(input).startsWith('/api/v1/assets/timeline?albumId=al1')
		).length
	).toBe(initial + 1);
	unmount(page);
	flushSync();
});

test('SSE album.deleted renders the not-found error state', async () => {
	let deleted = false;
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		if (String(input) === '/api/v1/albums/al1?limit=1') {
			if (deleted) {
				return jsonResponse(404, { code: 'ALBUM_NOT_FOUND', message: 'no such album' });
			}
			return jsonResponse(200, detailState);
		}
		if (String(input).startsWith('/api/v1/assets/timeline')) {
			return jsonResponse(200, { items: deleted ? [] : albumItems });
		}
		throw new Error(`unexpected fetch ${String(input)}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(tiles()).toHaveLength(2);

	deleted = true;
	fireAlbumEvent({ id: 4, name: 'album.deleted', albumId: 'al1', userId: 'u1' });
	await settle();

	expect(document.body.querySelector('.album-detail__failure')?.textContent).toContain(
		'This album does not exist or is not visible to you.'
	);
	expect(tiles()).toHaveLength(0);
	unmount(page);
	flushSync();
});

test('a foreign album event converges nothing', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	const initial = fetchMock.mock.calls.length;

	fireAlbumEvent({ id: 5, name: 'album.updated', albumId: 'al-elsewhere', userId: 'u2' });
	await settle();

	expect(fetchMock.mock.calls.length).toBe(initial);
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from the event stream', async () => {
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();
	// The detail's album listener and the wall's asset listener.
	expect(albumHandlers).toHaveLength(1);
	expect(assetHandlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(albumHandlers).toHaveLength(0);
	expect(assetHandlers).toHaveLength(0);
});
