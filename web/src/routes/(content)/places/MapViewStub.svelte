<script lang="ts">
	// The places page test's map double: the real MapView loads
	// MapLibre lazily and needs a layout-capable host, so the page's
	// selection wiring is exercised against this stand-in -- one pin
	// button per place (plus one bubble-shaped pin for the first two
	// when they share a country), the same onselect hand-back and the
	// same bindable base-layer props.
	import type { PlacePoint } from '$lib/map/tiles';

	type Props = {
		places?: readonly PlacePoint[];
		label: string;
		onselect?: (placeIds: number[]) => void;
		selected?: readonly number[] | null;
		describeCluster?: (placeIds: readonly number[]) => string | null;
		returnFocus?: readonly number[] | null;
		returnFocusToken?: number;
		tiles?: 'loading' | 'ready' | 'unconfigured' | 'unavailable';
		attribution?: string;
	};

	let {
		places = [],
		label,
		onselect,
		selected = null,
		describeCluster,
		returnFocus = null,
		returnFocusToken = 0,
		tiles = $bindable('loading'),
		attribution = $bindable('')
	}: Props = $props();

	function picked(ids: number[]): void {
		onselect?.(ids);
	}

	function pressed(ids: number[]): boolean {
		return selected !== null && ids.length === selected.length && ids.every((id) => selected?.includes(id));
	}
</script>

<div
	class="map map-stub"
	aria-label={label}
	role="region"
	data-tiles={tiles}
	data-attribution={attribution}
	data-return-focus={returnFocus?.join(',') ?? ''}
	data-return-focus-token={returnFocusToken}
	data-describe-cluster={describeCluster === undefined ? 'absent' : 'present'}
>
	{#each places as place (place.placeId)}
		<button
			class="map__pin-stub"
			type="button"
			data-place-id={place.placeId}
			aria-pressed={pressed([place.placeId])}
			onclick={() => picked([place.placeId])}
		>
			{place.name}
			{place.count}
		</button>
	{/each}
	{#if places.length > 1}
		<button
			class="map__pin-stub map__pin-stub--bubble"
			type="button"
			data-place-ids={places.map((place) => place.placeId).join(',')}
			aria-pressed={pressed(places.map((place) => place.placeId))}
			onclick={() => picked(places.map((place) => place.placeId))}
		>
			{places.reduce((sum, place) => sum + place.count, 0)}
		</button>
	{/if}
</div>
