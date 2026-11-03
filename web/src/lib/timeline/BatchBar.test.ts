import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import type { BatchAction } from '$lib/api/timeline';
import BatchBar from './BatchBar.svelte';

// The batch bar contract: it renders only with a selection, shows
// the count and the six batch actions (favorite/unfavorite collapsed
// into a state-labeled toggle), disables everything while an action
// runs, shows the mapped failure message inline, and clears through
// the clear action.

function mountBar(
	props: {
		count?: number;
		allFavorite?: boolean;
		errorKey?: string;
		busy?: boolean;
		actions?: readonly BatchAction[];
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
			onaction,
			onclear
		}
	});
	flushSync();
	return { bar, onaction, onclear };
}

function buttonByLabel(label: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.batchbar button'));
	const found = buttons.find((button) => button.textContent?.includes(label));
	if (found === undefined) {
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
	expect(favoriteButtons.map((button) => button.textContent?.trim())).toEqual(['Unfavorite']);
	favoriteButtons[0]?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(favoriteAction).toHaveBeenCalledWith('unfavorite');
	unmount(favoriteBar);
	flushSync();

	const { bar: hiddenBar } = mountBar({ count: 2, actions: ['unhide'] });
	const hiddenButtons = Array.from(
		document.body.querySelectorAll<HTMLButtonElement>('.batchbar button:not(.batchbar__clear)')
	);
	expect(hiddenButtons.map((button) => button.textContent?.trim())).toEqual(['Unhide']);
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

test('the zh-Hans locale renders through the shared table', async () => {
	setLocale('zh-Hans');
	const { bar } = mountBar({ count: 2 });
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('已选择 2 张');
	expect(buttonByLabel('移入回收站')).toBeTruthy();
	setLocale('en');
	unmount(bar);
	flushSync();
});
