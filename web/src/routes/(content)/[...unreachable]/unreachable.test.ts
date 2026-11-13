import { flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test } from 'vitest';
import { initI18n } from '$lib/i18n';
import UnreachableCard from '$lib/components/UnreachableCard.svelte';
import Page from './+page.svelte';

// The route-level face: every address the app does not know lands on
// the catch-all, inside the content shell, and renders the one card.
// The card takes no props at all -- there is no seam through which a
// path fragment could reach it -- so the tests pin both the copy and
// the absence of anything resource-shaped.

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

test('the card is one fixed face: the copy, the way back, the footnote', () => {
	const card = mount(UnreachableCard, { target: document.body });
	flushSync();

	const element = document.body.querySelector<HTMLElement>('.unreachable-card');
	expect(element?.textContent).toContain('This page will not open');
	expect(element?.textContent).toContain('It may have been deleted, or you may not have access to it.');
	expect(element?.querySelector<HTMLAnchorElement>('a.btn')?.getAttribute('href')).toBe('/timeline');
	expect(element?.querySelector('a.btn')?.textContent?.trim()).toBe('Back to the timeline');
	expect(element?.textContent).toContain('The address may be incomplete, or the share may have been withdrawn.');
	// The face never distinguishes the causes: one line, one action.
	expect(element?.querySelectorAll('a, button')).toHaveLength(1);
	// No title is a heading: the shell's bar owns the page's title.
	expect(element?.querySelector('h1, h2, h3')).toBeNull();
	unmount(card);
	flushSync();
});

test('the catch-all renders the card in the content area and reads nothing from the address', () => {
	const page = mount(Page, { target: document.body });
	flushSync();

	expect(document.body.querySelector('.unreachable-page .unreachable-card')).not.toBeNull();
	// The page owns no title of its own (the shell's bar carries the
	// wordmark) and no resource identifier of any kind.
	expect(document.body.querySelector('img')).toBeNull();
	expect(document.body.querySelector('[role="progressbar"]')).toBeNull();
	unmount(page);
	flushSync();
});
