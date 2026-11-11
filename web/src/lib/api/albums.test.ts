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

// The albums surface's data client (the albums-as-sources reshape,
// completed by album autonomy, renamed with the memory's birth): the
// wall list, the detail read (the deletion faces included), the
// directory creation (title + source path + audience, the honest
// skipped report and the write probe's sourceWritable fact -- both
// admin-guarded writes), the property patch (title, cover,
// deletability), the memory/directory delete, the path completion of
// the creation dialog's combobox, the audience's full-set PUT, the
// rescan trigger, the member list and the timeline preference --
// each behind the repo-wide error-code convention, and the write
// faces' message keys surfaced as error keys (the people client's
// requires-label pattern). Every case drives one call through an
// injected fetch mock.

const ownCard: AlbumSummary = {
	id: '0192f5b3-6c7d-8e9f-a0b1-2c3d4e5f6071',
	title: 'Road trip 2026',
	type: 'memory',
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
	type: 'directory',
	isShared: false,
	coverAssetIds: [],
	itemCount: 0,
	updatedAt: '2026-11-05T10:15:00Z',
	callerRole: 'owner'
};

const detailBody = {
	id: ownCard.id,
	title: ownCard.title,
	type: 'memory',
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
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => jsonResponse(401, { code: 'AUTH_INVALID_CREDENTIALS' }))
	);
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
	vi.stubGlobal('fetch', stubFetch({ '/api/v1/albums': () => jsonResponse(500, { code: 'INTERNAL' }) }));
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

test('fetchAlbumDetail reads the directory album source row and the deletion faces', async () => {
	const directoryDetail = {
		...detailBody,
		type: 'directory',
		rootPath: '/Volumes/Photos/family',
		sourceWritable: true,
		allowAppDeletion: false,
		deletable: false
	};
	vi.stubGlobal(
		'fetch',
		stubFetch({ [`/api/v1/albums/${ownCard.id}?limit=1`]: () => jsonResponse(200, directoryDetail) })
	);

	const outcome = await fetchAlbumDetail(ownCard.id);

	expect(outcome.ok).toBe(true);
	if (outcome.ok) {
		expect(outcome.detail.rootPath).toBe('/Volumes/Photos/family');
		// The probe's fact, the owner's switch and the derived
		// affordance ride the directory detail as-is.
		expect(outcome.detail.sourceWritable).toBe(true);
		expect(outcome.detail.allowAppDeletion).toBe(false);
		expect(outcome.detail.deletable).toBe(false);
	}

	// A drifted face (the fact is not the boolean it names) maps to
	// the generic failure, never a broken surface.
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}?limit=1`]: () =>
				jsonResponse(200, { ...directoryDetail, sourceWritable: 'maybe' })
		})
	);
	expect(await fetchAlbumDetail(ownCard.id)).toEqual({ ok: false, errorKey: 'internal' });
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

test('createAlbum POSTs the title, source path and audience, and answers the report with the probe fact', async () => {
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
				sourceWritable: false,
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
		// The write probe's fact rides the answer: false is honest
		// creation feedback (a read-only source), never a failure.
		expect(outcome.sourceWritable).toBe(false);
	}
});

test('createAlbum POSTs the autonomy switches with the drafts it carries', async () => {
	let first = true;
	const fetchMock = stubFetch({
		'/api/v1/albums': (init?: RequestInit) => {
			const body = JSON.parse(String(init?.body));
			if (first) {
				first = false;
				// The exclusion rides the draft verbatim; the
				// deletability switch rides only where the dialog
				// rendered it.
				expect(body).toEqual({
					title: 'Family archive',
					rootPath: '/Volumes/Photos/family',
					audience: [],
					inTimeline: false,
					allowAppDeletion: false
				});
			} else {
				expect(body).toEqual({
					title: 'Family archive',
					rootPath: '/Volumes/Photos/family',
					audience: []
				});
			}
			return jsonResponse(201, {
				...directoryCard,
				isShared: true,
				sourceWritable: true,
				skipped: []
			});
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	// Both switches carried: the exclusion off, the deletability off
	// (a writable pick's unchecked row).
	await createAlbum({
		title: 'Family archive',
		rootPath: '/Volumes/Photos/family',
		audience: [],
		allowAppDeletion: false,
		inTimeline: false
	});

	// Neither switch carried: the server's defaults govern.
	await createAlbum({ title: 'Family archive', rootPath: '/Volumes/Photos/family', audience: [] });
	expect(fetchMock.mock.calls).toHaveLength(2);
});

test('createAlbum maps the member 403 and refuses a body without the probe fact', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums': () =>
				jsonResponse(403, { code: 'AUTH_FORBIDDEN', message: 'administrator role required' })
		})
	);
	expect(await createAlbum({ title: 'X', rootPath: '/Volumes/Photos', audience: [] })).toEqual({
		ok: false,
		errorKey: 'auth_forbidden'
	});

	// The fact is required: a drifted answer maps to the generic
	// failure like any unparseable body.
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums': () => jsonResponse(201, { ...directoryCard, skipped: [] })
		})
	);
	expect(await createAlbum({ title: 'X', rootPath: '/Volumes/Photos', audience: [] })).toEqual({
		ok: false,
		errorKey: 'internal'
	});
});

test('createAlbum surfaces the path matrix codes unmangled', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums': () =>
				jsonResponse(400, { code: 'ALBUM_PATH_NOT_FOUND', message: 'the source path does not exist' })
		})
	);
	const outcome = await createAlbum({ title: 'X', rootPath: '/elsewhere', audience: [] });
	expect(outcome).toEqual({ ok: false, errorKey: 'album_path_not_found' });

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

test('completeAlbumPath reads the completion rows with their write facts', async () => {
	const fetchMock = stubFetch({
		'/api/v1/albums/path-completions?prefix=%2FVolumes%2FPhotos%2F2': () =>
			jsonResponse(200, {
				completions: [
					{ path: '/Volumes/Photos/2026-spring', writable: true },
					{ path: '/Volumes/Photos/2026-summer', writable: false }
				]
			})
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await completeAlbumPath('/Volumes/Photos/2');

	expect(outcome).toEqual({
		ok: true,
		completions: [
			{ path: '/Volumes/Photos/2026-spring', writable: true },
			{ path: '/Volumes/Photos/2026-summer', writable: false }
		]
	});
});

test('completeAlbumPath sends no query for the empty prefix (the root listing)', async () => {
	const fetchMock = stubFetch({
		'/api/v1/albums/path-completions': () => jsonResponse(200, { completions: [{ path: '/', writable: true }] })
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await completeAlbumPath('');

	expect(outcome).toEqual({ ok: true, completions: [{ path: '/', writable: true }] });
	expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/v1/albums/path-completions');
});

test('completeAlbumPath accepts the honest empty answer and maps failures', async () => {
	vi.stubGlobal(
		'fetch',
		stubFetch({ '/api/v1/albums/path-completions?prefix=%2Fnone': () => jsonResponse(200, { completions: [] }) })
	);
	expect(await completeAlbumPath('/none')).toEqual({ ok: true, completions: [] });

	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums/path-completions?prefix=%2Fnone': () => jsonResponse(500, { code: 'INTERNAL' })
		})
	);
	expect(await completeAlbumPath('/none')).toEqual({ ok: false, errorKey: 'internal' });

	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums/path-completions?prefix=%2Fnone': () => jsonResponse(200, { completions: ['no'] })
		})
	);
	expect(await completeAlbumPath('/none')).toEqual({ ok: false, errorKey: 'internal' });

	// A drifted row (the write fact missing) maps to the generic
	// failure like any unparseable body -- the dialog never renders
	// from a fact it does not have.
	vi.stubGlobal(
		'fetch',
		stubFetch({
			'/api/v1/albums/path-completions?prefix=%2Fnone': () => jsonResponse(200, { completions: [{ path: '/x' }] })
		})
	);
	expect(await completeAlbumPath('/none')).toEqual({ ok: false, errorKey: 'internal' });
});

test('patchAlbum PATCHes the carried subset and answers the card', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${ownCard.id}`]: (init?: RequestInit) => {
			expect(init?.method).toBe('PATCH');
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({ title: 'Renamed archive' });
			return jsonResponse(200, ownCard);
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await patchAlbum(ownCard.id, { title: 'Renamed archive' });
	expect(outcome).toEqual({ ok: true, album: ownCard });
});

test('patchAlbum PATCHes the deletability switch like every property field', async () => {
	const fetchMock = stubFetch({
		[`/api/v1/albums/${directoryCard.id}`]: (init?: RequestInit) => {
			expect(init?.method).toBe('PATCH');
			const body = JSON.parse(String(init?.body));
			expect(body).toEqual({ allowAppDeletion: false });
			return jsonResponse(200, directoryCard);
		}
	});
	vi.stubGlobal('fetch', fetchMock);

	const outcome = await patchAlbum(directoryCard.id, { allowAppDeletion: false });
	expect(outcome).toEqual({ ok: true, album: directoryCard });
});

test("patchAlbum surfaces the write faces' message keys", async () => {
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

	// The deletability switch's own refusal: a memory album answers
	// the message key, not a generic validation text.
	vi.stubGlobal(
		'fetch',
		stubFetch({
			[`/api/v1/albums/${ownCard.id}`]: () =>
				jsonResponse(400, {
					code: 'VALIDATION_FAILED',
					message: "the deletability switch is a directory album's own setting",
					details: { key: 'album_deletable_not_directory' }
				})
		})
	);
	expect(await patchAlbum(ownCard.id, { allowAppDeletion: false })).toEqual({
		ok: false,
		errorKey: 'album_deletable_not_directory'
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
				granted: ['0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f70', '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f71'],
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
		expect(outcome.report.skipped).toEqual([{ userId: '0192f4a1-5b6c-7d8e-9f0a-2b3c4d5e6f99', reason: 'owner' }]);
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

test('rescanAlbum accepts the in-flight join', async () => {
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
