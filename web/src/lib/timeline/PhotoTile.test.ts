import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import PhotoTile from './PhotoTile.svelte';
import type { TimelineItem } from '$lib/api/timeline';

// The tile contract: the blurhash placeholder renders first, the
// thumbnail (tile size) only requests once the asset is ready and
// crossfades in on load; selection toggles through the checkbox and
// the tile body while selection is active, and the tile is the
// viewer anchor otherwise. jsdom answers a null canvas context, so
// the decode path stays guarded here (its failure would throw and
// fail the mount, which the tests would catch).

const itemBase = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image',
	originalFilename: 'IMG_1234.jpg',
	fileSize: 4821933,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d659a2feaa0c55ad015',
	createdAt: '2026-10-29T09:12:00Z',
	takenAt: '2026-10-13T12:30:00Z',
	takenAtSource: 'exif',
	isTrashed: false,
	isFavorite: false,
	isReady: true,
	width: 4032,
	height: 3024,
	durationMs: null,
	blurhash: 'UHL6aAof00WBWB00ayWB00',
	livePhotoVideoId: null
} as const;

function item(overrides: Partial<TimelineItem> = {}): TimelineItem {
	return { id: 'a1', ...itemBase, ...overrides } as TimelineItem;
}

function mountTile(overrides: Partial<TimelineItem>, props: { selected?: boolean; selectable?: boolean } = {}) {
	const onselect = vi.fn();
	const tile = mount(PhotoTile, {
		target: document.body,
		props: {
			item: item(overrides),
			selected: props.selected ?? false,
			selectable: props.selectable ?? false,
			onselect
		}
	});
	flushSync();
	return { tile, onselect };
}

function tileRoot(): HTMLElement {
	const element = document.body.querySelector<HTMLElement>('.tile');
	if (element === null) {
		throw new Error('missing tile root');
	}
	return element;
}

function tileLink(): HTMLAnchorElement {
	const element = tileRoot().querySelector<HTMLAnchorElement>('a.tile__link');
	if (element === null) {
		throw new Error('missing tile link');
	}
	return element;
}

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

test('a ready asset requests the tile-size thumbnail over the blurhash placeholder', () => {
	const { tile } = mountTile({});
	const image = tileRoot().querySelector('img.tile__image');
	expect(image?.getAttribute('src')).toBe('/api/v1/assets/a1/thumbnail?size=tile');
	// The accessible name lives on the anchor; the image itself is
	// the anchor's visual content (an empty alt would repeat it).
	expect(image?.getAttribute('alt')).toBe('');
	expect(tileLink().getAttribute('aria-label')).toBe('IMG_1234.jpg');
	expect(image?.getAttribute('loading')).toBe('lazy');
	// The placeholder canvas renders underneath (the blurhash decode
	// draws into it where a canvas context exists).
	expect(tileRoot().querySelector('canvas.tile__placeholder')).not.toBeNull();
	unmount(tile);
	flushSync();
});

test('the thumbnail crossfades in on load and drops back on error', () => {
	const { tile } = mountTile({});
	const image = tileRoot().querySelector('img.tile__image');
	expect(image?.className).not.toContain('tile__image--loaded');

	image?.dispatchEvent(new Event('load'));
	flushSync();
	expect(image?.className).toContain('tile__image--loaded');

	image?.dispatchEvent(new Event('error'));
	flushSync();
	expect(image?.className).not.toContain('tile__image--loaded');
	unmount(tile);
	flushSync();
});

test('a not-ready asset renders the placeholder only, no thumbnail request', () => {
	const { tile } = mountTile({ isReady: false, blurhash: null });
	expect(tileRoot().querySelector('img.tile__image')).toBeNull();
	expect(tileRoot().querySelector('.tile__placeholder--empty')).not.toBeNull();
	unmount(tile);
	flushSync();
});

test('the tile anchors into the viewer while selection is inactive', () => {
	const { tile, onselect } = mountTile({});
	expect(tileLink().getAttribute('href')).toBe('/a/a1');

	tileLink().dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
	expect(onselect).not.toHaveBeenCalled();
	unmount(tile);
	flushSync();
});

test('with selection active, the tile body toggles instead of navigating', () => {
	const { tile, onselect } = mountTile({}, { selectable: true, selected: true });
	const click = new MouseEvent('click', { bubbles: true, cancelable: true });
	tileLink().dispatchEvent(click);
	flushSync();

	expect(click.defaultPrevented).toBe(true);
	expect(onselect).toHaveBeenCalledWith('a1', false);
	unmount(tile);
	flushSync();
});

test('the checkbox reflects the selection state and toggles without navigating', () => {
	const { tile, onselect } = mountTile({}, { selectable: true, selected: false });
	const checkbox = tileRoot().querySelector<HTMLButtonElement>('.tile__check');
	expect(checkbox?.getAttribute('role')).toBe('checkbox');
	expect(checkbox?.getAttribute('aria-checked')).toBe('false');
	expect(checkbox?.getAttribute('aria-label')).toBe('Select: IMG_1234.jpg');

	// The checkbox is a sibling of the anchor: its click never
	// reaches the link.
	const click = new MouseEvent('click', { bubbles: true, cancelable: true });
	checkbox?.dispatchEvent(click);
	flushSync();

	expect(click.defaultPrevented).toBe(false);
	expect(onselect).toHaveBeenCalledWith('a1', true);
	unmount(tile);
	flushSync();
});

test('badges: favorite star, live photo mark, video mark', () => {
	const favorite = mountTile({ isFavorite: true });
	expect(tileRoot().querySelector('.tile__badge--favorite')).not.toBeNull();
	unmount(favorite.tile);
	flushSync();

	const live = mountTile({ livePhotoVideoId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70' });
	expect(tileRoot().querySelector('.tile__badge--live')).not.toBeNull();
	unmount(live.tile);
	flushSync();

	const video = mountTile({ kind: 'video' });
	expect(tileRoot().querySelector('.tile__badge--video')).not.toBeNull();
	// A video never shows the live badge.
	expect(tileRoot().querySelector('.tile__badge--live')).toBeNull();
	unmount(video.tile);
	flushSync();
});
