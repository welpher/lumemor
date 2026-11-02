// Pure single-page app: no server-side rendering and no prerendering.
// adapter-static emits a single fallback page (the index.html app
// shell) plus the content-hashed _app/ assets; the client-side router
// owns every route once the shell loads. The Go server serves the
// shell for all non-API GET paths (see server/internal/web).
import { initI18n } from '$lib/i18n';

export const ssr = false;
export const prerender = false;

// Load the shared translations before the first render so no route
// ever paints empty strings (idempotent; runs once per app boot).
export async function load() {
	await initI18n();
}
