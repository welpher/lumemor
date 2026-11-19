// Theme state: the `.dark` class on <html>, the stored manual
// preference and the system preference it falls back to.
//
// The class itself (not a data attribute) drives the dark token set
// in theme.css. The default is `system`, which follows
// `prefers-color-scheme` live through a media query listener; the
// manual preferences (`light`/`dark`) persist in localStorage under
// `lumemor:theme`. app.html carries a tiny pre-paint script that
// applies the same rule before the app boots so first paint never
// flashes the wrong theme.

export type ThemePreference = 'system' | 'light' | 'dark';

const THEME_STORAGE_KEY = 'lumemor:theme';
const DARK_MEDIA_QUERY = '(prefers-color-scheme: dark)';

let preference = $state<ThemePreference>('system');
let systemPrefersDark = $state(false);
let media: MediaQueryList | null = null;
let started = false;

function prefersDark(): boolean {
	return preference === 'dark' || (preference === 'system' && systemPrefersDark);
}

function applyTheme(): void {
	document.documentElement.classList.toggle('dark', prefersDark());
}

function persistPreference(): void {
	try {
		localStorage.setItem(THEME_STORAGE_KEY, preference);
	} catch {
		// Storage can be unavailable (private mode, embedded webviews);
		// the theme still applies for this session.
	}
}

function onSystemChange(event: MediaQueryListEvent): void {
	systemPrefersDark = event.matches;
	applyTheme();
}

/** Initialize theme state once; safe to call again (no-op). */
export function initTheme(): void {
	if (started) {
		return;
	}
	started = true;
	try {
		const stored = localStorage.getItem(THEME_STORAGE_KEY);
		if (stored === 'light' || stored === 'dark' || stored === 'system') {
			preference = stored;
		}
	} catch {
		// Same as above: fall back to the system preference.
	}
	// jsdom (the vitest environment) does not implement matchMedia;
	// tests install a stub before calling initTheme.
	if (typeof window.matchMedia === 'function') {
		media = window.matchMedia(DARK_MEDIA_QUERY);
		systemPrefersDark = media.matches;
		media.addEventListener('change', onSystemChange);
	}
	applyTheme();
}

/** The stored preference, including `system`. */
export function themePreference(): ThemePreference {
	return preference;
}

/** Whether the effective theme is dark right now. */
/**
 * The account's stored theme as this module's preference: `auto` is
 * the OS-following state (the account row's word for it), and the two
 * manual choices pass through unchanged.
 */
export function themePreferenceOf(theme: string): ThemePreference {
	if (theme === 'light' || theme === 'dark') {
		return theme;
	}
	return 'system';
}

export function isDarkTheme(): boolean {
	return prefersDark();
}

/**
 * Set and persist a theme preference; `system` clears the manual
 * choice and follows the OS again.
 */
export function setThemePreference(next: ThemePreference): void {
	preference = next;
	persistPreference();
	applyTheme();
}

/** Flip between the light and dark themes (manual preference). */
export function toggleTheme(): void {
	setThemePreference(prefersDark() ? 'light' : 'dark');
}
