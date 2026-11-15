import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
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
	props: { selected?: boolean; selectable?: boolean; sizes?: string; withSelect?: boolean } = {}
) {
	const onselect = vi.fn();
	const tile = mount(PhotoTile, {
		target: document.body,
		props: {
			item: item(overrides),
			selected: props.selected ?? false,
			selectable: props.selectable ?? false,
			onselect: props.withSelect === false ? undefined : onselect
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

// The image-state stubs answer the element's own completion state, the
// signal a cache-hit tile has instead of a load event (jsdom answers
// false/0 for an untouched element, which is why the existing tests
// dispatch `load` by hand).
afterEach(() => {
	vi.restoreAllMocks();
});

/** Stub the image element's completion state: `complete` and
 *  `naturalWidth` are the two halves of the loaded test (only the pair
 *  distinguishes a decoded image from a broken one). */
function stubImageState(complete: boolean, naturalWidth: number): void {
	vi.spyOn(HTMLImageElement.prototype, 'complete', 'get').mockReturnValue(complete);
	vi.spyOn(HTMLImageElement.prototype, 'naturalWidth', 'get').mockReturnValue(naturalWidth);
}

test('a ready asset requests the view-size derivative over the blurhash placeholder', () => {
	const { tile } = mountTile({});
	const image = tileRoot().querySelector('img.tile__image');
	expect(image?.getAttribute('src')).toBe('/api/v1/assets/a1/thumbnail?size=view');
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

test('the tile serves the view derivative alone: no srcset, no sizes', () => {
	// The 2026-11-07 blurry fix: the tile/view DPR pair
	// is GONE. The pair relied on the browser re-evaluating a
	// responsive-image candidate after the justified engine re-packed
	// and the sizes hint changed on an already-loaded <img> -- which
	// it does not reliably do, so a resize left soft tiles behind.
	// The wall now always serves the view derivative (the tile box's
	// 250px uniform-grid derivative never matched the justified
	// wall's 200-500px cells anyway; a self-hosted LAN app's
	// bandwidth cost is negligible and photos are always sharp).
	// The viewer's image branch keeps its own view/original pair.
	const { tile } = mountTile({});
	const image = tileRoot().querySelector('img.tile__image');
	expect(image?.getAttribute('src')).toBe('/api/v1/assets/a1/thumbnail?size=view');
	expect(image?.getAttribute('srcset')).toBeNull();
	expect(image?.getAttribute('sizes')).toBeNull();
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

test('a cache-hit image shows on mount, with no load event ever dispatched', () => {
	// THE HOT-CACHE MOUNT (the resize blur's unit pin): the wall
	// re-packs on a container resize and moves tiles between rows, so
	// Svelte destroys and re-creates the tile; the fresh <img> gets its
	// src before onload is attached, and a fully decoded image from the
	// memory cache dispatches `load` synchronously inside that window.
	// The event is lost forever -- there is no second load. The tile
	// therefore reads the element's own completion state on mount: a
	// cache hit is on screen at once (the image loaded, the placeholder
	// hidden), with no event dispatched in this test at all.
	stubImageState(true, 1080);
	const { tile } = mountTile({});
	const image = tileRoot().querySelector<HTMLElement>('img.tile__image');
	const canvas = tileRoot().querySelector<HTMLElement>('canvas.tile__placeholder');

	expect(image?.className).toContain('tile__image--loaded');
	expect(image?.className).not.toContain('opacity-0');
	expect(canvas?.className).toContain('tile__placeholder--hidden');
	expect(canvas?.className).toContain('opacity-0');
	unmount(tile);
	flushSync();
});

test('a failed image stays on the placeholder: complete alone never counts', () => {
	// The bare-`complete` trap: a broken image reports complete true
	// with naturalWidth 0 (and so does an element whose src was never
	// set). Only `complete && naturalWidth > 0` is a loaded image --
	// this tile keeps the blurhash placeholder and never fades in.
	stubImageState(true, 0);
	const { tile } = mountTile({});
	const image = tileRoot().querySelector<HTMLElement>('img.tile__image');
	const canvas = tileRoot().querySelector<HTMLElement>('canvas.tile__placeholder');

	expect(image?.className).not.toContain('tile__image--loaded');
	expect(image?.className).toContain('opacity-0');
	expect(canvas?.className).not.toContain('tile__placeholder--hidden');
	expect(canvas?.className).not.toContain('opacity-0');
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

test('a tile with no onselect renders no checkbox: pure navigation', () => {
	// The sync album's wall: the callback's absence IS the
	// no-selection contract -- no check circle renders at all (not
	// even the hover reveal), and the tile body stays the viewer
	// anchor. The caller never wires a dead affordance.
	const { tile, onselect } = mountTile({}, { withSelect: false });
	expect(tileRoot().querySelector('.tile__check')).toBeNull();

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

test('the check affordance reveals on hover and keyboard focus regardless of the select mode', () => {
	// The selection entry IS the check. The affordance hides
	// at rest but its reveal utilities are unconditional -- hover and
	// keyboard focus surface it on EVERY tile (never only while a
	// select mode is armed), and its click enters the selection even
	// when the wall is not selectable yet. The role/aria-checked/
	// aria-label contract survives exactly.
	const { tile, onselect } = mountTile({}, { selectable: false, selected: false });
	const checkbox = tileRoot().querySelector<HTMLButtonElement>('.tile__check');
	expect(checkbox).not.toBeNull();

	// Hidden at rest, revealed by the pointer or the keyboard --
	// always, not gated on the selectable prop.
	expect(checkbox?.className).toContain('opacity-0');
	expect(checkbox?.className).toContain('group-hover:opacity-100');
	expect(checkbox?.className).toContain('group-focus-within:opacity-100');
	// The a11y contract is exactly the wall's checkbox role.
	expect(checkbox?.getAttribute('role')).toBe('checkbox');
	expect(checkbox?.getAttribute('aria-checked')).toBe('false');
	expect(checkbox?.getAttribute('aria-label')).toBe('Select: IMG_1234.jpg');

	// The first check-click enters the selection (the batch bar rises
	// at >=1 on the wall's side of the callback).
	checkbox?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
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
	expect(tileRoot().querySelector('.tile__live')).not.toBeNull();
	unmount(live.tile);
	flushSync();

	const video = mountTile({ kind: 'video' });
	expect(tileRoot().querySelector('.tile__badge--video')).not.toBeNull();
	// A video never shows the live mark.
	expect(tileRoot().querySelector('.tile__live')).toBeNull();
	unmount(video.tile);
	flushSync();
});

test('the live mark is a bare type glyph at the top-right, named through the anchor', () => {
	// The pin: no plate behind it (the selection check already owns a
	// filled circle at the top-left; a second disc reads as a second
	// control), a drop shadow for legibility over any photo, and the
	// accessible name taking the type through the anchor alone.
	const { tile } = mountTile({ livePhotoVideoId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70' });
	const mark = tileRoot().querySelector<HTMLElement>('.tile__live');
	expect(mark).not.toBeNull();
	expect(mark?.getAttribute('aria-hidden')).toBe('true');
	expect(mark?.className).toContain('top-2');
	expect(mark?.className).toContain('right-2');
	expect(mark?.className).toContain('drop-shadow');
	// No plate: no background utility and no night surface.
	expect(mark?.className).not.toContain('bg-night');
	expect(mark?.className).not.toContain('rounded-full');
	// The anchor carries the type in its accessible name.
	expect(tileLink().getAttribute('aria-label')).toBe('IMG_1234.jpg · Live Photo');

	// A plain photo says nothing extra.
	unmount(tile);
	flushSync();
	const plain = mountTile({});
	expect(plain.tile && tileLink().getAttribute('aria-label')).toBe('IMG_1234.jpg');
	expect(tileRoot().querySelector('.tile__live')).toBeNull();
	unmount(plain.tile);
	flushSync();
});

test('a hover asks for the moment: the stream mounts, plays, pushes and flips the mark', () => {
	// The moment is not fetched at rest (the hidden video costs no
	// byte until a hover asks); entering the tile mounts the
	// component row's play stream, muted/looping, and only a stream
	// that is actually PLAYING takes the push and flips the glyph.
	const { tile } = mountTile({ livePhotoVideoId: 'c2' });
	const root = tileRoot();
	expect(root.querySelector('video.tile__motion')).toBeNull();

	root.dispatchEvent(new MouseEvent('mouseenter'));
	flushSync();
	const motion = root.querySelector<HTMLVideoElement>('video.tile__motion');
	expect(motion).not.toBeNull();
	expect(motion?.getAttribute('src')).toBe('/api/v1/assets/c2/play');
	expect(motion?.muted).toBe(true);
	expect(motion?.loop).toBe(true);
	expect(motion?.hasAttribute('autoplay')).toBe(true);
	expect(motion?.hasAttribute('controls')).toBe(false);
	// The push waits for the stream: nothing moves yet.
	expect(root.querySelector('.tile__media--moving')).toBeNull();

	motion?.dispatchEvent(new Event('playing'));
	flushSync();
	expect(root.querySelector('.tile__media--moving')).not.toBeNull();
	// The mark flipped to the pause pair (two bars, one svg path).
	expect(root.querySelectorAll('.tile__live path')).toHaveLength(1);

	root.dispatchEvent(new MouseEvent('mouseleave'));
	flushSync();
	expect(root.querySelector('video.tile__motion')).toBeNull();
	expect(root.querySelector('.tile__media--moving')).toBeNull();
	expect(root.querySelectorAll('.tile__live circle')).toHaveLength(2);
	unmount(tile);
	flushSync();
});

test('a keyboard focus plays too, and losing it resets', () => {
	const { tile } = mountTile({ livePhotoVideoId: 'c2' });
	const root = tileRoot();
	root.dispatchEvent(new FocusEvent('focusin', { bubbles: true }));
	flushSync();
	expect(root.querySelector('video.tile__motion')).not.toBeNull();
	root.dispatchEvent(new FocusEvent('focusout', { bubbles: true }));
	flushSync();
	expect(root.querySelector('video.tile__motion')).toBeNull();
	unmount(tile);
	flushSync();
});

test('a moment that cannot play keeps the still and the type mark, silently', () => {
	// The component's transcode may still be running (or its file may
	// be gone): the stream errors, the still stays, the mark stays the
	// type glyph, and nothing is announced. The next hover asks again
	// -- the transcode may have landed in between.
	const { tile } = mountTile({ livePhotoVideoId: 'c2' });
	const root = tileRoot();
	root.dispatchEvent(new MouseEvent('mouseenter'));
	flushSync();
	const motion = root.querySelector<HTMLVideoElement>('video.tile__motion');
	motion?.dispatchEvent(new Event('error'));
	flushSync();
	expect(root.querySelector('video.tile__motion')).toBeNull();
	expect(root.querySelector('.tile__media--moving')).toBeNull();
	expect(root.querySelectorAll('.tile__live circle')).toHaveLength(2);
	expect(root.textContent).not.toContain('error');

	// Leaving forgets the refusal: the next entry asks again.
	root.dispatchEvent(new MouseEvent('mouseleave'));
	flushSync();
	root.dispatchEvent(new MouseEvent('mouseenter'));
	flushSync();
	expect(root.querySelector('video.tile__motion')).not.toBeNull();
	unmount(tile);
	flushSync();
});

test('a plain photo never mounts a motion stream', () => {
	const { tile } = mountTile({});
	tileRoot().dispatchEvent(new MouseEvent('mouseenter'));
	flushSync();
	expect(tileRoot().querySelector('video.tile__motion')).toBeNull();
	unmount(tile);
	flushSync();
});

test('the hover scrim rides over the photo: hidden at rest, revealed on hover and keyboard focus', () => {
	// The restrained bottom veil: the Night-based gradient stays
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
	// The placeholder drops to opacity-0 over the placeholder
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
