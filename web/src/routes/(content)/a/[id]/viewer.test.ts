import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import { clearWallItems, publishWallItems } from '$lib/timeline/wall.svelte';
import type { TimelineItem } from '$lib/api/timeline';
import Page from './+page.svelte';

// The asset viewer route: the navigation walk over the wall's
// published sequence (clamp at both ends, inert off-sequence deep
// links), the image / video / Live Photo stage, the not-ready wait
// that converges through the SSE asset.ready, the info drawer's
// description edit (PATCH body, error mapping, the stored text
// untouched on failure), the action bar's curation actions with
// their optimistic flips and rollback (the permanent delete left
// the viewer for the trash page in the R2-3 rework), and the
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
function detailBodyOf(
	source: TimelineItem,
	overrides: Record<string, unknown> = {}
): Record<string, unknown> {
	return {
		description: 'Original caption.',
		isHidden: false,
		exif: {},
		faces: [],
		tags: [],
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
		clusterId: `c-${id}`,
		label: null,
		...overrides
	};
}

/** One named person of the assign dialog's target list. */
function personTarget(clusterId: string, name: string): Record<string, unknown> {
	return {
		clusterId,
		name,
		faceCount: 12,
		representativeFaceId: `f-${clusterId}`,
		thumbnail: `/api/v1/faces/f-${clusterId}/thumbnail`,
		isHidden: false
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

function put(url: string, respond: () => Response | Promise<Response>): RouteRule {
	return { method: 'PUT', url, respond };
}

/** The default routes of the viewer surface (an image asset). */
function viewerRoutes(id: string, extra: RouteRule[] = []): RouteRule[] {
	return [get(`/api/v1/assets/${id}`, () => jsonResponse(200, detailBodyOf(sequence[1]!))), ...extra];
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
function mountViewer(id: string, rules: RouteRule[]): { app: ReturnType<typeof mount>; fetchMock: ReturnType<typeof vi.fn> } {
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

/** One icon-only button of the viewer's action bar, addressed by its aria-label. */
function barButton(label: string): HTMLButtonElement {
	const found = document.body.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing bar button ${label}`);
	}
	return found;
}

function fillTextarea(selector: string, value: string): void {
	const field = document.body.querySelector<HTMLTextAreaElement>(selector);
	if (field === null) {
		throw new Error(`missing textarea ${selector}`);
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
	// The info panel's open preference persists in localStorage (R2-9):
	// every test starts from the default (closed) unless it writes the
	// key itself.
	localStorage.clear();
	// Boot the session store as the sequence's owner: the drawer's
	// owner-only affordances resolve against this identity (the
	// identity seam's store carries it; the interceptor stays inert
	// because these routes answer 200s).
	vi.stubGlobal('fetch', vi.fn(async () =>
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
	));
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('authenticated');
	await initI18n();
	publishWallItems(sequence);
});

afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
	clearWallItems();
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
	expect(document.body.querySelector('.viewer__press')).toBeNull();

	// The filmstrip walks the sequence with the current outlined.
	const thumbs = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.viewer__thumb'));
	expect(thumbs.map((thumb) => thumb.getAttribute('data-filmstrip-id'))).toEqual(['a1', 'a2', 'a3']);
	expect(thumbs[1]?.getAttribute('aria-current')).toBe('true');
	expect(thumbs[1]?.querySelector('img')?.getAttribute('src')).toBe(
		'/api/v1/assets/a2/thumbnail?size=tile'
	);

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
		get('/api/v1/assets/a1', () => jsonResponse(200, detailBodyOf(sequence[0]!)))
	]);
	await settle();
	pressKey('ArrowLeft');
	expect(vi.mocked(goto)).toHaveBeenCalledTimes(2);
	unmount(head.app);
	flushSync();

	// The tail: ArrowRight is inert.
	const tail = mountViewer('a3', [
		get('/api/v1/assets/a3', () => jsonResponse(200, detailBodyOf(sequence[2]!)))
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
		get('/api/v1/assets/zz', () => jsonResponse(200, detailBodyOf(offPage)))
	]);
	await settle();

	// The off-sequence detail renders.
	expect(element('.viewer__title').textContent).toBe('OFF.jpg');
	expect(
		Array.from(document.body.querySelectorAll('img.viewer__image')).map((image) =>
			image.getAttribute('src')
		)
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
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		// The unknown code pins the shared fallback: the drawer
		// renders the generic message, never the raw code.
		patch('/api/v1/assets/a2', () => jsonResponse(502, { code: 'PROXY_TEAPOT' }))
	]);
	await settle();

	openDrawer();
	expect(element('.info__description').textContent).toBe('Original caption.');

	click(buttonByText('Edit'));
	fillTextarea('textarea.info__editor', 'My new caption.');
	click(buttonByText('Save'));
	await settle();

	// The exact PATCH body of the contract.
	const init = requestOf(fetchMock, 'PATCH', '/api/v1/assets/a2');
	expect(init).toBeDefined();
	expect(JSON.parse(String(init?.body))).toEqual({ description: 'My new caption.' });

	// The failure keeps the edit mode (the draft stays for a retry)
	// and shows the generic message, never the raw code.
	expect(document.body.querySelector('textarea.info__editor')).not.toBeNull();
	const error = element('.info__error');
	expect(error.getAttribute('role')).toBe('alert');
	expect(error.textContent).toBe('Something went wrong. Please try again.');
	expect(error.textContent).not.toContain('proxy_teapot');

	// The stored text never changed: canceling shows the original.
	click(buttonByText('Cancel'));
	expect(element('.info__description').textContent).toBe('Original caption.');
	unmount(app);
	flushSync();
});

test('a successful edit adopts the response description', async () => {
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		patch('/api/v1/assets/a2', () =>
			jsonResponse(200, detailBodyOf(sequence[1]!, { description: 'The saved caption.' }))
		)
	]);
	await settle();

	openDrawer();
	click(buttonByText('Edit'));
	fillTextarea('textarea.info__editor', 'The saved caption.');
	click(buttonByText('Save'));
	await settle();

	// The response IS the fresh detail: the read view adopts it.
	expect(document.body.querySelector('textarea.info__editor')).toBeNull();
	expect(element('.info__description').textContent).toBe('The saved caption.');
	unmount(app);
	flushSync();
});

test('the Live Photo press probes the component stream and toggles; not-ready keeps the still', async () => {
	const liveDetail = () =>
		jsonResponse(200, detailBodyOf(sequence[1]!, { livePhotoVideoId: 'c2' }));
	const probeCalls: string[] = [];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', liveDetail),
		get('/api/v1/assets/c2/play', () => {
			probeCalls.push('c2');
			return rangeResponse();
		})
	]);
	await settle();

	// The still renders first; the press toggles the motion on.
	expect(document.body.querySelector('video')).toBeNull();
	const toggle = element('.viewer__press');
	expect(toggle.getAttribute('aria-pressed')).toBe('false');
	click(toggle);
	await settle();

	// The probe hit the COMPONENT's play endpoint (the pair's
	// motion plays through the component id).
	expect(probeCalls).toEqual(['c2']);
	const live = document.body.querySelector<HTMLVideoElement>('video.viewer__live');
	expect(live?.getAttribute('src')).toBe('/api/v1/assets/c2/play');
	// Media playback flags are element properties, not attributes.
	expect(live?.autoplay).toBe(true);
	expect(live?.muted).toBe(true);
	expect(live?.loop).toBe(true);
	expect(element('.viewer__press').getAttribute('aria-pressed')).toBe('true');

	// The second press returns to the still.
	click(element('.viewer__press'));
	expect(document.body.querySelector('video.viewer__live')).toBeNull();
	unmount(app);
	flushSync();

	// The not-ready branch: the probe answers ASSET_NOT_READY and
	// the still photo stays (each press re-probes).
	let streamReady = false;
	const waiting = mountViewer('a2', [
		get('/api/v1/assets/a2', liveDetail),
		get('/api/v1/assets/c2/play', () =>
			streamReady ? rangeResponse() : jsonResponse(404, { code: 'ASSET_NOT_READY' })
		)
	]);
	await settle();
	click(element('.viewer__press'));
	await settle();
	expect(document.body.querySelector('video.viewer__live')).toBeNull();
	expect(document.body.querySelector('img.viewer__image')).not.toBeNull();

	// A later press lands once the stream appeared.
	streamReady = true;
	click(element('.viewer__press'));
	await settle();
	expect(document.body.querySelector('video.viewer__live')).not.toBeNull();
	unmount(waiting.app);
	flushSync();
});

test('a not-ready video shows the poster and waits for the SSE ready event', async () => {
	let videoReady = false;
	let streamReady = false;
	const { app } = mountViewer('v1', [
		get('/api/v1/assets/v1', () =>
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
	// window is observable before the failure. (The R2-3 rework: the
	// curation actions ride the viewer's action bar; the permanent
	// delete left the viewer for the trash page.)
	let release: ((response: Response) => void) | undefined;
	const batchAnswer = new Promise<Response>((resolve) => {
		release = resolve;
	});
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
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
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
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
		get(
			'/api/v1/assets/a2',
			() => jsonResponse(200, detailBodyOf(sequence[1]!, { isTrashed: true }))
		),
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
		get(
			'/api/v1/assets/a2',
			() => jsonResponse(200, detailBodyOf(sequence[1]!, { ownerId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70' }))
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
	// grayed out -- the web IA's pinned rule): no trash in the bar,
	// no edit in the drawer.
	expect(() => barButton('Trash')).toThrow();
	expect(document.body.textContent).not.toContain('Edit');
	expect(() => barButton('Favorite')).not.toThrow();

	// The per-user favorite still works for the viewer.
	click(barButton('Favorite'));
	await settle();
	const init = requestOf(fetchMock, 'POST', '/api/v1/assets/batch');
	expect(JSON.parse(String(init?.body))).toEqual({ action: 'favorite', ids: ['a2'] });
	unmount(app);
	flushSync();
});

test('the action bar carries the curation icon actions; the permanent delete is gone', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();

	// R2-3's bar: the close mark, the favorite toggle, the info-panel
	// toggle and (for the owner) the trash/restore pair, every button
	// an icon button named by its aria-label. R2-7 widens the zoom
	// controls to the in/out/reset trio.
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

	// With the info panel open, the action cluster keeps its overlay
	// guard -- the left shift (R2-3) now serves only the <lg overlay
	// fallback form (lg:me-0); at lg+ the split layout itself clears
	// the cluster (the split composition test below).
	const actions = () => document.body.querySelector('.viewer__bar-actions');
	click(barButton('Photo information'));
	expect(actions()?.className).toContain('me-[calc(min(360px,92vw)+1rem)]');
	expect(actions()?.className).toContain('lg:me-0');
	click(barButton('Photo information'));
	expect(actions()?.className).not.toContain('me-[calc(min(360px,92vw)+1rem)]');
	unmount(app);
	flushSync();
});

test('the info panel docks as a right split region at lg and falls back to the overlay below it', async () => {
	// R2-10's composition: the viewer root is a flex ROW -- the media
	// column flexes (flex-1 min-w-0, the bar and the stage live inside
	// it) and the panel rides BESIDE it as a real flex sibling
	// (lg:static: it occupies layout space, so the stage re-fits
	// through the flow instead of being covered). Below lg the same
	// panel keeps the fixed overlay-drawer form.
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
	// panel can never cover them at lg+).
	expect(main.contains(element('.viewer__bar'))).toBe(true);
	expect(main.contains(element('.viewer__stage'))).toBe(true);

	openDrawer();
	const panel = element('.info');
	// The panel is INSIDE the viewer's row (a flex sibling of the
	// media column), never a page-level overlay.
	expect(root.contains(panel)).toBe(true);
	// The docked form: static in the row, the same width family, no
	// floating shadow. The overlay form (fixed, full-height, shadowed)
	// rides the same element below lg.
	expect(panel.className).toContain('lg:static');
	expect(panel.className).toContain('lg:shrink-0');
	expect(panel.className).toContain('lg:shadow-none');
	expect(panel.className).toContain('w-[min(360px,92vw)]');
	expect(panel.className).toContain('fixed');
	expect(panel.className).toContain('shadow-overlay');
	// The zoom var the media transform rides (the engine drives it
	// per frame; 1 at rest).
	expect(element('.viewer__stage').style.getPropertyValue('--viewer-zoom')).toBe('1');
	unmount(app);
	flushSync();
});

test('the info panel open preference persists across viewer sessions', async () => {
	// R2-9: one localStorage key, default closed; the stored open
	// state restores on the next viewer mount, and closing persists
	// back.
	const first = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	expect(document.body.querySelector('.info')).toBeNull();

	openDrawer();
	expect(localStorage.getItem('lumemor:viewer:info-panel')).toBe('true');
	unmount(first.app);
	flushSync();

	// A fresh viewer session restores the open panel without a click.
	const second = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	expect(document.body.querySelector('.info')).not.toBeNull();

	// Closing it persists the closed preference for the next session.
	click(barButton('Photo information'));
	expect(localStorage.getItem('lumemor:viewer:info-panel')).toBe('false');
	unmount(second.app);
	flushSync();

	const third = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	expect(document.body.querySelector('.info')).toBeNull();
	unmount(third.app);
	flushSync();
});

test('the +/- keys and the double-click ride the gradual zoom; a neighbor resets it', async () => {
	// R2-7/8: the keyboard pair steps the same engine the bar's
	// buttons and the stage's double-click ride -- every change
	// animates, and the affordances key on the engine's target (the
	// zoomed stage flips synchronously with the intent). Navigating
	// to a neighbor snaps back to the resting scale.
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		get('/api/v1/assets/a1', () => jsonResponse(200, detailBodyOf(sequence[0]!)))
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
		get('/api/v1/assets/a1', () => jsonResponse(200, detailBodyOf(sequence[0]!)))
	]);
	await settle();
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(false);
	unmount(neighbor.app);
	flushSync();
});

test("a Live Photo's double press is the press pair, never a zoom toggle", async () => {
	const liveDetail = () =>
		jsonResponse(200, detailBodyOf(sequence[1]!, { livePhotoVideoId: 'c2' }));
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', liveDetail),
		get('/api/v1/assets/c2/play', () => rangeResponse())
	]);
	await settle();

	// The press affordance covers the stage; its double-click stays
	// the press pair (play/stop), so the zoom stays at rest.
	const press = element('.viewer__press');
	press.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
	flushSync();
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(false);

	// A double-click on the still itself toggles the zoom.
	element('.viewer__image').dispatchEvent(
		new MouseEvent('dblclick', { bubbles: true, cancelable: true })
	);
	flushSync();
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(true);
	unmount(app);
	flushSync();
});

test('the info panel is metadata-only: key-value rows with hairline dividers, no actions section', async () => {
	const { app } = mountViewer('a2', viewerRoutes('a2'));
	await settle();
	openDrawer();

	// No actions section survived the R2-3 rework: the curation
	// affordances live in the bar.
	expect(element('.info').textContent).not.toContain('Actions');

	// The file facts: label column faint, value column text, one
	// hairline between the rows.
	const facts = element('.info__facts');
	expect(facts.className).toContain('divide-y');
	const term = facts.querySelector('dt');
	const value = facts.querySelector('dd');
	expect(term?.className).toContain('text-text-faint');
	expect(value?.className).toContain('text-text');
	expect(value?.className).not.toContain('text-text-faint');
	unmount(app);
	flushSync();
});

test('the layered Escape closes the assign dialog, then the drawer, then leaves', async () => {
	const back = vi.spyOn(history, 'back').mockImplementation(() => undefined);
	// The modal layer is the page's face-assign dialog (the delete
	// dialog is gone with the permanent delete's departure).
	const faces = [faceSummary('f2')];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('c1', 'Ada')])),
		get('/api/v1/people/unnamed?limit=60', () => jsonResponse(200, { items: [] }))
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
	click(buttonByText('Edit'));
	const field = document.body.querySelector<HTMLTextAreaElement>('textarea.info__editor');
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
		get('/api/v1/assets/a2', () =>
			failing ? jsonResponse(500, { code: 'PROXY_TEAPOT' }) : jsonResponse(200, detailBodyOf(sequence[1]!))
		)
	]);
	await settle();

	const failure = element('.viewer__failure');
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
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!)))
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
	const detailCalls = fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2');
	handlers[0]?.({ id: 21, name: 'asset.updated', assetId: 'a2', kind: 'image' });
	await settle();
	const survived = Array.from(document.body.querySelectorAll<HTMLImageElement>('img.viewer__image'));
	expect(survived[0]).toBe(images[0]);
	expect(survived[0]?.classList.contains('viewer__image--loaded')).toBe(true);
	expect(element('.viewer__stage').classList.contains('viewer__stage--zoomed')).toBe(true);
	expect(
		fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2')
	).toHaveLength(detailCalls.length + 1);
	unmount(app);
	flushSync();

	// The navigation half: a neighbor mounts fresh (the plain-page
	// mock cannot mutate route params in place, so the in-instance
	// reset is approximated by the neighbor's mount -- the same
	// approximation the drawer's reset test uses). The new asset
	// starts unloaded and unzoomed: the crossfade begins over.
	const neighbor = mountViewer('a3', [
		get('/api/v1/assets/a3', () => jsonResponse(200, detailBodyOf(sequence[2]!)))
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
		get('/api/v1/assets/a2', () =>
			failing ? jsonResponse(500, { code: 'PROXY_TEAPOT' }) : jsonResponse(200, detailBodyOf(sequence[1]!))
		)
	]);
	// The wall snapshot says the asset IS a favorite; the initial
	// load fails before the seeding can run.
	publishWallItems([item('a2', '2026-11-01T09:00:00Z', { isFavorite: true })]);
	await settle();
	expect(element('.viewer__failure')).not.toBeNull();

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
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2')).toHaveLength(1);
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
		get('/api/v1/assets/v1', () => jsonResponse(200, detailBodyOf(videoSource, { isReady: false }))),
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
	publishWallItems([
		sequence[0]!,
		sequence[1]!,
		item('a3', '2026-10-13T12:30:00Z', { isReady: false })
	]);
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
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!))),
		post('/api/v1/assets/batch', () => batchAnswer)
	]);
	await settle();

	openDrawer();
	click(buttonByText('Edit'));
	fillTextarea('textarea.info__editor', 'My work in progress.');

	// The optimistic trash flip replaces the detail object (same
	// id): the flip is observable (the trashed banner) and the draft
	// and the edit mode survive the replacement.
	click(barButton('Trash'));
	await new Promise((resolve) => setTimeout(resolve, 0));
	expect(element('.info').textContent).toContain('This photo is in the trash.');
	expect(document.body.querySelector<HTMLTextAreaElement>('textarea.info__editor')?.value).toBe(
		'My work in progress.'
	);

	// The SSE convergence refetch replaces it again (same id): the
	// draft survives that replacement too.
	handlers[0]?.({ id: 14, name: 'asset.updated', assetId: 'a2', kind: 'image' });
	await settle();
	expect(fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2')).toHaveLength(2);
	expect(document.body.querySelector<HTMLTextAreaElement>('textarea.info__editor')?.value).toBe(
		'My work in progress.'
	);

	release?.(jsonResponse(200, { affected: 1, missing: [] }));
	await settle();
	unmount(app);
	flushSync();

	// An actual asset change (the panel-open navigation to the
	// neighbor) resets the edit mode: the neighbor's panel RESTORED
	// OPEN (R2-9: the preference persists across viewer sessions, and
	// this test's first half left it open), so no toggle click is
	// needed -- the read view renders and the draft never leaks
	// across assets.
	const neighbor = mountViewer('a1', [
		get('/api/v1/assets/a1', () =>
			jsonResponse(200, detailBodyOf(sequence[0]!, { description: 'The neighbor caption.' }))
		)
	]);
	await settle();
	expect(document.body.querySelector('textarea.info__editor')).toBeNull();
	expect(element('.info__description').textContent).toBe('The neighbor caption.');
	unmount(neighbor.app);
	flushSync();
});

test('concurrent convergence refetches apply newest-wins: a stale response never lands', async () => {
	const pending: Array<(response: Response) => void> = [];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => new Promise<Response>((resolve) => pending.push(resolve)))
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
		get('/api/v1/assets/a2', () =>
			failing ? jsonResponse(500, { code: 'PROXY_TEAPOT' }) : jsonResponse(200, detailBodyOf(sequence[1]!))
		)
	]);
	await settle();
	// The initial load failed: the error branch renders.
	expect(element('.viewer__failure').getAttribute('role')).toBe('alert');

	// The open asset's ready event lands the convergence refetch:
	// the success flips the route out of the error branch and the
	// stage renders.
	failing = false;
	handlers[0]?.({ id: 22, name: 'asset.ready', assetId: 'a2', kind: 'image' });
	await settle();
	expect(document.body.querySelector('.viewer__failure')).toBeNull();
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

test('a zoomed Live Photo press keeps the motion layer inside the zoomed stage', async () => {
	const liveDetail = () => jsonResponse(200, detailBodyOf(sequence[1]!, { livePhotoVideoId: 'c2' }));
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', liveDetail),
		get('/api/v1/assets/c2/play', () => rangeResponse())
	]);
	await settle();

	// Zoom in, then press the Live Photo: the motion layer mounts as
	// a child of the zoomed stage (the transform selector's DOM
	// contract -- .viewer__stage--zoomed .viewer__live -- so the
	// motion scales with the still, never pops unscaled).
	click(element('[aria-label="Zoom in"]'));
	click(element('.viewer__press'));
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
		faceSummary('f1', { label: { name: 'Ada', isHidden: false } }),
		faceSummary('f2'),
		faceSummary('f3', { clusterId: null })
	];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces })))
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
	// (The panel restored open -- R2-9's persistence, this test's own
	// first half left it open -- so no toggle click precedes the
	// assertion.)
	const empty = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: [] })))
	]);
	await settle();
	expect(document.body.querySelector('.info__face')).toBeNull();
	expect(element('.info__faces-empty').textContent).toBe('No people recognized in this photo.');
	unmount(empty.app);
	flushSync();
});

test('the assign flow moves a face onto an existing person and the chip converges', async () => {
	// The server's current answer: f2 sits on an unnamed cluster; the
	// move lands it on Ada's cluster and the refetch carries the label.
	let moved = false;
	const facesNow = () => [
		faceSummary('f1', { label: { name: 'Ada', isHidden: false } }),
		moved
			? faceSummary('f2', { clusterId: 'c1', label: { name: 'Ada', isHidden: false } })
			: faceSummary('f2')
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('c1', 'Ada')])),
		get('/api/v1/people/unnamed?limit=60', () =>
			jsonResponse(200, { items: [unnamedTarget('c9', 4)] })
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
	expect(
		Array.from(document.body.querySelectorAll('.info__face-name')).map((node) => node.textContent)
	).toEqual(['Ada', 'Ada']);
	unmount(app);
	flushSync();
});

test('a move on a not-yet-grouped face surfaces the not-grouped message', async () => {
	const faces = [faceSummary('f3', { clusterId: null })];
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces }))),
		get('/api/v1/people', () => jsonResponse(200, [personTarget('c1', 'Ada')])),
		get('/api/v1/people/unnamed?limit=60', () => jsonResponse(200, { items: [], nextCursor: null })),
		post('/api/v1/faces/f3/move', () => jsonResponse(404, { code: 'PERSON_NOT_FOUND' }))
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

	// The 404 stays the dialog's honest not-grouped message, never
	// the raw code's person-missing text.
	expect(element('[role="dialog"]')).not.toBeNull();
	const error = element('.cluster-picker__failure');
	expect(error.getAttribute('role')).toBe('alert');
	expect(error.textContent).toContain('has not been grouped yet');
	expect(error.textContent).not.toContain('person_not_found');
	unmount(app);
	flushSync();
});

test('an unnamed-cluster chip names its cluster as a new person; labeled chips are not offered', async () => {
	let named = false;
	const facesNow = () => [
		faceSummary('f1', { label: { name: 'Ada', isHidden: false } }),
		named ? faceSummary('f2', { label: { name: 'Grace', isHidden: false } }) : faceSummary('f2')
	];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() }))),
		get('/api/v1/people', () => jsonResponse(200, [])),
		get('/api/v1/people/unnamed?limit=60', () =>
			jsonResponse(200, { items: [unnamedTarget('c9', 4)] })
		),
		put('/api/v1/people/c-f2/label', () => {
			named = true;
			return jsonResponse(200, { clusterId: 'c-f2', name: 'Grace', isHidden: false });
		})
	]);
	await settle();
	openDrawer();

	// The labeled chip: no new-person mode (renaming an existing
	// person belongs to the people page) -- the picker only.
	click(element('[data-face-id="f1"]'));
	await settle();
	expect(document.body.querySelector('input[name="cluster-picker-mode"]')).toBeNull();
	click(buttonByText('Cancel'));
	await settle();
	expect(document.body.querySelector('[role="dialog"]')).toBeNull();

	// The unnamed chip: the new-person path with the family note.
	click(element('[data-face-id="f2"]'));
	await settle();
	const radios = Array.from(
		document.body.querySelectorAll<HTMLInputElement>('input[name="cluster-picker-mode"]')
	);
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
	expect(
		Array.from(document.body.querySelectorAll('.info__face-name')).map((node) => node.textContent)
	).toEqual(['Ada', 'Grace']);
	unmount(app);
	flushSync();
});

test('person events converge the chips through the refetch; unrelated events add no fetch', async () => {
	let labeled = false;
	const facesNow = () => [faceSummary('f2', labeled ? { label: { name: 'Ada', isHidden: false } } : {})];
	const { app, fetchMock } = mountViewer('a2', [
		get('/api/v1/assets/a2', () => jsonResponse(200, detailBodyOf(sequence[1]!, { faces: facesNow() })))
	]);
	await settle();
	openDrawer();
	expect(element('.info__face-name').textContent).toBe('Unnamed');
	const detailCalls = () =>
		fetchMock.mock.calls.filter(([input]) => String(input) === '/api/v1/assets/a2').length;
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

test('the drawer\'s file section renders the tags row: the caller\'s chips, the empty note, the assign affordance', async () => {
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () =>
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

	// The first part left the panel's open preference behind (R2-9):
	// drop it so the bare mount starts from the default (closed).
	localStorage.removeItem('lumemor:viewer:info-panel');
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
		get('/api/v1/assets/a2', () => {
			detailCalls += 1;
			return jsonResponse(
				200,
				detailBodyOf(sequence[1]!, { tags: detailCalls > 1 ? [tagEntry('t1', 'Beach')] : [] })
			);
		}),
		get('/api/v1/tags', () =>
			jsonResponse(200, [tagSummary('t1', 'Beach'), tagSummary('t2', 'Sunrise')])
		),
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
	radios.find((radio) => radio.value === 'new')?.dispatchEvent(
		new MouseEvent('click', { bubbles: true, cancelable: true })
	);
	radios.find((radio) => radio.value === 'new')?.dispatchEvent(
		new Event('change', { bubbles: true })
	);
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

test('the tags row\'s assign affordance is every viewer\'s own write (per-user state, never the owner\'s alone)', async () => {
	// A sharee's drawer: the description edit never renders (the
	// identity seam's read-only arm), but the tags affordance does
	// -- the tag write is the CALLER's own per-user row, and the
	// union deliberately carries it (B4).
	const { app } = mountViewer('a2', [
		get('/api/v1/assets/a2', () =>
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
	expect(info.textContent).not.toContain('Edit');
	expect(info.querySelector('.info__tag-chip')?.textContent?.trim()).toBe('Sunrise');
	expect(scopedButtonByText(info, 'Add tag')).toBeTruthy();

	unmount(app);
	flushSync();
});
