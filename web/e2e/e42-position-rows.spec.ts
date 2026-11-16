// The position-row story of the web app, walked end to end against
// the real compose stack (E42): a directory album mirrors its FOLDER
// -- every file the scan walks gets its own source-position row --
// while the library faces (the timeline, My Photos, favorites,
// search, places, memories, people and their counts) dedup the
// DISPLAY, one tile per content. The story's two halves are the two
// orders the model distinguishes:
//
//   * UPLOAD FIRST, THEN SCAN (the dual home): the photo is an
//     uploaded copy in My Photos AND a source copy in a read-only
//     folder's album. The timeline shows ONE tile (the upload), the
//     album shows its OWN copy, and the delete affordance answers
//     per COPY: the read-only source's detail offers no delete (the
//     application cannot delete what the folder owns), while the
//     uploaded copy's detail does. Deleting the upload leaves the
//     source copy standing -- the timeline's tile survives, loses
//     the delete button, and the toast said how many copies remain.
//
//   * SCAN FIRST, THEN UPLOAD (the upload arm's full-domain check):
//     the upload of content the owner's folders already hold answers
//     the duplicate state -- nothing lands in My Photos, no second
//     row exists. Only when the folder copy is GONE from disk (the
//     scan's orphan mark, a row no restore can reach) does the same
//     upload land, replacing the dead row with a fresh uploaded copy.
//
//   * THE RENAME dissolves under the position model: renaming the
//     file in its folder is one row's death and another's birth, so
//     the re-scan keeps it visible -- the old follow-up (a renamed
//     file turning invisible under content dedup) cannot happen.
//
// The stack's lifecycle is owned by the caller (one fresh stack per
// spec -- the first-run bootstrap below expects its 201); the spec
// itself only talks to http://localhost:8244.
//
// THE STACK ARRANGEMENT: the repository's compose file binds two
// fixture trees into the lumemor container. The read-only bind
// (`/mnt/e2e-source`) is where the dual-home folder is seeded -- the
// container can never write there, which IS the read-only source the
// no-delete assertion needs -- and the writable sibling
// (`/mnt/e2e-source-rw`) carries the reverse-order scene, whose file
// must leave the folder on the host before the upload can land. Both
// seeded directories are created and removed by this spec (nothing
// new is committed).
//
// THE SCENES, in order:
//
//  1. The bootstrap: the first-run administrator, the model group's
//     download converged through the admin API (the scans' pipelines
//     run the same fan-out an upload's does), and the dual-home
//     photo's bytes seeded into the read-only fixture tree.
//  2. The upload lands: My Photos holds one tile, the timeline one,
//     and the detail's twin count is zero.
//  3. The read-only album's creation scan ingests the SAME content
//     as its own position row: the album wall's one tile is not the
//     upload's id, the album's detail answers the read-only source,
//     and the timeline still shows exactly one tile (the upload).
//  4. The delete affordance answers per copy: the album's detail has
//     no Trash, the upload's detail has one. The trash lands, the
//     toast carries the twin note, the timeline's tile survives (the
//     position row represents the content now) and loses the delete
//     button too (its only home is the read-only source).
//  5. The reverse order: a writable folder's file is scanned first,
//     the same bytes then upload as a DUPLICATE (no new row, nothing
//     in My Photos), the file leaves the folder (the orphan mark),
//     and the re-upload lands as a fresh uploaded copy.
//  6. The rename: a file renamed inside its folder is re-scanned into
//     a fresh position row -- the album still shows it.

import { copyFile, mkdir, readFile, rename, rm } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

const ADMIN_EMAIL = 'admin@e42-e2e.invalid';
const ADMIN_NAME = 'Mei Chen';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The dual-home photo: the repository's large golden JPEG, uploaded
// AND seeded into the read-only source tree -- byte for byte, which
// is what makes the two rows one content.
const DUAL_HOME_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/golden-large.jpg');
const DUAL_HOME_NAME = 'IMG_E42_0001.jpg';
const DUAL_HOME_DIR = '/mnt/e2e-source/e42-dual-home';
const DUAL_HOME_HOST_DIR = path.join(specDir, 'fixtures/m3-source/e42-dual-home');
const DUAL_HOME_ALBUM = 'Dual home 2026';

// The reverse-order photo: distinct bytes (a second golden frame),
// seeded into the WRITABLE fixture bind so the host can take the
// file away between the scenes.
const REVERSE_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/portrait-orientation6.jpg');
const REVERSE_NAME = 'IMG_E42_0002.jpg';
const REVERSE_DIR = '/mnt/e2e-source-rw/e42-reverse';
const REVERSE_HOST_DIR = path.join(specDir, 'fixtures/m3-source-rw/e42-reverse');
const REVERSE_ALBUM = 'Reverse order 2026';

const DOWNLOAD_TIMEOUT_MS = 900_000;
// The UI scenes' bound: a detail page's own reads converge once the
// session is replayed and the app boots (well inside this).
const SCENE_WAIT_MS = 60_000;
const SCAN_TIMEOUT_MS = 120_000;
const SYNC_TIMEOUT_MS = 120_000;

// The storage-state file the first scene writes and later scenes
// replay (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/e42-state.json`;

// The ids the scenes carry between them (captured through the API).
let uploadId = '';
let positionId = '';
let albumId = '';
let reverseUploadId = '';
let reverseAlbumId = '';
let reversePositionId = '';

/** One timeline page's item ids, in order. */
async function timelineIds(request: APIRequestContext, query = ''): Promise<string[]> {
	const response = await request.get(`/api/v1/assets/timeline${query}`);
	expect(response.status()).toBe(200);
	const body = (await response.json()) as { items?: Array<{ id?: string }> };
	return (body.items ?? []).map((item) => item.id ?? '');
}

/** One album narrow's item ids, in order. */
async function albumIds(request: APIRequestContext, id: string): Promise<string[]> {
	return timelineIds(request, `?albumId=${encodeURIComponent(id)}`);
}

/** The caller's sync album ("My photos") id, read off the wall. */
async function syncAlbumId(request: APIRequestContext): Promise<string> {
	const response = await request.get('/api/v1/albums');
	expect(response.status()).toBe(200);
	const body = (await response.json()) as { albums?: Array<{ id?: string; type?: string }> };
	const card = (body.albums ?? []).find((album) => album.type === 'sync');
	expect(card?.id).toBeTruthy();
	return card?.id ?? '';
}

/** One asset's detail body, decoded. */
type Detail = {
	id?: string;
	contentCopies?: number;
	albums?: Array<{ id?: string; type?: string; sourceWritable?: boolean }>;
};

async function detailOf(request: APIRequestContext, id: string): Promise<Detail> {
	const response = await request.get(`/api/v1/assets/${encodeURIComponent(id)}?locale=en`);
	expect(response.status()).toBe(200);
	return (await response.json()) as Detail;
}

/** One upload through the asset API (the seeding pattern every
 * story uses), answering the response body. */
async function uploadFile(
	request: APIRequestContext,
	file: string,
	name: string,
	deviceAssetId: string
): Promise<{ status: number; body: { id?: string; state?: string } }> {
	const response = await request.post('/api/v1/assets', {
		multipart: {
			deviceAssetId,
			fileName: name,
			deviceCreatedAt: '2026-05-01T12:00:00Z',
			deviceTimeZone: 'UTC',
			photo: { name, mimeType: 'image/jpeg', buffer: await readFile(file) }
		}
	});
	return { status: response.status(), body: (await response.json()) as { id?: string; state?: string } };
}

/** One directory album through the API; the creation enqueues its
 * first scan, so the album converges without a manual press. */
async function createAlbum(request: APIRequestContext, title: string, rootPath: string): Promise<string> {
	const response = await request.post('/api/v1/albums', {
		data: { title, rootPath, inTimeline: true }
	});
	expect(response.status()).toBe(201);
	const body = (await response.json()) as { id?: string };
	expect(body.id).toBeTruthy();
	return body.id ?? '';
}

/** Waits until the given album's wall answers exactly `count` items,
 * answering that page's ids. */
async function waitAlbumCount(
	request: APIRequestContext,
	id: string,
	count: number,
	timeout = SCAN_TIMEOUT_MS
): Promise<string[]> {
	let ids: string[] = [];
	await expect
		.poll(
			async () => {
				ids = await albumIds(request, id);
				return ids.length;
			},
			{ timeout, intervals: [500, 1000, 2000] }
		)
		.toBe(count);
	return ids;
}

test.beforeAll(async ({ playwright }) => {
	// The two source trees, seeded per run and never committed: the
	// read-only one carries the dual-home photo, the writable one the
	// reverse-order photo. The sweep first is the determinism (a stale
	// file would leave the scans' counts with leftovers).
	for (const [dir, seed, name] of [
		[DUAL_HOME_HOST_DIR, DUAL_HOME_SOURCE, DUAL_HOME_NAME],
		[REVERSE_HOST_DIR, REVERSE_SOURCE, REVERSE_NAME]
	] as const) {
		await rm(dir, { recursive: true, force: true });
		await mkdir(dir, { recursive: true });
		await copyFile(seed, path.join(dir, name));
	}

	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: ADMIN_NAME, email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);
	await api.dispose();
});

test.afterAll(async () => {
	// The seeded trees are this spec's own: both leave so the fixture
	// directories stay committed-empty (the README alone).
	await rm(DUAL_HOME_HOST_DIR, { recursive: true, force: true });
	await rm(REVERSE_HOST_DIR, { recursive: true, force: true });
});

test('scene 1: the administrator signs in and the library bootstrap converges', async ({ page }) => {
	test.setTimeout(DOWNLOAD_TIMEOUT_MS + 120_000);
	await page.goto('/login?redirectTo=%2Ftimeline');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/timeline$/);

	// The model group: the scans' pipelines run the same fan-out an
	// upload's does, so the tiles converge the same way.
	const download = await page.request.post('/api/v1/admin/models/download');
	expect([202, 200]).toContain(download.status());
	await expect
		.poll(
			async () => {
				const jobs = await page.request.get('/api/v1/admin/jobs');
				expect(jobs.status()).toBe(200);
				const body = (await jobs.json()) as { jobs?: Array<{ kind?: string; state?: string }> };
				return body.jobs?.find((job) => job.kind === 'model_download')?.state ?? 'missing';
			},
			{ timeout: DOWNLOAD_TIMEOUT_MS, intervals: [1000, 2000, 5000] }
		)
		.toBe('succeeded');

	await page.context().storageState({ path: authStatePath });
});

test.describe('the admin scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the uploaded copy is the timeline tile and the only copy', async ({ page }) => {
		const upload = await uploadFile(page.request, DUAL_HOME_SOURCE, DUAL_HOME_NAME, 'e42-dual-home');
		expect(upload.status).toBe(201);
		expect(upload.body.state).toBe('created');
		uploadId = upload.body.id ?? '';
		expect(uploadId).not.toBe('');

		// One tile, the upload, and no twin yet.
		await expect
			.poll(async () => timelineIds(page.request), { timeout: 60_000, intervals: [500, 1000, 2000] })
			.toEqual([uploadId]);
		const detail = await detailOf(page.request, uploadId);
		expect(detail.contentCopies).toBe(0);
	});

	test('scene 3: the read-only album mirrors the folder as its own position row', async ({ page }) => {
		albumId = await createAlbum(page.request, DUAL_HOME_ALBUM, DUAL_HOME_DIR);
		const ids = await waitAlbumCount(page.request, albumId, 1);
		positionId = ids[0] ?? '';
		expect(positionId).not.toBe('');
		// The album's copy is NOT the upload: its own row, at its own
		// position (the mirror's whole point).
		expect(positionId).not.toBe(uploadId);

		// The album's detail answers the source's type and its
		// read-only fact, and the twin count now speaks.
		const position = await detailOf(page.request, positionId);
		const albumRef = (position.albums ?? []).find((album) => album.id === albumId);
		expect(albumRef?.type).toBe('directory');
		expect(albumRef?.sourceWritable).toBe(false);
		expect(position.contentCopies).toBe(1);
		const upload = await detailOf(page.request, uploadId);
		expect(upload.contentCopies).toBe(1);

		// The timeline still shows ONE tile for the content: the
		// upload (the live blob outranks the position row).
		await expect
			.poll(async () => timelineIds(page.request), { timeout: 30_000, intervals: [500, 1000, 2000] })
			.toEqual([uploadId]);

		// The wall's own face: the album page renders the tile under
		// the file's name -- and its link opens the position row.
		await page.goto(`/albums/${albumId}`);
		await expect(page.getByRole('link', { name: DUAL_HOME_NAME, exact: true })).toBeVisible({
			timeout: SCENE_WAIT_MS
		});
		await page.getByRole('link', { name: DUAL_HOME_NAME, exact: true }).click();
		await expect(page).toHaveURL(new RegExp(`/a/[0-9a-f-]{36}$`));
		expect(page.url()).toContain(positionId);
	});

	test('scene 4: the delete answers per copy -- the album has none, My photos has one', async ({ page }) => {
		// The read-only source's copy: no delete affordance at all
		// (the application cannot delete what the folder owns).
		await page.goto(`/a/${positionId}`);
		await expect(page.getByRole('button', { name: 'Favorite', exact: true })).toBeVisible({
			timeout: SCENE_WAIT_MS
		});
		await expect(page.getByRole('button', { name: 'Trash', exact: true })).toHaveCount(0);

		// The uploaded copy: the delete is there, and its landing echo
		// carries the twin note.
		await page.goto(`/a/${uploadId}`);
		const trash = page.getByRole('button', { name: 'Trash', exact: true });
		await expect(trash).toBeVisible({ timeout: SCENE_WAIT_MS });
		await trash.click();
		await expect(page.locator('.toast')).toContainText('Moved to the trash. 1 more copy of this photo remains.');

		// The survival: the timeline's tile is the position row now
		// (the group's representative), still one tile.
		await expect
			.poll(async () => timelineIds(page.request), { timeout: 30_000, intervals: [500, 1000, 2000] })
			.toEqual([positionId]);

		// And it lost the delete button: its only home is the
		// read-only source.
		await page.goto(`/a/${positionId}`);
		await expect(page.getByRole('button', { name: 'Favorite', exact: true })).toBeVisible({
			timeout: SCENE_WAIT_MS
		});
		await expect(page.getByRole('button', { name: 'Trash', exact: true })).toHaveCount(0);

		// The album wall keeps its copy (the folder never lost a byte).
		await page.goto(`/albums/${albumId}`);
		await expect(page.getByRole('link', { name: DUAL_HOME_NAME, exact: true })).toBeVisible({
			timeout: SCENE_WAIT_MS
		});
	});

	test('scene 5: a file scanned first makes the upload a duplicate; the vanished copy makes it land', async ({
		page
	}) => {
		test.setTimeout(SCAN_TIMEOUT_MS + SYNC_TIMEOUT_MS + 120_000);
		reverseAlbumId = await createAlbum(page.request, REVERSE_ALBUM, REVERSE_DIR);
		const ids = await waitAlbumCount(page.request, reverseAlbumId, 1);
		reversePositionId = ids[0] ?? '';
		expect(reversePositionId).not.toBe('');

		// The same bytes now upload: the content is already the
		// owner's (the source position row), so the answer is the
		// duplicate state and NOTHING lands in My photos.
		const syncId = await syncAlbumId(page.request);
		const syncBefore = (await albumIds(page.request, syncId)).length;
		const duplicate = await uploadFile(page.request, REVERSE_SOURCE, REVERSE_NAME, 'e42-reverse');
		expect(duplicate.status).toBe(200);
		expect(duplicate.body.state).toBe('duplicate');
		expect(duplicate.body.id).toBe(reversePositionId);
		expect((await albumIds(page.request, syncId)).length).toBe(syncBefore);

		// The folder copy leaves (the host takes the file away): the
		// scan's orphan mark leaves a dead row no restore can reach.
		await rm(path.join(REVERSE_HOST_DIR, REVERSE_NAME), { force: true });
		await expect
			.poll(
				async () => {
					const response = await page.request.get(`/api/v1/albums/${encodeURIComponent(reverseAlbumId)}`);
					expect(response.status()).toBe(200);
					const body = (await response.json()) as { items?: Array<{ id?: string }> };
					return (body.items ?? []).length;
				},
				{ timeout: SYNC_TIMEOUT_MS, intervals: [1000, 2000, 5000] }
			)
			.toBe(0);

		// The re-upload lands: the dead row is replaced by the fresh
		// uploaded copy, in My photos this time.
		const landed = await uploadFile(page.request, REVERSE_SOURCE, REVERSE_NAME, 'e42-reverse');
		expect(landed.status).toBe(201);
		expect(landed.body.state).toBe('created');
		reverseUploadId = landed.body.id ?? '';
		expect(reverseUploadId).not.toBe(reversePositionId);
		expect((await albumIds(page.request, syncId)).length).toBe(syncBefore + 1);
		const landedDetail = await detailOf(page.request, reverseUploadId);
		expect(landedDetail.contentCopies).toBe(0);
	});

	test('scene 6: a renamed file stays visible through the re-scan', async ({ page }) => {
		test.setTimeout(SCAN_TIMEOUT_MS + 60_000);
		// The rename dissolves under the position model: one row dies
		// (the old path), another is born (the new one).
		const renamed = 'IMG_E42_0001_renamed.jpg';
		await rename(path.join(DUAL_HOME_HOST_DIR, DUAL_HOME_NAME), path.join(DUAL_HOME_HOST_DIR, renamed));
		// The periodic folder sync may already hold the run (the
		// harness shortens its interval): the in-flight guard's 409 is
		// as good as an accepted trigger -- the walk that lands is the
		// same.
		const rescan = await page.request.post(`/api/v1/albums/${encodeURIComponent(albumId)}/rescan`);
		expect([200, 202, 409]).toContain(rescan.status());

		// The album still shows the photo -- under the new name, on a
		// fresh position row. The wait names the IDENTITY, not the
		// count: the old row is still a member until the walk resolves
		// it, and a count-only wait would accept that stale tile.
		let freshPosition = '';
		await expect
			.poll(
				async () => {
					const ids = await albumIds(page.request, albumId);
					freshPosition = ids.length === 1 && ids[0] !== positionId ? (ids[0] ?? '') : '';
					return freshPosition;
				},
				{ timeout: SCAN_TIMEOUT_MS, intervals: [500, 1000, 2000] }
			)
			.not.toBe('');
		// The timeline holds the story's two contents by now: the
		// reverse-order upload (scene 5) and this renamed file's fresh
		// row -- never the dead old row.
		await expect
			.poll(
				async () => {
					const ids = await timelineIds(page.request);
					return ids.slice().sort();
				},
				{ timeout: 30_000, intervals: [500, 1000, 2000] }
			)
			.toEqual([reverseUploadId, freshPosition].sort());
		await page.goto(`/albums/${albumId}`);
		await expect(page.getByRole('link', { name: renamed, exact: true })).toBeVisible({ timeout: SCENE_WAIT_MS });

		// The old tile's name is gone: the mirror follows the folder's
		// names, one row per position.
		await expect(page.getByRole('link', { name: DUAL_HOME_NAME, exact: true })).toHaveCount(0);
	});
});
