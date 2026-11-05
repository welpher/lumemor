import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import type { AlbumSummary } from '$lib/api/albums';
import Page from './+page.svelte';

// The album wall (web IA §2-§3): the cover-card grid in its two
// sections -- the caller's own albums, then the albums shared to
// them -- with the collage covers (one full-bleed, two split, three
// 2+1, four quad), the locked / shared / excluded badges, the counts
// and the honest owner attribution on the shared cards. The "New
// album" dialog creates one virtual album; every album.* SSE event
// converges through ONE fresh wall fetch (the collapsed single-flight
// of the people page), and the refetched SERVER answer decides what
// changed. The events module is mocked with a captured handler
// registry, the people page's pattern.

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
		description: '',
		type: 'virtual',
		locked: false,
		isShared: false,
		coverAssetIds: [],
		itemCount: 0,
		updatedAt: '2026-11-01T10:00:00Z',
		callerRole: 'owner',
		...overrides
	};
}

const roadTrip = summary('al1', {
	title: 'Road trip 2026',
	description: 'The coastal week.',
	coverAssetIds: ['a1', 'a2', 'a3', 'a4'],
	itemCount: 128,
	isShared: true
});
const familyArchive = summary('al2', {
	title: 'Family archive',
	type: 'directory',
	locked: true,
	itemCount: 2048,
	inTimeline: false
});
const quickPicks = summary('al3', {
	title: 'Quick picks',
	coverAssetIds: ['a9'],
	itemCount: 1
});
const graceHighlights = summary('al4', {
	title: "Grace's highlights",
	description: 'Shared with the family.',
	coverAssetIds: ['g1', 'g2'],
	itemCount: 17,
	callerRole: 'viewer',
	owner: { id: 'u2', email: 'grace@example.com' }
});

// The server's current answer; the convergence tests mutate it to
// stand for the server's post-event truth.
let wallState: AlbumSummary[] = [roadTrip, familyArchive, quickPicks, graceHighlights];

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
				const created = summary('al-new', { title: body.title, itemCount: 0 });
				wallState = [created, ...wallState];
				return jsonResponse(201, created);
			}
			return jsonResponse(200, { albums: wallState });
		},
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

function fillField(scope: ParentNode, value: string): void {
	const field = scope.querySelector<HTMLInputElement>('input');
	if (field === null) {
		throw new Error('missing dialog input');
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

function modal(): HTMLElement | null {
	return document.body.querySelector<HTMLElement>('.modal__panel');
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
	wallState = [roadTrip, familyArchive, quickPicks, graceHighlights];
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

test('the wall sections the albums off the callerRole: own first, shared after', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	const ownSection = document.body.querySelector('.albums__own-section');
	const sharedSection = document.body.querySelector('.albums__shared-section');
	expect(ownSection?.querySelector('.albums__section-title')?.textContent).toBe('Your albums');
	expect(sharedSection?.querySelector('.albums__section-title')?.textContent).toBe(
		'Shared with you'
	);

	const ownCards = Array.from(ownSection?.querySelectorAll<HTMLElement>('.album-card') ?? []);
	const sharedCards = Array.from(sharedSection?.querySelectorAll<HTMLElement>('.album-card') ?? []);
	expect(ownCards).toHaveLength(3);
	expect(sharedCards).toHaveLength(1);
	expect(sharedCards[0]?.querySelector('.album-card__title')?.textContent).toBe(
		"Grace's highlights"
	);
	// The honest attribution of another account's album.
	expect(sharedCards[0]?.querySelector('.album-card__owner')?.textContent).toBe(
		'Shared by grace@example.com'
	);
	// Cards link into the album's detail.
	expect(ownCards[0]?.getAttribute('href')).toBe('/albums/al1');
	expect(sharedCards[0]?.getAttribute('href')).toBe('/albums/al4');

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
	expect(coverImgs(0)).toEqual([
		'/api/v1/assets/a1/thumbnail?size=tile',
		'/api/v1/assets/a2/thumbnail?size=tile',
		'/api/v1/assets/a3/thumbnail?size=tile',
		'/api/v1/assets/a4/thumbnail?size=tile'
	]);
	expect(cards()[0]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--quad')).toBe(true);

	// Zero covers: the bare placeholder, no requests.
	expect(coverImgs(1)).toEqual([]);
	expect(cards()[1]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--empty')).toBe(true);

	// One cover: the full-bleed pin.
	expect(coverImgs(2)).toEqual(['/api/v1/assets/a9/thumbnail?size=tile']);
	expect(cards()[2]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--single')).toBe(true);

	// Two covers: the split pair.
	expect(coverImgs(3)).toEqual([
		'/api/v1/assets/g1/thumbnail?size=tile',
		'/api/v1/assets/g2/thumbnail?size=tile'
	]);
	expect(cards()[3]?.querySelector('.album-card__cover')?.classList.contains('album-card__cover--split')).toBe(true);

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
	expect(cells[2]?.classList.contains('album-card__cell--wide')).toBe(true);
	unmount(page);
	flushSync();
});

test('the badges ride the cards: locked, shared, excluded', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	const roadTripCard = cards()[0];
	expect(roadTripCard?.querySelector('.album-card__badge--shared')?.textContent?.trim()).toBe('Shared');
	expect(roadTripCard?.querySelector('.album-card__badge--locked')).toBeNull();
	expect(roadTripCard?.textContent).toContain('128 photos');

	const archiveCard = cards()[1];
	expect(archiveCard?.querySelector('.album-card__badge--locked')?.textContent?.trim()).toBe('Locked');
	// The exclusion badge rides the excludable cards only: the own
	// directory item here, and every shared item.
	expect(archiveCard?.querySelector('.album-card__badge--excluded')?.textContent?.trim()).toBe(
		'Not in timeline'
	);

	const sharedCard = cards()[3];
	expect(sharedCard?.querySelector('.album-card__badge--excluded')).toBeNull();
	expect(sharedCard?.textContent).toContain('17 photos');

	// The own virtual card carries no exclusion badge: it never feeds
	// the caller's own timeline.
	expect(cards()[2]?.querySelector('.album-card__badge--excluded')).toBeNull();

	unmount(page);
	flushSync();
});

test('an empty wall renders the empty state with the create affordance', async () => {
	wallState = [];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.albums__empty')?.textContent).toContain(
		'No albums yet. Create one to gather photos around a moment.'
	);
	expect(cards()).toHaveLength(0);
	expect(buttonByText('New album')).toBeTruthy();
	unmount(page);
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

	const failure = document.body.querySelector('.albums__failure');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('Something went wrong.');

	failing = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(cards()).toHaveLength(4);
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

	const failure = document.body.querySelector('.albums__failure');
	expect(failure?.textContent).toContain('Your session has ended.');
	expect(failure?.textContent).not.toContain('Incorrect email or password');
	unmount(page);
	flushSync();
});

test('the create dialog POSTs the title and the refetched wall gains the card', async () => {
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(4);

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('New album');
	fillField(dialog as ParentNode, 'Weekend in the hills');
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const posts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums' && init?.method === 'POST'
	);
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		title: 'Weekend in the hills',
		description: ''
	});
	// The refetch placed the created card at the head of the own group.
	expect(cards()[0]?.querySelector('.album-card__title')?.textContent).toBe(
		'Weekend in the hills'
	);
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('an invalid create draft never leaves the page', async () => {
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('New album').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	// Whitespace only: the local 1-64 rune bound refuses it.
	fillField(dialog as ParentNode, '   ');
	buttonByText('Create album', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const posts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/albums' && init?.method === 'POST'
	);
	expect(posts).toHaveLength(0);
	expect(dialog?.textContent).toContain('Titles span 1 to 64 characters.');
	unmount(page);
	flushSync();
});

test('SSE album.created converges: the refetched wall gains the card', async () => {
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(4);

	wallState = [summary('al9', { title: 'Fresh event album' }), ...wallState];
	fireAlbumEvent({ id: 2, name: 'album.created', albumId: 'al9', userId: 'u1' });
	await settle();

	expect(cards()[0]?.querySelector('.album-card__title')?.textContent).toBe('Fresh event album');
	unmount(page);
	flushSync();
});

test('SSE album.deleted converges: the refetched wall drops the card', async () => {
	const fetchMock = stubFetch(wallRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(4);

	wallState = wallState.filter((album) => album.id !== 'al1');
	fireAlbumEvent({ id: 3, name: 'album.deleted', albumId: 'al1', userId: 'u1' });
	await settle();

	expect(cards().map((card) => card.getAttribute('href'))).toEqual([
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

	expect(cards()).toHaveLength(4);
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
