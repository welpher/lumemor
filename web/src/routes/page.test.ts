import { flushSync, mount, unmount } from 'svelte';
import { isRedirect } from '@sveltejs/kit';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { probeSession } from '$lib/auth/session.svelte';
import Page from './+page.svelte';
import { load } from './+page';

// The root route: the component keeps its welcome smoke test, and
// the load resolves the deferred decision -- an authenticated
// visitor is redirected to the timeline (the app's home surface),
// while an unauthenticated one keeps the guard matrix (the runtime
// SessionGuard watcher bounces `/` to the login page).

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
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

const loadEvent = { parent: async () => ({}) } as Parameters<typeof load>[0];

beforeEach(async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await probeSession();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the root placeholder page renders its welcome content', async () => {
	await initI18n();
	const app = mount(Page, { target: document.body });
	flushSync();
	try {
		expect(document.body.textContent).toContain('Welcome to lumemor');
	} finally {
		unmount(app);
		flushSync();
	}
});

test('an authenticated visitor on the root is redirected to the timeline', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, sessionsBody))
	);
	await probeSession();

	let thrown: unknown;
	try {
		await load(loadEvent);
	} catch (error) {
		thrown = error;
	}
	expect(thrown).toBeDefined();
	expect(isRedirect(thrown)).toBe(true);
	const redirection = thrown as { status: number; location: string };
	expect(redirection.status).toBe(303);
	expect(redirection.location).toBe('/timeline');
});

test('an unauthenticated visitor on the root is not redirected by the load', async () => {
	await expect(load(loadEvent)).resolves.toBeUndefined();
});

test('an unverifiable session on the root is not redirected by the load', async () => {
	// The probe boots unauthenticated on a 5xx (the pinned choice);
	// the login page recovers from every cause.
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(500, { code: 'INTERNAL' }))
	);
	await probeSession();

	await expect(load(loadEvent)).resolves.toBeUndefined();
});
