// The justified row layout of the photo walls (the web polish batch
// one engine): a pure, deterministic packer that folds one group's
// items into rows of tiles rendered at their TRUE aspect ratios, so
// every row fills the container width exactly (the classic justified
// gallery) and the tile heights ride a target band instead of the
// uniform square grid.
//
// THE MATH (greedy row packing, zero dependencies):
//
// - One row accumulates items while its natural height -- the usable
//   container width (container minus the seams between tiles) over
//   the row's summed aspect ratios -- stays above the band's floor.
//   The first item that would drop the row below the floor starts
//   the NEXT row: the closed row's height is then inside or above
//   the band, its tiles render at height * aspect each, and the
//   widths plus the seams fill the container exactly (sub-pixel
//   floats are kept, never rounded -- the browser renders them).
// - A row closed by the floor rule is JUSTIFIED: it fills the width.
//   The one exception is a group's LAST row: its leftover items have
//   nothing to densify them with, so a natural height above the
//   target would render them huge. The tail row clamps to the target
//   height instead -- true aspects kept, the row left-aligned and
//   narrower than the container. That is the one honest trade the
//   classic justified wall makes for its tail.
// - Rows never span month or day group boundaries: `packWall` packs
//   every day group independently, so a group's tail row clamps (or
//   fills) on its own terms and no row mixes two groups' items.
// - Determinism: the pack is a pure function of (items, width) --
//   no dates, no randomness, no hidden state -- so pages appending
//   (keyset), events removing (SSE convergence) and the container
//   resizing all re-derive the same rows for the same inputs.
//
// THE TOKEN MIRROR: the three row-math constants below mirror the
// wall tokens of theme.css ("The justified photo wall" section) --
// the same discipline as the @theme/Layer 2 mirrors, because the row
// math runs in TypeScript where CSS custom properties cannot be
// read. Keep the pairs in sync.

import type { MonthGroup } from './grouping';
import type { TimelineItem } from '$lib/api/timeline';

/** Mirrors `--wall-row-target` (216px) of theme.css. */
export const WALL_ROW_TARGET = 216;

/** Mirrors `--wall-row-tolerance` (0.25) of theme.css. */
export const WALL_ROW_TOLERANCE = 0.25;

/** Mirrors `--wall-tile-seam` (2px) of theme.css. */
export const WALL_TILE_SEAM = 2;

/** Mirrors `--wall-fallback-aspect` (1.5) of theme.css. */
export const WALL_FALLBACK_ASPECT = 1.5;

/**
 * The width the wall packs against while no measured container
 * exists: the pages' `max-w-[1200px]` content column. jsdom (and
 * any layout-less host) has neither a ResizeObserver nor box
 * metrics, so this stands in and the rows still pack
 * deterministically -- the engine's unit tests pin the same numbers.
 */
export const WALL_MAX_WIDTH = 1200;

/** One laid-out tile: the item plus its computed pixel width. */
export type PackedTile = {
	item: TimelineItem;
	/** The tile's rendered width in CSS pixels (height is the row's). */
	width: number;
};

/** One packed row of tiles. */
export type PackedRow = {
	tiles: PackedTile[];
	/** The row's rendered height in CSS pixels. */
	height: number;
};

/**
 * The aspect ratio (width / height) an item renders at: its true
 * sensor-order dimensions, or the fallback aspect while the pipeline
 * has not refined them (the contract's 0 placeholder, or any
 * non-positive/non-finite value a malformed wire body carried).
 */
export function aspectOf(item: Pick<TimelineItem, 'width' | 'height'>): number {
	const { width, height } = item;
	if (
		typeof width !== 'number' ||
		typeof height !== 'number' ||
		!Number.isFinite(width) ||
		!Number.isFinite(height) ||
		width <= 0 ||
		height <= 0
	) {
		return WALL_FALLBACK_ASPECT;
	}
	return width / height;
}

/** The height band's floor: rows never close below it. */
const bandFloor = WALL_ROW_TARGET * (1 - WALL_ROW_TOLERANCE);

/** The natural height of a row of `count` tiles summing `aspectSum`. */
function naturalHeight(count: number, aspectSum: number, containerWidth: number): number {
	const usable = containerWidth - WALL_TILE_SEAM * (count - 1);
	return usable / aspectSum;
}

/** Close one row: tiles at height * aspect. A CLOSED row fills the
 * container width exactly (the justified invariant); a group's TAIL
 * row renders at the TARGET height with its natural widths --
 * never stretched, never grown (the reference's last-row behavior:
 * a day with two photos shows two ordinary-sized ones, and the day
 * BLOCK flows beside its neighbors). */
function closeRow(
	items: readonly TimelineItem[],
	aspectSum: number,
	containerWidth: number,
	isLast: boolean
): PackedRow {
	const natural = naturalHeight(items.length, aspectSum, containerWidth);
	const height = isLast ? Math.min(natural, WALL_ROW_TARGET) : natural;
	return {
		tiles: items.map((item) => ({ item, width: height * aspectOf(item) })),
		height
	};
}

/**
 * Pack one flat run of items into justified rows. `containerWidth`
 * must be positive; a non-positive value (a layout-less host asking
 * before any measurement) answers one single-tile row per item at
 * the target height -- deterministic and safe to render.
 */
export function packRows(items: readonly TimelineItem[], containerWidth: number): PackedRow[] {
	if (items.length === 0) {
		return [];
	}
	if (containerWidth <= WALL_TILE_SEAM) {
		// Nothing to justify against: every item stands alone at the
		// target height.
		return items.map((item) => ({
			tiles: [{ item, width: WALL_ROW_TARGET * aspectOf(item) }],
			height: WALL_ROW_TARGET
		}));
	}

	const rows: PackedRow[] = [];
	let row: TimelineItem[] = [];
	let aspectSum = 0;

	for (const item of items) {
		const aspect = aspectOf(item);
		const nextSum = aspectSum + aspect;
		const nextHeight = naturalHeight(row.length + 1, nextSum, containerWidth);
		if (row.length > 0 && nextHeight < bandFloor) {
			// The item would over-densify the row: close the row
			// without it and let the item open the next one.
			rows.push(closeRow(row, aspectSum, containerWidth, false));
			row = [item];
			aspectSum = aspect;
			continue;
		}
		row.push(item);
		aspectSum = nextSum;
	}
	if (row.length > 0) {
		rows.push(closeRow(row, aspectSum, containerWidth, true));
	}
	return rows;
}

/** One day group carrying its packed rows. */
export type PackedDay = {
	key: string;
	label: string;
	rows: PackedRow[];
	/**
	 * The day BLOCK's rendered width: a day whose rows all closed
	 * spans the container (its rows filled it); a day that never
	 * closed answers its single tail row's natural extent. Day
	 * blocks flow horizontally and wrap -- the wall reads as a
	 * gallery of days, sparse dates side by side instead of a hole
	 * on the right (the reference timeline's shape).
	 */
	width: number;
};

/** One month group carrying its packed day groups. */
export type PackedMonth = {
	key: string;
	label: string;
	days: PackedDay[];
};

/**
 * Pack the wall's month/day groups: every DAY group packs
 * independently, so no row ever spans a group boundary (the pinned
 * rule -- the month gap plus the headers carry the grouping, never a
 * mixed row).
 */
export function packWall(groups: readonly MonthGroup[], containerWidth: number): PackedMonth[] {
	return groups.map((group) => ({
		key: group.key,
		label: group.label,
		days: group.days.map((day) => {
			const rows = packRows(day.items, containerWidth);
			const width =
				rows.length > 1
					? containerWidth
					: (rows[0]?.tiles.reduce((sum, tile) => sum + tile.width, 0) ?? 0) +
						Math.max(0, (rows[0]?.tiles.length ?? 1) - 1) * WALL_TILE_SEAM;
			return { key: day.key, label: day.label, rows, width };
		})
	}));
}
