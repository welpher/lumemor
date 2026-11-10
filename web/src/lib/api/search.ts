// The search surface's data client: the URL state
// mapping, the request builder, the page read and the store-backed
// client the search page drives.
//
// THE TWO RESULT MODES (the pinned rule): a query whose trimmed text
// spans two or more runes is QUERY mode -- the server merges the text
// and semantic arms into one rank order, and the page renders the
// FLAT justified wall; anything less takes NO text path (the
// contract's own rule), the request is the pure filter browse, and
// the page renders the timeline's month-grouped wall. The mode
// follows the query's presence exactly the way the SERVER decides
// it, so the cursor's mode tag always matches the page's shape.
//
// THE URL IS THE STATE: the search page owns the query and the
// filter selections through the /search URL's parameters
// (deep-linkable); `filtersFromSearchParams` /
// `searchParamsFromFilters` are the two directions of that mapping,
// and `searchRequestOf` derives the REQUEST the contract wants (a
// sub-two-rune query drops out, the selected year set maps onto the
// from/to range -- min year start, max year end exclusive -- and the
// id lists ride the contract's CSV form parameters).
//
// THE SSE CONVERGENCE (the pinned semantics): the search page
// re-executes the current query on every asset.* event -- the
// refetch IS the convergence, no patching. `applyEvent` re-runs page
// one with keep semantics (the loaded tiles stay until the fresh
// answer lands; a failure keeps the last good state silently),
// collapsed single-flight: a burst of events costs at most one
// trailing re-execution.

import { get, writable } from 'svelte/store';
import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { AssetEvent } from '$lib/events';
import type { TimelineItem } from './timeline';
import type { components } from '../../../../openapi/generated/web/api';

/** One result tile, the timeline's exact item shape (the contract reuses it). */
export type SearchItem = TimelineItem;

/** The facet block of one search response, exactly the contract's shape. */
export type SearchFacets = components['schemas']['SearchFacets'];

/** The media families of the kinds filter and facet. */
export type SearchMediaKind = components['schemas']['MediaKind'];

/**
 * The search page's filter state -- the seven dimensions' selected
 * values plus the free-text query. This is the URL-facing shape; the
 * request derivation (`searchRequestOf`) is a pure function of it.
 */
export type SearchFilters = {
	/** The free-text query, as typed (the request trims it). */
	query: string;
	/** Selected people facet ids (face cluster ids). */
	personIds: string[];
	/** Selected places facet ids (GeoNames ids). */
	placeIds: number[];
	/** Selected years (the time-range facet's chips). */
	years: number[];
	/** Selected camera facet ids (`make|model`). */
	cameras: string[];
	/** Selected media families. */
	kinds: SearchMediaKind[];
	/** Selected album facet ids. */
	albumIds: string[];
	/** Selected tag facet ids (the caller's own). */
	tagIds: string[];
};

/** The request-derivation options of one page fetch. */
export type SearchPageOptions = {
	/** The continuation cursor of the previous page; null on page one. */
	cursor?: string | null;
	/** The places facet's display-name locale; the default is `en`. */
	locale?: string;
	/** Page size; the contract's default is 100. */
	limit?: number;
	/** The fetch implementation (tests inject a mock). */
	fetcher?: typeof fetch;
};

/** One search page: the items, the continuation and the facets. */
export type SearchPage = {
	items: SearchItem[];
	nextCursor: string | null;
	facets: SearchFacets;
};

/** The page fetch's outcome, the repo-wide error-key convention. */
export type SearchPageResult = { ok: true; page: SearchPage } | { ok: false; errorKey: string };

/** The client store's lifecycle. */
export type SearchStatus = 'idle' | 'loading' | 'ready' | 'error';

export type SearchState = {
	/** The loaded result tiles, in the page's order (rank or time). */
	items: SearchItem[];
	/** The facets of the last landed page; null before the first. */
	facets: SearchFacets | null;
	/** The continuation cursor; null exactly when no page follows. */
	nextCursor: string | null;
	status: SearchStatus;
	/** The error-code i18n key of the last failed page load ('' when none). */
	errorKey: string;
};

export type SearchClient = {
	/** The store half; the page binds it with `$search`. */
	subscribe: (run: (state: SearchState) => void) => () => void;
	/**
	 * Run (or re-run) the search from page one: the authoritative
	 * restart. The page effect calls it whenever the URL (or the
	 * locale) changes; a run while a fetch is in flight supersedes it
	 * (the stale response never lands).
	 */
	run: (filters: SearchFilters, locale: string) => Promise<void>;
	/** Fetch the next page through the continuation cursor (guarded). */
	loadMore: () => Promise<void>;
	/** React to one SSE activity event: re-execute the current search. */
	applyEvent: (event: AssetEvent) => void;
	/** Drop the state and park idle (tests and future route resets). */
	reset: () => void;
};

/** The contract's default page size. */
export const SEARCH_PAGE_LIMIT = 100;

/** The locale the request carries when the caller passes none. */
const DEFAULT_LOCALE = 'en';

/** The media families the kinds filter accepts (the contract's set). */
const MEDIA_KINDS: readonly SearchMediaKind[] = ['image', 'video'];

/**
 * The mode rule, the server's own: a query whose trimmed text spans
 * two or more runes takes the text path (QUERY mode); anything less
 * is the pure filter browse (BROWSE mode). The client mirrors it so
 * the request it builds and the wall it renders always agree with
 * the cursor's mode tag.
 */
export function isQueryMode(query: string): boolean {
	const runes = [...query.trim()].length;
	return runes >= 2;
}

/** Split one CSV parameter's non-empty segments. */
function csvOf(value: string | null): string[] {
	if (value === null) {
		return [];
	}
	return value
		.split(',')
		.map((segment) => segment.trim())
		.filter((segment) => segment !== '');
}

/** Parse one CSV parameter into integers, dropping the unparsable. */
function intCsvOf(value: string | null): number[] {
	return csvOf(value)
		.map((segment) => Number.parseInt(segment, 10))
		.filter((parsed) => Number.isInteger(parsed) && parsed > 0);
}

/**
 * Read the search state off the /search URL's parameters. Malformed
 * entries (a non-numeric place id, a kind outside the media
 * families, empty segments) drop out -- a hand-written URL never
 * poisons the request with values the contract would reject.
 */
export function filtersFromSearchParams(params: URLSearchParams): SearchFilters {
	const kinds = csvOf(params.get('kinds')).filter((kind): kind is SearchMediaKind =>
		(MEDIA_KINDS as readonly string[]).includes(kind)
	);
	return {
		query: params.get('query') ?? '',
		personIds: csvOf(params.get('personIds')),
		placeIds: intCsvOf(params.get('placeIds')),
		years: intCsvOf(params.get('years')),
		cameras: csvOf(params.get('cameras')),
		kinds,
		albumIds: csvOf(params.get('albumIds')),
		tagIds: csvOf(params.get('tagIds'))
	};
}

/**
 * Write the canonical deep-link URL: only the non-empty dimensions
 * carry parameters, in the filter order. A one-rune query stays in
 * the URL (the box shows what was typed; the REQUEST drops it).
 */
export function searchParamsFromFilters(filters: SearchFilters): URLSearchParams {
	const params = new URLSearchParams();
	const trimmed = filters.query.trim();
	if (trimmed !== '') {
		params.set('query', trimmed);
	}
	if (filters.personIds.length > 0) {
		params.set('personIds', filters.personIds.join(','));
	}
	if (filters.placeIds.length > 0) {
		params.set('placeIds', filters.placeIds.join(','));
	}
	if (filters.years.length > 0) {
		params.set('years', filters.years.join(','));
	}
	if (filters.cameras.length > 0) {
		params.set('cameras', filters.cameras.join(','));
	}
	if (filters.kinds.length > 0) {
		params.set('kinds', filters.kinds.join(','));
	}
	if (filters.albumIds.length > 0) {
		params.set('albumIds', filters.albumIds.join(','));
	}
	if (filters.tagIds.length > 0) {
		params.set('tagIds', filters.tagIds.join(','));
	}
	return params;
}

/**
 * The selected year set's from/to range: the min selected year's
 * start (inclusive) through the max selected year's end (exclusive).
 * A discontiguous selection widens the range across the gap -- the
 * contract's range is the only time filter, and the common
 * contiguous case is exact.
 */
function yearRangeOf(years: readonly number[]): { from: string; to: string } | null {
	if (years.length === 0) {
		return null;
	}
	const min = Math.min(...years);
	const max = Math.max(...years);
	return {
		from: `${min}-01-01T00:00:00Z`,
		to: `${max + 1}-01-01T00:00:00Z`
	};
}

/**
 * Build the request's query string (without the leading `?`). The
 * parameter order is fixed (query, the filter dimensions in facet
 * order, from/to, locale, limit, cursor) so tests and logs read one
 * canonical string. A sub-two-rune query takes NO text path -- the
 * request is the pure filter browse, exactly the server's rule.
 */
export function searchRequestOf(filters: SearchFilters, locale: string, limit: number, cursor: string | null): string {
	const params = new URLSearchParams();
	if (isQueryMode(filters.query)) {
		params.set('query', filters.query.trim());
	}
	if (filters.personIds.length > 0) {
		params.set('personIds', filters.personIds.join(','));
	}
	if (filters.placeIds.length > 0) {
		params.set('placeIds', filters.placeIds.join(','));
	}
	const range = yearRangeOf(filters.years);
	if (range !== null) {
		params.set('from', range.from);
		params.set('to', range.to);
	}
	if (filters.cameras.length > 0) {
		params.set('cameras', filters.cameras.join(','));
	}
	if (filters.kinds.length > 0) {
		params.set('kinds', filters.kinds.join(','));
	}
	if (filters.albumIds.length > 0) {
		params.set('albumIds', filters.albumIds.join(','));
	}
	if (filters.tagIds.length > 0) {
		params.set('tagIds', filters.tagIds.join(','));
	}
	params.set('locale', locale);
	params.set('limit', String(limit));
	if (cursor !== null) {
		params.set('cursor', cursor);
	}
	return params.toString();
}

/** The error-code i18n key of one failed response (the repo convention). */
async function errorKeyFrom(response: Response): Promise<string> {
	if (response.status === 401) {
		// A dead session shows the session message, never
		// login-credential text (the interceptor already flipped the
		// store; the guard is redirecting).
		return 'session_expired';
	}
	const envelope = await errorEnvelope(response);
	return envelope !== null ? envelope.code.toLowerCase() : 'internal';
}

/** Read one response's JSON body; null when it does not parse. */
async function bodyOf(response: Response): Promise<unknown> {
	try {
		return await response.json();
	} catch {
		return null;
	}
}

/** One timeline-shaped item, structurally validated; null when not one. */
function itemFrom(body: unknown): SearchItem | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<SearchItem>;
	if (typeof candidate.id !== 'string' || typeof candidate.takenAt !== 'string') {
		return null;
	}
	return body as SearchItem;
}

/** One facet entry's assetCount, validated; null when not a count. */
function countOf(value: unknown): number | null {
	if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) {
		return null;
	}
	return value;
}

/** Validate one facet dimension's entries; null when any entry is malformed. */
function facetOf<T>(body: unknown, key: keyof SearchFacets, validate: (entry: unknown) => T | null): T[] | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const entries = (body as Record<string, unknown>)[key];
	if (!Array.isArray(entries)) {
		return null;
	}
	const valid: T[] = [];
	for (const entry of entries) {
		const checked = validate(entry);
		if (checked === null) {
			return null;
		}
		valid.push(checked);
	}
	return valid;
}

/** The facets block, structurally validated; null when malformed. */
function facetsFrom(body: unknown): SearchFacets | null {
	const people = facetOf(body, 'people', (entry) => {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const candidate = entry as { id?: unknown; name?: unknown; assetCount?: unknown };
		if (typeof candidate.id !== 'string' || typeof candidate.name !== 'string') {
			return null;
		}
		return countOf(candidate.assetCount) === null ? null : entry;
	});
	const places = facetOf(body, 'places', (entry) => {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const candidate = entry as { placeId?: unknown; name?: unknown; country?: unknown; assetCount?: unknown };
		if (
			typeof candidate.placeId !== 'number' ||
			!Number.isInteger(candidate.placeId) ||
			typeof candidate.name !== 'string' ||
			typeof candidate.country !== 'string'
		) {
			return null;
		}
		return countOf(candidate.assetCount) === null ? null : entry;
	});
	const years = facetOf(body, 'years', (entry) => {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const candidate = entry as { year?: unknown; assetCount?: unknown };
		if (typeof candidate.year !== 'number' || !Number.isInteger(candidate.year)) {
			return null;
		}
		return countOf(candidate.assetCount) === null ? null : entry;
	});
	const cameras = facetOf(body, 'cameras', (entry) => {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const candidate = entry as {
			id?: unknown;
			make?: unknown;
			model?: unknown;
			assetCount?: unknown;
		};
		if (typeof candidate.id !== 'string') {
			return null;
		}
		if (candidate.make !== null && candidate.make !== undefined && typeof candidate.make !== 'string') {
			return null;
		}
		if (candidate.model !== null && candidate.model !== undefined && typeof candidate.model !== 'string') {
			return null;
		}
		return countOf(candidate.assetCount) === null ? null : entry;
	});
	const kinds = facetOf(body, 'kinds', (entry) => {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const candidate = entry as { kind?: unknown; assetCount?: unknown };
		if (typeof candidate.kind !== 'string' || !(MEDIA_KINDS as readonly string[]).includes(candidate.kind)) {
			return null;
		}
		return countOf(candidate.assetCount) === null ? null : entry;
	});
	const albums = facetOf(body, 'albums', (entry) => {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const candidate = entry as { id?: unknown; title?: unknown; assetCount?: unknown };
		if (typeof candidate.id !== 'string' || typeof candidate.title !== 'string') {
			return null;
		}
		return countOf(candidate.assetCount) === null ? null : entry;
	});
	const tags = facetOf(body, 'tags', (entry) => {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const candidate = entry as { id?: unknown; name?: unknown; assetCount?: unknown };
		if (typeof candidate.id !== 'string' || typeof candidate.name !== 'string') {
			return null;
		}
		return countOf(candidate.assetCount) === null ? null : entry;
	});
	if (
		people === null ||
		places === null ||
		years === null ||
		cameras === null ||
		kinds === null ||
		albums === null ||
		tags === null
	) {
		return null;
	}
	return { people, places, years, cameras, kinds, albums, tags } as SearchFacets;
}

/** One whole page body, structurally validated; null when not a page. */
function pageFrom(body: unknown): SearchPage | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const {
		items,
		nextCursor: cursor,
		facets
	} = body as {
		items?: unknown;
		nextCursor?: unknown;
		facets?: unknown;
	};
	if (!Array.isArray(items)) {
		return null;
	}
	for (const entry of items) {
		if (itemFrom(entry) === null) {
			return null;
		}
	}
	if (cursor !== undefined && typeof cursor !== 'string') {
		return null;
	}
	const checkedFacets = facetsFrom(facets);
	if (checkedFacets === null) {
		return null;
	}
	return {
		items: items as SearchItem[],
		nextCursor: cursor ?? null,
		facets: checkedFacets
	};
}

/**
 * GET /api/v1/search: one page of the merged search (or the filter
 * browse) with the facet counts. The cursor is the previous page's
 * opaque `nextCursor`, never assembled here.
 */
export async function searchPage(filters: SearchFilters, options: SearchPageOptions = {}): Promise<SearchPageResult> {
	const fetcher = options.fetcher ?? globalThis.fetch;
	const locale = options.locale ?? DEFAULT_LOCALE;
	const limit = options.limit ?? SEARCH_PAGE_LIMIT;
	const request = searchRequestOf(filters, locale, limit, options.cursor ?? null);
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/search?${request}`, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const page = pageFrom(await bodyOf(response));
	if (page === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, page };
}

/**
 * Build the search page's client. Two instances never share state,
 * so tests construct isolated clients and the page owns exactly one
 * per mount.
 */
export function createSearchClient(options: { fetcher?: typeof fetch; limit?: number } = {}): SearchClient {
	const fetcher = options.fetcher ?? globalThis.fetch;
	const limit = options.limit ?? SEARCH_PAGE_LIMIT;

	const store = writable<SearchState>({
		items: [],
		facets: null,
		nextCursor: null,
		status: 'idle',
		errorKey: ''
	});

	// The walk's parameters (the page effect's run sets them; the
	// SSE convergence re-uses them) and its generation: a new run
	// raises it, and a page fetch whose entry generation no longer
	// matches discards its response -- a URL change (or an event's
	// re-execution) while a page load is in flight must never let
	// the stale answer land over the fresh walk.
	let filters: SearchFilters | null = null;
	let locale = DEFAULT_LOCALE;
	let generation = 0;
	// The serialized fetch queue: one page fetch at a time, later
	// triggers chain after the running one (its generation guard
	// already discards whatever it would have written). A burst of
	// SSE events collapses into `convergeQueued`: one re-execution
	// is scheduled, the rest of the burst rides it.
	let convergeQueued = false;
	let chain: Promise<void> = Promise.resolve();

	/** Append one fetch action to the serialized queue. */
	function enqueue(action: () => Promise<void>): Promise<void> {
		const next = chain.then(action);
		// The chain never rejects (a rejected action is a bug, not a
		// state: fetchPage maps every failure into the store).
		chain = next.catch(() => undefined);
		return next;
	}

	/**
	 * One page fetch's shared body. `mode` decides how the state
	 * moves: 'restart' (the page effect's run: clear, load, errors
	 * park the box), 'append' (the walk's next page: keep the tiles,
	 * a failure keeps them with the retry affordance) and 'converge'
	 * (the SSE re-execution: keep the tiles until the fresh answer
	 * lands; a failure keeps the last good state silently).
	 */
	async function fetchPage(cursor: string | null, mode: 'restart' | 'append' | 'converge'): Promise<void> {
		const current = filters;
		if (current === null) {
			// reset() dropped the walk while this action waited in
			// the queue: nothing to fetch.
			return;
		}
		const run = generation;
		let pageCursor = cursor;
		if (mode === 'append') {
			// Re-read the cursor at fetch time: a convergence that
			// landed while this action waited in the queue may have
			// replaced the walk (the guard at call time saw the old
			// one).
			pageCursor = get(store).nextCursor;
			if (pageCursor === null) {
				return;
			}
		}
		if (mode === 'restart') {
			store.set({ items: [], facets: null, nextCursor: null, status: 'loading', errorKey: '' });
		} else if (mode === 'append') {
			store.update((state) => ({ ...state, status: 'loading', errorKey: '' }));
		}
		const outcome = await searchPage(current, { cursor: pageCursor, locale, limit, fetcher });
		if (run !== generation) {
			// Superseded mid-flight (a URL change, or an event's
			// re-execution): the fresher walk's own fetch decides.
			return;
		}
		if (!outcome.ok) {
			if (mode === 'converge' && get(store).items.length > 0) {
				// The convergence failure keeps the last good state;
				// the next event retries.
				return;
			}
			store.update((state) => ({ ...state, status: 'error', errorKey: outcome.errorKey }));
			return;
		}
		const page = outcome.page;
		if (mode === 'append') {
			// Belt and braces around the keyset's own disjointness:
			// pages append only ids not already loaded.
			const loaded = new Set(get(store).items.map((entry) => entry.id));
			store.update((state) => ({
				items: [...state.items, ...page.items.filter((entry) => !loaded.has(entry.id))],
				facets: page.facets,
				nextCursor: page.nextCursor,
				status: 'ready',
				errorKey: ''
			}));
			return;
		}
		store.set({
			items: page.items,
			facets: page.facets,
			nextCursor: page.nextCursor,
			status: 'ready',
			errorKey: ''
		});
	}

	async function run(next: SearchFilters, nextLocale: string): Promise<void> {
		filters = next;
		locale = nextLocale;
		// A run while a fetch is in flight supersedes it: the raised
		// generation discards the stale response when it lands, and
		// this action chains after it in the queue. A pending event
		// convergence is subsumed too -- the restart below IS the
		// re-execution with the freshest filters.
		generation += 1;
		convergeQueued = false;
		await enqueue(() => fetchPage(null, 'restart'));
	}

	async function loadMore(): Promise<void> {
		const before = get(store);
		if (before.status === 'loading' || before.status === 'idle') {
			// The in-flight guard: one page fetch per trigger.
			return;
		}
		if (before.nextCursor === null) {
			// Exhausted, or a failed first run whose retry is a
			// fresh run (the page's error box routes there).
			return;
		}
		await enqueue(() => fetchPage(null, 'append'));
	}

	function applyEvent(event: AssetEvent): void {
		// The pinned convergence: re-execute the current search on
		// every asset.* event (the refetch IS the convergence -- no
		// patching, the fresh page owns the truth).
		void event;
		if (filters === null) {
			// No search yet: nothing to converge.
			return;
		}
		// Supersede any in-flight fetch (its server snapshot
		// predates the event) and schedule the re-execution; a
		// burst collapses into the one scheduled action.
		generation += 1;
		if (convergeQueued) {
			return;
		}
		convergeQueued = true;
		void enqueue(async () => {
			convergeQueued = false;
			await fetchPage(null, 'converge');
		});
	}

	function reset(): void {
		generation += 1;
		filters = null;
		convergeQueued = false;
		store.set({
			items: [],
			facets: null,
			nextCursor: null,
			status: 'idle',
			errorKey: ''
		});
	}

	return {
		subscribe: store.subscribe,
		run,
		loadMore,
		applyEvent,
		reset
	};
}
