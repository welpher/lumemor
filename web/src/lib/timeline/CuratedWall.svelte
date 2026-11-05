<script lang="ts">
	// The curated wall the favorites and hidden pages render (M2d
	// Task 8): the timeline's own machinery -- the pagination client
	// (driven with the view's membership semantics), month and day
	// groups, PhotoTile, the selection bar, the SSE convergence and
	// the wallItems publication the asset viewer navigates --
	// assembled for a view whose rows are a membership question
	// rather than the aggregate stream. The page tells the wall its
	// endpoint, its view, its strings and its action subset; the
	// timeline page itself keeps its richer layout (the month
	// scrubber, the scroll tracking) and does not use this component.
	//
	// THE ALBUM WALL (M3c Task 5): the album detail's grid reuses
	// this wall over the timeline endpoint's `albumId` narrow -- the
	// album's own live items in the timeline's exact shape, paged and
	// converged the same way. The `resetKey` prop is the album's
	// membership convergence: an album.items-changed event lands with
	// ids only, so the wall restarts its walk and the fresh answer
	// owns the truth.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { SvelteSet } from 'svelte/reactivity';
	import { untrack } from 'svelte';
	import { createTimelineClient, type BatchAction, type WallView } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import { groupTimelineItems } from '$lib/timeline/grouping';
	import { publishWallItems } from '$lib/timeline/wall.svelte';
	import MonthGroups from '$lib/timeline/MonthGroups.svelte';
	import BatchBar from '$lib/timeline/BatchBar.svelte';
	import Button from '$lib/components/Button.svelte';

	type Props = {
		/** The timeline-shaped endpoint the wall walks. */
		path: string;
		/** Which membership view the client serves (see WallView). */
		view: WallView;
		/**
		 * The album whose own timeline this wall walks (the endpoint's
		 * `albumId` narrow). Absent on the favorites and hidden walls.
		 */
		albumId?: string;
		/** The i18n key of the view's name (the empty state's heading). */
		titleKey?: string;
		/**
		 * The empty state's heading as direct text -- the album wall
		 * passes the album's title; it overrides `titleKey`.
		 */
		heading?: string;
		/** The i18n key of the view's loading line. */
		loadingKey: string;
		/** The i18n key of the view's empty state text. */
		emptyKey: string;
		/** Which batch actions the selection bar offers (see BatchBar). */
		actions: readonly BatchAction[];
		/**
		 * The membership convergence: a change drops the loaded pages
		 * and restarts the walk from page one (the album wall's
		 * items-changed restart).
		 */
		resetKey?: number | string;
	};

	let {
		path,
		view,
		albumId,
		titleKey = '',
		heading = '',
		loadingKey,
		emptyKey,
		actions,
		resetKey = 0
	}: Props = $props();

	// The client is constructed ONCE per wall: a route's endpoint and
	// view never change while its wall is mounted (a route change
	// remounts the component), so the props are read through untrack
	// and the client owns exactly this wall for its lifetime.
	const wall = untrack(() => createTimelineClient({ path, view, albumId }));

	// The selection is a reactive id set: checkbox and tile clicks
	// toggle it, Esc and the bar's clear drop it, and the select mode
	// (the touch-friendly entry) keeps the checkboxes revealed.
	const selection = new SvelteSet<string>();
	let selectMode = $state(false);
	let batchErrorKey = $state('');
	let batchBusy = $state(false);

	const groups = $derived(groupTimelineItems($wall.items, $locale ?? 'en'));
	const selectable = $derived(selectMode || selection.size > 0);
	const selectedItems = $derived($wall.items.filter((timelineItem) => selection.has(timelineItem.id)));
	const allSelectedFavorite = $derived(
		selectedItems.length > 0 && selectedItems.every((timelineItem) => timelineItem.isFavorite)
	);
	const firstLoad = $derived($wall.items.length === 0 && $wall.status === 'loading');
	const settledEmpty = $derived(
		$wall.items.length === 0 && $wall.status === 'ready' && !$wall.hasMore
	);
	// The empty state's heading: the album wall passes the album's
	// title as direct text; the curated views pass an i18n key.
	const emptyHeading = $derived(heading !== '' ? heading : titleKey !== '' ? $_(titleKey) : '');

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	function toggleSelected(id: string, next: boolean): void {
		if (next) {
			selection.add(id);
		} else {
			selection.delete(id);
		}
	}

	function exitSelection(): void {
		selection.clear();
		selectMode = false;
	}

	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && (selection.size > 0 || selectMode)) {
			exitSelection();
		}
	}

	async function runAction(action: BatchAction): Promise<void> {
		if (batchBusy || selection.size === 0) {
			return;
		}
		batchBusy = true;
		batchErrorKey = '';
		const outcome = await wall.runBatch(action, [...selection]);
		batchBusy = false;
		if (outcome.ok) {
			// A successful action clears the selection; the explicit
			// select mode persists for the next curation pass.
			selection.clear();
		} else {
			// The failure keeps the selection (a retry affordance) and
			// surfaces the mapped message in the bar.
			batchErrorKey = outcome.errorKey;
		}
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// Publish the loaded page items as the viewer's navigation
	// sequence (see $lib/timeline/wall.svelte): the copy outlives
	// this route, so the /a/{id} viewer walks exactly the items this
	// wall rendered.
	$effect(() => {
		publishWallItems($wall.items);
	});

	// The walk itself: the first load and every resetKey change (the
	// album wall's membership convergence). A reset while a page load
	// is in flight discards the stale response (the client's
	// generation guard), so the fresh walk's answer is the only one
	// that lands.
	$effect(() => {
		void resetKey;
		untrack(() => {
			wall.reset();
			void wall.loadMore();
		});
	});

	// The live surfaces: the SSE subscription and the scroll sentinel
	// observer live for the wall's lifetime.
	$effect(() => {
		const unsubscribe = subscribeAssetEvents((event) => {
			void wall.applyEvent(event);
		});

		let observer: IntersectionObserver | undefined;
		if (observerAvailable && sentinel !== undefined) {
			observer = new IntersectionObserver(
				(entries) => {
					if (entries.some((entry) => entry.isIntersecting)) {
						// The client's own guards (in-flight, exhausted)
						// collapse bursts into one fetch.
						void wall.loadMore();
					}
				},
				{ rootMargin: '600px 0px' }
			);
			observer.observe(sentinel);
		}
		return () => {
			observer?.disconnect();
			unsubscribe();
		};
	});
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<section class="wall max-w-[1200px] mx-auto" aria-busy={$wall.status === 'loading'}>
	{#if $wall.items.length > 0}
		<div class="wall__toolbar flex justify-end mb-2">
			<Button
				variant="ghost"
				aria-pressed={selectMode}
				onclick={() => (selectMode = !selectMode)}
			>
				{selectMode ? $_('action_done') : $_('action_select')}
			</Button>
		</div>
	{/if}

	{#if firstLoad}
		<p class="wall__status my-12 text-center text-text-soft" role="status">{$_(loadingKey)}</p>
	{:else if settledEmpty}
		<div class="wall__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
			<h1 class="text-[22px] mb-2">{emptyHeading}</h1>
			<p class="text-text-soft">{$_(emptyKey)}</p>
		</div>
	{:else if $wall.status === 'error' && $wall.items.length === 0}
		<div class="wall__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($wall.errorKey)}</p>
			<Button variant="primary" onclick={() => void wall.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{:else}
		<!-- The month-grouped justified wall (the shared renderer): the
		     curated views render exactly the timeline's own wall -- the
		     one density language for every photo surface. -->
		<MonthGroups {groups} isSelected={(id) => selection.has(id)} {selectable} onselect={toggleSelected} />
	{/if}

	{#if $wall.status === 'error' && $wall.items.length > 0}
		<div class="wall__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($wall.errorKey)}</p>
			<Button variant="primary" onclick={() => void wall.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{/if}

	{#if !observerAvailable && $wall.hasMore && $wall.status === 'ready'}
		<div class="wall__pager flex justify-center my-5">
			<Button variant="ghost" onclick={() => void wall.loadMore()}>{$_('timeline_load_more')}</Button>
		</div>
	{/if}

	<div class="wall__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
</section>

<BatchBar
	count={selectedItems.length}
	allFavorite={allSelectedFavorite}
	errorKey={batchErrorKey}
	busy={batchBusy}
	{actions}
	onaction={runAction}
	onclear={exitSelection}
/>
