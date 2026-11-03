// The shared fetch wrapper of the web app: every API call goes
// through apiFetch, plus the response helpers the error-code
// convention needs (the error envelope and the rate-limit hint).
//
// THE 401 INTERCEPTOR (the pinned mechanism, see the report):
// a 401 response while the session store says `authenticated`
// means the session died mid-session (expired, revoked, disabled
// account). The wrapper flips the store to unauthenticated exactly
// once -- the flip is idempotent, and SessionGuard.svelte (mounted
// once in the root layout) reacts to the flip with the redirect to
// the login page -- so no route keeps rendering on a dead session
// and no fetch retries into a loop. A 401 while already
// unauthenticated (the boot probe, the login form's own credential
// failures) never triggers the interceptor.

import { forceLogout, sessionStatus } from './session.svelte';

/** The API error envelope: a stable code clients map to i18n keys. */
export type ApiErrorEnvelope = { code: string };

/**
 * Fetch an API path through the 401 interceptor. Callers keep the
 * Response and own their error mapping; only the global
 * mid-session-expiry reaction lives here. `fetcher` defaults to the
 * global fetch (the component-level norm); load functions pass the
 * fetch they receive from their load event, which is what SvelteKit
 * wants called inside a load.
 */
export async function apiFetch(
	path: string,
	init?: RequestInit,
	fetcher: typeof fetch = globalThis.fetch
): Promise<Response> {
	const response = await fetcher(path, init);
	if (response.status === 401 && sessionStatus() === 'authenticated') {
		forceLogout();
	}
	return response;
}

/**
 * Parse a response's error envelope; null when the body is not one
 * (a proxy answered, the connection broke mid-body). The code maps
 * to the i18n key by lowercasing (the repo-wide convention).
 */
export async function errorEnvelope(response: Response): Promise<ApiErrorEnvelope | null> {
	try {
		const body: unknown = await response.json();
		if (
			body !== null &&
			typeof body === 'object' &&
			typeof (body as ApiErrorEnvelope).code === 'string'
		) {
			return body as ApiErrorEnvelope;
		}
	} catch {
		// Not JSON: the caller falls back to the generic message.
	}
	return null;
}

/**
 * Parse the Retry-After header of a 429 as whole seconds; null when
 * absent or invalid (the login form then shows no countdown hint).
 */
export function parseRetryAfter(response: Response): number | null {
	const value = response.headers.get('Retry-After');
	if (value === null) {
		return null;
	}
	const seconds = Number.parseInt(value, 10);
	return Number.isFinite(seconds) && seconds > 0 ? seconds : null;
}
