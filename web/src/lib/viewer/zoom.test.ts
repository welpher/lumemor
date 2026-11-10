import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { createZoomEngine, VIEWER_ZOOM_BASE, VIEWER_ZOOM_MAX, type ZoomState } from './zoom';

// The gradual zoom engine's contract (R2-7/8): every input animates
// the scale toward a lattice target through rAF interpolation
// (~300ms, cubicOut), the scale clamps to the same 2x ceiling the old
// binary zoom had, and -- the risky half -- an input that lands
// mid-animation TAKES OVER FROM THE CURRENT INTERPOLATED SCALE, never
// from the target the interrupted animation was heading to.
//
// The clock is stubbed whole: requestAnimationFrame captures one
// pending callback, performance.now reads the harness's synthetic
// timeline, and advance() delivers one frame at a chosen offset. The
// engine therefore never depends on real timers here.

type FrameCallback = (now: number) => void;

let pending: FrameCallback | null = null;
let now = 0;

const states: ZoomState[] = [];

beforeEach(() => {
	pending = null;
	now = 0;
	states.length = 0;
	vi.stubGlobal('requestAnimationFrame', (callback: FrameCallback) => {
		pending = callback;
		return 1;
	});
	vi.stubGlobal('cancelAnimationFrame', () => {
		pending = null;
	});
	vi.spyOn(performance, 'now').mockImplementation(() => now);
});

afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
});

/** Deliver one animation frame at the given offset from the clock. */
function advance(delta: number): void {
	now += delta;
	const callback = pending;
	pending = null;
	callback?.(now);
}

function engine() {
	return createZoomEngine((state) => states.push({ ...state }));
}

test('a step input interpolates toward the lattice target and lands exactly', () => {
	const zoom = engine();

	zoom.zoomIn();
	// The target applies synchronously (the affordances key on it);
	// the scale still sits at the start.
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_BASE, target: 1.5 });
	expect(pending).not.toBeNull();

	advance(0);
	expect(states.at(-1)?.scale).toBe(VIEWER_ZOOM_BASE);
	advance(150);
	// Halfway through the clock, cubicOut has already covered 87.5%
	// of the distance (the fast start the ease is chosen for).
	expect(states.at(-1)?.scale).toBeCloseTo(1.4375, 5);
	advance(150);
	// Landed exactly on the lattice point; the loop stopped.
	expect(states.at(-1)).toEqual({ scale: 1.5, target: 1.5 });
	expect(pending).toBeNull();

	// The second step reaches the 2x ceiling the old binary zoom had.
	zoom.zoomIn();
	advance(300);
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_MAX, target: VIEWER_ZOOM_MAX });
});

test('the scale clamps at both ends and inputs at a clamp are inert', () => {
	const zoom = engine();

	zoom.zoomIn();
	advance(300);
	zoom.zoomIn();
	advance(300);
	// At the ceiling: another zoom-in changes nothing, schedules
	// nothing, emits nothing.
	const settled = states.length;
	zoom.zoomIn();
	expect(states.length).toBe(settled);
	expect(pending).toBeNull();
	expect(states.at(-1)?.target).toBe(VIEWER_ZOOM_MAX);

	zoom.zoomOut();
	advance(300);
	zoom.zoomOut();
	advance(300);
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_BASE, target: VIEWER_ZOOM_BASE });
	// At the floor: zoom-out is inert the same way.
	const floor = states.length;
	zoom.zoomOut();
	expect(states.length).toBe(floor);
	expect(pending).toBeNull();
});

test('an input mid-animation takes over from the current interpolated scale', () => {
	const zoom = engine();

	zoom.zoomIn();
	advance(100);
	const takeover = states.at(-1)?.scale ?? 0;
	expect(takeover).toBeGreaterThan(VIEWER_ZOOM_BASE);
	expect(takeover).toBeLessThan(1.5);

	// A second zoom-in while the first is still flying: the new
	// animation must start from the CURRENT scale -- not snap to (or
	// rebase on) the 1.5 it was heading to.
	zoom.zoomIn();
	expect(states.at(-1)?.target).toBe(VIEWER_ZOOM_MAX);
	advance(0);
	expect(states.at(-1)?.scale).toBeCloseTo(takeover, 5);
	// And it still lands on the lattice at the end.
	advance(300);
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_MAX, target: VIEWER_ZOOM_MAX });

	// The reverse: zoom-out mid-flight rebases the same way (from the
	// ceiling the first out heads for 1.5; the second, pressed while
	// that one flies, retargets to the base).
	zoom.zoomOut();
	advance(100);
	const reversing = states.at(-1)?.scale ?? 0;
	zoom.zoomOut();
	expect(states.at(-1)?.target).toBe(VIEWER_ZOOM_BASE);
	advance(0);
	expect(states.at(-1)?.scale).toBeCloseTo(reversing, 5);
	advance(300);
	expect(states.at(-1)?.scale).toBe(VIEWER_ZOOM_BASE);
});

test('the binary toggle flips between the base and the zoomed ceiling', () => {
	const zoom = engine();

	zoom.toggleZoom();
	advance(300);
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_MAX, target: VIEWER_ZOOM_MAX });

	zoom.toggleZoom();
	advance(300);
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_BASE, target: VIEWER_ZOOM_BASE });

	// From a mid-lattice step the toggle reads "zoomed": it returns
	// to the base, not to the ceiling.
	zoom.zoomIn();
	advance(300);
	zoom.toggleZoom();
	advance(300);
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_BASE, target: VIEWER_ZOOM_BASE });
});

test('the reset animates back to the base like every other input', () => {
	const zoom = engine();

	zoom.zoomIn();
	advance(300);
	zoom.reset();
	expect(states.at(-1)?.target).toBe(VIEWER_ZOOM_BASE);
	expect(pending).not.toBeNull();
	advance(300);
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_BASE, target: VIEWER_ZOOM_BASE });
});

test('the hard reset cancels a running animation and lands instantly', () => {
	const zoom = engine();

	zoom.zoomIn();
	advance(100);
	expect(pending).not.toBeNull();

	zoom.snapToBase();
	expect(states.at(-1)).toEqual({ scale: VIEWER_ZOOM_BASE, target: VIEWER_ZOOM_BASE });
	expect(pending).toBeNull();
});

test('destroy cancels the pending frame; nothing fires afterwards', () => {
	const zoom = engine();

	zoom.zoomIn();
	expect(pending).not.toBeNull();
	zoom.destroy();
	expect(pending).toBeNull();

	// The clock advancing after the destroy delivers nothing to the
	// (already captured) callback queue: no further state lands.
	const settled = states.length;
	advance(300);
	expect(states.length).toBe(settled);
});

test('a host without requestAnimationFrame lands the target instantly', () => {
	vi.unstubAllGlobals();
	vi.stubGlobal('requestAnimationFrame', undefined);

	const zoom = createZoomEngine((state) => states.push({ ...state }));
	zoom.zoomIn();
	// No frames exist to ride: the zoom settles on the lattice point
	// synchronously so the state machine never wedges mid-flight.
	expect(states.at(-1)).toEqual({ scale: 1.5, target: 1.5 });
});
