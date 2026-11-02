import { flushSync, mount, unmount } from 'svelte';
import { expect, test } from 'vitest';
import Page from './+page.svelte';

// The web-check gate's one smoke test: prove the Svelte pipeline
// compiles and renders a route component into a DOM. Later tasks
// replace it with real component and unit coverage.
test('the root placeholder page renders its welcome content', () => {
	const app = mount(Page, { target: document.body });
	flushSync();
	try {
		expect(document.body.textContent).toContain('Welcome to lumemor');
	} finally {
		unmount(app);
	}
});
