import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { anchored } from './anchored';

// The anchored floating layer (web IA §7.5's 2026-11-09 second
// revision): the pickers' shared machinery -- portal to body,
// placement under the anchor with the anchor's width, the near-bottom
// flip, the bounded max-height (long lists scroll internally), the
// scroll/resize repositioning, and the destroy cleanup. The geometry
// is driven by hand-stubbed rects (jsdom lays out nothing): the
// anchor's rect, the layer's height and the viewport's height are
// pinned per case, and the assertions read the style the action
// wrote.

const anchor = document.createElement('div');
const node = document.createElement('div');
/** The live action under test; the harness destroys it after every case (the listeners are real). */
let action: ReturnType<typeof anchored> | undefined;

/** Pin one anchor rect (the trigger's box). */
function anchorRect(rect: { left: number; top: number; bottom: number; width: number }): void {
	vi.spyOn(anchor, 'getBoundingClientRect').mockReturnValue({
		left: rect.left,
		top: rect.top,
		right: rect.left + rect.width,
		bottom: rect.bottom,
		width: rect.width,
		height: rect.bottom - rect.top,
		x: rect.left,
		y: rect.top,
		toJSON: () => ({})
	} as DOMRect);
}

/** Pin the layer's measured height (jsdom lays out nothing). */
function layerHeight(height: number): void {
	Object.defineProperty(node, 'offsetHeight', {
		get: () => height,
		configurable: true
	});
}

/** Pin the viewport's height. */
function viewport(height: number): void {
	vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(height);
}

/** Run the action under test (one per case; the harness cleans up). */
function run(): void {
	action = anchored(node, { anchor: () => anchor });
}

beforeEach(() => {
	document.body.textContent = '';
	node.removeAttribute('style');
});

afterEach(() => {
	action?.destroy?.();
	action = undefined;
	vi.restoreAllMocks();
});

test('the layer portals to the body, anchored under the trigger with the trigger\'s width', () => {
	viewport(768);
	anchorRect({ left: 100, top: 400, bottom: 500, width: 300 });
	layerHeight(200);

	run();

	// The portal: the layer lives on the body, out of every dialog DOM.
	expect(node.parentElement).toBe(document.body);
	// The placement: directly under the anchor, exactly its width,
	// the default gap of 8px.
	expect(node.style.left).toBe('100px');
	expect(node.style.width).toBe('300px');
	expect(node.style.top).toBe('508px');

});

test('a layer that would overflow the viewport\'s bottom flips up when above is the roomier side', () => {
	viewport(768);
	// The anchor sits near the bottom: below holds 52px, above 584px.
	anchorRect({ left: 40, top: 600, bottom: 700, width: 300 });
	layerHeight(200);

	run();

	// The flip: the layer's bottom edge rides the anchor's top,
	// the same gap away.
	expect(node.style.top).toBe(`${600 - 8 - 200}px`);

});

test('a short layer stays below even near the bottom (the flip needs the overflow AND the roomier side)', () => {
	viewport(768);
	anchorRect({ left: 40, top: 600, bottom: 700, width: 300 });
	layerHeight(40);

	run();

	// 700 + 8 + 40 = 748 < 768 - 8: below holds, no flip.
	expect(node.style.top).toBe('708px');

});

test('long lists are bounded: the max-height fits the roomier side, hard-capped', () => {
	viewport(768);
	anchorRect({ left: 40, top: 400, bottom: 500, width: 300 });
	layerHeight(200);

	run();

	// Below holds 768 - 500 - 8 - 8 = 252px, above 384px -- the
	// layer's internal scroll bound is the roomier side, capped.
	expect(node.style.maxHeight).toBe('320px');

	// A squeezed viewport with the anchor centered in it: both sides
	// hold under the minimum bound, so the layer's internal scroll
	// bound collapses to it.
	viewport(260);
	anchorRect({ left: 40, top: 130, bottom: 140, width: 300 });
	window.dispatchEvent(new Event('resize'));
	expect(node.style.maxHeight).toBe('120px');

});

test('scrolls and resizes re-place the layer; the anchor is read live', () => {
	viewport(768);
	anchorRect({ left: 100, top: 400, bottom: 500, width: 300 });
	layerHeight(120);

	run();
	expect(node.style.top).toBe('508px');

	// The modal body scrolls (the anchor moves): a non-bubbling
	// scroll on a descendant still reaches the window's CAPTURE arm
	// -- the capture phase runs for every targeted descendant, so
	// the inner scroll re-places the portaled layer.
	anchorRect({ left: 100, top: 100, bottom: 200, width: 300 });
	const scroller = document.createElement('div');
	document.body.appendChild(scroller);
	scroller.dispatchEvent(new Event('scroll', { bubbles: false }));
	expect(node.style.top).toBe('208px');

	// The resize arm re-places too.
	anchorRect({ left: 100, top: 300, bottom: 400, width: 300 });
	window.dispatchEvent(new Event('resize'));
	expect(node.style.top).toBe('408px');

});

test('destroy removes the listeners and the portaled node', () => {
	viewport(768);
	anchorRect({ left: 100, top: 400, bottom: 500, width: 300 });
	layerHeight(100);

	run();
	expect(node.parentElement).toBe(document.body);

	action?.destroy?.();
	expect(node.parentElement).toBeNull();

	// The listeners are gone: a scroll after the destroy neither
	// throws nor moves the detached node.
	anchorRect({ left: 100, top: 100, bottom: 200, width: 300 });
	window.dispatchEvent(new Event('scroll'));
	expect(node.style.top).toBe('508px');
});
