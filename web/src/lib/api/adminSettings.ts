// The admin storage settings client (the storage-template feature):
// the read behind GET /admin/settings/storage and the three-stage
// validated save behind PUT /admin/settings/storage.
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased
// (auth_forbidden for a signed-in non-admin, validation_failed for a
// rejected template), 'internal' for a network failure or an
// unparseable body, and 'session_expired' for a 401.
//
// The 200 bodies are structurally validated: a drifted or proxied
// answer must never render as a broken surface -- it maps to the
// generic failure like any other unparseable body. A 400 carries the
// server's specific validation reason (the compile or render failure)
// in the envelope message, which the settings section shows beside
// the shared i18n text: the reason names the admin's own template
// text, something no translation table can pre-render.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** The settings document, exactly the contract's StorageSettingsResponse shape. */
export type StorageSettings = components['schemas']['StorageSettingsResponse'];

/** The read's outcome: the settings document, or the failure's i18n key. */
export type StorageSettingsResult =
	| { ok: true; settings: StorageSettings }
	| { ok: false; errorKey: string };

/** The save's outcome: the saved settings, or the failure's i18n key. */
export type StorageSettingsSaveResult =
	| { ok: true; settings: StorageSettings }
	| { ok: false; errorKey: string; reason?: string };

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
