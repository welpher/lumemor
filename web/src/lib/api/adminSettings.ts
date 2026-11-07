// The admin settings clients (the storage-template feature and the
// settings feature): the read and three-stage validated save behind
// GET/PUT /admin/settings/storage, the trash retention pair behind
// GET/PUT /admin/settings/trash, and the video transcoding profile
// pair behind GET/PUT /admin/settings/transcoding.
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
export type StorageSettingsResult =
	| { ok: true; settings: StorageSettings }
	| { ok: false; errorKey: string };

/** The save's outcome: the saved settings, or the failure's i18n key. */
export type StorageSettingsSaveResult =
	| { ok: true; settings: StorageSettings }
	| { ok: false; errorKey: string; reason?: string };

/** The trash read's outcome: the settings, or the failure's i18n key. */
export type TrashSettingsResult =
	| { ok: true; settings: TrashSettings }
	| { ok: false; errorKey: string };

/** The trash save's outcome: the saved settings, or the failure's i18n key. */
export type TrashSettingsSaveResult =
	| { ok: true; settings: TrashSettings }
	| { ok: false; errorKey: string };

/** The transcoding read's outcome: the settings, or the failure's i18n key. */
export type TranscodingSettingsResult =
	| { ok: true; settings: TranscodingSettings }
	| { ok: false; errorKey: string };

/** The transcoding save's outcome: the saved settings, or the failure's i18n key. */
export type TranscodingSettingsSaveResult =
	| { ok: true; settings: TranscodingSettings }
	| { ok: false; errorKey: string };

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
export async function getStorageSettings(
	fetcher: typeof fetch = globalThis.fetch
): Promise<StorageSettingsResult> {
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
export async function getTrashSettings(
	fetcher: typeof fetch = globalThis.fetch
): Promise<TrashSettingsResult> {
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
