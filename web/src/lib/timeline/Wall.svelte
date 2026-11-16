<script lang="ts">
	// THE UNIFIED WALL: ONE rendering
	// component every photo surface composes -- the timeline, the
	// sync album (a pure-navigation wall, no selection), the
	// favorites and hidden wrappers, the search browse mode. The
	// component takes bare ITEMS: the month/day fold, the row layout
	// engine, the ResizeObserver re-pack, the PhotoTile tree and the
	// month axis all live inside -- the wall's density language is
	// one component, not four copies. The page-level concerns (the
	// data walk, the toolbar, the timeline prefs) never enter: a
	// page composes `<Wall items />` plus whatever it overlays.
	//
	// THE AXIS IS THE WALL'S OWN ORGAN (since 2026-11-08,
	// superseding the earlier split that kept the scrubber a
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
	// THE SELECTION FLOW IS THE WALL'S OWN ORGAN (since 2026-11-08,
	// superseding the earlier split that
	// kept the selection and the batch bar page-level overlays):
	// the hover-reveal check, the selection state (an internal
	// SvelteSet), the select mode, the Escape exit and the bottom
	// batch bar are ONE thing with the rendering. The pages keep
	// their data walks, their action subsets, their batch runs
	// (through `onbatch`, which the wall awaits while owning the
	// busy/error/clear choreography) and their dialogs (through the
	// add-tag and add-to-memory openers, which receive the ids
	// CAPTURED AT OPEN). The capability gate is the runners'
	// PRESENCE -- `onbatch` OR the memory's membership removal
	// (`onremovememory`): a wall whose caller wired either renders
	// the tiles' check affordance (the hover-reveal entry, the
	// select-mode reveal, the mid-selection tile-toggle) and composes
	// the bar; a wall with neither renders NO check at all -- the
	// tiles are pure click-to-viewer navigation (the sync album's
	// wall, the sync-album read-only pin). No page ever wires a dead
	// affordance. `selectMode` is the one bindable: a page-level
	// arming writes it; the wall owns the drops (Escape, the bar's
	// clear) and mirrors the armed state on its root as
	// `is-selecting`. Disarming the bound mode (the page bar's
	// switch pressed off) CLEARS the selection -- the switch and the
	// checks are one state machine, and the batch bar never outlives
	// it. The membership removal runs with the batch
	// choreography -- the wall awaits it, the success arm clears the
	// selection, a failure keeps it and surfaces the mapped error.
	//
	// THE SUCCESS ARM: a landed curation pass -- an `onbatch` run
	// that resolves ok, a removal that resolves ok, or a dialog flow
	// whose `done` continuation the page fires from its report's done
	// affordance -- clears the selection but PERSISTS an armed select
	// mode (the next pass starts immediately); a failure keeps the
	// selection (the retry affordance) and surfaces the mapped error
	// in the bar.
	//
	// THE ROWS: each day group packs independently (rows never span
	// a group boundary -- the pinned rule), rows fill the container
	// width at true tile aspects with the tile seam between them, and
	// a group's tail row clamps to the target height left-aligned.
	// The target itself is a clamped fraction of the container width
	// (the engine owns the math), so the photos scale with the
	// window. The pack re-derives whenever the container resizes (a
	// ResizeObserver feeds the measured width; a layout-less host
	// falls back to the design's reference width so the wall still
	// renders deterministically).
	//
	// THE HEADERS: opaque page-colored bands with NO dividing edge
	// (the density contrast IS the separation -- a hairline
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
	// THE TWO FRAMES (the map panel's narrow wall): `variant="panel"`
	// renders the SAME wall inside a panel -- its own scroll root
	// (`h-full overflow-y-auto`), the horizontal padding on that
	// root so the row math measures the true content width (the
	// no-horizontal-scrollbar discipline), and a compact 24px axis
	// inside the panel instead of the fixed right-edge rail. The
	// rows, headers, day flow and tiles are one rendering; only the
	// frame changes.
	//
	// PERFORMANCE: every row carries content-visibility with its
	// height known from the engine's math, so offscreen rows skip
	// their tiles' layout and paint while the scrollbar stays honest
	// (the row-level windowing the justified math makes exact); the
	// tiles keep their own content-visibility inside a partially
	// visible row.
	import { locale } from 'svelte-i18n';
	import { SvelteSet } from 'svelte/reactivity';
	import { packWall, WALL_FALLBACK_WIDTH, WALL_TILE_SEAM, type PackedMonth } from './layout';
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
		 * Which frame the wall renders in. `page` is the full-page
		 * wall whose scroll root is the shell's <main> and whose axis
		 * is the fixed right-edge rail. `panel` is the same wall in a
		 * narrow container (the map's place panel): the wall carries
		 * its OWN scroll root, the horizontal padding lives on that
		 * root so the row math measures the true content width, and
		 * a compact 24px axis sits inside the panel -- one component,
		 * one density language, two widths.
		 */
		variant?: 'page' | 'panel';
		/**
		 * Called when a panel-mode wall's own scroll root is near its
		 * bottom: the host appends the next page. The panel has no
		 * page-level sentinel outside the wall, so the wall reports
		 * the reach itself.
		 */
		onreachbottom?: () => void;
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
		/**
		 * Open the memory picker over the ids CAPTURED AT OPEN: the
		 * wall-selection entry, the same hand-over as the tag opener
		 * (the picker is modal, the ids ride it, the `done`
		 * continuation carries the pass's landing).
		 */
		onaddmemory?: (ids: readonly string[], done: () => void) => void;
		/**
		 * Remove the selected rows' memberships from the memory this
		 * wall renders -- the memory's own curation write, run with
		 * the batch choreography (the wall awaits it; ok clears the
		 * selection, a failure keeps it and surfaces the mapped
		 * error). Its PRESENCE widens the capability gate: a wall
		 * wired for the removal alone renders the check affordance
		 * and composes the bar exactly like an `onbatch` wall.
		 */
		onremovememory?: (ids: readonly string[]) => Promise<WallBatchOutcome>;
	};

	let {
		items,
		variant = 'page',
		onreachbottom,
		selectMode = $bindable(false),
		onbatch,
		actions,
		onaddtag,
		onaddmemory,
		onremovememory
	}: Props = $props();

	// The selection is the wall's own reactive id set: the tiles'
	// check affordances toggle it (the entry: the check
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
	// tile (the mid-selection reveal, the touch-friendly path). The
	// wall curates when either runner is wired -- a timeline batch
	// runner OR the memory's membership removal.
	const curatable = $derived((onbatch ?? onremovememory) !== undefined);
	const selectable = $derived(curatable && (selectMode || selection.size > 0));

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
		// The measured width is the CONTENT box: a panel wall carries
		// its horizontal padding on the scroll root itself, so the
		// client width includes the gutter and the row math must
		// subtract it (rows never outgrow the panel -- the
		// no-horizontal-scrollbar discipline).
		const style = getComputedStyle(element);
		const padStart = Number.parseFloat(style.paddingLeft) || 0;
		const padEnd = Number.parseFloat(style.paddingRight) || 0;
		if (element.clientWidth - padStart - padEnd > 0) {
			measuredWidth = element.clientWidth - padStart - padEnd;
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

	const containerWidth = $derived(measuredWidth > 0 ? measuredWidth : WALL_FALLBACK_WIDTH);
	const packed: PackedMonth[] = $derived(packWall(groups, containerWidth));

	// THE MONTH AXIS (the wall's own organ -- the header pin): the
	// rail's month entries fold from the same groups the rendering
	// packs, so the wall and its axis can never disagree (the
	// timeline page's duplicate fold is gone with this move).
	const months = $derived(groups.map((group) => ({ key: group.key, label: group.label })));

	let activeMonthKey = $state<string | null>(null);
	let scrollProgress = $state(0);
	let scrollScheduled = false;

	// THE RAIL'S GEOMETRY (the one mapping): per month, the scroll
	// position that lands its header on the reading line (the same
	// landing the jumps use), the scroll span those positions live
	// on, and the tail that lets the last month reach the line. The
	// dots, the press, the readout and the indicator all read these
	// three -- no second fold, no second truth.
	let monthTops = $state<readonly number[]>([]);
	let maxScrollTop = $state(0);
	let tailHeight = $state(0);

	/** The jump landing's offset below the scrollport top: the page
	 * wall's month headers land under the shell's own top bar (the
	 * 92px scroll margin) shifted by a page's pinned-context bar;
	 * the panel wall's root starts at its own header row, so its
	 * jump lands the section flush at the root's top. */
	function landingOffset(): number {
		return variant === 'panel' ? 0 : 92 + readStickyOffset();
	}

	/** The element this wall scrolls in: the shell's <main> for the
	 * page wall, the wall's own root for the panel wall. */
	function scrollRoot(): HTMLElement | null {
		if (variant === 'panel') {
			return wall ?? null;
		}
		return document.querySelector<HTMLElement>('main');
	}

	/** Jump one month group into view (the axis's keyboard walk).
	 * The smooth landing is the pointer path's courtesy; a
	 * reduced-motion host takes the instant jump instead. */
	function jumpToMonth(key: string): void {
		const element = wall?.querySelector<HTMLElement>(`[data-month-key="${key}"]`) ?? null;
		if (element === null) {
			return;
		}
		const behavior: ScrollBehavior =
			typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
				? 'auto'
				: 'smooth';
		if (variant === 'panel') {
			// The panel's own scroller: the section's offset inside
			// it is the exact jump target (scrollIntoView would walk
			// outer ancestors too, moving the page).
			wall?.scrollTo({ top: element.offsetTop, behavior });
			return;
		}
		if (typeof element.scrollIntoView === 'function') {
			element.scrollIntoView({ block: 'start', behavior });
		}
	}

	/** The pinned-offset var's pixel value (0 unless a page's own
	 * bar pins over the wall). An unset var answers an empty string;
	 * a layout-less host answers the same nothing. */
	function readStickyOffset(): number {
		if (wall === undefined || wall === null) {
			return 0;
		}
		const value = getComputedStyle(wall).getPropertyValue('--wall-sticky-offset');
		const parsed = Number.parseInt(value, 10);
		return Number.isNaN(parsed) ? 0 : Math.max(0, parsed);
	}

	/** THE ONE RAIL SYNC: the wall measures its own geometry once per
	 * scroll (or content, or resize) pass -- each month section's
	 * offset in the scroll content, the tail that lets the last
	 * month reach the reading line, the span those positions live
	 * on, the month the viewport currently reads, and the scroll
	 * progress. Everything the axis renders (dots, indicator,
	 * press, readout) reads this one pass's numbers. */
	function syncRailState(): void {
		if (wall === undefined || wall === null) {
			return;
		}
		const sections = Array.from(wall.querySelectorAll<HTMLElement>('[data-month-key]'));
		if (sections.length === 0) {
			monthTops = [];
			maxScrollTop = 0;
			tailHeight = 0;
			activeMonthKey = null;
			scrollProgress = 0;
			return;
		}
		// A layout-less host (tests) answers zero rects: keep the
		// last known state instead of pinning everything to zero.
		const first = sections[0]?.getBoundingClientRect();
		if (first === undefined || (first.top === 0 && first.height === 0)) {
			return;
		}
		const scroller = scrollRoot();
		const scrollerTop = scroller?.getBoundingClientRect().top ?? 0;
		const scrollTop = scroller?.scrollTop ?? 0;
		const viewport = scroller?.clientHeight ?? 0;
		const sticky = readStickyOffset();
		const anchor = variant === 'panel' ? 40 : sticky + 124;
		const landing = landingOffset();
		const contentTops = sections.map((section) => scrollTop + section.getBoundingClientRect().top - scrollerTop);
		// THE TAIL (the lead-out): the wall used to end flush, so the
		// last months could never reach the reading line and the axis
		// would lie about them. One tail spacer sized so the final
		// month can come up to the line at the span's far end; a
		// single-month wall gets none (there is nothing to scrub to).
		const lastHeight = sections[sections.length - 1]?.getBoundingClientRect().height ?? 0;
		tailHeight = sections.length > 1 ? Math.max(0, Math.round(viewport - landing - lastHeight)) : 0;
		// The span is the tail-corrected content bottom: the same
		// number the dots divide and the drag multiplies.
		const lastTop = contentTops[contentTops.length - 1] ?? 0;
		maxScrollTop = Math.max(0, lastTop + lastHeight + tailHeight - viewport);
		monthTops = contentTops.map((top) => Math.max(0, Math.round(top - landing)));
		// The active month: the last section whose top has passed the
		// anchor line (the sticky stack the headers pin below).
		let current: string | null = null;
		for (let i = 0; i < sections.length; i += 1) {
			if ((contentTops[i] ?? 0) <= scrollTop + anchor) {
				current = sections[i]?.dataset.monthKey ?? null;
			} else {
				break;
			}
		}
		activeMonthKey = current;
		scrollProgress = maxScrollTop > 0 ? Math.min(1, Math.max(0, scrollTop / maxScrollTop)) : 0;
	}

	/** THE TRUE DRAG: the month axis IS a scrollbar -- the pointer's
	 * rail fraction maps directly onto the wall's continuous scroll
	 * position (within months), applied frame-accurate with no
	 * smoothing (direct manipulation). The span is the rail sync's
	 * own tail-corrected number, so a press at a dot's height lands
	 * exactly that month's reading position. */
	function handleRailScrub(fraction: number): void {
		const scroller = scrollRoot();
		if (scroller === null || maxScrollTop <= 0) {
			return;
		}
		scroller.scrollTop = Math.min(1, Math.max(0, fraction)) * maxScrollTop;
	}

	/** The panel's walk hook: near the root's bottom, the host pages
	 * the next batch. The page wall keeps its sentinel observer.
	 * The tail is runway, not content: the reach measures the real
	 * content bottom, with the tail's own height out of the way. */
	function maybeReachBottom(): void {
		if (variant !== 'panel' || onreachbottom === undefined || wall === undefined || wall === null) {
			return;
		}
		const remaining = wall.scrollHeight - wall.scrollTop - wall.clientHeight - tailHeight;
		if (remaining <= 600) {
			onreachbottom();
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
			syncRailState();
			maybeReachBottom();
		});
	}

	// THE SCROLL ROOT'S OWN LISTENER: the page wall listens in the
	// capture phase at the document (the shell's <main> is an
	// ancestor the wall must observe without naming it per event);
	// the panel wall's root is its own scroll container, so one
	// element listener is exact. A layoutless host (tests) still
	// resolves the same functions.
	$effect(() => {
		if (variant === 'panel') {
			const element = wall;
			if (element === undefined) {
				return;
			}
			element.addEventListener('scroll', handleScrollEvent, { passive: true });
			return () => {
				element.removeEventListener('scroll', handleScrollEvent);
			};
		}
		document.addEventListener('scroll', handleScrollEvent, { capture: true, passive: true });
		return () => {
			document.removeEventListener('scroll', handleScrollEvent, { capture: true });
		};
	});

	// The wall's own content changes (a page appended, an event
	// removed a tile, a re-pack from a width change): the dots, the
	// tail, the stop and the progress must follow without waiting
	// for a scroll, and the panel's next page starts when the fresh
	// content still does not fill the root.
	$effect(() => {
		void packed;
		requestAnimationFrame(() => {
			syncRailState();
			maybeReachBottom();
		});
	});

	// The viewport's own resizes move the tail (it is sized off the
	// scroll root's height) without any re-pack: the same rAF pass
	// re-measures everything the axis reads.
	$effect(() => {
		window.addEventListener('resize', handleScrollEvent);
		return () => {
			window.removeEventListener('resize', handleScrollEvent);
		};
	});

	// The month axis IS the wall's scrollbar: while the PAGE wall
	// lives, the shell's scroll root hides its own bar (the axis's
	// position indicator carries the feedback). The panel wall's
	// root hides its bar by style (the compact axis is inside it).
	$effect(() => {
		if (variant === 'panel') {
			return;
		}
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

	// THE DISARM'S OTHER HALF: a page-level switch (the action bar's
	// select toggle) writes the bound mode, and turning it OFF ends
	// the selection too -- the switch is one state machine, never a
	// skin over a hidden set. An armed mode that arrives with an
	// empty set stays a no-op, so the initial write costs nothing.
	$effect(() => {
		if (!selectMode) {
			selection.clear();
		}
	});

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

	// The memory picker's opener: the same captured-ids hand-over
	// (the modal-pinned selection discipline).
	function openAddMemory(): void {
		if (onaddmemory === undefined || selection.size === 0) {
			return;
		}
		onaddmemory([...selection], passLanded);
	}

	// The membership removal: the page's write, the wall's
	// choreography -- exactly the batch run's shape (busy disables
	// the bar, the success arm clears the selection, a failure keeps
	// it and surfaces the mapped error).
	async function runRemoveMemory(): Promise<void> {
		if (onremovememory === undefined || batchBusy || selection.size === 0) {
			return;
		}
		batchBusy = true;
		batchErrorKey = '';
		const outcome = await onremovememory([...selection]);
		batchBusy = false;
		if (outcome.ok) {
			passLanded();
		} else {
			batchErrorKey = outcome.errorKey;
		}
	}
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<div class="wall {variant === 'panel' ? 'wall--panel relative h-full min-h-0' : ''}">
	<div
		class="wall__groups {variant === 'panel'
			? 'wall__groups--panel relative h-full overflow-y-auto px-1 pe-8 pb-2'
			: ''}"
		id={variant === 'panel' ? 'wall-panel-scroll' : undefined}
		class:is-selecting={selectMode}
		bind:this={wall}
	>
		{#each packed as month (month.key)}
			<!-- Both sticky headers ride above the rows; the section's
		     scroll margin leaves room for them under the axis's jumps
		     (the pinned-offset term shifts the landing below a page's
		     own pinned bar). The day header stacks under the month band
		     exactly (the band is 38px: py-1.5's 12px + the 26px line at
		     text-[20px]/1.3, probe-verified). The month header's top
		     rides the pinned-offset var (0 unless a page's own bar
		     pins over the wall). -->
			<section
				class="wall__month scroll-mt-[calc(92px_+_var(--wall-sticky-offset,0px))]"
				data-month-key={month.key}
			>
				<h2
					class="wall__month-header sticky top-[var(--wall-sticky-offset,0px)] z-(--z-sticky) mt-(--wall-month-gap) mb-2 py-1.5 text-[20px] bg-page"
				>
					{month.label}
				</h2>
				<!-- The day blocks flow horizontally and wrap (the day-flow
				     shape): a day is a header plus its justified
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
										     wall with no runner wired passes no toggle, so
										     PhotoTile renders no check at all (the pure
										     navigation walls); a curatable wall always passes
										     it and the reveal rides the selectable state. -->
											<PhotoTile
												item={tile.item}
												selected={selection.has(tile.item.id)}
												{selectable}
												onselect={curatable ? toggleSelected : undefined}
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
		{#if tailHeight > 0}
			<!-- THE TAIL (the lead-out): runway so the last month can
		     come up to the reading line -- without it the wall ends
		     flush and the axis would lie about the final months. The
		     height is the rail sync's own measure (the viewport minus
		     the reading line minus the last month); a single-month
		     wall gets none. -->
			<div class="wall__tail" style="height: {tailHeight}px" aria-hidden="true"></div>
		{/if}
	</div>

	{#if variant === 'panel' && months.length > 0}
		<!-- The panel's compact axis (the same organ, the narrow
		     slot): the ONE scrollbar control in its 24px frame -- the
		     press, the drag, the readout and the keyboard are the
		     page rail's, the geometry is the panel's own. -->
		<MonthScrubber
			variant="panel"
			controls="wall-panel-scroll"
			{months}
			tops={monthTops}
			maxScroll={maxScrollTop}
			activeKey={activeMonthKey}
			progress={scrollProgress}
			onjump={jumpToMonth}
			onscrub={handleRailScrub}
		/>
	{/if}
</div>

{#if variant !== 'panel'}
	<!-- The wall's own month axis: the photo
	     rendering and the rail are one thing). The scrubber is
	     viewport-fixed, so it renders from any wall depth; its empty
	     months guard hides it for an empty wall. -->
	<MonthScrubber
		controls="shell-main"
		{months}
		tops={monthTops}
		maxScroll={maxScrollTop}
		activeKey={activeMonthKey}
		progress={scrollProgress}
		onjump={jumpToMonth}
		onscrub={handleRailScrub}
	/>
{/if}

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
		onaddmemory={onaddmemory !== undefined ? openAddMemory : undefined}
		onremovememory={onremovememory !== undefined ? runRemoveMemory : undefined}
		onaction={runAction}
		onclear={exitSelection}
	/>
{/if}

<style>
	/* The panel wall's root: its compact axis IS the scrollbar (the
	 * same discipline the page wall applies to the shell's root), so
	 * the native bar hides -- the axis's own stops carry the
	 * position. */
	.wall__groups--panel {
		scrollbar-width: none;
	}

	.wall__groups--panel::-webkit-scrollbar {
		display: none;
	}

	/* The panel frame's type scale follows its width: the same wall,
	 * the narrow container's own header sizes -- one component, one
	 * density language, two widths. */
	.wall--panel .wall__month-header {
		font-size: 14px;
	}

	.wall--panel .wall__day-header {
		font-size: 11px;
	}
</style>
