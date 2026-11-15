<script lang="ts">
	// The search page: one clean result surface. Its own context bar
	// (the shell's topbar and drawer hide while this route is active)
	// carries the back link, the page title, the honest count and the
	// ONE search field, dead center: the bar is the three-column row
	// `1fr · 520px · 1fr` -- the identity on the left, the field in
	// the middle, the right column deliberately empty -- so the
	// field's center sits at the viewport's center at every width.
	// The body below is the results wall alone; the query IS the
	// filter, so no facet affordance rides this page (narrowing is
	// editing the query, not decorating it).
	//
	// THE URL IS THE STATE (the pinned rule): the query and any
	// deep-linked filter selection live in the /search URL's
	// parameters, so the page is deep-linkable and the topbar's box
	// (wherever the shell bar still renders) navigates here with
	// `query=`. Edits on this page (a query submit) REPLACE the URL
	// -- the URL is the app state, and the history stays one clean
	// step back to whatever surface led here.
	//
	// THE TWO RESULT STATES (the pinned rule): a query whose trimmed
	// text spans two or more runes is QUERY state -- the flat
	// justified wall, rank order preserved (no month/day grouping:
	// grouping would lie about the rank order), the count wording
	// "N matches". Anything less takes NO text path (the server's own
	// rule, mirrored here), the request is the pure browse, and the
	// wall renders the timeline's exact month-grouped day-block form
	// -- the browse state IS the recent-photos feed: an empty query
	// shows photos, never the no-results card (that card is for a
	// query that answered nothing), with the one hint line above the
	// wall saying so. Both states paginate through the mode-tagged
	// cursor (the client's walk).
	//
	// THE SSE CONVERGENCE (the pinned semantics): every asset.*
	// event re-executes the current search -- the refetch IS the
	// convergence, no patching; the fresh page owns the truth.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { untrack } from 'svelte';
	import {
		createSearchClient,
		filtersFromSearchParams,
		isQueryMode,
		searchParamsFromFilters,
		type SearchFilters
	} from '$lib/api/search';
	import { addTagAssets, createTag, listTags, type TagSummary } from '$lib/api/tags';
	import type { BatchAction, BatchOutcome } from '$lib/api/timeline';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import FlatWall from '$lib/timeline/FlatWall.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import AddTagDialog from '$lib/tags/AddTagDialog.svelte';
	import AddToMemoryDialog from '$lib/memories/AddToMemoryDialog.svelte';
	import Button from '$lib/components/Button.svelte';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import Search from '@lucide/svelte/icons/search';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	const search = createSearchClient();

	// THE WALLS' SELECTION (the curation ruling: BOTH search walls
	// carry the per-user subset, every role): the batch runner over
	// the client (the optimistic favorite flip and its rollback live
	// there; the visible-domain rule -- one union for everyone,
	// the owner resolution per caller -- is the endpoint's own) and
	// the add-tag dialog flow the walls' openers feed (the captured
	// ids, the done continuation). The bar renders the subset: the
	// favorite toggle and the tag affordance; hide/unhide/trash/
	// restore stay off -- ownership actions belong to the surfaces
	// that own them.
	const BATCH_ACTIONS: readonly BatchAction[] = ['favorite'];

	// The add-tag flow's state (the timeline page's pattern): the
	// vocabulary fetched on open, the honest report of the last
	// landed add, and the captured-ids + done-continuation pair the
	// walls' openers hand over. Tag writes are per-user state and
	// publish nothing, so the report is the outcome and the wall
	// never converges.
	let tagIds: readonly string[] = [];
	let tagDone: (() => void) | undefined;
	let tagOpen = $state(false);
	let tagBusy = $state(false);
	let tagErrorKey = $state('');
	let tagList = $state<TagSummary[]>([]);
	let tagLoading = $state(false);
	let tagReport = $state<{ added: number; skipped: number } | null>(null);

	// The add-to-memory flow's state: the same captured-ids +
	// done-continuation pair the walls' openers hand over -- the
	// dialog owns the flow's own state and writes.
	let memoryIds = $state<readonly string[]>([]);
	let memoryDone: (() => void) | undefined;
	let memoryOpen = $state(false);

	/** The walls' batch runner: the client owns the write. */
	function runBatch(action: BatchAction, ids: readonly string[]): Promise<BatchOutcome> {
		return search.runBatch(action, [...ids]);
	}

	/** Open the add-tag dialog over the wall's captured ids. */
	function openTagDialog(ids: readonly string[], done: () => void): void {
		tagIds = ids;
		tagDone = done;
		tagErrorKey = '';
		tagReport = null;
		tagOpen = true;
		tagList = [];
		tagLoading = true;
		void (async () => {
			const outcome = await listTags();
			if (!tagOpen) {
				return;
			}
			tagLoading = false;
			if (outcome.ok) {
				tagList = outcome.tags;
			} else {
				tagErrorKey = outcome.errorKey;
			}
		})();
	}

	// The existing-tag path: the honest {added, skipped} report of
	// the write's own answer lands in the dialog.
	async function addToExistingTag(tagId: string): Promise<void> {
		if (tagBusy || tagIds.length === 0) {
			return;
		}
		tagBusy = true;
		tagErrorKey = '';
		const outcome = await addTagAssets(tagId, [...tagIds]);
		tagBusy = false;
		if (!outcome.ok) {
			tagErrorKey = outcome.errorKey;
			return;
		}
		tagReport = outcome.report;
	}

	// The new-tag path: create the tag (idempotently), then tag the
	// captured ids with it.
	async function addToNewTag(name: string): Promise<void> {
		if (tagBusy || tagIds.length === 0) {
			return;
		}
		tagBusy = true;
		tagErrorKey = '';
		const created = await createTag(name);
		if (!created.ok) {
			tagBusy = false;
			tagErrorKey = created.errorKey;
			return;
		}
		const outcome = await addTagAssets(created.tag.id, [...tagIds]);
		tagBusy = false;
		if (!outcome.ok) {
			tagErrorKey = outcome.errorKey;
			return;
		}
		tagReport = outcome.report;
	}

	// The memory picker's opener (the wall's captured ids and done
	// continuation ride the dialog); the pass's landing fires the
	// success arm -- the wall clears its selection, no convergence
	// follows (adding to a memory changes no photo's state).
	function openMemoryDialog(ids: readonly string[], done: () => void): void {
		memoryIds = ids;
		memoryDone = done;
		memoryOpen = true;
	}

	function finishMemory(): void {
		memoryDone?.();
		memoryDone = undefined;
	}

	// The tag report's done: close the dialog and fire the wall's
	// done continuation -- the caller's own tags landed, no wall
	// convergence follows (per-user state, silent by contract), and
	// the wall clears its selection (the success arm).
	function finishTag(): void {
		tagOpen = false;
		tagReport = null;
		tagDone?.();
		tagDone = undefined;
	}

	// The URL state: the page derives its filters off the route URL,
	// and the run effect re-executes the search whenever the URL
	// (or the locale, whose display names ride the request)
	// changes. The query box's draft seeds from the URL and survives
	// typing (an effect write, never a bind onto the derived).
	const filters = $derived(filtersFromSearchParams(page.url.searchParams));
	const queryMode = $derived(isQueryMode(filters.query));
	let draft = $state('');

	$effect(() => {
		const current = filters;
		const activeLocale = $locale ?? 'en';
		untrack(() => {
			draft = current.query;
			void search.run(current, activeLocale);
		});
	});

	// The browse state's month/day fold lives inside the shared Wall
	// (the one renderer every photo surface composes); QUERY state
	// keeps the flat order. BOTH walls curate the per-user subset
	// (the curation ruling): the batch bar's favorite toggle + the
	// tag affordance, every role -- the endpoint's owner resolution
	// applies to the admin too, and the honest missing report
	// handles the edge.

	const firstLoad = $derived($search.items.length === 0 && $search.status === 'loading');
	const settledEmpty = $derived(
		$search.items.length === 0 && $search.status === 'ready' && $search.nextCursor === null
	);

	// IntersectionObserver exists in every browser jsdom is not; the
	// manual load-more button is the fallback surface (and the error
	// retry lives there too, because an observer that never
	// re-intersects cannot unstick an error).
	const observerAvailable = typeof IntersectionObserver === 'function';
	let sentinel: HTMLElement | undefined = $state();

	/**
	 * The count's wording: query state says "N matches", browse says
	 * "All N photos" (the browse feed is the whole library, not a
	 * filtered set); while a page follows, the loaded count is what
	 * is honest, so the "so far" arm takes over until the walk
	 * exhausts. The count follows the query -- it is never a stale
	 * total from a previous answer.
	 */
	const countLabel = $derived.by(() => {
		const count = $search.items.length;
		if ($search.nextCursor !== null) {
			return queryMode
				? $_('search_match_count_so_far', { values: { count } })
				: $_('search_browse_count_so_far', { values: { count } });
		}
		return queryMode
			? $_('search_match_count', { values: { count } })
			: $_('search_browse_count', { values: { count } });
	});

	/** Replace the URL with the next filter state (the deep-link). */
	function navigate(next: SearchFilters): void {
		const params = searchParamsFromFilters(next).toString();
		goto(params === '' ? '/search' : `/search?${params}`, { replaceState: true });
	}

	/** The query box's commit: the typed text becomes the URL state. */
	function submitQuery(event: SubmitEvent): void {
		event.preventDefault();
		navigate({ ...filters, query: draft });
	}

	// The shared unknown-code fallback (a code with no i18n entry
	// renders the generic message, never raw code jargon); the
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// Publish the loaded result items as the viewer's navigation
	// sequence (see $lib/timeline/wall.svelte): the copy outlives
	// this route, so the /a/{id} viewer walks exactly the items this
	// wall rendered -- the rank order in the query state, the time
	// order in the browse.
	$effect(() => {
		publishWallItems($search.items);
	});

	// The live surfaces: the SSE subscription and the scroll sentinel
	// observer live for the page's lifetime.
	$effect(() => {
		const unsubscribe = subscribeAssetEvents((event) => {
			search.applyEvent(event);
		});

		let observer: IntersectionObserver | undefined;
		if (observerAvailable && sentinel !== undefined) {
			observer = new IntersectionObserver(
				(entries) => {
					if (entries.some((entry) => entry.isIntersecting)) {
						// The client's own guards (in-flight, exhausted)
						// collapse bursts into one fetch.
						void search.loadMore();
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

<svelte:head>
	<title>{$_('nav_search')} · lumemor</title>
</svelte:head>

<!-- The page root pins the wall's sticky month headers below the
     context bar: the bar is py-2 (16px) around its tallest child
     (the 39px field row: 8+8 padding, the 21px line, the two 1px
     borders) plus its own 1px border -- 56px. The query state's
     flat wall has no headers to pin; the var is set once for both
     states so the browse state's headers land under the bar. -->
<section class="search" style:--wall-sticky-offset="56px" aria-busy={$search.status === 'loading'}>
	<!-- THE CONTEXT BAR: back + title + count on the left, the ONE
	     search field dead center (the three-column row's middle
	     column), the right column empty. The shell's own topbar hides
	     while this route is active (the content shell's route rule). -->
	<header
		class="search-bar sticky top-0 z-(--z-sticky-bar) grid grid-cols-[minmax(0,1fr)_minmax(0,520px)_minmax(0,1fr)] items-center gap-3 px-4 py-2 bg-card border-b border-border"
	>
		<div class="search-bar__identity flex min-w-0 items-center gap-2">
			<a
				class="search-bar__back inline-flex items-center justify-center w-9 h-9 shrink-0 rounded-control bg-transparent text-text-soft no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
				href="/timeline"
				aria-label={$_('search_back')}
			>
				<ChevronLeft size={20} aria-hidden="true" />
			</a>
			<h1 class="search-bar__title m-0 shrink-0 font-display font-bold text-[20px] leading-[1.3] text-text">
				{$_('nav_search')}
			</h1>
			<!-- The count idiom (glyph + digits; the wording is the
			     accessible name): it follows the query, and it yields
			     below lg -- the identity stays one bar-tall row. -->
			<CountLabel
				class="search-bar__count hidden lg:inline-flex font-body font-normal text-[13px] leading-[1.4] text-text-soft"
				count={$search.items.length}
				label={countLabel}
			/>
		</div>
		<!-- The field's own form: the keyboard's Enter commits the
		     query to the URL (the submit is the page's state write). -->
		<form class="search-bar__field relative min-w-0 w-full" role="search" onsubmit={submitQuery}>
			<Search size={16} class="absolute start-3 top-1/2 -translate-y-1/2 text-text-faint pointer-events-none" />
			<input
				class="search-query w-full ps-9 pe-3 py-2 font-body font-normal text-[15px] leading-[1.4] text-text bg-page border border-border-strong rounded-control transition-[border-color] duration-(--motion-fast) ease-standard"
				type="search"
				placeholder={$_('search_query_placeholder')}
				aria-label={$_('nav_search')}
				bind:value={draft}
			/>
		</form>
	</header>

	<!-- THE BROWSE HINT: the one line above the wall in the browse
	     state only -- an empty query is the recent-photos feed, and
	     the line says so; the query state carries no copy at all. -->
	{#if !queryMode}
		<div class="search__hint-wrap px-4 pt-3">
			<p class="search__hint m-0 mb-2 font-body font-normal text-[13px] leading-[1.5] text-text-soft">
				{$_('search_browse_hint')}
			</p>
		</div>
	{/if}

	<!-- The results: the two-state wall, the states, the pager. The
	     BROWSE wall is month-grouped and so carries the axis slot; the
	     QUERY wall is the flat justified rank order with no axis, so it
	     keeps the plain 8px margin. -->
	<div class="search__results {queryMode ? 'px-2' : 'px-2 pe-[72px]'} pt-3 pb-12">
		{#if firstLoad}
			<LoadingLine label={$_('search_loading')} />
		{:else if settledEmpty}
			{#if queryMode}
				<!-- The no-results card is the QUERY state's face only:
				     a query that answered nothing. An empty query never
				     lands here -- the browse state below owns that case. -->
				<StateCard title={$_('search_empty_title')} body={$_('search_empty_body')} />
			{:else}
				<!-- The browse feed with nothing to show: the library
				     (or the deep-linked filter) holds no photos -- the
				     no-data face, not the no-results one. -->
				<StateCard title={$_('timeline_empty_title')} body={$_('timeline_empty_body')} />
			{/if}
		{:else if $search.status === 'error' && $search.items.length === 0}
			<StateCard
				tone="error"
				body={errorText($search.errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void search.run(filters, $locale ?? 'en')}
			/>
		{:else if queryMode}
			<!-- QUERY state: the FLAT justified wall -- rank order
			     preserved, no grouping; the selection flow is the flat
			     wall's own organ (the same subset). -->
			<FlatWall
				items={$search.items}
				actions={BATCH_ACTIONS}
				onbatch={runBatch}
				onaddtag={openTagDialog}
				onaddmemory={openMemoryDialog}
			/>
		{:else}
			<!-- BROWSE state: the unified month-grouped day-block wall --
			     the same renderer every photo surface composes, with
			     the per-user subset. -->
			<Wall
				items={$search.items}
				actions={BATCH_ACTIONS}
				onbatch={runBatch}
				onaddtag={openTagDialog}
				onaddmemory={openMemoryDialog}
			/>
		{/if}

		{#if $search.status === 'error' && $search.items.length > 0}
			<StateCard
				tone="error"
				body={errorText($search.errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void search.loadMore()}
			/>
		{/if}

		{#if !observerAvailable && $search.nextCursor !== null && $search.status === 'ready'}
			<div class="search__pager flex justify-center my-5">
				<Button variant="ghost" onclick={() => void search.loadMore()}
					><ChevronsDown size={20} aria-hidden="true" />{$_('timeline_load_more')}</Button
				>
			</div>
		{/if}

		<div class="search__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
	</div>
</section>

<!-- The add-tag dialog the walls' openers feed (the captured ids,
     the honest report, the done continuation). -->
<AddTagDialog
	bind:open={tagOpen}
	tags={tagList}
	loading={tagLoading}
	busy={tagBusy}
	errorKey={tagErrorKey}
	report={tagReport}
	onpick={(tagId) => void addToExistingTag(tagId)}
	oncreate={(name) => void addToNewTag(name)}
	ondone={finishTag}
/>

<!-- The memory picker the walls' openers feed (the captured ids, the
     done continuation). -->
<AddToMemoryDialog bind:open={memoryOpen} ids={memoryIds} onpass={finishMemory} />
