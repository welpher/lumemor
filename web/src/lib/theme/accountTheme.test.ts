import { beforeEach, expect, test } from 'vitest';
import { adoptAccountTheme } from './accountTheme';
import { themePreference } from './theme.svelte';

// The account-theme adoption: the signed-in account's stored theme
// becomes this browser's preference. That is what makes a device with
// no stored preference (a new one, a cleared profile) open in the
// account's theme instead of the OS default.

function selfBody(theme: string): Record<string, unknown> {
	return {
		id: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
		name: 'Ada Lovelace',
		email: 'ada@example.com',
		role: 'admin',
		theme,
		locale: 'en',
		downloadFormat: 'zip',
		downloadIncludeVideos: true,
		downloadIncludeSidecars: false,
		downloadIncludeEmbeddedVideos: false
	};
}

function stubSelf(theme: string): typeof fetch {
	return async () =>
		new Response(JSON.stringify(selfBody(theme)), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		});
}

beforeEach(() => {
	localStorage.clear();
	document.documentElement.classList.remove('dark');
});

test('adopts the account theme the browser had never stored', async () => {
	await adoptAccountTheme(stubSelf('dark'));
	expect(themePreference()).toBe('dark');
	expect(document.documentElement.classList.contains('dark')).toBe(true);
});

test('the account auto theme becomes the system preference', async () => {
	await adoptAccountTheme(stubSelf('auto'));
	expect(themePreference()).toBe('system');
});

test('a failed read leaves the local choice standing', async () => {
	const failing: typeof fetch = async () => new Response('{}', { status: 500 });
	await adoptAccountTheme(failing);
	expect(document.documentElement.classList.contains('dark')).toBe(false);
});
