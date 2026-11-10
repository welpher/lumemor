import { createRawSnippet, flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Layout from './+layout.svelte';

// The tool shell's app frame (R2-1's consistency arm): the admin
// shell shares the content shell's structure (a top bar above one
// scrollable column), so it gets the same h-dvh treatment -- the
// top bar pinned by the frame, <main> the sole scroller, the scroll
// root focused for keyboard scrolling, and the push-navigation
// scroll reset. $app/navigation is mocked (goto for the back mark,
// afterNavigate captured like the content shell's test; the router
// delivers the mounting navigation after the render, so tests fire
// it explicitly).

const { navigationCallbacks } = vi.hoisted(() => ({
	navigationCallbacks: [] as Array<(navigation: { type: string }) => void>
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn(),
	afterNavigate: (callback: (navigation: { type: string }) => void) => {
		navigationCallbacks.push(callback);
	}
}));

function mountLayout(): ReturnType<typeof mount> {
	return mount(Layout, {
		target: document.body,
		props: {
			data: {},
			params: {},
			children: createRawSnippet(() => ({ render: () => '<p>tool body</p>' }))
		}
	});
}

beforeEach(async () => {
	document.body.textContent = '';
	navigationCallbacks.length = 0;
	vi.clearAllMocks();
	await initI18n();
});

test('the tool shell is an h-dvh frame whose one scroller is <main>', () => {
	const app = mountLayout();
	flushSync();

	const shell = document.body.querySelector<HTMLElement>('.tool');
	expect(shell?.className).toContain('h-dvh');
	expect(shell?.className).toContain('overflow-hidden');
	expect(shell?.className).toContain('flex');
	expect(shell?.className).toContain('flex-col');
	expect(shell?.className).not.toContain('min-h-screen');

	const main = document.body.querySelector<HTMLElement>('main.tool-main');
	expect(main?.className).toContain('overflow-y-auto');
	expect(main?.className).toContain('flex-1');
	expect(main?.className).toContain('min-h-0');
	// THE FULL-BLEED PIN (2026-11-09): main carries NO padding -- the
	// content column owns its own gutter, so the admin rail below
	// splits the scroll extent edge to edge.
	expect(main?.className).not.toContain('p-');
	expect(main?.getAttribute('tabindex')).toBe('-1');

	const topbar = document.body.querySelector<HTMLElement>('.tool-topbar');
	expect(topbar?.className).not.toContain('sticky');
	unmount(app);
	flushSync();
});

test('the frame focuses its scroll root and resets it on push navigations', () => {
	const app = mountLayout();
	flushSync();

	// The mounting navigation (delivered after the render) leaves the
	// focus on the scroll root.
	for (const callback of [...navigationCallbacks]) {
		callback({ type: 'enter' });
	}
	flushSync();
	const main = document.body.querySelector<HTMLElement>('main.tool-main');
	expect(document.activeElement).toBe(main);
	const setter = vi.fn();
	Object.defineProperty(main, 'scrollTop', {
		get: () => 0,
		set: setter,
		configurable: true
	});
	setter.mockClear();

	for (const callback of [...navigationCallbacks]) {
		callback({ type: 'link' });
	}
	flushSync();
	expect(setter).toHaveBeenCalledWith(0);
	setter.mockClear();

	for (const callback of [...navigationCallbacks]) {
		callback({ type: 'popstate' });
	}
	flushSync();
	expect(setter).not.toHaveBeenCalled();
	unmount(app);
	flushSync();
});

test('the back-to-content mark carries its icon', () => {
	const app = mountLayout();
	flushSync();

	const back = document.body.querySelector<HTMLButtonElement>('button[aria-label="Back to your photos"]');
	expect(back?.querySelector('svg.lucide-chevron-left')).not.toBeNull();
	unmount(app);
	flushSync();
});

// --- the offline strip and the `?` sheet (web IA §6, the shared shell organs) --

const { connectionState } = vi.hoisted(() => ({
	// The offline strip's truth, mocked at the store the strip
	// reads: the tool shell mounts the SAME strip the content shell
	// does -- its jobs and models surfaces subscribe to the shared
	// SSE stream, so a dropped connection is this shell's truth too.
	connectionState: {
		status: 'closed' as 'closed' | 'connecting' | 'open' | 'down',
		downSince: null as number | null
	}
}));

vi.mock('$lib/events/connection.svelte', () => ({
	streamStatus: () => connectionState.status,
	streamDownSince: () => connectionState.downSince,
	markStreamConnecting: () => undefined,
	markStreamOpen: () => undefined,
	markStreamDown: () => undefined,
	markStreamClosed: () => undefined
}));

beforeEach(() => {
	connectionState.status = 'closed';
	connectionState.downSince = null;
});

test('the offline strip pins above the tool topbar on a dropped connection', () => {
	// The healthy stream renders nothing.
	connectionState.status = 'open';
	const healthy = mountLayout();
	flushSync();
	expect(document.body.querySelector('.offline-strip')).toBeNull();
	unmount(healthy);
	flushSync();

	// The dropped connection rises the same yellow band, as the
	// frame's FIRST CHILD above the tool shell's simplified bar.
	connectionState.status = 'down';
	connectionState.downSince = Date.now();
	const dropped = mountLayout();
	flushSync();

	const shell = document.body.querySelector<HTMLElement>('.tool');
	if (shell === null) {
		throw new Error('missing tool shell');
	}
	const strip = document.body.querySelector<HTMLElement>('.offline-strip');
	expect(strip).not.toBeNull();
	expect(shell.firstElementChild).toBe(strip);
	expect(shell.firstElementChild?.nextElementSibling?.className).toContain('tool-topbar');

	unmount(dropped);
	flushSync();
});

test('the `?` sheet mounts with the tool shell and answers the toggle', () => {
	const app = mountLayout();
	flushSync();
	expect(document.body.querySelector('.modal__panel')).toBeNull();

	window.dispatchEvent(new KeyboardEvent('keydown', { key: '?', cancelable: true, bubbles: true }));
	flushSync();
	const sheet = document.body.querySelector<HTMLElement>('.modal__panel[role="dialog"]');
	expect(sheet?.getAttribute('aria-label')).toBe('Keyboard shortcuts');

	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true, bubbles: true }));
	flushSync();
	expect(document.body.querySelector('.modal__panel')).toBeNull();

	unmount(app);
	flushSync();
});
