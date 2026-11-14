// The admin model management client: the listing behind
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

/** One slot's four-state row, the contract's SlotStateResponse. */
export type SlotState = components['schemas']['SlotStateResponse'];

/** One official-list entry, the contract's ModelCatalogEntry. */
export type CatalogEntry = components['schemas']['ModelCatalogEntry'];

/** The contract's four slot states. */
export type SlotStateId = SlotState['state'];

/** One probe's state row, the contract's ModelProbeResponse. */
export type ModelProbe = components['schemas']['ModelProbeResponse'];

/** The switch answer's shape, the contract's ModelSourceSwitchResponse. */
export type SourceSwitchResult = components['schemas']['ModelSourceSwitchResponse'];

// The clip_text slot is the search's semantic arm (the CLIP text
// tower); it rides the same contract enum as the pipeline's slots.
const MODEL_SLOTS: readonly string[] = ['face_detect', 'face_recognize', 'clip_embed', 'clip_text', 'ocr'];

const MODEL_SOURCES: readonly string[] = ['downloaded', 'custom'];

const MODELS_ENDPOINT = '/api/v1/admin/models';

/** The listing's outcome: every model plus the download summary, or the failure's i18n key. */
export type ModelListResult =
	| {
			ok: true;
			models: AdminModel[];
			download: JobRun | null;
			slots: SlotState[];
			official: CatalogEntry[];
	  }
	| { ok: false; errorKey: string };

/** The download trigger's outcome: the joined or started run's id, or the failure's i18n key. */
export type ModelDownloadResult = { ok: true; jobRunId: string } | { ok: false; errorKey: string };

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
export type ModelRerunResult = { ok: true; jobRunId: string } | { ok: false; errorKey: string };

/** The custom scan's outcome: the models the scan registered, or the failure's i18n key. */
export type ModelScanResult = { ok: true; registered: AdminModel[] } | { ok: false; errorKey: string };

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
async function readJsonBody<T>(
	response: Response,
	validate: (body: unknown) => T | null
): Promise<{ ok: true; value: T } | { ok: false; errorKey: string }> {
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
export async function listModels(fetcher: typeof fetch = globalThis.fetch): Promise<ModelListResult> {
	let response: Response;
	try {
		response = await apiFetch(MODELS_ENDPOINT, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	const parsed = await readJsonBody(
		response,
		(
			body
		): {
			models: AdminModel[];
			download: JobRun | null;
			slots: SlotState[];
			official: CatalogEntry[];
		} | null => {
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
			const slots = slotStatesFrom((body as { slots?: unknown }).slots);
			if (slots === null) {
				return null;
			}
			const official = catalogFrom((body as { official?: unknown }).official);
			if (official === null) {
				return null;
			}
			return { models, download, slots, official };
		}
	);
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	return {
		ok: true,
		models: parsed.value.models,
		download: parsed.value.download,
		slots: parsed.value.slots,
		official: parsed.value.official
	};
}

const SLOT_STATES: readonly string[] = ['active', 'downloading', 'not_downloaded', 'failed'];
const SLOT_SOURCE_KINDS: readonly string[] = ['official', 'custom'];

/** Structurally validate one slot state row; null when it is not one. */
function slotStateFrom(body: unknown): SlotState | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<SlotState>;
	if (typeof candidate.slot !== 'string' || !MODEL_SLOTS.includes(candidate.slot)) {
		return null;
	}
	if (typeof candidate.state !== 'string' || !SLOT_STATES.includes(candidate.state)) {
		return null;
	}
	if (candidate.modelId !== null && typeof candidate.modelId !== 'string') {
		return null;
	}
	if (candidate.source !== null && candidate.source !== undefined) {
		const source = candidate.source as Partial<{
			kind: string;
			host: string | null;
			sha256: string | null;
			downloadedAt: string;
		}>;
		if (
			typeof source.kind !== 'string' ||
			!SLOT_SOURCE_KINDS.includes(source.kind) ||
			source.host === undefined ||
			source.sha256 === undefined ||
			typeof source.downloadedAt !== 'string'
		) {
			return null;
		}
		if (source.host !== null && typeof source.host !== 'string') {
			return null;
		}
		if (source.sha256 !== null && typeof source.sha256 !== 'string') {
			return null;
		}
	} else if (candidate.source === undefined) {
		return null;
	}
	if (candidate.progress !== null && candidate.progress !== undefined) {
		const progress = candidate.progress as { done?: unknown; total?: unknown };
		if (typeof progress.done !== 'number' || typeof progress.total !== 'number') {
			return null;
		}
	} else if (candidate.progress === undefined) {
		return null;
	}
	return body as SlotState;
}

/** Validate the listing's slot rows; null when the shape drifted. */
function slotStatesFrom(body: unknown): SlotState[] | null {
	if (!Array.isArray(body)) {
		return null;
	}
	const slots: SlotState[] = [];
	for (const entry of body) {
		const slot = slotStateFrom(entry);
		if (slot === null) {
			return null;
		}
		slots.push(slot);
	}
	return slots;
}

/** Structurally validate one official-list entry; null when it is not one. */
function catalogEntryFrom(body: unknown): CatalogEntry | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<CatalogEntry>;
	if (
		typeof candidate.id !== 'string' ||
		typeof candidate.slot !== 'string' ||
		!MODEL_SLOTS.includes(candidate.slot) ||
		typeof candidate.name !== 'string' ||
		typeof candidate.license !== 'string' ||
		typeof candidate.bytes !== 'number' ||
		typeof candidate.downloaded !== 'boolean'
	) {
		return null;
	}
	if (candidate.dim !== null && candidate.dim !== undefined && typeof candidate.dim !== 'number') {
		return null;
	}
	return body as CatalogEntry;
}

/** Validate the listing's official rows; null when the shape drifted. */
function catalogFrom(body: unknown): CatalogEntry[] | null {
	if (!Array.isArray(body)) {
		return null;
	}
	const entries: CatalogEntry[] = [];
	for (const entry of body) {
		const catalog = catalogEntryFrom(entry);
		if (catalog === null) {
			return null;
		}
		entries.push(catalog);
	}
	return entries;
}

/**
 * POST /admin/models/download: start (or join) the default model
 * group's download. The endpoint is idempotent -- an in-flight or
 * most recent run answers 200 with its identifier, a first call 202
 * -- so both statuses are the same outcome for the caller.
 */
export async function startModelDownload(fetcher: typeof fetch = globalThis.fetch): Promise<ModelDownloadResult> {
	let response: Response;
	try {
		response = await apiFetch(`${MODELS_ENDPOINT}/download`, { method: 'POST' }, fetcher);
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
	const parsed = await readJsonBody(
		response,
		(
			body
		): {
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
		}
	);
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
export async function rerunModel(id: string, fetcher: typeof fetch = globalThis.fetch): Promise<ModelRerunResult> {
	let response: Response;
	try {
		response = await apiFetch(`${MODELS_ENDPOINT}/${encodeURIComponent(id)}/rerun`, { method: 'POST' }, fetcher);
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
export async function scanModels(fetcher: typeof fetch = globalThis.fetch): Promise<ModelScanResult> {
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
		if (
			body === null ||
			typeof body !== 'object' ||
			!Array.isArray((body as { registered?: unknown }).registered)
		) {
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

/** The probe POST's outcome: the identifier to poll, or the failure's i18n key. */
export type ProbeStartResult = { ok: true; probeId: string } | { ok: false; errorKey: string };

/**
 * DELETE /admin/models/{id}: the removal of a registration that is
 * not its slot's active one (the active one is a switch first).
 */
export async function deleteModel(
	id: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<{ ok: true } | { ok: false; errorKey: string }> {
	let response: Response;
	try {
		response = await apiFetch(`${MODELS_ENDPOINT}/${encodeURIComponent(id)}`, { method: 'DELETE' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	return { ok: true };
}

/**
 * POST /admin/models/{slot}/probe: stage and probe one candidate
 * source. The multipart body carries the URL or the uploaded file;
 * the answer is 202 with the identifier the dialog polls.
 */
export async function probeModelSource(
	slot: ModelSlotId,
	source: { kind: 'url'; url: string; sha256?: string } | { kind: 'file'; file: File; sha256?: string },
	fetcher: typeof fetch = globalThis.fetch
): Promise<ProbeStartResult> {
	const form = new FormData();
	form.set('kind', source.kind);
	if (source.sha256 !== undefined && source.sha256 !== '') {
		form.set('sha256', source.sha256);
	}
	if (source.kind === 'url') {
		form.set('url', source.url);
	} else {
		form.set('file', source.file, source.file.name || 'model.onnx');
	}
	let response: Response;
	try {
		response = await apiFetch(
			`${MODELS_ENDPOINT}/${encodeURIComponent(slot)}/probe`,
			{ method: 'POST', body: form },
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
		const probeId = (body as { probeId?: unknown }).probeId;
		return typeof probeId === 'string' && probeId !== '' ? probeId : null;
	});
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	return { ok: true, probeId: parsed.value };
}

/** The probe GET's outcome: the probe's state row, or the failure's i18n key. */
export type ProbeReadResult = { ok: true; probe: ModelProbe } | { ok: false; errorKey: string };

/**
 * GET /admin/models/probes/{probeId}: one probe's current state. A
 * 404 (the probe expired) is the retry-shaped failure the dialog maps
 * onto "probe again".
 */
export async function readModelProbe(
	probeId: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<ProbeReadResult> {
	let response: Response;
	try {
		response = await apiFetch(`${MODELS_ENDPOINT}/probes/${encodeURIComponent(probeId)}`, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	const parsed = await readJsonBody(response, (body): ModelProbe | null => {
		if (body === null || typeof body !== 'object') {
			return null;
		}
		const candidate = body as Partial<ModelProbe>;
		if (typeof candidate.probeId !== 'string' || typeof candidate.state !== 'string') {
			return null;
		}
		if (!['queued', 'running', 'ok', 'error', 'warn'].includes(candidate.state)) {
			return null;
		}
		if (typeof candidate.hostDeclared !== 'boolean') {
			return null;
		}
		return body as ModelProbe;
	});
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	return { ok: true, probe: parsed.value };
}

/**
 * POST /admin/models/{slot}/source: the register+swap half of the
 * pipeline. 202 answers the switch's facts (the destructive arm's
 * report, the recalc run's id); every refusal is the mapped error.
 */
export async function switchModelSource(
	slot: ModelSlotId,
	request:
		| { kind: 'builtin'; modelId: string; confirmWipe?: boolean }
		| { kind: 'url' | 'file'; probeId: string; confirmWipe?: boolean },
	fetcher: typeof fetch = globalThis.fetch
): Promise<SourceSwitchOutcome> {
	let response: Response;
	try {
		response = await apiFetch(
			`${MODELS_ENDPOINT}/${encodeURIComponent(slot)}/source`,
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(request)
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await modelErrorKey(response) };
	}
	const parsed = await readJsonBody(response, (body): SourceSwitchResult | null => {
		if (body === null || typeof body !== 'object') {
			return null;
		}
		const candidate = body as Partial<SourceSwitchResult>;
		if (
			candidate.activated === null ||
			typeof candidate.activated !== 'object' ||
			typeof (candidate.activated as { id?: unknown }).id !== 'string' ||
			typeof (candidate.activated as { slot?: unknown }).slot !== 'string'
		) {
			return null;
		}
		return body as SourceSwitchResult;
	});
	if (!parsed.ok) {
		return { ok: false, errorKey: parsed.errorKey };
	}
	const value = parsed.value;
	return {
		ok: true,
		activated: value.activated,
		dimensionChanged: value.dimensionChanged ?? undefined,
		wipedEmbeddings: value.wipedEmbeddings ?? undefined,
		recalcRunId: value.recalcRunId ?? undefined
	};
}

/** The switch POST's outcome, with the destructive arm's facts when it ran. */
export type SourceSwitchOutcome =
	| {
			ok: true;
			activated: { id: string; slot: ModelSlotId };
			dimensionChanged?: { from: number; to: number };
			wipedEmbeddings?: number;
			recalcRunId?: string;
	  }
	| { ok: false; errorKey: string };
