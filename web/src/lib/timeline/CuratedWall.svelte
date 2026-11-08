<script lang="ts">
	// The curated wall wrapper the favorites and hidden pages
	// compose (M2d Task 8): the page-level concerns of a membership
	// view -- the pagination client (driven with the view's
	// membership semantics), the SSE convergence, the sentinel walk,
	// the selection state with its batch bar, the wallItems
	// publication the asset viewer navigates -- assembled around the
	// shared Wall's pure rendering. The page tells the wall its
	// endpoint, its view, its strings and its action subset; the
	// RENDERING is the one Wall component every photo surface
	// composes (the month axis is the wall's own organ -- the
	// 2026-11-08 pin -- so it rides along here exactly as it does
	// on the timeline; the album detail owns its own no-selection
	// walk). The wall here IS selectable: the tiles' check
	// affordance and the toolbar's touch-friendly select-mode toggle
	// enter the curation pass whose actions end rows' memberships.
	import { _ } from 'svelte-i18n';
	import { SvelteSet } from 'svelte/reactivity';
	import { untrack } from 'svelte';
	import { createTimelineClient, type BatchAction, type WallView } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import BatchBar from '$lib/timeline/BatchBar.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import Button from '$lib/components/Button.svelte';
	import ListChecks from '@lucide/svelte/icons/list-checks';
	import Check from '@lucide/svelte/icons/check';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	type Props = {
		/** The timeline-shaped endpoint the wall walks. */
		path: string;
		/** Which membership view the client serves (see WallView). */
		view: WallView;
		/** The i18n key of the view's name (the empty state's heading). */
		titleKey?: string;
		/** The i18n key of the view's loading line. */
		loadingKey: string;
		/** The i18n key of the view's empty state text. */
		emptyKey: string;
		/** Which batch actions the selection bar offers (see BatchBar). */
		actions: readonly BatchAction[];
	};

	let {
		path,
		view,
		titleKey = '',
		loadingKey,
		emptyKey,
		actions
	}: Props = $props();

	// The client is constructed ONCE per wall: a route's endpoint and
	// view never change while its wall is mounted (a route change
	// remounts the component), so the props are read through untrack
	// and the client owns exactly this wall for its lifetime.
	const wall = untrack(() => createTimelineClient({ path, view }));

	// The selection is a reactive id set: checkbox and tile clicks
	// toggle it, Esc and the bar's clear drop it, and the select mode
	// (the touch-friendly entry) keeps the checkboxes revealed.
	const selection = new SvelteSet<string>();
	let selectMode = $state(false);
	let batchErrorKey = $state('');
	let batchBusy = $state(false);

	const selectable = $derived(selectMode || selection.size > 0);
	const selectedItems = $derived($wall.items.filter((timelineItem) => selection.has(timelineItem.id)));
	const allSelectedFavorite = $derived(
		selectedItems.length > 0 && selectedItems.every((timelineItem) => timelineItem.isFavorite)
	);
	const firstLoad = $derived($wall.items.length === 0 && $wall.status === 'loading');
	const settledEmpty = $derived(
		$wall.items.length === 0 && $wall.status === 'ready' && !$wall.hasMore
	);

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

	// The walk itself: the first load on mount. A reset while a page
	// load is in flight discards the stale response (the client's
	// generation guard), so the fresh walk's answer is the only one
	// that lands.
	$effect(() => {
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

<section class="wall px-2" aria-busy={$wall.status === 'loading'}>
	{#if $wall.items.length > 0}
		<div class="wall__toolbar flex justify-end mb-2">
			<!-- The touch-friendly select-mode entry (icon-only, the
			     icon-first rule; the check affordance on the tiles is
			     the pointer/keyboard entry). -->
			<IconButton
				aria-pressed={selectMode}
				aria-label={selectMode ? $_('action_done') : $_('action_select')}
				onclick={() => (selectMode = !selectMode)}
			>
				{#if selectMode}
					<Check size={20} aria-hidden="true" />
				{:else}
					<ListChecks size={20} aria-hidden="true" />
				{/if}
			</IconButton>
		</div>
	{/if}

	{#if firstLoad}
		<p class="wall__status my-12 text-center text-text-soft animate-loading-in" role="status">{$_(loadingKey)}</p>
	{:else if settledEmpty}
		<div class="wall__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
			<h1 class="text-[22px] mb-2">{titleKey !== '' ? $_(titleKey) : ''}</h1>
			<p class="text-text-soft">{$_(emptyKey)}</p>
		</div>
	{:else if $wall.status === 'error' && $wall.items.length === 0}
		<div class="wall__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($wall.errorKey)}</p>
			<Button variant="primary" onclick={() => void wall.loadMore()}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{:else}
		<!-- The unified wall (pure rendering): this wrapper owns the
		     walk and the selection around it; the groups, the sticky
		     headers and the packed rows are the one wall component
		     every view renders. -->
		<Wall
			items={$wall.items}
			{selectable}
			isSelected={(id) => selection.has(id)}
			onselect={toggleSelected}
		/>
	{/if}

	{#if $wall.status === 'error' && $wall.items.length > 0}
		<div class="wall__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($wall.errorKey)}</p>
			<Button variant="primary" onclick={() => void wall.loadMore()}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
		</div>
	{/if}

	{#if !observerAvailable && $wall.hasMore && $wall.status === 'ready'}
		<div class="wall__pager flex justify-center my-5">
			<Button variant="ghost" onclick={() => void wall.loadMore()}><ChevronsDown size={20} aria-hidden="true" />{$_('timeline_load_more')}</Button>
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
