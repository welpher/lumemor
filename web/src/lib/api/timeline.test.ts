import { get } from 'svelte/store';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { createTimelineClient, TIMELINE_PAGE_LIMIT, type TimelineClient, type TimelineItem } from './timeline';
import { forceLogout, sessionStatus } from '$lib/auth/session.svelte';

// The pagination client contract: keyset cursor walk with the pinned
// page size, the in-flight guard (one fetch per trigger -- the M2d
// review focus), id deduplication, the error/retry affordance's
// state, the SSE convergence of loaded items, and the batch runner's
// optimistic updates with rollback. Every case drives the client
// through an injected fetch mock; the SSE half feeds applyEvent the
// parsed events the real subscription module would deliver.

const PAGE_ONE_END = '2026-10-01T00:00:00Z';

const itemDefaults = {
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image' as const,
	originalFilename: 'IMG_0001.jpg',
	fileSize: 1000,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d659a2feaa0c55ad015',
	createdAt: '2026-10-29T09:12:00Z',
	takenAtSource: 'exif' as const,
	isTrashed: false,
	isFavorite: false,
	isReady: true,
	width: 4000,
	height: 3000,
	durationMs: null,
	blurhash: 'UHL6aAof00WBWB00ayWB00',
	livePhotoVideoId: null
};

function item(id: string, takenAt: string, overrides: Partial<TimelineItem> = {}): TimelineItem {
	return { id, takenAt, ...itemDefaults, ...overrides };
}

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

function page(items: TimelineItem[], nextCursor?: string): Response {
	return jsonResponse(200, nextCursor === undefined ? { items } : { items, nextCursor });
}

function detailOf(timelineItem: TimelineItem, overrides: Record<string, unknown> = {}): Response {
	return jsonResponse(200, {
		description: '',
		exif: {},
		isHidden: false,
		...timelineItem,
		...overrides
	});
}

/** A fetch mock answering per-call route rules. */
type Route = (url: string, init?: RequestInit) => Response | Promise<Response>;

function clientWith(route: Route): TimelineClient {
	return createTimelineClient({
		fetcher: vi.fn(async (input: RequestInfo | URL, init?: RequestInit) =>
			route(String(input), init)
		)
	});
}

const pageOneItems = [
	item('a1', '2026-11-02T10:00:00Z'),
	item('a2', '2026-11-01T09:00:00Z'),
	item('a3', '2026-10-13T12:30:00Z')
];

beforeEach(async () => {
	// Boot the session store unauthenticated: the 401 interceptor
	// stays inert for these tests (no mid-session flips unless a
	// case arranges one).
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
});

test('the first page fetches without a cursor and walks nextCursor onward', async () => {
	const seen: string[] = [];
	const timeline = clientWith((url) => {
		seen.push(url);
		if (url === '/api/v1/assets/timeline?limit=100') {
			return page(pageOneItems, 'CURSOR-1');
		}
		if (url === '/api/v1/assets/timeline?limit=100&cursor=CURSOR-1') {
			return page([item('b1', '2026-09-01T08:00:00Z')]);
		}
		throw new Error(`unexpected fetch ${url}`);
	});

	await timeline.loadMore();
	expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
	expect(get(timeline).hasMore).toBe(true);

	await timeline.loadMore();
	expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3', 'b1']);
	expect(get(timeline).hasMore).toBe(false);

	// The exhausted client never fetches again.
	await timeline.loadMore();
	expect(seen).toHaveLength(2);
});

test('the page limit is the pinned contract default', async () => {
	const timeline = clientWith((url) => {
		expect(url.startsWith('/api/v1/assets/timeline?limit=')).toBe(true);
		return page([]);
	});
	expect(TIMELINE_PAGE_LIMIT).toBe(100);
	await timeline.loadMore();
});

test('the in-flight guard collapses concurrent triggers into one fetch', async () => {
	let releases = 0;
	const fetcher = vi.fn(async () => {
		releases += 1;
		return page(pageOneItems);
	});
	const timeline = createTimelineClient({ fetcher });

	// Two triggers in the same frame (the sentinel and a button).
	void timeline.loadMore();
	void timeline.loadMore();
	await timeline.loadMore();

	expect(fetcher).toHaveBeenCalledTimes(1);
	expect(releases).toBe(1);
});

test('a page repeating already-loaded ids appends nothing (dedup by construction)', async () => {
	let call = 0;
	const timeline = clientWith(() => {
		call += 1;
		// The second page repeats a1 (a server-side regression shape)
		// plus one genuinely new item.
		return call === 1
			? page(pageOneItems, 'CURSOR-1')
			: page([item('a1', '2026-11-02T10:00:00Z'), item('b1', '2026-09-01T08:00:00Z')]);
	});

	await timeline.loadMore();
	await timeline.loadMore();

	const ids = get(timeline).items.map((i) => i.id);
	expect(ids).toEqual(['a1', 'a2', 'a3', 'b1']);
	expect(new Set(ids).size).toBe(ids.length);
});

test('a failed page load parks in error with the code key; retry re-runs and recovers', async () => {
	let fail = true;
	const timeline = clientWith((url, init) => {
		if (fail) {
			return jsonResponse(500, { code: 'INTERNAL' });
		}
		return page(pageOneItems);
	});

	await timeline.loadMore();
	let state = get(timeline);
	expect(state.status).toBe('error');
	expect(state.errorKey).toBe('internal');
	expect(state.items).toEqual([]);

	fail = false;
	await timeline.loadMore();
	state = get(timeline);
	expect(state.status).toBe('ready');
	expect(state.errorKey).toBe('');
	expect(state.items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
});

test('a network failure maps to the internal key', async () => {
	const timeline = clientWith(() => Promise.reject(new TypeError('network down')));
	await timeline.loadMore();
	expect(get(timeline).errorKey).toBe('internal');
});

test('a mid-session 401 page load flips the session store (the interceptor path)', async () => {
	// Arrange an authenticated session first.
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(200, [])));
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('authenticated');

	const timeline = clientWith(() => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }));
	await timeline.loadMore();

	expect(get(timeline).status).toBe('error');
	expect(sessionStatus()).toBe('unauthenticated');
	forceLogout();
});

describe('SSE convergence', () => {
	test('asset.trashed and asset.deleted remove loaded tiles', async () => {
		const timeline = clientWith(() => page(pageOneItems));
		await timeline.loadMore();

		await timeline.applyEvent({ id: 1, name: 'asset.trashed', assetId: 'a2', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a3']);

		await timeline.applyEvent({ id: 2, name: 'asset.deleted', assetId: 'a1', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a3']);
	});

	test('a removed id cannot return through a racing page response (tombstone)', async () => {
		let call = 0;
		const timeline = clientWith(() => {
			call += 1;
			if (call === 1) {
				return page(pageOneItems, 'CURSOR-1');
			}
			// The second page's server snapshot predates the trash of a2.
			return page([item('a2', '2026-11-01T09:00:00Z'), item('b1', '2026-09-01T08:00:00Z')]);
		});
		await timeline.loadMore();
		await timeline.applyEvent({ id: 1, name: 'asset.trashed', assetId: 'a2', kind: 'image' });

		await timeline.loadMore();

		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a3', 'b1']);
	});

	test('asset.ready patches the loaded tile from the refetched detail', async () => {
		const timeline = clientWith((url) => {
			if (url.includes('a2')) {
				return detailOf(item('a2', '2026-11-01T09:00:00Z'), {
					isReady: true,
					blurhash: 'UHL6aAof00WBWB00ayWB10',
					width: 4032,
					height: 3024
				});
			}
			return page(pageOneItems.map((i) => item(i.id, i.takenAt, { isReady: false, blurhash: null })));
		});
		await timeline.loadMore();
		expect(get(timeline).items.find((i) => i.id === 'a2')?.blurhash).toBeNull();

		await timeline.applyEvent({ id: 3, name: 'asset.ready', assetId: 'a2', kind: 'image' });

		const patched = get(timeline).items.find((i) => i.id === 'a2');
		expect(patched?.isReady).toBe(true);
		expect(patched?.blurhash).toBe('UHL6aAof00WBWB00ayWB10');
		// The patch keeps its position and the neighbors stay put.
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
		// The loaded favorite flag survives a detail patch (the detail
		// endpoint carries no isFavorite).
		expect(patched?.isFavorite).toBe(false);
	});

	test('asset.updated with a hidden detail removes the tile; unhide returns it in place', async () => {
		let hidden = true;
		const timeline = clientWith((url) => {
			if (url.includes('a2')) {
				return detailOf(item('a2', '2026-11-01T09:00:00Z'), { isHidden: hidden });
			}
			return page(pageOneItems);
		});
		await timeline.loadMore();

		await timeline.applyEvent({ id: 1, name: 'asset.updated', assetId: 'a2', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a3']);

		hidden = false;
		await timeline.applyEvent({ id: 2, name: 'asset.updated', assetId: 'a2', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
	});

	test('asset.restored inserts a live item inside the loaded window, not below it', async () => {
		const timeline = clientWith((url) => {
			if (url.includes('r1')) {
				return detailOf(item('r1', '2026-10-20T10:00:00Z'));
			}
			if (url.includes('r2')) {
				return detailOf(item('r2', '2026-08-20T10:00:00Z'));
			}
			return page(pageOneItems);
		});
		await timeline.loadMore();

		// Inside the window: inserted at its sorted position.
		await timeline.applyEvent({ id: 1, name: 'asset.restored', assetId: 'r1', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'r1', 'a3']);

		// Below the oldest loaded item: unloaded territory, skipped.
		await timeline.applyEvent({ id: 2, name: 'asset.restored', assetId: 'r2', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'r1', 'a3']);
	});

	test('a 404 detail removes the tile (purged or foreign)', async () => {
		const timeline = clientWith((url) =>
			url.includes('a1') ? jsonResponse(404, { code: 'ASSET_NOT_FOUND' }) : page(pageOneItems)
		);
		await timeline.loadMore();

		await timeline.applyEvent({ id: 1, name: 'asset.ready', assetId: 'a1', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a2', 'a3']);
	});

	test('the first item of a settled-empty timeline is inserted', async () => {
		const timeline = clientWith((url) =>
			url.includes('n1') ? detailOf(item('n1', '2026-11-03T08:00:00Z')) : page([])
		);
		await timeline.loadMore();
		expect(get(timeline).items).toEqual([]);
		expect(get(timeline).hasMore).toBe(false);

		await timeline.applyEvent({ id: 1, name: 'asset.uploaded', assetId: 'n1', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['n1']);
	});

	test('a malformed event payload shape (failed detail) leaves the state alone', async () => {
		const timeline = clientWith((url) =>
			url.includes('a1') ? jsonResponse(200, { nothing: true }) : page(pageOneItems)
		);
		await timeline.loadMore();

		await timeline.applyEvent({ id: 1, name: 'asset.ready', assetId: 'a1', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
	});
});

describe('batch runner', () => {
	test('favorite flips the flags optimistically and reports the set outcome', async () => {
		const posts: Array<{ action: string; ids: string[] }> = [];
		const timeline = clientWith((url, init) => {
			if (url === '/api/v1/assets/batch') {
				posts.push(JSON.parse(String(init?.body)));
				return jsonResponse(200, { affected: 2, missing: [] });
			}
			return page(pageOneItems);
		});
		await timeline.loadMore();

		const outcome = await timeline.runBatch('favorite', ['a1', 'a2']);
		expect(outcome).toEqual({ ok: true, affected: 2, missing: [] });
		expect(posts).toEqual([{ action: 'favorite', ids: ['a1', 'a2'] }]);
		expect(get(timeline).items.filter((i) => i.isFavorite).map((i) => i.id)).toEqual(['a1', 'a2']);
		expect(get(timeline).items.find((i) => i.id === 'a3')?.isFavorite).toBe(false);
	});

	test('unfavorite clears the flags of an all-favorite selection', async () => {
		const timeline = clientWith((url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(200, { affected: 1, missing: [] });
			}
			return page(pageOneItems.map((i) => item(i.id, i.takenAt, { isFavorite: true })));
		});
		await timeline.loadMore();

		await timeline.runBatch('unfavorite', ['a1']);
		expect(get(timeline).items.find((i) => i.id === 'a1')?.isFavorite).toBe(false);
		expect(get(timeline).items.find((i) => i.id === 'a2')?.isFavorite).toBe(true);
	});

	test('trash removes the tiles optimistically and reports the missing partition', async () => {
		const timeline = clientWith((url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(200, { affected: 1, missing: ['zz'] });
			}
			return page(pageOneItems);
		});
		await timeline.loadMore();

		const outcome = await timeline.runBatch('trash', ['a2', 'zz']);
		expect(outcome).toEqual({ ok: true, affected: 1, missing: ['zz'] });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a3']);
	});

	test('a failed batch rolls the optimistic removal back at the sorted positions', async () => {
		const timeline = clientWith((url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(500, { code: 'INTERNAL' });
			}
			return page(pageOneItems);
		});
		await timeline.loadMore();

		const outcome = await timeline.runBatch('hide', ['a2', 'a3']);
		expect(outcome).toEqual({ ok: false, errorKey: 'internal' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
	});

	test('a failed favorite batch restores the pre-action flags', async () => {
		const timeline = clientWith((url) => {
			if (url === '/api/v1/assets/batch') {
				return Promise.reject(new TypeError('network down'));
			}
			return page(pageOneItems.map((i) => item(i.id, i.takenAt, { isFavorite: true })));
		});
		await timeline.loadMore();

		const outcome = await timeline.runBatch('unfavorite', ['a1']);
		expect(outcome).toEqual({ ok: false, errorKey: 'internal' });
		expect(get(timeline).items.find((i) => i.id === 'a1')?.isFavorite).toBe(true);
	});

	test('unhide and restore leave the loaded state untouched', async () => {
		const timeline = clientWith((url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(200, { affected: 0, missing: [] });
			}
			return page(pageOneItems);
		});
		await timeline.loadMore();

		await timeline.runBatch('restore', ['a1']);
		await timeline.runBatch('unhide', ['a1']);
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
	});

	test('an empty id list never reaches the endpoint', async () => {
		const timeline = clientWith(() => {
			throw new Error('no fetch expected');
		});
		const outcome = await timeline.runBatch('trash', []);
		expect(outcome).toEqual({ ok: false, errorKey: 'internal' });
	});

	test('rollback re-inserts older items at their sorted place after interleaved inserts', async () => {
		// a1 (newest) and a3 hidden; the rollback must re-place a3
		// after a2 even though the array now also carries r1.
		const timeline = clientWith((url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(500, { code: 'INTERNAL' });
			}
			if (url.includes('r1')) {
				return detailOf(item('r1', '2026-10-20T10:00:00Z'));
			}
			return page(pageOneItems);
		});
		await timeline.loadMore();
		await timeline.applyEvent({ id: 1, name: 'asset.restored', assetId: 'r1', kind: 'image' });

		const outcome = await timeline.runBatch('hide', ['a1', 'a3']);
		expect(outcome.ok).toBe(false);
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'r1', 'a3']);
	});
});
