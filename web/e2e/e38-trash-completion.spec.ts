// The trash completion story (E38), walked end to end against the
// real compose stack: the recovery view finishes its surface -- the
// note carries the instance's retention days, the toolbar owns the
// whole-trash pair (restore all, empty trash), and the tiles gain
// the wall's multi-select (check affordance, batch bar, confirmed
// batch delete) -- while the row-level actions stay exactly where
// they were, visible without any hover.
//
// The stack's lifecycle is owned by the caller (one fresh stack per
// spec -- the first-run bootstrap below expects its 201); the spec
// itself only talks to http://localhost:8244.
//
// THE SCENES, in order:
//
//  1. The bootstrap: the first-run administrator signs in, and the
//     empty trash answers its shape -- the note's number (the
//     default 30), the toolbar's pair present and disabled (never
//     hidden), and the empty state card.
//  2. Two photos upload through the asset API, get trashed through
//     the assets batch (the seeding path), and the trash page shows
//     their recovery rows: the row actions visible, the note's
//     number matching the admin-set retention.
//  3. Multi-select restore: two checks rise the batch bar, the
//     restore action fires the trash batch, the toast echoes the
//     count, the rows leave, and the photos are back on the
//     timeline.
//  4. Restore all: the toolbar's whole-trash restore sweeps the
//     re-trashed rows, the toast lands, the rows clear, and the
//     timeline holds both photos again.
//  5. Empty trash: the toolbar's dangerous arm opens its confirm
//     (the count in the sentence), the confirm purges everything,
//     the toast lands, and the empty state card returns.
//  6. The retention follows the setting: the administrator lowers
//     the instance's trash retention, and the same page's note and
//     the row's countdown chip show the new number on the next
//     listing.

import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, test, type APIRequestContext, type Page } from '@playwright/test';

test.describe.configure({ mode: 'serial' });

const specDir = fileURLToPath(new URL('.', import.meta.url));
const repoRoot = fileURLToPath(new URL('../..', import.meta.url));

// The compose stack's address (playwright.config.ts pins the same
// value as the suite's baseURL).
const BASE_URL = 'http://localhost:8244';

const ADMIN_EMAIL = 'admin@e38-e2e.invalid';
const ADMIN_NAME = 'Lin Trash';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The two photos of the story: real JPEGs from the server's test
// corpus, uploaded through the asset API (the seeding pattern every
// story uses).
const PHOTO_A_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/golden-large.jpg');
const PHOTO_B_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/portrait-orientation6.jpg');
const PHOTO_A_NAME = 'IMG_E38_0001.jpg';
const PHOTO_B_NAME = 'IMG_E38_0002.jpg';

// The retention the last scene lowers the instance to.
const NEW_RETENTION = 7;

const UPLOAD_TIMEOUT_MS = 120_000;

// The storage-state file the first scene writes and later scenes
// replay (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/e38-state.json`;

/** One trash recovery tile, addressed by its accessible name. */
function trashTile(page: Page, name: string) {
	return page.locator(`.trash__tile a.tile__link[aria-label="${name}"]`);
}

/** One row's icon action (the icon-first rule: the name is the label). */
function rowAction(page: Page, label: string) {
	return page.locator(`.trash__cell button[aria-label="${label}"]`).first();
}

/** The floating batch bar, once at least one tile is selected. */
function batchBar(page: Page, count: number) {
	return page.getByRole('toolbar', { name: `${count} selected` });
}

/** One wall tile on the timeline, addressed by its accessible name. */
function timelineTile(page: Page, name: string) {
	return page.locator(`a.tile__link[aria-label="${name}"]`).first();
}

/** One upload through the asset API, answering the created asset. */
async function uploadPhoto(
	request: APIRequestContext,
	file: string,
	name: string,
	deviceAssetId: string
): Promise<string> {
	const response = await request.post('/api/v1/assets', {
		multipart: {
			deviceAssetId,
			fileName: name,
			deviceCreatedAt: '2026-05-01T12:00:00Z',
			deviceTimeZone: 'UTC',
			photo: { name, mimeType: 'image/jpeg', buffer: await readFile(file) }
		}
	});
	expect(response.status()).toBe(201);
	const body = (await response.json()) as { id?: string };
	expect(body.id).toBeTruthy();
	return body.id ?? '';
}

/** Batch-trashes the given assets through the assets batch (the
 * same surface the walls' selection bar drives). */
async function trashAssets(request: APIRequestContext, ids: string[]): Promise<void> {
	const response = await request.post('/api/v1/assets/batch', {
		data: { action: 'trash', ids }
	});
	expect(response.status()).toBe(200);
}

/** Waits until the trash listing answers exactly the given count. */
async function waitTrashCount(request: APIRequestContext, count: number): Promise<void> {
	await expect
		.poll(
			async () => {
				const response = await request.get('/api/v1/trash');
				expect(response.status()).toBe(200);
				const body = (await response.json()) as { items?: unknown[] };
				return (body.items ?? []).length;
			},
			{ timeout: UPLOAD_TIMEOUT_MS, intervals: [500, 1000, 2000] }
		)
		.toBe(count);
}

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: ADMIN_NAME, email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	await api.dispose();
});

test('scene 1: the empty trash answers its shape -- the note, the disabled pair, the empty card', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Ftrash');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/trash$/);

	// The note's number: the fresh instance's default retention.
	await expect(page.getByText('Deleted photos stay here for 30 days', { exact: false })).toBeVisible();

	// The whole-trash pair: present and disabled on an empty trash,
	// never hidden.
	const restoreAll = page.getByRole('button', { name: 'Restore all', exact: true });
	await expect(restoreAll).toBeVisible();
	await expect(restoreAll).toBeDisabled();
	const empty = page.getByRole('button', { name: 'Empty trash', exact: true });
	await expect(empty).toBeVisible();
	await expect(empty).toBeDisabled();

	// The empty state card.
	await expect(page.getByText('The trash is empty', { exact: true })).toBeVisible();

	// Hand the session to the later scenes.
	await page.context().storageState({ path: authStatePath });
});

test.describe('the trash scenes', () => {
	test.use({ storageState: authStatePath });

	let photoA = '';
	let photoB = '';

	test('scene 2: two trashed photos render their recovery rows, the row actions visible', async ({ page }) => {
		photoA = await uploadPhoto(page.request, PHOTO_A_SOURCE, PHOTO_A_NAME, 'e38-device-a');
		photoB = await uploadPhoto(page.request, PHOTO_B_SOURCE, PHOTO_B_NAME, 'e38-device-b');
		// The tiles' thumbnails converge through the pipeline before
		// the rows are worth asserting (the recovery tile paints the
		// same serving copy the walls do).
		await expect
			.poll(async () => {
				const response = await page.request.get(`/api/v1/assets/${photoA}`);
				expect(response.status()).toBe(200);
				return ((await response.json()) as { isReady?: boolean }).isReady ?? false;
			})
			.toBe(true);
		await expect
			.poll(async () => {
				const response = await page.request.get(`/api/v1/assets/${photoB}`);
				expect(response.status()).toBe(200);
				return ((await response.json()) as { isReady?: boolean }).isReady ?? false;
			})
			.toBe(true);

		await trashAssets(page.request, [photoA, photoB]);
		await waitTrashCount(page.request, 2);

		await page.goto('/trash');
		await expect(trashTile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(trashTile(page, PHOTO_B_NAME)).toBeVisible();

		// The row actions are ALWAYS visible (the hover-hide reading
		// was verified false and pinned away): the icon buttons sit on
		// the row under the tile, no hover needed.
		await expect(rowAction(page, 'Restore')).toBeVisible();
		await expect(rowAction(page, 'Delete permanently')).toBeVisible();
		await expect(rowAction(page, 'Restore')).toBeEnabled();

		// The note still names the default retention.
		await expect(page.getByText('Deleted photos stay here for 30 days', { exact: false })).toBeVisible();
	});

	test('scene 3: selecting two rises the batch bar, the batch restore returns them to the timeline', async ({
		page
	}) => {
		await page.goto('/trash');
		await expect(trashTile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(trashTile(page, PHOTO_B_NAME)).toBeVisible();

		// The check affordances: hover-revealed, aria-checked, the
		// wall's own vocabulary.
		const checkA = trashTile(page, PHOTO_A_NAME).locator('..').getByRole('checkbox');
		const checkB = trashTile(page, PHOTO_B_NAME).locator('..').getByRole('checkbox');
		await checkA.hover();
		await checkA.click();
		await expect(batchBar(page, 1)).toBeVisible();
		await checkB.click();
		await expect(batchBar(page, 2)).toBeVisible();

		await batchBar(page, 2).getByRole('button', { name: 'Restore', exact: true }).click();

		// The toast echoes the count; the rows leave; the selection
		// clears (the bar leaves).
		await expect(page.getByText('Restored 2.', { exact: true })).toBeVisible();
		await expect(batchBar(page, 2)).toHaveCount(0);
		await expect(trashTile(page, PHOTO_A_NAME)).toHaveCount(0);
		await expect(trashTile(page, PHOTO_B_NAME)).toHaveCount(0);

		// The photos are back on the timeline.
		await page.goto('/timeline');
		await expect(timelineTile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(timelineTile(page, PHOTO_B_NAME)).toBeVisible();
	});

	test('scene 4: restore-all sweeps the re-trashed rows and the timeline holds them again', async ({ page }) => {
		await trashAssets(page.request, [photoA, photoB]);
		await waitTrashCount(page.request, 2);

		await page.goto('/trash');
		await expect(trashTile(page, PHOTO_A_NAME)).toBeVisible();

		// The toolbar's whole-trash restore runs directly (a whole
		// restore is reversible through the timeline's own trash
		// flow): the toast lands, the rows clear.
		await page.getByRole('button', { name: 'Restore all', exact: true }).click();
		await expect(page.getByText('All restored.', { exact: true })).toBeVisible();
		await expect(trashTile(page, PHOTO_A_NAME)).toHaveCount(0);
		await expect(trashTile(page, PHOTO_B_NAME)).toHaveCount(0);
		await expect(page.getByText('The trash is empty', { exact: true })).toBeVisible();

		await page.goto('/timeline');
		await expect(timelineTile(page, PHOTO_A_NAME)).toBeVisible();
		await expect(timelineTile(page, PHOTO_B_NAME)).toBeVisible();
	});

	test('scene 5: empty trash confirms first with the count, then the empty card returns', async ({ page }) => {
		await trashAssets(page.request, [photoA, photoB]);
		await waitTrashCount(page.request, 2);

		await page.goto('/trash');
		await expect(trashTile(page, PHOTO_A_NAME)).toBeVisible();

		// The confirm-first gate: the toolbar button opens the modal,
		// the count rides the sentence, nothing is destroyed yet.
		await page.getByRole('button', { name: 'Empty trash', exact: true }).click();
		const dialog = page.getByRole('dialog', { name: 'Empty the trash?', exact: true });
		await expect(dialog).toBeVisible();
		await expect(dialog).toContainText('all 2 photos in the trash');
		await expect(trashTile(page, PHOTO_A_NAME)).toBeVisible();

		// The confirm purges everything; the toast and the empty
		// state card land.
		await dialog.getByRole('button', { name: 'Empty trash', exact: true }).click();
		await expect(page.getByText('Trash emptied.', { exact: true })).toBeVisible();
		await expect(trashTile(page, PHOTO_A_NAME)).toHaveCount(0);
		await expect(trashTile(page, PHOTO_B_NAME)).toHaveCount(0);
		await expect(page.getByText('The trash is empty', { exact: true })).toBeVisible();

		// The timeline lost both photos for good.
		await page.goto('/timeline');
		await expect(timelineTile(page, PHOTO_A_NAME)).toHaveCount(0);
		await expect(timelineTile(page, PHOTO_B_NAME)).toHaveCount(0);
	});

	test('scene 6: the administrator lowers the retention and the note follows', async ({ page }) => {
		// One photo to read the note against (the photos of the story
		// are gone; the scene uploads a fresh one and trashes it).
		const photoC = await uploadPhoto(page.request, PHOTO_A_SOURCE, 'IMG_E38_0003.jpg', 'e38-device-c');
		await expect
			.poll(async () => {
				const response = await page.request.get(`/api/v1/assets/${photoC}`);
				expect(response.status()).toBe(200);
				return ((await response.json()) as { isReady?: boolean }).isReady ?? false;
			})
			.toBe(true);
		await trashAssets(page.request, [photoC]);
		await waitTrashCount(page.request, 1);

		// The administrator's surface: the instance's trash retention
		// lowers to the scene's number (the API the admin settings
		// page drives).
		const update = await page.request.put('/api/v1/admin/settings/trash', {
			data: { retentionDays: NEW_RETENTION }
		});
		expect(update.status()).toBe(200);

		// The same page's note follows the setting on the next
		// listing, and the row's countdown chip reads the same source.
		await page.goto('/trash');
		await expect(
			page.getByText(`Deleted photos stay here for ${NEW_RETENTION} days`, { exact: false })
		).toBeVisible();
		await expect(page.getByText(`${NEW_RETENTION} days left`, { exact: true })).toBeVisible();
	});
});
