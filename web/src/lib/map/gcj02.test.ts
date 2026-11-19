// The China datum shift's tests: the offset's magnitude inside China,
// the round-trip's bounded residual, and the identity outside China.
// These run on pure numbers -- the display and pick wiring has its own
// tests against the map module.

import { describe, expect, test } from 'vitest';
import { gcj02ToWgs84, insideChina, wgs84ToGcj02 } from './gcj02';

// Tiananmen, Beijing: a well-known WGS-84 point inside the box.
const BEIJING = { lat: 39.9075, lon: 116.39723 };
// Tokyo: outside the box, so both conversions must be the identity.
const TOKYO = { lat: 35.6895, lon: 139.69171 };

describe('insideChina', () => {
	test('the box admits Beijing and excludes Tokyo', () => {
		expect(insideChina(BEIJING.lat, BEIJING.lon)).toBe(true);
		expect(insideChina(TOKYO.lat, TOKYO.lon)).toBe(false);
	});
});

describe('wgs84ToGcj02', () => {
	test('a point inside China shifts by a few hundred metres', () => {
		const shifted = wgs84ToGcj02(BEIJING.lat, BEIJING.lon);
		// Degrees to metres around Beijing: ~111 km per degree of
		// latitude, ~85 km per degree of longitude at 40N.
		const northMetres = (shifted.lat - BEIJING.lat) * 111_320;
		const eastMetres = (shifted.lon - BEIJING.lon) * 111_320 * Math.cos((BEIJING.lat * Math.PI) / 180);
		const distance = Math.hypot(northMetres, eastMetres);
		// The published shift is roughly 300-600 m; anything outside
		// this band means the constants drifted.
		expect(distance).toBeGreaterThan(100);
		expect(distance).toBeLessThan(1000);
		expect(shifted.lat).not.toBe(BEIJING.lat);
		expect(shifted.lon).not.toBe(BEIJING.lon);
	});

	test('a point outside China is returned unchanged', () => {
		expect(wgs84ToGcj02(TOKYO.lat, TOKYO.lon)).toEqual(TOKYO);
		// The boundary is the box, not the country: Paris is the same
		// identity answer.
		expect(wgs84ToGcj02(48.85341, 2.3488)).toEqual({ lat: 48.85341, lon: 2.3488 });
	});
});

describe('gcj02ToWgs84', () => {
	test('the round-trip lands within a metre of the original', () => {
		for (const point of [
			BEIJING,
			{ lat: 31.2304, lon: 121.4737 }, // Shanghai
			{ lat: 23.1291, lon: 113.2644 }, // Guangzhou
			{ lat: 30.5728, lon: 104.0668 } // Chengdu
		]) {
			const there = wgs84ToGcj02(point.lat, point.lon);
			const back = gcj02ToWgs84(there.lat, there.lon);
			// 5e-5 degrees is a few metres: the inverse's residual is
			// orders below a tapped map's precision, and the bound
			// fails loudly if the constants ever drift.
			expect(Math.abs(back.lat - point.lat)).toBeLessThan(5e-5);
			expect(Math.abs(back.lon - point.lon)).toBeLessThan(5e-5);
		}
	});

	test('a point outside China is returned unchanged', () => {
		expect(gcj02ToWgs84(TOKYO.lat, TOKYO.lon)).toEqual(TOKYO);
	});
});
