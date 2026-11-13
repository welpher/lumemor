import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearToasts, toasts } from '$lib/components/toast.svelte';
import type { ClusterDetail } from '$lib/api/people';
import type { TimelineItem } from '$lib/api/timeline';
import Page from './+page.svelte';

// The person detail (web-ia §7.17/§7.21): the sticky context bar
// (back + the person's icon actions), the profile row (the 72px
// avatar, the name with its pencil, the photo count and the birth
// date), the photo WALL under it (the shared component over the
// timeline endpoint's cluster narrow), the avatar picker (faces only,
// the stored pick badged), the birth date dialog, the unname with its
// toast echo, the merge-into fold, and the unnamed variant (the name
// slot renders the localized unnamed title and the action set swaps
// the unname for the naming). $app/state and $app/navigation are
// mocked because the real router needs a running SvelteKit app; the
// events module is mocked with a captured handler registry, the
// timeline page's pattern.

const { pageState, handlers, assetHandlers } = vi.hoisted(() => ({
	pageState: { params: { id: 'c1' } as { id: string } },
	handlers: [] as Array<(event: unknown) => void>,
	assetHandlers: [] as Array<(event: unknown) => void>
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

/** One wall item of the person's photos (the timeline shape). */
function item(id: string, takenAt: string): TimelineItem {
	return {
		id,
		ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
		kind: 'image',
		originalFilename: 'IMG_1234.jpg',
		fileSize: 4821933,
		mimeType: 'image/jpeg',
		checksum: '9f86d081884c7d65a2feaa0c55ad015',
		createdAt: '2026-10-29T09:12:00Z',
		takenAt,
		takenAtSource: 'exif',
		isTrashed: false,
		isFavorite: false,
		isReady: true,
		width: 4032,
		height: 3024,
		durationMs: null,
		blurhash: 'UHL6aAof00WBWB00ayWB00',
		livePhotoVideoId: null
	};
}

const wallItems = [item('a1', '2026-10-13T12:30:00Z'), item('a2', '2026-10-12T08:00:00Z')];

// The server's current answer; the convergence tests mutate it to
// stand for the server's post-event truth.
let detailState: ClusterDetail = {
	clusterId: 'c1',
	label: { name: 'Ada' },
	birthDate: null,
	avatarFaceId: null,
	photoCount: 2,
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
		'/api/v1/assets/timeline?clusterId=c1&limit=100': () =>
			jsonResponse(200, { items: wallItems, nextCursor: null }),
		'/api/v1/people': () =>
			jsonResponse(200, [
				{
					clusterId: 'p1',
					name: 'Grace',
					photoCount: 9,
					representativeFaceId: 'f-p1',
					thumbnail: '/api/v1/faces/f-p1/thumbnail'
				}
			]),
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
	// The chains a mount runs (detail fetch -> the wall's page fetch ->
	// the adoptions) span several task turns, and the adoptions write
	// into Svelte's scheduler queue: a few timer rounds plus one forced
	// flush make the DOM agree with the settled state.
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

function modal(): HTMLElement | null {
	return document.body.querySelector<HTMLElement>('.modal__panel');
}

function buttonByLabel(label: string): HTMLButtonElement {
	const found = document.body.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing button labelled ${label}`);
	}
	return found;
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

/** Fire one parsed person event to every live subscriber. */
function firePersonEvent(event: Record<string, unknown>): void {
	for (const handler of [...handlers]) {
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
	handlers.length = 0;
	assetHandlers.length = 0;
	pageState.params.id = 'c1';
	vi.clearAllMocks();
	detailState = {
		clusterId: 'c1',
		label: { name: 'Ada' },
		birthDate: null,
		avatarFaceId: null,
		photoCount: 2,
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
	// The toast organ is shared module state: every case starts empty.
	clearToasts();
});

afterEach(() => {
	vi.unstubAllGlobals();
	clearToasts();
});

test('the named detail renders the context bar, the profile row and the photo wall', async () => {
	const stub = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', stub);
	const fetchMockURLs = () => stub.mock.calls.map(([input]) => String(input));
	const page = mountPage();
	await settle();

	// THE CONTEXT BAR (§7.21): back + the person's icon actions, the
	// bar itself sticky at the pinned rung.
	const bar = document.body.querySelector('.person-bar');
	expect(bar?.className).toContain('sticky');
	expect(bar?.className).toContain('border-b');
	const back = document.body.querySelector<HTMLAnchorElement>('.person-bar__back');
	expect(back?.getAttribute('href')).toBe('/people');
	expect(back?.getAttribute('aria-label')).toBe('Back to people');
	// The page root pins the wall's sticky offset to the bar's height.
	expect(
		document.body.querySelector<HTMLElement>('.person-detail')?.style.getPropertyValue('--wall-sticky-offset')
	).toBe('53px');

	// THE ACTIONS: avatar / unname (named) / merge-into -- and NO hide
	// action anywhere (§7.19's retirement).
	expect(() => buttonByLabel('Change avatar')).not.toThrow();
	expect(() => buttonByLabel('Remove name')).not.toThrow();
	expect(() => buttonByLabel('Merge into…')).not.toThrow();
	expect(() => buttonByLabel('Hide')).toThrow();
	expect(() => buttonByLabel('Unhide')).toThrow();
	expect(document.body.querySelector('.person-detail__badge')).toBeNull();

	// THE PROFILE ROW: the avatar (the best visible face -- no pick
	// yet), the name with its pencil, the photo count and the birth
	// date's add affordance.
	expect(document.body.querySelector('.person-profile__avatar img')?.getAttribute('src')).toBe(
		'/api/v1/faces/f1/thumbnail'
	);
	expect(document.body.querySelector('.person-profile__name')?.textContent?.trim()).toBe('Ada');
	expect(document.body.querySelector('.person-profile__rename')?.getAttribute('aria-label')).toBe('Rename');
	const count = document.body.querySelector('.person-profile__count');
	expect(count?.textContent?.trim()).toBe('2');
	expect(count?.getAttribute('aria-label')).toBe('2 photos');
	expect(document.body.querySelector('.person-profile__birthdate')?.textContent?.trim()).toBe('Add birth date');

	console.log('CALLS', fetchMockURLs());
	// THE WALL: the shared component over this person's photos (the
	// month group and the tiles render).
	expect(document.body.querySelector('.person-wall .wall__groups')).not.toBeNull();
	expect(document.body.querySelectorAll('.person-wall .tile').length).toBe(2);
	unmount(page);
	flushSync();
});

test('the unnamed variant renders "Unnamed" with the naming entry and no unname', async () => {
	detailState = { ...detailState, label: null };
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.person-profile__name')?.textContent?.trim()).toBe('Unnamed');
	// The bar's action set swaps the unname for the naming; the
	// profile row's pencil reads as the same naming entry.
	expect(() => buttonByLabel('Name this person')).not.toThrow();
	expect(() => buttonByLabel('Remove name')).toThrow();
	expect(document.body.querySelector('.person-profile__rename')?.getAttribute('aria-label')).toBe('Name this person');
	// The profile row still offers the birth date: the entity's
	// attribute does not depend on the name.
	expect(() => buttonByText('Add birth date')).not.toThrow();
	// The page's title follows the state.
	expect(document.title).toBe('Unnamed · lumemor');
	unmount(page);
	flushSync();
});

test('the birth date line renders through Intl with the machine value in <time>', async () => {
	detailState = { ...detailState, birthDate: '1992-03-15' };
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	const line = document.body.querySelector('.person-profile__birthdate');
	const time = line?.querySelector('time');
	expect(time?.getAttribute('datetime')).toBe('1992-03-15');
	// The rendered text is the locale's numeric form -- never a
	// hand-assembled date.
	expect(line?.textContent?.replace(/\s+/g, ' ').trim()).toBe('Born 3/15/1992');
	unmount(page);
	flushSync();
});

test('renaming PUTs the label and the profile row adopts the response', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/label': () => jsonResponse(200, { clusterId: 'c1', name: 'Ada Lovelace' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Rename').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	fillField(dialog as ParentNode, 'Ada Lovelace');
	buttonByText('Save name', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/label');
	expect(puts).toHaveLength(1);
	expect(JSON.parse(String(puts[0]?.[1]?.body))).toEqual({ name: 'Ada Lovelace' });
	expect(document.body.querySelector('.person-profile__name')?.textContent?.trim()).toBe('Ada Lovelace');
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('naming an unnamed cluster lands the named state in place', async () => {
	detailState = { ...detailState, label: null };
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/label': () => jsonResponse(200, { clusterId: 'c1', name: 'Ada' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Name this person').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Name this person');
	fillField(dialog as ParentNode, 'Ada');
	buttonByText('Save name', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(document.body.querySelector('.person-profile__name')?.textContent?.trim()).toBe('Ada');
	expect(() => buttonByLabel('Remove name')).not.toThrow();
	unmount(page);
	flushSync();
});

test('the avatar picker shows faces only, badges the stored pick and PUTs the choice', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/avatar': () => jsonResponse(200, { clusterId: 'c1', avatarFaceId: 'f2' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Change avatar').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Change avatar');
	expect(dialog?.textContent).toContain("Pick one of this person's faces as their avatar.");

	// The candidates are the person's FACES: the crops, no file name
	// and no source photo name anywhere in the grid.
	const options = Array.from((dialog as ParentNode).querySelectorAll<HTMLButtonElement>('.avatar-option'));
	expect(options).toHaveLength(2);
	expect(options[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f1/thumbnail');
	expect(dialog?.textContent).not.toContain('IMG_1234.jpg');
	// No pick is stored yet: no badge stands, and the footer's submit
	// waits for a pick.
	expect(dialog?.querySelector('.avatar-option__badge')).toBeNull();

	options[1]!.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(options[1]?.getAttribute('aria-pressed')).toBe('true');
	buttonByText('Set avatar', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/avatar');
	expect(puts).toHaveLength(1);
	expect(JSON.parse(String(puts[0]?.[1]?.body))).toEqual({ faceId: 'f2' });
	// The profile row's crop follows the stored pick (the avatar is
	// the entity's attribute, adopted from the response).
	expect(document.body.querySelector('.person-profile__avatar img')?.getAttribute('src')).toBe(
		'/api/v1/faces/f2/thumbnail'
	);
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the avatar picker badges the stored pick on open', async () => {
	detailState = { ...detailState, avatarFaceId: 'f2' };
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();

	buttonByLabel('Change avatar').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	const badges = Array.from((dialog as ParentNode).querySelectorAll<HTMLElement>('.avatar-option__badge'));
	expect(badges).toHaveLength(1);
	expect(badges[0]?.textContent?.trim()).toBe('Avatar');
	expect(badges[0]?.closest('button')?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f2/thumbnail');
	unmount(page);
	flushSync();
});

test('the birth date dialog PUTs the picked date and the profile row shows it', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/birthdate': () => jsonResponse(200, { clusterId: 'c1', birthDate: '1992-03-15' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByText('Add birth date').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Birth date');
	expect(dialog?.textContent).toContain("The birth date works out this person's age when a photo was taken.");
	// Unset: no clear arm stands (there is nothing to clear).
	expect(dialog?.textContent).not.toContain('Clear');
	const field = dialog?.querySelector<HTMLInputElement>('input[type="date"]');
	expect(field).not.toBeNull();
	field!.value = '1992-03-15';
	field!.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	buttonByText('Save', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/birthdate');
	expect(puts).toHaveLength(1);
	expect(JSON.parse(String(puts[0]?.[1]?.body))).toEqual({ birthDate: '1992-03-15' });
	const line = document.body.querySelector('.person-profile__birthdate');
	expect(line?.querySelector('time')?.getAttribute('datetime')).toBe('1992-03-15');
	expect(modal()).toBeNull();
	unmount(page);
	flushSync();
});

test('the birth date clear PUTs the explicit null', async () => {
	detailState = { ...detailState, birthDate: '1992-03-15' };
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/birthdate': () => jsonResponse(200, { clusterId: 'c1', birthDate: null })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	document.body
		.querySelector<HTMLButtonElement>('.person-profile__birthdate')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const dialog = modal();
	buttonByText('Clear', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const puts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1/birthdate');
	expect(JSON.parse(String(puts[0]?.[1]?.body))).toEqual({ birthDate: null });
	expect(document.body.querySelector('.person-profile__birthdate')?.textContent?.trim()).toBe('Add birth date');
	unmount(page);
	flushSync();
});

test('merge-into POSTs the merge and lands on the surviving person page', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/merge': () => jsonResponse(200, { survivingClusterId: 'p1', mergedClusterIds: ['c1'] })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Merge into…').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const dialog = modal();
	expect(dialog?.getAttribute('aria-label')).toBe('Merge into…');
	expect(dialog?.textContent).toContain('This cluster disappears and its name is lost.');
	expect(dialog?.textContent).toContain('Existing people');
	const rows = Array.from((dialog as ParentNode).querySelectorAll<HTMLButtonElement>('.person-picker__row'));
	expect(rows.map((row) => (row.textContent ?? '').replace(/\s+/g, ' ').trim())).toEqual(['Grace 9']);
	rows[0]!.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	buttonByText('Merge into…', dialog as ParentNode).dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const posts = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/merge');
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({
		sourceClusterIds: ['c1'],
		targetClusterId: 'p1'
	});
	expect(vi.mocked(goto)).toHaveBeenCalledWith('/people/p1');
	expect(toasts()).toHaveLength(0);
	unmount(page);
	flushSync();
});

test('the unname write is immediate: the toast echo and the unnamed variant in place', async () => {
	const fetchMock = stubFetch(
		detailRoutes({
			'/api/v1/people/c1/label': (init?: RequestInit) =>
				init?.method === 'DELETE'
					? new Response(null, { status: 204 })
					: jsonResponse(200, { clusterId: 'c1', name: 'Ada' })
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Remove name').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	const deletes = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/people/c1/label' && init?.method === 'DELETE'
	);
	expect(deletes).toHaveLength(1);
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(toasts().map((entry) => entry.text)).toEqual(['Name removed; this person is unnamed again.']);
	expect(document.body.querySelector('.person-profile__name')?.textContent?.trim()).toBe('Unnamed');
	expect(() => buttonByLabel('Name this person')).not.toThrow();
	expect(() => buttonByLabel('Remove name')).toThrow();
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
	// The bar stays (the way back is always there), the wall never
	// mounts.
	expect(document.body.querySelector('.person-bar')).not.toBeNull();
	expect(document.body.querySelector('.person-wall')).toBeNull();
	unmount(page);
	flushSync();
});

test('a foreign label event changes nothing the server does not change (server truth only)', async () => {
	detailState = { ...detailState, label: null };
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();
	expect(document.body.querySelector('.person-profile__name')?.textContent?.trim()).toBe('Unnamed');

	// Another account labeled THIS cluster; the refetched truth (the
	// caller's own label state) is still none.
	firePersonEvent({ id: 1, name: 'person.labeled', clusterId: 'c1', userId: 'someone-else' });
	await settle();
	expect(document.body.querySelector('.person-profile__name')?.textContent?.trim()).toBe('Unnamed');
	unmount(page);
	flushSync();
});

test('SSE person.avatar_updated and person.birthdate_updated converge this cluster only', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	const initial = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1').length;

	// Another person's entity write: nothing here moves.
	firePersonEvent({ id: 1, name: 'person.birthdate_updated', clusterId: 'c9', userId: 'u1' });
	await settle();
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1').length).toBe(initial);

	// This person's: the detail converges (the profile row is the
	// server's).
	detailState = { ...detailState, birthDate: '1990-01-02' };
	firePersonEvent({ id: 2, name: 'person.birthdate_updated', clusterId: 'c1', userId: 'u1' });
	await settle();
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people/c1').length).toBe(initial + 1);
	expect(document.body.querySelector('.person-profile__birthdate time')?.getAttribute('datetime')).toBe('1990-01-02');
	unmount(page);
	flushSync();
});

test('SSE person.merged dissolving this cluster renders the error state', async () => {
	let dissolved = false;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		const url = String(input);
		if (url === '/api/v1/people/c1') {
			if (dissolved) {
				return jsonResponse(404, { code: 'PERSON_NOT_FOUND', message: 'no such person' });
			}
			return jsonResponse(200, detailState);
		}
		if (url.startsWith('/api/v1/assets/timeline')) {
			return jsonResponse(200, { items: wallItems, nextCursor: null });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(document.body.querySelector('.person-wall')).not.toBeNull();

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
	expect(document.body.querySelector('.person-wall')).toBeNull();
	unmount(page);
	flushSync();
});

test('an asset hide (asset.updated) converges the profile row count too', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(document.body.querySelector('.person-profile__count')?.textContent?.trim()).toBe('2');

	// The batch hide's own event: the wall drops the tile, the profile
	// row's count follows the server.
	detailState = { ...detailState, photoCount: 1 };
	fireAssetEvent({ id: 6, name: 'asset.updated', assetId: 'a1' });
	await settle();
	expect(document.body.querySelector('.person-profile__count')?.textContent?.trim()).toBe('1');
	unmount(page);
	flushSync();
});

test('an asset trash converges the profile row count (the photo left the person)', async () => {
	const fetchMock = stubFetch(detailRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(document.body.querySelector('.person-profile__count')?.textContent?.trim()).toBe('2');

	detailState = { ...detailState, photoCount: 1 };
	fireAssetEvent({ id: 7, name: 'asset.trashed', assetId: 'a1' });
	await settle();
	expect(document.body.querySelector('.person-profile__count')?.textContent?.trim()).toBe('1');
	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from both streams', async () => {
	vi.stubGlobal('fetch', stubFetch(detailRoutes()));
	const page = mountPage();
	await settle();
	expect(handlers).toHaveLength(1);
	expect(assetHandlers.length).toBeGreaterThan(0);

	unmount(page);
	flushSync();
	expect(handlers).toHaveLength(0);
	expect(assetHandlers).toHaveLength(0);
});
