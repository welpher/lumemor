<script lang="ts">
	// The unified search page (M3d Task 4): one text box, the facet
	// chips of the seven dimensions, the honest count line, and the
	// results wall -- the web IA's /search row over the unified
	// search endpoint (the merged text and semantic paths, the
	// filter browse).
	//
	// THE URL IS THE STATE (the pinned rule): the query and every
	// filter selection live in the /search URL's parameters, so the
	// page is deep-linkable (a places card links straight into a
	// filtered browse) and the topbar's box navigates here with
	// `query=`. Edits on this page (a query submit, a chip toggle)
	// REPLACE the URL -- the URL is the app state, and the history
	// stays one clean step back to whatever surface led here.
	//
	// THE TWO RESULT MODES (the pinned rule): a query whose trimmed
	// text spans two or more runes is QUERY mode -- the flat
	// justified wall, rank order preserved (no month/day grouping:
	// grouping would lie about the rank order); anything less takes
	// NO text path (the server's own rule, mirrored here), the
	// request is the pure filter browse, and the wall renders the
	// timeline's exact month-grouped day-block form. Both paginate
	// through the mode-tagged cursor (the client's walk).
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
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import { publishWallItems } from '$lib/timeline/wall-sequence.svelte';
	import Wall from '$lib/timeline/Wall.svelte';
	import FlatWall from '$lib/timeline/FlatWall.svelte';
	import FacetChips, { type FacetChip } from '$lib/search/FacetChips.svelte';
	import Button from '$lib/components/Button.svelte';
	import Search from '@lucide/svelte/icons/search';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	const search = createSearchClient();

	// The URL state: the page derives its filters off the route URL,
	// and the run effect re-executes the search whenever the URL
	// (or the locale, whose places-facet names ride the request)
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

	// The browse mode's month/day fold lives inside the shared Wall
	// (the one renderer every photo surface composes); QUERY mode
	// keeps the flat order.

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

	/** The count line's label: the loaded count, "so far" while a
	 * page follows (the honest wire truth, never a claimed total). */
	const countLabel = $derived(
		$search.nextCursor === null
			? $_('search_result_count', { values: { count: $search.items.length } })
			: $_('search_result_count_so_far', { values: { count: $search.items.length } })
	);

	/**
	 * The chips of the current facets: one per entry of the seven
	 * dimensions, each carrying the honest rest-of-filters count and
	 * its selected state (the URL's filters decide, never the facet
	 * list alone -- a deep link's filter can predate the response).
	 */
	const chips = $derived.by(() => {
		const facets = $search.facets;
		if (facets === null) {
			return [];
		}
		const current = filters;
		const list: FacetChip[] = [];
		for (const entry of facets.people) {
			list.push({
				dimension: 'people',
				value: entry.id,
				text: `${$_('search_facet_people')} · ${entry.name}`,
				count: entry.assetCount,
				selected: current.personIds.includes(entry.id)
			});
		}
		for (const entry of facets.places) {
			list.push({
				dimension: 'places',
				value: String(entry.placeId),
				text: `${$_('search_facet_places')} · ${entry.name}`,
				count: entry.assetCount,
				selected: current.placeIds.includes(entry.placeId)
			});
		}
		for (const entry of facets.years) {
			list.push({
				dimension: 'years',
				value: String(entry.year),
				text: `${$_('search_facet_years')} · ${entry.year}`,
				count: entry.assetCount,
				selected: current.years.includes(entry.year)
			});
		}
		for (const entry of facets.cameras) {
			// The display halves: make and model joined, the raw id
			// the honest fallback when EXIF carried neither.
			const halves = [entry.make ?? null, entry.model ?? null].filter(
				(half) => half !== null && half !== ''
			);
			const label = halves.length > 0 ? halves.join(' ') : entry.id;
			list.push({
				dimension: 'cameras',
				value: entry.id,
				text: `${$_('search_facet_cameras')} · ${label}`,
				count: entry.assetCount,
				selected: current.cameras.includes(entry.id)
			});
		}
		for (const entry of facets.kinds) {
			const kindLabel =
				entry.kind === 'image' ? $_('search_kind_image') : $_('search_kind_video');
			list.push({
				dimension: 'kinds',
				value: entry.kind,
				text: `${$_('search_facet_kinds')} · ${kindLabel}`,
				count: entry.assetCount,
				selected: current.kinds.includes(entry.kind)
			});
		}
		for (const entry of facets.albums) {
			list.push({
				dimension: 'albums',
				value: entry.id,
				text: `${$_('search_facet_albums')} · ${entry.title}`,
				count: entry.assetCount,
				selected: current.albumIds.includes(entry.id)
			});
		}
		for (const entry of facets.tags) {
			list.push({
				dimension: 'tags',
				value: entry.id,
				text: `${$_('search_facet_tags')} · ${entry.name}`,
				count: entry.assetCount,
				selected: current.tagIds.includes(entry.id)
			});
		}
		return list;
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

	/** Toggle one chip's dimension value in the filter state. */
	function toggleChip(chip: FacetChip): void {
		const next: SearchFilters = { ...filters };
		switch (chip.dimension) {
			case 'people': {
				next.personIds = toggleEntry(filters.personIds, chip.value);
				break;
			}
			case 'places': {
				const placeId = Number.parseInt(chip.value, 10);
				next.placeIds = toggleEntry(filters.placeIds, placeId);
				break;
			}
			case 'years': {
				const year = Number.parseInt(chip.value, 10);
				next.years = toggleEntry(filters.years, year);
				break;
			}
			case 'cameras': {
				next.cameras = toggleEntry(filters.cameras, chip.value);
				break;
			}
			case 'kinds': {
				next.kinds = toggleEntry(filters.kinds, chip.value as SearchFilters['kinds'][number]);
				break;
			}
			case 'albums': {
				next.albumIds = toggleEntry(filters.albumIds, chip.value);
				break;
			}
			case 'tags': {
				next.tagIds = toggleEntry(filters.tagIds, chip.value);
				break;
			}
		}
		navigate(next);
	}

	/** One entry's toggle in a filter list (the generic dimension arm). */
	function toggleEntry<T>(entries: readonly T[], value: T): T[] {
		return entries.includes(value) ? entries.filter((entry) => entry !== value) : [...entries, value];
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
	// wall rendered -- the rank order in the query mode, the time
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

<section class="search px-2" aria-busy={$search.status === 'loading'}>
	<!-- The search head: the specimen's large query box, the facet
	     chips, the honest count line. The box is a form so the
	     keyboard's Enter commits the search. -->
	<div class="search__head px-2 pt-6">
		<form class="search-form relative max-w-[640px]" role="search" onsubmit={submitQuery}>
			<Search size={18} class="absolute start-3.5 top-1/2 -translate-y-1/2 text-text-faint pointer-events-none" />
			<input
				class="search-query w-full ps-11 pe-4 py-3 font-body font-normal text-[17px] leading-[1.4] text-text bg-card border border-border-strong rounded-control outline-none transition-[border-color] duration-(--motion-fast) ease-standard focus:border-accent"
				type="search"
				placeholder={$_('search_query_placeholder')}
				aria-label={$_('nav_search')}
				bind:value={draft}
			/>
		</form>
		{#if $search.facets !== null}
			<FacetChips {chips} ontoggle={toggleChip} />
			<p class="search-meta flex flex-wrap items-baseline gap-x-2.5 m-0 mt-2.5 text-[12px] leading-[1.4] text-text-faint">
				<b class="font-body font-bold text-text-soft tabular-nums">{countLabel}</b>
				<span>{$_('search_result_note')}</span>
			</p>
		{/if}
	</div>

	<!-- The results: the two-mode wall, the states, the pager. -->
	<div class="search__results px-2 pt-3 pb-12">
		{#if firstLoad}
			<p class="search__status my-12 text-center text-text-soft animate-loading-in" role="status">{$_('search_loading')}</p>
		{:else if settledEmpty}
			<div class="search-empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card">
				<h1 class="text-[22px] mb-2">{$_('search_empty_title')}</h1>
				<p class="text-text-soft">{$_('search_empty_body')}</p>
			</div>
		{:else if $search.status === 'error' && $search.items.length === 0}
			<div class="search__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
				<p>{errorText($search.errorKey)}</p>
				<Button variant="primary" onclick={() => void search.run(filters, $locale ?? 'en')}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
			</div>
		{:else if queryMode}
			<!-- QUERY mode: the FLAT justified wall -- rank order
			     preserved, no grouping, no selection (the tiles are pure
			     navigation). -->
			<FlatWall items={$search.items} />
		{:else}
			<!-- BROWSE mode: the unified month-grouped day-block wall --
			     the same renderer every photo surface composes, no
			     selection here either. -->
			<Wall items={$search.items} />
		{/if}

		{#if $search.status === 'error' && $search.items.length > 0}
			<div class="search__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text" role="alert">
				<p>{errorText($search.errorKey)}</p>
				<Button variant="primary" onclick={() => void search.loadMore()}><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button>
			</div>
		{/if}

		{#if !observerAvailable && $search.nextCursor !== null && $search.status === 'ready'}
			<div class="search__pager flex justify-center my-5">
				<Button variant="ghost" onclick={() => void search.loadMore()}><ChevronsDown size={20} aria-hidden="true" />{$_('timeline_load_more')}</Button>
			</div>
		{/if}

		<div class="search__sentinel h-px" bind:this={sentinel} aria-hidden="true"></div>
	</div>
</section>
