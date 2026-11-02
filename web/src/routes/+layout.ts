// Pure single-page app: no server-side rendering and no prerendering.
// adapter-static emits a single fallback page (the index.html app
// shell) plus the content-hashed _app/ assets; the client-side router
// owns every route once the shell loads. The Go server serves the
// shell for all non-API GET paths (see server/internal/web).
export const ssr = false;
export const prerender = false;
