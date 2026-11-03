import { beforeEach, expect, test, vi } from 'vitest';
import {
	assetOriginalUrl,
	assetPlayUrl,
	assetThumbnailUrl,
	deleteAsset,
	fetchAssetDetail,
	patchAsset,
	probeAssetPlay,
	runAssetBatch
} from './assets';
import { sessionStatus } from '$lib/auth/session.svelte';

// The single-asset client contract: the serving-URL builders, the
// detail fetch's error mapping, the PATCH body and its response
// adoption, the permanent delete's 204 / already-gone 404 outcomes,
// the batch mirror over POST /assets/batch, and the play-stream
// probe's one-byte range request. Every case drives the module
// through an injected fetch mock.

const detailBody = {
	id: 'a1',
	ownerId: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60',
	kind: 'image',
	originalFilename: 'IMG_1234.jpg',
	fileSize: 4821933,
	mimeType: 'image/jpeg',
	checksum: '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08',
	createdAt: '2026-10-29T09:12:00Z',
	takenAt: '2026-10-13T12:30:00Z',
	takenAtSource: 'exif',
	isTrashed: false,
	isHidden: false,
	description: 'Sunset over the lake.',
	isReady: true,
	width: 4032,
	height: 3024,
	durationMs: null,
	blurhash: 'UHL6aAof00WBWB00ayWB00',
	livePhotoVideoId: null,
	exif: {
		cameraMake: 'Lumemor',
		cameraModel: 'Golden Camera',
		lens: 'Golden 35mm f/2.8',
		iso: 200,
		aperture: 2.8,
		shutter: '1/250',
		focal: 35,
		lat: 48.8584,
		lon: 2.2945,
		alt: 35,
		tzOffset: '+05:30'
	}
};

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

beforeEach(async () => {
	// Boot the session store unauthenticated: the 401 interceptor
	// stays inert (no mid-session flips unless a case arranges one).
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
});

test('the URL builders assemble the contract paths and escape ids', () => {
	expect(assetThumbnailUrl('a1', 'tile')).toBe('/api/v1/assets/a1/thumbnail?size=tile');
	expect(assetThumbnailUrl('a1', 'view')).toBe('/api/v1/assets/a1/thumbnail?size=view');
	expect(assetThumbnailUrl('a1', 'original')).toBe('/api/v1/assets/a1/thumbnail?size=original');
	expect(assetOriginalUrl('a1')).toBe('/api/v1/assets/a1/original');
	expect(assetPlayUrl('a1')).toBe('/api/v1/assets/a1/play');
	// A hostile id never escapes the path segment.
	expect(assetThumbnailUrl('a/b?c', 'tile')).toBe(
		'/api/v1/assets/a%2Fb%3Fc/thumbnail?size=tile'
	);
});

test('the detail fetch returns the parsed detail on 200', async () => {
	const fetcher = vi.fn(async () => jsonResponse(200, detailBody));
	const outcome = await fetchAssetDetail('a1', fetcher);
	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.detail.description).toBe('Sunset over the lake.');
		expect(outcome.detail.exif.iso).toBe(200);
	}
	expect(fetcher).toHaveBeenCalledWith('/api/v1/assets/a1', undefined);
});

test('the detail fetch maps 404, 401, unknown codes and broken bodies', async () => {
	expect(
		await fetchAssetDetail('a1', vi.fn(async () => jsonResponse(404, { code: 'ASSET_NOT_FOUND' })))
	).toEqual({ ok: false, errorKey: 'asset_not_found' });
	expect(
		await fetchAssetDetail('a1', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })))
	).toEqual({ ok: false, errorKey: 'session_expired' });
	// An unknown code passes through lowercased; the shared renderer
	// (resolveErrorText) owns the generic-message fallback.
	expect(
		await fetchAssetDetail('a1', vi.fn(async () => jsonResponse(502, { code: 'PROXY_TEAPOT' })))
	).toEqual({ ok: false, errorKey: 'proxy_teapot' });
	expect(await fetchAssetDetail('a1', vi.fn(async () => jsonResponse(200, null)))).toEqual({
		ok: false,
		errorKey: 'internal'
	});
	// A detail whose id is not the requested one is not a detail.
	expect(
		await fetchAssetDetail('a1', vi.fn(async () => jsonResponse(200, { ...detailBody, id: 'zz' })))
	).toEqual({ ok: false, errorKey: 'internal' });
	expect(
		await fetchAssetDetail('a1', vi.fn(async () => Promise.reject(new Error('offline'))))
	).toEqual({ ok: false, errorKey: 'internal' });
});

test('the patch sends the exact body and adopts the response detail', async () => {
	const fetcher = vi.fn(async () => jsonResponse(200, { ...detailBody, description: 'New text.' }));
	const outcome = await patchAsset('a1', { description: 'New text.' }, fetcher);
	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.detail.description).toBe('New text.');
	}
	expect(fetcher).toHaveBeenCalledWith('/api/v1/assets/a1', {
		method: 'PATCH',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ description: 'New text.' })
	});
});

test('the patch maps validation, session and network failures', async () => {
	expect(
		await patchAsset('a1', { isHidden: true }, vi.fn(async () => jsonResponse(400, { code: 'VALIDATION_FAILED' })))
	).toEqual({ ok: false, errorKey: 'validation_failed' });
	expect(
		await patchAsset('a1', { isHidden: true }, vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })))
	).toEqual({ ok: false, errorKey: 'session_expired' });
	expect(
		await patchAsset('a1', { isHidden: true }, vi.fn(async () => Promise.reject(new Error('offline'))))
	).toEqual({ ok: false, errorKey: 'internal' });
	expect(
		await patchAsset('a1', { isHidden: true }, vi.fn(async () => jsonResponse(200, 'not an object')))
	).toEqual({ ok: false, errorKey: 'internal' });
});

test('the delete succeeds on 204 and treats a 404 as already gone', async () => {
	const gone = vi.fn(async () => new Response(null, { status: 204 }));
	expect(await deleteAsset('a1', gone)).toEqual({ ok: true });
	expect(gone).toHaveBeenCalledWith('/api/v1/assets/a1', { method: 'DELETE' });

	// The already-deleted row answers 404; the row is gone either
	// way (the upload queue's save-as pins the same reading).
	expect(
		await deleteAsset('a1', vi.fn(async () => jsonResponse(404, { code: 'ASSET_NOT_FOUND' })))
	).toEqual({ ok: true });
	expect(
		await deleteAsset('a1', vi.fn(async () => jsonResponse(500, { code: 'INTERNAL' })))
	).toEqual({ ok: false, errorKey: 'internal' });
	expect(
		await deleteAsset('a1', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })))
	).toEqual({ ok: false, errorKey: 'session_expired' });
});

test('the batch mirror posts the exact body and maps the outcomes', async () => {
	const fetcher = vi.fn(async () => jsonResponse(200, { affected: 1, missing: [] }));
	expect(await runAssetBatch('hide', ['a1'], fetcher)).toEqual({
		ok: true,
		affected: 1,
		missing: []
	});
	expect(fetcher).toHaveBeenCalledWith('/api/v1/assets/batch', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ action: 'hide', ids: ['a1'] })
	});

	expect(
		await runAssetBatch('trash', ['a1'], vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })))
	).toEqual({ ok: false, errorKey: 'session_expired' });
	expect(
		await runAssetBatch('trash', ['a1'], vi.fn(async () => jsonResponse(200, { affected: 'one' })))
	).toEqual({ ok: false, errorKey: 'internal' });
	// An empty id list never leaves the client.
	const unused = vi.fn();
	expect(await runAssetBatch('trash', [], unused)).toEqual({ ok: false, errorKey: 'internal' });
	expect(unused).not.toHaveBeenCalled();
});

test('the play probe sends a one-byte range and maps readiness', async () => {
	let seen: { url: string; init: RequestInit | undefined } | undefined;
	const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		seen = { url: String(input), init };
		// The documented satisfiable single range answer.
		return new Response('x', { status: 206, headers: { 'Content-Range': 'bytes 0-0/12345' } });
	});
	expect(await probeAssetPlay('a1', fetcher)).toEqual({ ok: true });
	expect(seen?.url).toBe('/api/v1/assets/a1/play');
	expect(seen?.init?.headers).toEqual({ Range: 'bytes=0-0' });

	// A transcode still in flight: the client keeps the poster and
	// waits for the SSE convergence.
	expect(
		await probeAssetPlay('a1', vi.fn(async () => jsonResponse(404, { code: 'ASSET_NOT_READY' })))
	).toEqual({ ok: false, errorKey: 'asset_not_ready' });
	// A plain 200 (the range ignored) still proves the stream.
	expect(
		await probeAssetPlay('a1', vi.fn(async () => new Response('video', { status: 200 })))
	).toEqual({ ok: true });
	expect(
		await probeAssetPlay('a1', vi.fn(async () => Promise.reject(new Error('offline'))))
	).toEqual({ ok: false, errorKey: 'internal' });
});
