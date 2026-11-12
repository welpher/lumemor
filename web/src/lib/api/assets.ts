// The single-asset data client of the viewer route: the detail
// fetch behind GET /api/v1/assets/{id}, the owner-level edits behind
// PATCH /assets/{id}, the permanent DELETE /assets/{id}, the
// single-id batch actions (the same POST /assets/batch the timeline
// wall runs, mirrored here without wall state), the readiness probe
// of the play stream, and the serving-URL builders every media
// surface of the viewer feeds from.
//
// THE CACHING PIN: the media endpoints (thumbnail, original, play)
// serve private immutable copies with a strong checksum-prefix ETag,
// so the BROWSER's cache is the caching layer -- a stable URL is a
// cache hit after the first fetch, and the browser's If-None-Match
// revalidation answers 304 on the rare expiry. The URL builders
// therefore never cache-bust (no timestamps, no query salts): the
// same asset id + size always assembles the same URL. The JSON
// detail is mutable asset state (PATCH edits it, SSE announces it),
// so it is fetched fresh every time and never memoized here -- the
// PATCH response IS the fresh detail and the caller adopts it.
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased,
// 'internal' for a network failure or an unparseable body, and
// 'session_expired' for a 401 (the interceptor has already flipped
// the session store; the viewer surfaces the session message, never
// login-credential text).

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { BatchAction } from '$lib/api/timeline';
import type { components } from '../../../../openapi/generated/web/api';

/** The detail endpoint's body, exactly the contract's AssetDetail. */
export type AssetDetail = components['schemas']['AssetDetail'];

/** One face of the detail's faces array, the contract's AssetFaceSummary. */
export type AssetFaceSummary = components['schemas']['AssetFaceSummary'];

/** One entry of the detail's `albums` array, the contract's AssetAlbumRef. */
export type AssetAlbumRef = components['schemas']['AssetAlbumRef'];

/** The detail's `place` face, the contract's AssetPlaceRef. */
export type AssetPlaceRef = components['schemas']['AssetPlaceRef'];

/** The PATCH body: either owner-level attribute, at least one. */
export type AssetPatch = components['schemas']['AssetPatch'];

/** A detail-bearing call's options: the fetch implementation and the place face's display locale. */
export type AssetDetailOptions = {
	/** The fetch implementation (tests inject a mock; the app uses the global). */
	fetcher?: typeof fetch;
	/**
	 * The display locale of the response's `place` face (`en` or
	 * `zh-Hans`, the places face's own parameter -- the web app owns
	 * the active locale and passes it along; the server never
	 * guesses it from headers). Absent means `en`.
	 */
	locale?: string;
};

/** A detail-bearing call's outcome: the detail, or the failure's i18n key. */
export type AssetDetailResult = { ok: true; detail: AssetDetail } | { ok: false; errorKey: string };

/** A stateless mutation's outcome (DELETE), or a batch run's. */
export type AssetMutationResult = { ok: true } | { ok: false; errorKey: string };

/** The batch endpoint's success report (affected ids, unresolvable ids). */
export type AssetBatchResult = { ok: true; affected: number; missing: string[] } | { ok: false; errorKey: string };

/** The serving sizes of the thumbnail endpoint, the contract's enum. */
export type ThumbnailSize = 'tile' | 'view' | 'original';

/**
 * The thumbnail serving URL of one asset. `tile` fits the timeline
 * box, `view` caps the long edge at 1080px (the viewer's first
 * layer), `original` is the full-resolution re-encode (the viewer's
 * zoom layer). Video assets answer their poster frame whatever the
 * size says.
 */
export function assetThumbnailUrl(id: string, size: ThumbnailSize): string {
	return `/api/v1/assets/${encodeURIComponent(id)}/thumbnail?size=${size}`;
}

/**
 * The original-file download URL of one asset. The endpoint always
 * answers an attachment (Content-Disposition), so this URL belongs
 * on a download affordance, never on an <img> or <video> src.
 */
export function assetOriginalUrl(id: string): string {
	return `/api/v1/assets/${encodeURIComponent(id)}/original`;
}

/**
 * The transcoded play stream URL of one video-family asset: a
 * standalone video, or the component video row of a Live Photo
 * (the pair's motion plays through the component's own id).
 */
export function assetPlayUrl(id: string): string {
	return `/api/v1/assets/${encodeURIComponent(id)}/play`;
}

/** Structurally validate one face summary; null when it is not one. */
function faceSummaryFrom(body: unknown): AssetFaceSummary | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<AssetFaceSummary>;
	if (typeof candidate.id !== 'string' || typeof candidate.thumbnail !== 'string') {
		return null;
	}
	if (typeof candidate.score !== 'number' || !Number.isFinite(candidate.score)) {
		return null;
	}
	// clusterId is null before the clustering assigns one; absent or
	// non-string is not a summary.
	if (typeof candidate.clusterId !== 'string' && candidate.clusterId !== null) {
		return null;
	}
	// label is the caller's label state or null (always present); a
	// present label must carry its two fields.
	if (candidate.label !== null) {
		if (typeof candidate.label !== 'object') {
			return null;
		}
		const label = candidate.label as Partial<NonNullable<AssetFaceSummary['label']>>;
		if (typeof label.name !== 'string') {
			return null;
		}
		if (label.isHidden !== true && label.isHidden !== false) {
			return null;
		}
	}
	return body as AssetFaceSummary;
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
	// The faces array is part of the fixed shape: absent, null or a
	// malformed entry is not a detail.
	if (!Array.isArray(candidate.faces)) {
		return null;
	}
	for (const entry of candidate.faces) {
		if (faceSummaryFrom(entry) === null) {
			return null;
		}
	}
	// The caller's tags array joined the fixed shape: absent,
	// null or a malformed entry is not a detail.
	if (!Array.isArray(candidate.tags)) {
		return null;
	}
	for (const entry of candidate.tags) {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const tag = entry as { id?: unknown; name?: unknown };
		if (typeof tag.id !== 'string' || typeof tag.name !== 'string') {
			return null;
		}
	}
	// The albums array joined the fixed shape (the info panel's deep
	// links): every entry carries its identity, title, type and the
	// caller's relation.
	if (!Array.isArray(candidate.albums)) {
		return null;
	}
	for (const entry of candidate.albums) {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const album = entry as Partial<AssetAlbumRef>;
		if (typeof album.id !== 'string' || typeof album.title !== 'string') {
			return null;
		}
		if (album.type !== 'memory' && album.type !== 'directory' && album.type !== 'sync') {
			return null;
		}
		if (album.callerRole !== 'owner' && album.callerRole !== 'viewer') {
			return null;
		}
	}
	// The place face is required and nullable: a present value must
	// carry the deep link's key and the display name.
	if (candidate.place !== null) {
		if (typeof candidate.place !== 'object') {
			return null;
		}
		const place = candidate.place as Partial<AssetPlaceRef>;
		if (typeof place.placeId !== 'number' || !Number.isInteger(place.placeId)) {
			return null;
		}
		if (typeof place.name !== 'string') {
			return null;
		}
	}
	return body as AssetDetail;
}

/**
 * GET /api/v1/assets/{id}: the asset detail. A 404 (unknown or
 * foreign id -- the anti-enumeration pair is indistinguishable)
 * maps to asset_not_found; a trashed asset answers 200 and arrives
 * like any other. The `locale` option rides the place face's
 * display join (the web app owns the active locale).
 */
export async function fetchAssetDetail(id: string, options: AssetDetailOptions = {}): Promise<AssetDetailResult> {
	const fetcher = options.fetcher ?? globalThis.fetch;
	const query = options.locale === undefined ? '' : `?locale=${encodeURIComponent(options.locale)}`;
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/assets/${encodeURIComponent(id)}${query}`, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { ok: false, errorKey: 'session_expired' };
	}
	if (!response.ok) {
		const envelope = await errorEnvelope(response);
		return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const detail = detailFrom(body);
	if (detail === null || detail.id !== id) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, detail };
}

/**
 * PATCH /api/v1/assets/{id}: edit the description and/or the hidden
 * switch. The response is the full detail after the edit (the
 * idempotent no-op answers it too), so the caller adopts it as the
 * fresh state -- no refetch. The `locale` option rides the response's
 * place face (the same-shape discipline: the PATCH serves the detail
 * exactly as the GET serves it).
 */
export async function patchAsset(
	id: string,
	patch: AssetPatch,
	options: AssetDetailOptions = {}
): Promise<AssetDetailResult> {
	const fetcher = options.fetcher ?? globalThis.fetch;
	const query = options.locale === undefined ? '' : `?locale=${encodeURIComponent(options.locale)}`;
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/assets/${encodeURIComponent(id)}${query}`,
			{
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(patch)
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { ok: false, errorKey: 'session_expired' };
	}
	if (!response.ok) {
		const envelope = await errorEnvelope(response);
		return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const detail = detailFrom(body);
	if (detail === null || detail.id !== id) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, detail };
}

/**
 * DELETE /assets/{id}: the permanent delete, the trash bypass. The
 * outcome is a plain mutation (204, no body). A 404 means the asset
 * is already gone -- the same already-cleaned reading the upload
 * queue's save-as path pins -- so the caller treats it as success.
 */
export async function deleteAsset(id: string, fetcher: typeof fetch = globalThis.fetch): Promise<AssetMutationResult> {
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/assets/${encodeURIComponent(id)}`, { method: 'DELETE' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { ok: false, errorKey: 'session_expired' };
	}
	if (response.ok || response.status === 404) {
		return { ok: true };
	}
	const envelope = await errorEnvelope(response);
	return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
}

/**
 * POST /api/v1/assets/batch over explicit ids: the network half of
 * the timeline wall's runner, mirrored without wall state. Set
 * semantics: every resolvable id applies (the idempotent no-op
 * counts as affected), unresolvable ids land in `missing`, and no
 * individual id fails the request. The viewer uses it for the
 * single current asset; the optimistic state and rollback are the
 * caller's (the page owns the local detail copy).
 */
export async function runAssetBatch(
	action: BatchAction,
	ids: string[],
	fetcher: typeof fetch = globalThis.fetch
): Promise<AssetBatchResult> {
	if (ids.length === 0) {
		return { ok: false, errorKey: 'internal' };
	}
	let response: Response;
	try {
		response = await apiFetch(
			'/api/v1/assets/batch',
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ action, ids })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { ok: false, errorKey: 'session_expired' };
	}
	if (!response.ok) {
		const envelope = await errorEnvelope(response);
		return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const report = body as { affected?: unknown; missing?: unknown };
	if (typeof report.affected !== 'number' || !Array.isArray(report.missing)) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, affected: report.affected, missing: report.missing as string[] };
}

/**
 * Probe whether one video-family id has a playable stream yet: a
 * one-byte range GET against the play endpoint (the documented
 * ServeContent range semantics make it cheap -- 206 plus a single
 * byte when the stream exists). A transcode still in flight answers
 * 404 ASSET_NOT_READY; the caller keeps the poster or the still
 * photo and waits. The probe exists because readiness is not always
 * in the detail: a Live Photo component never broadcasts its own
 * event, and its pair's photo row counts no transcode of its own.
 */
export async function probeAssetPlay(
	id: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AssetMutationResult> {
	let response: Response;
	try {
		response = await apiFetch(assetPlayUrl(id), { headers: { Range: 'bytes=0-0' } }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { ok: false, errorKey: 'session_expired' };
	}
	if (response.ok) {
		// 200 (full answer) or 206 (the one-byte range): the stream
		// exists.
		return { ok: true };
	}
	const envelope = await errorEnvelope(response);
	return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
}
