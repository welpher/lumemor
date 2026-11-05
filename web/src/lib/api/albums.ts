// The albums surface's data client (M3c Task 5): the wall list, the
// detail read, the virtual creation, the property patch, the virtual
// delete, the membership add, the share grant/revoke pair, the
// instance's member list and the timeline preference write -- each
// behind the repo-wide error-code convention: every failure outcome
// carries the error code's i18n key (the envelope code lowercased,
// 'internal' for a network failure or an unparseable body,
// 'session_expired' for a 401).
//
// THE MESSAGE-KEY EXCEPTIONS (the people client's requires-label
// pattern): the album write faces answer a handful of 400
// VALIDATION_FAILED whose `details` carry a message key rather than
// an error code of their own -- the cover validation
// (`album_cover_not_member`), the share target rules
// (`album_share_to_owner`, `album_share_user_invalid`) and the
// preference's not-excludable kinds (`album_pref_not_excludable`).
// The error-key helper surfaces those keys as the failure's
// errorKey, so the honest message renders through the same
// resolveErrorText path every code takes.
//
// The wall's two groups arrive concatenated (own first, shared
// after); the page sections them off each item's `callerRole`. The
// media URLs (every `thumbnail` field) arrive as strings from the
// server; no URL is assembled here.

import { apiFetch } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One wall card, exactly the contract's AlbumSummary shape. */
export type AlbumSummary = components['schemas']['AlbumSummary'];

/** One album's detail as the caller sees it. */
export type AlbumDetail = components['schemas']['AlbumDetail'];

/** One grantee row of the owner's detail `shares` list. */
export type AlbumShareEntry = components['schemas']['AlbumShareEntry'];

/** One account of the instance's minimal member list. */
export type UserSummary = components['schemas']['UserSummary'];

/** The share grant's response row. */
export type AlbumShareResponse = components['schemas']['AlbumShareResponse'];

/** The timeline preference write's response. */
export type AlbumTimelinePreferenceResponse = components['schemas']['AlbumTimelinePreferenceResponse'];

/** The wall list's outcome. */
export type AlbumListResult =
	| { ok: true; albums: AlbumSummary[] }
	| { ok: false; errorKey: string };

/** The detail read's outcome. */
export type AlbumDetailResult = { ok: true; detail: AlbumDetail } | { ok: false; errorKey: string };

/** The creation's outcome (the response IS the created wall card). */
export type AlbumCreateResult = { ok: true; album: AlbumSummary } | { ok: false; errorKey: string };

/** The property patch's outcome (the response is the card after the edit). */
export type AlbumPatchResult = { ok: true; album: AlbumSummary } | { ok: false; errorKey: string };

/** A body-less mutation's outcome (the delete, the revoke). */
export type AlbumMutationResult = { ok: true } | { ok: false; errorKey: string };

/** The membership add's honest report. */
export type AlbumItemsAddResult =
	| { ok: true; added: number; skipped: number }
	| { ok: false; errorKey: string };

/** The share grant's outcome. */
export type AlbumShareGrantResult =
	| { ok: true; share: AlbumShareResponse }
	| { ok: false; errorKey: string };

/** The member list's outcome. */
export type UserListResult = { ok: true; users: UserSummary[] } | { ok: false; errorKey: string };

/** The timeline preference write's outcome. */
export type AlbumPreferenceResult =
	| { ok: true; preference: AlbumTimelinePreferenceResponse }
	| { ok: false; errorKey: string };

/**
 * The lean page size of the metadata read: the detail page's grid
 * pages itself through the timeline endpoint's album narrow, so the
 * detail's own items page is waste -- one member is the least the
 * contract accepts (a limit below 1 answers 400).
 */
export const ALBUM_DETAIL_LIMIT = 1;

/** The album write faces' message keys, surfaced as error keys. */
const MESSAGE_KEYS: ReadonlySet<string> = new Set([
	'album_cover_not_member',
	'album_share_to_owner',
	'album_share_user_invalid',
	'album_pref_not_excludable'
]);

/**
 * Map one failed response onto the error-code i18n key, carrying the
 * write faces' message keys when the envelope's `details` hold one
 * (the VALIDATION_FAILED answers whose honest rendering is a message,
 * not the generic validation text).
 */
async function errorKeyFrom(response: Response): Promise<string> {
	if (response.status === 401) {
		// A dead session shows the session message, never
		// login-credential text (the interceptor already flipped the
		// store; the guard is redirecting).
		return 'session_expired';
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return 'internal';
	}
	if (body === null || typeof body !== 'object') {
		return 'internal';
	}
	const { code, details } = body as { code?: unknown; details?: unknown };
	if (typeof code !== 'string' || code === '') {
		return 'internal';
	}
	if (code === 'VALIDATION_FAILED' && details !== null && typeof details === 'object') {
		const key = (details as { key?: unknown }).key;
		if (typeof key === 'string' && MESSAGE_KEYS.has(key)) {
			return key;
		}
	}
	return code.toLowerCase();
}

/** Read one response's JSON body; null when it does not parse. */
async function bodyOf(response: Response): Promise<unknown> {
	try {
		return await response.json();
	} catch {
		return null;
	}
}

/** Whether one candidate is a plain object. */
function isObject(body: unknown): body is Record<string, unknown> {
	return body !== null && typeof body === 'object';
}

/** Structurally validate one wall card; null when it is not one. */
function summaryFrom(body: unknown): AlbumSummary | null {
	if (!isObject(body)) {
		return null;
	}
	const candidate = body as Partial<AlbumSummary>;
	if (typeof candidate.id !== 'string' || typeof candidate.title !== 'string') {
		return null;
	}
	if (typeof candidate.description !== 'string') {
		return null;
	}
	if (candidate.type !== 'virtual' && candidate.type !== 'directory') {
		return null;
	}
	if (candidate.locked !== true && candidate.locked !== false) {
		return null;
	}
	if (candidate.isShared !== true && candidate.isShared !== false) {
		return null;
	}
	if (!Array.isArray(candidate.coverAssetIds)) {
		return null;
	}
	if (typeof candidate.itemCount !== 'number' || !Number.isInteger(candidate.itemCount) || candidate.itemCount < 0) {
		return null;
	}
	if (typeof candidate.updatedAt !== 'string') {
		return null;
	}
	if (candidate.callerRole !== 'owner' && candidate.callerRole !== 'viewer') {
		return null;
	}
	if (candidate.inTimeline !== undefined && candidate.inTimeline !== true && candidate.inTimeline !== false) {
		return null;
	}
	if (candidate.owner !== undefined) {
		const owner = candidate.owner as Partial<AlbumSummary['owner']> | undefined;
		if (owner === undefined || typeof owner.id !== 'string' || typeof owner.email !== 'string') {
			return null;
		}
	}
	return body as AlbumSummary;
}

/** Structurally validate one share entry; null when it is not one. */
function shareEntryFrom(body: unknown): AlbumShareEntry | null {
	if (!isObject(body)) {
		return null;
	}
	const candidate = body as Partial<AlbumShareEntry>;
	if (typeof candidate.userId !== 'string' || typeof candidate.email !== 'string') {
		return null;
	}
	if (typeof candidate.grantedAt !== 'string') {
		return null;
	}
	return body as AlbumShareEntry;
}

/** Structurally validate one album detail; null when it is not one. */
function detailFrom(body: unknown, albumId: string): AlbumDetail | null {
	if (!isObject(body)) {
		return null;
	}
	const candidate = body as Partial<AlbumDetail>;
	if (candidate.id !== albumId || typeof candidate.title !== 'string') {
		return null;
	}
	if (typeof candidate.description !== 'string') {
		return null;
	}
	if (candidate.type !== 'virtual' && candidate.type !== 'directory') {
		return null;
	}
	if (candidate.locked !== true && candidate.locked !== false) {
		return null;
	}
	if (candidate.coverAssetId !== null && typeof candidate.coverAssetId !== 'string') {
		return null;
	}
	if (candidate.callerRole !== 'owner' && candidate.callerRole !== 'viewer') {
		return null;
	}
	if (typeof candidate.itemCount !== 'number' || !Number.isInteger(candidate.itemCount) || candidate.itemCount < 0) {
		return null;
	}
	if (!Array.isArray(candidate.items)) {
		return null;
	}
	if (candidate.nextCursor !== undefined && typeof candidate.nextCursor !== 'string') {
		return null;
	}
	if (candidate.shares !== undefined) {
		if (!Array.isArray(candidate.shares)) {
			return null;
		}
		for (const entry of candidate.shares) {
			if (shareEntryFrom(entry) === null) {
				return null;
			}
		}
	}
	if (candidate.inTimeline !== undefined && candidate.inTimeline !== true && candidate.inTimeline !== false) {
		return null;
	}
	return body as AlbumDetail;
}

/** Structurally validate one member-list entry; null when it is not one. */
function userSummaryFrom(body: unknown): UserSummary | null {
	if (!isObject(body)) {
		return null;
	}
	const candidate = body as Partial<UserSummary>;
	if (typeof candidate.id !== 'string' || typeof candidate.email !== 'string') {
		return null;
	}
	if (candidate.role !== 'admin' && candidate.role !== 'user') {
		return null;
	}
	if (candidate.isDisabled !== true && candidate.isDisabled !== false) {
		return null;
	}
	return body as UserSummary;
}

/**
 * GET /api/v1/albums: the caller's wall, the own group followed by
 * the shared group in one array. The page sections them off each
 * item's `callerRole` (the server's ordering owns each group).
 */
export async function listAlbums(
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumListResult> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/albums', undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!isObject(body) || !Array.isArray(body.albums)) {
		return { ok: false, errorKey: 'internal' };
	}
	const albums: AlbumSummary[] = [];
	for (const entry of body.albums) {
		const album = summaryFrom(entry);
		if (album === null) {
			return { ok: false, errorKey: 'internal' };
		}
		albums.push(album);
	}
	return { ok: true, albums };
}

/**
 * GET /api/v1/albums/{albumId}: the album's metadata as the caller
 * sees it -- `callerRole` decides the write affordances, `shares`
 * rides the owner's view alone, `inTimeline` the excludable kinds.
 * A 404 is the anti-enumeration pair (unknown id, another account's
 * unshared album, every sync album).
 */
export async function fetchAlbumDetail(
	albumId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumDetailResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}?limit=${ALBUM_DETAIL_LIMIT}`,
			undefined,
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const detail = detailFrom(await bodyOf(response), albumId);
	if (detail === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, detail };
}

/**
 * POST /api/v1/albums: create one virtual album (the only creatable
 * type). The response is the created album's wall card -- the
 * creation-time zeros: no member, no share, no cover.
 */
export async function createAlbum(
	title: string,
	description = '',
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumCreateResult> {
	let response: Response;
	try {
		response = await apiFetch(
			'/api/v1/albums',
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ title, description })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const album = summaryFrom(await bodyOf(response));
	if (album === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, album };
}

/**
 * PATCH /api/v1/albums/{albumId}: the owner's partial edit of the
 * properties (`title`, `description`, `coverAssetId`, `locked`, any
 * subset). The response is the album's wall card after the edit.
 */
export async function patchAlbum(
	albumId: string,
	patch: { title?: string; description?: string; coverAssetId?: string; locked?: boolean },
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumPatchResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}`,
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
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const album = summaryFrom(await bodyOf(response));
	if (album === null || album.id !== albumId) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, album };
}

/**
 * DELETE /api/v1/albums/{albumId}: delete the caller's virtual
 * album. The row goes with its membership and share rows (the
 * schema's cascades); 204, no body.
 */
export async function deleteAlbum(
	albumId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumMutationResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}`,
			{ method: 'DELETE' },
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.ok) {
		return { ok: true };
	}
	return { ok: false, errorKey: await errorKeyFrom(response) };
}

/**
 * POST /api/v1/albums/{albumId}/items: add assets to the caller's
 * virtual album with the batch surface's set semantics -- one honest
 * report `{added, skipped}` over the request's distinct ids, never
 * an error for an id the caller cannot act on.
 */
export async function addAlbumItems(
	albumId: string,
	assetIds: string[],
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumItemsAddResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}/items`,
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ assetIds })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!isObject(body)) {
		return { ok: false, errorKey: 'internal' };
	}
	const { added, skipped } = body as { added?: unknown; skipped?: unknown };
	if (typeof added !== 'number' || typeof skipped !== 'number') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, added, skipped };
}

/**
 * POST /api/v1/albums/{albumId}/shares: grant an account the album's
 * view ACL. The response is the share row -- the fresh row for a new
 * grantee, the existing one (its original grant time) for the
 * idempotent re-grant.
 */
export async function grantAlbumShare(
	albumId: string,
	userId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumShareGrantResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}/shares`,
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ userId })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!isObject(body)) {
		return { ok: false, errorKey: 'internal' };
	}
	const candidate = body as Partial<AlbumShareResponse>;
	if (
		typeof candidate.albumId !== 'string' ||
		typeof candidate.userId !== 'string' ||
		typeof candidate.grantedAt !== 'string'
	) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, share: body as AlbumShareResponse };
}

/**
 * DELETE /api/v1/albums/{albumId}/shares/{userId}: revoke one
 * account's view ACL (the owner or an administrator any grantee, a
 * grantee their own row -- the departure). Idempotent 204.
 */
export async function revokeAlbumShare(
	albumId: string,
	userId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumMutationResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}/shares/${encodeURIComponent(userId)}`,
			{ method: 'DELETE' },
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.ok) {
		return { ok: true };
	}
	return { ok: false, errorKey: await errorKeyFrom(response) };
}

/**
 * GET /api/v1/users: the instance's minimal member list, every
 * account `email` ascending -- the share dialog's pick list. The
 * entries carry the disabled switch so callers can present accounts
 * that cannot sign in for what they are.
 */
export async function listUsers(
	fetcher: typeof fetch = globalThis.fetch
): Promise<UserListResult> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/users', undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!isObject(body) || !Array.isArray(body.users)) {
		return { ok: false, errorKey: 'internal' };
	}
	const users: UserSummary[] = [];
	for (const entry of body.users) {
		const user = userSummaryFrom(entry);
		if (user === null) {
			return { ok: false, errorKey: 'internal' };
		}
		users.push(user);
	}
	return { ok: true, users };
}

/**
 * PUT /api/v1/albums/{albumId}/timeline-preference: the caller's own
 * per-user switch of whether the album feeds their aggregated
 * timeline. Self-affecting state: the write publishes NO event, this
 * response is the only view that changed.
 */
export async function setAlbumTimelinePreference(
	albumId: string,
	inTimeline: boolean,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumPreferenceResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}/timeline-preference`,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ inTimeline })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!isObject(body)) {
		return { ok: false, errorKey: 'internal' };
	}
	const candidate = body as Partial<AlbumTimelinePreferenceResponse>;
	if (candidate.albumId !== albumId || candidate.inTimeline !== inTimeline) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, preference: body as AlbumTimelinePreferenceResponse };
}
