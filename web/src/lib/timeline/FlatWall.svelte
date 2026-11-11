<script lang="ts">
	// The flat justified wall (the search's query mode):
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
	//
	// THE SELECTION FLOW (the search walls' curation ruling: BOTH
	// search walls carry the per-user subset): the same organ the
	// month-grouped Wall carries -- the hover-reveal check, the
	// selection set, the Escape exit and the bottom batch bar -- in
	// the flat wall's own body. The capability gate is `onbatch`'s
	// PRESENCE, exactly the month wall's rule: a caller that wires
	// the runner renders the check affordance and the bar; a wall
	// with no runner renders NO check at all (the tiles are pure
	// click-to-viewer navigation). No select-mode arming here (the
	// search page arms none -- the hover check and the first click
	// are the entry, as on every wall).
	import { SvelteSet } from 'svelte/reactivity';
	import { packRows, WALL_MAX_WIDTH, WALL_TILE_SEAM, type PackedRow } from './layout';
	import PhotoTile from './PhotoTile.svelte';
	import BatchBar from './BatchBar.svelte';
	import type { BatchAction, TimelineItem } from '$lib/api/timeline';

	/**
	 * The outcome the wall awaits from a page's batch run (the month
	 * wall's structural contract, shared verbatim).
	 */
	type WallBatchOutcome = { ok: true } | { ok: false; errorKey: string };

	type Props = {
		/** The result items, in the page's own order (rank order). */
		items: readonly TimelineItem[];
		/**
		 * Run one batch action over the wall's selection (the page's
		 * client owns the write and its convergence). ABSENT = this
		 * wall offers no curation: no check renders, no bar ever
		 * rises, the tiles are pure navigation. The callback's
		 * presence is the capability, so no caller ever wires a dead
		 * affordance.
		 */
		onbatch?: (action: BatchAction, ids: readonly string[]) => Promise<WallBatchOutcome>;
		/**
		 * Which of the bar's actions render, keyed like the batch API.
		 * The search walls pass the per-user subset their ruling pins
		 * (the favorite toggle; the tag flow rides the opener).
		 */
		actions?: readonly BatchAction[];
		/**
		 * Open the add-tag dialog over the ids CAPTURED AT OPEN (the
		 * tag flow carries its own dialog and endpoint, so it rides
		 * this opener, exactly the month wall's contract).
		 */
		onaddtag?: (ids: readonly string[], done: () => void) => void;
	};

	let { items, onbatch, actions, onaddtag }: Props = $props();

	// The selection is the wall's own reactive id set (the month
	// wall's semantics, shared): the tiles' check affordances toggle
	// it, Escape drops it, and a landed pass clears it (the success
	// arm; a failure keeps it -- the retry affordance).
	const selection = new SvelteSet<string>();

	let batchBusy = $state(false);
	/** The error-code i18n key of the last failed batch run ('' when none). */
	let batchErrorKey = $state('');

	// The check affordance renders while a selection is alive: the
	// hover reveals the entry, the first click enters the selection,
	// and the mid-selection reveal keeps every check visible.
	const selectable = $derived(onbatch !== undefined && selection.size > 0);

	const selectedItems = $derived(items.filter((item) => selection.has(item.id)));
	const allFavorite = $derived(selectedItems.length > 0 && selectedItems.every((item) => item.isFavorite));

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

	function toggleSelected(id: string, next: boolean): void {
		if (next) {
			selection.add(id);
		} else {
			selection.delete(id);
		}
	}

	/** The full exit: the selection drops (no armed mode to disarm). */
	function exitSelection(): void {
		selection.clear();
	}

	// The Escape exit: a live selection drops -- the same full exit
	// the bar's clear action runs.
	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && selection.size > 0) {
			exitSelection();
		}
	}

	// The batch run: the page's client owns the write; the wall owns
	// the choreography around the awaited promise -- busy disables
	// the bar's buttons, ok clears the selection (the success arm),
	// a failure keeps it (the retry affordance) and surfaces the
	// mapped error.
	async function runAction(action: BatchAction): Promise<void> {
		if (onbatch === undefined || batchBusy || selection.size === 0) {
			return;
		}
		batchBusy = true;
		batchErrorKey = '';
		const outcome = await onbatch(action, [...selection]);
		batchBusy = false;
		if (outcome.ok) {
			selection.clear();
		} else {
			batchErrorKey = outcome.errorKey;
		}
	}

	// The dialog opener: the ids are captured at open and handed to
	// the page together with the done continuation the pass's
	// landing fires (the bar's affordance renders only when the page
	// wired the opener).
	function openAddTag(): void {
		if (onaddtag === undefined || selection.size === 0) {
			return;
		}
		onaddtag([...selection], () => selection.clear());
	}
</script>

<svelte:window onkeydown={handleWindowKeydown} />

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

<!-- THE BOTTOM BATCH BAR: the same presentation-only bar the
     month wall composes, fed with the honest count, the
     all-favorite toggle state, this wall's busy/error choreography
     and the page's action subset. -->
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
