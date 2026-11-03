import { flushSync, mount, tick, unmount } from 'svelte';
import { beforeEach, expect, test } from 'vitest';
import { initI18n } from '$lib/i18n';
import Drawer from './Drawer.svelte';

// The drawer contract from the web IA: seven content sections,
// three of them live links (their real views are later tasks), four
// disabled M3 placeholders, Escape and an outside click closing it.

beforeEach(async () => {
	document.body.textContent = '';
	await initI18n();
});

test('closed renders nothing; open renders the seven sections in IA order', () => {
	const closed = mount(Drawer, { target: document.body, props: { open: false } });
	flushSync();
	expect(document.body.querySelector('.drawer__panel')).toBeNull();
	unmount(closed);
	flushSync();

	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	const panel = document.body.querySelector<HTMLElement>('nav.drawer__panel');
	expect(panel?.getAttribute('aria-label')).toBe('Navigation');

	// The IA's route-table order, all seven visible.
	const labels = [...document.body.querySelectorAll('.drawer__item')].map(
		(element) => element.textContent
	);
	expect(labels).toEqual(['Timeline', 'Albums', 'People', 'Places', 'Memories', 'Favorites', 'Trash']);

	// The live links: the timeline, favorites and trash routes.
	const links = [...document.body.querySelectorAll<HTMLAnchorElement>('a.drawer__item')].map(
		(link) => link.getAttribute('href')
	);
	expect(links).toEqual(['/timeline', '/favorites', '/trash']);

	// The M3 placeholders: present, visible, disabled.
	const disabled = [...document.body.querySelectorAll<HTMLButtonElement>('button.drawer__item--disabled')];
	expect(disabled.map((button) => button.textContent)).toEqual(['Albums', 'People', 'Places', 'Memories']);
	expect(disabled.every((button) => button.disabled)).toBe(true);
	expect(disabled.every((button) => button.title !== '')).toBe(true);

	// The coming-later footnote explains the disabled section.
	expect(panel?.textContent).toContain('The remaining sections arrive in a later update.');
	unmount(drawer);
	flushSync();
});

test('Escape closes the drawer', async () => {
	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
	await tick();

	expect(document.body.querySelector('.drawer__panel')).toBeNull();
	unmount(drawer);
	flushSync();
});

test('a click outside (the scrim) closes the drawer', async () => {
	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	const scrim = document.body.querySelector<HTMLDivElement>('.drawer__scrim');
	scrim?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await tick();

	expect(document.body.querySelector('.drawer__panel')).toBeNull();
	expect(document.body.querySelector('.drawer__scrim')).toBeNull();
	unmount(drawer);
	flushSync();
});

test('a click on the panel itself does not close the drawer', async () => {
	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	const panel = document.body.querySelector<HTMLElement>('nav.drawer__panel');
	panel?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await tick();

	expect(document.body.querySelector('.drawer__panel')).not.toBeNull();
	unmount(drawer);
	flushSync();
});

test('navigating through a drawer link closes the drawer', async () => {
	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	// jsdom would log a not-implemented navigation when a dispatched
	// click runs the anchor's default action; canceling the default
	// through a document-level listener silences it without touching
	// the component (whose own handler must never preventDefault --
	// the SvelteKit router owns the click).
	const cancelNavigation = (event: Event): void => event.preventDefault();
	document.addEventListener('click', cancelNavigation);
	try {
		const link = document.body.querySelector<HTMLAnchorElement>('a.drawer__item[href="/timeline"]');
		link?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
		await tick();
	} finally {
		document.removeEventListener('click', cancelNavigation);
	}

	expect(document.body.querySelector('.drawer__panel')).toBeNull();
	unmount(drawer);
	flushSync();
});
