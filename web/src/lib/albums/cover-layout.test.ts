import { expect, test } from 'vitest';
import { albumCoverCell, albumCoverGrid, memoryCoverCell, memoryCoverGrid } from './cover-layout';

// The cover collages' degradation ladder: the album card's 4:3
// square and the memory band's 3:1 strip lay out the frames the
// collection actually holds -- one fills, two split, three take the
// design's duo-plus-one fall, four the quad, and the band's five sit
// three-over-two. The tests pin the geometry, because a hole in the
// grid is exactly the bug this ladder prevents.

test('the album card capitalises 1/2/3/4 frames without a hole', () => {
	expect(albumCoverGrid(1)).toBe('grid grid-cols-1');
	expect(albumCoverGrid(2)).toBe('grid grid-cols-2');
	expect(albumCoverGrid(3)).toBe('grid grid-cols-2');
	expect(albumCoverGrid(4)).toBe('grid grid-cols-2');

	// Only the three-frame fall has a wide cell: the last frame spans
	// both columns under the pair.
	expect(albumCoverCell(3, 0)).toBe('');
	expect(albumCoverCell(3, 1)).toBe('');
	expect(albumCoverCell(3, 2)).toBe('col-span-2');
	expect(albumCoverCell(4, 2)).toBe('');
});

test('the memory band lays 1..6 frames without a hole', () => {
	expect(memoryCoverGrid(1)).toBe('grid grid-cols-1');
	expect(memoryCoverGrid(2)).toBe('grid grid-cols-2');
	expect(memoryCoverGrid(3)).toBe('grid grid-cols-3');
	expect(memoryCoverGrid(4)).toBe('grid grid-cols-2');
	// Five on the six-column rhythm: three of two columns, two of
	// three -- every cell spans the full width when summed.
	expect(memoryCoverGrid(5)).toBe('grid grid-cols-6 grid-rows-2');
	expect([0, 1, 2, 3, 4].map((index) => memoryCoverCell(5, index))).toEqual([
		'col-span-2',
		'col-span-2',
		'col-span-2',
		'col-span-3',
		'col-span-3'
	]);
	// The full band is the designed 3-across 2-down strip.
	expect(memoryCoverGrid(6)).toBe('grid grid-cols-3 grid-rows-2');
	expect([0, 1, 2, 3, 4, 5].map((index) => memoryCoverCell(6, index))).toEqual(['', '', '', '', '', '']);
});
