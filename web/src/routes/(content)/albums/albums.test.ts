import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearToasts, toasts } from '$lib/components/toast.svelte';
import type { AlbumSummary } from '$lib/api/albums';
import Page from './+page.svelte';

// The album wall: the cover-card grid in its two
// sections -- the caller's own albums (the pinned sync card of the
// default "My photos" album first, its localized display name keyed
// by `type === 'sync'`), then the albums shared to them (with the
// honest owner attribution) -- with the collage covers (one
// full-bleed, two split, three 2+1, four quad), the shared /
// excluded badges, the counts and the honest owner attribution on
// the shared cards. The "New album" dialog creates one directory
// album -- the entry is the ADMINISTRATOR'S alone: pointing
// at the file system is configuration, a member
// never sees the affordance), and the create's honest answers (the
// skipped report, the write probe's read-only fact) hold the dialog
// until acknowledged; the dialog's autonomy block carries the two
// switches (the exclusion always, the deletability row exactly for
// a writable pick). Every album.* SSE event converges through ONE
// fresh wall fetch (the collapsed single-flight of the people
// page), and the refetched SERVER answer decides what changed. The
// events module is mocked with a captured handler registry, the
// people page's pattern.

const { handlers } = vi.hoisted(() => ({
	handlers: [] as Array<(event: unknown) => void>
}));

vi.mock('$lib/events', () => ({
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

function summary(id: string, overrides: Partial<AlbumSummary> = {}): AlbumSummary {
	return {
		id,
		title: `Album ${id}`,
		type: 'directory',
		isShared: false,
		coverAssetIds: [],
		itemCount: 0,
		photoCount: 0,
		updatedAt: '2026-11-01T10:00:00Z',
		callerRole: 'owner',
		...overrides
	};
}

const roadTrip = summary('al1', {
	title: 'Road trip 2026',
	coverAssetIds: ['a1', 'a2', 'a3', 'a4'],
	itemCount: 128,
	photoCount: 128,
	isShared: true
});
// The pinned default card: the caller's own sync album, exactly the
// server's shape (the internal title 'Sync', the type the display
// name keys on, no excludable field).
const myPhotos = summary('sync1', {
	title: 'Sync',
	type: 'sync',
	coverAssetIds: ['s1', 's2'],
	itemCount: 1204,
	photoCount: 1204
});
const familyArchive = summary('al2', {
	title: 'Family archive',
	type: 'directory',
	itemCount: 2048,
	photoCount: 2048,
	inTimeline: false
});
const quickPicks = summary('al3', {
	title: 'Quick picks',
	coverAssetIds: ['a9'],
	itemCount: 1,
	photoCount: 1
});
const graceHighlights = summary('al4', {
	title: "Grace's highlights",
	coverAssetIds: ['g1', 'g2'],
	// The shared card's live members include three videos: the count
	// beside the image glyph reads the PHOTO count (17), never the
	// member count (20) -- the acceptance rider's own regression pin.
	itemCount: 20,
	photoCount: 17,
	callerRole: 'viewer',
	owner: { id: 'u2', name: 'Grace Hopper', email: 'grace@example.com' }
});

// The server's current answer; the convergence tests mutate it to
// stand for the server's post-event truth.
let wallState: AlbumSummary[] = [myPhotos, roadTrip, familyArchive, quickPicks, graceHighlights];

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

function wallRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/albums': (init?: RequestInit) => {
			if (init?.method === 'POST') {
				const body = JSON.parse(String(init.body)) as { title: string };
				const created = summary('al-new', { title: body.title, itemCount: 0, type: 'directory' });
				gainOwnCard(created);
				// The creation answer always carries the honest skipped
				// report and the write probe's fact (writable here; the
				// read-only arm is its own test below).
				return jsonResponse(201, { ...created, skipped: [], sourceWritable: true });
			}
			return jsonResponse(200, { albums: wallState });
		},
		...overrides
	};
}

/** Mirror the server's answer: the pinned sync card stays first, the
 * new own card heads the curated rest of the own group. */
function gainOwnCard(album: AlbumSummary): void {
	const pin = wallState[0]?.type === 'sync' ? [wallState[0]] : [];
	wallState = [...pin, album, ...wallState.slice(pin.length)];
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
}

function mountPage(): ReturnType<typeof mount> {
	const page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
}

function cards(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.album-card'));
}

function buttonByText(text: string, scope: ParentNode = document.body): HTMLButtonElement {
	const buttons = Array.from(scope.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
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

/** The instance's minimal member list (the audience picker's source). */
const usersBody = {
	users: [
		{ id: 'u1', name: 'Ada Lovelace', email: 'ada@example.com', role: 'admin', isDisabled: false },
		{ id: 'u2', name: 'Grace Hopper', email: 'grace@example.com', role: 'user', isDisabled: false },
		{ id: 'u3', name: 'Ivy Lee', email: 'ivy@example.com', role: 'user', isDisabled: false }
	]
};

/** Boot the session store signed in as one of the fixture accounts. */
async function bootWallSession(userId: string, role: 'admin' | 'user'): Promise<void> {
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
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
}

/** Fire one parsed album event to every live subscriber. */
function fireAlbumEvent(event: Record<string, unknown>): void {
	for (const handler of [...handlers]) {
		handler(event);
	}
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	vi.clearAllMocks();
	clearToasts();
	wallState = [myPhotos, roadTrip, familyArchive, quickPicks, graceHighlights];
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

test('the wall renders ONE list: own and shared cards mixed, no group labels', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	// No sections, no group labels (the design's single list); the
	// page's one 20px title lives in the bar with its count.
	const bar = document.body.querySelector('.albums__toolbar');
	expect(bar?.querySelector('.albums__title')?.textContent?.replace(/\s+/g, ' ').trim()).toBe('Albums · 5 albums');
	expect(document.body.querySelector('.albums__own-section')).toBeNull();
	expect(document.body.querySelector('.albums__shared-section')).toBeNull();
	expect(document.body.querySelector('.albums__section-title')).toBeNull();

	// One grid holds both roles: the sync card pinned first, then the
	// cards in the server's order (their recency ties here).
	expect(cards()).toHaveLength(5);
	const sharedCard = cards()[4];
	expect(sharedCard?.querySelector('.album-card__title')?.textContent).toBe("Grace's highlights");
	// The attribution chip names the owner, and the read-only badge
	// sits beside it on the same row.
	expect(sharedCard?.querySelector('.album-card__badge--owner')?.textContent?.trim()).toBe('Shared by Grace Hopper');
	expect(sharedCard?.querySelector('.album-card__badges .album-card__badge--readonly')?.textContent?.trim()).toBe(
		'Read-only'
	);
	// Cards link into the album's detail.
	expect(cards()[1]?.getAttribute('href')).toBe('/albums/al1');
	expect(sharedCard?.getAttribute('href')).toBe('/albums/al4');

	unmount(page);
	flushSync();
});

test('one list orders by the cards own recency with the sync card pinned first', async () => {
	wallState = [
		summary('al-old', { title: 'Old own', updatedAt: '2026-10-01T10:00:00Z' }),
		summary('al-shared', {
			title: 'Fresh shared',
			updatedAt: '2026-11-10T10:00:00Z',
			callerRole: 'viewer',
			owner: { id: 'u2', name: 'Grace Hopper', email: 'grace@example.com' }
		}),
		myPhotos,
		summary('al-new', { title: 'New own', updatedAt: '2026-11-05T10:00:00Z' })
	];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	// The default album stays the wall's first card; the rest is one
	// mixed list, most recently updated first (own and shared alike).
	expect(cards().map((card) => card.querySelector('.album-card__title')?.textContent)).toEqual([
		'My photos',
		'Fresh shared',
		'New own',
		'Old own'
	]);

	unmount(page);
	flushSync();
});

test('an owner who set no display name falls back to the email', async () => {
	// The account's name is the empty string (the contract serves it as
	// stored): the attribution renders the email -- a blank chip never
	// renders.
	wallState = [
		summary('al-nameless', {
			title: 'Nameless picks',
			callerRole: 'viewer',
			owner: { id: 'u9', name: '', email: 'nameless@example.com' }
		})
	];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	expect(cards()[0]?.querySelector('.album-card__badge--owner')?.textContent?.trim()).toBe(
		'Shared by nameless@example.com'
	);

	unmount(page);
	flushSync();
});

test('two owners under one name get the email as the chip title', async () => {
	// The name is not a key: two accounts called "Mum" share albums
	// with the caller, so each chip carries its owner's address as the
	// hover title.
	wallState = [
		summary('al-mum-a', {
			title: 'Mum one',
			updatedAt: '2026-11-02T10:00:00Z',
			callerRole: 'viewer',
			owner: { id: 'u5', name: 'Mum', email: 'mum@example.com' }
		}),
		summary('al-mum-b', {
			title: 'Mum two',
			updatedAt: '2026-11-01T10:00:00Z',
			callerRole: 'viewer',
			owner: { id: 'u6', name: 'Mum', email: 'mum2@example.com' }
		})
	];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	const chips = Array.from(document.body.querySelectorAll<HTMLElement>('.album-card__badge--owner'));
	expect(chips.map((chip) => chip.textContent?.trim())).toEqual(['Shared by Mum', 'Shared by Mum']);
	expect(chips.map((chip) => chip.getAttribute('title'))).toEqual(['mum@example.com', 'mum2@example.com']);

	unmount(page);
	flushSync();
});

test('the memory rows leave the album wall: the reference sets live on their own route', async () => {
	// The memories moved to their own wall; the album wall filters
	// the memory type out of BOTH groups -- the containers (the sync
	// card, the directory albums) are this page's alone.
	wallState = [
		myPhotos,
		familyArchive,
		summary('mem-own', { title: 'Spring Festival 2026', type: 'memory', itemCount: 12 }),
		summary('mem-shared', {
			title: 'Sanya trip',
			type: 'memory',
			callerRole: 'viewer',
			owner: { id: 'u2', name: 'Grace Hopper', email: 'grace@example.com' }
		})
	];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	expect(cards()).toHaveLength(2);
	for (const card of cards()) {
		expect(card.getAttribute('href')?.startsWith('/memories/')).toBe(false);
	}

	unmount(page);
	flushSync();
});

test('a wall holding only memories renders the empty state: their home is elsewhere', async () => {
	wallState = [summary('mem-own', { title: 'Spring Festival 2026', type: 'memory', itemCount: 12 })];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	expect(cards()).toHaveLength(0);
	expect(document.body.querySelector('.state-card')?.textContent).toContain(
		'Gather a trip or an occasion into an album.'
	);

	unmount(page);
	flushSync();
});

test('the pinned sync card leads the wall with the localized display name', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	const ownCards = [cards()[0]];

	// The first card is the caller's sync album: the display
	// name is the i18n key's, the stored internal title never
	// renders.
	expect(ownCards[0]?.querySelector('.album-card__title')?.textContent).toBe('My photos');
	expect(ownCards[0]?.textContent).not.toContain('Sync');
	expect(ownCards[0]?.getAttribute('href')).toBe('/albums/sync1');
	// The honest count and the derived cover, same as any album: the
	// count renders the glyph and the grouped number, the wording on
	// the accessible name.
	const syncCount = ownCards[0]?.querySelector('.album-card__count');
	expect(syncCount?.textContent?.trim()).toBe('1,204');
	expect(syncCount?.getAttribute('aria-label')).toBe('1,204 photos');
	const syncCoverImgs = Array.from(ownCards[0]?.querySelectorAll('img') ?? []).map(
		(img) => img.getAttribute('src') ?? ''
	);
	expect(syncCoverImgs).toEqual(['/api/v1/assets/s1/thumbnail?size=tile', '/api/v1/assets/s2/thumbnail?size=tile']);
	expect(ownCards[0]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--split')).toBe(true);
	// No badges: no share, no exclusion -- the default
	// album is system-managed and never excludable.
	expect(ownCards[0]?.querySelector('.album-card__badge--shared')).toBeNull();
	expect(ownCards[0]?.querySelector('.album-card__badge--excluded')).toBeNull();

	unmount(page);
	flushSync();
});

test('the collage cover renders per the member count: 4 quad, 2 split, 1 full-bleed, 0 bare', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	const coverImgs = (index: number): string[] =>
		(cards()[index]?.querySelectorAll('img') ?? []).length > 0
			? Array.from(cards()[index]?.querySelectorAll('img') ?? []).map((img) => img.getAttribute('src') ?? '')
			: [];

	// Four covers: the quad collage, tile-size serving copies.
	expect(coverImgs(1)).toEqual([
		'/api/v1/assets/a1/thumbnail?size=tile',
		'/api/v1/assets/a2/thumbnail?size=tile',
		'/api/v1/assets/a3/thumbnail?size=tile',
		'/api/v1/assets/a4/thumbnail?size=tile'
	]);
	expect(cards()[1]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--quad')).toBe(true);

	// Zero covers: the bare placeholder, no requests.
	expect(coverImgs(2)).toEqual([]);
	expect(cards()[2]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--empty')).toBe(true);

	// One cover: the full-bleed pin.
	expect(coverImgs(3)).toEqual(['/api/v1/assets/a9/thumbnail?size=tile']);
	expect(cards()[3]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--single')).toBe(true);

	// Two covers: the split pair.
	expect(coverImgs(4)).toEqual(['/api/v1/assets/g1/thumbnail?size=tile', '/api/v1/assets/g2/thumbnail?size=tile']);
	expect(cards()[4]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--split')).toBe(true);

	unmount(page);
	flushSync();
});

test('three covers fall to the 2+1 collage: two cells, then one spanning both', async () => {
	wallState = [summary('al5', { title: 'Trio', coverAssetIds: ['t1', 't2', 't3'] })];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	const cover = cards()[0]?.querySelector('.album-card__cover');
	expect(cover?.classList.contains('album-card__cover--duo-plus-one')).toBe(true);
	expect(cover?.querySelectorAll('img')).toHaveLength(3);
	const cells = Array.from(cover?.querySelectorAll<HTMLElement>('.album-card__cell') ?? []);
	// The third frame spans both columns (the duo-plus-one fall's wide
	// cell).
	expect(cells[2]?.className).toContain('col-span-2');
	unmount(page);
	flushSync();
});

test('the badges ride the cards: shared, read-only, excluded', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	// An own card carrying a share row: the owner's "someone else sees
	// this" badge -- and no read-only chip (that is the member's face).
	const roadTripCard = cards()[1];
	expect(roadTripCard?.querySelector('.album-card__badge--shared')?.textContent?.trim()).toBe('Shared');
	expect(roadTripCard?.querySelector('.album-card__badge--readonly')).toBeNull();
	expect(roadTripCard?.querySelector('.album-card__count')?.getAttribute('aria-label')).toBe('128 photos');

	// The exclusion badge rides the excludable cards only: the own
	// directory item here, and every shared item.
	const archiveCard = cards()[2];
	expect(archiveCard?.querySelector('.album-card__badge--excluded')?.textContent?.trim()).toBe('Not in timeline');

	// The shared card: no "shared" badge (the attribution says the same
	// thing from the member's side), the read-only chip beside it.
	const sharedCard = cards()[4];
	expect(sharedCard?.querySelector('.album-card__badge--shared')).toBeNull();
	expect(sharedCard?.querySelector('.album-card__badge--readonly')?.textContent?.trim()).toBe('Read-only');
	expect(sharedCard?.querySelector('.album-card__badge--excluded')).toBeNull();
	expect(sharedCard?.querySelector('.album-card__count')?.getAttribute('aria-label')).toBe('17 photos');

	// A card with no preference row renders no badge (the true
	// default): the exclusion badge rides the field alone.
	expect(cards()[3]?.querySelector('.album-card__badge--excluded')).toBeNull();

	// The badge row is always there -- an empty one keeps the cards
	// equal height -- and the attribution never takes a line of its own.
	for (const card of cards()) {
		expect(card.querySelector('.album-card__badges')).not.toBeNull();
	}

	// The lock badge is retired: no card renders one, whatever the
	// album's history.
	for (const card of cards()) {
		expect(card.querySelector('.album-card__badge--locked')).toBeNull();
	}

	unmount(page);
	flushSync();
});

test("an empty wall renders the empty state; the create affordance is the admin's alone", async () => {
	// An admin's empty wall carries the creation entry.
	await bootWallSession('u1', 'admin');
	wallState = [];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.state-card')?.textContent).toContain(
		'Gather a trip or an occasion into an album.'
	);
	expect(cards()).toHaveLength(0);
	expect(buttonByText('New album')).toBeTruthy();
	unmount(page);
	flushSync();

	// A member's wall (and the unauthenticated boot of the other
	// tests) renders NO creation affordance at all: pointing at the
	// file system is configuration, the server refuses the member's
	// POST anyway -- the visual gate is the honest half.
	await bootWallSession('u2', 'user');
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const memberPage = mountPage();
	await settle();
	const memberButton = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.textContent?.trim() === 'New album'
	);
	expect(memberButton).toBeUndefined();
	unmount(memberPage);
	flushSync();
});

test('a failed load shows the error box with a working retry', async () => {
	let failing = true;
	const fetchMock = vi.fn(async () => {
		if (failing) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return jsonResponse(200, { albums: wallState });
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.state-card--error');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('Something went wrong.');

	failing = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(cards()).toHaveLength(5);
	unmount(page);
	flushSync();
});

test('a dead session renders the session message, never credential text', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.state-card--error');
	expect(failure?.textContent).toContain('Your session has ended.');
	expect(failure?.textContent).not.toContain('Incorrect email or password');
	unmount(page);
	flushSync();
});

test('the create dialog POSTs the title, path and audience; the refetched wall gains the card', async () => {
	// The creation entry is the administrator's: the wall's
	// button renders for an admin's session alone.
	await bootWallSession('u1', 'admin');
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(5);

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('New album');
	const fields = Array.from(dialog?.querySelectorAll<HTMLInputElement>('input[id^="create-album-"]') ?? []);
	// The pinned order: the title above, the path below.
	expect(fields.map((field) => field.id)).toEqual(['create-album-title', 'create-album-path']);
	fillField(dialog as ParentNode, 'Weekend in the hills', 'create-album-title');
	fillField(dialog as ParentNode, '/Volumes/Photos/hills', 'create-album-path');
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums' && init?.method === 'POST'
	);
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		title: 'Weekend in the hills',
		rootPath: '/Volumes/Photos/hills',
		audience: [],
		inTimeline: true
	});
	// The refetch placed the created card after the pinned sync card,
	// at the head of the curated own group.
	expect(cards()[1]?.querySelector('.album-card__title')?.textContent).toBe('Weekend in the hills');
	expect(cards()[0]?.querySelector('.album-card__title')?.textContent).toBe('My photos');
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the create dialog renders the deletability row from the picked completion row and carries the switches', async () => {
	// The picked directory's writability is known at selection time
	// (the completion row's write fact): the writable pick renders
	// the deletability row (checked on, the server's default), the
	// read-only pick renders no row at all -- no dead affordance --
	// and the honest probe feedback explains the fact on the report
	// face. The fact rides the exact picked path: a hand edit drops
	// the row with it.
	await bootWallSession('u1', 'admin');
	const fetchMock = stubFetch(
		wallRoutes({
			'/api/v1/albums/path-completions?prefix=%2Fmnt': () =>
				jsonResponse(200, {
					completions: [
						{ path: '/mnt/rw-photos', writable: true },
						{ path: '/mnt/ro-photos', writable: false }
					]
				})
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	fillField(dialog as ParentNode, 'Family archive', 'create-album-title');

	// The block's initial shape: the exclusion toggle on (the album
	// feeds the creator's timeline), no deletability row (nothing
	// picked yet).
	expect(dialog?.querySelector('[data-testid="album-timeline-toggle"]')).not.toBeNull();
	const timelineToggle = dialog?.querySelector<HTMLInputElement>('[data-testid="album-timeline-toggle"]');
	expect(timelineToggle?.checked).toBe(true);
	expect(dialog?.querySelector('[data-testid="album-deletable-toggle"]')).toBeNull();

	// The typed prefix lands the completion rows in the portaled
	// layer; the writable row's pick renders the deletability row.
	fillField(dialog as ParentNode, '/mnt', 'create-album-path');
	await new Promise((resolve) => setTimeout(resolve, 300));
	await settle();
	const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.path-completion__row'));
	expect(rows.map((row) => row.textContent?.trim())).toEqual(['/mnt/rw-photos', '/mnt/ro-photos']);
	rows[0]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();

	const deletableToggle = dialog?.querySelector<HTMLInputElement>('[data-testid="album-deletable-toggle"]');
	expect(deletableToggle).not.toBeNull();
	expect(deletableToggle?.checked).toBe(true);

	// The fact rides the exact picked path: a hand edit drops the
	// stale row (and its draft) immediately -- no dead affordance
	// over a path the fact no longer describes.
	fillField(dialog as ParentNode, '/mnt/rw', 'create-album-path');
	flushSync();
	expect(dialog?.querySelector('[data-testid="album-deletable-toggle"]')).toBeNull();

	// The read-only pick: re-drill, pick the ro row -- the row does
	// not render at all (the honest absent-row arm).
	fillField(dialog as ParentNode, '/mnt', 'create-album-path');
	await new Promise((resolve) => setTimeout(resolve, 300));
	await settle();
	const secondRows = Array.from(document.body.querySelectorAll<HTMLElement>('.path-completion__row'));
	secondRows[1]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();
	expect(dialog?.querySelector('[data-testid="album-deletable-toggle"]')).toBeNull();

	// The exclusion off; the POST carries the exclusion draft and no
	// deletability switch (the row did not exist for this pick).
	timelineToggle!.checked = false;
	timelineToggle!.dispatchEvent(new Event('change', { bubbles: true }));
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums' && init?.method === 'POST'
	);
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		title: 'Family archive',
		rootPath: '/mnt/ro-photos',
		audience: [],
		inTimeline: false
	});
	unmount(page);
	flushSync();
});

test('the create dialog carries the picked audience into the POST', async () => {
	// Signed in as ada (u1, an admin -- the creation entry's gate):
	// the add-style picker's candidates are the other accounts (the
	// identity seam's self filter). The users listing feeds the
	// candidate rows; picking one adds its chip, and the POST carries
	// its id.
	await bootWallSession('u1', 'admin');
	const fetchMock = stubFetch(
		wallRoutes({
			'/api/v1/users': () => jsonResponse(200, usersBody)
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	fillField(dialog as ParentNode, 'Family archive', 'create-album-title');
	fillField(dialog as ParentNode, '/Volumes/Photos/family', 'create-album-path');

	// The empty audience renders the honest "only you" note.
	expect(dialog?.textContent).toContain('Only you');
	buttonByText('Add', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The candidate picker: the other accounts (the caller excluded).
	// The rows live in the PORTALED floating layer (the floating-layer revision:
	// document.body, outside the dialog's DOM) -- document-level
	// selectors find them.
	const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__row') ?? []);
	expect(rows.map((row) => row.querySelector('.audience-field__row-name')?.textContent)).toEqual([
		'Grace Hopper',
		'Ivy Lee'
	]);
	rows[1]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();

	// The chip landed (the picker stays open); the
	// caller is "only me" no more.
	expect(dialog?.querySelector('.audience-field__chip')?.textContent).toContain('Ivy Lee');
	expect(dialog?.textContent).not.toContain('Only you');

	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums' && init?.method === 'POST'
	);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		title: 'Family archive',
		rootPath: '/Volumes/Photos/family',
		audience: ['u3'],
		inTimeline: true
	});
	unmount(page);
	flushSync();
});

test('the create dialog surfaces the honest skipped report before closing', async () => {
	// Signed in as an administrator whose id holds no row in the
	// instance's member list: the picker offers every account, and
	// the server's skip report is the backstop -- the write lands,
	// the report names what did not (the picked account turned
	// unusable server-side between the listing and the write).
	await bootWallSession('u0', 'admin');
	const created = {
		...summary('al-new', { title: 'Family archive', type: 'directory' }),
		skipped: [{ userId: 'u1', reason: 'invalid_account' }],
		sourceWritable: true
	};
	const fetchMock = stubFetch(
		wallRoutes({
			'/api/v1/albums': (init?: RequestInit) => {
				if (init?.method === 'POST') {
					gainOwnCard(created);
					return jsonResponse(201, created);
				}
				return jsonResponse(200, { albums: wallState });
			},
			'/api/v1/users': () => jsonResponse(200, usersBody)
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	fillField(dialog as ParentNode, 'Family archive', 'create-album-title');
	fillField(dialog as ParentNode, '/Volumes/Photos/family', 'create-album-path');
	buttonByText('Add', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	// The rows live in the portaled floating layer (document-level).
	const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__row') ?? []);
	rows[0]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The write landed (the refetch converged the wall behind), and
	// the dialog holds the write's own honest report with the Done
	// affordance -- never a silent close. The toast belongs to the
	// CLOSING path alone: the report face never fires it.
	expect(cards()[1]?.querySelector('.album-card__title')?.textContent).toBe('Family archive');
	expect(modal()?.textContent).toContain('1 requested member was not added');
	expect(toasts()).toHaveLength(0);
	buttonByText('Done', modal() as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('a read-only source lands as a created album with the honest feedback, never a failure', async () => {
	// The write probe's false arm: the creation SUCCEEDED (a
	// read-only directory is a legitimate album), and the read-only
	// fact rides the house toast -- both sentences, one
	// acknowledgment -- while the dialog closes on the landed write
	// (the sample's create button is the design; the held report
	// face was the implementation's divergence). The skipped list
	// alone still holds the report face: a list cannot ride a toast.
	await bootWallSession('u1', 'admin');
	const created = {
		...summary('al-new', { title: 'Vault archive', type: 'directory' }),
		skipped: [],
		sourceWritable: false
	};
	const fetchMock = stubFetch(
		wallRoutes({
			'/api/v1/albums': (init?: RequestInit) => {
				if (init?.method === 'POST') {
					gainOwnCard(created);
					return jsonResponse(201, created);
				}
				return jsonResponse(200, { albums: wallState });
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	fillField(dialog as ParentNode, 'Vault archive', 'create-album-title');
	fillField(dialog as ParentNode, '/mnt/readonly/archive', 'create-album-path');
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The write landed and the wall converged; the read-only fact
	// rides the house toast -- both sentences, one acknowledgment --
	// and the dialog closes on the landed write.
	expect(cards()[1]?.querySelector('.album-card__title')?.textContent).toBe('Vault archive');
	expect(modal()).toBeNull();
	expect(toasts()).toHaveLength(1);
	expect(toasts()[0]?.text).toBe(
		'The album was created. This folder is read-only — its photos cannot be deleted through the app.'
	);
	expect(toasts()[0]?.tone).toBe('success');
	unmount(page);
	flushSync();
});

test('the create dialog routes a path-matrix failure under the path field', async () => {
	await bootWallSession('u1', 'admin');
	const fetchMock = stubFetch(
		wallRoutes({
			'/api/v1/albums': (init?: RequestInit) => {
				if (init?.method === 'POST') {
					return jsonResponse(400, {
						code: 'ALBUM_PATH_NOT_FOUND',
						message: 'the source path does not exist'
					});
				}
				return jsonResponse(200, { albums: wallState });
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	fillField(dialog as ParentNode, 'Weekend in the hills', 'create-album-title');
	fillField(dialog as ParentNode, '/Volume/Photos/hills', 'create-album-path');
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The failure state: the failure line sits under the path
	// field and the input renders invalid -- the message points back
	// at the completion, never a generic box.
	const pathInput = dialog?.querySelector<HTMLInputElement>('#create-album-path');
	expect(pathInput?.getAttribute('aria-invalid')).toBe('true');
	const failure = dialog?.querySelector('.path-completion__failure');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('That folder does not exist.');
	expect(modal()).not.toBeNull();
	unmount(page);
	flushSync();
});

test('an invalid create draft never leaves the page', async () => {
	await bootWallSession('u1', 'admin');
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	// The title's native ceiling is the contract's 64 (the input stops
	// the typing there; the local rune check owns the submit).
	const titleInput = dialog?.querySelector<HTMLInputElement>('#create-album-title');
	expect(titleInput?.required).toBe(true);
	expect(titleInput?.getAttribute('maxlength')).toBe('64');

	// Whitespace title: the local 1-64 rune bound refuses it.
	fillField(dialog as ParentNode, '   ', 'create-album-title');
	fillField(dialog as ParentNode, '/Volumes/Photos/x', 'create-album-path');
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums' && init?.method === 'POST'
	);
	expect(posts).toHaveLength(0);
	expect(dialog?.textContent).toContain('Titles span 1 to 64 characters.');

	// A title past the bound (65 runes, the length a programmatic or
	// pasted draft could still carry): the same local check refuses it.
	fillField(dialog as ParentNode, `${'a'.repeat(64)}b`, 'create-album-title');
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(posts).toHaveLength(0);
	expect(dialog?.textContent).toContain('Titles span 1 to 64 characters.');

	// A relative path: the local absolute-path mirror refuses it.
	fillField(dialog as ParentNode, 'A title', 'create-album-title');
	fillField(dialog as ParentNode, 'photos/family', 'create-album-path');
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(posts).toHaveLength(0);
	expect(dialog?.querySelector('.path-completion__failure')?.textContent).toContain('Pick the source folder');
	unmount(page);
	flushSync();
});

test('SSE album.created converges: the refetched wall gains the card', async () => {
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(5);

	gainOwnCard(summary('al9', { title: 'Fresh event album' }));
	fireAlbumEvent({ id: 2, name: 'album.created', albumId: 'al9', userId: 'u1' });
	await settle();

	// The pinned card still leads; the fresh card heads the curated
	// own group behind it.
	expect(cards()[0]?.querySelector('.album-card__title')?.textContent).toBe('My photos');
	expect(cards()[1]?.querySelector('.album-card__title')?.textContent).toBe('Fresh event album');
	unmount(page);
	flushSync();
});

test('SSE album.deleted converges: the refetched wall drops the card', async () => {
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(5);

	wallState = wallState.filter((album) => album.id !== 'al1');
	fireAlbumEvent({ id: 3, name: 'album.deleted', albumId: 'al1', userId: 'u1' });
	await settle();

	expect(cards().map((card) => card.getAttribute('href'))).toEqual([
		'/albums/sync1',
		'/albums/al2',
		'/albums/al3',
		'/albums/al4'
	]);
	unmount(page);
	flushSync();
});

test('a foreign write that moves nothing the server does not move changes nothing', async () => {
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	const initial = fetchMock.mock.calls.length;

	// Another account's event about an album this wall never sees: the
	// refetch happens, the server's answer keeps the wall as it is.
	fireAlbumEvent({ id: 4, name: 'album.items-changed', albumId: 'al-elsewhere', userId: 'u2' });
	await settle();

	expect(cards()).toHaveLength(5);
	expect(fetchMock.mock.calls.length).toBe(initial + 1);
	unmount(page);
	flushSync();
});

test('a burst of album events collapses into one trailing refetch', async () => {
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	const initial = fetchMock.mock.calls.length;

	fireAlbumEvent({ id: 5, name: 'album.updated', albumId: 'al1', userId: 'u1' });
	fireAlbumEvent({ id: 6, name: 'album.shared', albumId: 'al1', userId: 'grantee' });
	fireAlbumEvent({ id: 7, name: 'album.unshared', albumId: 'al4', userId: 'grantee' });
	await settle();

	// The in-flight guard: the wall's fresh answer plus ONE trailing
	// follow-up for the burst's tail, never three.
	expect(fetchMock.mock.calls.length - initial).toBe(2);
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from the event stream', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();
	expect(handlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(handlers).toHaveLength(0);
});
