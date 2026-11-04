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

<section class="timeline max-w-[1200px] mx-auto" aria-busy={$timeline.status === 'loading'}>
	<!-- The one toolbar of the wall: the hidden view's entry point
	     (the IA's timeline-header entry, a lightweight link while the
	     dropdown pattern it names waits for the settings surface of
	     M3) and, once tiles exist, the selection-mode toggle. -->
	<div class="timeline__toolbar flex justify-end items-center gap-3 mb-2">
		<a
			class="timeline__hidden-link mr-auto px-2.5 py-1.5 font-body font-semibold text-[14px] leading-[1.4] text-text-soft no-underline rounded-control transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/hidden"
		>{$_('hidden_title')}</a>
		{#if $timeline.items.length > 0}
			<Button
				variant="ghost"
				aria-pressed={selectMode}
				onclick={() => (selectMode = !selectMode)}
			>
				{selectMode ? $_('action_done') : $_('action_select')}
			</Button>
		{/if}
	</div>

	{#if firstLoad}
		<p class="timeline__status my-12 text-center text-text-soft" role="status">{$_('timeline_loading')}</p>
	{:else if settledEmpty}
		<div class="timeline__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
			<h1 class="text-[22px] mb-2">{$_('nav_timeline')}</h1>
			<p class="text-text-soft">{$_('timeline_empty')}</p>
		</div>
	{:else if $timeline.status === 'error' && $timeline.items.length === 0}
		<div class="timeline__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($timeline.errorKey)}</p>
			<Button variant="primary" onclick={() => void timeline.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{:else}
		{#each groups as group (group.key)}
			<!-- Both sticky headers ride above the tiles; the section's
			     scroll margin leaves room for them under the scrubber's
			     jumps. -->
			<section class="timeline__month scroll-mt-[92px]" data-month-key={group.key}>
				<h2 class="timeline__month-header sticky top-0 z-[10] mt-6 mb-3 py-2.5 px-3 text-[20px] bg-[color-mix(in_srgb,var(--color-page)_88%,transparent)] backdrop-blur-[6px] rounded-control">{group.label}</h2>
				{#each group.days as day (day.key)}
					<div class="timeline__day" data-day-key={day.key}>
						<!-- Stacks under the month header (44px + its margins). -->
						<h3 class="timeline__day-header sticky top-[44px] z-[9] mt-4 mb-2 py-1 px-2 font-body font-semibold text-[13px] tracking-[0.02em] text-text-soft bg-[color-mix(in_srgb,var(--color-page)_88%,transparent)] backdrop-blur-[6px] rounded-control">{day.label}</h3>
						<div class="timeline__wall grid grid-cols-[repeat(auto-fill,minmax(min(240px,100%),1fr))] gap-3">
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
		<div class="timeline__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
			<p>{errorText($timeline.errorKey)}</p>
			<Button variant="primary" onclick={() => void timeline.loadMore()}>{$_('action_retry')}</Button>
		</div>
	{/if}

	{#if !observerAvailable && $timeline.hasMore && $timeline.status === 'ready'}
		<div class="timeline__pager flex justify-center my-5">
			<Button variant="ghost" onclick={() => void timeline.loadMore()}>{$_('timeline_load_more')}</Button>
		</div>
	{/if}

	<div class="timeline__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
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
