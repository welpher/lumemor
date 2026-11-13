<script lang="ts">
	// The curated wall wrapper the favorites and hidden pages
	// compose: the page-level concerns of a membership
	// view -- the pagination client (driven with the view's
	// membership semantics), the SSE convergence, the sentinel walk,
	// the page action bar with its selection switch, the wallItems
	// publication the asset viewer navigates -- assembled around the
	// shared Wall. The RENDERING is the one Wall component every
	// photo surface composes, and so is the SELECTION FLOW: the month axis, the selection
	// state, the checks, the Escape exit and the batch bar all ride
	// the wall here exactly as they do on the timeline. This wrapper
	// tells the wall its endpoint, its view, its strings and its
	// action subset, runs the batch writes the wall awaits (through
	// `onbatch`), and owns the page bar: the title with its count on
	// the left, the selection switch at the far right. That switch is
	// the wall's bound select mode, and the wall keeps the state
	// machine's other half (disarming clears the selection). The
	// memory picker's opener is a passthrough:
	// a page that offers the wall-selection entry wires it, a page
	// that does not (the trash view owns no wall at all) never
	// renders the affordance. The album detail owns its own no-selection
	// walk (no onbatch wired, pure navigation).
	import { _ } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import { createTimelineClient, type BatchAction, type WallView } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import Button from '$lib/components/Button.svelte';
	import ListChecks from '@lucide/svelte/icons/list-checks';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	type Props = {
		/** The timeline-shaped endpoint the wall walks. */
		path: string;
		/** Which membership view the client serves (see WallView). */
		view: WallView;
		/** The i18n key of the view's name (the page bar's title). */
		titleKey?: string;
		/** The i18n key of the page bar's count segment. */
		countKey: string;
		/** The i18n key of the view's loading line. */
		loadingKey: string;
		/** The i18n key of the view's empty state text. */
		emptyKey: string;
		/** Which batch actions the wall's bar offers (see BatchBar). */
		actions: readonly BatchAction[];
		/**
		 * Open the memory picker over the wall's captured selection
		 * (the wall-selection entry, the page's dialog flow). Absent
		 * = the bar carries no memory affordance.
		 */
		onaddmemory?: (ids: readonly string[], done: () => void) => void;
	};

	let { path, view, titleKey = '', countKey, loadingKey, emptyKey, actions, onaddmemory }: Props = $props();

	// The client is constructed ONCE per wall: a route's endpoint and
	// view never change while its wall is mounted (a route change
	// remounts the component), so the props are read through untrack
	// and the client owns exactly this wall for its lifetime.
	const wall = untrack(() => createTimelineClient({ path, view }));

	// The wall's bound select mode: the toolbar's toggle below writes
	// it (the touch-friendly entry) and the wall owns the drops
	// (Escape, the bar's clear) -- the binding carries both
	// directions.
	let selectMode = $state(false);

	const firstLoad = $derived($wall.items.length === 0 && $wall.status === 'loading');
	const settledEmpty = $derived($wall.items.length === 0 && $wall.status === 'ready' && !$wall.hasMore);

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	// The wall's batch runner: this wrapper's walk owns the write
	// (the client's view-aware optimistic convergence -- unfavorite
	// on favorites, unhide on hidden ends the selected rows'
	// memberships); the wall owns the busy/error/clear choreography
	// around the promise it awaits here.
	async function runBatch(action: BatchAction, ids: readonly string[]) {
		return wall.runBatch(action, [...ids]);
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

<section class="wall-page" aria-busy={$wall.status === 'loading'}>
	<!-- THE ONE ROW (the page action bar): the page title with its
	     count on the left, the selection switch at the far right
	     (icon-only, the page's own action), one divider under the
	     row. The count renders only on a settled answer. The switch
	     is the REAL state machine: it arms the wall's select mode
	     (every check revealed), and disarming it drops the selection
	     (the batch bar goes with it) -- the wall owns both
	     directions through the binding. -->
	<div class="wall-page__toolbar flex items-center gap-3 px-4 pt-4 pb-4 border-b border-border">
		<h1 class="wall-page__title m-0 font-display font-bold text-[20px] leading-[1.3] text-text">
			{titleKey !== '' ? $_(titleKey) : ''}
			{#if $wall.status === 'ready'}
				<span
					class="wall-page__title-count font-body font-normal text-[15px] leading-[1.4] text-text-soft tabular-nums"
				>
					· {$_(countKey, { values: { count: $wall.items.length } })}
				</span>
			{/if}
		</h1>
		{#if $wall.items.length > 0}
			<div class="wall-page__actions ml-auto flex items-center gap-2">
				<IconButton
					class="aria-pressed:bg-accent-soft aria-pressed:text-accent-soft-text"
					aria-pressed={selectMode}
					aria-label={$_('action_select')}
					title={$_('action_select')}
					onclick={() => (selectMode = !selectMode)}
				>
					<ListChecks size={20} aria-hidden="true" />
				</IconButton>
			</div>
		{/if}
	</div>

	<!-- The wall's reserved month-axis slot (the wall is
	     full width + 8px margins + the 72px axis strip): the photos and
	     this wrapper's own states center in the width LEFT of the axis,
	     never under it. The page bar above keeps the page's own
	     full-width inset -- the bar is chrome, not wall. -->
	<div class="wall-page__content px-2 pe-[72px] pt-4">
		{#if firstLoad}
			<p class="wall__status my-12 text-center text-text-soft animate-loading-in" role="status">
				{$_(loadingKey)}
			</p>
		{:else if settledEmpty}
			<div
				class="wall__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
			>
				<p class="wall__empty-title mb-2 font-display font-bold text-[16px] leading-[1.3] text-text">
					{titleKey !== '' ? $_(titleKey) : ''}
				</p>
				<p class="text-text-soft">{$_(emptyKey)}</p>
			</div>
		{:else if $wall.status === 'error' && $wall.items.length === 0}
			<div
				class="wall__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
				role="alert"
			>
				<p>{errorText($wall.errorKey)}</p>
				<Button variant="primary" onclick={() => void wall.loadMore()}
					><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
				>
			</div>
		{:else}
			<!-- The unified wall (rendering AND selection): this wrapper
			     owns the walk and the bar above it; the groups, the
			     sticky headers, the packed rows, the selection state and
			     the batch bar are the one wall component every curatable
			     view renders. The action subset and the batch runner are
			     this view's own. -->
			<Wall items={$wall.items} bind:selectMode {actions} onbatch={runBatch} {onaddmemory} />
		{/if}

		{#if $wall.status === 'error' && $wall.items.length > 0}
			<div
				class="wall__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
				role="alert"
			>
				<p>{errorText($wall.errorKey)}</p>
				<Button variant="primary" onclick={() => void wall.loadMore()}
					><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
				>
			</div>
		{/if}

		{#if !observerAvailable && $wall.hasMore && $wall.status === 'ready'}
			<div class="wall__pager flex justify-center my-5">
				<Button variant="ghost" onclick={() => void wall.loadMore()}
					><ChevronsDown size={20} aria-hidden="true" />{$_('timeline_load_more')}</Button
				>
			</div>
		{/if}

		<div class="wall__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
	</div>
</section>
