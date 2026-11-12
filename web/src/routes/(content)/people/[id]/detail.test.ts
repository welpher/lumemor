import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearToasts, toasts } from '$lib/components/toast.svelte';
import type { ClusterDetail } from '$lib/api/people';
import Page from './+page.svelte';

// The cluster detail of the people surface: the label state (named
// header vs the unnamed header with its naming CTA), the face grid
// with the asset context and the links into the asset viewer, the
// rename (PUT label, the response adopted), the hidden toggle (PUT
// hidden; an unlabeled cluster surfaces the honest
// person_hide_requires_label message and keeps the naming nudge),
// merge-into (POST merge, then away to /people), the single-face move
// (POST /faces/{id}/move, then the refetched face list), the 404
// error state, and the SSE convergence (person.face_moved /
// person.merged refetch the detail when they touch this cluster).
// $app/state and $app/navigation are mocked because the real router
// needs a running SvelteKit app; the events module is mocked with a
// captured handler registry, the timeline page's pattern.

const { pageState, handlers } = vi.hoisted(() => ({
	pageState: { params: { id: 'c1' } as { id: string } },
	handlers: [] as Array<(event: unknown) => void>
}));

vi.mock('$app/state', () => ({ page: pageState }));
vi.mock('$app/navigation', () => ({
	goto: vi.fn()
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

import { goto } from '$app/navigation';

function face(id: string, assetId: string, file: string, takenAt: string): ClusterDetail['faces'][number] {
	return {
		id,
		assetId,
		score: 0.9,
		thumbnail: `/api/v1/faces/${id}/thumbnail`,
		asset: { takenAt, originalFilename: file }
	};
}

const adaFaces = [
	face('f1', 'a1', 'IMG_1234.jpg', '2026-10-13T12:30:00Z'),
	face('f2', 'a2', 'IMG_1235.jpg', '2026-10-12T08:00:00Z')
];

// The server's current answer; the convergence tests mutate it to
// stand for the server's post-event truth.
let detailState: ClusterDetail = {
	clusterId: 'c1',
	label: { name: 'Ada', isHidden: false },
	faces: adaFaces
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
		'/api/v1/people/c1': () => jsonResponse(200, detailState),
		'/api/v1/people': () =>
			jsonResponse(200, [
				{
					clusterId: 'p1',
					name: 'Grace',
					faceCount: 9,
					representativeFaceId: 'f-p1',
					thumbnail: '/api/v1/faces/f-p1/thumbnail',
					isHidden: false
				}
			]),
		'/api/v1/people/unnamed?limit=60': () =>
			jsonResponse(200, {
				items: [
					{
						clusterId: 'u1',
						faceCount: 3,
						representativeFaceId: 'f-u1',
						thumbnail: '/api/v1/faces/f-u1/thumbnail'
					}
				]
			}),
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

function faceCards(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.face-card'));
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
	for (const handler of [...handlers]) {
		handler(event);
	}
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	vi.clearAllMocks();
	detailState = {
		clusterId: 'c1',
		label: { name: 'Ada', isHidden: false },
		faces: adaFaces
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
	// The toast organ is shared module state: every case starts empty
	// (the echo assertions read exactly their own flow's toasts).
	clearToasts();
});

afterEach(() => {
	vi.unstubAllGlobals();
	clearToasts();
});

test('the labeled detail renders the name, the faces and their asset context', async () => {
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.person-detail__title')?.textContent).toBe('Ada');
	expect(document.body.querySelector('.person-detail__badge')).toBeNull();
	expect(faceCards()).toHaveLength(2);

	const first = faceCards()[0];
	expect(first?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f1/thumbnail');
	// The face links into the asset viewer at its asset.
	expect(first?.querySelector('a.face-card__link')?.getAttribute('href')).toBe('/a/a1');
	expect(first?.textContent).toContain('IMG_1234.jpg');
	expect(first?.textContent).toContain('Oct 13, 2026');
	expect(faceCards()[1]?.querySelector('a.face-card__link')?.getAttribute('href')).toBe('/a/a2');

	// The back link returns to the named region.
	expect(document.body.querySelector('.person-detail__back')?.getAttribute('href')).toBe('/people');
	unmount(page);
	flushSync();
});

test('the unnamed detail renders the unnamed header and the naming CTA', async () => {
	detailState = { clusterId: 'c1', label: null, faces: adaFaces };
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.person-detail__title')?.textContent).toBe('Unnamed person');
	expect(buttonByText('Name this person')).toBeTruthy();
	unmount(page);
	flushSync();
});

test('renaming PUTs the label and the header adopts the response', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/label': () =>
				jsonResponse(200, { clusterId: 'c1', name: 'Ada Lovelace', isHidden: false })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Rename').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	fillField(dialog as ParentNode, 'Ada Lovelace');
	buttonByText('Save name', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/label');
	expect(puts).toHaveLength(1);
	expect(JSON.parse(String(puts[0]?.[1]?.body))).toEqual({ name: 'Ada Lovelace' });
	expect(document.body.querySelector('.person-detail__title')?.textContent).toBe('Ada Lovelace');
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('hiding a named person PUTs the switch and the badge and notice follow', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/hidden': () => jsonResponse(200, { clusterId: 'c1', name: 'Ada', isHidden: true })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Hide').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/hidden');
	expect(puts).toHaveLength(1);
	expect(JSON.parse(String(puts[0]?.[1]?.body))).toEqual({ isHidden: true });
	expect(document.body.querySelector('.person-detail__badge')?.textContent).toBe('Hidden');
	expect(document.body.querySelector('.person-detail__notice')?.textContent).toContain(
		'They stay in your people list'
	);
	// The toggle flipped: the action now offers the unhide.
	expect(() => buttonByText('Unhide')).toBeTruthy();
	unmount(page);
	flushSync();
});

test('hiding an unlabeled person surfaces the requires-label error and keeps the naming nudge', async () => {
	detailState = { clusterId: 'c1', label: null, faces: adaFaces };
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/hidden': () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'the person carries no label to hide',
					details: { key: 'person_hide_requires_label' }
				})
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Hide').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const error = document.body.querySelector('.person-detail__hide-error');
	expect(error?.getAttribute('role')).toBe('alert');
	expect(error?.textContent).toBe('Name this person before hiding them.');
	// The honest nudge: the naming CTA stays the way out.
	expect(buttonByText('Name this person')).toBeTruthy();
	// No label write ever fired.
	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/label');
	expect(puts).toHaveLength(0);
	unmount(page);
	flushSync();
});

test("merge-into POSTs the merge and lands on the surviving person's page", async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/merge': () => jsonResponse(200, { survivingClusterId: 'p1', mergedClusterIds: ['c1'] })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Merge into…').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The merge face: the filterable person target list (this cluster
	// never its own target), the honest cost in the body.
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Merge into…');
	expect(dialog?.textContent).toContain('This cluster disappears and its name is lost.');
	expect(dialog?.textContent).toContain('Existing people');
	// The target list carries Grace's row (the face count beside the
	// name) -- the local filter narrows to her.
	const rows = Array.from((dialog as ParentNode).querySelectorAll<HTMLButtonElement>('.person-picker__row'));
	expect(rows.map((row) => (row.textContent ?? '').replace(/\s+/g, ' ').trim())).toEqual(['Grace 9']);
	const filter = (dialog as ParentNode).querySelector<HTMLInputElement>('.person-picker__filter');
	if (filter === null) {
		throw new Error('missing merge filter');
	}
	filter.value = 'grace';
	filter.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	expect(Array.from((dialog as ParentNode).querySelectorAll('.person-picker__row'))).toHaveLength(1);
	// No toast rides the merge: the landing is the feedback.
	rows[0]!.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	buttonByText('Merge into…', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/merge');
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		sourceClusterIds: ['c1'],
		targetClusterId: 'p1'
	});
	// The navigation lands on the SURVIVING cluster's person page (the
	// response's own answer names the destination).
	expect(vi.mocked(goto)).toHaveBeenCalledWith('/people/p1');
	expect(toasts()).toHaveLength(0);
	unmount(page);
	flushSync();
});

test('the unname write is immediate: no confirm, the toast echo, the unnamed state', async () => {
	// The label route answers both arms: the DELETE releases the
	// label (204), anything else is the rename's PUT shape.
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/label': (init?: RequestInit) =>
				init?.method === 'DELETE'
					? new Response(null, { status: 204 })
					: jsonResponse(200, { clusterId: 'c1', name: 'Ada', isHidden: false })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The unname affordance: icon-only, the labeled state's own row
	// (an unnamed cluster has nothing to unname).
	const unname = document.body.querySelector<HTMLButtonElement>('[aria-label="Remove name"]');
	expect(unname).not.toBeNull();
	unname!.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	// The DELETE ran once, immediately -- no dialog ever stood between
	// the click and the write (naming is reversible).
	const deletes = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/people/c1/label' && init?.method === 'DELETE'
	);
	expect(deletes).toHaveLength(1);
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();

	// The echo landed and the header adopted the unnamed state (the
	// naming CTA is the way back).
	expect(toasts().map((entry) => entry.text)).toEqual(['Name removed; this person is unnamed again.']);
	expect(document.body.querySelector('.person-detail__title')?.textContent?.trim()).toBe('Unnamed person');
	expect(() => buttonByText('Name this person')).not.toThrow();
	expect(document.body.querySelector('[aria-label="Remove name"]')).toBeNull();
	unmount(page);
	flushSync();
});

test('moving a face POSTs the move and the face list refetches', async () => {
	// The server's post-move truth: the moved face is gone from the
	// cluster when the page's convergence refetch lands.
	const afterMove: ClusterDetail = {
		clusterId: 'c1',
		label: { name: 'Ada', isHidden: false },
		faces: [adaFaces[1] as ClusterDetail['faces'][number]]
	};
	let moved = false;
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1': () => jsonResponse(200, moved ? afterMove : detailState),
			'/api/v1/faces/f1/move': () => {
				moved = true;
				return jsonResponse(200, { faceId: 'f1', clusterId: 'p1' });
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(faceCards()).toHaveLength(2);

	faceCards()[0]
		?.querySelector<HTMLButtonElement>('.face-card__move')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Move face');
	expect(dialog?.textContent).toContain('Pick the person this face joins.');
	chooseOption(dialog as ParentNode, 'p1');
	buttonByText('Move', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/faces/f1/move');
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({ targetClusterId: 'p1' });
	// The refetched detail dropped the moved face.
	expect(faceCards()).toHaveLength(1);
	expect(faceCards()[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f2/thumbnail');
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('an unknown or invisible cluster renders the person-not-found error', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/people/c1': () =>
				jsonResponse(404, { code: 'PERSON_NOT_FOUND', message: 'no such person for this caller' })
		})
	);
	const page = mountPage();
	await settle();

	const failure = document.body.querySelector('.person-detail__failure');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('This person does not exist or is not visible to you.');
	expect(faceCards()).toHaveLength(0);
	unmount(page);
	flushSync();
});

test('a foreign label event changes nothing the server does not change (server truth only)', async () => {
	detailState = { clusterId: 'c1', label: null, faces: adaFaces };
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(document.body.querySelector('.person-detail__title')?.textContent).toBe('Unnamed person');

	// Another account labeled THIS cluster; the refetched truth (the
	// caller's own label state) is still none, and the page invents
	// no name.
	firePersonEvent({ id: 1, name: 'person.labeled', clusterId: 'c1', label: 'Ada', userId: 'someone-else' });
	await settle();

	expect(document.body.querySelector('.person-detail__title')?.textContent).toBe('Unnamed person');
	expect(document.body.querySelector('.person-detail__title')?.textContent).not.toContain('Ada');
	unmount(page);
	flushSync();
});

test('SSE person.face_moved refetches when the move touches this cluster', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(faceCards()).toHaveLength(2);
	const initial = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1').length;

	// A face of another cluster moving between two other clusters
	// does not concern this page.
	firePersonEvent({ id: 1, name: 'person.face_moved', faceId: 'f9', clusterId: 'cX', userId: 'u1' });
	await settle();
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1').length).toBe(initial);

	// Our face left for another cluster: the refetch decides.
	detailState = {
		clusterId: 'c1',
		label: { name: 'Ada', isHidden: false },
		faces: [adaFaces[1] as ClusterDetail['faces'][number]]
	};
	firePersonEvent({ id: 2, name: 'person.face_moved', faceId: 'f1', clusterId: 'p1', userId: 'u1' });
	await settle();

	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1').length).toBe(initial + 1);
	expect(faceCards()).toHaveLength(1);
	unmount(page);
	flushSync();
});

test('SSE person.merged dissolving this cluster renders the error state', async () => {
	let dissolved = false;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (String(input) === '/api/v1/people/c1') {
			if (dissolved) {
				return jsonResponse(404, { code: 'PERSON_NOT_FOUND', message: 'no such person' });
			}
			return jsonResponse(200, detailState);
		}
		throw new Error(`unexpected fetch ${String(input)}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(faceCards()).toHaveLength(2);

	dissolved = true;
	firePersonEvent({
		id: 2,
		name: 'person.merged',
		survivingClusterId: 'p1',
		mergedClusterIds: ['c1'],
		userId: 'u1'
	});
	await settle();

	expect(document.body.querySelector('.person-detail__failure')?.textContent).toContain(
		'This person does not exist or is not visible to you.'
	);
	expect(faceCards()).toHaveLength(0);
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from the event stream', async () => {
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();
	expect(handlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(handlers).toHaveLength(0);
});
