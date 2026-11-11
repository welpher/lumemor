// The timeline wall's data client: a keyset-paginated store over
// GET /api/v1/assets/timeline, the SSE convergence of the loaded
// items, and the batch curation runner behind the selection bar.
//
// THE CLIENT SHAPE (the pinned design):
//
// - One Svelte writable store of `TimelineState` (the page
//   auto-subscribes with `$timeline`); every state change replaces
//   the items array or the item objects, never mutates them, so the
//   component diffing stays exact and cheap.
// - `loadMore()` walks the keyset pagination: the first page omits
//   the cursor, every later page passes the previous response's
//   `nextCursor` (an opaque server string, never assembled here),
//   with the contract's default page size of 100. It is the ONLY
//   page-fetching entry point, guarded in flight: while a page load
//   runs, further triggers return immediately -- one fetch per
//   trigger even when the IntersectionObserver sentinel and a manual
//   button fire in the same frame (no
//   duplicate fetches, no duplicate tiles).
// - Items are deduplicated by id: pages append only ids not already
//   loaded (belt and braces around the keyset's own disjointness),
//   and ids removed by an SSE event are tombstoned so a page
//   response that raced the event (its server snapshot predates the
//   removal) cannot resurrect the tile.
// - A failed page load parks the client in `error` with the error
//   code's i18n key; the page renders a retry affordance that calls
//   `loadMore()` again with the SAME cursor (idempotent position).
// - SSE: `applyEvent` maps `asset.trashed`/`asset.deleted` to a
//   direct removal and every other asset.* event to a convergence
//   refetch of GET /api/v1/assets/{id} -- not live (trashed, hidden
//   or gone) removes the tile, live patches or inserts it. The
//   convergence is SEQUENCED by the server-monotonic event id (see
//   the converge* maps in the client): a stale detail response never
//   applies over a newer event, and a removal supersedes any
//   in-flight convergence so a removed tile cannot be resurrected.
//   The detail carries no `isFavorite`, so a patch of a loaded tile
//   keeps the loaded flag (favorites publish no events: per-user
//   state, per the contract).
// - Batch: `runBatch` applies the optimistic state first (favorite
//   flags flip, hide/trash remove the tiles), POSTs
//   /api/v1/assets/batch, and rolls the optimistic state back from
//   the pre-action snapshot when the POST fails. Success relies on
//   SSE for convergence (hide/trash events remove what the optimism
//   already removed; favorites are silent by contract, so the
//   optimistic flag is the final state).
//
// The insertion rule for convergence: the loaded items are always
// the TOP PREFIX of the keyset ordering (pagination starts from the
// newest), so a live item belongs in the loaded window exactly when
// it is not older than the oldest loaded item; anything below the
// window belongs to an unloaded page and is skipped (the run walks
// there eventually).
//
// THE CURATED VIEWS: the favorites and hidden walls drive this same
// client through the `view` option -- the page shape, ordering and
// pagination are theirs unchanged, and exactly two behaviors shift:
// the SSE convergence's membership rule (the hidden wall keeps the
// hidden rows, not the live ones) and the optimistic arm of the one
// batch action that ends an item's membership (unfavorite on
// favorites, unhide on hidden removes the tiles). The favorites wall
// cannot evaluate its membership from the convergence detail alone
// (favoriteness is not part of AssetDetail), so its ended
// memberships are STICKY: the unfavorite arm parks the ids in a
// removal set the convergence must respect, and only a rollback (the
// unfavorite failed) or a reset clears it.
//
// THE ALBUM WALL: the album detail's grid drives this same
// client through the `albumId` option -- the endpoint's album narrow
// answers the album's own live items in the timeline's exact shape
// (keyset order, per-user favorite flags), so the grid reuses the
// wall wholesale. The album page's own wallResetKey convergence
// restarts the walk when an album.items-changed event lands for the
// album.

import { get, writable } from 'svelte/store';
import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { AssetEvent } from '$lib/events';
import type { components } from '../../../../openapi/generated/web/api';

/** One timeline tile, exactly the contract's TimelineItem shape. */
export type TimelineItem = components['schemas']['TimelineItem'];

/** The detail endpoint's body (the convergence refetch target). */
type AssetDetail = components['schemas']['AssetDetail'];

/** The batch endpoint's set actions, exactly the contract's enum. */
export type BatchAction = components['schemas']['BatchRequest']['action'];

/**
 * Which wall the client serves: the aggregate timeline, or one of
 * the two curated membership views that share its shape. The page
 * shape, ordering and pagination are identical everywhere; the view
 * parameter carries the only two behavioral differences -- the
 * membership rule of the SSE convergence and the optimistic arm of
 * the batch action that ends an item's membership.
 */
export type WallView = 'timeline' | 'favorites' | 'hidden';

/** The page-fetch lifecycle of the client. */
export type TimelineStatus = 'idle' | 'loading' | 'ready' | 'error';

export type TimelineState = {
	/** The loaded prefix of the timeline, newest first (keyset order). */
	items: TimelineItem[];
	status: TimelineStatus;
	/** False once the last page arrived (no `nextCursor`). */
	hasMore: boolean;
	/** The error-code i18n key of the last failed page load ('' when none). */
	errorKey: string;
};

/** The outcome of one batch action run. */
export type BatchOutcome = { ok: true; affected: number; missing: string[] } | { ok: false; errorKey: string };

export type TimelineClient = {
	/** The store half; the page binds it with `$timeline`. */
	subscribe: (run: (state: TimelineState) => void) => () => void;
	/** Fetch the next keyset page (the first page while idle). */
	loadMore: () => Promise<void>;
	/** Drop everything and park idle (tests and future route resets). */
	reset: () => void;
	/** React to one SSE activity event (the convergence entry point). */
	applyEvent: (event: AssetEvent) => Promise<void>;
	/** Run one curation action over selected ids, optimistically. */
	runBatch: (action: BatchAction, ids: string[]) => Promise<BatchOutcome>;
};

export type TimelineClientOptions = {
	/** The fetch implementation (tests inject a mock; the app uses the global). */
	fetcher?: typeof fetch;
	/** Page size; the constraint pins 100 (the contract default). */
	limit?: number;
	/** The timeline-shaped endpoint path (the aggregate timeline by default). */
	path?: string;
	/**
	 * Which wall the client serves (see WallView). The default is the
	 * aggregate timeline; the favorites and hidden pages pass their
	 * own view so the convergence and the optimistic batch follow
	 * that view's membership rule.
	 */
	view?: WallView;
	/**
	 * The album whose own timeline this wall walks: the endpoint's
	 * `albumId` narrow, carried on every page fetch (the album
	 * detail's grid reuses the wall over the album's own items).
	 */
	albumId?: string;
};

/** The contract's default page size. */
export const TIMELINE_PAGE_LIMIT = 100;

const BATCH_ENDPOINT = '/api/v1/assets/batch';

/**
 * Build a timeline client. Two instances never share state, so tests
 * construct isolated clients and the page owns exactly one per wall.
 */
export function createTimelineClient(options: TimelineClientOptions = {}): TimelineClient {
	const fetcher = options.fetcher ?? globalThis.fetch;
	const limit = options.limit ?? TIMELINE_PAGE_LIMIT;
	const path = options.path ?? '/api/v1/assets/timeline';
	const view = options.view ?? 'timeline';
	const albumId = options.albumId;

	const store = writable<TimelineState>({
		items: [],
		status: 'idle',
		hasMore: true,
		errorKey: ''
	});

	// The continuation cursor of the next page (null while on the
	// first page) and the ids SSE events removed -- a page response
	// racing an event must not resurrect either.
	let nextCursor: string | null = null;
	// The walk's generation: reset() raises it, and a page fetch
	// whose entry generation no longer matches discards its response
	// -- a reset while a page load is in flight (the album wall's
	// membership convergence) must never let the stale page land over
	// the fresh walk. The convergence claims already had this
	// discipline through their claim maps; this is the page-walk arm.
	let generation = 0;
	const tombstones = new Set<string>();
	// The ids this wall has evidence for inside the loaded range: the
	// page responses delivered them, or a convergence placed them in
	// items. A removal event evicts a tile but ends no membership:
	// when a later event re-converges a KNOWN id as a member, it
	// re-enters even below the loaded tail -- the next page's cursor
	// sits below the evicted position, so pagination will never
	// deliver it again, and the window rule's "unloaded page's
	// territory" premise is false for it. The set is cleared by
	// reset() with the rest of the client's memory.
	const knownIds = new Set<string>();
	// The caller's own feed facts, filled by ensureFeedPrefs below:
	// the wall cards carry the caller's per-album timeline preference
	// (the excludable cards -- the own directory albums and every
	// shared one; the own reference sets omit the switch, the sync
	// card is never excludable). The aggregate's insert gate reads
	// exactly this map, so a replayed or lived asset event for an
	// asset the caller's union holds WITHOUT the aggregate's feed
	// (an excluded album's rescan arms) can never insert a tile the
	// pages would never deliver.
	const feedPrefs = new Map<string, boolean>();
	// The favorites wall's ended memberships: the ids the unfavorite
	// arm removed. Favoriteness is NOT evaluable from the detail the
	// SSE convergence refetches (AssetDetail carries no isFavorite),
	// so without this set any later non-removal event for the asset
	// (a description edit's asset.updated, asset.ready,
	// asset.restored) would count it as a member again and resurrect
	// the tile. The set is sticky by design: only rollback() (the
	// failed unfavorite puts the tile back) and reset() (a fresh
	// wall) clear it -- a refavorite reaches this wall through a new
	// route mount, which resets the client.
	const removedMembers = new Set<string>();

	// Convergence sequencing (the out-of-order fix): SSE event ids
	// are server-monotonic, so each asset's convergence is OWNED by
	// the newest event that claimed it -- `convergeClaim` maps an
	// asset id to that event id. A detail response applies only when
	// its claiming id is still the asset's newest claim; a newer
	// event (or a removal) supersedes the in-flight response, which
	// is discarded instead of applied over fresher state. A removal
	// also RAISES the claim, so a stale live detail can never
	// resurrect a removed tile (the upsert path clears tombstones).
	//
	// The same maps collapse bursts: while one detail fetch runs
	// (`convergeRunning`), newer claims park in `convergeQueued`
	// (newest wins) and the single-flight loop re-converges with
	// the newest id once the running fetch lands -- a burst of N
	// events for one asset costs at most two fetches, and only the
	// newest event's detail ever applies.
	const convergeClaim = new Map<string, number>();
	const convergeQueued = new Map<string, number>();
	const convergeRunning = new Set<string>();

	/**
	 * The keyset ordering comparator: takenAt descending with the id
	 * descending as the tiebreak. Negative means `a` renders first
	 * (newer). Wire takenAt values are RFC 3339 with an optional
	 * fractional-second part of up to microsecond digits, so the
	 * comparison parses the timestamp (a plain string compare would
	 * misorder `...:00.5Z` against `...:00Z`).
	 */
	function newerFirst(a: TimelineItem, b: TimelineItem): number {
		const delta = timeValue(b.takenAt) - timeValue(a.takenAt);
		if (delta !== 0) {
			return delta;
		}
		return b.id < a.id ? 1 : b.id > a.id ? -1 : 0;
	}

	function timeValue(takenAt: string): number {
		const milliseconds = Date.parse(takenAt);
		if (Number.isNaN(milliseconds)) {
			return 0;
		}
		const fraction = /\.(\d+)/.exec(takenAt);
		const subSecond = fraction === null ? 0 : Number.parseInt(fraction[1], 10) / 10 ** fraction[1].length;
		return milliseconds + subSecond;
	}

	function setError(errorKey: string): void {
		store.update((state) => ({ ...state, status: 'error', errorKey }));
	}

	/** Structurally validate a page body; null when it is not a page. */
	function pageFrom(body: unknown): { items: TimelineItem[]; nextCursor: string | null } | null {
		if (body === null || typeof body !== 'object') {
			return null;
		}
		const { items, nextCursor: cursor } = body as { items?: unknown; nextCursor?: unknown };
		if (!Array.isArray(items)) {
			return null;
		}
		for (const entry of items) {
			if (entry === null || typeof entry !== 'object') {
				return null;
			}
			const candidate = entry as Partial<TimelineItem>;
			if (typeof candidate.id !== 'string' || typeof candidate.takenAt !== 'string') {
				return null;
			}
		}
		if (cursor !== undefined && typeof cursor !== 'string') {
			return null;
		}
		return { items: items as TimelineItem[], nextCursor: cursor ?? null };
	}

	/** Structurally validate a detail body; null when it is not a detail. */
	function detailFrom(body: unknown): AssetDetail | null {
		if (body === null || typeof body !== 'object') {
			return null;
		}
		const candidate = body as Partial<AssetDetail>;
		if (typeof candidate.id !== 'string' || typeof candidate.takenAt !== 'string') {
			return null;
		}
		if (candidate.isTrashed !== true && candidate.isTrashed !== false) {
			return null;
		}
		if (candidate.isHidden !== true && candidate.isHidden !== false) {
			return null;
		}
		return body as AssetDetail;
	}

	function removeIds(ids: string[]): void {
		const doomed = new Set(ids);
		for (const id of ids) {
			tombstones.add(id);
		}
		store.update((state) => ({
			...state,
			items: state.items.filter((timelineItem) => !doomed.has(timelineItem.id))
		}));
	}

	/** Insert one item at its sorted position, or patch the loaded one. */
	function upsertLiveItem(candidate: TimelineItem): void {
		if (removedMembers.has(candidate.id)) {
			// The favorites wall's ended membership: no re-insertion,
			// and the tombstone survives (this is the path that
			// clears one -- see the sticky set's comment).
			return;
		}
		tombstones.delete(candidate.id);
		const state = get(store);
		const index = state.items.findIndex((existing) => existing.id === candidate.id);
		if (index !== -1) {
			const items = [...state.items];
			items[index] = candidate;
			knownIds.add(candidate.id);
			store.update((current) => ({ ...current, items }));
			return;
		}
		if (state.items.length === 0) {
			// An empty and settled timeline receives its first item; an
			// idle or still-loading one skips (the in-flight page owns it).
			if (state.status === 'ready' && !state.hasMore) {
				knownIds.add(candidate.id);
				store.update((current) => ({ ...current, items: [candidate] }));
			}
			return;
		}
		const oldest = state.items[state.items.length - 1];
		if (oldest !== undefined && newerFirst(candidate, oldest) > 0 && !knownIds.has(candidate.id)) {
			// Older than the loaded prefix AND no evidence places it
			// inside the loaded range: an unloaded page's territory. A
			// known id is exempt (see knownIds) -- a page response or a
			// convergence placed it there, and nothing but this upsert
			// will bring it back.
			return;
		}
		const items = [...state.items];
		const insertAt = items.findIndex((existing) => newerFirst(candidate, existing) < 0);
		if (insertAt === -1) {
			items.push(candidate);
		} else {
			items.splice(insertAt, 0, candidate);
		}
		knownIds.add(candidate.id);
		store.update((current) => ({ ...current, items }));
	}

	/** Map a live detail onto the tile shape, keeping the loaded favorite flag. */
	function itemFromDetail(detail: AssetDetail): TimelineItem {
		const existing = get(store).items.find((timelineItem) => timelineItem.id === detail.id);
		return {
			id: detail.id,
			ownerId: detail.ownerId,
			kind: detail.kind,
			originalFilename: detail.originalFilename,
			fileSize: detail.fileSize,
			mimeType: detail.mimeType,
			checksum: detail.checksum,
			createdAt: detail.createdAt,
			takenAt: detail.takenAt,
			takenAtSource: detail.takenAtSource,
			isTrashed: false,
			isFavorite: existing?.isFavorite ?? false,
			isReady: detail.isReady,
			// The timeline item's dimensions are always present (the
			// justified wall's contract); the detail's stay nullable
			// while unrefined, so the 0 placeholder crosses here and
			// the wall's layout falls back to the assumed aspect until
			// the ready event refetches the refined values.
			width: detail.width ?? 0,
			height: detail.height ?? 0,
			durationMs: detail.durationMs,
			blurhash: detail.blurhash,
			livePhotoVideoId: detail.livePhotoVideoId
		};
	}

	/**
	 * The caller's feed facts, fetched once per client on the first
	 * unknown-id convergence that needs them (see feedPrefs). A failed
	 * fetch keeps the map empty: the gate then answers the contract's
	 * own true default -- the pre-gate behavior -- which is honest
	 * degradation, the API failing wholesale at that point and not
	 * this one face.
	 */
	let feedPrefsReady: Promise<void> | null = null;
	async function ensureFeedPrefs(): Promise<void> {
		if (feedPrefsReady === null) {
			feedPrefsReady = (async () => {
				try {
					const response = await apiFetch('/api/v1/albums', undefined, fetcher);
					if (!response.ok) {
						return;
					}
					const body = (await response.json()) as components['schemas']['AlbumWallResponse'];
					for (const card of body.albums ?? []) {
						if (typeof card.inTimeline === 'boolean') {
							feedPrefs.set(card.id, card.inTimeline);
						}
					}
				} catch {
					// The wall fetch failed: the map stays empty and the
					// gate falls back to the feed default.
				}
			})();
		}
		await feedPrefsReady;
	}

	/**
	 * Fetch and apply one asset's detail under its claiming event id.
	 * The claim may move while the fetch is in flight (a newer event
	 * or a removal); a superseded response never applies.
	 */
	async function convergeOnce(assetId: string, eventId: number): Promise<void> {
		if ((convergeClaim.get(assetId) ?? -1) > eventId) {
			// A newer event already owns the asset's convergence.
			return;
		}
		let response: Response;
		try {
			response = await apiFetch(`/api/v1/assets/${encodeURIComponent(assetId)}`, undefined, fetcher);
		} catch {
			// Network failure: keep the current state, later events
			// or the next pagination run converge.
			return;
		}
		if (convergeClaim.get(assetId) !== eventId) {
			// Superseded mid-flight (a newer event, or a removal):
			// the fresher claim's own fetch decides the state.
			return;
		}
		if (response.status === 404) {
			// Purged or foreign: out of the wall either way.
			removeIds([assetId]);
			return;
		}
		if (!response.ok) {
			return;
		}
		let body: unknown;
		try {
			body = await response.json();
		} catch {
			return;
		}
		const detail = detailFrom(body);
		if (detail === null || detail.id !== assetId) {
			return;
		}
		// THE AGGREGATE'S INSERT GATE: the stream replays its retained
		// ring on every fresh connect, and the events name assets the
		// caller's union holds WITHOUT the aggregate's feed -- an
		// excluded album's asset (the creator's own preference; the
		// rescan's ready and updated arms carry such events) must
		// never insert a tile the pages would never deliver. The
		// detail's albums face says where the asset lives for this
		// caller; the caller's per-album preference rides the wall
		// cards; the two together are the union's own rule. The sync
		// arm is self-evident (the core stream is never excludable, no
		// fetch), an own reference set never feeds (the wall card
		// omits the switch), and every other arm reads the caller's
		// pref with the contract's own true default. A page-delivered
		// id is exempt (knownIds): the server's own pages are the
		// membership, and the per-asset patch below must keep running
		// for their rows. The curated views keep the detail's own
		// rule: their membership is the caller's per-user rows, never
		// the album feed.
		if (albumId === undefined && view !== 'favorites' && view !== 'hidden' && !knownIds.has(assetId)) {
			const face = detail.albums ?? [];
			// The map arms (a directory album at either role, a shared
			// reference set) need the caller's pref facts; the sync arm
			// and the own-reference-set arm are self-evident.
			if (face.some((ref) => ref.type !== 'sync' && !(ref.type === 'virtual' && ref.callerRole === 'owner'))) {
				await ensureFeedPrefs();
			}
			const feeds = face.some((ref) => {
				if (ref.type === 'sync') {
					return true;
				}
				if (ref.type === 'virtual' && ref.callerRole === 'owner') {
					return false;
				}
				return feedPrefs.get(ref.id) ?? true;
			});
			if (!feeds) {
				return;
			}
		}
		// The view's membership rule, decided by the detail: the
		// timeline and favorites walls keep live (unhidden, untrashed)
		// rows; the hidden wall keeps exactly the hidden ones (an
		// unhidden row returns to the live views and leaves this
		// one). A trashed row leaves every wall. The favorites wall
		// adds one rule the detail cannot decide by itself: an id
		// whose membership the unfavorite arm ended (the sticky set)
		// is not a member, whatever the detail says.
		const member =
			!detail.isTrashed &&
			(view === 'hidden' ? detail.isHidden : !detail.isHidden) &&
			!(view === 'favorites' && removedMembers.has(assetId));
		if (!member) {
			removeIds([assetId]);
			return;
		}
		upsertLiveItem(itemFromDetail(detail));
	}

	/**
	 * The single-flight loop behind one asset's convergence: at most
	 * one detail fetch in flight at a time, re-converging with the
	 * newest queued claim until the queue drains.
	 */
	async function runConvergeLoop(assetId: string): Promise<void> {
		convergeRunning.add(assetId);
		try {
			while (convergeQueued.has(assetId)) {
				const eventId = convergeQueued.get(assetId) ?? -1;
				convergeQueued.delete(assetId);
				await convergeOnce(assetId, eventId);
			}
		} finally {
			convergeRunning.delete(assetId);
		}
	}

	/**
	 * The convergence entry point behind every non-removal asset
	 * event: the detail decides whether the tile leaves the wall
	 * (trashed, hidden, gone) or returns to it (restored, unhidden,
	 * refined), sequenced by the event id (see the maps above).
	 *
	 * THE ALBUM NARROW'S GATE: the album's wall answers the timeline
	 * endpoint's albumId narrow, whose pages ARE the membership -- an
	 * asset event for an id no page ever delivered is not this
	 * album's member, and the detail cannot say otherwise (it carries
	 * no album list). The event replays the stream's retained ring
	 * too (an upload pipeline's asset.ready for a photo that never
	 * joined this album), so without the gate those events would
	 * upsert foreign tiles into an empty or partial grid. The
	 * narrow's membership changes arrive as album.items-changed,
	 * which restarts the walk; the aggregate wall keeps its
	 * insert-on-event behavior behind the feed gate (convergeOnce:
	 * the detail's albums face plus the caller's per-album
	 * preference decide, so an excluded album's replayed events
	 * insert nothing while the sync stream's uploads still land).
	 */
	async function convergeFromDetail(assetId: string, eventId: number): Promise<void> {
		if (albumId !== undefined && !knownIds.has(assetId)) {
			// Not a member of THIS album: nothing to converge.
			return;
		}
		const prior = convergeClaim.get(assetId) ?? -1;
		if (eventId < prior) {
			// A newer event already owns the asset's convergence; its
			// fetch is the authoritative one.
			return;
		}
		convergeClaim.set(assetId, eventId);
		if (convergeRunning.has(assetId)) {
			// Collapse the burst: the running loop re-converges with
			// this (the newest) id when its fetch lands.
			convergeQueued.set(assetId, eventId);
			return;
		}
		convergeQueued.set(assetId, eventId);
		await runConvergeLoop(assetId);
	}

	/** Restore pre-action items after a failed batch, by id. */
	function rollback(ids: string[], snapshot: TimelineItem[]): void {
		// A failed membership-ending action ends its stickiness too:
		// the tile is back, and a later convergence may keep it (the
		// unfavorite never landed, so the membership never ended).
		for (const id of ids) {
			removedMembers.delete(id);
		}
		const doomed = new Set(ids);
		const before = snapshot.filter((timelineItem) => doomed.has(timelineItem.id));
		store.update((state) => {
			const items = [...state.items];
			for (const restored of before) {
				tombstones.delete(restored.id);
				const index = items.findIndex((existing) => existing.id === restored.id);
				if (index !== -1) {
					// Present (a favorite flip): restore the pre-action object.
					items[index] = restored;
					continue;
				}
				const insertAt = items.findIndex((existing) => newerFirst(restored, existing) < 0);
				if (insertAt === -1) {
					items.push(restored);
				} else {
					items.splice(insertAt, 0, restored);
				}
			}
			return { ...state, items };
		});
	}

	async function loadMore(): Promise<void> {
		const before = get(store);
		if (before.status === 'loading') {
			// The in-flight guard: one page fetch per trigger.
			return;
		}
		if (before.status === 'ready' && !before.hasMore) {
			return;
		}
		store.update((state) => ({ ...state, status: 'loading', errorKey: '' }));
		const run = generation;

		const narrow = albumId === undefined ? '' : `albumId=${encodeURIComponent(albumId)}&`;
		const query = `${narrow}limit=${limit}${nextCursor === null ? '' : `&cursor=${encodeURIComponent(nextCursor)}`}`;
		let response: Response;
		try {
			response = await apiFetch(`${path}?${query}`, undefined, fetcher);
		} catch {
			if (run === generation) {
				setError('internal');
			}
			return;
		}
		if (run !== generation) {
			// A reset restarted the walk while this page was in
			// flight: the fresh walk's own fetch decides the state.
			return;
		}
		if (!response.ok) {
			if (response.status === 401) {
				// A 401 here means the session died mid-wall (credential
				// failures only come from the login endpoint): the
				// interceptor has flipped the store and the guard is
				// redirecting -- the box shows the session message,
				// never login-credential text.
				setError('session_expired');
				return;
			}
			const envelope = await errorEnvelope(response);
			if (run === generation) {
				setError(envelope !== null ? envelope.code.toLowerCase() : 'internal');
			}
			return;
		}
		let body: unknown;
		try {
			body = await response.json();
		} catch {
			if (run === generation) {
				setError('internal');
			}
			return;
		}
		if (run !== generation) {
			return;
		}
		const page = pageFrom(body);
		if (page === null) {
			setError('internal');
			return;
		}

		const state = get(store);
		const loaded = new Set(state.items.map((timelineItem) => timelineItem.id));
		const appended = page.items.filter(
			(timelineItem) => !loaded.has(timelineItem.id) && !tombstones.has(timelineItem.id)
		);
		// Every id the response delivered becomes known: the wall
		// rendered it, or an event racing the response filtered it from
		// the render while the response still placed it inside the
		// loaded range (a response fresher than the event -- the
		// server had already restored the asset). Either way the
		// window rule's "unloaded page's territory" premise is false
		// for the id, so a later member convergence re-enters it.
		for (const delivered of page.items) {
			knownIds.add(delivered.id);
		}
		nextCursor = page.nextCursor;
		store.set({
			items: [...state.items, ...appended],
			status: 'ready',
			hasMore: nextCursor !== null,
			errorKey: ''
		});
	}

	function reset(): void {
		generation += 1;
		nextCursor = null;
		tombstones.clear();
		knownIds.clear();
		removedMembers.clear();
		// A reset invalidates every convergence claim: an in-flight
		// detail fetch lands to a cleared claim and is discarded (a
		// fresh event for the same asset starts a new, clean loop).
		convergeClaim.clear();
		convergeQueued.clear();
		store.set({ items: [], status: 'idle', hasMore: true, errorKey: '' });
	}

	async function applyEvent(event: AssetEvent): Promise<void> {
		if (event.name === 'asset.trashed' || event.name === 'asset.deleted') {
			// Unambiguous removals: no refetch needed. The removal's
			// id supersedes any convergence claim the asset holds, so
			// an in-flight live detail (whose upsert clears
			// tombstones) can never resurrect the tile.
			convergeClaim.set(event.assetId, Math.max(convergeClaim.get(event.assetId) ?? -1, event.id));
			convergeQueued.delete(event.assetId);
			removeIds([event.assetId]);
			return;
		}
		// ready / updated / restored / uploaded: converge on the
		// detail, sequenced by the event id.
		await convergeFromDetail(event.assetId, event.id);
	}

	async function runBatch(action: BatchAction, ids: string[]): Promise<BatchOutcome> {
		if (ids.length === 0) {
			return { ok: false, errorKey: 'internal' };
		}
		const snapshot = get(store).items;

		// The optimistic state first.
		switch (action) {
			case 'favorite':
			case 'unfavorite': {
				if (view === 'favorites' && action === 'unfavorite') {
					// Membership removal: an unfavorited asset leaves
					// the favorites view. No SSE event follows (favorites
					// are per-user state and silent by contract), so the
					// optimistic removal is the last word -- but NOT a
					// final state the client can forget: any later
					// non-removal event for the asset converges on a
					// detail that cannot evaluate favoriteness, so the
					// ids park in the sticky set (see removedMembers) to
					// keep the convergence from counting them back in. A
					// failed batch rolls the rows back and ends the
					// stickiness; a reset drops the whole wall.
					for (const id of ids) {
						removedMembers.add(id);
					}
					removeIds(ids);
					break;
				}
				const favorite = action === 'favorite';
				const selected = new Set(ids);
				store.update((state) => ({
					...state,
					items: state.items.map((timelineItem) =>
						selected.has(timelineItem.id) ? { ...timelineItem, isFavorite: favorite } : timelineItem
					)
				}));
				break;
			}
			case 'hide':
			case 'trash': {
				// Live views lose the tiles immediately; the SSE events
				// of the successful batch confirm the removal (no-op).
				removeIds(ids);
				break;
			}
			case 'unhide': {
				if (view === 'hidden') {
					// Membership removal: an unhidden asset returns to
					// the live views and leaves this one. The SSE
					// asset.updated event of the success converges the
					// same way (the detail's membership rule removes
					// the live row again -- a harmless no-op).
					removeIds(ids);
					break;
				}
				// The timeline wall: nothing visible is hidden, so the
				// local state is already correct, and the SSE
				// convergence inserts returning items into the loaded
				// window if they belong.
				break;
			}
			case 'restore': {
				// Nothing visible is trashed: the local state is
				// already correct, and the SSE convergence inserts
				// returning items into the loaded window if they
				// belong.
				break;
			}
		}

		let response: Response;
		try {
			response = await apiFetch(
				BATCH_ENDPOINT,
				{
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ action, ids })
				},
				fetcher
			);
		} catch {
			rollback(ids, snapshot);
			return { ok: false, errorKey: 'internal' };
		}
		if (!response.ok) {
			if (response.status === 401) {
				// Same rule as the page path: a dead session shows the
				// session message, not login-credential text.
				rollback(ids, snapshot);
				return { ok: false, errorKey: 'session_expired' };
			}
			const envelope = await errorEnvelope(response);
			rollback(ids, snapshot);
			return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
		}
		let body: unknown;
		try {
			body = await response.json();
		} catch {
			rollback(ids, snapshot);
			return { ok: false, errorKey: 'internal' };
		}
		const report = body as { affected?: unknown; missing?: unknown };
		if (typeof report.affected !== 'number' || !Array.isArray(report.missing)) {
			rollback(ids, snapshot);
			return { ok: false, errorKey: 'internal' };
		}
		return { ok: true, affected: report.affected, missing: report.missing as string[] };
	}

	return {
		subscribe: store.subscribe,
		loadMore,
		reset,
		applyEvent,
		runBatch
	};
}
