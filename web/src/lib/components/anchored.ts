// The anchored floating layer: the two dialog pickers -- the album path completion
// and the audience candidate list -- portal out of the Modal DOM to
// document.body as position:fixed layers anchored under their
// trigger, so the dialog NEVER changes height (the 2026-11-08
// inline-expansion design traded an offset shift for height churn:
// the modal grew and shrank every time a list appeared or folded --
// the wrong tool, retired) while the modal body's overflow can
// still never clip the layer (it lives outside every scroll
// container). No third-party popper: this action IS the whole
// machine -- portal, placement, flip, bounds and the reposition
// listeners -- in the repo's own voice.
//
// THE PLACEMENT CONTRACT:
// - The layer's left edge and width follow the anchor's rect (the
//   path input's full width; the audience chips row's width).
// - It places below the anchor by default and FLIPS UP when the
//   layer would run past the viewport's bottom edge and the anchor
//   leaves more room above than below.
// - Long lists scroll INTERNALLY: the layer's max-height is bounded
//   by the larger of the two sides' available space (hard-capped),
//   so the layer never overflows the viewport it floats in.
// - Every scroll (captured at the window: scrolls do not bubble,
//   but the modal body's own scrolling moves the anchor and must
//   re-place the layer) and every viewport resize re-places it.
//
// THE PORTAL: the action appends the node to document.body (out of
// the Modal's DOM, above the modal on the z ladder -- see
// --z-dropdown) and removes it on destroy. The layer's focus is
// deliberately OUTSIDE the Modal's focus trap; its
// own keydown handler owns its keys, and the trigger recovers the
// focus on close (the components' own discipline, unchanged).

import type { ActionReturn } from 'svelte/action';

export type AnchoredOptions = {
	/** The element the layer anchors under, read live at every placement. */
	anchor: () => HTMLElement | undefined;
	/** The gap between the anchor's edge and the layer (px). Default 8. */
	gap?: number;
};

/** The viewport breathing room the placement never eats into (px). */
const VIEWPORT_MARGIN = 8;
/** The hard cap of the layer's height; longer lists scroll internally. */
const HARD_MAX_HEIGHT = 320;
/** The smallest bound the max-height collapses to (a squeezed viewport). */
const MIN_MAX_HEIGHT = 120;

export function anchored(node: HTMLElement, options: AnchoredOptions): ActionReturn<AnchoredOptions> {
	const gap = options.gap ?? 8;

	function place(): void {
		const anchor = options.anchor();
		if (anchor === undefined) {
			return;
		}
		const rect = anchor.getBoundingClientRect();
		// The horizontal contract: the layer's left edge and width are
		// the anchor's own.
		node.style.left = `${rect.left}px`;
		node.style.width = `${rect.width}px`;
		// The bounds: the layer's max-height fits the roomier side,
		// hard-capped, so a long list scrolls inside the layer
		// instead of running off the viewport.
		const spaceBelow = window.innerHeight - rect.bottom - gap - VIEWPORT_MARGIN;
		const spaceAbove = rect.top - gap - VIEWPORT_MARGIN;
		node.style.maxHeight = `${Math.max(MIN_MAX_HEIGHT, Math.min(HARD_MAX_HEIGHT, Math.max(spaceBelow, spaceAbove)))}px`;
		// The placement: below by default; flip up only when the
		// layer would overflow the bottom AND above is the roomier
		// side (the measured height already respects the bounds).
		const height = node.offsetHeight;
		const below = rect.bottom + gap;
		if (below + height > window.innerHeight - VIEWPORT_MARGIN && spaceAbove > spaceBelow) {
			node.style.top = `${rect.top - gap - height}px`;
		} else {
			node.style.top = `${below}px`;
		}
	}

	// The portal: out of the Modal's DOM, onto the body.
	document.body.appendChild(node);
	// The class carries `position: fixed` and the ladder rung; the
	// geometry is all ours.
	place();
	// A second placement on the next frame: the layer's own content
	// (the rows) may settle after the first synchronous pass, and a
	// freshly-bound anchor is definitely resolved by then.
	const raf = requestAnimationFrame(place);

	// Reposition on every scroll and resize: the capture arm matters
	// most -- the modal body's own scrolling never bubbles, but its
	// capture phase passes the window on the way down.
	function onMove(): void {
		place();
	}
	window.addEventListener('scroll', onMove, { capture: true, passive: true });
	window.addEventListener('resize', onMove, { passive: true });

	return {
		destroy() {
			cancelAnimationFrame(raf);
			window.removeEventListener('scroll', onMove, { capture: true });
			window.removeEventListener('resize', onMove);
			node.remove();
		}
	};
}
