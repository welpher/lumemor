import { expect, test } from 'vitest';
import { resolveErrorText } from './errors';

// The shared unknown-code fallback rule, pinned on a fake translator
// (the components' tests exercise it through their reactive `$_`).

test('a known code renders its message', () => {
	const translate = (key: string): string => (key === 'internal' ? 'generic' : 'message');
	expect(resolveErrorText(translate, 'asset_not_found')).toBe('message');
});

test('an unknown code falls back to the generic message', () => {
	// svelte-i18n echoes an unknown key back as the text.
	const translate = (key: string): string => (key === 'internal' ? 'generic' : key);
	expect(resolveErrorText(translate, 'proxy_teapot')).toBe('generic');
});
