// The admin user management client (M2d Task 8): the listing behind
// GET /admin/users, the account creation behind POST /admin/users,
// and the disabled toggle behind PATCH /admin/users/{id}.
//
// ERROR MAPPING (the repo-wide convention): every failure outcome
// carries the error code's i18n key -- the envelope code lowercased
// (auth_forbidden for a signed-in non-admin, user_email_taken for a
// duplicate address, validation_failed for payload problems and the
// server's self-disable guard), 'internal' for a network failure or
// an unparseable body, and 'session_expired' for a 401.
//
// The 200/201 bodies are structurally validated: the account listing
// is a small bounded document, but a drifted or proxied answer must
// never render as a broken surface -- it maps to the generic
// failure like any other unparseable body.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import type { components } from '../../../../openapi/generated/web/api';

/** One account of the listing, exactly the contract's AdminUserResponse shape. */
export type AdminUser = components['schemas']['AdminUserResponse'];

/** The role enum of the create form, the contract's user_role. */
export type AdminUserRole = components['schemas']['UserRole'];

/** The listing's outcome: every account, or the failure's i18n key. */
export type AdminUserListResult =
	| { ok: true; users: AdminUser[] }
	| { ok: false; errorKey: string };

/** The create form's outcome: the created account, or the failure's i18n key. */
export type AdminUserCreateResult =
	| { ok: true; user: AdminUser }
	| { ok: false; errorKey: string };

/** The toggle's outcome: the account after the write, or the failure's i18n key. */
export type AdminUserUpdateResult =
	| { ok: true; user: AdminUser }
	| { ok: false; errorKey: string };

const USERS_ENDPOINT = '/api/v1/admin/users';

/** Structurally validate one account body; null when it is not one. */
function adminUserFrom(body: unknown): AdminUser | null {
	if (body === null || typeof body !== 'object') {
		return null;
	}
	const candidate = body as Partial<AdminUser>;
	if (typeof candidate.id !== 'string' || typeof candidate.name !== 'string') {
		return null;
	}
	if (typeof candidate.isDisabled !== 'boolean') {
		return null;
	}
	if (candidate.role !== 'admin' && candidate.role !== 'user') {
		return null;
	}
	return body as AdminUser;
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
 * GET /api/v1/admin/users: every account of the instance, oldest
 * first. A 403 is the signed-in non-admin caller (auth_forbidden),
 * not a session failure.
 */
export async function listAdminUsers(
	fetcher: typeof fetch = globalThis.fetch
): Promise<AdminUserListResult> {
	let response: Response;
	try {
		response = await apiFetch(USERS_ENDPOINT, undefined, fetcher);
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
	if (body === null || typeof body !== 'object' || !Array.isArray((body as { items?: unknown }).items)) {
		return { ok: false, errorKey: 'internal' };
	}
	const users: AdminUser[] = [];
	for (const entry of (body as { items: unknown[] }).items) {
		const user = adminUserFrom(entry);
		if (user === null) {
			return { ok: false, errorKey: 'internal' };
		}
		users.push(user);
	}
	return { ok: true, users };
}

/**
 * POST /admin/users: create one account (the wizard's rules, plus
 * the role). The 201 body is the created account, isDisabled false
 * by construction; a duplicate email maps to user_email_taken.
 */
export async function createAdminUser(
	input: { name: string; email: string; password: string; role: AdminUserRole },
	fetcher: typeof fetch = globalThis.fetch
): Promise<AdminUserCreateResult> {
	let response: Response;
	try {
		response = await apiFetch(
			USERS_ENDPOINT,
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: input.name === '' ? undefined : input.name,
					email: input.email,
					password: input.password,
					role: input.role
				})
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
	const user = adminUserFrom(body);
	if (user === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, user };
}

/**
 * PATCH /admin/users/{id}: the enable/disable toggle. The 200 body
 * is the account after the write (the caller adopts it as the fresh
 * state); a 404 maps to user_not_found (a stale row on the client),
 * and the server's self-disable guard maps to validation_failed.
 */
export async function setAdminUserDisabled(
	id: string,
	isDisabled: boolean,
	fetcher: typeof fetch = globalThis.fetch
): Promise<AdminUserUpdateResult> {
	let response: Response;
	try {
		response = await apiFetch(
			`${USERS_ENDPOINT}/${encodeURIComponent(id)}`,
			{
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ isDisabled })
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
	const user = adminUserFrom(body);
	if (user === null) {
		return { ok: false, errorKey: 'internal' };
	}
	return { ok: true, user };
}
