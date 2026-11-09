import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import AudienceField from './AudienceField.svelte';

// The audience field (web IA §7.6, the specimen's add-style pin):
// the granted accounts as chips (avatar + email + the remove
// mark), the dashed "Add" affordance, and the INLINE candidate
// picker -- the users listing's candidates (every account minus
// the draft's members minus the caller), the keyboard the keys
// footer names (↑/↓ highlight, Enter adds, Esc closes the picker
// alone), the per-open fetch with its honest failure line, and the
// disabled account's note. The candidate rows are state selectors
// (mousedown adds), the chip's remove mark is the one button.

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

test('the empty audience renders the honest "only you" note', () => {
	const fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
	mountField();

	expect(document.body.querySelector('.audience-field__only-me')?.textContent).toBe('Only you');
	expect(fetchMock).not.toHaveBeenCalled();
});

test('opening the picker fetches the member list and offers the other accounts', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The candidates: every account minus the caller (the identity
	// seam's self filter) minus the draft's members (none yet).
	expect(fetchMock).toHaveBeenCalledTimes(1);
	expect(rowNames()).toEqual(['grace@example.com', 'ivy@example.com', 'hopper@example.com']);
	// The keys footer rides the rows (the specimen's language).
	expect(document.body.querySelector('.audience-field__keys')?.textContent).toContain(
		'Enter Add'
	);
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
	expect(
		document.body.querySelectorAll('[role="option"]')[0]?.getAttribute('aria-selected')
	).toBe('true');
	listbox?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
	flushSync();

	// The chip landed and the picker STAYS open (the specimen's
	// state two: a chip exists while the candidates offer the next).
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
	// (two opens, two fetches).
	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	failing = false;
	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	expect(fetchMock).toHaveBeenCalledTimes(2);
	expect(rowNames()).toEqual(['grace@example.com', 'ivy@example.com', 'hopper@example.com']);
});

test('no candidates left renders the honest note', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1' });

	addButton().dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	// Add every candidate: the picker re-derives to nothing.
	const addAll = document.body.querySelectorAll<HTMLElement>('[role="option"]');
	for (const row of [...addAll]) {
		row.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true }));
		flushSync();
	}
	expect(rowNames()).toEqual([]);
	expect(document.body.querySelector('.audience-field__no-candidates')?.textContent).toContain(
		'No other account is available.'
	);
});

test('a busy write disables the affordances', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, usersBody));
	vi.stubGlobal('fetch', fetchMock);
	mountField({ selfId: 'u1', busy: true });

	expect(addButton().disabled).toBe(true);
});
