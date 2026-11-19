// The map module's tests: the tile source resolution (both document
// shapes, the refusals), the clustering rule, and the pins' GeoJSON and
// bounds. Everything here runs without the map library and without a
// network: the fetch is a fixture.

import { describe, expect, test } from 'vitest';
import {
	clusterPins,
	emptyStyle,
	placesBounds,
	placesFeatureCollection,
	resolveTileSource,
	tileSourceForTheme,
	type MapSources,
	type PlacePoint
} from './tiles';

/** A fixture response with one JSON body. */
function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const paris: PlacePoint = { placeId: 2988507, name: 'Paris', count: 17, lat: 48.85341, lon: 2.3488 };
const beijing: PlacePoint = { placeId: 1816670, name: '北京', count: 42, lat: 39.9075, lon: 116.39723 };

describe('resolveTileSource', () => {
	test('a style document is passed through with its sources attribution', async () => {
		const style = {
			version: 8,
			sources: {
				openmaptiles: { type: 'vector', url: 'https://tiles.example.net/planet.json', attribution: '© Example' }
			},
			layers: [{ id: 'water', type: 'fill', source: 'openmaptiles', 'source-layer': 'water' }]
		};
		const source = await resolveTileSource('https://tiles.example.net/style.json', async () =>
			jsonResponse(200, style)
		);
		expect(source.ok).toBe(true);
		if (!source.ok) {
			return;
		}
		expect(source.source.kind).toBe('style');
		expect(source.source.style).toEqual(style);
		expect(source.source.attribution).toBe('© Example');
	});

	test('a TileJSON descriptor becomes a one-layer raster style', async () => {
		const tilejson = {
			tilejson: '3.0.0',
			tiles: ['https://tiles.example.net/osm/{z}/{x}/{y}.png'],
			tileSize: 512,
			minzoom: 2,
			maxzoom: 7,
			attribution: '© OSM contributors'
		};
		const source = await resolveTileSource('https://tiles.example.net/tilejson.json', async () =>
			jsonResponse(200, tilejson)
		);
		expect(source.ok).toBe(true);
		if (!source.ok) {
			return;
		}
		expect(source.source.kind).toBe('tiles');
		expect(source.source.attribution).toBe('© OSM contributors');
		const raster = source.source.style.sources.base as { type: string; tiles: string[]; tileSize: number };
		expect(raster.type).toBe('raster');
		expect(raster.tiles).toEqual(tilejson.tiles);
		expect(raster.tileSize).toBe(512);
		expect(source.source.style.layers[0]).toMatchObject({ id: 'base', type: 'raster', source: 'base' });
	});

	test('a failing or meaningless document is the unavailable state', async () => {
		const cases: Array<[string, () => Promise<Response>]> = [
			['http error', async () => jsonResponse(503, {})],
			['not json', async () => new Response('<html>nope</html>', { status: 200 })],
			['an empty object', async () => jsonResponse(200, {})],
			['a descriptor without tiles', async () => jsonResponse(200, { tilejson: '3.0.0' })],
			['a network failure', async () => Promise.reject(new Error('offline'))]
		];
		for (const [name, fetcher] of cases) {
			const source = await resolveTileSource('https://tiles.example.net/source.json', fetcher);
			expect(source.ok, name).toBe(false);
		}
	});

	test('the empty base style is a real style with no layers', () => {
		const style = emptyStyle();
		expect(style.version).toBe(8);
		expect(style.layers).toEqual([]);
		expect(style.sources).toEqual({});
	});
});

describe('tileSourceForTheme', () => {
	const light = 'https://tiles.example.net/bright';
	const dark = 'https://tiles.example.net/dark';
	const sources = (overrides: Partial<MapSources> = {}): MapSources => ({
		light,
		dark,
		lightGcj02: false,
		darkGcj02: false,
		...overrides
	});

	test('a dark theme with a dark source loads it', () => {
		expect(tileSourceForTheme(sources(), true)).toEqual({ url: dark, gcj02: false });
	});

	test('a light theme always loads the light source', () => {
		expect(tileSourceForTheme(sources(), false)).toEqual({ url: light, gcj02: false });
	});

	test('the coordinate flag travels with the source that draws', () => {
		// A GCJ-02 light basemap and a standard dark one: each theme
		// takes its own source's datum.
		const mixed = sources({ lightGcj02: true, darkGcj02: false });
		expect(tileSourceForTheme(mixed, false)).toEqual({ url: light, gcj02: true });
		expect(tileSourceForTheme(mixed, true)).toEqual({ url: dark, gcj02: false });
	});

	test('an empty dark source makes the dark theme reuse the light one and its flag', () => {
		expect(tileSourceForTheme(sources({ dark: '' }), true)).toEqual({ url: light, gcj02: false });
		expect(tileSourceForTheme(sources({ dark: null, lightGcj02: true }), true)).toEqual({
			url: light,
			gcj02: true
		});
	});

	test('only a dark source configured: the light theme is unconfigured, the dark one loads it', () => {
		expect(tileSourceForTheme(sources({ light: '' }), false)).toBeNull();
		expect(tileSourceForTheme(sources({ light: null, darkGcj02: true }), true)).toEqual({
			url: dark,
			gcj02: true
		});
	});

	test('neither configured is the degraded state', () => {
		expect(tileSourceForTheme(sources({ light: '', dark: '' }), true)).toBeNull();
		expect(tileSourceForTheme(sources({ light: null, dark: null }), false)).toBeNull();
		expect(tileSourceForTheme(sources({ light: '', dark: null }), true)).toBeNull();
	});
});

describe('clusterPins', () => {
	const pin = (placeId: number, count: number, x: number, y: number) => ({
		placeId,
		name: `City ${placeId}`,
		count,
		x,
		y
	});

	test('distant pins stay separate and keep their names', () => {
		const markers = clusterPins([pin(1, 5, 0, 0), pin(2, 9, 200, 200)], 56);
		expect(markers).toHaveLength(2);
		// The order is the grouping's stable rule: heaviest city first.
		expect(markers.map((marker) => marker.name)).toEqual(['City 2', 'City 1']);
		expect(markers[0]?.placeIds).toEqual([2]);
		expect(markers[1]?.placeIds).toEqual([1]);
	});

	test('nearby pins merge into one bubble summing their photos', () => {
		const markers = clusterPins([pin(1, 5, 0, 0), pin(2, 9, 10, 10), pin(3, 20, 400, 400)], 56);
		expect(markers).toHaveLength(2);
		const bubble = markers.find((marker) => marker.name === null);
		expect(bubble?.count).toBe(14);
		expect(bubble?.placeIds).toEqual([1, 2]);
	});

	test('the same camera always yields the same markers', () => {
		const pins = [pin(3, 1, 100, 100), pin(1, 50, 105, 102), pin(2, 2, 108, 99)];
		const first = clusterPins(pins, 56);
		const second = clusterPins(pins, 56);
		expect(first).toEqual(second);
		// The bubble keeps the heaviest city as its anchor: the sort is
		// by count, so the grouping does not depend on input order.
		expect(first[0]?.placeIds).toEqual([1, 2, 3]);
	});
});

describe('the pins as GeoJSON', () => {
	test('one feature per place with [lon, lat] coordinates', () => {
		const collection = placesFeatureCollection([paris, beijing]);
		expect(collection.type).toBe('FeatureCollection');
		expect(collection.features).toHaveLength(2);
		expect(collection.features[0]?.geometry).toEqual({ type: 'Point', coordinates: [2.3488, 48.85341] });
		expect(collection.features[0]?.properties).toEqual({ placeId: 2988507, name: 'Paris', count: 17 });
	});

	test('the bounds span every pin, null when empty', () => {
		expect(placesBounds([])).toBeNull();
		expect(placesBounds([paris, beijing])).toEqual([2.3488, 39.9075, 116.39723, 48.85341]);
	});
});
