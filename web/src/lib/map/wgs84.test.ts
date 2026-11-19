import { describe, expect, test } from 'vitest';
import { isKnownNonWgs84Basemap } from './wgs84';

describe('isKnownNonWgs84Basemap', () => {
	test('the known Chinese providers are recognized', () => {
		expect(isKnownNonWgs84Basemap('https://webrd01.is.autonavi.com/appmaptile?style=7')).toBe(true);
		expect(isKnownNonWgs84Basemap('https://www.amap.com/style.json')).toBe(true);
		expect(isKnownNonWgs84Basemap('https://map.qq.com/style.json')).toBe(true);
		expect(isKnownNonWgs84Basemap('https://map.baidu.com/style.json')).toBe(true);
	});

	test('standard and self-hosted sources are not flagged', () => {
		expect(isKnownNonWgs84Basemap('https://tiles.openfreemap.org/styles/bright')).toBe(false);
		expect(isKnownNonWgs84Basemap('http://192.168.1.20:8080/tilejson.json')).toBe(false);
		expect(isKnownNonWgs84Basemap('https://tiles.example.net/style.json')).toBe(false);
	});

	test('blank and malformed values stay silent', () => {
		expect(isKnownNonWgs84Basemap('')).toBe(false);
		expect(isKnownNonWgs84Basemap('   ')).toBe(false);
		expect(isKnownNonWgs84Basemap('nonsense')).toBe(false);
	});
});
