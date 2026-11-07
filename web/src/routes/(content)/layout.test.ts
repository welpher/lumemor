import { createRawSnippet, flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { goto } from '$app/navigation';
import Layout from './+layout.svelte';

// The content shell's app frame (R2-1): the h-dvh root pins the top
// bar by layout (no sticky hack), <main> is the frame's one scroll
// container, and the browser's keyboard scrolling follows FOCUS -- a
// body-focused page no longer scrolls once the document itself stops
// scrolling, so the frame keeps the focus on its scroll root
// (tabindex -1 keeps it out of the tab order). $app/navigation is
// mocked because the real router needs a running SvelteKit app; the
// mock captures the afterNavigate callbacks the frame registers. The
// router delivers the MOUNTING navigation only after the route
// renders (bind:this included), so each test fires its navigation
// explicitly after the mount.

const { navigationCallbacks, pageState } = vi.hoisted(() => ({
	navigationCallbacks: [] as Array<(navigation: { type: string }) => void>,
	// The route state the layout's topbar-hiding rule reads: the
	// album detail's route id hides the shell topbar (R3, the page
	// owns its chrome there); every other route keeps it.
	pageState: {
		route: { id: '/(content)/timeline' as string | null }
	}
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn(),
	afterNavigate: (callback: (navigation: { type: string }) => void) => {
		navigationCallbacks.push(callback);
	}
}));
vi.mock('$app/state', () => ({ page: pageState }));

function mountLayout(): ReturnType<typeof mount> {
	return mount(Layout, {
		target: document.body,
		props: {
			data: {},
			params: {},
			children: createRawSnippet(() => ({ render: () => '<p>page body</p>' }))
		}
	});
}

/** Replace main's scrollTop with an observing accessor (jsdom clamps the real one to 0). */
function observeScrollTop(main: HTMLElement): ReturnType<typeof vi.fn> {
	const setter = vi.fn();
	Object.defineProperty(main, 'scrollTop', {
		get: () => 0,
		set: setter,
		configurable: true
	});
	return setter;
}

/** Fire one completed navigation to every registered callback (the router's delivery). */
function navigate(type: string): void {
	for (const callback of [...navigationCallbacks]) {
		callback({ type });
	}
	flushSync();
}

beforeEach(async () => {
	document.body.textContent = '';
	navigationCallbacks.length = 0;
	vi.clearAllMocks();
	await initI18n();
});

test('the shell is an h-dvh frame whose one scroller is <main>', () => {
	const app = mountLayout();
	flushSync();

	const shell = document.body.querySelector<HTMLElement>('.shell');
	expect(shell?.className).toContain('h-dvh');
	expect(shell?.className).toContain('overflow-hidden');
	expect(shell?.className).toContain('flex');
	expect(shell?.className).toContain('flex-col');
	// The document-level scrolling frame is gone.
	expect(shell?.className).not.toContain('min-h-screen');

	const main = document.body.querySelector<HTMLElement>('main.shell-main');
	expect(main?.className).toContain('overflow-y-auto');
	expect(main?.className).toContain('flex-1');
	expect(main?.className).toContain('min-h-0');

	// The top bar is pinned by the frame itself: no sticky positioning.
	const topbar = document.body.querySelector<HTMLElement>('.shell-topbar');
	expect(topbar?.className).not.toContain('sticky');
	unmount(app);
	flushSync();
});

test('the frame focuses its scroll root so keyboard scrolling follows it', () => {
	const app = mountLayout();
	flushSync();

	// The mounting navigation (delivered after the render) leaves the
	// focus on the scroll root.
	navigate('enter');
	const main = document.body.querySelector<HTMLElement>('main.shell-main');
	// Out of the tab order (a scroll vehicle, never a focus target),
	// but focusable so Space / PageDown / the arrows scroll it and a
	// click on any non-focusable content keeps the focus there too.
	expect(main?.getAttribute('tabindex')).toBe('-1');
	expect(document.activeElement).toBe(main);

	// A navigation whose focus already sits on a real element (an
	// autofocus field, a persisted link) never steals it: only the
	// body-focused state takes the scroll root.
	const button = document.body.querySelector<HTMLButtonElement>('button[aria-label="Open navigation"]');
	button?.focus();
	navigate('link');
	expect(document.activeElement).toBe(button);
	unmount(app);
	flushSync();
});

test('a push navigation resets the scroll root; a popstate keeps it', () => {
	const app = mountLayout();
	flushSync();
	const main = document.body.querySelector<HTMLElement>('main.shell-main');
	if (main === null) {
		throw new Error('missing main');
	}
	const setter = observeScrollTop(main);
	setter.mockClear();

	// A route entered by a link or a goto starts at the frame's top:
	// SvelteKit's document scroll reset no longer reaches <main>.
	navigate('link');
	expect(setter).toHaveBeenCalledWith(0);
	setter.mockClear();

	// A history navigation keeps the scroll root where the wall left
	// it (the back-from-viewer return: the wall remounts and refetches
	// either way -- nothing restores a stale offset).
	navigate('popstate');
	expect(setter).not.toHaveBeenCalled();
	unmount(app);
	flushSync();
});

test('the top bar hides while the album detail is active (the page owns its chrome)', () => {
	// R3 (the 2026-11-07 owner pin): the album detail replaces the
	// shell's global topbar with its own contextual bar -- while the
	// route is active the shell renders NO topbar (the main region
	// fills the frame), and every other route keeps it.
	pageState.route.id = '/(content)/albums/[id]';
	const album = mountLayout();
	flushSync();
	expect(document.body.querySelector('.shell-topbar')).toBeNull();
	const albumMain = document.body.querySelector<HTMLElement>('main.shell-main');
	expect(albumMain).not.toBeNull();
	unmount(album);
	flushSync();

	// Any other route: the topbar is back.
	pageState.route.id = '/(content)/timeline';
	const regular = mountLayout();
	flushSync();
	expect(document.body.querySelector('.shell-topbar')).not.toBeNull();
	unmount(regular);
	flushSync();
});

test('the top bar carries the lucide chrome icons (menu, search, account)', () => {
	const app = mountLayout();
	flushSync();

	// The hamburger: an icon-only button, its accessible name the
	// aria-label, the svg decorative and lucide (the 24-grid/2px
	// round-cap icon language of brand spec §5).
	const hamburger = document.body.querySelector<HTMLButtonElement>(
		'button[aria-label="Open navigation"]'
	);
	expect(hamburger?.querySelector('svg.lucide-menu')).not.toBeNull();

	// The search field's leading magnifier: decorative, inside the
	// field's wrap, before the input.
	const search = document.body.querySelector<HTMLInputElement>('input.topbar-search');
	const wrap = search?.closest('.topbar-search-wrap');
	expect(wrap).not.toBeNull();
	const icon = wrap?.querySelector('svg.lucide-search');
	expect(icon).not.toBeNull();
	expect(wrap?.firstElementChild).toBe(icon);

	// The account avatar: an icon-only button with its aria-label.
	const avatar = document.body.querySelector<HTMLButtonElement>('button[aria-label="Account"]');
	expect(avatar?.querySelector('svg.lucide-circle-user')).not.toBeNull();
	unmount(app);
	flushSync();
});

test('the top bar search box navigates to the search page on commit (Enter)', () => {
	const app = mountLayout();
	flushSync();

	// The box rides a search form: typing and committing (Enter)
	// navigates to /search with the typed text as the query -- a
	// navigation trigger, not live search; the search page owns the
	// query state from there.
	const form = document.body.querySelector<HTMLFormElement>('form.topbar-search-wrap');
	expect(form?.getAttribute('role')).toBe('search');
	const box = form?.querySelector<HTMLInputElement>('input.topbar-search');
	if (form === null || box === null) {
		throw new Error('missing topbar search form');
	}

	box!.value = '海边 日落';
	box!.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));

	// The navigation carries the encoded query.
	expect(goto).toHaveBeenCalledWith(`/search?query=${encodeURIComponent('海边 日落')}`);

	// An empty commit lands on the bare search page (the filter
	// browse's own face).
	box!.value = '   ';
	box!.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
	form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
	expect(goto).toHaveBeenLastCalledWith('/search');

	unmount(app);
	flushSync();
});
