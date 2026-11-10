import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import ShortcutsDialog from './ShortcutsDialog.svelte';

// The `?` shortcut sheet (web IA §6's pinned cheat sheet): the
// shared Modal over the shell listing every binding the app
// actually carries, in the two-column shape (kbd glyphs beside
// plain-language actions) grouped by the context each binding
// lives in. The tests pin the toggle (`?` opens, `?` again and Esc
// close), the guards (typing in a field, an open dialog owning the
// keyboard), and the collected content itself -- the sheet is the
// code's own binding inventory, so a row that lies about a
// non-existent binding is the failure this file exists to catch.

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

/** Press one key on the window (the shell-level handler's input). */
function press(key: string): KeyboardEvent {
	const event = new KeyboardEvent('keydown', { key, cancelable: true, bubbles: true });
	window.dispatchEvent(event);
	flushSync();
	return event;
}

/** Press one key targeted at a field (the editable guard's input). */
function pressOn(key: string, target: HTMLElement): KeyboardEvent {
	const event = new KeyboardEvent('keydown', { key, cancelable: true, bubbles: true });
	Object.defineProperty(event, 'target', { value: target });
	target.dispatchEvent(event);
	flushSync();
	return event;
}

function sheet(): HTMLElement | null {
	// The shared Modal's own panel (the test's fake dialogs below
	// also carry role=dialog; the sheet is the modal).
	return document.body.querySelector<HTMLElement>('.modal__panel[role="dialog"]');
}

function kbds(): string[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('kbd')).map((key) => key.textContent ?? '');
}

test('`?` toggles the sheet; the second `?` and Esc close it', () => {
	const dialog = mount(ShortcutsDialog, { target: document.body, props: { open: false } });
	flushSync();
	expect(sheet()).toBeNull();

	press('?');
	expect(sheet()).not.toBeNull();
	expect(sheet()?.getAttribute('aria-label')).toBe('Keyboard shortcuts');

	// The toggle's other arm: the same stroke closes.
	press('?');
	expect(sheet()).toBeNull();

	// The Modal's own Esc closes an open sheet.
	press('?');
	expect(sheet()).not.toBeNull();
	press('Escape');
	expect(sheet()).toBeNull();

	unmount(dialog);
	flushSync();
});

test('the sheet collects the live bindings in the two-column shape, grouped by context', () => {
	const dialog = mount(ShortcutsDialog, { target: document.body, props: { open: false } });
	press('?');

	// The group headings carry the contexts (Esc is a different verb
	// per context -- the grouping is what keeps the list honest).
	const headings = Array.from(document.body.querySelectorAll<HTMLElement>('.shortcuts__group-title')).map(
		(heading) => heading.textContent
	);
	expect(headings).toEqual([
		'Anywhere',
		'In the photo viewer',
		'While selecting photos',
		'In the path and audience pickers'
	]);

	// The key glyphs, kbd-styled, one <kbd> per key: the collected
	// inventory -- every binding the code actually carries (the
	// viewer's arrows/i/zoom/Esc, the shell's `?`/Esc and the `/`
	// focus-search, the wall's selection Esc, the pickers' keys).
	expect(kbds()).toEqual(['?', 'Esc', '/', '←', '→', 'i', '+', '−', 'Esc', 'Esc', '↑', '↓', 'Enter', 'Esc']);

	// The actions in plain language, one per row: the two-column
	// list (the key column beside the action column).
	const actions = Array.from(document.body.querySelectorAll<HTMLElement>('.shortcuts__action')).map(
		(action) => action.textContent
	);
	expect(actions).toContain('Focus the search box');
	expect(actions).toContain('Move to the previous or next photo');
	expect(actions).toContain('Toggle the info panel');
	expect(actions).toContain('Zoom the photo in or out');
	expect(actions).toContain('Close the info panel, then leave the viewer');
	expect(actions).toContain('Clear the selection');
	expect(actions).toContain('Move through the suggestions');
	expect(actions).toContain('Pick the highlighted suggestion');
	expect(actions).toContain('Close the suggestion list');
	expect(actions).toContain('Show or hide this list');
	expect(actions).toContain('Close the open menu, dialog or list');

	// The row shape: the kbd column beside the action column (the
	// two-column grid).
	const row = document.body.querySelector<HTMLElement>('.shortcuts__row');
	expect(row?.className).toContain('grid-cols-[auto_1fr]');
	expect(row?.querySelector('kbd')).not.toBeNull();

	unmount(dialog);
	flushSync();
});

test('typing in a field keeps the stroke: the sheet never steals a `?` aimed at an input', () => {
	const dialog = mount(ShortcutsDialog, { target: document.body, props: { open: false } });
	const field = document.createElement('input');
	document.body.appendChild(field);
	field.focus();

	pressOn('?', field);
	expect(sheet()).toBeNull();

	field.remove();
	unmount(dialog);
	flushSync();
});

test('an open dialog owns the keyboard: `?` never stacks the sheet over another dialog', () => {
	const dialog = mount(ShortcutsDialog, { target: document.body, props: { open: false } });

	// Another dialog is open (the layered dismissal the viewer
	// pins: the open dialog owns every other key).
	const other = document.createElement('div');
	other.setAttribute('role', 'dialog');
	document.body.appendChild(other);

	press('?');
	expect(sheet()).toBeNull();

	// With the other dialog gone, the stroke is the shell's again.
	other.remove();
	press('?');
	expect(sheet()).not.toBeNull();

	press('Escape');
	unmount(dialog);
	flushSync();
});

test("the open sheet traps the focus on its panel (the Modal's own discipline)", () => {
	const dialog = mount(ShortcutsDialog, { target: document.body, props: { open: false } });
	const field = document.createElement('input');
	document.body.appendChild(field);
	field.focus();

	press('?');
	// The Modal's trap: the focus left the field for the panel.
	expect(document.activeElement?.closest('[role="dialog"]')).not.toBeNull();
	expect(document.activeElement).not.toBe(field);

	press('Escape');
	// The close restores the invoking focus.
	expect(document.activeElement).toBe(field);

	field.remove();
	unmount(dialog);
	flushSync();
});
