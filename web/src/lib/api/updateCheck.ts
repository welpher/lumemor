// The release check's client: the cached state behind
// GET /server/update-check and the manual run behind
// POST /server/update-check.
//
// The two calls share one document shape, so they share one parser.
// The manual run is the only call of the pair that reaches the
// network -- and it is the SERVER that reaches it: a failed probe
// comes back as the `unknown` status, not as an error, so the face
// never renders a failure's red. A transport failure of the web
// request itself is the generic internal failure (the settings card
// already shows the last answer beside it).

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** The release check state, exactly the contract's UpdateCheckResponse shape. */
export type UpdateCheckState = components['schemas']['UpdateCheckResponse'];

/** The check's outcome: the state document, or the failure's i18n key. */
export type UpdateCheckResult = { ok: true; state: UpdateCheckState } | { ok: false; errorKey: string };

/** The check endpoint (the manual run is its POST). */
const UPDATE_CHECK_ENDPOINT = '/api/v1/server/update-check';

/** One response's error-code i18n key (the repo convention). */
async function errorKeyFrom(response: Response): Promise<string> {
	if (response.status === 401) {
		return 'session_expired';
	}
	const envelope = await errorEnvelope(response);
	return envelope !== null ? envelope.code.toLowerCase() : 'internal';
}

/** Structurally validate one check body; null when it is not one. */
function updateCheckFrom(body: unknown): UpdateCheckState | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<UpdateCheckState>;
	const statuses = ['disabled', 'unknown', 'up_to_date', 'update_available'];
	if (
		typeof candidate.enabled !== 'boolean' ||
		typeof candidate.status !== 'string' ||
		!statuses.includes(candidate.status) ||
		typeof candidate.currentVersion !== 'string'
	) {
		return null;
	}
	if (candidate.latestVersion !== null && typeof candidate.latestVersion !== 'string') {
		return null;
	}
	if (candidate.checkedAt !== null && typeof candidate.checkedAt !== 'string') {
		return null;
	}
	return body as UpdateCheckState;
}

/** One call over the check endpoint (GET reads the cache, POST runs the probe). */
async function call(method: 'GET' | 'POST', fetcher: typeof fetch): Promise<UpdateCheckResult> {
	let response: Response;
	try {
		response = await apiFetch(UPDATE_CHECK_ENDPOINT, { method }, fetcher);
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
	const state = updateCheckFrom(body);
	if (state === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, state };
}

/** GET /api/v1/server/update-check: the cached state, never a network probe. */
export function getUpdateCheck(fetcher: typeof fetch = globalThis.fetch): Promise<UpdateCheckResult> {
	return call('GET', fetcher);
}

/** POST /api/v1/server/update-check: probe now and answer the fresh state. */
export function runUpdateCheck(fetcher: typeof fetch = globalThis.fetch): Promise<UpdateCheckResult> {
	return call('POST', fetcher);
}
