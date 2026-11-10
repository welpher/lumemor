import { flushSync, mount, unmount } from 'svelte';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { initI18n } from '$lib/i18n';
import { subscribeAssetEvents } from './index';
import OfflineStrip from './OfflineStrip.svelte';

// The offline strip: a thin yellow band
// that rises when the shared SSE stream drops and falls when the
// browser's own retry recovers it -- the connection store's 'down'
// state rendered, the elapsed reconnecting seconds counted live.
// The strip never renders while the stream is healthy, connecting,
// or intentionally closed (no subscribers = nothing live = nothing
// lost). The tests drive the real chain: a mock EventSource opened
// by a real subscriber, its onopen/onerror fired by hand, the strip
// mounted in the document, the seconds ticked under fake timers.

type Listener = (event: MessageEvent) => void;

class MockEventSource {
	static instances: MockEventSource[] = [];

	url: string;
	closed = false;
	onopen: (() => void) | null = null;
	onerror: (() => void) | null = null;

	constructor(url: string) {
		this.url = url;
		MockEventSource.instances.push(this);
	}

	addEventListener(_name: string, _listener: Listener): void {
		// The strip's tests never deliver SSE blocks; the connection
		// lifecycle is the unit under observation.
	}

	close(): void {
		this.closed = true;
	}

	/** The browser's connection-landed report. */
	open(): void {
		this.onopen?.();
	}

	/** The browser's connection-lost report. */
	drop(): void {
		this.onerror?.();
	}
}

beforeEach(async () => {
	document.body.textContent = '';
	MockEventSource.instances = [];
	vi.stubGlobal('EventSource', MockEventSource);
	vi.useFakeTimers();
	await initI18n();
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
});

/** The strip's rendered line, when it stands. */
function strip(): HTMLElement | null {
	return document.body.querySelector<HTMLElement>('.offline-strip');
}

test('nothing renders while the stream is closed, connecting or open', () => {
	const sheet = mount(OfflineStrip, { target: document.body });
	flushSync();
	// No stream at all (no subscriber): nothing is live, nothing is lost.
	expect(strip()).toBeNull();

	const unsubscribe = subscribeAssetEvents(() => undefined);
	const stream = MockEventSource.instances[0];
	flushSync();
	// The constructed stream is still connecting: no face.
	expect(strip()).toBeNull();
	stream.open();
	flushSync();
	// The healthy connection: no face.
	expect(strip()).toBeNull();

	unsubscribe();
	unmount(sheet);
	flushSync();
});

test('the strip rises on a dropped connection with the elapsed seconds, and falls on recovery', async () => {
	const unsubscribe = subscribeAssetEvents(() => undefined);
	const stream = MockEventSource.instances[0];
	stream.open();

	const sheet = mount(OfflineStrip, { target: document.body });
	flushSync();
	expect(strip()).toBeNull();

	// The connection drops: the yellow band rises with the honest
	// copy and the live count (zero at the drop itself).
	stream.drop();
	flushSync();
	expect(strip()).not.toBeNull();
	expect(strip()?.getAttribute('role')).toBe('status');
	expect(strip()?.textContent).toContain('Connection lost — reconnecting (0s)');

	// The count ticks once a second: the elapsed reconnecting
	// window grows live.
	await vi.advanceTimersByTimeAsync(2000);
	flushSync();
	expect(strip()?.textContent).toContain('reconnecting (2s)');

	// The browser's own retry lands: the strip falls.
	stream.open();
	flushSync();
	expect(strip()).toBeNull();

	unsubscribe();
	unmount(sheet);
	flushSync();
});

test('the strip falls when the last subscriber leaves (the intentional close loses nothing)', () => {
	const unsubscribe = subscribeAssetEvents(() => undefined);
	const stream = MockEventSource.instances[0];
	stream.open();

	const sheet = mount(OfflineStrip, { target: document.body });
	stream.drop();
	flushSync();
	expect(strip()).not.toBeNull();

	// The last subscriber leaves: the stream closes ON PURPOSE --
	// the strip falls with it (no "connection lost" over a stream
	// nobody is using).
	unsubscribe();
	flushSync();
	expect(strip()).toBeNull();

	unmount(sheet);
	flushSync();
});

test('a first-connect failure rises the strip (the stream the pages expect is not there)', () => {
	const unsubscribe = subscribeAssetEvents(() => undefined);
	const sheet = mount(OfflineStrip, { target: document.body });

	MockEventSource.instances[0]?.drop();
	flushSync();
	expect(strip()?.textContent).toContain('Connection lost — reconnecting');

	unsubscribe();
	unmount(sheet);
	flushSync();
});

test('the strip carries its offline glyph, decorative, leading the line', () => {
	const unsubscribe = subscribeAssetEvents(() => undefined);
	const stream = MockEventSource.instances[0];
	stream.open();

	const sheet = mount(OfflineStrip, { target: document.body });
	stream.drop();
	flushSync();

	// The icon-first rule's status face: the glyph leads the text,
	// decorative (the line's words are the message).
	const icon = strip()?.querySelector<SVGSVGElement>('svg');
	expect(icon).not.toBeNull();
	expect(strip()?.firstElementChild).toBe(icon);
	expect(icon?.getAttribute('aria-hidden')).toBe('true');
	// The notice fill: the Sun yellow of the brand palette.
	expect(strip()?.className).toContain('bg-notice');
	expect(strip()?.className).toContain('text-navy-900');

	unsubscribe();
	unmount(sheet);
	flushSync();
});
