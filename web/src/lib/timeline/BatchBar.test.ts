import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import type { BatchAction } from '$lib/api/timeline';
import BatchBar from './BatchBar.svelte';

// The batch bar contract: it renders only with a selection, shows
// the count and the six batch actions (favorite/unfavorite collapsed
// into a state-labeled toggle) as ICON-ONLY buttons named by their
// aria-labels (the icon-first rule -- the bar is a toolbar), disables
// everything while an action runs, shows the mapped failure message
// inline, and clears through the clear action. The two dialog-style
// affordances (the add-tag and add-to-memory openers) and the
// memory's membership removal ride OPTIONAL callbacks, not the
// action set: absent callbacks render no button at all.

function mountBar(
	props: {
		count?: number;
		allFavorite?: boolean;
		errorKey?: string;
		busy?: boolean;
		actions?: readonly BatchAction[];
		onaddtag?: () => void;
		onaddmemory?: () => void;
		onremovememory?: () => void;
	} = {}
) {
	const onaction = vi.fn();
	const onclear = vi.fn();
	const bar = mount(BatchBar, {
		target: document.body,
		props: {
			count: props.count ?? 3,
			allFavorite: props.allFavorite ?? false,
			errorKey: props.errorKey ?? '',
			busy: props.busy ?? false,
			actions: props.actions,
			onaddtag: props.onaddtag,
			onaddmemory: props.onaddmemory,
			onremovememory: props.onremovememory,
			onaction,
			onclear
		}
	});
	flushSync();
	return { bar, onaction, onclear };
}

function buttonByLabel(label: string): HTMLButtonElement {
	// The bar's actions are icon-only (the icon-first rule): the
	// accessible name rides the aria-label.
	const found = document.body.querySelector<HTMLButtonElement>(`.batchbar button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing batch button ${label}`);
	}
	return found;
}

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

test('no selection means no bar', () => {
	const { bar } = mountBar({ count: 0 });
	expect(document.body.querySelector('.batchbar')).toBeNull();
	unmount(bar);
	flushSync();
});

test('an action subset renders only the named actions (the curated walls)', () => {
	// The favorites wall passes the toggle alone; the hidden wall
	// the unhide action.
	const { bar: favoriteBar, onaction: favoriteAction } = mountBar({
		count: 2,
		allFavorite: true,
		actions: ['favorite']
	});
	const favoriteButtons = Array.from(
		document.body.querySelectorAll<HTMLButtonElement>('.batchbar button:not(.batchbar__clear)')
	);
	expect(favoriteButtons.map((button) => button.getAttribute('aria-label'))).toEqual(['Unfavorite']);
	favoriteButtons[0]?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(favoriteAction).toHaveBeenCalledWith('unfavorite');
	unmount(favoriteBar);
	flushSync();

	const { bar: hiddenBar } = mountBar({ count: 2, actions: ['unhide'] });
	const hiddenButtons = Array.from(
		document.body.querySelectorAll<HTMLButtonElement>('.batchbar button:not(.batchbar__clear)')
	);
	expect(hiddenButtons.map((button) => button.getAttribute('aria-label'))).toEqual(['Unhide']);
	unmount(hiddenBar);
	flushSync();
});

test('the bar shows the selected count and the six batch actions', () => {
	const { bar, onaction } = mountBar({ count: 5 });

	const toolbar = document.body.querySelector<HTMLElement>('.batchbar');
	expect(toolbar?.getAttribute('role')).toBe('toolbar');
	expect(toolbar?.textContent).toContain('5 selected');

	for (const label of ['Favorite', 'Hide', 'Unhide', 'Trash', 'Restore']) {
		const button = buttonByLabel(label);
		expect(button.disabled, label).toBe(false);
		// The icon-first rule: every action carries its lucide glyph
		// (20px, decorative -- the aria-label stays the name).
		expect(button.querySelector('svg'), label).not.toBeNull();
		button.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	}
	expect(onaction).toHaveBeenCalledTimes(5);

	unmount(bar);
	flushSync();
});

test('the favorite toggle flips its label with the selection state', () => {
	const plain = mountBar({ count: 2, allFavorite: false });
	expect(buttonByLabel('Favorite')).toBeTruthy();
	expect(() => buttonByLabel('Unfavorite')).toThrow();
	plain.onaction.mockClear();
	buttonByLabel('Favorite').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(plain.onaction).toHaveBeenCalledWith('favorite');
	unmount(plain.bar);
	flushSync();

	const loved = mountBar({ count: 2, allFavorite: true });
	expect(buttonByLabel('Unfavorite')).toBeTruthy();
	expect(() => buttonByLabel('Favorite')).toThrow();
	buttonByLabel('Unfavorite').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(loved.onaction).toHaveBeenCalledWith('unfavorite');
	unmount(loved.bar);
	flushSync();
});

test('a busy run disables every action', () => {
	const { bar } = mountBar({ count: 2, busy: true });
	for (const label of ['Favorite', 'Hide', 'Unhide', 'Trash', 'Restore']) {
		expect(buttonByLabel(label).disabled, label).toBe(true);
	}
	unmount(bar);
	flushSync();
});

test('a failed action maps its error code inline, unknown codes fall back', () => {
	const mapped = mountBar({ count: 2, errorKey: 'internal' });
	const message = document.body.querySelector<HTMLParagraphElement>('.batchbar__error');
	expect(message?.getAttribute('role')).toBe('alert');
	expect(message?.textContent).toBe('Something went wrong. Please try again.');
	unmount(mapped.bar);
	flushSync();

	const unknown = mountBar({ count: 2, errorKey: 'not_a_real_code' });
	expect(document.body.querySelector('.batchbar__error')?.textContent).toBe(
		'Something went wrong. Please try again.'
	);
	unmount(unknown.bar);
	flushSync();
});

test('the clear action asks the page to drop the selection', () => {
	const { bar, onclear } = mountBar({ count: 2 });
	const clear = document.body.querySelector<HTMLButtonElement>('.batchbar__clear');
	expect(clear?.getAttribute('aria-label')).toBe('Clear selection');
	clear?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onclear).toHaveBeenCalledTimes(1);
	unmount(bar);
	flushSync();
});

test('an add-tag affordance renders only when the page offers it', () => {
	// Without the callback (the curated walls, the search wall) the
	// bar carries no tag action.
	const plain = mountBar({ count: 2, actions: ['favorite'] });
	expect(() => buttonByLabel('Add tag')).toThrow();
	unmount(plain.bar);
	flushSync();

	// With it, the button sits beside the album affordance and fires
	// the callback; a busy run disables it like every other.
	const onaddtag = vi.fn();
	const offering = mountBar({ count: 2, actions: ['favorite'], onaddtag });
	const add = buttonByLabel('Add tag');
	expect(add.disabled).toBe(false);
	add.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onaddtag).toHaveBeenCalledTimes(1);
	unmount(offering.bar);
	flushSync();

	const busy = mountBar({ count: 2, busy: true, onaddtag });
	expect(buttonByLabel('Add tag').disabled).toBe(true);
	unmount(busy.bar);
	flushSync();
});

test('an add-to-memory affordance renders only when the page offers it', () => {
	// Without the callback the bar carries no memory action: the
	// memory flow is not a batch action, it rides the page's opener
	// exactly like the tag flow.
	const plain = mountBar({ count: 2, actions: ['favorite'] });
	expect(() => buttonByLabel('Add to memory')).toThrow();
	unmount(plain.bar);
	flushSync();

	// With it, the sparkles button rides after the tag affordance and
	// fires the callback; a busy run disables it like every other.
	const onaddtag = vi.fn();
	const onaddmemory = vi.fn();
	const offering = mountBar({ count: 2, actions: ['favorite'], onaddtag, onaddmemory });
	const buttons = Array.from(
		document.body.querySelectorAll<HTMLButtonElement>('.batchbar button:not(.batchbar__clear)')
	);
	expect(buttons.map((button) => button.getAttribute('aria-label'))).toEqual([
		'Add tag',
		'Add to memory',
		'Favorite'
	]);
	const add = buttonByLabel('Add to memory');
	expect(add.disabled).toBe(false);
	add.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onaddmemory).toHaveBeenCalledTimes(1);
	unmount(offering.bar);
	flushSync();

	const busy = mountBar({ count: 2, busy: true, onaddmemory });
	expect(buttonByLabel('Add to memory').disabled).toBe(true);
	unmount(busy.bar);
	flushSync();
});

test('a remove-from-memory affordance renders only when the page offers it', () => {
	// Without the callback the bar carries no removal action: the
	// membership write is the memory's own, never a batch API action.
	const plain = mountBar({ count: 2, actions: [] });
	expect(() => buttonByLabel('Remove from memory')).toThrow();
	unmount(plain.bar);
	flushSync();

	// With it, the minus button rides the danger styling before the
	// clear mark (the memory wall's whole action set: the removal
	// alone) and fires the callback.
	const onremovememory = vi.fn();
	const offering = mountBar({ count: 2, actions: [], onremovememory });
	const remove = buttonByLabel('Remove from memory');
	expect(remove.disabled).toBe(false);
	expect(remove.classList.contains('icon-btn--danger')).toBe(true);
	remove.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onremovememory).toHaveBeenCalledTimes(1);
	unmount(offering.bar);
	flushSync();

	const busy = mountBar({ count: 2, busy: true, onremovememory });
	expect(buttonByLabel('Remove from memory').disabled).toBe(true);
	unmount(busy.bar);
	flushSync();
});

test('the zh-Hans locale renders through the shared table', async () => {
	setLocale('zh-Hans');
	const { bar } = mountBar({ count: 2 });
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('已选择 2 张');
	expect(buttonByLabel('移入回收站')).toBeTruthy();
	setLocale('en');
	unmount(bar);
	flushSync();
});
