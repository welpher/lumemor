import { beforeEach, expect, test, vi } from 'vitest';
import {
	completeAlbumPath,
	createAlbum,
	deleteAlbum,
	fetchAlbumDetail,
	listAlbums,
	listUsers,
	patchAlbum,
	rescanAlbum,
	setAlbumTimelinePreference,
	updateAlbumAudience,
	type AlbumSummary
} from './albums';
import { sessionStatus } from '$lib/auth/session.svelte';

// The albums surface's data client (E4's albums-as-sources reshape):
// the wall list, the detail read, the directory creation (title +
// source path + audience, the honest skipped report), the property
// patch, the virtual/directory delete, the path completion of the
// creation dialog's combobox, the audience's full-set PUT, the
// rescan trigger, the member list and the timeline preference --
// each behind the repo-wide error-code convention, and the cover
// validation's message key surfaced as an error key (the people
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

const directoryCard: AlbumSummary = {
	id: '0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6078',
	title: 'Family archive',
	description: '',
	type: 'directory',
	locked: false,
	isShared: false,
	coverAssetIds: [],
	itemCount: 0,
	updatedAt: '2026-11-05T10:15:00Z',
	callerRole: 'owner'
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
		stubFetch({ '/api/v1/albums': () => jsonResponse(200, { albums: [ownCard, directoryCard] }) })
	);

	const outcome = await listAlbums();

	expect(outcome).toEqual({ ok: true, albums: [ownCard, directoryCard] });
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

test('fetchAlbumDetail reads the directory album source row', async () => {
	const directoryDetail = {
		...detailBody,
		type: 'directory',
		rootPath: '/Volumes/Photos/family'
	};
	vi.stubGlobal(
		'fetch',
		stubFetch({ [`/api/v1/albums/${ownCard.id}?limit=1`]: () => jsonResponse(200, directoryDetail) })
	);

	const outcome = await fetchAlbumDetail(ownCard.id);

	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.detail.rootPath).toBe('/Volumes/Photos/family');
	}
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

test('createAlbum POSTs the title, source path and audience, and answers the report', async () => {
	const fetchMock = stubFetch({
		'/api/v1/albums': (init?: RequestInit) => {
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({
				title: 'Family archive',
				rootPath: '/Volumes/Photos/family',
				audience: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70']
			});
			return jsonResponse(201, {
				...directoryCard,
				isShared: true,
				skipped: [{ userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f99', reason: 'invalid_account' }]
			});
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await createAlbum({
		title: 'Family archive',
		rootPath: '/Volumes/Photos/family',
		audience: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70']
	});

	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.album.type).toBe('directory');
		expect(outcome.skipped).toEqual([
			{ userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f99', reason: 'invalid_account' }
		]);
	}
});

test('createAlbum surfaces the path matrix codes unmangled', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums': () =>
				jsonResponse(400, { code: 'ALBUM_PATH_OUTSIDE_ROOTS', message: 'outside the roots' })
		})
	);
	const outcome = await createAlbum({ title: 'X', rootPath: '/elsewhere', audience: [] });
	expect(outcome).toEqual({ ok: false, errorKey: 'album_path_outside_roots' });

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
	expect(await createAlbum({ title: '   ', rootPath: '/Volumes/Photos', audience: [] })).toEqual({
		ok: false,
		errorKey: 'validation_failed'
	});
});

test('completeAlbumPath reads the completion rows for a typed prefix', async () => {
	const fetchMock = stubFetch({
		'/api/v1/albums/path-completions?prefix=%2FVolumes%2FPhotos%2F2': () =>
			jsonResponse(200, {
				paths: ['/Volumes/Photos/2026-spring', '/Volumes/Photos/2026-summer']
			})
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await completeAlbumPath('/Volumes/Photos/2');

	expect(outcome).toEqual({
		ok: true,
		paths: ['/Volumes/Photos/2026-spring', '/Volumes/Photos/2026-summer']
	});
});

test('completeAlbumPath sends no query for the empty prefix (the roots listing)', async () => {
	const fetchMock = stubFetch({
		'/api/v1/albums/path-completions': () => jsonResponse(200, { paths: ['/Volumes/Photos'] })
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await completeAlbumPath('');

	expect(outcome).toEqual({ ok: true, paths: ['/Volumes/Photos'] });
	expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/albums/path-completions');
});

test('completeAlbumPath accepts the honest empty answer and maps failures', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/albums/path-completions?prefix=%2Fnone': () => jsonResponse(200, { paths: [] }) })
	);
	expect(await completeAlbumPath('/none')).toEqual({ ok: true, paths: [] });

	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums/path-completions?prefix=%2Fnone': () =>
				jsonResponse(500, { code: 'INTERNAL' })
		})
	);
	expect(await completeAlbumPath('/none')).toEqual({ ok: false, errorKey: 'internal' });

	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/albums/path-completions?prefix=%2Fnone': () => jsonResponse(200, { paths: 'no' }) })
	);
	expect(await completeAlbumPath('/none')).toEqual({ ok: false, errorKey: 'internal' });
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

test('updateAlbumAudience PUTs the full set and answers the honest report', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}/audience`]: (init?: RequestInit) => {
			expect(init?.method).toBe('PUT');
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({
				audience: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70', '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f71']
			});
			return jsonResponse(200, {
				albumId: ownCard.id,
				audience: [
					{
						userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70',
						email: 'grace@example.com',
						grantedAt: '2026-11-05T09:30:00Z'
					},
					{
						userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f71',
						email: 'hopper@example.com',
						grantedAt: '2026-11-05T09:30:00Z'
					}
				],
				granted: [
					'0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70',
					'0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f71'
				],
				revoked: [],
				skipped: [{ userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f99', reason: 'owner' }]
			});
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await updateAlbumAudience(ownCard.id, [
		'0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70',
		'0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f71'
	]);

	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.report.audience).toHaveLength(2);
		expect(outcome.report.skipped).toEqual([
			{ userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f99', reason: 'owner' }
		]);
	}
});

test('updateAlbumAudience maps the empty-set 400 and the sharee 403', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}/audience`]: () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: 'the audience field must ride the body',
					details: { fields: ['audience'] }
				})
		})
	);
	expect(await updateAlbumAudience(ownCard.id, [])).toEqual({
		ok: false,
		errorKey: 'validation_failed'
	});

	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}/audience`]: () =>
				jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'view-only share' })
		})
	);
	expect(await updateAlbumAudience(ownCard.id, ['u3'])).toEqual({
		ok: false,
		errorKey: 'auth_forbidden'
	});
});

test('rescanAlbum POSTs the trigger and answers the run identifier', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${directoryCard.id}/rescan`]: (init?: RequestInit) => {
			expect(init?.method).toBe('POST');
			expect(init?.body).toBeUndefined();
			return jsonResponse(202, { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10' });
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await rescanAlbum(directoryCard.id);

	expect(outcome).toEqual({
		ok: true,
		run: { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10' }
	});
});

test('rescanAlbum accepts the in-flight join and maps the locked 409', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${directoryCard.id}/rescan`]: () =>
				jsonResponse(200, { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f11' })
		})
	);
	expect(await rescanAlbum(directoryCard.id)).toEqual({
		ok: true,
		run: { jobRunId: '0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f11' }
	});

	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${directoryCard.id}/rescan`]: () =>
				jsonResponse(409, { code: 'ALBUM_LOCKED', message: 'the album is locked' })
		})
	);
	expect(await rescanAlbum(directoryCard.id)).toEqual({ ok: false, errorKey: 'album_locked' });
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
