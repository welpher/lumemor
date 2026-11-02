import { flushSync, mount, unmount } from 'svelte';
import { expect, test } from 'vitest';
import { initI18n } from '$lib/i18n';
import Page from './+page.svelte';

// The web-check gate's one smoke test: prove the Svelte pipeline
// compiles and renders a route component into a DOM, with the shared
// translation table loaded (the page title comes from i18n).
test('the root placeholder page renders its welcome content', async () => {
	await initI18n();
	const app = mount(Page, { target: document.body });
	flushSync();
	try {
		expect(document.body.textContent).toContain('Welcome to lumemor');
	} finally {
		unmount(app);
	}
});
