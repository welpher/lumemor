// The albums surface's data client (M3c Task 5, reshaped by E4's
// albums-as-sources): the wall list, the detail read, the directory
// creation (title + source path + audience, the honest skipped
// report), the property patch, the virtual/directory delete, the
// path completion of the creation dialog's combobox, the audience's
// full-set PUT, the rescan trigger, the instance's member list and
// the timeline preference write -- each behind the repo-wide
// error-code convention: every failure outcome carries the error
// code's i18n key (the envelope code lowercased, 'internal' for a
// network failure or an unparseable body, 'session_expired' for a
// 401).
//
// THE MESSAGE-KEY EXCEPTIONS (the people client's requires-label
// pattern): the album write faces answer a handful of 400
// VALIDATION_FAILED whose `details` carry a message key rather than
// an error code of their own -- the cover validation
// (`album_cover_not_member`) and the preference's not-excludable
// kinds (`album_pref_not_excludable`). The error-key helper surfaces
// those keys as the failure's errorKey, so the honest message
// renders through the same resolveErrorText path every code takes.
// (The share target rules' message keys retired with the
// grant/revoke pair -- the audience write reports its skips in the
// RESPONSE body, never as a 400.)
//
// The wall's two groups arrive concatenated (the pinned sync card
// first, then the own group, shared after); the page sections them
// off each item's `callerRole`. The media URLs (every `thumbnail`
// field) arrive as strings from the server; no URL is assembled
// here. A `type: 'sync'` card is the caller's own default album:
// its `title` is the system's internal label and the UI renders the
// localized display name (album_all_photos) keyed by the type --
// never the stored title.

import { apiFetch } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One wall card, exactly the contract's AlbumSummary shape. */
export type AlbumSummary = components['schemas']['AlbumSummary'];

/** One album's detail as the caller sees it. */
export type AlbumDetail = components['schemas']['AlbumDetail'];

/** One grantee row of the owner's detail `shares` list. */
export type AlbumShareEntry = components['schemas']['AlbumShareEntry'];

/** One requested audience id the write could not grant, with its reason. */
export type AlbumAudienceSkip = components['schemas']['AlbumAudienceSkip'];

/** One account of the instance's minimal member list. */
export type UserSummary = components['schemas']['UserSummary'];

/** The timeline preference write's response. */
export type AlbumTimelinePreferenceResponse = components['schemas']['AlbumTimelinePreferenceResponse'];

/** The audience write's honest report. */
export type AlbumAudienceResponse = components['schemas']['AlbumAudienceResponse'];

/** The rescan trigger's answer: the run's job_runs identifier. */
export type AlbumRescanResponse = components['schemas']['AlbumRescanResponse'];

/** The wall list's outcome. */
export type AlbumListResult =
	| { ok: true; albums: AlbumSummary[] }
	| { ok: false; errorKey: string };

/** The detail read's outcome. */
export type AlbumDetailResult = { ok: true; detail: AlbumDetail } | { ok: false; errorKey: string };

/**
 * The creation's outcome: the created wall card plus the honest
 * audience report -- the requested ids that did not become audience
 * (the dialog surfaces them before closing).
 */
export type AlbumCreateResult =
	| { ok: true; album: AlbumSummary; skipped: AlbumAudienceSkip[] }
	| { ok: false; errorKey: string };

/** The property patch's outcome (the response is the card after the edit). */
export type AlbumPatchResult = { ok: true; album: AlbumSummary } | { ok: false; errorKey: string };

/** A body-less mutation's outcome (the delete). */
export type AlbumMutationResult = { ok: true } | { ok: false; errorKey: string };

/**
 * The path completion's outcome: the pickable directories, at most
 * 20, the server's own honest answer (an empty list never an error).
 */
export type PathCompletionsResult =
	| { ok: true; paths: string[] }
	| { ok: false; errorKey: string };

/** The audience write's outcome (the response IS the honest report). */
export type AlbumAudienceResult =
	| { ok: true; report: AlbumAudienceResponse }
	| { ok: false; errorKey: string };

/**
 * The rescan trigger's outcome: the joined-or-created run's
 * identifier (200 joined an in-flight run, 202 created one).
 */
export type AlbumRescanResult = { ok: true; run: AlbumRescanResponse } | { ok: false; errorKey: string };

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
	if (candidate.type !== 'virtual' && candidate.type !== 'directory' && candidate.type !== 'sync') {
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

/** Structurally validate one audience skip entry; null when it is not one. */
function audienceSkipFrom(body: unknown): AlbumAudienceSkip | null {
	if (!isObject(body)) {
		return null;
	}
	const candidate = body as Partial<AlbumAudienceSkip>;
	if (typeof candidate.userId !== 'string' || candidate.userId === '') {
		return null;
	}
	if (candidate.reason !== 'owner' && candidate.reason !== 'invalid_account') {
		return null;
	}
	return body as AlbumAudienceSkip;
}

/**
 * Structurally validate the creation's answer: the created wall card
 * plus the required honest `skipped` report (empty when the whole
 * audience landed).
 */
function createResponseFrom(body: unknown): { album: AlbumSummary; skipped: AlbumAudienceSkip[] } | null {
	const album = summaryFrom(body);
	if (album === null) {
		return null;
	}
	if (!isObject(body) || !Array.isArray(body.skipped)) {
		return null;
	}
	const skipped: AlbumAudienceSkip[] = [];
	for (const entry of body.skipped) {
		const skip = audienceSkipFrom(entry);
		if (skip === null) {
			return null;
		}
		skipped.push(skip);
	}
	return { album, skipped };
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
	if (candidate.type !== 'virtual' && candidate.type !== 'directory' && candidate.type !== 'sync') {
		return null;
	}
	if (candidate.locked !== true && candidate.locked !== false) {
		return null;
	}
	if (candidate.coverAssetId !== null && typeof candidate.coverAssetId !== 'string') {
		return null;
	}
	// The source directory rides the directory albums only; a string
	// when present, never null (the edit dialog's read-only row).
	if (candidate.rootPath !== undefined && typeof candidate.rootPath !== 'string') {
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
 * GET /api/v1/albums: the caller's wall, the pinned sync card (the
 * default "My photos" album) followed by the own group and the
 * shared group in one array. The page sections them off each
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
 * rides the owner's view alone, `rootPath` the directory albums'
 * source row, `inTimeline` the excludable kinds. The caller's OWN
 * sync album answers here (read-only, the default album); a 404 is
 * the anti-enumeration pair (unknown id, another account's
 * unshared album, every foreign sync album).
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
 * POST /api/v1/albums: create one directory album -- the album IS a
 * source. The request names an existing directory (the server runs
 * the path validation matrix; every branch its own 400 error code)
 * and the audience the album starts with; the files stay where they
 * are, zero copy, zero move. The response is the created album's
 * wall card (the creation-time zeros: no member, no cover) plus the
 * honest `skipped` report of the requested audience ids that did
 * not land (an unknown, disabled or own id) -- never errors.
 */
export async function createAlbum(
	input: { title: string; rootPath: string; audience: readonly string[] },
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumCreateResult> {
	let response: Response;
	try {
		response = await apiFetch(
			'/api/v1/albums',
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					title: input.title,
					rootPath: input.rootPath,
					audience: [...input.audience]
				})
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const created = createResponseFrom(await bodyOf(response));
	if (created === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, album: created.album, skipped: created.skipped };
}

/**
 * GET /api/v1/albums/path-completions: the creation dialog's source
 * path completion. The domain is EXACTLY the creation validation
 * domain -- only directories at or under a configured mount root
 * (an empty allowlist answers nothing, so the first root is always
 * typed by hand or opened by an admin). The empty prefix lists the
 * roots themselves; a typed prefix narrows to the parent's matching
 * children; a trailing separator drills into the prefix itself.
 * The answer is at most 20 rows and the empty list is honest, never
 * an error.
 */
export async function completeAlbumPath(
	prefix: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<PathCompletionsResult> {
	const query = prefix === '' ? '' : `?prefix=${encodeURIComponent(prefix)}`;
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/albums/path-completions${query}`, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!isObject(body) || !Array.isArray(body.paths) || body.paths.some((path) => typeof path !== 'string')) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, paths: body.paths as string[] };
}

/**
 * PATCH /api/v1/albums/{albumId}: the owner's partial edit of the
 * properties (`title`, `coverAssetId`, `locked`, any
 * subset). The response is the album's wall card after the edit.
 */
export async function patchAlbum(
	albumId: string,
	patch: { title?: string; coverAssetId?: string; locked?: boolean },
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
 * PUT /api/v1/albums/{albumId}/audience: the album's audience as ONE
 * declarative write -- the request carries the complete grantee set,
 * the server converges the share rows onto exactly it (grants land,
 * departures leave, the same set twice is the silent no-op). The
 * response is the write's honest report: the audience as it stands
 * after the write, the transitions this write caused, and the
 * requested ids it skipped with each one's reason. An empty set is
 * the valid "only me"; the owner's own visibility is the ownership
 * row, never a share row, so the set can never remove the owner.
 */
export async function updateAlbumAudience(
	albumId: string,
	audience: readonly string[],
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumAudienceResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}/audience`,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ audience: [...audience] })
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
	const candidate = body as Partial<AlbumAudienceResponse>;
	if (candidate.albumId !== albumId) {
		return { ok: false, errorKey: 'internal' };
	}
	if (
		!Array.isArray(candidate.audience) ||
		!Array.isArray(candidate.granted) ||
		!Array.isArray(candidate.revoked) ||
		!Array.isArray(candidate.skipped)
	) {
		return { ok: false, errorKey: 'internal' };
	}
	const report = body as AlbumAudienceResponse;
	for (const entry of report.audience) {
		if (shareEntryFrom(entry) === null) {
			return { ok: false, errorKey: 'internal' };
		}
	}
	for (const id of report.granted) {
		if (typeof id !== 'string') {
			return { ok: false, errorKey: 'internal' };
		}
	}
	for (const id of report.revoked) {
		if (typeof id !== 'string') {
			return { ok: false, errorKey: 'internal' };
		}
	}
	for (const entry of report.skipped) {
		if (audienceSkipFrom(entry) === null) {
			return { ok: false, errorKey: 'internal' };
		}
	}
	return { ok: true, report };
}

/**
 * POST /api/v1/albums/{albumId}/rescan: start (or join) one rescan
 * of the directory album's source -- the three-arm diff that
 * reconciles the album's membership with what the filesystem holds.
 * The endpoint is idempotent per album: a run still in flight
 * answers 200 with its identifier and starts nothing; 202 carries
 * the freshly created run. The owner (or an administrator)
 * proceeds; the run's progress surfaces on the activity stream's
 * job.* events keyed by the returned identifier.
 */
export async function rescanAlbum(
	albumId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumRescanResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}/rescan`,
			{ method: 'POST' },
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!isObject(body) || typeof body.jobRunId !== 'string' || body.jobRunId === '') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, run: body as AlbumRescanResponse };
}

/**
 * DELETE /api/v1/albums/{albumId}: delete the caller's virtual or
 * directory album. The row goes with its membership and share rows
 * (the schema's cascades); 204, no body. (External assets the scan
 * ingested keep their rows -- the trash owns them now.)
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
 * GET /api/v1/users: the instance's minimal member list, every
 * account `email` ascending -- the audience picker's candidate
 * source. The entries carry the disabled switch so callers can
 * present accounts that cannot sign in for what they are (the
 * server's audience write skips them with `invalid_account`, the
 * honest report rather than a refusal).
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
