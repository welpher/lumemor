// The place domain's pure helpers: the country's display name, a
// place's capture span, and the map selection's panel identity. They
// are shared by the places page (the list cards, the map panel's
// head) and the place detail page, so the two faces can never word
// the same fact two ways.

import type { PlaceSummary } from '$lib/api/places';

/**
 * The country's display name: the client's locale table maps the ISO
 * code (the contract leaves the name to the client). A host without
 * the region data falls back to the code itself -- the honest label,
 * never a blank.
 */
export function countryName(code: string, activeLocale: string): string {
	try {
		const names = new Intl.DisplayNames([activeLocale], { type: 'region' });
		return names.of(code) ?? code;
	} catch {
		return code;
	}
}

/** The UTC calendar year of one wire takenAt; null when unparseable. */
export function yearOf(takenAt: string): number | null {
	const time = Date.parse(takenAt);
	if (Number.isNaN(time)) {
		return null;
	}
	return new Date(time).getUTCFullYear();
}

/**
 * The place's capture span in calendar years: the earliest member's
 * year and the latest member's year. Null only for a malformed wire
 * value (the contract always carries both ends).
 */
export function placeYearSpan(place: Pick<PlaceSummary, 'takenAtFrom' | 'takenAtTo'>): {
	from: number;
	to: number;
} | null {
	const from = yearOf(place.takenAtFrom);
	const to = yearOf(place.takenAtTo);
	if (from === null || to === null) {
		return null;
	}
	return { from: Math.min(from, to), to: Math.max(from, to) };
}

/**
 * The place's year text: one year when both ends fall in the same
 * calendar year, the range otherwise. The separator is the caller's
 * translated range pattern (the years themselves are locale digits
 * through the translation's own value interpolation).
 */
export function placeYearsText(
	place: Pick<PlaceSummary, 'takenAtFrom' | 'takenAtTo'>,
	range: (from: string, to: string) => string,
	digits: (year: number) => string
): string {
	const span = placeYearSpan(place);
	if (span === null) {
		return '';
	}
	const from = digits(span.from);
	if (span.from === span.to) {
		return from;
	}
	return range(from, digits(span.to));
}

/** The places one marker selection stands for, in the listing order. */
export function selectedPlaces(places: readonly PlaceSummary[], placeIds: readonly number[] | null): PlaceSummary[] {
	if (placeIds === null || placeIds.length === 0) {
		return [];
	}
	const wanted = new Set(placeIds);
	return places.filter((place) => wanted.has(place.placeId));
}

/**
 * The panel head's name for one selection (a city pin's place, or a
 * bubble's group of places):
 *
 * - one place answers its own name;
 * - several places sharing one country answer that country's display
 *   name (the bubble's honest area label -- the map's country-scale
 *   reading);
 * - a group spanning countries has no single place name, so the
 *   caller's area label ("This area") carries it -- never a member
 *   city's name standing in for the whole group.
 */
export function selectionName(selected: readonly PlaceSummary[], activeLocale: string, areaLabel: string): string {
	if (selected.length === 0) {
		return '';
	}
	if (selected.length === 1) {
		return selected[0]!.name;
	}
	const country = selected[0]!.country;
	if (selected.every((place) => place.country === country)) {
		return countryName(country, activeLocale);
	}
	return areaLabel;
}

/** The selection's photo count: the sum of its places' honest counts. */
export function selectionCount(selected: readonly PlaceSummary[]): number {
	return selected.reduce((sum, place) => sum + place.assetCount, 0);
}
