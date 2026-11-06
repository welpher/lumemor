<script lang="ts">
	// The flat justified wall (M3d Task 4, the search's query mode):
	// one ungrouped run of items packed by the SAME row layout engine
	// the month-grouped wall packs with (packRows at the measured
	// container width), rendered in the page's own order -- the rank
	// order of a text query's merged results. NO month or day
	// grouping here: the grouping is a time statement, and a rank
	// list is not a time statement -- grouping would lie about the
	// order the server merged the two paths into.
	//
	// The rows obey the engine's pinned rules: a closed row fills
	// the container width exactly at true tile aspects with the tile
	// seam between them; the run's tail row clamps to the target
	// height left-aligned. Pages appending re-derive the pack (the
	// engine's determinism: same items + same width, same rows), so
	// a tail row that clamped mid-walk re-justifies as the next page
	// arrives, and the final tail (the exhausted walk's own end) is
	// the honest one.
	//
	// PERFORMANCE: every row carries content-visibility with its
	// height known from the engine's math (the same row-level
	// windowing the month-grouped wall applies).
	import { packRows, WALL_MAX_WIDTH, WALL_TILE_SEAM, type PackedRow } from './layout';
	import PhotoTile from './PhotoTile.svelte';
	import type { TimelineItem } from '$lib/api/timeline';

	type Props = {
		/** The result items, in the page's own order (rank order). */
		items: readonly TimelineItem[];
		/** Whether one item is in the current selection. */
		isSelected: (id: string) => boolean;
		/** Whether checkboxes show and tile clicks toggle. */
		selectable: boolean;
		/** Toggle callback; the checkbox and the tile body both call it. */
		onselect: (id: string, next: boolean) => void;
	};

	let { items, isSelected, selectable, onselect }: Props = $props();

	let wall: HTMLElement | undefined = $state();
	let measuredWidth = $state(0);

	// The measured container width (a ResizeObserver tracks the
	// content box). jsdom has no observer and no layout: the fallback
	// below stands in and the pack stays deterministic.
	const observerAvailable = typeof ResizeObserver === 'function';

	$effect(() => {
		const element = wall;
		if (element === undefined) {
			return;
		}
		if (element.clientWidth > 0) {
			measuredWidth = element.clientWidth;
		}
		if (!observerAvailable) {
			return;
		}
		const observer = new ResizeObserver((entries) => {
			for (const entry of entries) {
				const inline = entry.contentBoxSize[0]?.inlineSize;
				const width = typeof inline === 'number' ? inline : entry.contentRect.width;
				if (typeof width === 'number' && width > 0) {
					measuredWidth = width;
				}
			}
		});
		observer.observe(element);
		return () => {
			observer.disconnect();
		};
	});

	const containerWidth = $derived(measuredWidth > 0 ? measuredWidth : WALL_MAX_WIDTH);
	const rows: PackedRow[] = $derived(packRows(items, containerWidth));
</script>

<div class="flat-wall" bind:this={wall}>
	{#each rows as row, rowIndex (rowIndex)}
		<!-- The row's height is the engine's exact math; the seam
		     rides the row's gap so the widths plus the gaps fill the
		     container. Offscreen rows skip their tiles' layout and
		     paint (the height is style-pinned, so the scrollbar
		     stays honest). -->
		<div
			class="flat-wall__row flex [content-visibility:auto]"
			style="height:{row.height}px; gap:{WALL_TILE_SEAM}px"
		>
			{#each row.tiles as tile (tile.item.id)}
				<div class="flat-wall__cell shrink-0" style="width:{tile.width}px">
					<PhotoTile
						item={tile.item}
						selected={isSelected(tile.item.id)}
						selectable={selectable}
						sizes="{tile.width}px"
						onselect={onselect}
					/>
				</div>
			{/each}
		</div>
	{/each}
</div>
