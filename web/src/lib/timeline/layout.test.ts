import { describe, expect, test } from 'vitest';
import {
	aspectOf,
	packRows,
	packWall,
	WALL_FALLBACK_ASPECT,
	WALL_MAX_WIDTH,
	WALL_ROW_TARGET,
	WALL_TILE_SEAM
} from './layout';
import type { MonthGroup } from './grouping';
import type { TimelineItem } from '$lib/api/timeline';

// The justified row layout engine's contract: exact row math (the
// widths plus the seams fill the container), the tolerance floor
// that closes rows, the clamped tail, group boundaries, determinism,
// the resize re-pack and the degenerate guards. The numbers here are
// the engine's own pinned arithmetic -- the same inputs the wall
// derives in a layout-less host (the 1200px content column fallback).

const itemDefaults = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image' as const,
	originalFilename: 'IMG_0001.jpg',
	fileSize: 1000,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d659a2feaa0c55ad015',
	createdAt: '2026-10-29T09:12:00Z',
	takenAt: '2026-10-13T12:30:00Z',
	takenAtSource: 'exif' as const,
	isTrashed: false,
	isFavorite: false,
	isReady: true,
	durationMs: null,
	blurhash: null,
	livePhotoVideoId: null
};

/** One item at an exact aspect (dimensions chosen to divide cleanly). */
function itemAt(id: string, aspect: number): TimelineItem {
	// Aspect as width/height with integers where possible; the engine
	// only ever reads the ratio.
	const height = 1000;
	return { id, ...itemDefaults, width: aspect * height, height } as TimelineItem;
}

/** The sum of a row's tile widths plus its seams. */
function rowExtent(row: { tiles: { width: number }[] }): number {
	return (
		row.tiles.reduce((total, tile) => total + tile.width, 0) +
		WALL_TILE_SEAM * (row.tiles.length - 1)
	);
}

const WIDTH = WALL_MAX_WIDTH;

describe('aspectOf', () => {
	test('the true sensor-order ratio, and the fallback while unrefined', () => {
		expect(aspectOf({ width: 4032, height: 3024 })).toBe(4032 / 3024);
		expect(aspectOf({ width: 1000, height: 2000 })).toBe(0.5);
		// The contract's 0 placeholder (a not-ready upload).
		expect(aspectOf({ width: 0, height: 0 })).toBe(WALL_FALLBACK_ASPECT);
		expect(aspectOf({ width: 4000, height: 0 })).toBe(WALL_FALLBACK_ASPECT);
		// A malformed wire body must not take the wall down.
		expect(aspectOf({ width: Number.NaN, height: 1000 })).toBe(WALL_FALLBACK_ASPECT);
		expect(aspectOf({ width: -3, height: 1000 })).toBe(WALL_FALLBACK_ASPECT);
	});
});

describe('packRows: row math', () => {
	test('a justified row fills the container exactly at true aspects', () => {
		// Eight 3:2 landscape tiles: the floor closes the row after the
		// fourth (the fifth would drop it below 216 * 0.75 = 162px).
		const rows = packRows(Array.from({ length: 8 }, (_, i) => itemAt(`t${i}`, 1.5)), WIDTH);
		expect(rows).toHaveLength(2);
		for (const row of rows) {
			expect(row.tiles).toHaveLength(4);
			// The exact-fill invariant: widths + seams = container.
			expect(rowExtent(row)).toBeCloseTo(WIDTH, 9);
			// Every tile renders at its true aspect.
			for (const tile of row.tiles) {
				expect(tile.width / row.height).toBeCloseTo(1.5, 9);
			}
			// The closed height sits in the accepted band.
			expect(row.height).toBeGreaterThanOrEqual(162);
			expect(row.height).toBeCloseTo(1194 / 6, 9); // (1200 - 3 seams of 2px) / (4 * 1.5)
		}
	});

	test('the floor closes the row: the over-densifying item opens the next one', () => {
		// Four squares close at (1200 - 6) / 4 = 298.5 -- above the
		// floor, so the row keeps accepting until a tile would drop it
		// below 162px: with squares that is the fifth tile
		// ((1200 - 4*2) / 5 = 238.4, still legal) ... the seventh is
		// ((1200 - 6*3) / 7 = 168.86, legal) and the eighth is
		// ((1200 - 7*3) / 8 = 147.4 -- below the floor).
		const rows = packRows(Array.from({ length: 8 }, (_, i) => itemAt(`t${i}`, 1)), WIDTH);
		expect(rows.map((row) => row.tiles.length)).toEqual([7, 1]);
		// The closed row's height is the exact fill of its seven tiles.
		expect(rows[0]?.height).toBeCloseTo((1200 - 6 * WALL_TILE_SEAM) / 7, 9);
		// The leftover single square is the group's last row: a lone
		// square's natural height (1194) is past the growth cap, so
		// the tail caps at 2x the target and renders left-aligned.
		expect(rows[1]?.height).toBe(WALL_ROW_TARGET * 2);
		expect(rows[1]?.tiles[0]?.width).toBe(WALL_ROW_TARGET * 2);
	});

	test('a sparse tail row grows to fill the container (the widow handling)', () => {
		// One panorama alone: natural height 1200/4 = 300 -- above the
		// 216px target, and the tail GROWS to it: the row fills the
		// width exactly (a lone photo is a big photo, never a shrunken
		// pair with a hole on the right).
		const rows = packRows([itemAt('pano', 4)], WIDTH);
		expect(rows).toHaveLength(1);
		const row = rows[0];
		expect(row?.height).toBe(300);
		expect(row?.tiles[0]?.width).toBe(1200);
		expect(rowExtent(row as never)).toBeCloseTo(WIDTH, 9);
	});

	test('an ultra-sparse tail caps at twice the target instead of filling the page', () => {
		// A lone portrait: natural height 1200 / 0.75 = 1600 -- the
		// page-filling extreme. The tail caps at 2x the target (432px)
		// and renders left-aligned: big, never a full page.
		const rows = packRows([itemAt('lone', 0.75)], WIDTH);
		expect(rows).toHaveLength(1);
		const row = rows[0];
		expect(row?.height).toBe(WALL_ROW_TARGET * 2);
		expect(row?.tiles[0]?.width).toBe(WALL_ROW_TARGET * 2 * 0.75);
		expect(rowExtent(row as never)).toBeLessThan(WIDTH);
	});

	test('a dense tail row keeps its natural exact fill', () => {
		// Four 3:2 tiles: the row closes only as the group's tail with
		// natural height 198.5 -- inside the band, no clamp.
		const rows = packRows(Array.from({ length: 4 }, (_, i) => itemAt(`t${i}`, 1.5)), WIDTH);
		expect(rows).toHaveLength(1);
		expect(rows[0]?.height).toBeCloseTo(1194 / 6, 9);
		expect(rowExtent(rows[0] as never)).toBeCloseTo(WIDTH, 9);
	});

	test('unrefined dimensions pack at the fallback aspect', () => {
		const notReady = { ...itemDefaults, id: 'fresh', width: 0, height: 0 } as TimelineItem;
		const rows = packRows([notReady], WIDTH);
		// A lone fallback-aspect item is an ultra-sparse tail: capped
		// at 2x the target, width = cap * 1.5.
		expect(rows[0]?.height).toBe(WALL_ROW_TARGET * 2);
		expect(rows[0]?.tiles[0]?.width).toBe(WALL_ROW_TARGET * 2 * WALL_FALLBACK_ASPECT);
	});

	test('an empty input answers no rows', () => {
		expect(packRows([], WIDTH)).toEqual([]);
	});

	test('a non-positive container width degrades to single-tile rows', () => {
		const rows = packRows([itemAt('a', 2), itemAt('b', 0.75)], 0);
		expect(rows).toHaveLength(2);
		for (const row of rows) {
			expect(row.tiles).toHaveLength(1);
			expect(row.height).toBe(WALL_ROW_TARGET);
		}
		expect(rows[0]?.tiles[0]?.width).toBe(WALL_ROW_TARGET * 2);
	});
});

describe('packRows: determinism and resize', () => {
	const items = Array.from({ length: 19 }, (_, i) =>
		itemAt(`t${i}`, [1.5, 0.75, 1, 2.4][i % 4] as number)
	);

	test('the pack is a pure function: the same inputs answer the same rows', () => {
		expect(packRows(items, WIDTH)).toEqual(packRows(items, WIDTH));
	});

	test('a narrower container re-packs into more rows with the same invariants', () => {
		const wide = packRows(items, WIDTH);
		const narrow = packRows(items, 480);
		expect(narrow.length).toBeGreaterThan(wide.length);
		for (const row of narrow) {
			// A row either fills the container exactly or is a capped
			// tail (its extent falls short and its height IS the cap);
			// either way the aspects hold.
			if (Math.abs(rowExtent(row) - 480) > 1) {
				expect(row.height).toBe(WALL_ROW_TARGET * 2);
			}
			for (const tile of row.tiles) {
				expect(tile.width / row.height).toBeCloseTo(
					tile.item.width / tile.item.height,
					7
				);
			}
		}
	});

	test('a wider container packs the same items into fewer or equal rows', () => {
		const narrow = packRows(items, 480);
		const wide = packRows(items, WIDTH);
		expect(wide.length).toBeLessThanOrEqual(narrow.length);
	});
});

describe('packWall: group boundaries', () => {
	function month(items: TimelineItem[], key = '2026-10'): MonthGroup {
		return {
			key,
			label: 'October 2026',
			days: [{ key: '2026-10-13', label: 'October 13', items }]
		};
	}

	test('rows never span day groups: each group packs independently', () => {
		// Two days whose items COULD share one justified row (two
		// 3:2 tiles each fill 1200px together) -- they must not.
		const dayA = [itemAt('a1', 1.5), itemAt('a2', 1.5)];
		const dayB = [itemAt('b1', 1.5), itemAt('b2', 1.5)];
		const groups: MonthGroup[] = [
			{
				key: '2026-10',
				label: 'October 2026',
				days: [
					{ key: '2026-10-01', label: 'October 1', items: dayA },
					{ key: '2026-10-02', label: 'October 2', items: dayB }
				]
			}
		];
		const packed = packWall(groups, WIDTH);
		expect(packed).toHaveLength(1);
		// One month, two days, each with its own tail row -- and each
		// tail FILLS the day's width at its natural height (two 3:2
		// tiles: (1200 - 2) / 3 = 399.33px tall).
		expect(packed[0]?.days).toHaveLength(2);
		for (const day of packed[0]?.days ?? []) {
			expect(day.rows).toHaveLength(1);
			expect(day.rows[0]?.height).toBeCloseTo(1198 / 3, 9);
		}
		// No row mixes the two groups' items.
		const rowIds = (packed[0]?.days ?? []).flatMap((day) =>
			day.rows.flatMap((row) => row.tiles.map((tile) => tile.item.id))
		);
		expect(rowIds).toEqual(['a1', 'a2', 'b1', 'b2']);
	});

	test('the month and day labels and keys survive the packing', () => {
		const packed = packWall([month([itemAt('a1', 1)])], WIDTH);
		expect(packed[0]?.key).toBe('2026-10');
		expect(packed[0]?.label).toBe('October 2026');
		expect(packed[0]?.days[0]?.key).toBe('2026-10-13');
		expect(packed[0]?.days[0]?.label).toBe('October 13');
	});
});
