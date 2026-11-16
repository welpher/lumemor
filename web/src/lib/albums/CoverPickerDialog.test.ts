import { flushSync, mount, tick, unmount } from 'svelte';
import { pickerState } from './coverPickerTestState.svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { clearToasts, toasts } from '$lib/components/toast.svelte';
import type { TimelineItem } from '$lib/api/timeline';
import CoverPickerDialog from './CoverPickerDialog.svelte';

// The cover picker: a cover is a SET of frames (four on the album
// card, six on the memory band) and the click order is the render
// order. The dialog seeds the stored pin, numbers the picks, refuses
// the over-full click with a toast, keeps Save off while the set is
// half-picked, saves the empty set as the automatic state, and marks
// a pinned frame the collection no longer holds.

function item(id: string, filename = `IMG_${id}.jpg`): TimelineItem {
	return {
		id,
		takenAt: '2026-09-10T12:00:00Z',
		originalFilename: filename,
		ownerId: 'u1',
		kind: 'image',
		fileSize: 1000,
		mimeType: 'image/jpeg',
		checksum: `checksum-${id}`,
		createdAt: '2026-09-10T12:00:00Z',
		takenAtSource: 'exif',
		isTrashed: false,
		isArchived: false,
		isFavorite: false,
		isReady: true,
		width: 4000,
		height: 3000,
		durationMs: null,
		blurhash: 'UHL6aAof00WBWB00ayWB00',
		livePhotoVideoId: null
	};
}

const pool = ['a1', 'a2', 'a3', 'a4', 'a5', 'a6', 'a7', 'a8'].map((id) => item(id));

function mountPicker(props: Record<string, unknown> = {}): {
	dialog: ReturnType<typeof mount>;
	onsubmit: ReturnType<typeof vi.fn>;
} {
	const onsubmit = vi.fn();
	const dialog = mount(CoverPickerDialog, {
		target: document.body,
		props: { open: true, items: pool, itemCount: pool.length, maxItems: 4, onsubmit, ...props }
	});
	flushSync();
	return { dialog, onsubmit };
}

function options(): HTMLButtonElement[] {
	return Array.from(document.body.querySelectorAll<HTMLButtonElement>('.cover-option'));
}

function optionOf(id: string): HTMLButtonElement {
	const found = options().find((button) => button.querySelector('img')?.getAttribute('src')?.includes(`/${id}/`));
	if (found === undefined) {
		throw new Error(`missing cover option ${id}`);
	}
	return found;
}

function badgeOf(id: string): string {
	return optionOf(id).querySelector('.cover-option__badge')?.textContent?.trim() ?? '';
}

function saveButton(): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === 'Save cover');
	if (found === undefined) {
		throw new Error('missing save button');
	}
	return found;
}

function statusLine(): string {
	return document.body.querySelector('.cover-picker__count')?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
}

beforeEach(async () => {
	document.body.textContent = '';
	clearToasts();
	await initI18n();
});

afterEach(() => {
	document.body.textContent = '';
});

test('the pick numbers its frames in click order and renumbers on removal', () => {
	const { dialog } = mountPicker();

	// The empty pick reads as the automatic state, and Save is live
	// (the empty set is the restore path).
	expect(statusLine()).toBe('Automatic (changes weekly)');
	expect(saveButton().disabled).toBe(false);

	optionOf('a3').click();
	optionOf('a1').click();
	optionOf('a5').click();
	flushSync();
	expect(badgeOf('a3')).toBe('1');
	expect(badgeOf('a1')).toBe('2');
	expect(badgeOf('a5')).toBe('3');
	expect(statusLine()).toBe('Selected 3 / 4 · 1 more to pick');
	// Half-picked cannot save.
	expect(saveButton().disabled).toBe(true);

	// Removing the second pick shifts the rest up -- the order is the
	// remaining picks' click order.
	optionOf('a1').click();
	flushSync();
	expect(badgeOf('a3')).toBe('1');
	expect(badgeOf('a5')).toBe('2');
	expect(badgeOf('a1')).toBe('');

	optionOf('a7').click();
	flushSync();
	expect(statusLine()).toBe('Selected 3 / 4 · 1 more to pick');
	expect(saveButton().disabled).toBe(true);
	optionOf('a2').click();
	flushSync();
	expect(statusLine()).toBe('Selected 4 / 4');
	expect(saveButton().disabled).toBe(false);

	unmount(dialog);
});

test('the over-full click is refused with a toast, not a fifth pick', () => {
	const { dialog } = mountPicker();
	for (const id of ['a1', 'a2', 'a3', 'a4']) {
		optionOf(id).click();
	}
	flushSync();
	expect(options().filter((button) => button.getAttribute('aria-pressed') === 'true')).toHaveLength(4);

	optionOf('a5').click();
	flushSync();
	expect(optionOf('a5').getAttribute('aria-pressed')).toBe('false');
	expect(toasts().map((entry) => entry.text)).toContain('A cover holds 4 photos.');
	// The full set is saveable.
	expect(saveButton().disabled).toBe(false);

	unmount(dialog);
});

test('the submit carries the pick order, and the clear sends the empty set', () => {
	const { dialog, onsubmit } = mountPicker();
	for (const id of ['a4', 'a2', 'a6', 'a1']) {
		optionOf(id).click();
	}
	flushSync();
	saveButton().click();
	expect(onsubmit).toHaveBeenLastCalledWith(['a4', 'a2', 'a6', 'a1']);

	document.body.querySelector<HTMLButtonElement>('.cover-picker__clear')?.click();
	flushSync();
	expect(statusLine()).toBe('Automatic (changes weekly)');
	saveButton().click();
	expect(onsubmit).toHaveBeenLastCalledWith([]);

	unmount(dialog);
});

test('the stored pin seeds the pick in its own order', () => {
	const { dialog } = mountPicker({ currentCoverIds: ['a6', 'a2', 'a7', 'a3'] });
	expect(badgeOf('a6')).toBe('1');
	expect(badgeOf('a2')).toBe('2');
	expect(badgeOf('a7')).toBe('3');
	expect(badgeOf('a3')).toBe('4');
	expect(statusLine()).toBe('Selected 4 / 4');

	unmount(dialog);
});

test('a pinned frame the collection no longer holds is marked and starts picked', () => {
	// The pool is the whole collection (items.length === itemCount),
	// so a stored id outside it is gone.
	const { dialog, onsubmit } = mountPicker({ currentCoverIds: ['a1', 'a2', 'a3', 'gone'] });
	const gone = document.body.querySelector<HTMLElement>('.cover-option--gone');
	expect(gone).not.toBeNull();
	expect(gone?.textContent).toContain('No longer in this album');
	expect(gone?.getAttribute('aria-pressed')).toBe('true');
	expect(gone?.querySelector('.cover-option__badge')?.textContent?.trim()).toBe('4');

	// Deselecting it saves the surviving three only after a re-pick
	// (the set is half-full) -- the picker never hides the gap.
	gone?.click();
	flushSync();
	expect(statusLine()).toBe('Selected 3 / 4 · 1 more to pick');
	expect(saveButton().disabled).toBe(true);

	// Picking the fill makes the set whole again and saves clean.
	optionOf('a5').click();
	flushSync();
	saveButton().click();
	expect(onsubmit).toHaveBeenLastCalledWith(['a1', 'a2', 'a3', 'a5']);

	unmount(dialog);
});

test('a fully dead pin clears from an empty collection', () => {
	// Every pinned frame left the album: the grid has no members, the
	// stored cells render as marked tiles, and the way back to the
	// rotation stays reachable (the server would otherwise keep a pin
	// nothing can render).
	const { dialog, onsubmit } = mountPicker({
		items: [],
		itemCount: 0,
		currentCoverIds: ['gone1', 'gone2', 'gone3', 'gone4']
	});
	expect(document.body.querySelector('.cover-picker__empty')).toBeNull();
	const gone = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.cover-option--gone'));
	expect(gone).toHaveLength(4);
	// The full dead set is still "N of N": saving it answers the
	// server's refusal, so the pick must be cleared first.
	for (const tile of gone) {
		tile.click();
	}
	flushSync();
	expect(document.body.querySelectorAll('.cover-option--gone[aria-pressed="true"]')).toHaveLength(0);
	expect(statusLine()).toBe('Automatic (changes weekly)');
	expect(saveButton().disabled).toBe(false);
	saveButton().click();
	expect(onsubmit).toHaveBeenLastCalledWith([]);

	unmount(dialog);
});

test('an incomplete pool never calls a stored frame missing', () => {
	// The grid holds one page of a larger collection: a pin from a
	// later page must not read as gone.
	const { dialog } = mountPicker({
		items: pool.slice(0, 4),
		itemCount: 40,
		currentCoverIds: ['a1', 'a2', 'a3', 'zz']
	});
	expect(document.body.querySelector('.cover-option--gone')).toBeNull();

	unmount(dialog);
});

test('the empty collection states the empty cover and cannot save a fill', () => {
	const { dialog } = mountPicker({ items: [], itemCount: 0 });
	expect(document.body.querySelector('.cover-picker__empty')?.textContent).toContain('This album has no photos yet.');
	expect(saveButton().disabled).toBe(true);
	expect(options()).toHaveLength(0);

	unmount(dialog);
});

test('a refused write states how many frames left the album', () => {
	const { dialog } = mountPicker({
		maxItems: 6,
		currentCoverIds: ['a1', 'a2', 'a3', 'a4', 'a5', 'a6'],
		errorKey: 'album_cover_not_member',
		invalidCount: 2
	});
	const line = document.body.querySelector('.cover-picker__failure');
	expect(line?.getAttribute('role')).toBe('alert');
	expect(line?.textContent).toContain('2 of the picked photos are no longer in the album');

	unmount(dialog);
});

test('a live detail refresh never wipes an in-progress pick', async () => {
	const onsubmit = vi.fn();
	pickerState.open = true;
	pickerState.currentCoverIds = [];
	const dialog = mount(CoverPickerDialog, {
		target: document.body,
		props: {
			get open() {
				return pickerState.open;
			},
			get currentCoverIds() {
				return pickerState.currentCoverIds;
			},
			items: pool,
			itemCount: pool.length,
			maxItems: 4,
			onsubmit
		}
	});
	flushSync();
	// A half-made pick: two frames numbered in click order.
	optionOf('a3').click();
	optionOf('a5').click();
	flushSync();
	expect(badgeOf('a3')).toBe('1');
	expect(badgeOf('a5')).toBe('2');

	// The parent's detail converges mid-pick (a live event replaced
	// the array): the seed belongs to the OPEN EDGE alone, so the
	// badges survive the fresh prop value.
	pickerState.currentCoverIds = ['a1', 'a2'];
	await tick();
	expect(badgeOf('a3')).toBe('1');
	expect(badgeOf('a5')).toBe('2');

	// Reopening is the one moment the stored pin reseeds.
	pickerState.open = false;
	await tick();
	pickerState.open = true;
	await tick();
	expect(badgeOf('a1')).toBe('1');
	expect(badgeOf('a2')).toBe('2');

	unmount(dialog);
	flushSync();
});
