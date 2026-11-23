import { describe, expect, test } from 'vitest';
import type { PlaceSummary } from '$lib/api/places';
import {
	clusterLabelValues,
	countryName,
	placeYearSpan,
	placeYearsText,
	selectedPlaces,
	selectionCount,
	selectionName
} from './places';

// The place domain's pure helpers: the client's region table, the
// capture span's wording, and the map selection's panel identity.

const paris: PlaceSummary = {
	placeId: 2988507,
	name: 'Paris',
	country: 'FR',
	assetCount: 17,
	latitude: 48.85341,
	longitude: 2.3488,
	takenAtFrom: '2024-01-21T09:05:00Z',
	takenAtTo: '2024-01-25T18:20:00Z',
	representativeCoverAssetIds: []
};

const nice: PlaceSummary = {
	...paris,
	placeId: 2990440,
	name: 'Nice',
	assetCount: 7,
	takenAtFrom: '2024-06-01T09:00:00Z',
	takenAtTo: '2024-06-02T09:00:00Z'
};

const beijing: PlaceSummary = {
	...paris,
	placeId: 1816670,
	name: '北京',
	country: 'CN',
	assetCount: 42,
	takenAtFrom: '2019-04-08T10:12:00Z',
	takenAtTo: '2026-08-02T15:40:00Z'
};

test('countryName maps the ISO code through the locale table', () => {
	expect(countryName('FR', 'en')).toBe('France');
	expect(countryName('CN', 'zh-Hans')).toBe('中国');
	// A code the table rejects outright falls back to the code itself.
	expect(countryName('F', 'en')).toBe('F');
});

test('the capture span answers whole years, one year when both ends agree', () => {
	expect(placeYearSpan(beijing)).toEqual({ from: 2019, to: 2026 });
	expect(placeYearSpan(paris)).toEqual({ from: 2024, to: 2024 });
	// A malformed wire value is no span (never an invented year).
	expect(placeYearSpan({ takenAtFrom: 'nope', takenAtTo: '2024-01-01T00:00:00Z' })).toBeNull();
});

test('the year text is one year or the translated range', () => {
	const range = (from: string, to: string) => `${from}-${to}`;
	const digits = (year: number) => String(year);
	expect(placeYearsText(paris, range, digits)).toBe('2024');
	expect(placeYearsText(beijing, range, digits)).toBe('2019-2026');
	expect(placeYearsText({ takenAtFrom: 'nope', takenAtTo: 'nope' }, range, digits)).toBe('');
});

test('the selection resolves to the listing rows, in listing order', () => {
	expect(selectedPlaces([beijing, paris, nice], [nice.placeId, beijing.placeId])).toEqual([beijing, nice]);
	expect(selectedPlaces([paris], null)).toEqual([]);
	expect(selectedPlaces([paris], [])).toEqual([]);
	expect(selectedPlaces([paris], [999])).toEqual([]);
});

test('the panel head names one place, a shared country, or the area', () => {
	// One place: its own name.
	expect(selectionName([beijing], 'en', 'This area')).toBe('北京');
	// Several places of one country: the country's display name.
	expect(selectionName([paris, nice], 'en', 'This area')).toBe('France');
	expect(selectionName([paris, nice], 'zh-Hans', '这一片')).toBe('法国');
	// A group spanning countries has no single place name: the area
	// label carries it (never a member city standing in for the
	// whole group).
	expect(selectionName([paris, beijing], 'en', 'This area')).toBe('This area');
	expect(selectionName([], 'en', 'This area')).toBe('');
});

test('the selection count sums the places\u2019 honest counts', () => {
	expect(selectionCount([paris, nice])).toBe(24);
	expect(selectionCount([])).toBe(0);
});

describe('the selection name never depends on the request order', () => {
	test('the same group answers the same name in either order', () => {
		expect(selectionName([nice, paris], 'en', 'This area')).toBe(selectionName([paris, nice], 'en', 'This area'));
	});
});

describe('a count-only bubble says where and how much', () => {
	test('one country answers its display name, the city count and the photo count', () => {
		expect(clusterLabelValues([paris, nice], 'en')).toEqual({ name: 'France', cities: 2, photos: 24 });
	});

	test('a group spanning countries has no single name', () => {
		expect(clusterLabelValues([paris, beijing], 'en')).toEqual({ name: null, cities: 2, photos: 59 });
	});

	test('a single place answers nothing: its named pin already says the name and count', () => {
		expect(clusterLabelValues([paris], 'en')).toBeNull();
	});

	test('the locale follows the message table', () => {
		expect(clusterLabelValues([paris, nice], 'zh-Hans')?.name).toBe('法国');
	});
});
