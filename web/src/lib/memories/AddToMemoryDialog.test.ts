import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { sessionStatus } from '$lib/auth/session.svelte';
import type { AlbumSummary } from '$lib/api/albums';
import AddToMemoryDialog from './AddToMemoryDialog.svelte';

// The memory picker: the wall-selection entry's dialog. The page
// hands it the ids captured at open and the done continuation; the
// dialog owns the flow -- the caller's own memories fetched fresh on
// open (the pick list), the pick-then-add write (a row click POSTs
// the captured ids; the row flips to the accent added state and
// stays inert), the inline create-then-add write (the create row
// expands to the title input; POST /memories then the same items
// POST; the new row lands at the head already added), the honest
// {added, skipped} report (the skip-note honesty line), the keyboard
// (↑/↓ move the highlight, Enter activates the highlighted row), and
// the Done footer (closes and fires the wall's success arm -- the
// wall never converges on this flow; adding to a memory changes no
// photo's state, the report IS the outcome).

function memory(id: string, overrides: Partial<AlbumSummary> = {}): AlbumSummary {
	return {
		id,
		title: `Memory ${id}`,
		type: 'memory',
		isShared: false,
		coverAssetIds: [],
		itemCount: 0,
		updatedAt: '2026-11-01T10:00:00Z',
		callerRole: 'owner',
		...overrides
	};
}

const springFestival = memory('mem1', {
	title: 'Spring Festival 2026',
	coverAssetIds: ['a1'],
	itemCount: 12
});
const paris = memory('mem2', { title: 'Paris trip', coverAssetIds: ['p1'], itemCount: 28 });
// The album wall's own rows: the picker's list keeps the caller's
// own memories alone.
const myPhotos = memory('sync1', { title: 'Sync', type: 'sync', itemCount: 1204 });
const familyArchive = memory('al1', { title: 'Family archive', type: 'directory', itemCount: 2048 });
const sanya = memory('mem3', {
	title: 'Sanya trip',
	callerRole: 'viewer',
	owner: { id: 'u2', email: 'grace@example.com' }
});

let wallState: AlbumSummary[] = [myPhotos, springFestival, paris, familyArchive, sanya];

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

/** The writes the stub observed: url + body pairs. */
type Write = { url: string; body: unknown };

function pickerRoutes(overrides: FetchRoute = {}): FetchRoute {
	return {
		'/api/v1/albums': () => jsonResponse(200, { albums: wallState }),
		'/api/v1/memories': (init?: RequestInit) => {
			const body = JSON.parse(String(init?.body)) as { title: string };
			const created = memory('mem-new', { title: body.title, itemCount: 0 });
			wallState = [created, ...wallState];
			return jsonResponse(201, { ...created, skipped: [] });
		},
		'/api/v1/albums/mem1/items': () => jsonResponse(200, { added: 2, skipped: 1 }),
		'/api/v1/albums/mem2/items': () => jsonResponse(200, { added: 3, skipped: 0 }),
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

function mountDialog(ids: readonly string[] = ['a1', 'a2', 'a3']): {
	dialog: ReturnType<typeof mount>;
	onpass: ReturnType<typeof vi.fn>;
} {
	const onpass = vi.fn();
	const dialog = mount(AddToMemoryDialog, {
		target: document.body,
		props: { open: true, ids, onpass }
	});
	flushSync();
	return { dialog, onpass };
}

function rows(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.add-memory__row'));
}

function createRow(): HTMLElement {
	const found = document.body.querySelector<HTMLElement>('.add-memory__create-row');
	if (found === null) {
		throw new Error('missing create row');
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

function observedWrites(fetchMock: ReturnType<typeof vi.fn>): Write[] {
	return fetchMock.mock.calls
		.filter(([, init]) => init?.method === 'POST' || init?.method === 'DELETE')
		.map(([input, init]) => ({ url: String(input), body: JSON.parse(String(init?.body)) }));
}

beforeEach(async () => {
	document.body.textContent = '';
	vi.clearAllMocks();
	wallState = [myPhotos, springFestival, paris, familyArchive, sanya];
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

test('the picker lists the caller own memories alone and picks on a row click', async () => {
	const fetchMock = stubFetch(pickerRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const { dialog } = mountDialog();
	await settle();

	// The list: the caller's own memories alone (the album wall's
	// rows and the shared memory are not pick targets), each with the
	// thumb, the title and the honest count.
	expect(rows()).toHaveLength(2);
	expect(rows()[0]?.textContent).toContain('Spring Festival 2026');
	expect(rows()[0]?.textContent).toContain('12 photos');
	expect(rows()[0]?.querySelector('img')?.getAttribute('src')).toBe('/api/v1/assets/a1/thumbnail?size=tile');

	rows()[0]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	await settle();

	// The write ran over the captured ids; the row flipped to the
	// accent added state and the skip-note renders the report's face.
	expect(observedWrites(fetchMock)).toEqual([
		{ url: '/api/v1/albums/mem1/items', body: { assetIds: ['a1', 'a2', 'a3'] } }
	]);
	expect(rows()[0]?.textContent).toContain('Added');
	expect(document.body.querySelector('.add-memory__skip-note')?.textContent).toContain(
		'1 photo is not visible right now'
	);

	// The added row is inert: a re-click writes nothing.
	rows()[0]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	await settle();
	expect(observedWrites(fetchMock)).toHaveLength(1);

	unmount(dialog);
	flushSync();
});

test('the inline create row expands, creates and adds in one flow', async () => {
	const fetchMock = stubFetch(
		pickerRoutes({ '/api/v1/albums/mem-new/items': () => jsonResponse(200, { added: 3, skipped: 0 }) })
	);
	vi.stubGlobal('fetch', fetchMock);
	const { dialog } = mountDialog();
	await settle();

	// The create row expands into the inline form.
	createRow().dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();
	const input = document.body.querySelector<HTMLInputElement>('#add-memory-create-title');
	if (input === null) {
		throw new Error('missing create input');
	}
	input.value = 'Camping 2026';
	input.dispatchEvent(new Event('input', { bubbles: true }));
	buttonByText('Create and add').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The create POST carried the title alone; the items POST
	// followed; the new row landed at the head already added.
	expect(observedWrites(fetchMock)).toEqual([
		{ url: '/api/v1/memories', body: { title: 'Camping 2026' } },
		{ url: '/api/v1/albums/mem-new/items', body: { assetIds: ['a1', 'a2', 'a3'] } }
	]);
	expect(rows()[0]?.textContent).toContain('Camping 2026');
	expect(rows()[0]?.textContent).toContain('Added');
	// The all-landed report renders no skip-note.
	expect(document.body.querySelector('.add-memory__skip-note')).toBeNull();

	unmount(dialog);
	flushSync();
});

test('an invalid create draft never leaves the page', async () => {
	const fetchMock = stubFetch(pickerRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const { dialog } = mountDialog();
	await settle();

	createRow().dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();
	const input = document.body.querySelector<HTMLInputElement>('#add-memory-create-title');
	if (input === null) {
		throw new Error('missing create input');
	}
	input.value = '   ';
	input.dispatchEvent(new Event('input', { bubbles: true }));
	buttonByText('Create and add').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The bound message renders inline; nothing was written.
	expect(document.body.querySelector('.add-memory__bound')?.textContent).toContain('Titles span 1 to 64 characters.');
	expect(observedWrites(fetchMock)).toHaveLength(0);

	unmount(dialog);
	flushSync();
});

test('the keyboard walks the rows and Enter picks the highlighted one', async () => {
	const fetchMock = stubFetch(pickerRoutes());
	vi.stubGlobal('fetch', fetchMock);
	const { dialog } = mountDialog();
	await settle();

	// The first row is highlighted by default; ArrowDown moves to the
	// second, Enter picks it.
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
	flushSync();
	expect(rows()[1]?.getAttribute('aria-selected')).toBe('true');
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
	await settle();

	expect(observedWrites(fetchMock)).toEqual([
		{ url: '/api/v1/albums/mem2/items', body: { assetIds: ['a1', 'a2', 'a3'] } }
	]);

	// ArrowDown onto the create row: Enter expands the form.
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
	flushSync();
	expect(createRow().getAttribute('aria-selected')).toBe('true');
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
	flushSync();
	expect(document.body.querySelector('#add-memory-create-title')).not.toBeNull();

	// The create input keeps its own keys: Enter there never picks.
	inputEnter();
	expect(observedWrites(fetchMock)).toHaveLength(1);

	unmount(dialog);
	flushSync();
});

/** Press Enter while the create input holds the focus. */
function inputEnter(): void {
	const input = document.body.querySelector<HTMLInputElement>('#add-memory-create-title');
	input?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
	flushSync();
}

test('a failed write maps the error and keeps the dialog', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			pickerRoutes({
				'/api/v1/albums/mem1/items': () =>
					jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'the sharee cannot curate' })
			})
		)
	);
	const { dialog } = mountDialog();
	await settle();

	rows()[0]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	await settle();

	expect(document.body.querySelector('.add-memory__failure')?.textContent).toBe(
		"You don't have permission to do this."
	);
	// The row did not flip: the write never landed.
	expect(rows()[0]?.textContent).not.toContain('Added');

	unmount(dialog);
	flushSync();
});

test('the Done footer fires the pass and closes; Cancel keeps it silent', async () => {
	const first = mountDialog(['a1']);
	await settle();

	// Cancel: the wall's continuation never fires (the selection
	// stays as the retry affordance).
	buttonByText('Cancel').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(first.onpass).not.toHaveBeenCalled();
	unmount(first.dialog);
	flushSync();

	// Done: the success arm fires (the report's acknowledge).
	const second = mountDialog(['a1']);
	await settle();
	buttonByText('Done').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(second.onpass).toHaveBeenCalledTimes(1);

	unmount(second.dialog);
	flushSync();
});

test('the list failure surfaces the mapped error and no rows', async () => {
	vi.stubGlobal('fetch', stubFetch({ '/api/v1/albums': () => jsonResponse(500, { code: 'INTERNAL' }) }));
	const { dialog } = mountDialog();
	await settle();

	expect(rows()).toHaveLength(0);
	expect(document.body.querySelector('.add-memory__failure')?.textContent).toBe(
		'Something went wrong. Please try again.'
	);

	unmount(dialog);
	flushSync();
});
