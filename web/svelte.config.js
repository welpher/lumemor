import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// Pure SPA: the adapter emits one fallback page (the index.html
		// app shell) plus the content-hashed _app/ assets; no route is
		// server-rendered or prerendered (see src/routes/+layout.ts).
		// The Go server embeds and serves the output from
		// server/internal/web.
		adapter: adapter({
			fallback: 'index.html'
		})
	}
};

export default config;
