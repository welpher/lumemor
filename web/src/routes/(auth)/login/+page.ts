import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import { sessionStatus } from '$lib/auth/session.svelte';

/**
 * The login route's load: the session guard's mirror (an
 * authenticated visitor is bounced to the timeline), the first-run
 * check (an instance without an administrator cannot sign anyone in
 * -- the wizard owns that flow), and the post-login target's
 * validation (only in-app paths pass; anything else falls back to
 * the timeline).
 */
export const load: PageLoad = async ({ url, fetch, parent }) => {
	// The root layout's load (i18n init + session probe) must have
	// resolved before the guard reads the store.
	await parent();

	if (sessionStatus() === 'authenticated') {
		redirect(303, '/timeline');
	}

	// First-run check: GET /server/info is the unauthenticated
	// capability report. An uninitialized instance cannot log anyone
	// in, so the page hands the browser over to the wizard (the
	// component performs the navigation; /setup is a server-rendered
	// page outside the SPA, never a client-side route). A failed
	// probe keeps the form: its own submit surfaces the failure.
	let setupRequired = false;
	try {
		const response = await fetch('/api/v1/server/info');
		if (response.ok) {
			const info: unknown = await response.json();
			setupRequired =
				info !== null &&
				typeof info === 'object' &&
				(info as { initialized?: unknown }).initialized === false;
		}
	} catch {
		// Unreachable or broken body: the form stays; the sign-in
		// attempt reports the problem.
	}

	return {
		setupRequired,
		redirectTo: _safeRedirectTarget(url.searchParams.get('redirectTo'))
	};
};

/**
 * Validate a post-login target from the query string: only an
 * in-app path (absolute, same-origin, not protocol-relative) is
 * honored; everything else -- a foreign URL, a malformed value, a
 * missing parameter -- falls back to the timeline. This is the
 * open-redirect guard of the redirectTo round-trip.
 *
 * The underscore prefix is required: SvelteKit validates route
 * module exports (load and the route options are the only bare
 * names allowed; helper exports must be underscore-prefixed).
 */
export function _safeRedirectTarget(value: string | null): string {
	if (
		value !== null &&
		value.startsWith('/') &&
		!value.startsWith('//') &&
		!/[\u0000-\u001f\u007f\\]/.test(value)
	) {
		return value;
	}
	return '/timeline';
}
