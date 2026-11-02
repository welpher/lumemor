/// <reference types="vitest/config" />
import { fileURLToPath } from 'node:url';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

// The repository root, one level above this web/ project. The shared
// translation table lives there in i18n/ and is read directly by the
// app through a relative import.meta.glob (see src/lib/i18n). The
// dev server only serves files inside server.fs.allow, so the
// repository root joins the project root there explicitly; builds
// need no such allowance, rollup resolves the glob at bundle time.
const projectRoot = fileURLToPath(new URL('.', import.meta.url));
const repositoryRoot = fileURLToPath(new URL('..', import.meta.url));

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		fs: {
			allow: [projectRoot, repositoryRoot]
		}
	},
	// Under vitest, resolve svelte through its browser export so
	// mount()/flushSync() work in jsdom; the build keeps the default
	// resolution. (process.env.VITEST is set only by vitest.)
	resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined,
	test: {
		// jsdom gives the smoke test a DOM to mount components into; the
		// SvelteKit plugin above compiles the components under test.
		environment: 'jsdom'
	}
});
