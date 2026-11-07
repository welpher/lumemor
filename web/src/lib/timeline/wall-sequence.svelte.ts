// The wall snapshot: the single place the timeline wall publishes
// its loaded page items so the asset viewer can walk them. The wall
// page owns a fresh pagination client per mount (the client's
// design), so its store dies with the route -- the viewer therefore
// reads THIS module-level copy instead, which the wall keeps current
// while mounted and which outlives the route change into the
// viewer.
//
// THE PINNED SEQUENCE RULE: the viewer navigates the wall's current
// page items exactly as the wall rendered them (keyset order, newest
// first) and never fetches the timeline itself. A deep link to an
// asset outside the snapshot (or with no snapshot at all -- a fresh
// browser entry straight into /a/{id}) still loads that asset's
// detail; its arrow keys are inert and the filmstrip simply does not
// highlight it. When the user returns to the timeline the wall
// remounts, refetches, and republishes a fresh snapshot.
//
// The wall's publish during remount transiently writes [] (a fresh
// client starts idle) and then the fetched pages -- correct, because
// the viewer is not mounted at that moment.

import type { TimelineItem } from '$lib/api/timeline';

let items = $state<TimelineItem[]>([]);

/** The wall's current page items, newest first (the navigation sequence). */
export function wallItems(): readonly TimelineItem[] {
	return items;
}

/**
 * Replace the snapshot wholesale. The wall page calls this from an
 * effect over its client store; item objects are never mutated, so
 * the snapshot shares the wall's exact array contents.
 */
export function publishWallItems(next: readonly TimelineItem[]): void {
	items = [...next];
}

/** Drop the snapshot (tests reset between cases). */
export function clearWallItems(): void {
	items = [];
}
