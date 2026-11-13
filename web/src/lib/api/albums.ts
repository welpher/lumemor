// The albums surface's data client (reshaped by the
// albums-as-sources and album-autonomy reworks, widened by the
// memory's birth): the wall list, the detail read (the deletion faces
// included -- the write probe's `sourceWritable` fact, the owner's
// `allowAppDeletion` switch, the derived `deletable` affordance), the
// directory creation (title + source path + audience + the two
// autonomy switches, the honest skipped report and the probe's
// read-only fact -- both admin-guarded writes), the memory creation
// (title + audience, every member's write -- a memory is personal
// curation, not configuration) and its curation pair (the membership
// add and remove over the items face, each answering its honest
// counts-only report), the property patch (title, cover,
// deletability), the memory/directory delete, the path completion of
// the creation dialog's combobox (the system-wide domain: the
// container's real visible filesystem, the managed tree never
// listed; every row carries the write fact the listing's probe
// discovered), the audience's full-set PUT, the rescan trigger, the
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
// (`album_cover_not_member`), the preference's not-excludable kinds
// (`album_pref_not_excludable`) and the deletability switch's
// not-a-directory refusal (`album_deletable_not_directory`). The
// error-key helper surfaces those keys as the failure's errorKey,
// so the honest message renders through the same resolveErrorText
// path every code takes. (The share target rules' message keys
// retired with the grant/revoke pair -- the audience write reports
// its skips in the RESPONSE body, never as a 400.)
//
// The wall arrives concatenated (the pinned sync card first, then the
// own group, shared after) with each item's `callerRole` discriminator
// beside it; the wall renders ONE list, which is why the page sorts the
// cards by their own `updatedAt` and lets the role key each card's face
// (a viewer card carries the read-only badge; an own card its shared
// state). The media URLs (every `thumbnail`
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

/** One completion row of the path listing: the directory and its write fact. */
export type AlbumPathCompletion = components['schemas']['AlbumPathCompletion'];

/** One grantee row of the owner's detail `shares` list. */
export type AlbumShareEntry = components['schemas']['AlbumShareEntry'];

/** The creation response's own shape (the summary card plus its facts). */
type AlbumCreateResponse = components['schemas']['AlbumCreateResponse'];

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
export type AlbumListResult = { ok: true; albums: AlbumSummary[] } | { ok: false; errorKey: string };

/** The detail read's outcome. */
export type AlbumDetailResult = { ok: true; detail: AlbumDetail } | { ok: false; errorKey: string };

/**
 * The memory creation's outcome: the created wall card and the honest
 * audience report (the requested ids that did not become audience --
 * the dialog surfaces them before closing). A memory owns no source
 * and no switches, so no probe facts ride this answer.
 */
export type MemoryCreateResult =
	{ ok: true; memory: AlbumSummary; skipped: AlbumAudienceSkip[] } | { ok: false; errorKey: string };

/**
 * The membership add's outcome: the honest counts-only report -- the
 * rows that landed and the distinct ids that stayed out (not visible
 * to the caller, not live, or already members).
 */
export type AddAlbumItemsResult = { ok: true; added: number; skipped: number } | { ok: false; errorKey: string };

/**
 * The membership remove's outcome: the honest counts-only report --
 * the rows that left and the distinct ids that were no members.
 */
export type RemoveAlbumItemsResult = { ok: true; removed: number; skipped: number } | { ok: false; errorKey: string };

/**
 * The creation's outcome: the created wall card, the honest audience
 * report (the requested ids that did not become audience -- the
 * dialog surfaces them before closing) and the write probe's
 * `sourceWritable` fact (false is honest creation feedback -- a
 * read-only source is a legitimate album -- never a failure).
 */
export type AlbumCreateResult =
	| { ok: true; album: AlbumSummary; skipped: AlbumAudienceSkip[]; sourceWritable: boolean }
	| { ok: false; errorKey: string };

/** The property patch's outcome (the response is the card after the edit). */
export type AlbumPatchResult = { ok: true; album: AlbumSummary } | { ok: false; errorKey: string };

/** A body-less mutation's outcome (the delete). */
export type AlbumMutationResult = { ok: true } | { ok: false; errorKey: string };

/**
 * The path completion's outcome: the pickable directories with each
 * row's write fact, at most 20, the server's own honest answer (an
 * empty list never an error).
 */
export type PathCompletionsResult = { ok: true; completions: AlbumPathCompletion[] } | { ok: false; errorKey: string };

/** The audience write's outcome (the response IS the honest report). */
export type AlbumAudienceResult = { ok: true; report: AlbumAudienceResponse } | { ok: false; errorKey: string };

/**
 * The rescan trigger's outcome: the joined-or-created run's
 * identifier (200 joined an in-flight run, 202 created one).
 */
export type AlbumRescanResult = { ok: true; run: AlbumRescanResponse } | { ok: false; errorKey: string };

/** The member list's outcome. */
export type UserListResult = { ok: true; users: UserSummary[] } | { ok: false; errorKey: string };

/** The timeline preference write's outcome. */
export type AlbumPreferenceResult =
	{ ok: true; preference: AlbumTimelinePreferenceResponse } | { ok: false; errorKey: string };

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
	'album_pref_not_excludable',
	'album_deletable_not_directory'
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
	if (candidate.type !== 'memory' && candidate.type !== 'directory' && candidate.type !== 'sync') {
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
		if (
			owner === undefined ||
			typeof owner.id !== 'string' ||
			typeof owner.email !== 'string' ||
			typeof owner.name !== 'string'
		) {
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
 * Structurally validate the creation's answer: the created wall card,
 * the required honest `skipped` report (empty when the whole audience
 * landed) and the required `sourceWritable` fact of the write probe
 * (the creation succeeded either way -- false is feedback, not
 * failure).
 */
function createResponseFrom(
	body: unknown
): { album: AlbumSummary; skipped: AlbumAudienceSkip[]; sourceWritable: boolean } | null {
	const album = summaryFrom(body);
	if (album === null) {
		return null;
	}
	if (!isObject(body) || !Array.isArray(body.skipped)) {
		return null;
	}
	if (typeof (body as Partial<AlbumCreateResponse>).sourceWritable !== 'boolean') {
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
	return { album, skipped, sourceWritable: (body as AlbumCreateResponse).sourceWritable };
}

/**
 * Structurally validate the memory creation's answer: the created
 * wall card plus the required honest `skipped` report (empty when the
 * whole audience landed). A memory's answer carries no probe facts --
 * nothing else to validate. The returned card is the summary shape
 * alone (the report fields are stripped -- they are the write's
 * answer, not the card's).
 */
function memoryCreateResponseFrom(body: unknown): { memory: AlbumSummary; skipped: AlbumAudienceSkip[] } | null {
	const memory = summaryFrom(body);
	if (memory === null) {
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
	const { skipped: _report, ...card } = body as AlbumSummary & { skipped: unknown };
	return { memory: card as AlbumSummary, skipped };
}

/**
 * Structurally validate one membership report: both counts present,
 * non-negative integers. The reports are counts-only by contract --
 * no per-id reasons ride them (the row's landing or absence is the
 * whole answer).
 */
function itemsReportFrom(
	body: unknown,
	addedKey: string,
	skippedKey: string
): { landed: number; skipped: number } | null {
	if (!isObject(body)) {
		return null;
	}
	const landed = (body as Record<string, unknown>)[addedKey];
	const skipped = (body as Record<string, unknown>)[skippedKey];
	if (
		typeof landed !== 'number' ||
		!Number.isInteger(landed) ||
		landed < 0 ||
		typeof skipped !== 'number' ||
		!Number.isInteger(skipped) ||
		skipped < 0
	) {
		return null;
	}
	return { landed, skipped };
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
	if (candidate.type !== 'memory' && candidate.type !== 'directory' && candidate.type !== 'sync') {
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
	// The deletion faces ride the directory albums: the probe's
	// `sourceWritable` fact (both roles), the owner's
	// `allowAppDeletion` switch and the derived `deletable`
	// affordance (the owner's view alone) -- each optional, but a
	// present value must be the boolean it names.
	if (candidate.sourceWritable !== undefined && typeof candidate.sourceWritable !== 'boolean') {
		return null;
	}
	if (candidate.allowAppDeletion !== undefined && typeof candidate.allowAppDeletion !== 'boolean') {
		return null;
	}
	if (candidate.deletable !== undefined && typeof candidate.deletable !== 'boolean') {
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
	// The display name is required on the wire and may be empty: the
	// pickers' fallback reads the value, never the field's presence.
	if (typeof candidate.name !== 'string') {
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
 * shared group in one array. The wall renders one list: the page
 * keys each card's face off `callerRole` and orders the cards by
 * their own `updatedAt` (the sync card stays pinned first).
 */
export async function listAlbums(fetcher: typeof fetch = globalThis.fetch): Promise<AlbumListResult> {
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
 * rides the owner's view alone, `rootPath` and the deletion faces
 * (`sourceWritable`, the owner's `allowAppDeletion` and the derived
 * `deletable`) the directory albums, `inTimeline` the excludable
 * kinds. The caller's OWN sync album answers here (read-only, the
 * default album); a 404 is the anti-enumeration pair (unknown id,
 * another account's unshared album, every foreign sync album).
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
 * source, and the write is the ADMINISTRATOR's (a member answers 403
 * before any part of the request is considered: pointing at the
 * file system is configuration, not curation). The request names an
 * existing directory of the container's real visible filesystem (the
 * server runs the path validation matrix; every branch its own 400
 * error code; the managed tree is refused on the resolved path) and
 * the audience the album starts with; the files stay where they
 * are, zero copy, zero move. The two autonomy switches ride the
 * creation: `allowAppDeletion` carries the dialog's deletability
 * toggle (present exactly when the picked completion row's
 * `writable` fact rendered the row -- the server's default governs
 * when the dialog omits it), `inTimeline` the creator's own
 * exclusion toggle (false lands the per-user preference row in the
 * creation itself). The response is the created album's wall card
 * (the creation-time zeros: no member, no cover), the honest
 * `skipped` report of the requested audience ids that did not land
 * (an unknown, disabled or own id) and the write probe's
 * `sourceWritable` fact -- a read-only source is a legitimate
 * album, `false` is the honest feedback, never a failure.
 */
export async function createAlbum(
	input: {
		title: string;
		rootPath: string;
		audience: readonly string[];
		allowAppDeletion?: boolean;
		inTimeline?: boolean;
	},
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
					audience: [...input.audience],
					// The switches ride only when the dialog carries them:
					// the deletability toggle exists exactly where the
					// picked row's write fact rendered it, and the
					// exclusion toggle always holds a draft.
					...(input.allowAppDeletion !== undefined ? { allowAppDeletion: input.allowAppDeletion } : {}),
					...(input.inTimeline !== undefined ? { inTimeline: input.inTimeline } : {})
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
	return {
		ok: true,
		album: created.album,
		skipped: created.skipped,
		sourceWritable: created.sourceWritable
	};
}

/**
 * POST /api/v1/memories: create one memory -- a reference set the
 * caller curates photo by photo. The write is EVERY MEMBER'S: a
 * memory is personal curation (who sees which of the caller's
 * picks), not configuration (no filesystem location is named), so
 * the face holds no role gate at all. The request carries the title
 * and the starting audience, nothing else: a memory owns no source
 * path, no deletability switch (its delete never moves files) and
 * no timeline switch (it never feeds its creator's timeline -- the
 * exclusion is the memory's fixed semantics, landed with the row
 * itself). The response is the created memory's wall card (the
 * creation-time constants: `callerRole` `owner`, `type` `memory`,
 * no members, no cover) plus the honest audience `skipped` report
 * (the requested ids that did not land -- the caller's own id, an
 * unknown or disabled account).
 */
export async function createMemory(
	input: { title: string; audience?: readonly string[] },
	fetcher: typeof fetch = globalThis.fetch
): Promise<MemoryCreateResult> {
	let response: Response;
	try {
		response = await apiFetch(
			'/api/v1/memories',
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					title: input.title,
					// The audience field rides only when the dialog picked
					// members: an absent field is the server's "only me".
					...(input.audience !== undefined ? { audience: [...input.audience] } : {})
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
	const created = memoryCreateResponseFrom(await bodyOf(response));
	if (created === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, memory: created.memory, skipped: created.skipped };
}

/**
 * POST /api/v1/albums/{albumId}/items: add membership rows to the
 * caller's memory -- the wall-selection curation entry's write. The
 * request carries the picked asset ids (1-200, repeated ids
 * collapse); a row lands only for a photo the creator currently
 * sees as a live row, everything else (foreign-unshared, non-live,
 * already a member) is skipped, never a per-item error -- the
 * response is the honest counts-only report `{added, skipped}`. A
 * no-op add still answers 200; the event family fires exactly when
 * a row lands.
 */
export async function addAlbumItems(
	albumId: string,
	assetIds: readonly string[],
	fetcher: typeof fetch = globalThis.fetch
): Promise<AddAlbumItemsResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}/items`,
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ assetIds: [...assetIds] })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const report = itemsReportFrom(await bodyOf(response), 'added', 'skipped');
	if (report === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, added: report.landed, skipped: report.skipped };
}

/**
 * DELETE /api/v1/albums/{albumId}/items: remove membership rows from
 * the caller's memory -- the tidy-up of the add's rows. The photos
 * are not the memory's: they live on in the albums they were
 * ingested through, and the references simply leave the set. The
 * response is the honest counts-only report `{removed, skipped}`.
 */
export async function removeAlbumItems(
	albumId: string,
	assetIds: readonly string[],
	fetcher: typeof fetch = globalThis.fetch
): Promise<RemoveAlbumItemsResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/albums/${encodeURIComponent(albumId)}/items`,
			{
				method: 'DELETE',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ assetIds: [...assetIds] })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const report = itemsReportFrom(await bodyOf(response), 'removed', 'skipped');
	if (report === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, removed: report.landed, skipped: report.skipped };
}

/**
 * GET /api/v1/albums/path-completions: the creation dialog's source
 * path completion. The write is the ADMINISTRATOR's (a member
 * answers 403 before the prefix is read), and the domain is
 * SYSTEM-WIDE (the allowlist retired
 * with it -- the boundary is the compose mount itself): the
 * container's real visible filesystem, drilled one segment at a
 * time, the managed tree never listed on any row. The empty prefix
 * lists the root itself (`/`); a typed prefix narrows to the
 * parent's matching children; a trailing separator drills into the
 * prefix. Every row carries the write fact the listing's probe
 * discovered (`writable`) -- the dialog renders the deletability
 * switch exactly for the picked writable row. The answer is at most
 * 20 rows and the empty list is honest, never an error.
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
	if (!isObject(body) || !Array.isArray(body.completions)) {
		return { ok: false, errorKey: 'internal' };
	}
	const completions: AlbumPathCompletion[] = [];
	for (const entry of body.completions) {
		// One row: the path and the probe's write fact, both required.
		const candidate = entry as Partial<AlbumPathCompletion>;
		if (!isObject(entry) || typeof candidate.path !== 'string' || typeof candidate.writable !== 'boolean') {
			return { ok: false, errorKey: 'internal' };
		}
		completions.push(entry as AlbumPathCompletion);
	}
	return { ok: true, completions };
}

/**
 * PATCH /api/v1/albums/{albumId}: the owner's partial edit of the
 * properties (`title`, `coverAssetId`, `allowAppDeletion`, any
 * subset). The response is the album's wall card after the edit.
 * The deletability switch is the DIRECTORY album's own (a memory
 * album answers the message key `album_deletable_not_directory`).
 */
export async function patchAlbum(
	albumId: string,
	patch: { title?: string; coverAssetId?: string; allowAppDeletion?: boolean },
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
		response = await apiFetch(`/api/v1/albums/${encodeURIComponent(albumId)}/rescan`, { method: 'POST' }, fetcher);
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
 * DELETE /api/v1/albums/{albumId}: remove the caller's memory or
 * directory album. The row goes with its membership, share and
 * preference rows (the schema's cascades). The MEMORY arm is the
 * light cascade: the references and the audience die, the photos are
 * untouched (they live in the albums they were ingested through).
 * The directory arm is the un-ingest -- every member no other album
 * references is purged from the library (its source file untouched),
 * the guarded survivors keep their rows. 204, no body.
 */
export async function deleteAlbum(
	albumId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AlbumMutationResult> {
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/albums/${encodeURIComponent(albumId)}`, { method: 'DELETE' }, fetcher);
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
export async function listUsers(fetcher: typeof fetch = globalThis.fetch): Promise<UserListResult> {
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
