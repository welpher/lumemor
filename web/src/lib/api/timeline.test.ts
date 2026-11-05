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

test('the album narrow rides every page fetch as the leading query parameter', async () => {
	const seen: string[] = [];
	const timeline = createTimelineClient({
		fetcher: vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			seen.push(url);
			if (url === '/api/v1/assets/timeline?albumId=al1&limit=100') {
				return page(pageOneItems, 'CURSOR-A');
			}
			if (url === '/api/v1/assets/timeline?albumId=al1&limit=100&cursor=CURSOR-A') {
				return page([item('b1', '2026-09-01T08:00:00Z')]);
			}
			throw new Error(`unexpected fetch ${url}`);
		}),
		albumId: 'al1'
	});

	await timeline.loadMore();
	await timeline.loadMore();

	expect(seen).toEqual([
		'/api/v1/assets/timeline?albumId=al1&limit=100',
		'/api/v1/assets/timeline?albumId=al1&limit=100&cursor=CURSOR-A'
	]);
	expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3', 'b1']);
});

test('a reset mid-flight discards the stale page response', async () => {
	// The first page-one fetch hangs until the test releases it; the
	// reset restarts the walk while it is in flight, so the stale
	// response must never apply over the fresh page one.
	let release: ((response: Response) => void) | undefined;
	let hanging = true;
	const timeline = createTimelineClient({
		fetcher: vi.fn(async (input: RequestInfo | URL) => {
			const url = String(input);
			if (url === '/api/v1/assets/timeline?limit=100' && hanging) {
				hanging = false;
				return new Promise<Response>((resolve) => {
					release = resolve;
				});
			}
			return page([item('fresh', '2026-11-05T10:00:00Z')]);
		})
	});

	const first = timeline.loadMore();
	timeline.reset();
	release?.(page(pageOneItems, 'CURSOR-1'));
	await first;

	await timeline.loadMore();

	const state = get(timeline);
	expect(state.items.map((i) => i.id)).toEqual(['fresh']);
	expect(state.status).toBe('ready');
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

test('a 401 page load maps to the session-expired key (no credential text on the wall)', async () => {
	const timeline = clientWith(() => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }));
	await timeline.loadMore();

	const state = get(timeline);
	expect(state.status).toBe('error');
	// A 401 here means the session died; the login endpoint's
	// credential message must never reach the wall's error box.
	expect(state.errorKey).toBe('session_expired');
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

	test('a restored id the wall previously loaded re-enters below the loaded tail', async () => {
		// The window rule's exemption for known ids (the e2e-found
		// bug): the trash event evicts a loaded tile, and the restore
		// re-converges it BELOW the new loaded tail. The next page's
		// cursor sits below the evicted position, so pagination will
		// never deliver the tile again -- the upsert must re-insert
		// it, or the wall silently loses a photo the server still
		// serves (the M2d full-chain story caught exactly this on the
		// trash page's restore).
		const timeline = clientWith((url) => {
			if (url.includes('a3')) {
				return detailOf(item('a3', '2026-10-13T12:30:00Z'));
			}
			// A next page waits below the loaded one: the sharper
			// variant -- the next page's cursor sits BELOW a3's
			// position, so nothing but this upsert can bring it back.
			return page(pageOneItems, 'cursor');
		});
		await timeline.loadMore();
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
		expect(get(timeline).hasMore).toBe(true);

		// The eviction: a3 (the loaded tail) leaves through its event.
		await timeline.applyEvent({ id: 1, name: 'asset.trashed', assetId: 'a3', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2']);

		// The restore: a3 is older than the new tail (a2), but it was
		// loaded before -- it re-enters at its sorted position.
		await timeline.applyEvent({ id: 2, name: 'asset.restored', assetId: 'a3', kind: 'image' });
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
	});

	test('a restored tile re-enters when the page response lands between the racing events', async () => {
		// The e2e-found race's other order: the replayed trash event
		// tombstones the id, then the page response -- fresher than
		// that event, the server had already restored the asset --
		// lands while the restore event's convergence is still in
		// flight. The tombstone filters the append, but the response
		// still placed the id inside the loaded range, so the
		// restore's upsert must re-insert it (the wall must not lose
		// a photo the listing still serves).
		let resolvePage: ((response: Response) => void) | undefined;
		const resolveDetail: Array<(response: Response) => void> = [];
		const timeline = clientWith((url) => {
			if (url.includes('a3')) {
				return new Promise<Response>((resolve) => resolveDetail.push(resolve));
			}
			// The page fetch is held so the events can race it.
			return new Promise<Response>((resolve) => {
				resolvePage = resolve;
			});
		});

		// The page load starts, held in flight.
		const loading = timeline.loadMore();

		// The replayed trash event tombstones a3 before the page
		// lands (the items are still empty -- the removal parks in
		// the tombstone alone).
		await timeline.applyEvent({ id: 1, name: 'asset.trashed', assetId: 'a3', kind: 'image' });
		// The restore event starts its convergence; its detail fetch
		// is held.
		const converging = timeline.applyEvent({
			id: 2,
			name: 'asset.restored',
			assetId: 'a3',
			kind: 'image'
		});
		await new Promise((resolve) => setTimeout(resolve, 0));

		// The page response lands first: the tombstone filters a3
		// from the append.
		resolvePage?.(page(pageOneItems));
		await loading;
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2']);

		// The restore's detail lands second: a3 is a member, and the
		// page response vouched for its position -- it re-enters.
		resolveDetail[0]?.(detailOf(item('a3', '2026-10-13T12:30:00Z')));
		await converging;
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
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

	describe('convergence sequencing (the out-of-order fix)', () => {
		/** Resolve a held fetch one macrotask later (the loop runs). */
		async function drain(): Promise<void> {
			await new Promise((resolve) => setTimeout(resolve, 0));
		}

		test('a superseded detail never applies, even when the newer fetch fails', async () => {
			const pending: Array<(response: Response) => void> = [];
			let detailCalls = 0;
			const timeline = clientWith((url) => {
				if (url.includes('a2')) {
					detailCalls += 1;
					// The newer claim's fetch dies on the network.
					if (detailCalls === 2) {
						return Promise.reject(new TypeError('network down'));
					}
					return new Promise<Response>((resolve) => pending.push(resolve));
				}
				return page(pageOneItems);
			});
			await timeline.loadMore();

			// Event 5's fetch is held in flight; event 6 supersedes it.
			const first = timeline.applyEvent({
				id: 5,
				name: 'asset.updated',
				assetId: 'a2',
				kind: 'image'
			});
			void timeline.applyEvent({ id: 6, name: 'asset.updated', assetId: 'a2', kind: 'image' });
			expect(pending).toHaveLength(1);

			// The held response carries event 5's stale snapshot (a
			// refined blurhash and width). It must be discarded under
			// event 6's claim -- applying it would leave the wall on
			// state the newer event already moved past.
			pending[0]?.(detailOf(item('a2', '2026-11-01T09:00:00Z'), { blurhash: 'STALE', width: 1111 }));
			await drain();
			expect(detailCalls).toBe(2);
			// Event 6's fetch fails: nothing applies, the tile keeps
			// its page-loaded values.
			await first;

			const kept = get(timeline).items.find((i) => i.id === 'a2');
			expect(kept?.blurhash).toBe('UHL6aAof00WBWB00ayWB00');
			expect(kept?.width).toBe(4000);
			expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
		});

		test('a removal supersedes an in-flight convergence: no resurrection', async () => {
			const pending: Array<(response: Response) => void> = [];
			const timeline = clientWith((url) => {
				if (url.includes('a2')) {
					return new Promise<Response>((resolve) => pending.push(resolve));
				}
				return page(pageOneItems);
			});
			await timeline.loadMore();

			// Event 5 starts a convergence that is still in flight
			// when the trash (event 6) removes the tile.
			const inFlight = timeline.applyEvent({
				id: 5,
				name: 'asset.ready',
				assetId: 'a2',
				kind: 'image'
			});
			await timeline.applyEvent({ id: 6, name: 'asset.trashed', assetId: 'a2', kind: 'image' });
			expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a3']);

			// The stale response answers with a LIVE detail; without
			// the removal's claim the upsert would clear the
			// tombstone and resurrect the tile.
			pending[0]?.(detailOf(item('a2', '2026-11-01T09:00:00Z')));
			await inFlight;

			expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a3']);
		});

		test('a burst of events for one asset collapses: only the newest claim fetches', async () => {
			const pending: Array<(response: Response) => void> = [];
			let detailCalls = 0;
			const timeline = clientWith((url) => {
				if (url.includes('a2')) {
					detailCalls += 1;
					return new Promise<Response>((resolve) => pending.push(resolve));
				}
				return page(pageOneItems);
			});
			await timeline.loadMore();

			// Three events in one breath: the first fetch runs, the
			// newer claims queue (newest wins), and only the newest
			// claim re-converges when the running fetch lands.
			const all = Promise.all([
				timeline.applyEvent({ id: 5, name: 'asset.updated', assetId: 'a2', kind: 'image' }),
				timeline.applyEvent({ id: 6, name: 'asset.updated', assetId: 'a2', kind: 'image' }),
				timeline.applyEvent({ id: 7, name: 'asset.updated', assetId: 'a2', kind: 'image' })
			]);
			expect(pending).toHaveLength(1);

			pending[0]?.(detailOf(item('a2', '2026-11-01T09:00:00Z'), { isHidden: true }));
			await drain();
			// Event 6 never fetches: 7 superseded it while queued.
			expect(detailCalls).toBe(2);
			expect(pending).toHaveLength(2);

			pending[1]?.(detailOf(item('a2', '2026-11-01T09:00:00Z'), { isHidden: false }));
			await all;

			// The newest event's live detail decided: a2 stays.
			expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
			expect(detailCalls).toBe(2);
		});
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

	test('a 401 batch failure maps to the session-expired key and rolls back', async () => {
		const timeline = clientWith((url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' });
			}
			return page(pageOneItems);
		});
		await timeline.loadMore();

		const outcome = await timeline.runBatch('trash', ['a1']);
		// The dead session shows the session message, never the
		// login endpoint's credential text.
		expect(outcome).toEqual({ ok: false, errorKey: 'session_expired' });
		// The optimistic removal rolled back (the guard redirects).
		expect(get(timeline).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
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

describe('curated view membership (favorites and hidden walls)', () => {
	// The two curated walls drive the same client through the view
	// option: the page walk is theirs unchanged, the convergence's
	// membership rule and the membership-ending batch action follow
	// the view instead.
	function viewClient(view: 'favorites' | 'hidden', route: Route): TimelineClient {
		return createTimelineClient({
			fetcher: vi.fn(async (input: RequestInfo | URL, init?: RequestInit) =>
				route(String(input), init)
			),
			path: `/api/v1/assets/${view}`,
			view
		});
	}

	test('unfavorite on the favorites wall removes the tiles (the membership ends)', async () => {
		const posts: Array<{ action: string; ids: string[] }> = [];
		const favorites = viewClient('favorites', (url, init) => {
			if (url === '/api/v1/assets/batch') {
				posts.push(JSON.parse(String(init?.body)));
				return jsonResponse(200, { affected: 2, missing: [] });
			}
			if (url === '/api/v1/assets/favorites?limit=100') {
				return page(pageOneItems.map((i) => item(i.id, i.takenAt, { isFavorite: true })));
			}
			throw new Error(`unexpected fetch ${url}`);
		});
		await favorites.loadMore();

		const outcome = await favorites.runBatch('unfavorite', ['a1', 'a2']);
		expect(outcome).toEqual({ ok: true, affected: 2, missing: [] });
		expect(posts).toEqual([{ action: 'unfavorite', ids: ['a1', 'a2'] }]);
		// The unfavorited rows leave the view: no SSE event follows
		// (favorites are silent), the optimistic removal is final.
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a3']);
	});

	test('an unfavorited row stays gone through later convergence events (sticky membership)', async () => {
		let pages = 0;
		const favorites = viewClient('favorites', (url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(200, { affected: 1, missing: [] });
			}
			if (/\/api\/v1\/assets\/a2$/.test(url)) {
				return detailOf(item('a2', '2026-11-01T09:00:00Z'));
			}
			if (url === '/api/v1/assets/favorites?limit=100') {
				pages += 1;
				// The second page's server snapshot predates the
				// unfavorite (a racing response).
				return pages === 1
					? page(pageOneItems.map((i) => item(i.id, i.takenAt, { isFavorite: true })), 'CURSOR-1')
					: page([item('a2', '2026-11-01T09:00:00Z', { isFavorite: true })]);
			}
			throw new Error(`unexpected fetch ${url}`);
		});
		await favorites.loadMore();

		const outcome = await favorites.runBatch('unfavorite', ['a2']);
		expect(outcome).toEqual({ ok: true, affected: 1, missing: [] });
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a1', 'a3']);

		// A later non-removal SSE event for the same asset (a
		// description PATCH publishes asset.updated) converges on a
		// live detail -- but favoriteness is not evaluable from the
		// detail, so the wall must keep the ended membership: the
		// tile stays gone.
		await favorites.applyEvent({ id: 1, name: 'asset.updated', assetId: 'a2', kind: 'image' });
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a1', 'a3']);

		// The tombstone survived the event: a racing page response
		// that still lists the asset cannot resurrect it either.
		await favorites.loadMore();
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a1', 'a3']);
	});

	test('a failed unfavorite on the favorites wall rolls the rows back', async () => {
		const favorites = viewClient('favorites', (url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(500, { code: 'INTERNAL' });
			}
			if (url === '/api/v1/assets/favorites?limit=100') {
				return page(pageOneItems.map((i) => item(i.id, i.takenAt, { isFavorite: true })));
			}
			throw new Error(`unexpected fetch ${url}`);
		});
		await favorites.loadMore();

		const outcome = await favorites.runBatch('unfavorite', ['a2']);
		expect(outcome).toEqual({ ok: false, errorKey: 'internal' });
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
	});

	test('a failed unfavorite clears the stickiness, so later events converge normally', async () => {
		const favorites = viewClient('favorites', (url) => {
			if (url === '/api/v1/assets/batch') {
				return jsonResponse(500, { code: 'INTERNAL' });
			}
			if (/\/api\/v1\/assets\/a2$/.test(url)) {
				return detailOf(item('a2', '2026-11-01T09:00:00Z', { isFavorite: true }));
			}
			if (url === '/api/v1/assets/favorites?limit=100') {
				return page(pageOneItems.map((i) => item(i.id, i.takenAt, { isFavorite: true })));
			}
			throw new Error(`unexpected fetch ${url}`);
		});
		await favorites.loadMore();

		const outcome = await favorites.runBatch('unfavorite', ['a2']);
		expect(outcome).toEqual({ ok: false, errorKey: 'internal' });
		// The rollback restored the tile...
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);

		// ...and ended the stickiness: a later asset.updated event
		// converges the row normally (the favorite stays a member).
		await favorites.applyEvent({ id: 1, name: 'asset.updated', assetId: 'a2', kind: 'image' });
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
		expect(get(favorites).items.find((i) => i.id === 'a2')?.isFavorite).toBe(true);
	});

	test('unhide on the hidden wall removes the tiles (the membership ends)', async () => {
		const hidden = viewClient('hidden', (url, init) => {
			if (url === '/api/v1/assets/batch') {
				expect(JSON.parse(String(init?.body))).toEqual({ action: 'unhide', ids: ['a2'] });
				return jsonResponse(200, { affected: 1, missing: [] });
			}
			if (url === '/api/v1/assets/hidden?limit=100') {
				return page(pageOneItems);
			}
			throw new Error(`unexpected fetch ${url}`);
		});
		await hidden.loadMore();

		await hidden.runBatch('unhide', ['a2']);
		expect(get(hidden).items.map((i) => i.id)).toEqual(['a1', 'a3']);
	});

	test('the hidden wall convergence keeps hidden rows and removes unhidden ones', async () => {
		const hidden = viewClient('hidden', (url) => {
			if (url.includes('a2')) {
				return detailOf(item('a2', '2026-11-01T09:00:00Z'), { isHidden: false });
			}
			if (url.includes('a1')) {
				// Still hidden: the convergence patches (or keeps) the
				// row, never removes it.
				return detailOf(item('a1', '2026-11-02T10:00:00Z', { isFavorite: true }), { isHidden: true });
			}
			if (url === '/api/v1/assets/hidden?limit=100') {
				return page(pageOneItems);
			}
			throw new Error(`unexpected fetch ${url}`);
		});
		await hidden.loadMore();

		// asset.updated (the unhide event) converges the now-live row
		// out of the hidden wall.
		await hidden.applyEvent({ id: 1, name: 'asset.updated', assetId: 'a2', kind: 'image' });
		expect(get(hidden).items.map((i) => i.id)).toEqual(['a1', 'a3']);

		// A row that stayed hidden (an edit of its description, say)
		// keeps its place.
		await hidden.applyEvent({ id: 2, name: 'asset.updated', assetId: 'a1', kind: 'image' });
		expect(get(hidden).items.map((i) => i.id)).toEqual(['a1', 'a3']);
	});

	test('the hidden wall drops a trashed row (the direct removal arm)', async () => {
		const hidden = viewClient('hidden', (url) => {
			if (url === '/api/v1/assets/hidden?limit=100') {
				return page(pageOneItems);
			}
			throw new Error(`unexpected fetch ${url}`);
		});
		await hidden.loadMore();

		// A hidden-then-trashed row leaves through the unambiguous
		// removal arm, exactly like the timeline wall (the listing
		// answers live rows only).
		await hidden.applyEvent({ id: 1, name: 'asset.trashed', assetId: 'a1', kind: 'image' });
		expect(get(hidden).items.map((i) => i.id)).toEqual(['a2', 'a3']);
	});

	test('the favorites wall convergence removes a hidden favorite and returns it unhidden', async () => {
		const details = new Map<string, boolean>();
		const favorites = viewClient('favorites', (url) => {
			const match = /\/api\/v1\/assets\/(a1)$/.exec(url);
			if (match !== null) {
				return detailOf(item('a1', '2026-11-02T10:00:00Z', { isFavorite: true }), {
					isHidden: details.get('a1') ?? false
				});
			}
			if (url === '/api/v1/assets/favorites?limit=100') {
				return page(pageOneItems.map((i) => item(i.id, i.takenAt, { isFavorite: true })));
			}
			throw new Error(`unexpected fetch ${url}`);
		});
		await favorites.loadMore();

		// Hidden: the favorite row survives, the view filters it out.
		details.set('a1', true);
		await favorites.applyEvent({ id: 1, name: 'asset.updated', assetId: 'a1', kind: 'image' });
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a2', 'a3']);

		// Unhidden again: the favorite returns inside the loaded
		// window, at its sorted place.
		details.set('a1', false);
		await favorites.applyEvent({ id: 2, name: 'asset.updated', assetId: 'a1', kind: 'image' });
		expect(get(favorites).items.map((i) => i.id)).toEqual(['a1', 'a2', 'a3']);
	});
});
