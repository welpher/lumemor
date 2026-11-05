<script lang="ts">
	// The month-grouped justified wall (the web polish batch one
	// rework): the grouping of the loaded items, the sticky month and
	// day headers, and the row layout engine's output, assembled once
	// for every wall -- the timeline page (its scrubber and scroll
	// tracking stay its own) and the curated walls (favorites, hidden,
	// the album detail's grid) render exactly this tree so the wall's
	// density language is one component, not four copies.
	//
	// THE ROWS: each day group packs independently (rows never span a
	// group boundary -- the pinned rule), rows fill the container
	// width at true tile aspects with the tile seam between them, and
	// a group's tail row clamps to the target height left-aligned.
	// The pack re-derives whenever the container resizes (a
	// ResizeObserver feeds the measured width; a layout-less host
	// falls back to the content column maximum so the wall still
	// renders deterministically).
	//
	// THE HEADERS: opaque page-colored bands with a hairline edge --
	// the restrained surface language (no translucency, no blur): the
	// tiles scroll under them and the 1px border separates chrome
	// from content. The month gap token above each group is the
	// breathing room the density contrast reads on.
	//
	// PERFORMANCE: every row carries content-visibility with its
	// height known from the engine's math, so offscreen rows skip
	// their tiles' layout and paint while the scrollbar stays honest
	// (the row-level windowing the justified math makes exact); the
	// tiles keep their own content-visibility inside a partially
	// visible row.
	import { packWall, WALL_MAX_WIDTH, WALL_TILE_SEAM, type PackedMonth } from './layout';
	import type { MonthGroup } from './grouping';
	import PhotoTile from './PhotoTile.svelte';

	type Props = {
		/** The grouped items (see groupTimelineItems). */
		groups: readonly MonthGroup[];
		/** Whether one item is in the current selection. */
		isSelected: (id: string) => boolean;
		/** Whether checkboxes show and tile clicks toggle. */
		selectable: boolean;
		/** Toggle callback; the checkbox and the tile body both call it. */
		onselect: (id: string, next: boolean) => void;
	};

	let { groups, isSelected, selectable, onselect }: Props = $props();

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
	const packed: PackedMonth[] = $derived(packWall(groups, containerWidth));
</script>

<div class="wall__groups" bind:this={wall}>
	{#each packed as month (month.key)}
		<!-- Both sticky headers ride above the rows; the section's
		     scroll margin leaves room for them under the scrubber's
		     jumps. -->
		<section class="wall__month scroll-mt-[92px]" data-month-key={month.key}>
			<h2 class="wall__month-header sticky top-0 z-(--z-sticky) mt-(--wall-month-gap) mb-3 py-2.5 px-3 text-[20px] bg-page border-b">{month.label}</h2>
			{#each month.days as day (day.key)}
				<div class="wall__day" data-day-key={day.key}>
					<!-- Stacks under the month header (44px + its margins). -->
					<h3 class="wall__day-header sticky top-[44px] z-(--z-sticky-sub) mt-4 mb-2 py-1 px-2 font-body font-semibold text-[13px] tracking-[0.02em] text-text-soft bg-page border-b">{day.label}</h3>
					{#each day.rows as row, rowIndex (rowIndex)}
						<!-- The row's height is the engine's exact math; the
						     seam rides the row's gap so the widths plus the
						     gaps fill the container. Offscreen rows skip their
						     tiles' layout and paint (the height is style-pinned,
						     so the scrollbar stays honest). -->
						<div
							class="wall__row flex [content-visibility:auto]"
							style="height:{row.height}px; gap:{WALL_TILE_SEAM}px"
						>
							{#each row.tiles as tile (tile.item.id)}
								<div class="wall__cell shrink-0" style="width:{tile.width}px">
									<PhotoTile
										item={tile.item}
										selected={isSelected(tile.item.id)}
										selectable={selectable}
										onselect={onselect}
									/>
								</div>
							{/each}
						</div>
					{/each}
				</div>
			{/each}
		</section>
	{/each}
</div>
