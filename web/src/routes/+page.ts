import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import { sessionStatus } from '$lib/auth/session.svelte';

/**
 * The `/`-versus-`/timeline` decision, resolved here: an
 * authenticated visitor landing on `/` is sent to the
 * timeline, the app's home surface -- the welcome placeholder has
 * nothing to offer once the wall exists. The login flow's default
 * target stays `/timeline`, so nothing else moves.
 *
 * An unauthenticated visitor keeps the existing guard path: `/` sits
 * outside the guarded route groups, so the load-time guard does not
 * cover it; the SessionGuard watcher (mounted once by the root
 * layout) bounces the rendered welcome page to
 * `/login?redirectTo=%2F`, the same matrix every guarded route
 * follows. `unknown` (the probe could not verify anything) follows
 * the unauthenticated arm: the login page recovers from every cause.
 */
export const load: PageLoad = async ({ parent }) => {
	await parent();
	if (sessionStatus() === 'authenticated') {
		redirect(303, '/timeline');
	}
};
