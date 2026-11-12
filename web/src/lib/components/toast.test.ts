import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { clearToasts, dismissToast, toast, toasts } from './toast.svelte';

// The toast store's contract: the three tones and their
// self-dismiss durations (success and info 3.2s, danger 6s), the
// click dismissal, the cap of three live toasts with the OLDEST
// dropping when a fourth arrives, and the idempotent dismissal that
// never races its own expiry.

beforeEach(() => {
	vi.useFakeTimers();
});

afterEach(() => {
	clearToasts();
	vi.useRealTimers();
});

test('a push lands oldest-first with its tone; the empty organ holds nothing', () => {
	expect(toasts()).toHaveLength(0);

	toast('Saved', 'success');
	toast('Removed', 'danger');
	toast('Noted', 'info');

	const live = toasts();
	expect(live.map((entry) => entry.text)).toEqual(['Saved', 'Removed', 'Noted']);
	expect(live.map((entry) => entry.tone)).toEqual(['success', 'danger', 'info']);
	expect(live.every((entry) => !entry.leaving && typeof entry.id === 'number')).toBe(true);
});

test('success and info self-dismiss after 3.2s; danger outlasts them at 6s', () => {
	toast('Saved', 'success');
	toast('Noted', 'info');
	toast('Removed', 'danger');

	vi.advanceTimersByTime(3199);
	expect(toasts()).toHaveLength(3);

	vi.advanceTimersByTime(1);
	// The two short tones left (their 150ms fade ends well before
	// the danger's expiry); the danger is still live.
	vi.advanceTimersByTime(150);
	expect(toasts().map((entry) => entry.text)).toEqual(['Removed']);

	vi.advanceTimersByTime(2800);
	expect(toasts()).toHaveLength(0);
});

test('a click dismissal drops the toast early, even the 6s danger', () => {
	const entry = toastsEntry('Removed', 'danger');

	dismissToast(entry.id);
	// The fade window first, then the queue drop.
	vi.advanceTimersByTime(150);
	expect(toasts()).toHaveLength(0);

	// The expiry timer is gone with it: advancing past the full
	// duration fires nothing (the queue stays empty).
	vi.advanceTimersByTime(10_000);
	expect(toasts()).toHaveLength(0);
});

test('a dismissal is idempotent: the second call never re-arms a fade', () => {
	const entry = toastsEntry('Saved', 'success');
	dismissToast(entry.id);
	dismissToast(entry.id);

	vi.advanceTimersByTime(150);
	expect(toasts()).toHaveLength(0);
});

test('past three live toasts the OLDEST drops outright, never fades', () => {
	toast('One');
	toast('Two');
	toast('Three');
	toast('Four');

	const live = toasts();
	expect(live.map((entry) => entry.text)).toEqual(['Two', 'Three', 'Four']);
	// The dropped toast left immediately: no leaving flag, no
	// pending timer -- the queue holds exactly the newest three.
	expect(live.every((entry) => entry.leaving === false)).toBe(true);

	// The survivors still expire on their own schedules.
	vi.advanceTimersByTime(3200 + 150);
	expect(toasts()).toHaveLength(0);
});

/** Push one toast and hand back its queue entry. */
function toastsEntry(text: string, tone: 'success' | 'danger' | 'info' = 'success') {
	toast(text, tone);
	return toasts()[0]!;
}
