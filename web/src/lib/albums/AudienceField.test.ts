import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import AudienceField from './AudienceField.svelte';

// The audience field:
// the granted accounts as chips (avatar + email + the remove
// mark), the dashed "Add" affordance, and the candidate picker as
// a PORTALED floating layer (the floating-layer shape:
// fixed, anchored under the chips row, the dialog never changing
// height for it) -- the users listing's candidates (every account
// minus the draft's members minus the caller), the keyboard the
// keys footer names (↑/↓ highlight, Enter adds, Esc closes the
// picker alone), the per-open fetch with its honest failure line,
// and the disabled account's note. The candidate rows are state
// selectors (mousedown adds), the chip's remove mark is the one
// button. The known-pool rule: a KNOWN-empty candidate pool
// disables the add affordance outright (a single-account instance,
// or the multi-add draining the last candidate) and an empty
// picker state is never shown -- the probe fetches the pool at
// mount, and the drain collapses the open picker.

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const usersBody = {
	users: [
		{ id: 'u1', email: 'ada@example.com', role: 'admin', isDisabled: false },
		{ id: 'u2', email: 'grace@example.com', role: 'user', isDisabled: false },
		{ id: 'u3', email: 'ivy@example.com', role: 'user', isDisabled: false },
		{ id: 'u4', email: 'hopper@example.com', role: 'user', isDisabled: true }
	]
};

/** The mounted field under test; the chips' DOM mirrors the
 * bindable draft (the rows render FROM the members state, so the
 * chip names read the component's writes back). */
let mounted: ReturnType<typeof mount> | null = null;

function mountField(props: { selfId?: string; busy?: boolean } = {}) {
	mounted = mount(AudienceField, {
		target: document.body,
		props: {
			members: [],
			selfId: props.selfId ?? '',
			busy: props.busy ?? false
		}
	});
	flushSync();
}

function addButton(): HTMLButtonElement {
	const found = document.body.querySelector<HTMLButtonElement>('.audience-field__add');
	if (found === null) {
		throw new Error('missing add button');
	}
	return found;
}

function rowNames(): (string | null)[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__row-name')).map(
		(row) => row.textContent
	);
}

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	flushSync();
}

beforeEach(async () => {
	document.body.textContent = '';
	mounted = null;
	await initI18n();
});

afterEach(() => {
	if (mounted !== null) {
		unmount(mounted);
		mounted = null;
	}
	vi.unstubAllGlobals();
});

test('the empty audience renders the honest "only you" note; the mount probe fetches the pool', () => {
	const fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
	mountField();

	expect(document.body.querySelector('.audience-field__only-me')?.textContent).toBe('Only you');
	// The known-pool rule needs the pool KNOWN before the first open (the
	// honest disabled button): the field probes the member list
	// once at mount. The listing fails here (the bare mock answers
	// nothing) -- an unknown pool keeps the button enabled, because
	// the open IS the failed listing's retry path.
	expect(fetchMock).toHaveBeenCalledTimes(1);
	expect(addButton().disabled).toBe(false);
});

test('opening the picker fetches the member list and offers the other accounts', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The candidates: every account minus the caller (the identity
	// seam's self filter) minus the draft's members (none yet). The
	// mount probe plus the per-open fetch: two listings, the
	// per-open discipline of the old share dialog.
	expect(fetchMock).toHaveBeenCalledTimes(2);
	expect(rowNames()).toEqual(['grace@example.com', 'ivy@example.com', 'hopper@example.com']);
	// The keys footer rides the rows.
	expect(document.body.querySelector('.audience-field__keys')?.textContent).toContain('Enter Add');
});

test('the keyboard: Enter adds the highlighted candidate, Esc closes the picker alone', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	const listbox = document.body.querySelector<HTMLElement>('#audience-field-picker');
	expect(listbox).not.toBeNull();

	// The first row is highlighted on arrival; Enter adds it.
	expect(document.body.querySelectorAll('[role="option"]')[0]?.getAttribute('aria-selected')).toBe('true');
	listbox?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
	flushSync();

	// The chip landed and the picker STAYS open (a
	// chip exists while the candidates offer the next).
	expect(
		Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__chip-name')).map(
			(chip) => chip.textContent
		)
	).toEqual(['grace@example.com']);
	expect(document.body.querySelector('#audience-field-picker')).not.toBeNull();

	// ↓ moves the highlight; Enter adds the second candidate (the
	// list re-derived after the first add: ivy heads it, hopper
	// follows).
	listbox?.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
	flushSync();
	listbox?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
	flushSync();
	expect(
		Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__chip-name')).map(
			(chip) => chip.textContent
		)
	).toEqual(['grace@example.com', 'hopper@example.com']);

	// Esc closes the picker alone -- the chips stay.
	listbox?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
	flushSync();
	expect(document.body.querySelector('#audience-field-picker')).toBeNull();
	expect(
		Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__chip-name')).map(
			(chip) => chip.textContent
		)
	).toEqual(['grace@example.com', 'hopper@example.com']);
});

test('a row adds on mousedown and the added account leaves the candidates', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const rows = document.body.querySelectorAll<HTMLElement>('[role="option"]');
	rows[1]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();

	// The chip landed; the candidate list re-derives without her.
	expect(
		Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__chip-name')).map(
			(chip) => chip.textContent
		)
	).toEqual(['ivy@example.com']);
	expect(rowNames()).toEqual(['grace@example.com', 'hopper@example.com']);
});

test('the chip avatar renders the email local part initial; the remove mark drops the member', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	document.body
		.querySelectorAll<HTMLElement>('[role="option"]')[0]
		?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();

	const chip = document.body.querySelector<HTMLElement>('.audience-field__chip');
	expect(chip?.querySelector('.audience-field__chip-avatar')?.textContent).toBe('G');
	// The remove mark: icon-only, named by its aria-label.
	const remove = chip?.querySelector<HTMLButtonElement>('.audience-field__chip-remove');
	expect(remove?.getAttribute('aria-label')).toBe('Remove grace@example.com');
	remove?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();

	// Back to the honest empty audience.
	expect(document.body.querySelector('.audience-field__only-me')?.textContent).toBe('Only you');
});

test('the disabled account renders its honest note and stays pickable', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const rows = document.body.querySelectorAll<HTMLElement>('[role="option"]');
	expect(rows[2]?.textContent).toContain('hopper@example.com');
	expect(rows[2]?.textContent).toContain('cannot sign in');
	// Still pickable: the server's skip report is the authority.
	rows[2]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
	flushSync();
	expect(
		Array.from(document.body.querySelectorAll<HTMLElement>('.audience-field__chip-name')).map(
			(chip) => chip.textContent
		)
	).toEqual(['hopper@example.com']);
});

test('the member-list failure renders inside the picker; the toggle retries', async () => {
	let failing = true;
	const fetchMock = vi.fn(async () => {
		if (failing) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return jsonResponse(200, usersBody);
	});
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	const failure = document.body.querySelector<HTMLElement>('.audience-field__failed');
	expect(failure?.getAttribute('role')).toBe('alert');
	expect(failure?.textContent).toContain('Could not load the member list');

	// The toggle affordance is the retry path: close, reopen, refetch
	// (the mount probe's failed listing, the first open's failed
	// listing, then the successful one -- three listings).
	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	failing = false;
	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(fetchMock).toHaveBeenCalledTimes(3);
	expect(rowNames()).toEqual(['grace@example.com', 'ivy@example.com', 'hopper@example.com']);
});

test('the multi-add drain collapses the picker and disables the add affordance', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	// Add every candidate: each add leaves the pool smaller, the
	// LAST one drains it -- and an empty picker state is never
	// shown: the picker collapses itself the moment the last row
	// leaves, the add button disables, and the no-candidates note
	// takes the reserved slot under the chips row.
	const addAll = document.body.querySelectorAll<HTMLElement>('[role="option"]');
	for (const row of [...addAll]) {
		row.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
		flushSync();
	}
	expect(rowNames()).toEqual([]);
	expect(document.body.querySelector('#audience-field-picker')).toBeNull();
	expect(addButton().disabled).toBe(true);
	expect(document.body.querySelector('.audience-field__no-candidates')?.textContent).toContain(
		'No other account is available.'
	);

	// The drain's focus discipline: the add button just disabled (an
	// unfocusable target), so the focus lands on the newest chip's
	// remove mark -- the one enabled button left in the section.
	// (The focus move rides the tick() promise; settle lets it land.)
	await settle();
	const marks = document.body.querySelectorAll<HTMLButtonElement>('.audience-field__chip-remove');
	expect(document.activeElement).toBe(marks[marks.length - 1]);

	// The disabled button opens nothing: the empty set earns no row.
	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(document.body.querySelector('#audience-field-picker')).toBeNull();
});

test('a single-account instance permanently disables the add affordance (the probed pool is the caller alone)', async () => {
	const fetchMock = vi.fn(async () =>
		jsonResponse(200, {
			users: [{ id: 'u1', email: 'ada@example.com', role: 'admin', isDisabled: false }]
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });
	await settle();

	// The probe landed: the pool is the caller alone -- KNOWN empty.
	// The button is disabled (reduced opacity, not-allowed cursor,
	// the honest native disabled semantics) with the reserved
	// no-candidates slot carrying the words; no click, no open.
	expect(fetchMock).toHaveBeenCalledTimes(1);
	expect(addButton().disabled).toBe(true);
	expect(addButton().className).toContain('disabled:opacity-55');
	expect(addButton().className).toContain('disabled:cursor-not-allowed');
	expect(document.body.querySelector('.audience-field__no-candidates')?.textContent).toContain(
		'No other account is available.'
	);

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(document.body.querySelector('#audience-field-picker')).toBeNull();

	// The reserved slot never moves: the note's line is always in
	// the layout (the reserved-slot rule -- the dialog's height is
	// invariant to the note's comings and goings).
	const slot = document.body.querySelector<HTMLElement>('.audience-field__slot');
	expect(slot).not.toBeNull();
	expect(slot?.className).toContain('h-[19px]');
});

test('a busy write disables the affordances', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1', busy: true });

	expect(addButton().disabled).toBe(true);
});
