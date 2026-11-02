import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { subscribeAssetEvents } from './index';

// The shared SSE subscription module: one EventSource per app, opened
// by the first subscriber and closed by the last unsubscribe, parsing
// the server's asset.* blocks and ignoring everything that is not the
// published payload shape. EventSource is a browser global jsdom does
// not implement, so the tests inject a mock through vi.stubGlobal and
// drive it by hand.

type Listener = (event: MessageEvent) => void;

class MockEventSource {
	static instances: MockEventSource[] = [];

	url: string;
	closed = false;
	readonly listeners = new Map<string, Set<Listener>>();

	constructor(url: string) {
		this.url = url;
		MockEventSource.instances.push(this);
	}

	addEventListener(name: string, listener: Listener): void {
		let set = this.listeners.get(name);
		if (set === undefined) {
			set = new Set();
			this.listeners.set(name, set);
		}
		set.add(listener);
	}

	close(): void {
		this.closed = true;
	}

	/** Deliver one SSE block the way the browser would. */
	emit(name: string, payload: unknown, id: number): void {
		const set = this.listeners.get(name);
		if (set === undefined) {
			return;
		}
		const message = new MessageEvent(name, {
			data: JSON.stringify(payload),
			lastEventId: String(id)
		});
		for (const listener of set) {
			listener(message);
		}
	}
}

beforeEach(() => {
	MockEventSource.instances = [];
	vi.stubGlobal('EventSource', MockEventSource);
});

afterEach(() => {
	vi.unstubAllGlobals();
});

test('the first subscriber opens the stream on the events path, the last closes it', () => {
	const received: number[] = [];
	const first = subscribeAssetEvents(() => received.push(1));
	const second = subscribeAssetEvents(() => received.push(2));

	expect(MockEventSource.instances).toHaveLength(1);
	expect(MockEventSource.instances[0]?.url).toBe('/api/v1/events');

	first();
	expect(MockEventSource.instances[0]?.closed).toBe(false);
	second();
	expect(MockEventSource.instances[0]?.closed).toBe(true);
});

test('a second subscription after closing opens a fresh stream', () => {
	const first = subscribeAssetEvents(() => undefined);
	first();
	expect(MockEventSource.instances).toHaveLength(1);

	const second = subscribeAssetEvents(() => undefined);
	expect(MockEventSource.instances).toHaveLength(2);
	second();
});

test('subscribers receive parsed asset events; unsubscribed ones do not', () => {
	const seen: Array<{ name: string; assetId: string }> = [];
	const active = subscribeAssetEvents((event) => seen.push({ name: event.name, assetId: event.assetId }));
	const gone = subscribeAssetEvents(() => seen.push({ name: 'should-not-happen', assetId: 'x' }));
	gone();

	const stream = MockEventSource.instances[0];
	stream?.emit('asset.trashed', { assetId: 'a1', kind: 'image' }, 7);

	expect(seen).toEqual([{ name: 'asset.trashed', assetId: 'a1' }]);
	active();
});

test('every asset event name the wall reacts to is subscribed', () => {
	const unsubscribe = subscribeAssetEvents(() => undefined);
	const stream = MockEventSource.instances[0];
	for (const name of [
		'asset.uploaded',
		'asset.ready',
		'asset.updated',
		'asset.deleted',
		'asset.trashed',
		'asset.restored'
	]) {
		expect(stream?.listeners, name).toBeTruthy();
	}
	unsubscribe();
});

test('malformed blocks are ignored, not dispatched', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribeAssetEvents((event) => seen.push(event));

	const stream = MockEventSource.instances[0];
	// A payload that is not the published shape.
	stream?.emit('asset.ready', { something: 'else' }, 1);
	// A kind outside the enum.
	stream?.emit('asset.ready', { assetId: 'a1', kind: 'document' }, 2);
	// A missing id line (lastEventId empty).
	stream?.emit('asset.ready', { assetId: 'a1', kind: 'image' }, Number.NaN);

	expect(seen).toEqual([]);
	unsubscribe();
});

test('a valid block reaches the listener with id, name, assetId and kind', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribeAssetEvents((event) => seen.push(event));

	MockEventSource.instances[0]?.emit('asset.ready', { assetId: 'a1', kind: 'video' }, 42);

	expect(seen).toEqual([{ id: 42, name: 'asset.ready', assetId: 'a1', kind: 'video' }]);
	unsubscribe();
});
