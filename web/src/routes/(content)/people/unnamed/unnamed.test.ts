import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import type { UnnamedClusterSummary } from '$lib/api/people';
import Page from './+page.svelte';

// The unnamed region of the people surface:
// the one-row bar (back + the title with the REGION's count, no
// action at all), the cards as ENTRANCES -- anchors into the person
// detail's unnamed variant, no check circles, no select mode, no
// batch bar -- the empty and error states, the keyset pagination and
// the SSE convergence (a person write or a clustering run replays the
// pages). The events module is mocked with a captured handler
// registry, the timeline page's pattern.

const { personHandlers, faceHandlers } = vi.hoisted(() => ({
	personHandlers: [] as Array<(event: unknown) => void>,
	faceHandlers: [] as Array<(event: unknown) => void>
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
	subscribeFaceEvents: (listener: (event: unknown) => void) => {
		faceHandlers.push(listener);
		return () => {
			const index = faceHandlers.indexOf(listener);
			if (index !== -1) {
				faceHandlers.splice(index, 1);
			}
		};
	}
}));

function cluster(id: string, faceCount: number): UnnamedClusterSummary {
	const face = `f-${id}`;
	return {
		clusterId: id,
		faceCount,
		representativeFaceId: face,
		thumbnail: `/api/v1/faces/${face}/thumbnail`
	};
}

// The server's current answer; the convergence tests mutate it to
// stand for the server's post-event truth.
let regionItems: UnnamedClusterSummary[] = [cluster('u1', 5), cluster('u2', 3)];
let regionTotal = 2;

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function stubFetch(): ReturnType<typeof vi.fn> {
	return vi.fn(async () => jsonResponse(200, { items: regionItems, nextCursor: null, totalCount: regionTotal }));
}

async function settle(): Promise<void> {
	// The load chain (fetch -> response.json -> the adoption) spans
	// several task turns, and the adoption writes land in Svelte's
	// scheduler queue: a few timer rounds plus one forced flush make
	// the DOM agree with the settled state.
	for (let round = 0; round < 4; round += 1) {
		await new Promise((resolve) => setTimeout(resolve, 0));
	}
	flushSync();
}

function mountPage(): ReturnType<typeof mount> {
	const page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
}

function cards(): HTMLAnchorElement[] {
	return Array.from(document.body.querySelectorAll<HTMLAnchorElement>('.cluster-card'));
}

/** Fire one parsed person event to every live subscriber. */
function firePersonEvent(event: Record<string, unknown>): void {
	for (const handler of [...personHandlers]) {
		handler(event);
	}
}

/** Fire one parsed clustering event to every live subscriber. */
function fireFaceEvent(event: Record<string, unknown>): void {
	for (const handler of [...faceHandlers]) {
		handler(event);
	}
}

beforeEach(async () => {
	document.body.textContent = '';
	personHandlers.length = 0;
	faceHandlers.length = 0;
	regionItems = [cluster('u1', 5), cluster('u2', 3)];
	regionTotal = 2;
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

test('the bar carries the back control and the title with the region count, and no action', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	const back = document.body.querySelector<HTMLAnchorElement>('.unnamed__back');
	expect(back?.getAttribute('href')).toBe('/people');
	expect(back?.getAttribute('aria-label')).toBe('Back to people');
	expect(document.body.querySelector('.unnamed__title')?.textContent?.replace(/\s+/g, ' ').trim()).toBe(
		'Unnamed · 2 groups'
	);
	// No selection surface anywhere -- no select toggle, no
	// merge action, no batch bar.
	const barButtons = Array.from(document.body.querySelectorAll<HTMLElement>('.unnamed__toolbar button'));
	expect(barButtons).toHaveLength(0);
	expect(document.body.querySelector('.batchbar')).toBeNull();
	expect(document.body.querySelector('.mergebar')).toBeNull();
	unmount(page);
	flushSync();
});

test('the cards are entrances into the same detail page, with no selection state', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	const found = cards();
	expect(found).toHaveLength(2);
	// An anchor, not a button: the card opens the person detail's
	// unnamed variant.
	expect(found[0]?.tagName.toLowerCase()).toBe('a');
	expect(found[0]?.getAttribute('href')).toBe('/people/u1?state=unnamed');
	expect(found[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f-u1/thumbnail');
	// The card's name slot reads the localized unnamed title and the
	// count is the region's own "N faces" (the tiles' honest count).
	expect(found[0]?.querySelector('.cluster-card__name')?.textContent?.trim()).toBe('Unnamed');
	expect(found[0]?.querySelector('.cluster-card__count')?.getAttribute('aria-label')).toBe('5 faces');
	// No check circle stands (the retired selection surface).
	expect(document.body.querySelector('.cluster-card__check')).toBeNull();
	unmount(page);
	flushSync();
});

test('the empty region carries the honest copy under the bar', async () => {
	regionItems = [];
	regionTotal = 0;
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	const empty = document.body.querySelector('.unnamed__empty');
	expect(empty?.textContent).toContain('Nothing to name yet');
	expect(empty?.textContent).toContain('Face clusters appear here once faces are grouped.');
	expect(cards()).toHaveLength(0);
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

	const failure = document.body.querySelector('.unnamed__failure');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('Something went wrong.');
	expect(document.body.querySelector('.unnamed__failure button')).not.toBeNull();
	unmount(page);
	flushSync();
});

test('SSE person.* events replay the region: the server answer decides', async () => {
	const fetchMock = stubFetch();
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(2);

	// The server named one cluster; the region lost it.
	regionItems = [cluster('u2', 3)];
	regionTotal = 1;
	firePersonEvent({ id: 1, name: 'person.labeled', clusterId: 'u1', userId: 'u1' });
	await settle();

	expect(cards()).toHaveLength(1);
	expect(document.body.querySelector('.unnamed__title')?.textContent).toContain('1 group');
	unmount(page);
	flushSync();
});

test('SSE face.clustered replays the region (a recognition run announced new clusters)', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(2);

	regionItems = [cluster('u1', 5), cluster('u2', 3), cluster('u3', 1)];
	regionTotal = 3;
	fireFaceEvent({ id: 2, name: 'face.clustered', totalClusters: 3, labeledClusters: 0 });
	await settle();

	expect(cards()).toHaveLength(3);
	expect(document.body.querySelector('.unnamed__title')?.textContent).toContain('3 groups');
	unmount(page);
	flushSync();
});

test('the region pages with the keyset cursor and appends without duplicates', async () => {
	// Two pages: the first carries a cursor, the second ends the
	// region.
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url.includes('cursor=')) {
				return jsonResponse(200, { items: [cluster('u2', 3)], nextCursor: null, totalCount: 2 });
			}
			return jsonResponse(200, { items: [cluster('u1', 5)], nextCursor: 'abc', totalCount: 2 });
		})
	);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(1);

	// The manual pager (no IntersectionObserver in jsdom) walks the
	// next page.
	const pager = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.unnamed__pager button'));
	expect(pager).toHaveLength(1);
	pager[0]!.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(cards()).toHaveLength(2);
	expect(cards()[1]?.getAttribute('href')).toBe('/people/u2?state=unnamed');
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from both streams', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(personHandlers).toHaveLength(1);
	expect(faceHandlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(personHandlers).toHaveLength(0);
	expect(faceHandlers).toHaveLength(0);
});
