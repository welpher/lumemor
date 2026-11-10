import { flushSync, mount, tick, unmount } from 'svelte';
import { addMessages } from 'svelte-i18n';
import { beforeEach, expect, test } from 'vitest';
import { APP_LOCALES, currentLocale, detectLocale, initI18n, localeLanguageName, setLocale, translate } from './index';
import Login from '../../routes/(auth)/login/+page.svelte';

// jsdom's navigator reports en-US; tests shadow the language
// properties per case to exercise the detection order.
function stubNavigatorLanguages(tags: string[]) {
	Object.defineProperty(window.navigator, 'languages', {
		value: tags,
		configurable: true
	});
	Object.defineProperty(window.navigator, 'language', {
		value: tags[0] ?? 'en',
		configurable: true
	});
}

beforeEach(() => {
	localStorage.clear();
	document.documentElement.lang = 'en';
});

test('any Chinese navigator language detects zh-Hans', () => {
	stubNavigatorLanguages(['zh-CN', 'en']);
	expect(detectLocale()).toBe('zh-Hans');
});

test('the registry derives from the shared directory and reads each locale raw name', () => {
	// The directory IS the registry: APP_LOCALES folds from the
	// shared i18n/ directory's file list (sorted), and the settings
	// picker reads each entry's `meta_language_name` RAW -- the name
	// is declared in its own language, never routed through $_().
	expect(APP_LOCALES).toEqual(['en', 'zh-Hans']);
	expect(localeLanguageName('en')).toBe('English');
	expect(localeLanguageName('zh-Hans')).toBe('简体中文');
});

test('non-Chinese navigator languages detect en', () => {
	stubNavigatorLanguages(['en-US']);
	expect(detectLocale()).toBe('en');
});

test('the stored locale wins over the navigator language', () => {
	stubNavigatorLanguages(['en-US']);
	localStorage.setItem('lumemor:locale', 'zh-Hans');
	expect(detectLocale()).toBe('zh-Hans');
});

test('init activates the detected locale and mirrors it onto <html lang>', async () => {
	stubNavigatorLanguages(['en-US']);
	await initI18n();
	expect(currentLocale()).toBe('en');
	expect(document.documentElement.lang).toBe('en');
	expect(translate('login_title')).toBe('Sign in');
});

test('missing keys in the active locale fall back to the en text', async () => {
	stubNavigatorLanguages(['en-US']);
	await initI18n();
	// Simulates a key that exists in the source table only: zh-Hans
	// has no fallback_probe, so the fallback locale must answer.
	addMessages('en', { fallback_probe: 'English fallback' });
	setLocale('zh-Hans');
	await tick();
	expect(translate('fallback_probe')).toBe('English fallback');
});

test('a locale switch updates <html lang> and the rendered text', async () => {
	stubNavigatorLanguages(['en-US']);
	await initI18n();
	setLocale('en');
	await tick();
	const app = mount(Login, {
		target: document.body,
		props: { data: { setupRequired: false, redirectTo: '/timeline' }, params: {} }
	});
	flushSync();
	expect(document.body.textContent).toContain('Sign in');
	setLocale('zh-Hans');
	await tick();
	expect(document.documentElement.lang).toBe('zh-Hans');
	expect(document.body.textContent).toContain('登录');
	expect(localStorage.getItem('lumemor:locale')).toBe('zh-Hans');
	unmount(app);
	document.body.textContent = '';
});
