import { beforeEach, expect, test, vi } from 'vitest';
import {
	createUploadQueue,
	uploadExtensionOf,
	UPLOAD_CONCURRENCY,
	type UploadItem,
	type XHRLike
} from './queue.svelte';

// The queue module contract, driven entirely through injected
// stubs (the XHR transport, the fetch half, the ids, the time zone
// and the SSE subscription):
//
//   - the multipart part order and content the upload contract pins
//     (text fields first, one `photo` part, no `liveVideo`, no
//     client-side checksum);
//   - the three-state settle: 201 created -> processing until the
//     SSE ready event; 200 duplicate -> done; 200 trashed ->
//     conflict plus the dialog and its two resolution paths
//     (batch restore / permanent DELETE + re-upload);
//   - the envelope codes mapped by the lowercase convention, the
//     401 reaction, the concurrency cap of three and FIFO order;
//   - the dismissal races (a settled response losing to a dismiss
//     must not resurrect the row).

// The SSE half is mocked: the queue subscribes on the first
// enqueue and the tests deliver parsed events through the
// captured listeners.
const eventBox = vi.hoisted(() => ({
	listeners: [] as Array<(event: { name: string; assetId: string }) => void>,
	emit(name: string, assetId: string): void {
		for (const listener of [...this.listeners]) {
			listener({ name, assetId } as Parameters<(typeof this.listeners)[number]>[0]);
		}
	}
}));

vi.mock('$lib/events', () => ({
	subscribeAssetEvents: (listener: (event: { name: string; assetId: string }) => void) => {
		eventBox.listeners.push(listener);
		return () => {
			const index = eventBox.listeners.indexOf(listener);
			if (index !== -1) {
				eventBox.listeners.splice(index, 1);
			}
		};
	}
}));

/** A scripted XMLHttpRequest stand-in: tests drive it directly. */
class StubXHR {
	upload = { onprogress: null as XHRLike['upload']['onprogress'] };
	onload: XHRLike['onload'] = null;
	onerror: XHRLike['onerror'] = null;
	onabort: XHRLike['onabort'] = null;
	status = 0;
	responseText = '';
	openedUrl = '';
	body: FormData | null = null;
	aborted = false;

	open(_method: string, url: string): void {
		this.openedUrl = url;
	}

	send(body: FormData): void {
		this.body = body;
	}

	abort(): void {
		this.aborted = true;
		this.onabort?.();
	}

	/** Test driver: report upload progress. */
	drive(loaded: number, total: number): void {
		this.upload.onprogress?.({ loaded, total, lengthComputable: true });
	}

	/** Test driver: settle the response and fire onload. */
	answer(status: number, body: unknown): void {
		this.status = status;
		this.responseText = JSON.stringify(body);
		this.onload?.();
	}

	/** Test driver: fail at the transport level. */
	fails(): void {
		this.onerror?.();
	}
}

function jsonFetch(status: number, body: unknown): ReturnType<typeof vi.fn> {
	return vi.fn(
		async () =>
			new Response(JSON.stringify(body), {
				status,
				headers: { 'Content-Type': 'application/json' }
			})
	);
}

type Rig = ReturnType<typeof buildQueue>;

function buildQueue(): {
	queue: ReturnType<typeof createUploadQueue>;
	fetch: ReturnType<typeof vi.fn>;
	unauthorized: ReturnType<typeof vi.fn>;
} {
	const xhrs: StubXHR[] = [];
	let counter = 0;
	const fetcherMock = jsonFetch(200, {});
	const unauthorized = vi.fn();
	const queue = createUploadQueue({
		xhrFactory: () => {
			const xhr = new StubXHR();
			xhrs.push(xhr);
			return xhr as unknown as XHRLike;
		},
		generateId: () => `id-${String(++counter)}`,
		timeZone: () => 'Asia/Shanghai',
		onUnauthorized: unauthorized as unknown as () => void,
		fetcher: fetcherMock as unknown as typeof fetch
	});
	// Keep the rig's XHRs reachable for the drivers below.
	(queue as unknown as { xhrs: StubXHR[] }).xhrs = xhrs;
	return { queue, fetch: fetcherMock, unauthorized };
}

function xhrsOf(rig: Rig): StubXHR[] {
	return (rig.queue as unknown as { xhrs: StubXHR[] }).xhrs;
}

function photoFile(name = 'photo.jpg'): File {
	return new File(['jpeg-bytes'], name, {
		type: 'image/jpeg',
		lastModified: Date.parse('2026-10-13T12:30:00Z')
	});
}

const assetBody = (id: string, state: string) => ({
	id,
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image',
	originalFilename: 'photo.jpg',
	fileSize: 10,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
	state,
	createdAt: '2026-10-29T09:12:00Z',
	takenAt: '2026-10-13T12:30:00Z',
	isTrashed: state === 'trashed'
});

beforeEach(() => {
	eventBox.listeners.length = 0;
});

test('uploadExtensionOf lowercases the dotless tail and answers empty unnamed files', () => {
	expect(uploadExtensionOf('IMG_0001.JPG')).toBe('jpg');
	expect(uploadExtensionOf('movie.MOV')).toBe('mov');
	expect(uploadExtensionOf('no-extension')).toBe('');
	expect(uploadExtensionOf('.hidden')).toBe('');
	expect(uploadExtensionOf('ends-with-dot.')).toBe('');
});

test('an unsupported extension parks failed without touching the wire; retry keeps it failed', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([new File(['text'], 'notes.txt')]);
	const [item] = rig.queue.items();
	expect(item.state).toBe('failed');
	expect(item.errorKey).toBe('validation_failed');
	expect(xhrsOf(rig)).toHaveLength(0);

	rig.queue.retry(item.id);
	expect(rig.queue.items()[0]?.state).toBe('failed');
});

test('the upload request carries the contract parts in order and the progress updates', () => {
	const rig = buildQueue();
	const file = photoFile();
	rig.queue.enqueueFiles([file]);
	expect(xhrsOf(rig)).toHaveLength(1);
	const xhr = xhrsOf(rig)[0] as StubXHR;
	expect(xhr.openedUrl).toBe('/api/v1/assets');

	const body = xhr.body as FormData;
	expect(Array.from(body.keys())).toEqual([
		'deviceAssetId',
		'fileName',
		'deviceCreatedAt',
		'deviceTimeZone',
		'photo'
	]);
	expect(body.get('deviceAssetId')).toBe('id-2');
	expect(body.get('fileName')).toBe('photo.jpg');
	expect(body.get('deviceCreatedAt')).toBe('2026-10-13T12:30:00.000Z');
	expect(body.get('deviceTimeZone')).toBe('Asia/Shanghai');
	// No client-side checksum (the server streams it), no liveVideo
	// part (web uploads photos only). The File part keeps the same
	// name/type/size; jsdom's FormData clones the File, so identity
	// is compared through the content it carries.
	expect(body.get('checksum')).toBeNull();
	expect(body.get('liveVideo')).toBeNull();
	const photoPart = body.get('photo') as File;
	expect(photoPart.name).toBe('photo.jpg');
	expect(photoPart.type).toBe('image/jpeg');
	expect(photoPart.size).toBe(10);

	xhr.drive(5, 20);
	expect(rig.queue.items()[0]?.progress).toBeCloseTo(0.25);
});

test('a created answer waits for the SSE ready event; duplicate completes immediately', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile('one.jpg'), photoFile('two.jpg')]);
	const [first, second] = xhrsOf(rig);
	(second as StubXHR).answer(200, assetBody('asset-dup', 'duplicate'));
	const dup = rig.queue.items()[1] as UploadItem;
	expect(dup.state).toBe('done');
	expect(dup.progress).toBe(1);
	expect(dup.asset?.id).toBe('asset-dup');

	(first as StubXHR).answer(201, assetBody('asset-new', 'created'));
	const created = rig.queue.items()[0] as UploadItem;
	expect(created.state).toBe('processing');

	// An unrelated asset id moves nothing.
	eventBox.emit('asset.ready', 'asset-other');
	expect(rig.queue.items()[0]?.state).toBe('processing');

	eventBox.emit('asset.ready', 'asset-new');
	expect(rig.queue.items()[0]?.state).toBe('done');

	// Both rows settled: the subscription released.
	rig.queue.dismiss(created.id);
	rig.queue.dismiss(dup.id);
	expect(eventBox.listeners).toHaveLength(0);
});

test('a trashed answer parks the row and its restore resolves through the batch endpoint', async () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile()]);
	const xhr = xhrsOf(rig)[0] as StubXHR;
	xhr.answer(200, assetBody('asset-trash', 'trashed'));
	const item = rig.queue.items()[0] as UploadItem;
	expect(item.state).toBe('conflict');
	expect(rig.queue.conflictItemId()).toBe(item.id);
	expect(rig.queue.items()).toHaveLength(1);

	rig.fetch.mockImplementation(
		vi.fn(
			async () =>
				new Response(JSON.stringify({ affected: 1, missing: [] }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		)
	);
	await rig.queue.resolveByRestore(item.id);
	expect((rig.fetch as ReturnType<typeof vi.fn>).mock.calls[0]?.[0]).toBe('/api/v1/assets/batch');
	expect(JSON.parse(String((rig.fetch as ReturnType<typeof vi.fn>).mock.calls[0]?.[1]?.body))).toEqual(
		{ action: 'restore', ids: ['asset-trash'] }
	);
	expect(rig.queue.items()[0]?.state).toBe('done');
	// The dialog closed on success.
	expect(rig.queue.conflictItemId()).toBeNull();
});

test('a restore the set semantics could not resolve keeps the row parked under asset_not_found', async () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile()]);
	const xhr = xhrsOf(rig)[0] as StubXHR;
	xhr.answer(200, assetBody('asset-trash', 'trashed'));
	const item = rig.queue.items()[0] as UploadItem;

	rig.fetch.mockImplementation(
		vi.fn(
			async () =>
				new Response(JSON.stringify({ affected: 0, missing: ['asset-trash'] }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		)
	);
	await rig.queue.resolveByRestore(item.id);
	expect(rig.queue.items()[0]?.state).toBe('conflict');
	expect(rig.queue.conflictItemId()).toBe(item.id);
	expect(rig.queue.conflictErrorKey()).toBe('asset_not_found');
});

test("a restore success of one row never closes another row's open dialog", async () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile('one.jpg'), photoFile('two.jpg')]);
	const [first, second] = xhrsOf(rig) as StubXHR[];
	(first as StubXHR).answer(200, assetBody('asset-1', 'trashed'));
	(second as StubXHR).answer(200, assetBody('asset-2', 'trashed'));
	const [rowOne, rowTwo] = rig.queue.items();
	expect(rig.queue.conflictItemId()).toBe(rowOne?.id);

	// The user decides row B while row A's decision settles late.
	rig.queue.openConflict(rowTwo?.id as string);
	rig.fetch.mockImplementation(
		vi.fn(
			async () =>
				new Response(JSON.stringify({ affected: 1, missing: [] }), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		)
	);
	await rig.queue.resolveByRestore(rowOne?.id as string);
	expect(rig.queue.items()[0]?.state).toBe('done');
	// Row B still holds the dialog.
	expect(rig.queue.conflictItemId()).toBe(rowTwo?.id);
});

test('save-as permanently deletes the trashed copy and re-uploads the same File', async () => {
	const rig = buildQueue();
	const file = photoFile();
	rig.queue.enqueueFiles([file]);
	const [first, ...rest] = xhrsOf(rig);
	expect(rest).toHaveLength(0);
	(first as StubXHR).answer(200, assetBody('asset-trash', 'trashed'));
	const item = rig.queue.items()[0] as UploadItem;

	rig.fetch.mockImplementation(
		vi.fn(async () => new Response(null, { status: 204 }))
	);
	await rig.queue.resolveBySaveAs(item.id);
	const calls = (rig.fetch as ReturnType<typeof vi.fn>).mock.calls;
	expect(calls).toHaveLength(1);
	expect(calls[0]?.[0]).toBe('/api/v1/assets/asset-trash');
	expect((calls[0]?.[1] as RequestInit).method).toBe('DELETE');

	// The old row is gone; a new one queued (already running, the
	// slot was free) with the SAME File bytes.
	const rows = rig.queue.items();
	expect(rows).toHaveLength(1);
	expect(rows[0]?.id).not.toBe(item.id);
	expect(rows[0]?.state).toBe('uploading');
	const replacement = xhrsOf(rig)[1] as StubXHR;
	const reupload = replacement.body?.get('photo') as File;
	expect(reupload.name).toBe('photo.jpg');
	expect(reupload.size).toBe(10);
});

test('a failed conflict action keeps the dialog open with the error key; closing works', async () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile()]);
	const xhr = xhrsOf(rig)[0] as StubXHR;
	xhr.answer(200, assetBody('asset-trash', 'trashed'));
	const item = rig.queue.items()[0] as UploadItem;

	rig.fetch.mockImplementation(jsonFetch(500, { code: 'INTERNAL' }));
	await rig.queue.resolveByRestore(item.id);
	expect(rig.queue.conflictErrorKey()).toBe('internal');
	expect(rig.queue.conflictItemId()).toBe(item.id);
	expect(rig.queue.items()[0]?.state).toBe('conflict');

	// Closing clears the error too; reopening a parked row works.
	rig.queue.closeConflict();
	expect(rig.queue.conflictErrorKey()).toBe('');
	expect(rig.queue.conflictItemId()).toBeNull();
	rig.queue.openConflict(item.id);
	expect(rig.queue.conflictItemId()).toBe(item.id);
});

test('a save-as delete answering 404 proceeds to the re-upload on the freed slot', async () => {
	const rig = buildQueue();
	const file = photoFile();
	rig.queue.enqueueFiles([file]);
	const first = xhrsOf(rig)[0] as StubXHR;
	first.answer(200, assetBody('asset-trash', 'trashed'));
	const item = rig.queue.items()[0] as UploadItem;

	// Another device purged the trashed copy while the decision was
	// parked: the DELETE 404s, and the re-upload proceeds anyway.
	rig.fetch.mockImplementation(jsonFetch(404, { code: 'ASSET_NOT_FOUND' }));
	await rig.queue.resolveBySaveAs(item.id);
	const rows = rig.queue.items();
	expect(rows).toHaveLength(1);
	expect(rows[0]?.id).not.toBe(item.id);
	const replacement = xhrsOf(rig)[1] as StubXHR;
	expect((replacement.body as FormData).get('photo')).toBeInstanceOf(File);
	expect(((replacement.body as FormData).get('photo') as File).name).toBe('photo.jpg');
});

test('an error envelope parks the upload failed under the lowercased code; retry re-uploads', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile()]);
	const xhr = xhrsOf(rig)[0] as StubXHR;
	xhr.answer(400, { code: 'ASSET_TOO_LARGE' });
	expect(rig.queue.items()[0]?.state).toBe('failed');
	expect(rig.queue.items()[0]?.errorKey).toBe('asset_too_large');

	rig.queue.retry(rig.queue.items()[0]?.id as string);
	// The pacer starts it immediately (a slot is free).
	expect(rig.queue.items()[0]?.state).toBe('uploading');
	// Second attempt carries a fresh deviceAssetId.
	const retryXhr = xhrsOf(rig)[1] as StubXHR;
	expect((retryXhr.body as FormData).get('deviceAssetId')).toBe('id-3');
});

test('a 401 upload reaction calls the session flip and parks the row under session_expired', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile()]);
	const xhr = xhrsOf(rig)[0] as StubXHR;
	xhr.answer(401, { code: 'SESSION_EXPIRED' });
	expect(rig.queue.items()[0]?.state).toBe('failed');
	expect(rig.queue.items()[0]?.errorKey).toBe('session_expired');
	expect(rig.unauthorized).toHaveBeenCalledOnce();
});

test('the transport failure parks the row failed and frees the slot', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile(), photoFile('second.jpg')]);
	const [first] = xhrsOf(rig);
	(first as StubXHR).fails();
	expect(rig.queue.items()[0]?.state).toBe('failed');
	expect(rig.queue.items()[1]?.state).toBe('uploading');
});

test('the concurrency cap holds at three and the pacer replays strictly FIFO', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([
		photoFile('one.jpg'),
		photoFile('two.jpg'),
		photoFile('three.jpg'),
		photoFile('four.jpg'),
		photoFile('five.jpg')
	]);
	expect(UPLOAD_CONCURRENCY).toBe(3);
	// activeCount = uploading + waiting for a slot.
	expect(rig.queue.activeCount()).toBe(5);
	expect(rig.queue.items().map((item) => item.state)).toEqual([
		'uploading',
		'uploading',
		'uploading',
		'queued',
		'queued'
	]);

	const [first, , third] = xhrsOf(rig) as StubXHR[];
	// The fourth file's slot opens as soon as the first settles.
	first.answer(201, assetBody('asset-4', 'created'));
	const fourth = xhrsOf(rig)[3] as StubXHR;
	// The sequence counts the item ids 1..5 before any wire call;
	// this upload opens the fourth slot.
	expect((fourth.body as FormData).get('deviceAssetId')).toBe('id-9');
	// The third row shifts on its own settle regardless of order.
	third.answer(200, assetBody('asset-3', 'duplicate'));
	const fifth = xhrsOf(rig)[4] as StubXHR;
	expect((fifth.body as FormData).get('deviceAssetId')).toBe('id-10');
});

test('a trashed answer for a later row parks it without stealing the open dialog; each row resolves on its own', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile('one.jpg'), photoFile('two.jpg')]);
	const [first, second] = xhrsOf(rig) as StubXHR[];
	(first as StubXHR).answer(200, assetBody('asset-1', 'trashed'));
	(second as StubXHR).answer(200, assetBody('asset-2', 'trashed'));
	const [rowOne, rowTwo] = rig.queue.items();
	expect(rowOne?.state).toBe('conflict');
	expect(rowTwo?.state).toBe('conflict');
	expect(rig.queue.conflictItemId()).toBe(rowOne?.id);

	// The second row opens its own dialog when the first closes.
	rig.queue.closeConflict();
	rig.queue.openConflict(rowTwo?.id as string);
	expect(rig.queue.conflictItemId()).toBe(rowTwo?.id);
});

test('dismiss aborts an in-flight upload and clears a conflict dialog', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile()]);
	const xhr = xhrsOf(rig)[0] as StubXHR;
	// While in flight, dismissal aborts the wire.
	rig.queue.dismiss(rig.queue.items()[0]?.id as string);
	expect(xhr.aborted).toBe(true);
	expect(rig.queue.items()).toHaveLength(0);
	expect(rig.queue.conflictItemId()).toBeNull();
	expect(rig.queue.activeCount()).toBe(0);
	expect(eventBox.listeners).toHaveLength(0);

	// A settled conflict row has no wire to abort, but dismissal
	// still clears the dialog it was holding.
	rig.queue.enqueueFiles([photoFile('again.jpg')]);
	const settled = xhrsOf(rig)[1] as StubXHR;
	settled.answer(200, assetBody('asset-trash', 'trashed'));
	expect(rig.queue.conflictItemId()).not.toBeNull();
	rig.queue.dismiss(rig.queue.items()[0]?.id as string);
	expect(settled.aborted).toBe(false);
	expect(rig.queue.items()).toHaveLength(0);
	expect(rig.queue.conflictItemId()).toBeNull();
});

test('an SSE trash/delete event removes the uploaded row and releases the subscription', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile()]);
	const xhr = xhrsOf(rig)[0] as StubXHR;
	xhr.answer(201, assetBody('asset-gone', 'created'));
	expect(eventBox.listeners).toHaveLength(1);

	eventBox.emit('asset.trashed', 'asset-gone');
	expect(rig.queue.items()).toHaveLength(0);
	expect(eventBox.listeners).toHaveLength(0);
});

test('SSE events that are neither ready nor removal never touch a queue row', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile('one.jpg'), photoFile('two.jpg')]);
	const [first, second] = xhrsOf(rig) as StubXHR[];
	(first as StubXHR).answer(200, assetBody('asset-restored', 'duplicate'));
	(second as StubXHR).answer(201, assetBody('asset-uploaded', 'created'));
	const [doneRow, processingRow] = rig.queue.items();
	expect(doneRow?.state).toBe('done');
	expect(processingRow?.state).toBe('processing');

	// The event names the removal branch must ignore: the batch
	// restore's own asset.restored (the queue row already patched
	// done), the commit's asset.uploaded (a stalled or replaying
	// stream can deliver it after the XHR settle), and the
	// visibility-only asset.updated (never a pipeline readiness
	// signal -- a processing row must not flip done early).
	eventBox.emit('asset.restored', 'asset-restored');
	eventBox.emit('asset.uploaded', 'asset-uploaded');
	eventBox.emit('asset.updated', 'asset-uploaded');
	expect(rig.queue.items()[0]?.state).toBe('done');
	expect(rig.queue.items()[1]?.state).toBe('processing');

	// The deletion event still removes.
	eventBox.emit('asset.deleted', 'asset-uploaded');
	expect(rig.queue.items()).toHaveLength(1);
	expect(rig.queue.items()[0]?.asset?.id).toBe('asset-restored');
});

test('a settled response losing the race to a dismiss never resurrects the row', () => {
	const rig = buildQueue();
	rig.queue.enqueueFiles([photoFile('one.jpg'), photoFile('two.jpg')]);
	const [first, second] = xhrsOf(rig) as StubXHR[];
	(first as StubXHR).answer(201, assetBody('asset-4', 'created'));
	// The second slot opened; dismiss it, then let its response
	// settle anyway (the wire had already answered).
	const racingId = rig.queue.items()[1]?.id as string;
	rig.queue.dismiss(racingId);
	expect(second.aborted).toBe(true);
	second.answer(200, assetBody('asset-dup', 'duplicate'));
	expect(rig.queue.items()).toHaveLength(1);
	expect(rig.queue.items()[0]?.asset?.id).toBe('asset-4');
});
