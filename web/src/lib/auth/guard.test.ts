import { beforeEach, afterEach, expect, test, vi } from 'vitest';
import { guardSessionRoute, isGuardedPath, loginRedirectTarget } from './guard';
import { login, probeSession, sessionStatus } from './session.svelte';
import { load as loginLoad, _safeRedirectTarget } from '../../routes/(auth)/login/+page';
import { load as rootLoad } from '../../routes/+layout';

// The guard routing matrix: load functions are invoked directly
// (they are plain async functions taking the load event), so the
// assertions cover the exact redirects a real navigation would see.
// SvelteKit's redirect() throws a plain object carrying the status
// and the location; that throw is what the assertions match.

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

const serverInfo = (initialized: boolean) => jsonResponse(200, { version: 'test', initialized, features: {} });

beforeEach(async () => {
	// Boot unauthenticated unless a case arranges otherwise.
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await probeSession();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

async function arrangeAuthenticated(): Promise<void> {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, [{ id: 's1', kind: 'WEB', createdAt: 'x', lastSeenAt: 'x' }]))
	);
	await login('ada@example.test', 'correct horse');
}

/** Invoke a load function with a minimal hand-built load event. */
function callLoad(
	load: (event: never) => unknown,
	url: URL,
	overrides: Partial<{ fetch: typeof fetch; parent: () => Promise<unknown> }> = {}
): Promise<unknown> {
	const event = {
		url,
		fetch: globalThis.fetch,
		parent: async () => ({}),
		...overrides
	};
	return Promise.resolve(load(event as never));
}

// --- guardSessionRoute (the (content) and (tool) group loads) ------

test('unauthenticated on a guarded route redirects to login with the current path', async () => {
	await expect(guardSessionRoute(new URL('http://localhost/timeline'), async () => ({}))).rejects.toMatchObject({
		status: 307,
		location: '/login?redirectTo=%2Ftimeline'
	});
});

test('unauthenticated on a deep link preserves the path and the query', async () => {
	await expect(guardSessionRoute(new URL('http://localhost/a/123?tab=exif'), async () => ({}))).rejects.toMatchObject(
		{
			status: 307,
			location: '/login?redirectTo=%2Fa%2F123%3Ftab%3Dexif'
		}
	);
});

test('authenticated on a guarded route passes through', async () => {
	await arrangeAuthenticated();

	await expect(guardSessionRoute(new URL('http://localhost/timeline'), async () => ({}))).resolves.toBeUndefined();
});

test('the guard awaits the parent load (the root layout probe) before deciding', async () => {
	// The parent resolves the session only when awaited first: the
	// guard must sequence behind it, or it would decide from stale
	// store state.
	const parent = vi.fn(async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => jsonResponse(200, []))
		);
		await probeSession();
		return {};
	});
	await expect(guardSessionRoute(new URL('http://localhost/timeline'), parent)).resolves.toBeUndefined();
	expect(parent).toHaveBeenCalled();
});

// --- the login route load ------------------------------------------

test('authenticated visiting /login is redirected to /timeline', async () => {
	await arrangeAuthenticated();

	await expect(callLoad(loginLoad as never, new URL('http://localhost/login'))).rejects.toMatchObject({
		status: 303,
		location: '/timeline'
	});
});

test('unauthenticated visiting /login gets the form with a validated target', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => serverInfo(true))
	);

	const data = (await callLoad(loginLoad as never, new URL('http://localhost/login'))) as {
		setupRequired: boolean;
		redirectTo: string;
	};

	expect(data.setupRequired).toBe(false);
	expect(data.redirectTo).toBe('/timeline');
});

test('an uninitialized server marks the page for the wizard handover', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => serverInfo(false))
	);

	const data = (await callLoad(loginLoad as never, new URL('http://localhost/login'))) as {
		setupRequired: boolean;
	};

	expect(data.setupRequired).toBe(true);
});

test('an unreachable server-info probe keeps the form', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => Promise.reject(new TypeError('network down')))
	);

	const data = (await callLoad(loginLoad as never, new URL('http://localhost/login'))) as {
		setupRequired: boolean;
	};

	expect(data.setupRequired).toBe(false);
});

test('a redirectTo round-trip value passes through validated', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => serverInfo(true))
	);

	const data = (await callLoad(
		loginLoad as never,
		new URL('http://localhost/login?redirectTo=%2Fa%2F42%3Ftab%3Dexif')
	)) as { redirectTo: string };

	expect(data.redirectTo).toBe('/a/42?tab=exif');
});

// --- safeRedirectTarget (the open-redirect guard) -------------------

test('safeRedirectTarget honors in-app paths and refuses everything else', () => {
	expect(_safeRedirectTarget('/timeline')).toBe('/timeline');
	expect(_safeRedirectTarget('/a/42?tab=exif')).toBe('/a/42?tab=exif');
	expect(_safeRedirectTarget('/')).toBe('/');
	// Foreign URLs, protocol-relative and backslash tricks, control
	// characters, and absent values all fall back to the timeline.
	expect(_safeRedirectTarget('https://evil.example/steal')).toBe('/timeline');
	expect(_safeRedirectTarget('//evil.example')).toBe('/timeline');
	expect(_safeRedirectTarget('/\\evil.example')).toBe('/timeline');
	expect(_safeRedirectTarget('/ok\u0000bad')).toBe('/timeline');
	expect(_safeRedirectTarget('')).toBe('/timeline');
	expect(_safeRedirectTarget(null)).toBe('/timeline');
});

// --- the shared predicate and URL builder --------------------------

test('isGuardedPath covers the whole SPA except the login route', () => {
	expect(isGuardedPath('/timeline')).toBe(true);
	expect(isGuardedPath('/timeline?cursor=abc')).toBe(true);
	expect(isGuardedPath('/admin/users')).toBe(true);
	expect(isGuardedPath('/')).toBe(true);
	expect(isGuardedPath('/login')).toBe(false);
	// The login page's own redirectTo round-trip must never read as a
	// guarded route -- that is the redirect loop the runtime watcher
	// would otherwise enter (caught in the live smoke test).
	expect(isGuardedPath('/login?redirectTo=%2Ftimeline')).toBe(false);
	expect(isGuardedPath('/login/')).toBe(false);
	expect(isGuardedPath('')).toBe(false);
});

test('loginRedirectTarget encodes the path and query for the round-trip', () => {
	expect(loginRedirectTarget('/timeline')).toBe('/login?redirectTo=%2Ftimeline');
	expect(loginRedirectTarget('/a/42?tab=exif')).toBe('/login?redirectTo=%2Fa%2F42%3Ftab%3Dexif');
});

test('loginRedirectTarget tags an expiry without losing the round-trip target', () => {
	expect(loginRedirectTarget('/albums/x?tab=people', true)).toBe(
		'/login?redirectTo=%2Falbums%2Fx%3Ftab%3Dpeople&expired=1'
	);
});

// --- the root layout load (the boot probe) -------------------------

test('the root layout load probes the session before the first render', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);

	// The load receives its fetch from the load event (SvelteKit
	// etiquette); the stubbed global stands in for it here.
	await rootLoad({ fetch: globalThis.fetch } as never);

	expect(sessionStatus()).toBe('unauthenticated');
});
