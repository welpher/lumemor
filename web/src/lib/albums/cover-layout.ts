// The cover collages' degradation ladder: how many frames a card has
// and where they sit. A cover is 4 or 6 frames, but a collection can
// hold fewer than the card's count -- the card then lays out the
// frames it has (the design's single / split / duo-plus-one / quad
// grammar), never a placeholder photo and never a letterboxed strip.
//
// Two shapes, two ladders:
//
// - The album card's 4:3 square: one frame fills it, two split it
//   left/right, three lay one wide frame under a pair, four make the
//   2x2 the card is designed for.
// - The memory band's 3:1 strip: one fills, two split, three sit in
//   one row of three, four make a 2x2, five sit three-over-two on a
//   six-column grid, and six are the 3-across 2-down band.
//
// The classes are Tailwind utilities (the geometry half of the
// two-class discipline); the card's semantic class stays on the
// element.

/** The album card's collage grid class for a frame count (1 never grids). */
export function albumCoverGrid(count: number): string {
	return count >= 2 ? 'grid grid-cols-2' : 'grid grid-cols-1';
}

/** The album card's per-cell class for one frame's position. */
export function albumCoverCell(count: number, index: number): string {
	// The three-frame fall: the third frame spans both columns under
	// the pair ("one large beside two small" read vertically).
	return count === 3 && index === 2 ? 'col-span-2' : '';
}

/** The memory band's collage grid class for a frame count. */
export function memoryCoverGrid(count: number): string {
	if (count <= 1) {
		return 'grid grid-cols-1';
	}
	if (count === 2) {
		return 'grid grid-cols-2';
	}
	if (count === 3) {
		return 'grid grid-cols-3';
	}
	if (count === 4) {
		return 'grid grid-cols-2';
	}
	if (count === 5) {
		return 'grid grid-cols-6 grid-rows-2';
	}
	return 'grid grid-cols-3 grid-rows-2';
}

/** The memory band's per-cell class for one frame's position. */
export function memoryCoverCell(count: number, index: number): string {
	if (count !== 5) {
		return '';
	}
	// Five on the six-column grid: the first row takes three frames of
	// two columns each, the second row the remaining two of three --
	// every cell keeps close to the band's own 2:1 rhythm.
	return index < 3 ? 'col-span-2' : 'col-span-3';
}
