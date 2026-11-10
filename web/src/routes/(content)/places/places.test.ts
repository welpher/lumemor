import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import Page from './+page.svelte';
import { sessionStatus } from '$lib/auth/session.svelte';

// The places page: the city card grid (the derived-cover collage,
// the shaded name, the honest per-caller count), the country label
// through the client's locale table (Intl region names), the map
// placeholder's honest copy, the empty state, the error box, the
// SSE refetch on the asset family, and the locale join riding the
// request. A city click deep-links into the filtered browse
// (/search?placeIds=). $app/navigation is not used by this page;
// the events module is mocked with a captured handler registry.

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
	representativeCoverAssetIds: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f72', '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f73']
};

const beijing = {
	placeId: 1816670,
	name: '北京',
	country: 'CN',
	assetCount: 42,
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

test('the grid renders the city cards with the honest counts and the deep links', async () => {
	const fetchMock = vi.fn(async () => jsonResponse(200, [beijing, paris]));
	vi.stubGlobal('fetch', fetchMock);
	const page = mountPage();
	await settle();

	expect(fetchMock).toHaveBeenCalledWith('/api/v1/places?locale=en', undefined);
	expect(cards()).toHaveLength(2);

	const first = cards()[0];
	// The deep link into the filtered browse.
	expect(first?.getAttribute('href')).toBe('/search?placeIds=1816670');
	// The shaded city name and the locale country label.
	expect(first?.querySelector('.place-card__name')?.textContent).toBe('北京');
	expect(first?.querySelector('.place-card__region')?.textContent).toBe('China');
	// The honest per-caller count.
	expect(first?.querySelector('.place-card__count')?.textContent).toContain('42');
	// The covers render from the representative ids.
	const cover = first?.querySelector('.place-card__cover');
	expect(cover?.querySelectorAll('img')).toHaveLength(1);

	const second = cards()[1];
	expect(second?.getAttribute('href')).toBe('/search?placeIds=2988507');
	expect(second?.querySelector('.place-card__region')?.textContent).toBe('France');
	expect(second?.querySelectorAll('img')).toHaveLength(2);

	// The page head's summary line and the map placeholder's honest
	// copy.
	expect(document.body.querySelector('.places__sub')?.textContent).toContain('2 cities');
	expect(document.body.querySelector('.places__sub')?.textContent).toContain('59 photos');
	expect(document.body.querySelector('.places__map')?.textContent).toContain(
		'The map view arrives with a self-hosted tile decision'
	);

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

	expect(cards()[0]?.querySelector('.place-card__region')?.textContent).toBe('中国');
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
	const empty = document.body.querySelector('.places__empty');
	expect(empty?.textContent).toContain('No places yet');
	expect(empty?.textContent).toContain('Photos whose GPS resolves to a city');
	// The map placeholder stays honest even with nothing to show.
	expect(document.body.querySelector('.places__map')).not.toBeNull();

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
