// The shared server-sent-events subscription of the web app.
//
// One refcounted EventSource on GET /api/v1/events feeds every
// consumer of the instance activity stream: the timeline wall (this
// milestone) and the upload queue overlay (the next task) both
// subscribe through `subscribeAssetEvents` and unsubscribe on
// unmount; the first subscriber opens the stream, the last one
// closes it.
//
// RECONNECT SEMANTICS: the browser's EventSource owns them. On a
// network drop it reconnects on its own and resends the
// `Last-Event-ID` header with the id of the last received event; the
// server replays everything newer than that id from its five-minute
// ring buffer and skips what fell out (the contract in
// openapi/lumemor.yaml). This module adds no reconnect logic of its
// own on purpose -- hand-rolling it would only diverge from the
// browser's.
//
// Event names arrive as SSE `event:` lines, so only
// addEventListener(name) receives them (the default `onmessage`
// handler never fires for named events). The asset.* payloads the
// server publishes are `{assetId, kind}` objects; anything else on
// the wire is ignored defensively rather than crashing subscribers.

export type AssetKind = 'image' | 'video';

/** The asset lifecycle events the server publishes on the stream. */
export type AssetEventName =
	| 'asset.uploaded'
	| 'asset.ready'
	| 'asset.updated'
	| 'asset.deleted'
	| 'asset.trashed'
	| 'asset.restored';

/** One parsed asset.* activity event, as the wall and upload queue consume it. */
export type AssetEvent = {
	/** The server's monotonically increasing event id (the resumption cursor). */
	id: number;
	name: AssetEventName;
	assetId: string;
	kind: AssetKind;
};

export type AssetEventListener = (event: AssetEvent) => void;

const EVENTS_PATH = '/api/v1/events';

const ASSET_EVENT_NAMES: readonly AssetEventName[] = [
	'asset.uploaded',
	'asset.ready',
	'asset.updated',
	'asset.deleted',
	'asset.trashed',
	'asset.restored'
];

const listeners = new Set<AssetEventListener>();
let source: EventSource | null = null;

/**
 * Parse one SSE block into an AssetEvent; null when the block is not
 * the payload shape this module subscribes to (a malformed body, a
 * missing id line -- the server's contract says both always hold, so
 * a violation means the block is not for us).
 */
function parseAssetEvent(name: AssetEventName, message: MessageEvent): AssetEvent | null {
	const id = Number.parseInt(message.lastEventId, 10);
	if (!Number.isFinite(id) || id < 0) {
		return null;
	}
	let payload: unknown;
	try {
		payload = JSON.parse(String(message.data));
	} catch {
		return null;
	}
	if (payload === null || typeof payload !== 'object') {
		return null;
	}
	const { assetId, kind } = payload as { assetId?: unknown; kind?: unknown };
	if (typeof assetId !== 'string' || assetId === '') {
		return null;
	}
	if (kind !== 'image' && kind !== 'video') {
		return null;
	}
	return { id, name, assetId, kind };
}

/**
 * Open the shared stream when the first listener arrives. The
 * EventSource constructor is read at call time (never at import
 * time) so test environments without one keep every subscriber inert
 * instead of throwing at module load.
 */
function connect(): void {
	if (source !== null) {
		return;
	}
	const EventSourceConstructor = globalThis.EventSource;
	if (typeof EventSourceConstructor !== 'function') {
		return;
	}
	source = new EventSourceConstructor(EVENTS_PATH);
	for (const name of ASSET_EVENT_NAMES) {
		source.addEventListener(name, (message: MessageEvent) => {
			const event = parseAssetEvent(name, message);
			if (event === null) {
				return;
			}
			for (const listener of listeners) {
				listener(event);
			}
		});
	}
}

/** Close the shared stream (the last listener left, or the app reset). */
function disconnect(): void {
	source?.close();
	source = null;
}

/**
 * Subscribe to the parsed asset.* events of the activity stream.
 * Returns the unsubscribe function; the stream closes when the last
 * subscriber unsubscribes, so a route leaving the DOM never leaves a
 * dangling connection behind.
 */
export function subscribeAssetEvents(listener: AssetEventListener): () => void {
	listeners.add(listener);
	connect();
	return () => {
		listeners.delete(listener);
		if (listeners.size === 0) {
			disconnect();
		}
	};
}
