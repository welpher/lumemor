import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import AddTagDialog from './AddTagDialog.svelte';
import type { TagSummary } from '$lib/api/tags';

// The add-tag dialog carries no <form> (its submit is a footer button),
// so the browser's constraint layer has nothing to validate: the local
// 1-64 rune check IS this dialog's pre-flight. The pin: an empty draft
// and an over-bound draft both stay on the page, with the same inline
// message the bound-violation slot has always shown, and the create
// never reaches the caller.

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

function mountDialog(
	overrides: {
		tags?: TagSummary[];
		oncreate?: (name: string) => void;
	} = {}
) {
	const dialog = mount(AddTagDialog, {
		target: document.body,
		props: {
			open: true,
			tags: overrides.tags ?? [],
			onpick: vi.fn(),
			oncreate: overrides.oncreate ?? vi.fn(),
			ondone: vi.fn()
		}
	});
	flushSync();
	return dialog;
}

/** Pick the dialog's "new tag" mode (the radio pair's second arm). */
function pickNewMode(): void {
	const radio = Array.from(document.body.querySelectorAll<HTMLInputElement>('input[type="radio"]')).find(
		(input) => input.value === 'new'
	);
	if (radio === null || radio === undefined) {
		throw new Error('missing the new-tag radio');
	}
	radio.click();
	flushSync();
}

/** Type into the new-tag field the way a reader does (input events). */
function typeName(value: string): void {
	const field = document.body.querySelector<HTMLInputElement>('input[name="add-tag-name"]');
	if (field === null) {
		throw new Error('missing the new-tag field');
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
}

function clickCreate(): void {
	const button = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button')).find(
		(candidate) => candidate.textContent?.trim() === 'Create and add'
	);
	if (button === undefined) {
		throw new Error('missing the create button');
	}
	button.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
}

test('the new-tag field carries the contract ceiling natively', () => {
	const dialog = mountDialog();
	pickNewMode();

	const field = document.body.querySelector<HTMLInputElement>('input[name="add-tag-name"]');
	expect(field?.required).toBe(true);
	expect(field?.getAttribute('maxlength')).toBe('64');

	unmount(dialog);
	flushSync();
});

test('an empty draft never leaves the page', () => {
	const oncreate = vi.fn();
	const dialog = mountDialog({ oncreate });
	pickNewMode();

	clickCreate();

	expect(oncreate).not.toHaveBeenCalled();
	expect(document.body.querySelector('.add-tag__bound')?.textContent).toContain('Tag names span 1 to 64 characters.');

	unmount(dialog);
	flushSync();
});

test('an over-bound draft never leaves the page either, and a valid one does', () => {
	const created: string[] = [];
	const dialog = mountDialog({ oncreate: (name) => created.push(name) });
	pickNewMode();

	typeName(`${'a'.repeat(64)}b`);
	clickCreate();
	expect(created).toEqual([]);
	expect(document.body.querySelector('.add-tag__bound')?.textContent).toContain('Tag names span 1 to 64 characters.');

	typeName('  weekend  ');
	clickCreate();
	expect(created).toEqual(['weekend']);

	unmount(dialog);
	flushSync();
});
