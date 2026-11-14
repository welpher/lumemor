// The places surface's data client: GET /places -- the
// caller's city grouping with the locale display-name join, behind
// the repo-wide error-code convention (every failure outcome
// carries the error code's i18n key: the envelope code lowercased,
// 'internal' for a network failure or an unparseable body,
// 'session_expired' for a 401).
//
// The locale is a REQUEST parameter: the web app owns the active
// locale and passes it along (the default en); a locale switch
// changes the name join only, never an asset row. The list is not
// paginated (the single-document shape the named-people listing
// uses) -- one call answers the whole grouping.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One city group, exactly the contract's PlaceSummary shape. */
export type PlaceSummary = components['schemas']['PlaceSummary'];

/** The city grouping read's outcome. */
export type PlacesListResult = { ok: true; places: PlaceSummary[] } | { ok: false; errorKey: string };

/** The locale the request carries when the caller passes none. */
const DEFAULT_LOCALE = 'en';

/** The error-code i18n key of one failed response (the repo convention). */
async function errorKeyFrom(response: Response): Promise<string> {
	if (response.status === 401) {
		// A dead session shows the session message, never
		// login-credential text (the interceptor already flipped the
		// store; the guard is redirecting).
		return 'session_expired';
	}
	const envelope = await errorEnvelope(response);
	return envelope !== null ? envelope.code.toLowerCase() : 'internal';
}

/** Read one response's JSON body; null when it does not parse. */
async function bodyOf(response: Response): Promise<unknown> {
	try {
		return await response.json();
	} catch {
		return null;
	}
}

/** One place summary, structurally validated; null when not one. */
function placeFrom(body: unknown): PlaceSummary | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<PlaceSummary>;
	if (
		typeof candidate.placeId !== 'number' ||
		!Number.isInteger(candidate.placeId) ||
		candidate.placeId <= 0 ||
		typeof candidate.name !== 'string' ||
		typeof candidate.country !== 'string'
	) {
		return null;
	}
	if (
		typeof candidate.assetCount !== 'number' ||
		!Number.isInteger(candidate.assetCount) ||
		candidate.assetCount < 0
	) {
		return null;
	}
	// The place's own coordinate: the map face pins it. A grouping
	// answer without it cannot draw the map, so it is a shape failure
	// rather than a silently misplaced pin.
	if (
		typeof candidate.latitude !== 'number' ||
		!Number.isFinite(candidate.latitude) ||
		typeof candidate.longitude !== 'number' ||
		!Number.isFinite(candidate.longitude)
	) {
		return null;
	}
	// The capture span: both ends always present on a listed place
	// (the card's year range and the list's ordering read them).
	if (
		typeof candidate.takenAtFrom !== 'string' ||
		Number.isNaN(Date.parse(candidate.takenAtFrom)) ||
		typeof candidate.takenAtTo !== 'string' ||
		Number.isNaN(Date.parse(candidate.takenAtTo))
	) {
		return null;
	}
	// The contract pins at least one cover candidate, but the card
	// renders an honest placeholder for a bare group -- the types
	// are what must hold, the cardinality degrades gracefully.
	if (
		!Array.isArray(candidate.representativeCoverAssetIds) ||
		candidate.representativeCoverAssetIds.some((id) => typeof id !== 'string')
	) {
		return null;
	}
	return body as PlaceSummary;
}

/**
 * GET /api/v1/places: the caller's city grouping, ordered by the
 * latest capture time descending. `locale` picks the display names'
 * join (the web app's active locale; the default en).
 */
export async function listPlaces(
	locale: string = DEFAULT_LOCALE,
	fetcher: typeof fetch = globalThis.fetch
): Promise<PlacesListResult> {
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/places?locale=${encodeURIComponent(locale)}`, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!Array.isArray(body)) {
		return { ok: false, errorKey: 'internal' };
	}
	const places: PlaceSummary[] = [];
	for (const entry of body) {
		const place = placeFrom(entry);
		if (place === null) {
			return { ok: false, errorKey: 'internal' };
		}
		places.push(place);
	}
	return { ok: true, places };
}

/** The place re-resolution trigger's outcome: the run's id, or the failure's i18n key. */
export type PlacesReresolveResult = { ok: true; jobRunId: string } | { ok: false; errorKey: string };

/**
 * POST /api/v1/admin/places/reresolve: start (or join) one full
 * place re-resolution run. The endpoint is idempotent -- an in-flight
 * run answers 200 with its identifier, a fresh one 202 -- so both
 * statuses are the same outcome for the caller.
 */
export async function reresolvePlaces(fetcher: typeof fetch = globalThis.fetch): Promise<PlacesReresolveResult> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/admin/places/reresolve', { method: 'POST' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (body === null || typeof body !== 'object') {
		return { ok: false, errorKey: 'internal' };
	}
	const jobRunId = (body as { jobRunId?: unknown }).jobRunId;
	if (typeof jobRunId !== 'string' || jobRunId === '') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, jobRunId };
}
