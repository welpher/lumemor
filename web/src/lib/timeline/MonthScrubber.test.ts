import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import MonthScrubber from './MonthScrubber.svelte';

// The month axis as ONE control: the rail is a scrollbar whose
// press jumps where you point and whose hold drags the wall, the
// readout follows the pointer and rides the indicator while the
// wall scrolls, and the keyboard is one tab stop walking the
// months. The geometry is the wall's own (`tops`/`maxScroll`), so
// the dot you see, the place a press takes you and the month the
// readout names are one mapping. jsdom has no layout, so the
// pointer tests stub the rail's bounding rect (height 140: the
// four-month fallback rail, whose usable span 124 makes clean
// fractions).

const months = [
	{ key: '2026-11', label: 'November 2026' },
	{ key: '2026-10', label: 'October 2026' },
	{ key: '2026-09', label: 'September 2026' },
	{ key: '2026-08', label: 'August 2026' }
];

// The wall's own landings: four equal months over a 100px span.
const tops = [0, 25, 50, 75];

type ScrubberProps = {
	months: readonly { key: string; label: string }[];
	tops?: readonly number[];
	maxScroll?: number;
	activeKey?: string | null;
	progress?: number;
	variant?: 'page' | 'panel';
	onjump: (key: string) => void;
	onscrub?: (fraction: number) => void;
};

function mountScrubber(props: Partial<ScrubberProps> = {}) {
	const onjump = vi.fn();
	const onscrub = vi.fn();
	const scrubber = mount(MonthScrubber, {
		target: document.body,
		props: { months, tops, maxScroll: 100, onjump, onscrub, ...props }
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

function stubRailRect(element: HTMLElement): void {
	element.getBoundingClientRect = () =>
		({
			top: 0,
			bottom: 140,
			height: 140,
			left: 0,
			right: 72,
			width: 72,
			x: 0,
			y: 0,
			toJSON: () => ({})
		}) as DOMRect;
}

function pointerEvent(type: string, clientY: number): PointerEvent {
	return new PointerEvent(type, { bubbles: true, cancelable: true, clientY });
}

function readout(): HTMLElement {
	const element = document.body.querySelector<HTMLElement>('.scrubber__readout');
	if (element === null) {
		throw new Error('missing scrubber readout');
	}
	return element;
}

function indicator(): HTMLElement {
	const element = document.body.querySelector<HTMLElement>('.scrubber__indicator');
	if (element === null) {
		throw new Error('missing scrubber indicator');
	}
	return element;
}

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

test('renders one scrollbar control, not a row of month buttons', () => {
	const { scrubber, onjump } = mountScrubber({ activeKey: '2026-10' });

	const nav = document.body.querySelector('nav.scrubber');
	expect(nav?.getAttribute('aria-label')).toBe('Jump to month');
	const control = rail();
	expect(control.getAttribute('role')).toBe('scrollbar');
	expect(control.getAttribute('tabindex')).toBe('0');
	expect(control.getAttribute('aria-orientation')).toBe('vertical');
	expect(control.getAttribute('aria-valuemin')).toBe('0');
	expect(control.getAttribute('aria-valuemax')).toBe('100');
	expect(control.className).toContain('cursor-row-resize');
	// No month button ever: the dots are decoration, the rail is the
	// only interactive thing on the axis.
	expect(control.querySelector('button')).toBeNull();
	const stops = document.body.querySelectorAll('.scrubber__stop');
	expect(stops).toHaveLength(4);
	// The year-only resting axis: the year labels the rail once per
	// year run (all four months are 2026).
	const years = Array.from(document.body.querySelectorAll('.scrubber__year'));
	expect(years.map((year) => year.textContent)).toEqual(['2026']);
	// The marks sit at the wall's own landings: four equal months
	// over a 100px span divide the 124px usable span evenly
	// (y = 8 + fraction * 124; the stop's top is the center minus 8).
	expect(Array.from(stops).map((stop) => (stop as HTMLElement).style.top)).toEqual(['0px', '31px', '62px', '93px']);
	// The scrollbar speaks the wall's reading: the progress is 0 at
	// the top and the accessible month name rides valuetext.
	expect(control.getAttribute('aria-valuenow')).toBe('0');
	expect(control.getAttribute('aria-valuetext')).toBe('October 2026');
	// At rest the readout is mounted but transparent (the fade is a
	// transition, not a mount) and the tick is out.
	expect(readout().style.opacity).toBe('0');
	expect(document.body.querySelector('.scrubber__tick')?.getAttribute('style')).toContain('opacity: 0');
	// The indicator rests at the top rim of the usable span.
	expect(indicator().style.top).toBe('7px');
	expect(onjump).not.toHaveBeenCalled();
	unmount(scrubber);
	flushSync();
});

test('the active month wears the accent dot', () => {
	const { scrubber } = mountScrubber({ activeKey: '2026-09' });

	const dots = Array.from(document.body.querySelectorAll<HTMLElement>('.scrubber__dot'));
	expect(dots[2]?.className).toContain('bg-accent');
	expect(dots[0]?.className).toContain('bg-border-strong');
	unmount(scrubber);
	flushSync();
});

test('a press jumps where you point, before the release', () => {
	const { scrubber, onjump, onscrub } = mountScrubber();
	const element = rail();
	stubRailRect(element);

	// A press at the rail's middle maps to the wall's middle: the
	// press IS the jump (the wall scrolls on the down, not the up).
	element.dispatchEvent(pointerEvent('pointerdown', 70));
	expect(onscrub).toHaveBeenCalledTimes(1);
	expect(onscrub).toHaveBeenCalledWith(0.5);
	expect(onjump).not.toHaveBeenCalled();
	unmount(scrubber);
	flushSync();
});

test('a held press drags the wall along and the readout follows the pointer', () => {
	const { scrubber, onscrub } = mountScrubber();
	const element = rail();
	stubRailRect(element);

	// The press lands mid-wall; the readout names the month under
	// the pointer and rides its height.
	element.dispatchEvent(pointerEvent('pointerdown', 70));
	flushSync();
	expect(onscrub).toHaveBeenCalledWith(0.5);
	expect(readout().textContent).toBe('September 2026');
	expect(readout().style.top).toBe('70px');
	// Mid-drag the solid bar stands down -- the tick is the position
	// marker while the pointer owns the axis.
	expect(indicator().style.opacity).toBe('0');
	expect(document.body.querySelector('.scrubber__tick')?.getAttribute('style')).toContain('opacity: 1');

	// Dragging to a quarter scrubs there (the moves land on the
	// pressed rail under pointer capture; the handler computes from
	// the rail rect either way).
	element.dispatchEvent(pointerEvent('pointermove', 39));
	flushSync();
	expect(onscrub).toHaveBeenLastCalledWith(0.25);
	expect(readout().textContent).toBe('October 2026');
	expect(readout().style.top).toBe('39px');

	// The release hands the position back to the solid bar.
	element.dispatchEvent(pointerEvent('pointerup', 39));
	flushSync();
	expect(indicator().style.opacity).toBe('1');
	unmount(scrubber);
	flushSync();
});

test('a hover reads the axis without moving the wall', () => {
	const { scrubber, onscrub } = mountScrubber();
	const element = rail();
	stubRailRect(element);

	// The hover reveals the readout (the month a press would land)
	// and the tick at the pointer -- and scrubs nothing.
	element.dispatchEvent(pointerEvent('pointermove', 101));
	flushSync();
	expect(onscrub).not.toHaveBeenCalled();
	expect(readout().textContent).toBe('August 2026');
	expect(readout().style.top).toBe('101px');
	expect(document.body.querySelector('.scrubber__tick')?.getAttribute('style')).toContain('opacity: 1');
	expect(indicator().style.opacity).toBe('1');

	// Leaving the rail fades both (the wall alone is awake then).
	element.dispatchEvent(pointerEvent('pointerleave', 0));
	flushSync();
	expect(readout().style.opacity).toBe('0');
	unmount(scrubber);
	flushSync();
});

test('a released drag lingers a beat before the readout fades', () => {
	vi.useFakeTimers();
	try {
		const { scrubber } = mountScrubber();
		const element = rail();
		stubRailRect(element);

		element.dispatchEvent(pointerEvent('pointerdown', 70));
		// The drag carries the pointer off the rail; the release (not
		// a hover) lingers the readout 700ms, then fades it.
		element.dispatchEvent(pointerEvent('pointerup', 70));
		flushSync();
		expect(readout().style.opacity).toBe('1');
		vi.advanceTimersByTime(700);
		flushSync();
		expect(readout().style.opacity).toBe('0');
		unmount(scrubber);
		flushSync();
	} finally {
		vi.useRealTimers();
	}
});

test('the wall alone scrolling wakes the readout on the indicator', () => {
	vi.useFakeTimers();
	try {
		const { scrubber } = mountScrubber({ activeKey: '2026-10', progress: 0.5 });

		// A scroll (no pointer, no focus): the readout rides the
		// indicator and names the month the wall reads -- and no tick
		// appears (a plain scroll has no aim).
		document.dispatchEvent(new Event('scroll'));
		flushSync();
		expect(readout().textContent).toBe('October 2026');
		expect(readout().style.opacity).toBe('1');
		expect(readout().style.top).toBe('70px');
		expect(document.body.querySelector('.scrubber__tick')?.getAttribute('style')).toContain('opacity: 0');

		// Nine hundred quiet milliseconds fade it again.
		vi.advanceTimersByTime(900);
		flushSync();
		expect(readout().style.opacity).toBe('0');
		unmount(scrubber);
		flushSync();
	} finally {
		vi.useRealTimers();
	}
});

test('the keyboard is one tab stop walking the months', () => {
	const { scrubber, onjump, onscrub } = mountScrubber();
	const element = rail();

	// From the top of the wall, the next month is the second one.
	element.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
	expect(onjump).toHaveBeenLastCalledWith('2026-10');
	element.dispatchEvent(new KeyboardEvent('keydown', { key: 'End', bubbles: true }));
	expect(onjump).toHaveBeenLastCalledWith('2026-08');
	element.dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true }));
	expect(onjump).toHaveBeenLastCalledWith('2026-11');
	// The walk scrubs nothing (it rides the page's own jump).
	expect(onscrub).not.toHaveBeenCalled();
	unmount(scrubber);
	flushSync();
});

test('the keyboard walks on from the month the wall reads', () => {
	const { scrubber, onjump } = mountScrubber({ activeKey: '2026-09' });
	const element = rail();

	element.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }));
	expect(onjump).toHaveBeenLastCalledWith('2026-08');
	element.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }));
	expect(onjump).toHaveBeenLastCalledWith('2026-10');
	unmount(scrubber);
	flushSync();
});

test('the keyboard focus shows the readout (the focus indicator), never the tick', () => {
	const { scrubber } = mountScrubber({ activeKey: '2026-10', progress: 0.5 });
	const element = rail();

	element.dispatchEvent(new FocusEvent('focus'));
	flushSync();
	expect(readout().style.opacity).toBe('1');
	expect(readout().textContent).toBe('October 2026');
	expect(document.body.querySelector('.scrubber__tick')?.getAttribute('style')).toContain('opacity: 0');

	element.dispatchEvent(new FocusEvent('blur'));
	flushSync();
	expect(readout().style.opacity).toBe('0');
	unmount(scrubber);
	flushSync();
});

test('a layout-less rail (zero height) scrubs to the top and names the first month', () => {
	const { scrubber, onscrub } = mountScrubber();
	const element = rail();

	// A rail with no layout scrubs to the very top; the readout's
	// fallback names the first month. No jump happens on a press.
	element.dispatchEvent(pointerEvent('pointerdown', 50));
	expect(onscrub).toHaveBeenCalledWith(0);
	element.dispatchEvent(pointerEvent('pointermove', 50));
	expect(onscrub).toHaveBeenCalledTimes(2);
	unmount(scrubber);
	flushSync();
});

test('the panel frame renders the same control in the compact slot', () => {
	const { scrubber, onscrub } = mountScrubber({ variant: 'panel' });

	const axis = document.body.querySelector('[data-testid="panel-axis"]');
	expect(axis).not.toBeNull();
	const control = axis?.querySelector('.scrubber__rail');
	expect(control?.getAttribute('role')).toBe('scrollbar');
	expect(control?.className).toContain('cursor-row-resize');
	expect(control?.querySelector('button')).toBeNull();
	// No fixed page rail in the panel frame.
	expect(document.body.querySelector('nav.scrubber')).toBeNull();
	// The compact dots sit at the same mapped fractions (the narrow
	// slot changes the geometry, never the mapping).
	const stops = Array.from(document.body.querySelectorAll<HTMLElement>('.scrubber__stop'));
	expect(stops.map((stop) => stop.style.top)).toEqual(['0px', '31px', '62px', '93px']);

	const element = axis?.querySelector<HTMLElement>('.scrubber__rail');
	if (element !== undefined && element !== null) {
		stubRailRect(element);
		element.dispatchEvent(pointerEvent('pointerdown', 70));
		expect(onscrub).toHaveBeenCalledWith(0.5);
	}
	unmount(scrubber);
	flushSync();
});

test('no months means no rail', () => {
	const scrubber = mount(MonthScrubber, {
		target: document.body,
		props: { months: [], tops: [], maxScroll: 0, onjump: vi.fn() }
	});
	flushSync();
	expect(document.body.querySelector('.scrubber')).toBeNull();
	unmount(scrubber);
	flushSync();
});
