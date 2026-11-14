<script lang="ts">
	// The place detail: ONE city's photos, the place domain's third
	// browse face. The context bar carries the city (20px), the
	// country (13px soft) and the honest photo count, with the back
	// affordance alone on the left and NOTHING on the right -- a
	// place is a filtered view, not a container: there is nothing to
	// manage (no properties, no cover, no delete, no map shortcut).
	// The body is the shared wall over this place's photos, with the
	// timeline's curation subset (favorite / hide / trash) on a
	// selection.
	//
	// THE READS: the grouping read resolves this place's localized
	// name, country, coordinate and count (the id is the address; the
	// names are the reader's language, never baked into a link), and
	// the wall walks the timeline endpoint's `placeIds` narrow. The
	// page keys the wall component per id, so another place's page
	// always walks its own photos.
	//
	// SSE: every asset.* event refetches the grouping (the count is
	// the server's honest answer) and converges the wall through its
	// own client. A place that leaves the grouping after its last
	// visible photo went away keeps its name but reports zero photos
	// -- the wall's own empty state carries the rest. A place that
	// never existed (or belongs to another account) renders the one
	// unreachable face, indistinguishable from a deleted one.
	import { page } from '$app/state';
	import { _ } from 'svelte-i18n';
	import { locale } from 'svelte-i18n';
	import { untrack } from 'svelte';
	import { listPlaces, type PlaceSummary } from '$lib/api/places';
	import { subscribeAssetEvents } from '$lib/events';
	import { resolveErrorText } from '$lib/i18n/errors';
	import { countryName } from '$lib/places/places';
	import StateCard from '$lib/components/StateCard.svelte';
	import UnreachableCard from '$lib/components/UnreachableCard.svelte';
	import LoadingLine from '$lib/components/LoadingLine.svelte';
	import CountLabel from '$lib/components/CountLabel.svelte';
	import PlaceWall from '$lib/places/PlaceWall.svelte';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';

	type PageStatus = 'loading' | 'ready' | 'error' | 'unreachable';

	/** The route's id: a positive integer, or null for a malformed address. */
	const placeId = $derived.by(() => {
		const raw = page.params.id ?? '';
		if (!/^[0-9]+$/.test(raw)) {
			return null;
		}
		const parsed = Number(raw);
		return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null;
	});

	let place = $state<PlaceSummary | null>(null);
	let status = $state<PageStatus>('loading');
	/** The error-code i18n key of the last failed load ('' when none). */
	let errorKey = $state('');

	// The single-flight guard: SSE bursts converge through one
	// trailing fetch (the places page's own collapse).
	let running = false;
	let queued = false;

	/**
	 * Resolve this place from the grouping read. A place absent from
	 * the answer keeps its last known face with a zero count (its
	 * last photo left); a place that never answered is the dead end.
	 */
	async function runLoad(): Promise<void> {
		if (running) {
			queued = true;
			return;
		}
		running = true;
		try {
			const outcome = await listPlaces($locale ?? 'en');
			if (outcome.ok) {
				const found = placeId === null ? undefined : outcome.places.find((entry) => entry.placeId === placeId);
				if (found !== undefined) {
					place = found;
					status = 'ready';
				} else if (place !== null) {
					place = { ...place, assetCount: 0 };
					status = 'ready';
				} else {
					status = 'unreachable';
				}
				errorKey = '';
			} else if (place === null) {
				status = 'error';
				errorKey = outcome.errorKey;
			}
		} finally {
			running = false;
			if (queued) {
				queued = false;
				void runLoad();
			}
		}
	}

	const titleText = $derived(place?.name ?? $_('nav_places'));
	const countryText = $derived(place === null ? '' : countryName(place.country, $locale ?? 'en'));

	/** The shared unknown-code fallback (see $lib/i18n/errors). */
	function errorText(key: string): string {
		return resolveErrorText($_, key);
	}

	// The load, the locale join and the SSE convergence: the first
	// load and every locale change re-run the fetch (the display
	// names ride the request); every asset event refreshes the honest
	// count. The subscription lives for the page's lifetime.
	$effect(() => {
		// The underscore read registers $locale as the dependency (a
		// locale switch re-runs the join).
		const _activeLocale = $locale ?? 'en';
		if (placeId === null) {
			status = 'unreachable';
			return;
		}
		untrack(() => {
			status = 'loading';
			place = null;
			errorKey = '';
			void runLoad();
		});
		const unsubscribe = subscribeAssetEvents(() => {
			void runLoad();
		});
		return unsubscribe;
	});
</script>

<svelte:head>
	<title>{titleText} · lumemor</title>
</svelte:head>

<!-- The page root pins the wall's sticky month headers below the
     contextual bar: the bar is py-2 (16) around an h-9 row (36) plus
     its 1px border -- 53px. -->
<section class="place-detail w-full" style:--wall-sticky-offset="53px">
	<!-- THE CONTEXTUAL BAR: back + the city, the country and the
	     count. The right side is EMPTY by decision -- a place has no
	     actions. The shell's own topbar hides while this route is
	     active (the content shell's route rule). -->
	<header
		class="place-bar sticky top-0 z-(--z-sticky-bar) flex items-center gap-2 px-4 py-2 bg-card border-b border-border"
	>
		<a
			class="place-bar__back inline-flex items-center justify-center w-9 h-9 shrink-0 rounded-control bg-transparent text-text-soft no-underline cursor-pointer transition-[background-color,color] duration-(--motion-fast) ease-standard hover:bg-accent-soft hover:text-accent-soft-text focus-visible:bg-accent-soft focus-visible:text-accent-soft-text"
			href="/places"
			aria-label={$_('place_back')}
		>
			<ChevronLeft size={20} aria-hidden="true" />
		</a>
		{#if place !== null}
			<h1
				class="place-bar__title m-0 min-w-0 truncate font-display font-bold text-[20px] leading-[1.3] text-text"
				data-place-name
			>
				{place.name}
			</h1>
			<span
				class="place-bar__country font-body font-normal text-[13px] leading-[1.4] text-text-soft whitespace-nowrap"
				data-place-country>{countryText}</span
			>
			<CountLabel
				class="place-bar__count font-body font-normal text-[13px] leading-[1.4] text-text-soft"
				count={place.assetCount}
				label={$_('places_photo_count', { values: { count: place.assetCount } })}
			/>
		{/if}
	</header>

	{#if status === 'loading'}
		<LoadingLine label={$_('place_loading')} />
	{:else if status === 'unreachable'}
		<!-- The dead end: one card, no name and no count -- nothing
		     that would say whether the place exists. The bar keeps the
		     way back alone. -->
		<UnreachableCard />
	{:else if status === 'error' && place === null}
		<StateCard
			tone="error"
			body={errorText(errorKey)}
			actionLabel={$_('action_retry')}
			onaction={() => void runLoad()}
		/>
	{:else if place !== null}
		<!-- The photo wall: keyed on the id, so another place's page
		     always walks its own photos. -->
		{#key place.placeId}
			<PlaceWall placeId={place.placeId} />
		{/key}
	{/if}
</section>
