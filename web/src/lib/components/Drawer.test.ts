import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { probeSession, sessionStatus } from '$lib/auth/session.svelte';
import Drawer from './Drawer.svelte';

// The drawer contract: eight content items (the
// albums wall's entry live since the albums surface; the hidden view's
// entry moved here from the timeline's toolbar), one
// disabled placeholder, the admin panel's role-gated entry (the
// identity seam: an admin's session renders it, a member's never
// does), Escape and an outside click closing it. The settings entry
// is GONE: the account dropdown's
// link is settings' only home, and this file pins the drawer's side
// of that promise -- no /settings door here, for any role.

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Boot the session store signed in with the given role (or out on null). */
async function bootSession(role: 'admin' | 'user' | null): Promise<void> {
	if (role === null) {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
		);
	} else {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () =>
				jsonResponse(200, [
					{
						id: '6f9c2d18-9a4c-4a1b-b1a1-000000000001',
						kind: 'WEB',
						userId: '6f9c2d18-9a4c-4a1b-b1a1-00000000000a',
						role,
						deviceName: 'Mozilla/5.0 (test)',
						createdAt: '2026-11-01T10:00:00Z',
						lastSeenAt: '2026-11-02T10:00:00Z'
					}
				])
			)
		);
	}
	await probeSession();
}

beforeEach(async () => {
	document.body.textContent = '';
	// The store is module state shared by this file's tests: every
	// case starts signed out and boots its own role when it needs one.
	await bootSession(null);
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('closed renders nothing; open renders the eight content items in IA order', () => {
	const closed = mount(Drawer, { target: document.body, props: { open: false } });
	flushSync();
	expect(document.body.querySelector('.drawer__panel')).toBeNull();
	unmount(closed);
	flushSync();

	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	const panel = document.body.querySelector<HTMLElement>('nav.drawer__panel');
	expect(panel?.getAttribute('aria-label')).toBe('Navigation');

	// The route-table order, all eight visible -- the hidden
	// view's entry moved here from the timeline's toolbar and
	// sits with the other list views (favorites/trash). The settings
	// entry is gone: the account dropdown is its only
	// home. The leading icons add a whitespace text node
	// before each label, so the text assertions trim.
	const labels = [...document.body.querySelectorAll('.drawer__item')].map((element) => element.textContent?.trim());
	expect(labels).toEqual(['Timeline', 'Albums', 'People', 'Places', 'Memories', 'Favorites', 'Hidden', 'Trash']);

	// Every item LEADS with its icon: the svg is the item's
	// first element child, decorative (aria-hidden), and the text
	// stays the accessible name -- the icon never replaces it.
	const items = [...document.body.querySelectorAll<HTMLElement>('.drawer__item')];
	expect(items).toHaveLength(8);
	for (const item of items) {
		const icon = item.querySelector<SVGSVGElement>('svg');
		expect(icon, 'leading icon present').not.toBeNull();
		expect(item.firstElementChild).toBe(icon);
		expect(icon?.getAttribute('aria-hidden')).toBe('true');
	}
	// The icon set is one distinct glyph per section, not one repeated.
	const iconClasses = new Set(items.map((item) => item.querySelector('svg')?.getAttribute('class') ?? ''));
	expect(iconClasses.size).toBe(8);

	// The live links: the timeline, the albums wall, the people,
	// the places grid, favorites, hidden and trash routes -- and
	// NO /settings door (the account dropdown owns it alone).
	const links = [...document.body.querySelectorAll<HTMLAnchorElement>('a.drawer__item')].map((link) =>
		link.getAttribute('href')
	);
	expect(links).toEqual(['/timeline', '/albums', '/people', '/places', '/favorites', '/hidden', '/trash']);
	expect(document.body.querySelector('a.drawer__item[href="/settings"]')).toBeNull();

	// The placeholder: present, visible, disabled.
	const disabled = [...document.body.querySelectorAll<HTMLButtonElement>('button.drawer__item--disabled')];
	expect(disabled.map((button) => button.textContent?.trim())).toEqual(['Memories']);
	expect(disabled.every((button) => button.disabled)).toBe(true);
	expect(disabled.every((button) => button.title !== '')).toBe(true);

	// The coming-later footnote explains the disabled section.
	expect(panel?.textContent).toContain('The remaining sections arrive in a later update.');

	// A signed-out caller holds no role: the admin entry never renders.
	expect(document.body.textContent).not.toContain('Admin panel');
	expect(document.body.querySelector('.drawer__divider')).toBeNull();
	unmount(drawer);
	flushSync();
});

test('an admin session renders the admin panel entry below the content list', async () => {
	await bootSession('admin');

	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	// The entry lands behind a divider, below the eight content items,
	// and points at the first live management section (/admin itself
	// has no index route). The settings entry stays absent for the
	// admin too: the dropdown is its one home for every role.
	const divider = document.body.querySelector('.drawer__divider');
	expect(divider).not.toBeNull();
	const admin = document.body.querySelector<HTMLAnchorElement>('a.drawer__item[href="/admin/users"]');
	expect(admin?.textContent?.trim()).toBe('Admin panel');
	// The admin entry leads with its own icon like every content item.
	expect(admin?.querySelector('svg')).not.toBeNull();
	expect(admin?.firstElementChild?.tagName.toLowerCase()).toBe('svg');

	const labels = [...document.body.querySelectorAll('.drawer__item')].map((element) => element.textContent?.trim());
	expect(labels).toEqual([
		'Timeline',
		'Albums',
		'People',
		'Places',
		'Memories',
		'Favorites',
		'Hidden',
		'Trash',
		'Admin panel'
	]);
	expect(document.body.querySelector('a.drawer__item[href="/settings"]')).toBeNull();
	unmount(drawer);
	flushSync();
});

test("a member session's drawer never renders the admin panel entry", async () => {
	await bootSession('user');

	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	expect(document.body.textContent).not.toContain('Admin panel');
	expect(document.body.querySelector('.drawer__divider')).toBeNull();
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

test('the nav items shift their leading padding on hover (the G10 trial)', () => {
	// The rail item's hover adds a small start-side nudge (~2px,
	// the 3 -> 3.5 spacing step) alongside the background change,
	// riding the item's existing transition so both arrive at the
	// theme's fast step together.
	const drawer = mount(Drawer, { target: document.body, props: { open: true } });
	flushSync();

	const link = document.body.querySelector<HTMLAnchorElement>('a.drawer__item[href="/timeline"]');
	expect(link?.className).toContain('ps-3');
	expect(link?.className).toContain('pe-3');
	expect(link?.className).toContain('hover:ps-3.5');
	expect(link?.className).toContain('transition-[background-color,padding]');
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
