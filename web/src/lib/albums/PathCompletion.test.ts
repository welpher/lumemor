import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import PathCompletion from './PathCompletion.svelte';

// The source-path completion combobox: the typed prefix in, the
// pickable directories out, the list a PORTALED floating layer
// (§7.5's 2026-11-09 second revision: fixed, anchored under the
// input through the shared `anchored` action, the dialog never
// changing height for it). The request discipline: debounced
// ~250ms, single-flight (a burst parks behind the in-flight
// request and ONE trailing fetch runs when it lands), and a stale
// answer never overwrites a newer one (the generation guard). The
// keyboard: ↑/↓ highlight, Enter completes, Esc closes the list
// alone. The honest faces: the empty prefix lists the container's
// root itself (E6's system-wide domain -- the allowlist and its
// no-roots posture retired with §7.5's third revision), an empty
// answer renders the plain no-match note, and the completion's own
// failure renders as a line that points back at the completion.
// The validation slot under the field swaps the hint and the
// failure at ONE height (§7.5's companion pin). Every case drives
// the component through an injected fetch mock under fake timers
// (the debounce is the unit under observation).

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Mount the combobox with a captured path binding. */
function mountField(props: { path?: string; failure?: string; hint?: string } = {}) {
	const state = { path: props.path ?? '', failure: props.failure ?? '', hint: props.hint ?? '' };
	const field = mount(PathCompletion, {
		target: document.body,
		props: {
			path: state.path,
			label: 'Path',
			id: 'test-path',
			placeholder: '/Volumes/Photos',
			hint: state.hint,
			failure: state.failure,
			onsubmit: undefined
		}
	});
	const component = field;
	// Svelte 5 mount returns the component instance; the bindable is
	// read back off the input.
	const input = () => document.body.querySelector<HTMLInputElement>('#test-path');
	const readPath = () => input()?.value ?? '';
	return { component, input, readPath };
}

beforeEach(async () => {
	document.body.textContent = '';
	vi.useFakeTimers();
	await initI18n();
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
});

/** Advance the fake clock and flush the microtask/DOM queue. */
async function advance(ms: number): Promise<void> {
	await vi.advanceTimersByTimeAsync(ms);
	flushSync();
}

/** Type one input event into the field. */
function type(value: string): void {
	const field = document.body.querySelector<HTMLInputElement>('#test-path');
	if (field === null) {
		throw new Error('missing path input');
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

/** Focus the field (the roots listing's trigger). */
function focus(): void {
	const field = document.body.querySelector<HTMLInputElement>('#test-path');
	field?.dispatchEvent(new Event('focus', { bubbles: true }));
}

/** The rendered suggestion rows' paths. */
function rows(): string[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.path-completion__row-path')).map(
		(row) => row.textContent ?? ''
	);
}

function keysFooter(): HTMLElement | null {
	return document.body.querySelector<HTMLElement>('.path-completion__keys');
}

test('a burst of keystrokes debounces into ONE request with the final prefix', async () => {
	const fetchMock = vi.fn(async (_input: RequestInfo | URL) => jsonResponse(200, { paths: ['/Volumes/Photos'] }));
	vi.stubGlobal('fetch', fetchMock);
	const { component } = mountField();

	type('/V');
	await advance(100);
	type('/Vo');
	await advance(100);
	type('/Vol');
	await advance(300);

	// One request, carrying the final prefix only.
	expect(fetchMock).toHaveBeenCalledTimes(1);
	expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/albums/path-completions?prefix=%2FVol');
	expect(rows()).toEqual(['/Volumes/Photos']);
	// The keys footer rides the row list (the specimen's language).
	expect(keysFooter()?.textContent).toContain('↑↓ Select');
	expect(keysFooter()?.textContent).toContain('Enter Complete');

	unmount(component);
	flushSync();
});

test('single-flight: an in-flight request parks later triggers, one trailing fetch converges', async () => {
	let held: ((paths: string[]) => void) | undefined;
	const fetchMock = vi.fn(async (_input: RequestInfo | URL) => {
		return new Promise<Response>((resolve) => {
			held = (paths: string[]) => resolve(jsonResponse(200, { paths }));
		});
	});
	vi.stubGlobal('fetch', fetchMock);
	const { component } = mountField();

	type('/a');
	await advance(300);
	expect(fetchMock).toHaveBeenCalledTimes(1);

	// Two more keystrokes land while the first request is in flight:
	// both park behind it -- no second request fires yet.
	type('/ab');
	await advance(300);
	type('/abc');
	await advance(300);
	expect(fetchMock).toHaveBeenCalledTimes(1);

	// The in-flight answer lands (the older prefix's rows, the
	// single-flight's honest transient) and exactly ONE trailing
	// fetch runs for the newest prefix.
	held?.(['/a-folder']);
	await advance(300);
	expect(fetchMock).toHaveBeenCalledTimes(2);
	expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/v1/albums/path-completions?prefix=%2Fabc');

	// The trailing answer owns the final state: the stale rows never
	// survive it.
	held?.(['/abc-folder']);
	await advance(0);
	expect(rows()).toEqual(['/abc-folder']);

	unmount(component);
	flushSync();
});

test('the keyboard: ↑/↓ highlight, Enter completes into the field, Esc closes the list alone', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, { paths: ['/Volumes/Photos', '/Volumes/PhoneBackup'] }));
	vi.stubGlobal('fetch', fetchMock);
	const { component, input, readPath } = mountField();

	type('/Vol');
	await advance(300);
	expect(rows()).toEqual(['/Volumes/Photos', '/Volumes/PhoneBackup']);
	// The first row is highlighted on arrival.
	const options = () => Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]'));
	expect(options()[0]?.getAttribute('aria-selected')).toBe('true');

	// ↓ moves the highlight to the second row.
	input()?.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
	flushSync();
	expect(options()[0]?.getAttribute('aria-selected')).toBe('false');
	expect(options()[1]?.getAttribute('aria-selected')).toBe('true');

	// ↑ moves it back (clamped, never below the first row).
	input()?.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }));
	flushSync();
	expect(options()[0]?.getAttribute('aria-selected')).toBe('true');

	// Enter completes the highlighted row into the field and closes
	// the list.
	input()?.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
	input()?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
	flushSync();
	expect(readPath()).toBe('/Volumes/PhoneBackup');
	expect(document.body.querySelector('.path-completion__suggest')).toBeNull();

	// Esc on a closed list is a no-op here (the dialog's own Escape
	// owns that stroke); on an open list it closes the list alone.
	type('/Vol');
	await advance(300);
	expect(document.body.querySelector('.path-completion__suggest')).not.toBeNull();
	input()?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
	flushSync();
	expect(document.body.querySelector('.path-completion__suggest')).toBeNull();
	// The draft keeps its text -- Esc closes, never clears.
	expect(readPath()).toBe('/Vol');

	unmount(component);
	flushSync();
});

test('a row completes on mousedown (the state-selector semantics)', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, { paths: ['/Volumes/Photos', '/Volumes/PhoneBackup'] }));
	vi.stubGlobal('fetch', fetchMock);
	const { component, readPath } = mountField();

	type('/Vol');
	await advance(300);
	const row = document.body.querySelectorAll<HTMLElement>('[role="option"]')[1];
	row?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();

	expect(readPath()).toBe('/Volumes/PhoneBackup');
	expect(document.body.querySelector('.path-completion__suggest')).toBeNull();

	unmount(component);
	flushSync();
});

test('the empty prefix lists the root itself; the honest empty answer is the plain no-match note (E6: the no-roots face retired)', async () => {
	// The system-wide domain: the empty prefix carries no query
	// string and the server answers the container's root.
	const fetchMock = vi.fn(async (_input: RequestInfo | URL) => jsonResponse(200, { paths: ['/'] }));
	vi.stubGlobal('fetch', fetchMock);
	const { component } = mountField();

	focus();
	await advance(300);
	expect(fetchMock).toHaveBeenCalledTimes(1);
	expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/albums/path-completions');
	expect(rows()).toEqual(['/']);

	unmount(component);
	flushSync();

	// The same empty answer (whatever the prefix): the plain
	// no-match note -- the allowlist and its no-roots posture
	// retired with E6, so there is no closed state to explain and
	// no admin jump to offer (the component reads no role at all).
	document.body.textContent = '';
	const empty = vi.fn(async (_input: RequestInfo | URL) => jsonResponse(200, { paths: [] }));
	vi.stubGlobal('fetch', empty);
	const again = mountField();
	focus();
	await advance(300);
	expect(document.body.querySelector('.path-completion__no-match')?.textContent).toContain('No matching folders');
	// The keys footer rides the row list only -- nothing to
	// navigate here.
	expect(keysFooter()).toBeNull();
	expect(document.body.querySelector('.path-completion__no-roots')).toBeNull();
	expect(document.body.querySelector('.path-completion__no-roots-action')).toBeNull();

	unmount(again.component);
	flushSync();
});

test("the validation slot swaps the hint and the failure at ONE height (§7.5's companion pin)", async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, { paths: [] }));
	vi.stubGlobal('fetch', fetchMock);
	const hint = 'Choose an existing folder — the album references it in place.';

	// The standing face: the hint fills the reserved slot.
	const withHint = mountField({ hint });
	const hintSlot = document.body.querySelector<HTMLElement>('.path-completion__slot');
	expect(hintSlot).not.toBeNull();
	expect(hintSlot?.textContent).toContain('Choose an existing folder');
	expect(hintSlot?.className).toContain('h-[37px]');
	expect(hintSlot?.className).toContain('path-completion__hint');
	expect(hintSlot?.className).not.toContain('path-completion__failure');
	// The input's described-by partner is the slot itself: the hint
	// is a description exactly as the failure is.
	expect(document.body.querySelector<HTMLInputElement>('#test-path')?.getAttribute('aria-describedby')).toBe(
		'test-path-note'
	);
	unmount(withHint.component);
	flushSync();

	// The failure face: the SAME slot, the SAME reserved height, the
	// failure's words and color replacing the hint's -- slot-replace,
	// never slot-add, so the dialog's height never moves for the
	// validation's comings and goings.
	const withFailure = mountField({ hint, failure: 'That folder does not exist.' });
	const failureSlot = document.body.querySelector<HTMLElement>('.path-completion__slot');
	expect(failureSlot).not.toBeNull();
	expect(failureSlot?.textContent).toContain('That folder does not exist.');
	expect(failureSlot?.textContent).not.toContain('Choose an existing folder');
	expect(failureSlot?.className).toContain('h-[37px]');
	expect(failureSlot?.className).toContain('path-completion__failure');
	expect(failureSlot?.className).not.toContain('path-completion__hint');
	expect(failureSlot?.getAttribute('role')).toBe('alert');
	expect(document.body.querySelector<HTMLInputElement>('#test-path')?.getAttribute('aria-invalid')).toBe('true');
	unmount(withFailure.component);
	flushSync();

	// The bare field (the settings' add-root row): the slot still
	// stands -- one reserved line -- empty and quiet.
	const bare = mountField();
	const bareSlot = document.body.querySelector<HTMLElement>('.path-completion__slot');
	expect(bareSlot).not.toBeNull();
	expect(bareSlot?.textContent).toBe('');
	expect(bareSlot?.className).toContain('h-[19px]');
	unmount(bare.component);
	flushSync();
});

test('a typed prefix with no match renders the honest empty note, never an error', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, { paths: [] }));
	vi.stubGlobal('fetch', fetchMock);
	const { component } = mountField();

	type('/none');
	await advance(300);
	expect(document.body.querySelector('.path-completion__no-match')?.textContent).toContain('No matching folders');

	unmount(component);
	flushSync();
});

test('the completion failure renders a line that points back at the completion', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(500, { code: 'INTERNAL' }));
	vi.stubGlobal('fetch', fetchMock);
	const { component } = mountField();

	type('/Vol');
	await advance(300);

	const failure = document.body.querySelector<HTMLElement>('.path-completion__failed');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('Could not load the folder suggestions');

	unmount(component);
	flushSync();
});

test('the write failure renders under the field with the invalid border', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, { paths: [] }));
	vi.stubGlobal('fetch', fetchMock);
	const { component, input } = mountField({ failure: 'That folder does not exist.' });

	// The caller's routed write failure: the line under the field,
	// the input invalid.
	expect(document.body.querySelector('.path-completion__failure')?.textContent).toBe('That folder does not exist.');
	expect(input()?.getAttribute('aria-invalid')).toBe('true');

	unmount(component);
	flushSync();
});

test('the server-bounded row list renders as it arrives (at most 20 from the contract)', async () => {
	const many = Array.from({ length: 20 }, (_, index) => `/Volumes/Photos/${index}`);
	const fetchMock = vi.fn(async (_input: RequestInfo | URL) => jsonResponse(200, { paths: many }));
	vi.stubGlobal('fetch', fetchMock);
	const { component } = mountField();

	type('/Volumes/Photos/');
	await advance(300);

	expect(rows()).toHaveLength(20);
	expect(rows()[19]).toBe('/Volumes/Photos/19');

	unmount(component);
	flushSync();
});
