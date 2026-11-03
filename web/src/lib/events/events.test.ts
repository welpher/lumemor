import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import { subscribeAssetEvents, subscribeJobEvents } from './index';

// The shared SSE subscription module: one EventSource per app, opened
// by the first subscriber and closed by the last unsubscribe, parsing
// the server's asset.* blocks and the job.* blocks and ignoring
// everything that is not the published payload shape. EventSource is
// a browser global jsdom does not implement, so the tests inject a
// mock through vi.stubGlobal and drive it by hand.

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

// --- job.* events (the admin surfaces' live progress) ----------------------

test('every job event name the admin surfaces react to is subscribed', () => {
	const unsubscribe = subscribeJobEvents(() => undefined);
	const stream = MockEventSource.instances[0];
	for (const name of ['job.started', 'job.progress', 'job.finished']) {
		expect(stream?.listeners, name).toBeTruthy();
	}
	unsubscribe();
});

test('a job subscription opens the stream on its own and the last unsubscribe closes it', () => {
	const first = subscribeJobEvents(() => undefined);
	const second = subscribeJobEvents(() => undefined);

	// The job subscribers ride the SAME shared stream, not their own.
	expect(MockEventSource.instances).toHaveLength(1);
	expect(MockEventSource.instances[0]?.url).toBe('/api/v1/events');

	first();
	expect(MockEventSource.instances[0]?.closed).toBe(false);
	second();
	expect(MockEventSource.instances[0]?.closed).toBe(true);
});

test('the stream stays open while either subscriber kind remains', () => {
	const assetUnsubscribe = subscribeAssetEvents(() => undefined);
	const jobUnsubscribe = subscribeJobEvents(() => undefined);
	expect(MockEventSource.instances).toHaveLength(1);

	// The asset subscriber leaving does not close the stream the job
	// subscriber still uses, and vice versa.
	assetUnsubscribe();
	expect(MockEventSource.instances[0]?.closed).toBe(false);
	jobUnsubscribe();
	expect(MockEventSource.instances[0]?.closed).toBe(true);
});

test('job subscribers receive parsed job events; asset subscribers do not, and vice versa', () => {
	const jobs: unknown[] = [];
	const assets: unknown[] = [];
	const jobUnsubscribe = subscribeJobEvents((event) => jobs.push(event));
	const assetUnsubscribe = subscribeAssetEvents((event) => assets.push(event));

	const stream = MockEventSource.instances[0];
	stream?.emit('job.progress', jobPayload({ done: '5' }), 8);
	stream?.emit('asset.ready', { assetId: 'a1', kind: 'image' }, 9);

	expect(jobs).toHaveLength(1);
	expect(assets).toHaveLength(1);

	jobUnsubscribe();
	assetUnsubscribe();
});

test('the all-strings job payload parses into numbers for progress bars', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribeJobEvents((event) => seen.push(event));

	// The server's payload is map[string]string on the wire: every
	// count arrives as a JSON string, and the consumer needs numbers.
	MockEventSource.instances[0]?.emit(
		'job.progress',
		{
			jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10',
			kind: 'model_download',
			total: '400371536',
			done: '104857600',
			failed: '0',
			state: 'running'
		},
		42
	);

	expect(seen).toEqual([
		{
			id: 42,
			name: 'job.progress',
			jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10',
			kind: 'model_download',
			total: 400371536,
			done: 104857600,
			failed: 0,
			state: 'running'
		}
	]);
	unsubscribe();
});

test('malformed job blocks are ignored, not dispatched', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribeJobEvents((event) => seen.push(event));

	const stream = MockEventSource.instances[0];
	// A payload that is not the published shape.
	stream?.emit('job.progress', { something: 'else' }, 1);
	// A state outside the run lifecycle.
	stream?.emit('job.progress', jobPayload({ state: 'exploded' }), 2);
	// A count that is not a number.
	stream?.emit('job.progress', jobPayload({ total: 'lots' }), 3);
	// A missing jobRunId.
	stream?.emit('job.progress', jobPayload({ jobRunId: '' }), 4);
	// A missing id line (lastEventId empty).
	stream?.emit('job.progress', jobPayload(), Number.NaN);

	expect(seen).toEqual([]);
	unsubscribe();
});

/** One well-formed job payload with per-test overrides. */
function jobPayload(overrides: Record<string, unknown> = {}): Record<string, string> {
	return {
		jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10',
		kind: 'model_download',
		total: '400371536',
		done: '0',
		failed: '0',
		state: 'running',
		...overrides
	};
}
