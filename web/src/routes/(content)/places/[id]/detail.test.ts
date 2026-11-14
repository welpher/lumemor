import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import Page from './+page.svelte';

// The place detail: the context bar (back + city 20px + country 13px
// + count, NO right-side actions), the place's own wall with its
// month axis, and the timeline's curation subset on a selection
// (favorite / hide / trash). The reads are the grouping read (the
// place's localized face) and the timeline placeIds narrow.

const { pageState, handlers } = vi.hoisted(() => ({
	pageState: { params: { id: '1816670' }, route: { id: '/(content)/places/[id]' } },
	handlers: [] as Array<(event: unknown) => void>
}));

vi.mock('$app/state', () => ({ page: pageState }));
vi.mock('$lib/events', () => ({
	subscribeAssetEvents: (listener: (event: unknown) => void) => {
		handlers.push(listener);
		return () => {
			const index = handlers.indexOf(listener);
			if (index !== -1) {
				handlers.splice(index, 1);
			}
		};
	}
}));

const beijing = {
	placeId: 1816670,
	name: '北京',
	country: 'CN',
	assetCount: 42,
	latitude: 39.9075,
	longitude: 116.39723,
	takenAtFrom: '2019-04-08T10:12:00Z',
	takenAtTo: '2026-08-02T15:40:00Z',
	representativeCoverAssetIds: []
};

function item(id: string) {
	return {
		id,
		takenAt: '2026-06-20T12:00:00Z',
		ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
		kind: 'image',
		originalFilename: `${id}.jpg`,
		fileSize: 1000,
		mimeType: 'image/jpeg',
		checksum: '9f86d081884c7d659a2feaa0c55ad015',
		createdAt: '2026-10-29T09:12:00Z',
		takenAtSource: 'exif',
		isTrashed: false,
		isFavorite: false,
		isReady: true,
		width: 4000,
		height: 3000,
		durationMs: null,
		blurhash: 'UHL6aAof00WBWB00ayWB00',
		livePhotoVideoId: null
	};
}

function json(body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
}

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
	await tick();
}

function mountPage(): ReturnType<typeof mount> {
	const component = mount(Page, { target: document.body, props: {} });
	flushSync();
	return component;
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	pageState.params = { id: '1816670' };
	await initI18n();
	setLocale('en');
	// The grouping read (the place's face) and the wall walk answer
	// their own shapes; the deep-session probe is not involved (the
	// interceptor stays inert on these 200s).
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url.startsWith('/api/v1/places')) {
				return json([beijing]);
			}
			if (url.startsWith('/api/v1/assets/timeline')) {
				expect(url).toContain('placeIds=1816670');
				return json({ items: [item('a1')] });
			}
			return json({});
		})
	);
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the bar carries the city, the country and the count, and no right-side action', async () => {
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('[data-place-name]')?.textContent).toBe('北京');
	expect(document.body.querySelector('[data-place-country]')?.textContent).toBe('China');
	expect(document.body.querySelector('.place-bar__count')?.getAttribute('aria-label')).toBe('42 photos');
	// The back link is the only affordance: the bar has no buttons at
	// all (a place has nothing to manage).
	const back = document.body.querySelector('.place-bar__back');
	expect(back?.getAttribute('href')).toBe('/places');
	expect(back?.getAttribute('aria-label')).toBe('Back to places');
	expect(document.body.querySelectorAll('.place-bar button').length).toBe(0);

	// The page title follows the place.
	expect(document.title).toBe('北京 · lumemor');

	unmount(page);
	flushSync();
});

test('the wall renders the place\u2019s photos with the month axis', async () => {
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('.wall__month-header')?.textContent?.trim()).toBe('June 2026');
	expect(document.body.querySelector('.tile')).not.toBeNull();
	expect(document.body.querySelector('.scrubber')).not.toBeNull();

	// The curation subset: a selection rises the bar with exactly
	// favorite / hide / trash.
	const check = document.body.querySelector<HTMLElement>('.tile__check');
	expect(check).not.toBeNull();
	check?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const bar = document.body.querySelector('.batchbar');
	expect(bar).not.toBeNull();
	const labels = Array.from(bar?.querySelectorAll('button') ?? []).map((button) => button.getAttribute('aria-label'));
	expect(labels).toContain('Favorite');
	expect(labels).toContain('Hide');
	expect(labels).toContain('Trash');
	expect(labels).not.toContain('Add tag');

	unmount(page);
	flushSync();
});

test('an unknown place renders the one unreachable face', async () => {
	pageState.params = { id: '999999' };
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('[data-place-name]')).toBeNull();
	expect(document.body.textContent).toContain('This page will not open');
	// The way back survives.
	expect(document.body.querySelector('.place-bar__back')?.getAttribute('href')).toBe('/places');

	unmount(page);
	flushSync();
});

test('a malformed id is the same dead end', async () => {
	pageState.params = { id: 'nope' };
	const page = mountPage();
	await settle();

	expect(document.body.textContent).toContain('This page will not open');

	unmount(page);
	flushSync();
});

test('an empty place states its empty face', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url.startsWith('/api/v1/places')) {
				return json([beijing]);
			}
			return json({ items: [] });
		})
	);
	const page = mountPage();
	await settle();

	expect(document.body.textContent).toContain('No photos here');
	expect(document.body.querySelector('.tile')).toBeNull();

	unmount(page);
	flushSync();
});
