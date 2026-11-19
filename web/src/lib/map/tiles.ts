// The map's tile source resolution: the one configured URL is either a
// style document (the vector-and-style form: sources, layers, sprites
// and glyphs) or a TileJSON descriptor (the raster form: tile
// templates plus attribution). The browser fetches the document (the
// host must send the usual cross-origin headers -- vector styles need
// them for glyphs and sprites anyway), and this module turns it into
// the style the map library renders.
//
// The empty base style is the honest-degradation foundation: a style
// with no base layer at all. The pins are added on top of it, so an
// instance with no tile source (or a source that cannot be loaded)
// still shows where the photos are and states the absence of the base
// layer -- never a blank canvas that pretends there is nothing to see.
//
// Everything here is pure and testable without the map library: the
// maplibre import is type-only and erased at build time.

import type { SourceSpecification, StyleSpecification } from 'maplibre-gl';

/** The pins' GeoJSON properties. */
export type PlacePoint = {
	placeId: number;
	name: string;
	count: number;
	lat: number;
	lon: number;
};

/** One resolved tile source. */
export type TileSource = {
	/** The style the map renders (the configured document or its raster wrap). */
	style: StyleSpecification;
	/** The attribution line the tile host requires, '' when none is declared. */
	attribution: string;
	/** Which form the document took. */
	kind: 'style' | 'tiles';
};

/** The outcome of resolving the configured URL. */
export type TileSourceResult = { ok: true; source: TileSource } | { ok: false };

/**
 * The configured source pair: the two URLs (null or the empty string
 * when unconfigured). The library is WGS-84 only; the source's own
 * datum is the administrator's concern, warned about in the settings.
 */
export type MapSources = {
	light: string | null;
	dark: string | null;
};

/** The source one theme draws. */
export type ActiveTileSource = { url: string };

/**
 * The tile source for one theme: the dark style when the theme is
 * dark AND one is configured, the light style otherwise (the dark
 * theme reuses the light source when it has none of its own). Null
 * when neither is configured -- the map's stated absence.
 *
 * Both URLs use the empty string for "unconfigured" as they arrive
 * from the config read (the read flattens the contract's null into
 * it), so a whitespace-free comparison is all the rule needs.
 */
export function tileSourceForTheme(sources: MapSources, darkTheme: boolean): ActiveTileSource | null {
	if (darkTheme && sources.dark !== null && sources.dark !== '') {
		return { url: sources.dark };
	}
	if (sources.light !== null && sources.light !== '') {
		return { url: sources.light };
	}
	return null;
}

/** The base style with no basemap: the degradation state's canvas. */
export function emptyStyle(): StyleSpecification {
	return { version: 8, sources: {}, layers: [] };
}

/** The raster wrap of one TileJSON descriptor. */
function rasterStyle(tilejson: Record<string, unknown>): StyleSpecification {
	const tileSize = typeof tilejson.tileSize === 'number' ? tilejson.tileSize : 256;
	const attribution = typeof tilejson.attribution === 'string' ? tilejson.attribution : '';
	const source: SourceSpecification = {
		type: 'raster',
		tiles: Array.isArray(tilejson.tiles) ? (tilejson.tiles as string[]) : [],
		tileSize
	};
	if (attribution !== '') {
		source.attribution = attribution;
	}
	if (typeof tilejson.minzoom === 'number') {
		source.minzoom = tilejson.minzoom;
	}
	if (typeof tilejson.maxzoom === 'number') {
		source.maxzoom = tilejson.maxzoom;
	}
	return {
		version: 8,
		sources: { base: source },
		layers: [{ id: 'base', type: 'raster', source: 'base' }]
	};
}

/** The attribution of one style document, joined over its sources. */
function styleAttribution(style: StyleSpecification): string {
	const lines: string[] = [];
	for (const source of Object.values(style.sources ?? {})) {
		const attribution = (source as { attribution?: unknown }).attribution;
		if (typeof attribution === 'string' && attribution !== '' && !lines.includes(attribution)) {
			lines.push(attribution);
		}
	}
	return lines.join(' · ');
}

/** Whether the document is a style (not merely a descriptor). */
function isStyleDocument(document: Record<string, unknown>): boolean {
	return document.version === 8 && Array.isArray(document.layers) && typeof document.sources === 'object';
}

/**
 * Resolve the configured tile source URL into the style the map
 * renders. The configured address is loaded once; a document that is
 * neither a style nor a TileJSON descriptor is the unavailable state,
 * and so is any transport failure. Callers keep drawing the pins over
 * emptyStyle() in both cases.
 */
export async function resolveTileSource(
	url: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<TileSourceResult> {
	let response: Response;
	try {
		response = await fetcher(url, { headers: { Accept: 'application/json' } });
	} catch {
		return { ok: false };
	}
	if (!response.ok) {
		return { ok: false };
	}
	let document: unknown;
	try {
		document = await response.json();
	} catch {
		return { ok: false };
	}
	if (document === null || typeof document !== 'object') {
		return { ok: false };
	}
	const record = document as Record<string, unknown>;
	if (isStyleDocument(record)) {
		const style = document as StyleSpecification;
		return { ok: true, source: { style, attribution: styleAttribution(style), kind: 'style' } };
	}
	if (Array.isArray(record.tiles) && record.tiles.length > 0) {
		const attribution = typeof record.attribution === 'string' ? record.attribution : '';
		return { ok: true, source: { style: rasterStyle(record), attribution, kind: 'tiles' } };
	}
	return { ok: false };
}

/** One pin projected onto the map's canvas. */
export type ProjectedPin = {
	placeId: number;
	name: string;
	count: number;
	x: number;
	y: number;
};

/** One rendered pin or cluster bubble. */
export type PinMarker = {
	key: string;
	x: number;
	y: number;
	count: number;
	/** The city name, null for a bubble (which stands for several). */
	name: string | null;
	/** The place ids the marker stands for: one pin's, or a bubble's members. */
	placeIds: number[];
};

/**
 * Group the projected pins into markers at one camera position: pins
 * whose canvas distance is within radius of a group's center join it,
 * and a group of one renders as that city's pin. The order is stable
 * (heaviest city first, then the place id), so the same camera always
 * yields the same markers -- a bubble keeps the anchor of its largest
 * city instead of jumping between frames.
 */
export function clusterPins(pins: readonly ProjectedPin[], radius: number): PinMarker[] {
	const groups: { x: number; y: number; members: ProjectedPin[] }[] = [];
	const ordered = [...pins].sort((a, b) => b.count - a.count || a.placeId - b.placeId);
	for (const pin of ordered) {
		const group = groups.find((candidate) => Math.hypot(candidate.x - pin.x, candidate.y - pin.y) <= radius);
		if (group === undefined) {
			groups.push({ x: pin.x, y: pin.y, members: [pin] });
			continue;
		}
		group.members.push(pin);
		group.x = group.members.reduce((sum, member) => sum + member.x, 0) / group.members.length;
		group.y = group.members.reduce((sum, member) => sum + member.y, 0) / group.members.length;
	}
	return groups.map((group) => {
		const placeIds = group.members.map((member) => member.placeId).sort((a, b) => a - b);
		if (group.members.length === 1) {
			const member = group.members[0]!;
			return {
				key: `place-${member.placeId}`,
				x: group.x,
				y: group.y,
				count: member.count,
				name: member.name,
				placeIds
			};
		}
		return {
			key: `cluster-${placeIds.join('-')}`,
			x: group.x,
			y: group.y,
			count: group.members.reduce((sum, member) => sum + member.count, 0),
			name: null,
			placeIds
		};
	});
}

/** The pins as one GeoJSON FeatureCollection for the map's source. */
export function placesFeatureCollection(places: readonly PlacePoint[]): GeoJSON.FeatureCollection {
	return {
		type: 'FeatureCollection',
		features: places.map((place) => ({
			type: 'Feature',
			geometry: { type: 'Point', coordinates: [place.lon, place.lat] },
			properties: {
				placeId: place.placeId,
				name: place.name,
				count: place.count
			}
		}))
	};
}

/** The [west, south, east, north] bounds of the points, null when empty. */
export function placesBounds(
	places: readonly Pick<PlacePoint, 'lat' | 'lon'>[]
): [number, number, number, number] | null {
	if (places.length === 0) {
		return null;
	}
	let west = places[0]!.lon;
	let east = west;
	let south = places[0]!.lat;
	let north = south;
	for (const place of places) {
		west = Math.min(west, place.lon);
		east = Math.max(east, place.lon);
		south = Math.min(south, place.lat);
		north = Math.max(north, place.lat);
	}
	return [west, south, east, north];
}
