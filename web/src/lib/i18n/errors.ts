// The shared error-code rendering rule of the web app.
//
// Server error codes map to i18n keys by lowercasing (the repo-wide
// convention), but an UNKNOWN code -- a proxy answer, a drifted
// contract -- has no key, and svelte-i18n then echoes the key itself,
// which would leak code jargon into the UI. Every error surface
// renders through this rule: an unknown code falls back to the
// generic message. The helper takes the translate function so
// components pass their reactive `$_` (a plain `get(_)` here would
// not be tracked by the template, and a locale switch would leave a
// rendered error line in the old language).

/**
 * Translate one error-code key with the unknown-code fallback: the
 * generic message when the code has no entry in the table.
 */
export function resolveErrorText(translate: (key: string) => string, key: string): string {
	const text = translate(key);
	return text === key ? translate('internal') : text;
}
