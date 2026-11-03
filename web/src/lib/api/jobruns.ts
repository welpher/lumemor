// The task panel's data client (M3a Task 5): the job runs listing
// behind GET /admin/jobs, polled by the panel's fallback cadence and
// converged by the SSE job.* events.
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased
// (auth_forbidden for a signed-in non-admin), 'internal' for a
// network failure or an unparseable body, and 'session_expired' for
// a 401.
//
// The 200 body is structurally validated: a drifted or proxied
// answer must never render as a broken surface -- it maps to the
// generic failure like any other unparseable body.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One job run row, exactly the contract's JobRunResponse shape. */
export type JobRun = components['schemas']['JobRunResponse'];

/** The run kinds of the contract's JobRunKind enum. */
const JOB_RUN_KINDS: readonly string[] = [
	'model_download',
	'face_detect_rerun',
	'face_recognize_rerun',
	'clip_embed_rerun',
	'ocr_rerun'
];

/** The run lifecycle states of the contract's JobRunState enum. */
const JOB_RUN_STATES: readonly string[] = ['queued', 'running', 'succeeded', 'failed'];

/** The listing's outcome: the runs newest first, or the failure's i18n key. */
export type JobRunListResult =
	| { ok: true; jobs: JobRun[] }
	| { ok: false; errorKey: string };

/** Structurally validate one run body; null when it is not one. */
export function jobRunFrom(body: unknown): JobRun | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<JobRun>;
	if (typeof candidate.id !== 'string' || candidate.id === '') {
		return null;
	}
	if (typeof candidate.kind !== 'string' || !JOB_RUN_KINDS.includes(candidate.kind)) {
		return null;
	}
	if (typeof candidate.state !== 'string' || !JOB_RUN_STATES.includes(candidate.state)) {
		return null;
	}
	if (
		typeof candidate.total !== 'number' ||
		typeof candidate.done !== 'number' ||
		typeof candidate.failed !== 'number'
	) {
		return null;
	}
	if (candidate.startedAt !== null && typeof candidate.startedAt !== 'string') {
		return null;
	}
	if (candidate.finishedAt !== null && typeof candidate.finishedAt !== 'string') {
		return null;
	}
	return body as JobRun;
}

/**
 * GET /api/v1/admin/jobs: the instance's job runs, newest first,
 * bounded by the limit (the contract's default applies when omitted).
 * A 403 is the signed-in non-admin caller (auth_forbidden), not a
 * session failure.
 */
export async function listJobRuns(
	limit?: number,
	fetcher: typeof fetch = globalThis.fetch
): Promise<JobRunListResult> {
	const path =
		limit === undefined
			? '/api/v1/admin/jobs'
			: `/api/v1/admin/jobs?limit=${encodeURIComponent(String(limit))}`;
	let response: Response;
	try {
		response = await apiFetch(path, undefined, fetcher);
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
	if (body === null || typeof body !== 'object' || !Array.isArray((body as { jobs?: unknown }).jobs)) {
		return { ok: false, errorKey: 'internal' };
	}
	const jobs: JobRun[] = [];
	for (const entry of (body as { jobs: unknown[] }).jobs) {
		const job = jobRunFrom(entry);
		if (job === null) {
			return { ok: false, errorKey: 'internal' };
		}
		jobs.push(job);
	}
	return { ok: true, jobs };
}
