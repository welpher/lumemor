// Shared i18n wiring for the web app, built on svelte-i18n.
//
// THE SHARED-DIRECTORY MECHANISM
//
// The message files are the repository-root `i18n/*.json` — the exact
// same files the mobile app consumes (the two clients share one
// translation table; there is no copy or build-time sync step). The
// web app reads them with a relative `import.meta.glob` from this
// module: `web/src/lib/i18n/` is four levels below the repository
// root, so `../../../../i18n/*.json` resolves to the shared table.
// Vite (dev and vitest) transforms the glob into module imports and
// rollup inlines the JSON into the bundle at build time; the dev
// server needs the repository root inside `server.fs.allow` to serve
// files outside the project root (see vite.config.ts).
//
// Locale handling: the initial locale is the stored preference, else
// detected from `navigator.language` (zh* -> zh-Hans, otherwise en).
// Every locale change is mirrored onto `document.documentElement.lang`
// so assistive tech announces the right language. Missing keys fall
// back to en through svelte-i18n's fallbackLocale.

import { get } from 'svelte/store';
import { addMessages, init, locale, _ } from 'svelte-i18n';

export type AppLocale = 'en' | 'zh-Hans';

/** Locales the UI officially maintains (mirrors the i18n directory). */
export const APP_LOCALES: readonly AppLocale[] = ['en', 'zh-Hans'];

const LOCALE_STORAGE_KEY = 'lumemor:locale';

// eager: true inlines every translation file into the bundle. The
// table is small (two flat files), so async per-locale chunks would
// only add a loading flash, not save meaningful bytes.
const messageModules = import.meta.glob('../../../../i18n/*.json', {
	eager: true,
	import: 'default'
}) as Record<string, Record<string, string>>;

function dictionaryFromGlob(
	modules: Record<string, Record<string, string>>
): Partial<Record<AppLocale, Record<string, string>>> {
	const dictionary: Partial<Record<AppLocale, Record<string, string>>> = {};
	for (const [path, messages] of Object.entries(modules)) {
		const fileName = path.slice(path.lastIndexOf('/') + 1);
		const localeId = fileName.slice(0, -'.json'.length) as AppLocale;
		if (APP_LOCALES.includes(localeId)) {
			dictionary[localeId] = messages;
		}
	}
	return dictionary;
}

function isAppLocale(value: string | null): value is AppLocale {
	return value !== null && (APP_LOCALES as readonly string[]).includes(value);
}

/**
 * Locale detection: stored preference first, then the browser
 * language (any Chinese variant maps to zh-Hans), then en.
 */
export function detectLocale(): AppLocale {
	try {
		const stored = localStorage.getItem(LOCALE_STORAGE_KEY);
		if (isAppLocale(stored)) {
			return stored;
		}
	} catch {
		// Storage unavailable: fall through to the browser language.
	}
	const languages = navigator.languages?.length ? navigator.languages : [navigator.language ?? 'en'];
	for (const tag of languages) {
		if (tag.toLowerCase().startsWith('zh')) {
			return 'zh-Hans';
		}
	}
	return 'en';
}

let initPromise: Promise<void> | null = null;

/** Initialize svelte-i18n with the shared messages; idempotent. */
export function initI18n(): Promise<void> {
	if (initPromise === null) {
		initPromise = (async () => {
			const initialLocale = detectLocale();
			// addMessages seeds the dictionary synchronously (the glob
			// already inlined every file), so init resolves immediately
			// and no route ever renders with messages still loading.
			const dictionary = dictionaryFromGlob(messageModules);
			for (const localeId of APP_LOCALES) {
				const messages = dictionary[localeId];
				if (messages) {
					addMessages(localeId, messages);
				}
			}
			await init({
				fallbackLocale: 'en',
				initialLocale
			});
			// Mirror every locale change (ours or svelte-i18n's) onto
			// <html lang>. Subscribing (rather than setting inside
			// setLocale) also covers programmatic locale store writes.
			// The subscription lives for the app's lifetime.
			locale.subscribe((current) => {
				if (current) {
					document.documentElement.lang = current;
				}
			});
		})();
	}
	return initPromise;
}

/** The active locale (en before initialization completes). */
export function currentLocale(): AppLocale {
	const current = get(locale) ?? null;
	return isAppLocale(current) ? current : 'en';
}

/** Switch the UI locale and remember the choice. */
export function setLocale(next: AppLocale): void {
	locale.set(next);
	try {
		localStorage.setItem(LOCALE_STORAGE_KEY, next);
	} catch {
		// Storage unavailable: the switch still applies for this session.
	}
}

/** Translate one key with the current locale (non-reactive reads). */
export function translate(
	key: string,
	values?: Record<string, string | number | boolean | Date | null | undefined>
): string {
	const formatter = get(_);
	return formatter(key, { values });
}
