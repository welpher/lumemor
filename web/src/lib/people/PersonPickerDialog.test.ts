import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test } from 'vitest';
import { initI18n } from '$lib/i18n';
import type { PersonSummary } from '$lib/api/people';
import PersonPickerDialog from './PersonPickerDialog.svelte';

// The person target picker's contract: the LOCAL name filter (case
// insensitive substring, no request), the list's own FIXED 264px
// scroll box, the selected row's edge-to-edge accent fill (the
// aria-checked state the row's utilities key on), the in-box
// centered empty state, the create-from-face row below the scroll
// box, and the choice's lifecycle (the first visible row is the
// default; a filter that hides the selection falls back; the
// reseed on open).

function person(clusterId: string, name: string): PersonSummary {
	return {
		clusterId,
		name,
		faceCount: 9,
		representativeFaceId: `f-${clusterId}`,
		thumbnail: `/api/v1/faces/f-${clusterId}/thumbnail`,
		isHidden: false
	};
}

const people = [person('p1', 'Grace'), person('p2', 'Mom'), person('p3', 'Dad'), person('p4', 'Grandma')];

const picks: string[] = [];
const creates: number[] = [];
const closes: number[] = [];

function mountDialog(overrides: Record<string, unknown> = {}): ReturnType<typeof mount> {
	const dialog = mount(PersonPickerDialog, {
		target: document.body,
		props: {
			open: true,
			titleKey: 'people_select_person_title',
			bodyKey: 'people_select_person_body',
			submitKey: 'viewer_assign_submit',
			people,
			oncreate: () => creates.push(creates.length),
			onclose: () => closes.push(closes.length),
			onsubmit: (clusterId: string) => picks.push(clusterId),
			...overrides
		}
	});
	flushSync();
	return dialog;
}

const mounted: Array<ReturnType<typeof mount>> = [];

beforeEach(async () => {
	document.body.textContent = '';
	picks.length = 0;
	creates.length = 0;
	closes.length = 0;
	await initI18n();
});

afterEach(() => {
	while (mounted.length > 0) {
		unmount(mounted.pop()!);
		flushSync();
	}
});

function rows(): HTMLButtonElement[] {
	return Array.from(document.body.querySelectorAll<HTMLButtonElement>('.person-picker__row'));
}

function filterInput(): HTMLInputElement {
	const input = document.body.querySelector<HTMLInputElement>('.person-picker__filter');
	if (input === null) {
		throw new Error('missing filter input');
	}
	return input;
}

function list(): HTMLUListElement {
	const element = document.body.querySelector<HTMLUListElement>('.person-picker__list');
	if (element === null) {
		throw new Error('missing list');
	}
	return element;
}

function typeFilter(value: string): void {
	const input = filterInput();
	input.value = value;
	input.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
}

function clickRow(row: HTMLButtonElement): void {
	row.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

function clickByText(text: string, scope: ParentNode = document.body): void {
	const found = Array.from(scope.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.textContent?.trim() === text
	);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	found.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

/** Click one icon-only button by its accessible name. */
function clickByLabel(label: string): void {
	const found = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.getAttribute('aria-label') === label
	);
	if (found === undefined) {
		throw new Error(`missing button ${label}`);
	}
	found.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

function rowText(): string[] {
	return rows().map((row) => (row.textContent ?? '').replace(/\s+/g, ' ').trim());
}

test('the dialog renders the people as avatar rows with their face counts', () => {
	mounted.push(mountDialog());

	expect(rowText()).toEqual(['Grace 9', 'Mom 9', 'Dad 9', 'Grandma 9']);
	// Radio semantics: one radiogroup, every row a radio, exactly one
	// checked (the first visible row is the default pick).
	expect(list().getAttribute('role')).toBe('radiogroup');
	expect(rows().map((row) => row.getAttribute('aria-checked'))).toEqual(['true', 'false', 'false', 'false']);
	// Every row carries its person's face count as the count span's
	// accessible name (the icon + number count language).
	const counts = Array.from(document.body.querySelectorAll('[role="img"]'));
	expect(counts.map((entry) => entry.getAttribute('aria-label'))).toEqual([
		'9 faces',
		'9 faces',
		'9 faces',
		'9 faces'
	]);
});

test('the filter is local and case-insensitive: typing narrows the rows, an empty term restores them', () => {
	mounted.push(mountDialog());

	typeFilter('mom');
	expect(rowText()).toEqual(['Mom 9']);

	// The case-insensitive arm: the uppercase term narrows the same.
	typeFilter('GRAND');
	expect(rowText()).toEqual(['Grandma 9']);

	typeFilter('');
	expect(rows()).toHaveLength(4);
});

test('the list box is a FIXED 264px with its own scroll, and the empty state centers inside it', () => {
	mounted.push(mountDialog());

	// The fixed height and the box's own scroll hold with rows
	// present...
	expect(list().className).toContain('h-[264px]');
	expect(list().className).toContain('overflow-y-auto');

	// ...and with NO matching rows: the empty state is the box's own
	// h-full centered child, never a sibling row below it.
	typeFilter('zzz');
	const empty = list().querySelector<HTMLElement>('.person-picker__empty');
	expect(empty?.textContent?.trim()).toBe('No people match.');
	expect(empty?.className).toContain('h-full');
	expect(empty?.className).toContain('justify-center');
	expect(list().querySelectorAll('.person-picker__row')).toHaveLength(0);
});

test('the selected row is the accent-filled edge: clicking a row moves the aria-checked state', () => {
	mounted.push(mountDialog());

	clickRow(rows()[1]!);
	expect(rows().map((row) => row.getAttribute('aria-checked'))).toEqual(['false', 'true', 'false', 'false']);
	// The row utilities key on aria-checked (the edge-to-edge fill
	// class pair the picked row carries).
	expect(rows()[1]!.className).toContain('aria-checked:bg-accent-soft');

	// A filter that hides the selection falls back to the first
	// visible row -- the submit can never carry a hidden person.
	typeFilter('dad');
	expect(rows().map((row) => row.getAttribute('aria-checked'))).toEqual(['true']);
});

test('the submit carries the picked person; the create row fires its own path', () => {
	mounted.push(mountDialog());

	clickRow(rows()[2]!);
	clickByText('Assign');
	expect(picks).toEqual(['p3']);

	clickByText('Create a new person from this face');
	expect(creates).toHaveLength(1);
	// The create row is the create path, never a submit: no extra pick.
	expect(picks).toEqual(['p3']);
});

test('the create row sits below the scroll box and stays reachable when the list is long', () => {
	mounted.push(
		mountDialog({
			people: Array.from({ length: 30 }, (_, index) => person(`p${index}`, `Person ${index}`))
		})
	);

	// The create row is OUTSIDE the list element (a sibling of the
	// box, never its child), so the list's scroll never buries it.
	const createRow = document.body.querySelector<HTMLElement>('.person-picker__create');
	expect(createRow).not.toBeNull();
	expect(list().contains(createRow!)).toBe(false);
});

test('the choice reseeds on open: a fresh open starts from the first visible row, never a stale pick', () => {
	const first = mountDialog();
	mounted.push(first);
	// Accumulate dialog-local state: a typed filter and a moved pick.
	typeFilter('dad');
	clickRow(rows()[0]!);
	// Close through the modal's own close mark (every close path).
	clickByLabel('Close');
	expect(closes).toHaveLength(1);
	expect(rows()).toHaveLength(0);

	// The next open starts over: the filter is empty and the first
	// row carries the default pick (the seeding effect's open arm).
	unmount(first);
	flushSync();
	mounted.pop();
	mounted.push(mountDialog());
	expect(filterInput().value).toBe('');
	expect(rows().map((row) => row.getAttribute('aria-checked'))).toEqual(['true', 'false', 'false', 'false']);
});

test('the loading state stands in for the list; the fetch failure renders the mapped error', () => {
	mounted.push(mountDialog({ loading: true }));
	expect(document.body.querySelector('.person-picker__loading')?.getAttribute('role')).toBe('status');
	expect(rows()).toHaveLength(0);

	mounted.push(mountDialog({ errorKey: 'internal' }));
	expect(document.body.querySelector('.person-picker__failure')?.getAttribute('role')).toBe('alert');
});
