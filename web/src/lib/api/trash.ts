// The trash view's data client: the bounded listing
// behind GET /api/v1/trash and the single restore behind POST
// /trash/{id}/restore. The permanent delete reuses the viewer
// client's deleteAsset (DELETE /assets/{id}, 404 = already gone =
// success), so it lives in $lib/api/assets, not here.
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased,
// 'internal' for a network failure or an unparseable body, and
// 'session_expired' for a 401 (the interceptor has already flipped
// the session store; the view surfaces the session message, never
// login-credential text).
//
// THE PINNED RESTORE CHOICE: the view restores through the
// single-asset endpoint, not the batch surface -- one row, one
// click, one request; the batch API stays the selection bar's tool.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One trashed asset, exactly the contract's TrashItem shape. */
export type TrashItem = components['schemas']['TrashItem'];

/** The listing's outcome: the bounded items, or the failure's i18n key. */
export type TrashListResult = { ok: true; items: TrashItem[] } | { ok: false; errorKey: string };

/** Structurally validate a listing body; null when it is not one. */
function trashListFrom(body: unknown): TrashItem[] | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const { items } = body as { items?: unknown };
	if (!Array.isArray(items)) {
		return null;
	}
	for (const entry of items) {
		if (entry === null || typeof entry !== 'object') {
			return null;
		}
		const candidate = entry as Partial<TrashItem>;
		if (typeof candidate.id !== 'string' || typeof candidate.remainingDays !== 'number') {
			return null;
		}
	}
	return items as TrashItem[];
}

/**
 * GET /api/v1/trash: the caller's trashed assets, newest deletion
 * first, bounded (the contract caps the list at 200 -- there is no
 * pagination).
 */
export async function fetchTrash(fetcher: typeof fetch = globalThis.fetch): Promise<TrashListResult> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/trash', undefined, fetcher);
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
	const items = trashListFrom(body);
	if (items === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, items };
}

/**
 * POST /api/v1/trash/{id}/restore: return one trashed asset to the
 * live views. The outcome is a plain mutation (204, no body). A 404
 * means the row is already gone (purged by the expiry job) or was
 * restored-then-purged -- either way it is no longer in the trash,
 * so the caller treats it as success and drops the row.
 */
export async function restoreTrashedAsset(
	id: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<{ ok: true } | { ok: false; errorKey: string }> {
	let response: Response;
	try {
		response = await apiFetch(`/api/v1/trash/${encodeURIComponent(id)}/restore`, { method: 'POST' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { ok: false, errorKey: 'session_expired' };
	}
	if (response.ok || response.status === 404) {
		// 404: the row left the trash already (the expiry job's
		// purge) -- the same already-cleaned reading the permanent
		// delete pins.
		return { ok: true };
	}
	const envelope = await errorEnvelope(response);
	return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
}
