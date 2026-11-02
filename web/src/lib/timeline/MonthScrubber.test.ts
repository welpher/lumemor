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
	const scrubber = mount(MonthScrubber, {
		target: document.body,
		props: { months, activeKey, onjump }
	});
	flushSync();
	return { scrubber, onjump };
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
	expect(Array.from(stops).map((stop) => stop.textContent)).toEqual(
		months.map((month) => month.label)
	);
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

test('dragging the rail scrubs by pointer height, and the trailing click is suppressed', () => {
	const { scrubber, onjump } = mountScrubber();

	const element = rail();
	element.getBoundingClientRect = () =>
		({ top: 0, bottom: 100, height: 100, left: 0, right: 20, width: 20, x: 0, y: 0, toJSON: () => ({}) }) as DOMRect;

	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');

	// Pointer down on the second stop (a quarter down the rail).
	stops[1]?.dispatchEvent(pointerEvent('pointerdown', 25));
	expect(onjump).toHaveBeenCalledWith('2026-10');

	// Drag to three quarters: the fourth month (index 3). Without a
	// pointer capture implementation the moves land on the stop the
	// test drives -- the handler computes from the rail rect either
	// way.
	stops[1]?.dispatchEvent(pointerEvent('pointermove', 75));
	expect(onjump).toHaveBeenCalledWith('2026-08');
	expect(onjump).toHaveBeenCalledTimes(2);

	// The click that follows the pointerup on a stop must not jump.
	stops[1]?.dispatchEvent(pointerEvent('pointerup', 75));
	stops[0]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onjump).toHaveBeenCalledTimes(2);
	unmount(scrubber);
	flushSync();
});

test('a layout-less rail (zero height) answers the first month on drag', () => {
	const { scrubber, onjump } = mountScrubber();

	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');
	// The press jumps to its own stop; the drag across a rail with no
	// layout falls back to the top month.
	stops[2]?.dispatchEvent(pointerEvent('pointerdown', 50));
	expect(onjump).toHaveBeenCalledWith('2026-09');
	stops[2]?.dispatchEvent(pointerEvent('pointermove', 50));
	expect(onjump).toHaveBeenCalledWith('2026-11');

	unmount(scrubber);
	flushSync();
});

test('a cancelled press does not swallow the next genuine tap', () => {
	const { scrubber, onjump } = mountScrubber();

	const stops = document.body.querySelectorAll<HTMLButtonElement>('.scrubber__stop');

	// The press jumps, then the host cancels the drag (a scroll
	// appropriation, say): no click follows a cancel.
	stops[1]?.dispatchEvent(pointerEvent('pointerdown', 25));
	expect(onjump).toHaveBeenCalledWith('2026-10');
	stops[1]?.dispatchEvent(pointerEvent('pointercancel', 25));

	// The next genuine tap must still jump (a sticky suppressClick
	// would swallow it exactly once).
	stops[2]?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	expect(onjump).toHaveBeenCalledWith('2026-09');
	expect(onjump).toHaveBeenCalledTimes(2);
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
