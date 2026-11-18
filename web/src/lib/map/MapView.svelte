<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { onMount } from 'svelte';
	import type { Map as MapLibreMap, StyleSpecification } from 'maplibre-gl';
	import { getMapConfig } from '$lib/api/mapConfig';
	import { isDarkTheme } from '$lib/theme/theme.svelte';
	import {
		clusterPins,
		emptyStyle,
		placesBounds,
		resolveTileSource,
		tileSourceForTheme,
		type PinMarker,
		type PlacePoint
	} from '$lib/map/tiles';

	/** The base layer's state, as the notice renders it. */
	export type MapState = 'loading' | 'ready' | 'unconfigured' | 'unavailable';

	type Props = {
		/** The pins: one per place, in the map's own coordinates. */
		places?: readonly PlacePoint[];
		/** One asset's location: the viewer's mini map. */
		point?: { lat: number; lon: number } | null;
		/** Whether to group nearby pins by zoom (the places map). */
		cluster?: boolean;
		/** The map's accessible name. */
		label: string;
		/**
		 * Whether the map answers gestures. The panel's location row
		 * is a STATIC display (the design pins it: no drag, no zoom,
		 * no click) -- the false form keeps the same tiles, the same
		 * pin and the same honest degradation, minus the handlers.
		 */
		interactive?: boolean;
		/**
		 * When present, one map click hands the picked coordinates
		 * over (the edit-location dialog's pin): the pointer gesture
		 * lands a point the caller owns. Longitude wraps into
		 * [-180, 180]; the Web Mercator canvas already bounds the
		 * latitude.
		 */
		onpick?: (at: { lat: number; lon: number }) => void;
		/** The id of the element describing the map's gesture (the picker's hint). */
		describedBy?: string;
		/**
		 * The place ids the view frames: a selected marker's places.
		 * Null (or absent) frames every pin -- the map's resting
		 * view. A selection change re-frames the camera around
		 * exactly those places.
		 */
		focus?: readonly number[] | null;
		/** The selected marker's place ids (its pin renders the pressed state). */
		selected?: readonly number[] | null;
		/** A cluster or pin click hands back the place ids it stands for. */
		onselect?: (placeIds: number[]) => void;
		/** The base layer's state (bindable: the page states it too). */
		tiles?: MapState;
		/** The tile host's attribution line (bindable for the page's footer). */
		attribution?: string;
	};

	let {
		places = [],
		point = null,
		cluster = true,
		label,
		interactive = true,
		onpick,
		describedBy,
		focus = null,
		selected = null,
		onselect,
		tiles = $bindable<MapState>('loading'),
		attribution = $bindable('')
	}: Props = $props();

	/** One rendered pin or cluster bubble. */
	type Marker = PinMarker;

	let container = $state<HTMLDivElement>();
	let map: MapLibreMap | null = null;
	let markers = $state<Marker[]>([]);
	/**
	 * The single-asset mini map's one marker. A state field, not a
	 * derived: the map handle is a plain variable (the library is
	 * loaded lazily, never a signal), so a derivation would cache the
	 * `null` of the first render and never see the map arrive. The
	 * refresh writes it beside the pins instead.
	 */
	let single = $state<Marker | null>(null);
	/** True while the map has not finished its first layout. */
	let ready = $state(false);
	/** The container observer (the 70/30 squeeze's canvas resize). */
	let resizeObserver: ResizeObserver | undefined;
	/**
	 * The configured URLs, once the read lands: the light style and
	 * the optional dark one (null when unconfigured). The theme
	 * effect reads this to pick a source on a live switch.
	 */
	let sources = $state<{ light: string | null; dark: string | null } | null>(null);
	/**
	 * The URL whose base layer the map currently renders (null = the
	 * empty style). The theme effect compares against it so a switch
	 * that resolves to the same source does not reload the tiles.
	 */
	let appliedUrl: string | null = null;
	/**
	 * The in-flight source switch's generation: a theme flipped twice
	 * while a document loads keeps only the latest answer.
	 */
	let sourceGeneration = 0;
	/**
	 * True while the applied base layer is the stated absence because
	 * the configured document could not be resolved. The idle handler
	 * must not read an empty style as "tiles loaded": that would clear
	 * the honest notice a moment after it appears.
	 */
	let sourceUnresolved = false;

	/**
	 * Recompute the overlay from the map's current camera: project
	 * every pin, group nearby ones when clustering is on, and hand the
	 * list to the template. Runs on every camera move.
	 */
	function refreshMarkers(): void {
		if (map === null) {
			return;
		}
		const projected = places.map((place) => {
			const pixel = map!.project([place.lon, place.lat]);
			return { placeId: place.placeId, name: place.name, count: place.count, x: pixel.x, y: pixel.y };
		});
		single = point === null ? null : assetMarker(map, point);
		if (cluster) {
			markers = clusterPins(projected, 56);
			return;
		}
		markers = projected.map((pin) => ({
			key: `place-${pin.placeId}`,
			x: pin.x,
			y: pin.y,
			count: pin.count,
			name: null,
			placeIds: [pin.placeId]
		}));
	}

	/** The mini map's one marker: the asset's own point, projected. */
	function assetMarker(instance: MapLibreMap, at: NonNullable<Props['point']>): Marker {
		const pixel = instance.project([at.lon, at.lat]);
		return { key: 'asset', x: pixel.x, y: pixel.y, count: 0, name: null, placeIds: [] };
	}

	/**
	 * Frame a set of places: the camera fits their bounding box with
	 * the container's padding, which is the aspect rule (the box
	 * fills the view as far as the container's own ratio allows --
	 * nothing inside it is cropped, and the map's own surface fills
	 * the rest). Null frames every pin. A single place has no
	 * extent, so it gets a small frame around its point instead of a
	 * collapsed box.
	 */
	function frameOn(ids: readonly number[] | null): void {
		if (map === null) {
			return;
		}
		// The container may have just changed size (the panel's
		// squeeze): resize first so the fit reads the fresh viewport.
		map.resize();
		const wanted = ids === null ? [...places] : places.filter((place) => ids.includes(place.placeId));
		const bounds = placesBounds(wanted);
		if (bounds === null) {
			return;
		}
		const [west, south, east, north] = bounds;
		if (west === east || south === north) {
			map.fitBounds(
				[
					[west - 0.4, south - 0.3],
					[east + 0.4, north + 0.3]
				],
				{ padding: 64, maxZoom: 9, animate: true }
			);
			return;
		}
		map.fitBounds(
			[
				[west, south],
				[east, north]
			],
			{ padding: 64, maxZoom: 8, animate: true }
		);
	}

	/** Whether one marker is the current selection (the aria state). */
	function isSelected(marker: Marker): boolean {
		if (selected === null || marker.placeIds.length !== selected.length) {
			return false;
		}
		return marker.placeIds.every((id) => selected.includes(id));
	}

	// The last camera frame the focus prop requested: a fresh frame
	// runs only when the SELECTION moves, never on every re-render
	// (a convergence refetch must not yank a camera the reader
	// panned).
	let framedKey = '';

	/**
	 * Swap the base layer in place: resolve the URL (null = the empty
	 * style, the stated absence) and set it on the live map. The
	 * camera is not part of the style -- the library keeps the center,
	 * zoom and bearing across a setStyle -- and the pins are DOM
	 * markers outside the style, so a switch moves neither.
	 */
	async function applySource(url: string | null): Promise<void> {
		const generation = ++sourceGeneration;
		if (url === null) {
			appliedUrl = null;
			sourceUnresolved = false;
			attribution = '';
			map?.setStyle(emptyStyle());
			tiles = 'unconfigured';
			return;
		}
		const source = await resolveTileSource(url);
		if (generation !== sourceGeneration) {
			return;
		}
		appliedUrl = url;
		sourceUnresolved = !source.ok;
		if (map === null) {
			return;
		}
		if (source.ok) {
			map.setStyle(source.source.style);
			attribution = source.source.attribution;
			tiles = 'ready';
			return;
		}
		map.setStyle(emptyStyle());
		attribution = '';
		tiles = 'unavailable';
	}

	// The instance: the config read, the source resolution, the lazy
	// library load, then the pins. One mount, one instance: the maps
	// are page-level surfaces, not widgets that come and go.
	onMount(() => {
		let disposed = false;
		void (async () => {
			const config = await getMapConfig();
			if (disposed) {
				return;
			}
			let style: StyleSpecification = emptyStyle();
			if (!config.ok) {
				tiles = 'unavailable';
			} else {
				sources = { light: config.tilejsonUrl, dark: config.darkTilejsonUrl };
				const wanted = tileSourceForTheme(config.tilejsonUrl, config.darkTilejsonUrl, isDarkTheme());
				if (wanted === null) {
					tiles = 'unconfigured';
				} else {
					const source = await resolveTileSource(wanted);
					if (disposed) {
						return;
					}
					// The applied URL is recorded even when the document
					// fails: the degradation is the answer for THIS source,
					// and the theme effect must not retry it on every run.
					appliedUrl = wanted;
					if (source.ok) {
						style = source.source.style;
						sourceUnresolved = false;
						attribution = source.source.attribution;
						tiles = 'ready';
					} else {
						sourceUnresolved = true;
						tiles = 'unavailable';
					}
				}
			}
			if (container === undefined) {
				return;
			}
			const maplibre = await import('maplibre-gl');
			await import('maplibre-gl/dist/maplibre-gl.css');
			if (disposed) {
				return;
			}
			const instance = new maplibre.Map({
				container,
				style,
				attributionControl: false,
				// The static form (the panel's location row) answers no
				// gesture: one prop, the same tiles and pins behind it.
				interactive,
				// The map is a browse surface: no rotation, no pitch --
				// a family looking at "where" wants north up.
				dragRotate: false,
				pitchWithRotate: false,
				touchPitch: false,
				...(cluster ? { maxZoom: 12 } : { maxZoom: 18 })
			});
			map = instance;

			// A tile or style fetch that fails is the honest state: the
			// notice appears, the pins stay. A later successful load
			// clears it again (a flaky link should not leave a stale
			// claim) -- unless the DOCUMENT itself never resolved, in
			// which case the empty style has nothing to load and the
			// notice is the answer.
			instance.on('error', () => {
				if (tiles === 'ready') {
					tiles = 'unavailable';
				}
			});
			instance.on('idle', () => {
				if (tiles === 'unavailable' && !sourceUnresolved && instance.areTilesLoaded()) {
					tiles = 'ready';
				}
			});
			instance.on('load', () => {
				ready = true;
				const bounds = placesBounds(places);
				framedKey = '';
				if (point !== null) {
					instance.jumpTo({ center: [point.lon, point.lat], zoom: 12 });
				} else if (bounds !== null) {
					instance.fitBounds(
						[
							[bounds[0], bounds[1]],
							[bounds[2], bounds[3]]
						],
						{ padding: 48, maxZoom: 7, animate: false }
					);
				}
				refreshMarkers();
			});
			instance.on('move', refreshMarkers);
			// The picker's arm: one click hands the clicked point over
			// (the pin follows through the point prop). The wrap keeps
			// the longitude in the contract's range.
			if (onpick !== undefined) {
				instance.on('click', (event) => {
					const at = event.lngLat.wrap();
					onpick({ lat: at.lat, lon: at.lng });
				});
			}

			// The map's container is a flex cell: opening the place
			// panel narrows it, and the library's own resize listener
			// watches the window, not the container. A container
			// observer keeps the canvas and the projected pins honest
			// across the 70/30 squeeze.
			if (typeof ResizeObserver === 'function') {
				const observer = new ResizeObserver(() => {
					instance.resize();
					refreshMarkers();
				});
				observer.observe(container);
				resizeObserver = observer;
			}
		})();
		return () => {
			disposed = true;
			resizeObserver?.disconnect();
			resizeObserver = undefined;
			map?.remove();
			map = null;
		};
	});

	// The pins converge in place: an SSE refetch replaces the array
	// and the overlay follows the fresh answer without re-creating the
	// map (the source rows are the same places; the counts move).
	$effect(() => {
		void places;
		void point;
		if (ready) {
			refreshMarkers();
		}
	});

	// The theme switch: a light/dark flip re-picks the source and
	// swaps the base layer in place. No page reload: the pins are DOM
	// markers and the camera is not part of the style, so both
	// survive the swap. A switch that resolves to the source already
	// loaded (a light-only configuration, or a theme with no dark
	// style of its own) does nothing.
	$effect(() => {
		const dark = isDarkTheme();
		if (!ready || sources === null) {
			return;
		}
		const url = tileSourceForTheme(sources.light, sources.dark, dark);
		if (url === appliedUrl) {
			return;
		}
		void applySource(url);
	});

	// The view framing: a selection move re-frames the camera around
	// that marker's places; clearing it returns to every pin. The key
	// comparison keeps a pan or a convergence from re-framing under
	// the reader.
	$effect(() => {
		const ids = focus;
		const key = ids === null || ids.length === 0 ? '' : [...ids].sort((a, b) => a - b).join(',');
		if (!ready || key === framedKey) {
			return;
		}
		framedKey = key;
		frameOn(ids === null || ids.length === 0 ? null : ids);
	});

	/** A pin or bubble click: the place ids it stands for. */
	/**
	 * The pin's visible size by photo count, the design's four-step
	 * ladder: 26 / 30 / 34 / 40 px. The value is a MINIMUM width, not a
	 * cap -- a four-digit count widens the circle into a capsule instead
	 * of clipping the digits.
	 */
	function pinDiameter(count: number): number {
		if (count < 10) return 26;
		if (count < 100) return 30;
		if (count < 1000) return 34;
		return 40;
	}

	function pick(marker: Marker): void {
		if (onselect !== undefined && marker.placeIds.length > 0) {
			onselect(marker.placeIds);
		}
	}
</script>

<!--
	The map surface: one MapLibre instance over the instance's tile
	source, with the places' pins drawn as an HTML overlay.

	THE TWO HALVES OF THE PRIVACY LINE: the base layer is loaded by the
	BROWSER straight from the tile host (the server never proxies a
	tile), while the pins come from the app's own places read -- a
	thumbnail link and a city name never cross the tile line. The map
	only tells the tile host what any map tells it: a viewport, a zoom
	level and the device's address.

	WHY THE PINS ARE HTML: cluster counts and city names are text, and a
	symbol layer needs a glyph endpoint the tile source may not carry.
	Drawing the pins as buttons over the canvas keeps the names in the
	instance's own language (the labels come from our read endpoint, not
	from baked-in basemap text), keeps them clickable and accessible,
	and lets the clustering rule stay in one tested function.

	THE DEGRADATION: no tile source, an unreachable source, or tiles
	failing to load all land on the same honest state -- the base layer
	is gone, the pins stay, and the notice says which part is missing.
	The empty base style is a real style with no layers, so the library
	never has to invent anything.

	THE THEME PAIR: the administrator may configure a dark style beside
	the light one. A live theme switch re-picks the source and swaps the
	base layer through setStyle -- no reload, and the pins (DOM markers
	outside the style) and the camera survive it.

	The library is loaded lazily: the wall pages never pay for it, and
	the two map surfaces (the places tab and the viewer's location card)
	import the same component.
-->

<div
	class="map relative h-full w-full overflow-hidden bg-[color-mix(in_srgb,var(--color-text)_3%,var(--color-card))]"
	aria-label={label}
	role="region"
	aria-describedby={describedBy}
>
	<!-- The map library's own stylesheet gives its container
	     `position: relative`. A positioned utility here (absolute
	     inset-0) therefore loses the cascade: the element is relative,
	     has no in-flow content of its own (the canvases it grows are
	     absolutely positioned), and collapses to zero height -- the
	     base tiles load fine and never draw. Size the container by
	     height and width instead: it fills the frame either way. -->
	<div class="map__canvas h-full w-full" bind:this={container}></div>

	<!-- The pins: HTML buttons over the canvas, positioned per camera
	     frame. The names and counts come from this instance's own read. -->
	<div class="map__pins absolute inset-0 pointer-events-none">
		{#each markers as marker (marker.key)}
			<!-- A city pin: the count pill with the city's name beneath it
			     (the name is ours, never the basemap's). A bubble: the
			     same pill in the accent tone, the group's photo count. -->
			<button
				class="map__pin map__pin--{marker.name === null
					? 'cluster'
					: 'place'} pointer-events-auto absolute -translate-x-1/2 -translate-y-1/2 inline-flex flex-col items-center gap-1 border-0 bg-transparent p-0 whitespace-nowrap cursor-pointer"
				style="left: {marker.x}px; top: {marker.y}px"
				type="button"
				data-place-count={marker.count}
				data-place-ids={marker.placeIds.join(',')}
				aria-pressed={isSelected(marker)}
				aria-label={marker.name === null
					? $_('places_map_pin_count', { values: { count: marker.count } })
					: $_('places_map_pin_label', { values: { name: marker.name, count: marker.count } })}
				title={marker.name === null
					? $_('places_map_pin_count', { values: { count: marker.count } })
					: $_('places_map_pin_label', { values: { name: marker.name, count: marker.count } })}
				onclick={() => pick(marker)}
			>
				<!-- Only the number: a map point already says "photos
				     here", and the icon is that sentence twice. The
				     size reads the count; the background carries the
				     identity (accent = a group, card = a named city)
				     and keeps the digits legible over the basemap. -->
				<span
					class="map__pin-pill inline-flex items-center justify-center rounded-full font-body font-semibold text-[13px] leading-[1.4] tabular-nums transition-[scale,border-color] duration-(--motion-fast) ease-standard hover:scale-105 {marker.name ===
					null
						? 'bg-accent text-accent-on shadow-overlay'
						: 'bg-card text-text border border-border-strong shadow-card'} {isSelected(marker)
						? 'ring-2 ring-accent'
						: marker.name === null
							? 'ring-2 ring-card'
							: ''}"
					style="min-width: {pinDiameter(marker.count)}px; height: {pinDiameter(
						marker.count
					)}px; padding-inline: 6px"
				>
					{marker.count}
				</span>
				{#if marker.name !== null}
					<span
						class="map__pin-name font-body font-semibold text-[12px] leading-[1.4] text-text [text-shadow:0_1px_2px_var(--color-card)]"
						>{marker.name}</span
					>
				{/if}
			</button>
		{/each}
		{#if single !== null}
			<span
				class="map__asset-pin pointer-events-none absolute -translate-x-1/2 -translate-y-1/2 inline-flex items-center justify-center w-4 h-4 rounded-full bg-accent ring-2 ring-card"
				style="left: {single.x}px; top: {single.y}px"
				aria-hidden="true"
			></span>
		{/if}
	</div>

	<!-- The honest degradation notice: the base layer's absence is
	     stated, and the pins are named as still available. -->
	{#if tiles === 'unconfigured'}
		<p
			class="map__notice absolute top-3 start-3 m-0 inline-flex items-center gap-1.5 px-2.5 py-1 rounded-control bg-notice text-navy-900 font-body font-semibold text-[12px] leading-[1.4]"
			role="status"
			data-map-state="unconfigured"
		>
			{$_('places_map_tiles_unconfigured')}
		</p>
	{:else if tiles === 'unavailable'}
		<p
			class="map__notice absolute top-3 start-3 m-0 inline-flex items-center gap-1.5 px-2.5 py-1 rounded-control bg-notice text-navy-900 font-body font-semibold text-[12px] leading-[1.4]"
			role="status"
			data-map-state="unavailable"
		>
			{$_('places_map_tiles_unavailable')}
		</p>
	{:else if tiles === 'loading'}
		<p
			class="map__notice absolute top-3 start-3 m-0 px-2.5 py-1 rounded-control bg-card text-text-soft border border-border font-body font-normal text-[12px] leading-[1.4]"
			role="status"
			data-map-state="loading"
		>
			{$_('places_map_loading')}
		</p>
	{/if}
</div>
