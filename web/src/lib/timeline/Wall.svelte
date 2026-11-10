<script lang="ts">
	// THE UNIFIED WALL (the 2026-11-07 owner pin 1): ONE rendering
	// component every photo surface composes -- the timeline, the
	// album detail (a pure-navigation wall, no selection), the
	// favorites and hidden wrappers, the search browse mode. The
	// component takes bare ITEMS: the month/day fold, the row layout
	// engine, the ResizeObserver re-pack, the PhotoTile tree and the
	// month axis all live inside -- the wall's density language is
	// one component, not four copies. The page-level concerns (the
	// data walk, the toolbar, the timeline prefs) never enter: a
	// page composes `<Wall items />` plus whatever it overlays.
	//
	// THE AXIS IS THE WALL'S OWN ORGAN (the 2026-11-08 owner pin,
	// superseding the 2026-11-05 split that kept the scrubber a
	// page-level overlay): the photo rendering and the right-edge
	// month axis are one thing, not a rendering plus a page's
	// overlay. Every month-grouped wall carries the rail -- its
	// months fold from the SAME internal groups the rendering packs
	// (no page re-folds), its jump and its active stop track the
	// wall's own [data-month-key] sections, and its drag drives the
	// shell's scroll root: the axis IS the wall's scrollbar, so the
	// scroll root's native bar hides while a wall lives. An empty
	// wall renders no rail (the scrubber's own guard).
	//
	// THE SELECTION FLOW IS THE WALL'S OWN ORGAN (the 2026-11-08
	// owner pin, web IA §7.4, superseding the 2026-11-07 split that
	// kept the selection and the batch bar page-level overlays):
	// the hover-reveal check, the selection state (an internal
	// SvelteSet), the select mode, the Escape exit and the bottom
	// batch bar are ONE thing with the rendering. The pages keep
	// their data walks, their action subsets, their batch runs
	// (through `onbatch`, which the wall awaits while owning the
	// busy/error/clear choreography) and their dialogs (through the
	// add-tag opener, which receives the ids CAPTURED AT OPEN). The
	// capability gate is `onbatch`'s PRESENCE: a wall whose caller
	// wired the runner renders the tiles' check affordance (the
	// hover-reveal entry, the select-mode reveal, the mid-selection
	// tile-toggle) and composes the bar; a wall with no runner
	// renders NO check at all -- the tiles are pure click-to-viewer
	// navigation (the album detail's wall, the sync-album read-only
	// pin). No page ever wires a dead affordance. `selectMode` is
	// the one bindable: a page-level arming writes it; the wall owns
	// the drops (Escape, the bar's clear). The add-to-album opener
	// is GONE with its flow (E4, web IA §7.5).
	//
	// THE SUCCESS ARM: a landed curation pass -- an `onbatch` run
	// that resolves ok, or a dialog flow whose `done` continuation
	// the page fires from its report's done affordance -- clears
	// the selection but PERSISTS an armed select mode (the next
	// pass starts immediately); a failure keeps the selection (the
	// retry affordance) and surfaces the mapped error in the bar.
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
	// stick at 0 like they always did. The offset moves the axis
	// with it: the jump landing (the section's scroll margin) and
	// the active-month anchor both shift by the same value, read
	// back from the computed var so the stop switches at the exact
	// line the headers pin to.
	//
	// PERFORMANCE: every row carries content-visibility with its
	// height known from the engine's math, so offscreen rows skip
	// their tiles' layout and paint while the scrollbar stays honest
	// (the row-level windowing the justified math makes exact); the
	// tiles keep their own content-visibility inside a partially
	// visible row.
	import { locale } from 'svelte-i18n';
	import { SvelteSet } from 'svelte/reactivity';
	import { packWall, WALL_MAX_WIDTH, WALL_TILE_SEAM, type PackedMonth } from './layout';
	import { groupTimelineItems } from './grouping';
	import type { BatchAction, TimelineItem } from '$lib/api/timeline';
	import PhotoTile from './PhotoTile.svelte';
	import MonthScrubber from './MonthScrubber.svelte';
	import BatchBar from './BatchBar.svelte';

	/**
	 * The outcome the wall awaits from a page's batch run. The
	 * client's richer BatchOutcome (with affected/missing) satisfies
	 * this structurally; the wall reads nothing but the ok/errorKey
	 * split -- the choreography is its own.
	 */
	type WallBatchOutcome = { ok: true } | { ok: false; errorKey: string };

	type Props = {
		/** The loaded items, newest first (the page owns the walk that fills this). */
		items: readonly TimelineItem[];
		/**
		 * The armed select mode (bindable): a page-level arming writes
		 * it; the wall owns the drops (Escape, the bar's clear). An
		 * armed mode reveals every check (the touch-friendly entry).
		 */
		selectMode?: boolean;
		/**
		 * Run one batch action over the wall's selection (the page's
		 * client owns the write and its optimistic convergence). ABSENT
		 * = this wall offers no curation: no check renders, no bar
		 * ever rises, the tiles are pure navigation (the album
		 * detail's wall). The callback's presence is the capability,
		 * so no caller ever wires a dead affordance.
		 */
		onbatch?: (action: BatchAction, ids: readonly string[]) => Promise<WallBatchOutcome>;
		/**
		 * Which of the bar's actions render, keyed like the batch API
		 * ('favorite' names the favorite/unfavorite toggle). The
		 * default is every action; the curated walls pass the subset
		 * their view makes sense of (unfavorite on favorites, unhide
		 * on hidden).
		 */
		actions?: readonly BatchAction[];
		/**
		 * Open the add-tag dialog over the ids CAPTURED AT OPEN
		 * (the dialog is modal, so the capture is semantically
		 * identical to reading the live selection at submit). The tag
		 * flow is not a batch action -- it carries its own dialog and
		 * endpoint -- so it rides this opener instead of the action
		 * set. The `done` continuation reports the pass's landing:
		 * the page fires it from the report's done affordance, and
		 * the wall applies its success arm (clear the selection,
		 * persist an armed mode).
		 */
		onaddtag?: (ids: readonly string[], done: () => void) => void;
	};

	let { items, selectMode = $bindable(false), onbatch, actions, onaddtag }: Props = $props();

	// The selection is the wall's own reactive id set: the tiles'
	// check affordances toggle it (the entry, R2-11: the check
	// reveals on hover/keyboard focus on every tile, and its first
	// click enters the selection), Esc and the bar's clear drop it,
	// and a landed pass clears it (the success arm). It never leaves
	// the component -- the pages reach its effects through the bar
	// and the captured ids the openers hand them.
	const selection = new SvelteSet<string>();

	let batchBusy = $state(false);
	/** The error-code i18n key of the last failed batch run ('' when none). */
	let batchErrorKey = $state('');

	// The check affordance renders exactly when the wall can curate
	// AND the selection is alive: an armed select mode or one checked
	// tile (the mid-selection reveal, the touch-friendly path).
	const selectable = $derived(onbatch !== undefined && (selectMode || selection.size > 0));

	const selectedItems = $derived(items.filter((timelineItem) => selection.has(timelineItem.id)));
	const allFavorite = $derived(
		selectedItems.length > 0 && selectedItems.every((timelineItem) => timelineItem.isFavorite)
	);

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

	// THE MONTH AXIS (the wall's own organ -- the header pin): the
	// rail's month entries fold from the same groups the rendering
	// packs, so the wall and its axis can never disagree (the
	// timeline page's duplicate fold is gone with this move).
	const months = $derived(
		groups.map((group) => ({
			key: group.key,
			label: group.label,
			count: group.days.reduce((total, day) => total + day.items.length, 0)
		}))
	);

	let activeMonthKey = $state<string | null>(null);
	let scrollProgress = $state(0);
	let scrollScheduled = false;

	/** Jump one month group into view (a rail tap or the keyboard). */
	function jumpToMonth(key: string): void {
		const element = wall?.querySelector<HTMLElement>(`[data-month-key="${key}"]`) ?? null;
		if (element !== null && typeof element.scrollIntoView === 'function') {
			element.scrollIntoView({ block: 'start', behavior: 'smooth' });
		}
	}

	/** The pinned-offset var's pixel value (0 unless a page's own
	 * bar pins over the wall). An unset var answers an empty string;
	 * a layout-less host answers the same nothing. */
	function readStickyOffset(): number {
		if (wall === undefined) {
			return 0;
		}
		const value = getComputedStyle(wall).getPropertyValue('--wall-sticky-offset');
		const parsed = Number.parseInt(value, 10);
		return Number.isNaN(parsed) ? 0 : Math.max(0, parsed);
	}

	/** Track which month the viewport reads, for the rail's stop. */
	function updateActiveMonth(): void {
		if (wall === undefined) {
			return;
		}
		const sections = Array.from(wall.querySelectorAll<HTMLElement>('[data-month-key]'));
		if (sections.length === 0) {
			return;
		}
		const first = sections[0]?.getBoundingClientRect();
		// A layout-less host (tests) answers zero rects: keep no stop.
		if (first === undefined || (first.top === 0 && first.height === 0)) {
			return;
		}
		// The frame's scroll root is the shell's <main>: the sticky
		// month/day header stack pins to its content top (main's p-6
		// padding below the pinned top bar), so the anchor keeps the
		// old rule's 100px -- the stack plus its grace -- below that
		// edge, shifted by the pinned-offset var when a page's own
		// bar pins over the wall (the album detail's contextual
		// bar): the stop switches at the same line the headers pin
		// to. A page mounted without the shell (the unit tests)
		// reads the viewport's top.
		const scroller = document.querySelector<HTMLElement>('main');
		const scrollerTop = scroller?.getBoundingClientRect().top ?? 0;
		const anchor = scrollerTop + readStickyOffset() + 124;
		let current: string | null = null;
		for (const section of sections) {
			if (section.getBoundingClientRect().top <= anchor) {
				current = section.dataset.monthKey ?? null;
			} else {
				break;
			}
		}
		activeMonthKey = current;
	}

	/** The wall's scroll progress (0-1) for the rail's position
	 * indicator: the scroll root's own fraction, read alongside the
	 * active-month tracking. */
	function updateScrollProgress(): void {
		const scroller = document.querySelector<HTMLElement>('main');
		if (scroller === null) {
			return;
		}
		const span = scroller.scrollHeight - scroller.clientHeight;
		scrollProgress = span > 0 ? Math.min(1, Math.max(0, scroller.scrollTop / span)) : 0;
	}

	/** THE TRUE DRAG: the month axis IS a scrollbar -- the pointer's
	 * rail fraction maps directly onto the wall's continuous scroll
	 * position (within months), applied frame-accurate with no
	 * smoothing (direct manipulation). */
	function handleRailScrub(fraction: number): void {
		const scroller = document.querySelector<HTMLElement>('main');
		if (scroller === null) {
			return;
		}
		const span = scroller.scrollHeight - scroller.clientHeight;
		if (span > 0) {
			scroller.scrollTop = fraction * span;
		}
	}

	/** The rAF-collapsed scroll notification (one pass per frame). */
	function handleScrollEvent(): void {
		if (scrollScheduled) {
			return;
		}
		scrollScheduled = true;
		requestAnimationFrame(() => {
			scrollScheduled = false;
			updateActiveMonth();
			updateScrollProgress();
		});
	}

	// THE FRAME MIGRATION'S RISKY EDGE (R2-1): scrolling lives in
	// the shell's <main>, and element scroll events do not bubble --
	// a window-level onscroll handler would silently never fire
	// again. The tracking therefore listens in the CAPTURE phase on
	// the document: the one registration that observes every scroll
	// container below it (the frame's main today, whatever scrolls
	// tomorrow).
	$effect(() => {
		document.addEventListener('scroll', handleScrollEvent, { capture: true, passive: true });
		return () => {
			document.removeEventListener('scroll', handleScrollEvent, { capture: true });
		};
	});

	// The month axis IS the wall's scrollbar: while this wall lives,
	// the scroll root's own bar hides (the axis's position indicator
	// carries the feedback). Other pages keep the thin custom bar.
	$effect(() => {
		const scroller = document.querySelector('main');
		scroller?.classList.add('wall-rail-scroll');
		return () => {
			scroller?.classList.remove('wall-rail-scroll');
		};
	});

	function toggleSelected(id: string, next: boolean): void {
		if (next) {
			selection.add(id);
		} else {
			selection.delete(id);
		}
	}

	/** The full exit: the selection drops AND an armed mode disarms. */
	function exitSelection(): void {
		selection.clear();
		selectMode = false;
	}

	// The Escape exit (the pages' own keydown handlers for this went
	// with the move): a live selection or an armed select mode drops
	// -- the same full exit the bar's clear action runs.
	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && (selection.size > 0 || selectMode)) {
			exitSelection();
		}
	}

	/** The success arm: a landed pass clears the selection; an
	 * armed select mode persists for the next one. */
	function passLanded(): void {
		selection.clear();
	}

	// The batch run: the page's client owns the write; the wall owns
	// the choreography around the awaited promise -- busy disables
	// the bar's buttons, ok clears the selection (the success arm,
	// an armed mode persists), a failure keeps it (the retry
	// affordance) and surfaces the mapped error.
	async function runAction(action: BatchAction): Promise<void> {
		if (onbatch === undefined || batchBusy || selection.size === 0) {
			return;
		}
		batchBusy = true;
		batchErrorKey = '';
		const outcome = await onbatch(action, [...selection]);
		batchBusy = false;
		if (outcome.ok) {
			passLanded();
		} else {
			batchErrorKey = outcome.errorKey;
		}
	}

	// The dialog opener: the ids are captured at open and handed to
	// the page together with the done continuation the pass's
	// landing fires (the bar's affordance renders only when the page
	// wired the opener -- the timeline's bar carries it, the curated
	// walls do not).
	function openAddTag(): void {
		if (onaddtag === undefined || selection.size === 0) {
			return;
		}
		onaddtag([...selection], passLanded);
	}
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<div class="wall__groups" bind:this={wall}>
	{#each packed as month (month.key)}
		<!-- Both sticky headers ride above the rows; the section's
		     scroll margin leaves room for them under the axis's jumps
		     (the pinned-offset term shifts the landing below a page's
		     own pinned bar). The day header stacks under the month band
		     exactly (the band is 38px: py-1.5's 12px + the 26px line at
		     text-[20px]/1.3, probe-verified). The month header's top
		     rides the pinned-offset var (0 unless a page's own bar
		     pins over the wall). -->
		<section class="wall__month scroll-mt-[calc(92px_+_var(--wall-sticky-offset,0px))]" data-month-key={month.key}>
			<h2
				class="wall__month-header sticky top-[var(--wall-sticky-offset,0px)] z-(--z-sticky) mt-(--wall-month-gap) mb-2 py-1.5 text-[20px] bg-page"
			>
				{month.label}
			</h2>
			<!-- The day blocks flow horizontally and wrap (the reference
			     timeline's shape): a day is a header plus its justified
			     stack at its own width, and sparse dates sit beside their
			     neighbors instead of leaving a hole. -->
			<div class="wall__day-flow flex flex-wrap gap-3">
				{#each month.days as day (day.key)}
					<div class="wall__day shrink-0" data-day-key={day.key} style="width:{day.width}px">
						<h3
							class="wall__day-header mt-1 mb-1.5 font-body font-semibold text-[13px] tracking-[0.02em] text-text-soft"
						>
							{day.label}
						</h3>
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
										<!-- The capability gate rides the onselect hand-off: a
									     wall with no onbatch runner passes no toggle, so
									     PhotoTile renders no check at all (the pure
									     navigation walls); a curatable wall always passes
									     it and the reveal rides the selectable state. -->
										<PhotoTile
											item={tile.item}
											selected={selection.has(tile.item.id)}
											{selectable}
											onselect={onbatch !== undefined ? toggleSelected : undefined}
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

<!-- The wall's own month axis (the 2026-11-08 pin: the photo
     rendering and the rail are one thing). The scrubber is
     viewport-fixed, so it renders from any wall depth; its empty
     months guard hides it for an empty wall. -->
<MonthScrubber
	{months}
	activeKey={activeMonthKey}
	progress={scrollProgress}
	onjump={jumpToMonth}
	onscrub={handleRailScrub}
/>

<!-- THE BOTTOM BATCH BAR (the 2026-11-08 selection pin's last
     organ): the wall composes the presentation-only bar whenever
     its selection holds at least one tile, feeding it the honest
     count, the all-favorite toggle state, its own busy/error
     choreography and the page's action subset. The dialog
     affordance renders only when the page wired its opener (the
     ids the opener captures flow to the page's dialogs). -->
{#if selection.size > 0}
	<BatchBar
		count={selection.size}
		{allFavorite}
		errorKey={batchErrorKey}
		busy={batchBusy}
		{actions}
		onaddtag={onaddtag !== undefined ? openAddTag : undefined}
		onaction={runAction}
		onclear={exitSelection}
	/>
{/if}
