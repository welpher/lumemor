import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import Login from './+page.svelte';

// The login form contract: labeled fields, a submit that disables
// while in flight, error codes mapped onto the shared i18n table,
// the 429 countdown hint, and the redirectTo round-trip on success.
// $app/navigation is mocked because the real goto needs a running
// SvelteKit app; the mock records the navigation target.

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
import { goto } from '$app/navigation';

type LoginData = { setupRequired: boolean; redirectTo: string };

function jsonResponse(
	status: number,
	body: unknown,
	headers: Record<string, string> = {}
): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json', ...headers }
	});
}

function mountLogin(data: LoginData = { setupRequired: false, redirectTo: '/timeline' }) {
	return mount(Login, { target: document.body, props: { data, params: {} } });
}

function fillField(selector: string, value: string): void {
	const field = document.body.querySelector<HTMLInputElement>(selector);
	if (field === null) {
		throw new Error(`missing field ${selector}`);
	}
	field.value = value;
	field.dispatchEvent(new Event('input', { bubbles: true }));
}

function submitForm(): void {
	const form = document.body.querySelector<HTMLFormElement>('form');
	if (form === null) {
		throw new Error('missing form');
	}
	form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
}

/** Let the submit handler's promise chain and the DOM catch up. */
async function settle(): Promise<void> {
	await new Promise((resolve) => setTimeout(resolve, 0));
	await new Promise<void>((resolve) => {
		requestAnimationFrame(() => resolve());
	}).catch(() => undefined);
}

beforeEach(async () => {
	document.body.textContent = '';
	vi.clearAllMocks();
	await initI18n();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the form renders labeled fields and the pairing placeholder notes', () => {
	const page = mountLogin();
	flushSync();

	const email = document.body.querySelector<HTMLInputElement>('#login-email');
	const password = document.body.querySelector<HTMLInputElement>('#login-password');
	expect(email?.type).toBe('email');
	expect(password?.type).toBe('password');
	// Both fields are labeled through the Input component's id/for tie.
	for (const id of ['login-email', 'login-password']) {
		const label = document.body.querySelector<HTMLLabelElement>(`label[for="${id}"]`);
		expect(label, id).not.toBeNull();
	}
	const text = document.body.textContent ?? '';
	expect(text).toContain('Sign in to your family photo library.');
	// M4 placeholder notes (pairing QR + manual URL hint).
	expect(text).toContain('Pairing a phone by scanning a QR code arrives with the mobile app.');
	expect(text).toContain("You can also sign in on the mobile app by entering this server's address.");
	unmount(page);
	flushSync();
});

test('a 401 maps to the auth_invalid_credentials message inline', async () => {
	const page = mountLogin();
	flushSync();

	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS', message: 'no' }))
	);
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'wrong');
	submitForm();
	await settle();

	const error = document.body.querySelector<HTMLParagraphElement>('.login__error');
	expect(error?.getAttribute('role')).toBe('alert');
	expect(error?.textContent).toBe('Incorrect email or password. Please try again.');
	// No countdown hint on a plain credential failure.
	expect(document.body.querySelector('.login__hint')).toBeNull();
	unmount(page);
	flushSync();
});

test('a 429 maps to auth_rate_limited plus the Retry-After countdown hint', async () => {
	const page = mountLogin();
	flushSync();

	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			jsonResponse(429, { code: 'AUTH_RATE_LIMITED', message: 'slow down' }, { 'Retry-After': '30' })
		)
	);
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'again');
	submitForm();
	await settle();

	const error = document.body.querySelector<HTMLParagraphElement>('.login__error');
	expect(error?.textContent).toBe('Too many attempts. Please try again later.');
	const hint = document.body.querySelector<HTMLParagraphElement>('.login__hint');
	expect(hint?.textContent).toBe('You can try again in 30 seconds.');
	unmount(page);
	flushSync();
});

test('a 429 without a Retry-After shows the message without a countdown', async () => {
	const page = mountLogin();
	flushSync();

	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(429, { code: 'AUTH_RATE_LIMITED', message: 'slow down' }))
	);
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'again');
	submitForm();
	await settle();

	expect(document.body.querySelector('.login__error')?.textContent).toBe(
		'Too many attempts. Please try again later.'
	);
	expect(document.body.querySelector('.login__hint')).toBeNull();
	unmount(page);
	flushSync();
});

test('retrying during a countdown clears the hint on a different outcome', async () => {
	const page = mountLogin();
	flushSync();

	// First attempt: rate-limited with a 30-second Retry-After, the
	// countdown hint ticking.
	vi.stubGlobal(
		'fetch',
		vi.fn(async () =>
			jsonResponse(429, { code: 'AUTH_RATE_LIMITED', message: 'slow down' }, { 'Retry-After': '30' })
		)
	);
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'again');
	submitForm();
	await settle();
	expect(document.body.querySelector('.login__hint')?.textContent).toBe(
		'You can try again in 30 seconds.'
	);

	// The user retries before the window ends and the server now
	// answers a plain credential failure: the stale countdown must be
	// gone -- only the credential error may show (the regression the
	// review found: a frozen "try again in N seconds" hint).
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS', message: 'no' }))
	);
	submitForm();
	await settle();

	expect(document.body.querySelector('.login__hint')).toBeNull();
	expect(document.body.querySelector('.login__error')?.textContent).toBe(
		'Incorrect email or password. Please try again.'
	);
	unmount(page);
	flushSync();
});

test('an unknown error code falls back to the generic message', async () => {
	const page = mountLogin();
	flushSync();

	// A code with no i18n key (a proxy answer, contract drift) must
	// not leak the raw key into the UI.
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(502, { code: 'MYSTERY_PROXY_CODE', message: 'what' }))
	);
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'x');
	submitForm();
	await settle();

	const error = document.body.querySelector<HTMLParagraphElement>('.login__error');
	expect(error?.textContent).toBe('Something went wrong. Please try again.');
	unmount(page);
	flushSync();
});

test('a network failure maps to the generic internal message', async () => {
	const page = mountLogin();
	flushSync();

	vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(new TypeError('network down'))));
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'x');
	submitForm();
	await settle();

	expect(document.body.querySelector('.login__error')?.textContent).toBe(
		'Something went wrong. Please try again.'
	);
	unmount(page);
	flushSync();
});

test('the submit button disables while the request is in flight', async () => {
	const page = mountLogin();
	flushSync();

	let release: ((response: Response) => void) | undefined;
	const gate = new Promise<Response>((resolve) => {
		release = resolve;
	});
	vi.stubGlobal('fetch', vi.fn(async () => gate));
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'x');

	const button = document.body.querySelector<HTMLButtonElement>('button[type="submit"]');
	expect(button?.disabled).toBe(false);
	submitForm();
	await settle();
	expect(button?.disabled).toBe(true);
	// Nothing renders an outcome while the request is in flight.
	expect(document.body.querySelector('.login__error')).toBeNull();

	release?.(jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }));
	await settle();
	expect(button?.disabled).toBe(false);
	expect(document.body.querySelector('.login__error')?.textContent).toContain('Incorrect email');
	unmount(page);
	flushSync();
});

test('a successful sign-in navigates to the redirectTo target', async () => {
	const page = mountLogin({ setupRequired: false, redirectTo: '/a/42?tab=exif' });
	flushSync();

	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, { id: 's1', kind: 'WEB', createdAt: 'x', lastSeenAt: 'x' }))
	);
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'correct horse');
	submitForm();
	await settle();

	expect(goto).toHaveBeenCalledTimes(1);
	expect(goto).toHaveBeenCalledWith('/a/42?tab=exif');
	// A credential failure never replaced the success path's form.
	expect(document.body.querySelector('.login__error')).toBeNull();
	unmount(page);
	flushSync();
});

test('a deep-link default target is /timeline', async () => {
	const page = mountLogin();
	flushSync();

	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(200, { id: 's1', kind: 'WEB', createdAt: 'x', lastSeenAt: 'x' }))
	);
	fillField('#login-email', 'ada@example.test');
	fillField('#login-password', 'correct horse');
	submitForm();
	await settle();

	expect(goto).toHaveBeenCalledWith('/timeline');
	unmount(page);
	flushSync();
});

test('the zh-Hans locale renders the form through the shared table', async () => {
	const { setLocale } = await import('$lib/i18n');
	const page = mountLogin();
	flushSync();

	setLocale('zh-Hans');
	await tick();
	const label = document.body.querySelector<HTMLLabelElement>('label[for="login-email"]');
	expect(label?.textContent).toBe('邮箱');
	expect(document.body.querySelector('button[type="submit"]')?.textContent).toBe('登录');
	unmount(page);
	flushSync();
	setLocale('en');
});
