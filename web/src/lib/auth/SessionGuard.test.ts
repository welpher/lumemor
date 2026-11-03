import { flushSync, mount, tick, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import SessionGuard from './SessionGuard.svelte';
import { login } from './session.svelte';

// The mid-session expiry flow: the interceptor flips the store, the
// watcher navigates. $app/navigation is mocked because the real goto
// needs a running SvelteKit app (and outside one it throws an
// unhandled rejection); the mock records the navigation target.

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
import { goto } from '$app/navigation';

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

beforeEach(() => {
	vi.clearAllMocks();
	document.body.textContent = '';
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, [{ id: 's1', kind: 'WEB', createdAt: 'x', lastSeenAt: 'x' }]))
	);
});

test('the watcher navigates to login when a guarded route goes unauthenticated mid-session', async () => {
	// Arrange a live session (the login form's success path).
	await login('ada@example.test', 'correct horse');

	const guard = mount(SessionGuard, { target: document.body, props: { currentPath: '/timeline' } });
	flushSync();
	expect(goto).not.toHaveBeenCalled();

	// A mid-session 401 flips the store; the watcher reacts with the
	// one navigation, carrying the round-trip target.
	const { forceLogout } = await import('./session.svelte');
	forceLogout();
	await tick();

	expect(goto).toHaveBeenCalledTimes(1);
	expect(goto).toHaveBeenCalledWith('/login?redirectTo=%2Ftimeline');
	unmount(guard);
	flushSync();
});

test('the watcher navigates with the path and query of the current route', async () => {
	await login('ada@example.test', 'correct horse');

	const guard = mount(SessionGuard, { target: document.body, props: { currentPath: '/a/42?tab=exif' } });
	flushSync();

	const { forceLogout } = await import('./session.svelte');
	forceLogout();
	await tick();

	expect(goto).toHaveBeenCalledWith('/login?redirectTo=%2Fa%2F42%3Ftab%3Dexif');
	unmount(guard);
	flushSync();
});

test('the watcher stays inert on the login route (no redirect loop)', async () => {
	const guard = mount(SessionGuard, { target: document.body, props: { currentPath: '/login' } });
	flushSync();

	const { forceLogout } = await import('./session.svelte');
	forceLogout();
	await tick();

	// The store flipped while unauthenticated on /login: that is the
	// login page's own normal state, not a mid-session expiry.
	expect(goto).not.toHaveBeenCalled();
	unmount(guard);
	flushSync();
});

test('the watcher stays inert on the login route carrying its redirectTo query', async () => {
	// The redirect the watcher itself produces lands here; re-firing
	// on this URL would compound the redirectTo parameter into an
	// infinite loop (the regression caught by the live smoke test).
	const guard = mount(SessionGuard, {
		target: document.body,
		props: { currentPath: '/login?redirectTo=%2Ftimeline' }
	});
	flushSync();

	const { forceLogout } = await import('./session.svelte');
	forceLogout();
	await tick();

	expect(goto).not.toHaveBeenCalled();
	unmount(guard);
	flushSync();
});

test('the watcher stays inert while the session is live', async () => {
	await login('ada@example.test', 'correct horse');

	const guard = mount(SessionGuard, { target: document.body, props: { currentPath: '/timeline' } });
	flushSync();
	await tick();

	expect(goto).not.toHaveBeenCalled();
	unmount(guard);
	flushSync();
});
