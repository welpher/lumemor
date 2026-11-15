<script lang="ts">
	// The "edit location" dialog of the viewer's info panel: the
	// place-name search and the map edited as one pick. The map
	// answers both gestures the design pins -- drag pans the base
	// layer (the library's own), one click drops the pin (the
	// library's click event; the pin rides the map's projection, so
	// it stays anchored to the content while the map moves) -- and
	// the search box is the keyboard path into the same result.
	//
	// THE SEARCH SOURCE: the instance's own place vocabulary (the
	// caller's resolved cities), fetched fresh on every open and
	// filtered locally -- a family's places are the places the photos
	// already name; selecting one pins its coordinates. There is no
	// geocoder behind the field, and the honest face is the family's
	// own vocabulary, never a fake global gazetteer.
	//
	// The page owns the write: `onsubmit` receives the picked pair;
	// the dialog owns the pin and the draft and closes itself on the
	// caller's success.
	import { _ } from 'svelte-i18n';
	import Modal from '$lib/components/Modal.svelte';
	import Button from '$lib/components/Button.svelte';
	import { resolveErrorText } from '$lib/i18n/errors';
	import MapView from '$lib/map/MapView.svelte';
	import { listPlaces, type PlaceSummary } from '$lib/api/places';
	import { currentLocale } from '$lib/i18n';
	import Search from '@lucide/svelte/icons/search';
	import Check from '@lucide/svelte/icons/check';

	type Props = {
		/** Open state; bind it from the caller, the dialog closes itself. */
		open?: boolean;
		/** The photo's current place name ('' when none resolved). */
		placeName: string;
		/** The photo's current coordinates (null when unlocated). */
		initial: { lat: number; lon: number } | null;
		/** Writes run one at a time: the submit disables while in flight. */
		busy?: boolean;
		/** The error-code i18n key of the last failed write ('' when none). */
		errorKey?: string;
		/** Run the write with the picked pair. */
		onsubmit: (lat: number, lon: number) => void;
	};

	let { open = $bindable(false), placeName, initial, busy = false, errorKey = '', onsubmit }: Props = $props();

	/** The pin (the picked coordinates; the write's payload). */
	let pin = $state<{ lat: number; lon: number } | null>(null);
	/** The search text and the instance's places behind it. */
	let query = $state('');
	let places = $state<PlaceSummary[]>([]);
	let placesLoading = $state(false);

	// Each open seeds the pin from the stored coordinates and fetches
	// the vocabulary fresh (the fetched answer owns the options, the
	// picker dialogs' discipline).
	$effect(() => {
		if (!open) {
			return;
		}
		pin = initial;
		query = '';
		let current = true;
		places = [];
		placesLoading = true;
		void (async () => {
			const outcome = await listPlaces(currentLocale());
			if (!current || !open) {
				// Closed mid-flight: the next open refetches.
				return;
			}
			placesLoading = false;
			if (outcome.ok) {
				places = outcome.places;
			}
		})();
		return () => {
			current = false;
		};
	});

	/** The vocabulary filtered by the query, case-insensitively. */
	const matches = $derived.by(() => {
		const needle = query.trim().toLowerCase();
		if (needle === '') {
			return places.slice(0, 6);
		}
		return places.filter((place) => place.name.toLowerCase().includes(needle)).slice(0, 6);
	});

	/** One search pick: the pin lands on the place's own coordinates. */
	function pickPlace(place: PlaceSummary): void {
		pin = { lat: place.latitude, lon: place.longitude };
	}

	/** One map pick: the pin lands where the click landed. */
	function pickPoint(at: { lat: number; lon: number }): void {
		pin = at;
	}

	/** The pin's coordinates line, recomputed on every move of the pin. */
	const coordsLine = $derived(pin === null ? '' : `${pin.lat.toFixed(4)}, ${pin.lon.toFixed(4)}`);

	function submit(): void {
		if (busy || pin === null) {
			return;
		}
		onsubmit(pin.lat, pin.lon);
	}
</script>

<Modal bind:open title={$_('viewer_edit_location_title')} maxWidth="440px">
	<div class="viewer-location-dialog flex flex-col gap-3">
		<p class="m-0 font-body font-normal text-[14px] leading-[1.5] text-text-soft">
			{$_('viewer_edit_location_note')}
		</p>
		<label class="relative block">
			<span class="sr-only">{$_('viewer_edit_location_search')}</span>
			<Search
				size={16}
				class="absolute start-3 top-1/2 -translate-y-1/2 text-text-faint pointer-events-none"
				aria-hidden="true"
			/>
			<input
				class="w-full ps-9 pe-3 py-[9px] font-body font-normal text-[15px] leading-[1.4] text-text bg-page border border-border-strong rounded-control"
				type="search"
				placeholder={$_('viewer_edit_location_search')}
				data-testid="viewer-location-search"
				bind:value={query}
			/>
		</label>
		{#if query.trim() !== ''}
			<ul
				class="m-0 p-0 -mt-2 list-none divide-y divide-border border border-border rounded-control overflow-hidden"
			>
				{#if placesLoading}
					<li class="px-3 py-2 font-body font-normal text-[13px] leading-[1.4] text-text-faint" role="status">
						{$_('viewer_info_loading')}
					</li>
				{:else if matches.length === 0}
					<li class="px-3 py-2 font-body font-normal text-[13px] leading-[1.4] text-text-faint">
						{$_('viewer_edit_location_no_match')}
					</li>
				{:else}
					{#each matches as place (place.placeId)}
						<li>
							<button
								class="flex w-full items-center justify-between gap-2 border-0 bg-transparent p-0 px-3 py-2 text-left cursor-pointer font-body font-normal text-[13px] leading-[1.4] text-text transition-[background-color] duration-(--motion-fast) ease-standard hover:bg-accent-soft"
								type="button"
								data-place-id={place.placeId}
								onclick={() => pickPlace(place)}
							>
								<span class="min-w-0 truncate">{place.name}</span>
								<span class="flex-none font-body font-normal text-[12px] text-text-faint tabular-nums">
									{place.latitude.toFixed(2)}, {place.longitude.toFixed(2)}
								</span>
							</button>
						</li>
					{/each}
				{/if}
			</ul>
		{/if}
		<div
			class="relative aspect-[5/4] overflow-hidden rounded-card border border-border"
			data-testid="viewer-location-picker"
		>
			<p
				id="viewer-location-map-hint"
				class="pointer-events-none absolute left-2 top-2 m-0 px-2 py-1 rounded-control bg-card/85 font-body font-normal text-[11px] leading-[1.4] text-text-soft shadow-card"
			>
				{$_('viewer_edit_location_hint')}
			</p>
			<MapView
				point={pin}
				cluster={false}
				label={$_('viewer_location_map_label')}
				onpick={pickPoint}
				describedBy="viewer-location-map-hint"
			/>
		</div>
		<p
			class="m-0 font-body font-normal text-[12px] leading-[1.4] text-text-faint tabular-nums"
			aria-live="polite"
			data-testid="viewer-location-coords-line"
		>
			{#if placeName !== ''}
				{placeName} ·
			{/if}{coordsLine !== '' ? coordsLine : $_('viewer_edit_location_no_pin')}
		</p>
		{#if errorKey !== ''}
			<p
				class="viewer-location-dialog__failure m-0 font-body font-medium text-[13px] leading-[1.4] text-danger-text"
				role="alert"
			>
				{resolveErrorText($_, errorKey)}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (open = false)}>{$_('common_cancel')}</Button>
		<Button variant="primary" disabled={busy || pin === null} onclick={submit}>
			<Check size={20} aria-hidden="true" />
			{$_('viewer_edit_location_submit')}
		</Button>
	{/snippet}
</Modal>
