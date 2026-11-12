// The people surface's data client: the three reads of the people
// domain (the named list, the unnamed region's keyset pages, the
// cluster detail) and the seven writes (label upsert, label delete,
// the hidden switch, the structural merge, the single-face move, the
// from-face person creation, the single-face detach), each behind
// the repo-wide error-code convention -- every failure outcome
// carries the error code's i18n key (the envelope code lowercased,
// 'internal' for a network failure or an unparseable body,
// 'session_expired' for a 401).
//
// THE ONE MESSAGE-KEY EXCEPTION: the hidden switch rides on the
// caller's label row, so hiding an UNLABELED cluster answers 400
// VALIDATION_FAILED whose `details` carry the message key
// `person_hide_requires_label` (not an error code of its own). The
// hidden-write helper surfaces that key as the failure's errorKey --
// the key exists in the i18n table, so the honest message renders
// through the same resolveErrorText path every code takes.
//
// The media URLs (every `thumbnail` field) arrive as strings from
// the server; no URL is assembled here.

import { apiFetch } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One named person of the caller's list, exactly the contract's shape. */
export type PersonSummary = components['schemas']['PersonSummary'];

/** One unnamed cluster of the caller's unnamed region. */
export type UnnamedClusterSummary = components['schemas']['UnnamedClusterSummary'];

/** One cluster's detail as the caller sees it. */
export type ClusterDetail = components['schemas']['ClusterDetail'];

/** The caller's label state of one cluster (the detail's `label`). */
export type PersonLabel = components['schemas']['PersonLabel'];

/** One face of a cluster's detail. */
export type FaceDetail = components['schemas']['FaceDetail'];

/** The label/hidden write's response. */
export type PersonLabelResponse = components['schemas']['PersonLabelResponse'];

/** The merge's response. */
export type MergePeopleResponse = components['schemas']['MergePeopleResponse'];

/** The face move's response. */
export type MoveFaceResponse = components['schemas']['MoveFaceResponse'];

/** The from-face person creation's response. */
export type ClusterFaceResponse = components['schemas']['ClusterFaceResponse'];

/** The named list's outcome. */
export type PeopleListResult = { ok: true; people: PersonSummary[] } | { ok: false; errorKey: string };

/** One keyset page of the unnamed region. */
export type UnnamedPage = { items: UnnamedClusterSummary[]; nextCursor: string | null };

/** The unnamed page fetch's outcome. */
export type UnnamedPageResult = { ok: true; page: UnnamedPage } | { ok: false; errorKey: string };

/** The cluster detail's outcome. */
export type ClusterResult = { ok: true; detail: ClusterDetail } | { ok: false; errorKey: string };

/** The label write's outcome (the response IS the fresh label state). */
export type LabelResult = { ok: true; label: PersonLabelResponse } | { ok: false; errorKey: string };

/** A body-less mutation's outcome (the label delete). */
export type MutationResult = { ok: true } | { ok: false; errorKey: string };

/** The merge's outcome. */
export type MergeResult = { ok: true; merge: MergePeopleResponse } | { ok: false; errorKey: string };

/** The face move's outcome. */
export type MoveResult = { ok: true; move: MoveFaceResponse } | { ok: false; errorKey: string };

/** The from-face person creation's outcome. */
export type ClusterFaceResult = { ok: true; cluster: ClusterFaceResponse } | { ok: false; errorKey: string };

/** The contract's default page size of the unnamed region. */
export const UNNAMED_PAGE_LIMIT = 60;

/**
 * Map one failed response onto the error-code i18n key, carrying the
 * hidden switch's message key when the envelope's `details` hold it
 * (the one VALIDATION_FAILED whose honest rendering is a message,
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
		if (key === 'person_hide_requires_label') {
			return 'person_hide_requires_label';
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

/** Structurally validate a person summary; null when it is not one. */
function personFrom(body: unknown): PersonSummary | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<PersonSummary>;
	if (
		typeof candidate.clusterId !== 'string' ||
		typeof candidate.name !== 'string' ||
		typeof candidate.representativeFaceId !== 'string' ||
		typeof candidate.thumbnail !== 'string'
	) {
		return null;
	}
	if (typeof candidate.faceCount !== 'number' || !Number.isInteger(candidate.faceCount) || candidate.faceCount < 0) {
		return null;
	}
	if (candidate.isHidden !== true && candidate.isHidden !== false) {
		return null;
	}
	return body as PersonSummary;
}

/** Structurally validate an unnamed cluster summary; null when not one. */
function clusterFrom(body: unknown): UnnamedClusterSummary | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<UnnamedClusterSummary>;
	if (
		typeof candidate.clusterId !== 'string' ||
		typeof candidate.representativeFaceId !== 'string' ||
		typeof candidate.thumbnail !== 'string'
	) {
		return null;
	}
	if (typeof candidate.faceCount !== 'number' || !Number.isInteger(candidate.faceCount) || candidate.faceCount < 0) {
		return null;
	}
	return body as UnnamedClusterSummary;
}

/** Structurally validate one unnamed page; null when it is not a page. */
function unnamedPageFrom(body: unknown): UnnamedPage | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const { items, nextCursor: cursor } = body as { items?: unknown; nextCursor?: unknown };
	if (!Array.isArray(items)) {
		return null;
	}
	for (const entry of items) {
		if (clusterFrom(entry) === null) {
			return null;
		}
	}
	if (cursor !== undefined && typeof cursor !== 'string') {
		return null;
	}
	return { items: items as UnnamedClusterSummary[], nextCursor: cursor ?? null };
}

/** Structurally validate one face detail; null when it is not one. */
function faceFrom(body: unknown): FaceDetail | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<FaceDetail>;
	if (
		typeof candidate.id !== 'string' ||
		typeof candidate.assetId !== 'string' ||
		typeof candidate.thumbnail !== 'string'
	) {
		return null;
	}
	if (typeof candidate.score !== 'number' || !Number.isFinite(candidate.score)) {
		return null;
	}
	const asset = candidate.asset as Partial<FaceDetail['asset']> | undefined;
	if (asset === undefined || typeof asset.takenAt !== 'string' || typeof asset.originalFilename !== 'string') {
		return null;
	}
	return body as FaceDetail;
}

/** Structurally validate a cluster detail; null when it is not one. */
function detailFrom(body: unknown): ClusterDetail | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<ClusterDetail>;
	if (typeof candidate.clusterId !== 'string' || !Array.isArray(candidate.faces)) {
		return null;
	}
	const label = candidate.label;
	if (label !== null && label !== undefined) {
		const labelCandidate = label as Partial<PersonLabel>;
		if (typeof labelCandidate.name !== 'string') {
			return null;
		}
		if (labelCandidate.isHidden !== true && labelCandidate.isHidden !== false) {
			return null;
		}
	}
	for (const entry of candidate.faces) {
		if (faceFrom(entry) === null) {
			return null;
		}
	}
	return body as ClusterDetail;
}

/** Structurally validate a label response; null when it is not one. */
function labelResponseFrom(body: unknown): PersonLabelResponse | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<PersonLabelResponse>;
	if (typeof candidate.clusterId !== 'string' || typeof candidate.name !== 'string') {
		return null;
	}
	if (candidate.isHidden !== true && candidate.isHidden !== false) {
		return null;
	}
	return body as PersonLabelResponse;
}

/**
 * GET /api/v1/people: the caller's named people, name-sorted by the
 * server. Hidden people stay listed carrying `isHidden` true.
 */
export async function listPeople(fetcher: typeof fetch = globalThis.fetch): Promise<PeopleListResult> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/people', undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (!Array.isArray(body)) {
		return { ok: false, errorKey: 'internal' };
	}
	const people: PersonSummary[] = [];
	for (const entry of body) {
		const person = personFrom(entry);
		if (person === null) {
			return { ok: false, errorKey: 'internal' };
		}
		people.push(person);
	}
	return { ok: true, people };
}

/**
 * GET /api/v1/people/unnamed: one keyset page of the caller's
 * unnamed clusters, largest face count first. `cursor` is null on
 * the first page and the previous response's `nextCursor` after it
 * (an opaque server string, never assembled here).
 */
export async function listUnnamedPeople(
	cursor: string | null,
	fetcher: typeof fetch = globalThis.fetch
): Promise<UnnamedPageResult> {
	const query = `limit=${UNNAMED_PAGE_LIMIT}${cursor === null ? '' : `&cursor=${encodeURIComponent(cursor)}`}`;
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/people/unnamed?${query}`, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const page = unnamedPageFrom(await bodyOf(response));
	if (page === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, page };
}

/**
 * GET /api/v1/people/{clusterId}: the cluster's detail as the caller
 * sees it. `label` is null in the unnamed state; a 404 is the
 * anti-enumeration pair (unknown id or invisible cluster).
 */
export async function fetchPersonCluster(
	clusterId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<ClusterResult> {
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/people/${encodeURIComponent(clusterId)}`, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const detail = detailFrom(await bodyOf(response));
	if (detail === null || detail.clusterId !== clusterId) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, detail };
}

/**
 * PUT /api/v1/people/{clusterId}/label: upsert the caller's label.
 * The response is the caller's label state after the write (the
 * trimmed name and the surviving hidden switch), so the caller
 * adopts it as the fresh state -- no refetch. A rename keeps the
 * hidden switch; duplicate names are allowed (no NAME_TAKEN).
 */
export async function labelPerson(
	clusterId: string,
	name: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<LabelResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/people/${encodeURIComponent(clusterId)}/label`,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const label = labelResponseFrom(await bodyOf(response));
	if (label === null || label.clusterId !== clusterId) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, label };
}

/**
 * DELETE /api/v1/people/{clusterId}/label: remove the caller's label
 * row. Idempotent once the cluster is visible (204, no body); the
 * person returns to the caller's unnamed region.
 */
export async function unlabelPerson(
	clusterId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<MutationResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/people/${encodeURIComponent(clusterId)}/label`,
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
 * PUT /api/v1/people/{clusterId}/hidden: flip the caller's hidden
 * switch. The response is the label state after the flip. Hiding an
 * UNLABELED cluster answers 400 whose details carry the message key
 * `person_hide_requires_label` -- surfaced as that errorKey so the
 * page renders the honest message and offers the naming dialog.
 */
export async function setPersonHidden(
	clusterId: string,
	isHidden: boolean,
	fetcher: typeof fetch = globalThis.fetch
): Promise<LabelResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/people/${encodeURIComponent(clusterId)}/hidden`,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ isHidden })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const label = labelResponseFrom(await bodyOf(response));
	if (label === null || label.clusterId !== clusterId) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, label };
}

/**
 * POST /api/v1/people/merge: fold 1-50 source clusters into one
 * target. The whole merge is one server transaction; the sources
 * dissolve WITH their label rows (a source's name dies -- the
 * honest documented cost), the target's label persists.
 */
export async function mergePeople(
	sourceClusterIds: string[],
	targetClusterId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<MergeResult> {
	let response: Response;
	try {
		response = await apiFetch(
			'/api/v1/people/merge',
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ sourceClusterIds, targetClusterId })
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
	if (body === null || typeof body !== 'object') {
		return { ok: false, errorKey: 'internal' };
	}
	const candidate = body as Partial<MergePeopleResponse>;
	if (typeof candidate.survivingClusterId !== 'string' || !Array.isArray(candidate.mergedClusterIds)) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, merge: body as MergePeopleResponse };
}

/**
 * POST /api/v1/faces/{faceId}/move: move one face onto an existing
 * cluster. A move onto the face's current cluster is an accepted
 * no-op; a move that empties the source cluster dissolves it (with
 * its labels -- the same honest cost as a merge).
 */
export async function moveFace(
	faceId: string,
	targetClusterId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<MoveResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/faces/${encodeURIComponent(faceId)}/move`,
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ targetClusterId })
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
	if (body === null || typeof body !== 'object') {
		return { ok: false, errorKey: 'internal' };
	}
	const candidate = body as Partial<MoveFaceResponse>;
	if (typeof candidate.faceId !== 'string' || typeof candidate.clusterId !== 'string') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, move: body as MoveFaceResponse };
}

/**
 * POST /api/v1/faces/{faceId}/cluster: the from-face person creation.
 * The face -- clustered or not -- becomes the sole member AND the
 * representative of a brand-new cluster; a repeat call on a
 * clustered face is another CREATE, never a no-op, and the vacated
 * source cluster may dissolve with its label rows (the server's
 * honest cost). Refusals: an unknown face, a face on an asset the
 * caller does not own, or a trashed, hidden or component asset --
 * the same byte-identical 404 as the move (anti-enumeration).
 */
export async function clusterFace(
	faceId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<ClusterFaceResult> {
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/faces/${encodeURIComponent(faceId)}/cluster`, { method: 'POST' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await errorKeyFrom(response) };
	}
	const body = await bodyOf(response);
	if (body === null || typeof body !== 'object') {
		return { ok: false, errorKey: 'internal' };
	}
	const candidate = body as Partial<ClusterFaceResponse>;
	if (typeof candidate.faceId !== 'string' || typeof candidate.clusterId !== 'string') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, cluster: body as ClusterFaceResponse };
}

/**
 * DELETE /api/v1/faces/{faceId}: the single-face detach. The face's
 * cluster assignment goes null -- back to the not-yet-grouped pool
 * it came from; the face row survives (it still lists among its
 * asset's faces, now carrying no person) and any later assign write
 * can re-home it. The source dissolves only when the detach emptied
 * it (the label cascade, the same honest cost a merge pays).
 * Refusals: the move's own matrix plus the NULL-cluster rule -- a
 * face part of no person answers the same 404.
 */
export async function detachFace(faceId: string, fetcher: typeof fetch = globalThis.fetch): Promise<MutationResult> {
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/faces/${encodeURIComponent(faceId)}`, { method: 'DELETE' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.ok) {
		return { ok: true };
	}
	return { ok: false, errorKey: await errorKeyFrom(response) };
}
