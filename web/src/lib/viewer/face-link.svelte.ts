// The two-way face linkage of the viewer: the on-photo face-box
// layer and the panel's face chips are two ends of ONE state -- a
// hover or a keyboard focus on either end highlights the other
// (the box turns accent, the chip turns accent border and soft
// background), and the linkage hides with a ~220ms grace so the
// pointer can travel from the chip to its box without the highlight
// dying on the way.
//
// The state is a module singleton on purpose: one viewer is mounted
// at a time (a full-screen route), and the two components that ride
// it (the stage's box layer, the panel's chips) must read the same
// signal without prop plumbing through the composition.

const GRACE_MS = 220;

/** The currently linked face id (null when the pointer rests). */
let linked = $state<string | null>(null);

/** The pending hide (the grace window); a new set cancels it. */
let hideTimer: ReturnType<typeof setTimeout> | undefined;

/** Link one face now (a hover or a focus landed on either end). */
export function linkFace(id: string): void {
	if (hideTimer !== undefined) {
		clearTimeout(hideTimer);
		hideTimer = undefined;
	}
	linked = id;
}

/** Unlink after the grace (a leave on either end; a re-enter cancels). */
export function unlinkFace(id: string): void {
	if (linked !== id) {
		return;
	}
	if (hideTimer !== undefined) {
		clearTimeout(hideTimer);
	}
	hideTimer = setTimeout(() => {
		hideTimer = undefined;
		linked = null;
	}, GRACE_MS);
}

/** Whether the given face is the linked one (the highlight's gate). */
export function isLinked(id: string): boolean {
	return linked === id;
}
