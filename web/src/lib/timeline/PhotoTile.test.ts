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

function mountTile(
	overrides: Partial<TimelineItem>,
	props: { selected?: boolean; selectable?: boolean; sizes?: string } = {}
) {
	const onselect = vi.fn();
	const tile = mount(PhotoTile, {
		target: document.body,
		props: {
			item: item(overrides),
			selected: props.selected ?? false,
			selectable: props.selectable ?? false,
			sizes: props.sizes,
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

test('the tile serves the tile and view derivatives through a DPR srcset', () => {
	// R2-2: the 250px tile box feeding a Retina-density wall tile
	// reads soft, so the tile offers the view derivative as its 2x
	// candidate. The x descriptors are the honest pair for the
	// endpoint's aspect-variable files (a w descriptor would have to
	// lie about the file's intrinsic width); the src stays the 1x
	// candidate as the no-srcset fallback.
	const { tile } = mountTile({});
	const image = tileRoot().querySelector('img.tile__image');
	expect(image?.getAttribute('srcset')).toBe(
		'/api/v1/assets/a1/thumbnail?size=tile 1x, /api/v1/assets/a1/thumbnail?size=view 2x'
	);
	// The width hint: the justified wall passes the row engine's
	// exact cell width; the default covers the callers that do not
	// know it (the trash grid's fluid cells).
	expect(image?.getAttribute('sizes')).toBe('240px');
	unmount(tile);
	flushSync();
});

test('a caller that knows the rendered width passes it as the sizes hint', () => {
	const { tile } = mountTile({}, { sizes: '288px' });
	const image = tileRoot().querySelector('img.tile__image');
	expect(image?.getAttribute('sizes')).toBe('288px');
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

test('the hover scrim rides over the photo: hidden at rest, revealed on hover and keyboard focus', () => {
	// G3's restrained bottom veil: the Night-based gradient stays
	// opacity-0 until the pointer or the keyboard focus enters the
	// tile, never takes a pointer event, and never scales the photo.
	const { tile } = mountTile({});
	const scrim = tileRoot().querySelector<HTMLElement>('.tile__scrim');
	expect(scrim).not.toBeNull();
	expect(scrim?.getAttribute('aria-hidden')).toBe('true');
	expect(scrim?.className).toContain('pointer-events-none');
	expect(scrim?.className).toContain('opacity-0');
	expect(scrim?.className).toContain('group-hover:opacity-100');
	expect(scrim?.className).toContain('group-focus-within:opacity-100');
	expect(scrim?.className).toContain('from-night/55');
	unmount(tile);
	flushSync();
});

test('a video tile carries the duration chip: tabular numerals over the night chip', () => {
	const { tile } = mountTile({ kind: 'video', durationMs: 61_500 });
	const badge = tileRoot().querySelector<HTMLElement>('.tile__badge--video');
	expect(badge).not.toBeNull();
	// The chip renders the play mark plus the m:ss duration.
	expect(badge?.textContent).toContain('1:02');
	expect(badge?.className).toContain('tabular-nums');
	expect(badge?.className).toContain('bg-night');
	expect(badge?.getAttribute('title')).toBe('Video');
	unmount(tile);
	flushSync();
});

test('a video tile without a known duration keeps the play mark alone', () => {
	const { tile } = mountTile({ kind: 'video', durationMs: null });
	const badge = tileRoot().querySelector<HTMLElement>('.tile__badge--video');
	expect(badge).not.toBeNull();
	expect(badge?.textContent).not.toContain(':');
	unmount(tile);
	flushSync();
});

test('the favorite star is a hover-revealed state riding the scrim zone', () => {
	// The state affordance the scrim reveals: the star rests
	// invisible in the bottom-left corner and fades in with the
	// scrim on hover and keyboard focus.
	const { tile } = mountTile({ isFavorite: true });
	const star = tileRoot().querySelector<HTMLElement>('.tile__badge--favorite');
	expect(star).not.toBeNull();
	expect(star?.className).toContain('bottom-1.5');
	expect(star?.className).toContain('opacity-0');
	expect(star?.className).toContain('group-hover:opacity-100');
	expect(star?.className).toContain('group-focus-within:opacity-100');
	unmount(tile);
	flushSync();
});

test('the blurhash placeholder cross-fades out when the image loads', () => {
	// G6: the placeholder drops to opacity-0 over the placeholder
	// fade token once the thumbnail's load event lands.
	const { tile } = mountTile({});
	const canvas = tileRoot().querySelector<HTMLElement>('canvas.tile__placeholder');
	const image = tileRoot().querySelector<HTMLElement>('img.tile__image');
	expect(canvas?.className).not.toContain('tile__placeholder--hidden');

	image?.dispatchEvent(new Event('load'));
	flushSync();
	expect(canvas?.className).toContain('tile__placeholder--hidden');
	expect(canvas?.className).toContain('duration-(--motion-placeholder-fade)');

	image?.dispatchEvent(new Event('error'));
	flushSync();
	// A failed load gives the placeholder back.
	expect(canvas?.className).not.toContain('tile__placeholder--hidden');
	unmount(tile);
	flushSync();
});

test('selection is geometric: the resting tile is bare, the selected tile retreats and rings', () => {
	// The visual semantics of the justified wall (adapted from the
	// old 3px inner-stroke + always-rounded look): a RESTING tile is
	// pure photo edge-to-edge -- no radius, no outline; the SELECTED
	// tile retreats (the scale), gains the radius and draws the
	// accent ring at its edge. No reflow -- transform and outline.
	const resting = mountTile({});
	const bare = tileRoot();
	expect(bare.className).not.toContain('rounded-control');
	expect(bare.className).not.toContain('outline');
	expect(bare.className).not.toContain('scale-');
	unmount(resting.tile);
	flushSync();

	const selected = mountTile({}, { selectable: true, selected: true });
	const geometry = tileRoot();
	expect(geometry.className).toContain('tile--selected');
	expect(geometry.className).toContain('rounded-control');
	expect(geometry.className).toContain('scale-(--wall-select-scale)');
	expect(geometry.className).toContain('[outline:var(--wall-select-ring)_solid_var(--color-accent)]');
	unmount(selected.tile);
	flushSync();
});
