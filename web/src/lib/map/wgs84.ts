// The library stores and displays WGS-84 only. Known Chinese basemap
// hosts draw in a different datum (Amap and Tencent in GCJ-02, Baidu
// in BD-09), so the settings face warns when one is pasted. The warning
// never blocks the save and never converts anything: a custom host
// cannot be classified, and the URL is loaded as-is.

const NON_WGS84_HOSTS = /(^|\.)(amap\.com|autonavi\.com|map\.qq\.com|maps\.qq\.com|gtimg\.com|baidu\.com|bdimg\.com)$/i;

/**
 * Whether the URL's host is a known non-WGS-84 basemap provider.
 * Blank and malformed values answer false: the shape validator owns
 * those, and this helper only decides whether to show the offset
 * warning.
 */
export function isKnownNonWgs84Basemap(rawUrl: string): boolean {
	const trimmed = rawUrl.trim();
	if (trimmed === '') {
		return false;
	}
	let host: string;
	try {
		host = new URL(trimmed).hostname;
	} catch {
		return false;
	}
	return NON_WGS84_HOSTS.test(host);
}
