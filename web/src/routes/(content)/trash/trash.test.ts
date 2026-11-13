import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';

// The trash page: the bounded listing renders with each row's
// retention countdown and actions, the restore walks the pinned
// single-asset endpoint (its 404 counting as the already-gone
// success), the permanent delete confirms first and drops the row on
// both 204 and 404, and the SSE restored/deleted events converge
// through a re-fetched listing -- replay-safe, so a stale restored
// event for an asset that is still trashed keeps its row. The events
// module is mocked with a captured handler registry, the same
// pattern as the timeline page test.

const { handlers } = vi.hoisted(() => ({
	handlers: [] as Array<(event: unknown) => void>
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
	}
}));

const trashDefaults = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image' as const,
	originalFilename: 'IMG_0001.jpg',
	fileSize: 1000,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d65a2feaa0c55ad015',
	createdAt: '2026-10-29T09:12:00Z',
	takenAt: '2026-10-13T12:30:00Z',
	takenAtSource: 'exif' as const,
	isReady: true,
	width: 4000,
	height: 3000,
	durationMs: null,
	blurhash: 'UHL6aAof00WBWB00ayWB00',
	livePhotoVideoId: null,
	deletedAt: '2026-10-30T11:04:05Z'
};

function trashItem(id: string, overrides: Record<string, unknown> = {}) {
	return { id, remainingDays: 30, ...trashDefaults, ...overrides };
}

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, () => Response>;

function trashRoutes(items: unknown[], overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/trash': () => jsonResponse(200, { items }),
		...overrides
	};
}

function stubFetch(route: FetchRoute): typeof fetch {
	return async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
		const url = String(input);
		const answer = route[url];
		if (answer === undefined) {
			throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
		}
		return answer();
	};
}

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
	await tick();
}

function mountPage(): ReturnType<typeof mount> {
	const page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
}

function buttonByText(text: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

/** The row actions are icon-only (the icon-first rule): the name is the aria-label. */
function buttonByLabel(label: string, scope: ParentNode = document.body): HTMLButtonElement {
	const found = scope.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing button labeled ${label}`);
	}
	return found;
}

function cells(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.trash__cell'));
}

/** The text of every rendered countdown line. */
function dayLabels(): string[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.trash__days')).map(
		(node) => node.textContent ?? ''
	);
}

function clickRestore(index: number): void {
	// The actions sit under the tile (whose own checkbox button is
	// not the target); match the Restore button of the row's action
	// group by its aria-label (the row actions are icon-only).
	const actions = cells()[index]?.querySelector<HTMLElement>('.trash__actions');
	const restore = actions?.querySelector<HTMLButtonElement>('button[aria-label="Restore"]');
	if (restore === null || restore === undefined) {
		throw new Error(`missing restore button of cell ${index}`);
	}
	restore.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	vi.clearAllMocks();
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

test('the listing renders each row with its countdown and actions', async () => {
	const items = [
		trashItem('a1', { remainingDays: 30 }),
		trashItem('a2', { remainingDays: 2 }),
		trashItem('a3', { remainingDays: 0 })
	];
	vi.stubGlobal('fetch', stubFetch(trashRoutes(items)));
	const page = mountPage();
	await settle();

	expect(cells()).toHaveLength(3);
	// The one-row action bar: the title with the row count, one
	// divider under it, and the retention note BELOW the divider (it
	// explains the list, it is not an action).
	expect(document.body.querySelector('.trash__title')?.textContent?.replace(/\s+/g, ' ').trim()).toBe('Trash 3');
	// The count is one labeled image: the wording is the accessible
	// name, the visible half the glyph and the number.
	const count = document.body.querySelector('.trash__title-count');
	expect(count?.getAttribute('role')).toBe('img');
	expect(count?.getAttribute('aria-label')).toBe('3 photos');
	const note = document.body.querySelector('.trash__note');
	const bar = document.body.querySelector('.trash__toolbar');
	expect(bar?.className).toContain('border-b');
	expect(bar?.contains(note)).toBe(false);
	expect(note?.closest('.trash__content')?.className).toContain('pt-4');
	// The countdown is locale-aware plural with the zero arm.
	expect(dayLabels()).toEqual(['30 days left', '2 days left', 'Expiring soon']);
	// The badge's sr-only sentence joins the file to its countdown: the
	// chip alone never says which photo is leaving when.
	const badgeLabels = Array.from(document.body.querySelectorAll<HTMLElement>('p.sr-only')).map(
		(node) => node.textContent?.replace(/\s+/g, ' ').trim() ?? ''
	);
	expect(badgeLabels).toEqual([
		'IMG_0001.jpg, 30 days left',
		'IMG_0001.jpg, 2 days left',
		'IMG_0001.jpg, expiring soon'
	]);
	expect(() => buttonByLabel('Restore')).not.toThrow();
	expect(() => buttonByLabel('Delete permanently')).not.toThrow();
	// No selection machinery at all -- the wall-selection entry
	// included: a trashed photo is not in the caller's current
	// sight, so nothing here joins a memory.
	expect(document.body.querySelector('.tile__check')).toBeNull();
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(page);
	flushSync();
});

test('an empty trash renders the empty state', async () => {
	vi.stubGlobal('fetch', stubFetch(trashRoutes([])));
	const page = mountPage();
	await settle();

	expect(cells()).toHaveLength(0);
	expect(document.body.textContent).toContain('The trash is empty');

	unmount(page);
	flushSync();
});

test('a failed listing renders the mapped error with a retry', async () => {
	let failed = true;
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (String(input) === '/api/v1/trash' && failed) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return jsonResponse(200, { items: [trashItem('a1')] });
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(document.body.textContent).toContain('Something went wrong');
	expect(cells()).toHaveLength(0);

	failed = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(cells()).toHaveLength(1);

	unmount(page);
	flushSync();
});

test('restore walks the single-asset endpoint and drops the row', async () => {
	const restored: string[] = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (url === '/api/v1/trash') {
			return jsonResponse(200, { items: [trashItem('a1'), trashItem('a2')] });
		}
		if (init?.method === 'POST' && url === '/api/v1/trash/a1/restore') {
			restored.push(url);
			return new Response(null, { status: 204 });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	clickRestore(0);
	await settle();
	// The pinned choice: the single restore, not the batch surface.
	expect(restored).toEqual(['/api/v1/trash/a1/restore']);
	expect(cells()).toHaveLength(1);

	unmount(page);
	flushSync();
});

test("restore's 404 counts as success: the row left the trash already", async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (url === '/api/v1/trash') {
			return jsonResponse(200, { items: [trashItem('a1')] });
		}
		if (init?.method === 'POST') {
			return jsonResponse(404, { code: 'ASSET_NOT_FOUND' });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	clickRestore(0);
	await settle();
	// The expiry job purged the row before the request landed: no
	// error surface, the row simply goes.
	expect(cells()).toHaveLength(0);

	unmount(page);
	flushSync();
});

test('a failed restore surfaces the mapped message and keeps the row', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (url === '/api/v1/trash') {
			return jsonResponse(200, { items: [trashItem('a1')] });
		}
		if (init?.method === 'POST') {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	clickRestore(0);
	await settle();
	expect(cells()).toHaveLength(1);
	expect(document.body.querySelector('[role="alert"]')?.textContent).toContain('Something went wrong');

	unmount(page);
	flushSync();
});

test('the permanent delete confirms first, then drops the row', async () => {
	const deleted: string[] = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (url === '/api/v1/trash') {
			return jsonResponse(200, { items: [trashItem('a1', { originalFilename: 'IMG_0001.jpg' })] });
		}
		if (init?.method === 'DELETE' && url === '/api/v1/assets/a1') {
			deleted.push(url);
			return new Response(null, { status: 204 });
		}
		throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	// The confirm-first gate: the delete button opens the modal, the
	// row stays until the confirm.
	buttonByLabel('Delete permanently').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(document.body.textContent).toContain('Delete permanently?');
	expect(document.body.textContent).toContain('IMG_0001.jpg');
	expect(cells()).toHaveLength(1);

	// Cancel keeps the row and closes the modal.
	buttonByText('Cancel').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(cells()).toHaveLength(1);
	expect(document.body.textContent).not.toContain('Delete permanently?');

	// The confirm runs the DELETE and drops the row.
	buttonByLabel('Delete permanently').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	buttonByText('Delete forever').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(deleted).toEqual(['/api/v1/assets/a1']);
	expect(cells()).toHaveLength(0);

	unmount(page);
	flushSync();
});

test("the permanent delete's 404 drops the row too (already gone)", async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (url === '/api/v1/trash') {
			return jsonResponse(200, { items: [trashItem('a1')] });
		}
		if (init?.method === 'DELETE') {
			return jsonResponse(404, { code: 'ASSET_NOT_FOUND' });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Delete permanently').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	buttonByText('Delete forever').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(cells()).toHaveLength(0);

	unmount(page);
	flushSync();
});

test('a failed permanent delete keeps the row and the modal message', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		if (url === '/api/v1/trash') {
			return jsonResponse(200, { items: [trashItem('a1')] });
		}
		if (init?.method === 'DELETE') {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		throw new Error(`unexpected fetch ${url}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	buttonByLabel('Delete permanently').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	buttonByText('Delete forever').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	// The modal stays with the mapped failure; the row is untouched.
	expect(cells()).toHaveLength(1);
	expect(document.body.querySelector('[role="alert"]')?.textContent).toContain('Something went wrong');

	unmount(page);
	flushSync();
});

test('SSE restored and deleted events converge through the refetched listing', async () => {
	// The listing is the truth: another device restored a1 and the
	// expiry job purged a2, so the server's next listing carries
	// neither row -- the events trigger the re-fetch, the answer
	// drops the rows.
	let current = [trashItem('a1'), trashItem('a2')];
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (String(input) === '/api/v1/trash') {
			return jsonResponse(200, { items: current });
		}
		throw new Error(`unexpected fetch ${String(input)}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cells()).toHaveLength(2);

	type DeliveredEvent = { id: number; name: string; assetId: string; kind: 'image' | 'video' };
	const deliver = (event: DeliveredEvent): void => {
		for (const handler of handlers) {
			handler(event);
		}
	};

	// Another device restored a1; the row leaves only when the
	// re-fetched listing drops it.
	current = [trashItem('a2')];
	deliver({ id: 1, name: 'asset.restored', assetId: 'a1', kind: 'image' });
	await settle();
	expect(fetchMock).toHaveBeenCalledTimes(2);
	expect(cells()).toHaveLength(1);

	// The expiry job purged a2: the same convergence, the same
	// answer-decides rule.
	current = [];
	deliver({ id: 2, name: 'asset.deleted', assetId: 'a2', kind: 'image' });
	await settle();
	expect(fetchMock).toHaveBeenCalledTimes(3);
	expect(cells()).toHaveLength(0);

	unmount(page);
	flushSync();
});

test('a replayed stale restored event keeps the row that is still trashed', async () => {
	// The stream's replay contract: a fresh subscriber receives the
	// retained ring, so an OLD restored event arrives for an asset
	// that was restored and then re-trashed (the m2 story's exact
	// flow). The listing still returns the row, so the row stays --
	// the event is a claim, never state.
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		if (String(input) === '/api/v1/trash') {
			return jsonResponse(200, { items: [trashItem('a1')] });
		}
		throw new Error(`unexpected fetch ${String(input)}`);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cells()).toHaveLength(1);

	for (const handler of handlers) {
		handler({ id: 1, name: 'asset.restored', assetId: 'a1', kind: 'image' });
	}
	await settle();
	// The convergence probe re-fetched and the listing confirmed the
	// row is in the trash right now.
	expect(fetchMock).toHaveBeenCalledTimes(2);
	expect(cells()).toHaveLength(1);

	unmount(page);
	flushSync();
});

test('other asset events leave the rows alone', async () => {
	vi.stubGlobal('fetch', stubFetch(trashRoutes([trashItem('a1')])));
	const page = mountPage();
	await settle();

	for (const handler of handlers) {
		handler({ id: 1, name: 'asset.ready', assetId: 'a1', kind: 'image' });
	}
	flushSync();
	expect(cells()).toHaveLength(1);

	unmount(page);
	flushSync();
});
