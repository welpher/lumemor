import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { clearToasts, toasts } from '$lib/components/toast.svelte';
import type { AlbumDetail, AlbumSummary } from '$lib/api/albums';
import type { TimelineItem } from '$lib/api/timeline';
import MemoryOverlay from './MemoryOverlay.svelte';

// The memory playback overlay: the reference set's own face, mounted
// over the memories page from `?memory=<id>`. These tests pin the
// machinery the page does not own -- the four-second advance and every
// stop the design names, the two faces (the play stage and the grid
// with its always-armed selection and the removal's reference-only
// semantics), the owner's arms and the member's read-only bar, the
// dialogs riding inside the overlay, the one dead end, and the focus
// trap. The events module is mocked with a captured registry; the
// fetch stub answers the detail read and the memory's item walk.
//
// THE REACHABILITY PIN: the memories wall is the authority. A settled
// wall that omits the id answers the same dead-end card as an unknown,
// foreign or withdrawn one -- no title, no count, no cover.

const { albumHandlers, assetHandlers } = vi.hoisted(() => ({
	albumHandlers: [] as Array<(event: unknown) => void>,
	assetHandlers: [] as Array<(event: unknown) => void>
}));

vi.mock('$lib/events', () => ({
	subscribeAlbumEvents: (listener: (event: unknown) => void) => {
		albumHandlers.push(listener);
		return () => {
			const index = albumHandlers.indexOf(listener);
			if (index !== -1) {
				albumHandlers.splice(index, 1);
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

const MEMORY_ID = '0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6071';

const itemDefaults = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image' as const,
	originalFilename: 'IMG_0001.jpg',
	fileSize: 1000,
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

function item(id: string, filename: string, takenAt: string): TimelineItem {
	return { ...itemDefaults, id, originalFilename: filename, takenAt } as TimelineItem;
}

const walkItems = [
	item('p1', 'IMG_0001.jpg', '2026-09-10T12:00:00Z'),
	item('p2', 'IMG_0002.jpg', '2026-09-09T12:00:00Z'),
	item('p3', 'IMG_0003.jpg', '2026-09-08T12:00:00Z')
];

const ownerCard: AlbumSummary = {
	id: MEMORY_ID,
	title: 'Late summer 2026',
	type: 'memory',
	isShared: false,
	coverAssetIds: ['p1'],
	itemCount: 3,
	updatedAt: '2026-09-10T12:00:00Z',
	callerRole: 'owner'
};

const sharedCard: AlbumSummary = {
	...ownerCard,
	isShared: true,
	callerRole: 'viewer',
	owner: { id: 'u2', name: 'Grace Hopper', email: 'grace@example.com' },
	inTimeline: true
};

function ownerDetail(overrides: Partial<AlbumDetail> = {}): AlbumDetail {
	return {
		id: MEMORY_ID,
		title: 'Late summer 2026',
		type: 'memory',
		coverAssetId: null,
		callerRole: 'owner',
		itemCount: 3,
		items: [],
		shares: [{ userId: 'u2', email: 'grace@example.com', grantedAt: '2026-09-11T12:00:00Z' }],
		...overrides
	};
}

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

type FetchRoute = Record<string, (init?: RequestInit) => Response>;

function routes(overrides: FetchRoute = {}): FetchRoute {
	return {
		[`/api/v1/albums/${MEMORY_ID}?limit=1`]: () => jsonResponse(200, ownerDetail()),
		[`/api/v1/assets/timeline?albumId=${MEMORY_ID}&limit=100`]: () => jsonResponse(200, { items: walkItems }),
		'/api/v1/users': () => jsonResponse(200, { users: [] }),
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
	for (let beat = 0; beat < 4; beat += 1) {
		await new Promise((resolve) => setTimeout(resolve, 0));
	}
	flushSync();
}

type OverlayProps = {
	open?: boolean;
	memoryId?: string;
	card?: AlbumSummary | null;
	wallStatus?: 'idle' | 'loading' | 'ready' | 'error';
	wallErrorKey?: string;
	ownerTitle?: string;
	mode?: 'play' | 'grid';
};

function mountOverlay(props: OverlayProps = {}): {
	app: ReturnType<typeof mount>;
	onclose: ReturnType<typeof vi.fn>;
	onmodechange: ReturnType<typeof vi.fn>;
} {
	const onclose = vi.fn();
	const onmodechange = vi.fn();
	const app = mount(MemoryOverlay, {
		target: document.body,
		props: {
			open: true,
			memoryId: MEMORY_ID,
			card: ownerCard,
			wallStatus: 'ready',
			mode: 'play',
			onclose,
			onmodechange,
			...props
		}
	});
	flushSync();
	return { app, onclose, onmodechange };
}

function dialog(): HTMLElement {
	const found = document.body.querySelector<HTMLElement>('.memory-overlay__panel');
	if (found === null) {
		throw new Error('missing overlay');
	}
	return found;
}

/** The control whose accessible name is its aria-label (the icon actions). */
function buttonByLabel(name: string): HTMLButtonElement {
	const found = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.getAttribute('aria-label') === name
	);
	if (found === undefined) {
		throw new Error(`missing button labelled ${name}`);
	}
	return found;
}

/** The control whose accessible name is its visible text (the labelled actions). */
function buttonByText(name: string, scope: ParentNode = document.body): HTMLButtonElement {
	const found = Array.from(scope.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.textContent?.trim() === name
	);
	if (found === undefined) {
		throw new Error(`missing button saying ${name}`);
	}
	return found;
}

beforeEach(async () => {
	document.body.textContent = '';
	albumHandlers.length = 0;
	assetHandlers.length = 0;
	clearToasts();
	await initI18n();
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
});

test('the overlay opens on the card and plays at once: the bar, the count, the first frame', async () => {
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app } = mountOverlay();
	await settle();

	// The opaque dialog: no scrim, the whole stage, the memory named.
	const panel = dialog();
	expect(panel.getAttribute('role')).toBe('dialog');
	expect(panel.getAttribute('aria-modal')).toBe('true');
	expect(panel.getAttribute('aria-label')).toBe('Memory playback');
	expect(panel.querySelector('.memory-overlay__title')?.textContent?.trim()).toBe('Late summer 2026');
	expect(panel.querySelector('.memory-overlay__count')?.getAttribute('aria-label')).toBe('3 photos');

	// The first frame, the view derivative, the honest position.
	const frame = panel.querySelector<HTMLImageElement>('.memory-overlay__frame img');
	expect(frame?.getAttribute('src')).toBe('/api/v1/assets/p1/thumbnail?size=view');
	expect(panel.querySelector('.memory-overlay__index')?.textContent?.replace(/\s+/g, ' ').trim()).toBe('1 / 3');

	// The advance started on open (the design's autoplay), and the
	// switch reads as the pause it now offers.
	expect(panel.querySelector('.memory-overlay__play')?.getAttribute('aria-pressed')).toBe('true');
	expect(panel.querySelector('.memory-overlay__play')?.getAttribute('aria-label')).toBe('Pause');

	unmount(app);
	flushSync();
});

test('the four-second advance steps once per beat and stops at the last photo', async () => {
	vi.useFakeTimers();
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app } = mountOverlay();
	await vi.advanceTimersByTimeAsync(10);
	flushSync();

	const index = (): string =>
		dialog().querySelector('.memory-overlay__index')?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
	expect(index()).toBe('1 / 3');

	await vi.advanceTimersByTimeAsync(4000);
	flushSync();
	expect(index()).toBe('2 / 3');

	await vi.advanceTimersByTimeAsync(4000);
	flushSync();
	expect(index()).toBe('3 / 3');
	// The last photo stops the run: no loop back to the start.
	expect(dialog().querySelector('.memory-overlay__play')?.getAttribute('aria-label')).toBe('Play');
	await vi.advanceTimersByTimeAsync(8000);
	flushSync();
	expect(index()).toBe('3 / 3');

	unmount(app);
	flushSync();
});

test('a manual step stops the run and only the play switch restarts it', async () => {
	vi.useFakeTimers();
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app } = mountOverlay();
	await vi.advanceTimersByTimeAsync(10);
	flushSync();

	// The arrow steps ONE photo and stops: no advance follows the step.
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
	flushSync();
	const index = (): string =>
		dialog().querySelector('.memory-overlay__index')?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
	expect(index()).toBe('2 / 3');
	expect(dialog().querySelector('.memory-overlay__play')?.getAttribute('aria-label')).toBe('Play');
	await vi.advanceTimersByTimeAsync(4000);
	flushSync();
	expect(index()).toBe('2 / 3');

	// The user's own restart runs again from here.
	buttonByLabel('Play').click();
	flushSync();
	expect(dialog().querySelector('.memory-overlay__play')?.getAttribute('aria-label')).toBe('Pause');
	await vi.advanceTimersByTimeAsync(4000);
	flushSync();
	expect(index()).toBe('3 / 3');

	unmount(app);
	flushSync();
});

test('the space key toggles the run', async () => {
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app } = mountOverlay();
	await settle();

	window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true, cancelable: true }));
	flushSync();
	expect(dialog().querySelector('.memory-overlay__play')?.getAttribute('aria-label')).toBe('Play');
	window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true, cancelable: true }));
	flushSync();
	expect(dialog().querySelector('.memory-overlay__play')?.getAttribute('aria-label')).toBe('Pause');

	unmount(app);
	flushSync();
});

test('the tab going to the background stops the run (no spinning behind it)', async () => {
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app } = mountOverlay();
	await settle();
	expect(dialog().querySelector('.memory-overlay__play')?.getAttribute('aria-label')).toBe('Pause');

	Object.defineProperty(document, 'hidden', { configurable: true, value: true });
	document.dispatchEvent(new Event('visibilitychange'));
	flushSync();
	expect(dialog().querySelector('.memory-overlay__play')?.getAttribute('aria-label')).toBe('Play');

	Object.defineProperty(document, 'hidden', { configurable: true, value: false });
	unmount(app);
	flushSync();
});

test('the grid face selects with one click and the batch bar counts the picks', async () => {
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app, onmodechange } = mountOverlay({ mode: 'grid' });
	await settle();

	// The grid face: every member as one square cell, no stage.
	const cells = Array.from(document.body.querySelectorAll<HTMLElement>('.memory-overlay__cell'));
	expect(cells).toHaveLength(3);
	expect(document.body.querySelector('.memory-overlay__frame')).toBeNull();
	expect(cells.every((cell) => cell.getAttribute('aria-pressed') === 'false')).toBe(true);

	// The selection is always armed: one click picks, a second drops.
	cells[0]?.click();
	flushSync();
	expect(cells[0]?.getAttribute('aria-pressed')).toBe('true');
	const bar = document.body.querySelector('[role="toolbar"]');
	expect(bar?.getAttribute('aria-label')).toBe('1 selected');
	cells[1]?.click();
	flushSync();
	expect(bar?.getAttribute('aria-label')).toBe('2 selected');

	// Cancel drops the selection and the bar with it.
	buttonByLabel('Clear selection').click();
	flushSync();
	expect(document.body.querySelector('[role="toolbar"]')).toBeNull();
	expect(cells[0]?.getAttribute('aria-pressed')).toBe('false');

	// The face switch is the page's write (the URL owns the state).
	buttonByText('Play', document.body.querySelector('.memory-overlay__modes') ?? document.body).click();
	flushSync();
	expect(onmodechange).toHaveBeenCalledWith('play');

	unmount(app);
	flushSync();
});

test('the removal ends references: the DELETE carries the picks, then the toast and the fresh count', async () => {
	let removed = false;
	vi.stubGlobal(
		'fetch',
		stubFetch(
			routes({
				[`/api/v1/assets/timeline?albumId=${MEMORY_ID}&limit=100`]: () =>
					jsonResponse(200, { items: removed ? walkItems.slice(0, 1) : walkItems }),
				[`/api/v1/albums/${MEMORY_ID}?limit=1`]: () =>
					jsonResponse(200, ownerDetail({ itemCount: removed ? 1 : 3 })),
				[`/api/v1/albums/${MEMORY_ID}/items`]: (init?: RequestInit) => {
					expect(init?.method).toBe('DELETE');
					expect(JSON.parse(String(init?.body))).toEqual({ assetIds: ['p1', 'p2'] });
					removed = true;
					return jsonResponse(200, { removed: 2, skipped: 0 });
				}
			})
		)
	);
	const { app } = mountOverlay({ mode: 'grid' });
	await settle();

	const cells = Array.from(document.body.querySelectorAll<HTMLElement>('.memory-overlay__cell'));
	cells[0]?.click();
	cells[1]?.click();
	flushSync();
	buttonByLabel('Remove from memory').click();
	await settle();

	// The success arm: the selection clears, the toast confirms, and
	// the count converges on the server's fresh answer.
	expect(document.body.querySelector('[role="toolbar"]')).toBeNull();
	expect(toasts().map((entry) => entry.text)).toContain('Removed from this memory.');
	expect(dialog().querySelector('.memory-overlay__count')?.getAttribute('aria-label')).toBe('1 photo');
	expect(document.body.querySelectorAll('.memory-overlay__cell')).toHaveLength(1);

	unmount(app);
	flushSync();
});

test('a failed removal keeps the selection and maps the error in the bar', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			routes({
				[`/api/v1/albums/${MEMORY_ID}/items`]: () => jsonResponse(500, { code: 'INTERNAL' })
			})
		)
	);
	const { app } = mountOverlay({ mode: 'grid' });
	await settle();

	document.body.querySelectorAll<HTMLElement>('.memory-overlay__cell')[0]?.click();
	flushSync();
	buttonByLabel('Remove from memory').click();
	await settle();

	// The retry affordance: the pick stays, the mapped line explains.
	const bar = document.body.querySelector('[role="toolbar"]');
	expect(bar?.getAttribute('aria-label')).toBe('1 selected');
	expect(bar?.querySelector('[role="alert"]')?.textContent).toContain('Something went wrong');

	unmount(app);
	flushSync();
});

test('an empty memory offers the state card with the play disabled', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			routes({
				[`/api/v1/assets/timeline?albumId=${MEMORY_ID}&limit=100`]: () => jsonResponse(200, { items: [] }),
				[`/api/v1/albums/${MEMORY_ID}?limit=1`]: () => jsonResponse(200, ownerDetail({ itemCount: 0 }))
			})
		)
	);
	const { app } = mountOverlay();
	await settle();

	const panel = dialog();
	expect(panel.querySelector('.state-card__title')?.textContent).toBe('This memory has no photos yet');
	const play = panel.querySelector<HTMLButtonElement>('.memory-overlay__play');
	expect(play?.disabled).toBe(true);
	expect(play?.getAttribute('aria-label')).toBe('Play');

	unmount(app);
	flushSync();
});

test('the settled wall without the id answers the dead end with no title and no actions', async () => {
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app, onclose } = mountOverlay({ card: null, wallStatus: 'ready' });
	await settle();

	const panel = dialog();
	expect(panel.querySelector('.unreachable-card')).not.toBeNull();
	// Nothing resource-shaped: no title, no count, no cover, no face
	// switch, no arms. The bar keeps its way back alone.
	expect(panel.querySelector('.memory-overlay__title')).toBeNull();
	expect(panel.querySelector('.memory-overlay__count')).toBeNull();
	expect(panel.querySelector('.memory-overlay__modes')).toBeNull();
	expect(panel.querySelector('.memory-overlay__frame')).toBeNull();
	expect(panel.querySelector('img')).toBeNull();
	buttonByLabel('Back to memories').click();
	expect(onclose).toHaveBeenCalledTimes(1);

	unmount(app);
	flushSync();
});

test('a wall still loading shows the one loading line', async () => {
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app } = mountOverlay({ card: null, wallStatus: 'loading' });
	await settle();

	expect(dialog().querySelector('[role="status"]')?.textContent).toContain('Loading your memories');

	unmount(app);
	flushSync();
});

test('the detail read failure is a retryable card, never the dead end', async () => {
	let failing = true;
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			const url = String(input);
			if (url.startsWith(`/api/v1/albums/${MEMORY_ID}?`)) {
				return failing ? jsonResponse(500, { code: 'INTERNAL' }) : jsonResponse(200, ownerDetail());
			}
			const answer = routes()[url];
			if (answer === undefined) {
				throw new Error(`unexpected fetch ${url}`);
			}
			return answer(init);
		})
	);
	const { app } = mountOverlay();
	await settle();

	const error = dialog().querySelector('.state-card--error');
	expect(error?.textContent).toContain('Something went wrong');
	expect(dialog().querySelector('.unreachable-card')).toBeNull();

	failing = false;
	buttonByText('Try again').click();
	await settle();
	expect(dialog().querySelector('.state-card--error')).toBeNull();
	expect(dialog().querySelector('.memory-overlay__title')?.textContent?.trim()).toBe('Late summer 2026');

	unmount(app);
	flushSync();
});

test("the owner's bar carries the three arms; the member's the attribution and the badge alone", async () => {
	vi.stubGlobal('fetch', stubFetch(routes()));
	const ownerMount = mountOverlay();
	await settle();
	for (const label of ['Edit memory', 'Change cover', 'Delete memory']) {
		expect(buttonByLabel(label)).toBeTruthy();
	}
	unmount(ownerMount.app);
	flushSync();
	document.body.textContent = '';

	vi.stubGlobal(
		'fetch',
		stubFetch(
			routes({
				[`/api/v1/albums/${MEMORY_ID}?limit=1`]: () =>
					jsonResponse(
						200,
						ownerDetail({
							callerRole: 'viewer',
							shares: undefined,
							inTimeline: true
						})
					)
			})
		)
	);
	const memberMount = mountOverlay({ card: sharedCard });
	await settle();

	const panel = dialog();
	expect(panel.querySelector('.memory-overlay__owner')?.textContent?.trim()).toBe('Shared by Grace Hopper');
	expect(panel.querySelector('.memory-overlay__badge')?.textContent?.trim()).toBe('Read-only');
	for (const label of ['Edit memory', 'Change cover', 'Delete memory']) {
		expect(panel.querySelector(`[aria-label="${label}"]`)).toBeNull();
	}

	unmount(memberMount.app);
	flushSync();
});

test('Escape closes the overlay and Tab stays inside it', async () => {
	vi.stubGlobal('fetch', stubFetch(routes()));
	const { app, onclose } = mountOverlay();
	await settle();

	// The focus landed on the panel (the trap's home).
	expect(document.activeElement).toBe(dialog());

	// Tab walks the panel's own controls: from the last one it wraps
	// to the first instead of escaping behind the stage.
	const buttons = Array.from(dialog().querySelectorAll<HTMLButtonElement>('button:not([disabled])'));
	const first = buttons[0];
	const last = buttons[buttons.length - 1];
	last?.focus();
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
	flushSync();
	expect(document.activeElement).toBe(first);
	first?.focus();
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true }));
	flushSync();
	expect(document.activeElement).toBe(last);

	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }));
	expect(onclose).toHaveBeenCalledTimes(1);

	unmount(app);
	flushSync();
});

test('the delete confirm states the light cascade and closes the overlay on success', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			routes({
				[`/api/v1/albums/${MEMORY_ID}`]: (init?: RequestInit) => {
					expect(init?.method).toBe('DELETE');
					return new Response(null, { status: 204 });
				}
			})
		)
	);
	const { app, onclose } = mountOverlay();
	await settle();

	buttonByLabel('Delete memory').click();
	flushSync();
	const modal = document.body.querySelector<HTMLElement>('.modal__panel');
	if (modal === null) {
		throw new Error('missing delete confirm');
	}
	// The semantics before the irreversible act: the references and
	// the audience go, the photos themselves stay.
	expect(modal.textContent).toContain('The photos themselves stay in your library');
	expect(modal.textContent).toContain('Late summer 2026');

	const confirm = Array.from(modal.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.textContent?.trim() === 'Delete memory'
	);
	confirm?.click();
	await settle();

	expect(onclose).toHaveBeenCalledTimes(1);

	unmount(app);
	flushSync();
});

test("a shared memory's bar carries the caller's own timeline switch and flips on the response", async () => {
	let inTimeline = true;
	const seen: Array<{ url: string; body: unknown }> = [];
	vi.stubGlobal(
		'fetch',
		stubFetch(
			routes({
				[`/api/v1/albums/${MEMORY_ID}?limit=1`]: () =>
					jsonResponse(200, ownerDetail({ callerRole: 'viewer', shares: undefined, inTimeline })),
				[`/api/v1/albums/${MEMORY_ID}/timeline-preference`]: (init?: RequestInit) => {
					expect(init?.method).toBe('PUT');
					const body = JSON.parse(String(init?.body)) as { inTimeline: boolean };
					seen.push({ url: `/api/v1/albums/${MEMORY_ID}/timeline-preference`, body });
					inTimeline = body.inTimeline;
					return jsonResponse(200, { albumId: MEMORY_ID, inTimeline });
				}
			})
		)
	);
	const { app } = mountOverlay({ card: sharedCard });
	await settle();

	// In the timeline: the action offers the exclusion (the glyph's
	// state-following label, the shared album's own wording).
	const exclude = buttonByLabel('Keep out of timeline');
	exclude.click();
	await settle();

	// The write is the caller's own preference (no event published);
	// the page flips from the response's stored switch.
	expect(seen).toEqual([{ url: `/api/v1/albums/${MEMORY_ID}/timeline-preference`, body: { inTimeline: false } }]);
	const include = buttonByLabel('Keep in timeline');
	include.click();
	await settle();
	expect(seen[1]?.body).toEqual({ inTimeline: true });

	unmount(app);
	flushSync();
});
