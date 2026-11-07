<script lang="ts">
	// THE UNIFIED WALL (the 2026-11-07 owner pin 1): ONE pure
	// rendering component every photo surface composes -- the
	// timeline (its scrubber, its selection and its batch bar stay
	// the page's own overlays), the album detail (a pure-navigation
	// wall, no selection), the favorites and hidden wrappers, the
	// search browse mode. The component takes bare ITEMS: the
	// month/day fold, the row layout engine, the ResizeObserver
	// re-pack and the PhotoTile tree all live inside -- the wall's
	// density language is one component, not four copies. The
	// page-level concerns (the data walk, the selection state, the
	// batch bar, the toolbar, the scrubber, the timeline prefs)
	// never enter: a page composes `<Wall items />` plus whatever it
	// overlays.
	//
	// THE SELECTION CONTRACT: `onselect`'s PRESENCE is the
	// capability. A wall whose caller wired the callback renders the
	// tiles' check affordance (the hover-reveal entry, the
	// select-mode reveal, the mid-selection tile-toggle); a wall
	// with no callback renders NO check at all -- the tiles are pure
	// click-to-viewer navigation (the album detail's wall, owner pin
	// 5). No page ever wires a dead affordance.
	//
	// THE ROWS: each day group packs independently (rows never span
	// a group boundary -- the pinned rule), rows fill the container
	// width at true tile aspects with the tile seam between them, and
	// a group's tail row clamps to the target height left-aligned.
	// The pack re-derives whenever the container resizes (a
	// ResizeObserver feeds the measured width; a layout-less host
	// falls back to the content column maximum so the wall still
	// renders deterministically).
	//
	// THE HEADERS: opaque page-colored bands with NO dividing edge
	// (R2-12: the density contrast IS the separation -- a hairline
	// under every band was the pre-rework drift away from the
	// photo-first language), the vertical rhythm tightened (the month
	// band's ~py-1.5/mb-2, the day header closer still). The tiles
	// scroll under them. The month gap token above each group is the
	// breathing room the density contrast reads on.
	//
	// THE PINNED OFFSET: a page whose own contextual chrome pins over
	// the wall (the album detail's sticky bar) sets
	// `--wall-sticky-offset` on an ancestor -- the month headers
	// then stick BELOW that bar instead of the scroll root's top
	// edge; every other page leaves the var unset and the headers
	// stick at 0 like they always did.
	//
	// PERFORMANCE: every row carries content-visibility with its
	// height known from the engine's math, so offscreen rows skip
	// their tiles' layout and paint while the scrollbar stays honest
	// (the row-level windowing the justified math makes exact); the
	// tiles keep their own content-visibility inside a partially
	// visible row.
	import { locale } from 'svelte-i18n';
	import { packWall, WALL_MAX_WIDTH, WALL_TILE_SEAM, type PackedMonth } from './layout';
	import { groupTimelineItems } from './grouping';
	import type { TimelineItem } from '$lib/api/timeline';
	import PhotoTile from './PhotoTile.svelte';

	type Props = {
		/** The loaded items, newest first (the page owns the walk that fills this). */
		items: readonly TimelineItem[];
		/** Whether checkboxes show and tile clicks toggle (a select mode or an active selection). */
		selectable?: boolean;
		/** Whether one item is in the current selection. */
		isSelected?: (id: string) => boolean;
		/**
		 * The selection toggle. ABSENT = this wall offers no selection:
		 * no check renders, the tiles are pure navigation (the album
		 * detail's wall).
		 */
		onselect?: (id: string, next: boolean) => void;
	};

	let {
		items,
		selectable = false,
		isSelected = () => false,
		onselect
	}: Props = $props();

	// The fold is the wall's own: bare items in, locale-aware month
	// and day groups out. The input order is the rendering order (the
	// keyset ordering, newest first) and the fold keeps it.
	const groups = $derived(groupTimelineItems(items, $locale ?? 'en'));

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
		     jumps. The day header stacks under the month band exactly
		     (the band is 38px: py-1.5's 12px + the 26px line at
		     text-[20px]/1.3, probe-verified). The month header's top
		     rides the pinned-offset var (0 unless a page's own bar
		     pins over the wall). -->
		<section class="wall__month scroll-mt-[92px]" data-month-key={month.key}>
			<h2 class="wall__month-header sticky top-[var(--wall-sticky-offset,0px)] z-(--z-sticky) mt-(--wall-month-gap) mb-2 py-1.5 text-[20px] bg-page">{month.label}</h2>
			<!-- The day blocks flow horizontally and wrap (the reference
			     timeline's shape): a day is a header plus its justified
			     stack at its own width, and sparse dates sit beside their
			     neighbors instead of leaving a hole. -->
			<div class="wall__day-flow flex flex-wrap gap-3">
			{#each month.days as day (day.key)}
				<div class="wall__day shrink-0" data-day-key={day.key} style="width:{day.width}px">
					<h3 class="wall__day-header mt-1 mb-1.5 font-body font-semibold text-[13px] tracking-[0.02em] text-text-soft">{day.label}</h3>
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
			</div>
		</section>
	{/each}
</div>
