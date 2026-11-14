// The background work's cost story, walked end to end against the
// real compose stack: the settings page's collapsed index, the image
// card's honest cost line and its explicit rebuild (the save never
// recomputes; the two scopes pay differently), the per-kind pause
// holding a new upload's thumbnail job, and the night window holding
// a video transcode until the window opens.
//
// THE STACK ARRANGEMENT: one fresh lifetime owns this spec (the
// first-run bootstrap creates the administrator). No release feed is
// ever contacted (the deployment seals the check off), and no model
// download is required: the pipeline's readiness counts the metadata
// and thumbnail (and transcode) jobs only.
//
// THE SCENES (in order):
//
//  1. The administrator signs in through the form and lands on the
//     settings page's collapsed index: seven folded cards, every
//     summary naming its card's current value, no card open in the
//     markup.
//  2. The fold is a real fold: clicking a summary opens it, a
//     keyboard Space toggles it, and a reload restores exactly the
//     cards the admin left open (the expanded set in localStorage --
//     the ones left closed stay closed).
//  3. The image card saves the document and NOTHING else: no
//     rebuild run appears behind the save's back (the job listing
//     stays empty of the kind). The rebuild dialog opens over the
//     collapsed card (the deep link), the missing scope is the
//     default, and the started run converges on the task panel.
//  4. The two scopes pay differently: with two photos whose AVIF
//     derivatives are already on disk and the encoding saved as
//     WebP, the missing rebuild skips them (the thumbnail still
//     serves AVIF), and the full rebuild re-derives at the saved
//     settings (the thumbnail serves WebP, the stale encoding
//     swept).
//  5. The pause: with the thumbnail kind held, a fresh upload's
//     thumbnail job sits held -- the asset exists, the thumbnail
//     endpoint answers not-ready -- and lifting the pause lets it
//     finish. The upload request itself was never blocked.
//  6. The night window: with the window placed strictly ahead of
//     now, a Live Photo pair's video transcode sits held (the play
//     endpoint answers not-ready), and switching the window off
//     lets it finish. Browsing was never blocked either way.

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

// The story's administrator: created through the first-run setup API
// in the suite's bootstrap (the acceptance scripts' pattern). The
// stack is always fresh, so the fixed identity is deterministic.
const ADMIN_EMAIL = 'admin@e33-e2e.invalid';
const ADMIN_PASSWORD = 'correct-horse-battery';

// The two seed photos (genuinely distinct portraits) and the Live
// Photo pair's video (the tiny real MP4 the upload stories seed).
const PHOTO_A = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait.jpg');
const PHOTO_B = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other.jpg');
const LIVE_PHOTO_SOURCE = path.join(repoRoot, 'server/internal/ml/testdata/face-portrait-other-second.jpg');
const LIVE_VIDEO_SOURCE = path.join(repoRoot, 'server/internal/media/testdata/tiny.mp4');

// The storage state scene 1 writes and the later scenes replay
// (git-ignored, recreated per run).
const authStatePath = `${specDir}/.auth/e33-state.json`;

// The photo day both seeds carry (the timeline's own grouping is not
// this story's business; the fixed day keeps the uploads ordered).
const PHOTO_DAY = '2026-09-18';

/** One budget body (the contract's RunBudgetSettingsUpdateRequest). */
function budgetBody(overrides: Record<string, unknown> = {}): Record<string, unknown> {
	return {
		concurrencyPreset: 'standard',
		nightlyEnabled: false,
		nightlyStart: '02:00',
		nightlyEnd: '06:00',
		pausedJobs: { thumbnail: false, metadata: false, transcode: false, ml: false, places: false },
		...overrides
	};
}

/** Save the whole run budget through the admin API. */
async function saveBudget(api: APIRequestContext, overrides: Record<string, unknown>): Promise<void> {
	const response = await api.put('/api/v1/admin/settings/run-budget', {
		data: budgetBody(overrides)
	});
	expect(response.status(), 'the run budget save').toBe(200);
}

/** Save the image settings through the admin API. */
async function saveImageSettings(api: APIRequestContext, overrides: Record<string, unknown>): Promise<void> {
	const response = await api.put('/api/v1/admin/settings/image', {
		data: {
			tileSize: 250,
			previewSize: 1080,
			format: 'avif',
			fullSize: false,
			rawEmbedded: true,
			...overrides
		}
	});
	expect(response.status(), 'the image settings save').toBe(200);
}

/**
 * The seed bytes carry a one-line marker: the same fixture reused
 * across scenes must be DISTINCT bytes, or the content-addressed
 * dedupe answers 200 with the existing asset instead of 201 with a
 * new one.
 */
function seedBytes(source: Buffer, index: number): Buffer {
	return Buffer.concat([source, Buffer.from(`\nE33-E2E-SEED-${index}\n`)]);
}

/** Upload one photo through the real upload API, returning the asset id. */
async function uploadPhoto(
	api: APIRequestContext,
	index: number,
	source: string,
	timeOffsetMinutes: number
): Promise<string> {
	const name = `IMG_E33_${index}.jpg`;
	const response = await api.post('/api/v1/assets', {
		multipart: {
			deviceAssetId: `e33-e2e-${index}`,
			fileName: name,
			deviceCreatedAt: `${PHOTO_DAY}T10:${String(timeOffsetMinutes).padStart(2, '0')}:00Z`,
			deviceTimeZone: 'UTC',
			photo: {
				name,
				mimeType: 'image/jpeg',
				buffer: seedBytes(await readFile(source), index)
			}
		}
	});
	expect(response.status(), `the upload of ${name}`).toBe(201);
	const created = (await response.json()) as { id: string };
	return created.id;
}

/**
 * The asset's thumbnail readiness: the endpoint answers 200 with the
 * derived file once the thumbnail job landed it, and 404
 * (asset_not_ready) while the job has not. The content type carries
 * the encoding the file was derived at.
 */
async function thumbnailHead(
	api: APIRequestContext,
	assetId: string
): Promise<{ status: number; contentType: string }> {
	const response = await api.get(`/api/v1/assets/${assetId}/thumbnail?size=tile`);
	return { status: response.status(), contentType: response.headers()['content-type'] ?? '' };
}

/** Poll the thumbnail until it answers 200 (the derived file exists). */
async function waitThumbnailReady(api: APIRequestContext, assetId: string): Promise<string> {
	let contentType = '';
	await expect
		.poll(
			async () => {
				const probe = await thumbnailHead(api, assetId);
				contentType = probe.contentType;
				return probe.status;
			},
			{ timeout: 120_000, intervals: [500, 1000, 2000] }
		)
		.toBe(200);
	return contentType;
}

/**
 * The pipeline's completion signal, both sizes: the tile answers
 * first (the job writes it before the view), so the view's own 200
 * is what "the pipeline finished deriving this asset" means from
 * the outside.
 */
async function waitPipelineDone(api: APIRequestContext, assetId: string): Promise<void> {
	await waitThumbnailReady(api, assetId);
	await expect
		.poll(async () => (await api.get(`/api/v1/assets/${assetId}/thumbnail?size=view`)).status(), {
			timeout: 120_000,
			intervals: [500, 1000, 2000]
		})
		.toBe(200);
}

/** Poll the play endpoint until it answers 200 (the transcode landed). */
async function waitPlayReady(api: APIRequestContext, assetId: string): Promise<void> {
	await expect
		.poll(
			async () => {
				const response = await api.get(`/api/v1/assets/${assetId}/play`);
				return response.status();
			},
			{ timeout: 180_000, intervals: [500, 1000, 2000] }
		)
		.toBe(200);
}

/** The rebuild runs of the listing, by kind. */
async function rebuildRuns(
	api: APIRequestContext
): Promise<Array<{ id: string; state: string; total: number; done: number; failed: number }>> {
	const response = await api.get('/api/v1/admin/jobs?limit=200');
	expect(response.status()).toBe(200);
	const body = (await response.json()) as {
		jobs: Array<{ kind: string; id: string; state: string; total: number; done: number; failed: number }>;
	};
	return body.jobs
		.filter((job) => job.kind === 'rebuild_derivatives')
		.map(({ id, state, total, done, failed }) => ({ id, state, total, done, failed }));
}

/** Poll until one rebuild run closed, returning its row. */
async function waitRebuildDone(
	api: APIRequestContext,
	runId: string
): Promise<{ state: string; total: number; done: number; failed: number }> {
	let run: { state: string; total: number; done: number; failed: number } | undefined;
	await expect
		.poll(
			async () => {
				const runs = await rebuildRuns(api);
				run = runs.find((entry) => entry.id === runId);
				return run !== undefined && (run.state === 'succeeded' || run.state === 'failed');
			},
			{ timeout: 180_000, intervals: [1000, 2000, 5000] }
		)
		.toBe(true);
	return run!;
}

/** Open one settings card through its summary (the native fold). */
async function openCard(page: Page, testId: string): Promise<void> {
	const summary = page.getByTestId(testId).locator('summary');
	await summary.click();
}

test.beforeAll(async ({ playwright }) => {
	// The first-run bootstrap, the acceptance scripts' pattern: the
	// setup API creates the administrator on the fresh instance.
	const api = await playwright.request.newContext({ baseURL: BASE_URL });
	const setup = await api.post('/api/v1/server/setup/admin', {
		data: { name: 'E33 E2E', email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(setup.status()).toBe(201);
	const login = await api.post('/api/v1/auth/login', {
		data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD }
	});
	expect(login.status()).toBe(200);
	await api.dispose();
});

// scene 1 signs in through the form inside the DEFAULT context (the
// storage state it writes is what the later scenes replay).
test('scene 1: sign in, then the settings page is the collapsed index', async ({ page }) => {
	await page.goto('/login?redirectTo=%2Fadmin%2Fsettings');
	await page.getByLabel('Email').fill(ADMIN_EMAIL);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Sign in', exact: true }).click();
	await expect(page).toHaveURL(/\/admin\/settings$/);

	// The index: seven folded cards, none open in the markup, and
	// every collapsed row naming its card's current value.
	const cards = page.locator('details.settings__card');
	await expect(cards).toHaveCount(7);
	for (let i = 0; i < 7; i += 1) {
		await expect(cards.nth(i)).not.toHaveAttribute('open');
	}
	const storageMeta = page.getByTestId('admin-settings-storage').locator('.settings__section-meta');
	await expect(storageMeta).toContainText('{{y}}/{{MM}}/{{filename}}');
	const imageMeta = page.getByTestId('admin-settings-image').locator('.settings__section-meta');
	await expect(imageMeta).toContainText('250');
	await expect(imageMeta).toContainText('1080');
	await expect(imageMeta).toContainText('AVIF');
	const trashMeta = page.getByTestId('admin-settings-trash').locator('.settings__section-meta');
	await expect(trashMeta).toContainText('30');

	// The admin session rides to every later scene.
	await page.context().storageState({ path: authStatePath });
});

test.describe('the administrator scenes', () => {
	test.use({ storageState: authStatePath });

	test('scene 2: the fold opens by pointer and by keyboard, and the expanded set survives a reload', async ({
		page
	}) => {
		await page.goto('/admin/settings');

		// Pointer: the image card's summary opens it.
		await openCard(page, 'admin-settings-image');
		await expect(page.getByTestId('admin-settings-image')).toHaveAttribute('open');

		// Keyboard: the update card's summary takes focus and Space
		// toggles it (the native fold's own behavior).
		const updateSummary = page.getByTestId('admin-settings-update').locator('summary');
		await updateSummary.press('Space');
		await expect(page.getByTestId('admin-settings-update')).toHaveAttribute('open');

		// Two open cards sit side by side (no accordion, no
		// mutual exclusion).
		await expect(page.getByTestId('admin-settings-image')).toHaveAttribute('open');
		await expect(page.getByTestId('admin-settings-update')).toHaveAttribute('open');
		await expect(page.getByTestId('admin-settings-storage')).not.toHaveAttribute('open');

		// The reload restores exactly the expanded set: the two the
		// admin opened are open, the rest stayed closed.
		await page.reload();
		await expect(page.getByTestId('admin-settings-image')).toHaveAttribute('open');
		await expect(page.getByTestId('admin-settings-update')).toHaveAttribute('open');
		await expect(page.getByTestId('admin-settings-storage')).not.toHaveAttribute('open');
		await expect(page.locator('details.settings__card[open]')).toHaveCount(2);
	});

	test('scene 3: the image card saves without recomputing, and the rebuild dialog starts the run', async ({
		page
	}) => {
		await page.goto('/admin/settings');
		await openCard(page, 'admin-settings-image');

		// The consequence line rides in the card: the save recomputes
		// nothing.
		await expect(page.getByTestId('admin-settings-image')).toContainText('Saving recomputes nothing');
		// The cost line names the (empty) library and the declared
		// estimate's floor.
		await expect(page.getByTestId('image-summary')).toContainText('about 1 min');

		// Save the document with the 320 tile. The dialog has not
		// been touched: no rebuild run may exist behind the save's
		// back.
		await page.locator('#admin-image-thumb').selectOption('320');
		await page.getByTestId('admin-settings-image').getByRole('button', { name: 'Save' }).click();
		await expect(page.getByTestId('image-saved')).toBeVisible();
		const api = page.request;
		const runs = await rebuildRuns(api);
		expect(runs).toEqual([]);

		// The rebuild dialog opens over whatever fold state the page
		// carries (a page-level layer, independent of the index): a
		// fresh visit lands collapsed, the deep link opens the dialog.
		await page.goto('/admin/settings?dialog=rebuild-derivatives');
		await expect(page.getByTestId('rebuild-dialog')).toBeVisible();
		// The missing scope is the default choice.
		await expect(page.locator('[data-testid="rebuild-dialog"] input[value="missing"]')).toBeChecked();
		await page.getByTestId('rebuild-start').click();
		await expect(page.getByTestId('rebuild-dialog')).toBeHidden();

		// The run converges on the task panel with its honest counts:
		// an empty library's scope is zero assets, visited zero times.
		await page.goto('/admin/jobs');
		const row = page.locator('article.jobs__row').filter({ hasText: 'Derivative rebuild' }).first();
		await expect(row).toBeVisible();
		await expect(row.getByText('Succeeded', { exact: true })).toBeVisible({ timeout: 60_000 });
	});

	test('scene 4: the two scopes pay differently -- missing skips, full re-derives', async ({ page }) => {
		const api = page.request;

		// The settings go back to the shipped tile first: the two
		// uploads derive their 250-box tiles, and only then does the
		// saved 320 posture exist.
		await page.goto('/admin/jobs');
		await saveImageSettings(api, { tileSize: 250 });
		const photoA = await uploadPhoto(api, 1, PHOTO_A, 0);
		const photoB = await uploadPhoto(api, 2, PHOTO_B, 1);
		await waitPipelineDone(api, photoA);
		await waitPipelineDone(api, photoB);
		// The read carries a fresh cache buster: the endpoint's
		// immutable cache policy would otherwise serve the run's
		// BEFORE tile from the browser cache.
		const tileWidthOf = (assetId: string): Promise<number> =>
			page.evaluate(async (url) => {
				const image = new Image();
				image.src = url;
				await image.decode();
				return image.naturalWidth;
			}, `${BASE_URL}/api/v1/assets/${assetId}/thumbnail?size=tile&bust=${Math.random()}`);
		const preWidth = await tileWidthOf(photoA);
		console.log(
			'DEBUGSCENE4 pre-missing width',
			preWidth,
			'settings',
			JSON.stringify(await (await api.get('/api/v1/admin/settings/image')).json())
		);
		expect(preWidth).toBe(200);

		// The missing rebuild: the scope skips what is present. Both
		// assets' current-format tiles exist (at 250): the run visits
		// both and rewrites NOTHING -- the tiles stay 250.
		await saveImageSettings(api, { tileSize: 320 });
		const startMissing = await api.post('/api/v1/admin/derivatives/rebuild', {
			data: { scope: 'missing' }
		});
		expect(startMissing.status()).toBe(202);
		const missingRun = ((await startMissing.json()) as { jobRunId: string }).jobRunId;
		const missing = await waitRebuildDone(api, missingRun);
		expect(missing.state).toBe('succeeded');
		expect(missing.total).toBe(2);
		expect(missing.done).toBe(2);
		const postMissingWidth = await tileWidthOf(photoA);
		console.log(
			'DEBUGSCENE4 post-missing width',
			postMissingWidth,
			'settings',
			JSON.stringify(await (await api.get('/api/v1/admin/settings/image')).json())
		);
		expect(postMissingWidth).toBe(200);
		expect(await tileWidthOf(photoB)).toBe(189);

		// The full rebuild re-derives BOTH at the saved settings: the
		// tiles come back at the 320 box (the portrait sources fit
		// 256x320 and 242x320).
		const startAll = await api.post('/api/v1/admin/derivatives/rebuild', {
			data: { scope: 'all' }
		});
		expect(startAll.status()).toBe(202);
		const allRun = ((await startAll.json()) as { jobRunId: string }).jobRunId;
		const all = await waitRebuildDone(api, allRun);
		expect(all.state).toBe('succeeded');
		expect(all.done).toBe(2);
		await expect.poll(async () => tileWidthOf(photoA), { timeout: 60_000, intervals: [500, 1000] }).toBe(256);
		await expect.poll(async () => tileWidthOf(photoB), { timeout: 60_000, intervals: [500, 1000] }).toBe(242);
	});

	test('scene 5: the pause holds a fresh upload without touching the request', async ({ page }) => {
		const api = page.request;

		// The thumbnail kind held. The upload REQUEST answers 201
		// immediately (the pause never touches the foreground): the
		// job is what sits held.
		await saveBudget(api, {
			pausedJobs: { thumbnail: true, metadata: false, transcode: false, ml: false, places: false }
		});
		const heldUpload = Date.now();
		const photoC = await uploadPhoto(api, 3, PHOTO_A, 2);
		const uploadLatency = Date.now() - heldUpload;
		expect(uploadLatency).toBeLessThan(10_000);

		// The metadata job finishes; the thumbnail job does not: the
		// endpoint answers not-ready.
		await page.waitForTimeout(5_000);
		expect((await thumbnailHead(api, photoC)).status).toBe(404);

		// The strip reflects the held switch (the admin sees WHY).
		await page.goto('/admin/jobs');
		await expect(page.getByTestId('run-pause-thumbnail')).toBeChecked();
		await expect(page.getByTestId('run-dial-standard')).toHaveAttribute('aria-pressed', 'true');

		// Lifting the pause: the held job wakes within the hold's
		// granularity and finishes.
		await saveBudget(api, {});
		const contentType = await waitThumbnailReady(api, photoC);
		expect(contentType).toBe('image/avif');
	});

	test('scene 6: the night window holds the transcode until the window opens', async ({ page }) => {
		const api = page.request;

		// The window placed strictly ahead of now (now+2h .. now+3h in
		// the server's own clock): the heavy work cannot run.
		const now = new Date();
		const start = new Date(now.getTime() + 2 * 3600 * 1000);
		const end = new Date(now.getTime() + 3 * 3600 * 1000);
		const clock = (d: Date): string =>
			`${String(d.getUTCHours()).padStart(2, '0')}:${String(d.getUTCMinutes()).padStart(2, '0')}`;
		await saveBudget(api, { nightlyEnabled: true, nightlyStart: clock(start), nightlyEnd: clock(end) });

		// The Live Photo pair: the photo's pipeline runs (light work),
		// the component video's transcode sits held -- the play
		// endpoint answers not-ready while the window is closed.
		const seed = await api.post('/api/v1/assets', {
			multipart: {
				deviceAssetId: 'e33-e2e-live',
				fileName: 'IMG_E33_LIVE.jpg',
				deviceCreatedAt: `${PHOTO_DAY}T11:00:00Z`,
				deviceTimeZone: 'UTC',
				photo: {
					name: 'IMG_E33_LIVE.jpg',
					mimeType: 'image/jpeg',
					buffer: await readFile(LIVE_PHOTO_SOURCE)
				},
				liveVideo: {
					name: 'IMG_E33_LIVE.mp4',
					mimeType: 'video/mp4',
					buffer: await readFile(LIVE_VIDEO_SOURCE)
				}
			}
		});
		expect(seed.status()).toBe(201);
		const seeded = (await seed.json()) as { id: string };
		const detail = await api.get(`/api/v1/assets/${seeded.id}`);
		const asset = (await detail.json()) as { livePhotoVideoId?: string };
		expect(asset.livePhotoVideoId).toBeTruthy();
		const componentId = asset.livePhotoVideoId as string;

		// The pair's still photo derives whatever the window says
		// (browsing is never held); the transcode does not land.
		await waitThumbnailReady(api, seeded.id);
		const heldProbe = await api.get(`/api/v1/assets/${componentId}/play`);
		expect(heldProbe.status()).toBe(404);

		// The strip renders the open window's times: the switch on,
		// the inputs alive.
		await page.goto('/admin/jobs');
		await expect(page.getByTestId('run-nightly-toggle')).toBeChecked();
		await expect(page.getByTestId('run-nightly-start')).toBeEnabled();
		await expect(page.getByTestId('run-nightly-start')).toHaveValue(clock(start));

		// The window off: the held transcode wakes and finishes -- the
		// pair's motion plays.
		await saveBudget(api, {});
		await waitPlayReady(api, componentId);
	});
});
