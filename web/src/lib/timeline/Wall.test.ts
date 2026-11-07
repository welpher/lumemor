import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Wall from './Wall.svelte';
import type { TimelineItem } from '$lib/api/timeline';

// The unified wall (the 2026-11-07 owner pin): ONE pure rendering
// component every photo surface composes -- the timeline, the album
// detail, the favorites/hidden wrappers, the search browse mode. It
// takes bare ITEMS (the month/day fold is its own), the row engine,
// the ResizeObserver re-pack and PhotoTile live inside it, and the
// page-level concerns (selection state, batch bar, scrubber, the
// data walk) never enter. The selection contract: `onselect`'s
// PRESENCE is the capability -- a wall with no callback renders NO
// check affordance at all (the album detail's pure-navigation wall,
// R5), while the walls that wire it keep the hover-reveal entry.

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
	isFavorite: false,
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

const items = [
	item('a1', '2026-11-02T10:00:00Z'),
	item('a2', '2026-11-01T09:00:00Z'),
	item('a3', '2026-10-13T12:30:00Z')
];

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

test('the wall folds bare items into the month/day tree and packs rows', () => {
	const wall = mount(Wall, { target: document.body, props: { items } });
	flushSync();

	// The grouping is the wall's own: bare items in, month and day
	// headers out (the shared wall__ density language).
	const months = Array.from(document.body.querySelectorAll('.wall__month-header'));
	expect(months.map((header) => header.textContent)).toEqual(['November 2026', 'October 2026']);
	const days = Array.from(document.body.querySelectorAll('.wall__day-header'));
	expect(days.map((header) => header.textContent)).toEqual([
		'Monday, November 2',
		'Sunday, November 1',
		'Tuesday, October 13'
	]);
	// jsdom has no observer/box metrics: the fallback width packs one
	// item per day group (the engine's determinism).
	const rows = Array.from(document.body.querySelectorAll('.wall__row'));
	expect(rows).toHaveLength(3);
	expect(document.body.querySelectorAll('.tile')).toHaveLength(3);

	unmount(wall);
	flushSync();
});

test('a wall without onselect renders no check affordance: the tiles are pure navigation', () => {
	// R5 (the album detail's wall): no callback wired, no selection
	// possible -- NO hover checkbox renders at all, and the tile body
	// stays the viewer anchor.
	const wall = mount(Wall, { target: document.body, props: { items } });
	flushSync();

	expect(document.body.querySelector('.tile__check')).toBeNull();
	const link = document.body.querySelector<HTMLAnchorElement>('a.tile__link');
	expect(link?.getAttribute('href')).toBe('/a/a1');

	const click = new MouseEvent('click', { bubbles: true, cancelable: true });
	link?.dispatchEvent(click);
	flushSync();
	expect(click.defaultPrevented).toBe(false);

	unmount(wall);
	flushSync();
});

test('a wall with onselect keeps the check affordance and forwards its toggles', () => {
	const onselect = vi.fn();
	const wall = mount(Wall, {
		target: document.body,
		props: { items, selectable: true, isSelected: (id: string) => id === 'a2', onselect }
	});
	flushSync();

	const checks = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.tile__check'));
	expect(checks).toHaveLength(3);
	// isSelected drives the aria state per tile.
	expect(checks[1]?.getAttribute('aria-checked')).toBe('true');
	expect(checks[0]?.getAttribute('aria-checked')).toBe('false');

	checks[0]?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(onselect).toHaveBeenCalledWith('a1', true);

	unmount(wall);
	flushSync();
});
