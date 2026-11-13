/// <reference types="vitest/config" />
import { fileURLToPath } from 'node:url';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { configDefaults } from 'vitest/config';
import { defineConfig, type Plugin } from 'vite';

// The repository root, one level above this web/ project. The shared
// translation table lives there in i18n/ and is read directly by the
// app through a relative import.meta.glob (see src/lib/i18n). The
// dev server only serves files inside server.fs.allow, so the
// repository root joins the project root there explicitly; builds
// need no such allowance, rollup resolves the glob at bundle time.
const projectRoot = fileURLToPath(new URL('.', import.meta.url));
const repositoryRoot = fileURLToPath(new URL('..', import.meta.url));

// The first-run wizard is a server-rendered page outside the SPA (it
// has to render without JavaScript, so it cannot be a client route),
// but it must wear the app's own stylesheet rather than a second skin.
// The compiled theme's file name is content-hashed, so the Go template
// cannot pin it; this emits the same bytes under one stable name
// (setup.css) for the wizard's <link>. The template's own utility
// classes are generated into that stylesheet through theme.css's
// @source of the static page directory.
function setupStylesheet(): Plugin {
	return {
		name: 'lumemor:setup-stylesheet',
		generateBundle(_options, bundle) {
			// The theme-bearing stylesheet is the one that DECLARES the
			// semantic tokens (component sheets only reference them), so
			// the wizard carries the tokens, the base layer and the
			// utilities it shares with the app -- never the app's
			// component rules it has no markup for.
			const themeSheets = Object.values(bundle)
				.flatMap((item) =>
					item.type === 'asset' && item.fileName.endsWith('.css')
						? [{ fileName: item.fileName, source: String(item.source) }]
						: []
				)
				.filter((sheet) => sheet.source.includes('--color-accent:'))
				.map((sheet) => sheet.fileName);
			if (themeSheets.length === 0) {
				return;
			}
			const imports = themeSheets.map((fileName) => `@import url("/${fileName}");`).join('\n');
			this.emitFile({ type: 'asset', fileName: 'setup.css', source: `${imports}\n` });
		}
	};
}

export default defineConfig({
	// The Tailwind v4 plugin must run before the SvelteKit plugin: it
	// compiles the utilities of src/lib/theme/theme.css (the brand
	// token @theme mapping) and scans the Svelte templates for the
	// class names to generate.
	plugins: [tailwindcss(), sveltekit(), setupStylesheet()],
	server: {
		// The dev server pins its port: 8243 sits beside the dogfood
		// API's 8244 (the two halves of the same local pair), and
		// strictPort makes an occupied port fail loudly instead of
		// silently drifting onto another number (or another project
		// quietly taking the other address family).
		// Pinned to loopback: the printed URL and the bound socket must
		// agree (an unpinned `localhost` binds one address family only,
		// which lets another project take the other family on the same
		// port -- the collision this pin removes).
		host: '127.0.0.1',
		port: 8243,
		strictPort: true,
		fs: {
			allow: [projectRoot, repositoryRoot]
		},
		// The dev proxy: forward the API (REST + SSE events) to a
		// running instance -- the dogfood stack at 8244 by default, or
		// any URL in LUMEMOR_DEV_API. The web app uses relative paths
		// throughout, so the proxy keeps them same-origin (cookies
		// included) while the browser sees a single origin.
		proxy: {
			'/api': {
				target: process.env.LUMEMOR_DEV_API ?? 'http://localhost:8244',
				changeOrigin: true,
				// SSE (the /api/v1/events stream) needs the ws upgrade
				// + a longer timeout or the proxy drops the stream.
				ws: true,
				timeout: 600000
			}
		}
	},
	// Under vitest, resolve svelte through its browser export so
	// mount()/flushSync() work in jsdom; the build keeps the default
	// resolution. (process.env.VITEST is set only by vitest.)
	resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined,
	test: {
		// jsdom gives the smoke test a DOM to mount components into; the
		// SvelteKit plugin above compiles the components under test.
		environment: 'jsdom',
		// The e2e specs under e2e/ match vitest's default *.spec.ts glob
		// but belong to Playwright (its own runner, a real server, no
		// jsdom) -- keep them out of the unit suite.
		exclude: [...configDefaults.exclude, 'e2e/**']
	}
});
