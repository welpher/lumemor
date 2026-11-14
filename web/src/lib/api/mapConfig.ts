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

/** The map configuration read's outcome. */
export type MapConfigResult = { ok: true; tilejsonUrl: string | null } | { ok: false; errorKey: string };

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
 * GET /api/v1/map/config: the tile source URL, or null when the
 * instance has none. Both are successful reads; only a transport or
 * shape failure is an error.
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
	if (body === null || typeof body !== 'object' || !('tilejsonUrl' in body)) {
		return { ok: false, errorKey: 'internal' };
	}
	const value = (body as { tilejsonUrl: unknown }).tilejsonUrl;
	if (value === null) {
		return { ok: true, tilejsonUrl: null };
	}
	if (typeof value !== 'string') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, tilejsonUrl: value === '' ? null : value };
}
