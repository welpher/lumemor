import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import MonthScrubber from './MonthScrubber.svelte';

// The month scrubber: one stop per loaded month, a click jumps to
// its group, a drag scrubs by the pointer's height on the rail, and
// the click that follows a drag never jumps twice. jsdom has no
// layout, so the drag test stubs the rail's bounding rect.

const months = [
	{ key: '2026-11', label: 'November 2026' },
	{ key: '2026-10', label: 'October 2026' },
	{ key: '2026-09', label: 'September 2026' },
	{ key: '2026-08', label: 'August 2026' }
];

function mountScrubber(activeKey: string | null = null) {
	const onjump = vi.fn();
	const onscrub = vi.fn();
	const scrubber = mount(MonthScrubber, {
		target: document.body,
		props: { months, activeKey, onjump, onscrub }
	});
	flushSync();
	return { scrubber, onjump, onscrub };
}

function rail(): HTMLElement {
	const element = document.body.querySelector<HTMLElement>('.scrubber__rail');
	if (element === null) {
		throw new Error('missing scrubber rail');
	}
	return element;
}

function pointerEvent(type: string, clientY: number): PointerEvent {
	return new PointerEvent(type, { bubbles: true, cancelable: true, clientY });
}

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

test('renders one labeled stop per month with the navigation label', () => {
	const { scrubber, onjump } = mountScrubber('2026-10');

	const nav = document.body.querySelector('nav.scrubber');
	expect(nav?.getAttribute('aria-label')).toBe('Jump to month');
	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');
	// The year-only resting axis: the year labels the rail once per
	// year run (all four months are 2026); the months ride as dots
	// whose accessible names carry the full labels.
	// At REST the rail carries the faint years, the dots, the
	// position indicator -- and no year-month bubble (presence wakes
	// it). The years label each year run once (all four months are
	// 2026).
	const years = document.body.querySelectorAll('.scrubber__year');
	expect(Array.from(years).map((year) => year.textContent)).toEqual(['2026']);
	expect(document.body.querySelector('.scrubber__label')).toBeNull();
	expect(document.body.querySelector('.scrubber__readout')).toBeNull();
	expect(document.body.querySelector('.scrubber__indicator')).not.toBeNull();
	expect(Array.from(stops).map((stop) => stop.getAttribute('aria-label'))).toEqual(
		months.map((month) => month.label)
	);
	// The wall's scrolling wakes the bubble: the active month's FULL
	// label rides the indicator.
	document.dispatchEvent(new Event('scroll'));
	flushSync();
	const readout = document.body.querySelector('.scrubber__readout');
	expect(readout?.textContent).toBe('October 2026');
	// The stops sit at data-proportional offsets: four months of
	// equal count divide the usable span evenly (0, 1/4, 1/2, 3/4 of
	// a 140px rail minus the stop height: 0/31/62/93px).
	expect(stops[0]?.style.top).toBe('0px');
	expect(stops[1]?.style.top).toBe('31px');
	expect(stops[2]?.style.top).toBe('62px');
	expect(stops[3]?.style.top).toBe('93px');
	expect(stops[1]?.getAttribute('aria-current')).toBe('true');
	expect(stops[0]?.getAttribute('aria-current')).toBeNull();
	expect(onjump).not.toHaveBeenCalled();
	unmount(scrubber);
	flushSync();
});

test('clicking a stop jumps to its month (the keyboard path)', () => {
	const { scrubber, onjump } = mountScrubber();

	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');
	stops[2]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();

	expect(onjump).toHaveBeenCalledTimes(1);
	expect(onjump).toHaveBeenCalledWith('2026-09');
	unmount(scrubber);
	flushSync();
});

test('dragging the rail scrubs continuously by pointer height, and the trailing click is suppressed', () => {
	const { scrubber, onjump, onscrub } = mountScrubber();

	const element = rail();
	element.getBoundingClientRect = () =>
		({
			top: 0,
			bottom: 100,
			height: 100,
			left: 0,
			right: 20,
			width: 20,
			x: 0,
			y: 0,
			toJSON: () => ({})
		}) as DOMRect;

	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');

	// THE TRUE DRAG: pressing at a quarter of the rail scrubs the
	// wall to a quarter of its scroll span -- the rail IS a
	// scrollbar; months never jump mid-drag.
	stops[1]?.dispatchEvent(pointerEvent('pointerdown', 25));
	expect(onscrub).toHaveBeenCalledWith(0.25);
	expect(onjump).not.toHaveBeenCalled();

	// Dragging to three quarters scrubs there (the moves land on the
	// pressed stop without a pointer capture implementation -- the
	// handler computes from the rail rect either way).
	stops[1]?.dispatchEvent(pointerEvent('pointermove', 75));
	expect(onscrub).toHaveBeenCalledWith(0.75);
	expect(onjump).not.toHaveBeenCalled();

	// The click that follows a MOVED drag must not jump: the release
	// position is the drag's destination, not a tap's intent.
	stops[1]?.dispatchEvent(pointerEvent('pointerup', 75));
	stops[0]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onjump).not.toHaveBeenCalled();
	unmount(scrubber);
	flushSync();
});

test('a pure tap (press and release without moving) jumps to the month', () => {
	const { scrubber, onjump, onscrub } = mountScrubber();

	const element = rail();
	element.getBoundingClientRect = () =>
		({
			top: 0,
			bottom: 100,
			height: 100,
			left: 0,
			right: 20,
			width: 20,
			x: 0,
			y: 0,
			toJSON: () => ({})
		}) as DOMRect;

	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');

	// A press without any move scrubs once (from its own position)
	// but stays a TAP: the trailing click jumps.
	stops[2]?.dispatchEvent(pointerEvent('pointerdown', 50));
	expect(onscrub).toHaveBeenCalledWith(0.5);
	stops[2]?.dispatchEvent(pointerEvent('pointerup', 50));
	stops[2]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onjump).toHaveBeenCalledWith('2026-09');
	unmount(scrubber);
	flushSync();
});

test('a layout-less rail (zero height) scrubs to the top and reveals the first month', () => {
	const { scrubber, onjump, onscrub } = mountScrubber();

	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');
	// A rail with no layout scrubs to the very top; the bubble's
	// fallback names the first month. No month jumps happen.
	stops[2]?.dispatchEvent(pointerEvent('pointerdown', 50));
	expect(onscrub).toHaveBeenCalledWith(0);
	expect(onjump).not.toHaveBeenCalled();
	stops[2]?.dispatchEvent(pointerEvent('pointermove', 50));
	expect(onscrub).toHaveBeenCalledTimes(2);
	expect(onjump).not.toHaveBeenCalled();
	unmount(scrubber);
	flushSync();
});

test('a cancelled press does not swallow the next genuine tap', () => {
	const { scrubber, onjump } = mountScrubber();
	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');

	// The press scrubs (not jumps), then the host cancels the drag
	// (a scroll appropriation, say): no click follows a cancel.
	stops[1]?.dispatchEvent(pointerEvent('pointerdown', 25));
	stops[1]?.dispatchEvent(pointerEvent('pointercancel', 25));

	// The next genuine tap must still jump (a sticky suppressClick
	// would swallow it exactly once).
	stops[2]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onjump).toHaveBeenCalledWith('2026-09');
	expect(onjump).toHaveBeenCalledTimes(1);
	unmount(scrubber);
	flushSync();
});

test('no months means no rail', () => {
	const scrubber = mount(MonthScrubber, {
		target: document.body,
		props: { months: [], activeKey: null, onjump: vi.fn() }
	});
	flushSync();
	expect(document.body.querySelector('.scrubber')).toBeNull();
	unmount(scrubber);
	flushSync();
});
