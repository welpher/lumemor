import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import type { PersonSummary } from '$lib/api/people';
import Page from './+page.svelte';

// The named region of the people surface: the person card grid
// (crop, name, face count, the hidden badge -- hidden people STAY
// listed), the empty and error states with their affordances, the
// card links into the cluster detail, and the SSE convergence: every
// person.* event refetches the list and the SERVER's answer decides
// what changed (labels are private per user, so an event from
// another account's write must move nothing the server does not
// move; the page never invents state). The events module is mocked
// with a captured handler registry, the timeline page's pattern.

const { handlers } = vi.hoisted(() => ({
	handlers: [] as Array<(event: unknown) => void>
}));

vi.mock('$lib/events', () => ({
	subscribePersonEvents: (listener: (event: unknown) => void) => {
		handlers.push(listener);
		return () => {
			const index = handlers.indexOf(listener);
			if (index !== -1) {
				handlers.splice(index, 1);
			}
		};
	}
}));

function person(id: string, name: string, faceCount: number, isHidden = false): PersonSummary {
	const face = `f-${id}`;
	return {
		clusterId: id,
		name,
		faceCount,
		representativeFaceId: face,
		thumbnail: `/api/v1/faces/${face}/thumbnail`,
		isHidden
	};
}

const ada = person('c1', 'Ada', 12);
const grace = person('c2', 'Grace', 3, true);
const ivy = person('c3', 'Ivy', 7);

// The server's current answer; the convergence tests mutate it to
// stand for the server's post-event truth.
let peopleState: PersonSummary[] = [ada, grace, ivy];

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function stubFetch(): ReturnType<typeof vi.fn> {
	return vi.fn(async () => jsonResponse(200, peopleState));
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
	return Array.from(document.body.querySelectorAll<HTMLElement>('.person-card'));
}

function buttonByText(text: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

/** Fire one parsed person event to every live subscriber. */
function firePersonEvent(event: Record<string, unknown>): void {
	for (const handler of [...handlers]) {
		handler(event);
	}
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	vi.clearAllMocks();
	peopleState = [ada, grace, ivy];
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

test('the grid renders the person cards: names, counts, links, one hidden badge', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	expect(cards()).toHaveLength(3);
	const first = cards()[0];
	expect(first?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f-c1/thumbnail');
	expect(first?.textContent).toContain('Ada');
	expect(first?.textContent).toContain('12 faces');
	// The card is the link into the cluster's detail page.
	expect(first?.getAttribute('href')).toBe('/people/c1');

	// Hidden people stay listed, visibly badged; the others are not.
	expect(cards()[1]?.classList.contains('person-card--hidden')).toBe(true);
	expect(cards()[1]?.querySelector('.person-card__badge')?.textContent).toBe('Hidden');
	expect(cards()[0]?.classList.contains('person-card--hidden')).toBe(false);
	expect(cards()[2]?.classList.contains('person-card--hidden')).toBe(false);

	unmount(page);
	flushSync();
});

test('an empty list renders the empty state, no cards', async () => {
	peopleState = [];
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.people__empty')?.textContent).toContain(
		'No people yet. Photos with faces will show up here.'
	);
	expect(cards()).toHaveLength(0);
	unmount(page);
	flushSync();
});

test('a failed load shows the error box with a working retry', async () => {
	let failing = true;
	const fetchMock = vi.fn(async () => {
		if (failing) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return jsonResponse(200, peopleState);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.people__failure');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('Something went wrong.');
	expect(failure?.textContent).toContain('Try again');

	failing = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(cards()).toHaveLength(3);
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

	const failure = document.body.querySelector('.people__failure');
	expect(failure?.textContent).toContain('Your session has ended.');
	expect(failure?.textContent).not.toContain('Incorrect email or password');
	unmount(page);
	flushSync();
});

test('SSE person.labeled converges: a newly labeled person appears through the refetch', async () => {
	peopleState = [ada];
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(1);

	peopleState = [ada, grace];
	firePersonEvent({ id: 2, name: 'person.labeled', clusterId: 'c2', label: 'Grace', userId: 'u1' });
	await settle();

	expect(cards()).toHaveLength(2);
	expect(cards()[1]?.textContent).toContain('Grace');
	unmount(page);
	flushSync();
});

test('SSE person.unlabeled converges: the card leaves the named region', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(3);

	peopleState = [ada, ivy];
	firePersonEvent({ id: 2, name: 'person.unlabeled', clusterId: 'c2', userId: 'u1' });
	await settle();

	expect(cards().map((card) => card.getAttribute('href'))).toEqual(['/people/c1', '/people/c3']);
	unmount(page);
	flushSync();
});

test('SSE person.merged converges: the sources vanish and the survivor carries the merged count', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();

	peopleState = [person('c1', 'Ada', 22)];
	firePersonEvent({
		id: 2,
		name: 'person.merged',
		survivingClusterId: 'c1',
		mergedClusterIds: ['c2', 'c3'],
		userId: 'u1'
	});
	await settle();

	expect(cards()).toHaveLength(1);
	expect(cards()[0]?.textContent).toContain('22 faces');
	unmount(page);
	flushSync();
});

test('SSE person.hidden converges: the badge follows the refetched switch', async () => {
	peopleState = [ada];
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(cards()[0]?.classList.contains('person-card--hidden')).toBe(false);

	peopleState = [person('c1', 'Ada', 12, true)];
	firePersonEvent({ id: 2, name: 'person.hidden', clusterId: 'c1', isHidden: true, userId: 'u1' });
	await settle();

	expect(cards()[0]?.classList.contains('person-card--hidden')).toBe(true);
	expect(cards()[0]?.querySelector('.person-card__badge')?.textContent).toBe('Hidden');
	unmount(page);
	flushSync();
});

test('a foreign label event moves nothing the server does not move (server truth only)', async () => {
	peopleState = [ada];
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(1);

	// Another account labeled a cluster of the global space; the
	// refetched answer (the caller's own named region) still does not
	// list it, so the page must not invent a card for it.
	firePersonEvent({ id: 2, name: 'person.labeled', clusterId: 'c9', label: 'Grace', userId: 'someone-else' });
	await settle();

	expect(cards()).toHaveLength(1);
	expect(cards()[0]?.textContent).toContain('Ada');
	unmount(page);
	flushSync();
});

test('a burst of events collapses into one trailing refetch', async () => {
	const fetchMock = stubFetch();
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	const initial = fetchMock.mock.calls.length;

	// Three events arrive in one breath while the first convergence
	// fetch is still in flight: the trailing two collapse into ONE
	// follow-up (the in-flight guard), never three.
	firePersonEvent({ id: 2, name: 'person.labeled', clusterId: 'c2', label: 'Grace', userId: 'u1' });
	firePersonEvent({ id: 3, name: 'person.unlabeled', clusterId: 'c3', userId: 'u1' });
	firePersonEvent({ id: 4, name: 'person.hidden', clusterId: 'c1', isHidden: true, userId: 'u1' });
	await settle();

	expect(fetchMock.mock.calls.length - initial).toBe(2);
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from the event stream', async () => {
	vi.stubGlobal('fetch', stubFetch());
	const page = mountPage();
	await settle();
	expect(handlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(handlers).toHaveLength(0);
});
