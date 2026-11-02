/// <reference types="vitest/config" />
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
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
