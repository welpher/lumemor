// The me-scoped user settings client (the settings feature): the
// profile read and patch behind GET/PATCH /users/me, the password
// change behind PUT /users/me/password, and the device list behind
// GET/DELETE /users/me/sessions. No userId appears anywhere -- the
// session IS the identity, the cookie decides the row.
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased
// (user_email_taken for a taken address, validation_failed for a
// rejected field), 'internal' for a network failure or an
// unparseable body, and 'session_expired' for a 401.
//
// The 200/201 bodies are structurally validated: a drifted or
// proxied answer must never render as a broken surface -- it maps
// to the generic failure like any other unparseable body.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** The caller's own account, exactly the contract's UserSelfResponse shape. */
export type UserSelf = components['schemas']['UserSelfResponse'];

/** One device entry, exactly the contract's DeviceSession shape. */
export type DeviceSession = components['schemas']['DeviceSession'];

/** The patch payload the settings sections assemble; every field rides as-is. */
export type UserSelfPatch = Partial<
	Pick<
		UserSelf,
		| 'name'
		| 'email'
		| 'theme'
		| 'locale'
		| 'downloadFormat'
		| 'downloadIncludeVideos'
		| 'downloadIncludeSidecars'
	>
>;

/** The read's outcome: the account document, or the failure's i18n key. */
export type UserSelfResult =
	| { ok: true; self: UserSelf }
	| { ok: false; errorKey: string };

/** The patch's outcome: the saved account state, or the failure's i18n key. */
export type UserSelfSaveResult =
	| { ok: true; self: UserSelf }
	| { ok: false; errorKey: string };

/** The password change's outcome; the i18n key of the failure. */
export type PasswordChangeResult = { ok: true } | { ok: false; errorKey: string };

/** The device list's outcome, or the failure's i18n key. */
export type DeviceListResult =
	| { ok: true; devices: DeviceSession[] }
	| { ok: false; errorKey: string };

/** The device revoke's outcome; the i18n key of the failure. */
export type DeviceRevokeResult = { ok: true } | { ok: false; errorKey: string };

const SELF_ENDPOINT = '/api/v1/users/me';
const PASSWORD_ENDPOINT = '/api/v1/users/me/password';
const DEVICES_ENDPOINT = '/api/v1/users/me/sessions';

/** Structurally validate one self body; null when it is not one. */
function selfFrom(body: unknown): UserSelf | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<UserSelf>;
	if (
		typeof candidate.id !== 'string' ||
		typeof candidate.name !== 'string' ||
		typeof candidate.email !== 'string' ||
		typeof candidate.role !== 'string' ||
		typeof candidate.theme !== 'string' ||
		typeof candidate.locale !== 'string' ||
		typeof candidate.downloadFormat !== 'string' ||
		typeof candidate.downloadIncludeVideos !== 'boolean' ||
		typeof candidate.downloadIncludeSidecars !== 'boolean'
	) {
		return null;
	}
	return body as UserSelf;
}

/** Structurally validate one device entry; null when it is not one. */
function deviceFrom(body: unknown): DeviceSession | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<DeviceSession>;
	if (
		typeof candidate.id !== 'string' ||
		typeof candidate.kind !== 'string' ||
		typeof candidate.lastSeenAt !== 'string' ||
		typeof candidate.isCurrent !== 'boolean'
	) {
		return null;
	}
	return body as DeviceSession;
}

/** Map a non-OK response onto the failure's i18n key. */
async function userErrorKey(response: Response): Promise<string> {
	if (response.status === 401) {
		return 'session_expired';
	}
	const envelope = await errorEnvelope(response);
	return envelope !== null ? envelope.code.toLowerCase() : 'internal';
}

/**
 * GET /api/v1/users/me: the caller's own account -- the profile
 * fields, the appearance pair and the download preferences.
 */
export async function getUserSelf(
	fetcher: typeof fetch = globalThis.fetch
): Promise<UserSelfResult> {
	let response: Response;
	try {
		response = await apiFetch(SELF_ENDPOINT, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await userErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const self = selfFrom(body);
	if (self === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, self };
}

/**
 * PATCH /api/v1/users/me: apply one patch (the section's own fields)
 * and answer the complete saved state. A 409 maps to
 * user_email_taken; a 400 maps to validation_failed.
 */
export async function updateUserSelf(
	patch: UserSelfPatch,
	fetcher: typeof fetch = globalThis.fetch
): Promise<UserSelfSaveResult> {
	let response: Response;
	try {
		response = await apiFetch(
			SELF_ENDPOINT,
			{
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(patch)
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await userErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	const self = selfFrom(body);
	if (self === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, self };
}

/**
 * PUT /api/v1/users/me/password: change the password. A successful
 * change revokes every OTHER session of the account (the current
 * one stays -- this browser keeps its cookie); a wrong current
 * password maps to validation_failed naming it, never to a 401
 * session-expiry.
 */
export async function changeUserPassword(
	current: string,
	next: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<PasswordChangeResult> {
	let response: Response;
	try {
		response = await apiFetch(
			PASSWORD_ENDPOINT,
			{
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ current, next })
			},
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.ok) {
		return { ok: true };
	}
	return { ok: false, errorKey: await userErrorKey(response) };
}

/**
 * GET /api/v1/users/me/sessions: the caller's active sessions, most
 * recently used first, with the current one marked.
 */
export async function listUserDevices(
	fetcher: typeof fetch = globalThis.fetch
): Promise<DeviceListResult> {
	let response: Response;
	try {
		response = await apiFetch(DEVICES_ENDPOINT, undefined, fetcher);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!response.ok) {
		return { ok: false, errorKey: await userErrorKey(response) };
	}
	let body: unknown;
	try {
		body = await response.json();
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (!Array.isArray(body)) {
		return { ok: false, errorKey: 'internal' };
	}
	const devices: DeviceSession[] = [];
	for (const entry of body) {
		const device = deviceFrom(entry);
		if (device === null) {
			return { ok: false, errorKey: 'internal' };
		}
		devices.push(device);
	}
	return { ok: true, devices };
}

/**
 * DELETE /api/v1/users/me/sessions/{id}: revoke one of the caller's
 * other devices. The current session answers 400 (validation_failed)
 * through the contract -- the sign-out flow is its own exit.
 */
export async function revokeUserDevice(
	id: string,
	fetcher: typeof fetch = globalThis.fetch
): Promise<DeviceRevokeResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`${DEVICES_ENDPOINT}/${id}`,
			{ method: 'DELETE' },
			fetcher
		);
	} catch {
		return { ok: false, errorKey: 'internal' };
	}
	if (response.ok) {
		return { ok: true };
	}
	return { ok: false, errorKey: await userErrorKey(response) };
}
