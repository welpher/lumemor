import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Wall from './Wall.svelte';
import type { TimelineItem } from '$lib/api/timeline';

// The unified wall: ONE rendering
// component every photo surface composes -- the timeline, the album
// detail, the favorites/hidden wrappers, the search browse mode. It
// takes bare ITEMS (the month/day fold is its own), the row engine,
// the ResizeObserver re-pack, PhotoTile AND the month axis live
// inside it (the 2026-11-08 axis pin: the photo rendering and the
// right-edge rail are one thing -- the axis folds from the same
// internal groups the rendering packs), and so does the SELECTION
// FLOW: the selection
// state, the select mode, the Escape exit and the batch bar are the
// wall's own, gated by `onbatch`'s PRESENCE -- a wall with no runner
// renders NO check affordance and NO bar at all (the sync album's
// pure-navigation wall), while the walls that wire it keep the
// hover-reveal entry and the choreography: a landed run clears the
// selection (an armed mode persists), a failure keeps it and maps
// its error into the bar, a busy run disables the buttons. The pages
// reach the flow through the bound selectMode, the batch runs and
// the dialog openers (the ids captured at open, the `done`
// continuation the pass's landing fires).

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

/** The wall's batch outcome: the ok/errorKey split the wall reads. */
type Outcome = { ok: true } | { ok: false; errorKey: string };

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

/** Drain the run's promise chain (the wall awaits the page's batch
 * run) and force the DOM flush. */
async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	flushSync();
}

/** Click one tile's check affordance (the selection entry). */
function clickTileCheck(index: number): void {
	const checks = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.tile__check'));
	checks[index]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
}

/** The icon-only bar actions name themselves by aria-label. */
function buttonByLabel(label: string): HTMLButtonElement {
	const found = document.body.querySelector<HTMLButtonElement>(`.batchbar button[aria-label="${label}"]`);
	if (found === null) {
		throw new Error(`missing batch button ${label}`);
	}
	return found;
}

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

test('a wall without onbatch renders no check and no bar: the tiles are pure navigation', () => {
	// The sync album's wall (restated on the onbatch gate): no
	// runner wired, no selection possible -- NO hover checkbox
	// renders at all, no bar ever rises, and the tile body stays the
	// viewer anchor.
	const wall = mount(Wall, { target: document.body, props: { items } });
	flushSync();

	expect(document.body.querySelector('.tile__check')).toBeNull();
	expect(document.body.querySelector('.batchbar')).toBeNull();
	const link = document.body.querySelector<HTMLAnchorElement>('a.tile__link');
	expect(link?.getAttribute('href')).toBe('/a/a1');

	const click = new MouseEvent('click', { bubbles: true, cancelable: true });
	link?.dispatchEvent(click);
	flushSync();
	expect(click.defaultPrevented).toBe(false);
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(wall);
	flushSync();
});

test('a wall with onbatch renders the check affordance and toggles its own selection', () => {
	const onbatch = vi.fn(async () => ({ ok: true }) as Outcome);
	const wall = mount(Wall, { target: document.body, props: { items, onbatch } });
	flushSync();

	const checks = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.tile__check'));
	expect(checks).toHaveLength(3);
	// The wall's selection starts empty: every check rests unchecked.
	expect(checks.map((check) => check.getAttribute('aria-checked'))).toEqual(['false', 'false', 'false']);

	// The check's click toggles the wall's own set: the aria state
	// follows, and the bar rises with the honest count.
	clickTileCheck(1);
	expect(checks[1]?.getAttribute('aria-checked')).toBe('true');
	expect(checks[0]?.getAttribute('aria-checked')).toBe('false');
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');

	// The tile body toggles too while the selection is alive (the
	// mid-selection reveal): the click never reaches the viewer
	// anchor.
	const link = document.body.querySelector<HTMLAnchorElement>('a.tile__link');
	const click = new MouseEvent('click', { bubbles: true, cancelable: true });
	link?.dispatchEvent(click);
	flushSync();
	expect(click.defaultPrevented).toBe(true);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('2 selected');

	unmount(wall);
	flushSync();
});

test('a landed batch run clears the selection and persists an armed select mode', async () => {
	const onbatch = vi.fn(async () => ({ ok: true }) as Outcome);
	// The armed mode (the timeline's ?album= handoff writes the
	// bindable; the curated toolbar toggles it).
	const wall = mount(Wall, {
		target: document.body,
		props: { items, selectMode: true, onbatch }
	});
	flushSync();
	expect(document.body.querySelector('.tile')?.className).toContain('tile--selectable');

	clickTileCheck(0);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');

	buttonByLabel('Trash').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	// The runner saw the wall's own selection.
	expect(onbatch).toHaveBeenCalledWith('trash', ['a1']);
	// The success arm: the selection cleared (the bar is gone) but
	// the armed mode PERSISTED (the checks stay revealed for the
	// next pass).
	expect(document.body.querySelector('.batchbar')).toBeNull();
	expect(document.body.querySelector('.tile')?.className).toContain('tile--selectable');
	expect(document.body.querySelector('.tile__check')?.getAttribute('aria-checked')).toBe('false');

	unmount(wall);
	flushSync();
});

test('a failed batch run keeps the selection and maps the error into the bar', async () => {
	const onbatch = vi.fn(async () => ({ ok: false, errorKey: 'internal' }) as Outcome);
	const wall = mount(Wall, { target: document.body, props: { items, onbatch } });
	flushSync();

	clickTileCheck(0);
	buttonByLabel('Hide').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	// The failure keeps the selection (the retry affordance) and
	// surfaces the mapped message.
	expect(onbatch).toHaveBeenCalledWith('hide', ['a1']);
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');
	expect(document.body.querySelector('.batchbar__error')?.textContent).toBe(
		'Something went wrong. Please try again.'
	);

	unmount(wall);
	flushSync();
});

test('a busy batch run disables the bar until the promise lands', async () => {
	let resolveRun: ((outcome: Outcome) => void) | undefined;
	const onbatch = vi.fn(
		() =>
			new Promise<Outcome>((resolve) => {
				resolveRun = resolve;
			})
	);
	const wall = mount(Wall, { target: document.body, props: { items, onbatch } });
	flushSync();

	clickTileCheck(0);
	buttonByLabel('Trash').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();

	// In flight: every action disables (one write at a time).
	for (const label of ['Favorite', 'Hide', 'Unhide', 'Trash', 'Restore']) {
		expect(buttonByLabel(label).disabled, label).toBe(true);
	}

	resolveRun?.({ ok: true });
	await settle();
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(wall);
	flushSync();
});

test('the dialog opener captures the ids at open; done clears the selection', () => {
	const onbatch = vi.fn(async () => ({ ok: true }) as Outcome);
	const onaddtag = vi.fn();
	const wall = mount(Wall, {
		target: document.body,
		props: { items, onbatch, onaddtag }
	});
	flushSync();

	clickTileCheck(0);
	clickTileCheck(2);
	buttonByLabel('Add tag').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();

	// The ids are the ones captured AT OPEN (the current selection,
	// handed over once) together with the done continuation.
	expect(onaddtag).toHaveBeenCalledTimes(1);
	const [ids, done] = onaddtag.mock.calls[0] as [string[], () => void];
	expect(ids).toEqual(['a1', 'a3']);
	expect(typeof done).toBe('function');

	// The pass lands (the page fires done from its report's done
	// affordance): the wall applies its success arm -- the selection
	// clears and the bar drops.
	done();
	flushSync();
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(wall);
	flushSync();
});

test('the bar carries no dialog affordance the page did not wire', () => {
	// The curated walls' shape: onbatch alone -- the tag opener is
	// the timeline's alone.
	const onbatch = vi.fn(async () => ({ ok: true }) as Outcome);
	const wall = mount(Wall, { target: document.body, props: { items, onbatch } });
	flushSync();

	clickTileCheck(0);
	expect(() => buttonByLabel('Add tag')).toThrow();
	expect(document.body.querySelector('.batchbar')?.textContent).toContain('1 selected');

	unmount(wall);
	flushSync();
});

test('the memory opener captures the ids at open; done clears the selection', () => {
	const onbatch = vi.fn(async () => ({ ok: true }) as Outcome);
	const onaddmemory = vi.fn();
	const wall = mount(Wall, {
		target: document.body,
		props: { items, onbatch, onaddmemory }
	});
	flushSync();

	clickTileCheck(1);
	buttonByLabel('Add to memory').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();

	// The ids are the ones captured AT OPEN together with the done
	// continuation -- the modal-pinned selection discipline, the same
	// hand-over as the tag opener.
	expect(onaddmemory).toHaveBeenCalledTimes(1);
	const [ids, done] = onaddmemory.mock.calls[0] as [string[], () => void];
	expect(ids).toEqual(['a2']);
	expect(typeof done).toBe('function');

	done();
	flushSync();
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(wall);
	flushSync();
});

test('a removal-only wall curates: the checks render, the minus action runs the write', async () => {
	// The memory wall's shape: NO timeline batch action at all (the
	// empty subset), the membership removal alone. The capability
	// gate widens to the removal runner's presence -- the checks
	// render exactly then, and the bar's action set is the minus
	// button and the clear mark.
	const onremovememory = vi.fn(async () => ({ ok: true }) as Outcome);
	const wall = mount(Wall, {
		target: document.body,
		props: { items, actions: [], onremovememory }
	});
	flushSync();

	// The checks render (the widened gate): the wall curates through
	// the removal alone.
	expect(document.body.querySelector('.tile__check')).not.toBeNull();

	clickTileCheck(0);
	const bar = document.body.querySelector('.batchbar');
	expect(bar?.textContent).toContain('1 selected');
	// No timeline action buttons ride the bar -- the removal alone.
	expect(() => buttonByLabel('Favorite')).toThrow();
	expect(() => buttonByLabel('Trash')).toThrow();

	buttonByLabel('Remove from memory').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	// The write ran over the captured selection and the success arm
	// cleared it (the bar drops).
	expect(onremovememory).toHaveBeenCalledWith(['a1']);
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(wall);
	flushSync();
});

test('a failed removal keeps the selection and maps the error in the bar', async () => {
	const onremovememory = vi.fn(async () => ({ ok: false, errorKey: 'internal' }) as Outcome);
	const wall = mount(Wall, {
		target: document.body,
		props: { items, actions: [], onremovememory }
	});
	flushSync();

	clickTileCheck(0);
	buttonByLabel('Remove from memory').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();

	// The failure arm: the selection stays (the retry affordance) and
	// the mapped error renders in the bar.
	expect(document.body.querySelector('.batchbar')).not.toBeNull();
	expect(document.body.querySelector('.batchbar__error')?.textContent).toBe(
		'Something went wrong. Please try again.'
	);

	unmount(wall);
	flushSync();
});

test('Escape drops a live selection and an armed select mode', () => {
	const onbatch = vi.fn(async () => ({ ok: true }) as Outcome);

	// A live selection (the mid-selection reveal) drops: the bar and
	// the reveal go with it.
	const live = mount(Wall, { target: document.body, props: { items, onbatch } });
	flushSync();
	clickTileCheck(0);
	expect(document.body.querySelector('.batchbar')).not.toBeNull();
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
	flushSync();
	expect(document.body.querySelector('.batchbar')).toBeNull();
	expect(document.body.querySelector('.tile')?.className).not.toContain('tile--selectable');
	unmount(live);
	flushSync();

	// An armed mode with nothing picked (the ?album= arrival before
	// the first check) drops the same way: the full exit disarms.
	const armed = mount(Wall, {
		target: document.body,
		props: { items, onbatch, selectMode: true }
	});
	flushSync();
	expect(document.body.querySelector('.tile')?.className).toContain('tile--selectable');
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
	flushSync();
	expect(document.body.querySelector('.tile')?.className).not.toContain('tile--selectable');
	expect(document.body.querySelector('.batchbar')).toBeNull();

	unmount(armed);
	flushSync();
});

test("the month axis is the wall's own organ: one stop per month, folded from the same groups", () => {
	// The photo rendering and the right-edge
	// rail are one component. The axis cannot disagree with the
	// wall -- its months fold from the SAME internal groups the
	// rendering packs, so the stops' labels are the month headers'
	// labels, verbatim and in order.
	const wall = mount(Wall, { target: document.body, props: { items } });
	flushSync();

	const rail = document.body.querySelector('.scrubber');
	expect(rail).not.toBeNull();
	const stops = Array.from(document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop'));
	expect(stops).toHaveLength(2);
	expect(stops.map((stop) => stop.getAttribute('aria-label'))).toEqual(['November 2026', 'October 2026']);
	// The wall's month headers carry the same strings (the
	// same-fold guarantee, asserted from both sides).
	const headers = Array.from(document.body.querySelectorAll('.wall__month-header'));
	expect(headers.map((header) => header.textContent)).toEqual(['November 2026', 'October 2026']);

	unmount(wall);
	flushSync();
});

test("an empty wall renders no rail (the scrubber's own guard)", () => {
	const wall = mount(Wall, { target: document.body, props: { items: [] } });
	flushSync();

	expect(document.body.querySelector('.scrubber')).toBeNull();
	expect(document.body.querySelector('.wall__month')).toBeNull();

	unmount(wall);
	flushSync();
});
