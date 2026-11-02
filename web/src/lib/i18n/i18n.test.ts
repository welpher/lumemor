import { flushSync, mount, tick, unmount } from 'svelte';
import { addMessages } from 'svelte-i18n';
import { beforeEach, expect, test } from 'vitest';
import { currentLocale, detectLocale, initI18n, setLocale, translate } from './index';
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
	const app = mount(Login, { target: document.body });
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
