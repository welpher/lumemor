import { beforeEach, expect, test, vi } from 'vitest';
import {
	initTheme,
	isDarkTheme,
	setThemePreference,
	themePreference,
	toggleTheme
} from './theme.svelte';

// jsdom does not implement matchMedia; this controllable stub stands
// in for the OS color-scheme preference. `dispatch` flips the matches
// value and notifies the change listeners.
function installMatchMedia(initialMatches: boolean) {
	const listeners = new Set<(event: MediaQueryListEvent) => void>();
	const query = {
		matches: initialMatches,
		media: '(prefers-color-scheme: dark)',
		onchange: null,
		addEventListener(_: string, listener: (event: MediaQueryListEvent) => void) {
			listeners.add(listener);
		},
		removeEventListener(_: string, listener: (event: MediaQueryListEvent) => void) {
			listeners.delete(listener);
		},
		dispatch(matches: boolean) {
			query.matches = matches;
			const event = { matches, media: query.media } as MediaQueryListEvent;
			for (const listener of listeners) {
				listener(event);
			}
		}
	};
	vi.stubGlobal('matchMedia', () => query);
	return query;
}

const media = installMatchMedia(false);

beforeEach(() => {
	document.documentElement.classList.remove('dark');
	localStorage.clear();
});

test('with no stored preference, the theme follows the system preference', () => {
	initTheme();
	expect(themePreference()).toBe('system');
	expect(document.documentElement.classList.contains('dark')).toBe(false);
	expect(isDarkTheme()).toBe(false);
});

test('a manual dark preference adds the .dark class to <html> and persists', () => {
	initTheme();
	setThemePreference('dark');
	expect(document.documentElement.classList.contains('dark')).toBe(true);
	expect(isDarkTheme()).toBe(true);
	expect(localStorage.getItem('lumemor:theme')).toBe('dark');
});

test('a manual light preference overrides a system preference for dark', () => {
	initTheme();
	setThemePreference('light');
	media.dispatch(true);
	expect(document.documentElement.classList.contains('dark')).toBe(false);
	expect(isDarkTheme()).toBe(false);
});

test('the system preference keeps working while the stored choice is system', () => {
	initTheme();
	setThemePreference('system');
	media.dispatch(true);
	expect(document.documentElement.classList.contains('dark')).toBe(true);
	media.dispatch(false);
	expect(document.documentElement.classList.contains('dark')).toBe(false);
});

test('toggleTheme flips between the light and dark themes', () => {
	initTheme();
	setThemePreference('light');
	toggleTheme();
	expect(document.documentElement.classList.contains('dark')).toBe(true);
	toggleTheme();
	expect(document.documentElement.classList.contains('dark')).toBe(false);
});
