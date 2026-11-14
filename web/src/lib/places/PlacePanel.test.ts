import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import PlacePanel from './PlacePanel.svelte';

// The map's place panel: the head names the selection (one place, its
// shared country, or the area label) and counts its photos; the body
// is the shared wall in the panel frame (month headers, the compact
// axis) with NO selection state and no batch bar. The walk is the
// timeline client over the placeIds narrow.

const { handlers } = vi.hoisted(() => ({
	handlers: [] as Array<(event: unknown) => void>
}));

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

const place = (overrides: Record<string, unknown>) => ({
	placeId: 2988507,
	name: 'Paris',
	country: 'FR',
	assetCount: 17,
	latitude: 48.85341,
	longitude: 2.3488,
	takenAtFrom: '2024-01-21T09:05:00Z',
	takenAtTo: '2024-01-25T18:20:00Z',
	representativeCoverAssetIds: [],
	...overrides
});

const paris = place({});
const nice = place({ placeId: 2990440, name: 'Nice', assetCount: 7 });
const beijing = place({
	placeId: 1816670,
	name: '北京',
	country: 'CN',
	assetCount: 42
});

function item(id: string, takenAt: string) {
	return {
		id,
		takenAt,
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

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
	await tick();
}

function mountPanel(places: ReturnType<typeof place>[], onclose = vi.fn()): ReturnType<typeof mount> {
	const component = mount(PlacePanel, { target: document.body, props: { places, onclose } });
	flushSync();
	return component;
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	await initI18n();
	setLocale('en');
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the panel names one place, counts its photos and renders the shared wall', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		expect(String(input)).toContain('placeIds=2988507');
		return new Response(JSON.stringify({ items: [item('a1', '2026-06-20T12:00:00Z')] }), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		});
	});
	vi.stubGlobal('fetch', fetchMock);
	const onclose = vi.fn();
	const panel = mountPanel([paris], onclose);
	await settle();

	expect(document.body.querySelector('[data-place-panel-name]')?.textContent).toBe('Paris');
	expect(document.body.querySelector('.places__panel-count')?.getAttribute('aria-label')).toBe('17 photos');

	// The body is the SHARED wall in its panel frame: the month
	// header, the panel root and the compact axis.
	expect(document.body.querySelector('.wall__groups--panel')).not.toBeNull();
	expect(document.body.querySelector('.wall__month-header')?.textContent?.trim()).toBe('June 2026');
	expect(document.body.querySelector('[data-testid="panel-axis"]')).not.toBeNull();

	// The panel is for looking: no check circles, no batch bar.
	expect(document.body.querySelector('.tile__check')).toBeNull();
	expect(document.body.querySelector('.batchbar')).toBeNull();

	// The close button hands back.
	document.body
		.querySelector<HTMLButtonElement>('.places__panel-close')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	expect(onclose).toHaveBeenCalledTimes(1);

	unmount(panel);
	flushSync();
});

test('a shared-country group is named by the country; the count sums its places', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(
			async () =>
				new Response(JSON.stringify({ items: [] }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		)
	);
	const panel = mountPanel([paris, nice]);
	await settle();

	expect(document.body.querySelector('[data-place-panel-name]')?.textContent).toBe('France');
	expect(document.body.querySelector('.places__panel-count')?.getAttribute('aria-label')).toBe('24 photos');
	// The walk carries both places' ids (the bubble's union read).
	expect(vi.mocked(fetch).mock.calls[0]?.[0]).toContain('placeIds=2988507,2990440');

	unmount(panel);
	flushSync();
});

test('a cross-country group falls back to the area label', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(
			async () =>
				new Response(JSON.stringify({ items: [] }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		)
	);
	const panel = mountPanel([paris, beijing]);
	await settle();

	expect(document.body.querySelector('[data-place-panel-name]')?.textContent).toBe('This area');

	unmount(panel);
	flushSync();
});

test('an empty spot states it in place, not as a blank wall', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(
			async () =>
				new Response(JSON.stringify({ items: [] }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		)
	);
	const panel = mountPanel([paris]);
	await settle();

	expect(document.body.querySelector('.places__panel-empty')?.textContent).toContain(
		'This place has no photos you can see right now.'
	);

	unmount(panel);
	flushSync();
});
