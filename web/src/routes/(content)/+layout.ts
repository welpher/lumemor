import type { LayoutLoad } from './$types';
import { guardSessionRoute } from '$lib/auth/guard';

// Content-shell routes sit behind the session guard: an
// unauthenticated visitor is sent to /login with the current path
// and query as the post-login target (see $lib/auth/guard for the
// full matrix). The root layout's load -- awaited through parent() --
// has already resolved the session state by the time this runs.
export const load: LayoutLoad = async ({ url, parent }) => {
	await guardSessionRoute(url, parent);
};
