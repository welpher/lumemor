// Session state and the client half of the web authentication
// contract (the M2a API surface): a boot-time probe of
// GET /api/v1/auth/sessions decides whether the app boots into the
// signed-in experience, the login form establishes a session, logout
// revokes it, and any API 401 mid-session flips the state back to
// unauthenticated (the interceptor in api.ts).
//
// The store is Svelte 5 rune module state: load functions and plain
// modules read it through sessionStatus(), components react to it
// through SessionGuard.svelte.

import { apiFetch, errorEnvelope, parseRetryAfter } from './api';

export type SessionStatus = 'unknown' | 'authenticated' | 'unauthenticated';

/** One entry of GET /api/v1/auth/sessions: one of the caller's devices. */
export type SessionEntry = {
	id: string;
	kind: string;
	deviceName?: string;
	createdAt: string;
	lastSeenAt: string;
	expiresAt?: string;
};

let status = $state<SessionStatus>('unknown');
let entries = $state<SessionEntry[]>([]);

/** The resolved session state; 'unknown' only before the boot probe. */
export function sessionStatus(): SessionStatus {
	return status;
}

/** The caller's active sessions across devices; empty when signed out. */
export function sessionEntries(): readonly SessionEntry[] {
	return entries;
}

function markUnauthenticated(): void {
	status = 'unauthenticated';
	entries = [];
}

/**
 * The boot probe: GET /api/v1/auth/sessions through the shared
 * wrapper. A 200 populates the store (the list itself proves the
 * cookie carries a live session); any other outcome -- 401, a server
 * error, an unreachable server -- boots the app unauthenticated. A
 * 5xx is deliberately treated like a 401: the session could not be
 * verified, and the login page is the one place that recovers from
 * every cause. The root layout's load passes its own fetch (the
 * SvelteKit-provided one) so the probe runs inside load etiquette.
 */
export async function probeSession(fetcher: typeof fetch = globalThis.fetch): Promise<void> {
	try {
		const response = await apiFetch('/api/v1/auth/sessions', undefined, fetcher);
		if (response.ok) {
			const body: unknown = await response.json();
			entries = Array.isArray(body) ? (body as SessionEntry[]) : [];
			status = 'authenticated';
			return;
		}
	} catch {
		// Offline or unreachable: the app boots signed out.
	}
	markUnauthenticated();
}

/** The client outcome of a login attempt: success or the failure's i18n key. */
export type LoginOutcome =
	| { ok: true }
	| { ok: false; errorKey: string; retryAfterSeconds: number | null };

/**
 * POST /api/v1/auth/login with the form's credentials. Success (the
 * Set-Cookie rides the response) establishes the session in the
 * store. Failure carries the error code's i18n key (the lowercase
 * convention; a body without an envelope answers with the generic
 * `internal` key) and, for a 429, the Retry-After seconds behind the
 * countdown hint.
 */
export async function login(email: string, password: string): Promise<LoginOutcome> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/auth/login', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ email, password })
		});
	} catch {
		// The network failed: no envelope, no code; the generic message.
		return { ok: false, errorKey: 'internal', retryAfterSeconds: null };
	}
	if (response.ok) {
		status = 'authenticated';
		return { ok: true };
	}
	const envelope = await errorEnvelope(response);
	return {
		ok: false,
		errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal',
		retryAfterSeconds: response.status === 429 ? parseRetryAfter(response) : null
	};
}

/**
 * POST /api/v1/auth/logout and clear the store. The POST is
 * best-effort: a failed call still clears the local state -- the UI
 * must never keep acting on a session it no longer trusts -- and the
 * server-side record dies by expiry at the latest. The redirect to
 * /login is the caller's (the avatar menu) or the SessionGuard
 * watcher's, whichever reacts first.
 */
export async function logout(): Promise<void> {
	try {
		await apiFetch('/api/v1/auth/logout', { method: 'POST' });
	} catch {
		// Same reasoning: the local state goes regardless.
	}
	markUnauthenticated();
}

/**
 * The 401 interceptor's action: flip the store to unauthenticated.
 * Idempotent by construction -- re-assigning the same status never
 * re-triggers the reactive watchers, which is what keeps a blizzard
 * of 401s from becoming a redirect loop. SessionGuard performs the
 * navigation.
 */
export function forceLogout(): void {
	markUnauthenticated();
}
