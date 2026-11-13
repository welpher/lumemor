// Route-entry guards of the session-protected route groups. The
// load-time half lives here (guardSessionRoute, called by the
// (content) and (tool) group layouts); the runtime half lives in
// SessionGuard.svelte (mid-session 401s). Together they form the
// guard matrix:
//
//	unauthenticated + guarded route -> /login?redirectTo=<path+query>
//	authenticated   + guarded route -> through
//	authenticated   + /login        -> /timeline
//	unauthenticated + /login        -> the form

import { redirect } from '@sveltejs/kit';
import { sessionStatus } from './session.svelte';

/**
 * Guard one route-group load: after the root layout's session probe
 * has resolved (`await parent()`), an unauthenticated visitor is
 * sent to the login page carrying the full current path and query as
 * the post-login target, so a deep link returns to itself after
 * sign-in. `unknown` (the probe could not verify anything) is
 * treated like unauthenticated: the login page is the only place
 * that recovers from every cause.
 */
export async function guardSessionRoute(url: URL, parent: () => Promise<unknown>): Promise<void> {
	await parent();
	if (sessionStatus() !== 'authenticated') {
		redirect(307, loginRedirectTarget(url.pathname + url.search));
	}
}

/**
 * Build the login redirect target for a guarded path: the path and
 * query survive the round-trip encoded. The one builder for the
 * load-time guard and the SessionGuard watcher keeps both halves of
 * the matrix emitting identical URLs.
 *
 * `expired` tags a mid-session 401's redirect: the login page then
 * says the session expired instead of greeting a plain sign-in. The
 * target itself is unchanged, so the post-login return still works.
 */
export function loginRedirectTarget(pathAndQuery: string, expired = false): string {
	const target = `/login?redirectTo=${encodeURIComponent(pathAndQuery)}`;
	return expired ? `${target}&expired=1` : target;
}

/**
 * Whether a route belongs to a session-guarded tree: everything in
 * the SPA except the login route. The input may carry a query (the
 * runtime watcher passes the full current path + query); the guard
 * decision looks at the path portion only, or the login page's own
 * `?redirectTo=` round-trip would read as a guarded route and loop.
 * The first-run wizard lives outside the SPA entirely (a full
 * document navigation, never a client-side route).
 */
export function isGuardedPath(path: string): boolean {
	const pathname = path.split('?', 1)[0];
	return pathname !== '' && pathname !== '/login' && !pathname.startsWith('/login/');
}
