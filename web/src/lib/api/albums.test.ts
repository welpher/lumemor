import { beforeEach, expect, test, vi } from 'vitest';
import {
	addAlbumItems,
	createAlbum,
	deleteAlbum,
	fetchAlbumDetail,
	grantAlbumShare,
	listAlbums,
	listUsers,
	patchAlbum,
	revokeAlbumShare,
	setAlbumTimelinePreference,
	type AlbumSummary
} from './albums';
import { sessionStatus } from '$lib/auth/session.svelte';

// The albums surface's data client: the wall list, the detail read,
// the creation/patch/delete writes, the curation add, the share
// grant/revoke pair, the member list and the timeline preference --
// each behind the repo-wide error-code convention, and the share
// target rules' message keys surfaced as error keys (the people
// client's requires-label pattern). Every case drives one call
// through an injected fetch mock.

const ownCard: AlbumSummary = {
	id: '0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6071',
	title: 'Road trip 2026',
	description: 'The coastal week in the van.',
	type: 'virtual',
	locked: false,
	isShared: true,
	coverAssetIds: [
		'0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6072',
		'0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6073',
		'0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6074',
		'0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6075'
	],
	itemCount: 128,
	updatedAt: '2026-11-04T18:20:00Z',
	callerRole: 'owner'
};

const sharedCard: AlbumSummary = {
	id: '0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6091',
	title: "Grace's highlights",
	description: 'Shared with the family.',
	type: 'virtual',
	locked: false,
	isShared: true,
	coverAssetIds: ['0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6092'],
	itemCount: 17,
	updatedAt: '2026-11-02T14:05:00Z',
	callerRole: 'viewer',
	owner: { id: '0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60', email: 'grace@example.com' },
	inTimeline: true
};

const detailBody = {
	id: ownCard.id,
	title: ownCard.title,
	description: ownCard.description,
	type: 'virtual',
	locked: false,
	coverAssetId: null,
	callerRole: 'owner',
	itemCount: 2,
	items: [
		{
			assetId: '0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6072',
			addedAt: '2026-11-04T18:19:00Z',
			asset: {
				takenAt: '2026-10-13T12:30:00Z',
				originalFilename: 'IMG_1234.jpg',
				kind: 'image',
				thumbnail: '/api/v1/assets/0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6072/thumbnail?size=tile'
			}
		}
	],
	nextCursor: 'GN4WpRPw0AABkvShW2x9jp8KKzxNXm9w',
	shares: [
		{
			userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70',
			email: 'grace@example.com',
			grantedAt: '2026-11-05T09:30:00Z'
		}
	]
};

function jsonResponse(status: number, body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

/** Route table for the fetch stub; every test installs its own. */
type FetchRoute = Record<string, (init?: RequestInit) => Response>;

function stubFetch(route: FetchRoute): ReturnType<typeof vi.fn> {
	return vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		const answer = route[url];
		if (answer === undefined) {
			throw new Error(`unexpected fetch ${url}`);
		}
		return answer(init);
	});
}

beforeEach(async () => {
	// Boot the session store unauthenticated: the 401 interceptor
	// stays inert for these tests (no mid-session flips unless a
	// case arranges one).
	vi.stubGlobal('fetch', vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' })));
	await import('$lib/auth/session.svelte').then((module) => module.probeSession());
	expect(sessionStatus()).toBe('unauthenticated');
});

test('listAlbums reads the wall: the own group then the shared group', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/albums': () => jsonResponse(200, { albums: [ownCard, sharedCard] }) })
	);

	const outcome = await listAlbums();

	expect(outcome).toEqual({ ok: true, albums: [ownCard, sharedCard] });
});

test('listAlbums reads the pinned sync card of the default album', async () => {
	// The caller's own sync album leads the wall: the internal
	// 'Sync' title rides the card, the type keys the display name.
	const syncCard: AlbumSummary = {
		id: '0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6061',
		title: 'Sync',
		description: '',
		type: 'sync',
		locked: false,
		isShared: false,
		coverAssetIds: [
			'0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6076',
			'0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6077'
		],
		itemCount: 1204,
		updatedAt: '2026-11-05T07:30:00Z',
		callerRole: 'owner'
	};
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/albums': () => jsonResponse(200, { albums: [syncCard, ownCard] }) })
	);

	const outcome = await listAlbums();

	expect(outcome).toEqual({ ok: true, albums: [syncCard, ownCard] });
});

test('listAlbums maps a dead session to the session key', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/albums': () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }) })
	);
	const outcome = await listAlbums();
	expect(outcome).toEqual({ ok: false, errorKey: 'session_expired' });
});

test('listAlbums falls back to the generic key on error codes and broken bodies', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/albums': () => jsonResponse(500, { code: 'INTERNAL' }) })
	);
	expect(await listAlbums()).toEqual({ ok: false, errorKey: 'internal' });

	vi.stubGlobal('fetch', stubFetch({ '/api/v1/albums': () => jsonResponse(200, { albums: 'no' }) }));
	expect(await listAlbums()).toEqual({ ok: false, errorKey: 'internal' });

	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums': () =>
				new Response('not json', { status: 200, headers: { 'Content-Type': 'application/json' } })
		})
	);
	expect(await listAlbums()).toEqual({ ok: false, errorKey: 'internal' });
});

test('fetchAlbumDetail reads the metadata page with the lean limit', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}?limit=1`]: () => jsonResponse(200, detailBody)
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await fetchAlbumDetail(ownCard.id);

	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.detail.id).toBe(ownCard.id);
		expect(outcome.detail.callerRole).toBe('owner');
		expect(outcome.detail.shares).toHaveLength(1);
	}
});

test('fetchAlbumDetail reads the caller own sync album read-only', async () => {
	// The default album's detail: the type 'sync' with the owner
	// role, no excludable field, and the owner's always-empty shares
	// list (no share row can name a sync album).
	const syncDetail = {
		id: ownCard.id,
		title: 'Sync',
		description: '',
		type: 'sync',
		locked: false,
		coverAssetId: null,
		callerRole: 'owner',
		itemCount: 1204,
		items: [],
		shares: []
	};
	vi.stubGlobal(
		'fetch',
		stubFetch({ [`/api/v1/albums/${ownCard.id}?limit=1`]: () => jsonResponse(200, syncDetail) })
	);

	const outcome = await fetchAlbumDetail(ownCard.id);

	expect(outcome).toEqual({ ok: true, detail: syncDetail });
});

test('fetchAlbumDetail maps the single-shape 404', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}?limit=1`]: () =>
				jsonResponse(404, { code: 'ALBUM_NOT_FOUND', message: 'no such album for this caller' })
		})
	);
	const outcome = await fetchAlbumDetail(ownCard.id);
	expect(outcome).toEqual({ ok: false, errorKey: 'album_not_found' });
});

test('createAlbum POSTs the title and description, and answers the card', async () => {
	const fetchMock = stubFetch({
		'/api/v1/albums': (init?: RequestInit) => {
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({ title: 'Road trip 2026', description: 'The coastal week.' });
			return jsonResponse(201, { ...ownCard, isShared: false, coverAssetIds: [], itemCount: 0 });
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await createAlbum('Road trip 2026', 'The coastal week.');

	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.album.id).toBe(ownCard.id);
	}
});

test('createAlbum maps the title-bound 400', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums': () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'title must span 1-64 characters',
					details: { fields: ['title'] }
				})
		})
	);
	const outcome = await createAlbum('   ');
	expect(outcome).toEqual({ ok: false, errorKey: 'validation_failed' });
});

test('patchAlbum PATCHes the carried subset and answers the card', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}`]: (init?: RequestInit) => {
			expect(init?.method).toBe('PATCH');
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({ locked: false });
			return jsonResponse(200, ownCard);
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await patchAlbum(ownCard.id, { locked: false });
	expect(outcome).toEqual({ ok: true, album: ownCard });
});

test('patchAlbum surfaces the locked 409 and the cover message key', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}`]: () =>
				jsonResponse(409, { code: 'ALBUM_LOCKED', message: 'the album is locked' })
		})
	);
	expect(await patchAlbum(ownCard.id, { title: 'X' })).toEqual({
		ok: false,
		errorKey: 'album_locked'
	});

	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}`]: () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'the cover must be a live member of this album',
					details: { key: 'album_cover_not_member' }
				})
		})
	);
	expect(await patchAlbum(ownCard.id, { coverAssetId: 'x' })).toEqual({
		ok: false,
		errorKey: 'album_cover_not_member'
	});
});

test('deleteAlbum issues the DELETE and accepts the 204', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}`]: (init?: RequestInit) => {
			expect(init?.method).toBe('DELETE');
			return new Response(null, { status: 204 });
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	expect(await deleteAlbum(ownCard.id)).toEqual({ ok: true });
});

test('addAlbumItems POSTs the id list and answers the honest report', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}/items`]: (init?: RequestInit) => {
			expect(init?.method).toBe('POST');
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({ assetIds: ['a1', 'a2', 'a3'] });
			return jsonResponse(200, { added: 2, skipped: 1 });
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await addAlbumItems(ownCard.id, ['a1', 'a2', 'a3']);
	expect(outcome).toEqual({ ok: true, added: 2, skipped: 1 });
});

test('addAlbumItems surfaces the locked 409', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}/items`]: () =>
				jsonResponse(409, { code: 'ALBUM_LOCKED', message: 'the album is locked' })
		})
	);
	expect(await addAlbumItems(ownCard.id, ['a1'])).toEqual({ ok: false, errorKey: 'album_locked' });
});

test('grantAlbumShare POSTs the grantee and answers the row', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}/shares`]: (init?: RequestInit) => {
			expect(init?.method).toBe('POST');
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({ userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70' });
			return jsonResponse(200, {
				albumId: ownCard.id,
				userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70',
				grantedAt: '2026-11-05T09:30:00Z'
			});
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await grantAlbumShare(ownCard.id, '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70');
	expect(outcome.ok).toBe(true);
});

test('grantAlbumShare surfaces the target-rule message keys', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}/shares`]: () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'an album owner cannot be its grantee',
					details: { key: 'album_share_to_owner' }
				})
		})
	);
	expect(await grantAlbumShare(ownCard.id, 'me-myself')).toEqual({
		ok: false,
		errorKey: 'album_share_to_owner'
	});

	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}/shares`]: () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'the share target is not a usable account',
					details: { key: 'album_share_user_invalid' }
				})
		})
	);
	expect(await grantAlbumShare(ownCard.id, 'unknown-user')).toEqual({
		ok: false,
		errorKey: 'album_share_user_invalid'
	});
});

test('revokeAlbumShare issues the DELETE and accepts the 204', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}/shares/0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70`]: (
			init?: RequestInit
		) => {
			expect(init?.method).toBe('DELETE');
			return new Response(null, { status: 204 });
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	expect(await revokeAlbumShare(ownCard.id, '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70')).toEqual({
		ok: true
	});
});

test('listUsers reads the minimal member list', async () => {
	const fetchMock = stubFetch({
		'/api/v1/users': () =>
			jsonResponse(200, {
				users: [
					{ id: 'u1', email: 'ada@example.com', role: 'admin', isDisabled: false },
					{ id: 'u2', email: 'grace@example.com', role: 'user', isDisabled: true }
				]
			})
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await listUsers();
	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.users).toHaveLength(2);
		expect(outcome.users[1]?.isDisabled).toBe(true);
	}
});

test('setAlbumTimelinePreference PUTs the switch and answers the stored state', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}/timeline-preference`]: (init?: RequestInit) => {
			expect(init?.method).toBe('PUT');
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({ inTimeline: false });
			return jsonResponse(200, { albumId: ownCard.id, inTimeline: false });
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await setAlbumTimelinePreference(ownCard.id, false);
	expect(outcome).toEqual({
		ok: true,
		preference: { albumId: ownCard.id, inTimeline: false }
	});
});

test('setAlbumTimelinePreference surfaces the not-excludable message key', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}/timeline-preference`]: () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'this album cannot be excluded',
					details: { key: 'album_pref_not_excludable' }
				})
		})
	);
	expect(await setAlbumTimelinePreference(ownCard.id, false)).toEqual({
		ok: false,
		errorKey: 'album_pref_not_excludable'
	});
});
