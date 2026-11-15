<script lang="ts">
	// The viewer route test's map double: the info panel's location
	// row renders the same MapView the places tab uses, and the real
	// component loads MapLibre lazily and needs a layout-capable host
	// -- the row's shape is exercised against this stand-in: one
	// region div carrying the label and the base-layer state, with
	// the point it was given.
	type Props = {
		point?: { lat: number; lon: number } | null;
		cluster?: boolean;
		label: string;
		interactive?: boolean;
		tiles?: 'loading' | 'ready' | 'unconfigured' | 'unavailable';
		attribution?: string;
	};

	let {
		point = null,
		cluster = true,
		label,
		interactive = true,
		tiles = $bindable('loading'),
		attribution = $bindable('')
	}: Props = $props();
</script>

<div
	class="map map-stub"
	role="region"
	aria-label={label}
	data-tiles={tiles}
	data-attribution={attribution}
	data-cluster={cluster}
	data-interactive={interactive}
	data-point={point === null ? '' : JSON.stringify(point)}
></div>
