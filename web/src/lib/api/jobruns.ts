// The task panel's data client: the job runs listing
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
	'ocr_rerun',
	'places_rerun',
	'album_rescan',
	'album_rescan_all',
	'rebuild_derivatives'
];

/** The run lifecycle states of the contract's JobRunState enum. */
const JOB_RUN_STATES: readonly string[] = ['queued', 'running', 'succeeded', 'failed', 'cancelled'];

/** The listing's outcome: the runs newest first, or the failure's i18n key. */
export type JobRunListResult = { ok: true; jobs: JobRun[] } | { ok: false; errorKey: string };

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
 * POST /api/v1/admin/jobs/{runId}/cancel: ask the run to end at its
 * next chunk boundary (E41). The 204 is the flag's whole receipt --
 * the driver's close and its finished event own the row's
 * convergence. A finished run answers 409 job_not_cancellable, an
 * unknown id 404 job_run_not_found.
 */
export async function cancelJobRun(
	runId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<{ ok: true } | { ok: false; errorKey: string }> {
	let response: Response;
	try {
		response = await apiFetch(
			`/api/v1/admin/jobs/${encodeURIComponent(runId)}/cancel`,
			{ method: 'POST' },
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { ok: false, errorKey: 'session_expired' };
	}
	if (response.status === 204) {
		return { ok: true };
	}
	const envelope = await errorEnvelope(response);
	return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
}

/**
 * GET /api/v1/admin/jobs: the instance's job runs, newest first,
 * bounded by the limit (the contract's default applies when omitted).
 * A 403 is the signed-in non-admin caller (auth_forbidden), not a
 * session failure.
 */
export async function listJobRuns(limit?: number, fetcher: typeof fetch = globalThis.fetch): Promise<JobRunListResult> {
	const path =
		limit === undefined ? '/api/v1/admin/jobs' : `/api/v1/admin/jobs?limit=${encodeURIComponent(String(limit))}`;
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

/** The task catalog's one read, exactly the contract's JobsSummaryResponse shape. */
export type JobsSummary = components['schemas']['JobsSummaryResponse'];

/** The summary read's outcome: the catalog's state, or the failure's i18n key. */
export type JobsSummaryResult = { ok: true; summary: JobsSummary } | { ok: false; errorKey: string };

/** The catalog's pipeline keys, in the contract's order. */
const PIPELINE_KEYS = [
	'thumbnail',
	'metadata',
	'transcode',
	'faceDetect',
	'faceRecognize',
	'clipEmbed',
	'ocr'
] as const;

/** The catalog's batch-run slots, in the contract's order. */
const RUN_KEYS = ['rebuildDerivatives', 'placesRerun', 'albumRescanAll', 'folderSync', 'modelDownload'] as const;

/** One catalog count triple: three numbers, or null. */
function pipelineCountsFrom(body: unknown): boolean {
	if (body === null || typeof body !== 'object') {
		return false;
	}
	const counts = body as { waiting?: unknown; active?: unknown; failed?: unknown };
	return typeof counts.waiting === 'number' && typeof counts.active === 'number' && typeof counts.failed === 'number';
}

/** One catalog run summary: the contract's JobRunSummary shape, or null. */
function runSummaryFrom(body: unknown): boolean {
	if (body === null) {
		return true;
	}
	if (typeof body !== 'object') {
		return false;
	}
	const run = body as { id?: unknown; state?: unknown; total?: unknown; done?: unknown; failed?: unknown };
	if (typeof run.id !== 'string' || run.id === '') {
		return false;
	}
	if (typeof run.state !== 'string' || !JOB_RUN_STATES.includes(run.state)) {
		return false;
	}
	return typeof run.total === 'number' && typeof run.done === 'number' && typeof run.failed === 'number';
}

/** Structurally validate one summary body; null when it is not one. */
export function jobsSummaryFrom(body: unknown): JobsSummary | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as {
		pipeline?: Record<string, unknown>;
		paused?: Record<string, unknown>;
		runs?: Record<string, unknown>;
	};
	if (candidate.pipeline === undefined || candidate.paused === undefined || candidate.runs === undefined) {
		return null;
	}
	for (const key of PIPELINE_KEYS) {
		if (!pipelineCountsFrom(candidate.pipeline[key])) {
			return null;
		}
	}
	for (const key of ['thumbnail', 'metadata', 'transcode', 'ml', 'places'] as const) {
		if (typeof candidate.paused[key] !== 'boolean') {
			return null;
		}
	}
	for (const key of RUN_KEYS) {
		if (!runSummaryFrom(candidate.runs[key])) {
			return null;
		}
	}
	return body as JobsSummary;
}

/**
 * GET /api/v1/admin/jobs/summary: the fixed task catalog's state --
 * the queue's per-kind counts, the budget's pause switches and the
 * latest run of each batch kind. A 403 is the signed-in non-admin
 * caller (auth_forbidden), not a session failure.
 */
export async function getJobsSummary(fetcher: typeof fetch = globalThis.fetch): Promise<JobsSummaryResult> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/admin/jobs/summary', undefined, fetcher);
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
	const summary = jobsSummaryFrom(body);
	if (summary === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, summary };
}

/**
 * POST /api/v1/admin/albums/rescan: start (or join) the instance-wide
 * album rescan sweep. A 202 carries the fresh run, a 200 the
 * in-flight sweep the request joined -- both are the same outcome for
 * the caller; the run's own row carries its state from there.
 */
export async function rescanAllAlbums(
	fetcher: typeof fetch = globalThis.fetch
): Promise<{ ok: true; jobRunId: string } | { ok: false; errorKey: string }> {
	let response: Response;
	try {
		response = await apiFetch('/api/v1/admin/albums/rescan', { method: 'POST' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.status === 401) {
		return { ok: false, errorKey: 'session_expired' };
	}
	if (response.status !== 200 && response.status !== 202) {
		const envelope = await errorEnvelope(response);
		return { ok: false, errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal' };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const jobRunId = (body as { jobRunId?: unknown } | null)?.jobRunId;
	if (typeof jobRunId !== 'string' || jobRunId === '') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, jobRunId };
}
