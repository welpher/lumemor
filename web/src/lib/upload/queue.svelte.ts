// The upload queue of the web app (the M2d plan's Task 5): a
// module-level singleton over the three-state upload contract,
//
//   POST /api/v1/assets            (one photo per request, multipart)
//   state: created (201) | duplicate (200) | trashed (200)
//
// with the pinned conflict semantics: a `trashed` answer parks the
// item and opens the dialog, whose restore choice RESTORES the
// existing row through the batch endpoint and whose save-as choice
// permanently DELETEs it (`DELETE /assets/{id}`, the trash bypass)
// and re-runs the upload from the same File. Live Photo pairing
// videos are captured by devices; the web picker always uploads a
// single `photo` part and never sends `liveVideo` (the contract
// marks it optional).
//
// ARCHITECTURE
//
// - The queue is a module singleton (like the events module): items
//   live across route navigation, the overlay component (the
//   (content) shell) renders them, and the top-bar upload button and
//   the full-page drop zone both enqueue into the same instance.
//   Consumers read the reactive getters directly (the session
//   store's rune-module pattern -- a getter called in a template
//   expression is tracked).
// - The transport is XHR, not fetch: fetch has no upload progress,
//   and the overlay's determinate bar needs `upload.onprogress`.
// - Concurrency is capped at three in flight, strictly FIFO by
//   enqueue order; the pacer starts the next queued item the moment
//   a slot frees.
// - The uploading app is content-addressed by the SERVER (it
//   checksums the stream): the client sends no checksum part. A
//   `created` answer leaves the item `processing` until the SSE
//   `asset.ready` for that asset id arrives; `duplicate` is done
//   immediately (a live asset is ready by definition).
// - Each File object is enqueued verbatim; the Review Focus ③
//   save-as semantics re-upload the very same bytes.
//
// THE STATE MACHINE (per item)
//
//   queued -> uploading -> created  -> processing -> done (SSE ready)
//                       -> duplicate -> done
//                       -> trashed   -> conflict -> {restore -> done
//                                                 | save-as (DELETE
//                                                   + new upload)}
//                       -> failure   -> failed  -> retry (re-queue)
//   any uploading state: dismiss = abort and remove
//
// The extension check on enqueue is the hint-level front end of the
// server's upload whitelist (`server/internal/storage/upload.go`,
// format set v1); the server stays authoritative forever.
//
// Reactive state is Svelte 5 rune module state (the session store's
// pattern); tests build isolated instances through
// `createUploadQueue` with injected transport and id/timezone stubs.

import { apiFetch, errorEnvelope } from '$lib/auth/api';
import { subscribeAssetEvents } from '$lib/events';
import { forceLogout } from '$lib/auth/session.svelte';
import type { components } from '../../../../openapi/generated/web/api';

/** One upload response body, exactly the contract's AssetResponse. */
type UploadAsset = components['schemas']['AssetResponse'];

/** The states of one queue item (see the state machine above). */
export type UploadItemState =
	| 'queued'
	| 'uploading'
	| 'conflict'
	| 'processing'
	| 'done'
	| 'failed';

/** One queue row as the overlay renders it. */
export type UploadItem = {
	/** The queue's own key (an injected unique id per enqueue). */
	id: string;
	/** The original file name (the contract's `fileName` part). */
	fileName: string;
	/** The file's byte size (the progress denominator). */
	size: number;
	/** The file object itself; untouched until the bytes go on the wire. */
	file: File;
	state: UploadItemState;
	/** XHR-reported progress, 0..1 (the determinate bar). */
	progress: number;
	/** The failed state's i18n key ('' otherwise). */
	errorKey: string;
	/** The last upload response (the conflict resolution reads its id). */
	asset: UploadAsset | null;
};

/** The XHR surface the queue uses (injected as a test stub). */
export type XHRLike = {
	upload: {
		onprogress:
			| ((event: { loaded: number; total: number; lengthComputable: boolean }) => void)
			| null;
	};
	onload: (() => void) | null;
	onerror: (() => void) | null;
	onabort: (() => void) | null;
	status: number;
	responseText: string;
	open: (method: string, url: string) => void;
	send: (body: FormData) => void;
	abort: () => void;
};

export type UploadQueueDeps = {
	xhrFactory: () => XHRLike;
	/** Unique id per queue item (tests inject a counter). */
	generateId: () => string;
	/** The capture time zone hint (the contract's optional part). */
	timeZone: () => string | null;
	/** The mid-session 401 reaction (the singleton wires forceLogout). */
	onUnauthorized: () => void;
	/** The fetch half (batch restore / permanent delete). */
	fetcher: typeof fetch;
};

/** The pinned concurrency cap of the M2d plan. */
export const UPLOAD_CONCURRENCY = 3;

/** The server's upload whitelist, format set v1 (the hint-level mirror). */
const UPLOAD_EXTENSIONS = new Set([
	'jpg',
	'jpeg',
	'png',
	'gif',
	'webp',
	'tif',
	'tiff',
	'heic',
	'avif',
	'mp4',
	'mov',
	'webm',
	'mkv'
]);

/** The file extension of a name, lowercase and dotless; '' when none. */
export function uploadExtensionOf(fileName: string): string {
	const dot = fileName.lastIndexOf('.');
	if (dot <= 0 || dot === fileName.length - 1) {
		return '';
	}
	return fileName.slice(dot + 1).toLowerCase();
}

/** Whether the file name passes the hint-level whitelist. */
export function uploadFormatSupported(fileName: string): boolean {
	return UPLOAD_EXTENSIONS.has(uploadExtensionOf(fileName));
}

export type UploadQueue = {
	/** The live queue, newest last (reactive through the getter). */
	items: () => readonly UploadItem[];
	/** How many items are in flight or waiting for a slot. */
	activeCount: () => number;
	/** The conflict item holding the dialog right now (null when none). */
	conflictItemId: () => string | null;
	/** The last failed conflict action's key ('' when none). */
	conflictErrorKey: () => string;
	/** Enqueue picked or dropped files; hint-rejects bad extensions. */
	enqueueFiles: (files: File[] | FileList) => void;
	/** Open the conflict dialog for a parked row (the resolve affordance). */
	openConflict: (itemId: string) => void;
	/** Close the dialog; the row stays parked for a later decision. */
	closeConflict: () => void;
	/** The dialog's keep choice: restore the trashed row (batch). */
	resolveByRestore: (itemId: string) => Promise<void>;
	/** The dialog's replace choice: permanent delete, then re-upload. */
	resolveBySaveAs: (itemId: string) => Promise<void>;
	/** Re-run a failed row (a fresh upload of the same file). */
	retry: (itemId: string) => void;
	/** Remove one row, aborting an in-flight upload first. */
	dismiss: (itemId: string) => void;
};

/** Build one isolated queue (tests inject the transport stubs; the
 * app's singleton lives at the bottom of this module). */
export function createUploadQueue(deps: UploadQueueDeps): UploadQueue {
	let items = $state<UploadItem[]>([]);
	// The XHRs of the in-flight rows (abort on dismissal).
	const inFlight = new Map<string, XHRLike>();
	// The conflict row that holds the dialog right now (null when closed).
	let conflictItem = $state<string | null>(null);
	let conflictErrorKey = $state('');

	// The SSE half: subscribed from the first enqueue, released when
	// the queue drains.
	let unsubscribeEvents: (() => void) | null = null;

	function ensureEventSubscription(): void {
		if (unsubscribeEvents === null) {
			unsubscribeEvents = subscribeAssetEvents(handleAssetEvent);
		}
	}

	function releaseEventSubscription(): void {
		if (unsubscribeEvents !== null && items.length === 0) {
			unsubscribeEvents();
			unsubscribeEvents = null;
		}
	}

	function handleAssetEvent(event: { name: string; assetId: string }): void {
		let released = false;
		items = items.flatMap((item) => {
			if (item.asset === null || item.asset.id !== event.assetId) {
				return [item];
			}
			if (event.name === 'asset.ready') {
				// The pipeline finished the upload's asset row: the
				// item is done. (asset.ready is the only readiness
				// signal; the server never publishes asset.updated
				// from the pipeline, so a hide/unhide of a row still
				// processing must not counterfeit completion.)
				if (item.state === 'processing') {
					return [{ ...item, state: 'done' as const }];
				}
				return [item];
			}
			// asset.deleted / asset.trashed: the just-uploaded asset
			// is gone (another device's decision); the row leaves the
			// queue. Every other event name -- uploaded (the queue
			// row already knows), restored (the queue row already
			// patched done), updated (visibility, irrelevant here) --
			// leaves the row exactly where it is: the queue is not a
			// view of the wall, only readiness and absence matter.
			if (
				(event.name === 'asset.deleted' || event.name === 'asset.trashed') &&
				(item.state === 'processing' || item.state === 'done')
			) {
				released = true;
				return [];
			}
			return [item];
		});
		if (released) {
			releaseEventSubscription();
		}
	}

	/** The fetch-based conflict/deletion calls share the interceptor. */
	async function call(
		path: string,
		init: RequestInit
	): Promise<{ ok: boolean; errorKey: string; body: unknown }> {
		try {
			const response = await apiFetch(path, init, deps.fetcher);
			if (response.ok) {
				let body: unknown = null;
				try {
					// 204 answers (the delete) carry no body.
					body = (await response.json()) as unknown;
				} catch {
					// Not JSON: the delete's empty answer.
				}
				return { ok: true, errorKey: '', body };
			}
			if (response.status === 401) {
				// The session died mid-decision: the store already
				// flipped (the interceptor) and the guard redirects;
				// the dialog shows the session message.
				deps.onUnauthorized();
				return { ok: false, errorKey: 'session_expired', body: null };
			}
			const envelope = await errorEnvelope(response);
			return {
				ok: false,
				errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal',
				body: null
			};
		} catch {
			return { ok: false, errorKey: 'internal', body: null };
		}
	}

	function patch(itemId: string, changes: Partial<UploadItem>): void {
		items = items.map((item) => (item.id === itemId ? { ...item, ...changes } : item));
	}

	/** The pacer: start queued work while slots are free (FIFO). */
	function pump(): void {
		for (const item of items) {
			if (inFlight.size >= UPLOAD_CONCURRENCY) {
				return;
			}
			if (item.state === 'queued') {
				startUpload(item);
			}
		}
	}

	function startUpload(item: UploadItem): void {
		const form = new FormData();
		// Part order is contract: every text field precedes the file
		// part, and the server validates the order.
		form.append('deviceAssetId', deps.generateId());
		form.append('fileName', item.fileName);
		const capturedAt = new Date(item.file.lastModified);
		if (!Number.isNaN(capturedAt.getTime())) {
			form.append('deviceCreatedAt', capturedAt.toISOString());
		}
		const zone = deps.timeZone();
		if (zone !== null && zone !== '') {
			form.append('deviceTimeZone', zone);
		}
		form.append('photo', item.file, item.fileName);

		const xhr = deps.xhrFactory();
		inFlight.set(item.id, xhr);
		patch(item.id, { state: 'uploading', progress: 0 });

		xhr.upload.onprogress = (event) => {
			if (event.lengthComputable && event.total > 0) {
				patch(item.id, { progress: Math.min(1, event.loaded / event.total) });
			}
		};
		xhr.onload = () => void settleUpload(item.id, xhr.status, xhr.responseText);
		xhr.onerror = () => {
			inFlight.delete(item.id);
			patch(item.id, { state: 'failed', errorKey: 'internal' });
			pump();
		};
		xhr.onabort = () => {
			// The user dismissed the row (dismiss removed it before
			// calling abort); free the slot and pace on.
			inFlight.delete(item.id);
			pump();
		};
		xhr.open('POST', '/api/v1/assets');
		xhr.send(form);
	}

	async function settleUpload(itemId: string, status: number, responseText: string): Promise<void> {
		inFlight.delete(itemId);
		pump();
		const item = items.find((candidate) => candidate.id === itemId);
		if (item === undefined) {
			// A settled response lost the race with a dismiss.
			return;
		}
		let body: unknown;
		try {
			body = JSON.parse(responseText) as unknown;
		} catch {
			patch(itemId, { state: 'failed', errorKey: 'internal' });
			return;
		}
		const envelope =
			body !== null && typeof body === 'object' && typeof (body as { code?: unknown }).code === 'string'
				? (body as { code: string })
				: null;
		const asset = body as Partial<UploadAsset>;
		if (status === 401) {
			// The XHR path bypasses the fetch interceptor: react
			// manually, with the same session message.
			deps.onUnauthorized();
			patch(itemId, { state: 'failed', errorKey: 'session_expired' });
			return;
		}
		if (typeof asset.id !== 'string') {
			// An error envelope (or any other non-asset body): the
			// endpoint's known codes map by the lowercase convention.
			patch(itemId, {
				state: 'failed',
				errorKey: envelope !== null ? envelope.code.toLowerCase() : 'internal'
			});
			return;
		}
		if (status === 201 || (status === 200 && asset.state === 'duplicate')) {
			// created: the pipeline is still working (isReady lives on
			// the detail, not here) -- wait for the SSE ready event.
			// duplicate: an identical LIVE asset exists; done.
			patch(itemId, {
				state: status === 201 ? 'processing' : 'done',
				errorKey: '',
				progress: 1,
				asset: body as UploadAsset
			});
			return;
		}
		if (status === 200 && asset.state === 'trashed') {
			patch(itemId, {
				state: 'conflict',
				errorKey: '',
				progress: 1,
				asset: body as UploadAsset
			});
			// Open the decision dialog for the first parked row; later
			// ones stay parked with their own resolve affordance.
			if (conflictItem === null) {
				conflictItem = itemId;
			}
			return;
		}
		// An unrecognized success body: the generic message.
		patch(itemId, { state: 'failed', errorKey: 'internal' });
	}

	function enqueueFiles(files: File[] | FileList): void {
		for (const file of Array.from(files)) {
			const fileName = typeof file.name === 'string' ? file.name : '';
			const extension = uploadExtensionOf(fileName);
			const unsupported = !uploadFormatSupported(fileName);
			items = [
				...items,
				{
					id: deps.generateId(),
					fileName,
					size: typeof file.size === 'number' ? file.size : 0,
					file,
					state: unsupported ? 'failed' : 'queued',
					progress: 0,
					errorKey: unsupported ? 'validation_failed' : '',
					asset: null
				}
			];
			ensureEventSubscription();
		}
		pump();
	}

	function dismiss(itemId: string): void {
		const xhr = inFlight.get(itemId);
		inFlight.delete(itemId);
		items = items.filter((item) => item.id !== itemId);
		if (conflictItem === itemId) {
			conflictItem = null;
		}
		if (xhr !== undefined) {
			xhr.abort();
		}
		releaseEventSubscription();
	}

	return {
		items: () => items,
		activeCount: () =>
			items.filter((item) => item.state === 'uploading' || item.state === 'queued').length,
		conflictItemId: () => conflictItem,
		conflictErrorKey: () => conflictErrorKey,
		enqueueFiles,
		openConflict: (itemId: string) => {
			conflictErrorKey = '';
			conflictItem = itemId;
		},
		closeConflict: () => {
			conflictErrorKey = '';
			conflictItem = null;
		},
		resolveByRestore: async (itemId: string) => {
			const item = items.find((candidate) => candidate.id === itemId);
			if (item === undefined || item.state !== 'conflict' || item.asset === null) {
				return;
			}
			const outcome = await call('/api/v1/assets/batch', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ action: 'restore', ids: [item.asset.id] })
			});
			if (outcome.ok) {
				const report = outcome.body as { missing?: unknown } | null;
				const missing = Array.isArray(report?.missing) ? (report.missing as unknown[]) : [];
				if (missing.includes(item.asset.id)) {
					// Set semantics: the id did not resolve -- the
					// trashed copy was purged by another device while
					// the decision was parked. The restore is a dead
					// end; keep the row parked with the honest report.
					conflictErrorKey = 'asset_not_found';
					return;
				}
				// The trashed asset is live again: the batch success is
				// the answer (the SSE restored event only re-inserts the
				// wall's tile -- the queue row stays done until
				// dismissed, like every terminal row).
				patch(itemId, { state: 'done', errorKey: '', progress: 1 });
				conflictErrorKey = '';
				// Only the row that holds the dialog releases it: a
				// late success of row A must not close row B's dialog
				// that the user opened meanwhile.
				if (conflictItem === itemId) {
					conflictItem = null;
				}
				return;
			}
			// Keep the rows parked; the dialog renders the failure.
			conflictErrorKey = outcome.errorKey;
		},
		resolveBySaveAs: async (itemId: string) => {
			const item = items.find((candidate) => candidate.id === itemId);
			if (item === undefined || item.state !== 'conflict' || item.asset === null) {
				return;
			}
			const outcome = await call(`/api/v1/assets/${encodeURIComponent(item.asset.id)}`, {
				method: 'DELETE'
			});
			if (!outcome.ok && outcome.errorKey !== 'asset_not_found') {
				conflictErrorKey = outcome.errorKey;
				return;
			}
			// A 404 answer means the trashed copy was purged by
			// another device while the decision was parked: the slot
			// is already free, so the re-upload proceeds the same.
			// The old row is permanently gone either way (the
			// content-addressed cleanup rule covers cross-owner
			// references), and the SAME file re-uploads as a fresh
			// asset.
			dismiss(itemId);
			enqueueFiles([item.file]);
		},
		retry: (itemId: string) => {
			const item = items.find((candidate) => candidate.id === itemId);
			if (
				item === undefined ||
				item.state !== 'failed' ||
				// The hint-level whitelist re-validates here too: a
				// hint-rejected row stays failed (the server would
				// reject the bytes again).
				!uploadFormatSupported(item.fileName)
			) {
				return;
			}
			patch(itemId, { state: 'queued', errorKey: '', progress: 0 });
			pump();
		},
		dismiss
	};
}

// The app's singleton queue (the module-level instance the overlay,
// the top-bar button and the drop zone all share).

let idCounter = 0;

const singleton = createUploadQueue({
	// The real XHR satisfies the injected surface structurally.
	xhrFactory: () => new XMLHttpRequest() as unknown as XHRLike,
	generateId: () => {
		idCounter += 1;
		return `web-${Date.now().toString(16)}-${idCounter}`;
	},
	timeZone: () => Intl.DateTimeFormat().resolvedOptions().timeZone ?? null,
	onUnauthorized: forceLogout,
	fetcher: globalThis.fetch
});

export const uploadQueueItems = singleton.items;
export const uploadQueueConflictId = singleton.conflictItemId;
export const uploadQueueConflictErrorKey = singleton.conflictErrorKey;
export const enqueueUploadFiles = singleton.enqueueFiles;
export const openUploadConflict = singleton.openConflict;
export const closeUploadConflict = singleton.closeConflict;
export const resolveUploadByRestore = singleton.resolveByRestore;
export const resolveUploadBySaveAs = singleton.resolveBySaveAs;
export const retryUpload = singleton.retry;
export const dismissUpload = singleton.dismiss;
