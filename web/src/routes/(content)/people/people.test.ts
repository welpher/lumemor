import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import type { PersonSummary } from '$lib/api/people';
import Page from './+page.svelte';

// The named region of the people surface: the one-row action bar
// (title + the person count, the name filter, the unnamed entry
// badged with the region's count), the cards (square crop, the
// caller's own name, the PHOTO count as icon + number), the empty /
// filtered-empty / error states, and the SSE convergence: every
// person.* event and every asset deletion refetches both reads and
// the SERVER's answer decides what changed (labels are private per
// user, so an event from another account's write must move nothing
// the server does not move). The events module is mocked with a
// captured handler registry, the timeline page's pattern.

const { personHandlers, assetHandlers } = vi.hoisted(() => ({
	personHandlers: [] as Array<(event: unknown) => void>,
	assetHandlers: [] as Array<(event: unknown) => void>
}));

vi.mock('$lib/events', () => ({
	subscribePersonEvents: (listener: (event: unknown) => void) => {
		personHandlers.push(listener);
		return () => {
			const index = personHandlers.indexOf(listener);
			if (index !== -1) {
				personHandlers.splice(index, 1);
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

function person(id: string, name: string, photoCount: number): PersonSummary {
	const face = `f-${id}`;
	return {
		clusterId: id,
		name,
		photoCount,
		representativeFaceId: face,
		thumbnail: `/api/v1/faces/${face}/thumbnail`
	};
}

const ada = person('c1', 'Ada', 14);
const grace = person('c2', 'Grace', 3);
const ivy = person('c3', 'Ivy', 7);

// The server's current answers; the convergence tests mutate them to
// stand for the server's post-event truth.
let peopleState: PersonSummary[] = [ada, grace, ivy];
let unnamedTotal = 12;

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function stubFetch(): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		if (url === '/api/v1/people') {
			return jsonResponse(200, peopleState);
		}
		if (url.startsWith('/api/v1/people/unnamed')) {
			return jsonResponse(200, { items: [], nextCursor: null, totalCount: unnamedTotal });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
}

async function settle(): Promise<void> {
	// The chain a load runs (fetch -> response.json -> the page's
	// adoption) spans several task turns, and the adoption writes land
	// in Svelte's scheduler queue: a few timer rounds plus one forced
	// flush make the DOM agree with the settled state.
	for (let round = 0; round < 3; round += 1) {
		await new Promise((resolve) => setTimeout(resolve, 0));
	}
	flushSync();
}

function mountPage(): ReturnType<typeof mount> {
	const page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
}

function cards(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.person-card'));
}

/** Fire one parsed person event to every live subscriber. */
function firePersonEvent(event: Record<string, unknown>): void {
	for (const handler of [...personHandlers]) {
		handler(event);
	}
}

/** Fire one parsed asset event to every live subscriber. */
function fireAssetEvent(event: Record<string, unknown>): void {
	for (const handler of [...assetHandlers]) {
		handler(event);
	}
}

beforeEach(async () => {
	document.body.textContent = '';
	personHandlers.length = 0;
	assetHandlers.length = 0;
	peopleState = [ada, grace, ivy];
	unnamedTotal = 12;
	// Boot the session store unauthenticated (the interceptor stays
	// inert; these reads answer 200s).
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

test('the bar carries the title, the count, the filter and the badged unnamed entry', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	// The one row: the title with the count, the filter on the
	// right, the unnamed entry last.
	const title = document.body.querySelector('.people__title');
	expect(title?.textContent?.replace(/\s+/g, ' ').trim()).toBe('People · 3 people');
	const filter = document.body.querySelector<HTMLInputElement>('.people__filter input');
	expect(filter?.getAttribute('placeholder')).toBe('Filter by name');
	const unnamed = document.body.querySelector<HTMLAnchorElement>('.people__unnamed-link');
	expect(unnamed?.getAttribute('href')).toBe('/people/unnamed');
	expect(unnamed?.getAttribute('aria-label')).toBe('Unnamed · 12 groups');
	expect(unnamed?.querySelector('.people__unnamed-badge')?.textContent).toBe('12');
	// The divider under the row is the bar's own bottom edge.
	expect(document.body.querySelector('.people__toolbar')?.className).toContain('border-b');
	unmount(page);
	flushSync();
});

test('the cards carry the square crop, the name and the photo count', async () => {
	const photoState: PersonSummary[] = [
		ada,
		// Two faces of one person in ONE photo against the old face
		// count: the card renders photos.
		person('c4', 'Mum', 1)
	];
	peopleState = photoState;
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	expect(cards()).toHaveLength(2);
	const first = cards()[0];
	expect(first?.getAttribute('href')).toBe('/people/c1');
	expect(first?.querySelector('.person-card__crop')?.className).toContain('aspect-square');
	expect(first?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f-c1/thumbnail');
	expect(first?.querySelector('.person-card__name')?.textContent).toBe('Ada');
	const count = first?.querySelector('.person-card__count');
	expect(count?.textContent?.trim()).toBe('14');
	expect(count?.getAttribute('aria-label')).toBe('14 photos');
	// The icon rides beside the number (the icon-first count).
	expect(count?.querySelector('svg')).not.toBeNull();
	unmount(page);
	flushSync();
});

test('the name filter narrows the grid and says so on a miss', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(3);

	const filter = document.body.querySelector<HTMLInputElement>('.people__filter input');
	filter!.value = 'gr';
	filter!.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	expect(cards()).toHaveLength(1);
	expect(cards()[0]?.textContent).toContain('Grace');
	// The bar's count is the REGION's (the list itself never shrank --
	// the filter is the page's own narrowing).
	expect(document.body.querySelector('.people__title')?.textContent).toContain('3 people');

	filter!.value = 'zz';
	filter!.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	expect(cards()).toHaveLength(0);
	expect(document.body.querySelector('.people__no-match')?.textContent?.trim()).toBe('No people match.');
	unmount(page);
	flushSync();
});

test('the empty region carries the honest copy and the unnamed entry', async () => {
	peopleState = [];
	unnamedTotal = 2;
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	const empty = document.body.querySelector('.people__empty');
	expect(empty?.textContent).toContain('No people yet');
	expect(empty?.textContent).toContain('Photos with faces will show up here.');
	expect(document.body.querySelector('.people__unnamed-badge')?.textContent).toBe('2');
	unmount(page);
	flushSync();
});

test('a failed load renders the error box with the retry affordance', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(500, { code: 'INTERNAL', message: 'boom' }))
	);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.people__failure');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('Something went wrong.');
	expect(document.body.querySelector('.people__failure button')).not.toBeNull();
	unmount(page);
	flushSync();
});

test('SSE person.* events converge: the refetched server answer is the truth', async () => {
	const fetchMock = stubFetch();
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(3);

	// The server renamed one person and another account labeled a new
	// cluster; the refetch answers the caller's own view.
	peopleState = [ada, person('c2', 'Gracie', 3)];
	firePersonEvent({ id: 2, name: 'person.labeled', clusterId: 'c2', userId: 'u1' });
	await settle();

	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people')).toHaveLength(2);
	expect(cards()).toHaveLength(2);
	expect(cards()[1]?.textContent).toContain('Gracie');
	unmount(page);
	flushSync();
});

test('SSE person.birthdate_updated and person.avatar_updated converge too (entity writes)', async () => {
	const fetchMock = stubFetch();
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	firePersonEvent({ id: 3, name: 'person.birthdate_updated', clusterId: 'c1', userId: 'u1' });
	await settle();
	firePersonEvent({ id: 4, name: 'person.avatar_updated', clusterId: 'c1', userId: 'u1' });
	await settle();

	// Two more loads: the avatar pick may move a card's crop, so the
	// named list converges on it.
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people')).toHaveLength(3);
	unmount(page);
	flushSync();
});

test('an asset deletion converges the counts (a purged face died with it)', async () => {
	const fetchMock = stubFetch();
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	peopleState = [ada, person('c2', 'Grace', 2)];
	fireAssetEvent({ id: 5, name: 'asset.deleted', assetId: 'a1' });
	await settle();
	expect(cards()[1]?.querySelector('.person-card__count')?.textContent?.trim()).toBe('2');

	// Any other asset event leaves the page alone (only the deletion
	// changes this list's membership and counts).
	const before = fetchMock.mock.calls.length;
	fireAssetEvent({ id: 6, name: 'asset.ready', assetId: 'a2' });
	await settle();
	expect(fetchMock.mock.calls.length).toBe(before);
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from both streams', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(personHandlers).toHaveLength(1);
	expect(assetHandlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(personHandlers).toHaveLength(0);
	expect(assetHandlers).toHaveLength(0);
});
