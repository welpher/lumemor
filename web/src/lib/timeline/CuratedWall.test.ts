import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import type { TimelineItem } from '$lib/api/timeline';
import CuratedWall from './CuratedWall.svelte';

// THE CURATED WALL'S PAGE BAR: the favorites/hidden wrapper owns the
// one-row action bar -- the title with its count on the left, the
// selection switch at the far right -- and that switch is a REAL
// state machine, not a skin: pressing it arms the wall's select mode
// (its root carries `is-selecting` and every tile's check is
// revealed), pressing it off drops the selection with it (the batch
// bar goes, the checks rest), and a tile's own check enters the same
// mode from the pointer side.

const itemDefaults = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image' as const,
	originalFilename: 'IMG_1234.jpg',
	fileSize: 4821933,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d65a2feaa0c55ad015',
	createdAt: '2026-10-29T09:12:00Z',
	takenAtSource: 'exif' as const,
	isTrashed: false,
	isFavorite: true,
	isReady: true,
	width: 4032,
	height: 3024,
	durationMs: null,
	blurhash: 'UHL6aAof00WB00ayWB00',
	livePhotoVideoId: null
};

function item(id: string, takenAt: string): TimelineItem {
	return { id, takenAt, ...itemDefaults } as TimelineItem;
}

const items = [item('a1', '2026-11-02T10:00:00Z'), item('a2', '2026-10-13T12:30:00Z')];

const props = {
	path: '/api/v1/assets/favorites',
	view: 'favorites' as const,
	titleKey: 'nav_favorites',
	countKey: 'favorites_bar_count',
	loadingKey: 'favorites_loading',
	emptyKey: 'favorites_empty',
	actions: ['favorite' as const]
};

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
	// The wall's own walk reads the global fetch at construction; the
	// stub answers one settled page (the last page: no cursor).
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => ({
			ok: true,
			status: 200,
			json: async () => ({ items })
		}))
	);
});

afterEach(() => {
	vi.unstubAllGlobals();
});

/** Drain the walk's promise chain and force the DOM flush. */
async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	flushSync();
}

function toggle(): HTMLButtonElement {
	const found = document.body.querySelector<HTMLButtonElement>('.wall-page__actions button');
	if (found === null) {
		throw new Error('missing the page bar select switch');
	}
	return found;
}

function checks(): HTMLButtonElement[] {
	return Array.from(document.body.querySelectorAll<HTMLButtonElement>('.tile__check'));
}

function click(element: HTMLElement | undefined): void {
	element?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

test('the page bar carries the title, its count and the selection switch', async () => {
	const wall = mount(CuratedWall, { target: document.body, props });
	await settle();

	const title = document.body.querySelector('.wall-page__title');
	expect(title?.textContent?.replace(/\s+/g, ' ').trim()).toBe('Favorites · 2 photos');
	const bar = document.body.querySelector('.wall-page__toolbar');
	expect(bar?.className).toContain('pb-4');
	expect(bar?.className).toContain('border-b');
	expect(toggle().getAttribute('aria-pressed')).toBe('false');
	expect(document.body.querySelector('.wall__groups')?.classList.contains('is-selecting')).toBe(false);
	// The resting wall reveals no check circle (the hover affordance is
	// the pointer entry, the switch is the touch entry).
	expect(checks()).toHaveLength(2);
	expect(checks()[0]?.className).toContain('opacity-0');

	unmount(wall);
	flushSync();
});

test('the switch arms the mode, and pressing it off clears the selection with it', async () => {
	const wall = mount(CuratedWall, { target: document.body, props });
	await settle();

	// ON: the pressed state, the wall root's marker and every check
	// circle revealed.
	click(toggle());
	expect(toggle().getAttribute('aria-pressed')).toBe('true');
	expect(document.body.querySelector('.wall__groups')?.classList.contains('is-selecting')).toBe(true);
	expect(checks()[0]?.className).toContain('tile__check--shown');
	expect(checks()[0]?.className).not.toContain('opacity-0');
	expect(document.body.querySelector('.batchbar')).toBeNull();

	// One tile check enters the same state machine: the bar rises with
	// the honest count.
	click(checks()[0]);
	expect(checks()[0]?.getAttribute('aria-checked')).toBe('true');
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');
	click(checks()[1]);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('2 selected');

	// Unchecking is a partial step: the bar's count follows, the armed
	// mode stays on (the switch is still pressed).
	click(checks()[1]);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');
	expect(toggle().getAttribute('aria-pressed')).toBe('true');

	// OFF: the count goes to zero and the bar leaves the surface --
	// the selection never outlives the mode that armed it.
	click(toggle());
	expect(toggle().getAttribute('aria-pressed')).toBe('false');
	expect(document.body.querySelector('.wall__groups')?.classList.contains('is-selecting')).toBe(false);
	expect(document.body.querySelector('.batchbar')).toBeNull();
	expect(checks().map((check) => check.getAttribute('aria-checked'))).toEqual(['false', 'false']);
	expect(checks()[0]?.className).toContain('opacity-0');

	unmount(wall);
	flushSync();
});

test('a tile check arms the reveal without pressing the page switch', async () => {
	// The pointer/keyboard entry and the page bar's switch are the same
	// state machine's two doors: one checked tile reveals every check
	// and raises the bar while the switch itself is still off.
	const wall = mount(CuratedWall, { target: document.body, props });
	await settle();

	click(checks()[1]);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');
	expect(checks()[0]?.className).toContain('tile__check--shown');
	expect(toggle().getAttribute('aria-pressed')).toBe('false');

	unmount(wall);
	flushSync();
});
