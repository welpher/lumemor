import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { probeSession, sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems } from '$lib/timeline/wall-sequence.svelte';
import type { AlbumDetail, AlbumSummary } from '$lib/api/albums';
import type { TimelineItem } from '$lib/api/timeline';
import Page from './+page.svelte';

// The album detail (the albums-as-sources reshape,
// completed by the autonomy face, simplified by the lock's
// retirement): the metadata header with its
// callerRole x type action set -- the owner of a memory album holds
// rename / set cover / delete, the owner of a directory album the
// properties edit (title + the read-only source row + the audience
// + the autonomy block of timeline exclusion / deletability)
// / rescan trigger with its job-event progress line / the remove
// (the un-ingest, behind its semantics-stating confirm), the owner
// of the sync album (the default album) NOTHING -- system-managed,
// read-only, the localized display name for the title -- a viewer
// the exclusion toggle alone (the departure went with the revoke
// endpoint it rode) -- over the wall that pages the album's own
// timeline, curating a DELETE-ONLY selection exactly when the
// server-derived `deletable` face holds (owner AND writable AND
// the switch). Every write adopts its own response or
// converges through the detail refetch; the album.* SSE events for
// THIS album refetch the metadata and restart the items walk on
// membership changes; the rescan's progress rides the job.* events
// keyed by the run the trigger answered. $app/state and
// $app/navigation are mocked because the real router needs a
// running SvelteKit app; the events module is mocked with a
// captured handler registry, the timeline page's pattern.

const { pageState, albumHandlers, assetHandlers, jobHandlers } = vi.hoisted(() => ({
	pageState: {
		params: { id: 'al1' } as { id: string },
		url: new URL('http://localhost/albums/al1')
	},
	albumHandlers: [] as Array<(event: unknown) => void>,
	assetHandlers: [] as Array<(event: unknown) => void>,
	jobHandlers: [] as Array<(event: unknown) => void>
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
	},
	subscribeJobEvents: (listener: (event: unknown) => void) => {
		jobHandlers.push(listener);
		return () => {
			const index = jobHandlers.indexOf(listener);
			if (index !== -1) {
				jobHandlers.splice(index, 1);
			}
		};
	}
}));

import { goto } from '$app/navigation';

function summary(overrides: Partial<AlbumSummary> = {}): AlbumSummary {
	return {
		id: 'al1',
		title: 'Road trip 2026',
		type: 'memory',
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

const albumItems = [item('a1', '2026-11-02T10:00:00Z'), item('a2', '2026-10-13T12:30:00Z')];

const usersBody = {
	users: [
		{ id: 'u1', email: 'ada@example.com', role: 'admin', isDisabled: false },
		{ id: 'u2', email: 'grace@example.com', role: 'user', isDisabled: false },
		{ id: 'u3', email: 'ivy@example.com', role: 'user', isDisabled: false },
		{ id: 'u4', email: 'hopper@example.com', role: 'user', isDisabled: true }
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
		'/api/v1/assets/timeline?albumId=al1&limit=100': () => jsonResponse(200, { items: albumItems }),
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

/** Fire one parsed job event to every live job subscriber. */
function fireJobEvent(event: Record<string, unknown>): void {
	for (const handler of [...jobHandlers]) {
		handler(event);
	}
}

beforeEach(async () => {
	document.body.textContent = '';
	albumHandlers.length = 0;
	assetHandlers.length = 0;
	jobHandlers.length = 0;
	vi.clearAllMocks();
	detailState = {
		id: 'al1',
		title: 'Road trip 2026',
		type: 'memory',
		coverAssetId: null,
		callerRole: 'owner',
		itemCount: 2,
		items: [],
		shares: [{ userId: 'u2', email: 'grace@example.com', grantedAt: '2026-11-05T09:30:00Z' }]
	};
	// Boot the session store unauthenticated (the interceptor stays
	// inert; these routes answer 200s).
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
	clearWallItems();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the owner detail renders the contextual bar, the count and the items grid', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// THE CONTEXTUAL BAR: the page's own chrome replaces the
	// shell's topbar while the album detail is active -- back, the
	// title, the badges and count, the type x role icon actions.
	// Paper palette (bg-card + border), pinned by stickiness over
	// the wall's own sticky month headers.
	const bar = document.body.querySelector<HTMLElement>('.album-bar');
	expect(bar).not.toBeNull();
	expect(bar?.className).toContain('sticky');
	expect(bar?.className).toContain('top-0');
	expect(bar?.className).toContain('z-(--z-sticky-bar)');
	expect(document.body.querySelector('.album-bar__title')?.textContent).toBe('Road trip 2026');
	// The back affordance: an icon-only anchor into the albums wall.
	const back = document.body.querySelector<HTMLAnchorElement>('.album-bar__back');
	expect(back?.getAttribute('href')).toBe('/albums');
	expect(back?.getAttribute('aria-label')).toBe('Back to albums');
	// The description line died whole (the 2026-11-08 owner
	// evaluation): the header carries the title alone, never a
	// description paragraph.
	expect(document.body.querySelector('.album-detail__description')).toBeNull();
	expect(document.body.querySelector('.album-detail__count')?.textContent).toBe('2 photos');

	// The owner of a memory album holds the write set: rename
	// (the dialog opener), the cover and the delete. The share
	// action and the
	// add-photos handoff are GONE with the curation removal
	// and the share dialog's fold-in: the audience is an
	// album property. The remove affordance is the directory
	// album's own.
	for (const label of ['Rename', 'Set cover…', 'Delete album']) {
		expect(() => buttonByLabel(label), label).not.toThrow();
	}
	for (const label of [
		'Share…',
		'Add photos',
		'Properties',
		'Rescan',
		'Remove album',
		'Leave album',
		'Lock',
		'Unlock'
	]) {
		expect(() => buttonByLabel(label), label).toThrow();
	}

	// THE NO-SELECTION WALL: a memory album's tiles are pure
	// navigation -- no hover checkbox, no select-mode entry, no
	// batch bar ever rises (the deletable face is the directory
	// album's own).
	expect(tiles()).toHaveLength(2);
	expect(document.body.querySelector('.tile__check')).toBeNull();
	expect(document.body.querySelector('.batchbar')).toBeNull();
	expect(document.body.textContent).not.toContain('Select');
	expect(tiles()[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/assets/a1/thumbnail?size=view');
	const wallCalls = fetchMock.mock.calls.filter(([input]) =>
		String(input).startsWith('/api/v1/assets/timeline?albumId=al1')
	);
	expect(wallCalls).toHaveLength(1);

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
	// disabled -- the audience is owner-domain, and the departure
	// went with the revoke endpoint it rode.
	for (const label of [
		'Rename',
		'Set cover…',
		'Lock',
		'Unlock',
		'Delete album',
		'Properties',
		'Rescan',
		'Remove album',
		'Leave album'
	]) {
		expect(() => buttonByLabel(label), label).toThrow();
	}
	// The one action a viewer holds: their own timeline preference
	// (the bar arm -- the owner's own toggle moved into the edit
	// dialog, but the viewer holds no dialog; the member side is
	// unchanged).
	expect(() => buttonByLabel('Keep out of timeline')).not.toThrow();
	expect(tiles()).toHaveLength(2);
	unmount(page);
	flushSync();
});

test('the own memory album offers no exclusion toggle: it never feeds the own timeline', async () => {
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	expect(() => buttonByLabel('Keep out of timeline')).toThrow();
	expect(() => buttonByLabel('Keep in timeline')).toThrow();
	unmount(page);
	flushSync();
});

test('the directory detail: the edit dialog, the autonomy faces and the live rescan trigger', async () => {
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		sourceWritable: true,
		allowAppDeletion: true,
		deletable: true,
		inTimeline: true
	};
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	expect(() => buttonByLabel('Properties')).not.toThrow();

	// The owner's exclusion toggle moved into the edit dialog:
	// the bar arm carries no calendar icon for the owner -- the
	// viewer's arm (their only action) keeps it.
	expect(() => buttonByLabel('Keep out of timeline')).toThrow();

	// The rescan trigger is LIVE: the directory album's own
	// refresh, not the placeholder.
	const rescan = buttonByLabel('Rescan');
	expect(rescan.disabled).toBe(false);

	// The remove action sits beside it: the directory album's own
	// un-ingest affordance.
	const remove = buttonByLabel('Remove album');
	expect(remove.disabled).toBe(false);

	// The memory album's write set is gone, and the directory album
	// holds neither curation, cover pin nor the memory delete.
	for (const label of ['Rename', 'Set cover…', 'Lock', 'Unlock', 'Delete album', 'Leave album']) {
		expect(() => buttonByLabel(label), label).toThrow();
	}

	// THE DELETABLE WALL: the derived face holds, so the wall's
	// selection machinery is wired -- the tiles carry the hover
	// check affordance and a selection raises the batch bar with the
	// DELETE-ONLY action subset (no favorite, no hide, no restore).
	const check = tiles()[0]?.querySelector<HTMLElement>('.tile__check');
	expect(check).not.toBeNull();
	check?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const bar = document.body.querySelector<HTMLElement>('.batchbar');
	expect(bar).not.toBeNull();
	expect(bar?.getAttribute('role')).toBe('toolbar');
	for (const label of ['Favorite', 'Hide', 'Unhide', 'Restore']) {
		expect(() => buttonByLabel(label, bar as ParentNode), label).toThrow();
	}
	expect(() => buttonByLabel('Trash', bar as ParentNode)).not.toThrow();
	unmount(page);
	flushSync();
});

test('the sync album detail: the localized title, the grid, and zero write affordances', async () => {
	// The caller's own sync album -- the default album. The stored
	// title is the internal system label; the header (and the
	// document title) render the localized display name. The action
	// set's sync arm renders NOTHING: system-managed, no write
	// affordance of any arm, absent never disabled.
	detailState = {
		...detailState,
		title: 'Sync',
		type: 'sync',
		coverAssetId: null,
		shares: []
	};
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.album-bar__title')?.textContent).toBe('My photos');
	expect(document.body.textContent).not.toContain('Sync');
	expect(document.body.querySelector('.album-detail__count')?.textContent).toBe('2 photos');
	// The item grid works like any album's: the tiles come from the
	// album-narrowed timeline page.
	expect(tiles()).toHaveLength(2);
	expect(tiles()[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/assets/a1/thumbnail?size=view');
	// No badges: never excluded (not excludable).
	expect(document.body.querySelector('.album-detail__badge--excluded')).toBeNull();

	for (const label of [
		'Rename',
		'Set cover…',
		'Lock',
		'Unlock',
		'Delete album',
		'Properties',
		'Rescan',
		'Remove album',
		'Keep out of timeline',
		'Keep in timeline',
		'Leave album'
	]) {
		expect(() => buttonByLabel(label), label).toThrow();
	}
	unmount(page);
	flushSync();
});

test('the properties dialog edits the title and PATCHes it alone', async () => {
	let titleSaved = false;
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1': (init?: RequestInit) => {
				expect(init?.method).toBe('PATCH');
				const body = JSON.parse(String(init?.body));
				// The description field died whole (the 2026-11-08
				// owner evaluation -- no PATCH face, no column): the
				// patch carries the title alone.
				expect(body).toEqual({ title: 'Road trip 2026 -- rerun' });
				titleSaved = true;
				return jsonResponse(
					200,
					summary({ title: 'Road trip 2026 -- rerun', updatedAt: '2026-11-05T11:02:00Z' })
				);
			},
			'/api/v1/albums/al1?limit=1': () =>
				titleSaved
					? jsonResponse(200, { ...detailState, title: 'Road trip 2026 -- rerun' })
					: jsonResponse(200, detailState)
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Rename').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Album properties');
	fillField(dialog as ParentNode, 'Road trip 2026 -- rerun', 'album-title');
	buttonByText('Save changes', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const patches = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'PATCH'
	);
	expect(patches).toHaveLength(1);
	expect(document.body.querySelector('.album-bar__title')?.textContent).toBe('Road trip 2026 -- rerun');
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the properties dialog edits the audience: the full-set PUT and the converged chips', async () => {
	// Signed in as ada (u1, the owner): the current grant (grace)
	// seeds the chips, the picker offers the other accounts (the
	// caller and the grant excluded), and saving PUTs the COMPLETE
	// set -- the added id and nothing else.
	let audienceSaved = false;
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1/audience': (init?: RequestInit) => {
				expect(init?.method).toBe('PUT');
				const body = JSON.parse(String(init?.body));
				expect(body).toEqual({ audience: ['u2', 'u3'] });
				audienceSaved = true;
				return jsonResponse(200, {
					albumId: 'al1',
					audience: [
						{ userId: 'u2', email: 'grace@example.com', grantedAt: '2026-11-05T09:30:00Z' },
						{ userId: 'u3', email: 'ivy@example.com', grantedAt: '2026-11-05T12:00:00Z' }
					],
					granted: ['u3'],
					revoked: [],
					skipped: []
				});
			},
			'/api/v1/albums/al1?limit=1': () => {
				if (audienceSaved) {
					return jsonResponse(200, {
						...detailState,
						isShared: undefined,
						shares: [
							{ userId: 'u2', email: 'grace@example.com', grantedAt: '2026-11-05T09:30:00Z' },
							{ userId: 'u3', email: 'ivy@example.com', grantedAt: '2026-11-05T12:00:00Z' }
						]
					});
				}
				return jsonResponse(200, detailState);
			}
		})
	);
	await bootDetailSession('u1', 'admin');
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Rename').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();

	// The chips seed from the current grants.
	const chipNames = Array.from(dialog?.querySelectorAll<HTMLElement>('.audience-field__chip-name') ?? []);
	expect(chipNames.map((chip) => chip.textContent)).toEqual(['grace@example.com']);

	// The picker: the other accounts (the caller and the grant
	// excluded -- hopper renders with the honest disabled note).
	// The rows live in the PORTALED floating layer (the floating-layer revision:
	// document.body, outside the dialog's DOM) -- document-level
	// selectors find them.
	buttonByText('Add', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__row') ?? []);
	expect(rows.map((row) => row.querySelector('.audience-field__row-name')?.textContent)).toEqual([
		'ivy@example.com',
		'hopper@example.com'
	]);
	expect(rows[1]?.textContent).toContain('cannot sign in');

	// The keyboard contract: the listbox drives, the loaded
	// set highlights its first row, Enter adds the highlighted
	// candidate, the picker stays open.
	const listbox = document.body.querySelector<HTMLElement>('#audience-field-picker');
	expect(listbox?.querySelectorAll('[role="option"]')[0]?.getAttribute('aria-selected')).toBe('true');
	listbox?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
	flushSync();
	const chipNamesAfter = Array.from(dialog?.querySelectorAll<HTMLElement>('.audience-field__chip-name') ?? []);
	expect(chipNamesAfter.map((chip) => chip.textContent)).toEqual(['grace@example.com', 'ivy@example.com']);

	// Esc closes the picker alone -- the dialog stays. (The portaled
	// layer leaves the document with the close; the dialog's own DOM
	// never carried it.)
	listbox?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
	flushSync();
	expect(document.body.querySelector('#audience-field-picker')).toBeNull();
	expect(modal()).not.toBeNull();

	buttonByText('Save changes', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1/audience' && init?.method === 'PUT'
	);
	expect(puts).toHaveLength(1);
	expect(modal()).toBeNull();

	// The converged detail's grantee list rides the next open: the
	// chips reseed from the refetched shares.
	buttonByLabel('Rename').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const chipNamesConverged = Array.from(modal()?.querySelectorAll<HTMLElement>('.audience-field__chip-name') ?? []);
	expect(chipNamesConverged.map((chip) => chip.textContent)).toEqual(['grace@example.com', 'ivy@example.com']);
	unmount(page);
	flushSync();
});

test('a failed audience write keeps the dialog with the mapped error', async () => {
	await bootDetailSession('u1', 'admin');
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1/audience': () =>
				jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'view-only share' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Rename').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	buttonByText('Add', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	// The rows live in the portaled floating layer (document-level).
	const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__row') ?? []);
	rows[0]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();
	buttonByText('Save changes', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(modal()?.textContent).toContain("You don't have permission to do this.");
	unmount(page);
	flushSync();
});

test('a no-change save writes nothing at all', async () => {
	// The full-set semantics' honest idle: the same title and the
	// same audience -- no PATCH (a same-value patch would still
	// publish), no PUT (the idempotent no-op), just the close.
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Rename').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	buttonByText('Save changes', modal() as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const writes = fetchMock.mock.calls.filter(
		([input, init]) =>
			(String(input) === '/api/v1/albums/al1' && init?.method === 'PATCH') ||
			(String(input) === '/api/v1/albums/al1/audience' && init?.method === 'PUT')
	);
	expect(writes).toHaveLength(0);
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the directory properties dialog shows the read-only source row', async () => {
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		inTimeline: true
	};
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	buttonByLabel('Properties').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	const source = dialog?.querySelector<HTMLElement>('.album-form__source');
	expect(source?.querySelector('.album-form__source-path')?.textContent).toBe('/Volumes/Photos/family');
	// The honest tag: the source is pinned at creation.
	expect(source?.querySelector('.album-form__source-tag')?.textContent).toBe('Source · fixed');
	unmount(page);
	flushSync();
});

test('the deletability row renders for a writable source and PATCHes the switch', async () => {
	let deletabilitySaved = false;
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		sourceWritable: true,
		allowAppDeletion: true,
		deletable: true,
		inTimeline: true
	};
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1': (init?: RequestInit) => {
				expect(JSON.parse(String(init?.body))).toEqual({ allowAppDeletion: false });
				deletabilitySaved = true;
				return jsonResponse(200, summary({ type: 'directory' }));
			},
			'/api/v1/albums/al1?limit=1': () =>
				deletabilitySaved
					? jsonResponse(200, { ...detailState, allowAppDeletion: false, deletable: false })
					: jsonResponse(200, detailState)
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Properties').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	const toggle = dialog?.querySelector<HTMLInputElement>('[data-testid="album-deletable-toggle"]');
	expect(toggle).not.toBeNull();
	expect(toggle?.checked).toBe(true);
	toggle!.checked = false;
	toggle!.dispatchEvent(new Event('change', { bubbles: true }));
	await settle();
	buttonByText('Save changes', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const patches = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'PATCH'
	);
	expect(patches).toHaveLength(1);
	expect(modal()).toBeNull();

	// The converged face: the derived affordance dies with the
	// switch, so the wall's selection machinery unwires -- no check,
	// no bar (the save's refetch carried deletable: false).
	await settle();
	expect(tiles()[0]?.querySelector('.tile__check')).toBeNull();
	unmount(page);
	flushSync();
});

test('a read-only source renders NO deletability row and NO wall selection (the row does not exist)', async () => {
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/mnt/readonly/archive',
		sourceWritable: false,
		allowAppDeletion: true,
		deletable: false,
		inTimeline: true
	};
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	// The wall is pure navigation: the derived face is false, no
	// check ever renders.
	expect(tiles()[0]?.querySelector('.tile__check')).toBeNull();

	buttonByLabel('Properties').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	// The row does not EXIST for a read-only source (no dead
	// affordance) -- the exclusion row stands, and the retired lock
	// leaves no row of its own.
	expect(dialog?.querySelector('[data-testid="album-deletable-toggle"]')).toBeNull();
	expect(dialog?.querySelector('[data-testid="album-lock-toggle"]')).toBeNull();
	expect(dialog?.querySelector('[data-testid="album-timeline-toggle"]')).not.toBeNull();
	unmount(page);
	flushSync();
});

test("the dialog's exclusion row writes the caller's own preference on save", async () => {
	let preferenceSaved = false;
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		sourceWritable: true,
		allowAppDeletion: true,
		deletable: true,
		inTimeline: true
	};
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1/timeline-preference': (init?: RequestInit) => {
				expect(JSON.parse(String(init?.body))).toEqual({ inTimeline: false });
				preferenceSaved = true;
				return jsonResponse(200, { albumId: 'al1', inTimeline: false });
			},
			'/api/v1/albums/al1?limit=1': () =>
				preferenceSaved
					? jsonResponse(200, { ...detailState, inTimeline: false })
					: jsonResponse(200, detailState)
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Properties').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	const toggle = dialog?.querySelector<HTMLInputElement>('[data-testid="album-timeline-toggle"]');
	expect(toggle?.checked).toBe(true);
	toggle!.checked = false;
	toggle!.dispatchEvent(new Event('change', { bubbles: true }));
	await settle();
	buttonByText('Save changes', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/albums/al1/timeline-preference');
	expect(puts).toHaveLength(1);
	// The badge and notice follow the converged preference.
	expect(document.body.querySelector('.album-detail__badge--excluded')?.textContent?.trim()).toBe('Not in timeline');
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the retired lock leaves no trace: no lock row, no frozen fields, one plain diffed save', async () => {
	// The lock's retirement is whole at the web face too: the dialog
	// carries no lock row, no field ever freezes, and a save writes
	// the plain diff of its draft -- no unlock-first sequencing, no
	// notice line, no badge (the deletion/rescan affordances obey
	// their own faces, never a lock).
	let titleSaved = false;
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		sourceWritable: true,
		allowAppDeletion: true,
		deletable: true,
		inTimeline: true
	};
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1': (init?: RequestInit) => {
				// The save writes the property diff in ONE patch.
				expect(JSON.parse(String(init?.body))).toEqual({ title: 'Family archive 2026' });
				titleSaved = true;
				return jsonResponse(200, summary({ type: 'directory', title: 'Family archive 2026' }));
			},
			'/api/v1/albums/al1?limit=1': () =>
				titleSaved
					? jsonResponse(200, { ...detailState, title: 'Family archive 2026' })
					: jsonResponse(200, detailState)
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Properties').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();

	// No lock row in the autonomy block; the fields are live.
	expect(dialog?.querySelector('[data-testid="album-lock-toggle"]')).toBeNull();
	expect(dialog?.querySelector<HTMLInputElement>('#album-title')?.disabled).toBe(false);
	expect(document.body.querySelector('.album-detail__badge--locked')).toBeNull();
	expect(document.body.querySelector('.album-detail__locked-notice')).toBeNull();

	fillField(dialog as ParentNode, 'Family archive 2026', 'album-title');
	buttonByText('Save changes', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const patches = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'PATCH'
	);
	expect(patches).toHaveLength(1);
	expect(document.body.querySelector('.album-bar__title')?.textContent).toBe('Family archive 2026');
	expect(modal()).toBeNull();
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

	buttonByLabel('Set cover…').dispatchEvent(new MouseEvent('click', { bubbles: true }));
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

	buttonByLabel('Set cover…').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.textContent).toContain('This album has no photos yet.');
	unmount(page);
	flushSync();
});

test('the rescan trigger POSTs and the job events drive the progress line', async () => {
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		itemCount: 0,
		inTimeline: true
	};
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/assets/timeline?albumId=al1&limit=100': () => jsonResponse(200, { items: [] }),
			'/api/v1/albums/al1/rescan': (init?: RequestInit) => {
				expect(init?.method).toBe('POST');
				return jsonResponse(202, { jobRunId: 'run-1' });
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// No run line before any trigger.
	expect(document.body.querySelector('[data-testid="album-rescan-progress"]')).toBeNull();

	buttonByLabel('Rescan').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1/rescan' && init?.method === 'POST'
	);
	expect(posts).toHaveLength(1);

	// The queued line lands (aria-live, the models download page's
	// mechanism), and the trigger disables while the run is live.
	let progress = document.body.querySelector<HTMLElement>('[data-testid="album-rescan-progress"]');
	expect(progress?.getAttribute('aria-live')).toBe('polite');
	expect(progress?.textContent).toContain('Rescan queued…');
	expect(buttonByLabel('Rescan').disabled).toBe(true);

	// The run's progress events (keyed by the identifier the trigger
	// answered) drive the line and the bar in place.
	fireJobEvent({
		id: 1,
		name: 'job.progress',
		jobRunId: 'run-1',
		kind: 'album_rescan',
		total: 12,
		done: 4,
		failed: 0,
		state: 'running'
	});
	await settle();
	progress = document.body.querySelector<HTMLElement>('[data-testid="album-rescan-progress"]');
	expect(progress?.textContent).toContain('Scanning… 4 of 12');
	const bar = progress?.querySelector<HTMLElement>('[role="progressbar"]');
	expect(bar?.getAttribute('aria-valuenow')).toBe('4');
	expect(bar?.getAttribute('aria-valuemax')).toBe('12');

	// A run of ANOTHER album (or another kind) never touches this
	// line -- the identifier keys the convergence.
	fireJobEvent({
		id: 2,
		name: 'job.progress',
		jobRunId: 'run-elsewhere',
		kind: 'album_rescan',
		total: 99,
		done: 99,
		failed: 0,
		state: 'running'
	});
	fireJobEvent({
		id: 3,
		name: 'job.progress',
		jobRunId: 'run-other-kind',
		kind: 'model_download',
		total: 50,
		done: 50,
		failed: 0,
		state: 'running'
	});
	await settle();
	progress = document.body.querySelector<HTMLElement>('[data-testid="album-rescan-progress"]');
	expect(progress?.textContent).toContain('Scanning… 4 of 12');

	// The finish lands the final state and re-enables the trigger.
	fireJobEvent({
		id: 4,
		name: 'job.finished',
		jobRunId: 'run-1',
		kind: 'album_rescan',
		total: 12,
		done: 12,
		failed: 0,
		state: 'succeeded'
	});
	await settle();
	progress = document.body.querySelector<HTMLElement>('[data-testid="album-rescan-progress"]');
	expect(progress?.textContent).toContain('Rescan complete.');
	expect(buttonByLabel('Rescan').disabled).toBe(false);
	unmount(page);
	flushSync();
});

test('a failed rescan trigger surfaces the mapped error on the shared line', async () => {
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		inTimeline: true
	};
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/albums/al1/rescan': () => jsonResponse(500, { code: 'INTERNAL', message: 'boom' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Rescan').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// No run line (nothing started); the bar's shared error line
	// carries the server's own answer.
	expect(document.body.querySelector('[data-testid="album-rescan-progress"]')).toBeNull();
	expect(document.body.querySelector('.album-detail__action-error')?.textContent).toContain('Something went wrong.');
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

	buttonByLabel('Delete album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Delete this album?');
	expect(dialog?.textContent).toContain('The photos themselves stay in your library.');
	buttonByText('Delete album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const deletes = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'DELETE'
	);
	expect(deletes).toHaveLength(1);
	expect(vi.mocked(goto)).toHaveBeenCalledWith('/albums');
	unmount(page);
	flushSync();
});

test('removing the directory album states the semantics and leaves for the wall', async () => {
	detailState = {
		...detailState,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		sourceWritable: true,
		allowAppDeletion: true,
		deletable: true,
		inTimeline: true
	};
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

	// The remove affordance opens its own confirm -- the un-ingest's
	// dialog, not the memory delete's.
	buttonByLabel('Remove album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Remove this album?');
	// THE PINNED SEMANTICS in the body: the album and its photo data
	// go, the source folder stays untouched.
	expect(dialog?.textContent).toContain('the source folder on disk stays untouched');
	buttonByText('Remove album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const removes = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums/al1' && init?.method === 'DELETE'
	);
	expect(removes).toHaveLength(1);
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

	buttonByLabel('Keep out of timeline').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/albums/al1/timeline-preference');
	expect(puts).toHaveLength(1);
	expect(document.body.querySelector('.album-detail__badge--excluded')?.textContent?.trim()).toBe('Not in timeline');
	expect(document.body.querySelector('.album-detail__excluded-notice')?.textContent).toContain(
		'stay out of your timeline'
	);
	// The toggle flipped: the action now offers the include.
	expect(() => buttonByLabel('Keep in timeline')).not.toThrow();
	unmount(page);
	flushSync();
});

test('an unknown or invisible album renders the not-found error with a retry', async () => {
	let failing = true;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
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

	expect(document.body.querySelector('.album-bar__title')?.textContent).toBe('Road trip 2026');
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
		fetchMock.mock.calls.filter(([input]) => String(input).startsWith('/api/v1/albums/al1?limit=1')).length
	).toBe(initial + 1);
	expect(document.body.querySelector('.album-bar__title')?.textContent).toBe('Road trip 2026 -- rerun');
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
		fetchMock.mock.calls.filter(([input]) => String(input).startsWith('/api/v1/assets/timeline?albumId=al1')).length
	).toBe(initial + 1);
	unmount(page);
	flushSync();
});

test('SSE album.deleted renders the not-found error state', async () => {
	let deleted = false;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
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
	// The detail's album listener, the wall's asset listener and the
	// rescan progress's job listener.
	expect(albumHandlers).toHaveLength(1);
	expect(assetHandlers).toHaveLength(1);
	expect(jobHandlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(albumHandlers).toHaveLength(0);
	expect(assetHandlers).toHaveLength(0);
	expect(jobHandlers).toHaveLength(0);
});

/** Boot the session store signed in as one of the fixture accounts. */
async function bootDetailSession(userId: string, role: 'admin' | 'user'): Promise<void> {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			jsonResponse(200, [
				{
					id: '6f9c2d18-9a4c-4a1b-b1a1-000000000001',
					kind: 'WEB',
					userId,
					role,
					deviceName: 'Mozilla/5.0 (test)',
					createdAt: '2026-11-01T10:00:00Z',
					lastSeenAt: '2026-11-02T10:00:00Z'
				}
			])
		)
	);
	await probeSession();
}
