import { createRawSnippet, flushSync, mount, unmount } from 'svelte';
import { beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Layout from './+layout.svelte';

// The admin section's layout under the full-bleed
// rework: the rail is a FLUSH FULL-HEIGHT TRACK (border-r as the
// only separator -- no outer radius, no outer border, no shadow, no
// gap to the content, no top offset; the rounded capsule is dead),
// and the content column owns the gutter main no longer carries
// (px-6/pt-4/pb-12), with the row's min-h-full floor keeping the
// track tall as the viewport even when a page's content is short.
// The 200px rail proportion is unchanged.

function mountLayout(): ReturnType<typeof mount> {
	return mount(Layout, {
		target: document.body,
		props: {
			data: {},
			params: {},
			children: createRawSnippet(() => ({ render: () => '<p>admin body</p>' }))
		}
	});
}

beforeEach(async () => {
	document.body.textContent = '';
	vi.clearAllMocks();
	await initI18n();
});

test('the rail is a flush full-height track: border-r, no capsule chrome, no gap', () => {
	const app = mountLayout();
	flushSync();

	const row = document.body.querySelector<HTMLElement>('.admin');
	expect(row?.className).toContain('flex');
	expect(row?.className).toContain('min-h-full');
	// No gap between the rail and the content column.
	expect(row?.className).not.toContain('gap-');

	const rail = document.body.querySelector<HTMLElement>('.admin-nav');
	expect(rail?.className).toContain('border-r');
	// The capsule chrome is dead: no outer border (beyond the
	// border-r), no radius, no shadow, no top offset. (The rail's
	// own internal gap-2 between the list and the footnote stays --
	// inner rhythm, not the capsule's outer gap.)
	expect(rail?.className).not.toContain('top-4');
	expect(rail?.className).not.toContain('rounded-card');
	expect(rail?.className).not.toContain('shadow-card');
	// The 200px proportion is unchanged.
	expect(rail?.className).toContain('w-[200px]');
	// The rail's FACE spans the column: the outer nav takes the row's
	// stretch (no self-start) so its background and border-r reach the
	// content column's full height.
	expect(rail?.className).not.toContain('self-start');
	expect(rail?.className).not.toContain('sticky');

	// The menu pins on the INNER layer: sticky + self-start (a
	// stretched sticky element has zero travel and scrolls away), the
	// viewport cap, and its own overflow so the menu scrolls alone.
	const pin = rail?.querySelector<HTMLElement>('.admin-nav__pin');
	expect(pin?.className).toContain('sticky');
	expect(pin?.className).toContain('top-0');
	expect(pin?.className).toContain('self-start');
	expect(pin?.className).toContain('max-h-[calc(100dvh-53px)]');
	expect(pin?.className).toContain('overflow-y-auto');

	unmount(app);
	flushSync();
});

test('the content column owns the gutter main no longer carries', () => {
	const app = mountLayout();
	flushSync();

	const body = document.body.querySelector<HTMLElement>('.admin-body');
	expect(body?.className).toContain('flex-1');
	expect(body?.className).toContain('min-w-0');
	expect(body?.className).toContain('px-6');
	expect(body?.className).toContain('pt-4');
	// The 48px bottom breathing room.
	expect(body?.className).toContain('pb-12');

	unmount(app);
	flushSync();
});

test('the nav renders the five sections with Machine learning named and Backup inert', () => {
	const app = mountLayout();
	flushSync();

	const nav = document.body.querySelector<HTMLElement>('.admin-nav');
	expect(nav?.getAttribute('aria-label')).toBe('Administration');
	const links = Array.from(nav?.querySelectorAll<HTMLAnchorElement>('a') ?? []);
	expect(links.map((link) => link.textContent?.trim())).toEqual([
		'Users',
		'Instance settings',
		'Machine learning',
		'Jobs'
	]);
	// The backup placeholder stays visible but inert.
	const placeholder = nav?.querySelector<HTMLButtonElement>('button[disabled]');
	expect(placeholder?.textContent?.trim()).toBe('Backup');

	unmount(app);
	flushSync();
});
