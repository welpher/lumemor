import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import Page from './+page.svelte';
import { placesPageState } from './places-page-state.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';

// The places page: the flat city list (the derived-cover collage, the
// shaded name, the country + capture years beside the honest count),
// the two-face switch (the list on screen, the map behind
// `?mode=map`), the map face's place panel (a pin opens it in place,
// the same pin / Escape / close collapses it), the list's scroll
// seat, the empty state, the error box, and the SSE refetch on the
// asset family. The address supplies the view; the events module is
// mocked with a captured handler registry and the map itself with the
// colocated stand-in (MapLibre needs a layout host).

const { gotoMock } = vi.hoisted(() => ({
	gotoMock: vi.fn(async () => undefined)
}));

vi.mock('$app/state', async () => ({
	page: (await import('./places-page-state.svelte')).placesPageState
}));
vi.mock('$app/navigation', async () => {
	const { placesPageState } = await import('./places-page-state.svelte');
	return {
		goto: gotoMock,
		// The real replaceState updates the page's address in place;
		// the panel's selection is read from it, so the double must
		// too.
		replaceState: (url: URL | string) => {
			placesPageState.url = new URL(String(url));
		}
	};
});
vi.mock('$lib/map/MapView.svelte', async () => ({
	default: (await import('./MapViewStub.svelte')).default
}));

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

const paris = {
	placeId: 2988507,
	name: 'Paris',
	country: 'FR',
	assetCount: 17,
	latitude: 48.85341,
	longitude: 2.3488,
	takenAtFrom: '2024-01-21T09:05:00Z',
	takenAtTo: '2024-01-25T18:20:00Z',
	representativeCoverAssetIds: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f72', '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f73']
};

const beijing = {
	placeId: 1816670,
	name: '北京',
	country: 'CN',
	assetCount: 42,
	latitude: 39.9075,
	longitude: 116.39723,
	takenAtFrom: '2019-04-08T10:12:00Z',
	takenAtTo: '2026-08-02T15:40:00Z',
	representativeCoverAssetIds: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70']
};

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
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
	const page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
}

function cards(): HTMLElement[] {
	return Array.from(document.body.querySelectorAll<HTMLElement>('.place-card'));
}

function buttonByText(text: string): HTMLButtonElement {
	const buttons = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button'));
	const found = buttons.find((button) => button.textContent?.trim() === text);
	if (found === undefined) {
		throw new Error(`missing button ${text}`);
	}
	return found;
}

beforeEach(async () => {
	document.body.textContent = '';
	handlers.length = 0;
	placesPageState.url = new URL('http://localhost/places');
	window.history.replaceState({}, '', '/places');
	vi.clearAllMocks();
	// Boot the session store unauthenticated (the interceptor stays
	// inert; these routes answer 200s).
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
	await initI18n();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the flat list renders the city cards with the span, the count and the place link', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, [beijing, paris]));
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(fetchMock).toHaveBeenCalledWith('/api/v1/places?locale=en', undefined);
	expect(cards()).toHaveLength(2);

	const first = cards()[0];
	// The card opens the place's own wall (never the search filter).
	expect(first?.getAttribute('href')).toBe('/places/1816670');
	// The shaded city name.
	expect(first?.querySelector('.place-card__name')?.textContent).toBe('北京');
	// The second line: the country and the capture years.
	expect(first?.querySelector('.place-card__years')?.textContent).toBe('China · 2019–2026');
	// The honest per-caller count (the glyph + the number; the
	// wording is the accessible name).
	expect(first?.querySelector('.place-card__count')?.textContent).toContain('42');
	expect(first?.querySelector('.place-card__count')?.getAttribute('aria-label')).toBe('42 photos');
	// The covers render from the representative ids.
	expect(first?.querySelector('.place-card__cover')?.querySelectorAll('img')).toHaveLength(1);

	const second = cards()[1];
	expect(second?.getAttribute('href')).toBe('/places/2988507');
	// One year when both ends agree.
	expect(second?.querySelector('.place-card__years')?.textContent).toBe('France · 2024');
	expect(second?.querySelectorAll('img')).toHaveLength(2);

	// No grouping headers: one grid, flat -- every child of the grid
	// is a card, and no heading rides between the toolbar and it.
	expect(document.body.querySelectorAll('.places__content h2').length).toBe(0);
	const grid = document.body.querySelector('.places__grid');
	expect(grid).not.toBeNull();
	for (const child of Array.from(grid?.children ?? [])) {
		expect(child.matches('a.place-card')).toBe(true);
	}

	// The page bar's count segment, and the view switch (the list is
	// the current face, the map is its own address).
	expect(document.body.querySelector('.places__title')?.textContent?.replace(/\s+/g, ' ').trim()).toBe(
		'Places · 2 cities'
	);
	const listView = document.body.querySelector('[data-places-mode="list"]');
	const mapView = document.body.querySelector('[data-places-mode="map"]');
	expect(listView?.getAttribute('aria-current')).toBe('page');
	expect(mapView?.getAttribute('aria-current')).toBeNull();
	expect(mapView?.getAttribute('href')).toBe('/places?mode=map');

	unmount(page);
	flushSync();
});

test('the locale join rides the request and the country labels', async () => {
	const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
		expect(String(input)).toBe('/api/v1/places?locale=zh-Hans');
		return jsonResponse(200, [beijing]);
	});
	vi.stubGlobal('fetch', fetchMock);
	setLocale('zh-Hans');
	const page = mountPage();
	await settle();

	expect(cards()[0]?.querySelector('.place-card__years')?.textContent).toBe('中国 · 2019–2026');
	setLocale('en');

	unmount(page);
	flushSync();
});

test('an empty grouping renders the empty state', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, []))
	);
	const page = mountPage();
	await settle();

	expect(cards()).toHaveLength(0);
	const empty = document.body.querySelector('.state-card');
	expect(empty?.textContent).toContain('No places yet');
	expect(empty?.textContent).toContain('Photos land here by city once their location resolves.');
	// The switch stands even with nothing to show.
	expect(document.body.querySelector('[data-places-mode="map"]')?.getAttribute('href')).toBe('/places?mode=map');

	unmount(page);
	flushSync();
});

test('a failed load shows the error box with a working retry', async () => {
	let failed = true;
	const fetchMock = vi.fn(async () => {
		if (failed) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return jsonResponse(200, [paris]);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(document.body.textContent).toContain('Something went wrong');

	failed = false;
	buttonByText('Try again').dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();
	expect(cards()).toHaveLength(1);

	unmount(page);
	flushSync();
});

test('the map face fills the frame and its pins open the place panel in place', async () => {
	placesPageState.url = new URL('http://localhost/places?mode=map');
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, [beijing, paris]))
	);
	const page = mountPage();
	await settle();

	// The map face replaces the list; the panel is closed at rest.
	expect(cards()).toHaveLength(0);
	expect(document.body.querySelector('[data-testid="places-map"]')).not.toBeNull();
	expect(document.body.querySelector('[data-testid="places-panel"]')).toBeNull();
	// The footer carries the grouping's honest counts.
	const counts = document.body.querySelector('.places__map-counts')?.textContent ?? '';
	expect(counts).toContain('2 cities');
	expect(counts).toContain('59 photos');
	expect(counts).toContain('2 countries');

	// A pin click opens the panel with the place's name and photo
	// count -- no navigation.
	const pin = document.body.querySelector<HTMLButtonElement>('.map__pin-stub[data-place-id="1816670"]');
	pin?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	const panel = document.body.querySelector('[data-testid="places-panel"]');
	expect(panel).not.toBeNull();
	expect(panel?.querySelector('[data-place-panel-name]')?.textContent).toBe('北京');
	expect(panel?.querySelector('.places__panel-count')?.getAttribute('aria-label')).toBe('42 photos');
	expect(gotoMock).not.toHaveBeenCalled();
	// The selection rides the address without navigating (the viewer
	// round trip restores the same spot).
	expect(placesPageState.url.searchParams.get('place')).toBe('1816670');
	// The selected pin carries its pressed state.
	expect(pin?.getAttribute('aria-pressed')).toBe('true');

	// The SAME pin again collapses the panel.
	pin?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(document.body.querySelector('[data-testid="places-panel"]')).toBeNull();
	expect(pin?.getAttribute('aria-pressed')).toBe('false');

	// A bubble's group opens as one selection: the panel sums the
	// places' photos. This fixture's group spans countries, so the
	// head falls back to the honest area label (never one member's
	// city standing in for the whole group).
	document.body
		.querySelector<HTMLButtonElement>('.map__pin-stub--bubble')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(document.body.querySelector('[data-place-panel-name]')?.textContent).toBe('This area');
	expect(document.body.querySelector('.places__panel-count')?.getAttribute('aria-label')).toBe('59 photos');

	// The close button collapses, and so does Escape.
	document.body
		.querySelector<HTMLButtonElement>('.places__panel-close')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(document.body.querySelector('[data-testid="places-panel"]')).toBeNull();

	pin?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	flushSync();
	expect(document.body.querySelector('[data-testid="places-panel"]')).not.toBeNull();
	window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
	flushSync();
	expect(document.body.querySelector('[data-testid="places-panel"]')).toBeNull();

	unmount(page);
	flushSync();
});

test('switching faces keeps the address and restores the list seat', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, [beijing, paris]))
	);
	const main = document.createElement('main');
	document.body.appendChild(main);
	let seat = 0;
	Object.defineProperty(main, 'scrollTop', {
		get: () => seat,
		set: (value: number) => {
			seat = value;
		},
		configurable: true
	});
	seat = 180;

	const page = mountPage();
	await settle();

	// To the map: the seat is captured, the address is the map face.
	document.body
		.querySelector<HTMLElement>('[data-places-mode="map"]')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();
	expect(gotoMock).toHaveBeenCalledWith('/places?mode=map');

	// The router lands: the page now renders the map face.
	placesPageState.url = new URL('http://localhost/places?mode=map');
	// The frame's navigation reset (the shell's afterNavigate) lands
	// every push at the top -- the seat must survive it.
	seat = 0;
	flushSync();
	expect(document.body.querySelector('[data-testid="places-map"]')).not.toBeNull();

	// Back to the list: the address flips and the seat returns.
	document.body
		.querySelector<HTMLElement>('[data-places-mode="list"]')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	await settle();
	expect(gotoMock).toHaveBeenCalledWith('/places');
	expect(seat).toBe(180);

	main.remove();
	unmount(page);
	flushSync();
});

test('the address restores the panel (the viewer round trip)', async () => {
	placesPageState.url = new URL('http://localhost/places?mode=map&place=1816670');
	window.history.replaceState({}, '', '/places?mode=map&place=1816670');
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, [beijing, paris]))
	);
	const page = mountPage();
	await settle();

	const panel = document.body.querySelector('[data-testid="places-panel"]');
	expect(panel).not.toBeNull();
	expect(panel?.querySelector('[data-place-panel-name]')?.textContent).toBe('北京');

	unmount(page);
	flushSync();
});

test('a list-face address drops the stray place parameter', async () => {
	// The parameter is the map face's state: on the list it selects
	// nothing, and the address sheds it so a reload or a share never
	// carries a selection the face cannot show.
	placesPageState.url = new URL('http://localhost/places?place=1816670');
	window.history.replaceState({}, '', '/places?place=1816670');
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, [beijing, paris]))
	);
	const page = mountPage();
	await settle();

	expect(document.body.querySelector('[data-testid="places-panel"]')).toBeNull();
	expect(cards()).toHaveLength(2);
	expect(placesPageState.url.searchParams.has('place')).toBe(false);

	unmount(page);
	flushSync();
});

test('an asset event refetches the grouping (the same family)', async () => {
	let calls = 0;
	const fetchMock = vi.fn(async () => {
		calls += 1;
		return jsonResponse(200, calls === 1 ? [paris] : [beijing, paris]);
	});
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();
	expect(cards()).toHaveLength(1);

	for (const handler of handlers) {
		handler({ id: 1, name: 'asset.ready', assetId: 'a1', kind: 'image' });
	}
	flushSync();
	await settle();

	// The fresh answer owns the truth.
	expect(cards()).toHaveLength(2);
	expect(fetchMock).toHaveBeenCalledTimes(2);

	unmount(page);
	flushSync();
});

test('leaving the page unsubscribes from the event stream', async () => {
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, [paris]))
	);
	const page = mountPage();
	await settle();
	expect(handlers).toHaveLength(1);

	unmount(page);
	flushSync();
	expect(handlers).toHaveLength(0);
});
