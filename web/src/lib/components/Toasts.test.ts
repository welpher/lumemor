import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import Toasts from './Toasts.svelte';
import { clearToasts, toast, toasts } from './toast.svelte';

// The toast host's contract: one polite live region that exists
// before any toast lands, one card-face button per live toast (the
// tone's announcement role -- status for success and info, alert
// for danger -- with a decorative 16px glyph and the resolved
// 13px/600 text), the click that dismisses early, and at most three
// rendered at once (the store's cap).

beforeEach(() => {
	document.body.textContent = '';
	vi.useFakeTimers();
});

afterEach(() => {
	unmountAll();
	clearToasts();
	vi.useRealTimers();
});

const mounted: Array<ReturnType<typeof mount>> = [];

function mountHost(): ReturnType<typeof mount> {
	const host = mount(Toasts, { target: document.body });
	mounted.push(host);
	flushSync();
	return host;
}

function unmountAll(): void {
	while (mounted.length > 0) {
		unmount(mounted.pop()!);
		flushSync();
	}
}

function toastButtons(): HTMLButtonElement[] {
	return Array.from(document.body.querySelectorAll<HTMLButtonElement>('.toast'));
}

test('the empty organ renders the polite live region alone', () => {
	mountHost();

	const region = document.body.querySelector<HTMLElement>('.toasts');
	expect(region).not.toBeNull();
	expect(region?.getAttribute('aria-live')).toBe('polite');
	expect(toastButtons()).toHaveLength(0);
});

test('one toast per push: the resolved text, the tone roles, the decorative glyph', () => {
	mountHost();
	toast('Saved the name.', 'success');
	toast('Removed the face.', 'danger');
	toast('Model rerun started.', 'info');
	flushSync();

	const buttons = toastButtons();
	expect(buttons.map((button) => button.textContent?.trim())).toEqual([
		'Saved the name.',
		'Removed the face.',
		'Model rerun started.'
	]);
	// The announcement roles: status for the success and info
	// tones, alert for the danger.
	expect(buttons[0]?.getAttribute('role')).toBe('status');
	expect(buttons[1]?.getAttribute('role')).toBe('alert');
	expect(buttons[2]?.getAttribute('role')).toBe('status');
	// The icon never carries the message: decorative, aria-hidden.
	for (const button of buttons) {
		const icon = button.querySelector<SVGSVGElement>('svg');
		expect(icon?.getAttribute('aria-hidden')).toBe('true');
	}
});

test('a click on the toast dismisses it early', () => {
	mountHost();
	toast('Removed the face.', 'danger');
	flushSync();

	const button = toastButtons()[0]!;
	button.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
	flushSync();
	vi.advanceTimersByTime(150);
	flushSync();

	expect(toastButtons()).toHaveLength(0);
	expect(toasts()).toHaveLength(0);
});

test('the host renders the store cap: the newest three, oldest dropped', () => {
	mountHost();
	toast('One');
	toast('Two');
	toast('Three');
	toast('Four');
	flushSync();

	expect(toastButtons().map((button) => button.textContent?.trim())).toEqual(['Two', 'Three', 'Four']);
});
