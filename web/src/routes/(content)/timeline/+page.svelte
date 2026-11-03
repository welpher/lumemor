<script lang="ts">
	// The timeline wall, the app's home surface: keyset pages of
	// tiles stream in through the pagination client (an
	// IntersectionObserver sentinel over a 600px prefetch margin, the
	// client's in-flight guard collapsing every burst into one
	// fetch), month and day groups carry sticky headers, the right
	// rail scrubs the loaded months, and selection turns the wall
	// into a curation surface with the floating batch bar.
	//
	// The SSE subscription lives for the page's lifetime: the first
	// subscriber opens the stream, leaving the page closes it, and
	// every asset event converges through the client. Render
	// windowing of v0 is the tiles' content-visibility (see
	// PhotoTile); full virtualization is an M3 optimization.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { SvelteSet } from 'svelte/reactivity';
	import { createTimelineClient, type BatchAction } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import { groupTimelineItems } from '$lib/timeline/grouping';
	import { publishWallItems } from '$lib/timeline/wall.svelte';
	import PhotoTile from '$lib/timeline/PhotoTile.svelte';
	import MonthScrubber from '$lib/timeline/MonthScrubber.svelte';
	import BatchBar from '$lib/timeline/BatchBar.svelte';
	import Button from '$lib/components/Button.svelte';

	const timeline = createTimelineClient();

	// The selection is a reactive id set: checkbox and tile clicks
	// toggle it, Esc and the bar's clear drop it, and the select mode
	// (the touch-friendly entry) keeps the checkboxes revealed.
	const selection = new SvelteSet<string>();
	let selectMode = $state(false);
	let batchErrorKey = $state('');
	let batchBusy = $state(false);
	let activeMonthKey = $state<string | null>(null);

	const groups = $derived(groupTimelineItems($timeline.items, $locale ?? 'en'));
	const months = $derived(groups.map((group) => ({ key: group.key, label: group.label })));
	const selectable = $derived(selectMode || selection.size > 0);
	const selectedItems = $derived($timeline.items.filter((timelineItem) => selection.has(timelineItem.id)));
	const allSelectedFavorite = $derived(
		selectedItems.length > 0 && selectedItems.every((timelineItem) => timelineItem.isFavorite)
	);
	const firstLoad = $derived($timeline.items.length === 0 && $timeline.status === 'loading');
	const settledEmpty = $derived(
		$timeline.items.length === 0 && $timeline.status === 'ready' && !$timeline.hasMore
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
		const outcome = await timeline.runBatch(action, [...selection]);
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

	/** Scroll one month group into view (the scrubber jump target). */
	function jumpToMonth(key: string): void {
		const element = document.querySelector<HTMLElement>(`[data-month-key="${key}"]`);
		if (element !== null && typeof element.scrollIntoView === 'function') {
			element.scrollIntoView({ block: 'start', behavior: 'smooth' });
		}
	}

	let scrollScheduled = false;

	/** Track which month the viewport reads, for the scrubber's stop. */
	function updateActiveMonth(): void {
		const sections = Array.from(document.querySelectorAll<HTMLElement>('[data-month-key]'));
		if (sections.length === 0) {
			return;
		}
		const first = sections[0]?.getBoundingClientRect();
		// A layout-less host (tests) answers zero rects: keep no stop.
		if (first === undefined || (first.top === 0 && first.height === 0)) {
			return;
		}
		const anchor = 100;
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

	function handleWindowScroll(): void {
		if (scrollScheduled) {
			return;
		}
		scrollScheduled = true;
		requestAnimationFrame(() => {
			scrollScheduled = false;
			updateActiveMonth();
		});
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
		publishWallItems($timeline.items);
	});

	$effect(() => {
		void timeline.loadMore();

		const unsubscribe = subscribeAssetEvents((event) => {
			void timeline.applyEvent(event);
		});

		let observer: IntersectionObserver | undefined;
		if (observerAvailable && sentinel !== undefined) {
			observer = new IntersectionObserver(
				(entries) => {
					if (entries.some((entry) => entry.isIntersecting)) {
						// The client's own guards (in-flight, exhausted)
						// collapse bursts into one fetch.
						void timeline.loadMore();
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

<svelte:window onkeydown={handleWindowKeydown} onscroll={handleWindowScroll} />

<svelte:head>
	<title>Timeline · lumemor</title>
</svelte:head>

<section class="timeline" aria-busy={$timeline.status === 'loading'}>
	{#if $timeline.items.length > 0}
		<div class="timeline__toolbar">
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
		<p class="timeline__status" role="status">{$_('timeline_loading')}</p>
	{:else if settledEmpty}
		<div class="timeline__empty">
			<h1>{$_('nav_timeline')}</h1>
			<p>{$_('timeline_empty')}</p>
		</div>
	{:else if $timeline.status === 'error' && $timeline.items.length === 0}
		<div class="timeline__failure" role="alert">
			<p>{errorText($timeline.errorKey)}</p>
			<Button variant="primary" onclick={() => void timeline.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{:else}
		{#each groups as group (group.key)}
			<section class="timeline__month" data-month-key={group.key}>
				<h2 class="timeline__month-header">{group.label}</h2>
				{#each group.days as day (day.key)}
					<div class="timeline__day" data-day-key={day.key}>
						<h3 class="timeline__day-header">{day.label}</h3>
						<div class="timeline__wall">
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

	{#if $timeline.status === 'error' && $timeline.items.length > 0}
		<div class="timeline__failure" role="alert">
			<p>{errorText($timeline.errorKey)}</p>
			<Button variant="primary" onclick={() => void timeline.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{/if}

	{#if !observerAvailable && $timeline.hasMore && $timeline.status === 'ready'}
		<div class="timeline__pager">
			<Button variant="ghost" onclick={() => void timeline.loadMore()}>{$_('timeline_load_more')}</Button>
		</div>
	{/if}

	<div class="timeline__sentinel" bind:this={sentinel} aria-hidden="true"></div>
</section>

<MonthScrubber {months} activeKey={activeMonthKey} onjump={jumpToMonth} />
<BatchBar
	count={selectedItems.length}
	allFavorite={allSelectedFavorite}
	errorKey={batchErrorKey}
	busy={batchBusy}
	onaction={runAction}
	onclear={exitSelection}
/>

<style>
	.timeline {
		max-width: 1200px;
		margin: 0 auto;
	}

	.timeline__toolbar {
		display: flex;
		justify-content: flex-end;
		margin-bottom: 8px;
	}

	.timeline__status {
		margin: 48px 0;
		text-align: center;
		color: var(--color-text-soft);
	}

	.timeline__empty {
		max-width: 420px;
		margin: 48px auto;
		padding: 32px 24px;
		text-align: center;
		background-color: var(--color-card);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-card);
		box-shadow: var(--shadow-card);
	}

	.timeline__empty h1 {
		font-size: 22px;
		margin-bottom: 8px;
	}

	.timeline__empty p {
		margin: 0;
		color: var(--color-text-soft);
	}

	.timeline__failure {
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

	.timeline__failure p {
		margin: 0;
	}

	.timeline__month {
		/* Both sticky headers ride above the tiles; the section's
		 * scroll margin leaves room for them under the scrubber's
		 * jumps. */
		scroll-margin-top: 92px;
	}

	.timeline__month-header {
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

	.timeline__day-header {
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

	.timeline__wall {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(240px, 100%), 1fr));
		gap: 12px;
	}

	.timeline__pager {
		display: flex;
		justify-content: center;
		margin: 20px 0;
	}

	.timeline__sentinel {
		height: 1px;
	}
</style>
