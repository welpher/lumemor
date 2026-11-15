// The tags surface's data client: the vocabulary
// listing, the idempotent create and the batch assignment add --
// the three reads and writes the web's tag faces need (the search
// facet's vocabulary is the search response's own; this module
// serves the add-tag dialog and the info drawer's tags row). Each
// call rides the repo-wide error-code convention: every failure
// outcome carries the error code's i18n key (the envelope code
// lowercased, 'internal' for a network failure or an unparseable
// body, 'session_expired' for a 401).
//
// Per-user discipline (tags are per-user state): the listing is always
// the CALLER's vocabulary, the create converges on the caller's own
// row, and the assignment add writes the caller's own rows -- no
// other account's tag ever surfaces through this client.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One tag of the caller's vocabulary, exactly the TagSummary shape. */
export type TagSummary = components['schemas']['TagSummary'];

/** The vocabulary listing's outcome. */
export type TagListResult = { ok: true; tags: TagSummary[] } | { ok: false; errorKey: string };

/** The create's outcome (the response IS the tag, fresh or existing). */
export type TagCreateResult = { ok: true; tag: TagSummary } | { ok: false; errorKey: string };

/** The honest report of one assignment add. */
export type TagAssignReport = { added: number; skipped: number };

/** The assignment add's outcome. */
export type TagAssignResult = { ok: true; report: TagAssignReport } | { ok: false; errorKey: string };

/** The untag half's report: the assignments that left, the ids that had none. */
export type TagRemoveReport = { removed: number; skipped: number };

/** The assignment remove's outcome. */
export type TagRemoveResult = { ok: true; report: TagRemoveReport } | { ok: false; errorKey: string };

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

/** One tag summary, structurally validated; null when not one. */
function tagFrom(body: unknown): TagSummary | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<TagSummary>;
	if (typeof candidate.id !== 'string' || typeof candidate.name !== 'string') {
		return null;
	}
	if (
		typeof candidate.assetCount !== 'number' ||
		!Number.isInteger(candidate.assetCount) ||
		candidate.assetCount < 0
	) {
		return null;
	}
	if (typeof candidate.createdAt !== 'string' || typeof candidate.updatedAt !== 'string') {
		return null;
	}
	return body as TagSummary;
}

/**
 * GET /api/v1/tags: the caller's whole vocabulary, name-sorted by
 * the server. Unpaginated (a family instance's vocabulary is a
 * bounded set, one document).
 */
export async function listTags(fetcher: typeof fetch = globalThis.fetch): Promise<TagListResult> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/tags', undefined, fetcher);
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
	const tags: TagSummary[] = [];
	for (const entry of body) {
		const tag = tagFrom(entry);
		if (tag === null) {
			return { ok: false, errorKey: 'internal' };
		}
		tags.push(tag);
	}
	return { ok: true, tags };
}

/**
 * POST /api/v1/tags: create one tag, idempotently. A fresh name
 * answers 201 with the new row; a name the caller already owns
 * answers 200 with the EXISTING tag (nothing written) -- both are
 * the same outcome here: the tag the caller may now assign. The
 * name is trimmed server-side; the client passes it through as
 * typed (the dialog pre-trims).
 */
export async function createTag(name: string, fetcher: typeof fetch = globalThis.fetch): Promise<TagCreateResult> {
	let response: Response;
	try {
		response = await apiFetch(
			'/api/v1/tags',
			{
				method: 'POST',
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
	const tag = tagFrom(await bodyOf(response));
	if (tag === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, tag };
}

/**
 * POST /api/v1/tags/{tagId}/assets: add the caller's tag to a batch
 * of asset ids (1-200). The answer is the honest report over the
 * request's distinct ids -- `added` counts the assignment rows the
 * request inserted, `skipped` the rest (already-tagged ids and ids
 * the caller cannot see alike); an all-skipped request is still a
 * 200, never an error.
 */
export async function addTagAssets(
	tagId: string,
	assetIds: string[],
	fetcher: typeof fetch = globalThis.fetch
): Promise<TagAssignResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/tags/${encodeURIComponent(tagId)}/assets`,
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
	if (body === null || typeof body !== 'object') {
		return { ok: false, errorKey: 'internal' };
	}
	const report = body as { added?: unknown; skipped?: unknown };
	if (
		typeof report.added !== 'number' ||
		!Number.isInteger(report.added) ||
		report.added < 0 ||
		typeof report.skipped !== 'number' ||
		!Number.isInteger(report.skipped) ||
		report.skipped < 0
	) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, report: { added: report.added, skipped: report.skipped } };
}

/**
 * DELETE /api/v1/tags/{tagId}/assets over explicit ids: the untag
 * half of the same batch surface, `{removed, skipped}`. The caller's
 * own assignment rows only; an id without the assignment is skipped
 * (never was, or already removed). The viewer's chip-remove button
 * drives it over the one open asset; the convergence is the caller's
 * detail refetch (the tags array IS the row's truth).
 */
export async function removeTagAssets(
	tagId: string,
	assetIds: string[],
	fetcher: typeof fetch = globalThis.fetch
): Promise<TagRemoveResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/tags/${encodeURIComponent(tagId)}/assets`,
			{
				method: 'DELETE',
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
	if (body === null || typeof body !== 'object') {
		return { ok: false, errorKey: 'internal' };
	}
	const report = body as { removed?: unknown; skipped?: unknown };
	if (
		typeof report.removed !== 'number' ||
		!Number.isInteger(report.removed) ||
		report.removed < 0 ||
		typeof report.skipped !== 'number' ||
		!Number.isInteger(report.skipped) ||
		report.skipped < 0
	) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, report: { removed: report.removed, skipped: report.skipped } };
}
