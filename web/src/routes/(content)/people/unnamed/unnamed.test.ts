import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import type { PersonSummary, UnnamedClusterSummary } from '$lib/api/people';
import Page from './+page.svelte';

// The unnamed region of the people surface: the keyset-paginated
// cluster grid with the manual pager (jsdom has no observer), the
// naming main path (card click -> dialog -> PUT label -> the card
// leaves the region), the batch organize path (multi-select -> merge
// dialog -> POST merge, with the inline naming of a selected target),
// and the SSE convergence: face.clustered gains a cluster and
// person.merged vanishes the dissolved sources, both through the
// replay refetch that preserves the loaded window. The events module
// is mocked with captured handler registries, the timeline page's
// pattern.

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

const c1 = cluster('c1', 8);
const c2 = cluster('c2', 5);
const c3 = cluster('c3', 2);
const c4 = cluster('c4', 2);

// The server's current answers; the convergence tests mutate them to
// stand for the server's post-event truth.
let pageOne: UnnamedClusterSummary[] = [c1, c2, c3];
let pageTwo: UnnamedClusterSummary[] = [c4];
// The named people the merge dialog offers as existing targets.
let peopleState: PersonSummary[] = [
	{
		clusterId: 'p1',
		name: 'Grace',
		faceCount: 9,
		representativeFaceId: 'f-p1',
		thumbnail: '/api/v1/faces/f-p1/thumbnail',
		isHidden: false
	}
];

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

function unnamedRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		// The follow-up cursor exists exactly while a second page
		// holds items -- the honest page shape the convergence tests
		// mutate alongside pageOne.
		'/api/v1/people/unnamed?limit=60': () =>
			jsonResponse(
				200,
				pageTwo.length > 0 ? { items: pageOne, nextCursor: 'C1' } : { items: pageOne }
			),
		'/api/v1/people/unnamed?limit=60&cursor=C1': () => jsonResponse(200, { items: pageTwo }),
		'/api/v1/people': () => jsonResponse(200, peopleState),
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
	return Array.from(document.body.querySelectorAll<HTMLElement>('.cluster-card'));
}

function buttonByText(text: string, scope: ParentNode = document.body): HTMLButtonElement {
	const buttons = Array.from(scope.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

function modal(): HTMLElement | null {
	return document.body.querySelector<HTMLElement>('.modal__panel');
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

/** Fire one parsed person event to every live subscriber. */
function firePersonEvent(event: Record<string, unknown>): void {
	for (const handler of [...personHandlers]) {
		handler(event);
	}
}

function fireFaceEvent(event: Record<string, unknown>): void {
	for (const handler of [...faceHandlers]) {
		handler(event);
	}
}

function clickCard(index: number): void {
	cards()[index]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

beforeEach(async () => {
	document.body.textContent = '';
	personHandlers.length = 0;
	faceHandlers.length = 0;
	vi.clearAllMocks();
	pageOne = [c1, c2, c3];
	pageTwo = [c4];
	peopleState = [
		{
			clusterId: 'p1',
			name: 'Grace',
			faceCount: 9,
			representativeFaceId: 'f-p1',
			thumbnail: '/api/v1/faces/f-p1/thumbnail',
			isHidden: false
		}
	];
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

test('the first page renders the cluster cards with their face counts', async () => {
	vi.stubGlobal('fetch', stubFetch(unnamedRoutes()));
	const page = mountPage();
	await settle();

	expect(cards()).toHaveLength(3);
	expect(cards()[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f-c1/thumbnail');
	expect(cards()[0]?.textContent).toContain('8 faces');
	expect(cards()[1]?.textContent).toContain('5 faces');
	expect(cards()[2]?.textContent).toContain('2 faces');

	// The manual pager covers the follow-up page (no observer here).
	expect(buttonByText('Load more')).toBeTruthy();
	unmount(page);
	flushSync();
});

test('the manual pager loads the next page and the last page retires it', async () => {
	vi.stubGlobal('fetch', stubFetch(unnamedRoutes()));
	const page = mountPage();
	await settle();

	buttonByText('Load more').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(cards()).toHaveLength(4);
	expect(cards()[3]?.textContent).toContain('2 faces');
	expect(() => buttonByText('Load more')).toThrow();
	unmount(page);
	flushSync();
});

test('an empty region renders the empty state, no cards, no pager', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/people/unnamed?limit=60': () => jsonResponse(200, { items: [] }) })
	);
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.unnamed__empty')?.textContent).toContain(
		'Nothing to name yet. Face clusters appear here once faces are grouped.'
	);
	expect(cards()).toHaveLength(0);
	expect(() => buttonByText('Load more')).toThrow();
	unmount(page);
	flushSync();
});

test('naming a cluster PUTs the label and the card leaves the unnamed region', async () => {
	const fetchMock = stubFetch(
		unnamedRoutes({
			'/api/v1/people/c1/label': () => jsonResponse(200, { clusterId: 'c1', name: 'Ada', isHidden: false })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	clickCard(0);
	const dialog = modal();
	expect(dialog?.getAttribute('role')).toBe('dialog');
	expect(dialog?.getAttribute('aria-label')).toBe('Name this person');

	fillField(dialog as ParentNode, 'Ada');
	buttonByText('Save name', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/label');
	expect(puts).toHaveLength(1);
	expect(JSON.parse(String(puts[0]?.[1]?.body))).toEqual({ name: 'Ada' });
	// The labeled cluster moved to the named region: the card is gone.
	expect(cards().map((card) => card.querySelector('img')?.getAttribute('src'))).toEqual([
		'/api/v1/faces/f-c2/thumbnail',
		'/api/v1/faces/f-c3/thumbnail'
	]);
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('a failed naming keeps the card and surfaces the message in the dialog', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			unnamedRoutes({
				'/api/v1/people/c1/label': () => jsonResponse(500, { code: 'INTERNAL' })
			})
		)
	);
	const page = mountPage();
	await settle();

	clickCard(0);
	const dialog = modal();
	fillField(dialog as ParentNode, 'Ada');
	buttonByText('Save name', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	expect(dialog?.textContent).toContain('Something went wrong. Please try again.');
	expect(cards()).toHaveLength(3);
	expect(modal()).not.toBeNull();
	unmount(page);
	flushSync();
});

test('an invalid name never leaves the page: the bound is checked locally', async () => {
	const fetchMock = stubFetch(unnamedRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	clickCard(0);
	const dialog = modal();
	// Whitespace only: the trimmed name is empty.
	fillField(dialog as ParentNode, '   ');
	buttonByText('Save name', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	expect(dialog?.textContent).toContain('Names span 1 to 64 characters.');
	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/label');
	expect(puts).toHaveLength(0);
	unmount(page);
	flushSync();
});

test('the batch merge folds two selected clusters into an existing person', async () => {
	const fetchMock = stubFetch(
		unnamedRoutes({
			'/api/v1/people/merge': () =>
				jsonResponse(200, { survivingClusterId: 'p1', mergedClusterIds: ['c1', 'c2'] })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.mergebar')).toBeNull();
	buttonByText('Select').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	clickCard(0);
	clickCard(1);

	const bar = document.body.querySelector('.mergebar');
	expect(bar?.textContent).toContain('2 selected');
	buttonByText('Merge').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Merge people');
	// The default target is the first existing person (two selected
	// clusters cannot host the merge -- that needs a third).
	expect(dialog?.textContent).toContain('Merged clusters disappear and their names are lost.');
	buttonByText('Merge', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/merge');
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		sourceClusterIds: ['c1', 'c2'],
		targetClusterId: 'p1'
	});
	// The dissolved sources vanished; the selection is gone with them.
	expect(cards().map((card) => card.querySelector('img')?.getAttribute('src'))).toEqual([
		'/api/v1/faces/f-c3/thumbnail'
	]);
	expect(document.body.querySelector('.mergebar')).toBeNull();
	unmount(page);
	flushSync();
});

test('merging into one of the selected names it inline first, then folds the others', async () => {
	const fetchMock = stubFetch(
		unnamedRoutes({
			'/api/v1/people/c2/label': () => jsonResponse(200, { clusterId: 'c2', name: 'Ada', isHidden: false }),
			'/api/v1/people/merge': () =>
				jsonResponse(200, { survivingClusterId: 'c2', mergedClusterIds: ['c1', 'c3'] })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Select').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	clickCard(0);
	clickCard(1);
	clickCard(2);
	buttonByText('Merge').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const dialog = modal();
	// Three selected: the selected clusters themselves become targets.
	chooseOption(dialog as ParentNode, 'c2');
	fillField(dialog as ParentNode, 'Ada');
	buttonByText('Merge', dialog as ParentNode).dispatchEvent(
		new MouseEvent('click', { bubbles: true })
	);
	await settle();

	// The inline naming PUT ran before the merge.
	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c2/label');
	expect(puts).toHaveLength(1);
	expect(JSON.parse(String(puts[0]?.[1]?.body))).toEqual({ name: 'Ada' });
	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/merge');
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		sourceClusterIds: ['c1', 'c3'],
		targetClusterId: 'c2'
	});
	// The sources dissolved and the named target left the unnamed
	// region: every selected card is gone.
	expect(cards()).toHaveLength(0);
	expect(document.body.querySelector('.mergebar')).toBeNull();
	unmount(page);
	flushSync();
});

test('a foreign label event moves nothing the server does not move (B4: server truth only)', async () => {
	vi.stubGlobal('fetch', stubFetch(unnamedRoutes()));
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(3);

	// Another account labeled one of the global clusters; this
	// caller's unnamed region (the caller's own unlabeled view)
	// still lists it, and the page invents no departure.
	firePersonEvent({ id: 2, name: 'person.labeled', clusterId: 'c1', label: 'Ada', userId: 'someone-else' });
	await settle();

	expect(cards()).toHaveLength(3);
	unmount(page);
	flushSync();
});

test('SSE face.clustered converges: a freshly grouped cluster appears in the region', async () => {
	vi.stubGlobal('fetch', stubFetch(unnamedRoutes()));
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(3);

	pageOne = [c1, c2, c3, cluster('c9', 1)];
	fireFaceEvent({ id: 1, name: 'face.clustered', totalClusters: 4, labeledClusters: 0 });
	await settle();

	expect(cards()).toHaveLength(4);
	expect(cards()[3]?.textContent).toContain('1 faces');
	unmount(page);
	flushSync();
});

test('SSE person.merged converges: the dissolved sources vanish from the region', async () => {
	vi.stubGlobal('fetch', stubFetch(unnamedRoutes()));
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(3);

	// The post-merge truth: the survivor carries the merged count and
	// the region holds nothing else (no follow-up page).
	pageOne = [cluster('c1', 15)];
	pageTwo = [];
	firePersonEvent({
		id: 2,
		name: 'person.merged',
		survivingClusterId: 'c1',
		mergedClusterIds: ['c2', 'c3'],
		userId: 'u1'
	});
	await settle();

	expect(cards()).toHaveLength(1);
	expect(cards()[0]?.textContent).toContain('15 faces');
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from both event kinds', async () => {
	vi.stubGlobal('fetch', stubFetch(unnamedRoutes()));
	const page = mountPage();
	await settle();
	expect(personHandlers).toHaveLength(1);
	expect(faceHandlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(personHandlers).toHaveLength(0);
	expect(faceHandlers).toHaveLength(0);
});
