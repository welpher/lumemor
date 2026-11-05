// The viewer's gradual zoom engine (R2-7/8): every zoom change -- the
// action bar's step buttons, the +/- keys, the double-click binary --
// animates the media's scale through requestAnimationFrame
// interpolation, and the animation is INTERRUPTIBLE: an input that
// lands mid-flight takes over from the CURRENT interpolated scale,
// never from the target the running animation was heading to.
//
// THE LADDER: the zoom rests on a step lattice (1, 1.5, 2 -- the same
// 2x ceiling the pre-rework binary zoom had). A step input moves the
// TARGET one lattice step (clamped at both ends), so quick repeated
// inputs advance the destination one step at a time while each new
// animation starts from wherever the scale actually is. The scale and
// the target are equal whenever the engine rests.
//
// THE CLOCK: the start is stamped with performance.now() at the
// input's call time and progress is measured against the rAF
// callback's own timestamp (the same time origin in every browser),
// so the first delivered frame already carries eased progress. A
// host without requestAnimationFrame (a stripped-down test
// environment) lands the target instantly -- the animation is an
// affordance, never a state the caller must wait for.
//
// The engine owns no DOM and no Svelte state: every transition calls
// the caller's apply with the current {scale, target} pair, once
// synchronously at the input and once per delivered frame.

/** The resting scale (the photo fits the stage). */
export const VIEWER_ZOOM_BASE = 1;
/** The lattice ceiling -- the same max the old binary zoom had. */
export const VIEWER_ZOOM_MAX = 2;
/** One lattice step (1, 1.5, 2). */
export const VIEWER_ZOOM_STEP = 0.5;

/** The interpolation length: the ~300ms the round-2 feedback pins. */
const VIEWER_ZOOM_DURATION_MS = 300;

/** The state the caller renders: the interpolated scale and its target. */
export type ZoomState = {
	/** The current scale (between lattice points while an animation runs). */
	scale: number;
	/** The lattice point the zoom is heading to (and rests on). */
	target: number;
};

export type ZoomEngine = {
	/** One step toward the ceiling; inert (and quiet) when already there. */
	zoomIn(): void;
	/** One step toward the base; inert (and quiet) when already there. */
	zoomOut(): void;
	/** The double-click binary: the base when zoomed, the ceiling when at rest. */
	toggleZoom(): void;
	/** Animate back to the base (the bar's reset affordance). */
	reset(): void;
	/** Cancel any animation and land on the base instantly (the per-asset reset). */
	snapToBase(): void;
	/** Cancel the animation loop (the component is going away). */
	destroy(): void;
};

/** cubicOut: the fast start, gentle landing the ease family pins. */
function cubicOut(t: number): number {
	const inv = 1 - t;
	return 1 - inv * inv * inv;
}

/**
 * Create one engine. `apply` receives every state transition; the
 * engine never touches the DOM itself.
 */
export function createZoomEngine(apply: (state: ZoomState) => void): ZoomEngine {
	let scale = VIEWER_ZOOM_BASE;
	let target = VIEWER_ZOOM_BASE;
	let startScale = VIEWER_ZOOM_BASE;
	let startTime = 0;
	let frame = 0;
	let running = false;

	function emit(): void {
		apply({ scale, target });
	}

	function onFrame(now: number): void {
		const elapsed = Math.max(0, now - startTime);
		const progress = Math.min(1, elapsed / VIEWER_ZOOM_DURATION_MS);
		scale = startScale + (target - startScale) * cubicOut(progress);
		if (progress >= 1) {
			// Landed: the scale sits exactly on the lattice point and
			// the loop ends.
			scale = target;
			running = false;
			frame = 0;
		} else if (typeof requestAnimationFrame === 'function') {
			frame = requestAnimationFrame(onFrame);
		} else {
			running = false;
			frame = 0;
		}
		emit();
	}

	function animateTo(next: number): void {
		if (next === target) {
			// An input that keeps the destination (a clamp end, a
			// re-press of the same intent) is inert: a running
			// animation simply continues to the target it already has.
			return;
		}
		// THE INTERRUPTION RULE: the new animation starts from the
		// CURRENT interpolated scale -- the value on screen -- so a
		// mid-flight input takes over seamlessly instead of snapping
		// back to the old target's geometry.
		startScale = scale;
		startTime = performance.now();
		target = next;
		// A mid-flight input must leave EXACTLY ONE frame chain: the
		// pending callback is cancelled before the new one is
		// scheduled, or the old id is overwritten and never
		// cancelled -- a duplicate chain that destroy() cannot fully
		// stop. (stop() is a no-op when nothing is scheduled.)
		stop();
		running = typeof requestAnimationFrame === 'function';
		// The target applies synchronously (the affordances key on
		// it); the scale follows through the frames.
		emit();
		if (!running) {
			scale = target;
			emit();
			return;
		}
		frame = requestAnimationFrame(onFrame);
	}

	function stop(): void {
		if (frame !== 0 && typeof cancelAnimationFrame === 'function') {
			cancelAnimationFrame(frame);
		}
		frame = 0;
		running = false;
	}

	function snapToBase(): void {
		stop();
		scale = VIEWER_ZOOM_BASE;
		target = VIEWER_ZOOM_BASE;
		startScale = VIEWER_ZOOM_BASE;
		emit();
	}

	return {
		zoomIn(): void {
			animateTo(Math.min(VIEWER_ZOOM_MAX, target + VIEWER_ZOOM_STEP));
		},
		zoomOut(): void {
			animateTo(Math.max(VIEWER_ZOOM_BASE, target - VIEWER_ZOOM_STEP));
		},
		toggleZoom(): void {
			animateTo(target > VIEWER_ZOOM_BASE ? VIEWER_ZOOM_BASE : VIEWER_ZOOM_MAX);
		},
		reset(): void {
			animateTo(VIEWER_ZOOM_BASE);
		},
		snapToBase,
		destroy(): void {
			stop();
		}
	};
}
