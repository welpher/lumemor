import { describe, expect, test } from 'vitest';
import {
	aspectOf,
	packRows,
	packWall,
	wallRowTarget,
	WALL_FALLBACK_ASPECT,
	WALL_FALLBACK_WIDTH,
	WALL_ROW_FACTOR,
	WALL_ROW_MAX,
	WALL_ROW_MIN,
	WALL_ROW_TARGET,
	WALL_ROW_TOLERANCE,
	WALL_TILE_SEAM
} from './layout';
import type { MonthGroup } from './grouping';
import type { TimelineItem } from '$lib/api/timeline';

// The justified row layout engine's contract: exact row math (the
// widths plus the seams fill the container), the tolerance floor
// that closes rows, the clamped tail, group boundaries, determinism,
// the resize re-pack, the proportional scale (the target rides the
// container width between its floor and ceiling) and the degenerate
// guards. The numbers here are the engine's own pinned arithmetic --
// the same inputs the wall derives in a layout-less host (the 1352px
// reference content width fallback).

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
	return row.tiles.reduce((total, tile) => total + tile.width, 0) + WALL_TILE_SEAM * (row.tiles.length - 1);
}

const WIDTH = WALL_FALLBACK_WIDTH;

/** The floor under the target at a width (the engine's closing rule). */
function floorAt(width: number): number {
	return wallRowTarget(width) * (1 - WALL_ROW_TOLERANCE);
}

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
		// Eight 3:2 landscape tiles at the reference width: the floor
		// closes the row after the fifth (the sixth would drop it below
		// 216 * 0.75 = 162px), and the leftover three are the tail.
		const rows = packRows(
			Array.from({ length: 8 }, (_, i) => itemAt(`t${i}`, 1.5)),
			WIDTH
		);
		expect(rows.map((row) => row.tiles.length)).toEqual([5, 3]);
		const closed = rows[0]!;
		// The exact-fill invariant: widths + seams = container.
		expect(rowExtent(closed)).toBeCloseTo(WIDTH, 9);
		// Every tile renders at its true aspect.
		for (const tile of closed.tiles) {
			expect(tile.width / closed.height).toBeCloseTo(1.5, 9);
		}
		// The closed height sits in the accepted band.
		expect(closed.height).toBeGreaterThanOrEqual(162);
		expect(closed.height).toBeCloseTo(1344 / 7.5, 9); // (1352 - 4 seams of 2px) / (5 * 1.5)
		// The tail clamps back to the target with its natural widths.
		const tail = rows[1]!;
		expect(tail.height).toBe(WALL_ROW_TARGET);
		expect(rowExtent(tail)).toBeLessThan(WIDTH);
	});

	test('the floor closes the row: the over-densifying item opens the next one', () => {
		// Squares at the reference width: the floor is 162px, so the
		// eight-tile row ((1352 - 14) / 8 = 167.25) is legal and the
		// ninth would drop to 148.4 -- it opens the next row instead.
		const rows = packRows(
			Array.from({ length: 12 }, (_, i) => itemAt(`t${i}`, 1)),
			WIDTH
		);
		expect(rows.map((row) => row.tiles.length)).toEqual([8, 4]);
		// The closed row's height is the exact fill of its eight tiles.
		expect(rows[0]?.height).toBeCloseTo((1352 - 7 * WALL_TILE_SEAM) / 8, 9);
		// The leftover four are the group's last row: the tail renders
		// at the target height, natural width.
		expect(rows[1]?.height).toBe(WALL_ROW_TARGET);
		expect(rows[1]?.tiles[0]?.width).toBe(WALL_ROW_TARGET);
	});

	test('a sparse tail sits at the target height with its natural width', () => {
		// One panorama alone: natural height 1352/4 = 338 -- above the
		// 216px target, and the tail SITS at the target: ordinary
		// height, natural width (the day block flows beside its
		// neighbors; the tail row's natural width).
		const rows = packRows([itemAt('pano', 4)], WIDTH);
		expect(rows).toHaveLength(1);
		const row = rows[0];
		expect(row?.height).toBe(WALL_ROW_TARGET);
		expect(row?.tiles[0]?.width).toBe(WALL_ROW_TARGET * 4);
		expect(rowExtent(row as never)).toBeLessThan(WIDTH);
	});

	test('a lone item is a tail at the target height with its natural width', () => {
		// A lone portrait: the tail renders at the target height (never
		// grown, never filling the page) with its natural width.
		const rows = packRows([itemAt('lone', 0.75)], WIDTH);
		expect(rows).toHaveLength(1);
		const row = rows[0];
		expect(row?.height).toBe(WALL_ROW_TARGET);
		expect(row?.tiles[0]?.width).toBe(WALL_ROW_TARGET * 0.75);
	});

	test('a dense tail row keeps its natural exact fill', () => {
		// Five 3:2 tiles: the row closes only as the group's tail with
		// natural height 179.2 -- below the target, no clamp.
		const rows = packRows(
			Array.from({ length: 5 }, (_, i) => itemAt(`t${i}`, 1.5)),
			WIDTH
		);
		expect(rows).toHaveLength(1);
		expect(rows[0]?.height).toBeCloseTo(1344 / 7.5, 9);
		expect(rowExtent(rows[0] as never)).toBeCloseTo(WIDTH, 9);
	});

	test('unrefined dimensions pack at the fallback aspect', () => {
		const notReady = { ...itemDefaults, id: 'fresh', width: 0, height: 0 } as TimelineItem;
		const rows = packRows([notReady], WIDTH);
		// A lone fallback-aspect item is a tail at the target height.
		expect(rows[0]?.height).toBe(WALL_ROW_TARGET);
		expect(rows[0]?.tiles[0]?.width).toBe(WALL_ROW_TARGET * WALL_FALLBACK_ASPECT);
	});

	test('an empty input answers no rows', () => {
		expect(packRows([], WIDTH)).toEqual([]);
	});

	test('a non-positive container width degrades to single-tile rows at the floor', () => {
		const rows = packRows([itemAt('a', 2), itemAt('b', 0.75)], 0);
		expect(rows).toHaveLength(2);
		for (const row of rows) {
			expect(row.tiles).toHaveLength(1);
			expect(row.height).toBe(WALL_ROW_MIN);
		}
		expect(rows[0]?.tiles[0]?.width).toBe(WALL_ROW_MIN * 2);
	});
});

describe('packRows: determinism and resize', () => {
	const items = Array.from({ length: 19 }, (_, i) => itemAt(`t${i}`, [1.5, 0.75, 1, 2.4][i % 4] as number));

	test('the pack is a pure function: the same inputs answer the same rows', () => {
		expect(packRows(items, WIDTH)).toEqual(packRows(items, WIDTH));
	});

	test('a narrower container re-packs into more rows with the same invariants', () => {
		const wide = packRows(items, WIDTH);
		const narrow = packRows(items, 480);
		expect(narrow.length).toBeGreaterThan(wide.length);
		for (const row of narrow) {
			// A row either fills the container exactly or is a tail
			// sitting at the target height (natural widths); either
			// way the aspects hold.
			if (Math.abs(rowExtent(row) - 480) > 1) {
				expect(row.height).toBe(wallRowTarget(480));
			}
			for (const tile of row.tiles) {
				expect(tile.width / row.height).toBeCloseTo(tile.item.width / tile.item.height, 7);
			}
		}
	});

	test('a wider container packs the same items into fewer or equal rows', () => {
		const narrow = packRows(items, 480);
		const wide = packRows(items, WIDTH);
		expect(wide.length).toBeLessThanOrEqual(narrow.length);
	});
});

describe('wallRowTarget: the proportional scale', () => {
	test('the target is a fraction of the width, clamped at both ends', () => {
		// The reference width keeps the design's rows exactly.
		expect(wallRowTarget(WALL_FALLBACK_WIDTH)).toBe(WALL_ROW_TARGET);
		expect(WALL_FALLBACK_WIDTH * WALL_ROW_FACTOR).toBeCloseTo(WALL_ROW_TARGET, 0);
		// A narrow container floors at the readability guard.
		expect(wallRowTarget(240)).toBe(WALL_ROW_MIN);
		expect(wallRowTarget(600)).toBe(WALL_ROW_MIN);
		expect(wallRowTarget(1000)).toBe(WALL_ROW_MIN);
		// The middle rides the fraction.
		expect(wallRowTarget(1100)).toBe(176);
		expect(wallRowTarget(1840)).toBe(294);
		// An ultra-wide container caps.
		expect(wallRowTarget(2000)).toBe(WALL_ROW_MAX);
		expect(wallRowTarget(2480)).toBe(WALL_ROW_MAX);
		// Degenerate asks (a layout-less host) answer the floor.
		expect(wallRowTarget(0)).toBe(WALL_ROW_MIN);
		expect(wallRowTarget(-40)).toBe(WALL_ROW_MIN);
		expect(wallRowTarget(Number.NaN)).toBe(WALL_ROW_MIN);
	});

	test('the target never decreases as the container widens', () => {
		let previous = 0;
		for (let width = 1; width <= 3000; width += 7) {
			const target = wallRowTarget(width);
			expect(target).toBeGreaterThanOrEqual(previous);
			expect(target).toBeGreaterThanOrEqual(WALL_ROW_MIN);
			expect(target).toBeLessThanOrEqual(WALL_ROW_MAX);
			previous = target;
		}
	});

	test('the same photo scales with the width squared, the count per row holding', () => {
		// The two viewport widths this check names: 1024 and 1920,
		// each minus the wall's edges and slot (the measured content
		// widths a page wall reports).
		const items = Array.from({ length: 20 }, (_, i) => itemAt(`t${i}`, 1.5));
		const narrow = packRows(items, 944);
		const wide = packRows(items, 1840);
		const narrowRow = narrow[0]!;
		const wideRow = wide[0]!;
		// The count per row is the scale's promise: the same tiles per
		// line at both widths.
		expect(narrowRow.tiles.length).toBe(wideRow.tiles.length);
		// The same photo's area grows with the width squared (within
		// the pixel rounding of the target's integer height).
		const narrowTile = narrowRow.tiles[0]!;
		const wideTile = wideRow.tiles[0]!;
		const areaRatio = (wideTile.width * wideRow.height) / (narrowTile.width * narrowRow.height);
		const widthRatio = 1840 / 944;
		expect(areaRatio / (widthRatio * widthRatio)).toBeCloseTo(1, 1);
	});

	test('the capped ultra-wide rows stop growing', () => {
		const items = Array.from({ length: 24 }, (_, i) => itemAt(`t${i}`, 1.5));
		const capped = packRows(items, 2480);
		const cappedFloor = floorAt(2480);
		// Every closed row sits above the capped band's floor, never at
		// an unbounded height.
		for (const row of capped.slice(0, -1)) {
			expect(row.height).toBeGreaterThanOrEqual(cappedFloor);
			expect(row.height).toBeLessThan(WALL_ROW_MAX * 1.5);
		}
	});
});

describe('packRows: the narrow frame (the map panel)', () => {
	// The panel is the narrow-container case: the same engine, a
	// smaller container. Its band floor rides the scale down (so the
	// panel's photos shrink with it -- that is the point), while the
	// floor still refuses the ribbon: a line that would squash its
	// photos into a strip closes instead. When a line is stretched,
	// the row grows TALLER rather than flattening the tiles.
	const PANEL_WIDTH = 420;
	const threeLandscape = [itemAt('p1', 1.5), itemAt('p2', 1.5), itemAt('p3', 1.5)];

	test('a three-photo line at panel width closes instead of ribboning', () => {
		const rows = packRows(threeLandscape, PANEL_WIDTH);
		// The naive one-row pack would be (420 - 4) / 4.5 = 92px -- a
		// ribbon the floor refuses. Two share the first line at
		// (420 - 2) / 3 = 139.3 and the third takes its own.
		expect(rows.map((row) => row.tiles.length)).toEqual([2, 1]);
		for (const row of rows) {
			expect(row.height).toBeGreaterThanOrEqual(floorAt(PANEL_WIDTH));
			// Every tile keeps its true aspect (width = height * aspect).
			for (const tile of row.tiles) {
				expect(tile.width / row.height).toBeCloseTo(tile.item.width / tile.item.height, 7);
			}
		}
	});

	test('a stretched line grows tall rather than flattening its photos', () => {
		// One landscape photo alone on a line: its natural height
		// (420 / 1.5 = 280) would tower over the panel's target, so the
		// tail clamps to the target -- tall, never a short wide strip.
		const [row] = packRows([itemAt('p1', 1.5)], PANEL_WIDTH);
		expect(row?.height).toBe(wallRowTarget(PANEL_WIDTH));
		expect(row?.tiles[0]?.width).toBe(wallRowTarget(PANEL_WIDTH) * 1.5);
		// Two photos that DO share a justified line keep their aspects
		// at the taller of the two container widths (the row grows with
		// the width, it never squashes the tiles).
		const wide = packRows(threeLandscape, 900);
		const shared = wide.find((candidate) => candidate.tiles.length > 1);
		expect(shared).toBeDefined();
		for (const tile of shared?.tiles ?? []) {
			expect(tile.width / shared!.height).toBeCloseTo(tile.item.width / tile.item.height, 7);
		}
	});

	test('the panel scales below the page wall: the narrow container shrinks its photos', () => {
		const items = Array.from({ length: 12 }, (_, i) => itemAt(`t${i}`, 1.5));
		const panel = packRows(items, PANEL_WIDTH);
		const page = packRows(items, WALL_FALLBACK_WIDTH);
		// The same run, a narrower container: the smallest line is
		// smaller than the page's smallest (the scale reaches the panel).
		expect(Math.min(...panel.map((row) => row.height))).toBeLessThan(Math.min(...page.map((row) => row.height)));
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
		// 3:2 tiles each would fill the reference width together) --
		// they must not.
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
		// One month, two days, each a single tail row at the TARGET
		// height -- and each day BLOCK its natural width (two 3:2
		// tiles at 216px: 216*3 + 2 seam = 650px), the blocks flowing
		// side by side.
		expect(packed[0]?.days).toHaveLength(2);
		for (const day of packed[0]?.days ?? []) {
			expect(day.rows).toHaveLength(1);
			expect(day.rows[0]?.height).toBe(WALL_ROW_TARGET);
			expect(day.width).toBe(WALL_ROW_TARGET * 3 + WALL_TILE_SEAM);
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
