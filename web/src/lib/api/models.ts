// The admin model management client (M3a Task 5): the listing behind
// GET /admin/models (with the download run summary), the download
// trigger behind POST /admin/models/download, the activation switch
// behind PATCH /admin/models/{id}, the rerun trigger behind POST
// /admin/models/{id}/rerun, and the custom scan behind POST
// /admin/models/scan.
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased
// (model_not_found, model_invalid, model_download_failed,
// auth_forbidden for a signed-in non-admin), 'internal' for a network
// failure or an unparseable body, and 'session_expired' for a 401.
//
// The 2xx bodies are structurally validated: a drifted or proxied
// answer must never render as a broken surface -- it maps to the
// generic failure like any other unparseable body.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import { jobRunFrom, type JobRun } from '$lib/api/jobruns';
import type { components } from '../../../../openapi/generated/web/api';

/** One registered model, exactly the contract's ModelResponse shape. */
export type AdminModel = components['schemas']['ModelResponse'];

/** The functional slots a model fills, the contract's ModelSlot enum. */
export type ModelSlotId = components['schemas']['ModelSlot'];

// The clip_text slot is the search's semantic arm (the CLIP text
// tower); it rides the same contract enum as the pipeline's slots.
const MODEL_SLOTS: readonly string[] = [
	'face_detect',
	'face_recognize',
	'clip_embed',
	'clip_text',
	'ocr'
];

const MODEL_SOURCES: readonly string[] = ['downloaded', 'custom'];

const MODELS_ENDPOINT = '/api/v1/admin/models';

/** The listing's outcome: every model plus the download summary, or the failure's i18n key. */
export type ModelListResult =
	| { ok: true; models: AdminModel[]; download: JobRun | null }
	| { ok: false; errorKey: string };

/** The download trigger's outcome: the joined or started run's id, or the failure's i18n key. */
export type ModelDownloadResult =
	| { ok: true; jobRunId: string }
	| { ok: false; errorKey: string };

/** The activation switch's outcome, with the destructive arm's facts when it ran. */
export type ModelActivateResult =
	| {
			ok: true;
			activated: { id: string; slot: ModelSlotId };
			dimensionChanged?: { from: number; to: number };
			wipedEmbeddings?: number;
	  }
	| { ok: false; errorKey: string };

/** The rerun trigger's outcome: the joined or started run's id, or the failure's i18n key. */
export type ModelRerunResult =
	| { ok: true; jobRunId: string }
	| { ok: false; errorKey: string };

/** The custom scan's outcome: the models the scan registered, or the failure's i18n key. */
export type ModelScanResult =
	| { ok: true; registered: AdminModel[] }
	| { ok: false; errorKey: string };

/** Structurally validate one model body; null when it is not one. */
function modelFrom(body: unknown): AdminModel | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<AdminModel>;
	if (typeof candidate.id !== 'string' || candidate.id === '') {
		return null;
	}
	if (typeof candidate.slot !== 'string' || !MODEL_SLOTS.includes(candidate.slot)) {
		return null;
	}
	if (typeof candidate.version !== 'string' || typeof candidate.license !== 'string') {
		return null;
	}
	if (typeof candidate.source !== 'string' || !MODEL_SOURCES.includes(candidate.source)) {
		return null;
	}
	if (typeof candidate.isActive !== 'boolean' || typeof candidate.filesPresent !== 'boolean') {
		return null;
	}
	if (candidate.notice !== null && typeof candidate.notice !== 'string') {
		return null;
	}
	if (candidate.dim !== null && typeof candidate.dim !== 'number') {
		return null;
	}
	if (
		candidate.inputSize !== null &&
		(typeof candidate.inputSize !== 'object' ||
			typeof candidate.inputSize.width !== 'number' ||
			typeof candidate.inputSize.height !== 'number')
	) {
		return null;
	}
	if (candidate.params === null || typeof candidate.params !== 'object') {
		return null;
	}
	return body as AdminModel;
}

/** Map a non-OK response onto the failure's i18n key. */
async function modelErrorKey(response: Response): Promise<string> {
	if (response.status === 401) {
		return 'session_expired';
	}
	const envelope = await errorEnvelope(response);
	return envelope !== null ? envelope.code.toLowerCase() : 'internal';
}

/** Read and validate a 2xx body, or the failure outcome for it. */
async function readJsonBody<T>(response: Response, validate: (body: unknown) => T | null): Promise<{ ok: true; value: T } | { ok: false; errorKey: string }> {
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const value = validate(body);
	if (value === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, value };
}

/**
 * GET /api/v1/admin/models: every registered model of the instance
 * plus the most recent model download run as the progress summary.
 * A 403 is the signed-in non-admin caller (auth_forbidden), not a
 * session failure.
 */
export async function listModels(
	fetcher: typeof fetch = globalThis.fetch
): Promise<ModelListResult> {
	let response: Response;
	try {
		response = await apiFetch(MODELS_ENDPOINT, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	const parsed = await readJsonBody(response, (body): { models: AdminModel[]; download: JobRun | null } | null => {
		if (body === null || typeof body !== 'object' || !Array.isArray((body as { models?: unknown }).models)) {
			return null;
		}
		const models: AdminModel[] = [];
		for (const entry of (body as { models: unknown[] }).models) {
			const model = modelFrom(entry);
			if (model === null) {
				return null;
			}
			models.push(model);
		}
		const downloadBody = (body as { download?: unknown }).download;
		if (downloadBody === undefined) {
			return null;
		}
		// A null download is the honest "no run yet" answer; a present
		// body must validate as one run row.
		const download = downloadBody === null ? null : jobRunFrom(downloadBody);
		if (downloadBody !== null && download === null) {
			return null;
		}
		return { models, download };
	});
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	return { ok: true, models: parsed.value.models, download: parsed.value.download };
}

/**
 * POST /admin/models/download: start (or join) the default model
 * group's download. The endpoint is idempotent -- an in-flight or
 * most recent run answers 200 with its identifier, a first call 202
 * -- so both statuses are the same outcome for the caller.
 */
export async function startModelDownload(
	fetcher: typeof fetch = globalThis.fetch
): Promise<ModelDownloadResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`${MODELS_ENDPOINT}/download`,
			{ method: 'POST' },
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	const parsed = await readJsonBody(response, (body): string | null => {
		if (body === null || typeof body !== 'object') {
			return null;
		}
		const jobRunId = (body as { jobRunId?: unknown }).jobRunId;
		return typeof jobRunId === 'string' && jobRunId !== '' ? jobRunId : null;
	});
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	return { ok: true, jobRunId: parsed.value };
}

/**
 * PATCH /admin/models/{id}: the activation switch (the model becomes
 * the active one of its slot). The 200 body reports the switch's
 * facts: a same-dimension switch carries only the activated model;
 * a cross-dimension clip switch additionally carries
 * dimensionChanged and wipedEmbeddings -- the destructive arm the
 * confirm dialog warns about.
 */
export async function activateModel(
	id: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<ModelActivateResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`${MODELS_ENDPOINT}/${encodeURIComponent(id)}`,
			{
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({})
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	const parsed = await readJsonBody(response, (body): {
		activated: { id: string; slot: ModelSlotId };
		dimensionChanged?: { from: number; to: number };
		wipedEmbeddings?: number;
	} | null => {
		if (body === null || typeof body !== 'object') {
			return null;
		}
		const activated = (body as { activated?: unknown }).activated;
		if (
			activated === null ||
			typeof activated !== 'object' ||
			typeof (activated as { id?: unknown }).id !== 'string' ||
			typeof (activated as { slot?: unknown }).slot !== 'string' ||
			!MODEL_SLOTS.includes((activated as { slot: string }).slot)
		) {
			return null;
		}
		const result: {
			activated: { id: string; slot: ModelSlotId };
			dimensionChanged?: { from: number; to: number };
			wipedEmbeddings?: number;
		} = {
			activated: activated as { id: string; slot: ModelSlotId }
		};
		const dimensionChanged = (body as { dimensionChanged?: unknown }).dimensionChanged;
		if (dimensionChanged !== undefined) {
			if (
				dimensionChanged === null ||
				typeof dimensionChanged !== 'object' ||
				typeof (dimensionChanged as { from?: unknown }).from !== 'number' ||
				typeof (dimensionChanged as { to?: unknown }).to !== 'number'
			) {
				return null;
			}
			result.dimensionChanged = dimensionChanged as { from: number; to: number };
		}
		const wipedEmbeddings = (body as { wipedEmbeddings?: unknown }).wipedEmbeddings;
		if (wipedEmbeddings !== undefined) {
			if (typeof wipedEmbeddings !== 'number') {
				return null;
			}
			result.wipedEmbeddings = wipedEmbeddings;
		}
		return result;
	});
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	return { ok: true, ...parsed.value };
}

/**
 * POST /admin/models/{id}/rerun: one run that redoes the ACTIVE
 * model's slot over the whole library. In-flight idempotence: a run
 * of the slot's kind still queued or running answers 200 with its
 * identifier, a first call 202 -- the same outcome for the caller.
 */
export async function rerunModel(
	id: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<ModelRerunResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`${MODELS_ENDPOINT}/${encodeURIComponent(id)}/rerun`,
			{ method: 'POST' },
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	const parsed = await readJsonBody(response, (body): string | null => {
		if (body === null || typeof body !== 'object') {
			return null;
		}
		const jobRunId = (body as { jobRunId?: unknown }).jobRunId;
		return typeof jobRunId === 'string' && jobRunId !== '' ? jobRunId : null;
	});
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	return { ok: true, jobRunId: parsed.value };
}

/**
 * POST /admin/models/scan: register the custom models found under
 * the server's models dir (verified against their pinned digests,
 * never activated). The 200 body reports what the scan registered.
 */
export async function scanModels(
	fetcher: typeof fetch = globalThis.fetch
): Promise<ModelScanResult> {
	let response: Response;
	try {
		response = await apiFetch(`${MODELS_ENDPOINT}/scan`, { method: 'POST' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	const parsed = await readJsonBody(response, (body): AdminModel[] | null => {
		if (body === null || typeof body !== 'object' || !Array.isArray((body as { registered?: unknown }).registered)) {
			return null;
		}
		const models: AdminModel[] = [];
		for (const entry of (body as { registered: unknown[] }).registered) {
			const model = modelFrom(entry);
			if (model === null) {
				return null;
			}
			models.push(model);
		}
		return models;
	});
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	return { ok: true, registered: parsed.value };
}
