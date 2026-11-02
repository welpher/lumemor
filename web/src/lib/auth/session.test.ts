import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { apiFetch, errorEnvelope, parseRetryAfter } from './api';
import {
	forceLogout,
	login,
	logout,
	probeSession,
	sessionEntries,
	sessionStatus
} from './session.svelte';

// The session store is module state shared by every test of this
// file; each case arranges its starting state through the store's
// own transitions (a 401 probe boots it unauthenticated).

function jsonResponse(
	status: number,
	body: unknown,
	headers: Record<string, string> = {}
): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json', ...headers }
	});
}

const sessionsBody = [
	{
		id: '6f9c2d18-9a4c-4a1b-b1a1-000000000001',
		kind: 'WEB',
		deviceName: 'Mozilla/5.0 (test)',
		createdAt: '2026-11-01T10:00:00Z',
		lastSeenAt: '2026-11-02T10:00:00Z'
	}
];

beforeEach(async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));
	// Boot the store unauthenticated: a fresh app with no cookie.
	await probeSession();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the boot probe populates the store on a 200 session list', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(200, sessionsBody)));

	await probeSession();

	expect(sessionStatus()).toBe('authenticated');
	expect(sessionEntries()).toEqual(sessionsBody);
});

test('the boot probe answers unauthenticated on a 401', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));

	await probeSession();

	expect(sessionStatus()).toBe('unauthenticated');
	expect(sessionEntries()).toEqual([]);
});

test('the boot probe answers unauthenticated on a server error', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(500, { code: 'INTERNAL' })));

	await probeSession();

	// An unverifiable session is treated like a missing one: the
	// login page is the only place that recovers from every cause.
	expect(sessionStatus()).toBe('unauthenticated');
});

test('the boot probe answers unauthenticated when the server is unreachable', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(new TypeError('network down'))));

	await expect(probeSession()).resolves.toBeUndefined();
	expect(sessionStatus()).toBe('unauthenticated');
});

test('a mid-session 401 flips the store to unauthenticated (the interceptor)', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(200, sessionsBody)));
	await probeSession();
	expect(sessionStatus()).toBe('authenticated');

	// Any API call answering 401 while authenticated = the session
	// died; the wrapper returns the response (the caller owns its
	// error mapping) and flips the store.
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));
	const response = await apiFetch('/api/v1/assets/timeline');

	expect(response.status).toBe(401);
	expect(sessionStatus()).toBe('unauthenticated');
	expect(sessionEntries()).toEqual([]);
});

test('a 401 while already unauthenticated never triggers the interceptor (no loop)', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));

	const response = await apiFetch('/api/v1/auth/login', { method: 'POST' });

	// The login form's own credential failure: the store stays
	// unauthenticated, the response is returned for the form's error
	// mapping -- the interceptor only reacts to mid-session expiry.
	expect(response.status).toBe(401);
	expect(sessionStatus()).toBe('unauthenticated');
});

test('forceLogout is idempotent: the flip happens once', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(200, sessionsBody)));
	await probeSession();

	forceLogout();
	forceLogout();

	expect(sessionStatus()).toBe('unauthenticated');
});

test('login success establishes the session', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(200, sessionsBody[0])));

	const outcome = await login('ada@example.test', 'correct horse');

	expect(outcome).toEqual({ ok: true });
	expect(sessionStatus()).toBe('authenticated');
});

test('login 401 maps to the auth_invalid_credentials i18n key', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS', message: 'no' }))
	);

	const outcome = await login('ada@example.test', 'wrong');

	expect(outcome).toEqual({ ok: false, errorKey: 'auth_invalid_credentials', retryAfterSeconds: null });
	expect(sessionStatus()).toBe('unauthenticated');
});

test('login 429 maps to auth_rate_limited and carries the Retry-After seconds', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			jsonResponse(429, { code: 'AUTH_RATE_LIMITED', message: 'slow down' }, { 'Retry-After': '30' })
		)
	);

	const outcome = await login('ada@example.test', 'again');

	expect(outcome).toEqual({ ok: false, errorKey: 'auth_rate_limited', retryAfterSeconds: 30 });
});

test('login 429 without a usable Retry-After carries no countdown', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(429, { code: 'AUTH_RATE_LIMITED', message: 'slow down' }))
	);

	const outcome = await login('ada@example.test', 'again');

	expect(outcome).toEqual({ ok: false, errorKey: 'auth_rate_limited', retryAfterSeconds: null });
});

test('login 400 maps to the validation_failed i18n key', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(400, { code: 'VALIDATION_FAILED', message: 'fields' }))
	);

	const outcome = await login('', '');

	expect(outcome).toEqual({ ok: false, errorKey: 'validation_failed', retryAfterSeconds: null });
});

test('login with a bodyless proxy answer maps to the internal key', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => new Response('<html>gateway error</html>', { status: 502 })));

	const outcome = await login('ada@example.test', 'x');

	expect(outcome).toEqual({ ok: false, errorKey: 'internal', retryAfterSeconds: null });
});

test('login on a network failure maps to the internal key', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(new TypeError('network down'))));

	const outcome = await login('ada@example.test', 'x');

	expect(outcome).toEqual({ ok: false, errorKey: 'internal', retryAfterSeconds: null });
});

test('logout posts the revoke and clears the store', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(200, sessionsBody)));
	await probeSession();

	const fetchMock = vi.fn(async () => new Response(null, { status: 204 }));
	vi.stubGlobal('fetch', fetchMock);

	await logout();

	expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/logout', { method: 'POST' });
	expect(sessionStatus()).toBe('unauthenticated');
});

test('logout clears the store even when the revoke call fails', async () => {
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(200, sessionsBody)));
	await probeSession();

	vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(new TypeError('network down'))));

	await expect(logout()).resolves.toBeUndefined();
	expect(sessionStatus()).toBe('unauthenticated');
});

test('errorEnvelope reads the code, tolerates non-envelope bodies', async () => {
	expect(await errorEnvelope(jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))).toEqual({
		code: 'AUTH_INVALID_CREDENTIALS'
	});
	expect(await errorEnvelope(jsonResponse(502, { nothing: 'here' }))).toBeNull();
	expect(await errorEnvelope(new Response('not json', { status: 500 }))).toBeNull();
});

test('parseRetryAfter reads whole seconds, rejects junk', () => {
	expect(parseRetryAfter(jsonResponse(429, {}, { 'Retry-After': '30' }))).toBe(30);
	expect(parseRetryAfter(jsonResponse(429, {}))).toBeNull();
	expect(parseRetryAfter(jsonResponse(429, {}, { 'Retry-After': 'soon' }))).toBeNull();
	expect(parseRetryAfter(jsonResponse(429, {}, { 'Retry-After': '0' }))).toBeNull();
});
