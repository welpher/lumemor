// The map's data client: GET /map/config -- the tile source the
// administrator configured, null when the instance has none. The
// tiles themselves never pass through this server: the browser's map
// library loads them straight from the tile host, and the pins the
// map draws come from the places read.
//
// The failure mapping is the repo-wide convention (the envelope code
// lowercased, 'internal' for a network failure or an unparseable
// body, 'session_expired' for a 401). A config read that fails is not
// a page failure: the map draws its stated absence and the rest of
// the page stands.

import { apiFetch, errorEnvelope } from '$lib/auth/api';

/** The map configuration read's outcome: the light and dark tile
 * source URLs, each null when the instance has none. */
export type MapConfigResult =
	| {
			ok: true;
			tilejsonUrl: string | null;
			darkTilejsonUrl: string | null;
	  }
	| { ok: false; errorKey: string };

/** The config endpoint. */
const MAP_CONFIG_ENDPOINT = '/api/v1/map/config';

/** One response's error-code i18n key (the repo convention). */
async function errorKeyFrom(response: Response): Promise<string> {
	if (response.status === 401) {
		return 'session_expired';
	}
	const envelope = await errorEnvelope(response);
	return envelope !== null ? envelope.code.toLowerCase() : 'internal';
}

/**
 * GET /api/v1/map/config: the light and dark tile source URLs, each
 * null when the instance has none. All are required by the contract,
 * so a body missing one is a shape failure; the empty string is the
 * same unconfigured state as null.
 */
export async function getMapConfig(fetcher: typeof fetch = globalThis.fetch): Promise<MapConfigResult> {
	let response: Response;
	try {
		response = await apiFetch(MAP_CONFIG_ENDPOINT, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (body === null || typeof body !== 'object') {
		return { ok: false, errorKey: 'internal' };
	}
	const light = tileSourceOf(body, 'tilejsonUrl');
	const dark = tileSourceOf(body, 'darkTilejsonUrl');
	if (light === null || dark === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return {
		ok: true,
		tilejsonUrl: light.value,
		darkTilejsonUrl: dark.value
	};
}

/**
 * One source field's value: `{value}` (null when unconfigured), or
 * null when the body carries no usable value for it.
 */
function tileSourceOf(body: object, key: 'tilejsonUrl' | 'darkTilejsonUrl'): { value: string | null } | null {
	if (!(key in body)) {
		return null;
	}
	const value = (body as Record<string, unknown>)[key];
	if (value === null) {
		return { value: null };
	}
	if (typeof value !== 'string') {
		return null;
	}
	return { value: value === '' ? null : value };
}
