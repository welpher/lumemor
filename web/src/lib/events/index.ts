// The shared server-sent-events subscription of the web app.
//
// One refcounted EventSource on GET /api/v1/events feeds every
// consumer of the instance activity stream: the timeline wall and
// the upload queue overlay (asset.* through `subscribeAssetEvents`)
// and the admin surfaces' live progress (job.* through
// `subscribeJobEvents`) all subscribe and unsubscribe on unmount;
// the first subscriber of any kind opens the stream, the last one
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
// server publishes are `{assetId, kind}` objects and the job.*
// payloads are `{jobRunId, kind, total, done, failed, state}` maps
// with every value a string (the counts parse through Number());
// anything else on the wire is ignored defensively rather than
// crashing subscribers.

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

/** The job run lifecycle events the server publishes on the stream. */
export type JobEventName = 'job.started' | 'job.progress' | 'job.finished';

/**
 * One parsed job.* event, as the admin surfaces consume it. The wire
 * payload is ALL STRINGS (the server publishes map[string]string),
 * so the counts are parsed with Number() here -- the consumers
 * receive numbers ready for progress bars.
 */
export type JobEvent = {
	/** The server's monotonically increasing event id (the resumption cursor). */
	id: number;
	name: JobEventName;
	/** The job_runs row the event announces. */
	jobRunId: string;
	/** The run's kind (model_download, the rerun kinds). */
	kind: string;
	/** The run's whole work, in the run's own units. */
	total: number;
	/** The work completed so far, in the same units. */
	done: number;
	/** The failed work count. */
	failed: number;
	/** The run's lifecycle state. */
	state: JobRunState;
};

export type JobEventListener = (event: JobEvent) => void;

/** The job run lifecycle states, the JobRunResponse contract's enum. */
export type JobRunState = 'queued' | 'running' | 'succeeded' | 'failed';

const EVENTS_PATH = '/api/v1/events';

const ASSET_EVENT_NAMES: readonly AssetEventName[] = [
	'asset.uploaded',
	'asset.ready',
	'asset.updated',
	'asset.deleted',
	'asset.trashed',
	'asset.restored'
];

const JOB_EVENT_NAMES: readonly JobEventName[] = [
	'job.started',
	'job.progress',
	'job.finished'
];

const JOB_RUN_STATES: readonly JobRunState[] = ['queued', 'running', 'succeeded', 'failed'];

const assetListeners = new Set<AssetEventListener>();
const jobListeners = new Set<JobEventListener>();
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
 * Parse one SSE block into a JobEvent; null when the block is not
 * the payload shape this module subscribes to. The wire payload is
 * all strings (the server publishes map[string]string), so the
 * counts run through Number() -- anything that does not parse to a
 * non-negative integer, or a state outside the run lifecycle, means
 * the block is not for us.
 */
function parseJobEvent(name: JobEventName, message: MessageEvent): JobEvent | null {
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
	const { jobRunId, kind, total, done, failed, state } = payload as {
		jobRunId?: unknown;
		kind?: unknown;
		total?: unknown;
		done?: unknown;
		failed?: unknown;
		state?: unknown;
	};
	if (typeof jobRunId !== 'string' || jobRunId === '') {
		return null;
	}
	if (typeof kind !== 'string' || kind === '') {
		return null;
	}
	if (typeof state !== 'string' || !JOB_RUN_STATES.includes(state as JobRunState)) {
		return null;
	}
	const counts = [Number(total), Number(done), Number(failed)];
	for (const count of counts) {
		if (!Number.isInteger(count) || count < 0) {
			return null;
		}
	}
	return {
		id,
		name,
		jobRunId,
		kind,
		total: counts[0],
		done: counts[1],
		failed: counts[2],
		state: state as JobRunState
	};
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
			for (const listener of assetListeners) {
				listener(event);
			}
		});
	}
	for (const name of JOB_EVENT_NAMES) {
		source.addEventListener(name, (message: MessageEvent) => {
			const event = parseJobEvent(name, message);
			if (event === null) {
				return;
			}
			for (const listener of jobListeners) {
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

/** The number of subscribers across both event kinds. */
function subscriberCount(): number {
	return assetListeners.size + jobListeners.size;
}

/**
 * Subscribe to the parsed asset.* events of the activity stream.
 * Returns the unsubscribe function; the stream closes when the last
 * subscriber of any kind unsubscribes, so a route leaving the DOM
 * never leaves a dangling connection behind.
 */
export function subscribeAssetEvents(listener: AssetEventListener): () => void {
	assetListeners.add(listener);
	connect();
	return () => {
		assetListeners.delete(listener);
		if (subscriberCount() === 0) {
			disconnect();
		}
	};
}

/**
 * Subscribe to the parsed job.* events of the activity stream -- the
 * job runs' live progress, riding the SAME refcounted EventSource as
 * the asset subscribers. Returns the unsubscribe function; the stream
 * closes when the last subscriber of any kind unsubscribes.
 */
export function subscribeJobEvents(listener: JobEventListener): () => void {
	jobListeners.add(listener);
	connect();
	return () => {
		jobListeners.delete(listener);
		if (subscriberCount() === 0) {
			disconnect();
		}
	};
}
