// Pure single-page app: no server-side rendering and no prerendering.
// adapter-static emits a single fallback page (the index.html app
// shell) plus the content-hashed _app/ assets; the client-side router
// owns every route once the shell loads. The Go server serves the
// shell for all non-API GET paths (see server/internal/web).
import { initI18n } from '$lib/i18n';
import { probeSession } from '$lib/auth/session.svelte';
import type { LayoutLoad } from './$types';

export const ssr = false;
export const prerender = false;

// Load the shared translations, then probe the session, before the
// first render: no route paints empty strings, and the session guard
// (the group-layout loads plus the SessionGuard watcher) decides
// from a resolved session state. Both steps are idempotent and run
// once per app boot -- client-side navigations re-use this layout
// node, so this load never re-runs (and never re-probes) mid-session.
export const load: LayoutLoad = async ({ fetch }) => {
	await initI18n();
	await probeSession(fetch);
};
