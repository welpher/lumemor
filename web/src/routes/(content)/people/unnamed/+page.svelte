<script lang="ts">
	// The unnamed region of the people surface (the design's action
	// bar, the card-as-entrance): the caller's unlabeled face
	// clusters as a keyset-paginated card grid, largest first, with
	// the region's whole count in the bar.
	//
	// THE CARDS ARE ENTRANCES, NOT A SELECTION SURFACE: no
	// check circles, no armed select mode, no batch bar, no batch
	// merge. A card opens the SAME person detail the named region
	// opens -- `/people/{clusterId}` -- in its unnamed variant
	// (the name slot reads the localized unnamed title and the primary
	// action names the person). The old "select clusters → merge them"
	// path retired with this block; the merge lives on the detail's
	// merge-into action.
	//
	// SSE CONVERGENCE: every person.* event and every face.clustered
	// event converges through ONE replay refetch, collapsed
	// single-flight. The replay re-walks the keyset pages from the
	// first until the loaded window's depth is covered (or the region
	// ends), so a scrolled page neither truncates nor misses the
	// fresh cluster a recognition run announced.
	import { _ } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import { listUnnamedPeople, type UnnamedClusterSummary } from '$lib/api/people';
	import { subscribeFaceEvents, subscribePersonEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import Button from '$lib/components/Button.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ScanFace from '@lucide/svelte/icons/scan-face';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let items = $state<UnnamedClusterSummary[]>([]);
	/** The caller's whole unnamed region (the bar's count). */
	let totalCount = $state(0);
	let status = $state<ListStatus>('idle');
	/** False once the last page arrived (no `nextCursor`). */
	let hasMore = $state(true);
	/** The error-code i18n key of the last failed page load ('' when none). */
	let errorKey = $state('');
	// The continuation cursor of the next page (null while on the
	// first page); never assembled here, only passed back.
	let nextCursor: string | null = null;

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	const firstLoad = $derived(items.length === 0 && status === 'loading');
	const settledEmpty = $derived(items.length === 0 && status === 'ready' && !hasMore);

	/** Fetch the next keyset page (the first page while idle). */
	async function loadMore(): Promise<void> {
		if (status === 'loading') {
			// The in-flight guard: one page fetch per trigger.
			return;
		}
		if (status === 'ready' && !hasMore) {
			return;
		}
		status = 'loading';
		const outcome = await listUnnamedPeople(nextCursor);
		if (!outcome.ok) {
			status = 'error';
			errorKey = outcome.errorKey;
			return;
		}
		// Pages append only ids not already loaded: the keyset is
		// disjoint by construction, but a convergence replay racing
		// this response must not duplicate a card.
		const seen = new Set(items.map((cluster) => cluster.clusterId));
		const appended = outcome.page.items.filter((cluster) => !seen.has(cluster.clusterId));
		items = [...items, ...appended];
		totalCount = outcome.page.totalCount;
		nextCursor = outcome.page.nextCursor;
		hasMore = nextCursor !== null;
		status = 'ready';
		errorKey = '';
	}

	// The single-flight collapse of the convergence replay: while one
	// replay runs, later events park in a queue and one trailing
	// replay runs when it lands.
	let converging = false;
	let convergeQueued = false;

	/**
	 * The convergence entry point behind every SSE event: replay the
	 * keyset pages from the first until the loaded window's depth is
	 * covered (or the region ends) and adopt the collected prefix.
	 * A failure keeps the last good state silently -- the next event
	 * retries.
	 */
	async function runConverge(): Promise<void> {
		if (converging) {
			convergeQueued = true;
			return;
		}
		converging = true;
		try {
			const seen = new Set<string>();
			let collected: UnnamedClusterSummary[] = [];
			let cursor: string | null = null;
			let regionTotal = 0;
			while (true) {
				const outcome = await listUnnamedPeople(cursor);
				if (!outcome.ok) {
					return;
				}
				for (const cluster of outcome.page.items) {
					if (!seen.has(cluster.clusterId)) {
						seen.add(cluster.clusterId);
						collected.push(cluster);
					}
				}
				regionTotal = outcome.page.totalCount;
				cursor = outcome.page.nextCursor;
				// The live window's depth, re-read every lap: a
				// loadMore that appends mid-replay must be covered by
				// the same replay, not truncated by a stale prefix.
				if (cursor === null || collected.length >= items.length) {
					break;
				}
			}
			items = collected;
			totalCount = regionTotal;
			nextCursor = cursor;
			hasMore = cursor !== null;
			if (status === 'error') {
				status = 'ready';
				errorKey = '';
			}
		} finally {
			converging = false;
			if (convergeQueued) {
				convergeQueued = false;
				void runConverge();
			}
		}
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The first page, the SSE subscriptions and the sentinel's
	// observer live for the page's lifetime: the first subscriber of
	// any kind opens the stream, leaving the page closes it. The
	// initial page load runs untracked: its guards read the reactive
	// status flags, and a tracked read would tie this effect to the
	// fetch's own state changes (the effect re-running on every page
	// landing -- the double fetch).
	$effect(() => {
		void untrack(() => loadMore());

		const unsubscribePerson = subscribePersonEvents(() => {
			void runConverge();
		});
		const unsubscribeFace = subscribeFaceEvents(() => {
			void runConverge();
		});

		let observer: IntersectionObserver | undefined;
		if (observerAvailable && sentinel !== undefined) {
			observer = new IntersectionObserver(
				(entries) => {
					if (entries.some((entry) => entry.isIntersecting)) {
						// The page's own guards (in-flight, exhausted)
						// collapse bursts into one fetch.
						void loadMore();
					}
				},
				{ rootMargin: '600px 0px' }
			);
			observer.observe(sentinel);
		}
		return () => {
			observer?.disconnect();
			unsubscribePerson();
			unsubscribeFace();
		};
	});
</script>

<svelte:head>
	<title>{$_('people_unnamed_title')} · lumemor</title>
</svelte:head>

<section class="unnamed max-w-[1200px] mx-auto px-4 pt-4 pb-12" aria-busy={status === 'loading'}>
	<!-- THE ONE ROW: the back control, the title with the
	     region's count -- and no action at all: the cards are the
	     entrances, not a selection surface. -->
	<div class="unnamed__toolbar flex items-center gap-3 pb-4 border-b border-border">
		<a
			class="unnamed__back inline-flex items-center justify-center w-9 h-9 shrink-0 rounded-control bg-transparent text-text-soft no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text"
			href="/people"
			aria-label={$_('people_back')}
		>
			<ChevronLeft size={20} aria-hidden="true" />
		</a>
		<h1 class="unnamed__title m-0 font-display font-bold text-[20px] leading-[1.3] text-text">
			{$_('people_unnamed_title')}
			{#if status === 'ready'}
				<span
					class="unnamed__title-count font-body font-normal text-[15px] leading-[1.4] text-text-soft tabular-nums"
				>
					· {$_('people_unnamed_count', { values: { count: totalCount } })}
				</span>
			{/if}
		</h1>
	</div>

	<div class="unnamed__content pt-4">
		{#if firstLoad}
			<LoadingLine label={$_('people_unnamed_loading')} />
		{:else if settledEmpty}
			<StateCard title={$_('people_unnamed_empty_title')} body={$_('people_unnamed_empty')} />
		{:else if status === 'error' && items.length === 0}
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void loadMore()}
			/>
		{:else}
			<div class="unnamed__grid grid grid-cols-[repeat(auto-fill,minmax(min(160px,100%),1fr))] gap-3">
				{#each items as cluster (cluster.clusterId)}
					<!-- The cluster card: a LINK into the person detail's
					     unnamed variant (the same page the named cards open).
					     No check circle, no pressed state: the card is an
					     entrance. -->
					<a
						class="cluster-card relative flex flex-col overflow-hidden rounded-card bg-card border border-border no-underline text-text cursor-pointer transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard hover:border-accent hover:shadow-card focus-visible:border-accent"
						href="/people/{encodeURIComponent(cluster.clusterId)}?state=unnamed"
					>
						<span class="cluster-card__crop relative block aspect-square bg-accent-soft" aria-hidden="true">
							<img
								class="absolute inset-0 w-full h-full object-cover"
								src={cluster.thumbnail}
								alt=""
								loading="lazy"
								decoding="async"
								draggable="false"
							/>
						</span>
						<span class="cluster-card__meta flex items-center gap-2 px-3 py-2">
							<span
								class="cluster-card__name min-w-0 font-body font-semibold text-[14px] leading-[1.4] text-text truncate"
								>{$_('people_unnamed_title')}</span
							>
							<span
								class="cluster-card__count ml-auto inline-flex items-center gap-1 font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums whitespace-nowrap"
								role="img"
								aria-label={$_('people_face_count', { values: { count: cluster.faceCount } })}
								title={$_('people_face_count', { values: { count: cluster.faceCount } })}
							>
								<ScanFace size={14} aria-hidden="true" />{cluster.faceCount}
							</span>
						</span>
					</a>
				{/each}
			</div>
		{/if}

		{#if status === 'error' && items.length > 0}
			<!-- A failed follow-up page while cards exist: the page stays,
			     the retry re-runs the same cursor (an idempotent position). -->
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void loadMore()}
			/>
		{/if}

		{#if !observerAvailable && hasMore && status === 'ready'}
			<div class="unnamed__pager flex justify-center my-5">
				<Button variant="ghost" onclick={() => void loadMore()}
					><ChevronsDown size={20} aria-hidden="true" />{$_('timeline_load_more')}</Button
				>
			</div>
		{/if}

		<div class="unnamed__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
	</div>
</section>
