<script lang="ts">
	// The places page: the city domain's two browse faces, one
	// address each.
	//
	// THE FLAT LIST: one full-width grid of city cards, no grouping
	// headers (a family's places are a handful of cities; a country
	// grouping would only add empty rows). Each card is the
	// derived-cover collage with the city name over it, then the
	// country and the capture years as the card's second line beside
	// the honest per-caller count. The list's order is the server's:
	// the most recently photographed city first.
	//
	// THE MAP: the full viewport height under the toolbar (the page
	// does not scroll in this face). The pins are the same grouping
	// in map coordinates, clustered by zoom; a pin or bubble click
	// opens the RIGHT panel -- the page never navigates. The map
	// yields to a 70/30 split (stacked half/half on narrow screens),
	// the camera re-frames the selected spot, and the panel renders
	// the shared wall over that spot's photos. Clicking the same
	// marker again, Escape, or the panel's close collapses it.
	//
	// THE VIEW IS AN ADDRESS: `?mode=map` is the map face's own
	// link, so a reload or a shared address lands on the same face.
	// Switching back to the list restores the list's scroll seat
	// (the frame resets it to the top on a push navigation; this
	// page puts it back).
	//
	// SSE: every asset.* event refetches the grouping (the places
	// family's own convergence -- the fresh answer owns the truth),
	// collapsed single-flight; a failed convergence keeps the last
	// good state silently (the next event retries). A locale change
	// refetches too: the name join rides the request.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { page } from '$app/state';
	import { goto, replaceState } from '$app/navigation';
	import { listPlaces, type PlaceSummary } from '$lib/api/places';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import StateCard from '$lib/components/StateCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import { tabIdle, tabSelected } from '$lib/components/selectedState';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import List from '@lucide/svelte/icons/list';
	import Map from '@lucide/svelte/icons/map';
	import MapView, { type MapState } from '$lib/map/MapView.svelte';
	import type { PlacePoint } from '$lib/map/tiles';
	import PlacePanel from '$lib/places/PlacePanel.svelte';
	import { clusterLabelValues, countryName, placeYearsText, selectedPlaces } from '$lib/places/places';
	import { untrack } from 'svelte';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let places = $state<PlaceSummary[]>([]);
	let status = $state<ListStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	/** The map face: the address decides (`?mode=map`), never local state alone. */
	const isMap = $derived(page.url.searchParams.get('mode') === 'map');
	/** The base layer's state, as the map reports it. */
	let mapState = $state<MapState>('loading');
	/** The tile host's attribution line, as the map resolves it. */
	let mapAttribution = $state('');

	// The list's scroll seat, captured when the map face opens and
	// restored when the list face returns (the frame's navigation
	// reset lands every push at the top).
	let listSeat = 0;

	// The single-flight guard: while one fetch runs, later triggers
	// park (the burst's tail) and one trailing fetch runs when it
	// lands -- the same collapse the albums wall applies.
	let running = false;
	let queued = false;

	/**
	 * Fetch the grouping and apply it. `converge` marks an
	 * SSE-driven run: the visible state stays (the cards swap in
	 * place on success) and a failure keeps the last good answer
	 * instead of raising the error box -- later events retry.
	 */
	async function runLoad(converge: boolean): Promise<void> {
		if (running) {
			queued = true;
			return;
		}
		running = true;
		if (!converge) {
			status = 'loading';
		}
		try {
			const outcome = await listPlaces($locale ?? 'en');
			if (outcome.ok) {
				places = outcome.places;
				errorKey = '';
				status = 'ready';
			} else if (!converge) {
				status = 'error';
				errorKey = outcome.errorKey;
			}
		} finally {
			running = false;
			if (queued) {
				queued = false;
				void runLoad(true);
			}
		}
	}

	const firstLoad = $derived(places.length === 0 && status === 'loading');
	const settledEmpty = $derived(places.length === 0 && status === 'ready');

	/** The pins of the map face: the same grouping, in map coordinates. */
	const mapPoints = $derived<PlacePoint[]>(
		places.map((place) => ({
			placeId: place.placeId,
			name: place.name,
			count: place.assetCount,
			lat: place.latitude,
			lon: place.longitude
		}))
	);

	/** The map footer's left line: cities, photos and countries of the grouping. */
	const mapSummary = $derived({
		cities: places.length,
		photos: places.reduce((sum, place) => sum + place.assetCount, 0),
		countries: new Set(places.map((place) => place.country)).size
	});

	/**
	 * The panel's selection: the clicked marker's place ids. It is
	 * read from the ADDRESS once at mount (`?place=<ids>`, the viewer
	 * round trip's restoration): the browser's own address is the
	 * source, because a popstate landing (the viewer's Escape) has
	 * resolved it before the component initializes while a router
	 * snapshot may still trail. The page owns the value afterwards;
	 * every change is written back into the address with a replace
	 * (never a navigation). The parameter belongs to the MAP face: on
	 * the list face it is read as nothing.
	 */
	const mountUrl = new URL(window.location.href);
	let selectedIds = $state<number[]>(
		mountUrl.searchParams.get('mode') === 'map' ? parsePlaceParam(mountUrl.searchParams.get('place')) : []
	);
	/** The selected places (the panel's rows), in the listing order. */
	const selected = $derived(selectedPlaces(places, selectedIds));
	const selectedKey = $derived(selectedIds.join(','));

	/** Whether two marker selections are the same marker's. */
	function sameSelection(a: readonly number[], b: readonly number[]): boolean {
		if (a.length !== b.length) {
			return false;
		}
		const wanted = new Set(a);
		return b.every((id) => wanted.has(id));
	}

	/** Parse the address's `place` parameter (the selected marker's ids). */
	function parsePlaceParam(raw: string | null): number[] {
		if (raw === null || raw === '') {
			return [];
		}
		const ids: number[] = [];
		for (const part of raw.split(',')) {
			if (!/^[0-9]+$/.test(part)) {
				return [];
			}
			ids.push(Number(part));
		}
		return ids;
	}

	/**
	 * Write the selection into the address (replace, never navigate):
	 * the panel's own state survives the viewer round trip -- a photo
	 * click leaves the map, and coming back restores the same spot and
	 * camera -- without ever turning a pin click into a page
	 * navigation.
	 */
	function syncSelection(next: number[] | null): void {
		selectedIds = next ?? [];
		const url = new URL(page.url);
		if (selectedIds.length === 0) {
			url.searchParams.delete('place');
		} else {
			url.searchParams.set('place', selectedIds.join(','));
		}
		replaceState(url, {});
	}

	/**
	 * A pin or bubble click: a fresh marker opens the panel (in
	 * place -- no navigation); the SAME marker again collapses it.
	 */
	function pickMarker(placeIds: number[]): void {
		if (placeIds.length === 0) {
			return;
		}
		if (sameSelection(selectedIds, placeIds)) {
			syncSelection(null);
			return;
		}
		syncSelection([...placeIds]);
	}

	/** The focus hand-back's request: the marker that opened the panel. */
	let returnFocus = $state<readonly number[] | null>(null);
	let returnFocusToken = $state(0);

	/** Whether the keyboard focus currently sits inside the panel. */
	function panelHasFocus(): boolean {
		const active = document.activeElement;
		return active instanceof HTMLElement && active.closest('[data-place-panel]') !== null;
	}

	/**
	 * Collapse the panel. Closing FROM INSIDE it hands the keyboard
	 * back to the marker that opened it (the sample's rule): the
	 * panel must not strand focus at the page top. A click on the
	 * marker itself needs no hand-back -- the focus is already there.
	 */
	function closePanel(): void {
		const returning = panelHasFocus() ? [...selectedIds] : null;
		syncSelection(null);
		if (returning !== null && returning.length > 0) {
			returnFocus = returning;
			returnFocusToken += 1;
		}
	}

	/**
	 * A count-only bubble's sentence: the shared country's name (when
	 * the group has one), the city count and the photo count. A named
	 * pin answers null -- its own label already says it.
	 */
	function describeCluster(placeIds: readonly number[]): string | null {
		const facts = clusterLabelValues(selectedPlaces(places, [...placeIds]), $locale ?? 'en');
		if (facts === null) {
			return null;
		}
		return facts.name === null
			? $_('places_map_pin_cluster_unnamed', {
					values: { cities: facts.cities, photos: facts.photos }
				})
			: $_('places_map_pin_cluster', { values: facts });
	}

	/**
	 * The face switch: the address is the state (`?mode=map`), and
	 * the list's seat survives the round trip.
	 */
	async function switchMode(next: 'list' | 'map'): Promise<void> {
		if ((next === 'map') === isMap) {
			return;
		}
		const main = document.querySelector<HTMLElement>('main');
		if (next === 'map') {
			listSeat = main?.scrollTop ?? 0;
			selectedIds = [];
			await goto('/places?mode=map');
			return;
		}
		selectedIds = [];
		await goto('/places');
		const seat = listSeat;
		requestAnimationFrame(() => {
			if (main !== null) {
				main.scrollTop = seat;
			}
		});
	}

	/** The view switch's face: the current one is a filled accent block,
	 * the other stays plain on the track (see selectedState.ts). */
	function viewClass(current: boolean): string {
		const state = current ? tabSelected : tabIdle;
		return `places__view inline-flex items-center gap-1 px-2.5 py-1 font-body font-semibold text-[13px] leading-[1.4] rounded-control no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard ${state}`;
	}

	/** The escape hatch: the panel collapses (the map returns to full width). */
	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && selectedIds.length > 0) {
			closePanel();
		}
	}

	// The shared unknown-code fallback (see $lib/i18n/errors): an
	// unknown error code has no i18n key, and svelte-i18n would echo
	// the key itself -- such codes render the generic message. The
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/** The card's year text: one year, or the span's two (the
	 * translation carries the dash; the digits are the locale's). */
	function yearsText(place: PlaceSummary): string {
		return placeYearsText(
			place,
			(from, to) => $_('places_year_range', { values: { from, to } }),
			(year) => String(year)
		);
	}

	/** The card's second line: the country and the capture years. */
	function cardSpan(place: PlaceSummary): string {
		return $_('places_card_span', {
			values: { country: countryName(place.country, $locale ?? 'en'), years: yearsText(place) }
		});
	}

	// The load, the locale join and the SSE subscription: the first
	// load and every locale change re-run the fetch (the display
	// names ride the request); every asset event converges through
	// one fresh fetch. The subscription lives for the page's
	// lifetime.
	$effect(() => {
		// The underscore read is deliberate: it registers $locale as
		// this effect's dependency (runLoad's own reads are untracked
		// below), so a locale switch re-runs the fetch whose display
		// names ride the request.
		const _activeLocale = $locale ?? 'en';
		untrack(() => {
			void runLoad(false);
		});
		const unsubscribe = subscribeAssetEvents(() => {
			void runLoad(true);
		});
		return unsubscribe;
	});

	// A selected place that left the grouping (its last visible
	// photo went away) collapses the panel: an empty selection has
	// nothing to look at -- and the address drops it with it.
	$effect(() => {
		if (selectedIds.length > 0 && status === 'ready' && selected.length === 0) {
			syncSelection(null);
		}
	});

	// The `place` parameter is the MAP face's own state. A list-face
	// address carrying one is a leftover (a stale deep link, a hand
	// edit, a back navigation that crossed faces): the parameter is
	// dropped so a reload or a shared address never carries a
	// selection the face cannot show. The browser's address decides
	// -- a router snapshot still trailing the popstate must not cost
	// the map its own selection.
	$effect(() => {
		if (isMap) {
			return;
		}
		const url = new URL(window.location.href);
		if (url.searchParams.get('mode') === 'map' || !url.searchParams.has('place')) {
			return;
		}
		url.searchParams.delete('place');
		replaceState(url, {});
	});
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<svelte:head>
	<title>{$_('nav_places')} · lumemor</title>
</svelte:head>

<!-- The page's container: full width + 16px. The map face is the
     frame's whole height (toolbar and footer fixed, the map eats the
     rest, the page does not scroll); the list face scrolls with the
     shell. -->
<section
	class="places w-full px-4 pt-4 {isMap ? 'places--map flex h-full min-h-0 flex-col overflow-hidden pb-3' : 'pb-12'}"
	aria-busy={status === 'loading'}
>
	<!-- THE ONE ROW: the page title with the city count on the left,
	     the two-face switch on the right. The count follows the
	     CARDS (one card IS one city), never the photos behind them. -->
	<div class="places__toolbar flex flex-none items-center gap-3 pb-4 border-b border-border">
		<h1 class="places__title m-0 font-display font-bold text-[20px] leading-[1.3] text-text">
			{$_('nav_places')}
			{#if status === 'ready'}
				<span
					class="places__title-count font-body font-normal text-[15px] leading-[1.4] text-text-soft tabular-nums"
				>
					· {$_('places_bar_count', { values: { count: places.length } })}
				</span>
			{/if}
		</h1>
		<!-- The view switch: the page's one action, an address not a
		     local toggle (`?mode=map` is linkable and survives a
		     reload). The click routes through the page so the list's
		     scroll seat survives. -->
		<div
			class="places__views ms-auto inline-flex items-center gap-0.5 rounded-control border border-border-strong p-0.5"
			role="group"
			aria-label={$_('places_view_switch')}
		>
			<a
				class={viewClass(!isMap)}
				href="/places"
				aria-current={isMap ? undefined : 'page'}
				data-places-mode="list"
				onclick={(event) => {
					event.preventDefault();
					void switchMode('list');
				}}
			>
				<List size={14} aria-hidden="true" />
				{$_('places_view_list')}
			</a>
			<a
				class={viewClass(isMap)}
				href="/places?mode=map"
				aria-current={isMap ? 'page' : undefined}
				data-places-mode="map"
				onclick={(event) => {
					event.preventDefault();
					void switchMode('map');
				}}
			>
				<Map size={14} aria-hidden="true" />
				{$_('places_view_map')}
			</a>
		</div>
	</div>

	<div class="places__content {isMap ? 'flex min-h-0 flex-1 flex-col pt-3' : 'pt-4'}">
		{#if firstLoad}
			<LoadingLine label={$_('places_loading')} />
		{:else if settledEmpty}
			<StateCard title={$_('places_empty_title')} body={$_('places_empty_body')} />
		{:else if status === 'error' && places.length === 0}
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void runLoad(false)}
			/>
		{:else if isMap}
			<!-- The map face: the map (and its place panel) fill the
			     rest of the frame; the counts and the tile source's
			     attribution ride the footer. -->
			<div class="places__map-face flex min-h-0 flex-1 flex-col gap-3">
				<div class="places__map-row flex min-h-0 flex-1 flex-col gap-4 sm:flex-row" data-place-row>
					<div
						class="places__map-card relative min-h-0 min-w-0 flex-1 overflow-hidden rounded-card border border-border"
						data-testid="places-map"
					>
						<MapView
							places={mapPoints}
							focus={selectedIds.length > 0 ? selectedIds : null}
							selected={selectedIds.length > 0 ? selectedIds : null}
							label={$_('places_map_label')}
							onselect={pickMarker}
							{describeCluster}
							{returnFocus}
							{returnFocusToken}
							bind:tiles={mapState}
							bind:attribution={mapAttribution}
						/>
					</div>
					{#if selected.length > 0}
						<!-- The selection changes remount the panel: its wall
						     walks exactly the selected spot's places. -->
						{#key selectedKey}
							<PlacePanel places={selected} onclose={closePanel} />
						{/key}
					{/if}
				</div>
				<div class="places__map-footer flex flex-none flex-wrap items-center justify-between gap-2">
					<p
						class="places__map-counts m-0 font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums"
					>
						{$_('places_map_summary', {
							values: {
								cities: mapSummary.cities,
								photos: mapSummary.photos,
								countries: mapSummary.countries
							}
						})}
					</p>
					{#if mapAttribution !== ''}
						<p
							class="places__map-attribution m-0 font-body font-normal text-[11px] leading-[1.4] text-text-faint"
						>
							{$_('places_map_attribution', { values: { source: mapAttribution } })}
						</p>
					{/if}
				</div>
			</div>
		{:else}
			<div class="places__grid grid grid-cols-[repeat(auto-fill,minmax(min(240px,100%),1fr))] gap-3.5">
				{#each places as place (place.placeId)}
					{@render card(place)}
				{/each}
			</div>
		{/if}

		{#if status === 'error' && places.length > 0}
			<!-- A failed convergence while cards exist: the page stays,
		     the next event retries; the box keeps the retry affordance. -->
			<StateCard
				tone="error"
				body={errorText(errorKey)}
				actionLabel={$_('action_retry')}
				onaction={() => void runLoad(false)}
			/>
		{/if}
	</div>
</section>

{#snippet card(place: PlaceSummary)}
	<!-- One city card: the derived-cover collage with the gradient
	     shade and the name over it, the country and the capture
	     years beside the honest per-caller count below. The click
	     opens the place's own wall. -->
	<a
		class="place-card flex flex-col overflow-hidden rounded-card bg-card border border-border no-underline text-text cursor-pointer transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard hover:border-accent hover:shadow-card"
		href="/places/{place.placeId}"
	>
		<span class="place-card__cover relative block aspect-[4/3] bg-accent-soft" aria-hidden="true">
			{#if place.representativeCoverAssetIds.length === 1}
				<img
					class="absolute inset-0 w-full h-full object-cover"
					src={`/api/v1/assets/${encodeURIComponent(place.representativeCoverAssetIds[0] ?? '')}/thumbnail?size=tile`}
					alt=""
					loading="lazy"
					decoding="async"
					draggable="false"
				/>
			{:else if place.representativeCoverAssetIds.length > 1}
				<!-- The collage cells: the 2+1 fall's third cell spans both
				     columns, the quad fills the 2x2 grid (the album wall's
				     own cover discipline). -->
				<span class="place-card__cells grid grid-cols-2 absolute inset-0">
					{#each place.representativeCoverAssetIds.slice(0, 4) as coverId, index (coverId)}
						<span
							class="place-card__cell {place.representativeCoverAssetIds.length === 3 && index === 2
								? 'col-span-2'
								: ''} overflow-hidden"
						>
							<img
								class="w-full h-full object-cover"
								src={`/api/v1/assets/${encodeURIComponent(coverId)}/thumbnail?size=tile`}
								alt=""
								loading="lazy"
								decoding="async"
								draggable="false"
							/>
						</span>
					{/each}
				</span>
			{:else}
				<!-- A bare group (the contract pins at least one candidate;
				     a malformed body degrades to the honest placeholder). -->
				<span class="place-card__placeholder absolute inset-0 grid place-items-center text-text-faint">
					<MapPin size={28} />
				</span>
			{/if}
			<!-- The gradient shade and the city name over the covers. -->
			<span class="place-card__shade absolute inset-0 bg-linear-to-t from-night/55 to-transparent"></span>
			<span
				class="place-card__name absolute left-3.5 right-3.5 bottom-2.5 font-body font-bold text-[17px] leading-[1.4] text-cream-text [text-shadow:0_1px_3px_rgb(0_0_0/0.4)]"
				>{place.name}</span
			>
		</span>
		<span class="place-card__body flex items-center justify-between gap-2 px-3.5 pt-2.5 pb-3">
			<span
				class="place-card__years font-body font-normal text-[13px] leading-[1.4] text-text-soft tabular-nums truncate"
				>{cardSpan(place)}</span
			>
			<CountLabel
				class="place-card__count font-body font-normal text-[13px] leading-[1.4] text-text-soft"
				count={place.assetCount}
				label={$_('places_photo_count', { values: { count: place.assetCount } })}
			/>
		</span>
	</a>
{/snippet}
