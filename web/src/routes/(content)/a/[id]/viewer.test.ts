import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems, publishWallItems } from '$lib/timeline/wall-sequence.svelte';
import { clearToasts, toasts } from '$lib/components/toast.svelte';
import type { TimelineItem } from '$lib/api/timeline';
import Page from './+page.svelte';

// The map double: the info panel's location row renders the same
// MapView the places tab uses, and the real component loads MapLibre
// lazily and needs a layout-capable host -- the row's shape is
// exercised against this stand-in (one region div, the same props).
vi.mock('$lib/map/MapView.svelte', async () => ({
	default: (await import('./MapStub.svelte')).default
}));

// The asset viewer route: the navigation walk over the wall's
// published sequence (clamp at both ends, inert off-sequence deep
// links), the image / video / Live Photo stage, the not-ready wait
// that converges through the SSE asset.ready, the info drawer's
// description edit (PATCH body, error mapping, the stored text
// untouched on failure), the action bar's curation actions with
// their optimistic flips and rollback (the permanent delete left
// the viewer for the trash page in the viewer rework), and the
// layered Escape. $app/state and $app/navigation are mocked because
// the real router needs a running SvelteKit app; the events module
// is mocked with a captured handler registry (the timeline page's
// pattern). The navigation mock captures afterNavigate callbacks
// and delivers the MOUNTING navigation's type per test ('link' for
// a wall click by default, 'enter' for a fresh-document deep link);
// later completed navigations fire through the captured callbacks.

const { pageState, handlers, personHandlers, navigationState, navigationCallbacks } = vi.hoisted(() => ({
	pageState: { params: { id: 'a1' } as { id: string } },
	handlers: [] as Array<(event: unknown) => void>,
	personHandlers: [] as Array<(event: unknown) => void>,
	navigationState: { mountType: 'link' as 'link' | 'enter' },
	navigationCallbacks: [] as Array<(navigation: { type: string }) => void>
}));

vi.mock('$app/state', () => ({ page: pageState }));
vi.mock('$app/navigation', () => ({
	goto: vi.fn(),
	afterNavigate: (callback: (navigation: { type: string }) => void) => {
		// SvelteKit delivers the navigation that mounted the route to
		// the freshly registered callback (the route mounts, then the
		// navigation completes); tests pick the mounting type.
		navigationCallbacks.push(callback);
		callback({ type: navigationState.mountType });
	}
}));
vi.mock('$lib/events', () => ({
	subscribeAssetEvents: (listener: (event: unknown) => void) => {
		handlers.push(listener);
		return () => {
			const index = handlers.indexOf(listener);
			if (index !== -1) {
				handlers.splice(index, 1);
			}
		};
	},
	subscribePersonEvents: (listener: (event: unknown) => void) => {
		personHandlers.push(listener);
		return () => {
			const index = personHandlers.indexOf(listener);
			if (index !== -1) {
				personHandlers.splice(index, 1);
			}
		};
	}
}));

import { goto } from '$app/navigation';

const itemDefaults = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image' as const,
	originalFilename: 'IMG_0001.jpg',
	fileSize: 4821933,
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

// The wall's published sequence: three ready photos, newest first
// (the middle one's Live Photo variant is a per-test detail
// override -- the sequence itself stays plain).
const sequence = [
	item('a1', '2026-11-02T10:00:00Z', { originalFilename: 'IMG_1.jpg' }),
	item('a2', '2026-11-01T09:00:00Z', { originalFilename: 'IMG_2.jpg' }),
	item('a3', '2026-10-13T12:30:00Z', { originalFilename: 'IMG_3.jpg' })
];

/** The detail body of a sequence item (the detail's extra fields). */
function detailBodyOf(source: TimelineItem, overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		description: 'Original caption.',
		isHidden: false,
		exif: {},
		takenAtZone: '+08:00',
		location: null,
		path: null,
		faces: [],
		tags: [],
		// The band's home album: a directory source the owner holds, so
		// the bar's owner-only curation arms render by default (the
		// sync-only arm has its own scene below).
		albums: [{ id: 'al-src', title: 'Album al-src', type: 'directory', callerRole: 'owner', sourceWritable: true }],
		place: null,
		...source,
		...overrides
	};
}

/** One face summary of the detail's faces array (the contract shape). */
function faceSummary(id: string, overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		id,
		score: 0.9,
		thumbnail: `/api/v1/faces/${id}/thumbnail`,
		bbox: { x: 200, y: 200, w: 300, h: 300 },
		clusterId: `c-${id}`,
		label: null,
		birthDate: null,
		...overrides
	};
}

/** One named person of the assign dialog's target list. */
function personTarget(clusterId: string, name: string): Record<string, unknown> {
	return {
		clusterId,
		name,
		photoCount: 12,
		representativeFaceId: `f-${clusterId}`,
		thumbnail: `/api/v1/faces/f-${clusterId}/thumbnail`
	};
}

/** One unnamed cluster of the assign dialog's target list. */
function unnamedTarget(clusterId: string, faceCount: number): Record<string, unknown> {
	return {
		clusterId,
		faceCount,
		representativeFaceId: `f-${clusterId}`,
		thumbnail: `/api/v1/faces/f-${clusterId}/thumbnail`
	};
}

const videoSource = item('v1', '2026-10-12T08:00:00Z', {
	kind: 'video',
	originalFilename: 'VID_1.mp4',
	mimeType: 'video/quicktime',
	fileSize: 9437184,
	durationMs: 12345,
	isReady: false,
	width: null,
	height: null,
	blurhash: null
});

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** A satisfiable single-range answer (the probe's success shape). */
function rangeResponse(): Response {
	return new Response('x', { status: 206, headers: { 'Content-Range': 'bytes 0-0/12345' } });
}

type RouteRule = { method: string; url: string; respond: () => Response | Promise<Response> };

function get(url: string, respond: () => Response | Promise<Response>): RouteRule {
	return { method: 'GET', url, respond };
}

function patch(url: string, respond: () => Response | Promise<Response>): RouteRule {
	return { method: 'PATCH', url, respond };
}

function post(url: string, respond: () => Response | Promise<Response>): RouteRule {
	return { method: 'POST', url, respond };
}

function del(url: string, respond: () => Response | Promise<Response>): RouteRule {
	return { method: 'DELETE', url, respond };
}

function put(url: string, respond: () => Response | Promise<Response>): RouteRule {
	return { method: 'PUT', url, respond };
}

/** The default routes of the viewer surface (an image asset). */
function viewerRoutes(id: string, extra: RouteRule[] = []): RouteRule[] {
	return [get(`/api/v1/assets/${id}?locale=en`, () => jsonResponse(200, detailBodyOf(sequence[1]!))), ...extra];
}

function stubFetch(rules: RouteRule[]): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		const method = (init?.method ?? 'GET').toUpperCase();
		const rule = rules.find((candidate) => candidate.method === method && candidate.url === url);
		if (rule === undefined) {
			throw new Error(`unexpected fetch ${method} ${url}`);
		}
		return rule.respond();
	});
}

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
	await tick();
}

/** Mount the page for one route id; the fetch mock is returned for call assertions. */
function mountViewer(
	id: string,
	rules: RouteRule[]
): { app: ReturnType<typeof mount>; fetchMock: ReturnType<typeof vi.fn> } {
	pageState.params.id = id;
	const fetchMock = stubFetch(rules);
	vi.stubGlobal('fetch', fetchMock);
	const app = mount(Page, { target: document.body });
	flushSync();
	return { app, fetchMock };
}

function pressKey(key: string, target: EventTarget = window): void {
	target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }));
	flushSync();
}

function element(selector: string): HTMLElement {
	const found = document.body.querySelector<HTMLElement>(selector);
	if (found === null) {
		throw new Error(`missing element ${selector}`);
	}
	return found;
}

function buttonByText(text: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

function click(control: HTMLElement): void {
	control.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

/** Fire one completed navigation to the mounted route (the router's afterNavigate delivery). */
function navigate(type: string): void {
	for (const callback of [...navigationCallbacks]) {
		callback({ type });
	}
	flushSync();
}

/** Open the info drawer through the top-bar toggle. */
function openDrawer(): void {
	click(element('[aria-label="Photo information"]'));
}

/** Arm the description's edit input: the click-to-edit line. */
function openDescriptionEdit(): void {
	click(element('button.info__description'));
}

/** Submit the description's edit through the input's Enter arm. */
function submitDescriptionEdit(): void {
	const field = element('input.info__editor');
	field.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, cancelable: true }));
	flushSync();
}

/** Cancel the description's edit through the input's Escape arm. */
function cancelDescriptionEdit(): void {
	const field = element('input.info__editor');
	field.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));
	flushSync();
}

/** One icon-only button of the viewer's action bar, addressed by its aria-label. */
function barButton(label: string): HTMLButtonElement {
	const found = document.body.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing bar button ${label}`);
	}
	return found;
}

function fillTextarea(selector: string, value: string): void {
	const field = document.body.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector);
	if (field === null) {
		throw new Error(`missing field ${selector}`);
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
}

function fillInput(field: HTMLInputElement, value: string): void {
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
}

/** Fire one parsed person event to every live subscriber. */
function firePersonEvent(event: Record<string, unknown>): void {
	for (const handler of [...personHandlers]) {
		handler(event);
	}
	flushSync();
}

/** The request a recorded fetch mock call made (method, url, body). */
function requestOf(fetchMock: ReturnType<typeof vi.fn>, method: string, url: string): RequestInit | undefined {
	const call = fetchMock.mock.calls.find(
		([input, init]) => String(input) === url && (init?.method ?? 'GET').toUpperCase() === method
	);
	return call?.[1];
}

// The signed-in owner of the test sequence's assets: the session the
// viewer's ownership comparisons resolve against (every
// itemDefaults.ownerId names this account).
const ownerId = '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60';

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	personHandlers.length = 0;
	navigationState.mountType = 'link';
	navigationCallbacks.length = 0;
	vi.clearAllMocks();
	clearWallItems();
	// The info panel's open preference persists in localStorage:
	// every test starts from the default (closed) unless it writes the
	// key itself.
	localStorage.clear();
	// Boot the session store as the sequence's owner: the drawer's
	// owner-only affordances resolve against this identity (the
	// identity seam's store carries it; the interceptor stays inert
	// because these routes answer 200s).
	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			jsonResponse(200, [
				{
					id: '6f9c2d18-9a4c-4a1b-b1a1-000000000001',
					kind: 'WEB',
					userId: ownerId,
					role: 'admin',
					deviceName: 'Mozilla/5.0 (test)',
					createdAt: '2026-11-01T10:00:00Z',
					lastSeenAt: '2026-11-02T10:00:00Z'
				}
			])
		)
	);
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('authenticated');
	await initI18n();
	publishWallItems(sequence);
	// The toast organ is module state shared across these tests:
	// every case starts with an empty queue (the echo assertions
	// read exactly their own flow's toasts).
	clearToasts();
});

afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
	clearWallItems();
	clearToasts();
});

test('an image deep link renders the serving layers and the filmstrip', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// The view-size copy and the full-resolution serving copy, the
	// PhotoTile crossfade pattern at viewer scale.
	const images = Array.from(document.body.querySelectorAll<HTMLImageElement>('img.viewer__image'));
	expect(images.map((image) => image.getAttribute('src'))).toEqual([
		'/api/v1/assets/a2/thumbnail?size=view',
		'/api/v1/assets/a2/thumbnail?size=original'
	]);
	expect(images[0]?.getAttribute('alt')).toBe('IMG_2.jpg');
	// No video, no live toggle on the plain image rendering.
	expect(document.body.querySelector('video')).toBeNull();
	expect(document.body.querySelector('.viewer__live-toggle')).toBeNull();

	// The filmstrip walks the sequence with the current outlined.
	const thumbs = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.viewer__thumb'));
	expect(thumbs.map((thumb) => thumb.getAttribute('data-filmstrip-id'))).toEqual(['a1', 'a2', 'a3']);
	expect(thumbs[1]?.getAttribute('aria-current')).toBe('true');
	expect(thumbs[1]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/assets/a2/thumbnail?size=tile');

	// The drawer's file facts render the detail's numbers.
	openDrawer();
	expect(element('.info').textContent).toContain('IMG_2.jpg');
	expect(element('.info').textContent).toContain('4.8 MB');
	expect(element('.info__description').textContent).toBe('Original caption.');
	unmount(app);
	flushSync();
});

test('arrow keys walk the sequence and clamp at both ends', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	pressKey('ArrowRight');
	expect(goto).toHaveBeenCalledWith('/a/a3', { replaceState: true });
	pressKey('ArrowLeft');
	expect(goto).toHaveBeenCalledWith('/a/a1', { replaceState: true });
	// The arrows replace the route: no history stacking between
	// neighbors (Escape's back stays the one-step return).
	expect(vi.mocked(goto)).toHaveBeenCalledTimes(2);
	unmount(app);
	flushSync();

	// The head: ArrowLeft is inert (the clamp).
	const head = mountViewer('a1', [
		get('/api/v1/assets/a1?locale=en', () => jsonResponse(200, detailBodyOf(sequence[0]!)))
	]);
	await settle();
	pressKey('ArrowLeft');
	expect(vi.mocked(goto)).toHaveBeenCalledTimes(2);
	unmount(head.app);
	flushSync();

	// The tail: ArrowRight is inert.
	const tail = mountViewer('a3', [
		get('/api/v1/assets/a3?locale=en', () => jsonResponse(200, detailBodyOf(sequence[2]!)))
	]);
	await settle();
	pressKey('ArrowRight');
	expect(vi.mocked(goto)).toHaveBeenCalledTimes(2);
	unmount(tail.app);
	flushSync();
});

test('a deep link off the sequence still loads the asset; arrows and filmstrip current are inert', async () => {
	const offPage = item('zz', '2026-10-10T08:00:00Z', { originalFilename: 'OFF.jpg' });
	const { app } = mountViewer('zz', [
		get('/api/v1/assets/zz?locale=en', () => jsonResponse(200, detailBodyOf(offPage)))
	]);
	await settle();

	// The off-sequence detail renders (the bar carries no filename
	// any more: the browser title is the name's one stage face).
	expect(document.title).toBe('OFF.jpg · lumemor');
	expect(
		Array.from(document.body.querySelectorAll('img.viewer__image')).map((image) => image.getAttribute('src'))
	).toContain('/api/v1/assets/zz/thumbnail?size=view');
	// No neighbor arrows, no current filmstrip thumb, inert arrows.
	expect(document.body.querySelector('.viewer__arrow')).toBeNull();
	expect(
		Array.from(document.body.querySelectorAll('.viewer__thumb')).every(
			(thumb) => thumb.getAttribute('aria-current') === null
		)
	).toBe(true);
	pressKey('ArrowLeft');
	pressKey('ArrowRight');
	expect(vi.mocked(goto)).not.toHaveBeenCalled();

	// A filmstrip click still navigates into the sequence.
	click(element('[data-filmstrip-id="a3"]'));
	expect(goto).toHaveBeenCalledWith('/a/a3', { replaceState: true });
	unmount(app);
	flushSync();
});

test('Escape and the back affordance leave through history.back', async () => {
	const back = vi.spyOn(history, 'back').mockImplementation(() => undefined);
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	pressKey('Escape');
	expect(back).toHaveBeenCalledTimes(1);

	click(element('[aria-label="Back to the timeline"]'));
	expect(back).toHaveBeenCalledTimes(2);
	unmount(app);
	flushSync();
});

test('the description edit sends the PATCH body and maps failures without touching the stored text', async () => {
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		// The unknown code pins the shared fallback: the drawer
		// renders the generic message, never the raw code.
		patch('/api/v1/assets/a2?locale=en', () => jsonResponse(502, { code: 'PROXY_TEAPOT' }))
	]);
	await settle();

	openDrawer();
	expect(element('.info__description').textContent).toBe('Original caption.');

	openDescriptionEdit();
	fillTextarea('input.info__editor', 'My new caption.');
	submitDescriptionEdit();
	await settle();

	// The exact PATCH body of the contract.
	const init = requestOf(fetchMock, 'PATCH', '/api/v1/assets/a2?locale=en');
	expect(init).toBeDefined();
	expect(JSON.parse(String(init?.body))).toEqual({ description: 'My new caption.' });

	// The failure keeps the edit mode (the draft stays for a retry)
	// and shows the generic message, never the raw code.
	expect(document.body.querySelector('input.info__editor')).not.toBeNull();
	const error = element('.info__error');
	expect(error.getAttribute('role')).toBe('alert');
	expect(error.textContent).toBe('Something went wrong. Please try again.');
	expect(error.textContent).not.toContain('proxy_teapot');

	// The stored text never changed: canceling shows the original.
	cancelDescriptionEdit();
	expect(element('.info__description').textContent).toBe('Original caption.');
	unmount(app);
	flushSync();
});

test('a successful edit adopts the response description', async () => {
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		patch('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { description: 'The saved caption.' }))
		)
	]);
	await settle();

	openDrawer();
	openDescriptionEdit();
	fillTextarea('input.info__editor', 'The saved caption.');
	submitDescriptionEdit();
	await settle();

	// The response IS the fresh detail: the read view adopts it.
	expect(document.body.querySelector('input.info__editor')).toBeNull();
	expect(element('.info__description').textContent).toBe('The saved caption.');
	unmount(app);
	flushSync();
});

test('the Live Photo toggle leads the action row and plays the motion in place', async () => {
	const liveDetail = () => jsonResponse(200, detailBodyOf(sequence[1]!, { livePhotoVideoId: 'c2' }));
	const probeCalls: string[] = [];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', liveDetail),
		get('/api/v1/assets/c2/play', () => {
			probeCalls.push('c2');
			return rangeResponse();
		})
	]);
	await settle();

	// The still renders first; the toggle is the action row's FIRST
	// button, icon-only, with the play arm's label and a false state.
	const toggle = element('.viewer__live-toggle');
	const actions = element('.viewer__bar-actions');
	expect(actions.firstElementChild).toBe(toggle);
	expect(toggle.getAttribute('aria-pressed')).toBe('false');
	expect(toggle.getAttribute('aria-label')).toBe('Play the Live Photo');
	expect(toggle.querySelector('svg')).not.toBeNull();
	expect(document.body.querySelector('video')).toBeNull();

	click(toggle);
	await settle();

	// The probe hit the COMPONENT's play endpoint (the pair's
	// motion plays through the component id), and the motion covers
	// the still once, muted, inside the same media box.
	expect(probeCalls).toEqual(['c2']);
	const live = document.body.querySelector<HTMLVideoElement>('video.viewer__live');
	expect(live?.getAttribute('src')).toBe('/api/v1/assets/c2/play');
	expect(live?.autoplay).toBe(true);
	expect(live?.muted).toBe(true);
	expect(live?.loop).toBe(false);
	expect(element('.viewer__media').contains(live!)).toBe(true);
	const pressed = element('.viewer__live-toggle');
	expect(pressed.getAttribute('aria-pressed')).toBe('true');
	expect(pressed.getAttribute('aria-label')).toBe('Stop the Live Photo');

	// The stream's own end returns the still -- the caption's clip,
	// not a loop.
	live?.dispatchEvent(new Event('ended'));
	flushSync();
	expect(document.body.querySelector('video.viewer__live')).toBeNull();
	expect(element('.viewer__live-toggle').getAttribute('aria-pressed')).toBe('false');

	// Playing again, then stopping with the same button: click-click.
	click(element('.viewer__live-toggle'));
	await settle();
	expect(document.body.querySelector('video.viewer__live')).not.toBeNull();
	click(element('.viewer__live-toggle'));
	expect(document.body.querySelector('video.viewer__live')).toBeNull();
	unmount(app);
	flushSync();

	// A refused stream returns the still silently (no toast, no error
	// card): the record is fine, the component's copy is not.
	const refused = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', liveDetail),
		get('/api/v1/assets/c2/play', () => rangeResponse())
	]);
	await settle();
	click(element('.viewer__live-toggle'));
	await settle();
	document.body.querySelector('video.viewer__live')?.dispatchEvent(new Event('error'));
	flushSync();
	expect(document.body.querySelector('video.viewer__live')).toBeNull();
	expect(document.body.querySelector('[role="alert"]')?.textContent ?? '').not.toContain('Live');
	unmount(refused.app);
	flushSync();

	// The not-ready branch: the probe answers ASSET_NOT_READY and
	// the still photo stays (each press re-probes).
	let streamReady = false;
	const waiting = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', liveDetail),
		get('/api/v1/assets/c2/play', () =>
			streamReady ? rangeResponse() : jsonResponse(404, { code: 'ASSET_NOT_READY' })
		)
	]);
	await settle();
	click(element('.viewer__live-toggle'));
	await settle();
	expect(document.body.querySelector('video.viewer__live')).toBeNull();
	expect(document.body.querySelector('img.viewer__image')).not.toBeNull();
	expect(element('.viewer__live-toggle').getAttribute('aria-pressed')).toBe('false');

	// A later press lands once the stream appeared.
	streamReady = true;
	click(element('.viewer__live-toggle'));
	await settle();
	expect(document.body.querySelector('video.viewer__live')).not.toBeNull();
	unmount(waiting.app);
	flushSync();
});

test('a shared member holds the Live Photo toggle: playing is a read', async () => {
	// The viewer mounts for a member (isOwner false): every editing
	// arm is absent from the bar, the toggle is not -- the motion is
	// part of what a share shows.
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { livePhotoVideoId: 'c2' }))
		),
		get('/api/v1/assets/c2/play', () => rangeResponse())
	]);
	await settle();
	expect(element('.viewer__live-toggle')).not.toBeNull();
	unmount(app);
	flushSync();
});

test('a not-ready video shows the poster and waits for the SSE ready event', async () => {
	let videoReady = false;
	let streamReady = false;
	const { app } = mountViewer('v1', [
		get('/api/v1/assets/v1?locale=en', () =>
			jsonResponse(200, {
				...detailBodyOf(videoSource, {
					isReady: videoReady,
					width: videoReady ? 1920 : null,
					height: videoReady ? 1080 : null
				})
			})
		),
		get('/api/v1/assets/v1/play', () =>
			streamReady ? rangeResponse() : jsonResponse(404, { code: 'ASSET_NOT_READY' })
		)
	]);
	await settle();

	// The probe answered not-ready: the poster frame carries the
	// stage with the processing note, and no <video> mounts.
	const poster = element('img.viewer__poster');
	expect(poster.getAttribute('src')).toBe('/api/v1/assets/v1/thumbnail?size=view');
	expect(element('.viewer__processing').getAttribute('role')).toBe('status');
	expect(document.body.querySelector('video.viewer__video')).toBeNull();

	// The asset's own ready event lands: the detail refetch flips
	// isReady and the re-probe finds the stream.
	videoReady = true;
	streamReady = true;
	handlers[0]?.({ id: 9, name: 'asset.ready', assetId: 'v1', kind: 'video' });
	await settle();

	const video = document.body.querySelector<HTMLVideoElement>('video.viewer__video');
	expect(video?.getAttribute('src')).toBe('/api/v1/assets/v1/play');
	expect(video?.getAttribute('poster')).toBe('/api/v1/assets/v1/thumbnail?size=view');
	expect(video?.controls).toBe(true);
	expect(document.body.querySelector('img.viewer__poster')).toBeNull();
	unmount(app);
	flushSync();
});

test('the bar trash action flips optimistically, rolls back on failure', async () => {
	// The batch hangs until the test releases it, so the optimistic
	// window is observable before the failure. (The viewer rework: the
	// curation actions ride the viewer's action bar; the permanent
	// delete left the viewer for the trash page.)
	let release: ((response: Response) => void) | undefined;
	const batchAnswer = new Promise<Response>((resolve) => {
		release = resolve;
	});
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		post('/api/v1/assets/batch', () => batchAnswer)
	]);
	await settle();

	// Trash: the bar's icon flips to Restore immediately (the
	// optimistic switch on the local detail).
	click(barButton('Trash'));
	await new Promise((resolve) => setTimeout(resolve, 0));
	expect(() => barButton('Restore')).not.toThrow();

	// The drawer's trashed notice carries the flip's visible state.
	openDrawer();
	expect(element('.info').textContent).toContain('This photo is in the trash.');

	// The failed batch rolls both back and maps the error into the
	// bar's alert line.
	release?.(jsonResponse(502, { code: 'PROXY_TEAPOT' }));
	await settle();
	expect(() => barButton('Trash')).not.toThrow();
	expect(element('.info').textContent).not.toContain('This photo is in the trash');
	expect(element('.viewer__action-error').textContent).toBe('Something went wrong. Please try again.');
	unmount(app);
	flushSync();
});

test('a successful favorite batch flips the bar label (per-user state, no events)', async () => {
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		post('/api/v1/assets/batch', () => jsonResponse(200, { affected: 1, missing: [] }))
	]);
	await settle();

	click(barButton('Favorite'));
	await settle();

	// The exact single-id batch body.
	const init = requestOf(fetchMock, 'POST', '/api/v1/assets/batch');
	expect(JSON.parse(String(init?.body))).toEqual({ action: 'favorite', ids: ['a2'] });
	// The optimistic flag is the final state (favorites are silent).
	expect(() => barButton('Unfavorite')).not.toThrow();
	unmount(app);
	flushSync();
});

test('a trashed asset offers Restore in the bar and runs the restore batch', async () => {
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { isTrashed: true }))),
		post('/api/v1/assets/batch', () => jsonResponse(200, { affected: 1, missing: [] }))
	]);
	await settle();

	// The trash/restore pair is one toggle: a trashed asset's bar
	// action is the restore.
	click(barButton('Restore'));
	await settle();

	const init = requestOf(fetchMock, 'POST', '/api/v1/assets/batch');
	expect(JSON.parse(String(init?.body))).toEqual({ action: 'restore', ids: ['a2'] });
	// The optimistic flip returns the trash affordance.
	expect(() => barButton('Trash')).not.toThrow();
	unmount(app);
	flushSync();
});

test("a viewer's bar keeps the per-user favorite, never the owner-only trash; the drawer stays read-only for them", async () => {
	// The identity seam's other arm: the session's account is NOT the
	// asset's owner (the owner shared it to them) -- the bar keeps
	// the per-user favorite and never renders the owner-only trash;
	// the drawer keeps every read and the description text without
	// its edit affordance. The face chips stay: their assign dialog
	// carries the per-user label write a sharee holds.
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { ownerId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70' }))
		),
		post('/api/v1/assets/batch', () => jsonResponse(200, { affected: 1, missing: [] }))
	]);
	await settle();

	// The reads render: the stored description text, the file facts;
	// the download affordance lives in the ACTION BAR now (an action,
	// not metadata -- available to any viewer, owner or sharee).
	openDrawer();
	expect(element('.info__description').textContent).toBe('Original caption.');
	expect(() => element('a.viewer__download')).not.toThrow();
	expect(element('a.viewer__download').getAttribute('href')).toBe('/api/v1/assets/a2/original');

	// The owner-only writes never render (read-only is visual, never
	// grayed out -- the pinned read-only rule): no trash in the bar,
	// no edit pencil in the drawer, and the description is a PLAIN
	// line (a live edit control is an owner affordance alone).
	expect(() => barButton('Trash')).toThrow();
	expect(() => barButton('Edit date')).toThrow();
	expect(() => barButton('Edit location')).toThrow();
	const description = document.body.querySelector('.info__description');
	expect(description?.tagName).toBe('P');
	expect(() => barButton('Favorite')).not.toThrow();

	// The per-user favorite still works for the viewer.
	click(barButton('Favorite'));
	await settle();
	const init = requestOf(fetchMock, 'POST', '/api/v1/assets/batch');
	expect(JSON.parse(String(init?.body))).toEqual({ action: 'favorite', ids: ['a2'] });
	unmount(app);
	flushSync();
});

test("the sync album's own photos ARE deletable; only the read-only directory's are not", async () => {
	// The owner's settled semantics (2026-11-16): the sync card's
	// blobs are the application's own -- its photos ARE deletable
	// (the card itself is not, and never was). The read-only
	// directory source is the one arm with no delete: its photo is
	// the scan's verdict alone, the source cannot take the
	// application's writes.
	const syncOnly = [{ id: 'sync-1', title: 'Sync', type: 'sync', callerRole: 'owner', sourceWritable: true }];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { albums: syncOnly }))),
		post('/api/v1/assets/batch', () => jsonResponse(200, { affected: 1, missing: [] }))
	]);
	await settle();

	expect(() => barButton('Favorite')).not.toThrow();
	expect(() => barButton('Trash')).not.toThrow();
	expect(() => barButton('Restore')).toThrow();
	unmount(app);
	flushSync();

	// The read-only directory: the only home's source cannot take
	// writes, so no delete affordance renders -- never a grayed one.
	const readOnlyDirectory = [
		{ id: 'ro-1', title: 'Family vault', type: 'directory', callerRole: 'owner', sourceWritable: false }
	];
	const { app: vault } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { albums: readOnlyDirectory }))
		),
		post('/api/v1/assets/batch', () => jsonResponse(200, { affected: 1, missing: [] }))
	]);
	await settle();
	expect(() => barButton('Favorite')).not.toThrow();
	expect(() => barButton('Trash')).toThrow();
	unmount(vault);
	flushSync();

	// The same photo in the trash: the restore arm renders (the
	// album's read-only verdict governs the delete, never the way
	// back).
	const trashed = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { albums: syncOnly, isTrashed: true }))
		),
		post('/api/v1/assets/batch', () => jsonResponse(200, { affected: 1, missing: [] }))
	]);
	await settle();
	expect(() => barButton('Restore')).not.toThrow();
	expect(() => barButton('Trash')).toThrow();
	unmount(trashed.app);
	flushSync();
});

test('the action bar carries the curation icon actions; the permanent delete is gone', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// The action bar: the close mark, the favorite toggle, the info-panel
	// toggle and (for the owner) the trash/restore pair, every button
	// an icon button named by its aria-label. The zoom
	// controls are the in/out/reset trio.
	expect(() => barButton('Back to the timeline')).not.toThrow();
	expect(() => barButton('Favorite')).not.toThrow();
	expect(() => barButton('Photo information')).not.toThrow();
	expect(() => barButton('Trash')).not.toThrow();
	expect(() => barButton('Zoom in')).not.toThrow();
	expect(() => barButton('Zoom out')).not.toThrow();
	expect(() => barButton('Reset zoom')).not.toThrow();

	// The permanent delete left the viewer entirely: no confirm
	// dialog, no affordance (it lives on the trash page alone).
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(document.body.textContent).not.toContain('Delete permanently');

	// The floating bar: no left-shift guard any more (the panel is a
	// static sibling from 640 up and a bottom drawer below -- neither
	// covers the top bar), and no filename row in the middle.
	const actions = () => document.body.querySelector('.viewer__bar-actions');
	click(barButton('Photo information'));
	expect(actions()?.className).toContain('ms-auto');
	expect(element('.viewer__bar').querySelector('.viewer__title')).toBeNull();
	expect(element('.viewer__bar').querySelector('p')).toBeNull();
	click(barButton('Photo information'));
	unmount(app);
	flushSync();
});

test('the info panel docks as a right split region from 640 and falls back to the bottom drawer below it', async () => {
	// The split composition: the viewer root is a flex ROW -- the media
	// column flexes (flex-1 min-w-0, the bar and the stage live inside
	// it) and the panel rides BESIDE it as a real flex sibling
	// (sm:static: it occupies layout space, so the stage re-fits
	// through the flow instead of being covered). Below 640 the same
	// panel keeps the bottom-drawer form over its scrim.
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// The frame: the root dropped its column layout for the row; the
	// media column owns the column flow.
	const root = element('.viewer');
	expect(root.className).not.toContain('flex-col');
	const main = element('.viewer__main');
	expect(main.className).toContain('flex-col');
	expect(main.className).toContain('flex-1');
	expect(main.className).toContain('min-w-0');
	// The bar and the stage both render inside the media column (the
	// panel can never cover them at 640+).
	expect(main.contains(element('.viewer__bar'))).toBe(true);
	expect(main.contains(element('.viewer__stage'))).toBe(true);

	openDrawer();
	const panel = element('.info');
	// The panel is INSIDE the viewer's row (a flex sibling of the
	// media column), never a page-level overlay.
	expect(root.contains(panel)).toBe(true);
	// The docked form: static in the row at sm+, the pinned fixed
	// 360px, no floating shadow. The drawer form (fixed to the bottom
	// edge, full width, shadowed) rides the same element below sm.
	expect(panel.className).toContain('sm:static');
	expect(panel.className).toContain('sm:shrink-0');
	expect(panel.className).toContain('sm:shadow-none');
	expect(panel.className).toContain('sm:w-[360px]');
	expect(panel.className).toContain('fixed');
	expect(panel.className).toContain('shadow-overlay');
	expect(panel.className).toContain('sm:border-l');
	expect(panel.className).toContain('rounded-t-overlay');
	// The zoom var the media box transform rides (the engine drives
	// it per frame; 1 at rest).
	expect(element('.viewer__stage').style.getPropertyValue('--viewer-zoom')).toBe('1');
	unmount(app);
	flushSync();
});

test('a media failure shows the will-not-open card and keeps the panel honest', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	openDrawer();

	// The record answered; the bytes did not (a deleted file, a
	// vanished disk).
	element('img.viewer__image').dispatchEvent(new Event('error'));
	flushSync();

	const card = element('[data-testid="viewer-stage-error"]');
	expect(card).not.toBeNull();
	expect(card.getAttribute('role')).toBeNull();
	expect(card.querySelector('[role="alert"]')).not.toBeNull();
	expect(card.textContent).toContain('This photo will not open');
	expect(card.textContent).toContain('It may have been deleted, or the file is gone from the disk.');

	// The panel keeps the record: the metadata outlives the file.
	expect(document.body.querySelector('.info__filename')?.textContent).toContain('IMG_2.jpg');
	expect(document.body.querySelector('[data-testid="viewer-info-state"]')).toBeNull();

	// The retry re-requests the bytes under a fresh URL, and the stage
	// comes back.
	click(buttonByText('Try again'));
	expect(document.body.querySelector('[data-testid="viewer-stage-error"]')).toBeNull();
	expect(element('img.viewer__image').getAttribute('src')).toBe('/api/v1/assets/a2/thumbnail?size=view&retry=1');

	unmount(app);
	flushSync();
});

test('the open panel carries the load state before the record lands', async () => {
	let release = (): void => undefined;
	const held = new Promise<void>((resolve) => {
		release = resolve;
	});
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL) => {
			if (String(input) === '/api/v1/assets/a2?locale=en') {
				await held;
				return jsonResponse(200, detailBodyOf(sequence[1]!));
			}
			throw new Error(`unexpected fetch ${String(input)}`);
		})
	);
	pageState.params.id = 'a2';
	localStorage.setItem('lumemor:viewer:info', 'true');
	const app = mount(Page, { target: document.body });
	flushSync();
	await tick();

	// The panel is open while the record is in flight: its place says
	// so instead of rendering a silent blank column.
	const state = element('[data-testid="viewer-info-state"]');
	expect(state.textContent).toContain('Loading the details…');
	expect(state.querySelector('[role="status"]')).not.toBeNull();

	release();
	await settle();
	// The record landed: the state face gives way to the real panel.
	expect(document.body.querySelector('[data-testid="viewer-info-state"]')).toBeNull();
	expect(document.body.querySelector('.info__filename')?.textContent).toContain('IMG_2.jpg');

	unmount(app);
	flushSync();
});

test('a failed record load keeps the open panel honest with its own retry', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL) => {
			if (String(input) === '/api/v1/assets/a2?locale=en') {
				return jsonResponse(500, { code: 'INTERNAL' });
			}
			throw new Error(`unexpected fetch ${String(input)}`);
		})
	);
	pageState.params.id = 'a2';
	localStorage.setItem('lumemor:viewer:info', 'true');
	const app = mount(Page, { target: document.body });
	flushSync();
	await settle();

	// The stage says it will not open; the panel says what it knows
	// (nothing yet) and offers the same retry -- never a blank column.
	expect(element('[data-testid="viewer-stage-error"]')).not.toBeNull();
	const state = element('[data-testid="viewer-info-state"]');
	expect(state.textContent).toContain('The details could not be loaded.');
	expect(state.querySelector('[role="alert"]')).not.toBeNull();
	const retry = Array.from(state.querySelectorAll('button')).find((button) =>
		button.textContent?.includes('Try again')
	);
	expect(retry).toBeDefined();

	unmount(app);
	flushSync();
});

test('the info panel open preference persists across viewer sessions', async () => {
	// One localStorage key, default closed; the stored open
	// state restores on the next viewer mount, and closing persists
	// back.
	const first = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	expect(document.body.querySelector('.info')).toBeNull();

	openDrawer();
	expect(localStorage.getItem('lumemor:viewer:info')).toBe('true');
	unmount(first.app);
	flushSync();

	// A fresh viewer session restores the open panel without a click.
	const second = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	expect(document.body.querySelector('.info')).not.toBeNull();

	// Closing it persists the closed preference for the next session.
	click(barButton('Photo information'));
	expect(localStorage.getItem('lumemor:viewer:info')).toBe('false');
	unmount(second.app);
	flushSync();

	const third = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	expect(document.body.querySelector('.info')).toBeNull();
	unmount(third.app);
	flushSync();
});

test('the +/- keys and the double-click ride the gradual zoom; a neighbor resets it', async () => {
	// The gradual zoom: the keyboard pair steps the same engine the bar's
	// buttons and the stage's double-click ride -- every change
	// animates, and the affordances key on the engine's target (the
	// zoomed stage flips synchronously with the intent). Navigating
	// to a neighbor snaps back to the resting scale.
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		get('/api/v1/assets/a1?locale=en', () => jsonResponse(200, detailBodyOf(sequence[0]!)))
	]);
	await settle();
	const stage = () => element('.viewer__stage');

	// The zoomed state is the zoomed class + the cursor contract.
	expect(stage().classList.contains('viewer__stage--zoomed')).toBe(false);
	pressKey('+');
	expect(stage().classList.contains('viewer__stage--zoomed')).toBe(true);
	pressKey('-');
	expect(stage().classList.contains('viewer__stage--zoomed')).toBe(false);

	// The double-click binary: on at rest, off when zoomed.
	stage().dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
	flushSync();
	expect(stage().classList.contains('viewer__stage--zoomed')).toBe(true);
	stage().dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
	flushSync();
	expect(stage().classList.contains('viewer__stage--zoomed')).toBe(false);

	// The zoomed state holds across the same-id replacement (the
	// per-asset reset only fires on an actual asset change -- the
	// stage-survival test) and the walk replaces the route.
	pressKey('+');
	handlers[0]?.({ id: 40, name: 'asset.updated', assetId: 'a2', kind: 'image' });
	await settle();
	expect(stage().classList.contains('viewer__stage--zoomed')).toBe(true);
	pressKey('ArrowLeft');
	expect(goto).toHaveBeenCalledWith('/a/a1', { replaceState: true });
	unmount(app);
	flushSync();

	// A fresh neighbor starts unzoomed: the engine's hard reset ran on
	// the asset change.
	const neighbor = mountViewer('a1', [
		get('/api/v1/assets/a1?locale=en', () => jsonResponse(200, detailBodyOf(sequence[0]!)))
	]);
	await settle();
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(false);
	unmount(neighbor.app);
	flushSync();
});

test('a playing motion owns the stage double-click, the still keeps it', async () => {
	const liveDetail = () => jsonResponse(200, detailBodyOf(sequence[1]!, { livePhotoVideoId: 'c2' }));
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', liveDetail),
		get('/api/v1/assets/c2/play', () => rangeResponse())
	]);
	await settle();

	// While the motion plays, a double-click on the stage is the
	// viewer double-taking at the movement, not a zoom request.
	click(element('.viewer__live-toggle'));
	await settle();
	const playing = document.body.querySelector<HTMLVideoElement>('video.viewer__live');
	playing?.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
	flushSync();
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(false);

	// Stopped, the still's double-click is the zoom toggle again.
	click(element('.viewer__live-toggle'));
	flushSync();
	element('.viewer__image').dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
	flushSync();
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(true);
	unmount(app);
	flushSync();
});

test('the info panel is the four sections: icon rows, no actions, no key-value columns', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	openDrawer();

	// No actions section survived the rework: the curation
	// affordances live in the bar.
	expect(element('.info').textContent).not.toContain('Actions');

	// The info rows: one line per fact (icon + content), one
	// hairline between the rows, no key-value columns.
	const facts = element('.info__facts');
	expect(facts.className).toContain('divide-y');
	expect(facts.querySelector('dt')).toBeNull();
	expect(facts.querySelector('dd')).toBeNull();
	// The icon-first row language: every fact row leads with a
	// decorative glyph.
	expect(facts.querySelector('li')?.querySelector('svg')).not.toBeNull();
	unmount(app);
	flushSync();
});

test('the layered Escape closes the assign dialog, then the drawer, then leaves', async () => {
	const back = vi.spyOn(history, 'back').mockImplementation(() => undefined);
	// The modal layer is the page's face-assign dialog (the delete
	// dialog is gone with the permanent delete's departure).
	const faces = [faceSummary('f2')];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('c1', 'Ada')])),
		get('/api/v1/people/unnamed?limit=60', () => jsonResponse(200, { items: [], totalCount: 0 }))
	]);
	await settle();

	openDrawer();
	expect(document.body.querySelector('.info')).not.toBeNull();

	// A face chip opens the assign dialog: Escape with it open closes
	// only the dialog (the viewer defers to the open modal).
	click(element('[data-face-id="f2"]'));
	await settle();
	expect(element('[role="dialog"]')).not.toBeNull();
	pressKey('Escape');
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(document.body.querySelector('.info')).not.toBeNull();
	expect(back).not.toHaveBeenCalled();

	// The drawer is still open underneath: the next Escape closes
	// the drawer, not the viewer.
	pressKey('Escape');
	expect(document.body.querySelector('.info')).toBeNull();
	expect(back).not.toHaveBeenCalled();

	// Nothing left to close: the viewer itself leaves.
	pressKey('Escape');
	expect(back).toHaveBeenCalledTimes(1);
	unmount(app);
	flushSync();
});

test('the i key toggles the drawer and typing never reaches the viewer keys', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	pressKey('i');
	expect(document.body.querySelector('.info')).not.toBeNull();
	pressKey('i');
	expect(document.body.querySelector('.info')).toBeNull();

	// Typing in the description editor: the arrows stay in the field.
	pressKey('i');
	openDescriptionEdit();
	const field = document.body.querySelector<HTMLInputElement>('input.info__editor');
	if (field === null) {
		throw new Error('missing editor');
	}
	pressKey('ArrowLeft', field);
	expect(vi.mocked(goto)).not.toHaveBeenCalled();
	unmount(app);
	flushSync();
});

test('a failed detail load shows the mapped error with a working retry', async () => {
	let failing = true;
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			failing ? jsonResponse(500, { code: 'PROXY_TEAPOT' }) : jsonResponse(200, detailBodyOf(sequence[1]!))
		)
	]);
	await settle();

	const failure = element('[data-testid="viewer-stage-error"] [role="alert"]');
	expect(failure.getAttribute('role')).toBe('alert');
	// The unknown code renders the generic message, never the code.
	expect(failure.textContent).toContain('Something went wrong. Please try again.');
	expect(failure.textContent).not.toContain('proxy_teapot');

	failing = false;
	click(buttonByText('Try again'));
	await settle();
	expect(element('img.viewer__image').getAttribute('src')).toBe('/api/v1/assets/a2/thumbnail?size=view');
	unmount(app);
	flushSync();
});

test('the stage keeps its loaded and zoomed state across same-id detail replacements', async () => {
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!)))
	]);
	await settle();

	// Both serving layers report their load (the crossfade target),
	// and the zoom is on.
	const images = Array.from(document.body.querySelectorAll<HTMLImageElement>('img.viewer__image'));
	images[0]?.dispatchEvent(new Event('load'));
	images[1]?.dispatchEvent(new Event('load'));
	flushSync();
	click(element('[aria-label="Zoom in"]'));
	flushSync();
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(true);
	expect(images[0]?.classList.contains('viewer__image--loaded')).toBe(true);

	// The same-id replacement (the SSE convergence refetch, the PATCH
	// adoption's shape): the stage must survive -- the img element
	// persists with its loaded class, the zoom holds, and the serving
	// copies are never re-requested (nothing re-fires a load the
	// browser would not send).
	const detailCalls = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2?locale=en');
	handlers[0]?.({ id: 21, name: 'asset.updated', assetId: 'a2', kind: 'image' });
	await settle();
	const survived = Array.from(document.body.querySelectorAll<HTMLImageElement>('img.viewer__image'));
	expect(survived[0]).toBe(images[0]);
	expect(survived[0]?.classList.contains('viewer__image--loaded')).toBe(true);
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(true);
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2?locale=en')).toHaveLength(
		detailCalls.length + 1
	);
	unmount(app);
	flushSync();

	// The navigation half: a neighbor mounts fresh (the plain-page
	// mock cannot mutate route params in place, so the in-instance
	// reset is approximated by the neighbor's mount -- the same
	// approximation the drawer's reset test uses). The new asset
	// starts unloaded and unzoomed: the crossfade begins over.
	const neighbor = mountViewer('a3', [
		get('/api/v1/assets/a3?locale=en', () => jsonResponse(200, detailBodyOf(sequence[2]!)))
	]);
	await settle();
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(false);
	expect(
		Array.from(document.body.querySelectorAll('img.viewer__image')).every(
			(image) => !image.classList.contains('viewer__image--loaded')
		)
	).toBe(true);
	unmount(neighbor.app);
	flushSync();
});

test('a failed initial load re-seeds the favorite flag when the convergence refetch recovers', async () => {
	let failing = true;
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			failing ? jsonResponse(500, { code: 'PROXY_TEAPOT' }) : jsonResponse(200, detailBodyOf(sequence[1]!))
		)
	]);
	// The wall snapshot says the asset IS a favorite; the initial
	// load fails before the seeding can run.
	publishWallItems([item('a2', '2026-11-01T09:00:00Z', { isFavorite: true })]);
	await settle();
	expect(element('[data-testid="viewer-stage-error"]')).not.toBeNull();

	failing = false;
	handlers[0]?.({ id: 22, name: 'asset.updated', assetId: 'a2', kind: 'image' });
	await settle();

	// The recovered bar shows the favorite state (the Unfavorite
	// action), not the default.
	expect(() => barButton('Unfavorite')).not.toThrow();
	unmount(app);
	flushSync();
});

test('a deleted-from-under-the-viewer asset leaves through SSE', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	handlers[0]?.({ id: 11, name: 'asset.deleted', assetId: 'a2', kind: 'image' });
	await settle();
	expect(goto).toHaveBeenCalledWith('/timeline');
	unmount(app);
	flushSync();
});

test('SSE events of other assets are ignored (the wall converges on remount)', async () => {
	const { app, fetchMock } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	handlers[0]?.({ id: 12, name: 'asset.ready', assetId: 'a3', kind: 'image' });
	await settle();
	// No refetch of the open asset, no navigation.
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2?locale=en')).toHaveLength(1);
	expect(goto).not.toHaveBeenCalled();
	unmount(app);
	flushSync();
});

test('leaving the viewer unsubscribes from the event stream', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	expect(handlers).toHaveLength(1);

	unmount(app);
	flushSync();
	expect(handlers).toHaveLength(0);
});

test('a not-ready video carries the stage on the placeholder when the poster request fails', async () => {
	const { app } = mountViewer('v1', [
		get('/api/v1/assets/v1?locale=en', () => jsonResponse(200, detailBodyOf(videoSource, { isReady: false }))),
		get('/api/v1/assets/v1/play', () => jsonResponse(404, { code: 'ASSET_NOT_READY' }))
	]);
	await settle();

	// The placeholder layer renders behind the poster frame (the
	// image branch's pattern), so a failed request never leaves a
	// broken image alone on the stage.
	const poster = element('img.viewer__poster');
	expect(poster.getAttribute('src')).toBe('/api/v1/assets/v1/thumbnail?size=view');
	expect(document.body.querySelector('.viewer__placeholder--empty')).not.toBeNull();

	// The thumbnail job has not run: the request answers
	// ASSET_NOT_READY and the error drops the poster img; the
	// skeleton placeholder carries the stage with the note.
	poster.dispatchEvent(new Event('error'));
	flushSync();
	expect(document.body.querySelector('img.viewer__poster')).toBeNull();
	expect(document.body.querySelector('.viewer__placeholder--empty')).not.toBeNull();
	expect(element('.viewer__processing').getAttribute('role')).toBe('status');
	unmount(app);
	flushSync();
});

test('filmstrip thumbnails only fire for ready entries; the session ready event recovers one', async () => {
	// The wall snapshot lands with a3 still processing (the snapshot
	// is frozen while the viewer has the wall unmounted).
	publishWallItems([sequence[0]!, sequence[1]!, item('a3', '2026-10-13T12:30:00Z', { isReady: false })]);
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// No thumbnail request for the not-ready entry; the ready ones
	// fire (the tile's pinned gate).
	expect(document.body.querySelector('[data-filmstrip-id="a3"] img')).toBeNull();
	expect(document.body.querySelector('[data-filmstrip-id="a1"] img')?.getAttribute('src')).toBe(
		'/api/v1/assets/a1/thumbnail?size=tile'
	);
	expect(document.body.querySelector('[data-filmstrip-id="a2"] img')?.getAttribute('src')).toBe(
		'/api/v1/assets/a2/thumbnail?size=tile'
	);

	// The session's ready event overlays the frozen snapshot: the
	// tile's request fires without leaving the viewer (and the other
	// assets' events still never refetch the open one).
	handlers[0]?.({ id: 13, name: 'asset.ready', assetId: 'a3', kind: 'image' });
	await settle();
	const recovered = document.body.querySelector<HTMLImageElement>('[data-filmstrip-id="a3"] img');
	expect(recovered?.getAttribute('src')).toBe('/api/v1/assets/a3/thumbnail?size=tile');
	unmount(app);
	flushSync();
});

test('an in-progress description draft survives same-id detail replacements', async () => {
	// The batch hangs until the test releases it, so the optimistic
	// window is observable.
	let release: ((response: Response) => void) | undefined;
	const batchAnswer = new Promise<Response>((resolve) => {
		release = resolve;
	});
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		post('/api/v1/assets/batch', () => batchAnswer)
	]);
	await settle();

	openDrawer();
	openDescriptionEdit();
	fillTextarea('input.info__editor', 'My work in progress.');

	// The optimistic trash flip replaces the detail object (same
	// id): the flip is observable (the trashed banner) and the draft
	// and the edit mode survive the replacement.
	click(barButton('Trash'));
	await new Promise((resolve) => setTimeout(resolve, 0));
	expect(element('.info').textContent).toContain('This photo is in the trash.');
	expect(document.body.querySelector<HTMLInputElement>('input.info__editor')?.value).toBe('My work in progress.');

	// The SSE convergence refetch replaces it again (same id): the
	// draft survives that replacement too.
	handlers[0]?.({ id: 14, name: 'asset.updated', assetId: 'a2', kind: 'image' });
	await settle();
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2?locale=en')).toHaveLength(2);
	expect(document.body.querySelector<HTMLInputElement>('input.info__editor')?.value).toBe('My work in progress.');

	release?.(jsonResponse(200, { affected: 1, missing: [] }));
	await settle();
	unmount(app);
	flushSync();

	// An actual asset change (the panel-open navigation to the
	// neighbor) resets the edit mode: the neighbor's panel RESTORED
	// OPEN (the preference persists across viewer sessions, and
	// this test's first half left it open), so no toggle click is
	// needed -- the read view renders and the draft never leaks
	// across assets.
	const neighbor = mountViewer('a1', [
		get('/api/v1/assets/a1?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[0]!, { description: 'The neighbor caption.' }))
		)
	]);
	await settle();
	expect(document.body.querySelector('input.info__editor')).toBeNull();
	expect(element('.info__description').textContent).toBe('The neighbor caption.');
	unmount(neighbor.app);
	flushSync();
});

test('concurrent convergence refetches apply newest-wins: a stale response never lands', async () => {
	const pending: Array<(response: Response) => void> = [];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => new Promise<Response>((resolve) => pending.push(resolve)))
	]);
	await settle();

	// The initial load lands first.
	pending[0]?.(jsonResponse(200, detailBodyOf(sequence[1]!)));
	await settle();

	// Two events in quick succession fire two concurrent refetches.
	handlers[0]?.({ id: 20, name: 'asset.updated', assetId: 'a2', kind: 'image' });
	handlers[0]?.({ id: 21, name: 'asset.updated', assetId: 'a2', kind: 'image' });
	await new Promise((resolve) => setTimeout(resolve, 0));
	expect(pending).toHaveLength(3);

	// The newer trigger's response lands first and applies...
	pending[2]?.(jsonResponse(200, detailBodyOf(sequence[1]!, { description: 'The newer state.' })));
	await settle();
	openDrawer();
	expect(element('.info__description').textContent).toBe('The newer state.');

	// ...then the slower stale one resolves: it must not overwrite
	// the newer state.
	pending[1]?.(jsonResponse(200, detailBodyOf(sequence[1]!, { description: 'The stale state.' })));
	await settle();
	expect(element('.info__description').textContent).toBe('The newer state.');
	unmount(app);
	flushSync();
});

test('a successful convergence refetch recovers a failed initial load', async () => {
	let failing = true;
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			failing ? jsonResponse(500, { code: 'PROXY_TEAPOT' }) : jsonResponse(200, detailBodyOf(sequence[1]!))
		)
	]);
	await settle();
	// The initial load failed: the error branch renders.
	expect(element('[data-testid="viewer-stage-error"] [role="alert"]').getAttribute('role')).toBe('alert');

	// The open asset's ready event lands the convergence refetch:
	// the success flips the route out of the error branch and the
	// stage renders.
	failing = false;
	handlers[0]?.({ id: 22, name: 'asset.ready', assetId: 'a2', kind: 'image' });
	await settle();
	expect(document.body.querySelector('[data-testid="viewer-stage-error"]')).toBeNull();
	expect(element('img.viewer__image').getAttribute('src')).toBe('/api/v1/assets/a2/thumbnail?size=view');
	unmount(app);
	flushSync();
});

test('Esc falls back to the timeline when no in-app history sits behind a deep link', async () => {
	const back = vi.spyOn(history, 'back').mockImplementation(() => undefined);
	// A fresh-document deep link: the mounting navigation is 'enter'.
	navigationState.mountType = 'enter';
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// A replaceState walk (the arrows / the filmstrip) adds no
	// history entry: the deep link still has nothing of the app
	// behind it, so back would exit the site.
	navigate('goto');
	pressKey('Escape');
	expect(goto).toHaveBeenCalledWith('/timeline');
	expect(back).not.toHaveBeenCalled();
	unmount(app);
	flushSync();
	vi.mocked(goto).mockClear();

	// From the wall (a link push) the in-app history exists, and a
	// replaceState walk does not consume it: Esc stays history.back.
	navigationState.mountType = 'link';
	const wall = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	navigate('goto');
	pressKey('Escape');
	expect(back).toHaveBeenCalledTimes(1);
	expect(goto).not.toHaveBeenCalledWith('/timeline');
	unmount(wall.app);
	flushSync();
});

test('a zoomed Live Photo keeps the motion layer inside the zoomed stage', async () => {
	const liveDetail = () => jsonResponse(200, detailBodyOf(sequence[1]!, { livePhotoVideoId: 'c2' }));
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', liveDetail),
		get('/api/v1/assets/c2/play', () => rangeResponse())
	]);
	await settle();

	// Zoom in, then play the Live Photo: the motion layer mounts as
	// a child of the zoomed stage (the transform selector's DOM
	// contract -- .viewer__stage--zoomed .viewer__live -- so the
	// motion scales with the still, never pops unscaled).
	click(element('[aria-label="Zoom in"]'));
	click(element('.viewer__live-toggle'));
	await settle();

	const stage = element('.viewer__stage');
	expect(stage.classList.contains('viewer__stage--zoomed')).toBe(true);
	const live = document.body.querySelector<HTMLVideoElement>('video.viewer__live');
	expect(live).not.toBeNull();
	expect(stage.contains(live!)).toBe(true);
	unmount(app);
	flushSync();
});

test('the drawer renders one chip per face with the caller-visible label states', async () => {
	const faces = [
		faceSummary('f1', { label: { name: 'Ada' } }),
		faceSummary('f2'),
		faceSummary('f3', { clusterId: null })
	];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces })))
	]);
	await settle();
	openDrawer();

	const chips = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.info__face'));
	expect(chips).toHaveLength(3);
	expect(chips[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/faces/f1/thumbnail');
	// The caller's label, the unnamed state, the not-grouped state.
	expect(chips.map((chip) => chip.querySelector('.info__face-name')?.textContent)).toEqual([
		'Ada',
		'Unnamed',
		'Not grouped yet'
	]);
	unmount(app);
	flushSync();

	// A faceless asset: the honest empty line, never a bare heading.
	// (The panel restored open -- the persistence, this test's own
	// first half left it open -- so no toggle click precedes the
	// assertion.)
	const empty = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: [] })))
	]);
	await settle();
	expect(document.body.querySelector('.info__face')).toBeNull();
	expect(element('.info__faces-empty').textContent).toBe('No people recognized in this photo.');
	unmount(empty.app);
	flushSync();
});

test('a named chip carries the capture-time age when the person has a birth date', async () => {
	// The detail's takenAt is 2026-11-01; the person was born
	// 1992-03-15, so the photo catches them at 34 years 7 months (the
	// person_age_year_months arm). The unnamed and not-grouped chips
	// carry no age -- there is no person, hence no birth date.
	const faces = [
		faceSummary('f1', { label: { name: 'Ada' }, birthDate: '1992-03-15' }),
		faceSummary('f2', { birthDate: '1992-03-15' }),
		faceSummary('f3', { clusterId: null })
	];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces })))
	]);
	await settle();
	openDrawer();

	const chips = Array.from(document.body.querySelectorAll<HTMLElement>('.info__face'));
	expect(chips).toHaveLength(3);
	expect(chips[0]?.querySelector('.info__face-name')?.textContent).toBe('Ada');
	expect(chips[0]?.querySelector('.info__face-age')?.textContent).toBe('34 years 7 months');
	// An unlabeled face of a person with a birth date shows no age:
	// the chip does not know whose it is.
	expect(chips[1]?.querySelector('.info__face-age')).toBeNull();
	expect(chips[2]?.querySelector('.info__face-age')).toBeNull();
	unmount(app);
	flushSync();
});

test("the drawer's browsable rows deep-link: albums, tags, named faces; the place row stays plain", async () => {
	// The approved L3 face: the rows that name a browsable context
	// render as links (the icon-first rule's leading glyph rides the
	// row types), the drawer stays read-only otherwise. The location
	// row is the one deliberate exception: the place name is PLAIN
	// text (the design's pin -- the geographic name does not carry
	// the reader away), and the album links open in a NEW TAB (the
	// open photo survives the trip).
	const detail = detailBodyOf(sequence[1]!, {
		exif: { cameraMake: 'Apple', cameraModel: 'iPhone 15' },
		tags: [{ id: 't1', name: 'Beach' }],
		albums: [
			{ id: 'al-1', title: 'Trip', type: 'directory', callerRole: 'owner', sourceWritable: true },
			{ id: 'mem-1', title: 'Paris trip', type: 'memory', callerRole: 'owner', sourceWritable: true },
			{ id: 'al-2', title: 'Sync', type: 'sync', callerRole: 'owner', sourceWritable: true }
		],
		place: { placeId: 2988507, name: 'Paris' },
		location: { lat: 48.8584, lon: 2.2945 },
		faces: [faceSummary('f1', { label: { name: 'Ada' } })]
	});
	const { app } = mountViewer('a2', [get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detail))]);
	await settle();
	openDrawer();

	// The albums row: one link per visible album, the sync card's
	// localized display name keyed by the type -- every link a NEW
	// TAB (target="_blank" rel="noopener").
	const albumLinks = Array.from(document.body.querySelectorAll<HTMLAnchorElement>('.info__album'));
	expect(albumLinks.map((link) => link.getAttribute('href'))).toEqual([
		'/albums/al-1',
		'/memories?memory=mem-1',
		'/albums/al-2'
	]);
	expect(albumLinks.map((link) => link.getAttribute('target'))).toEqual(['_blank', '_blank', '_blank']);
	expect(albumLinks.map((link) => link.getAttribute('rel'))).toEqual(['noopener', 'noopener', 'noopener']);
	expect(albumLinks[1]?.textContent?.trim()).toBe('Paris trip');
	expect(albumLinks[2]?.textContent?.trim()).toBe('My photos');

	// The tag chips: deep links into the tag-filtered browse, each
	// beside its own remove button (the × never nests in the link).
	const tagLinks = Array.from(document.body.querySelectorAll<HTMLAnchorElement>('.info__tag-chip a'));
	expect(tagLinks.map((link) => link.getAttribute('href'))).toEqual(['/search?tagIds=t1']);
	expect(tagLinks[0]?.textContent?.trim()).toBe('Beach');
	expect(tagLinks[0]?.closest('.info__tag-chip')?.querySelector('button')).not.toBeNull();

	// The camera line: one plain line (no deep link any more -- the
	// fact row reads, it does not navigate).
	expect(element('.info__facts').textContent).toContain('Apple iPhone 15');

	// The named face: the person page link; the unnamed arm keeps the
	// assign flow (its own test).
	const faceLink = document.body.querySelector<HTMLAnchorElement>('[data-face-id="f1"]');
	expect(faceLink?.getAttribute('href')).toBe('/people/c-f1');

	// The resolved place: plain text, never a link.
	expect(document.body.querySelector('.info__place')).toBeNull();
	expect(element('[data-testid="viewer-location"]').textContent).toContain('Paris');
	expect(document.body.querySelector('[data-testid="viewer-location"]')?.querySelector('a')).toBeNull();

	unmount(app);
	flushSync();
});

test('the assign flow moves a face onto an existing person and the chip converges', async () => {
	// The server's current answer: f2 sits on an unnamed cluster; the
	// move lands it on Ada's cluster and the refetch carries the label.
	let moved = false;
	const facesNow = () => [
		faceSummary('f1', { label: { name: 'Ada' } }),
		moved ? faceSummary('f2', { clusterId: 'c1', label: { name: 'Ada' } }) : faceSummary('f2')
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('c1', 'Ada')])),
		get('/api/v1/people/unnamed?limit=60', () =>
			jsonResponse(200, { items: [unnamedTarget('c9', 4)], totalCount: 1 })
		),
		post('/api/v1/faces/f2/move', () => {
			moved = true;
			return jsonResponse(200, { faceId: 'f2', clusterId: 'c1' });
		})
	]);
	await settle();
	openDrawer();
	click(element('[data-face-id="f2"]'));
	await settle();

	// The assign dialog opened and fetched both target lists.
	expect(element('[role="dialog"]').getAttribute('aria-label')).toBe('Assign person');
	const select = document.body.querySelector<HTMLSelectElement>('#cluster-picker-target');
	expect(select).not.toBeNull();
	// The face's own cluster (c-f2) is never a target; the named
	// people and the unnamed clusters are.
	expect(Array.from(select!.options).map((option) => option.value)).toEqual(['c1', 'c9']);
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people')).toHaveLength(1);

	click(buttonByText('Assign'));
	await settle();

	// The exact move body of the contract.
	const init = requestOf(fetchMock, 'POST', '/api/v1/faces/f2/move');
	expect(JSON.parse(String(init?.body))).toEqual({ targetClusterId: 'c1' });
	// The dialog closed and the chip reflects the target cluster's label.
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(Array.from(document.body.querySelectorAll('.info__face-name')).map((node) => node.textContent)).toEqual([
		'Ada',
		'Ada'
	]);
	unmount(app);
	flushSync();
});

test('a move on a not-yet-grouped face lands on the picked person', async () => {
	// The clusterless chip's assign flow is a move like any other:
	// the endpoint takes a clusterless source (no source cluster to
	// maintain), so the write lands and the chip converges to the
	// target's label.
	let moved = false;
	const facesNow = () => [
		moved ? faceSummary('f3', { clusterId: 'c1', label: { name: 'Ada' } }) : faceSummary('f3', { clusterId: null })
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('c1', 'Ada')])),
		get('/api/v1/people/unnamed?limit=60', () => jsonResponse(200, { items: [], nextCursor: null, totalCount: 0 })),
		post('/api/v1/faces/f3/move', () => {
			moved = true;
			return jsonResponse(200, { faceId: 'f3', clusterId: 'c1' });
		})
	]);
	await settle();
	openDrawer();
	click(element('[data-face-id="f3"]'));
	await settle();
	// The clusterless chip offers no new-person path (there is no
	// cluster to name).
	expect(document.body.querySelector('input[name="cluster-picker-mode"]')).toBeNull();
	click(buttonByText('Assign'));
	await settle();

	// The move writes the contract's body with no refusal: the
	// dialog closes and the chip carries the target's label.
	const init = requestOf(fetchMock, 'POST', '/api/v1/faces/f3/move');
	expect(JSON.parse(String(init?.body))).toEqual({ targetClusterId: 'c1' });
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(document.body.querySelector('.cluster-picker__failure')).toBeNull();
	expect(document.body.querySelectorAll('.info__face-name')[0]?.textContent).toBe('Ada');
	unmount(app);
	flushSync();
});

test('the assign dialog with no people at all says there are no people yet', async () => {
	// The assign dialog carries no filter: its empty state is the
	// honest no-people line, never the filtered list's no-match copy.
	const faces = [faceSummary('f3', { clusterId: null })];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces }))),
		get('/api/v1/people', () => jsonResponse(200, [])),
		get('/api/v1/people/unnamed?limit=60', () => jsonResponse(200, { items: [], nextCursor: null, totalCount: 0 }))
	]);
	await settle();
	openDrawer();
	click(element('[data-face-id="f3"]'));
	await settle();
	click(buttonByText('Assign'));
	await settle();

	const empty = document.body.querySelector('.cluster-picker__empty');
	expect(empty?.textContent?.trim()).toBe('No people yet.');
	expect(element('[role="dialog"]').textContent).not.toContain('No people match.');
	unmount(app);
	flushSync();
});

test('a named face chip deep-links into the person page; an unnamed chip names its cluster as a new person', async () => {
	let named = false;
	const facesNow = () => [
		faceSummary('f1', { label: { name: 'Ada' } }),
		named ? faceSummary('f2', { label: { name: 'Grace' } }) : faceSummary('f2')
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		get('/api/v1/people', () => jsonResponse(200, [])),
		get('/api/v1/people/unnamed?limit=60', () =>
			jsonResponse(200, { items: [unnamedTarget('c9', 4)], totalCount: 1 })
		),
		put('/api/v1/people/c-f2/label', () => {
			named = true;
			return jsonResponse(200, { clusterId: 'c-f2', name: 'Grace' });
		})
	]);
	await settle();
	openDrawer();

	// The labeled chip: a LINK into the person's page (the naming is
	// done, the page owns the rename) -- it never opens the assign
	// dialog, so no new-person mode exists for it at all.
	const namedChip = document.body.querySelector<HTMLAnchorElement>('[data-face-id="f1"]');
	expect(namedChip?.tagName).toBe('A');
	expect(namedChip?.getAttribute('href')).toBe('/people/c-f1');
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();

	// The unnamed chip: the new-person path with the family note.
	click(element('[data-face-id="f2"]'));
	await settle();
	const radios = Array.from(document.body.querySelectorAll<HTMLInputElement>('input[name="cluster-picker-mode"]'));
	expect(radios.map((radio) => radio.value)).toEqual(['existing', 'new']);
	click(radios[1]!);
	radios[1]!.dispatchEvent(new Event('change', { bubbles: true }));
	flushSync();
	const field = document.body.querySelector<HTMLInputElement>('input[name="new-person-name"]');
	expect(field).not.toBeNull();
	expect(element('.cluster-picker__note').textContent).toContain('shares the name');
	fillInput(field!, 'Grace');
	click(buttonByText('Save name'));
	await settle();

	// The exact label body on the face's CURRENT cluster (naming it,
	// not moving the face).
	const init = requestOf(fetchMock, 'PUT', '/api/v1/people/c-f2/label');
	expect(JSON.parse(String(init?.body))).toEqual({ name: 'Grace' });
	// The dialog closed and both chips carry their names.
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(Array.from(document.body.querySelectorAll('.info__face-name')).map((node) => node.textContent)).toEqual([
		'Ada',
		'Grace'
	]);
	unmount(app);
	flushSync();
});

test('person events converge the chips through the refetch; unrelated events add no fetch', async () => {
	let labeled = false;
	const facesNow = () => [faceSummary('f2', labeled ? { label: { name: 'Ada' } } : {})];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() })))
	]);
	await settle();
	openDrawer();
	expect(element('.info__face-name').textContent).toBe('Unnamed');
	const detailCalls = () =>
		fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2?locale=en').length;
	const before = detailCalls();

	// A person event off every rendered cluster and face: no refetch.
	firePersonEvent({ id: 30, name: 'person.labeled', clusterId: 'c-other', label: 'X', userId: 'u1' });
	await settle();
	expect(detailCalls()).toBe(before);

	// A label on a rendered cluster: the refetched answer decides the
	// chip (the event itself is only the trigger).
	labeled = true;
	firePersonEvent({ id: 31, name: 'person.labeled', clusterId: 'c-f2', label: 'Ada', userId: 'u1' });
	await settle();
	expect(element('.info__face-name').textContent).toBe('Ada');
	expect(detailCalls()).toBe(before + 1);

	// A face move touching a rendered face converges the same way.
	firePersonEvent({ id: 32, name: 'person.face_moved', faceId: 'f2', clusterId: 'c-f2', userId: 'u1' });
	await settle();
	expect(detailCalls()).toBe(before + 2);
	unmount(app);
	flushSync();
});

test('leaving the viewer unsubscribes from the person event stream', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	expect(personHandlers).toHaveLength(1);

	unmount(app);
	flushSync();
	expect(personHandlers).toHaveLength(0);
});

/** One entry of the detail's tags array (the contract's shape). */
function tagEntry(id: string, name: string): Record<string, unknown> {
	return { id, name };
}

/** One tag of the add-tag dialog's vocabulary list. */
function tagSummary(id: string, name: string): Record<string, unknown> {
	return {
		id,
		name,
		assetCount: 0,
		createdAt: '2026-11-06T09:00:00Z',
		updatedAt: '2026-11-06T09:00:00Z'
	};
}

/** One button by exact text, scoped INSIDE one dialog or panel (the
 * file's own buttonByText is document-wide and cannot be scoped). */
function scopedButtonByText(scope: ParentNode, text: string): HTMLButtonElement {
	const found = Array.from(scope.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.textContent?.trim() === text
	);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

test("the drawer's file section renders the tags row: the caller's chips, the empty note, the assign affordance", async () => {
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { tags: [tagEntry('t1', 'Beach')] }))
		)
	]);
	await settle();

	openDrawer();
	const info = element('.info');
	// The caller's own tag renders as a chip in the file section.
	const chip = info.querySelector<HTMLElement>('.info__tag-chip');
	expect(chip?.textContent?.trim()).toBe('Beach');
	expect(chip?.dataset.tagId).toBe('t1');
	// The assign affordance rides the row (per-user state: every
	// viewer's own write).
	expect(scopedButtonByText(info, 'Add tag')).toBeTruthy();

	unmount(app);
	flushSync();

	// The first part left the panel's open preference behind:
	// drop it so the bare mount starts from the default (closed).
	localStorage.removeItem('lumemor:viewer:info');
	// No tag yet: the honest empty note, the affordance still there.
	const { app: bare } = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	openDrawer();
	expect(element('.info').textContent).toContain('No tags yet.');
	expect(element('.info').querySelectorAll('.info__tag-chip')).toHaveLength(0);
	unmount(bare);
	flushSync();
});

test('the tag assign flow: picking an existing tag writes the single id, the report lands, Done converges the row', async () => {
	let detailCalls = 0;
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => {
			detailCalls += 1;
			return jsonResponse(
				200,
				detailBodyOf(sequence[1]!, { tags: detailCalls > 1 ? [tagEntry('t1', 'Beach')] : [] })
			);
		}),
		get('/api/v1/tags', () => jsonResponse(200, [tagSummary('t1', 'Beach'), tagSummary('t2', 'Sunrise')])),
		post('/api/v1/tags/t1/assets', () => jsonResponse(200, { added: 1, skipped: 0 }))
	]);
	await settle();

	openDrawer();
	click(scopedButtonByText(element('.info'), 'Add tag'));
	await settle();

	// The dialog offers the caller's vocabulary; picking the first
	// tag submits the single-asset batch.
	const dialog = element('.modal__panel');
	expect(dialog.getAttribute('aria-label')).toBe('Add tag');
	const options = Array.from(dialog.querySelectorAll('option'));
	expect(options.map((option) => option.value)).toEqual(['t1', 't2']);
	click(scopedButtonByText(dialog, 'Add tag'));
	await settle();

	// The write posted exactly this asset's id.
	const posts = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/tags/t1/assets' && init?.method === 'POST'
	);
	expect(posts).toHaveLength(1);
	expect(JSON.parse(String(posts[0]?.[1]?.body))).toEqual({ assetIds: ['a2'] });
	// The honest report renders inside the dialog.
	expect(element('.modal__panel').textContent).toContain('Added 1, skipped 0.');

	click(scopedButtonByText(element('.modal__panel'), 'Done'));
	await settle();

	// The done closed the dialog and converged through the detail
	// refetch: the tags array IS the row's truth.
	expect(document.body.querySelector('.modal__panel')).toBeNull();
	expect(element('.info__tag-chip').textContent?.trim()).toBe('Beach');

	unmount(app);
	flushSync();
});

test('the tag assign flow: the create-new path creates the tag idempotently, then tags the asset', async () => {
	const { app, fetchMock } = mountViewer('a2', [
		...viewerRoutes('a2'),
		get('/api/v1/tags', () => jsonResponse(200, [tagSummary('t1', 'Beach')])),
		post('/api/v1/tags', () => jsonResponse(201, tagSummary('t-new', 'Roadtrip'))),
		post('/api/v1/tags/t-new/assets', () => jsonResponse(200, { added: 1, skipped: 0 }))
	]);
	await settle();

	openDrawer();
	click(scopedButtonByText(element('.info'), 'Add tag'));
	await settle();

	const dialog = element('.modal__panel');
	const radios = Array.from(dialog.querySelectorAll<HTMLInputElement>('input[type="radio"]'));
	radios
		.find((radio) => radio.value === 'new')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	radios.find((radio) => radio.value === 'new')?.dispatchEvent(new Event('change', { bubbles: true }));
	flushSync();
	const field = dialog.querySelector<HTMLInputElement>('input:not([type="radio"])');
	if (field === null) {
		throw new Error('missing tag name field');
	}
	field.value = 'Roadtrip';
	field.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	click(scopedButtonByText(dialog, 'Create and add'));
	await settle();

	const creates = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/tags' && init?.method === 'POST'
	);
	expect(creates).toHaveLength(1);
	expect(JSON.parse(String(creates[0]?.[1]?.body))).toEqual({ name: 'Roadtrip' });
	const adds = fetchMock.mock.calls.filter(
		([input, init]) => String(input) === '/api/v1/tags/t-new/assets' && init?.method === 'POST'
	);
	expect(adds).toHaveLength(1);
	expect(JSON.parse(String(adds[0]?.[1]?.body))).toEqual({ assetIds: ['a2'] });
	expect(element('.modal__panel').textContent).toContain('Added 1, skipped 0.');

	unmount(app);
	flushSync();
});

test("the tags row's assign affordance is every viewer's own write (per-user state, never the owner's alone)", async () => {
	// A sharee's drawer: the description edit never renders (the
	// identity seam's read-only arm), but the tags affordance does
	// -- the tag write is the CALLER's own per-user row, and the
	// union deliberately carries it.
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(
				200,
				detailBodyOf(sequence[1]!, {
					ownerId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70',
					tags: [tagEntry('t2', 'Sunrise')]
				})
			)
		)
	]);
	await settle();

	openDrawer();
	const info = element('.info');
	// The read-only drawer: the description carries the shared text
	// as a plain line (never a live edit control), the pencils are
	// absent by their accessible names, and the per-user tag entry
	// stays.
	const description = info.querySelector('.info__description');
	expect(description?.tagName).toBe('P');
	expect(description?.textContent?.trim()).toBe('Original caption.');
	expect(info.querySelectorAll('button[aria-label="Edit date"], button[aria-label="Edit location"]')).toHaveLength(0);
	expect(info.querySelector('.info__tag-chip')?.textContent?.trim()).toBe('Sunrise');
	expect(scopedButtonByText(info, 'Add tag')).toBeTruthy();

	unmount(app);
	flushSync();
});

// --- the edit-people panel (the People section's pencil) -----------------------

/** The pencil that opens the edit-people panel. */
function peoplePencil(): HTMLButtonElement {
	const found = document.body.querySelector<HTMLButtonElement>('.info__faces-edit');
	if (found === null) {
		throw new Error('missing people pencil');
	}
	return found;
}

/** One edit-people panel row, addressed by its face id. */
function editRow(faceId: string): HTMLLIElement {
	const found = document.body.querySelector<HTMLLIElement>(`.faces-panel__row[data-face-id="${faceId}"]`);
	if (found === null) {
		throw new Error(`missing edit row ${faceId}`);
	}
	return found;
}

/** One row button of the edit-people panel, by aria-label. */
function editRowButton(faceId: string, label: string): HTMLButtonElement {
	const found = Array.from(editRow(faceId).querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.getAttribute('aria-label') === label
	);
	if (found === undefined) {
		throw new Error(`missing row button ${label}`);
	}
	return found;
}

test("the People pencil opens the edit-people panel in the drawer's place; the photo stays", async () => {
	const faces = [
		faceSummary('f1', { label: { name: 'Ada' } }),
		faceSummary('f2'),
		faceSummary('f3', { clusterId: null })
	];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces })))
	]);
	await settle();

	openDrawer();
	expect(document.body.querySelector('.info')).not.toBeNull();
	expect(document.body.querySelector('.faces-panel')).toBeNull();

	click(peoplePencil());
	await settle();

	// The panel stands in the information drawer's place -- the
	// drawer itself is still open underneath -- and the media stage
	// never unmounted (the photo stays visible).
	expect(document.body.querySelector('.faces-panel')).not.toBeNull();
	expect(document.body.querySelector('.info')).toBeNull();
	expect(document.body.querySelector('.viewer__image')).not.toBeNull();

	// The section's hint and one row per face, the three label states.
	expect(element('.faces-panel__hint').textContent).toContain('Every face in this photo');
	expect(
		Array.from(document.body.querySelectorAll<HTMLElement>('.faces-panel__row')).map((row) => row.dataset.faceId)
	).toEqual(['f1', 'f2', 'f3']);

	// The back affordance returns to the information drawer.
	click(element('.faces-panel__back'));
	await settle();
	expect(document.body.querySelector('.faces-panel')).toBeNull();
	expect(document.body.querySelector('.info')).not.toBeNull();
	unmount(app);
	flushSync();
});

test("the pencil is the owner's curation alone and never renders without faces", async () => {
	// A sharee's drawer: no pencil (the face correction writes are
	// the owner's), the chips keep their own assign flow.
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(
				200,
				detailBodyOf(sequence[1]!, {
					faces: [faceSummary('f2')],
					ownerId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70'
				})
			)
		)
	]);
	await settle();
	openDrawer();
	expect(document.body.querySelector('.info__faces-edit')).toBeNull();
	unmount(app);
	flushSync();

	// A photo with no faces: the section is the placeholder, no rows
	// to edit, no pencil.
	const { app: bare } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: [] })))
	]);
	await settle();
	// The first half's toggle persisted the open preference: the
	// drawer renders without another toggle.
	expect(document.body.querySelector('.info')).not.toBeNull();
	expect(document.body.querySelector('.info__faces-edit')).toBeNull();
	expect(element('.info__faces-empty').textContent).toContain('No people recognized');
	unmount(bare);
	flushSync();
});

test('the reassign stages locally, the check commits once, and the echo counts the edits', async () => {
	// The server's truth: f2 sits on an unnamed cluster; the commit
	// moves it onto Mom's cluster and the refetch carries the label.
	let moved = false;
	const facesNow = () => [
		faceSummary('f1', { label: { name: 'Ada' } }),
		moved ? faceSummary('f2', { clusterId: 'p2', label: { name: 'Mom' } }) : faceSummary('f2')
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('p1', 'Ada'), personTarget('p2', 'Mom')])),
		post('/api/v1/faces/f2/move', () => {
			moved = true;
			return jsonResponse(200, { faceId: 'f2', clusterId: 'p2' });
		})
	]);
	await settle();

	openDrawer();
	click(peoplePencil());
	await settle();
	click(editRowButton('f2', 'Reassign this face'));
	await settle();

	// The choose-person dialog: the filterable target list, the
	// face's own cluster never a target.
	expect(element('[role="dialog"]').getAttribute('aria-label')).toBe('Choose person');
	const filter = document.body.querySelector<HTMLInputElement>('.person-picker__filter');
	if (filter === null) {
		throw new Error('missing filter input');
	}
	expect(
		Array.from(document.querySelectorAll('.person-picker__row')).map((row) =>
			row.textContent?.replace(/\s+/g, ' ').trim()
		)
	).toEqual(['Ada 12', 'Mom 12']);
	// The local filter narrows the rows (no request rides it).
	const peopleFetches = () => fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/people').length;
	const before = peopleFetches();
	filter.value = 'mom';
	filter.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	expect(
		Array.from(document.querySelectorAll('.person-picker__row')).map((row) =>
			row.textContent?.replace(/\s+/g, ' ').trim()
		)
	).toEqual(['Mom 12']);
	expect(peopleFetches()).toBe(before);
	// No move has posted: the pick STAGES, never writes.
	expect(fetchMock.mock.calls.some(([input]) => String(input) === '/api/v1/faces/f2/move')).toBe(false);

	click(buttonByText('Assign'));
	await settle();

	// The dialog closed and the row carries the staged target with
	// its pending marker -- still no write.
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(editRow('f2').dataset.staged).toBe('true');
	expect(editRow('f2').textContent?.trim()).toBe('Mom');
	expect(fetchMock.mock.calls.some(([input]) => String(input) === '/api/v1/faces/f2/move')).toBe(false);
	// The apply gate opened with the stage.
	expect(document.querySelector<HTMLButtonElement>('.faces-panel__apply')?.disabled).toBe(false);

	// The one-shot commit: ONE move with the contract's body, the
	// panel returns to the drawer, the count echo lands, and the
	// convergence refetch re-renders the row from the server's truth.
	click(element('.faces-panel__apply'));
	await settle();
	const init = requestOf(fetchMock, 'POST', '/api/v1/faces/f2/move');
	expect(JSON.parse(String(init?.body))).toEqual({ targetClusterId: 'p2' });
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/faces/f2/move')).toHaveLength(1);
	expect(document.body.querySelector('.faces-panel')).toBeNull();
	expect(document.body.querySelector('.info')).not.toBeNull();
	expect(toasts().map((entry) => entry.text)).toEqual(['Made 1 change.']);
	// The information drawer returned with the server's converged
	// truth: the moved face's chip carries Mom's label.
	expect(Array.from(document.body.querySelectorAll('.info__face-name')).map((node) => node.textContent)).toEqual([
		'Ada',
		'Mom'
	]);
	unmount(app);
	flushSync();
});

test('the create-from-face stages and the commit seeds the person; the detach writes immediately', async () => {
	// f3 is not-yet-grouped: its only action is the reassign pencil,
	// whose dialog holds the create row.
	let created = false;
	const facesNow = () => [
		faceSummary('f1', { label: { name: 'Ada' } }),
		created ? faceSummary('f3', { clusterId: 'n1' }) : faceSummary('f3', { clusterId: null })
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('p1', 'Ada')])),
		post('/api/v1/faces/f3/cluster', () => {
			created = true;
			return jsonResponse(200, { faceId: 'f3', clusterId: 'n1' });
		}),
		del('/api/v1/faces/f1', () => new Response(null, { status: 204 }))
	]);
	await settle();

	openDrawer();
	click(peoplePencil());
	await settle();
	// The clusterless row carries the reassign pencil alone.
	expect(editRow('f3').querySelectorAll('button')).toHaveLength(1);
	click(editRowButton('f3', 'Reassign this face'));
	await settle();

	// The create row sits below the list; clicking it stages the
	// creation (no write yet).
	click(buttonByText('Create a new person from this face'));
	await settle();
	expect(editRow('f3').dataset.staged).toBe('true');
	expect(editRow('f3').textContent?.trim()).toBe('Create a new person from this face');
	expect(fetchMock.mock.calls.some(([input]) => String(input) === '/api/v1/faces/f3/cluster')).toBe(false);

	// The commit seeds the person through the creation endpoint; the
	// create carries its own echo and the count follows.
	click(element('.faces-panel__apply'));
	await settle();
	const init = requestOf(fetchMock, 'POST', '/api/v1/faces/f3/cluster');
	expect(init?.body).toBeUndefined();
	expect(toasts().map((entry) => entry.text)).toEqual(['Created a new person from this face.', 'Made 1 change.']);
	// The convergence refetch re-rendered the detail from the
	// server's truth (the initial load plus the commit's refetch).
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2?locale=en').length).toBe(2);
	unmount(app);
	flushSync();
});

test('the detach confirm names the person, the write is immediate, and the row leaves its person', async () => {
	// The server's truth after the detach: f1's membership is gone
	// -- the face row SURVIVES among its asset's faces, now
	// not-yet-grouped (the honest post-state the refetch carries).
	let detached = false;
	const facesNow = () => [
		detached ? faceSummary('f1', { clusterId: null, label: null }) : faceSummary('f1', { label: { name: 'Ada' } }),
		faceSummary('f2')
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		del('/api/v1/faces/f1', () => {
			detached = true;
			return new Response(null, { status: 204 });
		})
	]);
	await settle();

	openDrawer();
	click(peoplePencil());
	await settle();
	// The named row carries both affordances; the unnamed row's only
	// action is the pencil (the component's own pin, seen live).
	expect(editRow('f2').querySelector('[aria-label="Remove this face"]')).toBeNull();
	click(editRowButton('f1', 'Remove this face'));
	await settle();

	// The confirm names the person and states the honest cost.
	expect(element('[role="dialog"]').getAttribute('aria-label')).toBe('Remove this face');
	expect(element('.detach-dialog__body').textContent).toContain("Ada's face goes back to not grouped yet");

	click(buttonByText('Remove'));
	await settle();

	// The write ran immediately (a confirm, never a stage): the
	// panel STAYS, the echo lands, and the refetch re-renders the
	// row among the not-yet-grouped.
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/faces/f1')).toHaveLength(1);
	expect(document.body.querySelector('.faces-panel')).not.toBeNull();
	expect(toasts().map((entry) => entry.text)).toEqual(['Face removed; it is back among the not grouped yet.']);
	expect(editRow('f1').textContent?.trim()).toBe('Not grouped yet');
	expect(editRow('f1').querySelector('[aria-label="Remove this face"]')).toBeNull();
	unmount(app);
	flushSync();
});

test('the detach drops the row\u2019s staged reassign', async () => {
	// A staged reassign followed by a detach: the membership the
	// stage was aimed at is gone, so the stage dies with it -- the
	// row never keeps a pending dot the commit would turn into a
	// surprise move.
	let detached = false;
	const facesNow = () => [
		detached ? faceSummary('f1', { clusterId: null, label: null }) : faceSummary('f1', { label: { name: 'Ada' } })
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('p1', 'Ada'), personTarget('p2', 'Mom')])),
		post('/api/v1/faces/f1/move', () => jsonResponse(200, { faceId: 'f1', clusterId: 'p2' })),
		del('/api/v1/faces/f1', () => {
			detached = true;
			return new Response(null, { status: 204 });
		})
	]);
	await settle();

	openDrawer();
	click(peoplePencil());
	await settle();

	// Stage the reassign onto Mom (a pick never writes).
	click(editRowButton('f1', 'Reassign this face'));
	await settle();
	const momRow = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.person-picker__row')).find((row) =>
		row.textContent?.includes('Mom')
	);
	if (momRow === undefined) {
		throw new Error('missing Mom row');
	}
	click(momRow);
	click(buttonByText('Assign'));
	await settle();
	expect(editRow('f1').dataset.staged).toBe('true');
	expect(element('.faces-panel__apply').dataset.stagedCount).toBe('1');

	// The detach on the same row writes immediately and clears the
	// stage: the row carries no pending marker and the apply gate
	// has nothing left to commit.
	click(editRowButton('f1', 'Remove this face'));
	await settle();
	click(buttonByText('Remove'));
	await settle();
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/faces/f1')).toHaveLength(1);
	expect(editRow('f1').textContent?.trim()).toBe('Not grouped yet');
	expect(editRow('f1').dataset.staged).toBe('false');
	expect(element('.faces-panel__apply').dataset.stagedCount).toBe('0');
	expect(document.querySelector<HTMLButtonElement>('.faces-panel__apply')?.disabled).toBe(true);
	expect(toasts().map((entry) => entry.text)).toEqual(['Face removed; it is back among the not grouped yet.']);
	expect(fetchMock.mock.calls.some(([input]) => String(input) === '/api/v1/faces/f1/move')).toBe(false);
	unmount(app);
	flushSync();
});

test('the commit failure keeps the panel and the uncommitted stages; the error renders beside the apply', async () => {
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { faces: [faceSummary('f2')] }))
		),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('p1', 'Ada')])),
		post('/api/v1/faces/f2/move', () => jsonResponse(404, { code: 'PERSON_NOT_FOUND' }))
	]);
	await settle();

	openDrawer();
	click(peoplePencil());
	await settle();
	click(editRowButton('f2', 'Reassign this face'));
	await settle();
	click(buttonByText('Assign'));
	await settle();
	expect(editRow('f2').dataset.staged).toBe('true');

	click(element('.faces-panel__apply'));
	await settle();

	// The refusal kept the panel open with the honest error line and
	// the stage staged (a retry never re-picks); no echo landed.
	expect(document.body.querySelector('.faces-panel')).not.toBeNull();
	const error = element('.faces-panel__error');
	expect(error.getAttribute('role')).toBe('alert');
	expect(error.textContent).toContain('does not exist or is not visible');
	expect(editRow('f2').dataset.staged).toBe('true');
	expect(toasts()).toHaveLength(0);
	// One refused write only: the loop stopped, never retried itself.
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/faces/f2/move')).toHaveLength(1);
	unmount(app);
	flushSync();
});

test('the layered Escape falls from the pick dialog to the panel, then the drawer, then the route', async () => {
	const back = vi.spyOn(history, 'back').mockImplementation(() => undefined);
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { faces: [faceSummary('f2')] }))
		),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('p1', 'Ada')]))
	]);
	await settle();

	openDrawer();
	click(peoplePencil());
	await settle();
	click(editRowButton('f2', 'Reassign this face'));
	await settle();
	expect(document.body.querySelector('[role="dialog"]')).not.toBeNull();

	// The dialog owns the first Escape; the panel stays.
	pressKey('Escape');
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(document.body.querySelector('.faces-panel')).not.toBeNull();
	expect(back).not.toHaveBeenCalled();

	// The panel's Escape returns to the information drawer (its back
	// affordance's own behavior), not out of the viewer.
	pressKey('Escape');
	expect(document.body.querySelector('.faces-panel')).toBeNull();
	expect(document.body.querySelector('.info')).not.toBeNull();
	expect(back).not.toHaveBeenCalled();

	pressKey('Escape');
	expect(document.body.querySelector('.info')).toBeNull();
	pressKey('Escape');
	expect(back).toHaveBeenCalledTimes(1);
	unmount(app);
	flushSync();
});

test('the face creation and detach events of other sessions converge the detail', async () => {
	let detached = false;
	const facesNow = () => [
		faceSummary('f1', detached ? { clusterId: null, label: null } : { label: { name: 'Ada' } })
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() })))
	]);
	await settle();
	openDrawer();
	const fetches = () =>
		fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2?locale=en').length;
	const base = fetches();

	// A face creation naming this asset's face (another session
	// seeded a person from it): the refetch converges.
	firePersonEvent({ id: 1, name: 'person.face_clustered', faceId: 'f1', clusterId: 'n1', userId: 'u9' });
	await settle();
	expect(fetches()).toBe(base + 1);

	// A detach naming the cluster this asset's face sits on: the
	// refetch converges the row onto the server's post-state (the
	// state flips before the event -- the refetch reads it at fetch
	// time).
	detached = true;
	firePersonEvent({ id: 2, name: 'person.face_detached', faceId: 'f1', clusterId: 'c-f1', userId: 'u9' });
	await settle();
	expect(fetches()).toBe(base + 2);
	expect(element('.info__face-name').textContent?.trim()).toBe('Not grouped yet');

	// An event off every rendered face adds no fetch.
	firePersonEvent({ id: 3, name: 'person.face_detached', faceId: 'f9', clusterId: 'c9', userId: 'u9' });
	await settle();
	expect(fetches()).toBe(base + 2);
	unmount(app);
	flushSync();
});

// --- The stage and the panel's edit arms (the viewer's final form) ---------

// A controllable ResizeObserver: the stage's refit input. jsdom has
// none, so the contain rule's geometry is driven through this fake --
// one measurement in, one media-box size out.
class FakeResizeObserver {
	static live: FakeResizeObserver[] = [];
	callback: (entries: Array<{ contentRect: { width: number; height: number } }>) => void;
	constructor(callback: FakeResizeObserver['callback']) {
		this.callback = callback;
		FakeResizeObserver.live.push(this);
	}
	observe(): void {}
	unobserve(): void {}
	disconnect(): void {}
	/** Deliver one stage measurement (the test's own stage size). */
	measure(width: number, height: number): void {
		this.callback([{ contentRect: { width, height } }]);
		flushSync();
	}
}

/** The last stage observer (the viewer mounts one per stage). */
function stageObserver(): FakeResizeObserver {
	const found = FakeResizeObserver.live.at(-1);
	if (found === undefined) {
		throw new Error('no stage observer registered');
	}
	return found;
}

/** Deliver the media's DECODED pixel size (the contain rule's ratio input). */
function decodeStage(width: number, height: number): void {
	const image = document.body.querySelector<HTMLImageElement>('img.viewer__image');
	if (image === null) {
		throw new Error('missing the media image');
	}
	Object.defineProperty(image, 'naturalWidth', { value: width, configurable: true });
	Object.defineProperty(image, 'naturalHeight', { value: height, configurable: true });
	image.dispatchEvent(new Event('load'));
	flushSync();
}

/** The media box's rendered size, as the contain rule computed it. */
function mediaBoxSize(): { width: number; height: number } {
	const media = element('.viewer__media');
	return { width: Number.parseFloat(media.style.width), height: Number.parseFloat(media.style.height) };
}

test('the stage fits the media to one dimension: the box refits when the stage does', async () => {
	vi.stubGlobal('ResizeObserver', FakeResizeObserver);
	FakeResizeObserver.live.length = 0;
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// The decoded 3:2 landscape, on a 1280x720 stage: the height is
	// the short dimension, so the box fills it and the width follows
	// the ratio (not a percentage inset of the stage).
	decodeStage(1200, 800);
	stageObserver().measure(1280, 720);
	expect(mediaBoxSize()).toEqual({ width: 1080, height: 720 });

	// A 4:3 stage at the same width: the width is now the short
	// dimension, so the box fills the width instead.
	stageObserver().measure(1280, 1000);
	const wide = mediaBoxSize();
	expect(wide.width).toBe(1280);
	expect(wide.height).toBeCloseTo(853.33, 1);

	// A portrait photo (3:4) on the same stage: the height fills it
	// and the width follows -- as large as the stage allows, never
	// the old fixed inset.
	decodeStage(900, 1200);
	stageObserver().measure(1280, 1000);
	const tall = mediaBoxSize();
	expect(tall.height).toBeCloseTo(1000, 0);
	expect(tall.width).toBeCloseTo(750, 0);
	unmount(app);
	flushSync();
});

test('the live layers ride inside the media box: the face boxes, the drag layer, the zoom', async () => {
	vi.stubGlobal('ResizeObserver', FakeResizeObserver);
	FakeResizeObserver.live.length = 0;
	const faces = [faceSummary('f1', { bbox: { x: 260, y: 120, w: 150, h: 150 } })];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces })))
	]);
	await settle();
	decodeStage(1200, 800);
	stageObserver().measure(1280, 720);

	// THE REGRESSION PIN: the boxes are descendants of the media box,
	// so their percentage geometry follows the box through every
	// refit -- never siblings of it, never stage-relative.
	const media = element('.viewer__media');
	const layer = media.querySelector('[data-face-layer]');
	expect(layer).not.toBeNull();
	const box = media.querySelector<HTMLElement>('[data-face-box="f1"]');
	expect(box).not.toBeNull();
	// A square by construction: one width, the height follows.
	expect(box?.className).toContain('aspect-square');
	expect(box?.style.left).toBe('26%');
	expect(box?.style.top).toBe('12%');
	expect(box?.style.width).toBe('15%');
	// The box's own height is never set (the aspect does it).
	expect(box?.style.height).toBe('');

	// The zoom scales the box that carries the layers, so the photo
	// and its boxes enlarge together.
	expect(element('.viewer__stage').style.getPropertyValue('--viewer-zoom')).toBe('1');

	// The drag layer arms inside the same box.
	openDrawer();
	click(element('[aria-label="Add person"]'));
	expect(media.querySelector('[data-face-draw]')).not.toBeNull();
	unmount(app);
	flushSync();
});

test('one box at a time: a chip lights its own box, the shadow dims the rest, the plate flips', async () => {
	vi.stubGlobal('ResizeObserver', FakeResizeObserver);
	FakeResizeObserver.live.length = 0;
	vi.useFakeTimers({ shouldAdvanceTime: true });
	const faces = [
		faceSummary('f1', { bbox: { x: 260, y: 300, w: 150, h: 150 }, label: { name: 'Ada' } }),
		// A box 20 per-mille from the photo's top edge: too close to
		// hold a name plate above it, so its plate flips below.
		faceSummary('f2', { bbox: { x: 600, y: 20, w: 140, h: 140 } })
	];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces })))
	]);
	await settle();
	decodeStage(1200, 800);
	stageObserver().measure(1280, 720);
	openDrawer();

	const boxOf = (id: string) => document.body.querySelector<HTMLElement>(`[data-face-box="${id}"]`);
	// At rest every box is transparent and untouchable (the watching
	// view, never the debug view).
	expect(boxOf('f1')?.className).toContain('opacity-0');
	expect(boxOf('f2')?.className).toContain('opacity-0');

	// Pointing at a chip lights THAT box alone -- the box outside it
	// stays dark, and the linked one carries the huge-spread shadow
	// that dims everything outside it (the interior untouched).
	element('[data-face-id="f1"]').dispatchEvent(new MouseEvent('mouseenter', { bubbles: true }));
	flushSync();
	expect(boxOf('f1')?.className).toContain('opacity-100');
	expect(boxOf('f1')?.className).toContain('border-accent');
	expect(boxOf('f1')?.className).toContain('0_0_0_9999px');
	expect(boxOf('f2')?.className).toContain('opacity-0');
	// The chip lights back (the same state, both ends).
	expect(element('[data-face-id="f1"]').className).toContain('bg-accent-soft');
	// The name plate: white ground, dark text, above the box.
	const plate = boxOf('f1')?.querySelector('[data-face-label]');
	expect(plate?.textContent?.trim()).toBe('Ada');
	expect(plate?.className).toContain('bg-card/95');
	expect(plate?.className).toContain('text-text');
	expect(plate?.className).toContain('-top-7');

	// The near-top box flips its plate below the box instead.
	element('[data-face-id="f2"]').dispatchEvent(new MouseEvent('mouseenter', { bubbles: true }));
	flushSync();
	expect(boxOf('f2')?.className).toContain('opacity-100');
	expect(boxOf('f1')?.className).toContain('opacity-0');
	expect(boxOf('f2')?.querySelector('[data-face-label]')?.className).toContain('-bottom-7');

	// The grace: leaving the chip does not drop the highlight at
	// once -- the pointer may be travelling toward the box.
	element('[data-face-id="f2"]').dispatchEvent(new MouseEvent('mouseleave', { bubbles: true }));
	flushSync();
	expect(boxOf('f2')?.className).toContain('opacity-100');
	vi.advanceTimersByTime(240);
	flushSync();
	expect(boxOf('f2')?.className).toContain('opacity-0');
	vi.useRealTimers();
	unmount(app);
	flushSync();
});

test('the drawn face: a square anchored at the start, per-mille on the wire, a miss under 24px', async () => {
	vi.stubGlobal('ResizeObserver', FakeResizeObserver);
	FakeResizeObserver.live.length = 0;
	let created = false;
	const facesNow = () =>
		created ? [faceSummary('f9', { clusterId: null, bbox: { x: 125, y: 167, w: 250, h: 333 } })] : [];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		post('/api/v1/faces', () => {
			created = true;
			return jsonResponse(200, { id: 'f9', thumbnail: '/api/v1/faces/f9/thumbnail' });
		}),
		get('/api/v1/people', () => jsonResponse(200, [])),
		get('/api/v1/people/unnamed?limit=60', () => jsonResponse(200, { items: [], totalCount: 0 }))
	]);
	await settle();
	decodeStage(1200, 800);
	stageObserver().measure(1280, 720);
	openDrawer();

	// Arm the drag layer through the panel's add-person icon.
	click(element('[aria-label="Add person"]'));
	const layer = element('[data-face-draw]');
	expect(layer.textContent).toContain('Draw a box around the face');
	// The layer's geometry is its own rect (the media box's on-screen
	// area): jsdom has no layout, so the test states it.
	layer.getBoundingClientRect = () =>
		({ left: 0, top: 0, width: 800, height: 600, right: 800, bottom: 600, x: 0, y: 0 }) as DOMRect;
	Object.defineProperty(layer, 'setPointerCapture', { value: () => undefined, configurable: true });

	// One pointer event of the drag stream (jsdom has no PointerEvent:
	// a mouse event with the pointer type and id rides the same
	// handlers).
	const drag = (target: HTMLElement, type: string, x: number, y: number): void => {
		const event = new MouseEvent(type, { bubbles: true, cancelable: true, clientX: x, clientY: y, button: 0 });
		Object.defineProperty(event, 'pointerId', { value: 1 });
		target.dispatchEvent(event);
		flushSync();
	};

	// The layer's handlers attach one microtask after the element
	// lands (the library defers pointer listeners): let the arm settle
	// before the pointer stream arrives.
	await settle();

	// A miss: ten pixels is not a face -- the square tracks the
	// pointer, then the release throws it away (no dialog, no write).
	drag(layer, 'pointerdown', 100, 100);
	drag(layer, 'pointermove', 110, 110);
	expect(document.body.querySelector('[data-face-rect]')).not.toBeNull();
	drag(layer, 'pointerup', 110, 110);
	expect(document.body.querySelector('[data-face-draw]')).toBeNull();
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();
	expect(requestOf(fetchMock, 'POST', '/api/v1/faces')).toBeUndefined();

	// A real box: 200px square (the long edge sets the side), anchored
	// at the start and converted to the contract's per-mille units.
	await settle();
	click(element('[aria-label="Add person"]'));
	const layer2 = element('[data-face-draw]');
	layer2.getBoundingClientRect = layer.getBoundingClientRect;
	Object.defineProperty(layer2, 'setPointerCapture', { value: () => undefined, configurable: true });
	await settle();
	drag(layer2, 'pointerdown', 100, 100);
	drag(layer2, 'pointermove', 300, 260);
	drag(layer2, 'pointerup', 300, 260);
	await settle();
	const body = requestOf(fetchMock, 'POST', '/api/v1/faces')?.body;
	expect(JSON.parse(String(body))).toEqual({
		assetId: 'a2',
		bbox: { x: 125, y: 167, w: 250, h: 333 }
	});
	// The drawn face joins the same layer, the same linkage, and the
	// assign flow opens over it (the next step is who it is).
	expect(document.body.querySelector('[role="dialog"]')?.getAttribute('aria-label')).toBe('Assign person');
	expect(document.body.querySelector('[data-face-box="f9"]')).not.toBeNull();
	unmount(app);
	flushSync();
});

test('the panel is one switch: default off, remembered, the bar the close mark and the i key', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// Default off (the pinned preference): the photo gets the stage.
	expect(document.body.querySelector('.info')).toBeNull();
	const toggle = barButton('Photo information');
	expect(toggle.getAttribute('aria-pressed')).toBe('false');
	expect(toggle.getAttribute('aria-controls')).toBe('viewer-info');

	// The bar's button: pressed state synced, the panel named.
	click(toggle);
	expect(element('.info').getAttribute('id')).toBe('viewer-info');
	expect(barButton('Photo information').getAttribute('aria-pressed')).toBe('true');
	expect(localStorage.getItem('lumemor:viewer:info')).toBe('true');

	// The panel's own close mark is the same path.
	click(element('.info__close'));
	expect(document.body.querySelector('.info')).toBeNull();
	expect(localStorage.getItem('lumemor:viewer:info')).toBe('false');

	// The i key is the third door.
	pressKey('i');
	expect(document.body.querySelector('.info')).not.toBeNull();
	unmount(app);
	flushSync();
});

test('the filmstrip floats card-less with translucent thumbs; the arrows rest at 45%', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// No card around the strip: no ground, no radius, no border, no
	// blur -- the THUMBS are the translucent part.
	const strip = element('.viewer__filmstrip');
	expect(strip.className).toContain('max-w-[min(56%,560px)]');
	expect(strip.className).toContain('mask-image');
	expect(strip.className).not.toContain('rounded');
	expect(strip.className).not.toContain('border');
	expect(strip.className).not.toContain('backdrop-blur');
	// The current thumb reads 100% with the accent edge; its
	// neighbors rest translucent so the photo shows through.
	const thumbs = Array.from(document.body.querySelectorAll<HTMLElement>('.viewer__thumb'));
	expect(thumbs[1]?.className).toContain('opacity-100');
	expect(thumbs[1]?.className).toContain('border-accent');
	expect(thumbs[0]?.className).toContain('opacity-65');

	// The three-step arrows: 45% at rest, 100% on hover and keyboard
	// focus, 90% where there is no hover to lift them.
	const arrow = element('.viewer__arrow--left');
	expect(arrow.className).toContain('opacity-45');
	expect(arrow.className).toContain('hover:opacity-100');
	expect(arrow.className).toContain('focus-visible:opacity-100');
	expect(arrow.className).toContain('[@media(hover:none)]:opacity-90');
	unmount(app);
	flushSync();
});

test('the floating bar carries no filename and draws over the photo', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// Back on the left, the actions on the right, the middle open --
	// the name lives in the panel's header and the browser title alone.
	const bar = element('.viewer__bar');
	expect(bar.textContent).not.toContain('IMG_2.jpg');
	expect(bar.className).toContain('absolute');
	expect(bar.className).toContain('from-night/70');
	expect(bar.querySelector('.viewer__bar-actions')?.contains(barButton('Photo information'))).toBe(true);
	expect(document.title).toBe('IMG_2.jpg · lumemor');
	unmount(app);
	flushSync();
});

test('the info rows: the capture time with its zone, the size, the camera line, no filename row', async () => {
	const detail = detailBodyOf(sequence[1]!, {
		exif: { cameraMake: 'Apple', cameraModel: 'iPhone 15', focal: 24, aperture: 1.8, shutter: '0.008', iso: 100 }
	});
	const { app } = mountViewer('a2', [get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detail))]);
	await settle();
	openDrawer();

	// The capture time renders in its zone's wall clock with the zone
	// beside it (the pin: the zone must be visible).
	const facts = element('.info__facts');
	expect(facts.textContent).toContain('UTC+8');
	expect(facts.textContent).toContain('5:00');
	// The size and weight line, then the camera line (one line, the
	// six EXIF facts pressed together, never a fold).
	expect(facts.textContent).toContain('4,000 × 3,000');
	expect(facts.textContent).toContain('4.8 MB');
	expect(facts.textContent).toContain('Apple iPhone 15');
	expect(facts.textContent).toContain('24 mm');
	expect(facts.textContent).toContain('f/1.8');
	expect(facts.textContent).toContain('ISO 100');
	// No filename row: the header already names the file.
	expect(facts.textContent).not.toContain('IMG_2.jpg');
	expect(element('.info__filename').textContent).toContain('IMG_2.jpg');
	// The rows are icon-led, one per line, hairline-separated.
	expect(facts.className).toContain('divide-y');
	expect(facts.querySelectorAll('li').length).toBeGreaterThanOrEqual(4);
	unmount(app);
	flushSync();
});

test('the admin path row: collapsed until asked, one click to copy, absent for a member', async () => {
	const writeText = vi.fn(async () => undefined);
	Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
	const detail = detailBodyOf(sequence[1]!, { path: 'originals/9f8a3c/2026/11/IMG_0002.jpg' });
	const { app } = mountViewer('a2', [get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detail))]);
	await settle();
	openDrawer();

	// One line, collapsed: the machine location is there when asked
	// for and out of the way otherwise.
	const row = element('[data-testid="viewer-path"]');
	const toggle = row.querySelector('button');
	expect(toggle?.getAttribute('aria-expanded')).toBe('false');
	expect(row.textContent).toContain('Path');
	expect(row.textContent).not.toContain('originals/');
	click(toggle as HTMLElement);
	expect(toggle?.getAttribute('aria-expanded')).toBe('true');
	expect(row.textContent).toContain('originals/9f8a3c/2026/11/IMG_0002.jpg');

	// The copy arm: the clipboard write and its own confirmation.
	click(row.querySelector('[aria-label="Copy path"]') as HTMLElement);
	await settle();
	expect(writeText).toHaveBeenCalledWith('originals/9f8a3c/2026/11/IMG_0002.jpg');
	expect(toasts().map((entry) => entry.text)).toContain('Path copied.');
	unmount(app);
	flushSync();

	// A member's detail carries no path at all: the row never exists
	// (the gate is the field's presence, not a disabled state).
	const member = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!)))
	]);
	await settle();
	openDrawer();
	expect(document.body.querySelector('[data-testid="viewer-path"]')).toBeNull();
	unmount(member.app);
	flushSync();
});

test('the date row writes the wall clock and its zone as one value', async () => {
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { takenAt: '2026-08-12T10:42:00Z', takenAtZone: '+08:00' }))
		),
		patch('/api/v1/assets/a2?locale=en', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { takenAt: '2026-08-12T10:42:00Z', takenAtZone: '+08:00' }))
		)
	]);
	await settle();
	openDrawer();

	// The pencil opens the dialog seeded from the stored truth: the
	// wall clock in its zone, the zone selected.
	click(element('[aria-label="Edit date"]'));
	const dialog = element('[role="dialog"]');
	expect(dialog.getAttribute('aria-label')).toBe('Edit date');
	const field = dialog.querySelector<HTMLInputElement>('#viewer-edit-date-value');
	expect(field?.value).toBe('2026-08-12T18:42');
	const zone = dialog.querySelector<HTMLSelectElement>('#viewer-edit-date-zone');
	expect(zone?.value).toBe('+08:00');

	// A save composes the pair into the one RFC 3339 value the
	// contract wants: the wall clock plus the offset that wrote it.
	fillInput(field as HTMLInputElement, '2026-08-12T18:42');
	click(buttonByText('Save'));
	await settle();
	const body = JSON.parse(String(requestOf(fetchMock, 'PATCH', '/api/v1/assets/a2?locale=en')?.body));
	expect(body).toEqual({ takenAt: '2026-08-12T18:42:00+08:00' });
	expect(toasts().map((entry) => entry.text)).toContain('Date updated.');
	unmount(app);
	flushSync();
});

test('the location row: the static mini map, the coordinates, and nothing without a location', async () => {
	const located = detailBodyOf(sequence[1]!, {
		location: { lat: 36.0671, lon: 120.3826 },
		place: { placeId: 1816670, name: 'Qingdao' },
		exif: { alt: 12 }
	});
	const { app } = mountViewer('a2', [get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, located))]);
	await settle();
	openDrawer();

	// The last row of the info: the plain place name, the pencil, the
	// mini map (a static display -- it answers no gesture) and the
	// coordinates line with the camera's altitude.
	const row = element('[data-testid="viewer-location"]');
	expect(row.textContent).toContain('Qingdao');
	expect(row.querySelector('a')).toBeNull();
	expect(row.querySelector('[aria-label="Edit location"]')).not.toBeNull();
	const map = row.querySelector<HTMLElement>('.map');
	expect(map?.getAttribute('aria-label')).toBe('Location of this photo');
	expect(map?.getAttribute('data-interactive')).toBe('false');
	expect(map?.getAttribute('data-point')).toBe(JSON.stringify({ lat: 36.0671, lon: 120.3826 }));
	expect(row.textContent).toContain('36.0671, 120.3826');
	expect(row.textContent).toContain('Altitude 12 m');
	unmount(app);
	flushSync();

	// No GPS and no manual pin: the whole row stays unrendered (no
	// empty heading, no blank map).
	const blank = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!)))
	]);
	await settle();
	openDrawer();
	expect(document.body.querySelector('[data-testid="viewer-location"]')).toBeNull();
	unmount(blank.app);
	flushSync();
});

test("a tag chip's × removes that one assignment and the row converges", async () => {
	let removed = false;
	const tagsNow = () => (removed ? [] : [{ id: 't1', name: 'Beach' }]);
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2?locale=en', () => jsonResponse(200, detailBodyOf(sequence[1]!, { tags: tagsNow() }))),
		del('/api/v1/tags/t1/assets', () => {
			removed = true;
			return jsonResponse(200, { removed: 1, skipped: 0 });
		})
	]);
	await settle();
	openDrawer();

	// The × is its own button beside the name link (interactive
	// elements never nest): the name navigates, the × only removes.
	const chip = element('[data-tag-id="t1"]');
	const remove = chip.querySelector<HTMLButtonElement>('button[aria-label="Remove tag Beach"]');
	expect(remove).not.toBeNull();
	expect(chip.querySelector('a')?.getAttribute('href')).toBe('/search?tagIds=t1');
	click(remove as HTMLElement);
	await settle();

	// The write is the caller's own untag over this one id; the row
	// converges through the detail refetch (the tags array is the
	// truth) and the confirmation says what happened.
	const body = JSON.parse(String(requestOf(fetchMock, 'DELETE', '/api/v1/tags/t1/assets')?.body));
	expect(body).toEqual({ assetIds: ['a2'] });
	expect(toasts().map((entry) => entry.text)).toEqual(['Tag removed.']);
	expect(document.body.querySelector('[data-tag-id="t1"]')).toBeNull();
	expect(element('.info__tags-empty').textContent).toBe('No tags yet.');
	unmount(app);
	flushSync();
});

test('the description edit is a no-op when nothing changed: no request, no toast', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	openDrawer();

	// Enter on an untouched draft closes the editor silently: nothing
	// changed, so nothing is written and nothing is confirmed.
	openDescriptionEdit();
	submitDescriptionEdit();
	await settle();
	expect(element('.info__description').textContent).toBe('Original caption.');
	expect(toasts()).toEqual([]);
	unmount(app);
	flushSync();
});
