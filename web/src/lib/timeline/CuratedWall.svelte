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
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { SvelteSet } from 'svelte/reactivity';
	import { untrack } from 'svelte';
	import { createTimelineClient, type BatchAction, type WallView } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import { groupTimelineItems } from '$lib/timeline/grouping';
	import { publishWallItems } from '$lib/timeline/wall.svelte';
	import PhotoTile from '$lib/timeline/PhotoTile.svelte';
	import BatchBar from '$lib/timeline/BatchBar.svelte';
	import Button from '$lib/components/Button.svelte';

	type Props = {
		/** The timeline-shaped endpoint the wall walks. */
		path: string;
		/** Which membership view the client serves (see WallView). */
		view: WallView;
		/** The i18n key of the view's name (the empty state's heading). */
		titleKey: string;
		/** The i18n key of the view's loading line. */
		loadingKey: string;
		/** The i18n key of the view's empty state text. */
		emptyKey: string;
		/** Which batch actions the selection bar offers (see BatchBar). */
		actions: readonly BatchAction[];
	};

	let { path, view, titleKey, loadingKey, emptyKey, actions }: Props = $props();

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

	$effect(() => {
		void wall.loadMore();

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

<section class="wall" aria-busy={$wall.status === 'loading'}>
	{#if $wall.items.length > 0}
		<div class="wall__toolbar">
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
		<p class="wall__status" role="status">{$_(loadingKey)}</p>
	{:else if settledEmpty}
		<div class="wall__empty">
			<h1>{$_(titleKey)}</h1>
			<p>{$_(emptyKey)}</p>
		</div>
	{:else if $wall.status === 'error' && $wall.items.length === 0}
		<div class="wall__failure" role="alert">
			<p>{errorText($wall.errorKey)}</p>
			<Button variant="primary" onclick={() => void wall.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{:else}
		{#each groups as group (group.key)}
			<section class="wall__month" data-month-key={group.key}>
				<h2 class="wall__month-header">{group.label}</h2>
				{#each group.days as day (day.key)}
					<div class="wall__day" data-day-key={day.key}>
						<h3 class="wall__day-header">{day.label}</h3>
						<div class="wall__grid">
							{#each day.items as tileItem (tileItem.id)}
								<PhotoTile
									item={tileItem}
									selected={selection.has(tileItem.id)}
									selectable={selectable}
									onselect={toggleSelected}
								/>
							{/each}
						</div>
					</div>
				{/each}
			</section>
		{/each}
	{/if}

	{#if $wall.status === 'error' && $wall.items.length > 0}
		<div class="wall__failure" role="alert">
			<p>{errorText($wall.errorKey)}</p>
			<Button variant="primary" onclick={() => void wall.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{/if}

	{#if !observerAvailable && $wall.hasMore && $wall.status === 'ready'}
		<div class="wall__pager">
			<Button variant="ghost" onclick={() => void wall.loadMore()}>{$_('timeline_load_more')}</Button>
		</div>
	{/if}

	<div class="wall__sentinel" bind:this={sentinel} aria-hidden="true"></div>
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

<style>
	.wall {
		max-width: 1200px;
		margin: 0 auto;
	}

	.wall__toolbar {
		display: flex;
		justify-content: flex-end;
		margin-bottom: 8px;
	}

	.wall__status {
		margin: 48px 0;
		text-align: center;
		color: var(--color-text-soft);
	}

	.wall__empty {
		max-width: 420px;
		margin: 48px auto;
		padding: 32px 24px;
		text-align: center;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.wall__empty h1 {
		font-size: 22px;
		margin-bottom: 8px;
	}

	.wall__empty p {
		margin: 0;
		color: var(--color-text-soft);
	}

	.wall__failure {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		max-width: 420px;
		margin: 48px auto;
		padding: 20px;
		text-align: center;
		border: 1px solid var(--color-danger);
		border-radius: var(--radius-card);
		color: var(--color-danger-text);
	}

	.wall__failure p {
		margin: 0;
	}

	.wall__month {
		scroll-margin-top: 92px;
	}

	.wall__month-header {
		position: sticky;
		top: 0;
		z-index: 10;
		margin: 24px 0 12px;
		padding: 10px 12px;
		font-size: 20px;
		background-color: color-mix(in srgb, var(--color-page) 88%, transparent);
		backdrop-filter: blur(6px);
		border-radius: var(--radius-control);
	}

	.wall__day-header {
		position: sticky;
		/* Stacks under the month header (44px + its margins). */
		top: 44px;
		z-index: 9;
		margin: 16px 0 8px;
		padding: 4px 8px;
		font-family: var(--font-body);
		font-weight: 600;
		font-size: 13px;
		letter-spacing: 0.02em;
		color: var(--color-text-soft);
		background-color: color-mix(in srgb, var(--color-page) 88%, transparent);
		backdrop-filter: blur(6px);
		border-radius: var(--radius-control);
	}

	.wall__grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(240px, 100%), 1fr));
		gap: 12px;
	}

	.wall__pager {
		display: flex;
		justify-content: center;
		margin: 20px 0;
	}

	.wall__sentinel {
		height: 1px;
	}
</style>
