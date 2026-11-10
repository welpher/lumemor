import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import {
	subscribeAlbumEvents,
	subscribeAssetEvents,
	subscribeFaceEvents,
	subscribeJobEvents,
	subscribePersonEvents
} from './index';
import { streamDownSince, streamStatus } from './connection.svelte';

// The shared SSE subscription module: one EventSource per app, opened
// by the first subscriber and closed by the last unsubscribe, parsing
// the server's asset.* blocks, the job.* blocks and the people
// surface's person.* / face.clustered blocks and ignoring everything
// that is not the published payload shape. EventSource is a browser
// global jsdom does not implement, so the tests inject a mock through
// vi.stubGlobal and drive it by hand.

type Listener = (event: MessageEvent) => void;

class MockEventSource {
	static instances: MockEventSource[] = [];

	url: string;
	closed = false;
	readonly listeners = new Map<string, Set<Listener>>();
	/** The connection lifecycle callbacks the events lib reports into the store. */
	onopen: (() => void) | null = null;
	onerror: (() => void) | null = null;

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

	/** The browser's connection-landed report (onopen). */
	open(): void {
		this.onopen?.();
	}

	/** The browser's connection-lost report (onerror). */
	drop(): void {
		this.onerror?.();
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

// --- person.* events (the people surface's convergence) ----------------------

test('every person event name the people pages react to is subscribed', () => {
	const unsubscribe = subscribePersonEvents(() => undefined);
	const stream = MockEventSource.instances[0];
	for (const name of ['person.labeled', 'person.unlabeled', 'person.hidden', 'person.merged', 'person.face_moved']) {
		expect(stream?.listeners, name).toBeTruthy();
	}
	unsubscribe();
});

test('person subscribers receive the parsed person events, one shape per name', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribePersonEvents((event) => seen.push(event));

	// The wire payloads are all-strings maps (the server publishes
	// map[string]string) -- ids only, the label text and hidden
	// switch are one account's private state and never ride the
	// stream; the merged sources split on the comma.
	const stream = MockEventSource.instances[0];
	const userId = '0192f9a1-1a2b-3c4d-5e6f-7a8b9c0d1e2f';
	stream?.emit('person.labeled', { clusterId: 'c1', userId }, 1);
	stream?.emit('person.unlabeled', { clusterId: 'c1', userId }, 2);
	stream?.emit('person.hidden', { clusterId: 'c1', userId }, 3);
	stream?.emit('person.merged', { survivingClusterId: 't1', mergedClusterIds: 's1,s2', userId }, 4);
	stream?.emit('person.face_moved', { faceId: 'f1', clusterId: 't1', userId }, 5);

	expect(seen).toEqual([
		{ id: 1, name: 'person.labeled', clusterId: 'c1', userId },
		{ id: 2, name: 'person.unlabeled', clusterId: 'c1', userId },
		{ id: 3, name: 'person.hidden', clusterId: 'c1', userId },
		{ id: 4, name: 'person.merged', survivingClusterId: 't1', mergedClusterIds: ['s1', 's2'], userId },
		{ id: 5, name: 'person.face_moved', faceId: 'f1', clusterId: 't1', userId }
	]);
	unsubscribe();
});

test('malformed person blocks are ignored, not dispatched', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribePersonEvents((event) => seen.push(event));

	const stream = MockEventSource.instances[0];
	const userId = 'u1';
	// A payload that is not the published shape.
	stream?.emit('person.labeled', { something: 'else' }, 1);
	// A missing cluster id.
	stream?.emit('person.unlabeled', { userId }, 2);
	// An acting user id that is not a string.
	stream?.emit('person.labeled', { clusterId: 'c1', userId: 42 }, 3);
	// A cluster id that is only whitespace.
	stream?.emit('person.hidden', { clusterId: '', userId }, 4);
	// Merged sources that do not split into non-empty ids.
	stream?.emit('person.merged', { survivingClusterId: 't1', mergedClusterIds: '', userId }, 5);
	// A face move without a face id.
	stream?.emit('person.face_moved', { clusterId: 't1', userId }, 6);
	// A missing acting user id.
	stream?.emit('person.hidden', { clusterId: 'c1', isHidden: 'true' }, 7);
	// A missing id line (lastEventId empty).
	stream?.emit('person.labeled', { clusterId: 'c1', name: 'Ada', userId }, Number.NaN);

	expect(seen).toEqual([]);
	unsubscribe();
});

test('person subscribers ride the shared stream; the last unsubscribe closes it', () => {
	const assetUnsubscribe = subscribeAssetEvents(() => undefined);
	const personUnsubscribe = subscribePersonEvents(() => undefined);
	expect(MockEventSource.instances).toHaveLength(1);

	// The asset subscriber leaving does not close the stream the
	// person subscriber still uses.
	assetUnsubscribe();
	expect(MockEventSource.instances[0]?.closed).toBe(false);
	personUnsubscribe();
	expect(MockEventSource.instances[0]?.closed).toBe(true);
});

// --- album.* (the albums surface's write events) ------------------------------

test('every album event name the wall and detail react to is subscribed', () => {
	const unsubscribe = subscribeAlbumEvents(() => undefined);
	const stream = MockEventSource.instances[0];
	for (const name of [
		'album.created',
		'album.updated',
		'album.deleted',
		'album.items-changed',
		'album.shared',
		'album.unshared'
	]) {
		expect(stream?.listeners, name).toBeTruthy();
	}
	unsubscribe();
});

test('album subscribers receive the parsed album events, ids only', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribeAlbumEvents((event) => seen.push(event));

	// The wire payloads are all-strings maps (the server publishes
	// map[string]string): the album's id and one user's id -- the
	// GRANTEE on the two share events, the acting user otherwise --
	// never a title, a count or a lock state.
	const stream = MockEventSource.instances[0];
	stream?.emit('album.created', { albumId: 'al1', userId: 'u1' }, 21);
	stream?.emit('album.updated', { albumId: 'al1', userId: 'u1' }, 22);
	stream?.emit('album.deleted', { albumId: 'al1', userId: 'u1' }, 23);
	stream?.emit('album.items-changed', { albumId: 'al1', userId: 'u1' }, 24);
	stream?.emit('album.shared', { albumId: 'al2', userId: 'grantee' }, 25);
	stream?.emit('album.unshared', { albumId: 'al2', userId: 'grantee' }, 26);

	expect(seen).toEqual([
		{ id: 21, name: 'album.created', albumId: 'al1', userId: 'u1' },
		{ id: 22, name: 'album.updated', albumId: 'al1', userId: 'u1' },
		{ id: 23, name: 'album.deleted', albumId: 'al1', userId: 'u1' },
		{ id: 24, name: 'album.items-changed', albumId: 'al1', userId: 'u1' },
		{ id: 25, name: 'album.shared', albumId: 'al2', userId: 'grantee' },
		{ id: 26, name: 'album.unshared', albumId: 'al2', userId: 'grantee' }
	]);
	unsubscribe();
});

test('malformed album blocks are ignored, not dispatched', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribeAlbumEvents((event) => seen.push(event));

	const stream = MockEventSource.instances[0];
	// A payload that is not the published shape.
	stream?.emit('album.updated', { something: 'else' }, 1);
	// A missing album id.
	stream?.emit('album.updated', { userId: 'u1' }, 2);
	// An album id that is not a string.
	stream?.emit('album.updated', { albumId: 42, userId: 'u1' }, 3);
	// A missing user id.
	stream?.emit('album.created', { albumId: 'al1' }, 4);
	// An empty user id.
	stream?.emit('album.shared', { albumId: 'al1', userId: '' }, 5);
	// A missing id line (lastEventId empty).
	stream?.emit('album.unshared', { albumId: 'al1', userId: 'u1' }, Number.NaN);

	expect(seen).toEqual([]);
	unsubscribe();
});

test('album subscribers ride the shared stream; the last unsubscribe closes it', () => {
	const assetUnsubscribe = subscribeAssetEvents(() => undefined);
	const albumUnsubscribe = subscribeAlbumEvents(() => undefined);
	expect(MockEventSource.instances).toHaveLength(1);

	// The asset subscriber leaving does not close the stream the
	// album subscriber still uses.
	assetUnsubscribe();
	expect(MockEventSource.instances[0]?.closed).toBe(false);
	albumUnsubscribe();
	expect(MockEventSource.instances[0]?.closed).toBe(true);
});

// --- face.clustered (the recognition run's cluster totals) --------------------

test('face.clustered is subscribed and parses the all-strings counts into numbers', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribeFaceEvents((event) => seen.push(event));

	// The payload is map[string]string on the wire: both totals are
	// JSON strings, and the consumer needs numbers.
	MockEventSource.instances[0]?.emit('face.clustered', { totalClusters: '4', labeledClusters: '1' }, 9);

	expect(seen).toEqual([{ id: 9, name: 'face.clustered', totalClusters: 4, labeledClusters: 1 }]);
	unsubscribe();
});

test('malformed face.clustered blocks are ignored, not dispatched', () => {
	const seen: unknown[] = [];
	const unsubscribe = subscribeFaceEvents((event) => seen.push(event));

	const stream = MockEventSource.instances[0];
	// A payload that is not the published shape.
	stream?.emit('face.clustered', { something: 'else' }, 1);
	// A count that is not a number.
	stream?.emit('face.clustered', { totalClusters: 'lots', labeledClusters: '1' }, 2);
	// A negative count.
	stream?.emit('face.clustered', { totalClusters: '-1', labeledClusters: '1' }, 3);
	// A missing labeled total.
	stream?.emit('face.clustered', { totalClusters: '4' }, 4);
	// A missing id line (lastEventId empty).
	stream?.emit('face.clustered', { totalClusters: '4', labeledClusters: '1' }, Number.NaN);

	expect(seen).toEqual([]);
	unsubscribe();
});

// --- the connection store (the offline strip's truth) -------------------------

test('the connection store follows the stream lifecycle: connecting, open, down, closed', () => {
	const unsubscribe = subscribeAssetEvents(() => undefined);
	const stream = MockEventSource.instances[0];

	// The construction reports 'connecting' -- no strip face yet.
	expect(streamStatus()).toBe('connecting');
	// The browser's connection lands (onopen): the healthy state.
	stream.open();
	expect(streamStatus()).toBe('open');
	expect(streamDownSince()).toBeNull();

	// The browser loses the connection (onerror): the strip's face,
	// with the drop's timestamp (the elapsed clock's origin).
	stream.drop();
	expect(streamStatus()).toBe('down');
	expect(streamDownSince()).not.toBeNull();

	// A repeated error (another failed retry) keeps the FIRST drop's
	// timestamp: the elapsed count spans the whole reconnecting
	// window, never restarting at every attempt.
	const firstDrop = streamDownSince();
	stream.drop();
	expect(streamDownSince()).toBe(firstDrop);

	// The browser's own retry lands: recovery, the strip falls.
	stream.open();
	expect(streamStatus()).toBe('open');
	expect(streamDownSince()).toBeNull();

	// The intentional close (the last subscriber left): nothing is
	// lost when nothing is live -- the strip falls with it.
	unsubscribe();
	expect(streamStatus()).toBe('closed');
	expect(streamDownSince()).toBeNull();
});

test('an error during the initial connect is already the down state (the strip never waits for a first open)', () => {
	const unsubscribe = subscribeAssetEvents(() => undefined);
	const stream = MockEventSource.instances[0];

	// The server is unreachable from the very first attempt: the
	// browser reports the failure while still connecting -- the
	// stream the pages expect is not there, and the strip rises.
	stream.drop();
	expect(streamStatus()).toBe('down');
	expect(streamDownSince()).not.toBeNull();

	unsubscribe();
	expect(streamStatus()).toBe('closed');
});
