// The China datum shift: GCJ-02 is the obfuscated coordinate system
// Chinese basemaps (Amap, Tencent, Baidu's GCJ-02 ancestor) draw in,
// while photo EXIF GPS is standard WGS-84. The two differ by roughly
// 300-600 metres inside China, so a pin placed in library coordinates
// on a GCJ-02 basemap lands visibly off.
//
// THE LIBRARY STAYS WGS-84. The conversion lives only at the map's
// display and pick boundaries: pins and framing convert WGS-84 to the
// active source's datum before projection, and a picked point converts
// back before it reaches the caller. Nothing else in the app converts.
//
// The algorithm is the standard public-domain one: the Krasovsky 1940
// ellipsoid constants and the polynomial offset keyed on the position
// relative to 105E/35N. Outside China it is the identity: the offset
// belongs to one country's basemaps, not to the planet.
//
// Everything here is pure: no imports, no map library, no network.

/** Krasovsky 1940 semi-major axis (metres). */
const KRASOVSKY_A = 6_378_245.0;

/** Krasovsky 1940 first eccentricity squared. */
const KRASOVSKY_EE = 0.006693421622965943;

/** The China bounding box the offset applies inside (the usual one). */
const CHINA_MIN_LON = 73.66;
const CHINA_MAX_LON = 135.05;
const CHINA_MIN_LAT = 3.86;
const CHINA_MAX_LAT = 53.55;

/** One geographic point, degrees. */
export type LatLon = { lat: number; lon: number };

/**
 * Whether a point lies inside the China bounding box. The box is the
 * algorithm's own gate: outside it the datum shift is the identity, so
 * a Tokyo or Paris pin is never touched.
 */
export function insideChina(lat: number, lon: number): boolean {
	return lon > CHINA_MIN_LON && lon < CHINA_MAX_LON && lat > CHINA_MIN_LAT && lat < CHINA_MAX_LAT;
}

/** The latitude polynomial of the offset, at the box-relative point. */
function transformLat(x: number, y: number): number {
	let value = -100.0 + 2.0 * x + 3.0 * y + 0.2 * y * y + 0.1 * x * y + 0.2 * Math.sqrt(Math.abs(x));
	value += ((20.0 * Math.sin(6.0 * x * Math.PI) + 20.0 * Math.sin(2.0 * x * Math.PI)) * 2.0) / 3.0;
	value += ((20.0 * Math.sin(y * Math.PI) + 40.0 * Math.sin((y / 3.0) * Math.PI)) * 2.0) / 3.0;
	value += ((160.0 * Math.sin((y / 12.0) * Math.PI) + 320 * Math.sin((y * Math.PI) / 30.0)) * 2.0) / 3.0;
	return value;
}

/** The longitude polynomial of the offset, at the box-relative point. */
function transformLon(x: number, y: number): number {
	let value = 300.0 + x + 2.0 * y + 0.1 * x * x + 0.1 * x * y + 0.1 * Math.sqrt(Math.abs(x));
	value += ((20.0 * Math.sin(6.0 * x * Math.PI) + 20.0 * Math.sin(2.0 * x * Math.PI)) * 2.0) / 3.0;
	value += ((20.0 * Math.sin(x * Math.PI) + 40.0 * Math.sin((x / 3.0) * Math.PI)) * 2.0) / 3.0;
	value += ((150.0 * Math.sin((x / 12.0) * Math.PI) + 300.0 * Math.sin((x / 30.0) * Math.PI)) * 2.0) / 3.0;
	return value;
}

/** The WGS-84 to GCJ-02 offset at one point, in degrees. */
function offset(lat: number, lon: number): LatLon {
	const x = lon - 105.0;
	const y = lat - 35.0;
	let deltaLat = transformLat(x, y);
	let deltaLon = transformLon(x, y);
	const radLat = (lat / 180.0) * Math.PI;
	let magic = Math.sin(radLat);
	magic = 1 - KRASOVSKY_EE * magic * magic;
	const sqrtMagic = Math.sqrt(magic);
	deltaLat = (deltaLat * 180.0) / (((KRASOVSKY_A * (1 - KRASOVSKY_EE)) / (magic * sqrtMagic)) * Math.PI);
	deltaLon = (deltaLon * 180.0) / ((KRASOVSKY_A / sqrtMagic) * Math.cos(radLat) * Math.PI);
	return { lat: deltaLat, lon: deltaLon };
}

/**
 * WGS-84 -> GCJ-02: the display direction. A pin stored in library
 * coordinates becomes a point on a GCJ-02 basemap. Outside China the
 * point is returned unchanged.
 */
export function wgs84ToGcj02(lat: number, lon: number): LatLon {
	if (!insideChina(lat, lon)) {
		return { lat, lon };
	}
	const shift = offset(lat, lon);
	return { lat: lat + shift.lat, lon: lon + shift.lon };
}

/**
 * GCJ-02 -> WGS-84: the pick direction. A point tapped on a GCJ-02
 * basemap becomes a library coordinate before the caller stores it.
 * Outside China the point is returned unchanged.
 *
 * The forward offset is not analytically invertible, so the inverse
 * subtracts the offset measured at the GCJ point. The residual is a
 * couple of metres at most -- far below the precision of a tapped map
 * -- and the round-trip test pins it.
 */
export function gcj02ToWgs84(lat: number, lon: number): LatLon {
	if (!insideChina(lat, lon)) {
		return { lat, lon };
	}
	const shift = offset(lat, lon);
	return { lat: lat - shift.lat, lon: lon - shift.lon };
}
