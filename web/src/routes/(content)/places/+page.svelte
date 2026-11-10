<script lang="ts">
	// The places page (M3d Task 4, the specimen's city grid): the
	// caller's city grouping as a card grid -- the derived-cover
	// collage over the place's representative ids (the album wall's
	// own cover discipline: one full-bleed, two split, three the
	// 2+1 fall, four the quad), the gradient-shaded city name, the
	// country label and the honest per-caller count. A city click
	// deep-links into the filtered browse (/search?placeIds=) -- the
	// natural handoff, the search page owns the filter state.
	//
	// THE COUNTRY LABEL: the contract carries the ISO code and
	// leaves the display name to the client's own locale table --
	// Intl.DisplayNames(region) is that table, so no country name is
	// hardcoded (a locale switch relabels without a refetch).
	//
	// THE MAP PLACEHOLDER: the honest copy the plan pins -- the tile
	// source is a self-hosting decision of a later update, the city
	// grouping comes first. No map machinery, no stub.
	//
	// SSE: every asset.* event refetches the grouping (the places
	// family's own convergence -- the fresh answer owns the truth),
	// collapsed single-flight; a failed convergence keeps the last
	// good state silently (the next event retries). A locale change
	// refetches too: the name join rides the request.
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { listPlaces, type PlaceSummary } from '$lib/api/places';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { subscribeAssetEvents } from '$lib/events';
	import Button from '$lib/components/Button.svelte';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';
	import { untrack } from 'svelte';

	type ListStatus = 'idle' | 'loading' | 'ready' | 'error';

	let places = $state<PlaceSummary[]>([]);
	let status = $state<ListStatus>('idle');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

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

	/** The head's summary line: the cities and the photos counted honestly. */
	const totalPhotos = $derived(places.reduce((total, place) => total + place.assetCount, 0));

	// The shared unknown-code fallback (see $lib/i18n/errors): an
	// unknown error code has no i18n key, and svelte-i18n would echo
	// the key itself -- such codes render the generic message. The
	// reactive `$_` keeps the text locale-tracked.
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	/**
	 * The country's display name: the client's locale table maps the
	 * ISO code (the contract leaves the name to the client). A host
	 * without the region data falls back to the code itself -- the
	 * honest label, never a blank.
	 */
	function countryName(code: string, activeLocale: string): string {
		try {
			const names = new Intl.DisplayNames([activeLocale], { type: 'region' });
			return names.of(code) ?? code;
		} catch {
			return code;
		}
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
</script>

<svelte:head>
	<title>{$_('nav_places')} · lumemor</title>
</svelte:head>

<section class="places max-w-[1200px] mx-auto p-6" aria-busy={status === 'loading'}>
	<!-- The page head: the section's title and the honest summary. -->
	<div class="places__head flex items-baseline gap-3 mb-4">
		<h1 class="places__title m-0 font-display font-bold text-[24px] leading-[1.3] text-text">{$_('nav_places')}</h1>
		<span class="places__sub text-[13px] leading-[1.4] text-text-faint tabular-nums"
			>{$_('places_head_sub', { values: { cities: places.length, photos: totalPhotos } })}</span
		>
	</div>

	{#if firstLoad}
		<p class="places__loading my-12 text-center text-text-soft animate-loading-in" role="status">
			{$_('places_loading')}
		</p>
	{:else if settledEmpty}
		<div
			class="places__empty max-w-[420px] box-content mx-auto my-12 py-8 px-6 text-center bg-card border border-border rounded-card shadow-card"
		>
			<h2 class="text-[22px] mb-2">{$_('places_empty_title')}</h2>
			<p class="text-text-soft">{$_('places_empty_body')}</p>
		</div>
	{:else if status === 'error' && places.length === 0}
		<div
			class="places__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void runLoad(false)}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
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
		<div
			class="places__failure flex flex-col items-center gap-3 max-w-[420px] box-content mx-auto my-12 p-5 text-center border border-danger rounded-card text-danger-text"
			role="alert"
		>
			<p>{errorText(errorKey)}</p>
			<Button variant="primary" onclick={() => void runLoad(false)}
				><RotateCcw size={20} aria-hidden="true" />{$_('action_retry')}</Button
			>
		</div>
	{/if}

	<!-- The map placeholder's honest copy: the tile source is a
	     self-hosting decision of a later update; the city grouping
	     comes first. -->
	<p
		class="places__map m-0 mt-5 py-5 px-6 text-center font-body font-normal text-[13px] leading-[1.5] text-text-faint border border-dashed border-border-strong rounded-card"
	>
		{$_('places_map_placeholder')}
	</p>
</section>

{#snippet card(place: PlaceSummary)}
	<!-- One city card: the derived-cover collage with the gradient
	     shade and the name over it, the country label and the honest
	     per-caller count below. The click deep-links into the
	     filtered browse. -->
	<a
		class="place-card flex flex-col overflow-hidden rounded-card bg-card border border-border no-underline text-text cursor-pointer transition-[border-color,box-shadow] duration-(--motion-fast) ease-standard hover:border-accent hover:shadow-card focus-visible:border-accent"
		href="/search?placeIds={place.placeId}"
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
		<span class="place-card__body flex items-center justify-between px-3.5 pt-2.5 pb-3">
			<span class="place-card__region font-body font-normal text-[13px] leading-[1.4] text-text-soft truncate"
				>{countryName(place.country, $locale ?? 'en')}</span
			>
			<span
				class="place-card__count font-body font-normal text-[12px] leading-[1.4] text-text-faint whitespace-nowrap tabular-nums"
				>{$_('places_photo_count', { values: { count: place.assetCount } })}</span
			>
		</span>
	</a>
{/snippet}
