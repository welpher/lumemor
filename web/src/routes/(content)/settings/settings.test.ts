import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n, setLocale } from '$lib/i18n';
import { toasts } from '$lib/components/toast.svelte';
import Page from './+page.svelte';

// The user settings page: the five sections over the
// me-scoped endpoints. The profile save PATCHes and adopts the
// answer; the password form's own confirm guard blocks a mismatched
// pair before anything leaves the browser, and a successful change
// shows the revocation note and refreshes the device list; the
// appearance pair applies IMMEDIATELY (the html class flips on the
// theme pick, <html lang> follows the locale pick, and each pick
// persists its value through its own PATCH); the download
// preferences patch on every change; the devices list marks the
// current row (no revoke button on it) and revokes the others
// through the me-scoped delete.

function selfBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		id: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
		name: 'Ada Lovelace',
		email: 'ada@example.com',
		role: 'admin',
		theme: 'auto',
		locale: 'en',
		downloadFormat: 'zip',
		downloadIncludeVideos: true,
		downloadIncludeSidecars: false,
		downloadIncludeEmbeddedVideos: false,
		...overrides
	};
}

function deviceBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		id: '0192f3b2-4c5d-7e8f-8a9b-0c1d2e3f4a5b',
		kind: 'web',
		deviceName: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:132.0) Gecko/20100101 Firefox/132.0',
		lastSeenAt: '2026-11-06T09:15:00Z',
		ipAddress: '192.168.1.101',
		isCurrent: false,
		...overrides
	};
}

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** A 204 answer: no body at all (a 204-with-body Response throws). */
function noContentResponse(): Response {
	return new Response(null, { status: 204 });
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

/** The two read routes every mount fires, with default 200 bodies. */
function defaultReads(overrides: Partial<FetchRoute> = {}): FetchRoute {
	return {
		'/api/v1/users/me': () => jsonResponse(200, selfBody()),
		'/api/v1/users/me/sessions': () =>
			jsonResponse(200, [
				deviceBody({
					id: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
					deviceName:
						'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0 Safari/537.36',
					lastSeenAt: '2026-11-08T14:23:00Z',
					ipAddress: '192.168.1.100',
					isCurrent: true
				}),
				deviceBody()
			]),
		...overrides
	};
}

/** The mounted page under test, for the afterEach unmount. */
let page: ReturnType<typeof mount> | null = null;

function stubFetch(route: FetchRoute): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		const answer = route[url];
		if (answer === undefined) {
			throw new Error(`unexpected fetch ${init?.method ?? 'GET'} ${url}`);
		}
		return answer(init);
	});
}

async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
	await tick();
}

function mountPage(): ReturnType<typeof mount> {
	page = mount(Page, { target: document.body, props: {} });
	flushSync();
	return page;
}

function sectionOf(testId: string): HTMLElement {
	const section = document.body.querySelector<HTMLElement>(`[data-testid="${testId}"]`);
	if (section === null) {
		throw new Error(`missing section ${testId}`);
	}
	return section;
}

function fillField(id: string, value: string): void {
	const field = document.body.querySelector<HTMLInputElement>(`#${id}`);
	if (field === null) {
		throw new Error(`missing field ${id}`);
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

function submitSectionForm(testId: string): void {
	const form = sectionOf(testId).querySelector<HTMLFormElement>('form');
	if (form === null) {
		throw new Error(`missing form of ${testId}`);
	}
	form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
}

/**
 * Submit one section's form through the browser's own constraint
 * validation (the native pre-flight): requestSubmit refuses an invalid
 * form before the submit event, the path a reader's click takes.
 */
function requestSectionSubmit(testId: string): void {
	const form = sectionOf(testId).querySelector<HTMLFormElement>('form');
	if (form === null) {
		throw new Error(`missing form of ${testId}`);
	}
	form.requestSubmit();
	flushSync();
}

function pickSelect(testId: string, value: string): void {
	const select = document.body.querySelector<HTMLSelectElement>(`[data-testid="${testId}"]`);
	if (select === null) {
		throw new Error(`missing select ${testId}`);
	}
	select.value = value;
	select.dispatchEvent(new Event('change', { bubbles: true }));
}

function clickChip(testId: string): void {
	const chip = document.body.querySelector<HTMLButtonElement>(`[data-testid="${testId}"]`);
	if (chip === null) {
		throw new Error(`missing chip ${testId}`);
	}
	chip.click();
}

function callsOf(mock: ReturnType<typeof stubFetch>, method: string, suffix: string): RequestInit[] {
	return mock.mock.calls
		.filter((call) => String(call[0]).endsWith(suffix) && (call[1] as RequestInit | undefined)?.method === method)
		.map((call) => call[1] as RequestInit);
}

beforeEach(async () => {
	await initI18n();
	// The locale test flips the shared i18n store; every test starts
	// from the English table (the store is module state, the
	// assertions are English).
	setLocale('en');
	document.documentElement.lang = 'en';
	document.documentElement.classList.remove('dark');
	localStorage.clear();
});

afterEach(() => {
	if (page !== null) {
		unmount(page);
		page = null;
	}
	document.body.innerHTML = '';
	document.documentElement.classList.remove('dark');
	document.documentElement.lang = 'en';
	vi.unstubAllGlobals();
});

test('the read renders the five sections with the account values', async () => {
	vi.stubGlobal('fetch', stubFetch(defaultReads()));
	mountPage();
	await settle();

	for (const sectionId of [
		'settings-profile',
		'settings-password',
		'settings-appearance',
		'settings-download',
		'settings-devices'
	]) {
		expect(sectionOf(sectionId), `${sectionId} renders`).not.toBeNull();
	}

	const name = document.body.querySelector<HTMLInputElement>('#settings-name');
	const email = document.body.querySelector<HTMLInputElement>('#settings-email');
	expect(name?.value).toBe('Ada Lovelace');
	expect(email?.value).toBe('ada@example.com');

	// The appearance pair reflects the account row.
	expect(
		document.body
			.querySelector<HTMLButtonElement>('[data-testid="settings-theme-auto"]')
			?.getAttribute('aria-checked')
	).toBe('true');
	expect(document.body.querySelector<HTMLSelectElement>('[data-testid="settings-locale"]')?.value).toBe('en');

	// The devices: two rows, the current one badged, the raw user
	// agent condensed to the browser · platform pair, the meta line
	// carrying the address.
	const rows = document.body.querySelectorAll('[data-testid="settings-device-row"]');
	expect(rows.length).toBe(2);
	expect(document.querySelectorAll('[data-testid="settings-device-current"]').length).toBe(1);
	expect(document.body.textContent).toContain('Chrome · Linux');
	expect(document.body.textContent).toContain('Firefox · macOS');
	expect(document.body.textContent).toContain('192.168.1.100');

	// The current row carries no revoke button; the other one does.
	const currentRow = Array.from(rows).find((row) => row.querySelector('[data-testid="settings-device-current"]'));
	expect(currentRow?.querySelector('[data-testid="settings-device-revoke"]')).toBeNull();
	const otherRow = Array.from(rows).find((row) => row !== currentRow);
	expect(otherRow?.querySelector('[data-testid="settings-device-revoke"]')).not.toBeNull();
});

test('the profile save PATCHes the form and adopts the answer', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/users/me': (init?: RequestInit) =>
				init?.method === 'PATCH'
					? jsonResponse(200, selfBody({ name: 'Ada King', email: 'ada@lovelace.example' }))
					: jsonResponse(200, selfBody())
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	fillField('settings-name', 'Ada King');
	fillField('settings-email', 'ada@lovelace.example');
	submitSectionForm('settings-profile');
	await settle();

	const patches = callsOf(fetchMock, 'PATCH', '/users/me');
	expect(patches.length).toBe(1);
	expect(JSON.parse(String(patches[0].body))).toEqual({
		name: 'Ada King',
		email: 'ada@lovelace.example'
	});
	// The form adopted the saved state and the note shows.
	expect(document.body.querySelector<HTMLInputElement>('#settings-name')?.value).toBe('Ada King');
	expect(document.body.textContent).toContain('Profile saved.');
});

test('the profile save maps the 409 onto the shared i18n text', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/users/me': (init?: RequestInit) =>
					init?.method === 'PATCH'
						? jsonResponse(409, {
								code: 'USER_EMAIL_TAKEN',
								message: 'an account with this email already exists'
							})
						: jsonResponse(200, selfBody())
			})
		)
	);
	mountPage();
	await settle();

	fillField('settings-email', 'grace@example.com');
	submitSectionForm('settings-profile');
	await settle();

	// The failure lands UNDER the field it belongs to (the address is
	// the account identity), not in a toast and not at the page's top.
	const slot = document.body.querySelector('#settings-email-error');
	expect(slot?.textContent).toContain('An account with this email address already exists.');
	expect(slot?.getAttribute('role')).toBe('alert');
	expect(document.body.querySelector('#settings-email')?.getAttribute('aria-invalid')).toBe('true');
	expect(document.body.querySelector('#settings-name-error')).toBeNull();
	expect(toasts()).toHaveLength(0);
	// The form keeps the entered text for correcting.
	expect(document.body.querySelector<HTMLInputElement>('#settings-email')?.value).toBe('grace@example.com');
});

test('a validation failure naming a field stays under that field', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/users/me': (init?: RequestInit) =>
					init?.method === 'PATCH'
						? jsonResponse(400, {
								code: 'VALIDATION_FAILED',
								message: 'invalid profile payload',
								details: { fields: ['name'] }
							})
						: jsonResponse(200, selfBody())
			})
		)
	);
	mountPage();
	await settle();

	fillField('settings-name', '');
	submitSectionForm('settings-profile');
	await settle();

	expect(document.body.querySelector('#settings-name-error')?.textContent).toContain(
		'Please check your entries and try again.'
	);
	// The other field was not named, so it stays clean.
	expect(document.body.querySelector('#settings-email-error')).toBeNull();
	// The typed value survives the failed save.
	expect(document.body.querySelector<HTMLInputElement>('#settings-name')?.value).toBe('');
	expect(toasts()).toHaveLength(0);
});

test('an unnamed failure keeps its line in the section it belongs to', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/users/me': (init?: RequestInit) =>
					init?.method === 'PATCH'
						? jsonResponse(500, { code: 'INTERNAL', message: 'boom' })
						: jsonResponse(200, selfBody())
			})
		)
	);
	mountPage();
	await settle();

	fillField('settings-name', 'Ada King');
	submitSectionForm('settings-profile');
	await settle();

	// No field is implicated: the line sits in the profile section,
	// above its save -- still in place, still no toast.
	const section = sectionOf('settings-profile');
	expect(section?.querySelector('.settings__error')?.textContent).toContain('Something went wrong');
	expect(document.body.querySelector('#settings-name-error')).toBeNull();
	expect(toasts()).toHaveLength(0);
});

test('a mismatched confirm pair blocks the password change client-side', async () => {
	const fetchMock = stubFetch(defaultReads());
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	fillField('settings-password-current', 'correct horse battery staple');
	fillField('settings-password-new', 'another fine passphrase here');
	fillField('settings-password-confirm', 'a different passphrase entirely');
	submitSectionForm('settings-password');
	await settle();

	// Nothing left the browser: no PUT, and the shared mismatch
	// message shows under the confirm field.
	expect(callsOf(fetchMock, 'PUT', '/users/me/password').length).toBe(0);
	expect(document.body.textContent).toContain('The two new passwords do not match.');
});

test('the change form carries the native required pair and the 8-128 bound', async () => {
	const fetchMock = stubFetch(defaultReads());
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	// The fields mirror the change body: required, 8-128 runes. (The
	// short-value refusal itself is a browser-side dirty-value rule;
	// jsdom's programmatic values never trip it, so the attributes are
	// the pin here and the walkthrough is the short-value evidence.)
	const currentField = document.body.querySelector<HTMLInputElement>('#settings-password-current');
	const newField = document.body.querySelector<HTMLInputElement>('#settings-password-new');
	const confirmField = document.body.querySelector<HTMLInputElement>('#settings-password-confirm');
	expect(currentField?.required).toBe(true);
	expect(newField?.required).toBe(true);
	expect(newField?.getAttribute('minlength')).toBe('8');
	expect(newField?.getAttribute('maxlength')).toBe('128');
	expect(confirmField?.required).toBe(true);
	expect(confirmField?.getAttribute('minlength')).toBe('8');
	expect(confirmField?.getAttribute('maxlength')).toBe('128');

	// The empty form is the required guard's arm: the browser refuses
	// it before the submit event, so nothing is sent.
	fillField('settings-password-current', '');
	fillField('settings-password-new', 'another fine passphrase here');
	fillField('settings-password-confirm', 'another fine passphrase here');
	requestSectionSubmit('settings-password');
	await settle();
	expect(callsOf(fetchMock, 'PUT', '/users/me/password').length).toBe(0);

	// A filled pair still leaves (the native layer only pre-flights).
	fillField('settings-password-current', 'correct horse battery staple');
	requestSectionSubmit('settings-password');
	await settle();
	expect(callsOf(fetchMock, 'PUT', '/users/me/password').length).toBe(1);
});

test('a successful password change PUTs the pair, notes the revocation, and refreshes the devices', async () => {
	let devicesCalls = 0;
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/users/me/password': () => noContentResponse(),
			'/api/v1/users/me/sessions': () => {
				devicesCalls += 1;
				// The refresh after the change: the other device's row is
				// gone (the revocation the change performed).
				return jsonResponse(
					200,
					devicesCalls === 1
						? [deviceBody({ id: 'current-row', isCurrent: true }), deviceBody()]
						: [deviceBody({ id: 'current-row', isCurrent: true })]
				);
			}
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();
	expect(devicesCalls).toBe(1);

	fillField('settings-password-current', 'correct horse battery staple');
	fillField('settings-password-new', 'another fine passphrase here');
	fillField('settings-password-confirm', 'another fine passphrase here');
	submitSectionForm('settings-password');
	await settle();

	const puts = callsOf(fetchMock, 'PUT', '/users/me/password');
	expect(puts.length).toBe(1);
	expect(JSON.parse(String(puts[0].body))).toEqual({
		current: 'correct horse battery staple',
		next: 'another fine passphrase here'
	});
	// The note names the revocation; the list refetched and the other
	// device's row left with it.
	expect(document.body.textContent).toContain('Your other signed-in devices were signed out.');
	expect(devicesCalls).toBe(2);
	expect(document.querySelectorAll('[data-testid="settings-device-row"]').length).toBe(1);
});

test('a rejected password change maps the validation error, staying signed in', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/users/me/password': () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'current password does not match',
					details: { fields: ['current'] }
				})
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	fillField('settings-password-current', 'wrong current password');
	fillField('settings-password-new', 'another fine passphrase here');
	fillField('settings-password-confirm', 'another fine passphrase here');
	submitSectionForm('settings-password');
	await settle();

	// The server named the current field: the text sits under it, and
	// the typed value stays for the correction.
	const slot = document.body.querySelector('#settings-password-current-error');
	expect(slot?.textContent).toContain('Please check your entries and try again.');
	expect(document.body.querySelector<HTMLInputElement>('#settings-password-current')?.value).toBe(
		'wrong current password'
	);
	// The new pair was not named: no error under it, and the confirm
	// field's own mismatch slot stays clear.
	expect(document.body.querySelector('#settings-password-new-error')).toBeNull();
	expect(document.body.querySelector('#settings-password-confirm-error')).toBeNull();
	expect(toasts()).toHaveLength(0);
});

test('a rejected new password stays under the new field', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/users/me/password': () =>
					jsonResponse(400, {
						code: 'VALIDATION_FAILED',
						message: 'password must be 8 to 128 runes',
						details: { fields: ['next'] }
					})
			})
		)
	);
	mountPage();
	await settle();

	fillField('settings-password-current', 'correct horse battery staple');
	fillField('settings-password-new', 'short');
	fillField('settings-password-confirm', 'short');
	submitSectionForm('settings-password');
	await settle();

	// The server named the new field: the text sits under IT, and the
	// current field (which verified) stays clean.
	expect(document.body.querySelector('#settings-password-new-error')?.textContent).toContain(
		'Please check your entries and try again.'
	);
	expect(document.body.querySelector('#settings-password-current-error')).toBeNull();
	// The browser's own minlength guard is the pre-flight; the server
	// verdict here still keeps every typed value.
	expect(document.body.querySelector<HTMLInputElement>('#settings-password-new')?.value).toBe('short');
});

test('a failed appearance patch keeps its line in the appearance section', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/users/me': (init?: RequestInit) =>
					init?.method === 'PATCH'
						? jsonResponse(500, { code: 'INTERNAL', message: 'boom' })
						: jsonResponse(200, selfBody())
			})
		)
	);
	mountPage();
	await settle();

	document.body
		.querySelector<HTMLButtonElement>('[data-testid="settings-theme-dark"]')
		?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
	await settle();

	// The pick applied locally (the preference is the user's stated
	// intent) and the failure sits in the section, under the control
	// that fired it -- no toast.
	const section = sectionOf('settings-appearance');
	expect(section?.querySelector('.settings__error')?.textContent).toContain('Something went wrong');
	expect(document.body.querySelector('[data-testid="settings-theme-dark"]')?.getAttribute('aria-checked')).toBe(
		'true'
	);
	expect(toasts()).toHaveLength(0);
});

test('the theme pick applies immediately and persists the row', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/users/me': (init?: RequestInit) =>
				init?.method === 'PATCH'
					? jsonResponse(200, selfBody({ theme: 'dark' }))
					: jsonResponse(200, selfBody())
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	// Before: the html carries no dark class (the account is on auto
	// and the system preference here is light).
	expect(document.documentElement.classList.contains('dark')).toBe(false);

	clickChip('settings-theme-dark');
	await settle();

	// IMMEDIATE APPLY: the html class flipped on the pick, without a
	// reload; the PATCH persisted the same value on the row.
	expect(document.documentElement.classList.contains('dark')).toBe(true);
	const patches = callsOf(fetchMock, 'PATCH', '/users/me');
	expect(patches.length).toBe(1);
	expect(JSON.parse(String(patches[0].body))).toEqual({ theme: 'dark' });
	// The chip state follows the adopted answer.
	expect(
		document.body
			.querySelector<HTMLButtonElement>('[data-testid="settings-theme-dark"]')
			?.getAttribute('aria-checked')
	).toBe('true');

	// The light pick flips the class back the same way.
	clickChip('settings-theme-light');
	await settle();
	expect(document.documentElement.classList.contains('dark')).toBe(false);
});

test('the locale pick applies immediately and persists the row', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/users/me': (init?: RequestInit) =>
				init?.method === 'PATCH'
					? jsonResponse(200, selfBody({ locale: 'zh-Hans' }))
					: jsonResponse(200, selfBody())
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	expect(document.documentElement.lang).toBe('en');

	pickSelect('settings-locale', 'zh-Hans');
	await settle();

	// IMMEDIATE APPLY: the document language and the i18n store
	// reloaded on the pick; the PATCH persisted the same value.
	expect(document.documentElement.lang).toBe('zh-Hans');
	const patches = callsOf(fetchMock, 'PATCH', '/users/me');
	expect(patches.length).toBe(1);
	expect(JSON.parse(String(patches[0].body))).toEqual({ locale: 'zh-Hans' });
	// The section headings render through the fresh locale.
	expect(document.body.textContent).toContain('个人资料');
});

test('the download preferences patch on every change', async () => {
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/users/me': (init?: RequestInit) =>
				init?.method === 'PATCH'
					? jsonResponse(200, selfBody({ downloadFormat: 'tar', downloadIncludeSidecars: true }))
					: jsonResponse(200, selfBody())
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	// The format chip.
	clickChip('settings-format-tar');
	await settle();

	// The sidecar toggle.
	const sidecars = document.body.querySelector<HTMLInputElement>('[data-testid="settings-download-sidecars"]');
	sidecars!.checked = true;
	sidecars!.dispatchEvent(new Event('change', { bubbles: true }));
	await settle();

	// The embedded-motion toggle: off by default, one PATCH per change.
	const embedded = document.body.querySelector<HTMLInputElement>('[data-testid="settings-download-embedded"]');
	expect(embedded?.checked).toBe(false);
	embedded!.checked = true;
	embedded!.dispatchEvent(new Event('change', { bubbles: true }));
	await settle();

	const patches = callsOf(fetchMock, 'PATCH', '/users/me');
	expect(patches.length).toBe(3);
	expect(JSON.parse(String(patches[0].body))).toEqual({ downloadFormat: 'tar' });
	expect(JSON.parse(String(patches[1].body))).toEqual({ downloadIncludeSidecars: true });
	expect(JSON.parse(String(patches[2].body))).toEqual({ downloadIncludeEmbeddedVideos: true });
	// The controls adopted the saved state.
	expect(
		document.body
			.querySelector<HTMLButtonElement>('[data-testid="settings-format-tar"]')
			?.getAttribute('aria-checked')
	).toBe('true');
	expect(document.body.querySelector<HTMLInputElement>('[data-testid="settings-download-sidecars"]')?.checked).toBe(
		true
	);
});

test('revoking another device deletes its session and refetches the list', async () => {
	let devicesCalls = 0;
	const fetchMock = stubFetch(
		defaultReads({
			'/api/v1/users/me/sessions': () => {
				devicesCalls += 1;
				// The current row carries its own (Chrome) label, distinct
				// from the other device's Firefox one.
				return jsonResponse(
					200,
					devicesCalls === 1
						? [
								deviceBody({
									id: 'current-row',
									deviceName:
										'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0 Safari/537.36',
									isCurrent: true
								}),
								deviceBody()
							]
						: [
								deviceBody({
									id: 'current-row',
									deviceName:
										'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0 Safari/537.36',
									isCurrent: true
								})
							]
				);
			},
			'/api/v1/users/me/sessions/0192f3b2-4c5d-7e8f-8a9b-0c1d2e3f4a5b': () => noContentResponse()
		})
	);
	vi.stubGlobal('fetch', fetchMock);
	mountPage();
	await settle();

	const revoke = document.body.querySelector<HTMLButtonElement>('[data-testid="settings-device-revoke"]');
	revoke!.click();
	await settle();

	const deletes = callsOf(fetchMock, 'DELETE', '/users/me/sessions/0192f3b2-4c5d-7e8f-8a9b-0c1d2e3f4a5b');
	expect(deletes.length).toBe(1);
	// The list refetched and the revoked row left.
	expect(devicesCalls).toBe(2);
	expect(document.querySelectorAll('[data-testid="settings-device-row"]').length).toBe(1);
	expect(document.body.textContent).not.toContain('Firefox · macOS');
});

test('a failed device list read shows the mapped error with its retry', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch(
			defaultReads({
				'/api/v1/users/me/sessions': () =>
					jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' })
			})
		)
	);
	mountPage();
	await settle();

	// The account read succeeded (the sections render); the devices
	// section alone shows its failure.
	expect(document.body.textContent).toContain("You don't have permission to do this.");
	const retry = Array.from(document.body.querySelectorAll<HTMLButtonElement>('button')).find(
		(button) => button.textContent?.trim() === 'Try again'
	);
	expect(retry).toBeDefined();
});

test('an unparseable account read maps to the generic failure', async () => {
	vi.stubGlobal('fetch', stubFetch(defaultReads({ '/api/v1/users/me': () => jsonResponse(200, { nope: true }) })));
	mountPage();
	await settle();

	expect(document.body.textContent).toContain('Something went wrong');
});
