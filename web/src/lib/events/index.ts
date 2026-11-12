// The shared server-sent-events subscription of the web app.
//
// One refcounted EventSource on GET /api/v1/events feeds every
// consumer of the instance activity stream: the timeline wall and
// the upload queue overlay (asset.* through `subscribeAssetEvents`),
// the admin surfaces' live progress (job.* through
// `subscribeJobEvents`), the people surface's convergence
// (person.* and face.clustered through `subscribePersonEvents` /
// `subscribeFaceEvents`), the albums surface's convergence
// (album.* through `subscribeAlbumEvents`) all subscribe and
// unsubscribe on unmount; the first subscriber of any kind opens the
// stream, the last one closes it.
//
// RECONNECT SEMANTICS: the browser's EventSource owns them. On a
// network drop it reconnects on its own and resends the
// `Last-Event-ID` header with the id of the last received event; the
// server replays everything newer than that id from its five-minute
// ring buffer and skips what fell out (the contract in
// openapi/lumemor.yaml). This module adds no reconnect logic of its
// own on purpose -- hand-rolling it would only diverge from the
// browser's. It does OBSERVE the browser's connection state (the
// onopen/onerror handlers below) and report it into the connection
// store, so the offline strip can tell
// the user the truth: the stream is down, the browser is retrying,
// it recovered. Observation only -- the retry schedule stays the
// browser's secret (JS cannot read it), which is why the strip
// counts the elapsed reconnecting seconds, never a fabricated
// countdown.
//
// Event names arrive as SSE `event:` lines, so only
// addEventListener(name) receives them (the default `onmessage`
// handler never fires for named events). The asset.* payloads the
// server publishes are `{assetId, kind}` objects, the job.* payloads
// are `{jobRunId, kind, total, done, failed, state}` maps, the
// person.* payloads carry the cluster (or face) id beside the acting
// user's id (label and hidden state are private per user, so the
// subscriber filters by what it renders, never by another account's
// change), and face.clustered carries the library's cluster totals
// -- every value a string on the wire (the server publishes
// map[string]string), the counts parsing through Number(); anything
// else on the wire is ignored defensively rather than crashing
// subscribers.

import { markStreamClosed, markStreamConnecting, markStreamDown, markStreamOpen } from './connection.svelte';

export type AssetKind = 'image' | 'video';

/** The asset lifecycle events the server publishes on the stream. */
export type AssetEventName =
	'asset.uploaded' | 'asset.ready' | 'asset.updated' | 'asset.deleted' | 'asset.trashed' | 'asset.restored';

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
	/** The run's kind (model_download, the rerun kinds, the album rescan). */
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

/**
 * The people surface's person.* events: the label, hidden, merge,
 * face-move, face-creation and face-detach writes. Every payload
 * carries the acting user's id beside the cluster (or face) id
 * ONLY: the label text and the hidden switch are one account's
 * private state, so they never ride the shared stream -- a
 * subscriber filters by the ids it renders, never by another
 * account's change, and the SERVER's answer decides what actually
 * moved.
 *
 * `person.merged` carries the dissolved sources comma-joined on the
 * wire and split here. `person.face_clustered` carries the NEWLY
 * CREATED cluster's id; `person.face_detached` carries the cluster
 * the face LEFT (the face holds no cluster after that write, and
 * the converging surfaces need the person whose membership
 * changed).
 */
export type PersonEventName =
	| 'person.labeled'
	| 'person.unlabeled'
	| 'person.hidden'
	| 'person.merged'
	| 'person.face_moved'
	| 'person.face_clustered'
	| 'person.face_detached';

export type PersonEvent =
	| {
			/** The server's monotonically increasing event id (the resumption cursor). */
			id: number;
			name: 'person.labeled';
			clusterId: string;
			userId: string;
	  }
	| {
			id: number;
			name: 'person.unlabeled';
			clusterId: string;
			userId: string;
	  }
	| {
			id: number;
			name: 'person.hidden';
			clusterId: string;
			userId: string;
	  }
	| {
			id: number;
			name: 'person.merged';
			/** The surviving cluster that absorbed the sources. */
			survivingClusterId: string;
			/** The dissolved source clusters. */
			mergedClusterIds: string[];
			userId: string;
	  }
	| {
			id: number;
			name: 'person.face_moved';
			faceId: string;
			/** The cluster the face occupies after the move. */
			clusterId: string;
			userId: string;
	  }
	| {
			id: number;
			name: 'person.face_clustered';
			faceId: string;
			/** The cluster the creation built (the face's new home). */
			clusterId: string;
			userId: string;
	  }
	| {
			id: number;
			name: 'person.face_detached';
			faceId: string;
			/** The cluster the face left (its membership is gone after the write). */
			clusterId: string;
			userId: string;
	  };

export type PersonEventListener = (event: PersonEvent) => void;

/** The face clustering events the server publishes after a recognition run. */
export type FaceEventName = 'face.clustered';

/**
 * One parsed face.clustered event: the library's cluster totals
 * after a recognition run (all strings on the wire, parsed to
 * non-negative integers here). The payload carries no ids -- the
 * unnamed region converges by refetching, not by patching.
 */
export type FaceEvent = {
	/** The server's monotonically increasing event id (the resumption cursor). */
	id: number;
	name: FaceEventName;
	/** The library's whole face cluster count. */
	totalClusters: number;
	/** The clusters carrying at least one user label. */
	labeledClusters: number;
};

export type FaceEventListener = (event: FaceEvent) => void;

/**
 * The albums surface's album.* events: the creation, the property
 * patch, the delete, the membership curation and the view-ACL
 * sharing writes. Every payload carries the album's id and one
 * user's id ONLY: titles, covers, the lock state and
 * the membership itself are album state the subscribers refetch,
 * never bus payload. On the two share events the `userId` is the
 * GRANTEE whose view of the album changed (the grantor is derivable
 * server-side); on the other four it is the acting user.
 */
export type AlbumEventName =
	'album.created' | 'album.updated' | 'album.deleted' | 'album.items-changed' | 'album.shared' | 'album.unshared';

/** One parsed album.* activity event, as the wall and detail consume it. */
export type AlbumEvent = {
	/** The server's monotonically increasing event id (the resumption cursor). */
	id: number;
	name: AlbumEventName;
	/** The album the write touched. */
	albumId: string;
	/** The user the write concerns: the grantee on the share events, the actor otherwise. */
	userId: string;
};

export type AlbumEventListener = (event: AlbumEvent) => void;

const EVENTS_PATH = '/api/v1/events';

const ASSET_EVENT_NAMES: readonly AssetEventName[] = [
	'asset.uploaded',
	'asset.ready',
	'asset.updated',
	'asset.deleted',
	'asset.trashed',
	'asset.restored'
];

const JOB_EVENT_NAMES: readonly JobEventName[] = ['job.started', 'job.progress', 'job.finished'];

const JOB_RUN_STATES: readonly JobRunState[] = ['queued', 'running', 'succeeded', 'failed'];

const PERSON_EVENT_NAMES: readonly PersonEventName[] = [
	'person.labeled',
	'person.unlabeled',
	'person.hidden',
	'person.merged',
	'person.face_moved',
	'person.face_clustered',
	'person.face_detached'
];

const FACE_EVENT_NAMES: readonly FaceEventName[] = ['face.clustered'];

const ALBUM_EVENT_NAMES: readonly AlbumEventName[] = [
	'album.created',
	'album.updated',
	'album.deleted',
	'album.items-changed',
	'album.shared',
	'album.unshared'
];

const assetListeners = new Set<AssetEventListener>();
const jobListeners = new Set<JobEventListener>();
const personListeners = new Set<PersonEventListener>();
const faceListeners = new Set<FaceEventListener>();
const albumListeners = new Set<AlbumEventListener>();
let source: EventSource | null = null;

/** Parse the shared id + JSON payload half of one SSE block; null when malformed. */
function baseEvent(message: MessageEvent): { id: number; payload: unknown } | null {
	const id = Number.parseInt(message.lastEventId, 10);
	if (!Number.isFinite(id) || id < 0) {
		return null;
	}
	try {
		const payload = JSON.parse(String(message.data));
		if (payload !== null && typeof payload === 'object') {
			return { id, payload };
		}
	} catch {
		// Not JSON: the block is not for us.
	}
	return null;
}

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
 * Parse one SSE block into a PersonEvent; null when the block is not
 * the payload shape this module subscribes to. The wire payload is
 * all strings (the server publishes map[string]string), so the
 * hidden switch parses from 'true'/'false' and the merged sources
 * split on the comma -- anything else means the block is not for us.
 */
function parsePersonEvent(name: PersonEventName, message: MessageEvent): PersonEvent | null {
	const base = baseEvent(message);
	if (base === null) {
		return null;
	}
	const payload = base.payload as {
		clusterId?: unknown;
		userId?: unknown;
		survivingClusterId?: unknown;
		mergedClusterIds?: unknown;
		faceId?: unknown;
	};
	const { clusterId, userId } = payload;
	if (typeof userId !== 'string' || userId === '') {
		return null;
	}
	switch (name) {
		case 'person.labeled': {
			if (typeof clusterId !== 'string' || clusterId === '') {
				return null;
			}
			return { id: base.id, name, clusterId, userId };
		}
		case 'person.unlabeled': {
			if (typeof clusterId !== 'string' || clusterId === '') {
				return null;
			}
			return { id: base.id, name, clusterId, userId };
		}
		case 'person.hidden': {
			if (typeof clusterId !== 'string' || clusterId === '') {
				return null;
			}
			return { id: base.id, name, clusterId, userId };
		}
		case 'person.merged': {
			if (typeof payload.survivingClusterId !== 'string' || payload.survivingClusterId === '') {
				return null;
			}
			if (typeof payload.mergedClusterIds !== 'string' || payload.mergedClusterIds === '') {
				return null;
			}
			const mergedClusterIds = payload.mergedClusterIds.split(',');
			if (mergedClusterIds.some((merged) => merged === '')) {
				return null;
			}
			return { id: base.id, name, survivingClusterId: payload.survivingClusterId, mergedClusterIds, userId };
		}
		case 'person.face_moved': {
			if (typeof payload.faceId !== 'string' || payload.faceId === '') {
				return null;
			}
			if (typeof clusterId !== 'string' || clusterId === '') {
				return null;
			}
			return { id: base.id, name, faceId: payload.faceId, clusterId, userId };
		}
		// The creation and the detach share the moved face's wire
		// shape: the face id beside ONE cluster id (the new cluster
		// for the creation, the vacated one for the detach).
		case 'person.face_clustered':
		case 'person.face_detached': {
			if (typeof payload.faceId !== 'string' || payload.faceId === '') {
				return null;
			}
			if (typeof clusterId !== 'string' || clusterId === '') {
				return null;
			}
			return { id: base.id, name, faceId: payload.faceId, clusterId, userId };
		}
	}
}

/**
 * Parse one SSE block into a FaceEvent; null when the block is not
 * the payload shape this module subscribes to. The totals are
 * all-strings on the wire and must parse to non-negative integers.
 */
function parseFaceEvent(name: FaceEventName, message: MessageEvent): FaceEvent | null {
	const base = baseEvent(message);
	if (base === null) {
		return null;
	}
	const { totalClusters, labeledClusters } = base.payload as {
		totalClusters?: unknown;
		labeledClusters?: unknown;
	};
	if (typeof totalClusters !== 'string' || typeof labeledClusters !== 'string') {
		return null;
	}
	const totals = [Number(totalClusters), Number(labeledClusters)];
	for (const total of totals) {
		if (!Number.isInteger(total) || total < 0) {
			return null;
		}
	}
	return { id: base.id, name, totalClusters: totals[0], labeledClusters: totals[1] };
}

/**
 * Parse one SSE block into an AlbumEvent; null when the block is not
 * the payload shape this module subscribes to. The wire payload is
 * an all-strings map carrying the album's id and one user's id --
 * ids only, never album state.
 */
function parseAlbumEvent(name: AlbumEventName, message: MessageEvent): AlbumEvent | null {
	const base = baseEvent(message);
	if (base === null) {
		return null;
	}
	const { albumId, userId } = base.payload as { albumId?: unknown; userId?: unknown };
	if (typeof albumId !== 'string' || albumId === '') {
		return null;
	}
	if (typeof userId !== 'string' || userId === '') {
		return null;
	}
	return { id: base.id, name, albumId, userId };
}

/**
 * Open the shared stream when the first listener arrives. The
 * EventSource constructor is read at call time (never at import
 * time) so test environments without one keep every subscriber inert
 * instead of throwing at module load. The connection store joins the
 * ride from here: the construction reports 'connecting', the
 * browser's onopen reports the landed connection, its onerror
 * reports the lost one (the browser owns the retry; see the header).
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
	markStreamConnecting();
	source.onopen = () => {
		markStreamOpen();
	};
	source.onerror = () => {
		markStreamDown();
	};
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
	for (const name of PERSON_EVENT_NAMES) {
		source.addEventListener(name, (message: MessageEvent) => {
			const event = parsePersonEvent(name, message);
			if (event === null) {
				return;
			}
			for (const listener of personListeners) {
				listener(event);
			}
		});
	}
	for (const name of FACE_EVENT_NAMES) {
		source.addEventListener(name, (message: MessageEvent) => {
			const event = parseFaceEvent(name, message);
			if (event === null) {
				return;
			}
			for (const listener of faceListeners) {
				listener(event);
			}
		});
	}
	for (const name of ALBUM_EVENT_NAMES) {
		source.addEventListener(name, (message: MessageEvent) => {
			const event = parseAlbumEvent(name, message);
			if (event === null) {
				return;
			}
			for (const listener of albumListeners) {
				listener(event);
			}
		});
	}
}

/**
 * Close the shared stream (the last listener left, or the app reset).
 * The close is intentional, so the connection store reports 'closed'
 * -- nothing is lost when nothing is live, and the offline strip
 * falls with it.
 */
function disconnect(): void {
	source?.close();
	source = null;
	markStreamClosed();
}

/** The number of subscribers across all event kinds. */
function subscriberCount(): number {
	return assetListeners.size + jobListeners.size + personListeners.size + faceListeners.size + albumListeners.size;
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

/**
 * Subscribe to the parsed person.* events of the activity stream --
 * the people surface's label, hidden, merge and face-move writes,
 * riding the SAME refcounted EventSource as every other subscriber.
 * Returns the unsubscribe function; the stream closes when the last
 * subscriber of any kind unsubscribes.
 */
export function subscribePersonEvents(listener: PersonEventListener): () => void {
	personListeners.add(listener);
	connect();
	return () => {
		personListeners.delete(listener);
		if (subscriberCount() === 0) {
			disconnect();
		}
	};
}

/**
 * Subscribe to the parsed face.clustered events of the activity
 * stream -- the recognition runs' cluster totals, riding the SAME
 * refcounted EventSource as every other subscriber. Returns the
 * unsubscribe function; the stream closes when the last subscriber
 * of any kind unsubscribes.
 */
export function subscribeFaceEvents(listener: FaceEventListener): () => void {
	faceListeners.add(listener);
	connect();
	return () => {
		faceListeners.delete(listener);
		if (subscriberCount() === 0) {
			disconnect();
		}
	};
}

/**
 * Subscribe to the parsed album.* events of the activity stream --
 * the albums surface's creation, property, delete, curation and
 * sharing writes, riding the SAME refcounted EventSource as every
 * other subscriber. Returns the unsubscribe function; the stream
 * closes when the last subscriber of any kind unsubscribes.
 */
export function subscribeAlbumEvents(listener: AlbumEventListener): () => void {
	albumListeners.add(listener);
	connect();
	return () => {
		albumListeners.delete(listener);
		if (subscriberCount() === 0) {
			disconnect();
		}
	};
}
