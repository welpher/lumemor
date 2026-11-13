import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import type { AlbumSummary } from '$lib/api/albums';
import Page from './+page.svelte';

// The memory wall: ONE list of reference-set cards (no own/shared
// sections, no group labels -- the design's single list) -- the
// caller's own memories (the manual origin badge on every card) mixed
// with the memories shared to them (the honest owner attribution
// beside the read-only badge, the names-first rule). The "New
// memory" creation is EVERY
// MEMBER'S (a memory is personal curation, the opposite gate of the
// album wall's administrator-only creation), the create dialog is
// the title + the audience alone (no path, no switches -- a memory
// owns no source and never feeds its creator's timeline), and the
// write's honest `skipped` report holds the dialog until
// acknowledged. No memory card ever carries the timeline-exclusion
// badge: the exclusion is the memory's fixed semantics, not a
// signal. Every album.* SSE event converges through ONE fresh wall
// fetch (the albums page's collapsed single-flight); the refetched
// SERVER answer decides what changed. The events module is mocked
// with a captured handler registry, the albums page's pattern.

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
		title: `Memory ${id}`,
		type: 'memory',
		isShared: false,
		coverAssetIds: [],
		itemCount: 0,
		updatedAt: '2026-11-01T10:00:00Z',
		callerRole: 'owner',
		...overrides
	};
}

const springFestival = summary('mem1', {
	title: 'Spring Festival 2026',
	coverAssetIds: ['a1', 'a2', 'a3', 'a4'],
	itemCount: 12
});
const paris = summary('mem2', {
	title: 'Paris trip',
	coverAssetIds: ['p1', 'p2'],
	itemCount: 28
});
// The own memory card answers no `inTimeline` field (the memory
// never feeds the creator's timeline) -- and even a drifted field
// would render no badge here.
const weekendPark = summary('mem3', {
	title: 'Weekend park',
	coverAssetIds: ['w1'],
	itemCount: 6,
	inTimeline: true
});
const sanya = summary('mem4', {
	title: 'Sanya trip',
	coverAssetIds: ['s1', 's2', 's3'],
	itemCount: 12,
	callerRole: 'viewer',
	owner: { id: 'u2', name: 'Grace Hopper', email: 'grace@example.com' }
});
// The album wall's own rows: the memory wall filters them out (the
// memories moved to their own route; the albums page owns the
// containers).
const myPhotos = summary('sync1', { title: 'Sync', type: 'sync', itemCount: 1204 });
const familyArchive = summary('al1', {
	title: 'Family archive',
	type: 'directory',
	itemCount: 2048,
	inTimeline: false
});

// The server's current answer; the convergence tests mutate it to
// stand for the server's post-event truth.
let wallState: AlbumSummary[] = [myPhotos, springFestival, paris, weekendPark, sanya, familyArchive];

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
				const created = summary('mem-new', { title: body.title, itemCount: 0 });
				gainOwnCard(created);
				return jsonResponse(201, { ...created, skipped: [] });
			}
			return jsonResponse(200, { albums: wallState });
		},
		...overrides
	};
}

/** Mirror the server's answer: the new own card heads the memory rest. */
function gainOwnCard(memory: AlbumSummary): void {
	wallState = [memory, ...wallState];
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
	return Array.from(document.body.querySelectorAll<HTMLElement>('.memory-card'));
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
async function bootSession(userId: string, role: 'admin' | 'user'): Promise<void> {
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
	wallState = [myPhotos, springFestival, paris, weekendPark, sanya, familyArchive];
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

test('the wall keeps the memory rows alone in ONE list, newest first', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	// No sections, no group labels: the single list the spec pins.
	expect(document.body.querySelector('.memories__own-section')).toBeNull();
	expect(document.body.querySelector('.memories__shared-section')).toBeNull();
	expect(document.body.querySelector('.memories__section-title')).toBeNull();

	// The album wall's rows (the sync card, the directory album) are
	// NOT this page's: the memory wall answers memory rows alone, in
	// one grid ordered by the cards' own recency.
	expect(cards()).toHaveLength(4);
	expect(cards().map((card) => card.querySelector('.memory-card__title')?.textContent)).toEqual([
		'Spring Festival 2026',
		'Paris trip',
		'Weekend park',
		'Sanya trip'
	]);

	const [own, , , shared] = cards();
	// An own card: the title, the count, the manual origin badge.
	expect(own?.textContent).toContain('12 photos');
	expect(own?.querySelector('.memory-card__badge--manual')?.textContent?.trim()).toBe('Manual');
	expect(own?.querySelector('.memory-card__badge--readonly')).toBeNull();
	// The shared card: the attribution chip names the owner, the
	// read-only badge sits beside it on the same row, no manual badge.
	expect(shared?.querySelector('.memory-card__badge--owner')?.textContent?.trim()).toBe('Shared by Grace Hopper');
	expect(shared?.querySelector('.memory-card__badges .memory-card__badge--readonly')?.textContent?.trim()).toBe(
		'Read-only'
	);
	expect(shared?.querySelector('.memory-card__badge--manual')).toBeNull();
	// The caption carries NO stray owner line: the retired
	// card-level attribution stayed deleted, the attribution renders
	// exactly once (the badge), and no raw message placeholder leaks.
	expect(shared?.querySelector('.memory-card__owner')).toBeNull();
	expect(shared?.textContent).not.toContain('{name}');
	expect(shared?.querySelectorAll('.memory-card__badge--owner')).toHaveLength(1);
	// Cards link into the memory's own detail.
	expect(own?.getAttribute('href')).toBe('/memories/mem1');
	expect(shared?.getAttribute('href')).toBe('/memories/mem4');

	unmount(page);
	flushSync();
});

test('one list orders by the cards own recency, own and shared mixed', async () => {
	wallState = [
		summary('mem-old', { title: 'Old own', updatedAt: '2026-10-01T10:00:00Z' }),
		summary('mem-shared', {
			title: 'Fresh shared',
			updatedAt: '2026-11-10T10:00:00Z',
			callerRole: 'viewer',
			owner: { id: 'u2', name: 'Grace Hopper', email: 'grace@example.com' }
		}),
		summary('mem-new', { title: 'New own', updatedAt: '2026-11-05T10:00:00Z' })
	];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	expect(cards().map((card) => card.querySelector('.memory-card__title')?.textContent)).toEqual([
		'Fresh shared',
		'New own',
		'Old own'
	]);

	unmount(page);
	flushSync();
});

test('an owner who set no display name falls back to the email', async () => {
	// The account's name is the empty string (the contract serves it as
	// stored): the chip renders the email -- a blank attribution never
	// renders.
	wallState = [
		summary('mem-nameless', {
			title: 'Nameless picks',
			callerRole: 'viewer',
			owner: { id: 'u9', name: '', email: 'nameless@example.com' }
		})
	];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.memory-card__badge--owner')?.textContent?.trim()).toBe(
		'Shared by nameless@example.com'
	);

	unmount(page);
	flushSync();
});

test('two owners under one name get the email as the chip title', async () => {
	// The name is not a key: the wall shows two accounts called "Mum",
	// so each chip carries its owner's address as the hover title.
	wallState = [
		summary('mem-mum-a', {
			title: 'Mum one',
			updatedAt: '2026-11-02T10:00:00Z',
			callerRole: 'viewer',
			owner: { id: 'u5', name: 'Mum', email: 'mum@example.com' }
		}),
		summary('mem-mum-b', {
			title: 'Mum two',
			updatedAt: '2026-11-01T10:00:00Z',
			callerRole: 'viewer',
			owner: { id: 'u6', name: 'Mum', email: 'mum2@example.com' }
		})
	];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	const chips = Array.from(document.body.querySelectorAll<HTMLElement>('.memory-card__badge--owner'));
	expect(chips.map((chip) => chip.textContent?.trim())).toEqual(['Shared by Mum', 'Shared by Mum']);
	expect(chips.map((chip) => chip.getAttribute('title'))).toEqual(['mum@example.com', 'mum2@example.com']);

	unmount(page);
	flushSync();
});

test('no memory card carries the timeline-exclusion badge, whatever the field says', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	// The own card carrying an inTimeline field still renders no
	// clock badge: the memory never feeds the creator's timeline, so
	// the field is not a signal this wall shows.
	for (const card of cards()) {
		expect(card.textContent).not.toContain('Not in timeline');
		expect(card.querySelector('[class*="badge--excluded"]')).toBeNull();
	}

	unmount(page);
	flushSync();
});

test('the create button renders for a plain member: every member curates memories', async () => {
	await bootSession('u2', 'user');
	vi.stubGlobal('fetch', stubFetch(wallRoutes({ '/api/v1/users': () => jsonResponse(200, usersBody) })));
	const page = mountPage();
	await settle();

	// The OPPOSITE gate of the album wall: a member's session renders
	// the creation affordance (the write is theirs; the server holds
	// no role gate).
	expect(buttonByText('New memory')).toBeTruthy();

	unmount(page);
	flushSync();
});

test('the empty state offers the way in and points at the wall entry', async () => {
	wallState = [myPhotos, familyArchive];
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();

	const empty = document.body.querySelector('.state-card');
	expect(empty?.querySelector('.state-card__title')?.textContent).toBe('No memories yet');
	// The empty card's title is a 16px line, never a second page
	// heading: the page's one title sits in the bar above.
	expect(empty?.querySelector('h1')).toBeNull();
	expect(empty?.textContent).toContain('Gather a moment into a memory');
	// The one action is the creation; the copy carries the wall's
	// own entry, so the card never states a second route.
	expect(empty?.querySelectorAll('button')).toHaveLength(1);
	expect(empty?.querySelector('button')?.textContent).toBe('New memory');

	unmount(page);
	flushSync();
});

test('the create dialog POSTs the title and audience; the refetched wall gains the card', async () => {
	await bootSession('u1', 'admin');
	const seen: Array<{ url: string; init?: RequestInit }> = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		seen.push({ url, init });
		if (url === '/api/v1/users') {
			return jsonResponse(200, usersBody);
		}
		if (url === '/api/v1/memories') {
			const body = JSON.parse(String(init?.body)) as { title: string; audience: string[] };
			expect(body).toEqual({ title: 'Camping 2026', audience: ['u2'] });
			const created = summary('mem-new', { title: body.title, itemCount: 0, isShared: true });
			gainOwnCard(created);
			return jsonResponse(201, { ...created, skipped: [] });
		}
		if (url === '/api/v1/albums') {
			return jsonResponse(200, { albums: wallState });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('New memory').click();
	await settle();
	const dialog = modal();
	if (dialog === null) {
		throw new Error('missing create dialog');
	}
	fillField(dialog, 'Camping 2026', 'create-memory-title');
	// The audience picker's rows live in the portaled floating layer
	// (document-level); the pick lands a chip and the picker stays
	// open -- the mousedown adds, the chip's remove mark is the only
	// button among them.
	const add = buttonByText('Add', dialog);
	add.click();
	await settle();
	const rows = Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__row'));
	rows[0]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();
	buttonByText('Create', dialog).click();
	await settle();

	// The write ran, the wall behind converged (the refetch), and the
	// all-landed creation closed the dialog directly.
	expect(seen.map((call) => call.url)).toContain('/api/v1/memories');
	expect(cards()[0]?.querySelector('.memory-card__title')?.textContent).toBe('Camping 2026');
	expect(modal()).toBeNull();

	unmount(page);
	flushSync();
});

test('the create write failure stays in the dialog with the mapped error', async () => {
	await bootSession('u1', 'admin');
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums': () => jsonResponse(200, { albums: wallState }),
			'/api/v1/users': () => jsonResponse(200, usersBody),
			'/api/v1/memories': () =>
				jsonResponse(400, { code: 'VALIDATION_FAILED', message: 'title must span 1-64 characters' })
		})
	);
	const page = mountPage();
	await settle();

	buttonByText('New memory').click();
	await settle();
	const dialog = modal();
	if (dialog === null) {
		throw new Error('missing create dialog');
	}
	fillField(dialog, 'Camping 2026', 'create-memory-title');
	buttonByText('Create', dialog).click();
	await settle();

	expect(modal()).not.toBeNull();
	expect(modal()?.textContent).toContain('Please check your entries and try again.');

	unmount(page);
	flushSync();
});

test('SSE album events converge: a created memory lands, a deleted one leaves', async () => {
	vi.stubGlobal('fetch', stubFetch(wallRoutes()));
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(4);

	// The event carries ids only; the refetched server answer owns
	// the truth.
	gainOwnCard(summary('mem-5', { title: 'New year' }));
	fireAlbumEvent({ name: 'album.created', albumId: 'mem-5', userId: 'u1' });
	await settle();
	expect(cards()).toHaveLength(5);
	expect(cards()[0]?.querySelector('.memory-card__title')?.textContent).toBe('New year');

	wallState = wallState.filter((memory) => memory.id !== 'mem-5');
	fireAlbumEvent({ name: 'album.deleted', albumId: 'mem-5' });
	await settle();
	expect(cards()).toHaveLength(4);

	// A burst (two events) converges through the single-flight's one
	// trailing fetch: the wall's answer stays the server's, and the
	// page invents no card.
	gainOwnCard(summary('mem-6', { title: 'Quiet one' }));
	fireAlbumEvent({ name: 'album.updated', albumId: 'mem1' });
	fireAlbumEvent({ name: 'album.items-changed', albumId: 'mem1' });
	await settle();
	expect(cards()).toHaveLength(5);

	unmount(page);
	flushSync();
});

test('a failed initial load raises the error box with a working retry', async () => {
	let failing = true;
	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			failing ? jsonResponse(500, { code: 'INTERNAL' }) : jsonResponse(200, { albums: wallState })
		)
	);
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.state-card--error')).not.toBeNull();
	expect(document.body.querySelector('.state-card--error')?.textContent).toContain('Something went wrong');

	failing = false;
	buttonByText('Try again').click();
	await settle();

	expect(document.body.querySelector('.state-card--error')).toBeNull();
	expect(cards()).toHaveLength(4);

	unmount(page);
	flushSync();
});
