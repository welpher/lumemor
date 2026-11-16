// The trash view's data client: the bounded listing behind GET
// /api/v1/trash (its retentionDays rides along -- the note's number,
// the same setting the countdowns use), the single restore behind
// POST /trash/{id}/restore, the whole-trash pair (restore-all,
// empty) and the multi-select batch. The permanent delete of one row
// reuses the viewer client's deleteAsset (DELETE /assets/{id}, 404 =
// already gone = success), so it lives in $lib/api/assets, not here.
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased,
// 'internal' for a network failure or an unparseable body, and
// 'session_expired' for a 401 (the interceptor has already flipped
// the session store; the view surfaces the session message, never
// login-credential text).
//
// THE PINNED RESTORE CHOICE: a ROW restores through the single-asset
// endpoint, not the batch surface -- one row, one click, one
// request. The batch API is the selection bar's tool and the sweeps
// are the toolbar's; the row-level actions stay untouched by both.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One trashed asset, exactly the contract's TrashItem shape. */
export type TrashItem = components['schemas']['TrashItem'];

/** The listing's outcome: the bounded items with the note's number, or the failure's i18n key. */
export type TrashListResult = { ok: true; items: TrashItem[]; retentionDays: number } | { ok: false; errorKey: string };

/** Structurally validate a listing body; null when it is not one. */
function trashListFrom(body: unknown): { items: TrashItem[]; retentionDays: number } | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const { items, retentionDays } = body as { items?: unknown; retentionDays?: unknown };
	if (!Array.isArray(items) || typeof retentionDays !== 'number') {
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
	return { items: items as TrashItem[], retentionDays };
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
	return { ok: true, items: items.items, retentionDays: items.retentionDays };
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

/** The outcome of a whole-trash sweep or a trash batch: the report's counts, or the failure's i18n key. */
export type TrashActionOutcome = { ok: true; affected: number } | { ok: false; errorKey: string };

/** The outcome of a trash batch: the counts plus the ids that were not in the trash, or the failure's i18n key. */
export type TrashBatchOutcome = { ok: true; affected: number; missing: string[] } | { ok: false; errorKey: string };

/**
 * The mutation POSTs' shared shape: 200 with the report body, the
 * session's 401, anything else an envelope failure. The sweeps and
 * the batch have no not-found surface (the server scans what is in
 * the trash at the time of the request), so unlike the single-row
 * restore there is no 404-as-success reading here -- a failure is a
 * failure the page surfaces.
 */
async function postTrashReport(
	path: string,
	body?: unknown,
	fetcher: typeof fetch = globalThis.fetch
): Promise<{ status: number; affected: number; missing: string[]; errorKey: string }> {
	let response: Response;
	try {
		response = await apiFetch(
			path,
			body === undefined
				? { method: 'POST' }
				: {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify(body)
					},
			fetcher
		);
	} catch {
		return { status: 0, affected: 0, missing: [], errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { status: 401, affected: 0, missing: [], errorKey: 'session_expired' };
	}
	if (!response.ok) {
		const envelope = await errorEnvelope(response);
		return {
			status: response.status,
			affected: 0,
			missing: [],
			errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal'
		};
	}
	let report: unknown;
	try {
		report = await response.json();
	} catch {
		return { status: response.status, affected: 0, missing: [], errorKey: 'internal' };
	}
	if (
		report === null ||
		typeof report !== 'object' ||
		typeof (report as { affected?: unknown }).affected !== 'number'
	) {
		return { status: response.status, affected: 0, missing: [], errorKey: 'internal' };
	}
	const missing = (report as { missing?: unknown }).missing;
	return {
		status: response.status,
		affected: (report as { affected: number }).affected,
		missing:
			Array.isArray(missing) && missing.every((entry) => typeof entry === 'string') ? (missing as string[]) : [],
		errorKey: ''
	};
}

/**
 * POST /api/v1/trash/restore-all: the whole-trash restore. Every row
 * of the caller's trash returns to the live views; an empty trash
 * answers 0 (the idempotent no-op -- there is no not-found surface,
 * the server scans what is in the trash right now).
 */
export async function restoreAllTrash(fetcher: typeof fetch = globalThis.fetch): Promise<TrashActionOutcome> {
	const report = await postTrashReport('/api/v1/trash/restore-all', undefined, fetcher);
	if (report.errorKey !== '') {
		return { ok: false, errorKey: report.errorKey };
	}
	return { ok: true, affected: report.affected };
}

/**
 * POST /api/v1/trash/empty: the whole-trash delete. The caller's
 * client confirms first (the irreversible action is never one
 * accidental click away); the endpoint itself is the caller's own
 * trash alone, idempotent like the restore-all.
 */
export async function emptyTrash(fetcher: typeof fetch = globalThis.fetch): Promise<TrashActionOutcome> {
	const report = await postTrashReport('/api/v1/trash/empty', undefined, fetcher);
	if (report.errorKey !== '') {
		return { ok: false, errorKey: report.errorKey };
	}
	return { ok: true, affected: report.affected };
}

/**
 * POST /api/v1/trash/batch: the multi-select surface, one action
 * over a list of ids IN THE CALLER'S TRASH. The report partitions
 * the ids into affected and missing -- an id that was not in the
 * trash at the time (another account's, unknown, already live or
 * already purged) lands in missing; the caller hands the ids over
 * themselves, so the report hides nothing.
 */
export async function batchTrash(
	action: 'restore' | 'delete',
	ids: readonly string[],
	fetcher: typeof fetch = globalThis.fetch
): Promise<TrashBatchOutcome> {
	const report = await postTrashReport('/api/v1/trash/batch', { action, ids }, fetcher);
	if (report.errorKey !== '') {
		return { ok: false, errorKey: report.errorKey };
	}
	return { ok: true, affected: report.affected, missing: report.missing };
}
