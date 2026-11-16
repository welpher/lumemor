// The admin settings clients: the read and three-stage validated save
// behind GET/PUT /admin/settings/storage, the trash retention pair
// behind GET/PUT /admin/settings/trash, the video transcoding profile
// pair behind GET/PUT /admin/settings/transcoding, and the three
// networking pairs -- the release check's switch/feed/cadence, the map
// tile source and the model download policy. (the mount-root
// allowlist pair retired with the allowlist itself -- the boundary
// is the compose mount, so no list needs managing.)
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased
// (auth_forbidden for a signed-in non-admin, validation_failed for a
// rejected value), 'internal' for a network failure or an
// unparseable body, and 'session_expired' for a 401.
//
// The 200 bodies are structurally validated: a drifted or proxied
// answer must never render as a broken surface -- it maps to the
// generic failure like any other unparseable body. A 400 carries the
// server's specific validation reason (the template's compile or
// render failure) in the envelope message, which the storage section
// shows beside the shared i18n text: the reason names the admin's
// own template text, something no translation table can pre-render.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** The settings document, exactly the contract's StorageSettingsResponse shape. */
export type StorageSettings = components['schemas']['StorageSettingsResponse'];

/** The trash settings, exactly the contract's TrashSettingsResponse shape. */
export type TrashSettings = components['schemas']['TrashSettingsResponse'];

/** The transcoding settings, exactly the contract's TranscodingSettingsResponse shape. */
export type TranscodingSettings = components['schemas']['TranscodingSettingsResponse'];

/** The read's outcome: the settings document, or the failure's i18n key. */
export type StorageSettingsResult = { ok: true; settings: StorageSettings } | { ok: false; errorKey: string };

/** The save's outcome: the saved settings, or the failure's i18n key. */
export type StorageSettingsSaveResult =
	{ ok: true; settings: StorageSettings } | { ok: false; errorKey: string; reason?: string };

/** The trash read's outcome: the settings, or the failure's i18n key. */
export type TrashSettingsResult = { ok: true; settings: TrashSettings } | { ok: false; errorKey: string };

/** The trash save's outcome: the saved settings, or the failure's i18n key. */
export type TrashSettingsSaveResult = { ok: true; settings: TrashSettings } | { ok: false; errorKey: string };

/** The transcoding read's outcome: the settings, or the failure's i18n key. */
export type TranscodingSettingsResult = { ok: true; settings: TranscodingSettings } | { ok: false; errorKey: string };

/** The transcoding save's outcome: the saved settings, or the failure's i18n key. */
export type TranscodingSettingsSaveResult =
	{ ok: true; settings: TranscodingSettings } | { ok: false; errorKey: string };

const SETTINGS_ENDPOINT = '/api/v1/admin/settings/storage';

/** Structurally validate one settings body; null when it is not one. */
function storageSettingsFrom(body: unknown): StorageSettings | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<StorageSettings>;
	if (typeof candidate.template !== 'string' || typeof candidate.samplePath !== 'string') {
		return null;
	}
	if (!Array.isArray(candidate.presets) || candidate.presets.some((p) => typeof p !== 'string')) {
		return null;
	}
	return body as StorageSettings;
}

/** Map a non-OK response onto the failure's i18n key. */
async function adminErrorKey(response: Response): Promise<string> {
	if (response.status === 401) {
		return 'session_expired';
	}
	const envelope = await errorEnvelope(response);
	return envelope !== null ? envelope.code.toLowerCase() : 'internal';
}

/**
 * GET /api/v1/admin/settings/storage: the instance's storage template,
 * its sample render and the built-in presets. A 403 is the signed-in
 * non-admin caller (auth_forbidden), not a session failure.
 */
export async function getStorageSettings(fetcher: typeof fetch = globalThis.fetch): Promise<StorageSettingsResult> {
	let response: Response;
	try {
		response = await apiFetch(SETTINGS_ENDPOINT, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const settings = storageSettingsFrom(body);
	if (settings === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, settings };
}

/**
 * PUT /admin/settings/storage: validate and save one template. A 200
 * body is the saved settings (the caller adopts it as the fresh
 * state); a 400 maps to validation_failed and carries the server's
 * specific reason -- the compile or sample-render failure naming the
 * offending template text.
 */
export async function saveStorageSettings(
	template: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<StorageSettingsSaveResult> {
	let response: Response;
	try {
		response = await apiFetch(
			SETTINGS_ENDPOINT,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ template })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		if (response.status === 401) {
			return { ok: false, errorKey: 'session_expired' };
		}
		// One body read: the envelope's code maps to the i18n key and
		// its optional message carries the 400's specific reason.
		let code: string | null = null;
		let reason: string | undefined;
		try {
			const body: unknown = await response.json();
			if (body !== null && typeof body === 'object') {
				const envelope = body as { code?: unknown; message?: unknown };
				if (typeof envelope.code === 'string') {
					code = envelope.code;
				}
				if (typeof envelope.message === 'string') {
					reason = envelope.message;
				}
			}
		} catch {
			// Not JSON: the generic failure.
		}
		return {
			ok: false,
			errorKey: code !== null ? code.toLowerCase() : 'internal',
			reason
		};
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const settings = storageSettingsFrom(body);
	if (settings === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, settings };
}

const TRASH_ENDPOINT = '/api/v1/admin/settings/trash';
const TRANSCODING_ENDPOINT = '/api/v1/admin/settings/transcoding';

/** Structurally validate one trash settings body; null when it is not one. */
function trashSettingsFrom(body: unknown): TrashSettings | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<TrashSettings>;
	if (typeof candidate.retentionDays !== 'number' || !Number.isInteger(candidate.retentionDays)) {
		return null;
	}
	return body as TrashSettings;
}

/** Structurally validate one transcoding settings body; null when it is not one. */
function transcodingSettingsFrom(body: unknown): TranscodingSettings | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<TranscodingSettings>;
	if (
		typeof candidate.preset !== 'string' ||
		typeof candidate.concurrency !== 'number' ||
		typeof candidate.thumbnailFrame !== 'boolean' ||
		typeof candidate.keepOriginal !== 'boolean'
	) {
		return null;
	}
	return body as TranscodingSettings;
}

/**
 * GET /api/v1/admin/settings/trash: the instance's trash retention
 * in days. A 403 is the signed-in non-admin caller
 * (auth_forbidden), not a session failure.
 */
export async function getTrashSettings(fetcher: typeof fetch = globalThis.fetch): Promise<TrashSettingsResult> {
	let response: Response;
	try {
		response = await apiFetch(TRASH_ENDPOINT, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const settings = trashSettingsFrom(body);
	if (settings === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, settings };
}

/**
 * PUT /admin/settings/trash: save the retention (whole days, 0 to
 * 365; 0 is immediate permanent delete on the expiry job's next
 * run). The 200 body is the saved value; a 400 maps to
 * validation_failed.
 */
export async function saveTrashSettings(
	retentionDays: number,
	fetcher: typeof fetch = globalThis.fetch
): Promise<TrashSettingsSaveResult> {
	let response: Response;
	try {
		response = await apiFetch(
			TRASH_ENDPOINT,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ retentionDays })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const settings = trashSettingsFrom(body);
	if (settings === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, settings };
}

/**
 * GET /api/v1/admin/settings/transcoding: the instance's video
 * transcoding profile. A 403 is the signed-in non-admin caller
 * (auth_forbidden), not a session failure.
 */
export async function getTranscodingSettings(
	fetcher: typeof fetch = globalThis.fetch
): Promise<TranscodingSettingsResult> {
	let response: Response;
	try {
		response = await apiFetch(TRANSCODING_ENDPOINT, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const settings = transcodingSettingsFrom(body);
	if (settings === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, settings };
}

/**
 * PUT /admin/settings/transcoding: save the video transcoding
 * profile (the preset, the concurrency bound, the poster-frame
 * switch, the failure policy). The profile affects NEW ingests only
 * (the contract's scope pin); a 400 maps to validation_failed.
 */
export async function saveTranscodingSettings(
	settings: TranscodingSettings,
	fetcher: typeof fetch = globalThis.fetch
): Promise<TranscodingSettingsSaveResult> {
	let response: Response;
	try {
		response = await apiFetch(
			TRANSCODING_ENDPOINT,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(settings)
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const saved = transcodingSettingsFrom(body);
	if (saved === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, settings: saved };
}

/** The release check's settings, exactly the contract's shape. */
export type UpdateCheckSettings = components['schemas']['UpdateCheckSettingsResponse'];

/** The map's tile source settings, exactly the contract's shape. */
export type TilesSettings = components['schemas']['TilesSettingsResponse'];

/** The model store's policy, exactly the contract's shape. */
export type ModelSettings = components['schemas']['ModelSettingsResponse'];

/** A settings read's outcome: the document, or the failure's i18n key. */
export type NetworkingSettingsResult<T> = { ok: true; settings: T } | { ok: false; errorKey: string };

const UPDATE_CHECK_SETTINGS_ENDPOINT = '/api/v1/admin/settings/update-check';
const TILES_SETTINGS_ENDPOINT = '/api/v1/admin/settings/tiles';
const MODEL_SETTINGS_ENDPOINT = '/api/v1/admin/settings/models';

/** Structurally validate one release check settings body; null when it is not one. */
function updateCheckSettingsFrom(body: unknown): UpdateCheckSettings | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<UpdateCheckSettings>;
	if (
		typeof candidate.enabled !== 'boolean' ||
		typeof candidate.endpoint !== 'string' ||
		typeof candidate.intervalHours !== 'number' ||
		!Number.isInteger(candidate.intervalHours)
	) {
		return null;
	}
	return body as UpdateCheckSettings;
}

/** Structurally validate one tile settings body; null when it is not one. */
function tilesSettingsFrom(body: unknown): TilesSettings | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<TilesSettings>;
	if (typeof candidate.tilejsonUrl !== 'string') {
		return null;
	}
	return body as TilesSettings;
}

/** Structurally validate one model settings body; null when it is not one. */
function modelSettingsFrom(body: unknown): ModelSettings | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<ModelSettings>;
	if (typeof candidate.autoDownload !== 'boolean') {
		return null;
	}
	if (!Array.isArray(candidate.allowedHosts) || typeof candidate.mirrorBaseUrl !== 'string') {
		return null;
	}
	// The params (E41): the two knobs ride the document, both floats
	// in the closed 0-1 domain the server validates -- a drifted or
	// proxied answer never renders as a surface.
	const params = candidate.params as ModelSettings['params'] | undefined;
	if (
		params === undefined ||
		typeof params.faceDetectMinScore !== 'number' ||
		typeof params.faceRecognizeThreshold !== 'number'
	) {
		return null;
	}
	return body as ModelSettings;
}

/** One settings read over a JSON endpoint (the admin settings shape). */
async function readSettings<T>(
	endpoint: string,
	validate: (body: unknown) => T | null,
	fetcher: typeof fetch
): Promise<NetworkingSettingsResult<T>> {
	let response: Response;
	try {
		response = await apiFetch(endpoint, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const settings = validate(body);
	if (settings === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, settings };
}

/** One settings save over a JSON endpoint (the admin settings shape). */
async function writeSettings<T>(
	endpoint: string,
	payload: unknown,
	validate: (body: unknown) => T | null,
	fetcher: typeof fetch
): Promise<NetworkingSettingsResult<T>> {
	let response: Response;
	try {
		response = await apiFetch(
			endpoint,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const settings = validate(body);
	if (settings === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, settings };
}

/** GET /api/v1/admin/settings/update-check: the release check's knobs. */
export function getUpdateCheckSettings(
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<UpdateCheckSettings>> {
	return readSettings(UPDATE_CHECK_SETTINGS_ENDPOINT, updateCheckSettingsFrom, fetcher);
}

/** PUT /api/v1/admin/settings/update-check: save the switch, the feed and the cadence. */
export function saveUpdateCheckSettings(
	settings: UpdateCheckSettings,
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<UpdateCheckSettings>> {
	return writeSettings(UPDATE_CHECK_SETTINGS_ENDPOINT, settings, updateCheckSettingsFrom, fetcher);
}

/** GET /api/v1/admin/settings/tiles: the map's tile source ('' when none). */
export function getTilesSettings(
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<TilesSettings>> {
	return readSettings(TILES_SETTINGS_ENDPOINT, tilesSettingsFrom, fetcher);
}

/** PUT /api/v1/admin/settings/tiles: save the tile source ('' clears it). */
export function saveTilesSettings(
	tilejsonUrl: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<TilesSettings>> {
	return writeSettings(TILES_SETTINGS_ENDPOINT, { tilejsonUrl }, tilesSettingsFrom, fetcher);
}

/** GET /api/v1/admin/settings/models: the model store's policy. */
export function getModelSettings(
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<ModelSettings>> {
	return readSettings(MODEL_SETTINGS_ENDPOINT, modelSettingsFrom, fetcher);
}

/** PUT /api/v1/admin/settings/models: save the automatic download switch. */
export function saveModelSettings(
	settings: ModelSettings,
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<ModelSettings>> {
	return writeSettings(MODEL_SETTINGS_ENDPOINT, settings, modelSettingsFrom, fetcher);
}

/** The derivative image settings, exactly the contract's shape. */
export type ImageSettings = components['schemas']['ImageSettingsResponse'];

/** The run budget, exactly the contract's shape. */
export type RunBudgetSettings = components['schemas']['RunBudgetSettingsResponse'];

/** The derivative summary's cost line, exactly the contract's shape. */
export type DerivativesSummary = components['schemas']['DerivativesSummaryResponse'];

const IMAGE_SETTINGS_ENDPOINT = '/api/v1/admin/settings/image';
const RUN_BUDGET_ENDPOINT = '/api/v1/admin/settings/run-budget';
const DERIVATIVES_SUMMARY_ENDPOINT = '/api/v1/admin/derivatives/summary';
const DERIVATIVES_REBUILD_ENDPOINT = '/api/v1/admin/derivatives/rebuild';
const DERIVATIVES_REBUILD_STOP_ENDPOINT = '/api/v1/admin/derivatives/rebuild/stop';

/** Structurally validate one image settings body; null when it is not one. */
function imageSettingsFrom(body: unknown): ImageSettings | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<ImageSettings>;
	if (
		typeof candidate.tileSize !== 'number' ||
		typeof candidate.previewSize !== 'number' ||
		typeof candidate.format !== 'string' ||
		typeof candidate.fullSize !== 'boolean' ||
		typeof candidate.rawEmbedded !== 'boolean'
	) {
		return null;
	}
	return body as ImageSettings;
}

/** Structurally validate one run budget body; null when it is not one. */
function runBudgetSettingsFrom(body: unknown): RunBudgetSettings | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<RunBudgetSettings>;
	if (
		typeof candidate.concurrencyPreset !== 'string' ||
		typeof candidate.nightlyEnabled !== 'boolean' ||
		typeof candidate.nightlyStart !== 'string' ||
		typeof candidate.nightlyEnd !== 'string' ||
		candidate.pausedJobs === null ||
		typeof candidate.pausedJobs !== 'object' ||
		typeof candidate.pausedJobs.thumbnail !== 'boolean' ||
		typeof candidate.pausedJobs.metadata !== 'boolean' ||
		typeof candidate.pausedJobs.transcode !== 'boolean' ||
		typeof candidate.pausedJobs.ml !== 'boolean' ||
		typeof candidate.pausedJobs.places !== 'boolean'
	) {
		return null;
	}
	return body as RunBudgetSettings;
}

/** Structurally validate one derivative summary body; null when it is not one. */
function derivativesSummaryFrom(body: unknown): DerivativesSummary | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<DerivativesSummary>;
	if (
		typeof candidate.imageCount !== 'number' ||
		typeof candidate.estimateMinutes !== 'number' ||
		!Number.isInteger(candidate.imageCount) ||
		!Number.isInteger(candidate.estimateMinutes)
	) {
		return null;
	}
	return body as DerivativesSummary;
}

/** GET /api/v1/admin/settings/image: the derivative image settings. */
export function getImageSettings(
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<ImageSettings>> {
	return readSettings(IMAGE_SETTINGS_ENDPOINT, imageSettingsFrom, fetcher);
}

/** PUT /api/v1/admin/settings/image: save the sizes and encoding wholesale. */
export function saveImageSettings(
	settings: ImageSettings,
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<ImageSettings>> {
	return writeSettings(IMAGE_SETTINGS_ENDPOINT, settings, imageSettingsFrom, fetcher);
}

/** GET /api/v1/admin/settings/run-budget: the background work's budget. */
export function getRunBudgetSettings(
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<RunBudgetSettings>> {
	return readSettings(RUN_BUDGET_ENDPOINT, runBudgetSettingsFrom, fetcher);
}

/** PUT /api/v1/admin/settings/run-budget: save the budget wholesale. */
export function saveRunBudgetSettings(
	settings: RunBudgetSettings,
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<RunBudgetSettings>> {
	return writeSettings(RUN_BUDGET_ENDPOINT, settings, runBudgetSettingsFrom, fetcher);
}

/** GET /api/v1/admin/derivatives/summary: the image card's cost line. */
export function getDerivativesSummary(
	fetcher: typeof fetch = globalThis.fetch
): Promise<NetworkingSettingsResult<DerivativesSummary>> {
	return readSettings(DERIVATIVES_SUMMARY_ENDPOINT, derivativesSummaryFrom, fetcher);
}

/** One rebuild action's outcome: the run identifier, or the failure's i18n key. */
export type RebuildResult = { ok: true; jobRunId: string } | { ok: false; errorKey: string };

/** POST /api/v1/admin/derivatives/rebuild: start (or join) the rebuild run. */
export async function rebuildDerivatives(
	scope: 'missing' | 'all',
	fetcher: typeof fetch = globalThis.fetch
): Promise<RebuildResult> {
	let response: Response;
	try {
		response = await apiFetch(
			DERIVATIVES_REBUILD_ENDPOINT,
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ scope })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const candidate = body as { jobRunId?: unknown };
	if (typeof candidate?.jobRunId !== 'string') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, jobRunId: candidate.jobRunId };
}

/** POST /api/v1/admin/derivatives/rebuild/stop: ask the in-flight run to stop. */
export async function stopDerivativesRebuild(fetcher: typeof fetch = globalThis.fetch): Promise<RebuildResult> {
	let response: Response;
	try {
		response = await apiFetch(DERIVATIVES_REBUILD_STOP_ENDPOINT, { method: 'POST' }, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await adminErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const candidate = body as { jobRunId?: unknown };
	if (typeof candidate?.jobRunId !== 'string') {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, jobRunId: candidate.jobRunId };
}
